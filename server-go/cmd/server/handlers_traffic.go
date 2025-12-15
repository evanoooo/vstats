package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Traffic Management Handlers
// ============================================================================

// GetTrafficSummary returns a summary of all traffic stats
func (s *AppState) GetTrafficSummary(c *gin.Context) {
	if trafficManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Traffic manager not initialized"})
		return
	}

	summary := trafficManager.GetSummary()
	c.JSON(http.StatusOK, summary)
}

// GetTrafficStats returns traffic stats for all servers
func (s *AppState) GetTrafficStats(c *gin.Context) {
	if trafficManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Traffic manager not initialized"})
		return
	}

	stats := trafficManager.GetAllStats()
	c.JSON(http.StatusOK, stats)
}

// GetServerTrafficStats returns traffic stats for a specific server
func (s *AppState) GetServerTrafficStats(c *gin.Context) {
	if trafficManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Traffic manager not initialized"})
		return
	}

	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server_id is required"})
		return
	}

	stats := trafficManager.GetServerStats(serverID)
	if stats == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// UpdateTrafficLimit updates traffic limit for a server
func (s *AppState) UpdateTrafficLimit(c *gin.Context) {
	if trafficManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Traffic manager not initialized"})
		return
	}

	var req UpdateTrafficLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ServerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server_id is required"})
		return
	}

	// Validate threshold type
	if req.ThresholdType != "" && !ValidateThresholdType(req.ThresholdType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid threshold_type, must be one of: sum, max, up, down"})
		return
	}
	if req.ThresholdType == "" {
		req.ThresholdType = "sum"
	}

	// Validate reset day
	if req.ResetDay < 1 || req.ResetDay > 28 {
		req.ResetDay = 1
	}

	err := trafficManager.UpdateLimit(req.ServerID, req.MonthlyLimitGB, req.ThresholdType, req.ResetDay)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Also update the alert config if there's a traffic rule
	s.ConfigMu.Lock()
	if s.Config.AlertConfig != nil && s.Config.AlertConfig.Rules.Traffic.Enabled {
		// Find or create the traffic limit
		found := false
		for i, limit := range s.Config.AlertConfig.Rules.Traffic.Limits {
			if limit.ServerID == req.ServerID {
				s.Config.AlertConfig.Rules.Traffic.Limits[i].MonthlyGB = req.MonthlyLimitGB
				s.Config.AlertConfig.Rules.Traffic.Limits[i].Type = req.ThresholdType
				s.Config.AlertConfig.Rules.Traffic.Limits[i].ResetDay = req.ResetDay
				if req.Warning > 0 {
					s.Config.AlertConfig.Rules.Traffic.Limits[i].Warning = req.Warning
				}
				found = true
				break
			}
		}
		if !found && req.MonthlyLimitGB > 0 {
			// Add new limit
			warning := req.Warning
			if warning == 0 {
				warning = 80
			}
			s.Config.AlertConfig.Rules.Traffic.Limits = append(
				s.Config.AlertConfig.Rules.Traffic.Limits,
				TrafficLimit{
					ServerID:  req.ServerID,
					MonthlyGB: req.MonthlyLimitGB,
					Type:      req.ThresholdType,
					ResetDay:  req.ResetDay,
					Warning:   warning,
				},
			)
		}
		SaveConfig(s.Config)
	}
	s.ConfigMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ResetServerTraffic resets traffic for a server
func (s *AppState) ResetServerTraffic(c *gin.Context) {
	if trafficManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Traffic manager not initialized"})
		return
	}

	var req TrafficResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ServerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server_id is required"})
		return
	}

	err := trafficManager.ResetTraffic(req.ServerID, req.ResetToZero)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetTrafficHistory returns historical traffic records for a server
func (s *AppState) GetTrafficHistory(c *gin.Context) {
	if trafficManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Traffic manager not initialized"})
		return
	}

	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server_id is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "12")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 12
	}

	records, err := trafficManager.GetHistory(serverID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if records == nil {
		records = []TrafficRecord{}
	}

	c.JSON(http.StatusOK, TrafficHistoryResponse{
		Records: records,
		Total:   len(records),
	})
}

