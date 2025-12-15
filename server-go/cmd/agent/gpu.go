package main

import (
	"bufio"
	"encoding/csv"
	"encoding/xml"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// collectGPUMetrics collects GPU metrics from NVIDIA and AMD GPUs
func collectGPUMetrics() *GPUMetrics {
	var gpus []GPU

	// Try NVIDIA first
	nvidiaGPUs := collectNvidiaGPUs()
	gpus = append(gpus, nvidiaGPUs...)

	// Try AMD
	amdGPUs := collectAMDGPUs()
	gpus = append(gpus, amdGPUs...)

	// Try Intel (integrated GPUs)
	intelGPUs := collectIntelGPUs()
	gpus = append(gpus, intelGPUs...)

	if len(gpus) == 0 {
		return nil
	}

	return &GPUMetrics{
		GPUs: gpus,
	}
}

// ============================================================================
// NVIDIA GPU Collection (nvidia-smi)
// ============================================================================

// NvidiaSmiOutput represents the XML output from nvidia-smi -q -x
type NvidiaSmiOutput struct {
	XMLName       xml.Name   `xml:"nvidia_smi_log"`
	DriverVersion string     `xml:"driver_version"`
	CUDAVersion   string     `xml:"cuda_version"`
	GPUs          []NvidiaGPU `xml:"gpu"`
}

type NvidiaGPU struct {
	ID            string `xml:"id,attr"`
	ProductName   string `xml:"product_name"`
	PCIBus        string `xml:"pci>pci_bus_id"`
	FanSpeed      string `xml:"fan_speed"`
	Temperature   struct {
		Current string `xml:"gpu_temp"`
	} `xml:"temperature"`
	Utilization struct {
		GPU     string `xml:"gpu_util"`
		Memory  string `xml:"memory_util"`
		Encoder string `xml:"encoder_util"`
		Decoder string `xml:"decoder_util"`
	} `xml:"utilization"`
	Memory struct {
		Total string `xml:"total"`
		Used  string `xml:"used"`
		Free  string `xml:"free"`
	} `xml:"fb_memory_usage"`
	Power struct {
		Draw  string `xml:"power_draw"`
		Limit string `xml:"power_limit"`
	} `xml:"gpu_power_readings"`
	Clocks struct {
		Graphics string `xml:"graphics_clock"`
		Memory   string `xml:"mem_clock"`
	} `xml:"clocks"`
}

func collectNvidiaGPUs() []GPU {
	var gpus []GPU

	// Check if nvidia-smi is available
	nvidiaSmiPath, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return gpus
	}

	// Try XML format first (more detailed)
	cmd := exec.Command(nvidiaSmiPath, "-q", "-x")
	output, err := cmd.Output()
	if err == nil {
		gpus = parseNvidiaXML(output)
		if len(gpus) > 0 {
			return gpus
		}
	}

	// Fallback to CSV format
	gpus = collectNvidiaCSV(nvidiaSmiPath)
	return gpus
}

func parseNvidiaXML(data []byte) []GPU {
	var gpus []GPU
	var smiOutput NvidiaSmiOutput

	if err := xml.Unmarshal(data, &smiOutput); err != nil {
		return gpus
	}

	for i, nGPU := range smiOutput.GPUs {
		gpu := GPU{
			Index:         i,
			Name:          nGPU.ProductName,
			Vendor:        "NVIDIA",
			DriverVersion: smiOutput.DriverVersion,
			CUDAVersion:   smiOutput.CUDAVersion,
			PCIBus:        nGPU.PCIBus,
		}

		// Parse memory
		gpu.MemoryTotal = parseNvidiaMemory(nGPU.Memory.Total)
		gpu.MemoryUsed = parseNvidiaMemory(nGPU.Memory.Used)
		if gpu.MemoryTotal > 0 {
			gpu.MemoryPercent = float32(gpu.MemoryUsed) / float32(gpu.MemoryTotal) * 100
		}

		// Parse utilization
		gpu.Utilization = parseNvidiaPercent(nGPU.Utilization.GPU)
		gpu.EncoderUtil = parseNvidiaPercent(nGPU.Utilization.Encoder)
		gpu.DecoderUtil = parseNvidiaPercent(nGPU.Utilization.Decoder)

		// Parse temperature
		gpu.Temperature = parseNvidiaTemp(nGPU.Temperature.Current)

		// Parse fan speed
		gpu.FanSpeed = int(parseNvidiaPercent(nGPU.FanSpeed))

		// Parse power
		gpu.PowerDraw = parseNvidiaPower(nGPU.Power.Draw)
		gpu.PowerLimit = parseNvidiaPower(nGPU.Power.Limit)

		// Parse clocks
		gpu.ClockCore = parseNvidiaClock(nGPU.Clocks.Graphics)
		gpu.ClockMemory = parseNvidiaClock(nGPU.Clocks.Memory)

		gpus = append(gpus, gpu)
	}

	return gpus
}

