package main

import (
	"bufio"
	"context"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
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

// LocalMetricsCollector handles local metrics collection including ping
type LocalMetricsCollector struct {
	mu              sync.RWMutex
	lastNetworkRx   uint64
	lastNetworkTx   uint64
	lastNetworkTime time.Time
	pingResults     *PingMetrics
	pingResultsMu   sync.RWMutex
	pingTargets     []PingTargetConfig
	pingTargetsMu   sync.RWMutex
	gatewayIP       string
}

var localCollector *LocalMetricsCollector
var localCollectorOnce sync.Once

// GetLocalCollector returns the singleton local metrics collector
func GetLocalCollector() *LocalMetricsCollector {
	localCollectorOnce.Do(func() {
		localCollector = &LocalMetricsCollector{
			lastNetworkTime: time.Now(),
		}

		// Get initial network totals
		netIO, _ := net.IOCounters(true)
		for _, io := range netIO {
			localCollector.lastNetworkRx += io.BytesRecv
			localCollector.lastNetworkTx += io.BytesSent
		}

		// Detect gateway
		localCollector.gatewayIP = detectGateway()

		// Start background ping loop
		go localCollector.pingLoop()
	})
	return localCollector
}

// SetPingTargets updates the ping targets for local collector
func (lc *LocalMetricsCollector) SetPingTargets(targets []PingTargetConfig) {
	lc.pingTargetsMu.Lock()
	defer lc.pingTargetsMu.Unlock()
	lc.pingTargets = targets
}

// pingLoop runs ping tests periodically
func (lc *LocalMetricsCollector) pingLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		lc.pingTargetsMu.RLock()
		targets := lc.pingTargets
		lc.pingTargetsMu.RUnlock()

		if len(targets) == 0 {
			continue
		}

		results := collectLocalPingMetrics(targets)

		lc.pingResultsMu.Lock()
		lc.pingResults = results
		lc.pingResultsMu.Unlock()
	}
}

// getPingResults returns the cached ping results
func (lc *LocalMetricsCollector) getPingResults() *PingMetrics {
	lc.pingResultsMu.RLock()
	defer lc.pingResultsMu.RUnlock()
	return lc.pingResults
}

// collectLocalPingMetrics executes ping tests for given targets
func collectLocalPingMetrics(targets []PingTargetConfig) *PingMetrics {
	if len(targets) == 0 {
		return nil
	}

	var pingTargets []PingTarget
	pingedHosts := make(map[string]bool)

	for _, ct := range targets {
		if ct.Host == "" || pingedHosts[ct.Host] {
			continue
		}

		latency, packetLoss, status := pingHost(ct.Host)
		pingTargets = append(pingTargets, PingTarget{
			Name:       ct.Name,
			Host:       ct.Host,
			LatencyMs:  latency,
			PacketLoss: packetLoss,
			Status:     status,
		})
		pingedHosts[ct.Host] = true
	}

	if len(pingTargets) == 0 {
		return nil
	}

	return &PingMetrics{Targets: pingTargets}
}

