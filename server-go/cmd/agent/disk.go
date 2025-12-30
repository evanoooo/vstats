package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
)

// collectPhysicalDisks collects physical disk information with IO speed
func collectPhysicalDisks(currentIO map[string]disk.IOCountersStat, lastIO map[string]disk.IOCountersStat, lastTime time.Time) []DiskMetrics {
	var disks []DiskMetrics

	switch runtime.GOOS {
	case "linux":
		// Read from /sys/block to get physical disks
		entries, err := os.ReadDir("/sys/block")
		if err == nil {
			physicalDisks := make(map[string]*DiskMetrics)
			for _, entry := range entries {
				name := entry.Name()
				// Skip virtual devices
				if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") ||
					strings.HasPrefix(name, "dm-") || strings.HasPrefix(name, "sr") ||
					strings.HasPrefix(name, "fd") {
					continue
				}

				// Get disk size
				sizePath := filepath.Join("/sys/block", name, "size")
				sizeData, err := os.ReadFile(sizePath)
				if err != nil {
					continue
				}
				sectors, err := strconv.ParseUint(strings.TrimSpace(string(sizeData)), 10, 64)
				if err != nil || sectors == 0 {
					continue
				}
				total := sectors * 512 // Convert sectors to bytes

				// Get disk type
				diskType := detectDiskType(name)

				// Get model
				modelPath := filepath.Join("/sys/block", name, "device", "model")
				var model string
				if modelData, err := os.ReadFile(modelPath); err == nil {
					model = strings.TrimSpace(string(modelData))
				}

				// Get serial
				serialPath := filepath.Join("/sys/block", name, "device", "serial")
				var serial string
				if serialData, err := os.ReadFile(serialPath); err == nil {
					serial = strings.TrimSpace(string(serialData))
				}

				physicalDisks[name] = &DiskMetrics{
					Name:        name,
					Model:       model,
					Serial:      serial,
					Total:       total,
					DiskType:    diskType,
					MountPoints: []string{},
					Used:        0,
				}
			}

			// Map partitions to physical disks
			partitions, _ := disk.Partitions(false)
			for _, p := range partitions {
				partName := p.Device
				mountPoint := p.Mountpoint

				// Skip special mounts
				if strings.HasPrefix(mountPoint, "/snap") || strings.HasPrefix(mountPoint, "/boot/efi") {
					continue
				}

				// Find base device name
				baseName := strings.TrimPrefix(partName, "/dev/")
				if strings.Contains(baseName, "nvme") {
					// NVMe: nvme0n1p1 -> nvme0n1
					baseName = strings.Split(baseName, "p")[0]
				} else {
					// SATA/SCSI/VirtIO: sda1 -> sda, vda3 -> vda, xvda1 -> xvda
					// Use regex to extract letters followed by optional letters/digits before the partition number
					re := regexp.MustCompile(`^([a-z]+[a-z0-9]*?)(\d+)$`)
					if matches := re.FindStringSubmatch(baseName); len(matches) == 3 {
						baseName = matches[1]
					} else {
						// Fallback: extract non-digit prefix
						baseName = regexp.MustCompile(`^([^0-9]+)`).FindString(baseName)
					}
				}

				if diskMetrics, ok := physicalDisks[baseName]; ok {
					if mountPoint != "" && mountPoint != "none" {
						diskMetrics.MountPoints = append(diskMetrics.MountPoints, mountPoint)
					}
					// Update usage from partition
					if usage, err := disk.Usage(p.Mountpoint); err == nil {
						partUsed := usage.Total - usage.Free
						diskMetrics.Used += partUsed
					}
				}
			}

			// Calculate usage percent and IO speed, then convert to slice
			elapsed := time.Since(lastTime).Seconds()
			for _, d := range physicalDisks {
				if d.Total > 0 {
					d.UsagePercent = float32(float64(d.Used) / float64(d.Total) * 100)
				}

				// Calculate IO speed for this disk
				// On Linux, /proc/diskstats contains both physical disks (sda, nvme0n1) and partitions (sda1, nvme0n1p1)
				// Physical disk stats already include all partition IO, so we can directly use the physical disk stats
				// This is the same approach used by nmon and iotop
				if elapsed > 0.1 && len(currentIO) > 0 {
					var readSpeed, writeSpeed uint64

					// First, try to find exact match for physical disk (e.g., "sda", "nvme0n1")
					if io, ok := currentIO[d.Name]; ok {
						if lastIOStat, ok := lastIO[d.Name]; ok {
							readDiff := io.ReadBytes - lastIOStat.ReadBytes
							writeDiff := io.WriteBytes - lastIOStat.WriteBytes
							if io.ReadBytes >= lastIOStat.ReadBytes {
								readSpeed = uint64(float64(readDiff) / elapsed)
							}
							if io.WriteBytes >= lastIOStat.WriteBytes {
								writeSpeed = uint64(float64(writeDiff) / elapsed)
							}
						}
					} else {
						// If exact match not found, aggregate partition stats as fallback
						// This handles cases where gopsutil might only return partition-level stats
						for ioName, io := range currentIO {
							belongsToDisk := false

							if strings.HasPrefix(d.Name, "nvme") {
								// NVMe: nvme0n1p1 -> nvme0n1
								baseName := strings.Split(ioName, "p")[0]
								if baseName == d.Name {
									belongsToDisk = true
								}
							} else {
								// SATA/SCSI/VirtIO: sda1 -> sda, vda3 -> vda
								re := regexp.MustCompile(`^([a-z]+[a-z0-9]*?)(\d+)$`)
								var baseName string
								if matches := re.FindStringSubmatch(ioName); len(matches) == 3 {
									baseName = matches[1]
								} else {
									baseName = regexp.MustCompile(`^([^0-9]+)`).FindString(ioName)
								}
								if baseName == d.Name {
									belongsToDisk = true
								}
							}

							if belongsToDisk {
								if lastIOStat, ok := lastIO[ioName]; ok {
									readDiff := io.ReadBytes - lastIOStat.ReadBytes
									writeDiff := io.WriteBytes - lastIOStat.WriteBytes
									if io.ReadBytes >= lastIOStat.ReadBytes {
										readSpeed += uint64(float64(readDiff) / elapsed)
									}
									if io.WriteBytes >= lastIOStat.WriteBytes {
										writeSpeed += uint64(float64(writeDiff) / elapsed)
									}
								}
							}
						}
					}

					d.ReadSpeed = readSpeed
					d.WriteSpeed = writeSpeed
				}

				disks = append(disks, *d)
			}
		}
	case "darwin":
		// Use diskutil or fallback to partitions
		partitions, _ := disk.Partitions(false)
		physicalDisks := make(map[string]*DiskMetrics)
		for _, p := range partitions {
			name := p.Device
			mount := p.Mountpoint

			// Skip system volumes
			if strings.HasPrefix(mount, "/System") || strings.Contains(name, "synthesized") {
				continue
			}

			usage, err := disk.Usage(mount)
			if err != nil {
				continue
			}

			diskName := strings.TrimPrefix(name, "/dev/")
			if _, exists := physicalDisks[diskName]; !exists {
				physicalDisks[diskName] = &DiskMetrics{
					Name:         diskName,
					Total:        usage.Total,
					Used:         usage.Used,
					UsagePercent: float32(usage.UsedPercent),
					DiskType:     "SSD", // Most Macs use SSD
					MountPoints:  []string{mount},
				}
			}
		}
		// Calculate IO speed for macOS disks
		// On macOS, gopsutil uses I/O Kit framework (similar to Linux's /proc/diskstats)
		// Device names format: "disk0s1", "disk1s2" (disk0 = physical disk, s1 = partition)
		// Similar to Linux: we aggregate partition IO stats to get physical disk speed
		// This is the same approach used by Activity Monitor and iostat command
		elapsed := time.Since(lastTime).Seconds()
		for _, d := range physicalDisks {
			if elapsed > 0.1 && len(currentIO) > 0 {
				// Extract base disk name: "disk0s1" -> "disk0"
				baseDiskName := strings.Split(d.Name, "s")[0]
				var readSpeed, writeSpeed uint64

				// Aggregate IO stats from all partitions belonging to this physical disk
				for ioName, io := range currentIO {
					ioBaseName := strings.Split(ioName, "s")[0]
					if ioBaseName == baseDiskName {
						if lastIOStat, ok := lastIO[ioName]; ok {
							readDiff := io.ReadBytes - lastIOStat.ReadBytes
							writeDiff := io.WriteBytes - lastIOStat.WriteBytes
							if io.ReadBytes >= lastIOStat.ReadBytes {
								readSpeed += uint64(float64(readDiff) / elapsed)
							}
							if io.WriteBytes >= lastIOStat.WriteBytes {
								writeSpeed += uint64(float64(writeDiff) / elapsed)
							}
						}
					}
				}

				d.ReadSpeed = readSpeed
				d.WriteSpeed = writeSpeed
			}
			disks = append(disks, *d)
		}
	case "windows":
		// Use WMIC to get physical disks
		cmd := exec.Command("wmic", "diskdrive", "get", "DeviceID,Model,SerialNumber,Size,MediaType", "/format:csv")
		output, err := cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			firstLine := true
			physicalDisks := make(map[string]*DiskMetrics)
			for scanner.Scan() {
				if firstLine {
					firstLine = false
					continue
				}
				line := scanner.Text()
				parts := strings.Split(line, ",")
				if len(parts) >= 5 {
					deviceID := strings.TrimSpace(parts[1])
					model := strings.TrimSpace(parts[2])
					serial := strings.TrimSpace(parts[4])
					size, _ := strconv.ParseUint(strings.TrimSpace(parts[5]), 10, 64)
					mediaType := strings.TrimSpace(parts[3])

					if size > 0 {
						var diskType string
						if strings.Contains(mediaType, "SSD") || strings.Contains(mediaType, "Solid") {
							diskType = "SSD"
						} else if strings.Contains(mediaType, "HDD") || strings.Contains(mediaType, "Fixed") {
							diskType = "HDD"
						}

						name := strings.ReplaceAll(deviceID, "\\\\.\\", "")
						physicalDisks[name] = &DiskMetrics{
							Name:        name,
							Model:       model,
							Serial:      serial,
							Total:       size,
							DiskType:    diskType,
							MountPoints: []string{},
							Used:        0,
						}
					}
				}
			}

			// Get usage from partitions
			partitions, _ := disk.Partitions(false)
			for _, p := range partitions {
				mount := p.Mountpoint
				if mount != "" {
					if usage, err := disk.Usage(mount); err == nil {
						// On Windows, report partition usage directly if no physical disks found
						if len(physicalDisks) == 0 {
							disks = append(disks, DiskMetrics{
								Name:         mount,
								Total:        usage.Total,
								Used:         usage.Used,
								UsagePercent: float32(usage.UsedPercent),
								DiskType:     "SSD",
								MountPoints:  []string{mount},
							})
						}
					}
				}
			}

			// Calculate usage percent and IO speed for physical disks
			elapsed := time.Since(lastTime).Seconds()
			for _, d := range physicalDisks {
				if d.Total > 0 {
					d.UsagePercent = float32(float64(d.Used) / float64(d.Total) * 100)
				}

				// Calculate IO speed for Windows disks
				// On Windows, gopsutil uses WMI Performance Counters (similar to Linux's /proc/diskstats)
				// Device names format: "C:", "D:" (partition-level, not physical disk)
				// Windows Performance Counters provide partition-level IO stats
				// Note: Windows disk mapping is complex - partitions can span multiple physical disks
				// This is a simplified approach that aggregates all partition IO
				// Similar approach used by Resource Monitor (resmon.exe) and Performance Monitor
				if elapsed > 0.1 && len(currentIO) > 0 {
					var readSpeed, writeSpeed uint64

					// Aggregate IO from all partitions
					// Note: This is simplified - ideally we'd map partitions to physical disks
					// but Windows disk mapping requires WMI queries which is complex
					for ioName, io := range currentIO {
						if lastIOStat, ok := lastIO[ioName]; ok {
							readDiff := io.ReadBytes - lastIOStat.ReadBytes
							writeDiff := io.WriteBytes - lastIOStat.WriteBytes
							if io.ReadBytes >= lastIOStat.ReadBytes {
								readSpeed += uint64(float64(readDiff) / elapsed)
							}
							if io.WriteBytes >= lastIOStat.WriteBytes {
								writeSpeed += uint64(float64(writeDiff) / elapsed)
							}
						}
					}

					d.ReadSpeed = readSpeed
					d.WriteSpeed = writeSpeed
				}

				disks = append(disks, *d)
			}
		}
	}

	return disks
}

