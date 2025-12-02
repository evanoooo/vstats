package main

import (
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

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
		},
		Uptime:      uptime,
		LoadAverage: la,
	}
}

