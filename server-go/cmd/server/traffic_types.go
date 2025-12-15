package main

import (
	"time"
)

// ============================================================================
// Traffic Management Types
// ============================================================================

// TrafficThresholdType defines how traffic is calculated
type TrafficThresholdType string

const (
	// TrafficTypeSum - sum of upload and download traffic
	TrafficTypeSum TrafficThresholdType = "sum"
	// TrafficTypeMax - maximum of upload or download traffic
	TrafficTypeMax TrafficThresholdType = "max"
	// TrafficTypeUp - upload traffic only
	TrafficTypeUp TrafficThresholdType = "up"
	// TrafficTypeDown - download traffic only
	TrafficTypeDown TrafficThresholdType = "down"
)

// TrafficStats represents the current traffic statistics for a server
type TrafficStats struct {
	ServerID      string    `json:"server_id"`
	ServerName    string    `json:"server_name,omitempty"`
	PeriodStart   time.Time `json:"period_start"`    // Start of current billing period
	PeriodEnd     time.Time `json:"period_end"`      // End of current billing period
	ResetDay      int       `json:"reset_day"`       // Day of month to reset (1-28)
	TxBytes       uint64    `json:"tx_bytes"`        // Total uploaded bytes in this period
	RxBytes       uint64    `json:"rx_bytes"`        // Total downloaded bytes in this period
	TxBytesGB     float64   `json:"tx_bytes_gb"`     // Total uploaded in GB
	RxBytesGB     float64   `json:"rx_bytes_gb"`     // Total downloaded in GB
	TotalBytes    uint64    `json:"total_bytes"`     // Total bytes (tx + rx)
	TotalBytesGB  float64   `json:"total_bytes_gb"`  // Total in GB
	MonthlyLimitGB float64  `json:"monthly_limit_gb"` // Monthly limit in GB (0 = unlimited)
	ThresholdType string    `json:"threshold_type"`  // sum, max, up, down
	UsagePercent  float64   `json:"usage_percent"`   // Current usage percentage
	LastUpdated   time.Time `json:"last_updated"`
	// Baseline values for delta calculation
	BaselineTx    uint64    `json:"baseline_tx,omitempty"`
	BaselineRx    uint64    `json:"baseline_rx,omitempty"`
	BaselineTime  time.Time `json:"baseline_time,omitempty"`
}

// CalculateUsage calculates the usage based on threshold type
func (t *TrafficStats) CalculateUsage() float64 {
	switch TrafficThresholdType(t.ThresholdType) {
	case TrafficTypeSum:
		return t.TotalBytesGB
	case TrafficTypeMax:
		if t.TxBytesGB > t.RxBytesGB {
			return t.TxBytesGB
		}
		return t.RxBytesGB
	case TrafficTypeUp:
		return t.TxBytesGB
	case TrafficTypeDown:
		return t.RxBytesGB
	default:
		return t.TotalBytesGB
	}
}

// CalculatePercent calculates the usage percentage
func (t *TrafficStats) CalculatePercent() float64 {
	if t.MonthlyLimitGB <= 0 {
		return 0
	}
	usage := t.CalculateUsage()
	return (usage / t.MonthlyLimitGB) * 100
}

// TrafficRecord represents a historical traffic record
type TrafficRecord struct {
	ID           int64     `json:"id"`
	ServerID     string    `json:"server_id"`
	ServerName   string    `json:"server_name,omitempty"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	TxBytes      uint64    `json:"tx_bytes"`
	RxBytes      uint64    `json:"rx_bytes"`
	TxBytesGB    float64   `json:"tx_bytes_gb"`
	RxBytesGB    float64   `json:"rx_bytes_gb"`
	TotalBytes   uint64    `json:"total_bytes"`
	TotalBytesGB float64   `json:"total_bytes_gb"`
	MonthlyLimitGB float64 `json:"monthly_limit_gb"`
	UsagePercent float64   `json:"usage_percent"`
	CreatedAt    time.Time `json:"created_at"`
}

// TrafficSummary provides a summary of traffic for all servers
type TrafficSummary struct {
	TotalServers     int            `json:"total_servers"`
	ServersWithLimit int            `json:"servers_with_limit"`
	OverLimitCount   int            `json:"over_limit_count"`
	WarningCount     int            `json:"warning_count"`
	Stats            []TrafficStats `json:"stats"`
}

// TrafficHistoryQuery represents query parameters for traffic history
type TrafficHistoryQuery struct {
	ServerID  string `form:"server_id"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	Limit     int    `form:"limit"`
}

