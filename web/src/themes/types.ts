/**
 * VStats Theme System - Type Definitions
 * 
 * This file defines the structure for external themes that can be
 * installed from GitHub or other sources.
 */

// ============================================================================
// Theme Package Definition (theme.json)
// ============================================================================

/**
 * Theme manifest - the theme.json file structure
 */
export interface ThemeManifest {
  // Required fields
  id: string;              // Unique theme identifier (lowercase, no spaces)
  name: string;            // Display name
  version: string;         // Semantic version (e.g., "1.0.0")
  author: string;          // Author name or GitHub username
  description: string;     // Brief description
  
  // Localization
  nameZh?: string;         // Chinese name
  descriptionZh?: string;  // Chinese description
  
  // Theme properties
  isDark: boolean;         // Dark theme flag
  style: ThemeStyle;       // UI style type
  
  // Preview colors (for theme selector)
  preview: ThemePreview;
  
  // Typography
  fonts: ThemeFonts;
  
  // Styling
  borderRadius: string;    // CSS border-radius value
  cardStyle: string;       // Card style identifier
  
  // Optional metadata
  license?: string;        // License (e.g., "MIT")
  homepage?: string;       // Theme homepage URL
  repository?: string;     // Source repository URL
  keywords?: string[];     // Search keywords
  minVersion?: string;     // Minimum VStats version required
  
  // CSS file (relative to theme.json)
  cssFile?: string;        // Default: "theme.css"
  
  // Assets directory (relative to theme.json)
  assetsDir?: string;      // Default: "assets/"
  
  // Preview image (for theme gallery)
  previewImage?: string;   // Screenshot of theme
}

export type ThemeStyle = 'flat' | 'glass' | 'neumorphic' | 'brutalist' | 'minimal';

export interface ThemePreview {
  primary: string;     // Primary background color
  secondary: string;   // Secondary background color
  accent: string;      // Accent color
  background: string;  // Page background color
}

export interface ThemeFonts {
  heading: string;     // CSS font-family for headings
  body: string;        // CSS font-family for body text
  mono: string;        // CSS font-family for monospace text
}

// ============================================================================
// Installed Theme
// ============================================================================

/**
 * An installed theme (stored in config)
 */
export interface InstalledTheme {
  // From manifest
  manifest: ThemeManifest;
  
  // Installation info
  source: ThemeSource;      // Where theme came from
  installedAt: string;      // ISO timestamp
  updatedAt?: string;       // ISO timestamp of last update
  
  // Resolved CSS content (loaded once)
  cssContent?: string;      // Full CSS content
  
  // State
  enabled: boolean;         // Whether theme is available for selection
}

export interface ThemeSource {
  type: 'builtin' | 'github' | 'url' | 'local';
  
  // For GitHub: "user/repo" or "user/repo/path/to/theme"
  // For URL: direct URL to theme.json
  // For local: file path
  location: string;
  
  // Git reference (branch, tag, or commit)
  ref?: string;
}

// ============================================================================
// Theme Installation
// ============================================================================

/**
 * Request to install a theme
 */
export interface InstallThemeRequest {
  // Source specification
  source: string;           // GitHub: "user/repo" or "user/repo/path"
                           // URL: "https://example.com/theme.json"
                           // Local: "file:///path/to/theme"
  
  // Optional git ref
  ref?: string;             // Branch, tag, or commit hash
}

/**
 * Theme installation result
 */
export interface InstallThemeResult {
  success: boolean;
  theme?: InstalledTheme;
  error?: string;
}

// ============================================================================
// Theme Registry (for discovering themes)
// ============================================================================

/**
 * Theme entry in a theme registry
 */
export interface ThemeRegistryEntry {
  id: string;
  name: string;
  nameZh?: string;
  description: string;
  descriptionZh?: string;
  author: string;
  version: string;
  source: string;          // GitHub "user/repo" or URL
  preview: ThemePreview;
  previewImage?: string;
  downloads?: number;
  stars?: number;
  updatedAt?: string;
}

/**
 * Theme registry response
 */
export interface ThemeRegistry {
  version: string;
  themes: ThemeRegistryEntry[];
  updatedAt: string;
}

// ============================================================================
// Theme Context Types
// ============================================================================

/**
 * Full theme configuration (merged from manifest + CSS)
 */
export interface ThemeConfig {
  id: string;
  name: string;
  nameZh?: string;
  description: string;
  descriptionZh?: string;
  isDark: boolean;
  style: ThemeStyle;
  preview: ThemePreview;
  fonts: ThemeFonts;
  borderRadius: string;
  cardStyle: string;
  
  // Source info (null for builtin)
  source?: ThemeSource;
  
  // Whether this is a builtin theme
  isBuiltin: boolean;
}

// ============================================================================
// API Types
// ============================================================================

/**
 * List themes response
 */
export interface ListThemesResponse {
  builtin: ThemeConfig[];
  installed: InstalledTheme[];
  current: string;  // Current theme ID
}

/**
 * Theme update check response
 */
export interface ThemeUpdateCheck {
  themeId: string;
  currentVersion: string;
  latestVersion: string;
  hasUpdate: boolean;
  changelog?: string;
}

