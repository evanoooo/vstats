package main

import (
	"database/sql"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"vstats/internal/common"
)

// Re-export common types for convenience
type SystemMetrics = common.SystemMetrics
type OsInfo = common.OsInfo
type CpuMetrics = common.CpuMetrics
type MemoryMetrics = common.MemoryMetrics
type MemoryModule = common.MemoryModule
type DiskMetrics = common.DiskMetrics
type NetworkMetrics = common.NetworkMetrics
type NetworkInterface = common.NetworkInterface
type LoadAverage = common.LoadAverage
type PingMetrics = common.PingMetrics
type PingTarget = common.PingTarget

// ============================================================================
// Auth Types
// ============================================================================

type Claims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
}

type LoginRequest struct {
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ============================================================================
// OAuth Types
// ============================================================================

type OAuthStateData struct {
	Provider  string `json:"provider"`
	State     string `json:"state"`
	CreatedAt int64  `json:"created_at"`
}

type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type OAuthLoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Provider  string    `json:"provider"`
	Username  string    `json:"username"`
}

// OIDC Types
type OIDCDiscovery struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserinfoEndpoint      string   `json:"userinfo_endpoint"`
	JwksURI               string   `json:"jwks_uri"`
	ScopesSupported       []string `json:"scopes_supported"`
	ClaimsSupported       []string `json:"claims_supported"`
}

type OIDCTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope,omitempty"`
}

type OIDCUserInfo struct {
	Sub               string   `json:"sub"`
	Email             string   `json:"email,omitempty"`
	EmailVerified     bool     `json:"email_verified,omitempty"`
	Name              string   `json:"name,omitempty"`
	PreferredUsername string   `json:"preferred_username,omitempty"`
	Picture           string   `json:"picture,omitempty"`
	Groups            []string `json:"groups,omitempty"`
}

// CloudflareAccessClaims represents the JWT claims from Cloudflare Access
type CloudflareAccessClaims struct {
	Email    string   `json:"email"`
	Type     string   `json:"type"`
	Identity struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		Groups []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"groups,omitempty"`
	} `json:"identity,omitempty"`
}

// ============================================================================
// Server Management Types
// ============================================================================

type AddServerRequest struct {
	Name          string            `json:"name"`
	URL           string            `json:"url"`
	Location      string            `json:"location"`
	Provider      string            `json:"provider"`
	Tag           string            `json:"tag"`
	GroupID       string            `json:"group_id,omitempty"`     // Deprecated
	GroupValues   map[string]string `json:"group_values,omitempty"` // dimension_id -> option_id
	PriceAmount   string            `json:"price_amount,omitempty"`
	PricePeriod   string            `json:"price_period,omitempty"`
	PriceCurrency string            `json:"price_currency,omitempty"`
	PurchaseDate  string            `json:"purchase_date,omitempty"`
	ExpiryDate    string            `json:"expiry_date,omitempty"`
	AutoRenew     bool              `json:"auto_renew,omitempty"`
	TipBadge      string            `json:"tip_badge,omitempty"`
	Notes         string            `json:"notes,omitempty"`
}

type UpdateServerRequest struct {
	Name          *string            `json:"name,omitempty"`
	Location      *string            `json:"location,omitempty"`
	Provider      *string            `json:"provider,omitempty"`
	Tag           *string            `json:"tag,omitempty"`
	GroupID       *string            `json:"group_id,omitempty"`     // Deprecated
	GroupValues   *map[string]string `json:"group_values,omitempty"` // dimension_id -> option_id
	PriceAmount   *string            `json:"price_amount,omitempty"`
	PricePeriod   *string            `json:"price_period,omitempty"`
	PriceCurrency *string            `json:"price_currency,omitempty"`
	PurchaseDate  *string            `json:"purchase_date,omitempty"`
	ExpiryDate    *string            `json:"expiry_date,omitempty"`
	AutoRenew     *bool              `json:"auto_renew,omitempty"`
	TipBadge      *string            `json:"tip_badge,omitempty"`
	Notes         *string            `json:"notes,omitempty"`
}

// ============================================================================
// Group Management Types (Deprecated - for backward compatibility)
// ============================================================================

type AddGroupRequest struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type UpdateGroupRequest struct {
	Name      *string `json:"name,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

// ============================================================================
// Dimension Management Types
// ============================================================================

type AddDimensionRequest struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	Enabled   bool   `json:"enabled"`
	SortOrder int    `json:"sort_order"`
}

