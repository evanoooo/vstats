package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"sync"
	"text/template"
	"time"
)

// ============================================================================
// Alert Engine
// ============================================================================

// AlertEngine manages alert detection and notification
type AlertEngine struct {
	state          *AppState
	db             *sql.DB
	
	// Alert states
	activeAlerts   map[string]*AlertState // key: type:server_id
	alertsMu       sync.RWMutex
	
	// Tracking for threshold duration checks
	thresholdState map[string]*thresholdCheck // key: type:server_id
	thresholdMu    sync.Mutex
	
	// Notification cooldowns
	cooldowns      map[string]time.Time // key: type:server_id
	cooldownsMu    sync.RWMutex
	
	// Stop channel
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

// thresholdCheck tracks how long a metric has exceeded threshold
type thresholdCheck struct {
	startTime time.Time
	value     float64
	severity  string
}

// NewAlertEngine creates a new alert engine
func NewAlertEngine(state *AppState, db *sql.DB) *AlertEngine {
	return &AlertEngine{
		state:          state,
		db:             db,
		activeAlerts:   make(map[string]*AlertState),
		thresholdState: make(map[string]*thresholdCheck),
		cooldowns:      make(map[string]time.Time),
		stopCh:         make(chan struct{}),
	}
}

// Start begins the alert monitoring loop
func (e *AlertEngine) Start() {
	e.wg.Add(1)
	go e.monitorLoop()
	fmt.Println("🔔 Alert engine started")
}

// Stop gracefully stops the alert engine
func (e *AlertEngine) Stop() {
	close(e.stopCh)
	e.wg.Wait()
	fmt.Println("🔔 Alert engine stopped")
}

// monitorLoop runs the main monitoring cycle
func (e *AlertEngine) monitorLoop() {
	defer e.wg.Done()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.checkAlerts()
		}
	}
}

// checkAlerts runs all alert checks
func (e *AlertEngine) checkAlerts() {
	e.state.ConfigMu.RLock()
	config := e.state.Config
	alertConfig := config.AlertConfig
	e.state.ConfigMu.RUnlock()
	
	if alertConfig == nil || !alertConfig.Enabled {
		return
	}
	
	// Get current server states
	servers := e.getServerStates()
	
	// Check offline alerts
	if alertConfig.Rules.Offline.Enabled {
		e.checkOfflineAlerts(servers, alertConfig)
	}
	
	// Check load alerts
	if alertConfig.Rules.Load.Enabled {
		e.checkLoadAlerts(servers, alertConfig)
	}
	
	// Check traffic alerts
	if alertConfig.Rules.Traffic.Enabled {
		e.checkTrafficAlerts(servers, alertConfig)
	}
	
	// Check expiry alerts
	if alertConfig.Rules.Expiry.Enabled {
		e.checkExpiryAlerts(alertConfig)
	}
}

// serverState represents the current state of a server for alerting
type serverState struct {
	ID         string
	Name       string
	Online     bool
	LastSeen   time.Time
	CPU        float32
	Memory     float32
	Disk       float32
	TrafficRx  uint64
	TrafficTx  uint64
}

// getServerStates collects current state of all servers
func (e *AlertEngine) getServerStates() []serverState {
	e.state.ConfigMu.RLock()
	config := e.state.Config
	e.state.ConfigMu.RUnlock()
	
	e.state.AgentMetricsMu.RLock()
	agentMetrics := e.state.AgentMetrics
	e.state.AgentMetricsMu.RUnlock()
	
	var servers []serverState
	
	// Add remote servers
	for _, server := range config.Servers {
		state := serverState{
			ID:   server.ID,
			Name: server.Name,
		}
		
		if metrics, ok := agentMetrics[server.ID]; ok {
			state.Online = time.Since(metrics.LastUpdated).Seconds() < 30
			state.LastSeen = metrics.LastUpdated
			state.CPU = metrics.Metrics.CPU.Usage
			state.Memory = metrics.Metrics.Memory.UsagePercent
			if len(metrics.Metrics.Disks) > 0 {
				state.Disk = metrics.Metrics.Disks[0].UsagePercent
			}
			state.TrafficRx = metrics.Metrics.Network.TotalRx
			state.TrafficTx = metrics.Metrics.Network.TotalTx
		}
		
		servers = append(servers, state)
	}
	
	return servers
}

