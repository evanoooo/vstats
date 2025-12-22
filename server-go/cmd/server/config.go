package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"vstats/internal/common"
)

const (
	ConfigFilename = "vstats-config.json"
	DBFilename     = "vstats.db"
)

var (
	jwtSecret   string
	jwtSecretMu sync.RWMutex
)

type LocalNodeConfig struct {
	Name         string            `json:"name"`
	Location     string            `json:"location"`
	Provider     string            `json:"provider"`
	Tag          string            `json:"tag"`
	GroupID      string            `json:"group_id,omitempty"`     // Deprecated, for backward compatibility
	GroupValues  map[string]string `json:"group_values,omitempty"` // dimension_id -> option_id
	PriceAmount  string            `json:"price_amount,omitempty"`
	PricePeriod  string            `json:"price_period,omitempty"`
	PriceCurrency string           `json:"price_currency,omitempty"` // Currency code: USD, CNY, EUR, etc.
	PurchaseDate string            `json:"purchase_date,omitempty"`
	ExpiryDate   string            `json:"expiry_date,omitempty"`    // When the service expires
	AutoRenew    bool              `json:"auto_renew,omitempty"`     // Whether auto-renewal is enabled
	TipBadge     string            `json:"tip_badge,omitempty"`
	Notes        string            `json:"notes,omitempty"`          // Additional notes
}

// BackgroundConfig represents background settings for the site theme
type BackgroundConfig struct {
	Type          string `json:"type"` // gradient, bing, unsplash, custom, solid
	CustomUrl     string `json:"custom_url,omitempty"`
	UnsplashQuery string `json:"unsplash_query,omitempty"`
	SolidColor    string `json:"solid_color,omitempty"`
	Blur          int    `json:"blur,omitempty"`
	Opacity       int    `json:"opacity,omitempty"`
}

// ThemeSettings represents site-wide theme configuration
type ThemeSettings struct {
	ThemeId    string            `json:"theme_id"`
	Background *BackgroundConfig `json:"background,omitempty"`
}

type SiteSettings struct {
	SiteName        string         `json:"site_name"`
	SiteDescription string         `json:"site_description"`
	SocialLinks     []SocialLink   `json:"social_links"`
	Theme           *ThemeSettings `json:"theme,omitempty"`
}

type SocialLink struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
	Label    string `json:"label"`
}

type ProbeSettings struct {
	PingTargets []common.PingTargetConfig `json:"ping_targets"`
}

