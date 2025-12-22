/**
 * VStats Theme Manager
 * 
 * Handles theme installation, loading, and management.
 * Supports installing themes from GitHub repositories.
 */

import type {
  ThemeManifest,
  InstalledTheme,
  ThemeSource,
  InstallThemeRequest,
  InstallThemeResult,
  ThemeConfig,
} from './types';

// ============================================================================
// Constants
// ============================================================================

const GITHUB_RAW_BASE = 'https://raw.githubusercontent.com';
const THEME_STORAGE_KEY = 'vstats-installed-themes';
const THEME_CSS_STORAGE_KEY = 'vstats-theme-css';

// ============================================================================
// GitHub URL Parsing
// ============================================================================

interface ParsedGitHubSource {
  owner: string;
  repo: string;
  path: string;
  ref: string;
}

/**
 * Parse a GitHub source string into components
 * Formats supported:
 * - user/repo
 * - user/repo/path/to/theme
 * - user/repo@branch
 * - user/repo/path@tag
 */
export function parseGitHubSource(source: string, defaultRef = 'main'): ParsedGitHubSource {
  let ref = defaultRef;
  let cleanSource = source;
  
  // Extract ref if specified with @
  if (source.includes('@')) {
    const [pathPart, refPart] = source.split('@');
    cleanSource = pathPart;
    ref = refPart || defaultRef;
  }
  
  const parts = cleanSource.split('/').filter(Boolean);
  
  if (parts.length < 2) {
    throw new Error(`Invalid GitHub source: ${source}. Expected format: user/repo or user/repo/path`);
  }
  
  const owner = parts[0];
  const repo = parts[1];
  const path = parts.slice(2).join('/') || '';
  
  return { owner, repo, path, ref };
}

/**
 * Build raw GitHub URL for a file
 */
export function buildGitHubRawUrl(parsed: ParsedGitHubSource, filename: string): string {
  const pathPrefix = parsed.path ? `${parsed.path}/` : '';
  return `${GITHUB_RAW_BASE}/${parsed.owner}/${parsed.repo}/${parsed.ref}/${pathPrefix}${filename}`;
}

// ============================================================================
// Theme Fetching
// ============================================================================

/**
 * Fetch theme manifest from source
 */
export async function fetchThemeManifest(source: string, ref?: string): Promise<ThemeManifest> {
  let manifestUrl: string;
  
  if (source.startsWith('http://') || source.startsWith('https://')) {
    // Direct URL
    manifestUrl = source.endsWith('theme.json') ? source : `${source}/theme.json`;
  } else {
    // GitHub source
    const parsed = parseGitHubSource(source, ref || 'main');
    manifestUrl = buildGitHubRawUrl(parsed, 'theme.json');
  }
  
  const response = await fetch(manifestUrl);
  
  if (!response.ok) {
    throw new Error(`Failed to fetch theme manifest: ${response.status} ${response.statusText}`);
  }
  
  const manifest = await response.json() as ThemeManifest;
  
  // Validate required fields
  validateManifest(manifest);
  
  return manifest;
}

/**
 * Fetch theme CSS from source
 */
export async function fetchThemeCSS(source: string, manifest: ThemeManifest, ref?: string): Promise<string> {
  const cssFilename = manifest.cssFile || 'theme.css';
  let cssUrl: string;
  
  if (source.startsWith('http://') || source.startsWith('https://')) {
    // Direct URL - get base path
    const baseUrl = source.endsWith('theme.json') 
      ? source.replace(/theme\.json$/, '')
      : source.endsWith('/') ? source : `${source}/`;
    cssUrl = `${baseUrl}${cssFilename}`;
  } else {
    // GitHub source
    const parsed = parseGitHubSource(source, ref || 'main');
    cssUrl = buildGitHubRawUrl(parsed, cssFilename);
  }
  
  const response = await fetch(cssUrl);
  
  if (!response.ok) {
    throw new Error(`Failed to fetch theme CSS: ${response.status} ${response.statusText}`);
  }
  
  return await response.text();
}

/**
 * Validate theme manifest
 */
