package main

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ============================================================================
// Asset Management Types
// ============================================================================

// CostSummary represents cost statistics for a time period
type CostSummary struct {
	Currency       string             `json:"currency"`
	TotalMonthly   float64            `json:"total_monthly"`    // Normalized monthly cost
	TotalAnnual    float64            `json:"total_annual"`     // Normalized annual cost
	ServerCount    int                `json:"server_count"`     // Total servers with pricing
	ByProvider     map[string]float64 `json:"by_provider"`      // Monthly cost by provider
	ByCurrency     map[string]float64 `json:"by_currency"`      // Monthly cost by original currency
	ExpiringCount  int                `json:"expiring_count"`   // Servers expiring in 30 days
	ExpiredCount   int                `json:"expired_count"`    // Already expired servers
	AutoRenewCount int                `json:"auto_renew_count"` // Servers with auto-renew enabled
	Servers        []ServerCostInfo   `json:"servers,omitempty"`
}

// ServerCostInfo represents cost info for a single server
type ServerCostInfo struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Provider      string  `json:"provider"`
	PriceAmount   float64 `json:"price_amount"`
	PricePeriod   string  `json:"price_period"`
	PriceCurrency string  `json:"price_currency"`
	MonthlyNorm   float64 `json:"monthly_normalized"` // Normalized to monthly
	ExpiryDate    string  `json:"expiry_date,omitempty"`
	DaysLeft      *int    `json:"days_left,omitempty"`
	AutoRenew     bool    `json:"auto_renew"`
}

// ServerImportItem represents a server to import
type ServerImportItem struct {
	Name          string            `json:"name"`
	Location      string            `json:"location"`
	Provider      string            `json:"provider"`
	Tag           string            `json:"tag,omitempty"`
	GroupValues   map[string]string `json:"group_values,omitempty"`
	PriceAmount   string            `json:"price_amount,omitempty"`
	PricePeriod   string            `json:"price_period,omitempty"`
	PriceCurrency string            `json:"price_currency,omitempty"`
	PurchaseDate  string            `json:"purchase_date,omitempty"`
	ExpiryDate    string            `json:"expiry_date,omitempty"`
	AutoRenew     bool              `json:"auto_renew,omitempty"`
	TipBadge      string            `json:"tip_badge,omitempty"`
	Notes         string            `json:"notes,omitempty"`
}

