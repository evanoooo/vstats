package main

import (
	"time"
)

// ============================================================================
// Alert Configuration Types
// ============================================================================

// AlertConfig holds all alert-related configuration
type AlertConfig struct {
	Enabled         bool                     `json:"enabled"`
	Channels        []NotificationChannel    `json:"channels"`
	Rules           AlertRules               `json:"rules"`
	Templates       map[string]AlertTemplate `json:"templates,omitempty"`
	GlobalCooldown  int                      `json:"global_cooldown,omitempty"`  // Seconds between same type alerts
	RecoveryNotify  bool                     `json:"recovery_notify,omitempty"` // Send notification when alert recovers
}

// NotificationChannel represents a configured notification channel
type NotificationChannel struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"` // email, telegram, discord, webhook, bark, serverchan
	Name     string            `json:"name"`
	Enabled  bool              `json:"enabled"`
	Config   map[string]string `json:"config"`
	Priority int               `json:"priority,omitempty"` // Lower priority = notify first
}

// AlertRules contains all alert rule configurations
type AlertRules struct {
	Offline OfflineAlertRule `json:"offline"`
	Load    LoadAlertRule    `json:"load"`
	Traffic TrafficAlertRule `json:"traffic"`
	Expiry  ExpiryAlertRule  `json:"expiry"`
}

// ExpiryAlertRule configures expiry reminder alerts
type ExpiryAlertRule struct {
	Enabled     bool     `json:"enabled"`
	DaysBefore  []int    `json:"days_before"`   // Days before expiry to notify (e.g., [30, 14, 7, 3, 1])
	Channels    []string `json:"channels"`      // Channel IDs to notify
	Servers     []string `json:"servers"`       // Server IDs to monitor (empty = all)
	Exclude     []string `json:"exclude"`       // Server IDs to exclude
	ExcludeAuto bool     `json:"exclude_auto"`  // Exclude servers with auto_renew enabled
}

// OfflineAlertRule configures offline detection alerts
type OfflineAlertRule struct {
	Enabled     bool     `json:"enabled"`
	GracePeriod int      `json:"grace_period"` // Seconds before triggering alert
	Channels    []string `json:"channels"`     // Channel IDs to notify
	Servers     []string `json:"servers"`      // Server IDs to monitor (empty = all)
	Exclude     []string `json:"exclude"`      // Server IDs to exclude
}

// LoadAlertRule configures resource usage alerts
type LoadAlertRule struct {
	Enabled     bool             `json:"enabled"`
	CPU         *ThresholdConfig `json:"cpu,omitempty"`
	Memory      *ThresholdConfig `json:"memory,omitempty"`
	Disk        *ThresholdConfig `json:"disk,omitempty"`
	Channels    []string         `json:"channels"`
	Servers     []string         `json:"servers"` // Server IDs to monitor (empty = all)
	Exclude     []string         `json:"exclude"` // Server IDs to exclude
	Cooldown    int              `json:"cooldown,omitempty"` // Seconds between alerts for same server/metric
}

// ThresholdConfig configures threshold alerts
type ThresholdConfig struct {
	Warning   float32 `json:"warning,omitempty"`   // Warning threshold (%)
	Critical  float32 `json:"critical,omitempty"`  // Critical threshold (%)
	Duration  int     `json:"duration,omitempty"`  // Seconds above threshold before alert
}

// TrafficAlertRule configures traffic/bandwidth alerts
type TrafficAlertRule struct {
	Enabled     bool               `json:"enabled"`
	Limits      []TrafficLimit     `json:"limits"`
	Channels    []string           `json:"channels"`
	Cooldown    int                `json:"cooldown,omitempty"` // Hours between traffic alerts
}

