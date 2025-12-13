package main

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"vstats/internal/common"

	_ "modernc.org/sqlite"
)

// DBWriter serializes all database write operations through a channel
type DBWriter struct {
	db       *sql.DB
	writeCh  chan writeJob
	done     chan struct{}
	wg       sync.WaitGroup
}

type writeJob struct {
	fn     func(*sql.DB) error
	result chan error // nil for fire-and-forget
}

// Global DBWriter instance
var dbWriter *DBWriter

// ============================================================================
// Aggregation Buffer for batch writes
// ============================================================================

// AggBufferKey uniquely identifies a metrics bucket
type AggBufferKey struct {
	ServerID    string
	Granularity string
	Bucket      int64
}

// PingBufferKey uniquely identifies a ping bucket
type PingBufferKey struct {
	ServerID    string
	Granularity string
	Bucket      int64
	TargetName  string
}

// AggBuffer accumulates aggregated metrics for batch writing
type AggBuffer struct {
	mu          sync.Mutex
	metrics     map[AggBufferKey]*common.BucketData
	ping        map[PingBufferKey]*common.PingBucketData
	flushTicker *time.Ticker
	done        chan struct{}
}

// Global aggregation buffer
var aggBuffer *AggBuffer

// ============================================================================
// Metrics Buffer for batch real-time metrics writes
// ============================================================================

// MetricsBufferItem represents a single metrics item in buffer
type MetricsBufferItem struct {
	ServerID string
	Metrics  *SystemMetrics
}

// MetricsBuffer accumulates real-time metrics for batch writing
type MetricsBuffer struct {
	mu          sync.Mutex
	items       []MetricsBufferItem
	flushTicker *time.Ticker
	done        chan struct{}
	maxSize     int
}

// Global metrics buffer
var metricsBuffer *MetricsBuffer

// NewMetricsBuffer creates a new metrics buffer
func NewMetricsBuffer(flushInterval time.Duration, maxSize int) *MetricsBuffer {
	mb := &MetricsBuffer{
		items:       make([]MetricsBufferItem, 0, maxSize),
		flushTicker: time.NewTicker(flushInterval),
		done:        make(chan struct{}),
		maxSize:     maxSize,
	}
	go mb.flushLoop()
	return mb
}

// Add adds a metrics item to the buffer
func (mb *MetricsBuffer) Add(serverID string, metrics *SystemMetrics) {
	mb.mu.Lock()
	
	// Copy metrics to avoid race conditions
	copied := *metrics
	mb.items = append(mb.items, MetricsBufferItem{
		ServerID: serverID,
		Metrics:  &copied,
	})
	
	// Force flush if buffer is full
	if len(mb.items) >= mb.maxSize {
		items := mb.items
		mb.items = make([]MetricsBufferItem, 0, mb.maxSize)
		mb.mu.Unlock()
		mb.flushItems(items)
		return
	}
	
	mb.mu.Unlock()
}

// flushLoop periodically flushes the buffer
func (mb *MetricsBuffer) flushLoop() {
	for {
		select {
		case <-mb.flushTicker.C:
			mb.Flush()
		case <-mb.done:
			mb.Flush()
			return
		}
	}
}

// Flush writes all buffered data to database
func (mb *MetricsBuffer) Flush() {
	mb.mu.Lock()
	if len(mb.items) == 0 {
		mb.mu.Unlock()
		return
	}
	
	items := mb.items
	mb.items = make([]MetricsBufferItem, 0, mb.maxSize)
	mb.mu.Unlock()
	
	mb.flushItems(items)
}

// flushItems writes items to database
func (mb *MetricsBuffer) flushItems(items []MetricsBufferItem) {
	if len(items) == 0 || dbWriter == nil {
		return
	}
	
	dbWriter.WriteAsync(func(db *sql.DB) error {
		return batchStoreMetrics(db, items)
	})
}

// Close stops the buffer
func (mb *MetricsBuffer) Close() {
	mb.flushTicker.Stop()
	close(mb.done)
}

// GetLastMetricsTime returns the last metrics timestamp for a server
func GetLastMetricsTime(serverID string) *time.Time {
	if dbWriter == nil {
		return nil
	}
	
	db := dbWriter.GetDB()
	
	// Check multiple tables to find the latest timestamp
	var lastTime *time.Time
	
	// Check metrics_raw first (most recent data)
	var timestamp string
	err := db.QueryRow(`
		SELECT timestamp FROM metrics_raw 
		WHERE server_id = ? 
		ORDER BY timestamp DESC 
		LIMIT 1`, serverID).Scan(&timestamp)
	
	if err == nil && timestamp != "" {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			lastTime = &t
		}
	}
	
	// Also check aggregation tables for the latest bucket
	var bucket5sec, bucket2min int64
	db.QueryRow(`SELECT MAX(bucket) FROM metrics_5sec WHERE server_id = ?`, serverID).Scan(&bucket5sec)
	db.QueryRow(`SELECT MAX(bucket) FROM metrics_2min WHERE server_id = ?`, serverID).Scan(&bucket2min)
	
	// Convert buckets to time
	if bucket5sec > 0 {
		t := time.Unix(bucket5sec*5, 0).UTC()
		if lastTime == nil || t.After(*lastTime) {
			lastTime = &t
		}
	}
	if bucket2min > 0 {
		t := time.Unix(bucket2min*120, 0).UTC()
		if lastTime == nil || t.After(*lastTime) {
			lastTime = &t
		}
	}
	
	return lastTime
}

// GetLastAggregationBuckets returns the last bucket for each granularity for a server
func GetLastAggregationBuckets(serverID string) map[string]int64 {
	if dbWriter == nil {
		return nil
	}
	
	db := dbWriter.GetDB()
	buckets := make(map[string]int64)
	
	tables := map[string]string{
		"5sec":   "metrics_5sec",
		"2min":   "metrics_2min",
		"15min":  "metrics_15min_agg",
		"hourly": "metrics_hourly_agg",
		"daily":  "metrics_daily_agg",
	}
	
	for granularity, table := range tables {
		var bucket int64
		err := db.QueryRow(`SELECT MAX(bucket) FROM `+table+` WHERE server_id = ?`, serverID).Scan(&bucket)
		if err == nil && bucket > 0 {
			buckets[granularity] = bucket
		}
	}
	
	return buckets
}

