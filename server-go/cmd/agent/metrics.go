package main

import (
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gopsutilnet "github.com/shirou/gopsutil/v4/net"
)

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
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		lastNetworkTime:   time.Now(),
		lastDiskIO:        make(map[string]disk.IOCountersStat),
		lastDiskIOTime:    time.Now(),
		pingResults:       nil, // Will be set when ping targets are configured
		dailyTrafficStats: loadDailyTrafficStats(),
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

	// Only include ping if there are targets configured
	var pingPtr *PingMetrics
	if ping != nil && len(ping.Targets) > 0 {
		pingPtr = ping
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

// pingLoop runs in the background to periodically collect ping metrics
func (mc *MetricsCollector) pingLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mc.customTargetsMu.RLock()
		customTargets := mc.customPingTargets
		mc.customTargetsMu.RUnlock()

		results := collectPingMetrics(mc.gatewayIP, customTargets)

		mc.pingResultsMu.Lock()
		mc.pingResults = results
		mc.pingResultsMu.Unlock()
	}
}
