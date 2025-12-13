package common

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
	ReadSpeed    uint64   `json:"read_speed,omitempty"`  // Bytes per second
	WriteSpeed   uint64   `json:"write_speed,omitempty"` // Bytes per second
}

type NetworkMetrics struct {
	Interfaces []NetworkInterface `json:"interfaces"`
	TotalRx    uint64             `json:"total_rx"`
	TotalTx    uint64             `json:"total_tx"`
	RxSpeed    uint64             `json:"rx_speed"`
	TxSpeed    uint64             `json:"tx_speed"`
	DailyRx    uint64             `json:"daily_rx,omitempty"` // Daily received bytes
	DailyTx    uint64             `json:"daily_tx,omitempty"` // Daily transmitted bytes
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
	Type       string   `json:"type,omitempty"` // "icmp" or "tcp"
	Port       int      `json:"port,omitempty"` // Port for TCP connections
	LatencyMs  *float64 `json:"latency_ms"`
	PacketLoss float64  `json:"packet_loss"`
	Status     string   `json:"status"`
}

type PingTargetConfig struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Type string `json:"type,omitempty"` // "icmp" or "tcp", default "icmp"
	Port int    `json:"port,omitempty"` // Port for TCP connections, default 80
}

// ============================================================================
// Batch Metrics Types (for offline buffering and aggregation)
// ============================================================================

// AggregatedMetrics represents pre-aggregated metrics for efficient transmission
type AggregatedMetrics struct {
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	SampleCount int    `json:"sample_count"`

	// CPU
	CPUAvg float32 `json:"cpu_avg"`
	CPUMax float32 `json:"cpu_max"`

	// Memory
	MemoryAvg float32 `json:"memory_avg"`
	MemoryMax float32 `json:"memory_max"`

	// Disk
	DiskAvg float32 `json:"disk_avg"`
	DiskMax float32 `json:"disk_max"`

	// Network - cumulative counters
	NetRxMax uint64 `json:"net_rx_max"`
	NetTxMax uint64 `json:"net_tx_max"`

	// Load average
	LoadOneAvg     float64 `json:"load_one_avg"`
	LoadFiveAvg    float64 `json:"load_five_avg"`
	LoadFifteenAvg float64 `json:"load_fifteen_avg"`

	// Uptime
	UptimeMax uint64 `json:"uptime_max"`

	// Ping (optional)
	PingAvg *float64 `json:"ping_avg,omitempty"`

	// Last metrics snapshot for static data (hostname, OS info, etc.)
	LastMetrics *SystemMetrics `json:"last_metrics,omitempty"`
}

// BatchMetricsMessage is sent by agent with multiple metrics (for offline sync)
type BatchMetricsMessage struct {
	Type       string               `json:"type"` // "batch_metrics"
	BatchID    string               `json:"batch_id"`
	Metrics    []TimestampedMetrics `json:"metrics,omitempty"`
	Aggregated []*AggregatedMetrics `json:"aggregated,omitempty"`
}

// TimestampedMetrics wraps metrics with explicit timestamp for batch sending
type TimestampedMetrics struct {
	Timestamp string         `json:"timestamp"`
	Metrics   *SystemMetrics `json:"metrics"`
}

// BatchMetricsResponse is the server response for batch metrics
type BatchMetricsResponse struct {
	Type     string  `json:"type"` // "batch_ack"
	BatchID  string  `json:"batch_id"`
	Accepted int     `json:"accepted"`
	Rejected int     `json:"rejected"`
	LastSeen *string `json:"last_seen,omitempty"` // Last timestamp server has seen
	Error    string  `json:"error,omitempty"`
}

// ============================================================================
// Multi-Granularity Aggregation Types (for agent-side aggregation)
// ============================================================================

// BucketData represents a single aggregated data bucket
type BucketData struct {
	Bucket      int64   `json:"bucket"`       // Unix timestamp / interval
	CPUSum      float64 `json:"cpu_sum"`      // Sum of CPU usage for averaging
	CPUMax      float64 `json:"cpu_max"`      // Max CPU usage
	MemorySum   float64 `json:"memory_sum"`   // Sum of memory usage for averaging
	MemoryMax   float64 `json:"memory_max"`   // Max memory usage
	DiskSum     float64 `json:"disk_sum"`     // Sum of disk usage for averaging
	NetRx       uint64  `json:"net_rx"`       // Max network RX (cumulative counter)
	NetTx       uint64  `json:"net_tx"`       // Max network TX (cumulative counter)
	PingSum     float64 `json:"ping_sum"`     // Sum of ping latency for averaging
	PingCount   int     `json:"ping_count"`   // Number of ping samples
	SampleCount int     `json:"sample_count"` // Number of samples in this bucket
}

// PingBucketData represents ping metrics for a specific target in a bucket
type PingBucketData struct {
	Bucket       int64   `json:"bucket"`        // Unix timestamp / interval
	TargetName   string  `json:"target_name"`   // Ping target name
	TargetHost   string  `json:"target_host"`   // Ping target host
	LatencySum   float64 `json:"latency_sum"`   // Sum of latency for averaging
	LatencyMax   float64 `json:"latency_max"`   // Max latency
	LatencyCount int     `json:"latency_count"` // Number of latency samples
	OkCount      int     `json:"ok_count"`      // Number of successful pings
	FailCount    int     `json:"fail_count"`    // Number of failed pings
}

// GranularityData contains aggregated data for a specific time granularity
type GranularityData struct {
	Granularity string           `json:"granularity"` // "5sec", "2min", "15min", "hourly", "daily"
	Interval    int              `json:"interval"`    // Bucket interval in seconds
	Metrics     []BucketData     `json:"metrics"`     // Aggregated metrics buckets
	Ping        []PingBucketData `json:"ping,omitempty"` // Aggregated ping buckets
}

// MultiGranularityMetrics contains aggregated data at multiple granularities
type MultiGranularityMetrics struct {
	Type          string            `json:"type"` // "aggregated_metrics"
	Granularities []GranularityData `json:"granularities"`
	LastMetrics   *SystemMetrics    `json:"last_metrics,omitempty"` // Latest raw metrics for real-time display
}

// Granularity constants (bucket intervals in seconds)
const (
	Granularity5Sec   = 5     // 1H view: 720 points
	Granularity2Min   = 120   // 24H view: 720 points
	Granularity15Min  = 900   // 7D view: 672 points
	GranularityHourly = 3600  // 30D view: 720 points
	GranularityDaily  = 86400 // 1Y view: 365 points
)