// ImportResult represents the result of a bulk import
type ImportResult struct {
	Success  int      `json:"success"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
	ServerIDs []string `json:"server_ids,omitempty"`
}

// ServerExportItem represents a server for export
type ServerExportItem struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Location      string            `json:"location"`
	Provider      string            `json:"provider"`
	Tag           string            `json:"tag,omitempty"`
	GroupValues   map[string]string `json:"group_values,omitempty"`
	Token         string            `json:"token,omitempty"`
	PriceAmount   string            `json:"price_amount,omitempty"`
	PricePeriod   string            `json:"price_period,omitempty"`
	PriceCurrency string            `json:"price_currency,omitempty"`
	PurchaseDate  string            `json:"purchase_date,omitempty"`
	ExpiryDate    string            `json:"expiry_date,omitempty"`
	AutoRenew     bool              `json:"auto_renew,omitempty"`
	TipBadge      string            `json:"tip_badge,omitempty"`
	Notes         string            `json:"notes,omitempty"`
}

// ============================================================================
// Cost Statistics Handler
// ============================================================================

// GetCostStatistics returns cost analysis for all servers
func (s *AppState) GetCostStatistics(c *gin.Context) {
	includeCurrency := c.DefaultQuery("currency", "USD")
	includeServers := c.Query("include_servers") == "true"
	
	s.ConfigMu.RLock()
	servers := s.Config.Servers
	s.ConfigMu.RUnlock()
	
	now := time.Now()
	summary := CostSummary{
		Currency:   includeCurrency,
		ByProvider: make(map[string]float64),
		ByCurrency: make(map[string]float64),
	}
	
	var serverCosts []ServerCostInfo
	
	// Process remote servers
	for _, server := range servers {
		if server.PriceAmount == "" {
			continue
		}
		
		cost := parseServerCost(server.ID, server.Name, server.Provider, server.PriceAmount, server.PricePeriod, server.PriceCurrency, server.ExpiryDate, server.AutoRenew, now)
		if cost == nil {
			continue
		}
		
		summary.ServerCount++
		summary.TotalMonthly += cost.MonthlyNorm
		
		provider := server.Provider
		if provider == "" {
			provider = "Unknown"
		}
		summary.ByProvider[provider] += cost.MonthlyNorm
		
		currency := cost.PriceCurrency
		if currency == "" {
			currency = "USD"
		}
		summary.ByCurrency[currency] += cost.PriceAmount
		
		// Count expiry status
		if cost.DaysLeft != nil {
			if *cost.DaysLeft < 0 {
				summary.ExpiredCount++
			} else if *cost.DaysLeft <= 30 {
				summary.ExpiringCount++
			}
		}
		
		if server.AutoRenew {
			summary.AutoRenewCount++
		}
		
		serverCosts = append(serverCosts, *cost)
	}
	
	summary.TotalAnnual = summary.TotalMonthly * 12
	
	if includeServers {
		// Sort by monthly cost descending
		sort.Slice(serverCosts, func(i, j int) bool {
			return serverCosts[i].MonthlyNorm > serverCosts[j].MonthlyNorm
		})
		summary.Servers = serverCosts
	}
	
	c.JSON(http.StatusOK, summary)
}

// parseServerCost parses cost information for a server
func parseServerCost(id, name, provider, priceAmount, pricePeriod, priceCurrency, expiryDate string, autoRenew bool, now time.Time) *ServerCostInfo {
	// Parse price amount - extract numeric value
	numStr := strings.TrimLeft(priceAmount, "$€£¥₹₽₩฿")
	numStr = strings.ReplaceAll(numStr, ",", "")
	amount, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64)
	if err != nil {
		return nil
	}
	
	// Normalize to monthly
	var monthly float64
	period := strings.ToLower(pricePeriod)
	switch period {
	case "year", "yearly", "annual":
		monthly = amount / 12
	case "quarter", "quarterly":
		monthly = amount / 3
	case "week", "weekly":
		monthly = amount * 4.33 // Average weeks per month
	case "day", "daily":
		monthly = amount * 30
	default: // month, monthly, or empty
		monthly = amount
	}
	
	cost := &ServerCostInfo{
		ID:            id,
		Name:          name,
		Provider:      provider,
		PriceAmount:   amount,
		PricePeriod:   pricePeriod,
		PriceCurrency: priceCurrency,
		MonthlyNorm:   monthly,
		AutoRenew:     autoRenew,
	}
	
	// Calculate days left
	if expiryDate != "" {
		cost.ExpiryDate = expiryDate
		formats := []string{"2006-01-02", "2006-01-02T15:04:05Z", time.RFC3339}
		for _, format := range formats {
			if t, err := time.Parse(format, expiryDate); err == nil {
				days := int(t.Sub(now).Hours() / 24)
				cost.DaysLeft = &days
				break
			}
		}
	}
	
	return cost
}

// ============================================================================
// Bulk Import Handler
// ============================================================================

// ImportServers handles bulk import of servers
func (s *AppState) ImportServers(c *gin.Context) {
	var items []ServerImportItem
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}
	
	if len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No servers to import"})
		return
	}
	
	if len(items) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 100 servers per import"})
		return
	}
	
	result := ImportResult{
		ServerIDs: make([]string, 0),
	}
	
	s.ConfigMu.Lock()
	defer s.ConfigMu.Unlock()
	
	for i, item := range items {
		// Validate required fields
		if item.Name == "" {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: name is required", i+1))
			continue
		}
		
		server := RemoteServer{
			ID:            uuid.New().String(),
			Name:          item.Name,
			Location:      item.Location,
			Provider:      item.Provider,
			Tag:           item.Tag,
			Token:         uuid.New().String(),
			GroupValues:   item.GroupValues,
			PriceAmount:   item.PriceAmount,
			PricePeriod:   item.PricePeriod,
			PriceCurrency: item.PriceCurrency,
			PurchaseDate:  item.PurchaseDate,
			ExpiryDate:    item.ExpiryDate,
			AutoRenew:     item.AutoRenew,
			TipBadge:      item.TipBadge,
			Notes:         item.Notes,
		}
		
		s.Config.Servers = append(s.Config.Servers, server)
		result.Success++
		result.ServerIDs = append(result.ServerIDs, server.ID)
	}
	
	if result.Success > 0 {
		SaveConfig(s.Config)
		LogAuditFromContext(c, AuditActionServerCreate, AuditCategoryServer, "import", "", 
			fmt.Sprintf("%d servers", result.Success), 
			fmt.Sprintf("Bulk imported %d servers", result.Success))
	}
	
	c.JSON(http.StatusOK, result)
}

// ============================================================================
// Bulk Export Handler
// ============================================================================

// ExportServers handles bulk export of servers
func (s *AppState) ExportServers(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	includeTokens := c.Query("include_tokens") == "true"
	
	s.ConfigMu.RLock()
	servers := s.Config.Servers
	s.ConfigMu.RUnlock()
	
	var exports []ServerExportItem
	for _, server := range servers {
		item := ServerExportItem{
			ID:            server.ID,
			Name:          server.Name,
			Location:      server.Location,
			Provider:      server.Provider,
			Tag:           server.Tag,
			GroupValues:   server.GroupValues,
			PriceAmount:   server.PriceAmount,
			PricePeriod:   server.PricePeriod,
			PriceCurrency: server.PriceCurrency,
			PurchaseDate:  server.PurchaseDate,
			ExpiryDate:    server.ExpiryDate,
			AutoRenew:     server.AutoRenew,
			TipBadge:      server.TipBadge,
			Notes:         server.Notes,
		}
		if includeTokens {
			item.Token = server.Token
		}
		exports = append(exports, item)
	}
	
	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=servers.csv")
		
		writer := csv.NewWriter(c.Writer)
		// Write header
		header := []string{"id", "name", "location", "provider", "tag", "price_amount", "price_period", "price_currency", "purchase_date", "expiry_date", "auto_renew", "tip_badge", "notes"}
		if includeTokens {
			header = append(header, "token")
		}
		writer.Write(header)
		
		// Write rows
		for _, e := range exports {
			row := []string{
				e.ID, e.Name, e.Location, e.Provider, e.Tag,
				e.PriceAmount, e.PricePeriod, e.PriceCurrency,
				e.PurchaseDate, e.ExpiryDate,
				strconv.FormatBool(e.AutoRenew),
				e.TipBadge, e.Notes,
			}
			if includeTokens {
				row = append(row, e.Token)
			}
			writer.Write(row)
		}
		writer.Flush()
		
	default: // json
		c.Header("Content-Disposition", "attachment; filename=servers.json")
		c.JSON(http.StatusOK, gin.H{
			"servers":    exports,
			"exported_at": time.Now().Format(time.RFC3339),
			"count":      len(exports),
		})
	}
}

// ============================================================================
// Import from CSV Handler
// ============================================================================

// ImportServersCSV handles CSV import of servers
func (s *AppState) ImportServersCSV(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CSV format"})
		return
	}
	
	if len(records) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is empty or has no data rows"})
		return
	}
	
	// Parse header
	header := records[0]
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}
	
	// Required field check
	nameIdx, hasName := colIndex["name"]
	if !hasName {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV must have a 'name' column"})
		return
	}
	
	result := ImportResult{
		ServerIDs: make([]string, 0),
	}
	
	s.ConfigMu.Lock()
	defer s.ConfigMu.Unlock()
	
	for i, row := range records[1:] {
		if len(row) <= nameIdx || strings.TrimSpace(row[nameIdx]) == "" {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: name is required", i+2))
			continue
		}
		
		server := RemoteServer{
			ID:    uuid.New().String(),
			Name:  strings.TrimSpace(row[nameIdx]),
			Token: uuid.New().String(),
		}
		
		// Map optional columns
		if idx, ok := colIndex["location"]; ok && idx < len(row) {
			server.Location = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["provider"]; ok && idx < len(row) {
			server.Provider = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["tag"]; ok && idx < len(row) {
			server.Tag = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["price_amount"]; ok && idx < len(row) {
			server.PriceAmount = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["price_period"]; ok && idx < len(row) {
			server.PricePeriod = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["price_currency"]; ok && idx < len(row) {
			server.PriceCurrency = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["purchase_date"]; ok && idx < len(row) {
			server.PurchaseDate = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["expiry_date"]; ok && idx < len(row) {
			server.ExpiryDate = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["auto_renew"]; ok && idx < len(row) {
			server.AutoRenew = strings.ToLower(strings.TrimSpace(row[idx])) == "true" || row[idx] == "1"
		}
		if idx, ok := colIndex["tip_badge"]; ok && idx < len(row) {
			server.TipBadge = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["notes"]; ok && idx < len(row) {
			server.Notes = strings.TrimSpace(row[idx])
		}
		
		s.Config.Servers = append(s.Config.Servers, server)
		result.Success++
		result.ServerIDs = append(result.ServerIDs, server.ID)
	}
	
	if result.Success > 0 {
		SaveConfig(s.Config)
		LogAuditFromContext(c, AuditActionServerCreate, AuditCategoryServer, "import_csv", "",
			fmt.Sprintf("%d servers", result.Success),
			fmt.Sprintf("Bulk imported %d servers from CSV", result.Success))
	}
	
	c.JSON(http.StatusOK, result)
}

// ============================================================================
// Expiring Servers Handler
// ============================================================================

// GetExpiringServers returns servers that are expiring soon
func (s *AppState) GetExpiringServers(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 0 {
		days = 30
	}
	
	includeExpired := c.Query("include_expired") == "true"
	
	s.ConfigMu.RLock()
	servers := s.Config.Servers
	s.ConfigMu.RUnlock()
	
	now := time.Now()
	var expiring []gin.H
	
	// Check remote servers
	for _, server := range servers {
		if server.ExpiryDate == "" {
			continue
		}
		
		daysLeft := calculateDaysLeft(server.ExpiryDate, now)
		if daysLeft == nil {
			continue
		}
		
		if *daysLeft <= days || (includeExpired && *daysLeft < 0) {
			expiring = append(expiring, gin.H{
				"id":            server.ID,
				"name":          server.Name,
				"provider":      server.Provider,
				"expiry_date":   server.ExpiryDate,
				"days_left":     *daysLeft,
				"auto_renew":    server.AutoRenew,
				"price_amount":  server.PriceAmount,
				"price_period":  server.PricePeriod,
				"price_currency": server.PriceCurrency,
			})
		}
	}
	
	// Sort by days left ascending
	sort.Slice(expiring, func(i, j int) bool {
		di := expiring[i]["days_left"].(int)
		dj := expiring[j]["days_left"].(int)
		return di < dj
	})
	
	c.JSON(http.StatusOK, gin.H{
		"servers": expiring,
		"count":   len(expiring),
	})
}

// calculateDaysLeft calculates days until expiry
func calculateDaysLeft(expiryDateStr string, now time.Time) *int {
	formats := []string{"2006-01-02", "2006-01-02T15:04:05Z", time.RFC3339}
	for _, format := range formats {
		if t, err := time.Parse(format, expiryDateStr); err == nil {
			days := int(t.Sub(now).Hours() / 24)
			return &days
		}
	}
	return nil
}

// ============================================================================
// Update Expiry Alert Rule Handler
// ============================================================================

// UpdateExpiryRule updates the expiry alert rule configuration
func (s *AppState) UpdateExpiryRule(c *gin.Context) {
	var req struct {
		Enabled     *bool    `json:"enabled,omitempty"`
		DaysBefore  *[]int   `json:"days_before,omitempty"`
		Channels    *[]string `json:"channels,omitempty"`
		Servers     *[]string `json:"servers,omitempty"`
		Exclude     *[]string `json:"exclude,omitempty"`
		ExcludeAuto *bool    `json:"exclude_auto,omitempty"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	
	s.ConfigMu.Lock()
	defer s.ConfigMu.Unlock()
	
	if s.Config.AlertConfig == nil {
		s.Config.AlertConfig = &AlertConfig{}
	}
	
	rule := &s.Config.AlertConfig.Rules.Expiry
	
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.DaysBefore != nil {
		rule.DaysBefore = *req.DaysBefore
	}
	if req.Channels != nil {
		rule.Channels = *req.Channels
	}
	if req.Servers != nil {
		rule.Servers = *req.Servers
	}
	if req.Exclude != nil {
		rule.Exclude = *req.Exclude
	}
	if req.ExcludeAuto != nil {
		rule.ExcludeAuto = *req.ExcludeAuto
	}
	
	SaveConfig(s.Config)
	
	LogAuditFromContext(c, AuditActionRuleUpdate, AuditCategoryAlert, "expiry_rule", "", "",
		"Updated expiry alert rule configuration")
	
	c.JSON(http.StatusOK, rule)
}