// pingHost executes a ping test to the specified host
func pingHost(host string) (*float64, float64, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "3", "-w", "2000", host)
	} else if runtime.GOOS == "darwin" {
		cmd = exec.CommandContext(ctx, "ping", "-c", "3", "-W", "2000", host)
	} else {
		cmd = exec.CommandContext(ctx, "ping", "-c", "3", "-W", "2", host)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, 100.0, "error"
	}

	outputStr := string(output)
	status := "ok"
	packetLoss := 0.0
	var latency *float64

	// Parse packet loss
	if strings.Contains(outputStr, "100%") || strings.Contains(outputStr, "timeout") {
		status = "timeout"
		packetLoss = 100.0
	} else {
		packetLossRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)%\s*(?:packet\s+)?loss`)
		if matches := packetLossRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
			if loss, err := strconv.ParseFloat(matches[1], 64); err == nil {
				packetLoss = loss
			}
		} else {
			altRegex := regexp.MustCompile(`(\d+)\s+packets\s+transmitted.*?(\d+)\s+received`)
			if matches := altRegex.FindStringSubmatch(outputStr); len(matches) >= 3 {
				if transmitted, err1 := strconv.ParseFloat(matches[1], 64); err1 == nil {
					if received, err2 := strconv.ParseFloat(matches[2], 64); err2 == nil && transmitted > 0 {
						packetLoss = ((transmitted - received) / transmitted) * 100.0
					}
				}
			}
		}
	}

	// Parse average latency
	if runtime.GOOS == "windows" {
		avgRegex := regexp.MustCompile(`Average\s*=\s*(\d+(?:\.\d+)?)\s*ms`)
		if matches := avgRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
			if lat, err := strconv.ParseFloat(matches[1], 64); err == nil {
				latency = &lat
			}
		}
	} else {
		lines := strings.Split(outputStr, "\n")
		var statsLine string
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), "avg") || strings.Contains(line, "Average") {
				statsLine = line
				break
			}
		}

		if statsLine != "" {
			if strings.Contains(statsLine, "/") {
				parts := strings.Split(statsLine, "=")
				if len(parts) < 2 {
					parts = strings.Split(statsLine, ":")
				}
				if len(parts) >= 2 {
					values := strings.Fields(parts[1])
					if len(values) > 0 {
						nums := strings.Split(values[0], "/")
						if len(nums) >= 2 {
							if lat, err := strconv.ParseFloat(nums[1], 64); err == nil {
								latency = &lat
							}
						}
					}
				}
			} else {
				avgRegex := regexp.MustCompile(`Average\s*[=:]\s*(\d+(?:\.\d+)?)\s*ms`)
				if matches := avgRegex.FindStringSubmatch(statsLine); len(matches) > 1 {
					if lat, err := strconv.ParseFloat(matches[1], 64); err == nil {
						latency = &lat
					}
				}
			}
		}

		if latency == nil {
			msRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*ms`)
			matches := msRegex.FindAllStringSubmatch(outputStr, -1)
			if len(matches) > 0 {
				if lat, err := strconv.ParseFloat(matches[len(matches)-1][1], 64); err == nil {
					latency = &lat
				}
			}
		}
	}

	if packetLoss >= 100.0 {
		status = "timeout"
	} else if latency == nil && packetLoss > 0 {
		status = "error"
	} else if latency == nil {
		status = "error"
	}

	return latency, packetLoss, status
}

// detectGateway detects the default gateway IP
func detectGateway() string {
	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("ip", "route", "show", "default")
		output, err := cmd.Output()
		if err == nil {
			outputStr := string(output)
			fields := strings.Fields(outputStr)
			for i, field := range fields {
				if field == "via" && i+1 < len(fields) {
					gateway := fields[i+1]
					if strings.Contains(gateway, ".") && !strings.Contains(gateway, "/") {
						return gateway
					}
				}
			}
		}
	case "darwin":
		cmd := exec.Command("route", "-n", "get", "default")
		output, err := cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "gateway:") {
					parts := strings.Fields(line)
					if len(parts) > 1 {
						return parts[1]
					}
				}
			}
		}
	case "windows":
		cmd := exec.Command("powershell", "-Command", "(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Select-Object -First 1).NextHop")
		output, err := cmd.Output()
		if err == nil {
			gateway := strings.TrimSpace(string(output))
			if gateway != "" && strings.Contains(gateway, ".") {
				return gateway
			}
		}
	}
	return ""
}

func CollectMetrics() SystemMetrics {
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
		// Filter for main disk
		if p.Mountpoint != "/" && !strings.HasPrefix(p.Mountpoint, "C:") {
			continue
		}
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
		if name == "lo" || name == "lo0" ||
			strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "docker") ||
			strings.HasPrefix(name, "br-") ||
			strings.HasPrefix(name, "virbr") ||
			strings.HasPrefix(name, "utun") ||
			strings.HasPrefix(name, "awdl") ||
			strings.HasPrefix(name, "llw") {
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

	// Get ping results from local collector
	lc := GetLocalCollector()
	pingResults := lc.getPingResults()

	// Calculate network speed
	lc.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(lc.lastNetworkTime).Seconds()
	var rxSpeed, txSpeed uint64
	if elapsed > 0.1 {
		rxDiff := totalRx - lc.lastNetworkRx
		txDiff := totalTx - lc.lastNetworkTx
		if totalRx >= lc.lastNetworkRx {
			rxSpeed = uint64(float64(rxDiff) / elapsed)
		}
		if totalTx >= lc.lastNetworkTx {
			txSpeed = uint64(float64(txDiff) / elapsed)
		}
		lc.lastNetworkRx = totalRx
		lc.lastNetworkTx = totalTx
		lc.lastNetworkTime = now
	}
	lc.mu.Unlock()

	return SystemMetrics{
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
		Ping:        pingResults,
	}
}