// ============================================================================
// Offline Alert Detection
// ============================================================================

func (e *AlertEngine) checkOfflineAlerts(servers []serverState, config *AlertConfig) {
	rule := config.Rules.Offline
	gracePeriod := time.Duration(rule.GracePeriod) * time.Second
	
	for _, server := range servers {
		// Skip if server is in exclude list
		if contains(rule.Exclude, server.ID) {
			continue
		}
		// Skip if servers list is specified and server is not in it
		if len(rule.Servers) > 0 && !contains(rule.Servers, server.ID) {
			continue
		}
		
		alertKey := fmt.Sprintf("offline:%s", server.ID)
		
		if !server.Online {
			offlineDuration := time.Since(server.LastSeen)
			
			// Check if past grace period
			if offlineDuration >= gracePeriod {
				e.alertsMu.RLock()
				existing := e.activeAlerts[alertKey]
				e.alertsMu.RUnlock()
				
				if existing == nil {
					// Create new alert
					alert := &AlertState{
						ID:         GenerateRandomString(16),
						Type:       "offline",
						ServerID:   server.ID,
						ServerName: server.Name,
						Severity:   "critical",
						Status:     "firing",
						Message:    fmt.Sprintf("服务器 %s 已离线 %s", server.Name, formatDuration(offlineDuration)),
						StartedAt:  server.LastSeen,
						UpdatedAt:  time.Now(),
					}
					
					e.alertsMu.Lock()
					e.activeAlerts[alertKey] = alert
					e.alertsMu.Unlock()
					
					// Send notification
					e.notify(alert, config)
				} else {
					// Update existing alert
					existing.Message = fmt.Sprintf("服务器 %s 已离线 %s", server.Name, formatDuration(offlineDuration))
					existing.UpdatedAt = time.Now()
				}
			}
		} else {
			// Server is online - check for recovery
			e.alertsMu.RLock()
			existing := e.activeAlerts[alertKey]
			e.alertsMu.RUnlock()
			
			if existing != nil {
				e.resolveAlert(alertKey, config)
			}
		}
	}
}

// ============================================================================
// Load Alert Detection (CPU/Memory/Disk)
// ============================================================================

func (e *AlertEngine) checkLoadAlerts(servers []serverState, config *AlertConfig) {
	rule := config.Rules.Load
	
	for _, server := range servers {
		if !server.Online {
			continue
		}
		
		// Skip if server is in exclude list
		if contains(rule.Exclude, server.ID) {
			continue
		}
		// Skip if servers list is specified and server is not in it
		if len(rule.Servers) > 0 && !contains(rule.Servers, server.ID) {
			continue
		}
		
		// Check CPU
		if rule.CPU != nil {
			e.checkThreshold(server, "cpu", server.CPU, rule.CPU, rule.Cooldown, config)
		}
		
		// Check Memory
		if rule.Memory != nil {
			e.checkThreshold(server, "memory", server.Memory, rule.Memory, rule.Cooldown, config)
		}
		
		// Check Disk
		if rule.Disk != nil {
			e.checkThreshold(server, "disk", server.Disk, rule.Disk, rule.Cooldown, config)
		}
	}
}

