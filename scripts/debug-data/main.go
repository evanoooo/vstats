package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

// =============================================================================
// Types (matching server/agent types)
// =============================================================================

type SystemMetrics struct {
	Timestamp   time.Time      `json:"timestamp"`
	Hostname    string         `json:"hostname"`
	OS          OsInfo         `json:"os"`
	CPU         CpuMetrics     `json:"cpu"`
	Memory      MemoryMetrics  `json:"memory"`
	Disks       []DiskMetrics  `json:"disks"`
	Network     NetworkMetrics `json:"network"`
	Uptime      uint64         `json:"uptime"`
	LoadAverage LoadAverage    `json:"load_average"`
	Ping        *PingMetrics   `json:"ping,omitempty"`
	Version     string         `json:"version,omitempty"`
	IPAddresses []string       `json:"ip_addresses,omitempty"`
}

type OsInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Kernel  string `json:"kernel"`
	Arch    string `json:"arch"`
}

type CpuMetrics struct {
	Brand     string    `json:"brand"`
	Cores     int       `json:"cores"`
	Usage     float32   `json:"usage"`
	Frequency uint64    `json:"frequency"`
	PerCore   []float32 `json:"per_core"`
}

type MemoryMetrics struct {
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Available    uint64  `json:"available"`
	SwapTotal    uint64  `json:"swap_total"`
	SwapUsed     uint64  `json:"swap_used"`
	UsagePercent float32 `json:"usage_percent"`
}

type DiskMetrics struct {
	Name         string   `json:"name"`
	Model        string   `json:"model,omitempty"`
	Total        uint64   `json:"total"`
	DiskType     string   `json:"disk_type,omitempty"`
	MountPoints  []string `json:"mount_points,omitempty"`
	UsagePercent float32  `json:"usage_percent"`
	Used         uint64   `json:"used"`
	ReadSpeed    uint64   `json:"read_speed,omitempty"`
	WriteSpeed   uint64   `json:"write_speed,omitempty"`
}

type NetworkMetrics struct {
	Interfaces []NetworkInterface `json:"interfaces"`
	TotalRx    uint64             `json:"total_rx"`
	TotalTx    uint64             `json:"total_tx"`
	RxSpeed    uint64             `json:"rx_speed"`
	TxSpeed    uint64             `json:"tx_speed"`
	DailyRx    uint64             `json:"daily_rx,omitempty"`
	DailyTx    uint64             `json:"daily_tx,omitempty"`
}

