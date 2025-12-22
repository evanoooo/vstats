import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import { sanitizeUrl } from '../utils/security';
import { 
  BUILTIN_THEMES, 
  getBuiltinTheme,
} from '../themes/builtinThemes';
import type { ThemeConfig, InstalledTheme, ThemeSource } from '../themes/types';
import { 
  initThemeManager, 
  getInstalledThemes, 
  installedToConfig,
  installTheme as installThemeFromSource,
  uninstallTheme as removeTheme,
  applyThemeCSS,
  type InstallThemeResult,
} from '../themes';

// Re-export ThemeId for compatibility
export type { ThemeId } from '../themes/builtinThemes';

// 背景类型
export type BackgroundType = 'gradient' | 'bing' | 'unsplash' | 'custom' | 'solid';

export interface BackgroundConfig {
  type: BackgroundType;
  customUrl?: string;
  unsplashQuery?: string;
  solidColor?: string;
  blur?: number;
  opacity?: number;
}

// Server theme settings interface (matches backend and types.ts)
interface ServerThemeSettings {
  theme_id: string;
  background?: {
    type: BackgroundType;
    custom_url?: string;
    unsplash_query?: string;
    solid_color?: string;
    blur?: number;
    opacity?: number;
  };
}

// Server installed theme interface
interface ServerInstalledTheme {
  manifest: {
    id: string;
    name: string;
    version: string;
    author: string;
    description: string;
    nameZh?: string;
    descriptionZh?: string;
    isDark: boolean;
    style: string;
    preview: Record<string, string>;
    fonts: Record<string, string>;
    borderRadius: string;
    cardStyle: string;
  };
  source: {
    type: string;
    location: string;
    ref?: string;
  };
  installedAt: string;
  updatedAt?: string;
  cssContent?: string;
  enabled: boolean;
}

interface ThemeContextType {
  // Current theme
  themeId: string;
  theme: ThemeConfig;
  isDark: boolean;
  setTheme: (themeId: string) => void;
  
  // All available themes
  themes: ThemeConfig[];
  builtinThemes: ThemeConfig[];
  installedThemes: InstalledTheme[];
  
  // Background
  background: BackgroundConfig;
  setBackground: (config: BackgroundConfig) => void;
  backgroundUrl: string | null;
  refreshBackground: () => void;
  
  // Theme management
  installTheme: (source: string, ref?: string) => Promise<InstallThemeResult>;
  uninstallTheme: (themeId: string) => Promise<boolean>;
  refreshThemes: () => Promise<void>;
  
