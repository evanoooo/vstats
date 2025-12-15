package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// GeoIP API Handlers
// ============================================================================

// GetGeoIPConfig returns the current GeoIP configuration
func (s *AppState) GetGeoIPConfig(c *gin.Context) {
	service := GetGeoIPService()
	config := service.GetConfig()

	response := gin.H{
		"provider":           config.Provider,
		"mmdb_path":          config.MMDBPath,
		"ipinfo_token":       maskToken(config.IPInfoToken),
		"auto_update":        config.AutoUpdate,
		"update_interval_hr": config.UpdateIntervalHr,
		"last_update":        config.LastUpdate,
		"mmdb_loaded":        service.IsMMDBLoaded(),
	}

	c.JSON(http.StatusOK, response)
}

// UpdateGeoIPConfig updates the GeoIP configuration
func (s *AppState) UpdateGeoIPConfig(c *gin.Context) {
	var req struct {
		Provider         string `json:"provider"`
		MMDBPath         string `json:"mmdb_path"`
		IPInfoToken      string `json:"ipinfo_token"`
		AutoUpdate       bool   `json:"auto_update"`
		UpdateIntervalHr int    `json:"update_interval_hr"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validate provider
	validProviders := map[string]bool{"auto": true, "mmdb": true, "ip-api": true, "ipinfo": true}
	if !validProviders[req.Provider] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider. Must be: auto, mmdb, ip-api, or ipinfo"})
		return
	}

	// Get current config to preserve unchanged fields
	service := GetGeoIPService()
	currentConfig := service.GetConfig()

	// Update config
	newConfig := &GeoIPConfig{
		Provider:         req.Provider,
		MMDBPath:         req.MMDBPath,
		AutoUpdate:       req.AutoUpdate,
		UpdateIntervalHr: req.UpdateIntervalHr,
		LastUpdate:       currentConfig.LastUpdate,
	}

	// Only update token if provided (not masked)
	if req.IPInfoToken != "" && req.IPInfoToken != maskToken(currentConfig.IPInfoToken) {
		newConfig.IPInfoToken = req.IPInfoToken
	} else {
		newConfig.IPInfoToken = currentConfig.IPInfoToken
	}

	// Set default update interval
	if newConfig.UpdateIntervalHr <= 0 {
		newConfig.UpdateIntervalHr = 24
	}

	// Initialize service with new config
	if err := service.SetConfig(newConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to apply config: %v", err)})
		return
	}

	// Save to app config
	s.ConfigMu.Lock()
	s.Config.GeoIPConfig = newConfig
	SaveConfig(s.Config)
	s.ConfigMu.Unlock()

	LogAuditFromContext(c, AuditActionSettingsUpdate, AuditCategorySettings, "settings", "geoip", "GeoIP Config", "GeoIP configuration updated")

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"mmdb_loaded": service.IsMMDBLoaded(),
	})
}

// LookupGeoIP performs a GeoIP lookup for a single IP
func (s *AppState) LookupGeoIP(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IP address required"})
		return
	}

	service := GetGeoIPService()
	result, err := service.Lookup(ip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Lookup failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, result)
}

// LookupGeoIPBatch performs GeoIP lookup for multiple IPs
func (s *AppState) LookupGeoIPBatch(c *gin.Context) {
	var req struct {
		IPs []string `json:"ips"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if len(req.IPs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one IP required"})
		return
	}

	if len(req.IPs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 100 IPs per request"})
		return
	}

	service := GetGeoIPService()
	results := service.LookupBatch(req.IPs)

	c.JSON(http.StatusOK, results)
}

// RefreshServerGeoIP updates GeoIP data for all servers
func (s *AppState) RefreshServerGeoIP(c *gin.Context) {
	s.ConfigMu.Lock()
	defer s.ConfigMu.Unlock()

	service := GetGeoIPService()
	updated := 0
	errors := []string{}

	for i := range s.Config.Servers {
		server := &s.Config.Servers[i]
		if server.IP == "" {
			continue
		}

		result, err := service.Lookup(server.IP)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s (%s): %v", server.Name, server.IP, err))
			continue
		}

		// Update location with country code if not manually set or different
		if result.CountryCode != "" {
			oldLocation := server.Location
			server.Location = result.CountryCode
			server.GeoIP = &ServerGeoIP{
				CountryCode: result.CountryCode,
				CountryName: result.CountryName,
				City:        result.City,
				Region:      result.Region,
				Latitude:    result.Latitude,
				Longitude:   result.Longitude,
				UpdatedAt:   time.Now().Format(time.RFC3339),
			}
			if oldLocation != server.Location {
				updated++
			}
		}
	}

	// Update last update time
	if s.Config.GeoIPConfig != nil {
		s.Config.GeoIPConfig.LastUpdate = time.Now().Format(time.RFC3339)
	}

	SaveConfig(s.Config)

	LogAuditFromContext(c, AuditActionSettingsUpdate, AuditCategorySettings, "settings", "geoip", "GeoIP Refresh", fmt.Sprintf("Updated %d servers", updated))

	response := gin.H{
		"success":      true,
		"updated":      updated,
		"total":        len(s.Config.Servers),
		"last_update":  time.Now().Format(time.RFC3339),
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	c.JSON(http.StatusOK, response)
}

// ClearGeoIPCache clears the GeoIP lookup cache
func (s *AppState) ClearGeoIPCache(c *gin.Context) {
	service := GetGeoIPService()
	service.ClearCache()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetServerGeoIP returns GeoIP data for a specific server
func (s *AppState) GetServerGeoIP(c *gin.Context) {
	serverID := c.Param("id")

	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	for _, server := range s.Config.Servers {
		if server.ID == serverID {
			if server.GeoIP != nil {
				c.JSON(http.StatusOK, gin.H{
					"server_id":    server.ID,
					"server_name":  server.Name,
					"ip":           server.IP,
					"location":     server.Location,
					"geoip":        server.GeoIP,
					"flag":         CountryCodeToFlag(server.Location),
				})
				return
			}

			// No cached GeoIP, do a live lookup
			if server.IP != "" {
				service := GetGeoIPService()
				result, err := service.Lookup(server.IP)
				if err == nil {
					c.JSON(http.StatusOK, gin.H{
						"server_id":    server.ID,
						"server_name":  server.Name,
						"ip":           server.IP,
						"location":     server.Location,
						"geoip": &ServerGeoIP{
							CountryCode: result.CountryCode,
							CountryName: result.CountryName,
							City:        result.City,
							Region:      result.Region,
							Latitude:    result.Latitude,
							Longitude:   result.Longitude,
						},
						"flag": CountryCodeToFlag(result.CountryCode),
					})
					return
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"server_id":   server.ID,
				"server_name": server.Name,
				"ip":          server.IP,
				"location":    server.Location,
				"geoip":       nil,
				"flag":        CountryCodeToFlag(server.Location),
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
}

// ============================================================================
// Helper Functions
// ============================================================================

// maskToken masks a token for display (show first 4 and last 4 chars)
func maskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
