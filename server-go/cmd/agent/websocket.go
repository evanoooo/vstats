package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	InitialReconnectDelay  = 5 * time.Second
	MaxReconnectDelay      = 60 * time.Second
	AuthTimeout            = 10 * time.Second
	PingInterval           = 30 * time.Second
	BatchSyncInterval      = 30 * time.Second  // How often to sync offline data
	AggregationSyncInterval = 60 * time.Second // How often to sync aggregated data
)

type WebSocketClient struct {
	config       *AgentConfig
	collector    *MetricsCollector
	store        *LocalStore
	connected    bool
	connectedMu  sync.RWMutex
	lastSentTime time.Time
}

func NewWebSocketClient(config *AgentConfig) *WebSocketClient {
	wsc := &WebSocketClient{
		config:    config,
		collector: NewMetricsCollector(config.IntervalSecs),
	}

	// Initialize local storage if enabled
	if config.EnableOfflineStorage {
		store, err := NewLocalStore(config.DataDir)
		if err != nil {
			log.Printf("Warning: Failed to initialize offline storage: %v", err)
		} else {
			log.Printf("Offline storage enabled at %s", config.DataDir)
			wsc.store = store
		}
	}

	return wsc
}

func (wsc *WebSocketClient) isConnected() bool {
	wsc.connectedMu.RLock()
	defer wsc.connectedMu.RUnlock()
	return wsc.connected
}

func (wsc *WebSocketClient) setConnected(connected bool) {
	wsc.connectedMu.Lock()
	defer wsc.connectedMu.Unlock()
	wsc.connected = connected
}

func (wsc *WebSocketClient) Run() {
	reconnectDelay := InitialReconnectDelay

	// Start offline metrics collection goroutine
	offlineMetricsCh := make(chan *SystemMetrics, 100)
	go wsc.offlineCollector(offlineMetricsCh)

	for {
		log.Printf("Connecting to %s...", wsc.config.WSUrl())

		if err := wsc.connectAndRun(offlineMetricsCh); err != nil {
			log.Printf("Connection error: %v", err)
			wsc.setConnected(false)
		} else {
			log.Println("Connection closed normally")
			wsc.setConnected(false)
			reconnectDelay = InitialReconnectDelay
		}

		log.Printf("Reconnecting in %v...", reconnectDelay)
		time.Sleep(reconnectDelay)

		// Exponential backoff
		reconnectDelay *= 2
		if reconnectDelay > MaxReconnectDelay {
			reconnectDelay = MaxReconnectDelay
		}
	}
}

// offlineCollector collects metrics and stores them locally when disconnected
func (wsc *WebSocketClient) offlineCollector(metricsCh chan<- *SystemMetrics) {
	ticker := time.NewTicker(time.Duration(wsc.config.IntervalSecs) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !wsc.isConnected() && wsc.store != nil {
			// Collect metrics while offline and store with aggregation
			metrics := wsc.collector.Collect()
			if err := wsc.store.StoreWithAggregation(&metrics); err != nil {
				log.Printf("Failed to store offline metrics: %v", err)
			} else {
				pending := wsc.store.GetPendingCount()
				if pending%10 == 0 { // Log every 10 metrics
					log.Printf("Stored offline metrics (pending: %d)", pending)
				}
			}
		}
	}
}

