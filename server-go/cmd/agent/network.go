package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	gopsutilnet "github.com/shirou/gopsutil/v4/net"
)

// DailyTrafficStats tracks daily network traffic statistics
type DailyTrafficStats struct {
	mu           sync.RWMutex
	CurrentDate  string    `json:"current_date"` // Format: YYYY-MM-DD
	DayStartRx   uint64    `json:"day_start_rx"` // Total RX bytes at start of day
	DayStartTx   uint64    `json:"day_start_tx"` // Total TX bytes at start of day
	DailyRx      uint64    `json:"daily_rx"`     // Daily RX bytes (calculated)
	DailyTx      uint64    `json:"daily_tx"`     // Daily TX bytes (calculated)
	lastSaveTime time.Time // Last time stats were saved
	
	// Billing period tracking (based on server config)
	BillingPeriodStart string  `json:"billing_period_start,omitempty"` // Format: YYYY-MM-DD
	PeriodStartRx      uint64  `json:"period_start_rx,omitempty"`      // Total RX at billing period start
	PeriodStartTx      uint64  `json:"period_start_tx,omitempty"`      // Total TX at billing period start
	PeriodRx           uint64  `json:"period_rx,omitempty"`            // Billing period RX bytes
	PeriodTx           uint64  `json:"period_tx,omitempty"`            // Billing period TX bytes
	MonthlyLimitGB     float64 `json:"monthly_limit_gb,omitempty"`     // Monthly limit in GB (0 = unlimited)
	ThresholdType      string  `json:"threshold_type,omitempty"`       // "sum", "max", "up", "down"
	ResetDay           int     `json:"reset_day,omitempty"`            // Day of month to reset (1-28)
}

// getDailyTrafficStatsPath returns the path to the daily traffic stats file
func getDailyTrafficStatsPath() string {
	configDir := filepath.Dir(DefaultConfigPath())
	return filepath.Join(configDir, "daily-traffic.json")
}

// loadDailyTrafficStats loads daily traffic statistics from file
func loadDailyTrafficStats() *DailyTrafficStats {
	stats := &DailyTrafficStats{
		CurrentDate: time.Now().Format("2006-01-02"),
	}

	path := getDailyTrafficStatsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist, return default stats
		return stats
	}

	if err := json.Unmarshal(data, stats); err != nil {
		// Invalid file, return default stats
		return stats
	}

	return stats
}

// saveDailyTrafficStats saves daily traffic statistics to file
func (dts *DailyTrafficStats) save() error {
	dts.mu.RLock()
	defer dts.mu.RUnlock()

	path := getDailyTrafficStatsPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(dts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write stats file: %w", err)
	}

	return nil
}

// updateDailyTraffic updates daily traffic statistics
func (dts *DailyTrafficStats) updateDailyTraffic(totalRx, totalTx uint64) (dailyRx, dailyTx uint64) {
	dts.mu.Lock()
	defer dts.mu.Unlock()

	currentDate := time.Now().Format("2006-01-02")
	now := time.Now()
	shouldSave := false

	// Check if it's a new day
	if dts.CurrentDate != currentDate {
		// New day: reset counters
		dts.CurrentDate = currentDate
		dts.DayStartRx = totalRx
		dts.DayStartTx = totalTx
		dts.DailyRx = 0
		dts.DailyTx = 0
		shouldSave = true // Save immediately on new day
	} else {
		// Same day: calculate daily traffic
		if totalRx >= dts.DayStartRx {
			dts.DailyRx = totalRx - dts.DayStartRx
		} else {
			// Counter wrapped (unlikely but possible)
			dts.DayStartRx = totalRx
			dts.DailyRx = 0
		}

		if totalTx >= dts.DayStartTx {
			dts.DailyTx = totalTx - dts.DayStartTx
		} else {
			// Counter wrapped
			dts.DayStartTx = totalTx
			dts.DailyTx = 0
		}

		// Save periodically (every 5 minutes)
		if now.Sub(dts.lastSaveTime) >= 5*time.Minute {
			shouldSave = true
		}
	}

	// Also update billing period traffic
	dts.updateBillingPeriodTraffic(totalRx, totalTx)

	// Save if needed
	if shouldSave {
		dts.lastSaveTime = now
		go func() {
			dts.save()
		}()
	}

	return dts.DailyRx, dts.DailyTx
}

