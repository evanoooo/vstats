package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"vstats/internal/common"

	_ "modernc.org/sqlite"
)

// Granularity bucket intervals (in seconds)
const (
	Bucket5Sec   = 5     // For 1H view: 720 points
	Bucket2Min   = 120   // For 24H view: 720 points
	Bucket15Min  = 900   // For 7D view: 672 points
	BucketHourly = 3600  // For 30D view: 720 points
	BucketDaily  = 86400 // For 1Y view: 365 points
)

// Data retention periods
const (
	Retention5Sec   = 2 * time.Hour   // Keep 5sec data for 2 hours
	Retention2Min   = 26 * time.Hour  // Keep 2min data for 26 hours
	Retention15Min  = 8 * 24 * time.Hour // Keep 15min data for 8 days
	RetentionHourly = 32 * 24 * time.Hour // Keep hourly data for 32 days
	RetentionDaily  = 400 * 24 * time.Hour // Keep daily data for 400 days
)

// LocalStore handles offline metrics storage and aggregation
type LocalStore struct {
	db          *sql.DB
	mu          sync.Mutex
	maxAge      time.Duration // Maximum age of stored metrics (default 24h)
	maxRecords  int           // Maximum number of records to keep
	aggregation time.Duration // Aggregation interval (default 1 minute)
}

// StoredMetrics represents metrics stored locally for later transmission
type StoredMetrics struct {
	ID          int64          `json:"id"`
	Timestamp   time.Time      `json:"timestamp"`
	Metrics     *SystemMetrics `json:"metrics"`
	Aggregated  bool           `json:"aggregated"` // True if this is pre-aggregated data
	SampleCount int            `json:"sample_count"`
}

// AggregatedMetrics represents pre-aggregated metrics for efficient transmission
type AggregatedMetrics struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	SampleCount int       `json:"sample_count"`

	// CPU
	CPUAvg float32 `json:"cpu_avg"`
	CPUMax float32 `json:"cpu_max"`

	// Memory
	MemoryAvg float32 `json:"memory_avg"`
	MemoryMax float32 `json:"memory_max"`

	// Disk (first disk usage)
	DiskAvg float32 `json:"disk_avg"`
	DiskMax float32 `json:"disk_max"`

	// Network - we track max values since they're cumulative counters
	NetRxMax uint64 `json:"net_rx_max"`
	NetTxMax uint64 `json:"net_tx_max"`

	// Load average
	LoadOneAvg     float64 `json:"load_one_avg"`
	LoadFiveAvg    float64 `json:"load_five_avg"`
	LoadFifteenAvg float64 `json:"load_fifteen_avg"`

	// Uptime - use max value
	UptimeMax uint64 `json:"uptime_max"`

	// Last metrics snapshot for static data (hostname, OS info, etc.)
	LastMetrics *SystemMetrics `json:"last_metrics,omitempty"`
}

