package main

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

type MetricsCollector struct {
	mu                sync.RWMutex
	lastNetworkRx     uint64
	lastNetworkTx     uint64
	lastNetworkTime   time.Time
	pingResults       *PingMetrics
	pingResultsMu     sync.RWMutex
	customPingTargets []PingTargetConfig
	customTargetsMu   sync.RWMutex
	gatewayIP         string
	ipAddresses       []string
}

func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		lastNetworkTime: time.Now(),
		pingResults:     &PingMetrics{Targets: []PingTarget{}},
	}

	// Get initial network totals
	netIO, _ := net.IOCounters(true)
	for _, io := range netIO {
		mc.lastNetworkRx += io.BytesRecv
		mc.lastNetworkTx += io.BytesSent
	}

	// Detect gateway
	mc.gatewayIP = detectGateway()

	// Collect IP addresses
	mc.ipAddresses = collectIPAddresses()

	// Start background ping thread
	go mc.pingLoop()

	return mc
}

func (mc *MetricsCollector) SetPingTargets(targets []PingTargetConfig) {
	mc.customTargetsMu.Lock()
	defer mc.customTargetsMu.Unlock()
	mc.customPingTargets = targets
}

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
	swapInfo, _ := mem.SwapMemory()

	// Disk metrics
	partitions, _ := disk.Partitions(false)
	var diskMetrics []DiskMetrics
	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		diskMetrics = append(diskMetrics, DiskMetrics{
			Name:         p.Device,
			Total:        usage.Total,
			Used:         usage.Used,
			UsagePercent: float32(usage.UsedPercent),
			DiskType:     "SSD",
			MountPoints:  []string{p.Mountpoint},
		})
	}

	// Network metrics
	netIO, _ := net.IOCounters(true)
	var interfaces []NetworkInterface
	var totalRx, totalTx uint64

	for _, io := range netIO {
		// Filter out virtual interfaces
		name := strings.ToLower(io.Name)
		if isVirtualInterface(name) {
			continue
		}

		interfaces = append(interfaces, NetworkInterface{
			Name:      io.Name,
			RxBytes:   io.BytesRecv,
			TxBytes:   io.BytesSent,
			RxPackets: io.PacketsRecv,
			TxPackets: io.PacketsSent,
		})
		totalRx += io.BytesRecv
		totalTx += io.BytesSent
	}

	// Calculate network speed
	mc.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(mc.lastNetworkTime).Seconds()
	var rxSpeed, txSpeed uint64
	if elapsed > 0.1 {
		rxDiff := totalRx - mc.lastNetworkRx
		txDiff := totalTx - mc.lastNetworkTx
		if totalRx >= mc.lastNetworkRx {
			rxSpeed = uint64(float64(rxDiff) / elapsed)
		}
		if totalTx >= mc.lastNetworkTx {
			txSpeed = uint64(float64(txDiff) / elapsed)
		}
		mc.lastNetworkRx = totalRx
		mc.lastNetworkTx = totalTx
		mc.lastNetworkTime = now
	}
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
		},
		Disks: diskMetrics,
		Network: NetworkMetrics{
			Interfaces: interfaces,
			TotalRx:    totalRx,
			TotalTx:    totalTx,
			RxSpeed:    rxSpeed,
			TxSpeed:    txSpeed,
		},
		Uptime:      uptime,
		LoadAverage: la,
		Ping:        ping,
		Version:     "0.1.0",
	}

	if len(mc.ipAddresses) > 0 {
		metrics.IPAddresses = mc.ipAddresses
	}

	return metrics
}

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

func collectPingMetrics(gatewayIP string, customTargets []PingTargetConfig) *PingMetrics {
	var targets []PingTarget

	// Default targets
	defaultTargets := []struct {
		name string
		host string
	}{
		{"Google DNS", "8.8.8.8"},
		{"Cloudflare", "1.1.1.1"},
		{"Local Gateway", gatewayIP},
	}

	pingedHosts := make(map[string]bool)

	// Ping default targets
	for _, dt := range defaultTargets {
		if dt.host == "" {
			continue
		}
		if pingedHosts[dt.host] {
			continue
		}

		latency, packetLoss, status := pingHost(dt.host)
		targets = append(targets, PingTarget{
			Name:       dt.name,
			Host:       dt.host,
			LatencyMs:  latency,
			PacketLoss: packetLoss,
			Status:     status,
		})
		pingedHosts[dt.host] = true
	}

	// Ping custom targets
	for _, ct := range customTargets {
		if ct.Host == "" || pingedHosts[ct.Host] {
			continue
		}

		latency, packetLoss, status := pingHost(ct.Host)
		targets = append(targets, PingTarget{
			Name:       ct.Name,
			Host:       ct.Host,
			LatencyMs:  latency,
			PacketLoss: packetLoss,
			Status:     status,
		})
		pingedHosts[ct.Host] = true
	}

	return &PingMetrics{Targets: targets}
}

func pingHost(host string) (*float64, float64, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "3", "-w", "2000", host)
	} else {
		cmd = exec.CommandContext(ctx, "ping", "-c", "3", "-W", "2", host)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, 100.0, "error"
	}

	outputStr := string(output)
	// Simple parsing - in production, use proper regex
	status := "ok"
	packetLoss := 0.0
	var latency *float64

	if strings.Contains(outputStr, "100%") || strings.Contains(outputStr, "timeout") {
		status = "timeout"
		packetLoss = 100.0
	} else {
		// Try to extract latency (simplified)
		// In production, parse properly based on OS
		latency = floatPtr(50.0) // Placeholder
	}

	return latency, packetLoss, status
}

func detectGateway() string {
	// Simplified gateway detection
	// In production, implement proper detection for each OS
	return ""
}

func collectIPAddresses() []string {
	// Simplified IP collection
	// In production, implement proper collection for each OS
	return []string{}
}

func isVirtualInterface(name string) bool {
	return name == "lo" || name == "lo0" ||
		strings.HasPrefix(name, "veth") ||
		strings.HasPrefix(name, "docker") ||
		strings.HasPrefix(name, "br-") ||
		strings.HasPrefix(name, "virbr") ||
		strings.HasPrefix(name, "utun") ||
		strings.HasPrefix(name, "awdl") ||
		strings.HasPrefix(name, "llw")
}

func floatPtr(f float64) *float64 {
	return &f
}