// getDailyTraffic returns current daily traffic without updating
func (dts *DailyTrafficStats) getDailyTraffic() (dailyRx, dailyTx uint64) {
	dts.mu.RLock()
	defer dts.mu.RUnlock()
	return dts.DailyRx, dts.DailyTx
}

// SetTrafficConfig updates the traffic config from server
func (dts *DailyTrafficStats) SetTrafficConfig(monthlyLimitGB float64, thresholdType string, resetDay int) {
	dts.mu.Lock()
	defer dts.mu.Unlock()
	
	dts.MonthlyLimitGB = monthlyLimitGB
	dts.ThresholdType = thresholdType
	if resetDay < 1 || resetDay > 28 {
		resetDay = 1
	}
	dts.ResetDay = resetDay
	
	// Force save after config update
	go dts.save()
}

// getBillingPeriodStart calculates the billing period start date based on reset day
func getBillingPeriodStart(resetDay int, now time.Time) time.Time {
	year, month, day := now.Date()
	
	if day >= resetDay {
		// Current billing period started this month
		return time.Date(year, month, resetDay, 0, 0, 0, 0, now.Location())
	} else {
		// Current billing period started last month
		lastMonth := now.AddDate(0, -1, 0)
		return time.Date(lastMonth.Year(), lastMonth.Month(), resetDay, 0, 0, 0, 0, now.Location())
	}
}

// updateBillingPeriodTraffic updates billing period traffic based on reset day
func (dts *DailyTrafficStats) updateBillingPeriodTraffic(totalRx, totalTx uint64) (periodRx, periodTx uint64) {
	// If no reset day configured, return 0
	if dts.ResetDay == 0 {
		return 0, 0
	}
	
	now := time.Now()
	periodStart := getBillingPeriodStart(dts.ResetDay, now)
	periodStartStr := periodStart.Format("2006-01-02")
	
	// Check if we're in a new billing period
	if dts.BillingPeriodStart != periodStartStr {
		// New billing period: reset counters
		dts.BillingPeriodStart = periodStartStr
		dts.PeriodStartRx = totalRx
		dts.PeriodStartTx = totalTx
		dts.PeriodRx = 0
		dts.PeriodTx = 0
	} else {
		// Same billing period: calculate usage
		if totalRx >= dts.PeriodStartRx {
			dts.PeriodRx = totalRx - dts.PeriodStartRx
		} else {
			// Counter wrapped, reset baseline
			dts.PeriodStartRx = totalRx
			dts.PeriodRx = 0
		}
		
		if totalTx >= dts.PeriodStartTx {
			dts.PeriodTx = totalTx - dts.PeriodStartTx
		} else {
			// Counter wrapped, reset baseline
			dts.PeriodStartTx = totalTx
			dts.PeriodTx = 0
		}
	}
	
	return dts.PeriodRx, dts.PeriodTx
}

// GetBillingPeriodUsage returns the current billing period usage based on threshold type
func (dts *DailyTrafficStats) GetBillingPeriodUsage() (usageGB float64, limitGB float64, usagePercent float64) {
	dts.mu.RLock()
	defer dts.mu.RUnlock()
	
	if dts.ResetDay == 0 || dts.MonthlyLimitGB == 0 {
		return 0, 0, 0
	}
	
	var usageBytes uint64
	switch dts.ThresholdType {
	case "sum":
		usageBytes = dts.PeriodRx + dts.PeriodTx
	case "max":
		if dts.PeriodRx > dts.PeriodTx {
			usageBytes = dts.PeriodRx
		} else {
			usageBytes = dts.PeriodTx
		}
	case "up":
		usageBytes = dts.PeriodTx
	case "down":
		usageBytes = dts.PeriodRx
	default:
		usageBytes = dts.PeriodRx + dts.PeriodTx
	}
	
	usageGB = float64(usageBytes) / (1024 * 1024 * 1024)
	limitGB = dts.MonthlyLimitGB
	if limitGB > 0 {
		usagePercent = (usageGB / limitGB) * 100
	}
	
	return usageGB, limitGB, usagePercent
}