// NewLocalStore creates a new local storage instance
func NewLocalStore(dataDir string) (*LocalStore, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "metrics_buffer.db")
	db, err := sql.Open("sqlite", dbPath+"?_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better performance
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pending_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			metrics_json TEXT NOT NULL,
			aggregated INTEGER DEFAULT 0,
			sample_count INTEGER DEFAULT 1,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_pending_timestamp ON pending_metrics(timestamp);
		
		-- Track last sent timestamp to avoid duplicates
		CREATE TABLE IF NOT EXISTS sync_state (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		-- 5-second aggregated metrics (for 1H view, keep 2 hours)
		CREATE TABLE IF NOT EXISTS metrics_5sec (
			bucket INTEGER NOT NULL PRIMARY KEY,
			cpu_sum REAL NOT NULL DEFAULT 0,
			cpu_max REAL NOT NULL DEFAULT 0,
			memory_sum REAL NOT NULL DEFAULT 0,
			memory_max REAL NOT NULL DEFAULT 0,
			disk_sum REAL NOT NULL DEFAULT 0,
			net_rx INTEGER NOT NULL DEFAULT 0,
			net_tx INTEGER NOT NULL DEFAULT 0,
			ping_sum REAL NOT NULL DEFAULT 0,
			ping_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0
		) WITHOUT ROWID;

		-- 2-minute aggregated metrics (for 24H view, keep 26 hours)
		CREATE TABLE IF NOT EXISTS metrics_2min (
			bucket INTEGER NOT NULL PRIMARY KEY,
			cpu_sum REAL NOT NULL DEFAULT 0,
			cpu_max REAL NOT NULL DEFAULT 0,
			memory_sum REAL NOT NULL DEFAULT 0,
			memory_max REAL NOT NULL DEFAULT 0,
			disk_sum REAL NOT NULL DEFAULT 0,
			net_rx INTEGER NOT NULL DEFAULT 0,
			net_tx INTEGER NOT NULL DEFAULT 0,
			ping_sum REAL NOT NULL DEFAULT 0,
			ping_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0
		) WITHOUT ROWID;

		-- 15-minute aggregated metrics (for 7D view, keep 8 days)
		CREATE TABLE IF NOT EXISTS metrics_15min (
			bucket INTEGER NOT NULL PRIMARY KEY,
			cpu_sum REAL NOT NULL DEFAULT 0,
			cpu_max REAL NOT NULL DEFAULT 0,
			memory_sum REAL NOT NULL DEFAULT 0,
			memory_max REAL NOT NULL DEFAULT 0,
			disk_sum REAL NOT NULL DEFAULT 0,
			net_rx INTEGER NOT NULL DEFAULT 0,
			net_tx INTEGER NOT NULL DEFAULT 0,
			ping_sum REAL NOT NULL DEFAULT 0,
			ping_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0
		) WITHOUT ROWID;

		-- Hourly aggregated metrics (for 30D view, keep 32 days)
		CREATE TABLE IF NOT EXISTS metrics_hourly (
			bucket INTEGER NOT NULL PRIMARY KEY,
			cpu_sum REAL NOT NULL DEFAULT 0,
			cpu_max REAL NOT NULL DEFAULT 0,
			memory_sum REAL NOT NULL DEFAULT 0,
			memory_max REAL NOT NULL DEFAULT 0,
			disk_sum REAL NOT NULL DEFAULT 0,
			net_rx INTEGER NOT NULL DEFAULT 0,
			net_tx INTEGER NOT NULL DEFAULT 0,
			ping_sum REAL NOT NULL DEFAULT 0,
			ping_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0
		) WITHOUT ROWID;

		-- Daily aggregated metrics (for 1Y view, keep 400 days)
		CREATE TABLE IF NOT EXISTS metrics_daily (
			bucket INTEGER NOT NULL PRIMARY KEY,
			cpu_sum REAL NOT NULL DEFAULT 0,
			cpu_max REAL NOT NULL DEFAULT 0,
			memory_sum REAL NOT NULL DEFAULT 0,
			memory_max REAL NOT NULL DEFAULT 0,
			disk_sum REAL NOT NULL DEFAULT 0,
			net_rx INTEGER NOT NULL DEFAULT 0,
			net_tx INTEGER NOT NULL DEFAULT 0,
			ping_sum REAL NOT NULL DEFAULT 0,
			ping_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0
		) WITHOUT ROWID;

		-- 5-second ping aggregation
		CREATE TABLE IF NOT EXISTS ping_5sec (
			bucket INTEGER NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_max REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (bucket, target_name)
		) WITHOUT ROWID;

		-- 2-minute ping aggregation
		CREATE TABLE IF NOT EXISTS ping_2min (
			bucket INTEGER NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_max REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (bucket, target_name)
		) WITHOUT ROWID;

		-- 15-minute ping aggregation
		CREATE TABLE IF NOT EXISTS ping_15min (
			bucket INTEGER NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_max REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (bucket, target_name)
		) WITHOUT ROWID;

		-- Hourly ping aggregation
		CREATE TABLE IF NOT EXISTS ping_hourly (
			bucket INTEGER NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_max REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (bucket, target_name)
		) WITHOUT ROWID;

		-- Daily ping aggregation
		CREATE TABLE IF NOT EXISTS ping_daily (
			bucket INTEGER NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_max REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (bucket, target_name)
		) WITHOUT ROWID;
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	store := &LocalStore{
		db:          db,
		maxAge:      24 * time.Hour,
		maxRecords:  10000,
		aggregation: 1 * time.Minute,
	}

	// Start background cleanup
	go store.cleanupLoop()

	return store, nil
}

// Store saves metrics to local storage
func (s *LocalStore) Store(metrics *SystemMetrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO pending_metrics (timestamp, metrics_json, aggregated, sample_count)
		VALUES (?, ?, 0, 1)`,
		metrics.Timestamp.Format(time.RFC3339Nano),
		string(data),
	)
	return err
}

// StoreWithAggregation stores metrics and updates all aggregation buckets
func (s *LocalStore) StoreWithAggregation(metrics *SystemMetrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ts := metrics.Timestamp.Unix()

	// Calculate disk usage (use first disk)
	var diskUsage float64
	if len(metrics.Disks) > 0 {
		diskUsage = float64(metrics.Disks[0].UsagePercent)
	}

	// Calculate ping average
	var pingVal float64
	var pingCnt int
	if metrics.Ping != nil && len(metrics.Ping.Targets) > 0 {
		var sum float64
		var count int
		for _, t := range metrics.Ping.Targets {
			if t.LatencyMs != nil {
				sum += *t.LatencyMs
				count++
			}
		}
		if count > 0 {
			pingVal = sum
			pingCnt = count
		}
	}

	cpuUsage := float64(metrics.CPU.Usage)
	memUsage := float64(metrics.Memory.UsagePercent)

	// Update all granularity buckets
	buckets := []struct {
		table    string
		interval int64
	}{
		{"metrics_5sec", Bucket5Sec},
		{"metrics_2min", Bucket2Min},
		{"metrics_15min", Bucket15Min},
		{"metrics_hourly", BucketHourly},
		{"metrics_daily", BucketDaily},
	}

	for _, b := range buckets {
		bucket := ts / b.interval
		s.db.Exec(`
			INSERT INTO `+b.table+` (bucket, cpu_sum, cpu_max, memory_sum, memory_max, disk_sum, net_rx, net_tx, ping_sum, ping_count, sample_count)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
			ON CONFLICT(bucket) DO UPDATE SET
				cpu_sum = cpu_sum + excluded.cpu_sum,
				cpu_max = MAX(cpu_max, excluded.cpu_max),
				memory_sum = memory_sum + excluded.memory_sum,
				memory_max = MAX(memory_max, excluded.memory_max),
				disk_sum = disk_sum + excluded.disk_sum,
				net_rx = MAX(net_rx, excluded.net_rx),
				net_tx = MAX(net_tx, excluded.net_tx),
				ping_sum = ping_sum + excluded.ping_sum,
				ping_count = ping_count + excluded.ping_count,
				sample_count = sample_count + 1`,
			bucket,
			cpuUsage, cpuUsage,
			memUsage, memUsage,
			diskUsage,
			metrics.Network.TotalRx, metrics.Network.TotalTx,
			pingVal, pingCnt,
		)
	}

	// Store ping target aggregations
	if metrics.Ping != nil {
		for _, target := range metrics.Ping.Targets {
			latencyVal := float64(0)
			latencyMax := float64(0)
			latencyCnt := 0
			if target.LatencyMs != nil {
				latencyVal = *target.LatencyMs
				latencyMax = *target.LatencyMs
				latencyCnt = 1
			}
			okCnt := 0
			failCnt := 0
			if target.Status == "ok" {
				okCnt = 1
			} else {
				failCnt = 1
			}

			pingTables := []struct {
				table    string
				interval int64
			}{
				{"ping_5sec", Bucket5Sec},
				{"ping_2min", Bucket2Min},
				{"ping_15min", Bucket15Min},
				{"ping_hourly", BucketHourly},
				{"ping_daily", BucketDaily},
			}

			for _, b := range pingTables {
				bucket := ts / b.interval
				s.db.Exec(`
					INSERT INTO `+b.table+` (bucket, target_name, target_host, latency_sum, latency_max, latency_count, ok_count, fail_count)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
					ON CONFLICT(bucket, target_name) DO UPDATE SET
						target_host = excluded.target_host,
						latency_sum = latency_sum + excluded.latency_sum,
						latency_max = MAX(latency_max, excluded.latency_max),
						latency_count = latency_count + excluded.latency_count,
						ok_count = ok_count + excluded.ok_count,
						fail_count = fail_count + excluded.fail_count`,
					bucket, target.Name, target.Host,
					latencyVal, latencyMax, latencyCnt, okCnt, failCnt,
				)
			}
		}
	}

	return nil
}

// GetAggregatedData retrieves aggregated data for a specific granularity
func (s *LocalStore) GetAggregatedData(granularity string, sinceBucket int64) (*common.GranularityData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var table string
	var interval int

	switch granularity {
	case "5sec":
		table = "metrics_5sec"
		interval = Bucket5Sec
	case "2min":
		table = "metrics_2min"
		interval = Bucket2Min
	case "15min":
		table = "metrics_15min"
		interval = Bucket15Min
	case "hourly":
		table = "metrics_hourly"
		interval = BucketHourly
	case "daily":
		table = "metrics_daily"
		interval = BucketDaily
	default:
		return nil, nil
	}

	data := &common.GranularityData{
		Granularity: granularity,
		Interval:    interval,
	}

	// Query metrics
	rows, err := s.db.Query(`
		SELECT bucket, cpu_sum, cpu_max, memory_sum, memory_max, disk_sum, net_rx, net_tx, ping_sum, ping_count, sample_count
		FROM `+table+`
		WHERE bucket >= ?
		ORDER BY bucket ASC`, sinceBucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var bd common.BucketData
		if err := rows.Scan(&bd.Bucket, &bd.CPUSum, &bd.CPUMax, &bd.MemorySum, &bd.MemoryMax,
			&bd.DiskSum, &bd.NetRx, &bd.NetTx, &bd.PingSum, &bd.PingCount, &bd.SampleCount); err != nil {
			continue
		}
		data.Metrics = append(data.Metrics, bd)
	}

	// Query ping data
	pingTable := "ping_" + granularity
	if granularity == "hourly" || granularity == "daily" {
		pingTable = "ping_" + granularity
	}

	pingRows, err := s.db.Query(`
		SELECT bucket, target_name, target_host, latency_sum, latency_max, latency_count, ok_count, fail_count
		FROM `+pingTable+`
		WHERE bucket >= ?
		ORDER BY bucket ASC`, sinceBucket)
	if err == nil {
		defer pingRows.Close()
		for pingRows.Next() {
			var pd common.PingBucketData
			if err := pingRows.Scan(&pd.Bucket, &pd.TargetName, &pd.TargetHost,
				&pd.LatencySum, &pd.LatencyMax, &pd.LatencyCount, &pd.OkCount, &pd.FailCount); err != nil {
				continue
			}
			data.Ping = append(data.Ping, pd)
		}
	}

	return data, nil
}

// GetAllAggregatedData retrieves all granularity data since specified times
func (s *LocalStore) GetAllAggregatedData() (*common.MultiGranularityMetrics, error) {
	now := time.Now().Unix()

	result := &common.MultiGranularityMetrics{
		Type: "aggregated_metrics",
	}

	// Collect data for each granularity with appropriate time ranges
	granularities := []struct {
		name       string
		interval   int64
		retention  time.Duration
	}{
		{"5sec", Bucket5Sec, Retention5Sec},
		{"2min", Bucket2Min, Retention2Min},
		{"15min", Bucket15Min, Retention15Min},
		{"hourly", BucketHourly, RetentionHourly},
		{"daily", BucketDaily, RetentionDaily},
	}

	for _, g := range granularities {
		sinceBucket := (now - int64(g.retention.Seconds())) / g.interval
		data, err := s.GetAggregatedData(g.name, sinceBucket)
		if err != nil {
			continue
		}
		if data != nil && len(data.Metrics) > 0 {
			result.Granularities = append(result.Granularities, *data)
		}
	}

	return result, nil
}

// GetAggregatedDataSince retrieves data since specified buckets for each granularity
// Used for resumable sync - only sends data the server doesn't have
func (s *LocalStore) GetAggregatedDataSince(lastBuckets map[string]int64) (*common.MultiGranularityMetrics, error) {
	result := &common.MultiGranularityMetrics{
		Type: "aggregated_metrics",
	}

	granularities := []string{"5sec", "2min", "15min", "hourly", "daily"}

	for _, name := range granularities {
		// Get bucket from server's last known position, or 0 to send all
		sinceBucket := lastBuckets[name]
		if sinceBucket > 0 {
			// Start from the next bucket after what server has
			sinceBucket++
		}
		
		data, err := s.GetAggregatedData(name, sinceBucket)
		if err != nil {
			continue
		}
		if data != nil && len(data.Metrics) > 0 {
			result.Granularities = append(result.Granularities, *data)
		}
	}

	return result, nil
}

// StoreAggregated saves pre-aggregated metrics to local storage
func (s *LocalStore) StoreAggregated(agg *AggregatedMetrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(agg)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO pending_metrics (timestamp, metrics_json, aggregated, sample_count)
		VALUES (?, ?, 1, ?)`,
		agg.StartTime.Format(time.RFC3339Nano),
		string(data),
		agg.SampleCount,
	)
	return err
}

