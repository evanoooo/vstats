package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Theme Types
// ============================================================================

// ThemeManifest represents a theme.json file
type ThemeManifest struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Author        string                 `json:"author"`
	Description   string                 `json:"description"`
	NameZh        string                 `json:"nameZh,omitempty"`
	DescriptionZh string                 `json:"descriptionZh,omitempty"`
	IsDark        bool                   `json:"isDark"`
	Style         string                 `json:"style"`
	Preview       map[string]string      `json:"preview"`
	Fonts         map[string]string      `json:"fonts"`
	BorderRadius  string                 `json:"borderRadius"`
	CardStyle     string                 `json:"cardStyle"`
	License       string                 `json:"license,omitempty"`
	Homepage      string                 `json:"homepage,omitempty"`
	Repository    string                 `json:"repository,omitempty"`
	Keywords      []string               `json:"keywords,omitempty"`
	MinVersion    string                 `json:"minVersion,omitempty"`
	CSSFile       string                 `json:"cssFile,omitempty"`
	AssetsDir     string                 `json:"assetsDir,omitempty"`
	PreviewImage  string                 `json:"previewImage,omitempty"`
}

// ThemeSource represents where a theme was installed from
type ThemeSource struct {
	Type     string `json:"type"`     // builtin, github, url, local
	Location string `json:"location"` // GitHub: user/repo, URL: full URL
	Ref      string `json:"ref,omitempty"`
}

// InstalledTheme represents an installed theme
type InstalledTheme struct {
	Manifest    ThemeManifest `json:"manifest"`
	Source      ThemeSource   `json:"source"`
	InstalledAt string        `json:"installedAt"`
	UpdatedAt   string        `json:"updatedAt,omitempty"`
	CSSContent  string        `json:"cssContent,omitempty"`
	Enabled     bool          `json:"enabled"`
}

// InstallThemeRequest is the request body for installing a theme
type InstallThemeRequest struct {
	Source string `json:"source"` // GitHub: user/repo or user/repo/path, URL: full URL
	Ref    string `json:"ref,omitempty"`
}

// ThemesResponse is the response for listing themes
type ThemesResponse struct {
	Installed []InstalledTheme `json:"installed"`
	Current   string           `json:"current"`
}

// ============================================================================
// GitHub URL Helpers
// ============================================================================

const githubRawBase = "https://raw.githubusercontent.com"

type parsedGitHubSource struct {
	Owner string
	Repo  string
	Path  string
	Ref   string
}

func parseGitHubSource(source string, defaultRef string) (*parsedGitHubSource, error) {
	ref := defaultRef
	cleanSource := source

	// Extract ref if specified with @
	if strings.Contains(source, "@") {
		parts := strings.SplitN(source, "@", 2)
		cleanSource = parts[0]
		if len(parts) > 1 && parts[1] != "" {
			ref = parts[1]
		}
	}

	parts := strings.Split(cleanSource, "/")
	var filteredParts []string
	for _, p := range parts {
		if p != "" {
			filteredParts = append(filteredParts, p)
		}
	}

	if len(filteredParts) < 2 {
		return nil, fmt.Errorf("invalid GitHub source: %s. Expected format: user/repo or user/repo/path", source)
	}

	parsed := &parsedGitHubSource{
		Owner: filteredParts[0],
		Repo:  filteredParts[1],
		Ref:   ref,
	}

	if len(filteredParts) > 2 {
		parsed.Path = strings.Join(filteredParts[2:], "/")
	}

	return parsed, nil
}