// OAuth 2.0 Configuration
type OAuthProvider struct {
	Enabled      bool     `json:"enabled"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AllowedUsers []string `json:"allowed_users,omitempty"` // GitHub usernames or Google emails
}

// OIDCProvider represents a generic OpenID Connect provider configuration
type OIDCProvider struct {
	Enabled       bool     `json:"enabled"`
	Name          string   `json:"name"`                     // Display name (e.g., "Authentik", "Keycloak")
	Issuer        string   `json:"issuer"`                   // OIDC issuer URL (e.g., https://auth.example.com)
	ClientID      string   `json:"client_id"`
	ClientSecret  string   `json:"client_secret"`
	Scopes        []string `json:"scopes,omitempty"`         // Default: openid, email, profile
	AllowedUsers  []string `json:"allowed_users,omitempty"`  // Email addresses or subject IDs
	AllowedGroups []string `json:"allowed_groups,omitempty"` // Group names from OIDC claims
	UsernameClaim string   `json:"username_claim,omitempty"` // Claim to use as username (default: email)
}

// CloudflareAccessConfig represents Cloudflare Access (Zero Trust) configuration
type CloudflareAccessConfig struct {
	Enabled      bool     `json:"enabled"`
	TeamDomain   string   `json:"team_domain"`              // Your Cloudflare Access team domain (e.g., mycompany.cloudflareaccess.com)
	AUD          string   `json:"aud"`                      // Application Audience (AUD) tag from Cloudflare Access
	AllowedUsers []string `json:"allowed_users,omitempty"`  // Email addresses allowed to access
}

// SSOBinding represents a linked SSO identity to the admin account
type SSOBinding struct {
	Provider   string `json:"provider"`    // e.g., "github", "google", "oidc", "cloudflare"
	ProviderID string `json:"provider_id"` // Provider's unique identifier for OIDC
	Identifier string `json:"identifier"`  // Username/email from the provider
	BoundAt    string `json:"bound_at"`    // ISO timestamp when bound
}

type OAuthConfig struct {
	// Use centralized OAuth proxy (vstats.zsoft.cc)
	// When enabled, no need to configure individual OAuth apps
	UseCentralized bool `json:"use_centralized"`

	// Allowed users for centralized OAuth (GitHub usernames or Google emails)
	AllowedUsers []string `json:"allowed_users,omitempty"`

	// Self-hosted OAuth configuration (optional, for advanced users)
	GitHub *OAuthProvider `json:"github,omitempty"`
	Google *OAuthProvider `json:"google,omitempty"`

	// Generic OIDC Providers (support multiple OIDC providers)
	OIDC []*OIDCProvider `json:"oidc,omitempty"`

	// Cloudflare Access (Zero Trust) configuration
	CloudflareAccess *CloudflareAccessConfig `json:"cloudflare_access,omitempty"`

	// SSO Bindings - linked SSO identities to admin account
	Bindings []SSOBinding `json:"bindings,omitempty"`
}

// GroupDimension represents a grouping dimension (e.g., Region, Purpose)
type GroupDimension struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Key       string        `json:"key"`     // Unique key for the dimension
	Enabled   bool          `json:"enabled"` // Whether this dimension is enabled for grouping
	SortOrder int           `json:"sort_order"`
	Options   []GroupOption `json:"options"`
}

// GroupOption represents an option within a dimension
type GroupOption struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// ServerGroup - deprecated, kept for backward compatibility
type ServerGroup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// ServerGeoIP holds GeoIP data for a server
type ServerGeoIP struct {
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	City        string  `json:"city,omitempty"`
	Region      string  `json:"region,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	UpdatedAt   string  `json:"updated_at,omitempty"`
}

type RemoteServer struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Location     string            `json:"location"`
	Provider     string            `json:"provider"`
	Tag          string            `json:"tag"`
	Token        string            `json:"token"`
	Version      string            `json:"version"`
	IP           string            `json:"ip"`
	GroupID      string            `json:"group_id,omitempty"`     // Deprecated, for backward compatibility
	GroupValues  map[string]string `json:"group_values,omitempty"` // dimension_id -> option_id
	PriceAmount  string            `json:"price_amount,omitempty"`
	PricePeriod  string            `json:"price_period,omitempty"`
	PriceCurrency string           `json:"price_currency,omitempty"` // Currency code: USD, CNY, EUR, etc.
	PurchaseDate string            `json:"purchase_date,omitempty"`
	ExpiryDate   string            `json:"expiry_date,omitempty"`    // When the service expires
	AutoRenew    bool              `json:"auto_renew,omitempty"`     // Whether auto-renewal is enabled
	TipBadge     string            `json:"tip_badge,omitempty"`
	Notes        string            `json:"notes,omitempty"`          // Additional notes
	GeoIP        *ServerGeoIP      `json:"geoip,omitempty"`
}

// TLSConfig represents TLS/SSL configuration
type TLSConfig struct {
	Enabled bool   `json:"enabled"`
	Cert    string `json:"cert,omitempty"` // Path to certificate file
	Key     string `json:"key,omitempty"`  // Path to private key file
}

