package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ============================================================================
// Dashboard WebSocket Handler
// ============================================================================

func (s *AppState) HandleDashboardWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Get client IP
	clientIP := c.ClientIP()

	// Register client with IP
	client := &DashboardClient{
		Conn: conn,
		IP:   clientIP,
	}
	s.DashboardMu.Lock()
	s.DashboardClients[conn] = client
	s.DashboardMu.Unlock()

	// Unregister on exit
	defer func() {
		s.DashboardMu.Lock()
		delete(s.DashboardClients, conn)
		s.DashboardMu.Unlock()
	}()

	// Send initial state
	s.sendInitialState(client)

	// Handle incoming messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// StreamInitMessage is sent first with metadata and server count
type StreamInitMessage struct {
	Type            string           `json:"type"`
	TotalServers    int              `json:"total_servers"`
	Groups          []ServerGroup    `json:"groups,omitempty"`
	GroupDimensions []GroupDimension `json:"group_dimensions,omitempty"`
	SiteSettings    *SiteSettings    `json:"site_settings,omitempty"`
}

// StreamServerMessage is sent for each server
type StreamServerMessage struct {
	Type   string              `json:"type"`
	Index  int                 `json:"index"`
	Total  int                 `json:"total"`
	Server ServerMetricsUpdate `json:"server"`
}

// StreamEndMessage signals the end of initial data
type StreamEndMessage struct {
	Type string `json:"type"`
}

// sendInitialState sends pre-built snapshot to new dashboard client
func (s *AppState) sendInitialState(client *DashboardClient) {
	// Helper function to write with lock
	writeMessage := func(data []byte) error {
		client.WriteMu.Lock()
		defer client.WriteMu.Unlock()
		return client.Conn.WriteMessage(websocket.TextMessage, data)
	}

	// Try to use cached snapshot first
	s.SnapshotMu.RLock()
	snapshot := s.Snapshot
	s.SnapshotMu.RUnlock()

	if snapshot != nil && time.Since(snapshot.LastUpdated) < 10*time.Second {
		// Use cached snapshot - very fast!
		if err := writeMessage(snapshot.InitMessage); err != nil {
			return
		}
		for _, serverMsg := range snapshot.ServerMessages {
			if err := writeMessage(serverMsg); err != nil {
				return
			}
		}
		writeMessage(snapshot.EndMessage)
		return
	}

	// Snapshot too old or doesn't exist, build fresh data
	s.sendInitialStateFresh(client)
}

// sendInitialStateFresh builds and sends fresh state (used when snapshot is stale)
func (s *AppState) sendInitialStateFresh(client *DashboardClient) {
	s.ConfigMu.RLock()
	config := s.Config
	s.ConfigMu.RUnlock()

	s.AgentMetricsMu.RLock()
	agentMetrics := make(map[string]*AgentMetricsData)
	for k, v := range s.AgentMetrics {
		agentMetrics[k] = v
	}
	s.AgentMetricsMu.RUnlock()

	totalServers := 1 + len(config.Servers) // local + remote

	// Helper function to write with lock
	writeMessage := func(data []byte) error {
		client.WriteMu.Lock()
		defer client.WriteMu.Unlock()
		return client.Conn.WriteMessage(websocket.TextMessage, data)
	}

	// Step 1: Send init message with metadata (fast, allows UI to prepare)
	initMsg := StreamInitMessage{
		Type:            "stream_init",
		TotalServers:    totalServers,
		Groups:          config.Groups,
		GroupDimensions: config.GroupDimensions,
		SiteSettings:    &config.SiteSettings,
	}
	initData, _ := json.Marshal(initMsg)
	if err := writeMessage(initData); err != nil {
		return
	}

	// Step 2: Stream servers one by one
	index := 0

	// Local node first (usually fastest)
	localMetrics := CollectMetrics()
	localNode := config.LocalNode
	localName := "Dashboard Server"
	if localNode.Name != "" {
		localName = localNode.Name
	}
	provider := "Local"
	if localNode.Provider != "" {
		provider = localNode.Provider
	}

	localServer := StreamServerMessage{
		Type:  "stream_server",
		Index: index,
		Total: totalServers,
		Server: ServerMetricsUpdate{
			ServerID:     "local",
			ServerName:   localName,
			Location:     localNode.Location,
			Provider:     provider,
			Tag:          localNode.Tag,
			GroupID:      localNode.GroupID,
			GroupValues:  localNode.GroupValues,
			Version:      ServerVersion,
			IP:           "",
			Online:       true,
			Metrics:      &localMetrics,
			PriceAmount:  localNode.PriceAmount,
			PricePeriod:  localNode.PricePeriod,
			PurchaseDate: localNode.PurchaseDate,
			TipBadge:     localNode.TipBadge,
		},
	}
	localData, _ := json.Marshal(localServer)
	if err := writeMessage(localData); err != nil {
		return
	}
	index++

	// Remote servers
	for _, server := range config.Servers {
		metricsData := agentMetrics[server.ID]
		online := false
		if metricsData != nil {
			online = time.Since(metricsData.LastUpdated).Seconds() < 30
		}

		version := server.Version
		if metricsData != nil && metricsData.Metrics.Version != "" {
			version = metricsData.Metrics.Version
		}

		var metrics *SystemMetrics
		if metricsData != nil {
			metrics = &metricsData.Metrics
		}

		serverMsg := StreamServerMessage{
			Type:  "stream_server",
			Index: index,
			Total: totalServers,
			Server: ServerMetricsUpdate{
				ServerID:     server.ID,
				ServerName:   server.Name,
				Location:     server.Location,
				Provider:     server.Provider,
				Tag:          server.Tag,
				GroupID:      server.GroupID,
				GroupValues:  server.GroupValues,
				Version:      version,
				IP:           server.IP,
				Online:       online,
				Metrics:      metrics,
				PriceAmount:  server.PriceAmount,
				PricePeriod:  server.PricePeriod,
				PurchaseDate: server.PurchaseDate,
				TipBadge:     server.TipBadge,
			},
		}
		serverData, _ := json.Marshal(serverMsg)
		if err := writeMessage(serverData); err != nil {
			return
		}
		index++
	}

	// Step 3: Send end message
	endMsg := StreamEndMessage{Type: "stream_end"}
	endData, _ := json.Marshal(endMsg)
	writeMessage(endData)
}