func buildGitHubRawURL(parsed *parsedGitHubSource, filename string) string {
	pathPrefix := ""
	if parsed.Path != "" {
		pathPrefix = parsed.Path + "/"
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s%s", githubRawBase, parsed.Owner, parsed.Repo, parsed.Ref, pathPrefix, filename)
}

// ============================================================================
// Theme Handlers
// ============================================================================

// GetInstalledThemes returns all installed themes
func (s *AppState) GetInstalledThemes(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	themes := s.Config.InstalledThemes
	if themes == nil {
		themes = []InstalledTheme{}
	}

	currentTheme := "midnight" // default
	if s.Config.SiteSettings.Theme != nil && s.Config.SiteSettings.Theme.ThemeId != "" {
		currentTheme = s.Config.SiteSettings.Theme.ThemeId
	}

	c.JSON(http.StatusOK, ThemesResponse{
		Installed: themes,
		Current:   currentTheme,
	})
}

// InstallTheme installs a theme from a source
func (s *AppState) InstallTheme(c *gin.Context) {
	var req InstallThemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Source == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source is required"})
		return
	}

	// Determine source type
	sourceType := "github"
	if strings.HasPrefix(req.Source, "file://") {
		sourceType = "local"
	} else if strings.HasPrefix(req.Source, "http://") || strings.HasPrefix(req.Source, "https://") {
		sourceType = "url"
	}

	// Fetch manifest
	manifest, manifestURL, err := fetchThemeManifest(req.Source, req.Ref)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to fetch theme manifest: %v", err)})
		return
	}

	// Validate manifest
	if err := validateManifest(manifest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid theme manifest: %v", err)})
		return
	}

	// Fetch CSS
	cssContent, err := fetchThemeCSS(req.Source, manifest, req.Ref, manifestURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to fetch theme CSS: %v", err)})
		return
	}

	// Create installed theme
	installedTheme := InstalledTheme{
		Manifest: *manifest,
		Source: ThemeSource{
			Type:     sourceType,
			Location: req.Source,
			Ref:      req.Ref,
		},
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		CSSContent:  cssContent,
		Enabled:     true,
	}

	// Save to config
	s.ConfigMu.Lock()
	if s.Config.InstalledThemes == nil {
		s.Config.InstalledThemes = []InstalledTheme{}
	}

	// Check if already installed, update if so
	found := false
	for i, t := range s.Config.InstalledThemes {
		if t.Manifest.ID == manifest.ID {
			installedTheme.InstalledAt = t.InstalledAt
			installedTheme.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			s.Config.InstalledThemes[i] = installedTheme
			found = true
			break
		}
	}

	if !found {
		s.Config.InstalledThemes = append(s.Config.InstalledThemes, installedTheme)
	}

	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	LogAuditFromContext(c, AuditActionThemeInstall, AuditCategorySettings, "theme", manifest.ID, manifest.Name, fmt.Sprintf("Theme installed from %s", req.Source))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"theme":   installedTheme,
	})
}

// UninstallTheme removes an installed theme
func (s *AppState) UninstallTheme(c *gin.Context) {
	themeID := c.Param("id")
	if themeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Theme ID is required"})
		return
	}

	s.ConfigMu.Lock()
	defer s.ConfigMu.Unlock()

	if s.Config.InstalledThemes == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Theme not found"})
		return
	}

	found := false
	var themeName string
	for i, t := range s.Config.InstalledThemes {
		if t.Manifest.ID == themeID {
			themeName = t.Manifest.Name
			s.Config.InstalledThemes = append(s.Config.InstalledThemes[:i], s.Config.InstalledThemes[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Theme not found"})
		return
	}

	// If current theme is being uninstalled, switch to default
	if s.Config.SiteSettings.Theme != nil && s.Config.SiteSettings.Theme.ThemeId == themeID {
		s.Config.SiteSettings.Theme.ThemeId = "midnight"
	}

	SaveConfig(s.Config)

	LogAuditFromContext(c, AuditActionThemeUninstall, AuditCategorySettings, "theme", themeID, themeName, "Theme uninstalled")

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetThemeCSS returns the CSS for an installed theme
func (s *AppState) GetThemeCSS(c *gin.Context) {
	themeID := c.Param("id")
	if themeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Theme ID is required"})
		return
	}

	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	for _, t := range s.Config.InstalledThemes {
		if t.Manifest.ID == themeID {
			c.Header("Content-Type", "text/css")
			c.String(http.StatusOK, t.CSSContent)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Theme not found"})
}

// CheckThemeUpdate checks if a theme has updates available
func (s *AppState) CheckThemeUpdate(c *gin.Context) {
	themeID := c.Param("id")
	if themeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Theme ID is required"})
		return
	}

	s.ConfigMu.RLock()
	var installed *InstalledTheme
	for i, t := range s.Config.InstalledThemes {
		if t.Manifest.ID == themeID {
			installed = &s.Config.InstalledThemes[i]
			break
		}
	}
	s.ConfigMu.RUnlock()

	if installed == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Theme not installed"})
		return
	}

	// Fetch latest manifest
	latestManifest, _, err := fetchThemeManifest(installed.Source.Location, installed.Source.Ref)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"hasUpdate": false,
			"error":     fmt.Sprintf("Failed to check for updates: %v", err),
		})
		return
	}

	hasUpdate := latestManifest.Version != installed.Manifest.Version

	c.JSON(http.StatusOK, gin.H{
		"themeId":        themeID,
		"currentVersion": installed.Manifest.Version,
		"latestVersion":  latestManifest.Version,
		"hasUpdate":      hasUpdate,
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func fetchThemeManifest(source string, ref string) (*ThemeManifest, string, error) {
	var manifestURL string

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		if strings.HasSuffix(source, "theme.json") {
			manifestURL = source
		} else {
			if strings.HasSuffix(source, "/") {
				manifestURL = source + "theme.json"
			} else {
				manifestURL = source + "/theme.json"
			}
		}
	} else {
		// GitHub source
		defaultRef := "main"
		if ref != "" {
			defaultRef = ref
		}
		parsed, err := parseGitHubSource(source, defaultRef)
		if err != nil {
			return nil, "", err
		}
		manifestURL = buildGitHubRawURL(parsed, "theme.json")
	}

	resp, err := http.Get(manifestURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch manifest: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("manifest not found: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read manifest: %v", err)
	}

	var manifest ThemeManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, "", fmt.Errorf("invalid manifest JSON: %v", err)
	}

	return &manifest, manifestURL, nil
}

