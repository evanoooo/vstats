package main

import (
	"sync"
	"time"
)

// HistoryCache provides in-memory caching for history queries
type HistoryCache struct {
	mu      sync.RWMutex
	entries map[string]*HistoryCacheEntry
	ttl     time.Duration
}

// HistoryCacheEntry stores cached history data with metadata
type HistoryCacheEntry struct {
	Data        []HistoryPoint
	PingTargets []PingHistoryTarget
	LastBucket  int64     // Last bucket number for incremental updates
	UpdatedAt   time.Time
	Range       string
}

// Global cache instance
var historyCache *HistoryCache

// InitHistoryCache initializes the global history cache
func InitHistoryCache(ttl time.Duration) {
	historyCache = &HistoryCache{
		entries: make(map[string]*HistoryCacheEntry),
		ttl:     ttl,
	}
	// Start cleanup goroutine
	go historyCache.cleanup()
}

// cacheKey generates a cache key for a server and range
func cacheKey(serverID, rangeStr string) string {
	return serverID + ":" + rangeStr
}

// Get retrieves cached data if available and not expired
func (c *HistoryCache) Get(serverID, rangeStr string) (*HistoryCacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := cacheKey(serverID, rangeStr)
	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(entry.UpdatedAt) > c.ttl {
		return nil, false
	}

	return entry, true
}

// Set stores data in the cache
func (c *HistoryCache) Set(serverID, rangeStr string, data []HistoryPoint, pingTargets []PingHistoryTarget, lastBucket int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := cacheKey(serverID, rangeStr)
	c.entries[key] = &HistoryCacheEntry{
		Data:        data,
		PingTargets: pingTargets,
		LastBucket:  lastBucket,
		UpdatedAt:   time.Now(),
		Range:       rangeStr,
	}
}

// Update appends new data to existing cache entry
func (c *HistoryCache) Update(serverID, rangeStr string, newData []HistoryPoint, newPingTargets []PingHistoryTarget, newLastBucket int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := cacheKey(serverID, rangeStr)
	entry, exists := c.entries[key]
	if !exists {
		// No existing entry, create new
		c.entries[key] = &HistoryCacheEntry{
			Data:        newData,
			PingTargets: newPingTargets,
			LastBucket:  newLastBucket,
			UpdatedAt:   time.Now(),
			Range:       rangeStr,
		}
		return
	}

	// Calculate cutoff bucket based on range
	var cutoffBucket int64
	now := time.Now().UTC()
	switch rangeStr {
	case "1h":
		cutoffBucket = now.Add(-time.Hour).Unix() / 5
	case "24h", "":
		cutoffBucket = now.Add(-24*time.Hour).Unix() / 120
	default:
		// For other ranges, just replace
		c.entries[key] = &HistoryCacheEntry{
			Data:        newData,
			PingTargets: newPingTargets,
			LastBucket:  newLastBucket,
			UpdatedAt:   time.Now(),
			Range:       rangeStr,
		}
		return
	}

	// Filter out old data points and append new ones
	var filteredData []HistoryPoint
	for _, point := range entry.Data {
		bucket := timestampToBucket(point.Timestamp, rangeStr)
		if bucket >= cutoffBucket {
			filteredData = append(filteredData, point)
		}
	}

	// Append new data
	filteredData = append(filteredData, newData...)

	// Merge ping targets
	mergedPing := mergePingTargets(entry.PingTargets, newPingTargets, cutoffBucket, rangeStr)

	entry.Data = filteredData
	entry.PingTargets = mergedPing
	entry.LastBucket = newLastBucket
	entry.UpdatedAt = time.Now()
}

// timestampToBucket converts a timestamp string to bucket number
func timestampToBucket(timestamp string, rangeStr string) int64 {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// Try parsing as datetime format from SQLite
		t, err = time.Parse("2006-01-02 15:04:05", timestamp)
		if err != nil {
			return 0
		}
	}
	switch rangeStr {
	case "1h":
		return t.Unix() / 5
	case "24h", "":
		return t.Unix() / 120
	default:
		return t.Unix()
	}
}

// mergePingTargets merges old and new ping targets
func mergePingTargets(old, new []PingHistoryTarget, cutoffBucket int64, rangeStr string) []PingHistoryTarget {
	// Create a map of target name to data
	targetMap := make(map[string]*PingHistoryTarget)

	// Add filtered old data
	for _, target := range old {
		var filteredData []PingHistoryPoint
		for _, point := range target.Data {
			bucket := timestampToBucket(point.Timestamp, rangeStr)
			if bucket >= cutoffBucket {
				filteredData = append(filteredData, point)
			}
		}
		if len(filteredData) > 0 {
			targetMap[target.Name] = &PingHistoryTarget{
				Name: target.Name,
				Host: target.Host,
				Data: filteredData,
			}
		}
	}

	// Append new data
	for _, target := range new {
		if existing, ok := targetMap[target.Name]; ok {
			existing.Data = append(existing.Data, target.Data...)
		} else {
			targetMap[target.Name] = &PingHistoryTarget{
				Name: target.Name,
				Host: target.Host,
				Data: target.Data,
			}
		}
	}

	// Convert back to slice
	var result []PingHistoryTarget
	for _, t := range targetMap {
		result = append(result, *t)
	}
	return result
}

// Invalidate removes a cache entry
func (c *HistoryCache) Invalidate(serverID, rangeStr string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, cacheKey(serverID, rangeStr))
}

// cleanup periodically removes expired entries
func (c *HistoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.Sub(entry.UpdatedAt) > c.ttl*2 {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}