// RefreshSnapshot rebuilds the dashboard snapshot (called periodically)
func (s *AppState) RefreshSnapshot() {
	s.ConfigMu.RLock()
	config := s.Config
	s.ConfigMu.RUnlock()

	s.AgentMetricsMu.RLock()
	agentMetrics := make(map[string]*AgentMetricsData)
	for k, v := range s.AgentMetrics {
		agentMetrics[k] = v
	}
	s.AgentMetricsMu.RUnlock()

	totalServers := 1 + len(config.Servers)
	snapshot := &DashboardSnapshot{
		ServerMessages: make([][]byte, 0, totalServers),
		LastUpdated:    time.Now(),
	}

	// Build init message
	initMsg := StreamInitMessage{
		Type:            "stream_init",
		TotalServers:    totalServers,
		Groups:          config.Groups,
		GroupDimensions: config.GroupDimensions,
		SiteSettings:    &config.SiteSettings,
	}
	snapshot.InitMessage, _ = json.Marshal(initMsg)

	// Build local server message
	localMetrics := CollectMetrics()
	localNode := config.LocalNode
	localName := "Dashboard Server"
	if localNode.Name != "" {
		localName = localNode.Name
	}
	provider := "Local"
	if localNode.Provider != "" {
		provider = localNode.Provider
	}

	localServer := StreamServerMessage{
		Type:  "stream_server",
		Index: 0,
		Total: totalServers,
		Server: ServerMetricsUpdate{
			ServerID:     "local",
			ServerName:   localName,
			Location:     localNode.Location,
			Provider:     provider,
			Tag:          localNode.Tag,
			GroupID:      localNode.GroupID,
			GroupValues:  localNode.GroupValues,
			Version:      ServerVersion,
			IP:           "",
			Online:       true,
			Metrics:      &localMetrics,
			PriceAmount:  localNode.PriceAmount,
			PricePeriod:  localNode.PricePeriod,
			PurchaseDate: localNode.PurchaseDate,
			TipBadge:     localNode.TipBadge,
		},
	}
	localData, _ := json.Marshal(localServer)
	snapshot.ServerMessages = append(snapshot.ServerMessages, localData)

	// Build remote server messages
	index := 1
	for _, server := range config.Servers {
		metricsData := agentMetrics[server.ID]
		online := false
		if metricsData != nil {
			online = time.Since(metricsData.LastUpdated).Seconds() < 30
		}

		version := server.Version
		if metricsData != nil && metricsData.Metrics.Version != "" {
			version = metricsData.Metrics.Version
		}

		var metrics *SystemMetrics
		if metricsData != nil {
			metrics = &metricsData.Metrics
		}

		serverMsg := StreamServerMessage{
			Type:  "stream_server",
			Index: index,
			Total: totalServers,
			Server: ServerMetricsUpdate{
				ServerID:     server.ID,
				ServerName:   server.Name,
				Location:     server.Location,
				Provider:     server.Provider,
				Tag:          server.Tag,
				GroupID:      server.GroupID,
				GroupValues:  server.GroupValues,
				Version:      version,
				IP:           server.IP,
				Online:       online,
				Metrics:      metrics,
				PriceAmount:  server.PriceAmount,
				PricePeriod:  server.PricePeriod,
				PurchaseDate: server.PurchaseDate,
				TipBadge:     server.TipBadge,
			},
		}
		serverData, _ := json.Marshal(serverMsg)
		snapshot.ServerMessages = append(snapshot.ServerMessages, serverData)
		index++
	}

	// Build end message
	endMsg := StreamEndMessage{Type: "stream_end"}
	snapshot.EndMessage, _ = json.Marshal(endMsg)

	// Atomically replace snapshot
	s.SnapshotMu.Lock()
	s.Snapshot = snapshot
	s.SnapshotMu.Unlock()
}