function validateManifest(manifest: ThemeManifest): void {
  const required: (keyof ThemeManifest)[] = [
    'id', 'name', 'version', 'author', 'description',
    'isDark', 'style', 'preview', 'fonts', 'borderRadius', 'cardStyle'
  ];
  
  for (const field of required) {
    if (manifest[field] === undefined || manifest[field] === null) {
      throw new Error(`Invalid theme manifest: missing required field "${field}"`);
    }
  }
  
  // Validate ID format (lowercase alphanumeric with hyphens)
  if (!/^[a-z0-9-]+$/.test(manifest.id)) {
    throw new Error(`Invalid theme ID: "${manifest.id}". Must be lowercase alphanumeric with hyphens only.`);
  }
  
  // Validate version format
  if (!/^\d+\.\d+\.\d+/.test(manifest.version)) {
    throw new Error(`Invalid version format: "${manifest.version}". Use semantic versioning (e.g., 1.0.0).`);
  }
  
  // Validate style
  const validStyles = ['flat', 'glass', 'neumorphic', 'brutalist', 'minimal'];
  if (!validStyles.includes(manifest.style)) {
    throw new Error(`Invalid style: "${manifest.style}". Must be one of: ${validStyles.join(', ')}`);
  }
  
  // Validate preview colors
  if (!manifest.preview.primary || !manifest.preview.secondary || 
      !manifest.preview.accent || !manifest.preview.background) {
    throw new Error('Invalid preview: must include primary, secondary, accent, and background colors');
  }
  
  // Validate fonts
  if (!manifest.fonts.heading || !manifest.fonts.body || !manifest.fonts.mono) {
    throw new Error('Invalid fonts: must include heading, body, and mono font families');
  }
}

// ============================================================================
// Theme Installation
// ============================================================================

/**
 * Install a theme from source
 */
export async function installTheme(request: InstallThemeRequest): Promise<InstallThemeResult> {
  try {
    // Determine source type
    let sourceType: ThemeSource['type'];
    if (request.source.startsWith('file://')) {
      sourceType = 'local';
    } else if (request.source.startsWith('http://') || request.source.startsWith('https://')) {
      sourceType = 'url';
    } else {
      sourceType = 'github';
    }
    
    // Fetch manifest
    const manifest = await fetchThemeManifest(request.source, request.ref);
    
    // Check if already installed
    const installed = getInstalledThemes();
    const existing = installed.find(t => t.manifest.id === manifest.id);
    if (existing) {
      // Update existing theme
      const cssContent = await fetchThemeCSS(request.source, manifest, request.ref);
      
      const updatedTheme: InstalledTheme = {
        ...existing,
        manifest,
        cssContent,
        updatedAt: new Date().toISOString(),
        source: {
          type: sourceType,
          location: request.source,
          ref: request.ref,
        },
      };
      
      saveInstalledTheme(updatedTheme);
      applyThemeCSS(manifest.id, cssContent);
      
      return { success: true, theme: updatedTheme };
    }
    
    // Fetch CSS
    const cssContent = await fetchThemeCSS(request.source, manifest, request.ref);
    
    // Create installed theme record
    const theme: InstalledTheme = {
      manifest,
      source: {
        type: sourceType,
        location: request.source,
        ref: request.ref,
      },
      installedAt: new Date().toISOString(),
      cssContent,
      enabled: true,
    };
    
    // Save to storage
    saveInstalledTheme(theme);
    
    // Inject CSS
    applyThemeCSS(manifest.id, cssContent);
    
    return { success: true, theme };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
    };
  }
}

/**
 * Uninstall a theme
 */
export function uninstallTheme(themeId: string): boolean {
  const themes = getInstalledThemes();
  const index = themes.findIndex(t => t.manifest.id === themeId);
  
  if (index === -1) {
    return false;
  }
  
  themes.splice(index, 1);
  localStorage.setItem(THEME_STORAGE_KEY, JSON.stringify(themes));
  
  // Remove CSS
  removeThemeCSS(themeId);
  
  // Also remove from CSS storage
  const cssStorage = getThemeCSSStorage();
  delete cssStorage[themeId];
  localStorage.setItem(THEME_CSS_STORAGE_KEY, JSON.stringify(cssStorage));
  
  return true;
}

// ============================================================================
// Theme Storage
// ============================================================================

/**
 * Get all installed themes from local storage
 */
export function getInstalledThemes(): InstalledTheme[] {
  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY);
    return stored ? JSON.parse(stored) : [];
  } catch {
    return [];
  }
}

/**
 * Get theme CSS storage
 */