// detectGateway detects the default gateway IP address
func detectGateway() string {
	switch runtime.GOOS {
	case "linux":
		// Use 'ip route show default'
		cmd := exec.Command("ip", "route", "show", "default")
		output, err := cmd.Output()
		if err == nil {
			outputStr := string(output)
			// Parse: default via 192.168.1.1 dev eth0
			fields := strings.Fields(outputStr)
			for i, field := range fields {
				if field == "via" && i+1 < len(fields) {
					gateway := fields[i+1]
					if strings.Contains(gateway, ".") && !strings.Contains(gateway, "/") {
						return gateway
					}
				}
			}
		}
	case "darwin":
		// Use 'route -n get default'
		cmd := exec.Command("route", "-n", "get", "default")
		output, err := cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "gateway:") {
					parts := strings.Fields(line)
					if len(parts) > 1 {
						return parts[1]
					}
				}
			}
		}
	case "windows":
		// Use PowerShell to get default gateway
		cmd := exec.Command("powershell", "-Command", "(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Select-Object -First 1).NextHop")
		output, err := cmd.Output()
		if err == nil {
			gateway := strings.TrimSpace(string(output))
			if gateway != "" && strings.Contains(gateway, ".") {
				return gateway
			}
		}
		// Fallback: use 'route print'
		cmd = exec.Command("cmd", "/C", "route", "print", "0.0.0.0")
		output, err = cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			for scanner.Scan() {
				line := scanner.Text()
				fields := strings.Fields(line)
				if len(fields) >= 3 && fields[0] == "0.0.0.0" {
					gateway := fields[2]
					if strings.Contains(gateway, ".") && gateway != "0.0.0.0" {
						return gateway
					}
				}
			}
		}
	}
	return ""
}

// collectIPAddresses collects all IP addresses of the system
func collectIPAddresses() []string {
	var ips []string

	switch runtime.GOOS {
	case "linux":
		// Try 'hostname -I' first
		cmd := exec.Command("hostname", "-I")
		output, err := cmd.Output()
		if err == nil {
			fields := strings.Fields(string(output))
			for _, ip := range fields {
				if strings.Contains(ip, ".") && !strings.HasPrefix(ip, "127.") {
					ips = append(ips, ip)
				}
			}
		}
		// Fallback: use 'ip addr show'
		if len(ips) == 0 {
			cmd = exec.Command("ip", "addr", "show")
			output, err := cmd.Output()
			if err == nil {
				scanner := bufio.NewScanner(strings.NewReader(string(output)))
				for scanner.Scan() {
					line := scanner.Text()
					if strings.Contains(line, "inet ") && !strings.Contains(line, "127.0.0.1") {
						fields := strings.Fields(line)
						if len(fields) >= 2 {
							ip := strings.Split(fields[1], "/")[0]
							if strings.Contains(ip, ".") && !strings.HasPrefix(ip, "127.") {
								ips = append(ips, ip)
							}
						}
					}
				}
			}
		}
	case "darwin":
		// Use 'ifconfig'
		cmd := exec.Command("ifconfig")
		output, err := cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "inet ") && !strings.Contains(line, "127.0.0.1") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						ip := fields[1]
						if strings.Contains(ip, ".") && !strings.HasPrefix(ip, "127.") {
							ips = append(ips, ip)
						}
					}
				}
			}
		}
	case "windows":
		// Use PowerShell
		cmd := exec.Command("powershell", "-Command", "(Get-NetIPAddress -AddressFamily IPv4 | Where-Object { $_.IPAddress -ne '127.0.0.1' }).IPAddress")
		output, err := cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			for scanner.Scan() {
				ip := strings.TrimSpace(scanner.Text())
				if ip != "" && strings.Contains(ip, ".") && !strings.HasPrefix(ip, "127.") {
					ips = append(ips, ip)
				}
			}
		}
		// Fallback: use 'ipconfig'
		if len(ips) == 0 {
			cmd = exec.Command("ipconfig")
			output, err := cmd.Output()
			if err == nil {
				scanner := bufio.NewScanner(strings.NewReader(string(output)))
				for scanner.Scan() {
					line := scanner.Text()
					if strings.Contains(line, "IPv4") || strings.Contains(line, "IP Address") {
						parts := strings.Split(line, ":")
						if len(parts) >= 2 {
							ip := strings.TrimSpace(parts[1])
							if strings.Contains(ip, ".") && !strings.HasPrefix(ip, "127.") {
								ips = append(ips, ip)
							}
						}
					}
				}
			}
		}
	}

	return ips
}

// isVirtualInterface checks if a network interface is virtual
func isVirtualInterface(name string) bool {
	return name == "lo" || name == "lo0" ||
		strings.HasPrefix(name, "veth") ||
		strings.HasPrefix(name, "docker") ||
		strings.HasPrefix(name, "br-") ||
		strings.HasPrefix(name, "virbr") ||
		strings.HasPrefix(name, "utun") ||
		strings.HasPrefix(name, "awdl") ||
		strings.HasPrefix(name, "llw")
}

