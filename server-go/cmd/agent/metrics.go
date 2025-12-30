package main

import (
	"runtime"
	"strings"
	"sync"
	"time"

	"vstats/internal/common"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gopsutilnet "github.com/shirou/gopsutil/v4/net"
)

// PingAggKey is the key for ping aggregation map
type PingAggKey struct {
	Name   string
	Bucket int64
}

// PingAggData holds aggregated ping data for a bucket
type PingAggData struct {
	Host         string
	LatencySum   float64
	LatencyMax   float64
	LatencyCount int
	OkCount      int
	FailCount    int
}

// MetricsCollector collects system metrics
type MetricsCollector struct {
	mu                sync.RWMutex
	lastNetworkRx     uint64
	lastNetworkTx     uint64
	lastNetworkTime   time.Time
	lastDiskIO        map[string]disk.IOCountersStat // Map disk name to last IO stats
	lastDiskIOTime    time.Time
	pingResults       *PingMetrics
	pingResultsMu     sync.RWMutex
	customPingTargets []PingTargetConfig
	customTargetsMu   sync.RWMutex
	gatewayIP         string
	ipAddresses       []string
	dailyTrafficStats *DailyTrafficStats
	pingInterval time.Duration // Ping collection interval (matches metrics interval)
	// Ping aggregation (all granularities, computed by Agent)
	pingAgg       map[string]map[PingAggKey]*PingAggData // key: "2min", "15min", "hourly", "daily"
	pingAggMu     sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
// intervalSecs specifies the collection interval (used for ping to match metrics sending)
func NewMetricsCollector(intervalSecs uint64) *MetricsCollector {
	if intervalSecs == 0 {
		intervalSecs = 5 // Default to 5 seconds
	}
	mc := &MetricsCollector{
		lastNetworkTime:   time.Now(),
		lastDiskIO:        make(map[string]disk.IOCountersStat),
		lastDiskIOTime:    time.Now(),
		pingResults:       nil, // Will be set when ping targets are configured
		dailyTrafficStats: loadDailyTrafficStats(),
		pingInterval:      time.Duration(intervalSecs) * time.Second,
		pingAgg: map[string]map[PingAggKey]*PingAggData{
			"2min":   make(map[PingAggKey]*PingAggData),
			"15min":  make(map[PingAggKey]*PingAggData),
			"hourly": make(map[PingAggKey]*PingAggData),
			"daily":  make(map[PingAggKey]*PingAggData),
		},
	}

	// Get initial network totals
	netIO, _ := gopsutilnet.IOCounters(true)
	var totalRx, totalTx uint64
	for _, io := range netIO {
		name := strings.ToLower(io.Name)
		if !isVirtualInterface(name) {
			totalRx += io.BytesRecv
			totalTx += io.BytesSent
		}
	}
	mc.lastNetworkRx = totalRx
	mc.lastNetworkTx = totalTx

	// Initialize daily traffic stats with current totals
	mc.dailyTrafficStats.updateDailyTraffic(totalRx, totalTx)

	// Get initial disk IO stats
	diskIO, _ := disk.IOCounters()
	for name, io := range diskIO {
		mc.lastDiskIO[name] = io
	}

	// Detect gateway
	mc.gatewayIP = detectGateway()

	// Collect IP addresses
	mc.ipAddresses = collectIPAddresses()

	// Start background ping thread
	go mc.pingLoop()

	return mc
}

// SetPingTargets sets the ping targets configuration
func (mc *MetricsCollector) SetPingTargets(targets []PingTargetConfig) {
	mc.customTargetsMu.Lock()
	defer mc.customTargetsMu.Unlock()
	mc.customPingTargets = targets
}

// SetTrafficConfig updates the traffic configuration from server
func (mc *MetricsCollector) SetTrafficConfig(config *TrafficConfig) {
	if config == nil || mc.dailyTrafficStats == nil {
		return
	}
	mc.dailyTrafficStats.SetTrafficConfig(config.MonthlyLimitGB, config.ThresholdType, config.ResetDay)
}

// GetBillingPeriodUsage returns current billing period traffic usage
func (mc *MetricsCollector) GetBillingPeriodUsage() (usageGB float64, limitGB float64, usagePercent float64) {
	if mc.dailyTrafficStats == nil {
		return 0, 0, 0
	}
	return mc.dailyTrafficStats.GetBillingPeriodUsage()
}

// Collect collects all system metrics
func (mc *MetricsCollector) Collect() SystemMetrics {
	// CPU metrics
	cpuPercent, _ := cpu.Percent(200*time.Millisecond, true)
	cpuInfo, _ := cpu.Info()

	var cpuBrand string
	var cpuFreq uint64
	if len(cpuInfo) > 0 {
		cpuBrand = cpuInfo[0].ModelName
		cpuFreq = uint64(cpuInfo[0].Mhz)
	}

	var totalCPU float32
	perCore := make([]float32, len(cpuPercent))
	for i, p := range cpuPercent {
		perCore[i] = float32(p)
		totalCPU += float32(p)
	}
	if len(cpuPercent) > 0 {
		totalCPU /= float32(len(cpuPercent))
	}

	// Memory metrics
	memInfo, _ := mem.VirtualMemory()
	swapInfo := collectSwapInfo()
	memoryModules := collectMemoryModules()

	// Disk metrics - collect physical disks with IO speed
	mc.mu.Lock()
	diskIO, _ := disk.IOCounters()
	diskMetrics := collectPhysicalDisks(diskIO, mc.lastDiskIO, mc.lastDiskIOTime)
	mc.lastDiskIO = diskIO
	mc.lastDiskIOTime = time.Now()
	mc.mu.Unlock()

	// Network metrics
	netIO, _ := gopsutilnet.IOCounters(true)
	mc.mu.Lock()
	interfaces, totalRx, totalTx, rxSpeed, txSpeed, dailyRx, dailyTx, now := collectNetworkMetrics(
		netIO,
		mc.lastNetworkRx,
		mc.lastNetworkTx,
		mc.lastNetworkTime,
		mc.dailyTrafficStats,
	)
	mc.lastNetworkRx = totalRx
	mc.lastNetworkTx = totalTx
	mc.lastNetworkTime = now
	mc.mu.Unlock()

	// Load average
	loadAvg, _ := load.Avg()
	var la LoadAverage
	if loadAvg != nil {
		la = LoadAverage{
			One:     loadAvg.Load1,
			Five:    loadAvg.Load5,
			Fifteen: loadAvg.Load15,
		}
	}

	// Host info
	hostInfo, _ := host.Info()
	uptime, _ := host.Uptime()

	// Get cached ping results
	mc.pingResultsMu.RLock()
	ping := mc.pingResults
	mc.pingResultsMu.RUnlock()

	// Include ping if there are targets configured
	var pingPtr *PingMetrics
	if ping != nil && len(ping.Targets) > 0 {
		// Create a copy with all granularity aggregation data
		pingCopy := *ping
		pingCopy.Agg2Min = mc.getPingAgg("2min")
		pingCopy.Agg15Min = mc.getPingAgg("15min")
		pingCopy.AggHourly = mc.getPingAgg("hourly")
		pingCopy.AggDaily = mc.getPingAgg("daily")
		pingPtr = &pingCopy
	}

	// GPU metrics
	gpuMetrics := collectGPUMetrics()

	metrics := SystemMetrics{
		Timestamp: time.Now().UTC(),
		Hostname:  hostInfo.Hostname,
		OS: OsInfo{
			Name:    hostInfo.Platform,
			Version: hostInfo.PlatformVersion,
			Kernel:  hostInfo.KernelVersion,
			Arch:    runtime.GOARCH,
		},
		CPU: CpuMetrics{
			Brand:     cpuBrand,
			Cores:     len(cpuPercent),
			Usage:     totalCPU,
			Frequency: cpuFreq,
			PerCore:   perCore,
		},
		Memory: MemoryMetrics{
			Total:        memInfo.Total,
			Used:         memInfo.Used,
			Available:    memInfo.Available,
			SwapTotal:    swapInfo.Total,
			SwapUsed:     swapInfo.Used,
			UsagePercent: float32(memInfo.UsedPercent),
			Modules:      memoryModules,
		},
		Disks: diskMetrics,
		Network: NetworkMetrics{
			Interfaces: interfaces,
			TotalRx:    totalRx,
			TotalTx:    totalTx,
			RxSpeed:    rxSpeed,
			TxSpeed:    txSpeed,
			DailyRx:    dailyRx,
			DailyTx:    dailyTx,
		},
		Uptime:      uptime,
		LoadAverage: la,
		Ping:        pingPtr,
		GPU:         gpuMetrics,
		Version:     AgentVersion,
	}

	if len(mc.ipAddresses) > 0 {
		metrics.IPAddresses = mc.ipAddresses
	}

	return metrics
}

// Bucket intervals in seconds for each granularity
var pingBucketIntervals = map[string]int64{
	"2min":   120,   // 2 minutes
	"15min":  900,   // 15 minutes
	"hourly": 3600,  // 1 hour
	"daily":  86400, // 1 day
}

// How many buckets to keep for each granularity
var pingBucketKeep = map[string]int64{
	"2min":   2,  // Current + previous
	"15min":  2,  // Current + previous
	"hourly": 2,  // Current + previous
	"daily":  2,  // Current + previous
}

// pingLoop runs in the background to periodically collect ping metrics
func (mc *MetricsCollector) pingLoop() {
	// Use same interval as metrics sending for accurate sampling
	ticker := time.NewTicker(mc.pingInterval)
	defer ticker.Stop()

	for range ticker.C {
		mc.customTargetsMu.RLock()
		customTargets := mc.customPingTargets
		mc.customTargetsMu.RUnlock()

		results := collectPingMetrics(mc.gatewayIP, customTargets)

		// Update all granularity aggregations
		if results != nil && len(results.Targets) > 0 {
			mc.updatePingAggAll(results.Targets, results.Timestamp)
		}

		mc.pingResultsMu.Lock()
		mc.pingResults = results
		mc.pingResultsMu.Unlock()
	}
}

// updatePingAggAll updates ping aggregation for all granularities
func (mc *MetricsCollector) updatePingAggAll(targets []PingTarget, timestamp int64) {
	mc.pingAggMu.Lock()
	defer mc.pingAggMu.Unlock()

	for granularity, interval := range pingBucketIntervals {
		bucket := timestamp / interval
		aggMap := mc.pingAgg[granularity]

		// Clean up old buckets
		minBucket := bucket - pingBucketKeep[granularity]
		for key := range aggMap {
			if key.Bucket < minBucket {
				delete(aggMap, key)
			}
		}

		// Update aggregation for each target
		for _, target := range targets {
			key := PingAggKey{Name: target.Name, Bucket: bucket}
			agg, exists := aggMap[key]
			if !exists {
				agg = &PingAggData{Host: target.Host}
				aggMap[key] = agg
			}

			// Accumulate data
			if target.LatencyMs != nil {
				agg.LatencySum += *target.LatencyMs
				if *target.LatencyMs > agg.LatencyMax {
					agg.LatencyMax = *target.LatencyMs
				}
				agg.LatencyCount++
			}
			if target.Status == "ok" {
				agg.OkCount++
			} else {
				agg.FailCount++
			}
		}
	}
}

// getPingAgg returns aggregation data for a specific granularity
func (mc *MetricsCollector) getPingAgg(granularity string) []common.PingTargetAgg {
	mc.pingAggMu.RLock()
	defer mc.pingAggMu.RUnlock()

	aggMap, exists := mc.pingAgg[granularity]
	if !exists {
		return nil
	}

	var result []common.PingTargetAgg
	for key, agg := range aggMap {
		result = append(result, common.PingTargetAgg{
			Name:         key.Name,
			Host:         agg.Host,
			Bucket:       key.Bucket,
			LatencySum:   agg.LatencySum,
			LatencyMax:   agg.LatencyMax,
			LatencyCount: agg.LatencyCount,
			OkCount:      agg.OkCount,
			FailCount:    agg.FailCount,
		})
	}
	return result
}