function getThemeCSSStorage(): Record<string, string> {
  try {
    const stored = localStorage.getItem(THEME_CSS_STORAGE_KEY);
    return stored ? JSON.parse(stored) : {};
  } catch {
    return {};
  }
}

/**
 * Save an installed theme
 */
function saveInstalledTheme(theme: InstalledTheme): void {
  const themes = getInstalledThemes();
  const index = themes.findIndex(t => t.manifest.id === theme.manifest.id);
  
  if (index >= 0) {
    themes[index] = theme;
  } else {
    themes.push(theme);
  }
  
  localStorage.setItem(THEME_STORAGE_KEY, JSON.stringify(themes));
  
  // Also save CSS separately for quick access
  if (theme.cssContent) {
    const cssStorage = getThemeCSSStorage();
    cssStorage[theme.manifest.id] = theme.cssContent;
    localStorage.setItem(THEME_CSS_STORAGE_KEY, JSON.stringify(cssStorage));
  }
}

/**
 * Get installed theme by ID
 */
export function getInstalledTheme(themeId: string): InstalledTheme | undefined {
  return getInstalledThemes().find(t => t.manifest.id === themeId);
}

// ============================================================================
// CSS Injection
// ============================================================================

const STYLE_ID_PREFIX = 'vstats-theme-';

/**
 * Apply theme CSS to the document
 */
export function applyThemeCSS(themeId: string, css: string): void {
  const styleId = `${STYLE_ID_PREFIX}${themeId}`;
  
  // Remove existing style element if present
  const existing = document.getElementById(styleId);
  if (existing) {
    existing.remove();
  }
  
  // Create and inject new style element
  const style = document.createElement('style');
  style.id = styleId;
  style.textContent = css;
  document.head.appendChild(style);
}

/**
 * Remove theme CSS from the document
 */
export function removeThemeCSS(themeId: string): void {
  const styleId = `${STYLE_ID_PREFIX}${themeId}`;
  const existing = document.getElementById(styleId);
  if (existing) {
    existing.remove();
  }
}

/**
 * Load all installed theme CSS
 */
export function loadInstalledThemeCSS(): void {
  const cssStorage = getThemeCSSStorage();
  
  for (const [themeId, css] of Object.entries(cssStorage)) {
    applyThemeCSS(themeId, css);
  }
}

// ============================================================================
// Theme Config Conversion
// ============================================================================

/**
 * Convert InstalledTheme to ThemeConfig
 */
export function installedToConfig(installed: InstalledTheme): ThemeConfig {
  const { manifest, source } = installed;
  
  return {
    id: manifest.id,
    name: manifest.name,
    nameZh: manifest.nameZh,
    description: manifest.description,
    descriptionZh: manifest.descriptionZh,
    isDark: manifest.isDark,
    style: manifest.style,
    preview: manifest.preview,
    fonts: manifest.fonts,
    borderRadius: manifest.borderRadius,
    cardStyle: manifest.cardStyle,
    source,
    isBuiltin: false,
  };
}

// ============================================================================
// Theme Update Check
// ============================================================================

/**
 * Check if a theme has updates available
 */
export async function checkThemeUpdate(themeId: string): Promise<{
  hasUpdate: boolean;
  latestVersion?: string;
  error?: string;
}> {
  const installed = getInstalledTheme(themeId);
  
  if (!installed) {
    return { hasUpdate: false, error: 'Theme not installed' };
  }
  
  try {
    const latestManifest = await fetchThemeManifest(
      installed.source.location,
      installed.source.ref
    );
    
    const currentVersion = installed.manifest.version;
    const latestVersion = latestManifest.version;
    
    // Simple version comparison
    const hasUpdate = latestVersion !== currentVersion;
    
    return { hasUpdate, latestVersion };
  } catch (error) {
    return {
      hasUpdate: false,
      error: error instanceof Error ? error.message : 'Failed to check for updates',
    };
  }
}

/**
 * Update an installed theme to the latest version
 */
export async function updateTheme(themeId: string): Promise<InstallThemeResult> {
  const installed = getInstalledTheme(themeId);
  
  if (!installed) {
    return { success: false, error: 'Theme not installed' };
  }
  
  return installTheme({
    source: installed.source.location,
    ref: installed.source.ref,
  });
}

// ============================================================================
// Initialize
// ============================================================================

/**
 * Initialize theme manager - load installed theme CSS
 */
export function initThemeManager(): void {
  loadInstalledThemeCSS();
}