func (e *AlertEngine) checkThreshold(server serverState, metricType string, value float32, threshold *ThresholdConfig, cooldown int, config *AlertConfig) {
	alertKey := fmt.Sprintf("%s:%s", metricType, server.ID)
	
	// Determine severity based on thresholds
	var severity string
	var thresholdValue float32
	
	if threshold.Critical > 0 && value >= threshold.Critical {
		severity = "critical"
		thresholdValue = threshold.Critical
	} else if threshold.Warning > 0 && value >= threshold.Warning {
		severity = "warning"
		thresholdValue = threshold.Warning
	} else {
		// Below threshold - check for recovery
		e.thresholdMu.Lock()
		delete(e.thresholdState, alertKey)
		e.thresholdMu.Unlock()
		
		e.alertsMu.RLock()
		existing := e.activeAlerts[alertKey]
		e.alertsMu.RUnlock()
		
		if existing != nil {
			e.resolveAlert(alertKey, config)
		}
		return
	}
	
	// Track threshold duration
	e.thresholdMu.Lock()
	check := e.thresholdState[alertKey]
	if check == nil || check.severity != severity {
		// Start new threshold tracking
		e.thresholdState[alertKey] = &thresholdCheck{
			startTime: time.Now(),
			value:     float64(value),
			severity:  severity,
		}
		e.thresholdMu.Unlock()
		return
	}
	
	duration := time.Since(check.startTime)
	check.value = float64(value)
	e.thresholdMu.Unlock()
	
	// Check if duration threshold met
	requiredDuration := time.Duration(threshold.Duration) * time.Second
	if requiredDuration > 0 && duration < requiredDuration {
		return
	}
	
	// Check cooldown
	if !e.checkCooldown(alertKey, cooldown) {
		return
	}
	
	// Create or update alert
	e.alertsMu.RLock()
	existing := e.activeAlerts[alertKey]
	e.alertsMu.RUnlock()
	
	metricName := getMetricName(metricType)
	message := fmt.Sprintf("服务器 %s %s使用率达到 %.1f%%，超过%s阈值 %.1f%%", 
		server.Name, metricName, value, getSeverityName(severity), thresholdValue)
	
	if existing == nil {
		alert := &AlertState{
			ID:         GenerateRandomString(16),
			Type:       metricType,
			ServerID:   server.ID,
			ServerName: server.Name,
			Severity:   severity,
			Status:     "firing",
			Value:      float64(value),
			Threshold:  float64(thresholdValue),
			Message:    message,
			StartedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		
		e.alertsMu.Lock()
		e.activeAlerts[alertKey] = alert
		e.alertsMu.Unlock()
		
		e.notify(alert, config)
		e.setCooldown(alertKey, cooldown)
	} else {
		// Update if severity increased
		if severity == "critical" && existing.Severity == "warning" {
			existing.Severity = severity
			existing.Value = float64(value)
			existing.Threshold = float64(thresholdValue)
			existing.Message = message
			existing.UpdatedAt = time.Now()
			
			e.notify(existing, config)
			e.setCooldown(alertKey, cooldown)
		} else {
			existing.Value = float64(value)
			existing.UpdatedAt = time.Now()
		}
	}
}

// ============================================================================
// Traffic Alert Detection
// ============================================================================

func (e *AlertEngine) checkTrafficAlerts(servers []serverState, config *AlertConfig) {
	rule := config.Rules.Traffic
	
	// Get server names map
	serverNames := make(map[string]string)
	for _, s := range servers {
		serverNames[s.ID] = s.Name
	}
	
	for _, limit := range rule.Limits {
		serverID := limit.ServerID
		serverName := serverNames[serverID]
		if serverName == "" {
			serverName = serverID
		}
		
		// Get traffic from traffic manager (persistent data)
		var trafficGB float64
		if trafficManager != nil {
			txGB, rxGB, found := trafficManager.GetTrafficForAlert(serverID)
			if !found {
				// Fallback to real-time data from server state
				for _, s := range servers {
					if s.ID == serverID {
						txGB = float64(s.TrafficTx) / (1024 * 1024 * 1024)
						rxGB = float64(s.TrafficRx) / (1024 * 1024 * 1024)
						break
					}
				}
			}
			
			// Calculate traffic based on type
			switch limit.Type {
			case "sum":
				trafficGB = txGB + rxGB
			case "up":
				trafficGB = txGB
			case "down":
				trafficGB = rxGB
			case "max":
				if rxGB > txGB {
					trafficGB = rxGB
				} else {
					trafficGB = txGB
				}
			default:
				trafficGB = txGB + rxGB
			}
		} else {
			// Fallback: use real-time data
			for _, s := range servers {
				if s.ID == serverID {
					switch limit.Type {
					case "sum":
						trafficGB = float64(s.TrafficRx+s.TrafficTx) / (1024 * 1024 * 1024)
					case "up":
						trafficGB = float64(s.TrafficTx) / (1024 * 1024 * 1024)
					case "down":
						trafficGB = float64(s.TrafficRx) / (1024 * 1024 * 1024)
					case "max":
						rx := float64(s.TrafficRx) / (1024 * 1024 * 1024)
						tx := float64(s.TrafficTx) / (1024 * 1024 * 1024)
						if rx > tx {
							trafficGB = rx
						} else {
							trafficGB = tx
						}
					default:
						trafficGB = float64(s.TrafficRx+s.TrafficTx) / (1024 * 1024 * 1024)
					}
					break
				}
			}
		}
		
		// Calculate percentage
		percent := (trafficGB / limit.MonthlyGB) * 100
		warningThreshold := limit.Warning
		if warningThreshold == 0 {
			warningThreshold = 80
		}
		
		alertKey := fmt.Sprintf("traffic:%s", serverID)
		
		if percent >= float64(warningThreshold) {
			// Check cooldown (traffic alerts use hours)
			cooldownHours := rule.Cooldown
			if cooldownHours == 0 {
				cooldownHours = 24
			}
			if !e.checkCooldown(alertKey, cooldownHours*3600) {
				continue
			}
			
			var severity string
			if percent >= 100 {
				severity = "critical"
			} else {
				severity = "warning"
			}
			
			message := fmt.Sprintf("服务器 %s 本月流量已使用 %.2f GB，达到限额 %.2f GB 的 %.1f%%", 
				serverName, trafficGB, limit.MonthlyGB, percent)
			
			alert := &AlertState{
				ID:         GenerateRandomString(16),
				Type:       "traffic",
				ServerID:   serverID,
				ServerName: serverName,
				Severity:   severity,
				Status:     "firing",
				Value:      trafficGB,
				Threshold:  limit.MonthlyGB,
				Message:    message,
				StartedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			
			e.alertsMu.Lock()
			e.activeAlerts[alertKey] = alert
			e.alertsMu.Unlock()
			
			e.notify(alert, config)
			e.setCooldown(alertKey, cooldownHours*3600)
		}
	}
}

// ============================================================================
// Expiry Alert Detection
// ============================================================================

func (e *AlertEngine) checkExpiryAlerts(config *AlertConfig) {
	rule := config.Rules.Expiry
	
	e.state.ConfigMu.RLock()
	servers := e.state.Config.Servers
	e.state.ConfigMu.RUnlock()
	
	now := time.Now()
	
	// Check remote servers
	for _, server := range servers {
		if server.ExpiryDate == "" {
			continue
		}
		
		// Skip if server is in exclude list
		if contains(rule.Exclude, server.ID) {
			continue
		}
		// Skip if servers list is specified and server is not in it
		if len(rule.Servers) > 0 && !contains(rule.Servers, server.ID) {
			continue
		}
		// Skip if auto-renew is enabled and exclude_auto is true
		if rule.ExcludeAuto && server.AutoRenew {
			continue
		}
		
		e.checkServerExpiry(server.ID, server.Name, server.ExpiryDate, server.Provider, server.PriceAmount, server.PriceCurrency, server.PricePeriod, now, config)
	}
}

func (e *AlertEngine) checkServerExpiry(serverID, serverName, expiryDateStr, provider, priceAmount, priceCurrency, pricePeriod string, now time.Time, config *AlertConfig) {
	rule := config.Rules.Expiry
	
	// Parse expiry date (support multiple formats)
	var expiryDate time.Time
	var err error
	formats := []string{"2006-01-02", "2006-01-02T15:04:05Z", time.RFC3339}
	for _, format := range formats {
		expiryDate, err = time.Parse(format, expiryDateStr)
		if err == nil {
			break
		}
	}
	if err != nil {
		return // Skip if date is invalid
	}
	
	// Calculate days until expiry
	daysLeft := int(expiryDate.Sub(now).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}
	
	// Check if we should alert based on days_before thresholds
	shouldAlert := false
	for _, threshold := range rule.DaysBefore {
		if daysLeft <= threshold && daysLeft >= 0 {
			shouldAlert = true
			break
		}
	}
	
	if !shouldAlert {
		return
	}
	
	alertKey := fmt.Sprintf("expiry:%s:%d", serverID, daysLeft)
	
	// Check cooldown - use 24 hours per day threshold to avoid duplicate notifications
	if !e.checkCooldown(alertKey, 24*3600) {
		return
	}
	
	// Format price display
	priceDisplay := ""
	if priceAmount != "" {
		currency := priceCurrency
		if currency == "" {
			currency = "USD"
		}
		period := pricePeriod
		if period == "" {
			period = "month"
		}
		priceDisplay = fmt.Sprintf("%s %s/%s", priceAmount, currency, period)
	}
	
	// Create alert
	severity := "warning"
	if daysLeft <= 3 {
		severity = "critical"
	}
	
	message := fmt.Sprintf("服务器 %s 将于 %s 到期，剩余 %d 天", serverName, expiryDate.Format("2006-01-02"), daysLeft)
	
	alert := &AlertState{
		ID:         GenerateRandomString(16),
		Type:       "expiry",
		ServerID:   serverID,
		ServerName: serverName,
		Severity:   severity,
		Status:     "firing",
		Value:      float64(daysLeft),
		Threshold:  float64(rule.DaysBefore[0]), // Use first threshold as reference
		Message:    message,
		StartedAt:  now,
		UpdatedAt:  now,
	}
	
	// Render and send notification
	e.notifyExpiry(alert, expiryDate.Format("2006-01-02"), daysLeft, provider, priceDisplay, config)
	e.setCooldown(alertKey, 24*3600) // 24 hour cooldown per day threshold
}

func (e *AlertEngine) notifyExpiry(alert *AlertState, expiryDate string, daysLeft int, provider, price string, config *AlertConfig) {
	rule := config.Rules.Expiry
	channelIDs := rule.Channels
	
	// Use all channels if none specified
	if len(channelIDs) == 0 {
		for _, ch := range config.Channels {
			if ch.Enabled {
				channelIDs = append(channelIDs, ch.ID)
			}
		}
	}
	
	// Render message from template
	tmpl, ok := config.Templates["expiry"]
	if !ok {
		tmpl = AlertTemplate{
			Title:  "[提醒] {{ .ServerName }} 即将到期",
			Body:   "服务器 {{ .ServerName }} 将于 {{ .ExpiryDate }} 到期，剩余 {{ .DaysLeft }} 天。\n服务商: {{ .Provider }}\n价格: {{ .Price }}",
			Format: "text",
		}
	}
	
	data := map[string]interface{}{
		"Severity":   alert.Severity,
		"ServerName": alert.ServerName,
		"ServerID":   alert.ServerID,
		"ExpiryDate": expiryDate,
		"DaysLeft":   daysLeft,
		"Provider":   provider,
		"Price":      price,
	}
	
	title := renderTemplateString(tmpl.Title, data)
	body := renderTemplateString(tmpl.Body, data)
	
	// Send to each channel
	for _, chID := range channelIDs {
		var channel *NotificationChannel
		for _, ch := range config.Channels {
			if ch.ID == chID && ch.Enabled {
				channel = &ch
				break
			}
		}
		if channel == nil {
			continue
		}
		
		notifier, err := CreateNotifier(*channel)
		if err != nil {
			fmt.Printf("⚠️ Failed to create notifier for channel %s: %v\n", channel.Name, err)
			continue
		}
		
		if err := notifier.Send(title, body); err != nil {
			fmt.Printf("⚠️ Failed to send expiry notification via %s: %v\n", channel.Name, err)
		} else {
			fmt.Printf("📢 Expiry notification sent via %s: %s\n", channel.Name, title)
		}
	}
	
	now := time.Now()
	alert.NotifiedAt = &now
}

// ============================================================================
// Notification Helpers
// ============================================================================

func (e *AlertEngine) notify(alert *AlertState, config *AlertConfig) {
	// Find channels for this alert type
	var channelIDs []string
	switch alert.Type {
	case "offline":
		channelIDs = config.Rules.Offline.Channels
	case "cpu", "memory", "disk":
		channelIDs = config.Rules.Load.Channels
	case "traffic":
		channelIDs = config.Rules.Traffic.Channels
	}
	
	// Use all channels if none specified
	if len(channelIDs) == 0 {
		for _, ch := range config.Channels {
			if ch.Enabled {
				channelIDs = append(channelIDs, ch.ID)
			}
		}
	}
	
	// Render message from template
	title, body := e.renderTemplate(alert, config)
	
	// Send to each channel
	for _, chID := range channelIDs {
		var channel *NotificationChannel
		for _, ch := range config.Channels {
			if ch.ID == chID && ch.Enabled {
				channel = &ch
				break
			}
		}
		if channel == nil {
			continue
		}
		
		notifier, err := CreateNotifier(*channel)
		if err != nil {
			fmt.Printf("⚠️ Failed to create notifier for channel %s: %v\n", channel.Name, err)
			continue
		}
		
		if err := notifier.Send(title, body); err != nil {
			fmt.Printf("⚠️ Failed to send notification via %s: %v\n", channel.Name, err)
		} else {
			fmt.Printf("📢 Alert notification sent via %s: %s\n", channel.Name, title)
		}
	}
	
	now := time.Now()
	alert.NotifiedAt = &now
}

func (e *AlertEngine) notifyRecovery(alert *AlertState, config *AlertConfig) {
	if !config.RecoveryNotify {
		return
	}
	
	// Create recovery template data
	data := map[string]interface{}{
		"ServerName": alert.ServerName,
		"ServerID":   alert.ServerID,
		"AlertType":  getMetricName(alert.Type),
		"Duration":   formatDuration(time.Since(alert.StartedAt)),
	}
	
	// Render recovery template
	tmpl := config.Templates["recovery"]
	title := renderTemplateString(tmpl.Title, data)
	body := renderTemplateString(tmpl.Body, data)
	
	// Find channels for this alert type
	var channelIDs []string
	switch alert.Type {
	case "offline":
		channelIDs = config.Rules.Offline.Channels
	case "cpu", "memory", "disk":
		channelIDs = config.Rules.Load.Channels
	case "traffic":
		channelIDs = config.Rules.Traffic.Channels
	case "expiry":
		channelIDs = config.Rules.Expiry.Channels
	}
	
	if len(channelIDs) == 0 {
		for _, ch := range config.Channels {
			if ch.Enabled {
				channelIDs = append(channelIDs, ch.ID)
			}
		}
	}
	
	for _, chID := range channelIDs {
		var channel *NotificationChannel
		for _, ch := range config.Channels {
			if ch.ID == chID && ch.Enabled {
				channel = &ch
				break
			}
		}
		if channel == nil {
			continue
		}
		
		notifier, err := CreateNotifier(*channel)
		if err != nil {
			continue
		}
		
		if err := notifier.Send(title, body); err != nil {
			fmt.Printf("⚠️ Failed to send recovery notification via %s: %v\n", channel.Name, err)
		} else {
			fmt.Printf("✅ Recovery notification sent via %s: %s\n", channel.Name, title)
		}
	}
}

func (e *AlertEngine) resolveAlert(alertKey string, config *AlertConfig) {
	e.alertsMu.Lock()
	alert := e.activeAlerts[alertKey]
	if alert == nil {
		e.alertsMu.Unlock()
		return
	}
	
	now := time.Now()
	alert.Status = "resolved"
	alert.ResolvedAt = &now
	delete(e.activeAlerts, alertKey)
	e.alertsMu.Unlock()
	
	// Send recovery notification
	e.notifyRecovery(alert, config)
	
	// Store in history
	e.storeAlertHistory(alert)
}

func (e *AlertEngine) storeAlertHistory(alert *AlertState) {
	if e.db == nil {
		return
	}
	
	var duration int64
	if alert.ResolvedAt != nil {
		duration = int64(alert.ResolvedAt.Sub(alert.StartedAt).Seconds())
	}
	
	dbWriter.WriteAsync(func(db *sql.DB) error {
		_, err := db.Exec(`
			INSERT INTO alert_history (alert_id, type, server_id, server_name, severity, value, threshold, message, started_at, resolved_at, duration, notified)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			alert.ID, alert.Type, alert.ServerID, alert.ServerName,
			alert.Severity, alert.Value, alert.Threshold, alert.Message,
			alert.StartedAt.Format(time.RFC3339),
			formatNullableTime(alert.ResolvedAt),
			duration,
			alert.NotifiedAt != nil,
		)
		return err
	})
}

func (e *AlertEngine) renderTemplate(alert *AlertState, config *AlertConfig) (string, string) {
	tmpl, ok := config.Templates[alert.Type]
	if !ok {
		return alert.Type + " Alert", alert.Message
	}
	
	data := map[string]interface{}{
		"Severity":   alert.Severity,
		"ServerName": alert.ServerName,
		"ServerID":   alert.ServerID,
		"Value":      alert.Value,
		"Threshold":  alert.Threshold,
		"Duration":   formatDuration(time.Since(alert.StartedAt)),
		"LastSeen":   alert.StartedAt.Format("2006-01-02 15:04:05"),
		"AlertType":  getMetricName(alert.Type),
	}
	
	// Calculate percent for traffic alerts
	if alert.Type == "traffic" && alert.Threshold > 0 {
		data["Percent"] = fmt.Sprintf("%.1f", (alert.Value/alert.Threshold)*100)
	}
	
	title := renderTemplateString(tmpl.Title, data)
	body := renderTemplateString(tmpl.Body, data)
	
	return title, body
}

// ============================================================================
// Cooldown Management
// ============================================================================

func (e *AlertEngine) checkCooldown(key string, cooldownSeconds int) bool {
	if cooldownSeconds <= 0 {
		return true
	}
	
	e.cooldownsMu.RLock()
	lastNotify, exists := e.cooldowns[key]
	e.cooldownsMu.RUnlock()
	
	if !exists {
		return true
	}
	
	return time.Since(lastNotify) >= time.Duration(cooldownSeconds)*time.Second
}

func (e *AlertEngine) setCooldown(key string, cooldownSeconds int) {
	e.cooldownsMu.Lock()
	e.cooldowns[key] = time.Now()
	e.cooldownsMu.Unlock()
}

// ============================================================================
// Public Methods
// ============================================================================

// GetActiveAlerts returns all current active alerts
func (e *AlertEngine) GetActiveAlerts() []AlertState {
	e.alertsMu.RLock()
	defer e.alertsMu.RUnlock()
	
	alerts := make([]AlertState, 0, len(e.activeAlerts))
	for _, alert := range e.activeAlerts {
		alerts = append(alerts, *alert)
	}
	return alerts
}

// GetAlertStats returns alert statistics
func (e *AlertEngine) GetAlertStats() AlertStats {
	e.alertsMu.RLock()
	defer e.alertsMu.RUnlock()
	
	stats := AlertStats{}
	for _, alert := range e.activeAlerts {
		stats.TotalFiring++
		if alert.Severity == "critical" {
			stats.Critical++
		} else {
			stats.Warning++
		}
	}
	
	// Get server counts
	e.state.ConfigMu.RLock()
	stats.ServersTotal = len(e.state.Config.Servers) + 1 // +1 for local
	e.state.ConfigMu.RUnlock()
	
	e.state.AgentMetricsMu.RLock()
	for _, metrics := range e.state.AgentMetrics {
		if time.Since(metrics.LastUpdated).Seconds() < 30 {
			stats.ServersOnline++
		}
	}
	e.state.AgentMetricsMu.RUnlock()
	stats.ServersOnline++ // Local is always online
	
	return stats
}

// MuteAlert mutes an alert
func (e *AlertEngine) MuteAlert(alertID string) bool {
	e.alertsMu.Lock()
	defer e.alertsMu.Unlock()
	
	for _, alert := range e.activeAlerts {
		if alert.ID == alertID {
			alert.Muted = true
			return true
		}
	}
	return false
}

// ============================================================================
// Helper Functions
// ============================================================================

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getMetricName(metricType string) string {
	switch metricType {
	case "cpu":
		return "CPU"
	case "memory":
		return "内存"
	case "disk":
		return "磁盘"
	case "offline":
		return "离线"
	case "traffic":
		return "流量"
	case "expiry":
		return "到期"
	default:
		return metricType
	}
}

func getSeverityName(severity string) string {
	switch severity {
	case "critical":
		return "严重"
	case "warning":
		return "警告"
	default:
		return severity
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d分钟", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d小时%d分钟", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%d天%d小时", int(d.Hours())/24, int(d.Hours())%24)
}

func formatNullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

func renderTemplateString(tmplStr string, data interface{}) string {
	tmpl, err := template.New("alert").Parse(tmplStr)
	if err != nil {
		return tmplStr
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return tmplStr
	}
	return buf.String()
}