func collectNvidiaCSV(nvidiaSmiPath string) []GPU {
	var gpus []GPU

	// Use CSV format for simpler parsing
	cmd := exec.Command(nvidiaSmiPath,
		"--query-gpu=index,name,memory.total,memory.used,utilization.gpu,temperature.gpu,fan.speed,power.draw,power.limit,clocks.gr,clocks.mem,driver_version,pci.bus_id",
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	reader := csv.NewReader(strings.NewReader(string(output)))
	records, err := reader.ReadAll()
	if err != nil {
		return gpus
	}

	for _, record := range records {
		if len(record) < 13 {
			continue
		}

		idx, _ := strconv.Atoi(strings.TrimSpace(record[0]))
		memTotal, _ := strconv.ParseUint(strings.TrimSpace(record[2]), 10, 64)
		memUsed, _ := strconv.ParseUint(strings.TrimSpace(record[3]), 10, 64)
		util, _ := strconv.ParseFloat(strings.TrimSpace(record[4]), 32)
		temp, _ := strconv.Atoi(strings.TrimSpace(record[5]))
		fan, _ := strconv.Atoi(strings.TrimSpace(record[6]))
		powerDraw, _ := strconv.ParseFloat(strings.TrimSpace(record[7]), 32)
		powerLimit, _ := strconv.ParseFloat(strings.TrimSpace(record[8]), 32)
		clockCore, _ := strconv.ParseUint(strings.TrimSpace(record[9]), 10, 32)
		clockMem, _ := strconv.ParseUint(strings.TrimSpace(record[10]), 10, 32)

		gpu := GPU{
			Index:         idx,
			Name:          strings.TrimSpace(record[1]),
			Vendor:        "NVIDIA",
			MemoryTotal:   memTotal * 1024 * 1024, // Convert MiB to bytes
			MemoryUsed:    memUsed * 1024 * 1024,
			Utilization:   float32(util),
			Temperature:   temp,
			FanSpeed:      fan,
			PowerDraw:     float32(powerDraw),
			PowerLimit:    float32(powerLimit),
			ClockCore:     uint32(clockCore),
			ClockMemory:   uint32(clockMem),
			DriverVersion: strings.TrimSpace(record[11]),
			PCIBus:        strings.TrimSpace(record[12]),
		}

		if gpu.MemoryTotal > 0 {
			gpu.MemoryPercent = float32(gpu.MemoryUsed) / float32(gpu.MemoryTotal) * 100
		}

		gpus = append(gpus, gpu)
	}

	return gpus
}

// Helper functions for parsing nvidia-smi XML values
func parseNvidiaMemory(s string) uint64 {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	
	var multiplier uint64 = 1
	if strings.HasSuffix(s, " mib") {
		s = strings.TrimSuffix(s, " mib")
		multiplier = 1024 * 1024
	} else if strings.HasSuffix(s, " gib") {
		s = strings.TrimSuffix(s, " gib")
		multiplier = 1024 * 1024 * 1024
	} else if strings.HasSuffix(s, " mb") {
		s = strings.TrimSuffix(s, " mb")
		multiplier = 1000 * 1000
	} else if strings.HasSuffix(s, " gb") {
		s = strings.TrimSuffix(s, " gb")
		multiplier = 1000 * 1000 * 1000
	}

	val, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return val * multiplier
}

func parseNvidiaPercent(s string) float32 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, " %")
	s = strings.TrimSuffix(s, "%")
	val, _ := strconv.ParseFloat(strings.TrimSpace(s), 32)
	return float32(val)
}

