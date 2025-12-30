/**
 * Settings Page Type Definitions
 */

import type { SiteSettings } from '../../types';
import type { BackgroundType } from '../../context/ThemeContext';

// Server label with color
export interface ServerLabel {
  name: string;
  color: string; // red, orange, amber, yellow, lime, green, emerald, teal, cyan, sky, blue, indigo, violet, purple, fuchsia, pink, rose
}

// Remote server interface
export interface RemoteServer {
  id: string;
  name: string;
  url: string;
  location: string;
  provider: string;
  tag?: string;
  group_id?: string; // Deprecated
  group_values?: Record<string, string>; // dimension_id -> option_id
  version?: string;
  token?: string;
  ip?: string;
  // Extended metadata
  price_amount?: string;
  price_period?: string;
  price_currency?: string;
  purchase_date?: string;
  expiry_date?: string;
  auto_renew?: boolean;
  remaining_value?: string;
  tip_badge?: string;
  notes?: string;
  labels?: ServerLabel[];
  // Sale/rent settings
  sale_status?: '' | 'rent' | 'sell'; // Sale status: empty, rent, sell
  sale_contact_url?: string; // Contact URL for rent/sell
  // Traffic settings
  traffic_limit_gb?: number; // Monthly traffic limit in GB (0 = unlimited)
  traffic_threshold_type?: 'sum' | 'max' | 'up' | 'down'; // How traffic is calculated
  traffic_reset_day?: number; // Day of month to reset (1-28)
}

// Ping target configuration
export interface PingTargetConfig {
  name: string;
  host: string;
  type?: string; // "icmp" or "tcp", default "icmp"
  port?: number; // Port for TCP connections, default 80
}

// Probe settings
export interface ProbeSettings {
  ping_targets: PingTargetConfig[];
}

// Theme settings section props
export interface ThemeSettingsSectionProps {
  isAuthenticated: boolean;
  token: string | null;
  siteSettings: SiteSettings;
  onSiteSettingsChange: (settings: SiteSettings) => void;
}

// Background option type
export interface BackgroundOption {
  type: BackgroundType;
  name: string;
  nameZh: string;
  icon: string;
}

// Unsplash preset type
export interface UnsplashPreset {
  query: string;
  label: string;
  labelZh: string;
}

// Audit log interface
export interface AuditLog {
  id: number;
  timestamp: string;
  action: string;
  category: string;
  user_ip: string;
  user_agent?: string;
  target_type?: string;
  target_id?: string;
  target_name?: string;
  details?: string;
  status: string;
  error_message?: string;
}

// Audit log stats interface
export interface AuditLogStats {
  total: number;
  today: number;
  errors: number;
  by_category: Record<string, number>;
  top_actions: Record<string, number>;
  oldest_timestamp?: string;
}

// Audit logs section props
export interface AuditLogsSectionProps {
  token: string | null;
  isZh: boolean;
}

// OIDC provider form interface
export interface OIDCProviderForm {
  id: string;
  enabled: boolean;
  name: string;
  issuer: string;
  client_id: string;
  client_secret: string;
  scopes: string;
  allowed_users: string;
  allowed_groups: string;
  username_claim: string;
}

// SSO binding interface
export interface SSOBinding {
  provider: string;
  provider_id: string;
  identifier: string;
  bound_at: string;
}

// OAuth settings interface
export interface OAuthSettings {
  use_centralized?: boolean;
  allowed_users?: string[];
  github?: { enabled: boolean; client_id: string; has_secret: boolean; allowed_users: string[] };
  google?: { enabled: boolean; client_id: string; has_secret: boolean; allowed_users: string[] };
  oidc?: { id: string; enabled: boolean; name: string; issuer: string; client_id: string; has_secret: boolean; scopes: string[]; allowed_users: string[]; allowed_groups: string[]; username_claim: string }[];
  cloudflare_access?: { enabled: boolean; team_domain: string; aud: string; allowed_users: string[] };
  bindings?: SSOBinding[];
}

// OAuth form interface
export interface OAuthForm {
  use_centralized: boolean;
  allowed_users: string;
  github: { enabled: boolean; client_id: string; client_secret: string; allowed_users: string };
  google: { enabled: boolean; client_id: string; client_secret: string; allowed_users: string };
  oidc: OIDCProviderForm[];
  cloudflare_access: { enabled: boolean; team_domain: string; aud: string; allowed_users: string };
}

// Platform option
export interface PlatformOption {
  value: string;
  label: string;
}

