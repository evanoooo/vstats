package main

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// OAuth state storage (in-memory, should use Redis in production)
var (
	oauthStates   = make(map[string]*OAuthStateData)
	oauthStatesMu sync.RWMutex
)

const CentralizedOAuthURL = "https://vstats-oauth-proxy.zsai001.workers.dev"

// ============================================================================
// OAuth 2.0 Handlers
// ============================================================================

// GetOAuthProviders returns available OAuth providers (public)
func (s *AppState) GetOAuthProviders(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	providers := make(map[string]bool)
	centralized := false

	if s.Config.OAuth != nil {
		// Check centralized OAuth first
		if s.Config.OAuth.UseCentralized {
			centralized = true
			providers["github"] = true
			providers["google"] = true
		} else {
			// Self-hosted OAuth
			if s.Config.OAuth.GitHub != nil && s.Config.OAuth.GitHub.Enabled && s.Config.OAuth.GitHub.ClientID != "" {
				providers["github"] = true
			}
			if s.Config.OAuth.Google != nil && s.Config.OAuth.Google.Enabled && s.Config.OAuth.Google.ClientID != "" {
				providers["google"] = true
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"providers":   providers,
		"centralized": centralized,
	})
}

// GetOAuthSettings returns OAuth configuration (admin only)
func (s *AppState) GetOAuthSettings(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	// Return safe version without secrets
	response := gin.H{
		"use_centralized": false,
		"allowed_users":   []string{},
	}

	if s.Config.OAuth != nil {
		response["use_centralized"] = s.Config.OAuth.UseCentralized
		response["allowed_users"] = s.Config.OAuth.AllowedUsers

		if s.Config.OAuth.GitHub != nil {
			response["github"] = gin.H{
				"enabled":       s.Config.OAuth.GitHub.Enabled,
				"client_id":     s.Config.OAuth.GitHub.ClientID,
				"has_secret":    s.Config.OAuth.GitHub.ClientSecret != "",
				"allowed_users": s.Config.OAuth.GitHub.AllowedUsers,
			}
		}
		if s.Config.OAuth.Google != nil {
			response["google"] = gin.H{
				"enabled":       s.Config.OAuth.Google.Enabled,
				"client_id":     s.Config.OAuth.Google.ClientID,
				"has_secret":    s.Config.OAuth.Google.ClientSecret != "",
				"allowed_users": s.Config.OAuth.Google.AllowedUsers,
			}
		}

		// OIDC providers
		if s.Config.OAuth.OIDC != nil {
			oidcProviders := []gin.H{}
			for i, oidc := range s.Config.OAuth.OIDC {
				oidcProviders = append(oidcProviders, gin.H{
					"id":             fmt.Sprintf("oidc_%d", i),
					"enabled":        oidc.Enabled,
					"name":           oidc.Name,
					"issuer":         oidc.Issuer,
					"client_id":      oidc.ClientID,
					"has_secret":     oidc.ClientSecret != "",
					"scopes":         oidc.Scopes,
					"allowed_users":  oidc.AllowedUsers,
					"allowed_groups": oidc.AllowedGroups,
					"username_claim": oidc.UsernameClaim,
				})
			}
			response["oidc"] = oidcProviders
		}

		// Cloudflare Access
		if s.Config.OAuth.CloudflareAccess != nil {
			response["cloudflare_access"] = gin.H{
				"enabled":       s.Config.OAuth.CloudflareAccess.Enabled,
				"team_domain":   s.Config.OAuth.CloudflareAccess.TeamDomain,
				"aud":           s.Config.OAuth.CloudflareAccess.AUD,
				"allowed_users": s.Config.OAuth.CloudflareAccess.AllowedUsers,
			}
		}

		// SSO Bindings
		if s.Config.OAuth.Bindings != nil {
			response["bindings"] = s.Config.OAuth.Bindings
		}
	}

	c.JSON(http.StatusOK, response)
}

// UpdateOAuthSettings updates OAuth configuration
func (s *AppState) UpdateOAuthSettings(c *gin.Context) {
	var req struct {
		UseCentralized *bool    `json:"use_centralized,omitempty"`
		AllowedUsers   []string `json:"allowed_users,omitempty"`
		GitHub         *struct {
			Enabled      bool     `json:"enabled"`
			ClientID     string   `json:"client_id"`
			ClientSecret string   `json:"client_secret,omitempty"`
			AllowedUsers []string `json:"allowed_users"`
		} `json:"github,omitempty"`
		Google *struct {
			Enabled      bool     `json:"enabled"`
			ClientID     string   `json:"client_id"`
			ClientSecret string   `json:"client_secret,omitempty"`
			AllowedUsers []string `json:"allowed_users"`
		} `json:"google,omitempty"`
		// OIDC providers
		OIDC []struct {
			ID            string   `json:"id,omitempty"`
			Enabled       bool     `json:"enabled"`
			Name          string   `json:"name"`
			Issuer        string   `json:"issuer"`
			ClientID      string   `json:"client_id"`
			ClientSecret  string   `json:"client_secret,omitempty"`
			Scopes        []string `json:"scopes,omitempty"`
			AllowedUsers  []string `json:"allowed_users,omitempty"`
			AllowedGroups []string `json:"allowed_groups,omitempty"`
			UsernameClaim string   `json:"username_claim,omitempty"`
		} `json:"oidc,omitempty"`
		// Cloudflare Access
		CloudflareAccess *struct {
			Enabled      bool     `json:"enabled"`
			TeamDomain   string   `json:"team_domain"`
			AUD          string   `json:"aud"`
			AllowedUsers []string `json:"allowed_users,omitempty"`
		} `json:"cloudflare_access,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	s.ConfigMu.Lock()
	defer s.ConfigMu.Unlock()

	if s.Config.OAuth == nil {
		s.Config.OAuth = &OAuthConfig{}
	}

	// Update centralized OAuth settings
	if req.UseCentralized != nil {
		s.Config.OAuth.UseCentralized = *req.UseCentralized
	}
	if req.AllowedUsers != nil {
		s.Config.OAuth.AllowedUsers = req.AllowedUsers
	}

	// Update self-hosted OAuth settings
	if req.GitHub != nil {
		if s.Config.OAuth.GitHub == nil {
			s.Config.OAuth.GitHub = &OAuthProvider{}
		}
		s.Config.OAuth.GitHub.Enabled = req.GitHub.Enabled
		s.Config.OAuth.GitHub.ClientID = req.GitHub.ClientID
		if req.GitHub.ClientSecret != "" {
			s.Config.OAuth.GitHub.ClientSecret = req.GitHub.ClientSecret
		}
		s.Config.OAuth.GitHub.AllowedUsers = req.GitHub.AllowedUsers
	}

	if req.Google != nil {
		if s.Config.OAuth.Google == nil {
			s.Config.OAuth.Google = &OAuthProvider{}
		}
		s.Config.OAuth.Google.Enabled = req.Google.Enabled
		s.Config.OAuth.Google.ClientID = req.Google.ClientID
		if req.Google.ClientSecret != "" {
			s.Config.OAuth.Google.ClientSecret = req.Google.ClientSecret
		}
		s.Config.OAuth.Google.AllowedUsers = req.Google.AllowedUsers
	}

	// Update OIDC providers
	if req.OIDC != nil {
		// Build new OIDC list
		oidcProviders := make([]*OIDCProvider, 0, len(req.OIDC))
		for i, oidcReq := range req.OIDC {
			provider := &OIDCProvider{
				Enabled:       oidcReq.Enabled,
				Name:          oidcReq.Name,
				Issuer:        oidcReq.Issuer,
				ClientID:      oidcReq.ClientID,
				Scopes:        oidcReq.Scopes,
				AllowedUsers:  oidcReq.AllowedUsers,
				AllowedGroups: oidcReq.AllowedGroups,
				UsernameClaim: oidcReq.UsernameClaim,
			}
			// Preserve existing secret if not provided
			if oidcReq.ClientSecret != "" {
				provider.ClientSecret = oidcReq.ClientSecret
			} else if s.Config.OAuth.OIDC != nil && i < len(s.Config.OAuth.OIDC) {
				provider.ClientSecret = s.Config.OAuth.OIDC[i].ClientSecret
			}
			oidcProviders = append(oidcProviders, provider)
		}
		s.Config.OAuth.OIDC = oidcProviders
		// Clear discovery cache for updated providers
		oidcDiscoveryCacheMu.Lock()
		oidcDiscoveryCache = make(map[string]*OIDCDiscovery)
		oidcDiscoveryCacheMu.Unlock()
	}

	// Update Cloudflare Access
	if req.CloudflareAccess != nil {
		if s.Config.OAuth.CloudflareAccess == nil {
			s.Config.OAuth.CloudflareAccess = &CloudflareAccessConfig{}
		}
		s.Config.OAuth.CloudflareAccess.Enabled = req.CloudflareAccess.Enabled
		s.Config.OAuth.CloudflareAccess.TeamDomain = req.CloudflareAccess.TeamDomain
		s.Config.OAuth.CloudflareAccess.AUD = req.CloudflareAccess.AUD
		s.Config.OAuth.CloudflareAccess.AllowedUsers = req.CloudflareAccess.AllowedUsers
	}

	SaveConfig(s.Config)

	LogAuditFromContext(c, AuditActionOAuthSettingsUpdate, AuditCategorySettings, "settings", "oauth", "OAuth Settings", "OAuth settings updated")

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// GitHub OAuth handlers
func (s *AppState) GitHubOAuthStart(c *gin.Context) {
	s.ConfigMu.RLock()
	oauth := s.Config.OAuth
	s.ConfigMu.RUnlock()

	if oauth == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth not configured"})
		return
	}

	state := uuid.New().String()

	oauthStatesMu.Lock()
	oauthStates[state] = &OAuthStateData{
		Provider:  "github",
		State:     state,
		CreatedAt: time.Now().Unix(),
	}
	oauthStatesMu.Unlock()

	// Clean up old states (older than 10 minutes)
	go cleanupOAuthStates()

	var authURL string

	if oauth.UseCentralized {
		// Use centralized OAuth proxy
		callbackURL := getCallbackURL(c, "proxy")
		authURL = fmt.Sprintf(
			"%s/oauth/github?redirect_uri=%s&state=%s",
			CentralizedOAuthURL,
			url.QueryEscape(callbackURL),
			state,
		)
	} else {
		// Self-hosted OAuth
		if oauth.GitHub == nil || !oauth.GitHub.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "GitHub OAuth not configured"})
			return
		}
		authURL = fmt.Sprintf(
			"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=read:user user:email&state=%s",
			oauth.GitHub.ClientID,
			url.QueryEscape(getCallbackURL(c, "github")),
			state,
		)
	}

	c.JSON(http.StatusOK, gin.H{"url": authURL})
}

func (s *AppState) GitHubOAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		redirectWithError(c, "Missing code or state parameter")
		return
	}

	// Verify state
	oauthStatesMu.Lock()
	stateData, exists := oauthStates[state]
	if exists {
		delete(oauthStates, state)
	}
	oauthStatesMu.Unlock()

	if !exists || stateData.Provider != "github" {
		redirectWithError(c, "Invalid state parameter")
		return
	}

	s.ConfigMu.RLock()
	oauth := s.Config.OAuth
	s.ConfigMu.RUnlock()

	if oauth == nil || oauth.GitHub == nil {
		redirectWithError(c, "GitHub OAuth not configured")
		return
	}

	// Exchange code for token
	tokenResp, err := exchangeGitHubCode(code, oauth.GitHub.ClientID, oauth.GitHub.ClientSecret, getCallbackURL(c, "github"))
	if err != nil {
		redirectWithError(c, "Failed to exchange code: "+err.Error())
		return
	}

	// Get user info
	user, err := getGitHubUser(tokenResp.AccessToken)
	if err != nil {
		redirectWithError(c, "Failed to get user info: "+err.Error())
		return
	}

	// Check if user is allowed
	if !isUserAllowed(oauth.GitHub.AllowedUsers, user.Login) {
		redirectWithError(c, "User not authorized: "+user.Login)
		return
	}

	// Generate JWT token
	token, expiresAt, err := generateJWTToken(user.Login, "github")
	if err != nil {
		redirectWithError(c, "Failed to generate token")
		return
	}

	// Redirect to frontend with token
	redirectWithToken(c, token, expiresAt, "github", user.Login)
}

// Google OAuth handlers
func (s *AppState) GoogleOAuthStart(c *gin.Context) {
	s.ConfigMu.RLock()
	oauth := s.Config.OAuth
	s.ConfigMu.RUnlock()

	if oauth == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth not configured"})
		return
	}

	state := uuid.New().String()

	oauthStatesMu.Lock()
	oauthStates[state] = &OAuthStateData{
		Provider:  "google",
		State:     state,
		CreatedAt: time.Now().Unix(),
	}
	oauthStatesMu.Unlock()

	go cleanupOAuthStates()

	var authURL string

	if oauth.UseCentralized {
		// Use centralized OAuth proxy
		callbackURL := getCallbackURL(c, "proxy")
		authURL = fmt.Sprintf(
			"%s/oauth/google?redirect_uri=%s&state=%s",
			CentralizedOAuthURL,
			url.QueryEscape(callbackURL),
			state,
		)
	} else {
		// Self-hosted OAuth
		if oauth.Google == nil || !oauth.Google.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Google OAuth not configured"})
			return
		}
		authURL = fmt.Sprintf(
			"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid email profile&state=%s&access_type=offline",
			oauth.Google.ClientID,
			url.QueryEscape(getCallbackURL(c, "google")),
			state,
		)
	}

	c.JSON(http.StatusOK, gin.H{"url": authURL})
}

func (s *AppState) GoogleOAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		redirectWithError(c, "Missing code or state parameter")
		return
	}

	// Verify state
	oauthStatesMu.Lock()
	stateData, exists := oauthStates[state]
	if exists {
		delete(oauthStates, state)
	}
	oauthStatesMu.Unlock()

	if !exists || stateData.Provider != "google" {
		redirectWithError(c, "Invalid state parameter")
		return
	}

	s.ConfigMu.RLock()
	oauth := s.Config.OAuth
	s.ConfigMu.RUnlock()

	if oauth == nil || oauth.Google == nil {
		redirectWithError(c, "Google OAuth not configured")
		return
	}

	// Exchange code for token
	tokenResp, err := exchangeGoogleCode(code, oauth.Google.ClientID, oauth.Google.ClientSecret, getCallbackURL(c, "google"))
	if err != nil {
		redirectWithError(c, "Failed to exchange code: "+err.Error())
		return
	}

	// Get user info
	user, err := getGoogleUser(tokenResp.AccessToken)
	if err != nil {
		redirectWithError(c, "Failed to get user info: "+err.Error())
		return
	}

	// Check if user is allowed
	if !isUserAllowed(oauth.Google.AllowedUsers, user.Email) {
		redirectWithError(c, "User not authorized: "+user.Email)
		return
	}

	// Generate JWT token
	token, expiresAt, err := generateJWTToken(user.Email, "google")
	if err != nil {
		redirectWithError(c, "Failed to generate token")
		return
	}

	// Redirect to frontend with token
	redirectWithToken(c, token, expiresAt, "google", user.Email)
}

// ProxyOAuthCallback handles OAuth callback from centralized OAuth proxy (vstats.zsoft.cc)
func (s *AppState) ProxyOAuthCallback(c *gin.Context) {
	state := c.Query("state")
	provider := c.Query("provider")
	user := c.Query("user")
	errorMsg := c.Query("error")

	// If there's an error from the proxy
	if errorMsg != "" {
		redirectWithError(c, errorMsg)
		return
	}

	if state == "" || provider == "" || user == "" {
		redirectWithError(c, "Missing required parameters")
		return
	}

	// Verify state
	oauthStatesMu.Lock()
	stateData, exists := oauthStates[state]
	if exists {
		delete(oauthStates, state)
	}
	oauthStatesMu.Unlock()

	if !exists {
		redirectWithError(c, "Invalid or expired state parameter")
		return
	}

	// Verify the provider matches what we initiated
	expectedProvider := stateData.Provider
	if provider != expectedProvider {
		redirectWithError(c, "Provider mismatch")
		return
	}

	// Check if user is allowed
	s.ConfigMu.RLock()
	oauth := s.Config.OAuth
	s.ConfigMu.RUnlock()

	if oauth == nil {
		redirectWithError(c, "OAuth not configured")
		return
	}

	// Check allowed users (from centralized config)
	if !isUserAllowed(oauth.AllowedUsers, user) {
		redirectWithError(c, "User not authorized: "+user)
		return
	}

	// Generate JWT token
	token, expiresAt, err := generateJWTToken(user, provider)
	if err != nil {
		redirectWithError(c, "Failed to generate token")
		return
	}

	// Redirect to frontend with token
	redirectWithToken(c, token, expiresAt, provider, user)
}

// ============================================================================
// OAuth Helper Functions
// ============================================================================

func getCallbackURL(c *gin.Context, provider string) string {
	protocol := "https"

	// Priority: X-Forwarded-Proto header > TLS detection > localhost fallback
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		// Trust the X-Forwarded-Proto header from nginx
		protocol = proto
	} else if c.Request.TLS != nil {
		// Direct TLS connection
		protocol = "https"
	} else if strings.Contains(c.Request.Host, "localhost") || strings.HasPrefix(c.Request.Host, "127.") {
		// Localhost fallback
		protocol = "http"
	}

	return fmt.Sprintf("%s://%s/api/auth/oauth/%s/callback", protocol, c.Request.Host, provider)
}

func exchangeGitHubCode(code, clientID, clientSecret, redirectURI string) (*GitHubTokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp GitHubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	return &tokenResp, nil
}

func getGitHubUser(accessToken string) (*GitHubUser, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func exchangeGoogleCode(code, clientID, clientSecret, redirectURI string) (*GoogleTokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, _ := http.NewRequest("POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp GoogleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	return &tokenResp, nil
}

func getGoogleUser(accessToken string) (*GoogleUserInfo, error) {
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func isUserAllowed(allowedUsers []string, identifier string) bool {
	// If no allowed users specified, deny all users
	if len(allowedUsers) == 0 {
		return false
	}

	for _, u := range allowedUsers {
		if strings.EqualFold(u, identifier) {
			return true
		}
	}
	return false
}

func generateJWTToken(sub, provider string) (string, time.Time, error) {
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      sub,
		"provider": provider,
		"exp":      expiresAt.Unix(),
	})

	tokenString, err := token.SignedString([]byte(GetJWTSecret()))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func redirectWithToken(c *gin.Context, token string, expiresAt time.Time, provider, username string) {
	// Redirect to frontend OAuth callback page
	redirectURL := fmt.Sprintf("/oauth-callback?token=%s&expires=%d&provider=%s&user=%s",
		url.QueryEscape(token),
		expiresAt.Unix(),
		provider,
		url.QueryEscape(username),
	)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func redirectWithError(c *gin.Context, message string) {
	redirectURL := fmt.Sprintf("/oauth-callback?error=%s", url.QueryEscape(message))
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func cleanupOAuthStates() {
	oauthStatesMu.Lock()
	defer oauthStatesMu.Unlock()

	now := time.Now().Unix()
	for state, data := range oauthStates {
		// Remove states older than 10 minutes
		if now-data.CreatedAt > 600 {
			delete(oauthStates, state)
		}
	}
}

// ============================================================================
// Generic OIDC Provider Handlers
// ============================================================================

// OIDC discovery cache
var (
	oidcDiscoveryCache   = make(map[string]*OIDCDiscovery)
	oidcDiscoveryCacheMu sync.RWMutex
)

// getOIDCDiscovery fetches and caches OIDC discovery document
func getOIDCDiscovery(issuer string) (*OIDCDiscovery, error) {
	// Check cache first
	oidcDiscoveryCacheMu.RLock()
	if cached, ok := oidcDiscoveryCache[issuer]; ok {
		oidcDiscoveryCacheMu.RUnlock()
		return cached, nil
	}
	oidcDiscoveryCacheMu.RUnlock()

	// Fetch discovery document
	discoveryURL := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OIDC discovery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery returned status %d", resp.StatusCode)
	}

	var discovery OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, fmt.Errorf("failed to parse OIDC discovery: %w", err)
	}

	// Cache the result
	oidcDiscoveryCacheMu.Lock()
	oidcDiscoveryCache[issuer] = &discovery
	oidcDiscoveryCacheMu.Unlock()

	return &discovery, nil
}

// GetOIDCProviders returns available OIDC providers
func (s *AppState) GetOIDCProviders(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	providers := []gin.H{}

	if s.Config.OAuth != nil && s.Config.OAuth.OIDC != nil {
		for i, oidc := range s.Config.OAuth.OIDC {
			if oidc.Enabled && oidc.ClientID != "" && oidc.Issuer != "" {
				providers = append(providers, gin.H{
					"id":   fmt.Sprintf("oidc_%d", i),
					"name": oidc.Name,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

// OIDCOAuthStart initiates OIDC authentication
func (s *AppState) OIDCOAuthStart(c *gin.Context) {
	providerID := c.Param("provider_id")

	s.ConfigMu.RLock()
	oauth := s.Config.OAuth
	s.ConfigMu.RUnlock()

	if oauth == nil || oauth.OIDC == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OIDC not configured"})
		return
	}

	// Parse provider ID (format: oidc_0, oidc_1, etc.)
	var idx int
	if _, err := fmt.Sscanf(providerID, "oidc_%d", &idx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	if idx < 0 || idx >= len(oauth.OIDC) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider not found"})
		return
	}

	provider := oauth.OIDC[idx]
	if !provider.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider not enabled"})
		return
	}

	// Get OIDC discovery
	discovery, err := getOIDCDiscovery(provider.Issuer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get OIDC configuration: " + err.Error()})
		return
	}

	state := uuid.New().String()

	oauthStatesMu.Lock()
	oauthStates[state] = &OAuthStateData{
		Provider:  providerID,
		State:     state,
		CreatedAt: time.Now().Unix(),
	}
	oauthStatesMu.Unlock()

	go cleanupOAuthStates()

	// Build scopes
	scopes := provider.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	authURL := fmt.Sprintf(
		"%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		discovery.AuthorizationEndpoint,
		url.QueryEscape(provider.ClientID),
		url.QueryEscape(getCallbackURL(c, "oidc/"+providerID)),
		url.QueryEscape(strings.Join(scopes, " ")),
		state,
	)

	c.JSON(http.StatusOK, gin.H{"url": authURL})
}

// OIDCOAuthCallback handles OIDC callback
func (s *AppState) OIDCOAuthCallback(c *gin.Context) {
	providerID := c.Param("provider_id")
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		redirectWithError(c, "Missing code or state parameter")
		return
	}

	// Verify state
	oauthStatesMu.Lock()
	stateData, exists := oauthStates[state]
	if exists {
		delete(oauthStates, state)
	}
	oauthStatesMu.Unlock()

	if !exists || stateData.Provider != providerID {
		redirectWithError(c, "Invalid state parameter")
		return
	}

	s.ConfigMu.RLock()
	oauth := s.Config.OAuth
	s.ConfigMu.RUnlock()

	if oauth == nil || oauth.OIDC == nil {
		redirectWithError(c, "OIDC not configured")
		return
	}

	// Parse provider ID
	var idx int
	if _, err := fmt.Sscanf(providerID, "oidc_%d", &idx); err != nil {
		redirectWithError(c, "Invalid provider ID")
		return
	}

	if idx < 0 || idx >= len(oauth.OIDC) {
		redirectWithError(c, "Provider not found")
		return
	}

	provider := oauth.OIDC[idx]

	// Get OIDC discovery
	discovery, err := getOIDCDiscovery(provider.Issuer)
	if err != nil {
		redirectWithError(c, "Failed to get OIDC configuration")
		return
	}

	// Exchange code for token
	tokenResp, err := exchangeOIDCCode(code, provider, discovery, getCallbackURL(c, "oidc/"+providerID))
	if err != nil {
		redirectWithError(c, "Failed to exchange code: "+err.Error())
		return
	}

	// Get user info
	userInfo, err := getOIDCUserInfo(tokenResp.AccessToken, discovery)
	if err != nil {
		redirectWithError(c, "Failed to get user info: "+err.Error())
		return
	}

	// Determine username based on claim configuration
	username := userInfo.Email
	if provider.UsernameClaim != "" {
		switch provider.UsernameClaim {
		case "sub":
			username = userInfo.Sub
		case "preferred_username":
			if userInfo.PreferredUsername != "" {
				username = userInfo.PreferredUsername
			}
		case "name":
			if userInfo.Name != "" {
				username = userInfo.Name
			}
		}
	}

	if username == "" {
		redirectWithError(c, "Could not determine username from OIDC claims")
		return
	}

	// Check if user is allowed (by email or groups)
	allowed := false
	if len(provider.AllowedUsers) > 0 {
		allowed = isUserAllowed(provider.AllowedUsers, username) ||
			isUserAllowed(provider.AllowedUsers, userInfo.Email) ||
			isUserAllowed(provider.AllowedUsers, userInfo.Sub)
	}
	if !allowed && len(provider.AllowedGroups) > 0 {
		for _, group := range userInfo.Groups {
			if isUserAllowed(provider.AllowedGroups, group) {
				allowed = true
				break
			}
		}
	}
	if !allowed && len(provider.AllowedUsers) == 0 && len(provider.AllowedGroups) == 0 {
		// No restrictions configured - deny by default
		redirectWithError(c, "No allowed users or groups configured")
		return
	}
	if !allowed {
		redirectWithError(c, "User not authorized: "+username)
		return
	}

	// Generate JWT token
	token, expiresAt, err := generateJWTToken(username, providerID)
	if err != nil {
		redirectWithError(c, "Failed to generate token")
		return
	}

	// Redirect to frontend with token
	redirectWithToken(c, token, expiresAt, providerID, username)
}

func exchangeOIDCCode(code string, provider *OIDCProvider, discovery *OIDCDiscovery, redirectURI string) (*OIDCTokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", provider.ClientID)
	data.Set("client_secret", provider.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, _ := http.NewRequest("POST", discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp OIDCTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	return &tokenResp, nil
}

func getOIDCUserInfo(accessToken string, discovery *OIDCDiscovery) (*OIDCUserInfo, error) {
	req, _ := http.NewRequest("GET", discovery.UserinfoEndpoint, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo OIDCUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// ============================================================================
// Cloudflare Access Handlers
// ============================================================================

// CloudflareAccessStart initiates Cloudflare Access authentication
// Note: Cloudflare Access works differently - it protects the entire app or specific paths
// Users are redirected to Cloudflare Access login automatically
func (s *AppState) CloudflareAccessStart(c *gin.Context) {
	s.ConfigMu.RLock()
	oauth := s.Config.OAuth
	s.ConfigMu.RUnlock()

	if oauth == nil || oauth.CloudflareAccess == nil || !oauth.CloudflareAccess.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cloudflare Access not configured"})
		return
	}

	// Generate state for CSRF protection
	state := uuid.New().String()

	oauthStatesMu.Lock()
	oauthStates[state] = &OAuthStateData{
		Provider:  "cloudflare",
		State:     state,
		CreatedAt: time.Now().Unix(),
	}
	oauthStatesMu.Unlock()

	go cleanupOAuthStates()

	// Redirect to a special endpoint that Cloudflare Access protects
	// The callback will verify the CF-Access-JWT-Assertion header
	callbackURL := getCallbackURL(c, "cloudflare")
	authURL := fmt.Sprintf("%s?state=%s", callbackURL, state)

	c.JSON(http.StatusOK, gin.H{"url": authURL})
}

// CloudflareAccessCallback handles Cloudflare Access callback
func (s *AppState) CloudflareAccessCallback(c *gin.Context) {
	state := c.Query("state")

	// Verify state if present (optional for CF Access)
	if state != "" {
		oauthStatesMu.Lock()
		stateData, exists := oauthStates[state]
		if exists {
			delete(oauthStates, state)
		}
		oauthStatesMu.Unlock()

		if !exists || stateData.Provider != "cloudflare" {
			redirectWithError(c, "Invalid state parameter")
			return
		}
	}

	s.ConfigMu.RLock()
	oauth := s.Config.OAuth
	s.ConfigMu.RUnlock()

	if oauth == nil || oauth.CloudflareAccess == nil || !oauth.CloudflareAccess.Enabled {
		redirectWithError(c, "Cloudflare Access not configured")
		return
	}

	cfConfig := oauth.CloudflareAccess

	// Get the Cloudflare Access JWT from headers
	cfJWT := c.GetHeader("Cf-Access-Jwt-Assertion")
	if cfJWT == "" {
		// Also check cookie
		cookie, err := c.Cookie("CF_Authorization")
		if err != nil || cookie == "" {
			redirectWithError(c, "No Cloudflare Access token found. Please access this page through Cloudflare Access.")
			return
		}
		cfJWT = cookie
	}

	// Verify the JWT
	claims, err := verifyClouflareAccessJWT(cfJWT, cfConfig)
	if err != nil {
		redirectWithError(c, "Invalid Cloudflare Access token: "+err.Error())
		return
	}

	email := claims.Email
	if email == "" && claims.Identity.Email != "" {
		email = claims.Identity.Email
	}

	if email == "" {
		redirectWithError(c, "No email found in Cloudflare Access token")
		return
	}

	// Check if user is allowed
	if len(cfConfig.AllowedUsers) > 0 && !isUserAllowed(cfConfig.AllowedUsers, email) {
		redirectWithError(c, "User not authorized: "+email)
		return
	}

	// Generate JWT token
	token, expiresAt, err := generateJWTToken(email, "cloudflare")
	if err != nil {
		redirectWithError(c, "Failed to generate token")
		return
	}

	// Redirect to frontend with token
	redirectWithToken(c, token, expiresAt, "cloudflare", email)
}

// verifyClouflareAccessJWT verifies Cloudflare Access JWT
func verifyClouflareAccessJWT(tokenString string, config *CloudflareAccessConfig) (*CloudflareAccessClaims, error) {
	// Fetch Cloudflare's public keys
	certsURL := fmt.Sprintf("https://%s/cdn-cgi/access/certs", config.TeamDomain)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(certsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Cloudflare certs: %w", err)
	}
	defer resp.Body.Close()

	var certsResp struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&certsResp); err != nil {
		return nil, fmt.Errorf("failed to parse Cloudflare certs: %w", err)
	}

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get kid from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("no kid in token header")
		}

		// Find matching key
		for _, keyData := range certsResp.Keys {
			var keyInfo struct {
				Kid string `json:"kid"`
				N   string `json:"n"`
				E   string `json:"e"`
			}
			if err := json.Unmarshal(keyData, &keyInfo); err != nil {
				continue
			}
			if keyInfo.Kid == kid {
				// Parse RSA public key
				return parseRSAPublicKey(keyInfo.N, keyInfo.E)
			}
		}
		return nil, fmt.Errorf("no matching key found for kid: %s", kid)
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	// Verify audience
	aud, _ := mapClaims["aud"].([]interface{})
	audMatch := false
	for _, a := range aud {
		if aStr, ok := a.(string); ok && aStr == config.AUD {
			audMatch = true
			break
		}
	}
	if !audMatch {
		// Also check if aud is a string
		if audStr, ok := mapClaims["aud"].(string); ok && audStr == config.AUD {
			audMatch = true
		}
	}
	if !audMatch {
		return nil, fmt.Errorf("audience mismatch")
	}

	// Convert to CloudflareAccessClaims
	claims := &CloudflareAccessClaims{}
	claims.Email, _ = mapClaims["email"].(string)
	claims.Type, _ = mapClaims["type"].(string)

	// Parse identity if present
	if identity, ok := mapClaims["identity"].(map[string]interface{}); ok {
		claims.Identity.ID, _ = identity["id"].(string)
		claims.Identity.Name, _ = identity["name"].(string)
		claims.Identity.Email, _ = identity["email"].(string)
	}

	return claims, nil
}

// parseRSAPublicKey parses RSA public key from JWK n and e values
func parseRSAPublicKey(nStr, eStr string) (interface{}, error) {
	// Decode n (modulus)
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode e (exponent)
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}
	var e int
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

// ============================================================================
// SSO Binding Management
// ============================================================================

// GetSSOBindings returns current SSO bindings
func (s *AppState) GetSSOBindings(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	bindings := []SSOBinding{}
	if s.Config.OAuth != nil {
		bindings = s.Config.OAuth.Bindings
	}

	c.JSON(http.StatusOK, gin.H{"bindings": bindings})
}

// AddSSOBinding adds a new SSO binding
func (s *AppState) AddSSOBinding(c *gin.Context) {
	var req struct {
		Provider   string `json:"provider" binding:"required"`
		ProviderID string `json:"provider_id"`
		Identifier string `json:"identifier" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	s.ConfigMu.Lock()
	defer s.ConfigMu.Unlock()

	if s.Config.OAuth == nil {
		s.Config.OAuth = &OAuthConfig{}
	}

	// Check if binding already exists
	for _, b := range s.Config.OAuth.Bindings {
		if b.Provider == req.Provider && b.Identifier == req.Identifier {
			c.JSON(http.StatusConflict, gin.H{"error": "Binding already exists"})
			return
		}
	}

	binding := SSOBinding{
		Provider:   req.Provider,
		ProviderID: req.ProviderID,
		Identifier: req.Identifier,
		BoundAt:    time.Now().UTC().Format(time.RFC3339),
	}

	s.Config.OAuth.Bindings = append(s.Config.OAuth.Bindings, binding)
	SaveConfig(s.Config)

	LogAuditFromContext(c, "sso_binding_add", AuditCategoryAuth, "binding", req.Provider, req.Identifier, "SSO binding added")

	c.JSON(http.StatusOK, gin.H{"status": "added", "binding": binding})
}

// DeleteSSOBinding removes an SSO binding
func (s *AppState) DeleteSSOBinding(c *gin.Context) {
	provider := c.Param("provider")
	identifier := c.Query("identifier")

	if provider == "" || identifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider and identifier required"})
		return
	}

	s.ConfigMu.Lock()
	defer s.ConfigMu.Unlock()

	if s.Config.OAuth == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No bindings found"})
		return
	}

	found := false
	newBindings := []SSOBinding{}
	for _, b := range s.Config.OAuth.Bindings {
		if b.Provider == provider && b.Identifier == identifier {
			found = true
			continue
		}
		newBindings = append(newBindings, b)
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Binding not found"})
		return
	}

	s.Config.OAuth.Bindings = newBindings
	SaveConfig(s.Config)

	LogAuditFromContext(c, "sso_binding_delete", AuditCategoryAuth, "binding", provider, identifier, "SSO binding removed")

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// GetOAuthProvidersExtended returns all available OAuth providers including OIDC and Cloudflare
func (s *AppState) GetOAuthProvidersExtended(c *gin.Context) {
	s.ConfigMu.RLock()
	defer s.ConfigMu.RUnlock()

	providers := make(map[string]interface{})
	centralized := false

	if s.Config.OAuth != nil {
		// Check centralized OAuth first
		if s.Config.OAuth.UseCentralized {
			centralized = true
			providers["github"] = true
			providers["google"] = true
		} else {
			// Self-hosted OAuth
			if s.Config.OAuth.GitHub != nil && s.Config.OAuth.GitHub.Enabled && s.Config.OAuth.GitHub.ClientID != "" {
				providers["github"] = true
			}
			if s.Config.OAuth.Google != nil && s.Config.OAuth.Google.Enabled && s.Config.OAuth.Google.ClientID != "" {
				providers["google"] = true
			}
		}

		// OIDC providers
		if s.Config.OAuth.OIDC != nil {
			oidcProviders := []gin.H{}
			for i, oidc := range s.Config.OAuth.OIDC {
				if oidc.Enabled && oidc.ClientID != "" && oidc.Issuer != "" {
					oidcProviders = append(oidcProviders, gin.H{
						"id":   fmt.Sprintf("oidc_%d", i),
						"name": oidc.Name,
					})
				}
			}
			if len(oidcProviders) > 0 {
				providers["oidc"] = oidcProviders
			}
		}

		// Cloudflare Access
		if s.Config.OAuth.CloudflareAccess != nil && s.Config.OAuth.CloudflareAccess.Enabled {
			providers["cloudflare"] = true
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"providers":   providers,
		"centralized": centralized,
	})
}
