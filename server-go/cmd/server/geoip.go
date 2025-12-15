package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

// ============================================================================
// GeoIP Configuration
// ============================================================================

// GeoIPConfig holds the GeoIP configuration
type GeoIPConfig struct {
	// Provider: "mmdb", "ip-api", "ipinfo", "auto" (try mmdb first, then api)
	Provider string `json:"provider"`
	// MMDB file path (relative to config dir or absolute)
	MMDBPath string `json:"mmdb_path,omitempty"`
	// IPInfo API token (optional, for higher rate limits)
	IPInfoToken string `json:"ipinfo_token,omitempty"`
	// Auto-update settings
	AutoUpdate       bool   `json:"auto_update"`
	UpdateIntervalHr int    `json:"update_interval_hr,omitempty"` // Hours between updates, default 24
	LastUpdate       string `json:"last_update,omitempty"`        // ISO8601 timestamp
}

// GeoIPResult represents the result of a GeoIP lookup
type GeoIPResult struct {
	IP          string `json:"ip"`
	CountryCode string `json:"country_code"` // ISO 3166-1 alpha-2
	CountryName string `json:"country_name"`
	City        string `json:"city,omitempty"`
	Region      string `json:"region,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	ASN         string `json:"asn,omitempty"`
	Org         string `json:"org,omitempty"`
	Cached      bool   `json:"cached,omitempty"`
}

// ============================================================================
// GeoIP Service
// ============================================================================

// GeoIPService provides IP geolocation functionality
type GeoIPService struct {
	mu       sync.RWMutex
	config   *GeoIPConfig
	mmdb     *geoip2.Reader
	cache    map[string]*GeoIPResult
	cacheTTL time.Duration
}

var (
	geoIPService *GeoIPService
	geoIPOnce    sync.Once
)

// GetGeoIPService returns the singleton GeoIP service
func GetGeoIPService() *GeoIPService {
	geoIPOnce.Do(func() {
		geoIPService = &GeoIPService{
			cache:    make(map[string]*GeoIPResult),
			cacheTTL: 24 * time.Hour,
		}
	})
	return geoIPService
}

// Initialize initializes the GeoIP service with config
func (s *GeoIPService) Initialize(config *GeoIPConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config
	if config == nil {
		s.config = &GeoIPConfig{
			Provider:   "auto",
			AutoUpdate: true,
		}
	}

	// Set default provider
	if s.config.Provider == "" {
		s.config.Provider = "auto"
	}

	// Try to load MMDB if configured or in auto mode
	if s.config.Provider == "mmdb" || s.config.Provider == "auto" {
		if err := s.loadMMDB(); err != nil {
			if s.config.Provider == "mmdb" {
				return fmt.Errorf("failed to load MMDB: %w", err)
			}
			// In auto mode, just log and continue with API fallback
			fmt.Printf("⚠️ MMDB not available, will use API fallback: %v\n", err)
		}
	}

	return nil
}

// loadMMDB loads the MaxMind MMDB database
func (s *GeoIPService) loadMMDB() error {
	// Close existing reader if any
	if s.mmdb != nil {
		s.mmdb.Close()
		s.mmdb = nil
	}

	// Determine MMDB path
	mmdbPath := s.config.MMDBPath
	if mmdbPath == "" {
		// Default paths to search
		searchPaths := []string{
			"GeoLite2-City.mmdb",
			"GeoLite2-Country.mmdb",
			filepath.Join(getExeDir(), "GeoLite2-City.mmdb"),
			filepath.Join(getExeDir(), "GeoLite2-Country.mmdb"),
			"/usr/share/GeoIP/GeoLite2-City.mmdb",
			"/usr/share/GeoIP/GeoLite2-Country.mmdb",
		}
		for _, path := range searchPaths {
			if _, err := os.Stat(path); err == nil {
				mmdbPath = path
				break
			}
		}
	}

	if mmdbPath == "" {
		return fmt.Errorf("no MMDB file found")
	}

	// Make path absolute if relative
	if !filepath.IsAbs(mmdbPath) {
		mmdbPath = filepath.Join(getExeDir(), mmdbPath)
	}

	// Open MMDB
	reader, err := geoip2.Open(mmdbPath)
	if err != nil {
		return fmt.Errorf("failed to open MMDB %s: %w", mmdbPath, err)
	}

	s.mmdb = reader
	fmt.Printf("✅ Loaded GeoIP database: %s\n", mmdbPath)
	return nil
}

// Close closes the GeoIP service
func (s *GeoIPService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mmdb != nil {
		s.mmdb.Close()
		s.mmdb = nil
	}
}

// Lookup looks up the geolocation of an IP address
func (s *GeoIPService) Lookup(ipStr string) (*GeoIPResult, error) {
	// Normalize IP
	ipStr = normalizeIP(ipStr)
	if ipStr == "" {
		return nil, fmt.Errorf("invalid IP address")
	}

	// Check cache first
	s.mu.RLock()
	if cached, ok := s.cache[ipStr]; ok {
		s.mu.RUnlock()
		result := *cached
		result.Cached = true
		return &result, nil
	}
	config := s.config
	mmdb := s.mmdb
	s.mu.RUnlock()

	var result *GeoIPResult
	var err error

	// Try lookup based on provider
	provider := "auto"
	if config != nil {
		provider = config.Provider
	}

	switch provider {
	case "mmdb":
		result, err = s.lookupMMDB(ipStr, mmdb)
	case "ip-api":
		result, err = s.lookupIPAPI(ipStr)
	case "ipinfo":
		result, err = s.lookupIPInfo(ipStr, config)
	case "auto":
		// Try MMDB first, then fall back to APIs
		if mmdb != nil {
			result, err = s.lookupMMDB(ipStr, mmdb)
		}
		if result == nil || err != nil {
			result, err = s.lookupIPAPI(ipStr)
		}
		if result == nil || err != nil {
			result, err = s.lookupIPInfo(ipStr, config)
		}
	default:
		result, err = s.lookupIPAPI(ipStr)
	}

	if err != nil {
		return nil, err
	}

	// Cache result
	s.mu.Lock()
	s.cache[ipStr] = result
	s.mu.Unlock()

	return result, nil
}

// lookupMMDB performs MMDB lookup
func (s *GeoIPService) lookupMMDB(ipStr string, reader *geoip2.Reader) (*GeoIPResult, error) {
	if reader == nil {
		return nil, fmt.Errorf("MMDB not loaded")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP: %s", ipStr)
	}

	// Try City database first
	city, err := reader.City(ip)
	if err == nil && city.Country.IsoCode != "" {
		result := &GeoIPResult{
			IP:          ipStr,
			CountryCode: city.Country.IsoCode,
			CountryName: city.Country.Names["en"],
			Latitude:    city.Location.Latitude,
			Longitude:   city.Location.Longitude,
		}
		if len(city.City.Names) > 0 {
			result.City = city.City.Names["en"]
		}
		if len(city.Subdivisions) > 0 {
			result.Region = city.Subdivisions[0].Names["en"]
		}
		return result, nil
	}

	// Fall back to Country database
	country, err := reader.Country(ip)
	if err != nil {
		return nil, fmt.Errorf("MMDB lookup failed: %w", err)
	}

	if country.Country.IsoCode == "" {
		return nil, fmt.Errorf("no country data for IP: %s", ipStr)
	}

	return &GeoIPResult{
		IP:          ipStr,
		CountryCode: country.Country.IsoCode,
		CountryName: country.Country.Names["en"],
	}, nil
}

// IP-API response structure
type ipAPIResponse struct {
	Status      string  `json:"status"`
	Message     string  `json:"message,omitempty"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	AS          string  `json:"as"`
	Org         string  `json:"org"`
}