// TrafficLimit configures a traffic limit for a server
type TrafficLimit struct {
	ServerID    string  `json:"server_id"`
	MonthlyGB   float64 `json:"monthly_gb"`           // Monthly traffic limit in GB
	Type        string  `json:"type"`                 // sum, max, up, down
	ResetDay    int     `json:"reset_day,omitempty"`  // Day of month to reset (1-28, 0 = 1st)
	Warning     float32 `json:"warning,omitempty"`    // Warning at X% of limit (default 80)
}

// AlertTemplate defines a notification template
type AlertTemplate struct {
	Title   string `json:"title"`
	Body    string `json:"body"`
	Format  string `json:"format,omitempty"` // text, html, markdown
}

// ============================================================================
// Alert State Types
// ============================================================================

// AlertState tracks the current state of an alert
type AlertState struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`        // offline, cpu, memory, disk, traffic
	ServerID    string     `json:"server_id"`
	ServerName  string     `json:"server_name"`
	Severity    string     `json:"severity"`    // warning, critical
	Status      string     `json:"status"`      // firing, resolved
	Value       float64    `json:"value"`       // Current value (e.g., CPU %, traffic GB)
	Threshold   float64    `json:"threshold"`   // Threshold that was crossed
	Message     string     `json:"message"`
	StartedAt   time.Time  `json:"started_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	NotifiedAt  *time.Time `json:"notified_at,omitempty"`
	Muted       bool       `json:"muted"`
}

// AlertHistory records historical alert data
type AlertHistory struct {
	ID          int64      `json:"id"`
	AlertID     string     `json:"alert_id"`
	Type        string     `json:"type"`
	ServerID    string     `json:"server_id"`
	ServerName  string     `json:"server_name"`
	Severity    string     `json:"severity"`
	Value       float64    `json:"value"`
	Threshold   float64    `json:"threshold"`
	Message     string     `json:"message"`
	StartedAt   time.Time  `json:"started_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	Duration    int64      `json:"duration"` // Seconds
	Notified    bool       `json:"notified"`
}

// NotificationEvent records a notification that was sent
type NotificationEvent struct {
	ID         int64     `json:"id"`
	AlertID    string    `json:"alert_id"`
	ChannelID  string    `json:"channel_id"`
	Type       string    `json:"type"`        // alert_type
	ServerID   string    `json:"server_id"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	Status     string    `json:"status"`      // sent, failed, pending
	Error      string    `json:"error,omitempty"`
	SentAt     time.Time `json:"sent_at"`
	RetryCount int       `json:"retry_count"`
}

// ============================================================================
// Alert API Request/Response Types
// ============================================================================

// UpdateAlertConfigRequest updates alert configuration
type UpdateAlertConfigRequest struct {
	Enabled        *bool                     `json:"enabled,omitempty"`
	Channels       *[]NotificationChannel    `json:"channels,omitempty"`
	Rules          *AlertRules               `json:"rules,omitempty"`
	Templates      *map[string]AlertTemplate `json:"templates,omitempty"`
	GlobalCooldown *int                      `json:"global_cooldown,omitempty"`
	RecoveryNotify *bool                     `json:"recovery_notify,omitempty"`
}

// AddChannelRequest adds a new notification channel
type AddChannelRequest struct {
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	Enabled  bool              `json:"enabled"`
	Config   map[string]string `json:"config"`
	Priority int               `json:"priority,omitempty"`
}

// TestChannelRequest tests a notification channel
type TestChannelRequest struct {
	ChannelID string `json:"channel_id,omitempty"`
	// Or inline channel config for testing before saving
	Type   string            `json:"type,omitempty"`
	Config map[string]string `json:"config,omitempty"`
}

// AlertsResponse returns current alert states
type AlertsResponse struct {
	Alerts []AlertState   `json:"alerts"`
	Stats  AlertStats     `json:"stats"`
}

// AlertStats provides summary statistics
type AlertStats struct {
	TotalFiring   int `json:"total_firing"`
	Critical      int `json:"critical"`
	Warning       int `json:"warning"`
	ServersOnline int `json:"servers_online"`
	ServersTotal  int `json:"servers_total"`
}