type AppConfig struct {
	AdminPasswordHash string            `json:"admin_password_hash"`
	JWTSecret         string            `json:"jwt_secret"`
	Port              string            `json:"port,omitempty"`
	Host              string            `json:"host,omitempty"` // Listen address (0.0.0.0, [::], or specific IP)
	DualStack         bool             `json:"dual_stack,omitempty"` // Enable dual-stack (IPv4 + IPv6) support
	TLS               *TLSConfig        `json:"tls,omitempty"`   // TLS/SSL configuration
	Servers           []RemoteServer    `json:"servers"`
	Groups            []ServerGroup     `json:"groups,omitempty"` // Deprecated, for backward compatibility
	GroupDimensions   []GroupDimension  `json:"group_dimensions,omitempty"`
	SiteSettings      SiteSettings      `json:"site_settings"`
	LocalNode         LocalNodeConfig   `json:"local_node"`
	ProbeSettings     ProbeSettings     `json:"probe_settings"`
	OAuth             *OAuthConfig      `json:"oauth,omitempty"`
	AlertConfig       *AlertConfig      `json:"alert_config,omitempty"`
	AuditLogSettings  *AuditLogSettings `json:"audit_log_settings,omitempty"`
	GeoIPConfig       *GeoIPConfig      `json:"geoip_config,omitempty"`
	InstalledThemes   []InstalledTheme  `json:"installed_themes,omitempty"` // External themes installed from GitHub
}

func getExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func GetConfigPath() string {
	// Allow override via environment variable
	if configPath := os.Getenv("VSTATS_CONFIG_PATH"); configPath != "" {
		return configPath
	}
	return filepath.Join(getExeDir(), ConfigFilename)
}

func GetDBPath() string {
	// Allow override via environment variable
	if dbPath := os.Getenv("VSTATS_DB_PATH"); dbPath != "" {
		return dbPath
	}
	return filepath.Join(getExeDir(), DBFilename)
}

func GetJWTSecret() string {
	jwtSecretMu.RLock()
	defer jwtSecretMu.RUnlock()
	if jwtSecret == "" {
		return "fallback-secret"
	}
	return jwtSecret
}

func InitJWTSecret(secret string) {
	jwtSecretMu.Lock()
	defer jwtSecretMu.Unlock()
	jwtSecret = secret
}

func GenerateRandomString(length int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

// GetDefaultGroupDimensions returns the default group dimensions
func GetDefaultGroupDimensions() []GroupDimension {
	return []GroupDimension{
		{
			ID:        "region",
			Name:      "地区",
			Key:       "region",
			Enabled:   true,
			SortOrder: 0,
			Options: []GroupOption{
				{ID: "asia", Name: "亚洲", SortOrder: 0},
				{ID: "america", Name: "美洲", SortOrder: 1},
				{ID: "europe", Name: "欧洲", SortOrder: 2},
				{ID: "other", Name: "其他", SortOrder: 3},
			},
		},
		{
			ID:        "purpose",
			Name:      "用途",
			Key:       "purpose",
			Enabled:   true,
			SortOrder: 1,
			Options: []GroupOption{
				{ID: "production", Name: "生产", SortOrder: 0},
				{ID: "staging", Name: "预发", SortOrder: 1},
				{ID: "development", Name: "开发", SortOrder: 2},
				{ID: "testing", Name: "测试", SortOrder: 3},
			},
		},
		{
			ID:        "group",
			Name:      "分组",
			Key:       "group",
			Enabled:   false,
			SortOrder: 2,
			Options:   []GroupOption{},
		},
	}
}

func NewAppConfigWithRandomPassword() (*AppConfig, string) {
	password := GenerateRandomString(16)
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	config := &AppConfig{
		AdminPasswordHash: string(hash),
		JWTSecret:         GenerateRandomString(64),
		Servers:           []RemoteServer{},
		Groups:            []ServerGroup{},
		GroupDimensions:   GetDefaultGroupDimensions(),
		SiteSettings: SiteSettings{
			SiteName:        "vStats Dashboard",
			SiteDescription: "Real-time Server Monitoring",
			SocialLinks:     []SocialLink{},
		},
		LocalNode:     LocalNodeConfig{},
		ProbeSettings: ProbeSettings{PingTargets: []common.PingTargetConfig{}},
	}
	return config, password
}

func (c *AppConfig) ResetPassword() string {
	password := GenerateRandomString(16)
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	c.AdminPasswordHash = string(hash)
	return password
}

func LoadConfig() (*AppConfig, *string) {
	path := GetConfigPath()
	fmt.Printf("📂 Loading config from: %s\n", path)

	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("⚠️  Failed to read config: %v, using defaults\n", err)
			config, password := NewAppConfigWithRandomPassword()
			saveConfigNow(config) // Immediate save for initialization
			InitJWTSecret(config.JWTSecret)
			return config, &password
		}

		var config AppConfig
		if err := json.Unmarshal(data, &config); err != nil {
			fmt.Printf("⚠️  Failed to parse config: %v, using defaults\n", err)
			newConfig, password := NewAppConfigWithRandomPassword()
			saveConfigNow(newConfig) // Immediate save for initialization
			InitJWTSecret(newConfig.JWTSecret)
			return newConfig, &password
		}

		// Verify password hash looks valid
		if len(config.AdminPasswordHash) < 4 || config.AdminPasswordHash[:3] != "$2a" && config.AdminPasswordHash[:3] != "$2b" {
			fmt.Println("⚠️  Invalid password hash format, regenerating...")
			password := GenerateRandomString(16)
			hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			config.AdminPasswordHash = string(hash)
			saveConfigNow(&config) // Immediate save for password
			fmt.Printf("🔑 New password: %s\n", password)
		} else {
			fmt.Printf("✅ Password hash loaded (%d chars)\n", len(config.AdminPasswordHash))
		}

		// Ensure jwt_secret exists
		if config.JWTSecret == "" {
			config.JWTSecret = GenerateRandomString(64)
			saveConfigNow(&config) // Immediate save for JWT secret
		}

		// Initialize default group dimensions if not present
		if len(config.GroupDimensions) == 0 {
			config.GroupDimensions = GetDefaultGroupDimensions()
			saveConfigNow(&config) // Immediate save for defaults
			fmt.Println("✅ Initialized default group dimensions")
		}

		InitJWTSecret(config.JWTSecret)
		return &config, nil
	}

	// First run - generate random password
	config, password := NewAppConfigWithRandomPassword()
	saveConfigNow(config) // Immediate save for initialization
	InitJWTSecret(config.JWTSecret)
	return config, &password
}