// GetPendingCount returns the number of pending metrics
func (s *LocalStore) GetPendingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM pending_metrics").Scan(&count)
	return count
}

// GetPendingMetrics retrieves pending metrics for transmission (up to limit)
func (s *LocalStore) GetPendingMetrics(limit int) ([]StoredMetrics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT id, timestamp, metrics_json, aggregated, sample_count
		FROM pending_metrics
		ORDER BY timestamp ASC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []StoredMetrics
	for rows.Next() {
		var id int64
		var timestamp, metricsJSON string
		var aggregated int
		var sampleCount int

		if err := rows.Scan(&id, &timestamp, &metricsJSON, &aggregated, &sampleCount); err != nil {
			continue
		}

		ts, _ := time.Parse(time.RFC3339Nano, timestamp)

		var metrics SystemMetrics
		if aggregated == 0 {
			if err := json.Unmarshal([]byte(metricsJSON), &metrics); err != nil {
				continue
			}
		}

		results = append(results, StoredMetrics{
			ID:          id,
			Timestamp:   ts,
			Metrics:     &metrics,
			Aggregated:  aggregated == 1,
			SampleCount: sampleCount,
		})
	}

	return results, nil
}

// GetPendingAggregated retrieves pending aggregated metrics
func (s *LocalStore) GetPendingAggregated(limit int) ([]*AggregatedMetrics, []int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT id, metrics_json
		FROM pending_metrics
		WHERE aggregated = 1
		ORDER BY timestamp ASC
		LIMIT ?`, limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var results []*AggregatedMetrics
	var ids []int64
	for rows.Next() {
		var id int64
		var metricsJSON string

		if err := rows.Scan(&id, &metricsJSON); err != nil {
			continue
		}

		var agg AggregatedMetrics
		if err := json.Unmarshal([]byte(metricsJSON), &agg); err != nil {
			continue
		}

		results = append(results, &agg)
		ids = append(ids, id)
	}

	return results, ids, nil
}

// DeleteByIDs removes metrics by their IDs after successful transmission
func (s *LocalStore) DeleteByIDs(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Build query with placeholders
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, id := range ids {
		if _, err := tx.Exec("DELETE FROM pending_metrics WHERE id = ?", id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// AggregateOldMetrics aggregates old raw metrics into 1-minute buckets
func (s *LocalStore) AggregateOldMetrics() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find metrics older than aggregation interval that aren't aggregated
	cutoff := time.Now().Add(-s.aggregation)

	rows, err := s.db.Query(`
		SELECT id, timestamp, metrics_json
		FROM pending_metrics
		WHERE aggregated = 0 AND timestamp < ?
		ORDER BY timestamp ASC`, cutoff.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	defer rows.Close()

	// Group by minute bucket
	buckets := make(map[int64][]*SystemMetrics)
	bucketIDs := make(map[int64][]int64)

	for rows.Next() {
		var id int64
		var timestamp, metricsJSON string

		if err := rows.Scan(&id, &timestamp, &metricsJSON); err != nil {
			continue
		}

		ts, _ := time.Parse(time.RFC3339Nano, timestamp)
		bucketKey := ts.Unix() / 60 // 1-minute buckets

		var metrics SystemMetrics
		if err := json.Unmarshal([]byte(metricsJSON), &metrics); err != nil {
			continue
		}

		buckets[bucketKey] = append(buckets[bucketKey], &metrics)
		bucketIDs[bucketKey] = append(bucketIDs[bucketKey], id)
	}

	// Aggregate each bucket
	for bucketKey, metricsList := range buckets {
		if len(metricsList) < 2 {
			// Not worth aggregating single metrics
			continue
		}

		agg := aggregateMetricsList(metricsList)
		agg.StartTime = time.Unix(bucketKey*60, 0).UTC()
		agg.EndTime = time.Unix((bucketKey+1)*60, 0).UTC()

		// Store aggregated and delete raw
		data, _ := json.Marshal(agg)

		tx, err := s.db.Begin()
		if err != nil {
			continue
		}

		// Insert aggregated
		tx.Exec(`
			INSERT INTO pending_metrics (timestamp, metrics_json, aggregated, sample_count)
			VALUES (?, ?, 1, ?)`,
			agg.StartTime.Format(time.RFC3339Nano),
			string(data),
			agg.SampleCount,
		)

		// Delete raw metrics
		for _, id := range bucketIDs[bucketKey] {
			tx.Exec("DELETE FROM pending_metrics WHERE id = ?", id)
		}

		tx.Commit()
	}

	return nil
}

// aggregateMetricsList creates an AggregatedMetrics from a list of SystemMetrics
func aggregateMetricsList(metrics []*SystemMetrics) *AggregatedMetrics {
	if len(metrics) == 0 {
		return nil
	}

	agg := &AggregatedMetrics{
		SampleCount: len(metrics),
		LastMetrics: metrics[len(metrics)-1],
	}

	var cpuSum, memSum, diskSum float32
	var loadOneSum, loadFiveSum, loadFifteenSum float64

	for _, m := range metrics {
		// CPU
		cpuSum += m.CPU.Usage
		if m.CPU.Usage > agg.CPUMax {
			agg.CPUMax = m.CPU.Usage
		}

		// Memory
		memSum += m.Memory.UsagePercent
		if m.Memory.UsagePercent > agg.MemoryMax {
			agg.MemoryMax = m.Memory.UsagePercent
		}

		// Disk (first disk)
		if len(m.Disks) > 0 {
			diskSum += m.Disks[0].UsagePercent
			if m.Disks[0].UsagePercent > agg.DiskMax {
				agg.DiskMax = m.Disks[0].UsagePercent
			}
		}

		// Network (cumulative - use max)
		if m.Network.TotalRx > agg.NetRxMax {
			agg.NetRxMax = m.Network.TotalRx
		}
		if m.Network.TotalTx > agg.NetTxMax {
			agg.NetTxMax = m.Network.TotalTx
		}

		// Load average
		loadOneSum += m.LoadAverage.One
		loadFiveSum += m.LoadAverage.Five
		loadFifteenSum += m.LoadAverage.Fifteen

		// Uptime (use max)
		if m.Uptime > agg.UptimeMax {
			agg.UptimeMax = m.Uptime
		}
	}

	n := float32(len(metrics))
	agg.CPUAvg = cpuSum / n
	agg.MemoryAvg = memSum / n
	agg.DiskAvg = diskSum / n
	agg.LoadOneAvg = loadOneSum / float64(len(metrics))
	agg.LoadFiveAvg = loadFiveSum / float64(len(metrics))
	agg.LoadFifteenAvg = loadFifteenSum / float64(len(metrics))

	return agg
}

// cleanupLoop periodically cleans up old data
func (s *LocalStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
		s.AggregateOldMetrics()
	}
}

// cleanup removes old metrics and enforces limits
func (s *LocalStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete metrics older than maxAge
	cutoff := time.Now().Add(-s.maxAge).Format(time.RFC3339Nano)
	s.db.Exec("DELETE FROM pending_metrics WHERE timestamp < ?", cutoff)

	// Enforce max records limit
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM pending_metrics").Scan(&count)
	if count > s.maxRecords {
		excess := count - s.maxRecords
		s.db.Exec(`
			DELETE FROM pending_metrics WHERE id IN (
				SELECT id FROM pending_metrics ORDER BY timestamp ASC LIMIT ?
			)`, excess)
	}

	// Clean up aggregation tables based on retention periods
	now := time.Now().Unix()

	// 5sec: keep for 2 hours
	cutoff5sec := (now - int64(Retention5Sec.Seconds())) / Bucket5Sec
	s.db.Exec("DELETE FROM metrics_5sec WHERE bucket < ?", cutoff5sec)
	s.db.Exec("DELETE FROM ping_5sec WHERE bucket < ?", cutoff5sec)

	// 2min: keep for 26 hours
	cutoff2min := (now - int64(Retention2Min.Seconds())) / Bucket2Min
	s.db.Exec("DELETE FROM metrics_2min WHERE bucket < ?", cutoff2min)
	s.db.Exec("DELETE FROM ping_2min WHERE bucket < ?", cutoff2min)

	// 15min: keep for 8 days
	cutoff15min := (now - int64(Retention15Min.Seconds())) / Bucket15Min
	s.db.Exec("DELETE FROM metrics_15min WHERE bucket < ?", cutoff15min)
	s.db.Exec("DELETE FROM ping_15min WHERE bucket < ?", cutoff15min)

	// hourly: keep for 32 days
	cutoffHourly := (now - int64(RetentionHourly.Seconds())) / BucketHourly
	s.db.Exec("DELETE FROM metrics_hourly WHERE bucket < ?", cutoffHourly)
	s.db.Exec("DELETE FROM ping_hourly WHERE bucket < ?", cutoffHourly)

	// daily: keep for 400 days
	cutoffDaily := (now - int64(RetentionDaily.Seconds())) / BucketDaily
	s.db.Exec("DELETE FROM metrics_daily WHERE bucket < ?", cutoffDaily)
	s.db.Exec("DELETE FROM ping_daily WHERE bucket < ?", cutoffDaily)
}

// SetLastSentTimestamp records the last successfully sent timestamp
func (s *LocalStore) SetLastSentTimestamp(ts time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO sync_state (key, value)
		VALUES ('last_sent', ?)`, ts.Format(time.RFC3339Nano))
	return err
}

