//go:build windows
// +build windows

package main

import (
	"fmt"
)

// SetupSignalHandler is a no-op on Windows
// Windows doesn't support SIGHUP
func SetupSignalHandler(state *AppState) {
	// No-op on Windows
}

// findAndSignalServer is not supported on Windows
func findAndSignalServer() error {
	return fmt.Errorf("auto-restart is not supported on Windows. Please restart the server manually")
}
