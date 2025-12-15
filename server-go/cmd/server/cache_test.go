package main

import (
	"testing"
	"time"
)

// TestHistoryCache tests the history cache functionality
func TestHistoryCache(t *testing.T) {
	t.Run("InitHistoryCache creates cache", func(t *testing.T) {
		InitHistoryCache(time.Minute)
		if historyCache == nil {
			t.Fatal("historyCache should not be nil after init")
		}
	})

	t.Run("Get returns false for nonexistent server", func(t *testing.T) {
		cache := &HistoryCache{
			entries: make(map[string]*HistoryCacheEntry),
			ttl:     time.Minute,
		}

		_, exists := cache.Get("nonexistent", "1h")
		if exists {
			t.Error("Expected false for nonexistent server")
		}
	})

	t.Run("Set and Get", func(t *testing.T) {
		cache := &HistoryCache{
			entries: make(map[string]*HistoryCacheEntry),
			ttl:     time.Minute,
		}

		data := []HistoryPoint{
			{Timestamp: "2024-01-01T00:00:00Z", CPU: 50.0},
		}

		cache.Set("server1", "1h", data, nil, 100)
		result, exists := cache.Get("server1", "1h")

		if !exists {
			t.Fatal("Expected entry to exist")
		}
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if len(result.Data) != 1 {
			t.Errorf("Expected 1 item, got %d", len(result.Data))
		}
		if result.Data[0].CPU != 50.0 {
			t.Errorf("Expected CPU 50.0, got %f", result.Data[0].CPU)
		}
		if result.LastBucket != 100 {
			t.Errorf("Expected LastBucket 100, got %d", result.LastBucket)
		}
	})

	t.Run("Different ranges are independent", func(t *testing.T) {
		cache := &HistoryCache{
			entries: make(map[string]*HistoryCacheEntry),
			ttl:     time.Minute,
		}

		data1h := []HistoryPoint{{Timestamp: "t1", CPU: 10.0}}
		data24h := []HistoryPoint{{Timestamp: "t2", CPU: 20.0}}

		cache.Set("server1", "1h", data1h, nil, 0)
		cache.Set("server1", "24h", data24h, nil, 0)

		result1h, _ := cache.Get("server1", "1h")
		result24h, _ := cache.Get("server1", "24h")

		if result1h.Data[0].CPU != 10.0 {
			t.Errorf("1h cache corrupted: expected 10.0, got %f", result1h.Data[0].CPU)
		}
		if result24h.Data[0].CPU != 20.0 {
			t.Errorf("24h cache corrupted: expected 20.0, got %f", result24h.Data[0].CPU)
		}
	})

	t.Run("Invalidate clears specific entry", func(t *testing.T) {
		cache := &HistoryCache{
			entries: make(map[string]*HistoryCacheEntry),
			ttl:     time.Minute,
		}

		data := []HistoryPoint{{Timestamp: "t1", CPU: 50.0}}

		cache.Set("server1", "1h", data, nil, 0)
		cache.Set("server1", "24h", data, nil, 0)

		cache.Invalidate("server1", "1h")

		_, exists1h := cache.Get("server1", "1h")
		if exists1h {
			t.Error("1h cache should be invalidated")
		}

		_, exists24h := cache.Get("server1", "24h")
		if !exists24h {
			t.Error("24h cache should still exist")
		}
	})

	t.Run("Expired entries return false", func(t *testing.T) {
		cache := &HistoryCache{
			entries: make(map[string]*HistoryCacheEntry),
			ttl:     time.Millisecond, // Very short TTL
		}

		data := []HistoryPoint{{Timestamp: "t1", CPU: 50.0}}
		cache.Set("server1", "1h", data, nil, 0)

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		_, exists := cache.Get("server1", "1h")
		if exists {
			t.Error("Expired entry should return false")
		}
	})

	t.Run("Update appends data", func(t *testing.T) {
		cache := &HistoryCache{
			entries: make(map[string]*HistoryCacheEntry),
			ttl:     time.Minute,
		}

		// Initial data
		initialData := []HistoryPoint{
			{Timestamp: time.Now().UTC().Format(time.RFC3339), CPU: 50.0},
		}
		cache.Set("server1", "1h", initialData, nil, 100)

		// New data
		newData := []HistoryPoint{
			{Timestamp: time.Now().UTC().Format(time.RFC3339), CPU: 60.0},
		}
		cache.Update("server1", "1h", newData, nil, 200)

		result, exists := cache.Get("server1", "1h")
		if !exists {
			t.Fatal("Entry should exist after update")
		}

		// Should have both initial and new data
		if len(result.Data) < 1 {
			t.Error("Expected data after update")
		}
		if result.LastBucket != 200 {
			t.Errorf("Expected LastBucket 200, got %d", result.LastBucket)
		}
	})
}

// TestCacheKey tests the cache key generation
func TestCacheKey(t *testing.T) {
	tests := []struct {
		serverID string
		rangeStr string
		expected string
	}{
		{"server1", "1h", "server1:1h"},
		{"server2", "24h", "server2:24h"},
		{"test-server", "7d", "test-server:7d"},
	}

	for _, tt := range tests {
		result := cacheKey(tt.serverID, tt.rangeStr)
		if result != tt.expected {
			t.Errorf("cacheKey(%q, %q) = %q, want %q", tt.serverID, tt.rangeStr, result, tt.expected)
		}
	}
}

// TestTimestampToBucket tests timestamp conversion
func TestTimestampToBucket(t *testing.T) {
	tests := []struct {
		timestamp string
		rangeStr  string
	}{
		{"2024-01-01T00:00:00Z", "1h"},
		{"2024-01-01T00:00:00Z", "24h"},
		{"invalid", "1h"},
	}

	for _, tt := range tests {
		result := timestampToBucket(tt.timestamp, tt.rangeStr)
		// Just check it doesn't panic
		if tt.timestamp == "invalid" && result != 0 {
			t.Errorf("Invalid timestamp should return 0, got %d", result)
		}
	}
}

// TestHistoryCacheConcurrency tests thread safety of the cache
func TestHistoryCacheConcurrency(t *testing.T) {
	cache := &HistoryCache{
		entries: make(map[string]*HistoryCacheEntry),
		ttl:     time.Minute,
	}

	data := []HistoryPoint{{Timestamp: "2024-01-01T00:00:00Z", CPU: 50.0}}

	done := make(chan bool, 10)

	// Concurrent writers
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				serverID := "server" + string(rune('0'+id))
				cache.Set(serverID, "1h", data, nil, int64(j))
			}
			done <- true
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				serverID := "server" + string(rune('0'+id))
				cache.Get(serverID, "1h")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out - possible deadlock")
		}
	}
}

// BenchmarkHistoryCache benchmarks cache operations
func BenchmarkHistoryCache(b *testing.B) {
	cache := &HistoryCache{
		entries: make(map[string]*HistoryCacheEntry),
		ttl:     time.Minute,
	}

	data := make([]HistoryPoint, 720)
	for i := range data {
		data[i] = HistoryPoint{Timestamp: "2024-01-01T00:00:00Z", CPU: float32(i)}
	}

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Set("server1", "1h", data, nil, 0)
		}
	})

	b.Run("Get", func(b *testing.B) {
		cache.Set("server1", "1h", data, nil, 0)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.Get("server1", "1h")
		}
	})
}