func (s *AppState) BroadcastMetrics(msg string) {
	s.DashboardMu.RLock()
	clients := make([]*DashboardClient, 0, len(s.DashboardClients))
	for _, client := range s.DashboardClients {
		if client != nil && client.Conn != nil {
			clients = append(clients, client)
		}
	}
	s.DashboardMu.RUnlock()

	msgBytes := []byte(msg)
	for _, client := range clients {
		client.WriteMu.Lock()
		err := client.Conn.WriteMessage(websocket.TextMessage, msgBytes)
		client.WriteMu.Unlock()

		if err != nil {
			s.DashboardMu.Lock()
			delete(s.DashboardClients, client.Conn)
			s.DashboardMu.Unlock()
			client.Conn.Close()
		}
	}
}

// ============================================================================
// Agent WebSocket Handler
// ============================================================================

func (s *AppState) HandleAgentWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	clientIP := c.ClientIP()
	var authenticatedServerID string

	// Create channel for sending commands
	sendChan := make(chan []byte, 16)
	done := make(chan struct{})

	// Goroutine to send commands to agent
	go func() {
		for {
			select {
			case msg := <-sendChan:
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					log.Printf("Failed to send message to agent: %v", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Handle incoming messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var agentMsg AgentMessage
		if err := json.Unmarshal(message, &agentMsg); err != nil {
			continue
		}

		switch agentMsg.Type {
		case "auth":
			if agentMsg.ServerID != "" && agentMsg.Token != "" {
				s.ConfigMu.Lock()
				var server *RemoteServer
				for i := range s.Config.Servers {
					if s.Config.Servers[i].ID == agentMsg.ServerID {
						if s.Config.Servers[i].Token == agentMsg.Token {
							server = &s.Config.Servers[i]
							authenticatedServerID = agentMsg.ServerID

							// Update version
							if agentMsg.Version != "" && server.Version != agentMsg.Version {
								server.Version = agentMsg.Version
								SaveConfig(s.Config)
							}

							// Register connection
							s.AgentConnsMu.Lock()
							s.AgentConns[agentMsg.ServerID] = &AgentConnection{
								Conn:     conn,
								SendChan: sendChan,
							}
							s.AgentConnsMu.Unlock()

							// Send auth success with probe config and last data time
							response := map[string]interface{}{
								"type":   "auth",
								"status": "ok",
							}
							if len(s.Config.ProbeSettings.PingTargets) > 0 {
								response["ping_targets"] = s.Config.ProbeSettings.PingTargets
							}
							
							// Get last metrics time for resumable sync
							if lastTime := GetLastMetricsTime(agentMsg.ServerID); lastTime != nil {
								response["last_seen"] = lastTime.Format(time.RFC3339)
							}
							
							// Get last buckets for each granularity
							if lastBuckets := GetLastAggregationBuckets(agentMsg.ServerID); len(lastBuckets) > 0 {
								response["last_buckets"] = lastBuckets
							}
							
							data, _ := json.Marshal(response)
							conn.WriteMessage(websocket.TextMessage, data)
							log.Printf("Agent %s authenticated", agentMsg.ServerID)
						} else {
							conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"auth","status":"error","message":"Invalid token"}`))
						}
						break
					}
				}
				if server == nil {
					conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"auth","status":"error","message":"Server not found"}`))
				}
				s.ConfigMu.Unlock()
			}

		case "metrics":
			if authenticatedServerID != "" && agentMsg.Metrics != nil {
				// Store to database asynchronously via channel queue with deduplication
				StoreMetricsWithDedup(authenticatedServerID, agentMsg.Metrics)

				// Determine IP address
				agentIP := clientIP
				if len(agentMsg.Metrics.IPAddresses) > 0 {
					agentIP = agentMsg.Metrics.IPAddresses[0]
				}

				// Update version and IP in config
				s.ConfigMu.Lock()
				for i := range s.Config.Servers {
					if s.Config.Servers[i].ID == authenticatedServerID {
						changed := false
						if agentMsg.Metrics.Version != "" && s.Config.Servers[i].Version != agentMsg.Metrics.Version {
							s.Config.Servers[i].Version = agentMsg.Metrics.Version
							changed = true
						}
						if s.Config.Servers[i].IP != agentIP {
							s.Config.Servers[i].IP = agentIP
							changed = true
						}
						if changed {
							SaveConfig(s.Config)
						}
						break
					}
				}
				s.ConfigMu.Unlock()

				// Update in-memory state
				s.AgentMetricsMu.Lock()
				s.AgentMetrics[authenticatedServerID] = &AgentMetricsData{
					ServerID:    authenticatedServerID,
					Metrics:     *agentMsg.Metrics,
					LastUpdated: time.Now(),
				}
				s.AgentMetricsMu.Unlock()
			} else {
				conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"Not authenticated"}`))
			}

		case "batch_metrics":
			if authenticatedServerID == "" {
				conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"Not authenticated"}`))
				continue
			}

			accepted, rejected := s.handleBatchMetrics(authenticatedServerID, &agentMsg)
			
			// Send acknowledgment
			ackResponse := map[string]interface{}{
				"type":     "batch_ack",
				"batch_id": agentMsg.BatchID,
				"accepted": accepted,
				"rejected": rejected,
			}
			ackData, _ := json.Marshal(ackResponse)
			conn.WriteMessage(websocket.TextMessage, ackData)
			
			log.Printf("Batch %s from %s: accepted=%d, rejected=%d", 
				agentMsg.BatchID, authenticatedServerID, accepted, rejected)

		case "aggregated_metrics":
			if authenticatedServerID == "" {
				conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"Not authenticated"}`))
				continue
			}

			// Store multi-granularity aggregated data from agent
			if len(agentMsg.Granularities) > 0 {
				StoreMultiGranularityMetrics(authenticatedServerID, agentMsg.Granularities)
			}

			// Update in-memory state with last metrics if provided
			if agentMsg.LastMetrics != nil {
				s.AgentMetricsMu.Lock()
				s.AgentMetrics[authenticatedServerID] = &AgentMetricsData{
					ServerID:    authenticatedServerID,
					Metrics:     *agentMsg.LastMetrics,
					LastUpdated: time.Now(),
				}
				s.AgentMetricsMu.Unlock()
			}
		}
	}

	// Cleanup on disconnect
	close(done) // Stop the send goroutine
	if authenticatedServerID != "" {
		log.Printf("Agent %s disconnected", authenticatedServerID)
		s.AgentConnsMu.Lock()
		delete(s.AgentConns, authenticatedServerID)
		s.AgentConnsMu.Unlock()
	}
}

// handleBatchMetrics processes batch metrics from an agent
func (s *AppState) handleBatchMetrics(serverID string, msg *AgentMessage) (accepted, rejected int) {
	// Process raw metrics
	for _, tm := range msg.BatchItems {
		if tm.Metrics == nil {
			rejected++
			continue
		}

		// Parse timestamp
		ts, err := time.Parse(time.RFC3339Nano, tm.Timestamp)
		if err != nil {
			ts, err = time.Parse(time.RFC3339, tm.Timestamp)
			if err != nil {
				rejected++
				continue
			}
		}

		// Update metrics timestamp
		tm.Metrics.Timestamp = ts

		// Store with deduplication
		if StoreBatchMetrics(serverID, tm.Metrics) {
			accepted++
		} else {
			rejected++ // Duplicate or error
		}
	}

	// Process aggregated metrics
	for _, agg := range msg.Aggregated {
		if agg == nil {
			rejected++
			continue
		}

		// Store aggregated metrics
		if StoreAggregatedMetrics(serverID, agg) {
			accepted++
		} else {
			rejected++
		}
	}

	// Update in-memory state with the latest metrics if available
	if len(msg.BatchItems) > 0 {
		lastItem := msg.BatchItems[len(msg.BatchItems)-1]
		if lastItem.Metrics != nil {
			s.AgentMetricsMu.Lock()
			s.AgentMetrics[serverID] = &AgentMetricsData{
				ServerID:    serverID,
				Metrics:     *lastItem.Metrics,
				LastUpdated: time.Now(),
			}
			s.AgentMetricsMu.Unlock()
		}
	} else if len(msg.Aggregated) > 0 && msg.Aggregated[len(msg.Aggregated)-1].LastMetrics != nil {
		lastAgg := msg.Aggregated[len(msg.Aggregated)-1]
		s.AgentMetricsMu.Lock()
		s.AgentMetrics[serverID] = &AgentMetricsData{
			ServerID:    serverID,
			Metrics:     *lastAgg.LastMetrics,
			LastUpdated: time.Now(),
		}
		s.AgentMetricsMu.Unlock()
	}

	return accepted, rejected
}