// getInterfaceDetails gets MAC address and link speed for a network interface
func getInterfaceDetails(name string) (string, uint32) {
	var mac string
	var speed uint32

	switch runtime.GOOS {
	case "linux":
		// Read MAC address
		macPath := filepath.Join("/sys/class/net", name, "address")
		if data, err := os.ReadFile(macPath); err == nil {
			addr := strings.TrimSpace(string(data))
			if addr != "00:00:00:00:00:00" {
				mac = strings.ToUpper(addr)
			}
		}
		// Read link speed (in Mbps)
		speedPath := filepath.Join("/sys/class/net", name, "speed")
		if data, err := os.ReadFile(speedPath); err == nil {
			if s, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 32); err == nil && s > 0 {
				speed = uint32(s)
			}
		}
	case "darwin":
		// Use ifconfig to get MAC
		cmd := exec.Command("ifconfig", name)
		output, err := cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "ether ") {
					parts := strings.Fields(line)
					if len(parts) > 1 {
						mac = strings.ToUpper(parts[1])
					}
				}
			}
		}
		// Use networksetup for speed
		cmd = exec.Command("networksetup", "-getMedia", name)
		output, err = cmd.Output()
		if err == nil {
			outputStr := strings.ToLower(string(output))
			if strings.Contains(outputStr, "1000") {
				speed = 1000
			} else if strings.Contains(outputStr, "100") {
				speed = 100
			} else if strings.Contains(outputStr, "10") {
				speed = 10
			}
		}
	case "windows":
		// Use PowerShell
		cmd := exec.Command("powershell", "-Command", fmt.Sprintf("Get-NetAdapter -Name '%s' | Select-Object -Property MacAddress,LinkSpeed | ConvertTo-Json", name))
		output, err := cmd.Output()
		if err == nil {
			var data map[string]interface{}
			if json.Unmarshal(output, &data) == nil {
				if macAddr, ok := data["MacAddress"].(string); ok {
					mac = strings.ToUpper(macAddr)
				}
				if linkSpeed, ok := data["LinkSpeed"].(string); ok {
					// Parse "1 Gbps" or "100 Mbps"
					parts := strings.Fields(linkSpeed)
					if len(parts) >= 2 {
						if num, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
							if strings.HasPrefix(parts[1], "G") {
								speed = uint32(num * 1000)
							} else {
								speed = uint32(num)
							}
						}
					}
				}
			}
		}
	}

	return mac, speed
}

// collectNetworkMetrics collects network interface metrics
func collectNetworkMetrics(netIO []gopsutilnet.IOCountersStat, lastRx, lastTx uint64, lastTime time.Time, dailyStats *DailyTrafficStats) ([]NetworkInterface, uint64, uint64, uint64, uint64, uint64, uint64, time.Time) {
	var interfaces []NetworkInterface
	var totalRx, totalTx uint64

	for _, io := range netIO {
		// Filter out virtual interfaces
		name := strings.ToLower(io.Name)
		if isVirtualInterface(name) {
			continue
		}

		// Get interface details (MAC address and speed)
		mac, speed := getInterfaceDetails(io.Name)

		interfaces = append(interfaces, NetworkInterface{
			Name:      io.Name,
			MAC:       mac,
			Speed:     speed,
			RxBytes:   io.BytesRecv,
			TxBytes:   io.BytesSent,
			RxPackets: io.PacketsRecv,
			TxPackets: io.PacketsSent,
		})
		totalRx += io.BytesRecv
		totalTx += io.BytesSent
	}

	// Calculate network speed
	now := time.Now()
	elapsed := now.Sub(lastTime).Seconds()
	var rxSpeed, txSpeed uint64
	if elapsed > 0.1 {
		rxDiff := totalRx - lastRx
		txDiff := totalTx - lastTx
		if totalRx >= lastRx {
			rxSpeed = uint64(float64(rxDiff) / elapsed)
		}
		if totalTx >= lastTx {
			txSpeed = uint64(float64(txDiff) / elapsed)
		}
	}

	// Update daily traffic statistics
	var dailyRx, dailyTx uint64
	if dailyStats != nil {
		dailyRx, dailyTx = dailyStats.updateDailyTraffic(totalRx, totalTx)
	}

	return interfaces, totalRx, totalTx, rxSpeed, txSpeed, dailyRx, dailyTx, now
}
