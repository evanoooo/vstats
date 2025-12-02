package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func LoadConfig(path string) (*AgentConfig, error) {
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