func parseNvidiaTemp(s string) int {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, " C")
	s = strings.TrimSuffix(s, "C")
	val, _ := strconv.Atoi(strings.TrimSpace(s))
	return val
}

func parseNvidiaPower(s string) float32 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, " W")
	s = strings.TrimSuffix(s, "W")
	val, _ := strconv.ParseFloat(strings.TrimSpace(s), 32)
	return float32(val)
}

func parseNvidiaClock(s string) uint32 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, " MHz")
	s = strings.TrimSuffix(s, "MHz")
	val, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	return uint32(val)
}

// ============================================================================
// AMD GPU Collection (rocm-smi / radeontop)
// ============================================================================

func collectAMDGPUs() []GPU {
	var gpus []GPU

	// Try rocm-smi first (for ROCm-enabled GPUs)
	gpus = collectAMDRocmSMI()
	if len(gpus) > 0 {
		return gpus
	}

	// Try amdgpu via sysfs (Linux)
	if runtime.GOOS == "linux" {
		gpus = collectAMDSysfs()
	}

	return gpus
}

func collectAMDRocmSMI() []GPU {
	var gpus []GPU

	rocmSmiPath, err := exec.LookPath("rocm-smi")
	if err != nil {
		return gpus
	}

	// Get GPU info using rocm-smi with CSV-like output
	cmd := exec.Command(rocmSmiPath, "--showid", "--showtemp", "--showuse", "--showmeminfo", "vram", "--showpower", "--showclocks", "--csv")
	output, err := cmd.Output()
	if err != nil {
		// Try simpler format
		return collectAMDRocmSMISimple(rocmSmiPath)
	}

	reader := csv.NewReader(strings.NewReader(string(output)))
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return collectAMDRocmSMISimple(rocmSmiPath)
	}

	// Parse header to find column indices
	header := records[0]
	colIdx := make(map[string]int)
	for i, col := range header {
		colIdx[strings.ToLower(strings.TrimSpace(col))] = i
	}

	for i, record := range records[1:] {
		gpu := GPU{
			Index:  i,
			Vendor: "AMD",
		}

		// Extract values based on header positions
		if idx, ok := colIdx["gpu"]; ok && idx < len(record) {
			gpu.Name = strings.TrimSpace(record[idx])
		}
		if idx, ok := colIdx["temperature"]; ok && idx < len(record) {
			temp, _ := strconv.Atoi(strings.TrimSpace(record[idx]))
			gpu.Temperature = temp
		}
		if idx, ok := colIdx["gpu use (%)"]; ok && idx < len(record) {
			util, _ := strconv.ParseFloat(strings.TrimSpace(record[idx]), 32)
			gpu.Utilization = float32(util)
		}
		if idx, ok := colIdx["vram total"]; ok && idx < len(record) {
			mem, _ := strconv.ParseUint(strings.TrimSpace(record[idx]), 10, 64)
			gpu.MemoryTotal = mem
		}
		if idx, ok := colIdx["vram used"]; ok && idx < len(record) {
			mem, _ := strconv.ParseUint(strings.TrimSpace(record[idx]), 10, 64)
			gpu.MemoryUsed = mem
		}
		if idx, ok := colIdx["power (w)"]; ok && idx < len(record) {
			power, _ := strconv.ParseFloat(strings.TrimSpace(record[idx]), 32)
			gpu.PowerDraw = float32(power)
		}

		if gpu.MemoryTotal > 0 {
			gpu.MemoryPercent = float32(gpu.MemoryUsed) / float32(gpu.MemoryTotal) * 100
		}

		gpus = append(gpus, gpu)
	}

	return gpus
}