// lookupIPAPI performs lookup using ip-api.com (free, 45 req/min limit)
func (s *GeoIPService) lookupIPAPI(ipStr string) (*GeoIPResult, error) {
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,country,countryCode,region,regionName,city,lat,lon,as,org", ipStr)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ip-api request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ip-api read failed: %w", err)
	}

	var data ipAPIResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("ip-api parse failed: %w", err)
	}

	if data.Status != "success" {
		return nil, fmt.Errorf("ip-api error: %s", data.Message)
	}

	return &GeoIPResult{
		IP:          ipStr,
		CountryCode: data.CountryCode,
		CountryName: data.Country,
		City:        data.City,
		Region:      data.RegionName,
		Latitude:    data.Lat,
		Longitude:   data.Lon,
		ASN:         data.AS,
		Org:         data.Org,
	}, nil
}

// IPInfo response structure
type ipInfoResponse struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Timezone string `json:"timezone"`
}

// lookupIPInfo performs lookup using ipinfo.io
func (s *GeoIPService) lookupIPInfo(ipStr string, config *GeoIPConfig) (*GeoIPResult, error) {
	url := fmt.Sprintf("https://ipinfo.io/%s/json", ipStr)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ipinfo request creation failed: %w", err)
	}

	// Add token if available
	if config != nil && config.IPInfoToken != "" {
		req.Header.Set("Authorization", "Bearer "+config.IPInfoToken)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ipinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ipinfo read failed: %w", err)
	}

	var data ipInfoResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("ipinfo parse failed: %w", err)
	}

	if data.Country == "" {
		return nil, fmt.Errorf("ipinfo: no country data")
	}

	result := &GeoIPResult{
		IP:          ipStr,
		CountryCode: data.Country,
		City:        data.City,
		Region:      data.Region,
		Org:         data.Org,
	}

	// Parse location
	if data.Loc != "" {
		parts := strings.Split(data.Loc, ",")
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%f", &result.Latitude)
			fmt.Sscanf(parts[1], "%f", &result.Longitude)
		}
	}

	return result, nil
}

