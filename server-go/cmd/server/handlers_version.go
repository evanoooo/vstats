package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Version Check Handlers
// ============================================================================

type ServerVersionInfo struct {
	Version string `json:"version"`
	Channel string `json:"channel"` // "stable" or "nightly"
}

// NightlyVersionInfo contains nightly version details
type NightlyVersionInfo struct {
	Version     string `json:"version"`
	Commit      string `json:"commit,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	DockerTag   string `json:"docker_tag,omitempty"`
	Changelog   string `json:"changelog,omitempty"`
}

// ExtendedVersionInfo contains both stable and nightly version info
type ExtendedVersionInfo struct {
	Current         string              `json:"current"`
	Channel         string              `json:"channel"` // "stable" or "nightly"
	Latest          *string             `json:"latest,omitempty"`
	LatestNightly   *NightlyVersionInfo `json:"latest_nightly,omitempty"`
	UpdateAvailable bool                `json:"update_available"`
	NightlyAvailable bool               `json:"nightly_available"`
}

// Version cache for nightly releases
var (
	nightlyVersionCache    *NightlyVersionInfo
	nightlyVersionCacheMu  sync.RWMutex
	nightlyVersionCacheExp time.Time
)

const nightlyVersionCacheTTL = 5 * time.Minute

func GetServerVersion(c *gin.Context) {
	channel := "stable"
	if strings.Contains(ServerVersion, "nightly") {
		channel = "nightly"
	}
	c.JSON(http.StatusOK, ServerVersionInfo{
		Version: ServerVersion,
		Channel: channel,
	})
}

func CheckLatestVersion(c *gin.Context) {
	// Check if user wants nightly channel
	channel := c.Query("channel") // "stable", "nightly", or empty for both

	latest, err := fetchLatestGitHubVersion("zsai001", "vstats")
	updateAvailable := false
	if err == nil && latest != nil && *latest != ServerVersion {
		// Only mark update available if current version is not nightly or latest is newer
		if !strings.Contains(ServerVersion, "nightly") {
			updateAvailable = true
		}
	}

	// Fetch nightly version info
	nightlyInfo, _ := fetchNightlyVersion("zsai001", "vstats")
	nightlyAvailable := false
	if nightlyInfo != nil && nightlyInfo.Version != ServerVersion {
		nightlyAvailable = true
	}

	// If user specifically requested nightly channel, check nightly updates
	if channel == "nightly" && strings.Contains(ServerVersion, "nightly") && nightlyInfo != nil {
		if nightlyInfo.Version != ServerVersion {
			updateAvailable = true
		}
	}

	c.JSON(http.StatusOK, ExtendedVersionInfo{
		Current:          ServerVersion,
		Channel:          getVersionChannel(ServerVersion),
		Latest:           latest,
		LatestNightly:    nightlyInfo,
		UpdateAvailable:  updateAvailable,
		NightlyAvailable: nightlyAvailable,
	})
}

// getVersionChannel determines if the version is stable or nightly
func getVersionChannel(version string) string {
	if strings.Contains(version, "nightly") {
		return "nightly"
	}
	return "stable"
}

// fetchNightlyVersion fetches the latest nightly release info
func fetchNightlyVersion(owner, repo string) (*NightlyVersionInfo, error) {
	// Check cache first
	nightlyVersionCacheMu.RLock()
	if nightlyVersionCache != nil && time.Now().Before(nightlyVersionCacheExp) {
		cached := nightlyVersionCache
		nightlyVersionCacheMu.RUnlock()
		return cached, nil
	}
	nightlyVersionCacheMu.RUnlock()

	// Fetch from GitHub
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/nightly", owner, repo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "vstats-server")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Parse release info
	info := &NightlyVersionInfo{
		DownloadURL: "https://github.com/" + owner + "/" + repo + "/releases/tag/nightly",
		DockerTag:   "nightly",
	}

	// Extract version from release name (format: "🌙 Nightly Build (x.y.z-nightly.YYYYMMDD.HHMMSS)")
	if name, ok := result["name"].(string); ok {
		if start := strings.Index(name, "("); start != -1 {
			if end := strings.Index(name, ")"); end != -1 && end > start {
				info.Version = name[start+1 : end]
			}
		}
	}

	// Extract changelog from body
	if body, ok := result["body"].(string); ok {
		// Extract the changes section
		if idx := strings.Index(body, "### Changes in this build"); idx != -1 {
			changelog := body[idx+len("### Changes in this build"):]
			if endIdx := strings.Index(changelog, "###"); endIdx != -1 {
				changelog = changelog[:endIdx]
			}
			info.Changelog = strings.TrimSpace(changelog)
		}
	}

	// Extract timestamp
	if publishedAt, ok := result["published_at"].(string); ok {
		info.Timestamp = publishedAt
	}

	// Extract commit from tag reference
	if targetCommitish, ok := result["target_commitish"].(string); ok {
		if len(targetCommitish) >= 7 {
			info.Commit = targetCommitish[:7]
		}
	}

	// Update cache
	nightlyVersionCacheMu.Lock()
	nightlyVersionCache = info
	nightlyVersionCacheExp = time.Now().Add(nightlyVersionCacheTTL)
	nightlyVersionCacheMu.Unlock()

	return info, nil
}

// ============================================================================
// Version Update Notification
// ============================================================================

// VersionUpdateMessage is sent to dashboards when a new version is available
type VersionUpdateMessage struct {
	Type            string              `json:"type"` // "version_update"
	Current         string              `json:"current"`
	Channel         string              `json:"channel"`
	Latest          *string             `json:"latest,omitempty"`
	LatestNightly   *NightlyVersionInfo `json:"latest_nightly,omitempty"`
	UpdateAvailable bool                `json:"update_available"`
	NightlyAvailable bool               `json:"nightly_available"`
}

var (
	lastKnownStableVersion  string
	lastKnownNightlyVersion string
	versionCheckMu          sync.Mutex
)

// StartVersionCheckLoop starts a background loop to check for version updates
func StartVersionCheckLoop(state *AppState) {
	// Initial check after 30 seconds
	time.Sleep(30 * time.Second)

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	// Do initial check
	checkAndNotifyVersionUpdate(state)

	for range ticker.C {
		checkAndNotifyVersionUpdate(state)
	}
}

// checkAndNotifyVersionUpdate checks for new versions and notifies connected dashboards
func checkAndNotifyVersionUpdate(state *AppState) {
	versionCheckMu.Lock()
	defer versionCheckMu.Unlock()

	channel := getVersionChannel(ServerVersion)
	var newVersionFound bool

	// Check stable version
	stableVersion, err := fetchLatestGitHubVersion("zsai001", "vstats")
	if err == nil && stableVersion != nil && *stableVersion != lastKnownStableVersion {
		if lastKnownStableVersion != "" {
			// New stable version found
			newVersionFound = true
			fmt.Printf("🆕 New stable version available: %s\n", *stableVersion)
		}
		lastKnownStableVersion = *stableVersion
	}

	// Check nightly version
	nightlyInfo, err := fetchNightlyVersion("zsai001", "vstats")
	if err == nil && nightlyInfo != nil && nightlyInfo.Version != lastKnownNightlyVersion {
		if lastKnownNightlyVersion != "" {
			// New nightly version found
			newVersionFound = true
			fmt.Printf("🌙 New nightly version available: %s\n", nightlyInfo.Version)
		}
		lastKnownNightlyVersion = nightlyInfo.Version
	}

	// Notify connected dashboards if new version found
	if newVersionFound {
		updateAvailable := false
		nightlyAvailable := false

		// Determine update availability based on current channel
		if channel == "stable" && stableVersion != nil && *stableVersion != ServerVersion {
			updateAvailable = true
		}
		if channel == "nightly" && nightlyInfo != nil && nightlyInfo.Version != ServerVersion {
			updateAvailable = true
		}
		if nightlyInfo != nil && nightlyInfo.Version != ServerVersion {
			nightlyAvailable = true
		}

		msg := VersionUpdateMessage{
			Type:             "version_update",
			Current:          ServerVersion,
			Channel:          channel,
			Latest:           stableVersion,
			LatestNightly:    nightlyInfo,
			UpdateAvailable:  updateAvailable,
			NightlyAvailable: nightlyAvailable,
		}

		if data, err := json.Marshal(msg); err == nil {
			state.BroadcastMetrics(string(data))
			fmt.Printf("📢 Notified %d dashboard(s) about version update\n", len(state.DashboardClients))
		}
	}
}

// CheckVersionOnStartup performs an initial version check on server startup
func CheckVersionOnStartup() {
	channel := getVersionChannel(ServerVersion)
	fmt.Printf("📦 Server version: %s (channel: %s)\n", ServerVersion, channel)

	// Check for updates (non-blocking)
	go func() {
		time.Sleep(5 * time.Second)

		if channel == "nightly" {
			// Check nightly updates
			nightlyInfo, err := fetchNightlyVersion("zsai001", "vstats")
			if err == nil && nightlyInfo != nil {
				lastKnownNightlyVersion = nightlyInfo.Version
				if nightlyInfo.Version != ServerVersion {
					fmt.Printf("🌙 Nightly update available: %s -> %s\n", ServerVersion, nightlyInfo.Version)
					fmt.Printf("   Download: %s\n", nightlyInfo.DownloadURL)
				}
			}
		}

		// Always check stable version
		stableVersion, err := fetchLatestGitHubVersion("zsai001", "vstats")
		if err == nil && stableVersion != nil {
			lastKnownStableVersion = *stableVersion
			if channel == "stable" && *stableVersion != ServerVersion {
				fmt.Printf("🆕 Update available: %s -> %s\n", ServerVersion, *stableVersion)
			}
		}
	}()
}

// ============================================================================
// Server Upgrade Handler
// ============================================================================

type UpgradeServerRequest struct {
	Force bool `json:"force"`
}

type UpgradeServerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Output  string `json:"output,omitempty"`
}

func UpgradeServer(c *gin.Context) {
	var req UpgradeServerRequest
	c.ShouldBindJSON(&req)

	// Always use --force flag to ensure reinstall even if version matches
	// This ensures users can reinstall/repair if needed
	upgradeCmd := "curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- --upgrade --force"

	// Use nohup and setsid to run in a completely detached process
	// that survives the server shutdown during upgrade:
	// - setsid creates a new session (detaches from parent process group)
	// - nohup ignores SIGHUP signal
	// - Redirect output to log file for debugging
	logFile := "/tmp/vstats-upgrade.log"
	detachedCmd := fmt.Sprintf("nohup setsid bash -c '%s' > %s 2>&1 &", upgradeCmd, logFile)

	// Execute the detached command - use Start() not Run() so we don't wait
	cmd := exec.Command("bash", "-c", detachedCmd)
	err := cmd.Start()

	if err != nil {
		c.JSON(http.StatusOK, UpgradeServerResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start upgrade: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, UpgradeServerResponse{
		Success: true,
		Message: "Upgrade started in background (force mode). The server will restart shortly. Check /tmp/vstats-upgrade.log for details.",
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func fetchLatestGitHubVersion(owner, repo string) (*string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "vstats-server")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	tagName, ok := result["tag_name"].(string)
	if !ok {
		return nil, fmt.Errorf("no tag_name in response")
	}

	// Remove 'v' prefix if present
	if len(tagName) > 0 && tagName[0] == 'v' {
		tagName = tagName[1:]
	}

	return &tagName, nil
}