func fetchThemeCSS(source string, manifest *ThemeManifest, ref string, manifestURL string) (string, error) {
	cssFilename := "theme.css"
	if manifest.CSSFile != "" {
		cssFilename = manifest.CSSFile
	}

	var cssURL string

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		baseURL := manifestURL
		if strings.HasSuffix(baseURL, "theme.json") {
			baseURL = strings.TrimSuffix(baseURL, "theme.json")
		}
		if !strings.HasSuffix(baseURL, "/") {
			baseURL = baseURL + "/"
		}
		cssURL = baseURL + cssFilename
	} else {
		// GitHub source
		defaultRef := "main"
		if ref != "" {
			defaultRef = ref
		}
		parsed, err := parseGitHubSource(source, defaultRef)
		if err != nil {
			return "", err
		}
		cssURL = buildGitHubRawURL(parsed, cssFilename)
	}

	resp, err := http.Get(cssURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch CSS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CSS not found: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read CSS: %v", err)
	}

	return string(body), nil
}

func validateManifest(manifest *ThemeManifest) error {
	if manifest.ID == "" {
		return fmt.Errorf("missing required field: id")
	}
	if manifest.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if manifest.Version == "" {
		return fmt.Errorf("missing required field: version")
	}
	if manifest.Author == "" {
		return fmt.Errorf("missing required field: author")
	}
	if manifest.Description == "" {
		return fmt.Errorf("missing required field: description")
	}

	// Validate ID format
	idRegex := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !idRegex.MatchString(manifest.ID) {
		return fmt.Errorf("invalid theme ID: must be lowercase alphanumeric with hyphens")
	}

	// Validate version format
	versionRegex := regexp.MustCompile(`^\d+\.\d+\.\d+`)
	if !versionRegex.MatchString(manifest.Version) {
		return fmt.Errorf("invalid version format: use semantic versioning (e.g., 1.0.0)")
	}

	// Validate style
	validStyles := map[string]bool{
		"flat":       true,
		"glass":      true,
		"neumorphic": true,
		"brutalist":  true,
		"minimal":    true,
	}
	if !validStyles[manifest.Style] {
		return fmt.Errorf("invalid style: must be one of flat, glass, neumorphic, brutalist, minimal")
	}

	// Validate preview
	if manifest.Preview == nil {
		return fmt.Errorf("missing required field: preview")
	}
	requiredPreviewFields := []string{"primary", "secondary", "accent", "background"}
	for _, field := range requiredPreviewFields {
		if manifest.Preview[field] == "" {
			return fmt.Errorf("missing preview field: %s", field)
		}
	}

	// Validate fonts
	if manifest.Fonts == nil {
		return fmt.Errorf("missing required field: fonts")
	}
	requiredFontFields := []string{"heading", "body", "mono"}
	for _, field := range requiredFontFields {
		if manifest.Fonts[field] == "" {
			return fmt.Errorf("missing font field: %s", field)
		}
	}

	return nil
}

// Audit action constants for themes
const (
	AuditActionThemeInstall   = "theme_install"
	AuditActionThemeUninstall = "theme_uninstall"
)