func collectAMDRocmSMISimple(rocmSmiPath string) []GPU {
	var gpus []GPU

	// Get basic info
	cmd := exec.Command(rocmSmiPath, "-a")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var currentGPU *GPU
	gpuIndex := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "GPU[") {
			if currentGPU != nil && currentGPU.Name != "" {
				gpus = append(gpus, *currentGPU)
			}
			currentGPU = &GPU{
				Index:  gpuIndex,
				Vendor: "AMD",
			}
			gpuIndex++
		}

		if currentGPU == nil {
			continue
		}

		if strings.Contains(line, "GPU Temperature") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				temp := strings.TrimSpace(parts[1])
				temp = strings.TrimSuffix(temp, "c")
				temp = strings.TrimSuffix(temp, "C")
				t, _ := strconv.Atoi(strings.TrimSpace(temp))
				currentGPU.Temperature = t
			}
		}

		if strings.Contains(line, "GPU use (%)") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				util := strings.TrimSpace(parts[1])
				util = strings.TrimSuffix(util, "%")
				u, _ := strconv.ParseFloat(strings.TrimSpace(util), 32)
				currentGPU.Utilization = float32(u)
			}
		}

		if strings.Contains(line, "VRAM Total") || strings.Contains(line, "vram Total") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				mem := parseAMDMemory(parts[1])
				currentGPU.MemoryTotal = mem
			}
		}

		if strings.Contains(line, "VRAM Used") || strings.Contains(line, "vram Used") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				mem := parseAMDMemory(parts[1])
				currentGPU.MemoryUsed = mem
			}
		}

		if strings.Contains(line, "Card series:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				currentGPU.Name = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "Power (Watts)") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				power, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 32)
				currentGPU.PowerDraw = float32(power)
			}
		}
	}

	if currentGPU != nil && currentGPU.Name != "" {
		if currentGPU.MemoryTotal > 0 {
			currentGPU.MemoryPercent = float32(currentGPU.MemoryUsed) / float32(currentGPU.MemoryTotal) * 100
		}
		gpus = append(gpus, *currentGPU)
	}

	return gpus
}

func parseAMDMemory(s string) uint64 {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	var multiplier uint64 = 1
	if strings.HasSuffix(s, "mb") || strings.HasSuffix(s, "m") {
		s = strings.TrimSuffix(s, "mb")
		s = strings.TrimSuffix(s, "m")
		multiplier = 1024 * 1024
	} else if strings.HasSuffix(s, "gb") || strings.HasSuffix(s, "g") {
		s = strings.TrimSuffix(s, "gb")
		s = strings.TrimSuffix(s, "g")
		multiplier = 1024 * 1024 * 1024
	} else if strings.HasSuffix(s, "kb") || strings.HasSuffix(s, "k") {
		s = strings.TrimSuffix(s, "kb")
		s = strings.TrimSuffix(s, "k")
		multiplier = 1024
	}

	val, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return val * multiplier
}

