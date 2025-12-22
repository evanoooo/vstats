package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Global alert engine instance
var alertEngine *AlertEngine

// ============================================================================
// Alert Configuration Handlers
// ============================================================================

// GetAlertConfig returns the current alert configuration
func (s *AppState) GetAlertConfig(c *gin.Context) {
	s.ConfigMu.RLock()
	config := s.Config.AlertConfig
	s.ConfigMu.RUnlock()

	if config == nil {
		config = &AlertConfig{}
		defaultConfig := GetDefaultAlertConfig()
		*config = defaultConfig
	}

	c.JSON(http.StatusOK, config)
}

// UpdateAlertConfig updates the alert configuration
func (s *AppState) UpdateAlertConfig(c *gin.Context) {
	var req UpdateAlertConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.ConfigMu.Lock()
	if s.Config.AlertConfig == nil {
		defaultConfig := GetDefaultAlertConfig()
		s.Config.AlertConfig = &defaultConfig
	}

	config := s.Config.AlertConfig

	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.Channels != nil {
		config.Channels = *req.Channels
	}
	if req.Rules != nil {
		config.Rules = *req.Rules
	}
	if req.Templates != nil {
		config.Templates = *req.Templates
	}
	if req.GlobalCooldown != nil {
		config.GlobalCooldown = *req.GlobalCooldown
	}
	if req.RecoveryNotify != nil {
		config.RecoveryNotify = *req.RecoveryNotify
	}

	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	LogAuditFromContext(c, AuditActionAlertConfigUpdate, AuditCategoryAlert, "settings", "alert_config", "Alert Config", "Alert configuration updated")

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ============================================================================
// Notification Channel Handlers
// ============================================================================

// GetChannels returns all notification channels
func (s *AppState) GetChannels(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	var channels []NotificationChannel
	if s.Config.AlertConfig != nil {
		channels = s.Config.AlertConfig.Channels
	}
	if channels == nil {
		channels = []NotificationChannel{}
	}

	c.JSON(http.StatusOK, channels)
}

// AddChannel adds a new notification channel
func (s *AppState) AddChannel(c *gin.Context) {
	var req AddChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channel := NotificationChannel{
		ID:       GenerateRandomString(12),
		Type:     req.Type,
		Name:     req.Name,
		Enabled:  req.Enabled,
		Config:   req.Config,
		Priority: req.Priority,
	}

	// Validate channel configuration
	notifier, err := CreateNotifier(channel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := notifier.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Configuration validation failed: " + err.Error()})
		return
	}

	s.ConfigMu.Lock()
	if s.Config.AlertConfig == nil {
		defaultConfig := GetDefaultAlertConfig()
		s.Config.AlertConfig = &defaultConfig
	}
	s.Config.AlertConfig.Channels = append(s.Config.AlertConfig.Channels, channel)
	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	LogAuditFromContext(c, AuditActionChannelCreate, AuditCategoryAlert, "channel", channel.ID, channel.Name, "Notification channel created: "+string(channel.Type))

	c.JSON(http.StatusOK, channel)
}

// UpdateChannel updates an existing notification channel
func (s *AppState) UpdateChannel(c *gin.Context) {
	channelID := c.Param("id")

	var req AddChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.ConfigMu.Lock()
	defer s.ConfigMu.Unlock()

	if s.Config.AlertConfig == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	found := false
	for i := range s.Config.AlertConfig.Channels {
		if s.Config.AlertConfig.Channels[i].ID == channelID {
			s.Config.AlertConfig.Channels[i].Type = req.Type
			s.Config.AlertConfig.Channels[i].Name = req.Name
			s.Config.AlertConfig.Channels[i].Enabled = req.Enabled
			s.Config.AlertConfig.Channels[i].Config = req.Config
			s.Config.AlertConfig.Channels[i].Priority = req.Priority
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	SaveConfig(s.Config)

	LogAuditFromContext(c, AuditActionChannelUpdate, AuditCategoryAlert, "channel", channelID, req.Name, "Notification channel updated")

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteChannel deletes a notification channel
func (s *AppState) DeleteChannel(c *gin.Context) {
	channelID := c.Param("id")

	s.ConfigMu.Lock()
	defer s.ConfigMu.Unlock()

	if s.Config.AlertConfig == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	found := false
	channels := make([]NotificationChannel, 0)
	for _, ch := range s.Config.AlertConfig.Channels {
		if ch.ID == channelID {
			found = true
			continue
		}
		channels = append(channels, ch)
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	s.Config.AlertConfig.Channels = channels
	SaveConfig(s.Config)

	LogAuditFromContext(c, AuditActionChannelDelete, AuditCategoryAlert, "channel", channelID, "", "Notification channel deleted")

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// TestChannel tests a notification channel
func (s *AppState) TestChannel(c *gin.Context) {
	var req TestChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var channel NotificationChannel

	if req.ChannelID != "" {
		// Test existing channel
		s.ConfigMu.RLock()
		if s.Config.AlertConfig != nil {
			for _, ch := range s.Config.AlertConfig.Channels {
				if ch.ID == req.ChannelID {
					channel = ch
					break
				}
			}
		}
		s.ConfigMu.RUnlock()

		if channel.ID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
	} else {
		// Test inline configuration
		channel = NotificationChannel{
			Type:   req.Type,
			Config: req.Config,
		}
	}

	notifier, err := CreateNotifier(channel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := notifier.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed: " + err.Error()})
		return
	}

	title := "vStats 测试通知"
	message := "这是一条测试通知消息。\n发送时间: " + time.Now().Format("2006-01-02 15:04:05")

	if err := notifier.Send(title, message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send test notification: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Test notification sent successfully"})
}

// ============================================================================
// Alert State Handlers
// ============================================================================

// GetAlerts returns current active alerts and statistics
func (s *AppState) GetAlerts(c *gin.Context) {
	if alertEngine == nil {
		c.JSON(http.StatusOK, AlertsResponse{
			Alerts: []AlertState{},
			Stats:  AlertStats{},
		})
		return
	}

	alerts := alertEngine.GetActiveAlerts()
	stats := alertEngine.GetAlertStats()

	c.JSON(http.StatusOK, AlertsResponse{
		Alerts: alerts,
		Stats:  stats,
	})
}

// GetAlertHistory returns historical alerts
func (s *AppState) GetAlertHistory(c *gin.Context) {
	serverID := c.Query("server_id")
	alertType := c.Query("type")
	limit := c.DefaultQuery("limit", "100")

	db := dbWriter.GetDB()

	query := `
		SELECT id, alert_id, type, server_id, server_name, severity, value, threshold, message, started_at, resolved_at, duration, notified
		FROM alert_history
		WHERE 1=1`
	args := []interface{}{}

	if serverID != "" {
		query += " AND server_id = ?"
		args = append(args, serverID)
	}
	if alertType != "" {
		query += " AND type = ?"
		args = append(args, alertType)
	}

	query += " ORDER BY started_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var history []AlertHistory
	for rows.Next() {
		var h AlertHistory
		var startedAt string
		var resolvedAtPtr *string

		err := rows.Scan(
			&h.ID, &h.AlertID, &h.Type, &h.ServerID, &h.ServerName,
			&h.Severity, &h.Value, &h.Threshold, &h.Message,
			&startedAt, &resolvedAtPtr, &h.Duration, &h.Notified,
		)
		if err != nil {
			continue
		}

		h.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		if resolvedAtPtr != nil {
			t, _ := time.Parse(time.RFC3339, *resolvedAtPtr)
			h.ResolvedAt = &t
		}

		history = append(history, h)
	}

	if history == nil {
		history = []AlertHistory{}
	}

	c.JSON(http.StatusOK, history)
}

// MuteAlert mutes an active alert
func (s *AppState) MuteAlert(c *gin.Context) {
	alertID := c.Param("id")

	if alertEngine == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert engine not running"})
		return
	}

	if alertEngine.MuteAlert(alertID) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
	}
}

// ============================================================================
// Alert Rules Handlers
// ============================================================================

// UpdateOfflineRule updates offline alert rules
func (s *AppState) UpdateOfflineRule(c *gin.Context) {
	var rule OfflineAlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.ConfigMu.Lock()
	if s.Config.AlertConfig == nil {
		defaultConfig := GetDefaultAlertConfig()
		s.Config.AlertConfig = &defaultConfig
	}
	s.Config.AlertConfig.Rules.Offline = rule
	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateLoadRule updates load alert rules
func (s *AppState) UpdateLoadRule(c *gin.Context) {
	var rule LoadAlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.ConfigMu.Lock()
	if s.Config.AlertConfig == nil {
		defaultConfig := GetDefaultAlertConfig()
		s.Config.AlertConfig = &defaultConfig
	}
	s.Config.AlertConfig.Rules.Load = rule
	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateTrafficRule updates traffic alert rules
func (s *AppState) UpdateTrafficRule(c *gin.Context) {
	var rule TrafficAlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.ConfigMu.Lock()
	if s.Config.AlertConfig == nil {
		defaultConfig := GetDefaultAlertConfig()
		s.Config.AlertConfig = &defaultConfig
	}
	s.Config.AlertConfig.Rules.Traffic = rule
	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ============================================================================
// Alert Templates Handlers
// ============================================================================

// GetAlertTemplates returns all alert templates
func (s *AppState) GetAlertTemplates(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	var templates map[string]AlertTemplate
	if s.Config.AlertConfig != nil && s.Config.AlertConfig.Templates != nil {
		templates = s.Config.AlertConfig.Templates
	} else {
		// Return defaults
		defaultConfig := GetDefaultAlertConfig()
		templates = defaultConfig.Templates
	}

	c.JSON(http.StatusOK, templates)
}

// UpdateAlertTemplate updates a specific alert template
func (s *AppState) UpdateAlertTemplate(c *gin.Context) {
	templateKey := c.Param("key")

	var template AlertTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.ConfigMu.Lock()
	if s.Config.AlertConfig == nil {
		defaultConfig := GetDefaultAlertConfig()
		s.Config.AlertConfig = &defaultConfig
	}
	if s.Config.AlertConfig.Templates == nil {
		defaultConfig := GetDefaultAlertConfig()
		s.Config.AlertConfig.Templates = defaultConfig.Templates
	}
	s.Config.AlertConfig.Templates[templateKey] = template
	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true})
}