// ============================================================================
// Default Configuration
// ============================================================================

// GetDefaultAlertConfig returns a sensible default alert configuration
func GetDefaultAlertConfig() AlertConfig {
	return AlertConfig{
		Enabled:        false,
		Channels:       []NotificationChannel{},
		GlobalCooldown: 300, // 5 minutes
		RecoveryNotify: true,
		Templates: map[string]AlertTemplate{
			"offline": {
				Title:  "[{{ .Severity }}] {{ .ServerName }} 离线告警",
				Body:   "服务器 {{ .ServerName }} 已离线 {{ .Duration }}。\n上次在线时间: {{ .LastSeen }}",
				Format: "text",
			},
			"cpu": {
				Title:  "[{{ .Severity }}] {{ .ServerName }} CPU 告警",
				Body:   "服务器 {{ .ServerName }} CPU 使用率达到 {{ .Value }}%，超过阈值 {{ .Threshold }}%。",
				Format: "text",
			},
			"memory": {
				Title:  "[{{ .Severity }}] {{ .ServerName }} 内存告警",
				Body:   "服务器 {{ .ServerName }} 内存使用率达到 {{ .Value }}%，超过阈值 {{ .Threshold }}%。",
				Format: "text",
			},
			"disk": {
				Title:  "[{{ .Severity }}] {{ .ServerName }} 磁盘告警",
				Body:   "服务器 {{ .ServerName }} 磁盘使用率达到 {{ .Value }}%，超过阈值 {{ .Threshold }}%。",
				Format: "text",
			},
			"traffic": {
				Title:  "[{{ .Severity }}] {{ .ServerName }} 流量告警",
				Body:   "服务器 {{ .ServerName }} 本月流量已使用 {{ .Value }}GB，达到限额 {{ .Threshold }}GB 的 {{ .Percent }}%。",
				Format: "text",
			},
			"expiry": {
				Title:  "[提醒] {{ .ServerName }} 即将到期",
				Body:   "服务器 {{ .ServerName }} 将于 {{ .ExpiryDate }} 到期，剩余 {{ .DaysLeft }} 天。\n服务商: {{ .Provider }}\n价格: {{ .Price }}",
				Format: "text",
			},
			"recovery": {
				Title:  "[恢复] {{ .ServerName }} {{ .AlertType }} 告警已恢复",
				Body:   "服务器 {{ .ServerName }} 的 {{ .AlertType }} 告警已恢复正常。\n持续时间: {{ .Duration }}",
				Format: "text",
			},
		},
		Rules: AlertRules{
			Offline: OfflineAlertRule{
				Enabled:     false,
				GracePeriod: 60, // 1 minute
				Channels:    []string{},
				Servers:     []string{},
				Exclude:     []string{},
			},
			Load: LoadAlertRule{
				Enabled: false,
				CPU: &ThresholdConfig{
					Warning:  80,
					Critical: 95,
					Duration: 60, // 1 minute
				},
				Memory: &ThresholdConfig{
					Warning:  85,
					Critical: 95,
					Duration: 60,
				},
				Disk: &ThresholdConfig{
					Warning:  80,
					Critical: 90,
					Duration: 0, // Immediate for disk
				},
				Channels: []string{},
				Servers:  []string{},
				Exclude:  []string{},
				Cooldown: 300, // 5 minutes
			},
			Traffic: TrafficAlertRule{
				Enabled:  false,
				Limits:   []TrafficLimit{},
				Channels: []string{},
				Cooldown: 24, // 24 hours
			},
			Expiry: ExpiryAlertRule{
				Enabled:     false,
				DaysBefore:  []int{30, 14, 7, 3, 1}, // Notify 30, 14, 7, 3, 1 days before
				Channels:    []string{},
				Servers:     []string{},
				Exclude:     []string{},
				ExcludeAuto: true, // By default, don't notify for auto-renew servers
			},
		},
	}
}
