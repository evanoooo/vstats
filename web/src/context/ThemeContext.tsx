import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import { sanitizeUrl } from '../utils/security';

// 主题类型定义
export type ThemeId = 
  | 'midnight'       // 午夜深蓝 - 经典科技风
  | 'daylight'       // 晴空日光 - 清新简约
  | 'cyberpunk'      // 赛博朋克 - 霓虹科幻
  | 'terminal'       // 终端黑客 - 复古终端
  | 'glassmorphism'  // 毛玻璃 - 现代透明
  | 'neumorphism'    // 新拟态 - 软UI风格
  | 'brutalist'      // 野兽派 - 大胆粗犷
  | 'minimal';       // 极简主义 - 纯净留白

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

// 主题配置
export interface ThemeConfig {
  id: ThemeId;
  name: string;
  nameZh: string;
  description: string;
  descriptionZh: string;
  isDark: boolean;
  style: 'flat' | 'glass' | 'neumorphic' | 'brutalist' | 'minimal';
  preview: {
    primary: string;
    secondary: string;
    accent: string;
    background: string;
  };
  fonts: {
    heading: string;
    body: string;
    mono: string;
  };
  borderRadius: string;
  cardStyle: string;
}

export const THEMES: ThemeConfig[] = [
  {
    id: 'midnight',
    name: 'Midnight Tech',
    nameZh: '午夜科技',
    description: 'Classic dark tech theme with blue accents',
    descriptionZh: '经典深色科技风，蓝色强调',
    isDark: true,
    style: 'glass',
    preview: {
      primary: '#020617',
      secondary: '#0f172a',
      accent: '#3b82f6',
      background: '#020617'
    },
    fonts: {
      heading: '"SF Pro Display", -apple-system, sans-serif',
      body: '"Inter", system-ui, sans-serif',
      mono: '"SF Mono", "Fira Code", monospace'
    },
    borderRadius: '16px',
    cardStyle: 'glass'
  },
  {
    id: 'daylight',
    name: 'Daylight',
    nameZh: '晴空日光',
    description: 'Clean and bright with soft shadows',
    descriptionZh: '清新明亮，柔和阴影',
    isDark: false,
    style: 'flat',
    preview: {
      primary: '#ffffff',
      secondary: '#f8fafc',
      accent: '#0ea5e9',
      background: '#f1f5f9'
    },
    fonts: {
      heading: '"Plus Jakarta Sans", sans-serif',
      body: '"Inter", system-ui, sans-serif',
      mono: '"JetBrains Mono", monospace'
    },
    borderRadius: '20px',
    cardStyle: 'elevated'
  },
  {
    id: 'cyberpunk',
    name: 'Cyberpunk 2077',
    nameZh: '赛博朋克',
    description: 'Neon lights, glitch effects, futuristic',
    descriptionZh: '霓虹灯光，故障艺术，未来感',
    isDark: true,
    style: 'brutalist',
    preview: {
      primary: '#0a0a0f',
      secondary: '#1a1a2e',
      accent: '#ff00ff',
      background: '#0a0a0f'
    },
    fonts: {
      heading: '"Orbitron", "Rajdhani", sans-serif',
      body: '"Rajdhani", "Share Tech Mono", sans-serif',
      mono: '"Share Tech Mono", monospace'
    },
    borderRadius: '4px',
    cardStyle: 'neon'
  },
  {
    id: 'terminal',
    name: 'Hacker Terminal',
    nameZh: '黑客终端',
    description: 'Retro terminal with scanlines',
    descriptionZh: '复古终端，扫描线效果',
    isDark: true,
    style: 'minimal',
    preview: {
      primary: '#0d0d0d',
      secondary: '#1a1a1a',
      accent: '#00ff41',
      background: '#0d0d0d'
    },
    fonts: {
      heading: '"VT323", "Fira Code", monospace',
      body: '"Fira Code", "IBM Plex Mono", monospace',
      mono: '"Fira Code", monospace'
    },
    borderRadius: '0px',
    cardStyle: 'terminal'
  },
  {
    id: 'glassmorphism',
    name: 'Glass UI',
    nameZh: '毛玻璃',
    description: 'Frosted glass with vibrant backgrounds',
    descriptionZh: '磨砂玻璃效果，鲜艳背景',
    isDark: true,
    style: 'glass',
    preview: {
      primary: 'rgba(255,255,255,0.1)',
      secondary: 'rgba(255,255,255,0.05)',
      accent: '#a855f7',
      background: '#1e1b4b'
    },
    fonts: {
      heading: '"Poppins", sans-serif',
      body: '"Inter", system-ui, sans-serif',
      mono: '"JetBrains Mono", monospace'
    },
    borderRadius: '24px',
    cardStyle: 'frosted'
  },
  {
    id: 'neumorphism',
    name: 'Soft UI',
    nameZh: '新拟态',
    description: 'Soft shadows and embossed elements',
    descriptionZh: '柔和阴影，浮雕效果',
    isDark: false,
    style: 'neumorphic',
    preview: {
      primary: '#e0e5ec',
      secondary: '#e0e5ec',
      accent: '#6366f1',
      background: '#e0e5ec'
    },
    fonts: {
      heading: '"Nunito", sans-serif',
      body: '"Nunito", system-ui, sans-serif',
      mono: '"Source Code Pro", monospace'
    },
    borderRadius: '20px',
    cardStyle: 'neumorphic'
  },
  {
    id: 'brutalist',
    name: 'Brutalist',
    nameZh: '野兽派',
    description: 'Bold, raw, unapologetic design',
    descriptionZh: '大胆粗犷，原始风格',
    isDark: false,
    style: 'brutalist',
    preview: {
      primary: '#ffffff',
      secondary: '#f5f5f5',
      accent: '#ff0000',
      background: '#ffffff'
    },
    fonts: {
      heading: '"Archivo Black", "Impact", sans-serif',
      body: '"Space Mono", monospace',
      mono: '"Space Mono", monospace'
    },
    borderRadius: '0px',
    cardStyle: 'brutalist'
  },
  {
    id: 'minimal',
    name: 'Minimal Zen',
    nameZh: '极简禅意',
    description: 'Maximum whitespace, minimal elements',
    descriptionZh: '大量留白，极致简约',
    isDark: false,
    style: 'minimal',
    preview: {
      primary: '#fafafa',
      secondary: '#ffffff',
      accent: '#18181b',
      background: '#fafafa'
    },
    fonts: {
      heading: '"DM Sans", sans-serif',
      body: '"DM Sans", system-ui, sans-serif',
      mono: '"DM Mono", monospace'
    },
    borderRadius: '8px',
    cardStyle: 'minimal'
  }
];