func (wsc *WebSocketClient) connectAndRun(offlineMetricsCh chan<- *SystemMetrics) error {
	wsURL := wsc.config.WSUrl()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	log.Println("Connected to WebSocket server")

	// Send authentication message
	authMsg := AuthMessage{
		Type:     "auth",
		ServerID: wsc.config.ServerID,
		Token:    wsc.config.AgentToken,
		Version:  AgentVersion,
	}

	authData, err := json.Marshal(authMsg)
	if err != nil {
		return fmt.Errorf("failed to serialize auth message: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, authData); err != nil {
		return fmt.Errorf("failed to send auth message: %w", err)
	}

	log.Println("Sent authentication message")

	// Wait for auth response
	conn.SetReadDeadline(time.Now().Add(AuthTimeout))
	_, message, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to receive auth response: %w", err)
	}

	var response ServerResponse
	if err := json.Unmarshal(message, &response); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}

	if response.Status != "ok" {
		return fmt.Errorf("authentication failed: %s", response.Message)
	}

	// Update ping targets from server config if provided
	if len(response.PingTargets) > 0 {
		log.Printf("Received %d ping targets from server", len(response.PingTargets))
		wsc.collector.SetPingTargets(response.PingTargets)
	}

	// Store last seen timestamp from server (for deduplication)
	if response.LastSeen != nil {
		log.Printf("Server last seen timestamp: %s", *response.LastSeen)
	}
	
	// Store last buckets for resumable sync
	var lastBuckets map[string]int64
	if len(response.LastBuckets) > 0 {
		lastBuckets = response.LastBuckets
		log.Printf("Server last buckets: %v", lastBuckets)
	}

	log.Println("Authentication successful!")

	// Reset read deadline
	conn.SetReadDeadline(time.Time{})

	// Mark as connected
	wsc.setConnected(true)

	// Sync missing data since last server checkpoint
	go wsc.syncMissingData(conn, lastBuckets)
	
	// Sync offline data if any
	go wsc.syncOfflineData(conn)

	// Start metrics sending loop
	metricsTicker := time.NewTicker(time.Duration(wsc.config.IntervalSecs) * time.Second)
	defer metricsTicker.Stop()

	pingTicker := time.NewTicker(PingInterval)
	defer pingTicker.Stop()

	// Aggregation sync ticker (send aggregated data periodically)
	aggSyncTicker := time.NewTicker(AggregationSyncInterval)
	defer aggSyncTicker.Stop()

	// Handle incoming messages
	done := make(chan error, 1)
	batchAckCh := make(chan *ServerResponse, 10)

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				done <- err
				return
			}

			var response ServerResponse
			if err := json.Unmarshal(message, &response); err != nil {
				continue
			}

			switch response.Type {
			case "error":
				log.Printf("Server error: %s", response.Message)
			case "batch_ack":
				// Handle batch acknowledgment
				select {
				case batchAckCh <- &response:
				default:
				}
			case "command":
				if response.Command == "update" {
					if response.Force {
						log.Println("Received FORCE update command from server")
					} else {
						log.Println("Received update command from server")
					}
					wsc.handleUpdateCommand(response.DownloadURL, response.Force)
				}
			case "config":
				// Handle runtime config update (e.g., ping targets)
				if len(response.PingTargets) > 0 {
					log.Printf("Received updated ping targets from server: %d targets", len(response.PingTargets))
					wsc.collector.SetPingTargets(response.PingTargets)
				} else {
					log.Println("Received config update: clearing ping targets")
					wsc.collector.SetPingTargets(nil)
				}
			}
		}
	}()

	for {
		select {
		case <-metricsTicker.C:
			metrics := wsc.collector.Collect()
			
			// Store metrics with aggregation locally
			if wsc.store != nil {
				wsc.store.StoreWithAggregation(&metrics)
			}
			
			msg := MetricsMessage{
				Type:    "metrics",
				Metrics: metrics,
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to serialize metrics: %v", err)
				continue
			}

			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return fmt.Errorf("failed to send metrics: %w", err)
			}
			wsc.lastSentTime = time.Now()

		case <-aggSyncTicker.C:
			// Periodically send aggregated data to server
			wsc.sendAggregatedData(conn)

		case <-pingTicker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return fmt.Errorf("failed to send ping: %w", err)
			}

		case err := <-done:
			return err
		}
	}
}

// sendAggregatedData sends all aggregated data to the server
func (wsc *WebSocketClient) sendAggregatedData(conn *websocket.Conn) {
	if wsc.store == nil {
		return
	}

	aggData, err := wsc.store.GetAllAggregatedData()
	if err != nil {
		log.Printf("Failed to get aggregated data: %v", err)
		return
	}

	if aggData == nil || len(aggData.Granularities) == 0 {
		return
	}

	data, err := json.Marshal(aggData)
	if err != nil {
		log.Printf("Failed to serialize aggregated data: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("Failed to send aggregated data: %v", err)
	}
}

// syncMissingData syncs data that server doesn't have yet (resumable sync)
func (wsc *WebSocketClient) syncMissingData(conn *websocket.Conn, lastBuckets map[string]int64) {
	if wsc.store == nil {
		return
	}
	
	// If no last buckets info, just do a full sync
	if len(lastBuckets) == 0 {
		log.Println("No server checkpoint, sending full aggregated data...")
		wsc.sendAggregatedData(conn)
		return
	}
	
	log.Println("Syncing missing data since server checkpoint...")
	
	// Get data since the server's last known buckets
	result, err := wsc.store.GetAggregatedDataSince(lastBuckets)
	if err != nil {
		log.Printf("Failed to get missing data: %v", err)
		return
	}
	
	if result == nil || len(result.Granularities) == 0 {
		log.Println("No missing data to sync")
		return
	}
	
	// Count total buckets
	totalBuckets := 0
	for _, g := range result.Granularities {
		totalBuckets += len(g.Metrics)
	}
	
	if totalBuckets == 0 {
		log.Println("No missing data to sync")
		return
	}
	
	log.Printf("Syncing %d missing buckets across %d granularities...", totalBuckets, len(result.Granularities))
	
	data, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to serialize missing data: %v", err)
		return
	}
	
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("Failed to send missing data: %v", err)
		return
	}
	
	log.Println("Missing data sync complete")
}

