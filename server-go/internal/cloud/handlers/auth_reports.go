package handlers

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"vstats/internal/cloud/database"
	"vstats/internal/cloud/models"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Auth Reports Handlers
// ============================================================================

// ReportAuthRequest represents the request body for reporting an auth event
type ReportAuthRequest struct {
	SiteURL  string `json:"site_url" binding:"required"`
	Provider string `json:"provider" binding:"required"`
	Username string `json:"username" binding:"required"`
}

// ReportAuth receives auth reports from self-hosted sites
// This endpoint is public - any site can report their auth events
func ReportAuth(c *gin.Context) {
	var req ReportAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate provider
	if req.Provider != "github" && req.Provider != "google" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider, must be 'github' or 'google'"})
		return
	}

	// Extract host from site URL
	parsedURL, err := url.Parse(req.SiteURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site_url"})
		return
	}

	siteHost := parsedURL.Host
	if siteHost == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid site_url: missing host"})
		return
	}

	// Get client info
	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	report := &models.AuthReport{
		SiteURL:   req.SiteURL,
		SiteHost:  siteHost,
		Provider:  req.Provider,
		Username:  req.Username,
		IPAddress: &ipAddress,
		UserAgent: &userAgent,
	}

	ctx := context.Background()
	if err := database.CreateAuthReport(ctx, report); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ============================================================================
// Admin Stats Handlers (require authentication)
// ============================================================================

// GetAuthOverallStats returns overall auth statistics
func GetAuthOverallStats(c *gin.Context) {
	ctx := context.Background()

	stats, err := database.GetAuthOverallStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetAuthDailyStats returns daily auth statistics
func GetAuthDailyStats(c *gin.Context) {
	days := 30
	if d, err := strconv.Atoi(c.Query("days")); err == nil && d > 0 {
		days = d
	}

	ctx := context.Background()

	stats, err := database.GetAuthDailyStats(ctx, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get daily stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// GetAuthSiteStats returns site statistics
func GetAuthSiteStats(c *gin.Context) {
	limit := 100
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	ctx := context.Background()

	stats, err := database.GetAuthSiteStats(ctx, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get site stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sites": stats})
}

// GetAuthUsersBySite returns users for a specific site
func GetAuthUsersBySite(c *gin.Context) {
	siteHost := c.Param("site_host")
	if siteHost == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "site_host is required"})
		return
	}

	limit := 100
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	ctx := context.Background()

	reports, err := database.GetAuthUsersBySite(ctx, siteHost, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get site users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

// GetAuthUsersByDate returns auth reports for a specific date
func GetAuthUsersByDate(c *gin.Context) {
	date := c.Param("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}

	limit := 100
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	ctx := context.Background()

	reports, err := database.GetAuthUsersByDate(ctx, date, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get reports for date"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

