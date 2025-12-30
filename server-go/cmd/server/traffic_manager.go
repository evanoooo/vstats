package main

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// Traffic Manager
// ============================================================================

// TrafficManager handles traffic statistics collection and management
type TrafficManager struct {
	state    *AppState
	db       *sql.DB
	
	// Cache for current period traffic
	stats    map[string]*TrafficStats // server_id -> stats
	statsMu  sync.RWMutex
	
	// Previous network counters for delta calculation
	prevCounters map[string]*networkCounter
	countersMu   sync.Mutex
	
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

type networkCounter struct {
	rx        uint64
	tx        uint64
	timestamp time.Time
}

// Global traffic manager instance
var trafficManager *TrafficManager

// NewTrafficManager creates a new traffic manager
func NewTrafficManager(state *AppState, db *sql.DB) *TrafficManager {
	return &TrafficManager{
		state:        state,
		db:           db,
		stats:        make(map[string]*TrafficStats),
		prevCounters: make(map[string]*networkCounter),
		stopCh:       make(chan struct{}),
	}
}

// Start begins the traffic monitoring
func (m *TrafficManager) Start() {
	// Initialize database tables
	m.initTables()
	
	// Load existing stats from database
	m.loadStats()
	
	// Start the monitoring loop
	m.wg.Add(1)
	go m.monitorLoop()
	
	fmt.Println("📊 Traffic manager started")
}

// Stop gracefully stops the traffic manager
func (m *TrafficManager) Stop() {
	close(m.stopCh)
	m.wg.Wait()
	
	// Save final stats
	m.saveAllStats()
	fmt.Println("📊 Traffic manager stopped")
}

// initTables creates the traffic-related database tables
func (m *TrafficManager) initTables() {
	// Current period traffic stats
	m.db.Exec(`
		CREATE TABLE IF NOT EXISTS traffic_stats (
			server_id TEXT PRIMARY KEY,
			period_start TEXT NOT NULL,
			period_end TEXT NOT NULL,
			reset_day INTEGER NOT NULL DEFAULT 1,
			tx_bytes INTEGER NOT NULL DEFAULT 0,
			rx_bytes INTEGER NOT NULL DEFAULT 0,
			monthly_limit_gb REAL NOT NULL DEFAULT 0,
			threshold_type TEXT NOT NULL DEFAULT 'sum',
			baseline_tx INTEGER NOT NULL DEFAULT 0,
			baseline_rx INTEGER NOT NULL DEFAULT 0,
			baseline_time TEXT,
			last_updated TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	
	// Historical traffic records (archived periods)
	m.db.Exec(`
		CREATE TABLE IF NOT EXISTS traffic_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT NOT NULL,
			period_start TEXT NOT NULL,
			period_end TEXT NOT NULL,
			tx_bytes INTEGER NOT NULL,
			rx_bytes INTEGER NOT NULL,
			monthly_limit_gb REAL NOT NULL DEFAULT 0,
			usage_percent REAL NOT NULL DEFAULT 0,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(server_id, period_start)
		)
	`)
	m.db.Exec("CREATE INDEX IF NOT EXISTS idx_traffic_history_server ON traffic_history(server_id, period_start)")
	
	// Daily traffic for detailed charts
	m.db.Exec(`
		CREATE TABLE IF NOT EXISTS traffic_daily (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT NOT NULL,
			date TEXT NOT NULL,
			tx_bytes INTEGER NOT NULL DEFAULT 0,
			rx_bytes INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0,
			UNIQUE(server_id, date)
		)
	`)
	m.db.Exec("CREATE INDEX IF NOT EXISTS idx_traffic_daily_server ON traffic_daily(server_id, date)")
}

// loadStats loads current traffic stats from database
func (m *TrafficManager) loadStats() {
	rows, err := m.db.Query(`
		SELECT server_id, period_start, period_end, reset_day, tx_bytes, rx_bytes,
		       monthly_limit_gb, threshold_type, baseline_tx, baseline_rx, baseline_time, last_updated
		FROM traffic_stats
	`)
	if err != nil {
		fmt.Printf("⚠️ Failed to load traffic stats: %v\n", err)
		return
	}
	defer rows.Close()
	
	m.statsMu.Lock()
	defer m.statsMu.Unlock()
	
	now := time.Now()
	for rows.Next() {
		var stats TrafficStats
		var periodStart, periodEnd, lastUpdated string
		var baselineTime *string
		
		err := rows.Scan(
			&stats.ServerID, &periodStart, &periodEnd, &stats.ResetDay,
			&stats.TxBytes, &stats.RxBytes, &stats.MonthlyLimitGB,
			&stats.ThresholdType, &stats.BaselineTx, &stats.BaselineRx, &baselineTime, &lastUpdated,
		)
		if err != nil {
			continue
		}
		
		stats.PeriodStart, _ = time.Parse(time.RFC3339, periodStart)
		stats.PeriodEnd, _ = time.Parse(time.RFC3339, periodEnd)
		stats.LastUpdated, _ = time.Parse(time.RFC3339, lastUpdated)
		if baselineTime != nil && *baselineTime != "" {
			stats.BaselineTime, _ = time.Parse(time.RFC3339, *baselineTime)
		}
		
		// Check if period needs reset
		if now.After(stats.PeriodEnd) {
			m.archiveAndReset(&stats, now)
		}
		
		// Calculate derived values
		stats.TxBytesGB = BytesToGB(stats.TxBytes)
		stats.RxBytesGB = BytesToGB(stats.RxBytes)
		stats.TotalBytes = stats.TxBytes + stats.RxBytes
		stats.TotalBytesGB = BytesToGB(stats.TotalBytes)
		stats.UsagePercent = stats.CalculatePercent()
		
		m.stats[stats.ServerID] = &stats
	}
}

// monitorLoop runs the main monitoring cycle
func (m *TrafficManager) monitorLoop() {
	defer m.wg.Done()
	
	// Update every minute
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	// Daily save ticker
	saveTicker := time.NewTicker(5 * time.Minute)
	defer saveTicker.Stop()
	
	// Check period reset at midnight
	resetTicker := time.NewTicker(time.Hour)
	defer resetTicker.Stop()
	
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.collectTraffic()
		case <-saveTicker.C:
			m.saveAllStats()
		case <-resetTicker.C:
			m.checkPeriodResets()
		}
	}
}

// collectTraffic collects current traffic from all servers
func (m *TrafficManager) collectTraffic() {
	m.state.ConfigMu.RLock()
	config := m.state.Config
	m.state.ConfigMu.RUnlock()
	
	m.state.AgentMetricsMu.RLock()
	agentMetrics := m.state.AgentMetrics
	m.state.AgentMetricsMu.RUnlock()
	
	now := time.Now()
	today := now.Format("2006-01-02")
	
	// Collect from remote servers
	for _, server := range config.Servers {
		metrics, ok := agentMetrics[server.ID]
		if !ok || time.Since(metrics.LastUpdated) > 60*time.Second {
			continue
		}
		
		m.updateServerTraffic(server.ID, metrics.Metrics.Network.TotalTx, metrics.Metrics.Network.TotalRx, now, today)
	}
}

// updateServerTraffic updates traffic for a single server
func (m *TrafficManager) updateServerTraffic(serverID string, currentTx, currentRx uint64, now time.Time, today string) {
	m.countersMu.Lock()
	prev := m.prevCounters[serverID]
	m.prevCounters[serverID] = &networkCounter{
		tx:        currentTx,
		rx:        currentRx,
		timestamp: now,
	}
	m.countersMu.Unlock()
	
	// Skip first reading (no previous to compare)
	if prev == nil {
		return
	}
	
	// Calculate delta (handle counter reset)
	var deltaTx, deltaRx uint64
	if currentTx >= prev.tx {
		deltaTx = currentTx - prev.tx
	}
	if currentRx >= prev.rx {
		deltaRx = currentRx - prev.rx
	}
	
	// Skip if no traffic
	if deltaTx == 0 && deltaRx == 0 {
		return
	}
	
	// Filter out unreasonably large jumps (likely counter reset or system restart)
	// Max 100GB per minute is unrealistic
	maxDelta := uint64(100 * 1024 * 1024 * 1024)
	if deltaTx > maxDelta || deltaRx > maxDelta {
		return
	}
	
	m.statsMu.Lock()
	stats := m.stats[serverID]
	if stats == nil {
		// Initialize new stats for this server
		periodStart, periodEnd := GetPeriodBounds(1, now)
		stats = &TrafficStats{
			ServerID:      serverID,
			PeriodStart:   periodStart,
			PeriodEnd:     periodEnd,
			ResetDay:      1,
			ThresholdType: "sum",
			LastUpdated:   now,
		}
		m.stats[serverID] = stats
	}
	
	// Update traffic
	stats.TxBytes += deltaTx
	stats.RxBytes += deltaRx
	stats.TxBytesGB = BytesToGB(stats.TxBytes)
	stats.RxBytesGB = BytesToGB(stats.RxBytes)
	stats.TotalBytes = stats.TxBytes + stats.RxBytes
	stats.TotalBytesGB = BytesToGB(stats.TotalBytes)
	stats.UsagePercent = stats.CalculatePercent()
	stats.LastUpdated = now
	m.statsMu.Unlock()
	
	// Update daily stats
	m.updateDailyTraffic(serverID, deltaTx, deltaRx, today)
}

// updateDailyTraffic updates daily traffic record
func (m *TrafficManager) updateDailyTraffic(serverID string, deltaTx, deltaRx uint64, date string) {
	if dbWriter == nil {
		return
	}
	
	dbWriter.WriteAsync(func(db *sql.DB) error {
		_, err := db.Exec(`
			INSERT INTO traffic_daily (server_id, date, tx_bytes, rx_bytes, sample_count)
			VALUES (?, ?, ?, ?, 1)
			ON CONFLICT(server_id, date) DO UPDATE SET
				tx_bytes = tx_bytes + excluded.tx_bytes,
				rx_bytes = rx_bytes + excluded.rx_bytes,
				sample_count = sample_count + 1
		`, serverID, date, deltaTx, deltaRx)
		return err
	})
}

// checkPeriodResets checks and resets periods that have ended
func (m *TrafficManager) checkPeriodResets() {
	now := time.Now()
	
	m.statsMu.Lock()
	defer m.statsMu.Unlock()
	
	for _, stats := range m.stats {
		if now.After(stats.PeriodEnd) {
			m.archiveAndReset(stats, now)
		}
	}
}

// archiveAndReset archives current period and starts new one
func (m *TrafficManager) archiveAndReset(stats *TrafficStats, now time.Time) {
	// Archive to history
	if stats.TxBytes > 0 || stats.RxBytes > 0 {
		if dbWriter != nil {
			usagePercent := stats.CalculatePercent()
			dbWriter.WriteAsync(func(db *sql.DB) error {
				_, err := db.Exec(`
					INSERT OR REPLACE INTO traffic_history 
					(server_id, period_start, period_end, tx_bytes, rx_bytes, monthly_limit_gb, usage_percent)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`,
					stats.ServerID,
					stats.PeriodStart.Format(time.RFC3339),
					stats.PeriodEnd.Format(time.RFC3339),
					stats.TxBytes, stats.RxBytes,
					stats.MonthlyLimitGB, usagePercent,
				)
				return err
			})
		}
	}
	
	// Calculate new period
	periodStart, periodEnd := GetPeriodBounds(stats.ResetDay, now)
	
	// Reset stats
	stats.PeriodStart = periodStart
	stats.PeriodEnd = periodEnd
	stats.TxBytes = 0
	stats.RxBytes = 0
	stats.TxBytesGB = 0
	stats.RxBytesGB = 0
	stats.TotalBytes = 0
	stats.TotalBytesGB = 0
	stats.UsagePercent = 0
	stats.BaselineTx = 0
	stats.BaselineRx = 0
	stats.BaselineTime = time.Time{}
	stats.LastUpdated = now
	
	fmt.Printf("📊 Traffic period reset for server %s, new period: %s to %s\n",
		stats.ServerID,
		periodStart.Format("2006-01-02"),
		periodEnd.Format("2006-01-02"))
}

// saveAllStats saves all stats to database
func (m *TrafficManager) saveAllStats() {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()
	
	if dbWriter == nil {
		return
	}
	
	for _, stats := range m.stats {
		s := *stats // Copy to avoid race
		dbWriter.WriteAsync(func(db *sql.DB) error {
			var baselineTime *string
			if !s.BaselineTime.IsZero() {
				t := s.BaselineTime.Format(time.RFC3339)
				baselineTime = &t
			}
			
			_, err := db.Exec(`
				INSERT INTO traffic_stats 
				(server_id, period_start, period_end, reset_day, tx_bytes, rx_bytes,
				 monthly_limit_gb, threshold_type, baseline_tx, baseline_rx, baseline_time, last_updated)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(server_id) DO UPDATE SET
					period_start = excluded.period_start,
					period_end = excluded.period_end,
					reset_day = excluded.reset_day,
					tx_bytes = excluded.tx_bytes,
					rx_bytes = excluded.rx_bytes,
					monthly_limit_gb = excluded.monthly_limit_gb,
					threshold_type = excluded.threshold_type,
					baseline_tx = excluded.baseline_tx,
					baseline_rx = excluded.baseline_rx,
					baseline_time = excluded.baseline_time,
					last_updated = excluded.last_updated
			`,
				s.ServerID,
				s.PeriodStart.Format(time.RFC3339),
				s.PeriodEnd.Format(time.RFC3339),
				s.ResetDay, s.TxBytes, s.RxBytes,
				s.MonthlyLimitGB, s.ThresholdType,
				s.BaselineTx, s.BaselineRx, baselineTime,
				s.LastUpdated.Format(time.RFC3339),
			)
			return err
		})
	}
}

// ============================================================================
// Public API Methods
// ============================================================================

// GetAllStats returns all current traffic stats
func (m *TrafficManager) GetAllStats() []TrafficStats {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()
	
	// Get server names
	m.state.ConfigMu.RLock()
	config := m.state.Config
	m.state.ConfigMu.RUnlock()
	
	serverNames := make(map[string]string)
	for _, server := range config.Servers {
		serverNames[server.ID] = server.Name
	}
	
	stats := make([]TrafficStats, 0, len(m.stats))
	for _, s := range m.stats {
		stat := *s
		stat.ServerName = serverNames[stat.ServerID]
		stats = append(stats, stat)
	}
	return stats
}

// GetServerStats returns traffic stats for a specific server
func (m *TrafficManager) GetServerStats(serverID string) *TrafficStats {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()
	
	if stats, ok := m.stats[serverID]; ok {
		s := *stats
		return &s
	}
	return nil
}

// UpdateLimit updates the traffic limit for a server
func (m *TrafficManager) UpdateLimit(serverID string, monthlyLimitGB float64, thresholdType string, resetDay int) error {
	if resetDay < 1 || resetDay > 28 {
		resetDay = 1
	}
	if !ValidateThresholdType(thresholdType) {
		thresholdType = "sum"
	}
	
	now := time.Now()
	
	m.statsMu.Lock()
	stats := m.stats[serverID]
	if stats == nil {
		// Create new stats
		periodStart, periodEnd := GetPeriodBounds(resetDay, now)
		stats = &TrafficStats{
			ServerID:      serverID,
			PeriodStart:   periodStart,
			PeriodEnd:     periodEnd,
			ResetDay:      resetDay,
			ThresholdType: thresholdType,
			MonthlyLimitGB: monthlyLimitGB,
			LastUpdated:   now,
		}
		m.stats[serverID] = stats
	} else {
		// Update existing
		oldResetDay := stats.ResetDay
		stats.MonthlyLimitGB = monthlyLimitGB
		stats.ThresholdType = thresholdType
		stats.ResetDay = resetDay
		
		// If reset day changed, recalculate period
		if oldResetDay != resetDay {
			stats.PeriodStart, stats.PeriodEnd = GetPeriodBounds(resetDay, now)
		}
		
		stats.UsagePercent = stats.CalculatePercent()
		stats.LastUpdated = now
	}
	m.statsMu.Unlock()
	
	// Save immediately
	m.saveAllStats()
	
	return nil
}

// ResetTraffic resets traffic for a server
func (m *TrafficManager) ResetTraffic(serverID string, toZero bool) error {
	now := time.Now()
	
	m.statsMu.Lock()
	stats := m.stats[serverID]
	if stats == nil {
		m.statsMu.Unlock()
		return fmt.Errorf("server not found: %s", serverID)
	}
	
	if toZero {
		// Reset to zero (start fresh)
		stats.TxBytes = 0
		stats.RxBytes = 0
		stats.TxBytesGB = 0
		stats.RxBytesGB = 0
		stats.TotalBytes = 0
		stats.TotalBytesGB = 0
		stats.UsagePercent = 0
	}
	
	// Update baseline
	m.countersMu.Lock()
	if prev := m.prevCounters[serverID]; prev != nil {
		stats.BaselineTx = prev.tx
		stats.BaselineRx = prev.rx
		stats.BaselineTime = now
	}
	m.countersMu.Unlock()
	
	stats.LastUpdated = now
	m.statsMu.Unlock()
	
	// Save immediately
	m.saveAllStats()
	
	return nil
}

// GetHistory returns historical traffic records for a server
func (m *TrafficManager) GetHistory(serverID string, limit int) ([]TrafficRecord, error) {
	if limit <= 0 {
		limit = 12 // Default to 12 months
	}
	
	query := `
		SELECT id, server_id, period_start, period_end, tx_bytes, rx_bytes, monthly_limit_gb, usage_percent, created_at
		FROM traffic_history
		WHERE server_id = ?
		ORDER BY period_start DESC
		LIMIT ?
	`
	
	rows, err := m.db.Query(query, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var records []TrafficRecord
	for rows.Next() {
		var r TrafficRecord
		var periodStart, periodEnd, createdAt string
		
		err := rows.Scan(&r.ID, &r.ServerID, &periodStart, &periodEnd,
			&r.TxBytes, &r.RxBytes, &r.MonthlyLimitGB, &r.UsagePercent, &createdAt)
		if err != nil {
			continue
		}
		
		r.PeriodStart, _ = time.Parse(time.RFC3339, periodStart)
		r.PeriodEnd, _ = time.Parse(time.RFC3339, periodEnd)
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		r.TxBytesGB = BytesToGB(r.TxBytes)
		r.RxBytesGB = BytesToGB(r.RxBytes)
		r.TotalBytes = r.TxBytes + r.RxBytes
		r.TotalBytesGB = BytesToGB(r.TotalBytes)
		
		records = append(records, r)
	}
	
	return records, nil
}

// GetDailyTraffic returns daily traffic data for a server
func (m *TrafficManager) GetDailyTraffic(serverID string, days int) ([]DailyTrafficRecord, error) {
	if days <= 0 {
		days = 30
	}
	
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	
	rows, err := m.db.Query(`
		SELECT date, tx_bytes, rx_bytes
		FROM traffic_daily
		WHERE server_id = ? AND date >= ?
		ORDER BY date ASC
	`, serverID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var records []DailyTrafficRecord
	for rows.Next() {
		var r DailyTrafficRecord
		var txBytes, rxBytes uint64
		
		err := rows.Scan(&r.Date, &txBytes, &rxBytes)
		if err != nil {
			continue
		}
		
		r.TxBytesGB = BytesToGB(txBytes)
		r.RxBytesGB = BytesToGB(rxBytes)
		r.TotalGB = r.TxBytesGB + r.RxBytesGB
		
		records = append(records, r)
	}
	
	return records, nil
}

// GetSummary returns a summary of all traffic stats
func (m *TrafficManager) GetSummary() TrafficSummary {
	stats := m.GetAllStats()
	
	summary := TrafficSummary{
		TotalServers: len(stats),
		Stats:        stats,
	}
	
	for _, s := range stats {
		if s.MonthlyLimitGB > 0 {
			summary.ServersWithLimit++
			if s.UsagePercent >= 100 {
				summary.OverLimitCount++
			} else if s.UsagePercent >= 80 {
				summary.WarningCount++
			}
		}
	}
	
	return summary
}

// GetTrafficForAlert returns traffic data for alert engine
func (m *TrafficManager) GetTrafficForAlert(serverID string) (txGB, rxGB float64, found bool) {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()
	
	if stats, ok := m.stats[serverID]; ok {
		return stats.TxBytesGB, stats.RxBytesGB, true
	}
	return 0, 0, false
}
