/**
 * VStats Builtin Themes
 * 
 * Only the default midnight theme is bundled.
 * Other themes have been migrated to vstats-theme repository
 * and can be installed from GitHub.
 */

import type { ThemeConfig } from './types';

// Re-export ThemeId for compatibility
export type ThemeId = 'midnight';

export const BUILTIN_THEMES: ThemeConfig[] = [
  {
    id: 'midnight',
    name: 'Midnight Tech',
    nameZh: '午夜科技',
    description: 'Classic dark tech theme with blue accents',
    descriptionZh: '深邃蓝黑渐变，科技蓝光，毛玻璃卡片',
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
    cardStyle: 'glass',
    isBuiltin: true,
  },
];

/**
 * Get a builtin theme by ID
 */
export function getBuiltinTheme(id: string): ThemeConfig | undefined {
  return BUILTIN_THEMES.find(t => t.id === id);
}

/**
 * Check if a theme ID is a builtin theme
 */
export function isBuiltinTheme(id: string): boolean {
  return BUILTIN_THEMES.some(t => t.id === id);
}
