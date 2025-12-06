package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const ConfigFilename = "vstats-agent.json"

type AgentConfig struct {
	DashboardURL string `json:"dashboard_url"`
	ServerID     string `json:"server_id"`
	AgentToken   string `json:"agent_token"`
	ServerName   string `json:"server_name"`
	Location     string `json:"location"`
	Provider     string `json:"provider"`
	IntervalSecs uint64 `json:"interval_secs"`
}

func DefaultConfigPath() string {
	// Check for environment variable override
	if envPath := os.Getenv("VSTATS_CONFIG_PATH"); envPath != "" {
		return envPath
	}

	// Try system-wide locations first
	if configDir := os.Getenv("PROGRAMDATA"); configDir != "" {
		path := filepath.Join(configDir, "vstats-agent", ConfigFilename)
		if _, err := os.Stat(filepath.Dir(path)); err == nil {
			return path
		}
	}

	// Try /etc/vstats-agent/
	if _, err := os.Stat("/etc/vstats-agent"); err == nil {
		return "/etc/vstats-agent/" + ConfigFilename
	}

	// Try /opt/vstats-agent/
	if _, err := os.Stat("/opt/vstats-agent/config.json"); err == nil {
		return "/opt/vstats-agent/config.json"
	}

	// Fall back to user config directory
	if configDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(configDir, "vstats-agent", ConfigFilename)
	}

	// Last resort: current directory
	return ConfigFilename
}

// LoadConfigFromEnv loads configuration from environment variables
// Returns nil if required environment variables are not set
func LoadConfigFromEnv() *AgentConfig {
	dashboardURL := os.Getenv("VSTATS_DASHBOARD_URL")
	serverID := os.Getenv("VSTATS_SERVER_ID")
	agentToken := os.Getenv("VSTATS_AGENT_TOKEN")

	// Required fields
	if dashboardURL == "" || serverID == "" || agentToken == "" {
		return nil
	}

	intervalSecs := uint64(5)
	if intervalStr := os.Getenv("VSTATS_INTERVAL_SECS"); intervalStr != "" {
		if parsed, err := strconv.ParseUint(intervalStr, 10, 64); err == nil && parsed > 0 {
			intervalSecs = parsed
		}
	}

	return &AgentConfig{
		DashboardURL: dashboardURL,
		ServerID:     serverID,
		AgentToken:   agentToken,
		ServerName:   os.Getenv("VSTATS_SERVER_NAME"),
		Location:     os.Getenv("VSTATS_LOCATION"),
		Provider:     os.Getenv("VSTATS_PROVIDER"),
		IntervalSecs: intervalSecs,
	}
}

func LoadConfig(path string) (*AgentConfig, error) {
	// First, try to load from environment variables
	if envConfig := LoadConfigFromEnv(); envConfig != nil {
		return envConfig, nil
	}

	// Fall back to config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.IntervalSecs == 0 {
		config.IntervalSecs = 5
	}

	return &config, nil
}

func SaveConfig(config *AgentConfig, path string) error {
	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (c *AgentConfig) WSUrl() string {
	url := c.DashboardURL
	if len(url) > 4 && url[:4] == "http" {
		if url[:5] == "https" {
			url = "wss" + url[5:]
		} else {
			url = "ws" + url[4:]
		}
	}
	return fmt.Sprintf("%s/ws/agent", url)
}

