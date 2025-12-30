package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Metrics Handlers
// ============================================================================

func (s *AppState) GetMetrics(c *gin.Context) {
	// Local metrics collection has been removed.
	// All metrics should be reported by agents.
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Local metrics collection is not available",
		"message": "Please install an agent to report metrics",
	})
}

func (s *AppState) GetAllMetrics(c *gin.Context) {
	s.ConfigMu.RLock()
	servers := s.Config.Servers
	s.ConfigMu.RUnlock()

	s.AgentMetricsMu.RLock()
	defer s.AgentMetricsMu.RUnlock()

	var updates []ServerMetricsUpdate
	for _, server := range servers {
		metricsData := s.AgentMetrics[server.ID]
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

		updates = append(updates, ServerMetricsUpdate{
			ServerID:      server.ID,
			ServerName:    server.Name,
			Location:      server.Location,
			Provider:      server.Provider,
			Tag:           server.Tag,
			GroupID:       server.GroupID,
			GroupValues:   server.GroupValues,
			Version:       version,
			IP:            server.IP,
			Online:        online,
			Metrics:       metrics,
			PriceAmount:   server.PriceAmount,
			PricePeriod:   server.PricePeriod,
			PriceCurrency: server.PriceCurrency,
			PurchaseDate:  server.PurchaseDate,
			ExpiryDate:    server.ExpiryDate,
			AutoRenew:     server.AutoRenew,
			TipBadge:      server.TipBadge,
			Notes:         server.Notes,
			GeoIP:         server.GeoIP,
			SaleStatus:    server.SaleStatus,
			SaleContactURL: server.SaleContactURL,
		})
	}

	c.JSON(http.StatusOK, updates)
}

// ============================================================================
// History Handler
// ============================================================================

func (s *AppState) GetHistory(c *gin.Context, db *sql.DB) {
	serverID := c.Param("server_id")
	rangeStr := c.DefaultQuery("range", "24h")
	dataType := c.DefaultQuery("type", "all") // "ping", "metrics", or "all"
	sinceStr := c.Query("since")              // Bucket number for incremental updates

	var sinceBucket int64
	if sinceStr != "" {
		fmt.Sscanf(sinceStr, "%d", &sinceBucket)
	}

	// Only use cache for 1h and 24h ranges with type=all
	useCache := (rangeStr == "1h" || rangeStr == "24h" || rangeStr == "") && dataType == "all" && historyCache != nil

	// Check cache first (for full queries only, not incremental)
	if useCache && sinceBucket == 0 {
		if cached, ok := historyCache.Get(serverID, rangeStr); ok {
			c.JSON(http.StatusOK, HistoryResponse{
				ServerID:    serverID,
				Range:       rangeStr,
				Data:        cached.Data,
				PingTargets: cached.PingTargets,
				LastBucket:  cached.LastBucket,
			})
			return
		}
	}

	var data []HistoryPoint
	var pingTargets []PingHistoryTarget
	var metricsErr, pingErr error
	var lastBucket int64

	if dataType == "all" {
		// Run both queries in parallel for better performance
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			data, metricsErr = GetHistorySince(db, serverID, rangeStr, sinceBucket)
		}()

		go func() {
			defer wg.Done()
			pingTargets, pingErr = GetPingHistorySince(db, serverID, rangeStr, sinceBucket)
		}()

		wg.Wait()

		if metricsErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
			return
		}
		// Ignore ping errors, just return empty if failed
		_ = pingErr
	} else if dataType == "metrics" {
		data, metricsErr = GetHistorySince(db, serverID, rangeStr, sinceBucket)
		if metricsErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
			return
		}
	} else if dataType == "ping" {
		pingTargets, _ = GetPingHistorySince(db, serverID, rangeStr, sinceBucket)
	}

	// Calculate last bucket from the data
	now := time.Now().UTC()
	switch rangeStr {
	case "1h":
		lastBucket = now.Unix() / 5
	case "24h", "":
		lastBucket = now.Unix() / 120
	}

	// Update cache for full queries
	if useCache && sinceBucket == 0 {
		historyCache.Set(serverID, rangeStr, data, pingTargets, lastBucket)
	} else if useCache && sinceBucket > 0 {
		// Update cache with new data for incremental queries
		historyCache.Update(serverID, rangeStr, data, pingTargets, lastBucket)
	}

	c.JSON(http.StatusOK, HistoryResponse{
		ServerID:    serverID,
		Range:       rangeStr,
		Data:        data,
		PingTargets: pingTargets,
		LastBucket:  lastBucket,
		Incremental: sinceBucket > 0,
	})
}

// ============================================================================
// Health Check
// ============================================================================

func HealthCheck(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

// ============================================================================
// Online Users Handler
// ============================================================================

type OnlineUsersResponse struct {
	Count int `json:"count"`
}

func (s *AppState) GetOnlineUsers(c *gin.Context) {
	count := s.GetOnlineUsersCount()
	c.JSON(http.StatusOK, OnlineUsersResponse{Count: count})
}