// syncOfflineData sends buffered offline data to the server
func (wsc *WebSocketClient) syncOfflineData(conn *websocket.Conn) {
	if wsc.store == nil {
		return
	}

	// First, aggregate any old raw metrics
	wsc.store.AggregateOldMetrics()

	pendingCount := wsc.store.GetPendingCount()
	if pendingCount == 0 {
		return
	}

	log.Printf("Syncing %d offline metrics to server...", pendingCount)

	// Send in batches
	batchSize := wsc.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for wsc.isConnected() {
		// Get pending metrics
		stored, err := wsc.store.GetPendingMetrics(batchSize)
		if err != nil || len(stored) == 0 {
			break
		}

		// Also get aggregated metrics
		aggregated, aggIDs, err := wsc.store.GetPendingAggregated(batchSize)
		if err != nil {
			log.Printf("Failed to get aggregated metrics: %v", err)
		}

		// Build batch message
		batchID := uuid.New().String()
		batch := BatchMetricsMessage{
			Type:    "batch_metrics",
			BatchID: batchID,
		}

		var rawIDs []int64
		for _, s := range stored {
			if !s.Aggregated && s.Metrics != nil {
				// Set the timestamp on the metrics
				s.Metrics.Timestamp = s.Timestamp
				batch.Metrics = append(batch.Metrics, TimestampedMetrics{
					Timestamp: s.Timestamp.Format(time.RFC3339Nano),
					Metrics:   s.Metrics,
				})
				rawIDs = append(rawIDs, s.ID)
			}
		}

		// Convert local AggregatedMetrics to common type
		for _, agg := range aggregated {
			commonAgg := &CommonAggregatedMetrics{
				StartTime:      agg.StartTime.Format(time.RFC3339Nano),
				EndTime:        agg.EndTime.Format(time.RFC3339Nano),
				SampleCount:    agg.SampleCount,
				CPUAvg:         agg.CPUAvg,
				CPUMax:         agg.CPUMax,
				MemoryAvg:      agg.MemoryAvg,
				MemoryMax:      agg.MemoryMax,
				DiskAvg:        agg.DiskAvg,
				DiskMax:        agg.DiskMax,
				NetRxMax:       agg.NetRxMax,
				NetTxMax:       agg.NetTxMax,
				LoadOneAvg:     agg.LoadOneAvg,
				LoadFiveAvg:    agg.LoadFiveAvg,
				LoadFifteenAvg: agg.LoadFifteenAvg,
				UptimeMax:      agg.UptimeMax,
				LastMetrics:    agg.LastMetrics,
			}
			batch.Aggregated = append(batch.Aggregated, commonAgg)
		}

		if len(batch.Metrics) == 0 && len(batch.Aggregated) == 0 {
			break
		}

		// Send batch
		data, err := json.Marshal(batch)
		if err != nil {
			log.Printf("Failed to serialize batch: %v", err)
			break
		}

		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Failed to send batch: %v", err)
			break
		}

		log.Printf("Sent batch %s with %d raw metrics and %d aggregated metrics",
			batchID, len(batch.Metrics), len(batch.Aggregated))

		// Delete sent metrics (optimistic - assume server will accept)
		// In production, you might want to wait for batch_ack
		allIDs := append(rawIDs, aggIDs...)
		if err := wsc.store.DeleteByIDs(allIDs); err != nil {
			log.Printf("Failed to delete sent metrics: %v", err)
		}

		// Small delay between batches
		time.Sleep(100 * time.Millisecond)
	}

	remaining := wsc.store.GetPendingCount()
	if remaining > 0 {
		log.Printf("Sync incomplete: %d metrics remaining", remaining)
	} else {
		log.Println("Offline sync complete")
	}
}