// batchStoreMetrics stores multiple metrics in a single transaction
func batchStoreMetrics(db *sql.DB, items []MetricsBufferItem) error {
	if len(items) == 0 {
		return nil
	}
	
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// Prepare statements for batch insert
	rawStmt, err := tx.Prepare(`
		INSERT INTO metrics_raw (server_id, timestamp, cpu_usage, memory_usage, disk_usage, net_rx, net_tx, load_1, load_5, load_15, ping_ms, bucket_5min, bucket_5sec)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer rawStmt.Close()
	
	stmt5sec, err := tx.Prepare(`
		INSERT INTO metrics_5sec (server_id, bucket, cpu_sum, cpu_max, memory_sum, memory_max, disk_sum, net_rx, net_tx, ping_sum, ping_count, sample_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(server_id, bucket) DO UPDATE SET
			cpu_sum = cpu_sum + excluded.cpu_sum,
			cpu_max = MAX(cpu_max, excluded.cpu_max),
			memory_sum = memory_sum + excluded.memory_sum,
			memory_max = MAX(memory_max, excluded.memory_max),
			disk_sum = disk_sum + excluded.disk_sum,
			net_rx = MAX(net_rx, excluded.net_rx),
			net_tx = MAX(net_tx, excluded.net_tx),
			ping_sum = ping_sum + excluded.ping_sum,
			ping_count = ping_count + excluded.ping_count,
			sample_count = sample_count + 1`)
	if err != nil {
		return err
	}
	defer stmt5sec.Close()
	
	stmt2min, err := tx.Prepare(`
		INSERT INTO metrics_2min (server_id, bucket, cpu_sum, cpu_max, memory_sum, memory_max, disk_sum, net_rx, net_tx, ping_sum, ping_count, sample_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(server_id, bucket) DO UPDATE SET
			cpu_sum = cpu_sum + excluded.cpu_sum,
			cpu_max = MAX(cpu_max, excluded.cpu_max),
			memory_sum = memory_sum + excluded.memory_sum,
			memory_max = MAX(memory_max, excluded.memory_max),
			disk_sum = disk_sum + excluded.disk_sum,
			net_rx = MAX(net_rx, excluded.net_rx),
			net_tx = MAX(net_tx, excluded.net_tx),
			ping_sum = ping_sum + excluded.ping_sum,
			ping_count = ping_count + excluded.ping_count,
			sample_count = sample_count + 1`)
	if err != nil {
		return err
	}
	defer stmt2min.Close()
	
	for _, item := range items {
		metrics := item.Metrics
		serverID := item.ServerID
		
		var diskUsage float32 = 0
		if len(metrics.Disks) > 0 {
			diskUsage = metrics.Disks[0].UsagePercent
		}
		
		timestamp := metrics.Timestamp.Format(time.RFC3339)
		bucket5min := metrics.Timestamp.Unix() / 120
		bucket5sec := metrics.Timestamp.Unix() / 5
		
		// Get ping
		var pingMs *float64
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
				avg := sum / float64(count)
				pingMs = &avg
				pingVal = avg
				pingCnt = 1
			}
		}
		
		// Insert raw
		rawStmt.Exec(
			serverID, timestamp,
			metrics.CPU.Usage, metrics.Memory.UsagePercent, diskUsage,
			metrics.Network.TotalRx, metrics.Network.TotalTx,
			metrics.LoadAverage.One, metrics.LoadAverage.Five, metrics.LoadAverage.Fifteen,
			pingMs, bucket5min, bucket5sec,
		)
		
		// Insert to 5sec aggregation
		stmt5sec.Exec(
			serverID, bucket5sec,
			float64(metrics.CPU.Usage), float64(metrics.CPU.Usage),
			float64(metrics.Memory.UsagePercent), float64(metrics.Memory.UsagePercent),
			float64(diskUsage),
			metrics.Network.TotalRx, metrics.Network.TotalTx,
			pingVal, pingCnt,
		)
		
		// Insert to 2min aggregation
		stmt2min.Exec(
			serverID, bucket5min,
			float64(metrics.CPU.Usage), float64(metrics.CPU.Usage),
			float64(metrics.Memory.UsagePercent), float64(metrics.Memory.UsagePercent),
			float64(diskUsage),
			metrics.Network.TotalRx, metrics.Network.TotalTx,
			pingVal, pingCnt,
		)
	}
	
	return tx.Commit()
}

// NewAggBuffer creates a new aggregation buffer
func NewAggBuffer(flushInterval time.Duration) *AggBuffer {
	ab := &AggBuffer{
		metrics:     make(map[AggBufferKey]*common.BucketData),
		ping:        make(map[PingBufferKey]*common.PingBucketData),
		flushTicker: time.NewTicker(flushInterval),
		done:        make(chan struct{}),
	}
	go ab.flushLoop()
	return ab
}

// Add adds aggregated data to the buffer
func (ab *AggBuffer) Add(serverID string, granularities []common.GranularityData) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	for _, g := range granularities {
		// Add metrics
		for _, m := range g.Metrics {
			key := AggBufferKey{
				ServerID:    serverID,
				Granularity: g.Granularity,
				Bucket:      m.Bucket,
			}
			
			if existing, ok := ab.metrics[key]; ok {
				// Merge with existing data - take max values for cumulative, sum for averages
				existing.CPUSum = m.CPUSum // Replace with latest (agent has full picture)
				if m.CPUMax > existing.CPUMax {
					existing.CPUMax = m.CPUMax
				}
				existing.MemorySum = m.MemorySum
				if m.MemoryMax > existing.MemoryMax {
					existing.MemoryMax = m.MemoryMax
				}
				existing.DiskSum = m.DiskSum
				if m.NetRx > existing.NetRx {
					existing.NetRx = m.NetRx
				}
				if m.NetTx > existing.NetTx {
					existing.NetTx = m.NetTx
				}
				existing.PingSum = m.PingSum
				existing.PingCount = m.PingCount
				existing.SampleCount = m.SampleCount
			} else {
				// Copy the data
				copied := m
				ab.metrics[key] = &copied
			}
		}

		// Add ping data
		for _, p := range g.Ping {
			key := PingBufferKey{
				ServerID:    serverID,
				Granularity: g.Granularity,
				Bucket:      p.Bucket,
				TargetName:  p.TargetName,
			}

			if existing, ok := ab.ping[key]; ok {
				// Merge - replace with latest from agent
				existing.TargetHost = p.TargetHost
				existing.LatencySum = p.LatencySum
				if p.LatencyMax > existing.LatencyMax {
					existing.LatencyMax = p.LatencyMax
				}
				existing.LatencyCount = p.LatencyCount
				existing.OkCount = p.OkCount
				existing.FailCount = p.FailCount
			} else {
				copied := p
				ab.ping[key] = &copied
			}
		}
	}
}

// flushLoop periodically flushes the buffer to database
func (ab *AggBuffer) flushLoop() {
	for {
		select {
		case <-ab.flushTicker.C:
			ab.Flush()
		case <-ab.done:
			ab.Flush() // Final flush
			return
		}
	}
}

// Flush writes all buffered data to the database
func (ab *AggBuffer) Flush() {
	ab.mu.Lock()
	metricsCount := len(ab.metrics)
	pingCount := len(ab.ping)
	
	if metricsCount == 0 && pingCount == 0 {
		ab.mu.Unlock()
		return
	}

	// Take ownership of current buffers
	metrics := ab.metrics
	ping := ab.ping
	ab.metrics = make(map[AggBufferKey]*common.BucketData)
	ab.ping = make(map[PingBufferKey]*common.PingBucketData)
	ab.mu.Unlock()

	// Write to database
	if dbWriter != nil {
		dbWriter.WriteAsync(func(db *sql.DB) error {
			err := flushAggBufferToDB(db, metrics, ping)
			if err != nil {
				fmt.Printf("⚠️ Aggregation buffer flush error: %v\n", err)
			}
			return err
		})
	}
}

// Stats returns current buffer statistics
func (ab *AggBuffer) Stats() (metricsCount, pingCount int) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	return len(ab.metrics), len(ab.ping)
}

// Close stops the buffer and flushes remaining data
func (ab *AggBuffer) Close() {
	ab.flushTicker.Stop()
	close(ab.done)
}

// flushAggBufferToDB writes buffered data to database using batch inserts
func flushAggBufferToDB(db *sql.DB, metrics map[AggBufferKey]*common.BucketData, ping map[PingBufferKey]*common.PingBucketData) error {
	if len(metrics) == 0 && len(ping) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Group metrics by granularity for batch insert
	metricsByGranularity := make(map[string][]struct {
		serverID string
		data     *common.BucketData
	})

	for key, data := range metrics {
		metricsByGranularity[key.Granularity] = append(metricsByGranularity[key.Granularity], struct {
			serverID string
			data     *common.BucketData
		}{key.ServerID, data})
	}

	// Batch insert metrics for each granularity
	for granularity, items := range metricsByGranularity {
		table := getMetricsTable(granularity)
		if table == "" {
			continue
		}

		// Build batch insert with UPSERT
		if len(items) > 0 {
			err := batchUpsertMetrics(tx, table, items)
			if err != nil {
				fmt.Printf("Error batch inserting to %s: %v\n", table, err)
			}
		}
	}

	// Group ping by granularity
	pingByGranularity := make(map[string][]struct {
		serverID string
		data     *common.PingBucketData
	})

	for key, data := range ping {
		pingByGranularity[key.Granularity] = append(pingByGranularity[key.Granularity], struct {
			serverID string
			data     *common.PingBucketData
		}{key.ServerID, data})
	}

	// Batch insert ping for each granularity
	for granularity, items := range pingByGranularity {
		table := getPingTable(granularity)
		if table == "" {
			continue
		}

		if len(items) > 0 {
			err := batchUpsertPing(tx, table, items)
			if err != nil {
				fmt.Printf("Error batch inserting to %s: %v\n", table, err)
			}
		}
	}

	return tx.Commit()
}

// getMetricsTable returns the table name for a granularity
func getMetricsTable(granularity string) string {
	switch granularity {
	case "5sec":
		return "metrics_5sec"
	case "2min":
		return "metrics_2min"
	case "15min":
		return "metrics_15min_agg"
	case "hourly":
		return "metrics_hourly_agg"
	case "daily":
		return "metrics_daily_agg"
	default:
		return ""
	}
}

// getPingTable returns the ping table name for a granularity
func getPingTable(granularity string) string {
	switch granularity {
	case "5sec":
		return "ping_5sec"
	case "2min":
		return "ping_2min"
	case "15min":
		return "ping_15min_agg"
	case "hourly":
		return "ping_hourly_agg"
	case "daily":
		return "ping_daily_agg"
	default:
		return ""
	}
}

// batchUpsertMetrics performs batch upsert for metrics
func batchUpsertMetrics(tx *sql.Tx, table string, items []struct {
	serverID string
	data     *common.BucketData
}) error {
	if len(items) == 0 {
		return nil
	}

	// SQLite supports multi-row INSERT, process in chunks
	const chunkSize = 100
	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunk := items[i:end]

		var valueStrings []string
		var valueArgs []interface{}

		for _, item := range chunk {
			valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			valueArgs = append(valueArgs,
				item.serverID, item.data.Bucket,
				item.data.CPUSum, item.data.CPUMax,
				item.data.MemorySum, item.data.MemoryMax,
				item.data.DiskSum,
				item.data.NetRx, item.data.NetTx,
				item.data.PingSum, item.data.PingCount,
				item.data.SampleCount,
			)
		}

		query := fmt.Sprintf(`
			INSERT INTO %s (server_id, bucket, cpu_sum, cpu_max, memory_sum, memory_max, disk_sum, net_rx, net_tx, ping_sum, ping_count, sample_count)
			VALUES %s
			ON CONFLICT(server_id, bucket) DO UPDATE SET
				cpu_sum = excluded.cpu_sum,
				cpu_max = MAX(%s.cpu_max, excluded.cpu_max),
				memory_sum = excluded.memory_sum,
				memory_max = MAX(%s.memory_max, excluded.memory_max),
				disk_sum = excluded.disk_sum,
				net_rx = MAX(%s.net_rx, excluded.net_rx),
				net_tx = MAX(%s.net_tx, excluded.net_tx),
				ping_sum = excluded.ping_sum,
				ping_count = excluded.ping_count,
				sample_count = excluded.sample_count`,
			table, strings.Join(valueStrings, ","), table, table, table, table)

		_, err := tx.Exec(query, valueArgs...)
		if err != nil {
			return err
		}
	}

	return nil
}

// batchUpsertPing performs batch upsert for ping data
func batchUpsertPing(tx *sql.Tx, table string, items []struct {
	serverID string
	data     *common.PingBucketData
}) error {
	if len(items) == 0 {
		return nil
	}

	const chunkSize = 100
	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunk := items[i:end]

		var valueStrings []string
		var valueArgs []interface{}

		for _, item := range chunk {
			valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
			valueArgs = append(valueArgs,
				item.serverID, item.data.Bucket, item.data.TargetName, item.data.TargetHost,
				item.data.LatencySum, item.data.LatencyMax, item.data.LatencyCount,
				item.data.OkCount, item.data.FailCount,
			)
		}

		query := fmt.Sprintf(`
			INSERT INTO %s (server_id, bucket, target_name, target_host, latency_sum, latency_max, latency_count, ok_count, fail_count)
			VALUES %s
			ON CONFLICT(server_id, target_name, bucket) DO UPDATE SET
				target_host = excluded.target_host,
				latency_sum = excluded.latency_sum,
				latency_max = MAX(%s.latency_max, excluded.latency_max),
				latency_count = excluded.latency_count,
				ok_count = excluded.ok_count,
				fail_count = excluded.fail_count`,
			table, strings.Join(valueStrings, ","), table)

		_, err := tx.Exec(query, valueArgs...)
		if err != nil {
			return err
		}
	}

	return nil
}

// NewDBWriter creates a new database writer with a buffered channel
func NewDBWriter(db *sql.DB, bufferSize int) *DBWriter {
	w := &DBWriter{
		db:      db,
		writeCh: make(chan writeJob, bufferSize),
		done:    make(chan struct{}),
	}
	w.wg.Add(1)
	go w.processWrites()
	return w
}

// processWrites handles all write operations sequentially
func (w *DBWriter) processWrites() {
	defer w.wg.Done()
	for {
		select {
		case job := <-w.writeCh:
			err := job.fn(w.db)
			if job.result != nil {
				job.result <- err
			} else if err != nil {
				fmt.Printf("Database write error: %v\n", err)
			}
		case <-w.done:
			// Drain remaining jobs before exiting
			for {
				select {
				case job := <-w.writeCh:
					err := job.fn(w.db)
					if job.result != nil {
						job.result <- err
					}
				default:
					return
				}
			}
		}
	}
}

// WriteAsync queues a write operation (fire-and-forget)
func (w *DBWriter) WriteAsync(fn func(*sql.DB) error) {
	select {
	case w.writeCh <- writeJob{fn: fn, result: nil}:
	default:
		fmt.Println("Warning: write queue full, dropping write")
	}
}

// WriteSync queues a write operation and waits for result
func (w *DBWriter) WriteSync(fn func(*sql.DB) error) error {
	result := make(chan error, 1)
	w.writeCh <- writeJob{fn: fn, result: result}
	return <-result
}

// Close stops the writer and waits for pending writes
func (w *DBWriter) Close() {
	close(w.done)
	w.wg.Wait()
}

// GetDB returns the underlying database for read operations
func (w *DBWriter) GetDB() *sql.DB {
	return w.db
}

func InitDatabase() (*sql.DB, error) {
	// Open database with busy_timeout as fallback
	db, err := sql.Open("sqlite", GetDBPath()+"?_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent read access
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		fmt.Printf("Warning: Failed to enable WAL mode: %v\n", err)
	}

	// Set synchronous to NORMAL for better performance while still being safe
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		fmt.Printf("Warning: Failed to set synchronous mode: %v\n", err)
	}

	// Create tables
	_, err = db.Exec(`
		-- Raw metrics (keep for 24 hours)
		-- Note: bucket_5min column added via migration for existing databases
		CREATE TABLE IF NOT EXISTS metrics_raw (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			cpu_usage REAL NOT NULL,
			memory_usage REAL NOT NULL,
			disk_usage REAL NOT NULL,
			net_rx INTEGER NOT NULL,
			net_tx INTEGER NOT NULL,
			load_1 REAL NOT NULL,
			load_5 REAL NOT NULL,
			load_15 REAL NOT NULL,
			ping_ms REAL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		);
		
		-- 15-minute aggregated metrics (keep for 7 days, for 7d range with 720 points)
		CREATE TABLE IF NOT EXISTS metrics_15min (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT NOT NULL,
			bucket_start TEXT NOT NULL,
			cpu_avg REAL NOT NULL,
			cpu_max REAL NOT NULL,
			memory_avg REAL NOT NULL,
			memory_max REAL NOT NULL,
			disk_avg REAL NOT NULL,
			net_rx_total INTEGER NOT NULL,
			net_tx_total INTEGER NOT NULL,
			ping_avg REAL,
			sample_count INTEGER NOT NULL,
			UNIQUE(server_id, bucket_start)
		);
		
		CREATE INDEX IF NOT EXISTS idx_metrics_15min_server_time ON metrics_15min(server_id, bucket_start);
		
		-- Hourly aggregated metrics (keep for 30 days)
		CREATE TABLE IF NOT EXISTS metrics_hourly (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT NOT NULL,
			hour_start TEXT NOT NULL,
			cpu_avg REAL NOT NULL,
			cpu_max REAL NOT NULL,
			memory_avg REAL NOT NULL,
			memory_max REAL NOT NULL,
			disk_avg REAL NOT NULL,
			net_rx_total INTEGER NOT NULL,
			net_tx_total INTEGER NOT NULL,
			ping_avg REAL,
			sample_count INTEGER NOT NULL,
			UNIQUE(server_id, hour_start)
		);
		
		-- Daily aggregated metrics (keep forever)
		CREATE TABLE IF NOT EXISTS metrics_daily (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT NOT NULL,
			date TEXT NOT NULL,
			cpu_avg REAL NOT NULL,
			cpu_max REAL NOT NULL,
			memory_avg REAL NOT NULL,
			memory_max REAL NOT NULL,
			disk_avg REAL NOT NULL,
			net_rx_total INTEGER NOT NULL,
			net_tx_total INTEGER NOT NULL,
			uptime_percent REAL NOT NULL,
			ping_avg REAL,
			sample_count INTEGER NOT NULL,
			UNIQUE(server_id, date)
		);
		
		-- Create indexes (bucket_5min index created after migration)
		CREATE INDEX IF NOT EXISTS idx_metrics_raw_server_time ON metrics_raw(server_id, timestamp);
		CREATE INDEX IF NOT EXISTS idx_metrics_hourly_server_time ON metrics_hourly(server_id, hour_start);
		CREATE INDEX IF NOT EXISTS idx_metrics_daily_server_time ON metrics_daily(server_id, date);
		
		-- Ping metrics per target (keep for 24 hours)
		-- Note: bucket_5min column added via migration for existing databases
		CREATE TABLE IF NOT EXISTS ping_raw (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_ms REAL,
			packet_loss REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'ok'
		);
		
		CREATE INDEX IF NOT EXISTS idx_ping_raw_server_time ON ping_raw(server_id, timestamp);
		CREATE INDEX IF NOT EXISTS idx_ping_raw_target ON ping_raw(server_id, target_name, timestamp);
		
		-- 15-minute aggregated ping metrics (keep for 7 days)
		CREATE TABLE IF NOT EXISTS ping_15min (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT NOT NULL,
			bucket_start TEXT NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_avg REAL,
			latency_max REAL,
			packet_loss_avg REAL NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL,
			UNIQUE(server_id, target_name, bucket_start)
		);
		
		CREATE INDEX IF NOT EXISTS idx_ping_15min_server_time ON ping_15min(server_id, bucket_start);
		CREATE INDEX IF NOT EXISTS idx_ping_15min_target ON ping_15min(server_id, target_name, bucket_start);
		
		-- Hourly aggregated ping metrics (keep for 30 days)
		CREATE TABLE IF NOT EXISTS ping_hourly (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT NOT NULL,
			hour_start TEXT NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_avg REAL,
			latency_max REAL,
			packet_loss_avg REAL NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL,
			UNIQUE(server_id, target_name, hour_start)
		);
		
		CREATE INDEX IF NOT EXISTS idx_ping_hourly_server_time ON ping_hourly(server_id, hour_start);
		CREATE INDEX IF NOT EXISTS idx_ping_hourly_target ON ping_hourly(server_id, target_name, hour_start);
		
		-- Daily aggregated ping metrics (keep forever)
		CREATE TABLE IF NOT EXISTS ping_daily (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT NOT NULL,
			date TEXT NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_avg REAL,
			latency_max REAL,
			packet_loss_avg REAL NOT NULL DEFAULT 0,
			uptime_percent REAL NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL,
			UNIQUE(server_id, target_name, date)
		);
		
		CREATE INDEX IF NOT EXISTS idx_ping_daily_server_time ON ping_daily(server_id, date);
		CREATE INDEX IF NOT EXISTS idx_ping_daily_target ON ping_daily(server_id, target_name, date);
	`)
	if err != nil {
		return nil, err
	}

	// Migration: Add ping_ms column if it doesn't exist
	db.Exec("ALTER TABLE metrics_raw ADD COLUMN ping_ms REAL")
	db.Exec("ALTER TABLE metrics_hourly ADD COLUMN ping_avg REAL")
	db.Exec("ALTER TABLE metrics_daily ADD COLUMN ping_avg REAL")

	// Migration: Add bucket_5min column for efficient 24h sampling (actually stores 2-min buckets for 720 points)
	db.Exec("ALTER TABLE metrics_raw ADD COLUMN bucket_5min INTEGER")
	db.Exec("ALTER TABLE ping_raw ADD COLUMN bucket_5min INTEGER")

	// Create indexes for bucket_5min (ignore error if already exists)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_metrics_raw_server_bucket ON metrics_raw(server_id, bucket_5min)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_ping_raw_server_bucket ON ping_raw(server_id, bucket_5min)")

	// Backfill bucket for existing data - only if there are NULL values (check first for fast startup)
	var needsBackfill5min int
	db.QueryRow("SELECT 1 FROM metrics_raw WHERE bucket_5min IS NULL OR bucket_5min > 100000000 LIMIT 1").Scan(&needsBackfill5min)
	if needsBackfill5min == 1 {
		fmt.Println("⏳ Backfilling bucket_5min for metrics_raw (one-time migration)...")
		db.Exec("UPDATE metrics_raw SET bucket_5min = CAST(strftime('%s', timestamp) AS INTEGER) / 120 WHERE bucket_5min IS NULL OR bucket_5min > 100000000")
	}
	db.QueryRow("SELECT 1 FROM ping_raw WHERE bucket_5min IS NULL OR bucket_5min > 100000000 LIMIT 1").Scan(&needsBackfill5min)
	if needsBackfill5min == 1 {
		fmt.Println("⏳ Backfilling bucket_5min for ping_raw (one-time migration)...")
		db.Exec("UPDATE ping_raw SET bucket_5min = CAST(strftime('%s', timestamp) AS INTEGER) / 120 WHERE bucket_5min IS NULL OR bucket_5min > 100000000")
	}

	// Migration: Add bucket_5sec column for efficient 1h sampling (5-sec buckets for 720 points over 1h)
	db.Exec("ALTER TABLE metrics_raw ADD COLUMN bucket_5sec INTEGER")
	db.Exec("ALTER TABLE ping_raw ADD COLUMN bucket_5sec INTEGER")

	// Create indexes for bucket_5sec (ignore error if already exists)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_metrics_raw_server_bucket_5sec ON metrics_raw(server_id, bucket_5sec)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_ping_raw_server_bucket_5sec ON ping_raw(server_id, bucket_5sec)")

	// Backfill bucket_5sec for existing data - only if there are NULL values (check first for fast startup)
	var needsBackfill5sec int
	db.QueryRow("SELECT 1 FROM metrics_raw WHERE bucket_5sec IS NULL LIMIT 1").Scan(&needsBackfill5sec)
	if needsBackfill5sec == 1 {
		fmt.Println("⏳ Backfilling bucket_5sec for metrics_raw (one-time migration)...")
		db.Exec("UPDATE metrics_raw SET bucket_5sec = CAST(strftime('%s', timestamp) AS INTEGER) / 5 WHERE bucket_5sec IS NULL")
	}
	db.QueryRow("SELECT 1 FROM ping_raw WHERE bucket_5sec IS NULL LIMIT 1").Scan(&needsBackfill5sec)
	if needsBackfill5sec == 1 {
		fmt.Println("⏳ Backfilling bucket_5sec for ping_raw (one-time migration)...")
		db.Exec("UPDATE ping_raw SET bucket_5sec = CAST(strftime('%s', timestamp) AS INTEGER) / 5 WHERE bucket_5sec IS NULL")
	}

	// Create real-time aggregation tables for fast queries
	db.Exec(`
		-- 5-second aggregated metrics (for 1h queries, ~720 points per server)
		CREATE TABLE IF NOT EXISTS metrics_5sec (
			server_id TEXT NOT NULL,
			bucket INTEGER NOT NULL,
			cpu_sum REAL NOT NULL DEFAULT 0,
			cpu_max REAL NOT NULL DEFAULT 0,
			memory_sum REAL NOT NULL DEFAULT 0,
			memory_max REAL NOT NULL DEFAULT 0,
			disk_sum REAL NOT NULL DEFAULT 0,
			net_rx INTEGER NOT NULL DEFAULT 0,
			net_tx INTEGER NOT NULL DEFAULT 0,
			ping_sum REAL NOT NULL DEFAULT 0,
			ping_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, bucket)
		) WITHOUT ROWID
	`)

	db.Exec(`
		-- 2-minute aggregated metrics (for 24h queries, ~720 points per server)
		CREATE TABLE IF NOT EXISTS metrics_2min (
			server_id TEXT NOT NULL,
			bucket INTEGER NOT NULL,
			cpu_sum REAL NOT NULL DEFAULT 0,
			cpu_max REAL NOT NULL DEFAULT 0,
			memory_sum REAL NOT NULL DEFAULT 0,
			memory_max REAL NOT NULL DEFAULT 0,
			disk_sum REAL NOT NULL DEFAULT 0,
			net_rx INTEGER NOT NULL DEFAULT 0,
			net_tx INTEGER NOT NULL DEFAULT 0,
			ping_sum REAL NOT NULL DEFAULT 0,
			ping_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, bucket)
		) WITHOUT ROWID
	`)

	// New aggregation tables for agent-side aggregation (15min, hourly, daily)
	db.Exec(`
		-- 15-minute aggregated metrics (for 7d queries, from agent)
		CREATE TABLE IF NOT EXISTS metrics_15min_agg (
			server_id TEXT NOT NULL,
			bucket INTEGER NOT NULL,
			cpu_sum REAL NOT NULL DEFAULT 0,
			cpu_max REAL NOT NULL DEFAULT 0,
			memory_sum REAL NOT NULL DEFAULT 0,
			memory_max REAL NOT NULL DEFAULT 0,
			disk_sum REAL NOT NULL DEFAULT 0,
			net_rx INTEGER NOT NULL DEFAULT 0,
			net_tx INTEGER NOT NULL DEFAULT 0,
			ping_sum REAL NOT NULL DEFAULT 0,
			ping_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, bucket)
		) WITHOUT ROWID
	`)

	db.Exec(`
		-- Hourly aggregated metrics (for 30d queries, from agent)
		CREATE TABLE IF NOT EXISTS metrics_hourly_agg (
			server_id TEXT NOT NULL,
			bucket INTEGER NOT NULL,
			cpu_sum REAL NOT NULL DEFAULT 0,
			cpu_max REAL NOT NULL DEFAULT 0,
			memory_sum REAL NOT NULL DEFAULT 0,
			memory_max REAL NOT NULL DEFAULT 0,
			disk_sum REAL NOT NULL DEFAULT 0,
			net_rx INTEGER NOT NULL DEFAULT 0,
			net_tx INTEGER NOT NULL DEFAULT 0,
			ping_sum REAL NOT NULL DEFAULT 0,
			ping_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, bucket)
		) WITHOUT ROWID
	`)

	db.Exec(`
		-- Daily aggregated metrics (for 1y queries, from agent)
		CREATE TABLE IF NOT EXISTS metrics_daily_agg (
			server_id TEXT NOT NULL,
			bucket INTEGER NOT NULL,
			cpu_sum REAL NOT NULL DEFAULT 0,
			cpu_max REAL NOT NULL DEFAULT 0,
			memory_sum REAL NOT NULL DEFAULT 0,
			memory_max REAL NOT NULL DEFAULT 0,
			disk_sum REAL NOT NULL DEFAULT 0,
			net_rx INTEGER NOT NULL DEFAULT 0,
			net_tx INTEGER NOT NULL DEFAULT 0,
			ping_sum REAL NOT NULL DEFAULT 0,
			ping_count INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, bucket)
		) WITHOUT ROWID
	`)

	db.Exec(`
		-- 5-second aggregated ping metrics (for 1h queries)
		CREATE TABLE IF NOT EXISTS ping_5sec (
			server_id TEXT NOT NULL,
			bucket INTEGER NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_max REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, target_name, bucket)
		) WITHOUT ROWID
	`)

	db.Exec(`
		-- 2-minute aggregated ping metrics (for 24h queries)
		CREATE TABLE IF NOT EXISTS ping_2min (
			server_id TEXT NOT NULL,
			bucket INTEGER NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_max REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, target_name, bucket)
		) WITHOUT ROWID
	`)

	// New ping aggregation tables for agent-side aggregation
	db.Exec(`
		-- 15-minute aggregated ping metrics (for 7d queries, from agent)
		CREATE TABLE IF NOT EXISTS ping_15min_agg (
			server_id TEXT NOT NULL,
			bucket INTEGER NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_max REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, target_name, bucket)
		) WITHOUT ROWID
	`)

	db.Exec(`
		-- Hourly aggregated ping metrics (for 30d queries, from agent)
		CREATE TABLE IF NOT EXISTS ping_hourly_agg (
			server_id TEXT NOT NULL,
			bucket INTEGER NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_max REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, target_name, bucket)
		) WITHOUT ROWID
	`)

	db.Exec(`
		-- Daily aggregated ping metrics (for 1y queries, from agent)
		CREATE TABLE IF NOT EXISTS ping_daily_agg (
			server_id TEXT NOT NULL,
			bucket INTEGER NOT NULL,
			target_name TEXT NOT NULL,
			target_host TEXT NOT NULL,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_max REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			ok_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, target_name, bucket)
		) WITHOUT ROWID
	`)

	// Run ANALYZE in background to avoid slow startup
	go func() {
		time.Sleep(10 * time.Second) // Wait for server to fully start
		db.Exec("ANALYZE")
	}()

	return db, nil
}

// StoreMetricsAsync queues metrics storage (fire-and-forget)
func StoreMetricsAsync(serverID string, metrics *SystemMetrics) {
	if dbWriter == nil {
		return
	}
	// Copy data to avoid race conditions
	m := *metrics
	sid := serverID
	dbWriter.WriteAsync(func(db *sql.DB) error {
		return storeMetricsInternal(db, sid, &m)
	})
}

// StoreMetricsWithDedup stores metrics with deduplication check
// Uses buffered writes for better performance with high agent count
func StoreMetricsWithDedup(serverID string, metrics *SystemMetrics) {
	// Use metrics buffer for batched writes
	if metricsBuffer != nil {
		metricsBuffer.Add(serverID, metrics)
		return
	}
	
	// Fallback to direct write
	if dbWriter == nil {
		return
	}
	m := *metrics
	sid := serverID
	dbWriter.WriteAsync(func(db *sql.DB) error {
		return storeMetricsWithDedupInternal(db, sid, &m)
	})
}

// StoreBatchMetrics stores a single metric from a batch, returns true if stored (not duplicate)
func StoreBatchMetrics(serverID string, metrics *SystemMetrics) bool {
	if dbWriter == nil {
		return false
	}
	m := *metrics
	sid := serverID
	
	result := make(chan bool, 1)
	dbWriter.WriteAsync(func(db *sql.DB) error {
		stored := storeMetricsWithDedupInternal(db, sid, &m) == nil
		select {
		case result <- stored:
		default:
		}
		return nil
	})
	
	// Non-blocking - assume success
	return true
}

// StoreAggregatedMetrics stores pre-aggregated metrics from agent
func StoreAggregatedMetrics(serverID string, agg *common.AggregatedMetrics) bool {
	if dbWriter == nil || agg == nil {
		return false
	}
	
	dbWriter.WriteAsync(func(db *sql.DB) error {
		return storeAggregatedMetricsInternal(db, serverID, agg)
	})
	
	return true
}

// StoreMultiGranularityMetrics stores pre-aggregated metrics at multiple granularities from agent
// Uses buffered writes for better performance
func StoreMultiGranularityMetrics(serverID string, granularities []common.GranularityData) bool {
	if len(granularities) == 0 {
		return false
	}
	
	// Use aggregation buffer for batched writes
	if aggBuffer != nil {
		aggBuffer.Add(serverID, granularities)
		return true
	}
	
	// Fallback to direct write if buffer not initialized
	if dbWriter == nil {
		return false
	}
	
	dbWriter.WriteAsync(func(db *sql.DB) error {
		return storeMultiGranularityMetricsInternal(db, serverID, granularities)
	})
	
	return true
}

// storeMultiGranularityMetricsInternal stores multi-granularity aggregated data
func storeMultiGranularityMetricsInternal(db *sql.DB, serverID string, granularities []common.GranularityData) error {
	for _, g := range granularities {
		// Determine which table to use based on granularity
		var metricsTable, pingTable string
		switch g.Granularity {
		case "5sec":
			metricsTable = "metrics_5sec"
			pingTable = "ping_5sec"
		case "2min":
			metricsTable = "metrics_2min"
			pingTable = "ping_2min"
		case "15min":
			metricsTable = "metrics_15min_agg"
			pingTable = "ping_15min_agg"
		case "hourly":
			metricsTable = "metrics_hourly_agg"
			pingTable = "ping_hourly_agg"
		case "daily":
			metricsTable = "metrics_daily_agg"
			pingTable = "ping_daily_agg"
		default:
			continue
		}

		// Store metrics buckets
		for _, m := range g.Metrics {
			db.Exec(`
				INSERT INTO `+metricsTable+` (server_id, bucket, cpu_sum, cpu_max, memory_sum, memory_max, disk_sum, net_rx, net_tx, ping_sum, ping_count, sample_count)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(server_id, bucket) DO UPDATE SET
					cpu_sum = excluded.cpu_sum,
					cpu_max = MAX(cpu_max, excluded.cpu_max),
					memory_sum = excluded.memory_sum,
					memory_max = MAX(memory_max, excluded.memory_max),
					disk_sum = excluded.disk_sum,
					net_rx = MAX(net_rx, excluded.net_rx),
					net_tx = MAX(net_tx, excluded.net_tx),
					ping_sum = excluded.ping_sum,
					ping_count = excluded.ping_count,
					sample_count = excluded.sample_count`,
				serverID, m.Bucket,
				m.CPUSum, m.CPUMax,
				m.MemorySum, m.MemoryMax,
				m.DiskSum,
				m.NetRx, m.NetTx,
				m.PingSum, m.PingCount,
				m.SampleCount,
			)
		}

		// Store ping buckets
		for _, p := range g.Ping {
			db.Exec(`
				INSERT INTO `+pingTable+` (server_id, bucket, target_name, target_host, latency_sum, latency_max, latency_count, ok_count, fail_count)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(server_id, target_name, bucket) DO UPDATE SET
					target_host = excluded.target_host,
					latency_sum = excluded.latency_sum,
					latency_max = MAX(latency_max, excluded.latency_max),
					latency_count = excluded.latency_count,
					ok_count = excluded.ok_count,
					fail_count = excluded.fail_count`,
				serverID, p.Bucket, p.TargetName, p.TargetHost,
				p.LatencySum, p.LatencyMax, p.LatencyCount, p.OkCount, p.FailCount,
			)
		}
	}

	return nil
}

// storeMetricsWithDedupInternal stores metrics with timestamp-based deduplication
func storeMetricsWithDedupInternal(db *sql.DB, serverID string, metrics *SystemMetrics) error {
	timestamp := metrics.Timestamp.Format(time.RFC3339)
	bucket5sec := metrics.Timestamp.Unix() / 5
	
	// Check if we already have data for this exact timestamp
	var exists int
	err := db.QueryRow(`
		SELECT 1 FROM metrics_raw 
		WHERE server_id = ? AND timestamp = ?
		LIMIT 1`, serverID, timestamp).Scan(&exists)
	
	if err == nil && exists == 1 {
		// Duplicate - skip
		return nil
	}
	
	// Also check if we have data in the same 5-second bucket to avoid near-duplicates
	err = db.QueryRow(`
		SELECT 1 FROM metrics_raw 
		WHERE server_id = ? AND bucket_5sec = ?
		LIMIT 1`, serverID, bucket5sec).Scan(&exists)
	
	if err == nil && exists == 1 {
		// Near-duplicate in same bucket - skip
		return nil
	}
	
	// No duplicate, store normally
	return storeMetricsInternal(db, serverID, metrics)
}

// storeAggregatedMetricsInternal stores pre-aggregated metrics
func storeAggregatedMetricsInternal(db *sql.DB, serverID string, agg *common.AggregatedMetrics) error {
	// Parse timestamps
	startTime, err := time.Parse(time.RFC3339Nano, agg.StartTime)
	if err != nil {
		startTime, err = time.Parse(time.RFC3339, agg.StartTime)
		if err != nil {
			return err
		}
	}
	
	endTime, err := time.Parse(time.RFC3339Nano, agg.EndTime)
	if err != nil {
		endTime, err = time.Parse(time.RFC3339, agg.EndTime)
		if err != nil {
			endTime = startTime.Add(time.Minute)
		}
	}
	
	// Calculate bucket based on start time
	bucket2min := startTime.Unix() / 120
	
	// Check for existing data in this bucket
	var exists int
	err = db.QueryRow(`
		SELECT 1 FROM metrics_2min 
		WHERE server_id = ? AND bucket = ?
		LIMIT 1`, serverID, bucket2min).Scan(&exists)
	
	if err == nil && exists == 1 {
		// Already have aggregated data for this bucket - merge or skip
		// For now, skip to avoid duplicates
		return nil
	}
	
	// Store in 2-minute aggregation table
	_, err = db.Exec(`
		INSERT INTO metrics_2min (server_id, bucket, cpu_sum, cpu_max, memory_sum, memory_max, disk_sum, net_rx, net_tx, ping_sum, ping_count, sample_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(server_id, bucket) DO UPDATE SET
			cpu_sum = cpu_sum + excluded.cpu_sum,
			cpu_max = MAX(cpu_max, excluded.cpu_max),
			memory_sum = memory_sum + excluded.memory_sum,
			memory_max = MAX(memory_max, excluded.memory_max),
			disk_sum = disk_sum + excluded.disk_sum,
			net_rx = MAX(net_rx, excluded.net_rx),
			net_tx = MAX(net_tx, excluded.net_tx),
			ping_sum = ping_sum + excluded.ping_sum,
			ping_count = ping_count + excluded.ping_count,
			sample_count = sample_count + excluded.sample_count`,
		serverID, bucket2min,
		float64(agg.CPUAvg)*float64(agg.SampleCount), float64(agg.CPUMax),
		float64(agg.MemoryAvg)*float64(agg.SampleCount), float64(agg.MemoryMax),
		float64(agg.DiskAvg)*float64(agg.SampleCount),
		agg.NetRxMax, agg.NetTxMax,
		0.0, 0, // ping values (if available)
		agg.SampleCount,
	)
	if err != nil {
		return err
	}
	
	// Also store last metrics snapshot as a raw entry for recent data queries
	if agg.LastMetrics != nil {
		agg.LastMetrics.Timestamp = endTime
		storeMetricsWithDedupInternal(db, serverID, agg.LastMetrics)
	}
	
	return nil
}

// StoreMetrics stores metrics synchronously (legacy, for compatibility)
func StoreMetrics(db *sql.DB, serverID string, metrics *SystemMetrics) error {
	if dbWriter != nil {
		m := *metrics
		sid := serverID
		return dbWriter.WriteSync(func(db *sql.DB) error {
			return storeMetricsInternal(db, sid, &m)
		})
	}
	return storeMetricsInternal(db, serverID, metrics)
}

func storeMetricsInternal(db *sql.DB, serverID string, metrics *SystemMetrics) error {
	var diskUsage float32 = 0
	if len(metrics.Disks) > 0 {
		diskUsage = metrics.Disks[0].UsagePercent
	}

	timestamp := metrics.Timestamp.Format(time.RFC3339)
	// Pre-compute 2-minute bucket for efficient 24h sampling (720 points over 24h)
	bucket5min := metrics.Timestamp.Unix() / 120
	// Pre-compute 5-second bucket for efficient 1h sampling (720 points over 1h)
	bucket5sec := metrics.Timestamp.Unix() / 5

	// Get average ping latency from all targets
	var pingMs *float64
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
			avg := sum / float64(count)
			pingMs = &avg
		}
	}

	// Insert raw data (for debugging and fallback)
	_, err := db.Exec(`
		INSERT INTO metrics_raw (server_id, timestamp, cpu_usage, memory_usage, disk_usage, net_rx, net_tx, load_1, load_5, load_15, ping_ms, bucket_5min, bucket_5sec)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		serverID,
		timestamp,
		metrics.CPU.Usage,
		metrics.Memory.UsagePercent,
		diskUsage,
		metrics.Network.TotalRx,
		metrics.Network.TotalTx,
		metrics.LoadAverage.One,
		metrics.LoadAverage.Five,
		metrics.LoadAverage.Fifteen,
		pingMs,
		bucket5min,
		bucket5sec,
	)
	if err != nil {
		return err
	}

	// UPSERT to 5-second aggregation table (for 1h queries)
	pingVal := float64(0)
	pingCnt := 0
	if pingMs != nil {
		pingVal = *pingMs
		pingCnt = 1
	}
	db.Exec(`
		INSERT INTO metrics_5sec (server_id, bucket, cpu_sum, cpu_max, memory_sum, memory_max, disk_sum, net_rx, net_tx, ping_sum, ping_count, sample_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(server_id, bucket) DO UPDATE SET
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
		serverID, bucket5sec,
		float64(metrics.CPU.Usage), float64(metrics.CPU.Usage),
		float64(metrics.Memory.UsagePercent), float64(metrics.Memory.UsagePercent),
		float64(diskUsage),
		metrics.Network.TotalRx, metrics.Network.TotalTx,
		pingVal, pingCnt,
	)

	// UPSERT to 2-minute aggregation table (for 24h queries)
	db.Exec(`
		INSERT INTO metrics_2min (server_id, bucket, cpu_sum, cpu_max, memory_sum, memory_max, disk_sum, net_rx, net_tx, ping_sum, ping_count, sample_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(server_id, bucket) DO UPDATE SET
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
		serverID, bucket5min,
		float64(metrics.CPU.Usage), float64(metrics.CPU.Usage),
		float64(metrics.Memory.UsagePercent), float64(metrics.Memory.UsagePercent),
		float64(diskUsage),
		metrics.Network.TotalRx, metrics.Network.TotalTx,
		pingVal, pingCnt,
	)

	// Store individual ping targets
	if metrics.Ping != nil {
		for _, target := range metrics.Ping.Targets {
			// Insert raw ping data
			db.Exec(`
				INSERT INTO ping_raw (server_id, timestamp, target_name, target_host, latency_ms, packet_loss, status, bucket_5min, bucket_5sec)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				serverID, timestamp, target.Name, target.Host,
				target.LatencyMs, target.PacketLoss, target.Status,
				bucket5min, bucket5sec,
			)

			// Prepare values for ping aggregation
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

			// UPSERT to ping_5sec (for 1h queries)
			db.Exec(`
				INSERT INTO ping_5sec (server_id, bucket, target_name, target_host, latency_sum, latency_max, latency_count, ok_count, fail_count)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(server_id, target_name, bucket) DO UPDATE SET
					target_host = excluded.target_host,
					latency_sum = latency_sum + excluded.latency_sum,
					latency_max = MAX(latency_max, excluded.latency_max),
					latency_count = latency_count + excluded.latency_count,
					ok_count = ok_count + excluded.ok_count,
					fail_count = fail_count + excluded.fail_count`,
				serverID, bucket5sec, target.Name, target.Host,
				latencyVal, latencyMax, latencyCnt, okCnt, failCnt,
			)

			// UPSERT to ping_2min (for 24h queries)
			db.Exec(`
				INSERT INTO ping_2min (server_id, bucket, target_name, target_host, latency_sum, latency_max, latency_count, ok_count, fail_count)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(server_id, target_name, bucket) DO UPDATE SET
					target_host = excluded.target_host,
					latency_sum = latency_sum + excluded.latency_sum,
					latency_max = MAX(latency_max, excluded.latency_max),
					latency_count = latency_count + excluded.latency_count,
					ok_count = ok_count + excluded.ok_count,
					fail_count = fail_count + excluded.fail_count`,
				serverID, bucket5min, target.Name, target.Host,
				latencyVal, latencyMax, latencyCnt, okCnt, failCnt,
			)
		}
	}

	return nil
}

func Aggregate15Min(db *sql.DB) error {
	if dbWriter != nil {
		return dbWriter.WriteSync(aggregate15MinInternal)
	}
	return aggregate15MinInternal(db)
}

func aggregate15MinInternal(db *sql.DB) error {
	// Aggregate raw data from the last hour into 15-minute buckets
	// This runs every 15 minutes, processing data from 15-30 minutes ago
	now := time.Now().UTC()
	// Round down to the previous 15-minute boundary
	minuteOffset := now.Minute() % 15
	bucketEnd := now.Add(-time.Duration(minuteOffset) * time.Minute).Truncate(time.Minute)
	bucketStart := bucketEnd.Add(-15 * time.Minute)

	_, err := db.Exec(`
		INSERT OR REPLACE INTO metrics_15min (server_id, bucket_start, cpu_avg, cpu_max, memory_avg, memory_max, disk_avg, net_rx_total, net_tx_total, ping_avg, sample_count)
		SELECT 
			server_id,
			? as bucket_start,
			AVG(cpu_usage),
			MAX(cpu_usage),
			AVG(memory_usage),
			MAX(memory_usage),
			AVG(disk_usage),
			MAX(net_rx) - MIN(net_rx),
			MAX(net_tx) - MIN(net_tx),
			AVG(ping_ms),
			COUNT(*)
		FROM metrics_raw
		WHERE timestamp >= ? AND timestamp < ?
		GROUP BY server_id`,
		bucketStart.Format(time.RFC3339),
		bucketStart.Format(time.RFC3339),
		bucketEnd.Format(time.RFC3339))
	if err != nil {
		return err
	}

	// Aggregate ping data into 15-minute buckets
	_, err = db.Exec(`
		INSERT OR REPLACE INTO ping_15min (server_id, bucket_start, target_name, target_host, latency_avg, latency_max, packet_loss_avg, ok_count, fail_count, sample_count)
		SELECT 
			server_id,
			? as bucket_start,
			target_name,
			target_host,
			AVG(latency_ms),
			MAX(latency_ms),
			AVG(packet_loss),
			SUM(CASE WHEN status = 'ok' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status != 'ok' THEN 1 ELSE 0 END),
			COUNT(*)
		FROM ping_raw
		WHERE timestamp >= ? AND timestamp < ?
		GROUP BY server_id, target_name, target_host`,
		bucketStart.Format(time.RFC3339),
		bucketStart.Format(time.RFC3339),
		bucketEnd.Format(time.RFC3339))
	return err
}

func AggregateHourly(db *sql.DB) error {
	if dbWriter != nil {
		return dbWriter.WriteSync(aggregateHourlyInternal)
	}
	return aggregateHourlyInternal(db)
}

func aggregateHourlyInternal(db *sql.DB) error {
	hourAgo := time.Now().UTC().Add(-time.Hour)
	hourStart := hourAgo.Format("2006-01-02T15:00:00Z")

	_, err := db.Exec(`
		INSERT OR REPLACE INTO metrics_hourly (server_id, hour_start, cpu_avg, cpu_max, memory_avg, memory_max, disk_avg, net_rx_total, net_tx_total, ping_avg, sample_count)
		SELECT 
			server_id,
			strftime('%Y-%m-%dT%H:00:00Z', bucket_start) as hour,
			AVG(cpu_avg),
			MAX(cpu_max),
			AVG(memory_avg),
			MAX(memory_max),
			AVG(disk_avg),
			SUM(net_rx_total),
			SUM(net_tx_total),
			AVG(ping_avg),
			SUM(sample_count)
		FROM metrics_15min
		WHERE bucket_start >= ? AND bucket_start < datetime(?, '+1 hour')
		GROUP BY server_id, hour`, hourStart, hourStart)
	if err != nil {
		return err
	}

	// Aggregate ping data into hourly buckets
	_, err = db.Exec(`
		INSERT OR REPLACE INTO ping_hourly (server_id, hour_start, target_name, target_host, latency_avg, latency_max, packet_loss_avg, ok_count, fail_count, sample_count)
		SELECT 
			server_id,
			strftime('%Y-%m-%dT%H:00:00Z', bucket_start) as hour,
			target_name,
			target_host,
			AVG(latency_avg),
			MAX(latency_max),
			AVG(packet_loss_avg),
			SUM(ok_count),
			SUM(fail_count),
			SUM(sample_count)
		FROM ping_15min
		WHERE bucket_start >= ? AND bucket_start < datetime(?, '+1 hour')
		GROUP BY server_id, target_name, target_host, hour`, hourStart, hourStart)
	return err
}

func AggregateDaily(db *sql.DB) error {
	if dbWriter != nil {
		return dbWriter.WriteSync(aggregateDailyInternal)
	}
	return aggregateDailyInternal(db)
}

func aggregateDailyInternal(db *sql.DB) error {
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

	_, err := db.Exec(`
		INSERT OR REPLACE INTO metrics_daily (server_id, date, cpu_avg, cpu_max, memory_avg, memory_max, disk_avg, net_rx_total, net_tx_total, uptime_percent, sample_count)
		SELECT 
			server_id,
			date(hour_start) as day,
			AVG(cpu_avg),
			MAX(cpu_max),
			AVG(memory_avg),
			MAX(memory_max),
			AVG(disk_avg),
			SUM(net_rx_total),
			SUM(net_tx_total),
			(COUNT(*) * 100.0 / 24.0),
			SUM(sample_count)
		FROM metrics_hourly
		WHERE date(hour_start) = ?
		GROUP BY server_id, day`, yesterday)
	if err != nil {
		return err
	}

	// Aggregate ping data into daily buckets
	_, err = db.Exec(`
		INSERT OR REPLACE INTO ping_daily (server_id, date, target_name, target_host, latency_avg, latency_max, packet_loss_avg, uptime_percent, sample_count)
		SELECT 
			server_id,
			date(hour_start) as day,
			target_name,
			target_host,
			AVG(latency_avg),
			MAX(latency_max),
			AVG(packet_loss_avg),
			(SUM(ok_count) * 100.0 / (SUM(ok_count) + SUM(fail_count))),
			SUM(sample_count)
		FROM ping_hourly
		WHERE date(hour_start) = ?
		GROUP BY server_id, target_name, target_host, day`, yesterday)
	return err
}

func CleanupOldData(db *sql.DB) error {
	if dbWriter != nil {
		return dbWriter.WriteSync(cleanupOldDataInternal)
	}
	return cleanupOldDataInternal(db)
}

func cleanupOldDataInternal(db *sql.DB) error {
	// Delete raw data older than 24 hours
	cutoffRaw := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	if _, err := db.Exec("DELETE FROM metrics_raw WHERE timestamp < ?", cutoffRaw); err != nil {
		return err
	}

	// Delete ping raw data older than 24 hours
	if _, err := db.Exec("DELETE FROM ping_raw WHERE timestamp < ?", cutoffRaw); err != nil {
		return err
	}

	// Delete 5-second aggregation data older than 2 hours
	cutoff5sec := time.Now().UTC().Add(-2*time.Hour).Unix() / 5
	db.Exec("DELETE FROM metrics_5sec WHERE bucket < ?", cutoff5sec)
	db.Exec("DELETE FROM ping_5sec WHERE bucket < ?", cutoff5sec)

	// Delete 2-minute aggregation data older than 26 hours
	cutoff2min := time.Now().UTC().Add(-26*time.Hour).Unix() / 120
	db.Exec("DELETE FROM metrics_2min WHERE bucket < ?", cutoff2min)
	db.Exec("DELETE FROM ping_2min WHERE bucket < ?", cutoff2min)

	// Delete 15-min aggregation data (agent-provided) older than 8 days
	cutoff15minAgg := time.Now().UTC().Add(-8*24*time.Hour).Unix() / 900
	db.Exec("DELETE FROM metrics_15min_agg WHERE bucket < ?", cutoff15minAgg)
	db.Exec("DELETE FROM ping_15min_agg WHERE bucket < ?", cutoff15minAgg)

	// Delete hourly aggregation data (agent-provided) older than 32 days
	cutoffHourlyAgg := time.Now().UTC().Add(-32*24*time.Hour).Unix() / 3600
	db.Exec("DELETE FROM metrics_hourly_agg WHERE bucket < ?", cutoffHourlyAgg)
	db.Exec("DELETE FROM ping_hourly_agg WHERE bucket < ?", cutoffHourlyAgg)

	// Delete daily aggregation data (agent-provided) older than 400 days
	cutoffDailyAgg := time.Now().UTC().Add(-400*24*time.Hour).Unix() / 86400
	db.Exec("DELETE FROM metrics_daily_agg WHERE bucket < ?", cutoffDailyAgg)
	db.Exec("DELETE FROM ping_daily_agg WHERE bucket < ?", cutoffDailyAgg)

	// Delete old pre-aggregated 15-min data older than 7 days (legacy)
	cutoff15min := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	db.Exec("DELETE FROM metrics_15min WHERE bucket_start < ?", cutoff15min)
	db.Exec("DELETE FROM ping_15min WHERE bucket_start < ?", cutoff15min)

	// Delete old pre-aggregated hourly data older than 30 days (legacy)
	cutoffHourly := time.Now().UTC().AddDate(0, 0, -30).Format(time.RFC3339)
	db.Exec("DELETE FROM metrics_hourly WHERE hour_start < ?", cutoffHourly)
	db.Exec("DELETE FROM ping_hourly WHERE hour_start < ?", cutoffHourly)

	// Update query planner statistics after cleanup
	db.Exec("ANALYZE")

	return nil
}

func GetHistory(db *sql.DB, serverID, rangeStr string) ([]HistoryPoint, error) {
	return GetHistorySince(db, serverID, rangeStr, 0)
}

// GetHistorySince returns history data since a specific bucket (for incremental queries)
func GetHistorySince(db *sql.DB, serverID, rangeStr string, sinceBucket int64) ([]HistoryPoint, error) {
	var data []HistoryPoint
	var rows *sql.Rows
	var err error

	switch rangeStr {
	case "1h":
		// Read directly from pre-aggregated 5-second table (no GROUP BY needed!)
		cutoffBucket := time.Now().UTC().Add(-time.Hour).Unix() / 5
		if sinceBucket > cutoffBucket {
			cutoffBucket = sinceBucket
		}
		rows, err = db.Query(`
			SELECT 
				strftime('%Y-%m-%dT%H:%M:%SZ', bucket * 5, 'unixepoch') as timestamp,
				CASE WHEN sample_count > 0 THEN cpu_sum / sample_count ELSE 0 END as cpu_usage,
				CASE WHEN sample_count > 0 THEN memory_sum / sample_count ELSE 0 END as memory_usage,
				CASE WHEN sample_count > 0 THEN disk_sum / sample_count ELSE 0 END as disk_usage,
				net_rx,
				net_tx,
				CASE WHEN ping_count > 0 THEN ping_sum / ping_count ELSE NULL END as ping_ms,
				bucket
			FROM metrics_5sec 
			WHERE server_id = ? AND bucket >= ?
			ORDER BY bucket ASC
			LIMIT 720`, serverID, cutoffBucket)

	case "24h":
		// Read directly from pre-aggregated 2-minute table (no GROUP BY needed!)
		cutoffBucket := time.Now().UTC().Add(-24*time.Hour).Unix() / 120
		if sinceBucket > cutoffBucket {
			cutoffBucket = sinceBucket
		}
		rows, err = db.Query(`
			SELECT 
				strftime('%Y-%m-%dT%H:%M:%SZ', bucket * 120, 'unixepoch') as timestamp,
				CASE WHEN sample_count > 0 THEN cpu_sum / sample_count ELSE 0 END as cpu_usage,
				CASE WHEN sample_count > 0 THEN memory_sum / sample_count ELSE 0 END as memory_usage,
				CASE WHEN sample_count > 0 THEN disk_sum / sample_count ELSE 0 END as disk_usage,
				net_rx,
				net_tx,
				CASE WHEN ping_count > 0 THEN ping_sum / ping_count ELSE NULL END as ping_ms,
				bucket
			FROM metrics_2min 
			WHERE server_id = ? AND bucket >= ?
			ORDER BY bucket ASC
			LIMIT 720`, serverID, cutoffBucket)

	case "7d":
		// 7d with 15-min buckets (672 points max) - try agent-aggregated data first
		cutoffBucket := time.Now().UTC().Add(-7*24*time.Hour).Unix() / 900
		var count int
		db.QueryRow(`SELECT COUNT(*) FROM metrics_15min_agg WHERE server_id = ? AND bucket >= ?`,
			serverID, cutoffBucket).Scan(&count)

		if count > 0 {
			// Use agent-aggregated 15-min data
			rows, err = db.Query(`
				SELECT 
					strftime('%Y-%m-%dT%H:%M:%SZ', bucket * 900, 'unixepoch') as timestamp,
					CASE WHEN sample_count > 0 THEN cpu_sum / sample_count ELSE 0 END as cpu_usage,
					CASE WHEN sample_count > 0 THEN memory_sum / sample_count ELSE 0 END as memory_usage,
					CASE WHEN sample_count > 0 THEN disk_sum / sample_count ELSE 0 END as disk_usage,
					net_rx,
					net_tx,
					CASE WHEN ping_count > 0 THEN ping_sum / ping_count ELSE NULL END as ping_ms
				FROM metrics_15min_agg 
				WHERE server_id = ? AND bucket >= ?
				ORDER BY bucket ASC
				LIMIT 720`, serverID, cutoffBucket)
		} else {
			// Fall back to old pre-aggregated 15-min data (for backward compatibility)
			cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
			db.QueryRow(`SELECT COUNT(*) FROM metrics_15min WHERE server_id = ? AND bucket_start >= ?`,
				serverID, cutoff).Scan(&count)
			
			if count > 0 {
				rows, err = db.Query(`
					SELECT bucket_start, cpu_avg, memory_avg, disk_avg, net_rx_total, net_tx_total, ping_avg
					FROM metrics_15min 
					WHERE server_id = ? AND bucket_start >= ?
					ORDER BY bucket_start ASC
					LIMIT 720`, serverID, cutoff)
			} else {
				// Fall back to real-time aggregation from raw data (15-min buckets = 900 seconds)
				rows, err = db.Query(`
					SELECT 
						strftime('%Y-%m-%dT%H:%M:%SZ', (strftime('%s', timestamp) / 900) * 900, 'unixepoch') as bucket_start,
						AVG(cpu_usage) as cpu_avg,
						AVG(memory_usage) as memory_avg,
						AVG(disk_usage) as disk_avg,
						MAX(net_rx) - MIN(net_rx) as net_rx_total,
						MAX(net_tx) - MIN(net_tx) as net_tx_total,
						AVG(ping_ms) as ping_avg
					FROM metrics_raw 
					WHERE server_id = ? AND timestamp >= ?
					GROUP BY strftime('%s', timestamp) / 900
					ORDER BY bucket_start ASC
					LIMIT 720`, serverID, cutoff)
			}
		}

	case "30d":
		// 30d with hourly buckets (720 points max) - try agent-aggregated data first
		cutoffBucket := time.Now().UTC().AddDate(0, 0, -30).Unix() / 3600
		var count int
		db.QueryRow(`SELECT COUNT(*) FROM metrics_hourly_agg WHERE server_id = ? AND bucket >= ?`,
			serverID, cutoffBucket).Scan(&count)

		if count > 0 {
			// Use agent-aggregated hourly data
			rows, err = db.Query(`
				SELECT 
					strftime('%Y-%m-%dT%H:00:00Z', bucket * 3600, 'unixepoch') as timestamp,
					CASE WHEN sample_count > 0 THEN cpu_sum / sample_count ELSE 0 END as cpu_usage,
					CASE WHEN sample_count > 0 THEN memory_sum / sample_count ELSE 0 END as memory_usage,
					CASE WHEN sample_count > 0 THEN disk_sum / sample_count ELSE 0 END as disk_usage,
					net_rx,
					net_tx,
					CASE WHEN ping_count > 0 THEN ping_sum / ping_count ELSE NULL END as ping_ms
				FROM metrics_hourly_agg 
				WHERE server_id = ? AND bucket >= ?
				ORDER BY bucket ASC
				LIMIT 720`, serverID, cutoffBucket)
		} else {
			// Fall back to old pre-aggregated hourly data (for backward compatibility)
			cutoff := time.Now().UTC().AddDate(0, 0, -30).Format(time.RFC3339)
			db.QueryRow(`SELECT COUNT(*) FROM metrics_hourly WHERE server_id = ? AND hour_start >= ?`,
				serverID, cutoff).Scan(&count)

			if count > 0 {
				rows, err = db.Query(`
					SELECT hour_start, cpu_avg, memory_avg, disk_avg, net_rx_total, net_tx_total, ping_avg
					FROM metrics_hourly WHERE server_id = ? AND hour_start >= ?
					ORDER BY hour_start ASC
					LIMIT 720`, serverID, cutoff)
			} else {
				// Try 15-min table
				var count15 int
				db.QueryRow(`SELECT COUNT(*) FROM metrics_15min WHERE server_id = ? AND bucket_start >= ?`,
					serverID, cutoff).Scan(&count15)

				if count15 > 0 {
					rows, err = db.Query(`
						SELECT 
							strftime('%Y-%m-%dT%H:00:00Z', bucket_start) as hour_start,
							AVG(cpu_avg) as cpu_avg,
							AVG(memory_avg) as memory_avg,
							AVG(disk_avg) as disk_avg,
							SUM(net_rx_total) as net_rx_total,
							SUM(net_tx_total) as net_tx_total,
							AVG(ping_avg) as ping_avg
						FROM metrics_15min 
						WHERE server_id = ? AND bucket_start >= ?
						GROUP BY strftime('%Y-%m-%dT%H:00:00Z', bucket_start)
						ORDER BY hour_start ASC
						LIMIT 720`, serverID, cutoff)
				} else {
					// Fall back to raw data with hourly aggregation
					rows, err = db.Query(`
						SELECT 
							strftime('%Y-%m-%dT%H:00:00Z', timestamp) as hour_start,
							AVG(cpu_usage) as cpu_avg,
							AVG(memory_usage) as memory_avg,
							AVG(disk_usage) as disk_avg,
							MAX(net_rx) - MIN(net_rx) as net_rx_total,
							MAX(net_tx) - MIN(net_tx) as net_tx_total,
							AVG(ping_ms) as ping_avg
						FROM metrics_raw 
						WHERE server_id = ? AND timestamp >= ?
						GROUP BY strftime('%Y-%m-%dT%H:00:00Z', timestamp)
						ORDER BY hour_start ASC
						LIMIT 720`, serverID, cutoff)
				}
			}
		}

	case "1y":
		// 1y with daily buckets (365 points max) - try agent-aggregated data first
		cutoffBucket := time.Now().UTC().AddDate(0, 0, -365).Unix() / 86400
		var count int
		db.QueryRow(`SELECT COUNT(*) FROM metrics_daily_agg WHERE server_id = ? AND bucket >= ?`,
			serverID, cutoffBucket).Scan(&count)

		if count > 0 {
			// Use agent-aggregated daily data
			rows, err = db.Query(`
				SELECT 
					strftime('%Y-%m-%dT00:00:00Z', bucket * 86400, 'unixepoch') as timestamp,
					CASE WHEN sample_count > 0 THEN cpu_sum / sample_count ELSE 0 END as cpu_usage,
					CASE WHEN sample_count > 0 THEN memory_sum / sample_count ELSE 0 END as memory_usage,
					CASE WHEN sample_count > 0 THEN disk_sum / sample_count ELSE 0 END as disk_usage,
					net_rx,
					net_tx,
					CASE WHEN ping_count > 0 THEN ping_sum / ping_count ELSE NULL END as ping_ms
				FROM metrics_daily_agg 
				WHERE server_id = ? AND bucket >= ?
				ORDER BY bucket ASC
				LIMIT 365`, serverID, cutoffBucket)
		} else {
			// Fall back to old pre-aggregated hourly data (for backward compatibility)
			cutoff := time.Now().UTC().AddDate(0, 0, -365).Format(time.RFC3339)
			db.QueryRow(`SELECT COUNT(*) FROM metrics_hourly WHERE server_id = ? AND hour_start >= ?`,
				serverID, cutoff).Scan(&count)

			if count > 0 {
				// Use hourly data with 12-hour grouping
				rows, err = db.Query(`
					SELECT 
						MIN(hour_start) as timestamp,
						AVG(cpu_avg) as cpu_avg,
						AVG(memory_avg) as memory_avg,
						AVG(disk_avg) as disk_avg,
						SUM(net_rx_total) as net_rx_total,
						SUM(net_tx_total) as net_tx_total,
						AVG(ping_avg) as ping_avg
					FROM metrics_hourly 
					WHERE server_id = ? AND hour_start >= ?
					GROUP BY date(hour_start), (CAST(strftime('%H', hour_start) AS INTEGER) / 12)
					ORDER BY MIN(hour_start) ASC
					LIMIT 730`, serverID, cutoff)
			} else {
				// Fall back to raw data with 12-hour aggregation
				rows, err = db.Query(`
					SELECT 
						MIN(timestamp) as timestamp,
						AVG(cpu_usage) as cpu_avg,
						AVG(memory_usage) as memory_avg,
						AVG(disk_usage) as disk_avg,
						MAX(net_rx) - MIN(net_rx) as net_rx_total,
						MAX(net_tx) - MIN(net_tx) as net_tx_total,
						AVG(ping_ms) as ping_avg
					FROM metrics_raw 
					WHERE server_id = ? AND timestamp >= ?
					GROUP BY date(timestamp), (CAST(strftime('%H', timestamp) AS INTEGER) / 12)
					ORDER BY MIN(timestamp) ASC
					LIMIT 730`, serverID, cutoff)
			}
		}

	default:
		// Default to 24h - read from pre-aggregated table
		cutoffBucket := time.Now().UTC().Add(-24*time.Hour).Unix() / 120
		if sinceBucket > cutoffBucket {
			cutoffBucket = sinceBucket
		}
		rows, err = db.Query(`
			SELECT 
				strftime('%Y-%m-%dT%H:%M:%SZ', bucket * 120, 'unixepoch') as timestamp,
				CASE WHEN sample_count > 0 THEN cpu_sum / sample_count ELSE 0 END as cpu_usage,
				CASE WHEN sample_count > 0 THEN memory_sum / sample_count ELSE 0 END as memory_usage,
				CASE WHEN sample_count > 0 THEN disk_sum / sample_count ELSE 0 END as disk_usage,
				net_rx,
				net_tx,
				CASE WHEN ping_count > 0 THEN ping_sum / ping_count ELSE NULL END as ping_ms,
				bucket
			FROM metrics_2min 
			WHERE server_id = ? AND bucket >= ?
			ORDER BY bucket ASC
			LIMIT 720`, serverID, cutoffBucket)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Check if we're reading from aggregated tables (1h or 24h) which have bucket column
	useAggregated := rangeStr == "1h" || rangeStr == "24h" || rangeStr == ""

	for rows.Next() {
		var point HistoryPoint
		var bucket int64
		var scanErr error
		if useAggregated {
			scanErr = rows.Scan(&point.Timestamp, &point.CPU, &point.Memory, &point.Disk, &point.NetRx, &point.NetTx, &point.PingMs, &bucket)
		} else {
			scanErr = rows.Scan(&point.Timestamp, &point.CPU, &point.Memory, &point.Disk, &point.NetRx, &point.NetTx, &point.PingMs)
		}
		if scanErr != nil {
			continue
		}
		data = append(data, point)
	}

	return data, nil
}

func GetPingHistory(db *sql.DB, serverID, rangeStr string) ([]PingHistoryTarget, error) {
	return GetPingHistorySince(db, serverID, rangeStr, 0)
}

// GetPingHistorySince returns ping history data since a specific bucket (for incremental queries)
func GetPingHistorySince(db *sql.DB, serverID, rangeStr string, sinceBucket int64) ([]PingHistoryTarget, error) {
	var rows *sql.Rows
	var err error

	switch rangeStr {
	case "1h":
		// Read directly from pre-aggregated 5-second table (no GROUP BY needed!)
		cutoffBucket := time.Now().UTC().Add(-time.Hour).Unix() / 5
		if sinceBucket > cutoffBucket {
			cutoffBucket = sinceBucket
		}
		rows, err = db.Query(`
			SELECT 
				target_name,
				target_host,
				strftime('%Y-%m-%dT%H:%M:%SZ', bucket * 5, 'unixepoch') as timestamp,
				CASE WHEN latency_count > 0 THEN latency_sum / latency_count ELSE NULL END as latency_ms,
				CASE WHEN fail_count > 0 THEN 'error' ELSE 'ok' END as status
			FROM ping_5sec 
			WHERE server_id = ? AND bucket >= ?
			ORDER BY target_name, bucket ASC`, serverID, cutoffBucket)

	case "24h":
		// Read directly from pre-aggregated 2-minute table (no GROUP BY needed!)
		cutoffBucket := time.Now().UTC().Add(-24*time.Hour).Unix() / 120
		if sinceBucket > cutoffBucket {
			cutoffBucket = sinceBucket
		}
		rows, err = db.Query(`
			SELECT 
				target_name,
				target_host,
				strftime('%Y-%m-%dT%H:%M:%SZ', bucket * 120, 'unixepoch') as timestamp,
				CASE WHEN latency_count > 0 THEN latency_sum / latency_count ELSE NULL END as latency_ms,
				CASE WHEN fail_count > 0 THEN 'error' ELSE 'ok' END as status
			FROM ping_2min 
			WHERE server_id = ? AND bucket >= ?
			ORDER BY target_name, bucket ASC`, serverID, cutoffBucket)

	case "7d":
		// 7d with 15-min buckets (672 points max) - try agent-aggregated data first
		cutoffBucket := time.Now().UTC().Add(-7*24*time.Hour).Unix() / 900
		var count int
		db.QueryRow(`SELECT COUNT(*) FROM ping_15min_agg WHERE server_id = ? AND bucket >= ?`,
			serverID, cutoffBucket).Scan(&count)

		if count > 0 {
			// Use agent-aggregated 15-min ping data
			rows, err = db.Query(`
				SELECT 
					target_name,
					target_host,
					strftime('%Y-%m-%dT%H:%M:%SZ', bucket * 900, 'unixepoch') as timestamp,
					CASE WHEN latency_count > 0 THEN latency_sum / latency_count ELSE NULL END as latency_ms,
					CASE WHEN fail_count > 0 THEN 'error' ELSE 'ok' END as status
				FROM ping_15min_agg 
				WHERE server_id = ? AND bucket >= ?
				ORDER BY target_name, bucket ASC`, serverID, cutoffBucket)
		} else {
			// Fall back to old pre-aggregated 15-min data
			cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
			db.QueryRow(`SELECT COUNT(*) FROM ping_15min WHERE server_id = ? AND bucket_start >= ?`,
				serverID, cutoff).Scan(&count)

			if count > 0 {
				rows, err = db.Query(`
					SELECT 
						target_name,
						target_host,
						bucket_start,
						latency_avg as latency_ms,
						CASE WHEN fail_count > 0 THEN 'error' ELSE 'ok' END as status
					FROM ping_15min 
					WHERE server_id = ? AND bucket_start >= ?
					ORDER BY target_name, bucket_start ASC`, serverID, cutoff)
			} else {
				// Fall back to real-time aggregation from raw data
				rows, err = db.Query(`
					SELECT 
						target_name,
						target_host,
						strftime('%Y-%m-%dT%H:%M:%SZ', (strftime('%s', timestamp) / 900) * 900, 'unixepoch') as bucket_start,
						AVG(latency_ms) as latency_ms,
						MIN(status) as status
					FROM ping_raw 
					WHERE server_id = ? AND timestamp >= ?
					GROUP BY target_name, target_host, strftime('%s', timestamp) / 900
					ORDER BY target_name, bucket_start ASC`, serverID, cutoff)
			}
		}

	case "30d":
		// 30d with hourly buckets (720 points max) - try agent-aggregated data first
		cutoffBucket := time.Now().UTC().AddDate(0, 0, -30).Unix() / 3600
		var count int
		db.QueryRow(`SELECT COUNT(*) FROM ping_hourly_agg WHERE server_id = ? AND bucket >= ?`,
			serverID, cutoffBucket).Scan(&count)

		if count > 0 {
			// Use agent-aggregated hourly ping data
			rows, err = db.Query(`
				SELECT 
					target_name,
					target_host,
					strftime('%Y-%m-%dT%H:00:00Z', bucket * 3600, 'unixepoch') as timestamp,
					CASE WHEN latency_count > 0 THEN latency_sum / latency_count ELSE NULL END as latency_ms,
					CASE WHEN fail_count > 0 THEN 'error' ELSE 'ok' END as status
				FROM ping_hourly_agg 
				WHERE server_id = ? AND bucket >= ?
				ORDER BY target_name, bucket ASC`, serverID, cutoffBucket)
		} else {
			// Fall back to old pre-aggregated hourly data
			cutoff := time.Now().UTC().AddDate(0, 0, -30).Format(time.RFC3339)
			db.QueryRow(`SELECT COUNT(*) FROM ping_hourly WHERE server_id = ? AND hour_start >= ?`,
				serverID, cutoff).Scan(&count)

			if count > 0 {
				rows, err = db.Query(`
					SELECT 
						target_name,
						target_host,
						hour_start,
						latency_avg as latency_ms,
						CASE WHEN fail_count > 0 THEN 'error' ELSE 'ok' END as status
					FROM ping_hourly 
					WHERE server_id = ? AND hour_start >= ?
					ORDER BY target_name, hour_start ASC`, serverID, cutoff)
			} else {
				// Try 15-min table first
				var count15 int
				db.QueryRow(`SELECT COUNT(*) FROM ping_15min WHERE server_id = ? AND bucket_start >= ?`,
					serverID, cutoff).Scan(&count15)

				if count15 > 0 {
					rows, err = db.Query(`
						SELECT 
							target_name,
							target_host,
							strftime('%Y-%m-%dT%H:00:00Z', bucket_start) as hour_start,
							AVG(latency_avg) as latency_ms,
							CASE WHEN SUM(fail_count) > 0 THEN 'error' ELSE 'ok' END as status
						FROM ping_15min 
						WHERE server_id = ? AND bucket_start >= ?
						GROUP BY target_name, target_host, strftime('%Y-%m-%dT%H:00:00Z', bucket_start)
						ORDER BY target_name, hour_start ASC`, serverID, cutoff)
				} else {
					// Fall back to raw data with hourly aggregation
					rows, err = db.Query(`
						SELECT 
							target_name,
							target_host,
							strftime('%Y-%m-%dT%H:00:00Z', timestamp) as hour_start,
							AVG(latency_ms) as latency_ms,
							MIN(status) as status
						FROM ping_raw 
						WHERE server_id = ? AND timestamp >= ?
						GROUP BY target_name, target_host, strftime('%Y-%m-%dT%H:00:00Z', timestamp)
						ORDER BY target_name, hour_start ASC`, serverID, cutoff)
				}
			}
		}

	case "1y":
		// 1y with daily buckets (365 points max) - try agent-aggregated data first
		cutoffBucket := time.Now().UTC().AddDate(0, 0, -365).Unix() / 86400
		var count int
		db.QueryRow(`SELECT COUNT(*) FROM ping_daily_agg WHERE server_id = ? AND bucket >= ?`,
			serverID, cutoffBucket).Scan(&count)

		if count > 0 {
			// Use agent-aggregated daily ping data
			rows, err = db.Query(`
				SELECT 
					target_name,
					target_host,
					strftime('%Y-%m-%dT00:00:00Z', bucket * 86400, 'unixepoch') as timestamp,
					CASE WHEN latency_count > 0 THEN latency_sum / latency_count ELSE NULL END as latency_ms,
					CASE WHEN fail_count > 0 THEN 'error' ELSE 'ok' END as status
				FROM ping_daily_agg 
				WHERE server_id = ? AND bucket >= ?
				ORDER BY target_name, bucket ASC`, serverID, cutoffBucket)
		} else {
			// Fall back to old pre-aggregated hourly data
			cutoff := time.Now().UTC().AddDate(0, 0, -365).Format(time.RFC3339)
			db.QueryRow(`SELECT COUNT(*) FROM ping_hourly WHERE server_id = ? AND hour_start >= ?`,
				serverID, cutoff).Scan(&count)

			if count > 0 {
				// Use hourly data with 12-hour grouping
				rows, err = db.Query(`
					SELECT 
						target_name,
						target_host,
						MIN(hour_start) as timestamp,
						AVG(latency_avg) as latency_ms,
						CASE WHEN SUM(fail_count) > 0 THEN 'error' ELSE 'ok' END as status
					FROM ping_hourly 
					WHERE server_id = ? AND hour_start >= ?
					GROUP BY target_name, target_host, date(hour_start), (CAST(strftime('%H', hour_start) AS INTEGER) / 12)
					ORDER BY target_name, MIN(hour_start) ASC`, serverID, cutoff)
			} else {
				// Fall back to raw data with 12-hour aggregation
				rows, err = db.Query(`
					SELECT 
						target_name,
						target_host,
						MIN(timestamp) as timestamp,
						AVG(latency_ms) as latency_ms,
						MIN(status) as status
				FROM ping_raw 
				WHERE server_id = ? AND timestamp >= ?
				GROUP BY target_name, target_host, date(timestamp), (CAST(strftime('%H', timestamp) AS INTEGER) / 12)
				ORDER BY target_name, MIN(timestamp) ASC`, serverID, cutoff)
			}
		}

	default:
		// Default to 24h - read from pre-aggregated table
		cutoffBucket := time.Now().UTC().Add(-24*time.Hour).Unix() / 120
		if sinceBucket > cutoffBucket {
			cutoffBucket = sinceBucket
		}
		rows, err = db.Query(`
			SELECT 
				target_name,
				target_host,
				strftime('%Y-%m-%dT%H:%M:%SZ', bucket * 120, 'unixepoch') as timestamp,
				CASE WHEN latency_count > 0 THEN latency_sum / latency_count ELSE NULL END as latency_ms,
				CASE WHEN fail_count > 0 THEN 'error' ELSE 'ok' END as status
			FROM ping_2min 
			WHERE server_id = ? AND bucket >= ?
			ORDER BY target_name, bucket ASC`, serverID, cutoffBucket)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	targetsMap := make(map[string]*PingHistoryTarget)
	for rows.Next() {
		var name, host, timestamp, status string
		var latencyMs *float64

		if err := rows.Scan(&name, &host, &timestamp, &latencyMs, &status); err != nil {
			continue
		}

		if _, exists := targetsMap[name]; !exists {
			targetsMap[name] = &PingHistoryTarget{
				Name: name,
				Host: host,
				Data: []PingHistoryPoint{},
			}
		}

		targetsMap[name].Data = append(targetsMap[name].Data, PingHistoryPoint{
			Timestamp: timestamp,
			LatencyMs: latencyMs,
			Status:    status,
		})
	}

	var targets []PingHistoryTarget
	for _, t := range targetsMap {
		targets = append(targets, *t)
	}

	return targets, nil
}

