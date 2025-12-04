package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/bcrypt"
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
	GroupID      string            `json:"group_id,omitempty"`      // Deprecated, for backward compatibility
	GroupValues  map[string]string `json:"group_values,omitempty"` // dimension_id -> option_id
	PriceAmount  string            `json:"price_amount,omitempty"`
	PricePeriod  string            `json:"price_period,omitempty"`
	PurchaseDate string            `json:"purchase_date,omitempty"`
	TipBadge     string            `json:"tip_badge,omitempty"`
}

type SiteSettings struct {
	SiteName        string       `json:"site_name"`
	SiteDescription string       `json:"site_description"`
	SocialLinks     []SocialLink `json:"social_links"`
}

type SocialLink struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
	Label    string `json:"label"`
}

type PingTargetConfig struct {
	Name string `json:"name"`
	Host string `json:"host"`
}

type ProbeSettings struct {
	PingTargets []PingTargetConfig `json:"ping_targets"`
}

// OAuth 2.0 Configuration
type OAuthProvider struct {
	Enabled      bool     `json:"enabled"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AllowedUsers []string `json:"allowed_users,omitempty"` // GitHub usernames or Google emails
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
}

// GroupDimension represents a grouping dimension (e.g., Region, Purpose)
type GroupDimension struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Key       string        `json:"key"`        // Unique key for the dimension
	Enabled   bool          `json:"enabled"`    // Whether this dimension is enabled for grouping
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
	GroupID      string            `json:"group_id,omitempty"`      // Deprecated, for backward compatibility
	GroupValues  map[string]string `json:"group_values,omitempty"` // dimension_id -> option_id
	PriceAmount  string            `json:"price_amount,omitempty"`
	PricePeriod  string            `json:"price_period,omitempty"`
	PurchaseDate string            `json:"purchase_date,omitempty"`
	TipBadge     string            `json:"tip_badge,omitempty"`
}

type AppConfig struct {
	AdminPasswordHash string           `json:"admin_password_hash"`
	JWTSecret         string           `json:"jwt_secret"`
	Port              string           `json:"port,omitempty"`
	Servers           []RemoteServer   `json:"servers"`
	Groups            []ServerGroup    `json:"groups,omitempty"`     // Deprecated, for backward compatibility
	GroupDimensions   []GroupDimension `json:"group_dimensions,omitempty"`
	SiteSettings      SiteSettings     `json:"site_settings"`
	LocalNode         LocalNodeConfig  `json:"local_node"`
	ProbeSettings     ProbeSettings    `json:"probe_settings"`
	OAuth             *OAuthConfig     `json:"oauth,omitempty"`
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
		ProbeSettings: ProbeSettings{PingTargets: []PingTargetConfig{}},
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
			SaveConfig(config)
			InitJWTSecret(config.JWTSecret)
			return config, &password
		}

		var config AppConfig
		if err := json.Unmarshal(data, &config); err != nil {
			fmt.Printf("⚠️  Failed to parse config: %v, using defaults\n", err)
			newConfig, password := NewAppConfigWithRandomPassword()
			SaveConfig(newConfig)
			InitJWTSecret(newConfig.JWTSecret)
			return newConfig, &password
		}

		// Verify password hash looks valid
		if len(config.AdminPasswordHash) < 4 || config.AdminPasswordHash[:3] != "$2a" && config.AdminPasswordHash[:3] != "$2b" {
			fmt.Println("⚠️  Invalid password hash format, regenerating...")
			password := GenerateRandomString(16)
			hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			config.AdminPasswordHash = string(hash)
			SaveConfig(&config)
			fmt.Printf("🔑 New password: %s\n", password)
		} else {
			fmt.Printf("✅ Password hash loaded (%d chars)\n", len(config.AdminPasswordHash))
		}

		// Ensure jwt_secret exists
		if config.JWTSecret == "" {
			config.JWTSecret = GenerateRandomString(64)
			SaveConfig(&config)
		}

		// Initialize default group dimensions if not present
		if len(config.GroupDimensions) == 0 {
			config.GroupDimensions = GetDefaultGroupDimensions()
			SaveConfig(&config)
			fmt.Println("✅ Initialized default group dimensions")
		}

		InitJWTSecret(config.JWTSecret)
		return &config, nil
	}

	// First run - generate random password
	config, password := NewAppConfigWithRandomPassword()
	SaveConfig(config)
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
	
	SaveConfig(config)
	
	// Re-initialize JWT secret in case server is running
	InitJWTSecret(config.JWTSecret)
	
	return password
}

func SaveConfig(config *AppConfig) {
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