func (wsc *WebSocketClient) handleUpdateCommand(downloadURL string, force bool) {
	if force {
		log.Println("Starting FORCE self-update process (will update regardless of version)...")
	} else {
		log.Println("Starting self-update process...")
	}

	// Get the current executable path
	currentExe, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get current executable path: %v", err)
		return
	}

	// Determine download URL and check version
	url := downloadURL
	var latestVersion string
	if url == "" {
		// Build GitHub Releases URL based on OS and architecture
		osName := runtime.GOOS
		arch := runtime.GOARCH
		
		// Map Go architecture names to release naming
		if arch == "amd64" {
			arch = "amd64"
		} else if arch == "arm64" {
			arch = "arm64"
		} else if arch == "386" {
			arch = "386"
		}
		
		// Determine binary name
		binaryName := fmt.Sprintf("vstats-agent-%s-%s", osName, arch)
		if osName == "windows" {
			binaryName += ".exe"
		}
		
		// Try to get latest version from GitHub API
		latestVersion = "latest"
		if latest, err := fetchLatestGitHubVersion("zsai001", "vstats"); err == nil && latest != nil {
			latestVersion = *latest
			
			// Skip update if already on latest version (unless force is true)
			// Compare versions without 'v' prefix
			latestVersionClean := strings.TrimPrefix(latestVersion, "v")
			currentVersionClean := strings.TrimPrefix(AgentVersion, "v")
			if !force && latestVersionClean == currentVersionClean {
				log.Printf("Already on latest version %s, skipping update", AgentVersion)
				return
			}
			log.Printf("Update available: current=%s, latest=%s", AgentVersion, latestVersion)
		}
		
		// Build GitHub Releases download URL
		url = fmt.Sprintf("https://github.com/zsai001/vstats/releases/download/%s/%s", latestVersion, binaryName)
		log.Printf("No download URL provided, using GitHub Releases: %s", url)
	} else {
		log.Printf("Using provided download URL: %s", url)
	}
	
	if force {
		log.Printf("Force update enabled, current version: %s", AgentVersion)
	}

	log.Printf("Downloading update from: %s", url)

	// Download to a temporary file
	tempPath := currentExe + ".new"

	if err := downloadFile(url, tempPath); err != nil {
		log.Printf("Failed to download update: %v", err)
		return
	}

	log.Println("Download complete, applying update...")

	// On Unix, set execute permissions
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempPath, 0755); err != nil {
			log.Printf("Failed to set permissions: %v", err)
			os.Remove(tempPath)
			return
		}
	}

	// Backup current executable
	backupPath := currentExe + ".backup"
	if err := os.Rename(currentExe, backupPath); err != nil {
		log.Printf("Failed to backup current executable: %v", err)
		os.Remove(tempPath)
		return
	}

	// Move new executable to current path
	if err := os.Rename(tempPath, currentExe); err != nil {
		log.Printf("Failed to install new executable: %v", err)
		// Try to restore backup
		os.Rename(backupPath, currentExe)
		return
	}

	// Remove backup
	os.Remove(backupPath)

	log.Println("Update installed successfully! Restarting...")

	// Restart the agent using systemd-run to avoid being killed by cgroup
	if runtime.GOOS == "linux" {
		// Use systemd-run --no-block to run restart in an independent transient unit
		// This prevents the restart command from being killed when vstats-agent stops
		cmd := exec.Command("systemd-run", "--no-block", "systemctl", "restart", "vstats-agent")
		if err := cmd.Start(); err != nil {
			log.Printf("Failed to schedule restart via systemd-run: %v", err)
			// Fallback to direct systemctl (may not work in all cases)
			exec.Command("systemctl", "restart", "vstats-agent").Start()
		} else {
			log.Println("Restart scheduled via systemd-run")
		}
	} else if runtime.GOOS == "windows" {
		// On Windows, use sc.exe to restart the service
		cmd := exec.Command("cmd", "/C", "sc", "stop", "vstats-agent", "&&", "timeout", "/t", "2", "&&", "sc", "start", "vstats-agent")
		cmd.Start()
	}

	// Give systemd-run a moment to register the restart command
	time.Sleep(500 * time.Millisecond)

	// Exit to allow restart
	os.Exit(0)
}

// downloadFile downloads a file from URL to path
func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(path)
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// fetchLatestGitHubVersion fetches the latest release version from GitHub
func fetchLatestGitHubVersion(owner, repo string) (*string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "vstats-agent")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	tagName, ok := result["tag_name"].(string)
	if !ok {
		return nil, fmt.Errorf("no tag_name in response")
	}

	// Keep the original tag name (with 'v' prefix) for download URL
	return &tagName, nil
}
