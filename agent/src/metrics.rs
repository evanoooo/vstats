use chrono::Utc;
use sysinfo::{CpuRefreshKind, Disks, Networks, System};
use std::time::Duration;
use std::process::Command;
use std::sync::{Arc, Mutex};
use std::thread;

use crate::types::{
    CpuMetrics, DiskMetrics, LoadAverage, MemoryMetrics, MemoryModule, NetworkInterface, NetworkMetrics,
    OsInfo, SystemMetrics, PingMetrics, PingTarget, PingTargetConfig,
};

/// Default ping targets for latency monitoring
const DEFAULT_PING_TARGETS: &[(&str, &str)] = &[
    ("Google DNS", "8.8.8.8"),
    ("Cloudflare", "1.1.1.1"),
    ("Local Gateway", ""),  // Will be detected
];

/// Metrics collector that maintains state for accurate CPU measurements
pub struct MetricsCollector {
    sys: System,
    disks: Disks,
    networks: Networks,
    hostname: String,
    os_info: OsInfo,
    // Track previous network readings for speed calculation
    last_network_rx: u64,
    last_network_tx: u64,
    last_network_time: std::time::Instant,
    // Ping metrics (updated in background)
    ping_results: Arc<Mutex<Option<PingMetrics>>>,
    #[allow(dead_code)] // Used during initialization for background thread
    gateway_ip: Option<String>,
    // Cached IP addresses
    ip_addresses: Vec<String>,
    // Custom ping targets from server config
    custom_ping_targets: Arc<Mutex<Option<Vec<PingTargetConfig>>>>,
}

impl MetricsCollector {
    pub fn new() -> Self {
        let mut sys = System::new_all();
        
        // Initial CPU refresh to get baseline
        sys.refresh_cpu_specifics(CpuRefreshKind::everything());
        std::thread::sleep(Duration::from_millis(200));
        sys.refresh_cpu_specifics(CpuRefreshKind::everything());
        
        let hostname = System::host_name().unwrap_or_else(|| "unknown".to_string());
        
        let os_info = OsInfo {
            name: System::name().unwrap_or_else(|| "Unknown".to_string()),
            version: System::os_version().unwrap_or_else(|| "Unknown".to_string()),
            kernel: System::kernel_version().unwrap_or_else(|| "Unknown".to_string()),
            arch: std::env::consts::ARCH.to_string(),
        };
        
        let networks = Networks::new_with_refreshed_list();
        
        // Get initial network totals
        let (init_rx, init_tx) = networks.iter().fold((0u64, 0u64), |(rx, tx), (_, data)| {
            (rx.saturating_add(data.total_received()), tx.saturating_add(data.total_transmitted()))
        });
        
        // Detect default gateway
        let gateway_ip = Self::detect_gateway();
        
        // Initialize ping results
        let ping_results = Arc::new(Mutex::new(None));
        
        // Initialize custom ping targets
        let custom_ping_targets: Arc<Mutex<Option<Vec<PingTargetConfig>>>> = Arc::new(Mutex::new(None));
        
        // Start background ping thread
        let ping_results_clone = Arc::clone(&ping_results);
        let custom_targets_clone = Arc::clone(&custom_ping_targets);
        let gateway_clone = gateway_ip.clone();
        thread::spawn(move || {
            loop {
                // Check for custom targets from server config
                let custom_targets = custom_targets_clone.lock().ok().and_then(|guard| guard.clone());
                let results = Self::collect_ping_with_targets(&gateway_clone, custom_targets.as_ref());
                if let Ok(mut guard) = ping_results_clone.lock() {
                    *guard = Some(results);
                }
                thread::sleep(Duration::from_secs(10)); // Ping every 10 seconds
            }
        });
        
        // Collect IP addresses
        let ip_addresses = Self::collect_ip_addresses();
        
        Self {
            sys,
            disks: Disks::new_with_refreshed_list(),
            networks,
            hostname,
            os_info,
            last_network_rx: init_rx,
            last_network_tx: init_tx,
            last_network_time: std::time::Instant::now(),
            ping_results,
            gateway_ip,
            ip_addresses,
            custom_ping_targets,
        }
    }
    
    /// Update ping targets from server configuration
    pub fn set_ping_targets(&self, targets: Vec<PingTargetConfig>) {
        if let Ok(mut guard) = self.custom_ping_targets.lock() {
            if targets.is_empty() {
                *guard = None; // Use defaults if empty
            } else {
                *guard = Some(targets);
            }
        }
    }
    