func collectAMDSysfs() []GPU {
	var gpus []GPU

	// Look for AMD GPU in /sys/class/drm/
	// This is a basic implementation that reads from sysfs
	// A more complete implementation would parse hwmon data

	// Try to find AMD GPU cards
	cmd := exec.Command("ls", "/sys/class/drm/")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	cardDirs := strings.Fields(string(output))
	gpuIndex := 0

	for _, cardDir := range cardDirs {
		if !strings.HasPrefix(cardDir, "card") || strings.Contains(cardDir, "-") {
			continue
		}

		// Check if it's an AMD GPU by looking for amdgpu driver
		driverPath := "/sys/class/drm/" + cardDir + "/device/driver"
		driverLink, err := exec.Command("readlink", "-f", driverPath).Output()
		if err != nil {
			continue
		}

		if !strings.Contains(string(driverLink), "amdgpu") {
			continue
		}

		gpu := GPU{
			Index:  gpuIndex,
			Vendor: "AMD",
			Name:   "AMD GPU",
		}

		basePath := "/sys/class/drm/" + cardDir + "/device"

		// Try to get GPU name
		if vendorData, err := exec.Command("cat", basePath+"/vendor").Output(); err == nil {
			if strings.TrimSpace(string(vendorData)) == "0x1002" {
				gpu.Vendor = "AMD"
			}
		}

		// Get VRAM info from hwmon or mem_info_vram
		if vramTotal, err := exec.Command("cat", basePath+"/mem_info_vram_total").Output(); err == nil {
			mem, _ := strconv.ParseUint(strings.TrimSpace(string(vramTotal)), 10, 64)
			gpu.MemoryTotal = mem
		}
		if vramUsed, err := exec.Command("cat", basePath+"/mem_info_vram_used").Output(); err == nil {
			mem, _ := strconv.ParseUint(strings.TrimSpace(string(vramUsed)), 10, 64)
			gpu.MemoryUsed = mem
		}

		// Get GPU utilization from gpu_busy_percent
		if gpuBusy, err := exec.Command("cat", basePath+"/gpu_busy_percent").Output(); err == nil {
			util, _ := strconv.ParseFloat(strings.TrimSpace(string(gpuBusy)), 32)
			gpu.Utilization = float32(util)
		}

		// Look for hwmon for temperature
		hwmonDirs, _ := exec.Command("ls", basePath+"/hwmon/").Output()
		if len(hwmonDirs) > 0 {
			hwmonDir := strings.Fields(string(hwmonDirs))[0]
			hwmonPath := basePath + "/hwmon/" + hwmonDir

			// Temperature
			if temp, err := exec.Command("cat", hwmonPath+"/temp1_input").Output(); err == nil {
				t, _ := strconv.Atoi(strings.TrimSpace(string(temp)))
				gpu.Temperature = t / 1000 // millidegrees to degrees
			}

			// Power
			if power, err := exec.Command("cat", hwmonPath+"/power1_average").Output(); err == nil {
				p, _ := strconv.ParseFloat(strings.TrimSpace(string(power)), 32)
				gpu.PowerDraw = float32(p) / 1000000 // microwatts to watts
			}
		}

		if gpu.MemoryTotal > 0 {
			gpu.MemoryPercent = float32(gpu.MemoryUsed) / float32(gpu.MemoryTotal) * 100
		}

		gpus = append(gpus, gpu)
		gpuIndex++
	}

	return gpus
}

// ============================================================================
// Intel GPU Collection (intel_gpu_top)
// ============================================================================

func collectIntelGPUs() []GPU {
	var gpus []GPU

	if runtime.GOOS != "linux" {
		return gpus
	}

	// Check for Intel integrated graphics via sysfs
	cmd := exec.Command("ls", "/sys/class/drm/")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	cardDirs := strings.Fields(string(output))
	gpuIndex := 0

	for _, cardDir := range cardDirs {
		if !strings.HasPrefix(cardDir, "card") || strings.Contains(cardDir, "-") {
			continue
		}

		basePath := "/sys/class/drm/" + cardDir + "/device"

		// Check if it's an Intel GPU
		vendorData, err := exec.Command("cat", basePath+"/vendor").Output()
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(vendorData)) != "0x8086" {
			continue
		}

		gpu := GPU{
			Index:  gpuIndex,
			Vendor: "Intel",
			Name:   "Intel GPU",
		}

		// Try to get model name
		if deviceData, err := exec.Command("cat", basePath+"/device").Output(); err == nil {
			gpu.Name = "Intel GPU (" + strings.TrimSpace(string(deviceData)) + ")"
		}

		// Intel GPUs share system memory, so memory stats are not directly available
		// We could parse /proc/meminfo for video memory but it's not accurate

		gpus = append(gpus, gpu)
		gpuIndex++
	}

	return gpus
}
