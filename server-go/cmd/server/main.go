package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Enable pprof
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Version will be set at build time via -ldflags
var ServerVersion = "dev"

func main() {
	// Check for command line arguments
	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "version", "--version", "-v":
			fmt.Printf("vstats-server version %s\n", ServerVersion)
			os.Exit(0)
		case "--check":
			showDiagnostics()
			return
		case "--reset-password":
			password := ResetAdminPassword()
			fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
			fmt.Println("║                    🔑 PASSWORD RESET                           ║")
			fmt.Println("╠════════════════════════════════════════════════════════════════╣")
			fmt.Printf("║  New admin password: %-40s ║\n", password)
			fmt.Printf("║  Config file: %-47s ║\n", GetConfigPath())
			fmt.Println("╚════════════════════════════════════════════════════════════════╝")

			// Try to signal running server to reload config
			if err := findAndSignalServer(); err != nil {
				fmt.Printf("\n⚠️  %v\n", err)

				// Provide specific help based on error type
				if sigErr, ok := err.(*SignalError); ok {
					switch sigErr.Type {
					case "permission_denied":
						fmt.Println("\n💡 The server is running but you don't have permission to signal it.")
						fmt.Println("   Try one of the following:")
						fmt.Printf("     1. Run with sudo: sudo %s --reset-password\n", os.Args[0])
						fmt.Println("     2. Restart the service: sudo systemctl restart vstats")
					case "not_found":
						fmt.Println("\n💡 No running server found. The new password will take effect")
						fmt.Println("   when the server starts.")
					default:
						// Check if Windows (SignalError message contains "Windows")
						if strings.Contains(sigErr.Message, "Windows") {
							fmt.Println("\n💡 Please restart the server manually:")
							fmt.Println("     - Stop the service: sc stop vstats")
							fmt.Println("     - Start the service: sc start vstats")
							fmt.Println("   Or restart from Services management console.")
						} else {
							fmt.Println("\n💡 Please restart the server manually:")
							fmt.Println("     systemctl restart vstats")
						}
					}
				} else {
					fmt.Println("   If server is running, please restart it manually:")
					fmt.Println("     systemctl restart vstats")
				}
			} else {
				fmt.Println("\n✅ Server has been notified to reload the new password.")
			}
			return
		}
	}

	// Initialize database
	db, err := InitDatabase()
	if err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize the database writer for serialized writes
	// With batch buffers, only a few write jobs per second, so 100 is plenty
	dbWriter = NewDBWriter(db, 100)
	defer dbWriter.Close()

	// Initialize metrics buffer for batched real-time metrics writes
	// Flush every 1 second or when buffer reaches 1000 items
	metricsBuffer = NewMetricsBuffer(1*time.Second, 1000)
	defer metricsBuffer.Close()

	// Initialize aggregation buffer for batched writes (flush every 1 second)
	aggBuffer = NewAggBuffer(1 * time.Second)
	defer aggBuffer.Close()
	fmt.Println("📊 Batch write buffers initialized (flush every 1s, supports 3000+ agents)")

	// Initialize history cache with 10 second TTL
	InitHistoryCache(10 * time.Second)

	fmt.Printf("📦 Database initialized: %s\n", GetDBPath())
	fmt.Printf("⚙️  Config file: %s\n", GetConfigPath())

	// Load config
	config, initialPassword := LoadConfig()
	if initialPassword != nil {
		fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
		fmt.Println("║              🎉 FIRST RUN - SAVE YOUR PASSWORD!               ║")
		fmt.Println("╠════════════════════════════════════════════════════════════════╣")
		fmt.Printf("║  Admin password: %-44s ║\n", *initialPassword)
		fmt.Println("║                                                                ║")
		fmt.Println("║  ⚠️  Save this password! It won't be shown again.              ║")
		fmt.Println("║  To reset: sudo /opt/vstats/vstats-server --reset-password     ║")
		fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	}

	// Create app state
	state := &AppState{
		Config:           config,
		MetricsBroadcast: make(chan string, 16),
		AgentMetrics:     make(map[string]*AgentMetricsData),
		AgentConns:       make(map[string]*AgentConnection),
		LastSent: &LastSentState{
			Servers: make(map[string]*struct {
				Online  bool
				Metrics *CompactMetrics
			}),
		},
		DashboardClients: make(map[*websocket.Conn]*DashboardClient),
		DB:               db,
	}

	// Initialize local metrics collector with ping targets
	localCollector := GetLocalCollector()
	if len(config.ProbeSettings.PingTargets) > 0 {
		localCollector.SetPingTargets(config.ProbeSettings.PingTargets)
		fmt.Printf("📡 Ping targets configured: %d targets\n", len(config.ProbeSettings.PingTargets))
	}

	// Setup signal handler for config reload (SIGHUP)
	SetupSignalHandler(state)

	// Start pprof server for profiling (only in dev or when VSTATS_PPROF is set)
	if os.Getenv("VSTATS_PPROF") != "" {
		pprofAddr := os.Getenv("VSTATS_PPROF")
		if pprofAddr == "1" || pprofAddr == "true" {
			pprofAddr = ":6060"
		}
		go func() {
			fmt.Printf("🔬 pprof server listening on %s\n", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				fmt.Printf("⚠️  pprof server error: %v\n", err)
			}
		}()
	}

	// Check version on startup
	CheckVersionOnStartup()

	// Start background tasks
	go metricsBroadcastLoop(state) // Combined: refresh snapshot + broadcast delta updates
	// NOTE: aggregation15MinLoop and aggregationLoop removed - aggregation now done on agent side
	go cleanupLoop(db)
	go StartVersionCheckLoop(state) // Check for version updates periodically

	// Start traffic manager
	trafficManager = NewTrafficManager(state, db)
	trafficManager.Start()
	defer trafficManager.Stop()

	// Start alert engine if enabled
	alertEngine = NewAlertEngine(state, db)
	alertEngine.Start()
	defer alertEngine.Stop()

	// Initialize GeoIP service
	geoipService := GetGeoIPService()
	if err := geoipService.Initialize(config.GeoIPConfig); err != nil {
		fmt.Printf("⚠️ GeoIP initialization warning: %v\n", err)
	} else {
		if geoipService.IsMMDBLoaded() {
			fmt.Println("🌍 GeoIP service initialized with MMDB database")
		} else {
			fmt.Println("🌍 GeoIP service initialized (API fallback mode)")
		}
	}
	defer geoipService.Close()

	// Start GeoIP auto-update if enabled
	if config.GeoIPConfig != nil && config.GeoIPConfig.AutoUpdate {
		go geoIPAutoUpdateLoop(state)
	}

	// Setup routes
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Trust proxy headers (for X-Forwarded-Proto, X-Forwarded-For, etc.)
	// This allows the app to correctly detect HTTPS when behind nginx
	r.SetTrustedProxies([]string{"127.0.0.1", "::1"}) // Trust localhost proxies
	// Also trust all proxies if VSTATS_TRUST_ALL_PROXIES is set
	if os.Getenv("VSTATS_TRUST_ALL_PROXIES") == "true" {
		r.SetTrustedProxies(nil) // nil means trust all proxies
	}

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "*")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Public routes
	r.GET("/health", HealthCheck)
	r.GET("/api/metrics", state.GetMetrics)
	r.GET("/api/metrics/all", state.GetAllMetrics)
	r.GET("/api/online-users", state.GetOnlineUsers)
	r.GET("/api/history/:server_id", func(c *gin.Context) {
		state.GetHistory(c, db)
	})
	r.GET("/api/servers", state.GetServers)
	r.GET("/api/groups", state.GetGroups)
	r.GET("/api/dimensions", state.GetDimensions) // Public: get all dimensions for grouping
	r.GET("/api/settings/site", state.GetSiteSettings)
	r.GET("/api/wallpaper/bing", GetBingWallpaper)
	r.GET("/api/wallpaper/unsplash", GetUnsplashWallpaper)
	r.GET("/api/wallpaper/proxy", GetCustomWallpaper)
	r.GET("/api/wallpaper/proxy/image", GetCustomWallpaperImage)
	r.POST("/api/auth/login", state.Login)
	r.GET("/api/auth/verify", AuthMiddleware(), state.VerifyToken)

	// OAuth 2.0 routes (public)
	r.GET("/api/auth/oauth/providers", state.GetOAuthProvidersExtended) // Extended version with OIDC/CF support
	r.GET("/api/auth/oauth/github", state.GitHubOAuthStart)
	r.GET("/api/auth/oauth/github/callback", state.GitHubOAuthCallback)
	r.GET("/api/auth/oauth/google", state.GoogleOAuthStart)
	r.GET("/api/auth/oauth/google/callback", state.GoogleOAuthCallback)
	r.GET("/api/auth/oauth/proxy/callback", state.ProxyOAuthCallback) // Centralized OAuth callback
	// OIDC routes
	r.GET("/api/auth/oauth/oidc/providers", state.GetOIDCProviders)
	r.GET("/api/auth/oauth/oidc/:provider_id", state.OIDCOAuthStart)
	r.GET("/api/auth/oauth/oidc/:provider_id/callback", state.OIDCOAuthCallback)
	// Cloudflare Access routes
	r.GET("/api/auth/oauth/cloudflare", state.CloudflareAccessStart)
	r.GET("/api/auth/oauth/cloudflare/callback", state.CloudflareAccessCallback)
	r.GET("/api/install-command", AuthMiddleware(), state.GetInstallCommand)
	r.GET("/api/version", GetServerVersion)
	r.GET("/version", GetServerVersion)
	r.GET("/api/version/check", CheckLatestVersion)
	r.GET("/agent.sh", state.GetAgentScript)
	r.GET("/agent.ps1", state.GetAgentPowerShellScript)
	r.GET("/agent-upgrade.ps1", state.GetAgentUpgradePowerShellScript)
	r.GET("/agent-uninstall.ps1", state.GetAgentUninstallPowerShellScript)
	r.GET("/ws", state.HandleDashboardWS)
	r.GET("/ws/agent", state.HandleAgentWS)

	// Protected routes
	protected := r.Group("/")
	protected.Use(AuthMiddleware())
	{
		protected.POST("/api/servers", state.AddServer)
		protected.DELETE("/api/servers/:id", state.DeleteServer)
		protected.PUT("/api/servers/:id", state.UpdateServer)
		protected.POST("/api/servers/:id/update", state.UpdateAgent)
		protected.POST("/api/auth/password", state.ChangePassword)
		protected.POST("/api/agent/register", state.RegisterAgent)
		protected.PUT("/api/settings/site", state.UpdateSiteSettings)
		protected.GET("/api/settings/local-node", state.GetLocalNodeConfig)
		protected.PUT("/api/settings/local-node", state.UpdateLocalNodeConfig)
		protected.GET("/api/settings/probe", state.GetProbeSettings)
		protected.PUT("/api/settings/probe", state.UpdateProbeSettings)
		protected.POST("/api/server/upgrade", UpgradeServer)
		// OAuth settings (admin only)
		protected.GET("/api/settings/oauth", state.GetOAuthSettings)
		protected.PUT("/api/settings/oauth", state.UpdateOAuthSettings)
		// SSO binding management (admin only)
		protected.GET("/api/sso/bindings", state.GetSSOBindings)
		protected.POST("/api/sso/bindings", state.AddSSOBinding)
		protected.DELETE("/api/sso/bindings/:provider", state.DeleteSSOBinding)
		// Group management (GET is public, mutations are protected)
		protected.POST("/api/groups", state.AddGroup)
		protected.PUT("/api/groups/:id", state.UpdateGroup)
		protected.DELETE("/api/groups/:id", state.DeleteGroup)
		// Dimension management (GET is public, mutations are protected)
		protected.POST("/api/dimensions", state.AddDimension)
		protected.PUT("/api/dimensions/:id", state.UpdateDimension)
		protected.DELETE("/api/dimensions/:id", state.DeleteDimension)
		// Dimension options management
		protected.POST("/api/dimensions/:id/options", state.AddOption)
		protected.PUT("/api/dimensions/:id/options/:option_id", state.UpdateOption)
		protected.DELETE("/api/dimensions/:id/options/:option_id", state.DeleteOption)
		// Alert settings and management (admin only)
		protected.GET("/api/settings/alerts", state.GetAlertConfig)
		protected.PUT("/api/settings/alerts", state.UpdateAlertConfig)
		protected.GET("/api/alerts/channels", state.GetChannels)
		protected.POST("/api/alerts/channels", state.AddChannel)
		protected.PUT("/api/alerts/channels/:id", state.UpdateChannel)
		protected.DELETE("/api/alerts/channels/:id", state.DeleteChannel)
		protected.POST("/api/alerts/channels/test", state.TestChannel)
		protected.GET("/api/alerts", state.GetAlerts)
		protected.GET("/api/alerts/history", state.GetAlertHistory)
		protected.POST("/api/alerts/:id/mute", state.MuteAlert)
		protected.PUT("/api/alerts/rules/offline", state.UpdateOfflineRule)
		protected.PUT("/api/alerts/rules/load", state.UpdateLoadRule)
		protected.PUT("/api/alerts/rules/traffic", state.UpdateTrafficRule)
		protected.GET("/api/alerts/templates", state.GetAlertTemplates)
		protected.PUT("/api/alerts/templates/:key", state.UpdateAlertTemplate)
		// Audit log management
		protected.GET("/api/audit-logs", state.GetAuditLogs)
		protected.GET("/api/audit-logs/export", state.ExportAuditLogs)
		protected.GET("/api/audit-logs/stats", state.GetAuditLogStats)
		protected.GET("/api/settings/audit-log", state.GetAuditLogSettings)
		protected.PUT("/api/settings/audit-log", state.UpdateAuditLogSettings)
		// GeoIP management
		protected.GET("/api/settings/geoip", state.GetGeoIPConfig)
		protected.PUT("/api/settings/geoip", state.UpdateGeoIPConfig)
		protected.GET("/api/geoip/lookup", state.LookupGeoIP)
		protected.POST("/api/geoip/lookup/batch", state.LookupGeoIPBatch)
		protected.POST("/api/geoip/refresh", state.RefreshServerGeoIP)
		protected.POST("/api/geoip/cache/clear", state.ClearGeoIPCache)
		protected.GET("/api/servers/:id/geoip", state.GetServerGeoIP)
		// Asset management
		protected.GET("/api/assets/cost-statistics", state.GetCostStatistics)
		protected.GET("/api/assets/expiring", state.GetExpiringServers)
		protected.POST("/api/servers/import", state.ImportServers)
		protected.POST("/api/servers/import/csv", state.ImportServersCSV)
		protected.GET("/api/servers/export", state.ExportServers)
		protected.PUT("/api/alerts/rules/expiry", state.UpdateExpiryRule)
		// Traffic management
		protected.GET("/api/traffic/summary", state.GetTrafficSummary)
		protected.GET("/api/traffic/stats", state.GetTrafficStats)
		protected.GET("/api/traffic/stats/:server_id", state.GetServerTrafficStats)
		protected.PUT("/api/traffic/limit", state.UpdateTrafficLimit)
		protected.POST("/api/traffic/reset", state.ResetServerTraffic)
		protected.GET("/api/traffic/history/:server_id", state.GetTrafficHistory)
		protected.GET("/api/traffic/daily/:server_id", state.GetTrafficDaily)
		protected.GET("/api/traffic/limits", state.GetAllTrafficLimits)
		protected.PUT("/api/traffic/limits/batch", state.BatchUpdateTrafficLimits)
		protected.DELETE("/api/traffic/limits/:server_id", state.DeleteTrafficLimit)
	}

	// Static file serving
	webDir := getWebDir()
	if webDir != "" {
		// Serve static files from web directory
		r.Static("/assets", webDir+"/assets")
		r.Static("/logos", webDir+"/logos") // Serve logo files
		r.StaticFile("/favicon.ico", webDir+"/favicon.ico")
		r.StaticFile("/vite.svg", webDir+"/vite.svg")
		r.GET("/", func(c *gin.Context) {
			c.File(webDir + "/index.html")
		})
		r.NoRoute(func(c *gin.Context) {
			// For SPA, serve index.html for all non-API routes
			path := c.Request.URL.Path
			if !strings.HasPrefix(path, "/api") &&
				!strings.HasPrefix(path, "/ws") &&
				!strings.HasPrefix(path, "/agent.sh") &&
				!strings.HasPrefix(path, "/agent.ps1") &&
				!strings.HasPrefix(path, "/agent-upgrade.ps1") &&
				!strings.HasPrefix(path, "/agent-uninstall.ps1") &&
				!strings.HasPrefix(path, "/logos") &&
				!strings.HasPrefix(path, "/assets") {
				c.File(webDir + "/index.html")
			} else {
				c.Status(404)
			}
		})
	} else {
		// Fallback to embedded HTML
		r.NoRoute(func(c *gin.Context) {
			if c.Request.URL.Path == "/" || c.Request.URL.Path == "/index.html" {
				c.Header("Content-Type", "text/html")
				c.String(200, embeddedIndexHTML)
				return
			}
			c.Status(404)
		})
	}

	// Get port with priority: config > environment variable > default
	port := config.Port
	if port == "" {
		port = os.Getenv("VSTATS_PORT")
	}
	if port == "" {
		port = "3001"
	}

	// Get host with priority: config > environment variable > default
	host := config.Host
	if host == "" {
		host = os.Getenv("VSTATS_HOST")
	}
	if host == "" {
		host = "0.0.0.0" // Default to IPv4 all interfaces
	}

	// Check if dual-stack mode is enabled
	dualStack := config.DualStack
	if !dualStack && os.Getenv("VSTATS_DUAL_STACK") == "true" {
		dualStack = true
	}

	// Determine protocol and address format
	useTLS := config.TLS != nil && config.TLS.Enabled && config.TLS.Cert != "" && config.TLS.Key != ""
	protocol := "http"
	wsProtocol := "ws"
	if useTLS {
		protocol = "https"
		wsProtocol = "wss"
	}

	// Start server(s)
	if dualStack {
		// Dual-stack mode: listen on both IPv4 and IPv6
		fmt.Printf("🌐 Dual-stack mode enabled (IPv4 + IPv6)\n")
		
		// Verify TLS certificates if enabled
		if useTLS {
			if _, err := os.Stat(config.TLS.Cert); err != nil {
				fmt.Printf("❌ TLS certificate file not found: %s\n", config.TLS.Cert)
				os.Exit(1)
			}
			if _, err := os.Stat(config.TLS.Key); err != nil {
				fmt.Printf("❌ TLS private key file not found: %s\n", config.TLS.Key)
				os.Exit(1)
			}
			fmt.Printf("🔒 TLS enabled: cert=%s, key=%s\n", config.TLS.Cert, config.TLS.Key)
		}

		// Start IPv4 listener
		ipv4Addr := "0.0.0.0:" + port
		fmt.Printf("🚀 Server (IPv4) running on %s://0.0.0.0:%s\n", protocol, port)
		fmt.Printf("📡 Agent WebSocket (IPv4): %s://0.0.0.0:%s/ws/agent\n", wsProtocol, port)

		// Start IPv6 listener
		ipv6Addr := "[::]:" + port
		fmt.Printf("🚀 Server (IPv6) running on %s://[::]:%s\n", protocol, port)
		fmt.Printf("📡 Agent WebSocket (IPv6): %s://[::]:%s/ws/agent\n", wsProtocol, port)
		fmt.Printf("🔑 Reset password: sudo /opt/vstats/vstats-server --reset-password\n")

		// Create HTTP server
		srv := &http.Server{
			Addr:    ipv4Addr,
			Handler: r,
		}

		// Start IPv4 listener in goroutine
		go func() {
			var err error
			if useTLS {
				err = srv.ListenAndServeTLS(config.TLS.Cert, config.TLS.Key)
			} else {
				err = srv.ListenAndServe()
			}
			if err != nil && err != http.ErrServerClosed {
				fmt.Printf("❌ IPv4 listener error: %v\n", err)
				os.Exit(1)
			}
		}()

		// Start IPv6 listener in main goroutine
		srv6 := &http.Server{
			Addr:    ipv6Addr,
			Handler: r,
		}

		if useTLS {
			if err := srv6.ListenAndServeTLS(config.TLS.Cert, config.TLS.Key); err != nil {
				fmt.Printf("Failed to start IPv6 server: %v\n", err)
				os.Exit(1)
			}
		} else {
			if err := srv6.ListenAndServe(); err != nil {
				fmt.Printf("Failed to start IPv6 server: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		// Single-stack mode: listen on specified address
		// Format address for display and listen
		// IPv6 addresses need special handling (contain colons but no dots)
		isIPv6 := strings.Contains(host, ":") && !strings.Contains(host, ".")
		displayAddr := host
		listenAddr := host + ":" + port
		
		if isIPv6 {
			// Remove brackets if present for processing
			cleanHost := strings.Trim(host, "[]")
			// Format for display (with brackets)
			displayAddr = "[" + cleanHost + "]"
			// Format for listen (with brackets)
			listenAddr = "[" + cleanHost + "]:" + port
		}

		fmt.Printf("🚀 Server running on %s://%s:%s\n", protocol, displayAddr, port)
		fmt.Printf("📡 Agent WebSocket: %s://%s:%s/ws/agent\n", wsProtocol, displayAddr, port)
		fmt.Printf("🔑 Reset password: sudo /opt/vstats/vstats-server --reset-password\n")

		// Start server with TLS if configured
		if useTLS {
			// Verify certificate files exist
			if _, err := os.Stat(config.TLS.Cert); err != nil {
				fmt.Printf("❌ TLS certificate file not found: %s\n", config.TLS.Cert)
				os.Exit(1)
			}
			if _, err := os.Stat(config.TLS.Key); err != nil {
				fmt.Printf("❌ TLS private key file not found: %s\n", config.TLS.Key)
				os.Exit(1)
			}

			fmt.Printf("🔒 TLS enabled: cert=%s, key=%s\n", config.TLS.Cert, config.TLS.Key)
			if err := r.RunTLS(listenAddr, config.TLS.Cert, config.TLS.Key); err != nil {
				fmt.Printf("Failed to start server: %v\n", err)
				os.Exit(1)
			}
		} else {
			if err := r.Run(listenAddr); err != nil {
				fmt.Printf("Failed to start server: %v\n", err)
				os.Exit(1)
			}
		}
	}
}

func showDiagnostics() {
	configPath := GetConfigPath()
	dbPath := GetDBPath()

	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    🔍 DIAGNOSTICS                              ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")

	exe, _ := os.Executable()
	fmt.Printf("║  Executable: %-48s ║\n", exe)
	fmt.Printf("║  Config: %-52s ║\n", configPath)
	fmt.Printf("║  Config exists: %-45s ║\n", boolToStr(fileExists(configPath)))
	fmt.Printf("║  Database: %-50s ║\n", dbPath)
	fmt.Printf("║  Database exists: %-43s ║\n", boolToStr(fileExists(dbPath)))

	if fileExists(configPath) {
		data, err := os.ReadFile(configPath)
		if err == nil {
			var config map[string]interface{}
			if json.Unmarshal(data, &config) == nil {
				hash, _ := config["admin_password_hash"].(string)
				hasHash := hash != "" && (hash[:3] == "$2a" || hash[:3] == "$2b")
				fmt.Printf("║  Password hash valid: %-39s ║\n", boolToStr(hasHash))

				servers, _ := config["servers"].([]interface{})
				fmt.Printf("║  Servers configured: %-40d ║\n", len(servers))
			}
		}
	}

	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func metricsBroadcastLoop(state *AppState) {
	// Build initial snapshot
	state.RefreshSnapshot()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Single data collection for both snapshot and broadcast
		state.ConfigMu.RLock()
		config := state.Config
		state.ConfigMu.RUnlock()

		state.AgentMetricsMu.RLock()
		agentMetrics := make(map[string]*AgentMetricsData)
		for k, v := range state.AgentMetrics {
			agentMetrics[k] = v
		}
		state.AgentMetricsMu.RUnlock()

		// Collect local metrics ONCE
		localMetrics := CollectMetrics()

		// === Build delta updates for connected dashboards ===
		var deltaUpdates []CompactServerUpdate

		// Check local server
		localCompact := CompactMetricsFromSystem(&localMetrics)
		state.LastSentMu.Lock()
		localPrev := state.LastSent.Servers["local"]
		state.LastSentMu.Unlock()

		localChanged := localPrev == nil || localCompact.HasChanged(localPrev.Metrics)
		if localChanged {
			var diffMetrics *CompactMetrics
			if localPrev != nil {
				diffMetrics = localCompact.Diff(localPrev.Metrics)
			} else {
				diffMetrics = localCompact
			}

			if !diffMetrics.IsEmpty() {
				deltaUpdates = append(deltaUpdates, CompactServerUpdate{
					ID: "local",
					On: boolPtr(true),
					M:  diffMetrics,
				})
			}

			state.LastSentMu.Lock()
			state.LastSent.Servers["local"] = &struct {
				Online  bool
				Metrics *CompactMetrics
			}{
				Online:  true,
				Metrics: localCompact,
			}
			state.LastSentMu.Unlock()
		}

		// Check remote servers
		for _, server := range config.Servers {
			metricsData := agentMetrics[server.ID]
			online := false
			if metricsData != nil {
				online = time.Since(metricsData.LastUpdated).Seconds() < 30
			}

			currentMetrics := &CompactMetrics{}
			if metricsData != nil {
				currentMetrics = CompactMetricsFromSystem(&metricsData.Metrics)
			}

			state.LastSentMu.Lock()
			prev := state.LastSent.Servers[server.ID]
			state.LastSentMu.Unlock()

			prevOnline := false
			var prevMetrics *CompactMetrics
			if prev != nil {
				prevOnline = prev.Online
				prevMetrics = prev.Metrics
			} else {
				prevMetrics = &CompactMetrics{}
			}

			onlineChanged := online != prevOnline
			metricsChanged := online && currentMetrics.HasChanged(prevMetrics)

			if onlineChanged || metricsChanged {
				update := CompactServerUpdate{
					ID: server.ID,
				}

				if onlineChanged {
					update.On = &online
				}

				if metricsChanged && online {
					update.M = currentMetrics.Diff(prevMetrics)
				}

				if update.On != nil || (update.M != nil && !update.M.IsEmpty()) {
					deltaUpdates = append(deltaUpdates, update)
				}

				state.LastSentMu.Lock()
				state.LastSent.Servers[server.ID] = &struct {
					Online  bool
					Metrics *CompactMetrics
				}{
					Online:  online,
					Metrics: currentMetrics,
				}
				state.LastSentMu.Unlock()
			}
		}

		// Broadcast if there are changes
		if len(deltaUpdates) > 0 {
			msg := DeltaMessage{
				Type: "delta",
				Ts:   time.Now().Unix(),
				D:    deltaUpdates,
			}

			if data, err := json.Marshal(msg); err == nil {
				state.BroadcastMetrics(string(data))
			}
		}

		// === Refresh snapshot using already collected data ===
		state.RefreshSnapshotWithData(config, agentMetrics, &localMetrics)
	}
}

// NOTE: aggregation15MinLoop and aggregationLoop removed
// Aggregation is now performed on the agent side and sent to server
// This reduces server CPU load and allows agents to maintain their own historical data

func cleanupLoop(db *sql.DB) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if err := CleanupOldData(db); err != nil {
			fmt.Printf("Failed to cleanup old data: %v\n", err)
		}
		// Cleanup old audit logs (default 30 days retention)
		if err := CleanupAuditLogs(db, 30); err != nil {
			fmt.Printf("Failed to cleanup audit logs: %v\n", err)
		}
	}
}

