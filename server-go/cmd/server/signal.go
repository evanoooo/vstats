//go:build !windows
// +build !windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// SetupSignalHandler sets up signal handlers for graceful operations
// SIGHUP: Reload password from config file
func SetupSignalHandler(state *AppState) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)

	go func() {
		for sig := range sigs {
			switch sig {
			case syscall.SIGHUP:
				fmt.Println("\n📥 Received SIGHUP, reloading config...")
				reloadConfig(state)
			}
		}
	}()
}

// reloadConfig reloads the configuration from disk
func reloadConfig(state *AppState) {
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("❌ Failed to read config: %v\n", err)
		return
	}

	var newConfig AppConfig
	if err := json.Unmarshal(data, &newConfig); err != nil {
		fmt.Printf("❌ Failed to parse config: %v\n", err)
		return
	}

	// Validate the new password hash
	if len(newConfig.AdminPasswordHash) < 4 ||
		(newConfig.AdminPasswordHash[:3] != "$2a" && newConfig.AdminPasswordHash[:3] != "$2b") {
		fmt.Println("❌ Invalid password hash format in config")
		return
	}

	// Update the config in memory
	state.ConfigMu.Lock()
	state.Config.AdminPasswordHash = newConfig.AdminPasswordHash
	if newConfig.JWTSecret != "" {
		state.Config.JWTSecret = newConfig.JWTSecret
		InitJWTSecret(newConfig.JWTSecret)
	}
	state.ConfigMu.Unlock()

	fmt.Println("✅ Config reloaded successfully - new password is now active")
}

// findAndSignalServer finds a running vstats-server process and sends SIGHUP
func findAndSignalServer() error {
	// Get current PID to exclude self
	currentPID := os.Getpid()

	// Read /proc to find vstats-server processes
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return fmt.Errorf("cannot read /proc: %w", err)
	}

	signaled := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory name is a number (PID)
		pid := 0
		if _, err := fmt.Sscanf(entry.Name(), "%d", &pid); err != nil {
			continue
		}

		// Skip self
		if pid == currentPID {
			continue
		}

		// Read cmdline to check if it's vstats-server
		cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			continue
		}

		// Check if this is vstats-server (not with --reset-password)
		if containsCmdline(cmdline, "vstats-server") && !containsCmdline(cmdline, "--reset-password") {
			// Found a running server, send SIGHUP
			proc, err := os.FindProcess(pid)
			if err != nil {
				continue
			}

			if err := proc.Signal(syscall.SIGHUP); err != nil {
				fmt.Printf("⚠️  Failed to signal process %d: %v\n", pid, err)
				continue
			}

			fmt.Printf("✅ Sent SIGHUP to vstats-server (PID: %d)\n", pid)
			signaled = true
		}
	}

	if !signaled {
		return fmt.Errorf("no running vstats-server found")
	}

	return nil
}

// containsCmdline checks if cmdline contains the substring
// cmdline from /proc uses null bytes as separators
func containsCmdline(cmdline []byte, substr string) bool {
	// Replace null bytes with spaces for easier searching
	s := string(cmdline)
	s = strings.ReplaceAll(s, "\x00", " ")
	return strings.Contains(s, substr)
}