  // Sync from server
  applyServerSettings: (settings: ServerThemeSettings | null) => void;
  getServerSettings: () => ServerThemeSettings;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

// 默认背景配置
const DEFAULT_BACKGROUND: BackgroundConfig = {
  type: 'gradient',
  blur: 0,
  opacity: 100
};

// Fetch Bing wallpaper through our proxy API (server-side proxy to avoid CORS)
const fetchBingWallpaper = async (): Promise<string> => {
  try {
    const response = await fetch('/api/wallpaper/bing');
    
    if (!response.ok) {
      console.warn(`Bing wallpaper proxy returned ${response.status}: ${response.statusText}`);
      return 'https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1920&q=80';
    }
    
    const data = await response.json();
    if (data && data.url) {
      return data.url;
    }
    
    console.warn('Bing wallpaper proxy returned invalid response:', data);
  } catch (e) {
    console.error('Failed to fetch Bing wallpaper through proxy:', e);
  }
  return 'https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1920&q=80';
};

// Fetch custom wallpaper URL through our proxy API to avoid CORS
const fetchCustomWallpaper = async (imageURL: string): Promise<string> => {
  try {
    try {
      const url = new URL(imageURL, window.location.origin);
      if (url.origin === window.location.origin) {
        return imageURL;
      }
    } catch {
      if (!imageURL.startsWith('http://') && !imageURL.startsWith('https://')) {
        return imageURL;
      }
    }

    const response = await fetch(`/api/wallpaper/proxy?url=${encodeURIComponent(imageURL)}`);
    
    if (!response.ok) {
      console.warn(`Custom wallpaper proxy returned ${response.status}: ${response.statusText}`);
      return imageURL;
    }
    
    const data = await response.json();
    if (data && data.url) {
      return data.url;
    }
    
    console.warn('Custom wallpaper proxy returned invalid response:', data);
  } catch (e) {
    console.error('Failed to fetch custom wallpaper through proxy:', e);
  }
  return imageURL;
};

// Fetch Unsplash image through our proxy API (server-side proxy to avoid CORS)
const fetchUnsplashImage = async (query: string = 'nature,landscape'): Promise<string> => {
  try {
    const keywords = query || 'nature,landscape,abstract';
    const response = await fetch(`/api/wallpaper/unsplash?query=${encodeURIComponent(keywords)}`);
    
    if (!response.ok) {
      console.warn(`Unsplash wallpaper proxy returned ${response.status}: ${response.statusText}`);
      return `https://source.unsplash.com/1920x1080/?${encodeURIComponent(keywords)}&t=${Date.now()}`;
    }
    
    const data = await response.json();
    if (data && data.url) {
      return data.url;
    }
    
    console.warn('Unsplash wallpaper proxy returned invalid response:', data);
  } catch (e) {
    console.error('Failed to fetch Unsplash image through proxy:', e);
  }
  const keywords = query || 'nature,landscape,abstract';
  return `https://source.unsplash.com/1920x1080/?${encodeURIComponent(keywords)}&t=${Date.now()}`;
};

// Convert server format to local format
const serverToLocalBackground = (serverBg: ServerThemeSettings['background']): BackgroundConfig => {
  if (!serverBg) return DEFAULT_BACKGROUND;
  const validTypes: BackgroundType[] = ['gradient', 'bing', 'unsplash', 'custom', 'solid'];
  const bgType = validTypes.includes(serverBg.type as BackgroundType) ? serverBg.type : 'gradient';
  const safeCustomUrl = sanitizeUrl(serverBg.custom_url) || undefined;
  return {
    type: bgType,
    customUrl: safeCustomUrl,
    unsplashQuery: serverBg.unsplash_query,
    solidColor: serverBg.solid_color,
    blur: serverBg.blur ?? 0,
    opacity: serverBg.opacity ?? 100,
  };
};

// Convert local format to server format
const localToServerBackground = (localBg: BackgroundConfig): ServerThemeSettings['background'] => {
  const safeCustomUrl = sanitizeUrl(localBg.customUrl);
  return {
    type: localBg.type as BackgroundType,
    custom_url: safeCustomUrl || undefined,
    unsplash_query: localBg.unsplashQuery,
    solid_color: localBg.solidColor,
    blur: localBg.blur,
    opacity: localBg.opacity,
  };
};

// Convert server installed theme to local format
const serverToLocalInstalledTheme = (server: ServerInstalledTheme): InstalledTheme => {
  return {
    manifest: {
      id: server.manifest.id,
      name: server.manifest.name,
      version: server.manifest.version,
      author: server.manifest.author,
      description: server.manifest.description,
      nameZh: server.manifest.nameZh,
      descriptionZh: server.manifest.descriptionZh,
      isDark: server.manifest.isDark,
      style: server.manifest.style as any,
      preview: server.manifest.preview as any,
      fonts: server.manifest.fonts as any,
      borderRadius: server.manifest.borderRadius,
      cardStyle: server.manifest.cardStyle,
    },
    source: server.source as ThemeSource,
    installedAt: server.installedAt,
    updatedAt: server.updatedAt,
    cssContent: server.cssContent,
    enabled: server.enabled,
  };
};

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [themeId, setThemeId] = useState<string>(() => {
    const stored = localStorage.getItem('vstats-theme-id');
    if (stored) {
      // Check if it's a valid builtin theme
      if (BUILTIN_THEMES.find(t => t.id === stored)) {
        return stored;
      }
      // Check if it's an installed theme (will be validated later)
      return stored;
    }
    // Legacy migration
    const oldTheme = localStorage.getItem('vstats-theme');
    if (oldTheme === 'light') return 'daylight';
    if (window.matchMedia('(prefers-color-scheme: light)').matches) return 'daylight';
    return 'midnight';
  });

  const [background, setBackgroundState] = useState<BackgroundConfig>(() => {
    const stored = localStorage.getItem('vstats-background');
    if (stored) {
      try {
        return JSON.parse(stored);
      } catch { /* ignore */ }
    }
    return DEFAULT_BACKGROUND;
  });