// 默认背景配置
const DEFAULT_BACKGROUND: BackgroundConfig = {
  type: 'gradient',
  blur: 0,
  opacity: 100
};

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

interface ThemeContextType {
  themeId: ThemeId;
  theme: ThemeConfig;
  isDark: boolean;
  setTheme: (themeId: ThemeId) => void;
  themes: ThemeConfig[];
  background: BackgroundConfig;
  setBackground: (config: BackgroundConfig) => void;
  backgroundUrl: string | null;
  refreshBackground: () => void;
  // Sync from server
  applyServerSettings: (settings: ServerThemeSettings | null) => void;
  // For saving to server (returns the server format)
  getServerSettings: () => ServerThemeSettings;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

// Fetch Bing wallpaper through our proxy API (server-side proxy to avoid CORS)
const fetchBingWallpaper = async (): Promise<string> => {
  try {
    const response = await fetch('/api/wallpaper/bing');
    
    if (!response.ok) {
      console.warn(`Bing wallpaper proxy returned ${response.status}: ${response.statusText}`);
      // Use fallback image if proxy fails
      return 'https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1920&q=80';
    }
    
    const data = await response.json();
    if (data && data.url) {
      return data.url;
    }
    
    console.warn('Bing wallpaper proxy returned invalid response:', data);
  } catch (e) {
    console.error('Failed to fetch Bing wallpaper through proxy:', e);
    // Never directly call Bing API - always use fallback
  }
  // Fallback image (Unsplash)
  return 'https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1920&q=80';
};

// Fetch custom wallpaper URL through our proxy API to avoid CORS
const fetchCustomWallpaper = async (imageURL: string): Promise<string> => {
  try {
    // Check if it's a relative URL or same-origin - use directly
    try {
      const url = new URL(imageURL, window.location.origin);
      if (url.origin === window.location.origin) {
        // Same origin, use directly
        return imageURL;
      }
    } catch {
      // If URL parsing fails, it might be a relative URL
      if (!imageURL.startsWith('http://') && !imageURL.startsWith('https://')) {
        return imageURL;
      }
    }

    // For external URLs, check if we need proxy
    const response = await fetch(`/api/wallpaper/proxy?url=${encodeURIComponent(imageURL)}`);
    
    if (!response.ok) {
      console.warn(`Custom wallpaper proxy returned ${response.status}: ${response.statusText}`);
      // Fallback to original URL - might work if CORS headers are present
      return imageURL;
    }
    
    const data = await response.json();
    if (data && data.url) {
      return data.url;
    }
    
    console.warn('Custom wallpaper proxy returned invalid response:', data);
  } catch (e) {
    console.error('Failed to fetch custom wallpaper through proxy:', e);
    // Fallback to original URL
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
      // Use fallback if proxy fails
      return `https://source.unsplash.com/1920x1080/?${encodeURIComponent(keywords)}&t=${Date.now()}`;
    }
    
    const data = await response.json();
    if (data && data.url) {
      return data.url;
    }
    
    console.warn('Unsplash wallpaper proxy returned invalid response:', data);
  } catch (e) {
    console.error('Failed to fetch Unsplash image through proxy:', e);
    // Never directly call Unsplash API - use fallback URL
  }
  // Fallback: return the redirect URL directly with timestamp to avoid caching
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

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [themeId, setThemeId] = useState<ThemeId>(() => {
    const stored = localStorage.getItem('vstats-theme-id') as ThemeId;
    if (stored && THEMES.find(t => t.id === stored)) {
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

  const theme = THEMES.find(t => t.id === themeId) || THEMES[0];

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
    THEMES.forEach(t => classList.remove(`theme-${t.id}`));
    classList.remove('light-theme', 'dark-theme');
    
    // 添加新主题类
    classList.add(`theme-${themeId}`);
    classList.add(theme.isDark ? 'dark-theme' : 'light-theme');
    
    // 设置 CSS 变量
    document.documentElement.style.setProperty('--theme-border-radius', theme.borderRadius);
    document.documentElement.style.setProperty('--theme-font-heading', theme.fonts.heading);
    document.documentElement.style.setProperty('--theme-font-body', theme.fonts.body);
    document.documentElement.style.setProperty('--theme-font-mono', theme.fonts.mono);
  }, [themeId, theme]);

  useEffect(() => {
    localStorage.setItem('vstats-background', JSON.stringify(background));
  }, [background]);

  const setTheme = (newThemeId: ThemeId) => setThemeId(newThemeId);
  
  const setBackground = (config: BackgroundConfig) => setBackgroundState(config);

  // Apply settings from server (called when WebSocket receives site_settings)
  const applyServerSettings = useCallback((settings: ServerThemeSettings | null) => {
    if (!settings) return;
    
    // Apply theme ID
    if (settings.theme_id && THEMES.find(t => t.id === settings.theme_id)) {
      setThemeId(settings.theme_id as ThemeId);
    }
    
    // Apply background settings
    if (settings.background) {
      setBackgroundState(serverToLocalBackground(settings.background));
    }
    
    setServerSettingsApplied(true);
  }, []);

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
      themes: THEMES,
      background,
      setBackground,
      backgroundUrl,
      refreshBackground,
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