// LookupBatch looks up multiple IPs
func (s *GeoIPService) LookupBatch(ips []string) map[string]*GeoIPResult {
	results := make(map[string]*GeoIPResult)
	for _, ip := range ips {
		if result, err := s.Lookup(ip); err == nil {
			results[ip] = result
		}
	}
	return results
}

// ClearCache clears the lookup cache
func (s *GeoIPService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]*GeoIPResult)
}

// GetConfig returns the current config
func (s *GeoIPService) GetConfig() *GeoIPConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.config == nil {
		return &GeoIPConfig{Provider: "auto"}
	}
	return s.config
}

// SetConfig updates the configuration
func (s *GeoIPService) SetConfig(config *GeoIPConfig) error {
	return s.Initialize(config)
}

// IsMMDBLoaded returns whether MMDB is loaded
func (s *GeoIPService) IsMMDBLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mmdb != nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// normalizeIP extracts and normalizes an IP address
func normalizeIP(ipStr string) string {
	ipStr = strings.TrimSpace(ipStr)

	// Handle IPv6 with port [::1]:8080
	if strings.HasPrefix(ipStr, "[") {
		if idx := strings.LastIndex(ipStr, "]"); idx > 0 {
			ipStr = ipStr[1:idx]
		}
	}

	// Handle host:port
	if host, _, err := net.SplitHostPort(ipStr); err == nil {
		ipStr = host
	}

	// Validate IP
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}

	return ip.String()
}

// CountryCodeToFlag converts ISO 3166-1 alpha-2 country code to flag emoji
func CountryCodeToFlag(code string) string {
	if len(code) != 2 {
		return ""
	}
	code = strings.ToUpper(code)
	// Each letter becomes a regional indicator symbol (A=🇦, B=🇧, etc.)
	offset := rune(0x1F1E6 - 'A')
	return string(rune(code[0])+offset) + string(rune(code[1])+offset)
}

// ============================================================================
// Country name mapping for common codes
// ============================================================================

var countryNames = map[string]string{
	"US": "United States",
	"CN": "China",
	"JP": "Japan",
	"KR": "South Korea",
	"HK": "Hong Kong",
	"TW": "Taiwan",
	"SG": "Singapore",
	"DE": "Germany",
	"FR": "France",
	"GB": "United Kingdom",
	"NL": "Netherlands",
	"RU": "Russia",
	"CA": "Canada",
	"AU": "Australia",
	"IN": "India",
	"BR": "Brazil",
}

// GetCountryName returns the country name for a code
func GetCountryName(code string) string {
	code = strings.ToUpper(code)
	if name, ok := countryNames[code]; ok {
		return name
	}
	return code
}