// GetTrafficDaily returns daily traffic data for charts
func (s *AppState) GetTrafficDaily(c *gin.Context) {
	if trafficManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Traffic manager not initialized"})
		return
	}

	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server_id is required"})
		return
	}

	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}

	records, err := trafficManager.GetDailyTraffic(serverID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if records == nil {
		records = []DailyTrafficRecord{}
	}

	// Get server name and limit
	var serverName string
	var limitGB float64

	s.ConfigMu.RLock()
	for _, server := range s.Config.Servers {
		if server.ID == serverID {
			serverName = server.Name
			break
		}
	}
	if serverID == "local" {
		serverName = s.Config.LocalNode.Name
		if serverName == "" {
			serverName = "Local Server"
		}
	}
	s.ConfigMu.RUnlock()

	stats := trafficManager.GetServerStats(serverID)
	if stats != nil {
		limitGB = stats.MonthlyLimitGB
	}

	// Calculate totals
	var totalTxGB, totalRxGB float64
	for _, r := range records {
		totalTxGB += r.TxBytesGB
		totalRxGB += r.RxBytesGB
	}

	c.JSON(http.StatusOK, TrafficChartData{
		ServerID:   serverID,
		ServerName: serverName,
		Daily:      records,
		TotalTxGB:  totalTxGB,
		TotalRxGB:  totalRxGB,
		TotalGB:    totalTxGB + totalRxGB,
		LimitGB:    limitGB,
	})
}

// GetAllTrafficLimits returns all traffic limits from alert config
func (s *AppState) GetAllTrafficLimits(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	var limits []TrafficLimit
	if s.Config.AlertConfig != nil {
		limits = s.Config.AlertConfig.Rules.Traffic.Limits
	}
	if limits == nil {
		limits = []TrafficLimit{}
	}

	c.JSON(http.StatusOK, limits)
}

// BatchUpdateTrafficLimits updates multiple traffic limits at once
func (s *AppState) BatchUpdateTrafficLimits(c *gin.Context) {
	if trafficManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Traffic manager not initialized"})
		return
	}

	var req struct {
		Limits []UpdateTrafficLimitRequest `json:"limits"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, limit := range req.Limits {
		if limit.ServerID == "" {
			continue
		}
		if limit.ThresholdType == "" {
			limit.ThresholdType = "sum"
		}
		if limit.ResetDay < 1 || limit.ResetDay > 28 {
			limit.ResetDay = 1
		}
		trafficManager.UpdateLimit(limit.ServerID, limit.MonthlyLimitGB, limit.ThresholdType, limit.ResetDay)
	}

	// Update alert config
	s.ConfigMu.Lock()
	if s.Config.AlertConfig == nil {
		defaultConfig := GetDefaultAlertConfig()
		s.Config.AlertConfig = &defaultConfig
	}

	// Rebuild limits from request
	newLimits := make([]TrafficLimit, 0, len(req.Limits))
	for _, limit := range req.Limits {
		if limit.ServerID == "" || limit.MonthlyLimitGB <= 0 {
			continue
		}
		warning := limit.Warning
		if warning == 0 {
			warning = 80
		}
		newLimits = append(newLimits, TrafficLimit{
			ServerID:  limit.ServerID,
			MonthlyGB: limit.MonthlyLimitGB,
			Type:      limit.ThresholdType,
			ResetDay:  limit.ResetDay,
			Warning:   warning,
		})
	}
	s.Config.AlertConfig.Rules.Traffic.Limits = newLimits
	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true, "count": len(newLimits)})
}

// DeleteTrafficLimit removes a traffic limit for a server
func (s *AppState) DeleteTrafficLimit(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server_id is required"})
		return
	}

	// Update traffic manager
	if trafficManager != nil {
		trafficManager.UpdateLimit(serverID, 0, "sum", 1)
	}

	// Remove from alert config
	s.ConfigMu.Lock()
	if s.Config.AlertConfig != nil {
		limits := make([]TrafficLimit, 0)
		for _, limit := range s.Config.AlertConfig.Rules.Traffic.Limits {
			if limit.ServerID != serverID {
				limits = append(limits, limit)
			}
		}
		s.Config.AlertConfig.Rules.Traffic.Limits = limits
		SaveConfig(s.Config)
	}
	s.ConfigMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true})
}