  const [backgroundUrl, setBackgroundUrl] = useState<string | null>(null);
  const [serverSettingsApplied, setServerSettingsApplied] = useState(false);
  const [installedThemes, setInstalledThemes] = useState<InstalledTheme[]>([]);

  // Initialize theme manager and load installed themes
  useEffect(() => {
    initThemeManager();
    
    // Load local installed themes
    const localThemes = getInstalledThemes();
    setInstalledThemes(localThemes);
    
    // Also fetch from server
    fetchInstalledThemes();
  }, []);

  // Fetch installed themes from server
  const fetchInstalledThemes = async () => {
    try {
      const response = await fetch('/api/themes');
      if (response.ok) {
        const data = await response.json();
        if (data.installed && Array.isArray(data.installed)) {
          const themes = data.installed.map(serverToLocalInstalledTheme);
          setInstalledThemes(themes);
          
          // Apply CSS for each installed theme
          for (const theme of themes) {
            if (theme.cssContent) {
              applyThemeCSS(theme.manifest.id, theme.cssContent);
            }
          }
        }
      }
    } catch (e) {
      console.error('Failed to fetch installed themes:', e);
    }
  };

  // Get current theme config
  const theme = (() => {
    // Check builtin themes first
    const builtin = getBuiltinTheme(themeId);
    if (builtin) return builtin;
    
    // Check installed themes
    const installed = installedThemes.find(t => t.manifest.id === themeId);
    if (installed) return installedToConfig(installed);
    
    // Fallback to midnight
    return BUILTIN_THEMES[0];
  })();

  // Combine all available themes
  const allThemes: ThemeConfig[] = [
    ...BUILTIN_THEMES,
    ...installedThemes.filter(t => t.enabled).map(installedToConfig),
  ];

  // 获取背景图
  const refreshBackground = useCallback(async () => {
    if (background.type === 'bing') {
      const url = await fetchBingWallpaper();
      setBackgroundUrl(url);
    } else if (background.type === 'unsplash') {
      const url = await fetchUnsplashImage(background.unsplashQuery);
      setBackgroundUrl(url);
    } else if (background.type === 'custom' && background.customUrl) {
      const url = await fetchCustomWallpaper(background.customUrl);
      setBackgroundUrl(url);
    } else {
      setBackgroundUrl(null);
    }
  }, [background.type, background.customUrl, background.unsplashQuery]);

  useEffect(() => {
    refreshBackground();
  }, [refreshBackground]);

  useEffect(() => {
    localStorage.setItem('vstats-theme-id', themeId);
    localStorage.setItem('vstats-theme', theme.isDark ? 'dark' : 'light');
    
    // 移除所有主题类
    const classList = document.documentElement.classList;
    [...BUILTIN_THEMES, ...installedThemes.map(t => ({ id: t.manifest.id }))].forEach(t => {
      classList.remove(`theme-${t.id}`);
    });
    classList.remove('light-theme', 'dark-theme');
    
    // 添加新主题类
    classList.add(`theme-${themeId}`);
    classList.add(theme.isDark ? 'dark-theme' : 'light-theme');
    
    // 设置 CSS 变量
    document.documentElement.style.setProperty('--theme-border-radius', theme.borderRadius);
    document.documentElement.style.setProperty('--theme-font-heading', theme.fonts.heading);
    document.documentElement.style.setProperty('--theme-font-body', theme.fonts.body);
    document.documentElement.style.setProperty('--theme-font-mono', theme.fonts.mono);
  }, [themeId, theme, installedThemes]);

  useEffect(() => {
    localStorage.setItem('vstats-background', JSON.stringify(background));
  }, [background]);

  const setTheme = (newThemeId: string) => setThemeId(newThemeId);
  
  const setBackground = (config: BackgroundConfig) => setBackgroundState(config);

