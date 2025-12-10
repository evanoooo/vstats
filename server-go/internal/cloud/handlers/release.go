package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"vstats/internal/cloud/redis"

	"github.com/gin-gonic/gin"
)

const (
	GitHubRepo     = "zsai001/vstats"
	GitHubAPI      = "https://api.github.com/repos/" + GitHubRepo + "/releases/latest"
	GitHubDownload = "https://github.com/" + GitHubRepo + "/releases/download"

	// Cache durations
	VersionCacheDuration = 5 * time.Minute
	BinaryCacheDuration  = 1 * time.Hour
)

// GitHubRelease represents GitHub release API response
type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
	Published  string `json:"published_at"`
	TarballURL string `json:"tarball_url"`
	ZipballURL string `json:"zipball_url"`
	Assets     []struct {
		Name               string `json:"name"`
		Size               int64  `json:"size"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// GetLatestVersion returns the latest version info
func GetLatestVersion(c *gin.Context) {
	ctx := context.Background()

	// Try to get from cache
	info, err := redis.GetReleaseInfo(ctx)
	if err == nil && info != nil {
		c.JSON(http.StatusOK, gin.H{
			"version":      info.Version,
			"name":         info.Name,
			"published_at": info.PublishedAt,
			"cached":       true,
		})
		return
	}

	// Fetch from GitHub
	info, err = fetchLatestRelease(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Failed to fetch latest version: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"version":      info.Version,
		"name":         info.Name,
		"published_at": info.PublishedAt,
		"cached":       false,
	})
}

// GetLatestVersionText returns just the version string (for shell scripts)
func GetLatestVersionText(c *gin.Context) {
	ctx := context.Background()

	// Try to get from cache
	info, err := redis.GetReleaseInfo(ctx)
	if err == nil && info != nil {
		c.String(http.StatusOK, info.Version)
		return
	}

	// Fetch from GitHub
	info, err = fetchLatestRelease(ctx)
	if err != nil {
		c.String(http.StatusServiceUnavailable, "")
		return
	}

	c.String(http.StatusOK, info.Version)
}

// DownloadBinary proxies and caches binary downloads
func DownloadBinary(c *gin.Context) {
	binaryName := c.Param("name")
	if binaryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Binary name required"})
		return
	}

	// Validate binary name format
	validPrefixes := []string{
		"vstats-server-",
		"vstats-agent-",
		"vstats-cli-",
		"web-dist",
	}
	isValid := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(binaryName, prefix) {
			isValid = true
			break
		}
	}
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid binary name"})
		return
	}

	ctx := context.Background()

	// Get latest version
	info, err := redis.GetReleaseInfo(ctx)
	if err != nil || info == nil {
		info, err = fetchLatestRelease(ctx)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to get release info"})
			return
		}
	}

	// Check if binary is cached
	data, contentType, err := redis.GetBinaryCache(ctx, info.Version, binaryName)
	if err == nil && len(data) > 0 {
		c.Header("X-Cache", "HIT")
		c.Header("X-Version", info.Version)
		c.Data(http.StatusOK, contentType, data)
		return
	}

	// Download from GitHub
	downloadURL := fmt.Sprintf("%s/%s/%s", GitHubDownload, info.Version, binaryName)

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to download from GitHub"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("GitHub returned status %d", resp.StatusCode)})
		return
	}

	// Read the binary
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read binary"})
		return
	}

	// Determine content type
	contentType = "application/octet-stream"
	if strings.HasSuffix(binaryName, ".tar.gz") {
		contentType = "application/gzip"
	} else if strings.HasSuffix(binaryName, ".zip") {
		contentType = "application/zip"
	}

	// Cache the binary (async to not block response)
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = redis.SetBinaryCache(cacheCtx, info.Version, binaryName, data, contentType, BinaryCacheDuration)
	}()

	c.Header("X-Cache", "MISS")
	c.Header("X-Version", info.Version)
	c.Data(http.StatusOK, contentType, data)
}

// DownloadBinaryVersion downloads a specific version
func DownloadBinaryVersion(c *gin.Context) {
	version := c.Param("version")
	binaryName := c.Param("name")

	if version == "" || binaryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Version and binary name required"})
		return
	}

	// Validate binary name format (same as DownloadBinary)
	validPrefixes := []string{
		"vstats-server-",
		"vstats-agent-",
		"vstats-cli-",
		"web-dist",
	}
	isValid := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(binaryName, prefix) {
			isValid = true
			break
		}
	}
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid binary name"})
		return
	}

	// Validate version format (should start with v)
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	ctx := context.Background()

	// Check cache
	data, contentType, err := redis.GetBinaryCache(ctx, version, binaryName)
	if err == nil && len(data) > 0 {
		c.Header("X-Cache", "HIT")
		c.Header("X-Version", version)
		c.Data(http.StatusOK, contentType, data)
		return
	}

	// Download from GitHub
	downloadURL := fmt.Sprintf("%s/%s/%s", GitHubDownload, version, binaryName)

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to download from GitHub"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("GitHub returned status %d", resp.StatusCode)})
		return
	}

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read binary"})
		return
	}

	contentType = "application/octet-stream"
	if strings.HasSuffix(binaryName, ".tar.gz") {
		contentType = "application/gzip"
	} else if strings.HasSuffix(binaryName, ".zip") {
		contentType = "application/zip"
	}

	// Cache the binary
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = redis.SetBinaryCache(cacheCtx, version, binaryName, data, contentType, BinaryCacheDuration)
	}()

	c.Header("X-Cache", "MISS")
	c.Header("X-Version", version)
	c.Data(http.StatusOK, contentType, data)
}

// ListAssets returns available assets for latest version
func ListAssets(c *gin.Context) {
	ctx := context.Background()

	info, err := redis.GetReleaseInfo(ctx)
	if err != nil || info == nil {
		info, err = fetchLatestRelease(ctx)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to get release info"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"version": info.Version,
		"assets":  info.Assets,
	})
}

// RefreshCache forces a cache refresh
func RefreshCache(c *gin.Context) {
	ctx := context.Background()

	// Clear existing cache
	_ = redis.DeleteReleaseInfo(ctx)

	// Fetch fresh data
	info, err := fetchLatestRelease(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to fetch release: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cache refreshed",
		"version": info.Version,
	})
}

// fetchLatestRelease fetches release info from GitHub and caches it
func fetchLatestRelease(ctx context.Context) (*redis.ReleaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", GitHubAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "vstats-cloud/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	// Build assets map
	assets := make(map[string]string)
	for _, asset := range release.Assets {
		assets[asset.Name] = asset.BrowserDownloadURL
	}

	info := &redis.ReleaseInfo{
		Version:     release.TagName,
		Name:        release.Name,
		Body:        release.Body,
		PublishedAt: release.Published,
		Assets:      assets,
		CachedAt:    time.Now().Unix(),
	}

	// Cache the info
	if err := redis.SetReleaseInfo(ctx, info, VersionCacheDuration); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: Failed to cache release info: %v\n", err)
	}

	return info, nil
}