func ResetAdminPassword() string {
	path := GetConfigPath()
	var config *AppConfig

	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err == nil {
			var c AppConfig
			if json.Unmarshal(data, &c) == nil {
				config = &c
			}
		}
	}

	if config == nil {
		config, _ = NewAppConfigWithRandomPassword()
	}

	password := config.ResetPassword()

	// Ensure JWT secret exists before saving
	if config.JWTSecret == "" {
		config.JWTSecret = GenerateRandomString(64)
	}

	saveConfigNow(config) // Immediate save for password reset

	// Re-initialize JWT secret in case server is running
	InitJWTSecret(config.JWTSecret)

	return password
}

// Config save debouncing - prevents excessive disk I/O
var (
	configDirty     bool
	configDirtyMu   sync.Mutex
	configSaveTimer *time.Timer
	pendingConfig   *AppConfig
)

const configSaveDelay = 5 * time.Second // Batch saves within 5 seconds

// SaveConfig marks config as dirty and schedules a debounced save
func SaveConfig(config *AppConfig) {
	configDirtyMu.Lock()
	defer configDirtyMu.Unlock()

	pendingConfig = config
	configDirty = true

	// If timer already running, it will save the latest config
	if configSaveTimer != nil {
		return
	}

	// Schedule save after delay
	configSaveTimer = time.AfterFunc(configSaveDelay, func() {
		configDirtyMu.Lock()
		if !configDirty || pendingConfig == nil {
			configDirtyMu.Unlock()
			return
		}
		cfg := pendingConfig
		configDirty = false
		configSaveTimer = nil
		configDirtyMu.Unlock()

		saveConfigNow(cfg)
	})
}

// SaveConfigImmediate saves config immediately (for critical operations like password reset)
func SaveConfigImmediate(config *AppConfig) {
	configDirtyMu.Lock()
	if configSaveTimer != nil {
		configSaveTimer.Stop()
		configSaveTimer = nil
	}
	configDirty = false
	pendingConfig = nil
	configDirtyMu.Unlock()

	saveConfigNow(config)
}

// saveConfigNow performs the actual file write
func saveConfigNow(config *AppConfig) {
	path := GetConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf("Failed to serialize config: %v\n", err)
		return
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		fmt.Printf("Failed to write config: %v\n", err)
	}
}