  // Install a theme from source
  const installTheme = async (source: string, ref?: string): Promise<InstallThemeResult> => {
    try {
      // Try server-side installation first
      const response = await fetch('/api/themes/install', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('vstats-token')}`,
        },
        body: JSON.stringify({ source, ref }),
      });
      
      if (response.ok) {
        const data = await response.json();
        if (data.success && data.theme) {
          const theme = serverToLocalInstalledTheme(data.theme);
          setInstalledThemes(prev => {
            const exists = prev.findIndex(t => t.manifest.id === theme.manifest.id);
            if (exists >= 0) {
              const newThemes = [...prev];
              newThemes[exists] = theme;
              return newThemes;
            }
            return [...prev, theme];
          });
          
          // Apply CSS
          if (theme.cssContent) {
            applyThemeCSS(theme.manifest.id, theme.cssContent);
          }
          
          return { success: true, theme };
        }
        return { success: false, error: data.error || 'Unknown error' };
      }
      
      // If server fails (e.g., not authenticated), try client-side
      const result = await installThemeFromSource({ source, ref });
      if (result.success && result.theme) {
        setInstalledThemes(prev => [...prev, result.theme!]);
      }
      return result;
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to install theme',
      };
    }
  };

  // Uninstall a theme
  const uninstallTheme = async (targetThemeId: string): Promise<boolean> => {
    try {
      // Try server-side uninstall first
      const response = await fetch(`/api/themes/${targetThemeId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('vstats-token')}`,
        },
      });
      
      if (response.ok) {
        setInstalledThemes(prev => prev.filter(t => t.manifest.id !== targetThemeId));
        
        // If current theme is uninstalled, switch to default
        if (themeId === targetThemeId) {
          setThemeId('midnight');
        }
        
        return true;
      }
      
      // Try client-side removal
      const success = removeTheme(targetThemeId);
      if (success) {
        setInstalledThemes(prev => prev.filter(t => t.manifest.id !== targetThemeId));
        if (themeId === targetThemeId) {
          setThemeId('midnight');
        }
      }
      return success;
    } catch (error) {
      console.error('Failed to uninstall theme:', error);
      return false;
    }
  };

  // Refresh themes from server
  const refreshThemes = async () => {
    await fetchInstalledThemes();
  };

  // Apply settings from server (called when WebSocket receives site_settings)
  const applyServerSettings = useCallback((settings: ServerThemeSettings | null) => {
    if (!settings) return;
    
    // Apply theme ID
    const validThemeId = settings.theme_id;
    if (validThemeId) {
      // Check if it's valid (builtin or installed)
      const isBuiltin = BUILTIN_THEMES.some(t => t.id === validThemeId);
      const isInstalled = installedThemes.some(t => t.manifest.id === validThemeId);
      
      if (isBuiltin || isInstalled) {
        setThemeId(validThemeId);
      }
    }
    
    // Apply background settings
    if (settings.background) {
      setBackgroundState(serverToLocalBackground(settings.background));
    }
    
    setServerSettingsApplied(true);
  }, [installedThemes]);

  // Get settings in server format (for saving)
  const getServerSettings = useCallback((): ServerThemeSettings => {
    return {
      theme_id: themeId,
      background: localToServerBackground(background),
    };
  }, [themeId, background]);

  // Fetch initial settings from server on mount
  useEffect(() => {
    if (serverSettingsApplied) return;
    
    const fetchServerSettings = async () => {
      try {
        const response = await fetch('/api/settings/site');
        if (response.ok) {
          const data = await response.json();
          if (data.theme) {
            applyServerSettings(data.theme);
          }
        }
      } catch (e) {
        console.error('Failed to fetch site settings:', e);
      }
    };
    
    fetchServerSettings();
  }, [applyServerSettings, serverSettingsApplied]);

  // Listen for WebSocket site settings updates
  useEffect(() => {
    const handleSiteSettingsUpdate = (event: CustomEvent) => {
      const siteSettings = event.detail;
      if (siteSettings?.theme) {
        applyServerSettings(siteSettings.theme);
      }
    };
    
    window.addEventListener('vstats-site-settings', handleSiteSettingsUpdate as EventListener);
    return () => {
      window.removeEventListener('vstats-site-settings', handleSiteSettingsUpdate as EventListener);
    };
  }, [applyServerSettings]);

  return (
    <ThemeContext.Provider value={{ 
      themeId, 
      theme, 
      isDark: theme.isDark, 
      setTheme, 
      themes: allThemes,
      builtinThemes: BUILTIN_THEMES,
      installedThemes,
      background,
      setBackground,
      backgroundUrl,
      refreshBackground,
      installTheme,
      uninstallTheme,
      refreshThemes,
      applyServerSettings,
      getServerSettings,
    }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider');
  }
  return context;
}

// Re-export THEMES for backward compatibility
export const THEMES = BUILTIN_THEMES;

// Re-export ThemeConfig for backward compatibility
export type { ThemeConfig };