    /// Collect local IP addresses (IPv4 only, excluding loopback)
    fn collect_ip_addresses() -> Vec<String> {
        let mut ips = Vec::new();
        
        #[cfg(target_os = "linux")]
        {
            if let Ok(output) = Command::new("hostname")
                .args(["-I"])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                for ip in stdout.split_whitespace() {
                    // Only include IPv4 addresses
                    if ip.contains('.') && !ip.starts_with("127.") {
                        ips.push(ip.to_string());
                    }
                }
            }
            
            // Fallback to ip addr
            if ips.is_empty() {
                if let Ok(output) = Command::new("ip")
                    .args(["addr", "show"])
                    .output()
                {
                    let stdout = String::from_utf8_lossy(&output.stdout);
                    for line in stdout.lines() {
                        if line.contains("inet ") && !line.contains("127.0.0.1") {
                            if let Some(ip_part) = line.split_whitespace().nth(1) {
                                if let Some(ip) = ip_part.split('/').next() {
                                    ips.push(ip.to_string());
                                }
                            }
                        }
                    }
                }
            }
        }
        
        #[cfg(target_os = "macos")]
        {
            if let Ok(output) = Command::new("ifconfig")
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                for line in stdout.lines() {
                    let line = line.trim();
                    if line.starts_with("inet ") && !line.contains("127.0.0.1") {
                        if let Some(ip) = line.split_whitespace().nth(1) {
                            ips.push(ip.to_string());
                        }
                    }
                }
            }
        }
        
        #[cfg(target_os = "windows")]
        {
            // Try PowerShell first for cleaner output
            if let Ok(output) = Command::new("powershell")
                .args(["-Command", "(Get-NetIPAddress -AddressFamily IPv4 | Where-Object { $_.IPAddress -ne '127.0.0.1' }).IPAddress"])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                for line in stdout.lines() {
                    let ip = line.trim();
                    if !ip.is_empty() && ip.contains('.') && !ip.starts_with("127.") {
                        ips.push(ip.to_string());
                    }
                }
            }
            
            // Fallback to ipconfig
            if ips.is_empty() {
                if let Ok(output) = Command::new("ipconfig")
                    .output()
                {
                    let stdout = String::from_utf8_lossy(&output.stdout);
                    for line in stdout.lines() {
                        if line.contains("IPv4") || line.contains("IP Address") {
                            if let Some(ip_part) = line.split(':').nth(1) {
                                let ip = ip_part.trim();
                                if ip.contains('.') && !ip.starts_with("127.") {
                                    ips.push(ip.to_string());
                                }
                            }
                        }
                    }
                }
            }
        }
        
        ips
    }
    
    /// Detect default gateway IP
    fn detect_gateway() -> Option<String> {
        #[cfg(target_os = "linux")]
        {
            if let Ok(output) = Command::new("ip")
                .args(["route", "show", "default"])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                // Parse: default via 192.168.1.1 dev eth0
                for word in stdout.split_whitespace() {
                    if word.contains('.') && !word.contains('/') {
                        return Some(word.to_string());
                    }
                }
            }
        }
        
        #[cfg(target_os = "macos")]
        {
            if let Ok(output) = Command::new("route")
                .args(["-n", "get", "default"])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                for line in stdout.lines() {
                    if line.trim().starts_with("gateway:") {
                        if let Some(ip) = line.split(':').nth(1) {
                            return Some(ip.trim().to_string());
                        }
                    }
                }
            }
        }
        
        #[cfg(target_os = "windows")]
        {
            // Use 'route print' to get the default gateway
            if let Ok(output) = Command::new("cmd")
                .args(["/C", "route", "print", "0.0.0.0"])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                // Look for lines containing "0.0.0.0" that indicate default route
                for line in stdout.lines() {
                    let parts: Vec<&str> = line.split_whitespace().collect();
                    // Route table format: Network Destination, Netmask, Gateway, Interface, Metric
                    // Look for 0.0.0.0 destination with a valid gateway IP
                    if parts.len() >= 3 && parts[0] == "0.0.0.0" {
                        let gateway = parts[2];
                        // Validate it looks like an IP address
                        if gateway.contains('.') && gateway != "0.0.0.0" {
                            return Some(gateway.to_string());
                        }
                    }
                }
            }
            
            // Alternative: use PowerShell
            if let Ok(output) = Command::new("powershell")
                .args(["-Command", "(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Select-Object -First 1).NextHop"])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
                if !stdout.is_empty() && stdout.contains('.') {
                    return Some(stdout);
                }
            }
        }
        
        None
    }
    
    /// Ping a host and return latency in milliseconds
    fn ping_host(host: &str) -> (Option<f64>, f64, String) {
        #[cfg(target_os = "linux")]
        let args = ["-c", "3", "-W", "2", host];
        
        #[cfg(target_os = "macos")]
        let args = ["-c", "3", "-W", "2000", host];
        
        #[cfg(target_os = "windows")]
        let args = ["-n", "3", "-w", "2000", host];
        
        match Command::new("ping").args(&args).output() {
            Ok(output) => {
                let stdout = String::from_utf8_lossy(&output.stdout);
                
                // Parse packet loss
                let packet_loss = if let Some(loss_line) = stdout.lines()
                    .find(|l| l.contains("packet loss") || l.contains("loss"))
                {
                    // Extract percentage like "0% packet loss" or "33.3% packet loss"
                    loss_line.split_whitespace()
                        .find(|w| w.ends_with('%'))
                        .and_then(|w| w.trim_end_matches('%').parse::<f64>().ok())
                        .unwrap_or(0.0)
                } else {
                    if output.status.success() { 0.0 } else { 100.0 }
                };
                
                // Parse average latency
                let latency = if let Some(stats_line) = stdout.lines()
                    .find(|l| l.contains("avg") || l.contains("Average"))
                {
                    // Format varies: "min/avg/max/mdev = 1.234/2.345/3.456/0.567 ms"
                    // or "Minimum = 1ms, Maximum = 3ms, Average = 2ms"
                    if stats_line.contains('/') {
                        stats_line.split('=').last()
                            .and_then(|s| s.split('/').nth(1))
                            .and_then(|s| s.trim().parse::<f64>().ok())
                    } else {
                        stats_line.split_whitespace()
                            .filter_map(|w| w.trim_end_matches("ms").parse::<f64>().ok())
                            .last()
                    }
                } else {
                    None
                };
                
                let status = if packet_loss >= 100.0 {
                    "timeout".to_string()
                } else {
                    "ok".to_string()
                };
                
                (latency, packet_loss, status)
            }
            Err(_) => (None, 100.0, "error".to_string()),
        }
    }
    
    /// Collect ping metrics with custom targets from server config
    fn collect_ping_with_targets(gateway_ip: &Option<String>, custom_targets: Option<&Vec<PingTargetConfig>>) -> PingMetrics {
        let mut targets = Vec::new();
        
        // Use custom targets if provided, otherwise use defaults
        if let Some(custom) = custom_targets {
            for target in custom {
                if target.host.is_empty() {
                    continue;
                }
                let (latency, packet_loss, status) = Self::ping_host(&target.host);
                targets.push(PingTarget {
                    name: target.name.clone(),
                    host: target.host.clone(),
                    latency_ms: latency,
                    packet_loss,
                    status,
                });
            }
        } else {
            // Fallback to default targets
            for (name, host) in DEFAULT_PING_TARGETS {
                let actual_host = if host.is_empty() {
                    // Use gateway IP if available
                    match gateway_ip {
                        Some(gw) => gw.clone(),
                        None => continue,
                    }
                } else {
                    host.to_string()
                };
                
                let (latency, packet_loss, status) = Self::ping_host(&actual_host);
                
                targets.push(PingTarget {
                    name: name.to_string(),
                    host: actual_host,
                    latency_ms: latency,
                    packet_loss,
                    status,
                });
            }
        }
        
        PingMetrics { targets }
    }
    
    /// Refresh and collect current system metrics
    pub fn collect(&mut self) -> SystemMetrics {
        // Refresh all metrics
        self.sys.refresh_cpu_specifics(CpuRefreshKind::everything());
        self.sys.refresh_memory();
        self.disks.refresh();
        self.networks.refresh();
        
        let network = self.collect_network();
        
        // Get cached ping results
        let ping = self.ping_results.lock().ok().and_then(|guard| guard.clone());
        
        SystemMetrics {
            timestamp: Utc::now(),
            hostname: self.hostname.clone(),
            os: self.os_info.clone(),
            cpu: self.collect_cpu(),
            memory: self.collect_memory(),
            disks: self.collect_disks(),
            network,
            uptime: System::uptime(),
            load_average: self.collect_load_average(),
            ping,
            version: Some(env!("CARGO_PKG_VERSION").to_string()),
            ip_addresses: if self.ip_addresses.is_empty() { None } else { Some(self.ip_addresses.clone()) },
        }
    }
    
    fn collect_cpu(&self) -> CpuMetrics {
        let cpus = self.sys.cpus();
        let global_usage: f32 = cpus.iter().map(|c| c.cpu_usage()).sum::<f32>() / cpus.len() as f32;
        let per_core: Vec<f32> = cpus.iter().map(|c| c.cpu_usage()).collect();
        let frequency = cpus.first().map(|c| c.frequency()).unwrap_or(0);
        let brand = cpus.first()
            .map(|c| c.brand().to_string())
            .unwrap_or_else(|| "Unknown".to_string());
        
        CpuMetrics {
            brand,
            cores: cpus.len(),
            usage: global_usage,
            frequency,
            per_core,
        }
    }
    
    fn collect_memory(&self) -> MemoryMetrics {
        let total = self.sys.total_memory();
        let used = self.sys.used_memory();
        let available = self.sys.available_memory();
        let swap_total = self.sys.total_swap();
        let swap_used = self.sys.used_swap();
        
        let usage_percent = if total > 0 {
            (used as f32 / total as f32) * 100.0
        } else {
            0.0
        };
        
        // Collect memory module details
        let modules = Self::collect_memory_modules();
        
        MemoryMetrics {
            total,
            used,
            available,
            swap_total,
            swap_used,
            usage_percent,
            modules,
        }
    }
    
    /// Collect memory module information (DDR type, frequency, slots)
    fn collect_memory_modules() -> Vec<MemoryModule> {
        let mut modules = Vec::new();
        
        #[cfg(target_os = "linux")]
        {
            // Try dmidecode first (requires root)
            if let Ok(output) = Command::new("dmidecode")
                .args(["-t", "memory"])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                let mut current_module: Option<MemoryModule> = None;
                
                for line in stdout.lines() {
                    let line = line.trim();
                    
                    if line.starts_with("Memory Device") {
                        if let Some(m) = current_module.take() {
                            if m.size > 0 {
                                modules.push(m);
                            }
                        }
                        current_module = Some(MemoryModule {
                            slot: None,
                            size: 0,
                            mem_type: None,
                            speed: None,
                            manufacturer: None,
                        });
                    } else if let Some(ref mut m) = current_module {
                        if let Some(val) = line.strip_prefix("Size:") {
                            let val = val.trim();
                            if val != "No Module Installed" {
                                // Parse "16 GB" or "16384 MB"
                                let parts: Vec<&str> = val.split_whitespace().collect();
                                if parts.len() >= 2 {
                                    if let Ok(num) = parts[0].parse::<u64>() {
                                        m.size = match parts[1].to_uppercase().as_str() {
                                            "GB" => num * 1024 * 1024 * 1024,
                                            "MB" => num * 1024 * 1024,
                                            "KB" => num * 1024,
                                            _ => num,
                                        };
                                    }
                                }
                            }
                        } else if let Some(val) = line.strip_prefix("Type:") {
                            let val = val.trim();
                            if val != "Unknown" && !val.is_empty() {
                                m.mem_type = Some(val.to_string());
                            }
                        } else if let Some(val) = line.strip_prefix("Speed:") {
                            let val = val.trim();
                            if let Some(speed_str) = val.split_whitespace().next() {
                                if let Ok(speed) = speed_str.parse::<u32>() {
                                    m.speed = Some(speed);
                                }
                            }
                        } else if let Some(val) = line.strip_prefix("Locator:") {
                            m.slot = Some(val.trim().to_string());
                        } else if let Some(val) = line.strip_prefix("Manufacturer:") {
                            let val = val.trim();
                            if val != "Unknown" && !val.is_empty() && val != "Not Specified" {
                                m.manufacturer = Some(val.to_string());
                            }
                        }
                    }
                }
                
                if let Some(m) = current_module {
                    if m.size > 0 {
                        modules.push(m);
                    }
                }
            }
            
            // Fallback: read from /proc/meminfo for basic info
            if modules.is_empty() {
                // Can't get detailed info without dmidecode, return empty
            }
        }
        
        #[cfg(target_os = "windows")]
        {
            // Use WMIC to get memory info
            if let Ok(output) = Command::new("wmic")
                .args(["memorychip", "get", "Capacity,Speed,MemoryType,Manufacturer,DeviceLocator", "/format:csv"])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                for line in stdout.lines().skip(1) {
                    let parts: Vec<&str> = line.split(',').collect();
                    if parts.len() >= 5 {
                        let size = parts[1].trim().parse::<u64>().unwrap_or(0);
                        if size > 0 {
                            let mem_type_code = parts[3].trim().parse::<u32>().unwrap_or(0);
                            let mem_type = match mem_type_code {
                                20 => Some("DDR".to_string()),
                                21 => Some("DDR2".to_string()),
                                24 => Some("DDR3".to_string()),
                                26 => Some("DDR4".to_string()),
                                34 => Some("DDR5".to_string()),
                                _ => None,
                            };
                            
                            modules.push(MemoryModule {
                                slot: if parts[2].trim().is_empty() { None } else { Some(parts[2].trim().to_string()) },
                                size,
                                mem_type,
                                speed: parts[4].trim().parse::<u32>().ok(),
                                manufacturer: if parts[3].trim().is_empty() { None } else { Some(parts[3].trim().to_string()) },
                            });
                        }
                    }
                }
            }
        }
        
        #[cfg(target_os = "macos")]
        {
            // Use system_profiler
            if let Ok(output) = Command::new("system_profiler")
                .args(["SPMemoryDataType", "-json"])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                if let Ok(json) = serde_json::from_str::<serde_json::Value>(&stdout) {
                    if let Some(memory_data) = json.get("SPMemoryDataType").and_then(|v| v.as_array()) {
                        for item in memory_data {
                            if let Some(items) = item.get("_items").and_then(|v| v.as_array()) {
                                for module in items {
                                    let size_str = module.get("dimm_size").and_then(|v| v.as_str()).unwrap_or("");
                                    let size = if size_str.contains("GB") {
                                        size_str.replace(" GB", "").trim().parse::<u64>().unwrap_or(0) * 1024 * 1024 * 1024
                                    } else {
                                        0
                                    };
                                    
                                    if size > 0 {
                                        modules.push(MemoryModule {
                                            slot: module.get("_name").and_then(|v| v.as_str()).map(|s| s.to_string()),
                                            size,
                                            mem_type: module.get("dimm_type").and_then(|v| v.as_str()).map(|s| s.to_string()),
                                            speed: module.get("dimm_speed").and_then(|v| v.as_str())
                                                .and_then(|s| s.split_whitespace().next())
                                                .and_then(|s| s.parse::<u32>().ok()),
                                            manufacturer: module.get("dimm_manufacturer").and_then(|v| v.as_str()).map(|s| s.to_string()),
                                        });
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
        
        modules
    }
    
    fn collect_disks(&self) -> Vec<DiskMetrics> {
        // Collect physical disks instead of partitions
        Self::collect_physical_disks(&self.disks)
    }
    
    /// Collect physical disk information
    fn collect_physical_disks(partitions: &Disks) -> Vec<DiskMetrics> {
        let mut physical_disks: std::collections::HashMap<String, DiskMetrics> = std::collections::HashMap::new();
        
        #[cfg(target_os = "linux")]
        {
            // Read from /sys/block to get physical disks
            if let Ok(entries) = std::fs::read_dir("/sys/block") {
                for entry in entries.flatten() {
                    let name = entry.file_name().to_string_lossy().to_string();
                    
                    // Skip virtual devices, loop devices, ram disks
                    if name.starts_with("loop") || name.starts_with("ram") || 
                       name.starts_with("dm-") || name.starts_with("sr") ||
                       name.starts_with("fd") {
                        continue;
                    }
                    
                    // Get disk size
                    let size_path = format!("/sys/block/{}/size", name);
                    let total = std::fs::read_to_string(&size_path)
                        .ok()
                        .and_then(|s| s.trim().parse::<u64>().ok())
                        .map(|sectors| sectors * 512) // Convert sectors to bytes
                        .unwrap_or(0);
                    
                    if total == 0 {
                        continue;
                    }
                    
                    // Get disk type
                    let disk_type = Self::detect_disk_type(&format!("/dev/{}", name));
                    
                    // Get model
                    let model_path = format!("/sys/block/{}/device/model", name);
                    let model = std::fs::read_to_string(&model_path)
                        .ok()
                        .map(|s| s.trim().to_string())
                        .filter(|s| !s.is_empty());
                    
                    // Get serial (may require root)
                    let serial_path = format!("/sys/block/{}/device/serial", name);
                    let serial = std::fs::read_to_string(&serial_path)
                        .ok()
                        .map(|s| s.trim().to_string())
                        .filter(|s| !s.is_empty());
                    
                    physical_disks.insert(name.clone(), DiskMetrics {
                        name: name.clone(),
                        model,
                        serial,
                        total,
                        disk_type,
                        mount_points: Vec::new(),
                        usage_percent: 0.0,
                        used: 0,
                    });
                }
            }
            
            // Map partitions to physical disks and calculate usage
            for partition in partitions.iter() {
                let part_name = partition.name().to_string_lossy().to_string();
                let mount_point = partition.mount_point().to_string_lossy().to_string();
                
                // Skip special mounts
                if mount_point.starts_with("/snap") || mount_point.starts_with("/boot/efi") {
                    continue;
                }
                
                // Find the physical disk for this partition
                // e.g., sda1 -> sda, nvme0n1p1 -> nvme0n1
                let base_name = part_name
                    .trim_start_matches("/dev/")
                    .chars()
                    .take_while(|c| !c.is_ascii_digit() || part_name.contains("nvme"))
                    .collect::<String>();
                
                let base_name = if base_name.contains("nvme") {
                    // For NVMe: nvme0n1p1 -> nvme0n1
                    base_name.split('p').next().unwrap_or(&base_name).to_string()
                } else {
                    // For SATA/SCSI: sda1 -> sda
                    base_name.chars().take_while(|c| !c.is_ascii_digit()).collect()
                };
                
                if let Some(disk) = physical_disks.get_mut(&base_name) {
                    if !mount_point.is_empty() && mount_point != "none" {
                        disk.mount_points.push(mount_point);
                    }
                    
                    // Update usage from partition
                    let part_total = partition.total_space();
                    let part_available = partition.available_space();
                    let part_used = part_total.saturating_sub(part_available);
                    
                    disk.used = disk.used.saturating_add(part_used);
                }
            }
            
            // Calculate usage percent
            for disk in physical_disks.values_mut() {
                if disk.total > 0 {
                    disk.usage_percent = (disk.used as f32 / disk.total as f32) * 100.0;
                }
            }
        }
        
        #[cfg(target_os = "windows")]
        {
            // Use WMIC to get physical disks
            if let Ok(output) = Command::new("wmic")
                .args(["diskdrive", "get", "DeviceID,Model,SerialNumber,Size,MediaType", "/format:csv"])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                for line in stdout.lines().skip(1) {
                    let parts: Vec<&str> = line.split(',').collect();
                    if parts.len() >= 5 {
                        let device_id = parts[1].trim();
                        let model = parts[2].trim();
                        let serial = parts[4].trim();
                        let size = parts[5].trim().parse::<u64>().unwrap_or(0);
                        let media_type = parts[3].trim();
                        
                        if size > 0 {
                            let disk_type = if media_type.contains("SSD") || media_type.contains("Solid") {
                                Some("SSD".to_string())
                            } else if media_type.contains("HDD") || media_type.contains("Fixed") {
                                Some("HDD".to_string())
                            } else {
                                None
                            };
                            
                            let name = device_id.replace("\\\\.\\", "");
                            physical_disks.insert(name.clone(), DiskMetrics {
                                name,
                                model: if model.is_empty() { None } else { Some(model.to_string()) },
                                serial: if serial.is_empty() { None } else { Some(serial.to_string()) },
                                total: size,
                                disk_type,
                                mount_points: Vec::new(),
                                usage_percent: 0.0,
                                used: 0,
                            });
                        }
                    }
                }
            }
            
            // Get usage from partitions
            for partition in partitions.iter() {
                let mount = partition.mount_point().to_string_lossy().to_string();
                if !mount.is_empty() {
                    // On Windows, just report partition usage directly
                    let total = partition.total_space();
                    let available = partition.available_space();
                    let used = total.saturating_sub(available);
                    let usage = if total > 0 { (used as f32 / total as f32) * 100.0 } else { 0.0 };
                    
                    // Add as a logical disk if no physical disks found
                    if physical_disks.is_empty() {
                        physical_disks.insert(mount.clone(), DiskMetrics {
                            name: mount.clone(),
                            model: None,
                            serial: None,
                            total,
                            disk_type: Some("SSD".to_string()), // Assume SSD
                            mount_points: vec![mount],
                            usage_percent: usage,
                            used,
                        });
                    }
                }
            }
        }
        
        #[cfg(target_os = "macos")]
        {
            // Use diskutil to get physical disks
            if let Ok(output) = Command::new("diskutil")
                .args(["list", "-plist"])
                .output()
            {
                // Parse plist output - simplified approach
                let stdout = String::from_utf8_lossy(&output.stdout);
                // For macOS, fall back to partition-based reporting
                for partition in partitions.iter() {
                    let name = partition.name().to_string_lossy().to_string();
                    let mount = partition.mount_point().to_string_lossy().to_string();
                    
                    // Skip system volumes
                    if mount.starts_with("/System") || name.contains("synthesized") {
                        continue;
                    }
                    
                    let total = partition.total_space();
                    let available = partition.available_space();
                    let used = total.saturating_sub(available);
                    let usage = if total > 0 { (used as f32 / total as f32) * 100.0 } else { 0.0 };
                    
                    if total > 0 && (mount == "/" || !mount.is_empty()) {
                        let disk_name = name.trim_start_matches("/dev/").to_string();
                        if !physical_disks.contains_key(&disk_name) {
                            physical_disks.insert(disk_name.clone(), DiskMetrics {
                                name: disk_name,
                                model: None,
                                serial: None,
                                total,
                                disk_type: Some("SSD".to_string()), // Most Macs use SSD
                                mount_points: vec![mount],
                                usage_percent: usage,
                                used,
                            });
                        }
                    }
                }
                let _ = stdout; // Suppress unused warning
            }
        }
        
        physical_disks.into_values().collect()
    }
    
    /// Detect disk type (SSD, HDD, NVMe)
    #[allow(dead_code)]
    fn detect_disk_type(disk_name: &str) -> Option<String> {
        // Extract device name from path (e.g., /dev/sda1 -> sda, /dev/nvme0n1p1 -> nvme0n1)
        let device = disk_name
            .trim_start_matches("/dev/")
            .trim_end_matches(|c: char| c.is_ascii_digit());
        
        // NVMe detection by name
        if device.starts_with("nvme") {
            return Some("NVMe".to_string());
        }
        
        // Get base device (remove partition number)
        #[allow(unused_variables)]
        let base_device: String = device.chars().take_while(|c| !c.is_ascii_digit()).collect();
        
        #[cfg(target_os = "linux")]
        {
            // Check rotational flag: 0 = SSD, 1 = HDD
            let rotational_path = format!("/sys/block/{}/queue/rotational", base_device);
            if let Ok(content) = std::fs::read_to_string(&rotational_path) {
                return match content.trim() {
                    "0" => Some("SSD".to_string()),
                    "1" => Some("HDD".to_string()),
                    _ => None,
                };
            }
            
            // Fallback: check if it's a virtual device
            if base_device.starts_with("vd") || base_device.starts_with("xvd") {
                // Virtual disks are usually backed by SSDs in cloud environments
                return Some("SSD".to_string());
            }
        }
        
        #[cfg(target_os = "macos")]
        {
            // On macOS, most disks are SSDs nowadays, but we can't easily detect
            // Return None to not show incorrect info
        }
        
        #[cfg(target_os = "windows")]
        {
            // On Windows, disk type detection requires WMI which is complex
            // Virtual disks in VMs are usually SSDs
            if disk_name.contains("VBOX") || disk_name.contains("VMware") || disk_name.contains("Virtual") {
                return Some("SSD".to_string());
            }
        }
        
        None
    }
    
    fn collect_network(&mut self) -> NetworkMetrics {
        let mut total_rx: u64 = 0;
        let mut total_tx: u64 = 0;
        
        // Filter to only include physical network interfaces
        let interfaces: Vec<NetworkInterface> = self.networks
            .iter()
            .filter(|(name, _)| Self::is_physical_interface(name))
            .map(|(name, data)| {
                let rx = data.total_received();
                let tx = data.total_transmitted();
                total_rx = total_rx.saturating_add(rx);
                total_tx = total_tx.saturating_add(tx);
                
                // Try to get MAC address and speed
                let (mac, speed) = Self::get_interface_details(name);
                
                NetworkInterface {
                    name: name.to_string(),
                    mac,
                    speed,
                    rx_bytes: rx,
                    tx_bytes: tx,
                    rx_packets: data.total_packets_received(),
                    tx_packets: data.total_packets_transmitted(),
                }
            })
            .collect();
        
        // Calculate speed (bytes per second)
        let now = std::time::Instant::now();
        let elapsed_secs = now.duration_since(self.last_network_time).as_secs_f64();
        
        let (rx_speed, tx_speed) = if elapsed_secs > 0.1 {
            // Only calculate if enough time has passed
            let rx_diff = total_rx.saturating_sub(self.last_network_rx);
            let tx_diff = total_tx.saturating_sub(self.last_network_tx);
            
            // If totals went down (counter reset), use 0 for this interval
            let rx_speed = if total_rx >= self.last_network_rx {
                (rx_diff as f64 / elapsed_secs) as u64
            } else {
                0
            };
            let tx_speed = if total_tx >= self.last_network_tx {
                (tx_diff as f64 / elapsed_secs) as u64
            } else {
                0
            };
            
            // Update tracking
            self.last_network_rx = total_rx;
            self.last_network_tx = total_tx;
            self.last_network_time = now;
            
            (rx_speed, tx_speed)
        } else {
            // Not enough time passed, return 0 to avoid spikes
            (0, 0)
        };
        
        NetworkMetrics {
            interfaces,
            total_rx,
            total_tx,
            rx_speed,
            tx_speed,
        }
    }
    
    /// Check if a network interface is physical (not virtual/loopback)
    fn is_physical_interface(name: &str) -> bool {
        // Exclude loopback
        if name == "lo" || name == "lo0" {
            return false;
        }
        
        // Exclude common virtual interfaces
        let virtual_prefixes = [
            "veth",     // Docker/container virtual ethernet
            "docker",   // Docker bridge
            "br-",      // Linux bridges
            "virbr",    // Libvirt bridges
            "vnet",     // KVM virtual networks
            "vmnet",    // VMware virtual networks
            "vbox",     // VirtualBox
            "tap",      // TAP devices
            "tun",      // TUN devices
            "dummy",    // Dummy interfaces
            "bond",     // Bonding (might want physical, but skip for simplicity)
            "team",     // Team interfaces
            "wg",       // WireGuard
            "tailscale", // Tailscale
            "utun",     // macOS tunnel interfaces
            "gif",      // macOS generic tunnel
            "stf",      // macOS 6to4 tunnel
            "awdl",     // macOS Apple Wireless Direct Link
            "llw",      // macOS Low Latency WLAN
            "ap",       // Access point interfaces
            "p2p",      // P2P interfaces
        ];
        
        let name_lower = name.to_lowercase();
        for prefix in virtual_prefixes {
            if name_lower.starts_with(prefix) {
                return false;
            }
        }
        
        // On Linux, check if it's a virtual device
        #[cfg(target_os = "linux")]
        {
            let virtual_path = format!("/sys/devices/virtual/net/{}", name);
            if std::path::Path::new(&virtual_path).exists() {
                return false;
            }
        }
        
        true
    }
    
    /// Get additional interface details (MAC address, link speed)
    fn get_interface_details(name: &str) -> (Option<String>, Option<u32>) {
        let mut mac = None;
        let mut speed = None;
        
        #[cfg(target_os = "linux")]
        {
            // Read MAC address
            let mac_path = format!("/sys/class/net/{}/address", name);
            if let Ok(content) = std::fs::read_to_string(&mac_path) {
                let addr = content.trim().to_uppercase();
                if addr != "00:00:00:00:00:00" {
                    mac = Some(addr);
                }
            }
            
            // Read link speed (in Mbps)
            let speed_path = format!("/sys/class/net/{}/speed", name);
            if let Ok(content) = std::fs::read_to_string(&speed_path) {
                if let Ok(s) = content.trim().parse::<i32>() {
                    if s > 0 {
                        speed = Some(s as u32);
                    }
                }
            }
        }
        
        #[cfg(target_os = "macos")]
        {
            // Use ifconfig to get MAC
            if let Ok(output) = Command::new("ifconfig")
                .arg(name)
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                for line in stdout.lines() {
                    let line = line.trim();
                    if line.starts_with("ether ") {
                        if let Some(addr) = line.split_whitespace().nth(1) {
                            mac = Some(addr.to_uppercase());
                        }
                    }
                }
            }
            
            // Use networksetup for speed
            if let Ok(output) = Command::new("networksetup")
                .args(["-getMedia", name])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                if stdout.contains("1000") {
                    speed = Some(1000);
                } else if stdout.contains("100") {
                    speed = Some(100);
                } else if stdout.contains("10") {
                    speed = Some(10);
                }
            }
        }
        
        #[cfg(target_os = "windows")]
        {
            // Use PowerShell
            if let Ok(output) = Command::new("powershell")
                .args(["-Command", &format!(
                    "Get-NetAdapter -Name '{}' | Select-Object -Property MacAddress,LinkSpeed | ConvertTo-Json",
                    name
                )])
                .output()
            {
                let stdout = String::from_utf8_lossy(&output.stdout);
                if let Ok(json) = serde_json::from_str::<serde_json::Value>(&stdout) {
                    mac = json.get("MacAddress").and_then(|v| v.as_str()).map(|s| s.to_string());
                    if let Some(link_speed) = json.get("LinkSpeed").and_then(|v| v.as_str()) {
                        // Parse "1 Gbps" or "100 Mbps"
                        let parts: Vec<&str> = link_speed.split_whitespace().collect();
                        if parts.len() >= 2 {
                            if let Ok(num) = parts[0].parse::<u32>() {
                                speed = Some(if parts[1].starts_with('G') { num * 1000 } else { num });
                            }
                        }
                    }
                }
            }
        }
        
        (mac, speed)
    }
    
    fn collect_load_average(&self) -> LoadAverage {
        let load = System::load_average();
        LoadAverage {
            one: load.one,
            five: load.five,
            fifteen: load.fifteen,
        }
    }
}

impl Default for MetricsCollector {
    fn default() -> Self {
        Self::new()
    }
}

