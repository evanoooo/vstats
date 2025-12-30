package main

import (
	"encoding/json"
	"log"
	"net/http"

	"vstats/internal/common"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Site Settings Handlers
// ============================================================================

func (s *AppState) GetSiteSettings(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()
	c.JSON(http.StatusOK, s.Config.SiteSettings)
}

func (s *AppState) UpdateSiteSettings(c *gin.Context) {
	var settings SiteSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	s.ConfigMu.Lock()
	s.Config.SiteSettings = settings
	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	// Broadcast the updated settings to all connected dashboard clients
	s.BroadcastSiteSettings(&settings)

	LogAuditFromContext(c, AuditActionSiteSettingsUpdate, AuditCategorySettings, "settings", "site", "Site Settings", "Site settings updated")

	c.Status(http.StatusOK)
}

// BroadcastSiteSettings sends updated site settings (including theme) to all connected clients
func (s *AppState) BroadcastSiteSettings(settings *SiteSettings) {
	msg := map[string]interface{}{
		"type":          "site_settings",
		"site_settings": settings,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal site settings: %v", err)
		return
	}

	s.DashboardMu.RLock()
	defer s.DashboardMu.RUnlock()

	for conn := range s.DashboardClients {
		if err := conn.WriteMessage(1, data); err != nil {
			log.Printf("Failed to broadcast site settings: %v", err)
		}
	}
}

// ============================================================================
// Probe Settings Handlers
// ============================================================================

func (s *AppState) GetProbeSettings(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()
	c.JSON(http.StatusOK, s.Config.ProbeSettings)
}

func (s *AppState) UpdateProbeSettings(c *gin.Context) {
	var settings ProbeSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	s.ConfigMu.Lock()
	s.Config.ProbeSettings = settings
	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	// Broadcast new ping targets to all connected agents
	s.BroadcastPingTargets(settings.PingTargets)

	LogAuditFromContext(c, AuditActionProbeSettingsUpdate, AuditCategorySettings, "settings", "probe", "Probe Settings", "Probe settings updated")

	c.Status(http.StatusOK)
}

// BroadcastPingTargets sends updated ping targets to all connected agents
func (s *AppState) BroadcastPingTargets(targets []common.PingTargetConfig) {
	msg := map[string]interface{}{
		"type":         "config",
		"ping_targets": targets,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal ping targets: %v", err)
		return
	}

	s.AgentConnsMu.RLock()
	defer s.AgentConnsMu.RUnlock()

	for serverID, conn := range s.AgentConns {
		select {
		case conn.SendChan <- data:
			log.Printf("Sent ping targets update to agent %s", serverID)
		default:
			log.Printf("Failed to send ping targets to agent %s (channel full)", serverID)
		}
	}
}

// ============================================================================
// Affiliate Provider Settings Handlers
// ============================================================================

func (s *AppState) GetAffProviders(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()
	
	providers := s.Config.AffProviders
	if providers == nil {
		providers = []AffProvider{}
	}
	c.JSON(http.StatusOK, providers)
}

func (s *AppState) UpdateAffProviders(c *gin.Context) {
	var providers []AffProvider
	if err := c.ShouldBindJSON(&providers); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	s.ConfigMu.Lock()
	s.Config.AffProviders = providers
	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	LogAuditFromContext(c, AuditActionSettingsUpdate, AuditCategorySettings, "settings", "aff_providers", "Aff Providers", "Affiliate providers updated")

	c.Status(http.StatusOK)
}

// GetAffProvidersPublic returns affiliate providers for public access (dashboard)
func (s *AppState) GetAffProvidersPublic(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()
	
	// Only return enabled providers with necessary fields
	var publicProviders []map[string]interface{}
	for _, p := range s.Config.AffProviders {
		if p.Enabled {
			publicProviders = append(publicProviders, map[string]interface{}{
				"name":     p.Name,
				"aff_link": p.AffLink,
				"logo_url": p.LogoURL,
			})
		}
	}
	
	if publicProviders == nil {
		publicProviders = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, publicProviders)
}
