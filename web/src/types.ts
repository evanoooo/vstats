export interface SystemMetrics {
  timestamp: string;
  hostname: string;
  os: OsInfo;
  cpu: CpuMetrics;
  memory: MemoryMetrics;
  disks: DiskMetrics[];
  network: NetworkMetrics;
  uptime: number;
  load_average: LoadAverage;
  ping?: PingMetrics;
  gpu?: GPUMetrics;
  version?: string;
}

export interface OsInfo {
  name: string;
  version: string;
  kernel: string;
  arch: string;
}

export interface CpuMetrics {
  brand: string;
  cores: number;
  usage: number;
  frequency: number;
  per_core: number[];
}

export interface MemoryMetrics {
  total: number;
  used: number;
  available: number;
  swap_total: number;
  swap_used: number;
  usage_percent: number;
  modules?: MemoryModule[];
}

export interface MemoryModule {
  slot?: string;
  size: number;
  mem_type?: string;  // "DDR4", "DDR5", etc.
  speed?: number;     // MHz
  manufacturer?: string;
}

export interface DiskMetrics {
  name: string;
  model?: string;
  serial?: string;
  total: number;
  disk_type?: string;  // "SSD", "HDD", "NVMe"
  mount_points?: string[];
  usage_percent: number;
  used: number;
}

export interface NetworkMetrics {
  interfaces: NetworkInterface[];
  total_rx: number;
  total_tx: number;
  rx_speed?: number;
  tx_speed?: number;
}

export interface NetworkInterface {
  name: string;
  mac?: string;
  speed?: number;  // Mbps
  rx_bytes: number;
  tx_bytes: number;
  rx_packets: number;
  tx_packets: number;
}

export interface LoadAverage {
  one: number;
  five: number;
  fifteen: number;
}

export interface GPUMetrics {
  gpus: GPU[];
}

export interface GPU {
  index: number;
  name: string;
  vendor: string;  // "NVIDIA", "AMD", "Intel"
  memory_total: number;
  memory_used: number;
  memory_percent: number;
  utilization: number;
  temperature?: number;
  fan_speed?: number;
  power_draw?: number;
  power_limit?: number;
  clock_core?: number;
  clock_memory?: number;
  driver_version?: string;
  cuda_version?: string;
  pci_bus?: string;
  encoder_util?: number;
  decoder_util?: number;
}

export interface PingMetrics {
  targets: PingTarget[];
}

export interface PingTarget {
  name: string;
  host: string;
  type?: string; // "icmp" or "tcp"
  port?: number; // Port for TCP connections
  latency_ms: number | null;
  packet_loss: number;
  status: string;
}

// GeoIP data
export interface GeoIPData {
  country_code: string;
  country_name: string;
  city?: string;
  region?: string;
  latitude?: number;
  longitude?: number;
  updated_at?: string;
}

// Server Groups (Deprecated - for backward compatibility)
export interface ServerGroup {
  id: string;
  name: string;
  sort_order: number;
}

// Group Dimensions - Multi-dimensional grouping
export interface GroupOption {
  id: string;
  name: string;
  sort_order: number;
}

export interface GroupDimension {
  id: string;
  name: string;
  key: string;
  enabled: boolean;
  sort_order: number;
  options: GroupOption[];
}

// Background configuration
export interface BackgroundConfig {
  type: 'gradient' | 'bing' | 'unsplash' | 'custom' | 'solid';
  custom_url?: string;
  unsplash_query?: string;
  solid_color?: string;
  blur?: number;
  opacity?: number;
}

// Theme settings
export interface ThemeSettings {
  theme_id: string;
  background?: BackgroundConfig;
}

// Site Settings
export interface SiteSettings {
  site_name: string;
  site_description: string;
  social_links: SocialLink[];
  theme?: ThemeSettings;
}

export interface SocialLink {
  platform: string;  // github, twitter, telegram, email, website, discord, etc.
  url: string;
  label?: string;
}

// History Data
export interface HistoryPoint {
  timestamp: string;
  cpu: number;
  memory: number;
  disk: number;
  net_rx: number;
  net_tx: number;
  ping_ms?: number;
}

export interface HistoryResponse {
  server_id: string;
  range: string;
  data: HistoryPoint[];
  ping_targets?: PingHistoryTarget[];
}

export interface PingHistoryTarget {
  name: string;
  host: string;
  data: PingHistoryPoint[];
}

export interface PingHistoryPoint {
  timestamp: string;
  latency_ms: number | null;
  status: string;
}

// ============================================================================
// Alert Types
// ============================================================================

export interface AlertConfig {
  enabled: boolean;
  channels: NotificationChannel[];
  rules: AlertRules;
  templates?: Record<string, AlertTemplate>;
  global_cooldown?: number;
  recovery_notify?: boolean;
}

export interface NotificationChannel {
  id: string;
  type: 'email' | 'telegram' | 'discord' | 'webhook' | 'bark' | 'serverchan';
  name: string;
  enabled: boolean;
  config: Record<string, string>;
  priority?: number;
}

export interface AlertRules {
  offline: OfflineAlertRule;
  load: LoadAlertRule;
  traffic: TrafficAlertRule;
}

export interface OfflineAlertRule {
  enabled: boolean;
  grace_period: number;
  channels: string[];
  servers: string[];
  exclude: string[];
}

export interface ThresholdConfig {
  warning?: number;
  critical?: number;
  duration?: number;
}

export interface LoadAlertRule {
  enabled: boolean;
  cpu?: ThresholdConfig;
  memory?: ThresholdConfig;
  disk?: ThresholdConfig;
  channels: string[];
  servers: string[];
  exclude: string[];
  cooldown?: number;
}

export interface TrafficLimit {
  server_id: string;
  monthly_gb: number;
  type: 'sum' | 'max' | 'up' | 'down';
  reset_day?: number;
  warning?: number;
}

export interface TrafficAlertRule {
  enabled: boolean;
  limits: TrafficLimit[];
  channels: string[];
  cooldown?: number;
}

export interface AlertTemplate {
  title: string;
  body: string;
  format?: string;
}

export interface AlertState {
  id: string;
  type: string;
  server_id: string;
  server_name: string;
  severity: 'warning' | 'critical';
  status: 'firing' | 'resolved';
  value: number;
  threshold: number;
  message: string;
  started_at: string;
  updated_at: string;
  resolved_at?: string;
  notified_at?: string;
  muted: boolean;
}

export interface AlertStats {
  total_firing: number;
  critical: number;
  warning: number;
  servers_online: number;
  servers_total: number;
}

export interface AlertsResponse {
  alerts: AlertState[];
  stats: AlertStats;
}

export interface AlertHistory {
  id: number;
  alert_id: string;
  type: string;
  server_id: string;
  server_name: string;
  severity: string;
  value: number;
  threshold: number;
  message: string;
  started_at: string;
  resolved_at?: string;
  duration: number;
  notified: boolean;
}

// ============================================================================
// Affiliate Provider Types
// ============================================================================

export interface AffProvider {
  id: string;
  name: string;       // Provider name to match (e.g., "Vultr", "DigitalOcean")
  aff_link: string;   // Affiliate link URL
  logo_url?: string;  // Custom logo URL (optional)
  enabled: boolean;
}

// Public affiliate provider (returned for dashboard)
export interface PublicAffProvider {
  name: string;
  aff_link: string;
  logo_url?: string;
}
