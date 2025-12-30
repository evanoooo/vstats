package common

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

// TrafficConfig contains traffic limit settings from server
type TrafficConfig struct {
	MonthlyLimitGB float64 `json:"monthly_limit_gb"` // Monthly traffic limit in GB (0 = unlimited)
	ThresholdType  string  `json:"threshold_type"`   // "sum", "max", "up", "down"
	ResetDay       int     `json:"reset_day"`        // Day of month to reset (1-28)
}

type ServerResponse struct {
	Type        string             `json:"type"`
	Status      string             `json:"status,omitempty"`
	Message     string             `json:"message,omitempty"`
	Command     string             `json:"command,omitempty"`
	DownloadURL string             `json:"download_url,omitempty"`
	Force       bool               `json:"force,omitempty"`
	PingTargets []PingTargetConfig `json:"ping_targets,omitempty"`
	// Traffic config
	TrafficConfig *TrafficConfig `json:"traffic_config,omitempty"`
	// Batch metrics response fields
	BatchID   string  `json:"batch_id,omitempty"`
	Accepted  int     `json:"accepted,omitempty"`
	Rejected  int     `json:"rejected,omitempty"`
	LastSeen  *string `json:"last_seen,omitempty"` // Last timestamp server has seen for this server
	// Resumable sync fields - last bucket for each granularity
	LastBuckets map[string]int64 `json:"last_buckets,omitempty"` // granularity -> last bucket
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

