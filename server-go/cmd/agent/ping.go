package main

import (
	"context"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// collectPingMetrics collects ping metrics for configured targets
func collectPingMetrics(gatewayIP string, customTargets []PingTargetConfig) *PingMetrics {
	// If no custom targets configured, return nil (no ping)
	if len(customTargets) == 0 {
		return nil
	}

	var targets []PingTarget
	pingedHosts := make(map[string]bool)

	// Only ping custom targets from dashboard configuration
	for _, ct := range customTargets {
		if ct.Host == "" || pingedHosts[ct.Host] {
			continue
		}

		// Determine type (default to icmp)
		targetType := ct.Type
		if targetType == "" {
			targetType = "icmp"
		}

		var latency *float64
		var packetLoss float64
		var status string

		if targetType == "tcp" {
			// Use TCP connection test
			port := ct.Port
			if port == 0 {
				port = 80 // Default to HTTP port
			}
			latency, status = testTCPConnection(ct.Host, port)
			if status == "ok" {
				packetLoss = 0.0
			} else {
				packetLoss = 100.0
			}
		} else {
			// Use ICMP ping
			latency, packetLoss, status = pingHost(ct.Host)
		}

		targets = append(targets, PingTarget{
			Name:       ct.Name,
			Host:       ct.Host,
			Type:       targetType,
			Port:       ct.Port,
			LatencyMs:  latency,
			PacketLoss: packetLoss,
			Status:     status,
		})
		pingedHosts[ct.Host] = true
	}

	// Return nil if no valid targets after filtering
	if len(targets) == 0 {
		return nil
	}

	return &PingMetrics{
		Targets:   targets,
		Timestamp: time.Now().Unix(), // Record when ping was actually collected
	}
}

// testTCPConnection tests TCP connection latency
func testTCPConnection(host string, port int) (*float64, string) {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()

	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return nil, "error"
	}
	defer conn.Close()

	latency := float64(time.Since(start).Nanoseconds()) / 1000000.0 // Convert to milliseconds
	return &latency, "ok"
}

// pingHost performs ICMP ping to a host
func pingHost(host string) (*float64, float64, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "3", "-w", "2000", host)
	} else if runtime.GOOS == "darwin" {
		// macOS uses -W with milliseconds
		cmd = exec.CommandContext(ctx, "ping", "-c", "3", "-W", "2000", host)
	} else {
		// Linux uses -W with seconds
		cmd = exec.CommandContext(ctx, "ping", "-c", "3", "-W", "2", host)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, 100.0, "error"
	}

	outputStr := string(output)
	status := "ok"
	packetLoss := 0.0
	var latency *float64

	// Parse packet loss
	if strings.Contains(outputStr, "100%") || strings.Contains(outputStr, "timeout") {
		status = "timeout"
		packetLoss = 100.0
	} else {
		// Extract packet loss percentage - try multiple patterns
		packetLossRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)%\s*(?:packet\s+)?loss`)
		if matches := packetLossRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
			if loss, err := strconv.ParseFloat(matches[1], 64); err == nil {
				packetLoss = loss
			}
		} else {
			// Try alternative format: "3 packets transmitted, 0 received, 100% packet loss"
			altRegex := regexp.MustCompile(`(\d+)\s+packets\s+transmitted.*?(\d+)\s+received`)
			if matches := altRegex.FindStringSubmatch(outputStr); len(matches) >= 3 {
				if transmitted, err1 := strconv.ParseFloat(matches[1], 64); err1 == nil {
					if received, err2 := strconv.ParseFloat(matches[2], 64); err2 == nil && transmitted > 0 {
						packetLoss = ((transmitted - received) / transmitted) * 100.0
					}
				}
			}
		}
	}

	// Parse average latency - reference Rust implementation
	if runtime.GOOS == "windows" {
		// Windows format: "Average = 12ms" or "Average = 12ms, Maximum = 15ms"
		avgRegex := regexp.MustCompile(`Average\s*=\s*(\d+(?:\.\d+)?)\s*ms`)
		if matches := avgRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
			if lat, err := strconv.ParseFloat(matches[1], 64); err == nil {
				latency = &lat
			}
		}
	} else {
		// Linux/macOS format: "min/avg/max/mdev = 1.234/2.345/3.456/0.567 ms"
		// First try to find line containing "avg" or "Average"
		lines := strings.Split(outputStr, "\n")
		var statsLine string
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), "avg") || strings.Contains(line, "Average") {
				statsLine = line
				break
			}
		}

		if statsLine != "" {
			// Try min/avg/max format: "min/avg/max/mdev = 1.234/2.345/3.456/0.567 ms"
			if strings.Contains(statsLine, "/") {
				// Extract the part after "=" or ":"
				parts := strings.Split(statsLine, "=")
				if len(parts) < 2 {
					parts = strings.Split(statsLine, ":")
				}
				if len(parts) >= 2 {
					values := strings.Fields(parts[1])
					if len(values) > 0 {
						// Split by "/" and get the second value (avg)
						nums := strings.Split(values[0], "/")
						if len(nums) >= 2 {
							if lat, err := strconv.ParseFloat(nums[1], 64); err == nil {
								latency = &lat
							}
						}
					}
				}
			} else {
				// Try "Average = Xms" format (macOS sometimes uses this)
				avgRegex := regexp.MustCompile(`Average\s*[=:]\s*(\d+(?:\.\d+)?)\s*ms`)
				if matches := avgRegex.FindStringSubmatch(statsLine); len(matches) > 1 {
					if lat, err := strconv.ParseFloat(matches[1], 64); err == nil {
						latency = &lat
					}
				}
			}
		}

		// Fallback: find all numbers followed by "ms" and take the last one (usually average)
		if latency == nil {
			msRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*ms`)
			matches := msRegex.FindAllStringSubmatch(outputStr, -1)
			if len(matches) > 0 {
				// Take the last match (usually the average in summary)
				if lat, err := strconv.ParseFloat(matches[len(matches)-1][1], 64); err == nil {
					latency = &lat
				}
			}
		}
	}

	// Determine status
	if packetLoss >= 100.0 {
		status = "timeout"
	} else if latency == nil && packetLoss > 0 {
		status = "error"
	} else if latency == nil {
		status = "error"
	}

	return latency, packetLoss, status
}