// geoIPAutoUpdateLoop periodically updates GeoIP data for all servers
func geoIPAutoUpdateLoop(state *AppState) {
	// Initial delay to let server start up
	time.Sleep(30 * time.Second)

	// Determine update interval
	intervalHr := 24
	state.ConfigMu.RLock()
	if state.Config.GeoIPConfig != nil && state.Config.GeoIPConfig.UpdateIntervalHr > 0 {
		intervalHr = state.Config.GeoIPConfig.UpdateIntervalHr
	}
	state.ConfigMu.RUnlock()

	ticker := time.NewTicker(time.Duration(intervalHr) * time.Hour)
	defer ticker.Stop()

	// Do initial update
	updateServerGeoIP(state)

	for range ticker.C {
		// Check if auto-update is still enabled
		state.ConfigMu.RLock()
		enabled := state.Config.GeoIPConfig != nil && state.Config.GeoIPConfig.AutoUpdate
		state.ConfigMu.RUnlock()

		if !enabled {
			return
		}

		updateServerGeoIP(state)
	}
}

// updateServerGeoIP updates GeoIP data for all servers
func updateServerGeoIP(state *AppState) {
	service := GetGeoIPService()

	state.ConfigMu.Lock()
	defer state.ConfigMu.Unlock()

	updated := 0
	for i := range state.Config.Servers {
		server := &state.Config.Servers[i]
		if server.IP == "" {
			continue
		}

		result, err := service.Lookup(server.IP)
		if err != nil {
			continue
		}

		if result.CountryCode != "" {
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
			updated++
		}
	}

	if updated > 0 {
		if state.Config.GeoIPConfig != nil {
			state.Config.GeoIPConfig.LastUpdate = time.Now().Format(time.RFC3339)
		}
		SaveConfig(state.Config)
		fmt.Printf("🌍 GeoIP auto-update: updated %d servers\n", updated)
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// getWebDir finds the web directory containing the frontend assets
func getWebDir() string {
	// Check VSTATS_WEB_DIR environment variable
	if webDir := os.Getenv("VSTATS_WEB_DIR"); webDir != "" {
		if _, err := os.Stat(filepath.Join(webDir, "index.html")); err == nil {
			return webDir
		}
		if _, err := os.Stat(filepath.Join(webDir, "dist", "index.html")); err == nil {
			return filepath.Join(webDir, "dist")
		}
	}

	// Check relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		paths := []string{
			filepath.Join(exeDir, "..", "web", "dist"),
			filepath.Join(exeDir, "web", "dist"),
			filepath.Join(exeDir, "..", "..", "web", "dist"),
			filepath.Join(exeDir, "..", "dist"),
		}
		for _, p := range paths {
			if abs, err := filepath.Abs(p); err == nil {
				if _, err := os.Stat(filepath.Join(abs, "index.html")); err == nil {
					return abs
				}
			}
		}
	}

	// Check common locations
	paths := []string{
		"./web/dist",
		"./web",
		"./dist",
		"../web/dist",
		"/opt/vstats/web",
	}
	for _, p := range paths {
		if abs, err := filepath.Abs(p); err == nil {
			if _, err := os.Stat(filepath.Join(abs, "index.html")); err == nil {
				return abs
			}
		}
	}

	return ""
}

const embeddedIndexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>vStats - Server Monitor</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { 
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
      color: #e8e8e8; min-height: 100vh;
      display: flex; align-items: center; justify-content: center;
    }
    .container { text-align: center; padding: 2rem; }
    h1 { font-size: 3rem; margin-bottom: 1rem; background: linear-gradient(90deg, #00d9ff, #00ff88); 
         -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
    p { color: #888; margin-bottom: 2rem; }
    .status { background: rgba(0,217,255,0.1); border: 1px solid rgba(0,217,255,0.3);
              border-radius: 12px; padding: 2rem; margin-top: 2rem; }
    .status h2 { color: #00d9ff; margin-bottom: 1rem; }
    code { background: rgba(0,0,0,0.3); padding: 0.5rem 1rem; border-radius: 6px; 
           display: block; margin: 0.5rem 0; font-size: 0.9rem; }
  </style>
</head>
<body>
  <div class="container">
    <h1>vStats</h1>
    <p>Server Monitoring Dashboard</p>
    <div class="status">
      <h2>Server is Running</h2>
      <p>Web assets not found. API is available at:</p>
      <code>GET /api/metrics</code>
      <code>GET /api/history/:server_id?range=1h|24h|7d|30d</code>
      <code>GET /api/settings/site</code>
    </div>
  </div>
</body>
</html>`