// TrafficHistoryResponse represents the response for traffic history
type TrafficHistoryResponse struct {
	Records []TrafficRecord `json:"records"`
	Total   int             `json:"total"`
}

// UpdateTrafficLimitRequest updates traffic limit for a server
type UpdateTrafficLimitRequest struct {
	ServerID      string  `json:"server_id"`
	MonthlyLimitGB float64 `json:"monthly_limit_gb"`
	ThresholdType string  `json:"threshold_type"` // sum, max, up, down
	ResetDay      int     `json:"reset_day"`      // 1-28
	Warning       float32 `json:"warning"`        // Warning threshold percentage (default 80)
}

// TrafficResetRequest manually resets traffic for a server
type TrafficResetRequest struct {
	ServerID    string `json:"server_id"`
	ResetToZero bool   `json:"reset_to_zero"` // If true, reset to zero; otherwise reset baseline
}

// DailyTrafficRecord represents daily traffic for charts
type DailyTrafficRecord struct {
	Date       string  `json:"date"`       // YYYY-MM-DD
	TxBytesGB  float64 `json:"tx_bytes_gb"`
	RxBytesGB  float64 `json:"rx_bytes_gb"`
	TotalGB    float64 `json:"total_gb"`
}

// TrafficChartData represents traffic chart data for a server
type TrafficChartData struct {
	ServerID   string               `json:"server_id"`
	ServerName string               `json:"server_name"`
	Period     string               `json:"period"` // e.g., "2024-01"
	Daily      []DailyTrafficRecord `json:"daily"`
	TotalTxGB  float64              `json:"total_tx_gb"`
	TotalRxGB  float64              `json:"total_rx_gb"`
	TotalGB    float64              `json:"total_gb"`
	LimitGB    float64              `json:"limit_gb"`
}

// ============================================================================
// Helper Functions
// ============================================================================

// BytesToGB converts bytes to gigabytes
func BytesToGB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024 * 1024)
}

// GBToBytes converts gigabytes to bytes
func GBToBytes(gb float64) uint64 {
	return uint64(gb * 1024 * 1024 * 1024)
}

// GetPeriodBounds calculates the start and end of the billing period
// based on the reset day and current time
func GetPeriodBounds(resetDay int, now time.Time) (start, end time.Time) {
	if resetDay < 1 || resetDay > 28 {
		resetDay = 1
	}

	year, month, day := now.Date()

	// Determine if we're before or after the reset day this month
	if day >= resetDay {
		// Current period started this month
		start = time.Date(year, month, resetDay, 0, 0, 0, 0, now.Location())
		// Period ends next month
		nextMonth := month + 1
		nextYear := year
		if nextMonth > 12 {
			nextMonth = 1
			nextYear++
		}
		end = time.Date(nextYear, nextMonth, resetDay, 0, 0, 0, 0, now.Location())
	} else {
		// Current period started last month
		prevMonth := month - 1
		prevYear := year
		if prevMonth < 1 {
			prevMonth = 12
			prevYear--
		}
		start = time.Date(prevYear, prevMonth, resetDay, 0, 0, 0, 0, now.Location())
		// Period ends this month
		end = time.Date(year, month, resetDay, 0, 0, 0, 0, now.Location())
	}

	return start, end
}

// GetDefaultResetDay returns the default reset day (1st of month)
func GetDefaultResetDay() int {
	return 1
}

// ValidateThresholdType validates the threshold type
func ValidateThresholdType(t string) bool {
	switch TrafficThresholdType(t) {
	case TrafficTypeSum, TrafficTypeMax, TrafficTypeUp, TrafficTypeDown:
		return true
	default:
		return false
	}
}