type UpdateDimensionRequest struct {
	Name      *string `json:"name,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

type AddOptionRequest struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type UpdateOptionRequest struct {
	Name      *string `json:"name,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

// Re-export common registration types
type AgentRegisterRequest = common.RegisterRequest
type AgentRegisterResponse = common.RegisterResponse

// ============================================================================
// History Types
// ============================================================================

type HistoryPoint struct {
	Timestamp string   `json:"timestamp"`
	CPU       float32  `json:"cpu"`
	Memory    float32  `json:"memory"`
	Disk      float32  `json:"disk"`
	NetRx     int64    `json:"net_rx"`
	NetTx     int64    `json:"net_tx"`
	PingMs    *float64 `json:"ping_ms,omitempty"`
}

type HistoryResponse struct {
	ServerID    string              `json:"server_id"`
	Range       string              `json:"range"`
	Data        []HistoryPoint      `json:"data"`
	PingTargets []PingHistoryTarget `json:"ping_targets,omitempty"`
	LastBucket  int64               `json:"last_bucket,omitempty"`  // For incremental updates
	Incremental bool                `json:"incremental,omitempty"` // True if this is an incremental response
}

type PingHistoryTarget struct {
	Name string             `json:"name"`
	Host string             `json:"host"`
	Data []PingHistoryPoint `json:"data"`
}

type PingHistoryPoint struct {
	Timestamp string   `json:"timestamp"`
	LatencyMs *float64 `json:"latency_ms"`
	Status    string   `json:"status"`
}

// ============================================================================
// WebSocket Message Types
// ============================================================================

type AgentMetricsData struct {
	ServerID    string
	Metrics     SystemMetrics
	LastUpdated time.Time
}

type DashboardMessage struct {
	Type            string                `json:"type"`
	Servers         []ServerMetricsUpdate `json:"servers"`
	Groups          []ServerGroup         `json:"groups,omitempty"` // Deprecated
	GroupDimensions []GroupDimension      `json:"group_dimensions,omitempty"`
	SiteSettings    *SiteSettings         `json:"site_settings,omitempty"`
}

type ServerMetricsUpdate struct {
	ServerID      string            `json:"server_id"`
	ServerName    string            `json:"server_name"`
	Location      string            `json:"location"`
	Provider      string            `json:"provider"`
	Tag           string            `json:"tag"`
	GroupID       string            `json:"group_id,omitempty"`     // Deprecated
	GroupValues   map[string]string `json:"group_values,omitempty"` // dimension_id -> option_id
	Version       string            `json:"version"`
	IP            string            `json:"ip"`
	Online        bool              `json:"online"`
	Metrics       *SystemMetrics    `json:"metrics"`
	PriceAmount   string            `json:"price_amount,omitempty"`
	PricePeriod   string            `json:"price_period,omitempty"`
	PriceCurrency string            `json:"price_currency,omitempty"`
	PurchaseDate  string            `json:"purchase_date,omitempty"`
	ExpiryDate    string            `json:"expiry_date,omitempty"`
	AutoRenew     bool              `json:"auto_renew,omitempty"`
	TipBadge      string            `json:"tip_badge,omitempty"`
	Notes         string            `json:"notes,omitempty"`
	GeoIP         *ServerGeoIP      `json:"geoip,omitempty"`
}

type DeltaMessage struct {
	Type string                `json:"type"`
	Ts   int64                 `json:"ts"`
	D    []CompactServerUpdate `json:"d,omitempty"`
}

type CompactServerUpdate struct {
	ID string          `json:"id"`
	On *bool           `json:"on,omitempty"`
	M  *CompactMetrics `json:"m,omitempty"`
}

type CompactMetrics struct {
	C  *uint8  `json:"c,omitempty"`
	M  *uint8  `json:"m,omitempty"`
	D  *uint8  `json:"d,omitempty"`
	Rx *uint64 `json:"rx,omitempty"`
	Tx *uint64 `json:"tx,omitempty"`
	Up *uint64 `json:"up,omitempty"`
}

func (cm *CompactMetrics) IsEmpty() bool {
	return cm.C == nil && cm.M == nil && cm.D == nil && cm.Rx == nil && cm.Tx == nil && cm.Up == nil
}

func (cm *CompactMetrics) HasChanged(other *CompactMetrics) bool {
	return cm.C != other.C || cm.M != other.M || cm.D != other.D || cm.Rx != other.Rx || cm.Tx != other.Tx
}

func (cm *CompactMetrics) Diff(prev *CompactMetrics) *CompactMetrics {
	diff := &CompactMetrics{}
	if cm.C != nil && (prev.C == nil || *cm.C != *prev.C) {
		diff.C = cm.C
	}
	if cm.M != nil && (prev.M == nil || *cm.M != *prev.M) {
		diff.M = cm.M
	}
	if cm.D != nil && (prev.D == nil || *cm.D != *prev.D) {
		diff.D = cm.D
	}
	if cm.Rx != nil && (prev.Rx == nil || *cm.Rx != *prev.Rx) {
		diff.Rx = cm.Rx
	}
	if cm.Tx != nil && (prev.Tx == nil || *cm.Tx != *prev.Tx) {
		diff.Tx = cm.Tx
	}
	return diff
}

func CompactMetricsFromSystem(m *SystemMetrics) *CompactMetrics {
	cpu := uint8(m.CPU.Usage)
	mem := uint8(m.Memory.UsagePercent)
	var disk *uint8
	if len(m.Disks) > 0 {
		d := uint8(m.Disks[0].UsagePercent)
		disk = &d
	}
	rx := m.Network.RxSpeed
	tx := m.Network.TxSpeed
	up := m.Uptime
	return &CompactMetrics{
		C:  &cpu,
		M:  &mem,
		D:  disk,
		Rx: &rx,
		Tx: &tx,
		Up: &up,
	}
}

type AgentMessage struct {
	Type     string         `json:"type"`
	ServerID string         `json:"server_id,omitempty"`
	Token    string         `json:"token,omitempty"`
	Version  string         `json:"version,omitempty"`
	Metrics  *SystemMetrics `json:"metrics,omitempty"`
	// Batch metrics fields
	BatchID    string                       `json:"batch_id,omitempty"`
	BatchItems []common.TimestampedMetrics  `json:"metrics_batch,omitempty"` // For batch raw metrics
	Aggregated []*common.AggregatedMetrics  `json:"aggregated,omitempty"`    // For aggregated metrics
	// Multi-granularity aggregated metrics (new)
	Granularities []common.GranularityData `json:"granularities,omitempty"` // For multi-granularity data
	LastMetrics   *SystemMetrics           `json:"last_metrics,omitempty"`  // Latest metrics snapshot
}

type AgentCommand struct {
	Type        string `json:"type"`
	Command     string `json:"command"`
	DownloadURL string `json:"download_url,omitempty"`
	Force       bool   `json:"force,omitempty"`
}

type UpdateAgentRequest struct {
	DownloadURL string `json:"download_url,omitempty"`
	Force       bool   `json:"force,omitempty"`
}

type UpdateAgentResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ============================================================================
// Audit Log Types
// ============================================================================

// AuditLogCategory represents the category of audit action
type AuditLogCategory string

const (
	AuditCategoryAuth     AuditLogCategory = "auth"
	AuditCategoryServer   AuditLogCategory = "server"
	AuditCategorySettings AuditLogCategory = "settings"
	AuditCategoryAlert    AuditLogCategory = "alert"
	AuditCategorySystem   AuditLogCategory = "system"
)

// AuditLogAction represents the specific action
type AuditLogAction string

const (
	// Auth actions
	AuditActionLogin              AuditLogAction = "login"
	AuditActionLoginFailed        AuditLogAction = "login_failed"
	AuditActionLogout             AuditLogAction = "logout"
	AuditActionPasswordChange     AuditLogAction = "password_change"
	AuditActionOAuthLogin         AuditLogAction = "oauth_login"
	AuditActionOAuthLoginFailed   AuditLogAction = "oauth_login_failed"

	// Server actions
	AuditActionServerCreate       AuditLogAction = "server_create"
	AuditActionServerUpdate       AuditLogAction = "server_update"
	AuditActionServerDelete       AuditLogAction = "server_delete"
	AuditActionServerUpgrade      AuditLogAction = "server_upgrade"
	AuditActionAgentRegister      AuditLogAction = "agent_register"
	AuditActionAgentConnect       AuditLogAction = "agent_connect"
	AuditActionAgentDisconnect    AuditLogAction = "agent_disconnect"

	// Settings actions
	AuditActionSettingsUpdate     AuditLogAction = "settings_update"
	AuditActionSiteSettingsUpdate AuditLogAction = "site_settings_update"
	AuditActionProbeSettingsUpdate AuditLogAction = "probe_settings_update"
	AuditActionOAuthSettingsUpdate AuditLogAction = "oauth_settings_update"
	AuditActionLocalNodeUpdate    AuditLogAction = "local_node_update"

	// Alert actions
	AuditActionAlertConfigUpdate  AuditLogAction = "alert_config_update"
	AuditActionChannelCreate      AuditLogAction = "channel_create"
	AuditActionChannelUpdate      AuditLogAction = "channel_update"
	AuditActionChannelDelete      AuditLogAction = "channel_delete"
	AuditActionChannelTest        AuditLogAction = "channel_test"
	AuditActionAlertMute          AuditLogAction = "alert_mute"
	AuditActionRuleUpdate         AuditLogAction = "rule_update"
	AuditActionTemplateUpdate     AuditLogAction = "template_update"

	// Group/Dimension actions
	AuditActionGroupCreate        AuditLogAction = "group_create"
	AuditActionGroupUpdate        AuditLogAction = "group_update"
	AuditActionGroupDelete        AuditLogAction = "group_delete"
	AuditActionDimensionCreate    AuditLogAction = "dimension_create"
	AuditActionDimensionUpdate    AuditLogAction = "dimension_update"
	AuditActionDimensionDelete    AuditLogAction = "dimension_delete"
	AuditActionOptionCreate       AuditLogAction = "option_create"
	AuditActionOptionUpdate       AuditLogAction = "option_update"
	AuditActionOptionDelete       AuditLogAction = "option_delete"
)

// AuditLog represents a single audit log entry
type AuditLog struct {
	ID           int64            `json:"id"`
	Timestamp    string           `json:"timestamp"`
	Action       AuditLogAction   `json:"action"`
	Category     AuditLogCategory `json:"category"`
	UserIP       string           `json:"user_ip"`
	UserAgent    string           `json:"user_agent,omitempty"`
	TargetType   string           `json:"target_type,omitempty"`
	TargetID     string           `json:"target_id,omitempty"`
	TargetName   string           `json:"target_name,omitempty"`
	Details      string           `json:"details,omitempty"`
	Status       string           `json:"status"`
	ErrorMessage string           `json:"error_message,omitempty"`
}

// AuditLogEntry is used for creating new audit log entries
type AuditLogEntry struct {
	Action       AuditLogAction
	Category     AuditLogCategory
	UserIP       string
	UserAgent    string
	TargetType   string
	TargetID     string
	TargetName   string
	Details      string
	Status       string
	ErrorMessage string
}

// AuditLogQuery represents query parameters for fetching audit logs
type AuditLogQuery struct {
	Page       int              `form:"page"`
	Limit      int              `form:"limit"`
	Category   AuditLogCategory `form:"category"`
	Action     AuditLogAction   `form:"action"`
	StartDate  string           `form:"start_date"`
	EndDate    string           `form:"end_date"`
	Search     string           `form:"search"`
}

// AuditLogResponse represents the response for audit log queries
type AuditLogResponse struct {
	Logs  []AuditLog `json:"logs"`
	Total int64      `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
}

// AuditLogSettings represents the audit log retention settings
type AuditLogSettings struct {
	RetentionDays int  `json:"retention_days"`
	Enabled       bool `json:"enabled"`
}

type InstallCommand struct {
	Command   string `json:"command"`
	ScriptURL string `json:"script_url"`
}

type VersionInfo struct {
	Current         string  `json:"current"`
	Latest          *string `json:"latest,omitempty"`
	UpdateAvailable bool    `json:"update_available"`
}

// ============================================================================
// Dashboard Snapshot (pre-serialized for fast delivery)
// ============================================================================

// DashboardSnapshot holds pre-built data for new dashboard connections
type DashboardSnapshot struct {
	InitMessage    []byte            // Pre-serialized StreamInitMessage
	ServerMessages [][]byte          // Pre-serialized StreamServerMessage for each server
	EndMessage     []byte            // Pre-serialized StreamEndMessage
	LastUpdated    time.Time         // When the snapshot was last updated
	ServerHashes   map[string]uint64 // Hash of each server's state to detect changes
}

// ============================================================================
// App State
// ============================================================================

type LastSentState struct {
	Servers map[string]*struct {
		Online  bool
		Metrics *CompactMetrics
	}
}

type AgentConnection struct {
	Conn     *websocket.Conn
	SendChan chan []byte
}

// DashboardClient represents a connected dashboard client with its IP
type DashboardClient struct {
	Conn    *websocket.Conn
	IP      string
	WriteMu sync.Mutex // Protects concurrent writes to the connection
}

type AppState struct {
	Config           *AppConfig
	ConfigMu         sync.RWMutex
	MetricsBroadcast chan string
	AgentMetrics     map[string]*AgentMetricsData
	AgentMetricsMu   sync.RWMutex
	AgentConns       map[string]*AgentConnection
	AgentConnsMu     sync.RWMutex
	LastSent         *LastSentState
	LastSentMu       sync.RWMutex
	DashboardClients map[*websocket.Conn]*DashboardClient
	DashboardMu      sync.RWMutex
	DB               *sql.DB
	// Pre-built snapshot for fast dashboard delivery
	Snapshot         *DashboardSnapshot
	SnapshotMu       sync.RWMutex
}

// GetOnlineUsersCount returns the number of unique IPs connected to the dashboard
func (s *AppState) GetOnlineUsersCount() int {
	s.DashboardMu.RLock()
	defer s.DashboardMu.RUnlock()

	uniqueIPs := make(map[string]bool)
	for _, client := range s.DashboardClients {
		if client != nil && client.IP != "" {
			uniqueIPs[client.IP] = true
		}
	}
	return len(uniqueIPs)
}
