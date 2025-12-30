/**
 * Theme Settings Section Component
 * Handles theme selection, background settings, and theme installation
 */

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useTheme, type BackgroundType } from '../../context/ThemeContext';
import { showToast } from '../../components/Toast';
import { BACKGROUND_OPTIONS, UNSPLASH_PRESETS } from './constants';
import type { ThemeSettingsSectionProps } from './types';

export function ThemeSettingsSection({ 
  isAuthenticated, 
  token, 
  siteSettings, 
  onSiteSettingsChange 
}: ThemeSettingsSectionProps) {
  const { i18n } = useTranslation();
  const { 
    themeId, 
    setTheme, 
    themes, 
    installedThemes, 
    installTheme, 
    uninstallTheme, 
    background, 
    setBackground, 
    backgroundUrl, 
    refreshBackground, 
    getServerSettings 
  } = useTheme();
  
  const [hoveredTheme, setHoveredTheme] = useState<string | null>(null);
  const [customUrl, setCustomUrl] = useState(background.customUrl || '');
  const [unsplashQuery, setUnsplashQuery] = useState(background.unsplashQuery || 'nature,landscape');
  const [solidColor, setSolidColor] = useState(background.solidColor || '#1a1a2e');
  const [bgBlur, setBgBlur] = useState(background.blur || 0);
  const [bgOpacity, setBgOpacity] = useState(background.opacity || 100);
  const [saving, setSaving] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const isZh = i18n.language.startsWith('zh');
  
  // Theme installation states
  const [showThemeInstaller, setShowThemeInstaller] = useState(false);
  const [themeSource, setThemeSource] = useState('zsai001/vstats-theme/themes/daylight');
  const [themeRef, setThemeRef] = useState('');
  const [isInstalling, setIsInstalling] = useState(false);
  const [installError, setInstallError] = useState<string | null>(null);
  const [installSuccess, setInstallSuccess] = useState<string | null>(null);

  // Handle theme installation
  const handleInstallTheme = async () => {
    if (!themeSource.trim()) {
      setInstallError(isZh ? '请输入主题来源' : 'Please enter theme source');
      return;
    }

    setIsInstalling(true);
    setInstallError(null);
    setInstallSuccess(null);

    try {
      const result = await installTheme(themeSource.trim(), themeRef.trim() || undefined);
      
      if (result.success) {
        setInstallSuccess(isZh 
          ? `主题 "${result.theme?.manifest.name}" 安装成功！` 
          : `Theme "${result.theme?.manifest.name}" installed successfully!`
        );
        setThemeSource('');
        setThemeRef('');
        showToast(isZh ? '主题安装成功' : 'Theme installed successfully', 'success');
      } else {
        setInstallError(result.error || (isZh ? '安装失败' : 'Installation failed'));
      }
    } catch (e) {
      setInstallError(e instanceof Error ? e.message : (isZh ? '未知错误' : 'Unknown error'));
    } finally {
      setIsInstalling(false);
    }
  };

  // Handle theme uninstallation
  const handleUninstallTheme = async (id: string, name: string) => {
    if (!confirm(isZh ? `确定要卸载主题 "${name}" 吗？` : `Are you sure you want to uninstall "${name}"?`)) {
      return;
    }

    const success = await uninstallTheme(id);
    if (success) {
      showToast(isZh ? `主题 "${name}" 已卸载` : `Theme "${name}" uninstalled`, 'success');
    } else {
      showToast(isZh ? '卸载失败' : 'Uninstall failed', 'error');
    }
  };

  // Save theme settings to server (optionally with a specific theme ID)
  const saveThemeSettings = async (overrideThemeId?: string) => {
    if (!isAuthenticated || !token) {
      showToast(isZh ? '请先登录' : 'Please login first', 'error');
      return;
    }
    
    setSaving(true);
    try {
      const themeSettings = getServerSettings();
      // If an override theme ID is provided, use it instead
      if (overrideThemeId) {
        themeSettings.theme_id = overrideThemeId;
      }
      const updatedSiteSettings = {
        ...siteSettings,
        theme: themeSettings,
      };
      
      const res = await fetch('/api/settings/site', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify(updatedSiteSettings),
      });
      
      if (res.ok) {
        onSiteSettingsChange(updatedSiteSettings);
        setSaveSuccess(true);
        showToast(isZh ? '主题设置已保存' : 'Theme settings saved', 'success');
        setTimeout(() => setSaveSuccess(false), 3000);
      } else {
        showToast(isZh ? '保存失败' : 'Failed to save', 'error');
      }
    } catch (e) {
      console.error('Failed to save theme settings:', e);
      showToast(isZh ? '保存失败' : 'Failed to save', 'error');
    }
    setSaving(false);
  };

  // Handle theme selection - sets theme locally and saves to server
  const handleThemeSelect = (newThemeId: string) => {
    setTheme(newThemeId);
    // Auto-save to server if authenticated
    if (isAuthenticated && token) {
      saveThemeSettings(newThemeId);
    }
  };

  const handleBackgroundTypeChange = (type: BackgroundType) => {
    setBackground({
      ...background,
      type,
      customUrl: type === 'custom' ? customUrl : undefined,
      unsplashQuery: type === 'unsplash' ? unsplashQuery : undefined,
      solidColor: type === 'solid' ? solidColor : undefined,
    });
  };

  const handleApplyCustomUrl = () => {
    if (customUrl) {
      setBackground({ ...background, type: 'custom', customUrl });
    }
  };

  const handleApplyUnsplash = (query?: string) => {
    const q = query || unsplashQuery;
    setUnsplashQuery(q);
    setBackground({ ...background, type: 'unsplash', unsplashQuery: q });
  };

  const handleApplySolidColor = () => {
    setBackground({ ...background, type: 'solid', solidColor });
  };

  const handleBlurChange = (value: number) => {
    setBgBlur(value);
    setBackground({ ...background, blur: value });
  };

  const handleOpacityChange = (value: number) => {
    setBgOpacity(value);
    setBackground({ ...background, opacity: value });
  };

  return (
    <div className="space-y-6 mb-6">
      {/* Theme Selection Card */}
      <div className="nezha-card p-6">
        <h2 className="text-lg font-bold text-white flex items-center gap-2 mb-4">
          <span className="w-2 h-2 rounded-full bg-gradient-to-r from-pink-500 to-purple-500"></span>
          {isZh ? '主题风格' : 'Theme Style'}
        </h2>
        
        <p className="text-sm text-gray-400 mb-6">
          {isZh ? '每个主题都有独特的视觉设计和动效' : 'Each theme has unique visual design and animations'}
        </p>

        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {themes.map((theme) => {
            const isSelected = themeId === theme.id;
            const isHovered = hoveredTheme === theme.id;
            
            return (
              <button
                key={theme.id}
                onClick={() => handleThemeSelect(theme.id)}
                onMouseEnter={() => setHoveredTheme(theme.id)}
                onMouseLeave={() => setHoveredTheme(null)}
                className={`
                  relative group p-4 rounded-xl border-2 transition-all duration-300 text-left
                  ${isSelected 
                    ? 'border-emerald-500 ring-2 ring-emerald-500/30 scale-[1.02]' 
                    : 'border-white/10 hover:border-white/30 hover:scale-[1.01]'
                  }
                `}
                style={{
                  background: `linear-gradient(135deg, ${theme.preview.background}ee 0%, ${theme.preview.primary}ee 100%)`
                }}
              >
                {/* Theme Preview Mini Card */}
                <div 
                  className="w-full h-16 rounded-lg mb-3 relative overflow-hidden border border-white/10"
                  style={{ backgroundColor: theme.preview.background }}
                >
                  {/* Mini card inside preview */}
                  <div 
                    className="absolute bottom-2 left-2 right-2 h-8 rounded"
                    style={{ 
                      backgroundColor: theme.preview.primary,
                      boxShadow: `0 2px 8px ${theme.preview.accent}30`
                    }}
                  />
                  {/* Accent line */}
                  <div 
                    className="absolute top-0 left-0 right-0 h-1"
                    style={{ backgroundColor: theme.preview.accent }}
                  />
                </div>

                {/* Theme Name */}
                <div className="font-semibold text-sm text-white mb-1">
                  {isZh ? theme.nameZh : theme.name}
                </div>

                {/* Theme Style Badge */}
                <div className="flex items-center gap-2">
                  <span className={`text-[10px] px-1.5 py-0.5 rounded ${
                    theme.isDark ? 'bg-gray-700 text-gray-300' : 'bg-amber-100 text-amber-700'
                  }`}>
                    {theme.isDark ? (isZh ? '深色' : 'Dark') : (isZh ? '浅色' : 'Light')}
                  </span>
                  <span className="text-[10px] px-1.5 py-0.5 rounded bg-white/10 text-gray-400 capitalize">
                    {theme.style}
                  </span>
                </div>

                {/* Selected Indicator */}
                {isSelected && (
                  <div className="absolute -top-1 -right-1 w-6 h-6 bg-emerald-500 rounded-full flex items-center justify-center shadow-lg">
                    <svg className="w-3.5 h-3.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                    </svg>
                  </div>
                )}

                {/* Hover Effect */}
                <div 
                  className={`
                    absolute inset-0 rounded-xl transition-opacity duration-300 pointer-events-none
                    ${isHovered && !isSelected ? 'opacity-100' : 'opacity-0'}
                  `}
                  style={{
                    background: `radial-gradient(circle at center, ${theme.preview.accent}20 0%, transparent 70%)`
                  }}
                />
              </button>
            );
          })}
        </div>

        {/* Install Theme Section */}
        {isAuthenticated && (
          <div className="mt-6 pt-6 border-t border-white/10">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="font-semibold text-white">
                  {isZh ? '安装更多主题' : 'Install More Themes'}
                </h3>
                <p className="text-sm text-gray-400 mt-1">
                  {isZh ? '从 GitHub 安装第三方主题' : 'Install third-party themes from GitHub'}
                </p>
              </div>
              <button
                onClick={() => setShowThemeInstaller(!showThemeInstaller)}
                className="px-4 py-2 rounded-lg bg-purple-500/10 hover:bg-purple-500/20 text-purple-400 text-sm font-medium transition-colors flex items-center gap-2"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                </svg>
                {showThemeInstaller 
                  ? (isZh ? '收起' : 'Hide') 
                  : (isZh ? '安装主题' : 'Install Theme')}
              </button>
            </div>

            {showThemeInstaller && (
              <div className="space-y-4 p-4 rounded-xl bg-white/5 border border-white/10">
                {/* Recommended Themes */}
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {isZh ? '推荐主题' : 'Recommended Themes'}
                  </label>
                  <div className="grid grid-cols-1 gap-2">
                    {[
                      { 
                        source: 'zsai001/vstats-theme/themes/daylight', 
                        name: 'Daylight', 
                        nameZh: '日光',
                        desc: 'Clean light theme',
                        descZh: '清新明亮主题',
                        colors: ['#f8fafc', '#3b82f6', '#10b981']
                      },
                    ].map((theme) => (
                      <button
                        key={theme.source}
                        onClick={() => {
                          setThemeSource(theme.source);
                          setInstallError(null);
                          setInstallSuccess(null);
                        }}
                        disabled={isInstalling}
                        className={`
                          flex items-center gap-3 p-3 rounded-lg border transition-all text-left
                          ${themeSource === theme.source 
                            ? 'border-purple-500 bg-purple-500/10' 
                            : 'border-white/10 hover:border-white/30 bg-white/5'
                          }
                        `}
                      >
                        <div className="flex -space-x-1">
                          {theme.colors.map((color, i) => (
                            <div 
                              key={i}
                              className="w-5 h-5 rounded-full border-2 border-white/20"
                              style={{ backgroundColor: color }}
                            />
                          ))}
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium text-white">
                            {isZh ? theme.nameZh : theme.name}
                          </div>
                          <div className="text-xs text-gray-400 truncate">
                            {isZh ? theme.descZh : theme.desc}
                          </div>
                        </div>
                        {themeSource === theme.source && (
                          <svg className="w-4 h-4 text-purple-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                          </svg>
                        )}
                      </button>
                    ))}
                  </div>
                </div>

                <div className="border-t border-white/10 pt-4">
                  <p className="text-sm text-gray-400 mb-2">
                    {isZh 
                      ? '或从 GitHub 安装其他主题：' 
                      : 'Or install other themes from GitHub:'
                    }
                  </p>
                  <ul className="text-xs text-gray-500 space-y-1 ml-4 mb-3">
                    <li><code className="bg-white/10 px-1.5 py-0.5 rounded">user/repo</code> - {isZh ? '仓库根目录' : 'Repository root'}</li>
                    <li><code className="bg-white/10 px-1.5 py-0.5 rounded">user/repo/themes/my-theme</code> - {isZh ? '子目录' : 'Subdirectory'}</li>
                    <li><code className="bg-white/10 px-1.5 py-0.5 rounded">user/repo@v1.0.0</code> - {isZh ? '指定版本/分支' : 'Specific version/branch'}</li>
                  </ul>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-1">
                      {isZh ? '主题来源' : 'Theme Source'} *
                    </label>
                    <input
                      type="text"
                      value={themeSource}
                      onChange={(e) => setThemeSource(e.target.value)}
                      placeholder="user/repo 或 https://..."
                      disabled={isInstalling}
                      className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-1">
                      {isZh ? '版本/分支 (可选)' : 'Version/Branch (optional)'}
                    </label>
                    <input
                      type="text"
                      value={themeRef}
                      onChange={(e) => setThemeRef(e.target.value)}
                      placeholder="main, v1.0.0, ..."
                      disabled={isInstalling}
                      className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                    />
                  </div>
                </div>

                <button
                  onClick={handleInstallTheme}
                  disabled={isInstalling || !themeSource.trim()}
                  className={`
                    w-full px-4 py-2.5 rounded-lg text-white text-sm font-medium transition-all flex items-center justify-center gap-2
                    ${isInstalling || !themeSource.trim()
                      ? 'bg-gray-600 cursor-not-allowed' 
                      : 'bg-purple-500 hover:bg-purple-600'
                    }
                  `}
                >
                  {isInstalling ? (
                    <>
                      <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                      </svg>
                      {isZh ? '安装中...' : 'Installing...'}
                    </>
                  ) : (
                    <>
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                      </svg>
                      {isZh ? '安装主题' : 'Install Theme'}
                    </>
                  )}
                </button>

                {installError && (
                  <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
                    {installError}
                  </div>
                )}
                {installSuccess && (
                  <div className="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-sm">
                    {installSuccess}
                  </div>
                )}
              </div>
            )}

            {/* Installed Themes List */}
            {installedThemes.length > 0 && (
              <div className="mt-4 p-4 rounded-xl bg-white/5 border border-white/10">
                <h4 className="font-medium text-white mb-3">
                  {isZh ? '已安装的第三方主题' : 'Installed Third-party Themes'} ({installedThemes.length})
                </h4>
                <div className="space-y-3">
                  {installedThemes.map((installed) => (
                    <div key={installed.manifest.id} className="flex items-center gap-4 p-3 rounded-lg bg-white/5">
                      <div 
                        className="w-12 h-12 rounded-lg flex-shrink-0"
                        style={{
                          background: `linear-gradient(135deg, ${installed.manifest.preview.primary} 0%, ${installed.manifest.preview.secondary} 50%, ${installed.manifest.preview.accent} 100%)`
                        }}
                      />
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-white">
                          {isZh && installed.manifest.nameZh ? installed.manifest.nameZh : installed.manifest.name}
                        </div>
                        <div className="text-xs text-gray-500">
                          v{installed.manifest.version} • {installed.manifest.author}
                        </div>
                        <div className="text-xs text-gray-400 truncate">
                          {isZh && installed.manifest.descriptionZh 
                            ? installed.manifest.descriptionZh 
                            : installed.manifest.description
                          }
                        </div>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        {themeId === installed.manifest.id ? (
                          <span className="px-2 py-1 rounded bg-emerald-500/20 text-emerald-400 text-xs">
                            {isZh ? '使用中' : 'Active'}
                          </span>
                        ) : (
                          <button
                            onClick={() => handleThemeSelect(installed.manifest.id)}
                            className="px-3 py-1 rounded bg-white/10 hover:bg-white/20 text-white text-xs transition-colors"
                          >
                            {isZh ? '使用' : 'Use'}
                          </button>
                        )}
                        <button
                          onClick={() => handleUninstallTheme(installed.manifest.id, installed.manifest.name)}
                          className="px-3 py-1 rounded bg-red-500/10 hover:bg-red-500/20 text-red-400 text-xs transition-colors"
                        >
                          {isZh ? '卸载' : 'Uninstall'}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Theme Development Link */}
            <div className="mt-4 text-center text-sm text-gray-500">
              {isZh ? '想要创建自己的主题？' : 'Want to create your own theme?'}
              <a 
                href="https://github.com/zsai001/vstats/blob/main/docs/THEME-DEVELOPMENT.md" 
                target="_blank" 
                rel="noopener noreferrer"
                className="text-purple-400 hover:text-purple-300 ml-1"
              >
                {isZh ? '查看主题开发指南' : 'View Theme Development Guide'}
              </a>
            </div>
          </div>
        )}
      </div>

      {/* Background Settings Card */}
      <div className="nezha-card p-6">
        <h2 className="text-lg font-bold text-white flex items-center gap-2 mb-4">
          <span className="w-2 h-2 rounded-full bg-gradient-to-r from-blue-500 to-cyan-500"></span>
          {isZh ? '背景设置' : 'Background Settings'}
        </h2>

        {/* Background Type Selection */}
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3 mb-6">
          {BACKGROUND_OPTIONS.map((option) => (
            <button
              key={option.type}
              onClick={() => handleBackgroundTypeChange(option.type)}
              className={`
                p-3 rounded-xl border-2 transition-all text-center
                ${background.type === option.type 
                  ? 'border-blue-500 bg-blue-500/10' 
                  : 'border-white/10 hover:border-white/30 bg-white/5'
                }
              `}
            >
              <div className="text-2xl mb-1">{option.icon}</div>
              <div className="text-xs font-medium text-white">
                {isZh ? option.nameZh : option.name}
              </div>
            </button>
          ))}
        </div>

        {/* Background Type Specific Settings */}
        {background.type === 'custom' && (
          <div className="space-y-3 mb-6 p-4 rounded-lg bg-white/5 border border-white/10">
            <label className="block text-sm font-medium text-gray-300">
              {isZh ? '图片 URL' : 'Image URL'}
            </label>
            <div className="flex gap-2">
              <input
                type="url"
                value={customUrl}
                onChange={(e) => setCustomUrl(e.target.value)}
                placeholder="https://example.com/image.jpg"
                className="flex-1 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
              />
              <button
                onClick={handleApplyCustomUrl}
                className="px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-sm font-medium transition-colors"
              >
                {isZh ? '应用' : 'Apply'}
              </button>
            </div>
          </div>
        )}

        {background.type === 'unsplash' && (
          <div className="space-y-4 mb-6 p-4 rounded-lg bg-white/5 border border-white/10">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                {isZh ? '快速选择' : 'Quick Select'}
              </label>
              <div className="flex flex-wrap gap-2">
                {UNSPLASH_PRESETS.map((preset) => (
                  <button
                    key={preset.query}
                    onClick={() => handleApplyUnsplash(preset.query)}
                    className={`
                      px-3 py-1.5 rounded-full text-xs font-medium transition-all
                      ${unsplashQuery === preset.query 
                        ? 'bg-blue-500 text-white' 
                        : 'bg-white/10 text-gray-300 hover:bg-white/20'
                      }
                    `}
                  >
                    {isZh ? preset.labelZh : preset.label}
                  </button>
                ))}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                {isZh ? '自定义关键词' : 'Custom Keywords'}
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={unsplashQuery}
                  onChange={(e) => setUnsplashQuery(e.target.value)}
                  placeholder="nature,landscape"
                  className="flex-1 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
                />
                <button
                  onClick={() => handleApplyUnsplash()}
                  className="px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-sm font-medium transition-colors"
                >
                  {isZh ? '应用' : 'Apply'}
                </button>
                <button
                  onClick={refreshBackground}
                  className="px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 text-white text-sm font-medium transition-colors"
                  title={isZh ? '换一张' : 'Refresh'}
                >
                  🔄
                </button>
              </div>
            </div>
          </div>
        )}

        {background.type === 'solid' && (
          <div className="space-y-3 mb-6 p-4 rounded-lg bg-white/5 border border-white/10">
            <label className="block text-sm font-medium text-gray-300">
              {isZh ? '选择颜色' : 'Select Color'}
            </label>
            <div className="flex gap-3 items-center">
              <input
                type="color"
                value={solidColor}
                onChange={(e) => setSolidColor(e.target.value)}
                className="w-12 h-12 rounded-lg cursor-pointer border-0"
              />
              <input
                type="text"
                value={solidColor}
                onChange={(e) => setSolidColor(e.target.value)}
                className="flex-1 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50 font-mono"
              />
              <button
                onClick={handleApplySolidColor}
                className="px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-sm font-medium transition-colors"
              >
                {isZh ? '应用' : 'Apply'}
              </button>
            </div>
          </div>
        )}

        {background.type === 'bing' && (
          <div className="mb-6 p-4 rounded-lg bg-white/5 border border-white/10">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-white">
                  {isZh ? 'Bing 每日壁纸' : 'Bing Daily Wallpaper'}
                </div>
                <div className="text-xs text-gray-400">
                  {isZh ? '每天自动更新微软 Bing 精选壁纸' : 'Automatically updates with Microsoft Bing featured wallpaper daily'}
                </div>
              </div>
              <button
                onClick={refreshBackground}
                className="px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 text-white text-sm font-medium transition-colors"
              >
                🔄 {isZh ? '刷新' : 'Refresh'}
              </button>
            </div>
          </div>
        )}

        {/* Background Adjustments */}
        {(background.type === 'bing' || background.type === 'unsplash' || background.type === 'custom') && (
          <div className="space-y-4 p-4 rounded-lg bg-white/5 border border-white/10">
            <div className="text-sm font-medium text-gray-300 mb-2">
              {isZh ? '背景调整' : 'Background Adjustments'}
            </div>
            
            <div className="space-y-3">
              <div>
                <div className="flex justify-between text-xs text-gray-400 mb-1">
                  <span>{isZh ? '模糊程度' : 'Blur'}</span>
                  <span>{bgBlur}px</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max="30"
                  value={bgBlur}
                  onChange={(e) => handleBlurChange(Number(e.target.value))}
                  className="w-full h-2 bg-white/10 rounded-lg appearance-none cursor-pointer"
                />
              </div>
              
              <div>
                <div className="flex justify-between text-xs text-gray-400 mb-1">
                  <span>{isZh ? '遮罩透明度' : 'Overlay Opacity'}</span>
                  <span>{100 - bgOpacity}%</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max="100"
                  value={bgOpacity}
                  onChange={(e) => handleOpacityChange(Number(e.target.value))}
                  className="w-full h-2 bg-white/10 rounded-lg appearance-none cursor-pointer"
                />
              </div>
            </div>
          </div>
        )}

        {/* Background Preview */}
        {backgroundUrl && (
          <div className="mt-4">
            <div className="text-xs text-gray-400 mb-2">{isZh ? '当前背景预览' : 'Current Background Preview'}</div>
            <div 
              className="w-full h-32 rounded-lg bg-cover bg-center relative overflow-hidden"
              style={{ 
                backgroundImage: `url(${backgroundUrl})`,
                filter: `blur(${bgBlur}px)`
              }}
            >
              <div 
                className="absolute inset-0 bg-black"
                style={{ opacity: (100 - bgOpacity) / 100 }}
              />
            </div>
          </div>
        )}
      </div>

      {/* Save Theme Settings Button */}
      {isAuthenticated && (
        <div className="nezha-card p-6">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="font-semibold text-white">
                {isZh ? '保存主题设置' : 'Save Theme Settings'}
              </h3>
              <p className="text-sm text-gray-400 mt-1">
                {isZh ? '将主题设置应用到全站，所有用户都能看到' : 'Apply theme settings site-wide for all users'}
              </p>
            </div>
            <button
              onClick={() => saveThemeSettings()}
              disabled={saving}
              className={`
                px-6 py-2.5 rounded-lg text-white text-sm font-medium transition-all
                ${saving 
                  ? 'bg-gray-600 cursor-not-allowed' 
                  : saveSuccess 
                    ? 'bg-emerald-500 hover:bg-emerald-600' 
                    : 'bg-blue-500 hover:bg-blue-600'
                }
              `}
            >
              {saving ? (
                <span className="flex items-center gap-2">
                  <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  {isZh ? '保存中...' : 'Saving...'}
                </span>
              ) : saveSuccess ? (
                <span className="flex items-center gap-2">
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  {isZh ? '已保存' : 'Saved'}
                </span>
              ) : (
                isZh ? '保存到全站' : 'Save Site-wide'
              )}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default ThemeSettingsSection;