// detectDiskType detects if a disk is SSD, HDD, or NVMe
func detectDiskType(diskName string) string {
	switch runtime.GOOS {
	case "linux":
		// NVMe detection: check by name first
		if strings.HasPrefix(diskName, "nvme") {
			return "NVMe"
		}

		// Additional NVMe detection: check scheduler (NVMe uses "none" scheduler)
		schedulerPath := filepath.Join("/sys/block", diskName, "queue", "scheduler")
		if schedulerData, err := os.ReadFile(schedulerPath); err == nil {
			scheduler := strings.TrimSpace(string(schedulerData))
			// NVMe devices typically use "none" scheduler and show it as [none]
			if strings.Contains(scheduler, "[none]") {
				// Double check it's not a virtual device or VirtIO disk
				if !strings.HasPrefix(diskName, "loop") && !strings.HasPrefix(diskName, "ram") &&
					!strings.HasPrefix(diskName, "dm-") && !strings.HasPrefix(diskName, "vd") &&
					!strings.HasPrefix(diskName, "xvd") {
					return "NVMe"
				}
			}
		}

		// Check device class to confirm NVMe
		classPath := filepath.Join("/sys/block", diskName, "device", "class")
		if classData, err := os.ReadFile(classPath); err == nil {
			class := strings.TrimSpace(string(classData))
			// NVMe devices have class 0x010802 (block storage, NVMe)
			if class == "0x010802" {
				return "NVMe"
			}
		}

		// Check rotational flag: 0 = SSD (SATA/NVMe), 1 = HDD
		rotationalPath := filepath.Join("/sys/block", diskName, "queue", "rotational")
		if data, err := os.ReadFile(rotationalPath); err == nil {
			rotational := strings.TrimSpace(string(data))
			if rotational == "0" {
				// Non-rotational, but not NVMe (already checked above), so it's a SATA SSD
				return "SSD"
			} else if rotational == "1" {
				return "HDD"
			}
		}

		// Fallback: check if it's a virtual device
		if strings.HasPrefix(diskName, "vd") || strings.HasPrefix(diskName, "xvd") {
			return "SSD" // Virtual disks are usually backed by SSDs
		}

	case "darwin":
		// macOS: Check if it's NVMe by device name pattern
		// NVMe devices on macOS typically appear as disk0, disk1, etc.
		// We can check using diskutil info, but for simplicity, we'll use a heuristic
		// Most modern Macs use NVMe SSDs, but we can't easily distinguish without diskutil
		// For now, return "SSD" as fallback (most Macs use SSD)
		return "SSD"

	case "windows":
		// Windows detection is handled in collectPhysicalDisks using WMIC
		// This function is not typically called for Windows in the current implementation
		return ""
	}

	return ""
}
