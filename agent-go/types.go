package main

import "time"

// ============================================================================
// System Metrics Types
// ============================================================================

type SystemMetrics struct {
	Timestamp   time.Time      `json:"timestamp"`
	Hostname    string         `json:"hostname"`
	OS          OsInfo         `json:"os"`
	CPU         CpuMetrics     `json:"cpu"`
	Memory      MemoryMetrics  `json:"memory"`
	Disks       []DiskMetrics  `json:"disks"`
	Network     NetworkMetrics `json:"network"`
	Uptime      uint64         `json:"uptime"`
	LoadAverage LoadAverage    `json:"load_average"`
	Ping        *PingMetrics   `json:"ping,omitempty"`
	Version     string         `json:"version,omitempty"`
	IPAddresses []string       `json:"ip_addresses,omitempty"`
}

type OsInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Kernel  string `json:"kernel"`
	Arch    string `json:"arch"`
}

type CpuMetrics struct {
	Brand     string    `json:"brand"`
	Cores     int       `json:"cores"`
	Usage     float32   `json:"usage"`
	Frequency uint64    `json:"frequency"`
	PerCore   []float32 `json:"per_core"`
}

type MemoryMetrics struct {
	Total        uint64         `json:"total"`
	Used         uint64         `json:"used"`
	Available    uint64         `json:"available"`
	SwapTotal    uint64         `json:"swap_total"`
	SwapUsed     uint64         `json:"swap_used"`
	UsagePercent float32        `json:"usage_percent"`
	Modules      []MemoryModule `json:"modules,omitempty"`
}

type MemoryModule struct {
	Slot         string `json:"slot,omitempty"`
	Size         uint64 `json:"size"`
	MemType      string `json:"mem_type,omitempty"`
	Speed        uint32 `json:"speed,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
}

type DiskMetrics struct {
	Name         string   `json:"name"`
	Model        string   `json:"model,omitempty"`
	Serial       string   `json:"serial,omitempty"`
	Total        uint64   `json:"total"`
	DiskType     string   `json:"disk_type,omitempty"`
	MountPoints  []string `json:"mount_points,omitempty"`
	UsagePercent float32  `json:"usage_percent"`
	Used         uint64   `json:"used"`
}

type NetworkMetrics struct {
	Interfaces []NetworkInterface `json:"interfaces"`
	TotalRx    uint64             `json:"total_rx"`
	TotalTx    uint64             `json:"total_tx"`
	RxSpeed    uint64             `json:"rx_speed"`
	TxSpeed    uint64             `json:"tx_speed"`
}

type NetworkInterface struct {
	Name      string `json:"name"`
	MAC       string `json:"mac,omitempty"`
	Speed     uint32 `json:"speed,omitempty"`
	RxBytes   uint64 `json:"rx_bytes"`
	TxBytes   uint64 `json:"tx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	TxPackets uint64 `json:"tx_packets"`
}

type LoadAverage struct {
	One     float64 `json:"one"`
	Five    float64 `json:"five"`
	Fifteen float64 `json:"fifteen"`
}

type PingMetrics struct {
	Targets []PingTarget `json:"targets"`
}

type PingTarget struct {
	Name       string   `json:"name"`
	Host       string   `json:"host"`
	LatencyMs  *float64 `json:"latency_ms"`
	PacketLoss float64  `json:"packet_loss"`
	Status     string   `json:"status"`
}

type PingTargetConfig struct {
	Name string `json:"name"`
	Host string `json:"host"`
}

// ============================================================================
// WebSocket Message Types
// ============================================================================

type AuthMessage struct {
	Type     string `json:"type"`
	ServerID string `json:"server_id"`
	Token    string `json:"token"`
	Version  string `json:"version"`
}

type MetricsMessage struct {
	Type    string        `json:"type"`
	Metrics SystemMetrics `json:"metrics"`
}

type ServerResponse struct {
	Type        string            `json:"type"`
	Status      string            `json:"status,omitempty"`
	Message     string            `json:"message,omitempty"`
	Command     string            `json:"command,omitempty"`
	DownloadURL string            `json:"download_url,omitempty"`
	PingTargets []PingTargetConfig `json:"ping_targets,omitempty"`
}

// ============================================================================
// Registration Types
// ============================================================================

type RegisterRequest struct {
	Name     string `json:"name"`
	Location string `json:"location"`
	Provider string `json:"provider"`
}

type RegisterResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