// GetLastSentTimestamp returns the last successfully sent timestamp
func (s *LocalStore) GetLastSentTimestamp() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	var value string
	err := s.db.QueryRow(`SELECT value FROM sync_state WHERE key = 'last_sent'`).Scan(&value)
	if err != nil {
		return time.Time{}
	}

	ts, _ := time.Parse(time.RFC3339Nano, value)
	return ts
}

// Close closes the database connection
func (s *LocalStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetDataDir returns the default data directory for the agent
func GetDataDir() string {
	// Check environment variable
	if dir := os.Getenv("VSTATS_DATA_DIR"); dir != "" {
		return dir
	}

	// Platform-specific defaults
	switch {
	case fileExists("/etc/vstats-agent"):
		return "/var/lib/vstats-agent"
	case fileExists("/opt/vstats-agent"):
		return "/opt/vstats-agent/data"
	default:
		// User config directory
		if configDir, err := os.UserConfigDir(); err == nil {
			return filepath.Join(configDir, "vstats-agent", "data")
		}
		return "./data"
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MetricsBuffer manages in-memory buffering before storing to local storage
type MetricsBuffer struct {
	mu       sync.Mutex
	buffer   []*SystemMetrics
	maxSize  int
	store    *LocalStore
	interval time.Duration
	stopCh   chan struct{}
}

// NewMetricsBuffer creates a new metrics buffer
func NewMetricsBuffer(store *LocalStore, flushInterval time.Duration, maxSize int) *MetricsBuffer {
	mb := &MetricsBuffer{
		buffer:   make([]*SystemMetrics, 0, maxSize),
		maxSize:  maxSize,
		store:    store,
		interval: flushInterval,
		stopCh:   make(chan struct{}),
	}

	go mb.flushLoop()
	return mb
}

// Add adds a metrics snapshot to the buffer
func (mb *MetricsBuffer) Add(metrics *SystemMetrics) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	// Deep copy to avoid race conditions
	copied := *metrics
	mb.buffer = append(mb.buffer, &copied)

	// Flush if buffer is full
	if len(mb.buffer) >= mb.maxSize {
		mb.flushLocked()
	}
}

// Flush forces a buffer flush
func (mb *MetricsBuffer) Flush() {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.flushLocked()
}

func (mb *MetricsBuffer) flushLocked() {
	if len(mb.buffer) == 0 {
		return
	}

	// Aggregate if multiple samples
	if len(mb.buffer) > 1 {
		agg := aggregateMetricsList(mb.buffer)
		agg.StartTime = mb.buffer[0].Timestamp
		agg.EndTime = mb.buffer[len(mb.buffer)-1].Timestamp
		if err := mb.store.StoreAggregated(agg); err != nil {
			log.Printf("Failed to store aggregated metrics: %v", err)
		}
	} else if len(mb.buffer) == 1 {
		if err := mb.store.Store(mb.buffer[0]); err != nil {
			log.Printf("Failed to store metrics: %v", err)
		}
	}

	mb.buffer = mb.buffer[:0]
}

func (mb *MetricsBuffer) flushLoop() {
	ticker := time.NewTicker(mb.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mb.Flush()
		case <-mb.stopCh:
			mb.Flush()
			return
		}
	}
}

// Stop stops the buffer flush loop
func (mb *MetricsBuffer) Stop() {
	close(mb.stopCh)
}

