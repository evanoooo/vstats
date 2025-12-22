/**
 * VStats Theme System
 * 
 * External theme support with GitHub installation.
 */

// Export types
export * from './types';

// Export theme manager functions
export {
  // GitHub parsing
  parseGitHubSource,
  buildGitHubRawUrl,
  
  // Theme fetching
  fetchThemeManifest,
  fetchThemeCSS,
  
  // Installation
  installTheme,
  uninstallTheme,
  
  // Storage
  getInstalledThemes,
  getInstalledTheme,
  
  // CSS injection
  applyThemeCSS,
  removeThemeCSS,
  loadInstalledThemeCSS,
  
  // Conversion
  installedToConfig,
  
  // Updates
  checkThemeUpdate,
  updateTheme,
  
  // Initialize
  initThemeManager,
} from './themeManager';

// Export builtin themes
export { BUILTIN_THEMES, getBuiltinTheme } from './builtinThemes';

