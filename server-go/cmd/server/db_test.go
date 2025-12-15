package main

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestHelper provides test utilities
type TestHelper struct {
	db     *sql.DB
	dbPath string
}

// NewTestHelper creates a test helper with an in-memory database
func NewTestHelper(t *testing.T) *TestHelper {
	t.Helper()

	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "vstats_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()

	db, err := sql.Open("sqlite", dbPath+"?_busy_timeout=5000")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Enable WAL mode
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")

	return &TestHelper{
		db:     db,
		dbPath: dbPath,
	}
}

// Close cleans up the test helper
func (h *TestHelper) Close() {
	if h.db != nil {
		h.db.Close()
	}
	os.Remove(h.dbPath)
	os.Remove(h.dbPath + "-wal")
	os.Remove(h.dbPath + "-shm")
}

// InitTestTables creates minimal tables for testing
func (h *TestHelper) InitTestTables(t *testing.T) {
	t.Helper()

	_, err := h.db.Exec(`
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
			bucket_5min INTEGER,
			bucket_5sec INTEGER,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		);
		
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
		) WITHOUT ROWID;
		
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
		) WITHOUT ROWID;

		CREATE INDEX IF NOT EXISTS idx_metrics_raw_server_time ON metrics_raw(server_id, timestamp);
		CREATE INDEX IF NOT EXISTS idx_metrics_raw_server_bucket ON metrics_raw(server_id, bucket_5min);
		CREATE INDEX IF NOT EXISTS idx_metrics_raw_server_bucket_5sec ON metrics_raw(server_id, bucket_5sec);
	`)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}
}

// TestMetricsBuffer tests the MetricsBuffer functionality
func TestMetricsBuffer(t *testing.T) {
	t.Run("NewMetricsBuffer", func(t *testing.T) {
		mb := NewMetricsBuffer(time.Second, 100)
		if mb == nil {
			t.Fatal("NewMetricsBuffer returned nil")
		}
		defer mb.Close()

		if mb.maxSize != 100 {
			t.Errorf("Expected maxSize 100, got %d", mb.maxSize)
		}
	})

	t.Run("Add items to buffer", func(t *testing.T) {
		mb := NewMetricsBuffer(time.Hour, 100) // Long interval to prevent auto-flush
		defer mb.Close()

		metrics := &SystemMetrics{
			Timestamp: time.Now(),
			CPU:       CpuMetrics{Usage: 50.0},
			Memory:    MemoryMetrics{UsagePercent: 60.0},
		}

		mb.Add("server1", metrics)

		mb.mu.Lock()
		count := len(mb.items)
		mb.mu.Unlock()

		if count != 1 {
			t.Errorf("Expected 1 item in buffer, got %d", count)
		}
	})

	t.Run("Flush clears buffer", func(t *testing.T) {
		mb := NewMetricsBuffer(time.Hour, 100)
		defer mb.Close()

		metrics := &SystemMetrics{
			Timestamp: time.Now(),
			CPU:       CpuMetrics{Usage: 50.0},
		}

		mb.Add("server1", metrics)
		mb.Flush()

		mb.mu.Lock()
		count := len(mb.items)
		mb.mu.Unlock()

		if count != 0 {
			t.Errorf("Expected 0 items after flush, got %d", count)
		}
	})
}

// TestAggBuffer tests the AggBuffer functionality
func TestAggBuffer(t *testing.T) {
	t.Run("NewAggBuffer", func(t *testing.T) {
		ab := NewAggBuffer(time.Second)
		if ab == nil {
			t.Fatal("NewAggBuffer returned nil")
		}
		defer ab.Close()
	})

	t.Run("Stats returns counts", func(t *testing.T) {
		ab := NewAggBuffer(time.Hour)
		defer ab.Close()

		metricsCount, pingCount := ab.Stats()
		if metricsCount != 0 || pingCount != 0 {
			t.Errorf("Expected empty stats, got metrics=%d, ping=%d", metricsCount, pingCount)
		}
	})
}

// TestGetMetricsTable tests table name resolution
func TestGetMetricsTable(t *testing.T) {
	tests := []struct {
		granularity string
		expected    string
	}{
		{"5sec", "metrics_5sec"},
		{"2min", "metrics_2min"},
		{"15min", "metrics_15min_agg"},
		{"hourly", "metrics_hourly_agg"},
		{"daily", "metrics_daily_agg"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.granularity, func(t *testing.T) {
			result := getMetricsTable(tt.granularity)
			if result != tt.expected {
				t.Errorf("getMetricsTable(%q) = %q, want %q", tt.granularity, result, tt.expected)
			}
		})
	}
}

// TestGetPingTable tests ping table name resolution
func TestGetPingTable(t *testing.T) {
	tests := []struct {
		granularity string
		expected    string
	}{
		{"5sec", "ping_5sec"},
		{"2min", "ping_2min"},
		{"15min", "ping_15min_agg"},
		{"hourly", "ping_hourly_agg"},
		{"daily", "ping_daily_agg"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.granularity, func(t *testing.T) {
			result := getPingTable(tt.granularity)
			if result != tt.expected {
				t.Errorf("getPingTable(%q) = %q, want %q", tt.granularity, result, tt.expected)
			}
		})
	}
}

// TestDBWriter tests the DBWriter functionality
func TestDBWriter(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()

	t.Run("NewDBWriter", func(t *testing.T) {
		writer := NewDBWriter(helper.db, 100)
		if writer == nil {
			t.Fatal("NewDBWriter returned nil")
		}
		writer.Close()
	})

	t.Run("GetDB returns database", func(t *testing.T) {
		writer := NewDBWriter(helper.db, 100)
		defer writer.Close()

		db := writer.GetDB()
		if db == nil {
			t.Error("GetDB returned nil")
		}
		if db != helper.db {
			t.Error("GetDB returned different database")
		}
	})

	t.Run("WriteSync executes operation", func(t *testing.T) {
		writer := NewDBWriter(helper.db, 100)
		defer writer.Close()

		executed := false
		err := writer.WriteSync(func(db *sql.DB) error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("WriteSync returned error: %v", err)
		}
		if !executed {
			t.Error("WriteSync did not execute operation")
		}
	})
}

// TestAggBufferKey tests buffer key comparison
func TestAggBufferKey(t *testing.T) {
	key1 := AggBufferKey{ServerID: "s1", Granularity: "5sec", Bucket: 100}
	key2 := AggBufferKey{ServerID: "s1", Granularity: "5sec", Bucket: 100}
	key3 := AggBufferKey{ServerID: "s2", Granularity: "5sec", Bucket: 100}

	if key1 != key2 {
		t.Error("Identical keys should be equal")
	}
	if key1 == key3 {
		t.Error("Different keys should not be equal")
	}
}