type NetworkInterface struct {
	Name      string `json:"name"`
	MAC       string `json:"mac,omitempty"`
	Speed     uint32 `json:"speed,omitempty"`
	RxBytes   uint64 `json:"rx_bytes"`
	TxBytes   uint64 `json:"tx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	TxPackets uint64 `json:"tx_packets"`
}

type LoadAverage struct {
	One     float64 `json:"one"`
	Five    float64 `json:"five"`
	Fifteen float64 `json:"fifteen"`
}

type PingMetrics struct {
	Targets []PingTarget `json:"targets"`
}

type PingTarget struct {
	Name       string   `json:"name"`
	Host       string   `json:"host"`
	Type       string   `json:"type,omitempty"`
	Port       int      `json:"port,omitempty"`
	LatencyMs  *float64 `json:"latency_ms"`
	PacketLoss float64  `json:"packet_loss"`
	Status     string   `json:"status"`
}

// =============================================================================
// Multi-Granularity Aggregation Types (new agent protocol)
// =============================================================================

// BucketData represents a single aggregated data bucket
type BucketData struct {
	Bucket      int64   `json:"bucket"`
	CPUSum      float64 `json:"cpu_sum"`
	CPUMax      float64 `json:"cpu_max"`
	MemorySum   float64 `json:"memory_sum"`
	MemoryMax   float64 `json:"memory_max"`
	DiskSum     float64 `json:"disk_sum"`
	NetRx       uint64  `json:"net_rx"`
	NetTx       uint64  `json:"net_tx"`
	PingSum     float64 `json:"ping_sum"`
	PingCount   int     `json:"ping_count"`
	SampleCount int     `json:"sample_count"`
}

// PingBucketData represents ping metrics for a specific target in a bucket
type PingBucketData struct {
	Bucket       int64   `json:"bucket"`
	TargetName   string  `json:"target_name"`
	TargetHost   string  `json:"target_host"`
	LatencySum   float64 `json:"latency_sum"`
	LatencyMax   float64 `json:"latency_max"`
	LatencyCount int     `json:"latency_count"`
	OkCount      int     `json:"ok_count"`
	FailCount    int     `json:"fail_count"`
}

// GranularityData contains aggregated data for a specific time granularity
type GranularityData struct {
	Granularity string           `json:"granularity"`
	Interval    int              `json:"interval"`
	Metrics     []BucketData     `json:"metrics"`
	Ping        []PingBucketData `json:"ping,omitempty"`
}

// MultiGranularityMetrics contains aggregated data at multiple granularities
type MultiGranularityMetrics struct {
	Type          string            `json:"type"`
	Granularities []GranularityData `json:"granularities"`
	LastMetrics   *SystemMetrics    `json:"last_metrics,omitempty"`
}

// Granularity constants
const (
	Granularity5Sec   = 5
	Granularity2Min   = 120
	Granularity15Min  = 900
	GranularityHourly = 3600
	GranularityDaily  = 86400
)

type AuthMessage struct {
	Type     string `json:"type"`
	ServerID string `json:"server_id"`
	Token    string `json:"token"`
	Version  string `json:"version"`
}

type MetricsMessage struct {
	Type    string        `json:"type"`
	Metrics SystemMetrics `json:"metrics"`
}

type ServerResponse struct {
	Type        string           `json:"type"`
	Status      string           `json:"status"`
	Message     string           `json:"message,omitempty"`
	LastBuckets map[string]int64 `json:"last_buckets,omitempty"`
}

// RemoteServer represents a server config (response from API)
type RemoteServer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Provider string `json:"provider"`
	Token    string `json:"token"`
}

// AddServerRequest represents the request to add a server
type AddServerRequest struct {
	Name     string `json:"name"`
	Location string `json:"location"`
	Provider string `json:"provider"`
}

// AppConfig represents the config file
type AppConfig struct {
	Servers []RemoteServer `json:"servers"`
}

// =============================================================================
// Data Generators
// =============================================================================

var (
	cpuBrands = []string{
		"Intel Core i9-12900K",
		"AMD Ryzen 9 5950X",
		"Intel Xeon E5-2680",
		"AMD EPYC 7763",
		"Apple M1 Pro",
		"Intel Core i7-11700K",
		"AMD Ryzen 7 5800X",
	}

	osNames = []string{
		"Ubuntu", "Debian", "CentOS",
	}

	locations = []string{
		"🇺🇸 Los Angeles", "🇺🇸 New York", "🇺🇸 Seattle", "🇺🇸 Dallas",
		"🇩🇪 Frankfurt", "🇬🇧 London", "🇫🇷 Paris", "🇳🇱 Amsterdam",
		"🇯🇵 Tokyo", "🇸🇬 Singapore", "🇭🇰 Hong Kong", "🇰🇷 Seoul",
		"🇦🇺 Sydney", "🇨🇦 Toronto", "🇧🇷 São Paulo",
	}

	providers = []string{
		"AWS", "Google Cloud", "Azure",
	}

	pingTargets = []struct {
		Name string
		Host string
	}{
		{"Cloudflare", "1.1.1.1"},
		{"Google DNS", "8.8.8.8"},
		{"OpenDNS", "208.67.222.222"},
		{"Quad9", "9.9.9.9"},
		{"Google", "google.com"},
	}
)

// SimulatedServer holds state for a simulated server
type SimulatedServer struct {
	ID        string
	Name      string
	Token     string
	Location  string
	Provider  string
	OSName    string
	OSVersion string
	CPUBrand  string
	CPUCores  int
	MemTotal  uint64
	DiskTotal uint64
	Uptime    uint64

	// Dynamic state
	cpuBase     float32
	memBase     float32
	diskBase    float32
	networkBase uint64
}

func NewSimulatedServer(index int) *SimulatedServer {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(index)))

	cores := []int{2, 4, 8, 16, 32, 64}[r.Intn(6)]
	memGB := []uint64{2, 4, 8, 16, 32, 64, 128}[r.Intn(7)]
	diskGB := []uint64{40, 80, 160, 320, 500, 1000, 2000}[r.Intn(7)]

	return &SimulatedServer{
		ID:          fmt.Sprintf("debug-server-%03d", index),
		Name:        fmt.Sprintf("Server-%03d", index),
		Token:       fmt.Sprintf("debug-token-%03d", index),
		Location:    locations[r.Intn(len(locations))],
		Provider:    providers[r.Intn(len(providers))],
		OSName:      osNames[r.Intn(len(osNames))],
		OSVersion:   fmt.Sprintf("%d.%d", r.Intn(10)+18, r.Intn(10)),
		CPUBrand:    cpuBrands[r.Intn(len(cpuBrands))],
		CPUCores:    cores,
		MemTotal:    memGB * 1024 * 1024 * 1024,
		DiskTotal:   diskGB * 1024 * 1024 * 1024,
		Uptime:      uint64(r.Intn(365*24*3600) + 3600),
		cpuBase:     float32(r.Intn(40) + 10),
		memBase:     float32(r.Intn(40) + 20),
		diskBase:    float32(r.Intn(60) + 10),
		networkBase: uint64(r.Intn(1000000000) + 1000000),
	}
}

func (s *SimulatedServer) GenerateMetrics(timestamp time.Time) SystemMetrics {
	r := rand.New(rand.NewSource(timestamp.UnixNano() + int64(len(s.ID))))

	// CPU with some variation
	cpuUsage := s.cpuBase + float32(r.Intn(30)) - 10
	if cpuUsage < 0 {
		cpuUsage = 0
	}
	if cpuUsage > 100 {
		cpuUsage = 100
	}

	perCore := make([]float32, s.CPUCores)
	for i := range perCore {
		perCore[i] = cpuUsage + float32(r.Intn(20)) - 10
		if perCore[i] < 0 {
			perCore[i] = 0
		}
		if perCore[i] > 100 {
			perCore[i] = 100
		}
	}

	// Memory with some variation
	memUsage := s.memBase + float32(r.Intn(20)) - 5
	if memUsage < 0 {
		memUsage = 0
	}
	if memUsage > 95 {
		memUsage = 95
	}
	memUsed := uint64(float64(s.MemTotal) * float64(memUsage) / 100)

	// Disk (relatively stable)
	diskUsage := s.diskBase + float32(r.Intn(5)) - 2
	if diskUsage < 0 {
		diskUsage = 0
	}
	if diskUsage > 95 {
		diskUsage = 95
	}
	diskUsed := uint64(float64(s.DiskTotal) * float64(diskUsage) / 100)

	// Network traffic
	rxSpeed := uint64(r.Intn(100000000))  // up to 100MB/s
	txSpeed := uint64(r.Intn(50000000))   // up to 50MB/s
	totalRx := s.networkBase + uint64(timestamp.Unix())*rxSpeed/10
	totalTx := s.networkBase/2 + uint64(timestamp.Unix())*txSpeed/10

	// Generate ping results
	var pingResults *PingMetrics
	targets := make([]PingTarget, len(pingTargets))
	for i, pt := range pingTargets {
		latency := 5.0 + float64(r.Intn(200))
		status := "ok"
		packetLoss := 0.0

		// Occasionally simulate issues
		if r.Float32() < 0.05 { // 5% chance of issues
			if r.Float32() < 0.5 {
				status = "timeout"
				packetLoss = 100.0
				latency = 0
			} else {
				packetLoss = float64(r.Intn(30) + 10)
			}
		}

		latencyPtr := &latency
		if status == "timeout" {
			latencyPtr = nil
		}

		targets[i] = PingTarget{
			Name:       pt.Name,
			Host:       pt.Host,
			Type:       "icmp",
			LatencyMs:  latencyPtr,
			PacketLoss: packetLoss,
			Status:     status,
		}
	}
	pingResults = &PingMetrics{Targets: targets}

	return SystemMetrics{
		Timestamp: timestamp,
		Hostname:  s.Name,
		OS: OsInfo{
			Name:    s.OSName,
			Version: s.OSVersion,
			Kernel:  "5.15.0-generic",
			Arch:    "amd64",
		},
		CPU: CpuMetrics{
			Brand:     s.CPUBrand,
			Cores:     s.CPUCores,
			Usage:     cpuUsage,
			Frequency: 3600,
			PerCore:   perCore,
		},
		Memory: MemoryMetrics{
			Total:        s.MemTotal,
			Used:         memUsed,
			Available:    s.MemTotal - memUsed,
			SwapTotal:    s.MemTotal / 2,
			SwapUsed:     0,
			UsagePercent: memUsage,
		},
		Disks: []DiskMetrics{
			{
				Name:         "sda",
				Model:        "Samsung SSD 980",
				Total:        s.DiskTotal,
				DiskType:     "SSD",
				MountPoints:  []string{"/"},
				UsagePercent: diskUsage,
				Used:         diskUsed,
				ReadSpeed:    uint64(r.Intn(500000000)),
				WriteSpeed:   uint64(r.Intn(300000000)),
			},
		},
		Network: NetworkMetrics{
			Interfaces: []NetworkInterface{
				{
					Name:      "eth0",
					MAC:       fmt.Sprintf("00:00:00:%02x:%02x:%02x", r.Intn(256), r.Intn(256), r.Intn(256)),
					Speed:     1000,
					RxBytes:   totalRx,
					TxBytes:   totalTx,
					RxPackets: totalRx / 1500,
					TxPackets: totalTx / 1500,
				},
			},
			TotalRx: totalRx,
			TotalTx: totalTx,
			RxSpeed: rxSpeed,
			TxSpeed: txSpeed,
			DailyRx: totalRx / 10,
			DailyTx: totalTx / 10,
		},
		Uptime: s.Uptime + uint64(time.Since(timestamp.Add(-time.Hour*2)).Seconds()),
		LoadAverage: LoadAverage{
			One:     float64(cpuUsage) / 25,
			Five:    float64(cpuUsage) / 30,
			Fifteen: float64(cpuUsage) / 40,
		},
		Ping:        pingResults,
		Version:     "v0.0.0-debug",
		IPAddresses: []string{fmt.Sprintf("10.0.%d.%d", r.Intn(256), r.Intn(256))},
	}
}

// GenerateAggregatedData generates multi-granularity aggregated data for a time range
func (s *SimulatedServer) GenerateAggregatedData(startTime, endTime time.Time) MultiGranularityMetrics {
	granularities := []struct {
		name     string
		interval int
	}{
		{"5sec", Granularity5Sec},
		{"2min", Granularity2Min},
		{"15min", Granularity15Min},
		{"hourly", GranularityHourly},
		{"daily", GranularityDaily},
	}

	result := MultiGranularityMetrics{
		Type:          "aggregated_metrics",
		Granularities: make([]GranularityData, 0, len(granularities)),
	}

	// Generate latest metrics
	lastMetrics := s.GenerateMetrics(endTime)
	result.LastMetrics = &lastMetrics

	for _, g := range granularities {
		gd := GranularityData{
			Granularity: g.name,
			Interval:    g.interval,
			Metrics:     make([]BucketData, 0),
			Ping:        make([]PingBucketData, 0),
		}

		// Calculate bucket range
		startBucket := (startTime.Unix() / int64(g.interval)) * int64(g.interval)
		endBucket := (endTime.Unix() / int64(g.interval)) * int64(g.interval)

		for bucket := startBucket; bucket <= endBucket; bucket += int64(g.interval) {
			// Generate aggregated metrics for this bucket
			bucketTime := time.Unix(bucket, 0)
			r := rand.New(rand.NewSource(bucket + int64(len(s.ID))))

			// Samples in bucket depends on granularity
			samplesPerBucket := g.interval / 5 // assuming 5s collection interval
			if samplesPerBucket < 1 {
				samplesPerBucket = 1
			}

			cpuSum := 0.0
			cpuMax := 0.0
			memSum := 0.0
			memMax := 0.0
			diskSum := 0.0
			pingSum := 0.0
			pingCount := 0

			for i := 0; i < samplesPerBucket; i++ {
				cpu := float64(s.cpuBase) + float64(r.Intn(30)) - 10
				if cpu < 0 {
					cpu = 0
				}
				if cpu > 100 {
					cpu = 100
				}
				cpuSum += cpu
				if cpu > cpuMax {
					cpuMax = cpu
				}

				mem := float64(s.memBase) + float64(r.Intn(20)) - 5
				if mem < 0 {
					mem = 0
				}
				if mem > 95 {
					mem = 95
				}
				memSum += mem
				if mem > memMax {
					memMax = mem
				}

				disk := float64(s.diskBase) + float64(r.Intn(5)) - 2
				if disk < 0 {
					disk = 0
				}
				if disk > 95 {
					disk = 95
				}
				diskSum += disk

				// Ping
				latency := 5.0 + float64(r.Intn(200))
				pingSum += latency
				pingCount++
			}

			// Network (cumulative at bucket end)
			netRx := s.networkBase + uint64(bucket)*uint64(r.Intn(1000000))
			netTx := s.networkBase/2 + uint64(bucket)*uint64(r.Intn(500000))

			bd := BucketData{
				Bucket:      bucket,
				CPUSum:      cpuSum,
				CPUMax:      cpuMax,
				MemorySum:   memSum,
				MemoryMax:   memMax,
				DiskSum:     diskSum,
				NetRx:       netRx,
				NetTx:       netTx,
				PingSum:     pingSum,
				PingCount:   pingCount,
				SampleCount: samplesPerBucket,
			}
			gd.Metrics = append(gd.Metrics, bd)

			// Generate ping buckets for each target
			for _, pt := range pingTargets {
				latencySum := 0.0
				latencyMax := 0.0
				okCount := 0
				failCount := 0

				for i := 0; i < samplesPerBucket; i++ {
					if r.Float32() < 0.95 { // 95% success
						latency := 5.0 + float64(r.Intn(200))
						latencySum += latency
						if latency > latencyMax {
							latencyMax = latency
						}
						okCount++
					} else {
						failCount++
					}
				}

				pbd := PingBucketData{
					Bucket:       bucket,
					TargetName:   pt.Name,
					TargetHost:   pt.Host,
					LatencySum:   latencySum,
					LatencyMax:   latencyMax,
					LatencyCount: okCount,
					OkCount:      okCount,
					FailCount:    failCount,
				}
				gd.Ping = append(gd.Ping, pbd)
			}

			_ = bucketTime // silence unused warning
		}

		result.Granularities = append(result.Granularities, gd)
	}

	return result
}

// =============================================================================
// Main Functions
// =============================================================================

// promptForToken prompts the user to enter the admin token securely
func promptForToken() string {
	fmt.Print("🔑 Enter admin token: ")

	// Try to read password securely (hidden input)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // New line after hidden input
		if err != nil {
			log.Printf("⚠️  Failed to read secure input: %v", err)
			return ""
		}
		return strings.TrimSpace(string(password))
	}

	// Fallback to regular input (for piped input)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("⚠️  Failed to read input: %v", err)
		return ""
	}
	return strings.TrimSpace(input)
}

func main() {
	serverCount := flag.Int("count", 100, "Number of simulated servers")
	serverURL := flag.String("server", "http://localhost:3001", "Server URL")
	historyHours := flag.Float64("hours", 2.0, "Hours of history data to generate")
	mode := flag.String("mode", "both", "Mode: history, realtime, or both")
	adminToken := flag.String("token", "", "Admin token for API authentication (will prompt if not provided)")
	interval := flag.Int("interval", 3, "Metrics reporting interval in seconds (for realtime mode)")
	aggInterval := flag.Int("agg-interval", 60, "Aggregated metrics sync interval in seconds (for realtime mode)")
	cleanup := flag.Bool("cleanup", false, "Remove all debug servers and exit")

	flag.Parse()

	// Prompt for token if not provided
	token := *adminToken
	if token == "" {
		token = promptForToken()
	}
	if token == "" {
		log.Fatal("❌ Admin token is required")
	}

	log.Printf("🚀 vStats Debug Data Generator (New Protocol)")
	log.Printf("   Server URL: %s", *serverURL)
	log.Printf("   Server Count: %d", *serverCount)
	log.Printf("   Mode: %s", *mode)

	// Cleanup mode
	if *cleanup {
		log.Printf("🧹 Cleaning up debug servers...")
		cleanupDebugServers(*serverURL, token)
		return
	}

	// First, cleanup existing debug servers
	log.Printf("🧹 Cleaning up existing debug servers...")
	cleanupDebugServers(*serverURL, token)

	// Register servers via API
	log.Printf("📝 Registering %d debug servers via API...", *serverCount)
	servers := registerServersViaAPI(*serverURL, token, *serverCount)
	if len(servers) == 0 {
		log.Fatal("❌ Failed to register any servers")
	}
	log.Printf("✅ Registered %d servers", len(servers))

	switch *mode {
	case "history":
		generateHistoryData(servers, *serverURL, *historyHours)
	case "realtime":
		runRealtimeAgents(servers, *serverURL, *interval, *aggInterval)
	case "both":
		generateHistoryData(servers, *serverURL, *historyHours)
		runRealtimeAgents(servers, *serverURL, *interval, *aggInterval)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

// registerServersViaAPI registers debug servers using the HTTP API (concurrent)
func registerServersViaAPI(serverURL, adminToken string, count int) []*SimulatedServer {
	client := &http.Client{Timeout: 10 * time.Second}

	var wg sync.WaitGroup
	var mu sync.Mutex
	servers := make([]*SimulatedServer, 0, count)
	var registeredCount int32

	sem := make(chan struct{}, 100) // 100 concurrent registrations

	for i := 0; i < count; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			// Create simulated server template
			sim := NewSimulatedServer(idx)

			// Register via API
			reqBody := AddServerRequest{
				Name:     fmt.Sprintf("Debug-Server-%03d", idx),
				Location: sim.Location,
				Provider: sim.Provider,
			}

			bodyBytes, _ := json.Marshal(reqBody)
			req, err := http.NewRequest("POST", serverURL+"/api/servers", bytes.NewReader(bodyBytes))
			if err != nil {
				log.Printf("❌ Failed to create request for server %d: %v", idx, err)
				return
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+adminToken)

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("❌ Failed to register server %d: %v", idx, err)
				return
			}

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				log.Printf("❌ Failed to register server %d: %s - %s", idx, resp.Status, string(body))
				return
			}

			var registered RemoteServer
			if err := json.NewDecoder(resp.Body).Decode(&registered); err != nil {
				resp.Body.Close()
				log.Printf("❌ Failed to decode response for server %d: %v", idx, err)
				return
			}
			resp.Body.Close()

			// Update simulated server with real ID and Token
			sim.ID = registered.ID
			sim.Token = registered.Token
			sim.Name = registered.Name

			mu.Lock()
			servers = append(servers, sim)
			mu.Unlock()

			newCount := atomic.AddInt32(&registeredCount, 1)
			if newCount%20 == 0 {
				log.Printf("   Registered %d/%d servers...", newCount, count)
			}
		}(i)
	}

	wg.Wait()
	return servers
}

// cleanupDebugServers removes all debug servers (concurrent)
func cleanupDebugServers(serverURL, adminToken string) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Get all servers
	req, _ := http.NewRequest("GET", serverURL+"/api/servers", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("⚠️  Failed to get servers: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("⚠️  Failed to get servers: %s", resp.Status)
		return
	}

	var servers []RemoteServer
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		log.Printf("⚠️  Failed to decode servers: %v", err)
		return
	}

	// Find debug servers to delete
	var toDelete []string
	for _, srv := range servers {
		if len(srv.Name) >= 13 && srv.Name[:13] == "Debug-Server-" {
			toDelete = append(toDelete, srv.ID)
		}
	}

	if len(toDelete) == 0 {
		return
	}

	// Delete concurrently with semaphore
	var wg sync.WaitGroup
	var deletedCount int32
	sem := make(chan struct{}, 20) // 20 concurrent deletions

	for _, id := range toDelete {
		wg.Add(1)
		sem <- struct{}{}
		go func(serverID string) {
			defer wg.Done()
			defer func() { <-sem }()

			delReq, _ := http.NewRequest("DELETE", serverURL+"/api/servers/"+serverID, nil)
			delReq.Header.Set("Authorization", "Bearer "+adminToken)

			delResp, err := client.Do(delReq)
			if err != nil {
				return
			}
			delResp.Body.Close()

			if delResp.StatusCode == http.StatusOK {
				atomic.AddInt32(&deletedCount, 1)
			}
		}(id)
	}

	wg.Wait()

	if deletedCount > 0 {
		log.Printf("   Deleted %d existing debug servers", deletedCount)
	}
}

func generateHistoryData(servers []*SimulatedServer, serverURL string, hours float64) {
	log.Printf("📊 Generating %.1f hours of history data for %d servers (new protocol)...", hours, len(servers))

	// Calculate time range
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(hours * float64(time.Hour)))

	log.Printf("   Time range: %s to %s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	// Parse WebSocket URL
	u, err := url.Parse(serverURL)
	if err != nil {
		log.Fatalf("Invalid server URL: %v", err)
	}

	wsScheme := "ws"
	if u.Scheme == "https" {
		wsScheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/ws/agent", wsScheme, u.Host)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // Limit concurrent connections

	for _, server := range servers {
		wg.Add(1)
		sem <- struct{}{}

		go func(s *SimulatedServer) {
			defer wg.Done()
			defer func() { <-sem }()

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				log.Printf("❌ [%s] Failed to connect: %v", s.ID, err)
				return
			}
			defer conn.Close()

			// Authenticate
			authMsg := AuthMessage{
				Type:     "auth",
				ServerID: s.ID,
				Token:    s.Token,
				Version:  "v0.0.0-debug",
			}
			if err := conn.WriteJSON(authMsg); err != nil {
				log.Printf("❌ [%s] Auth failed: %v", s.ID, err)
				return
			}

			// Wait for auth response
			var resp ServerResponse
			if err := conn.ReadJSON(&resp); err != nil {
				log.Printf("❌ [%s] Auth response failed: %v", s.ID, err)
				return
			}
			if resp.Status != "ok" {
				log.Printf("❌ [%s] Auth rejected: %s", s.ID, resp.Message)
				return
			}

			// Generate and send aggregated historical data
			aggData := s.GenerateAggregatedData(startTime, endTime)

			if err := conn.WriteJSON(aggData); err != nil {
				log.Printf("❌ [%s] Failed to send aggregated data: %v", s.ID, err)
				return
			}

			// Count total buckets
			totalBuckets := 0
			for _, g := range aggData.Granularities {
				totalBuckets += len(g.Metrics)
			}

			log.Printf("✅ [%s] Sent aggregated history: %d granularities, %d total buckets",
				s.ID, len(aggData.Granularities), totalBuckets)
		}(server)
	}

	wg.Wait()
	log.Printf("✅ History data generation complete!")
}

func runRealtimeAgents(servers []*SimulatedServer, serverURL string, intervalSecs, aggIntervalSecs int) {
	log.Printf("🔄 Starting %d realtime agents (metrics: %ds, aggregation: %ds)...",
		len(servers), intervalSecs, aggIntervalSecs)

	// Parse WebSocket URL
	u, err := url.Parse(serverURL)
	if err != nil {
		log.Fatalf("Invalid server URL: %v", err)
	}

	wsScheme := "ws"
	if u.Scheme == "https" {
		wsScheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/ws/agent", wsScheme, u.Host)

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan struct{})
	var wg sync.WaitGroup

	// Start agents
	for _, server := range servers {
		wg.Add(1)
		go func(s *SimulatedServer) {
			defer wg.Done()
			runSingleAgent(s, wsURL, intervalSecs, aggIntervalSecs, done)
		}(server)

		// Stagger connections
		time.Sleep(50 * time.Millisecond)
	}

	log.Printf("✅ All %d agents started! Press Ctrl+C to stop.", len(servers))

	// Wait for shutdown signal
	<-sigChan
	log.Printf("\n🛑 Shutting down agents...")
	close(done)
	wg.Wait()
	log.Printf("✅ All agents stopped")
}

func runSingleAgent(server *SimulatedServer, wsURL string, intervalSecs, aggIntervalSecs int, done chan struct{}) {
	reconnectDelay := 5 * time.Second

	for {
		select {
		case <-done:
			return
		default:
		}

		if err := agentLoop(server, wsURL, intervalSecs, aggIntervalSecs, done); err != nil {
			log.Printf("⚠️  [%s] Disconnected: %v, reconnecting in %v...", server.ID, err, reconnectDelay)
		}

		select {
		case <-done:
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func agentLoop(server *SimulatedServer, wsURL string, intervalSecs, aggIntervalSecs int, done chan struct{}) error {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer conn.Close()

	// Authenticate
	authMsg := AuthMessage{
		Type:     "auth",
		ServerID: server.ID,
		Token:    server.Token,
		Version:  "v0.0.0-debug",
	}
	if err := conn.WriteJSON(authMsg); err != nil {
		return fmt.Errorf("auth send failed: %w", err)
	}

	// Wait for auth response
	var resp ServerResponse
	if err := conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("auth response failed: %w", err)
	}
	if resp.Status != "ok" {
		return fmt.Errorf("auth rejected: %s", resp.Message)
	}

	log.Printf("✅ [%s] Connected and authenticated", server.ID)

	// Tickers for metrics and aggregation sync
	metricsTicker := time.NewTicker(time.Duration(intervalSecs) * time.Second)
	aggTicker := time.NewTicker(time.Duration(aggIntervalSecs) * time.Second)
	defer metricsTicker.Stop()
	defer aggTicker.Stop()

	// Read messages in background
	errChan := make(chan error, 1)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Track aggregation window
	lastAggSync := time.Now().Add(-time.Duration(aggIntervalSecs) * time.Second)

	for {
		select {
		case <-done:
			return nil
		case err := <-errChan:
			return err
		case <-metricsTicker.C:
			// Send real-time metrics (legacy format, still supported)
			metrics := server.GenerateMetrics(time.Now())
			msg := MetricsMessage{
				Type:    "metrics",
				Metrics: metrics,
			}
			if err := conn.WriteJSON(msg); err != nil {
				return fmt.Errorf("send metrics failed: %w", err)
			}
		case <-aggTicker.C:
			// Send aggregated metrics (new protocol)
			now := time.Now()
			aggData := server.GenerateAggregatedData(lastAggSync, now)
			if err := conn.WriteJSON(aggData); err != nil {
				return fmt.Errorf("send aggregated metrics failed: %w", err)
			}
			lastAggSync = now
		}
	}
}
