/**
 * Theme Installer Component
 * 
 * UI for installing themes from GitHub repositories.
 */

import { useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { useTranslation } from 'react-i18next';

interface ThemeInstallerProps {
  onClose?: () => void;
}

export function ThemeInstaller({ onClose }: ThemeInstallerProps) {
  const { i18n } = useTranslation();
  const { installTheme, uninstallTheme, installedThemes, themeId, setTheme } = useTheme();
  const [source, setSource] = useState('');
  const [ref, setRef] = useState('');
  const [isInstalling, setIsInstalling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const isZh = i18n.language === 'zh';

  const handleInstall = async () => {
    if (!source.trim()) {
      setError(isZh ? '请输入主题来源' : 'Please enter theme source');
      return;
    }

    setIsInstalling(true);
    setError(null);
    setSuccess(null);

    try {
      const result = await installTheme(source.trim(), ref.trim() || undefined);
      
      if (result.success) {
        setSuccess(isZh 
          ? `主题 "${result.theme?.manifest.name}" 安装成功！` 
          : `Theme "${result.theme?.manifest.name}" installed successfully!`
        );
        setSource('');
        setRef('');
      } else {
        setError(result.error || (isZh ? '安装失败' : 'Installation failed'));
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : (isZh ? '未知错误' : 'Unknown error'));
    } finally {
      setIsInstalling(false);
    }
  };

  const handleUninstall = async (id: string, name: string) => {
    if (!confirm(isZh ? `确定要卸载主题 "${name}" 吗？` : `Are you sure you want to uninstall "${name}"?`)) {
      return;
    }

    const success = await uninstallTheme(id);
    if (success) {
      setSuccess(isZh ? `主题 "${name}" 已卸载` : `Theme "${name}" uninstalled`);
    } else {
      setError(isZh ? '卸载失败' : 'Uninstall failed');
    }
  };

  return (
    <div className="theme-installer">
      {/* Header */}
      <div className="theme-installer-header">
        <h2>{isZh ? '主题管理' : 'Theme Manager'}</h2>
        {onClose && (
          <button onClick={onClose} className="theme-installer-close">
            ✕
          </button>
        )}
      </div>

      {/* Install Section */}
      <div className="theme-installer-section">
        <h3>{isZh ? '安装主题' : 'Install Theme'}</h3>
        <p className="theme-installer-hint">
          {isZh 
            ? '从 GitHub 安装主题，支持以下格式：' 
            : 'Install themes from GitHub using these formats:'
          }
        </p>
        <ul className="theme-installer-formats">
          <li><code>user/repo</code> - {isZh ? '仓库根目录' : 'Repository root'}</li>
          <li><code>user/repo/themes/my-theme</code> - {isZh ? '子目录' : 'Subdirectory'}</li>
          <li><code>user/repo@v1.0.0</code> - {isZh ? '指定版本/分支' : 'Specific version/branch'}</li>
          <li><code>https://example.com/theme.json</code> - {isZh ? '直接 URL' : 'Direct URL'}</li>
        </ul>

        <div className="theme-installer-form">
          <div className="theme-installer-input-group">
            <label>{isZh ? '主题来源' : 'Theme Source'}</label>
            <input
              type="text"
              value={source}
              onChange={(e) => setSource(e.target.value)}
              placeholder="user/repo 或 https://..."
              disabled={isInstalling}
            />
          </div>

          <div className="theme-installer-input-group">
            <label>{isZh ? '版本/分支 (可选)' : 'Version/Branch (optional)'}</label>
            <input
              type="text"
              value={ref}
              onChange={(e) => setRef(e.target.value)}
              placeholder="main, v1.0.0, ..."
              disabled={isInstalling}
            />
          </div>

          <button
            onClick={handleInstall}
            disabled={isInstalling || !source.trim()}
            className="theme-installer-btn"
          >
            {isInstalling ? (isZh ? '安装中...' : 'Installing...') : (isZh ? '安装主题' : 'Install Theme')}
          </button>
        </div>

        {error && <div className="theme-installer-error">{error}</div>}
        {success && <div className="theme-installer-success">{success}</div>}
      </div>

      {/* Installed Themes Section */}
      {installedThemes.length > 0 && (
        <div className="theme-installer-section">
          <h3>{isZh ? '已安装的主题' : 'Installed Themes'}</h3>
          <div className="theme-installer-list">
            {installedThemes.map((theme) => (
              <div key={theme.manifest.id} className="theme-installer-item">
                <div className="theme-installer-item-preview">
                  <div 
                    className="theme-preview-swatch"
                    style={{
                      background: `linear-gradient(135deg, ${theme.manifest.preview.primary} 0%, ${theme.manifest.preview.secondary} 50%, ${theme.manifest.preview.accent} 100%)`
                    }}
                  />
                </div>
                <div className="theme-installer-item-info">
                  <div className="theme-installer-item-name">
                    {isZh && theme.manifest.nameZh ? theme.manifest.nameZh : theme.manifest.name}
                  </div>
                  <div className="theme-installer-item-meta">
                    v{theme.manifest.version} • {theme.manifest.author}
                  </div>
                  <div className="theme-installer-item-desc">
                    {isZh && theme.manifest.descriptionZh 
                      ? theme.manifest.descriptionZh 
                      : theme.manifest.description
                    }
                  </div>
                  <div className="theme-installer-item-source">
                    <span className="theme-source-badge">{theme.source.type}</span>
                    <span className="theme-source-location">{theme.source.location}</span>
                  </div>
                </div>
                <div className="theme-installer-item-actions">
                  {themeId === theme.manifest.id ? (
                    <span className="theme-active-badge">{isZh ? '当前使用' : 'Active'}</span>
                  ) : (
                    <button
                      onClick={() => setTheme(theme.manifest.id)}
                      className="theme-installer-btn-small"
                    >
                      {isZh ? '使用' : 'Use'}
                    </button>
                  )}
                  <button
                    onClick={() => handleUninstall(theme.manifest.id, theme.manifest.name)}
                    className="theme-installer-btn-small theme-installer-btn-danger"
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
      <div className="theme-installer-section theme-installer-footer">
        <p>
          {isZh ? '想要创建自己的主题？' : 'Want to create your own theme?'}
          <a 
            href="https://github.com/user/vstats/blob/main/docs/THEME-DEVELOPMENT.md" 
            target="_blank" 
            rel="noopener noreferrer"
          >
            {isZh ? '查看主题开发指南' : 'View Theme Development Guide'}
          </a>
        </p>
      </div>

      <style>{`
        .theme-installer {
          padding: 1.5rem;
          max-width: 600px;
          margin: 0 auto;
        }

        .theme-installer-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 1.5rem;
          padding-bottom: 1rem;
          border-bottom: 1px solid var(--border-primary, rgba(255,255,255,0.1));
        }

        .theme-installer-header h2 {
          margin: 0;
          font-size: 1.5rem;
          font-weight: 600;
        }

        .theme-installer-close {
          background: none;
          border: none;
          font-size: 1.25rem;
          cursor: pointer;
          opacity: 0.6;
          padding: 0.5rem;
        }

        .theme-installer-close:hover {
          opacity: 1;
        }

        .theme-installer-section {
          margin-bottom: 2rem;
        }

        .theme-installer-section h3 {
          font-size: 1.1rem;
          font-weight: 600;
          margin-bottom: 0.75rem;
        }

        .theme-installer-hint {
          font-size: 0.9rem;
          opacity: 0.8;
          margin-bottom: 0.5rem;
        }

        .theme-installer-formats {
          font-size: 0.85rem;
          margin: 0 0 1rem 1.5rem;
          opacity: 0.7;
        }

        .theme-installer-formats code {
          background: var(--bg-input, rgba(255,255,255,0.1));
          padding: 0.15rem 0.4rem;
          border-radius: 4px;
          font-family: var(--theme-font-mono);
        }

        .theme-installer-form {
          display: flex;
          flex-direction: column;
          gap: 1rem;
        }

        .theme-installer-input-group {
          display: flex;
          flex-direction: column;
          gap: 0.4rem;
        }

        .theme-installer-input-group label {
          font-size: 0.85rem;
          font-weight: 500;
        }

        .theme-installer-input-group input {
          padding: 0.75rem 1rem;
          border-radius: var(--theme-border-radius, 8px);
          border: 1px solid var(--border-primary, rgba(255,255,255,0.2));
          background: var(--bg-input, rgba(255,255,255,0.05));
          font-size: 0.95rem;
        }

        .theme-installer-input-group input:focus {
          outline: none;
          border-color: var(--theme-accent, #3b82f6);
        }

        .theme-installer-btn {
          padding: 0.75rem 1.5rem;
          border-radius: var(--theme-border-radius, 8px);
          border: none;
          background: var(--theme-accent, #3b82f6);
          color: white;
          font-weight: 500;
          cursor: pointer;
          transition: opacity 0.2s;
        }

        .theme-installer-btn:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .theme-installer-btn:hover:not(:disabled) {
          opacity: 0.9;
        }

        .theme-installer-error {
          margin-top: 1rem;
          padding: 0.75rem 1rem;
          background: rgba(239, 68, 68, 0.15);
          border: 1px solid rgba(239, 68, 68, 0.3);
          border-radius: 8px;
          color: #f87171;
          font-size: 0.9rem;
        }

        .theme-installer-success {
          margin-top: 1rem;
          padding: 0.75rem 1rem;
          background: rgba(16, 185, 129, 0.15);
          border: 1px solid rgba(16, 185, 129, 0.3);
          border-radius: 8px;
          color: #34d399;
          font-size: 0.9rem;
        }

        .theme-installer-list {
          display: flex;
          flex-direction: column;
          gap: 1rem;
        }

        .theme-installer-item {
          display: flex;
          gap: 1rem;
          padding: 1rem;
          border-radius: var(--theme-border-radius, 12px);
          background: var(--bg-card, rgba(255,255,255,0.05));
          border: 1px solid var(--border-primary, rgba(255,255,255,0.1));
        }

        .theme-installer-item-preview {
          flex-shrink: 0;
        }

        .theme-preview-swatch {
          width: 60px;
          height: 60px;
          border-radius: 8px;
          border: 1px solid rgba(255,255,255,0.1);
        }

        .theme-installer-item-info {
          flex: 1;
          min-width: 0;
        }

        .theme-installer-item-name {
          font-weight: 600;
          font-size: 1rem;
          margin-bottom: 0.25rem;
        }

        .theme-installer-item-meta {
          font-size: 0.8rem;
          opacity: 0.6;
          margin-bottom: 0.35rem;
        }

        .theme-installer-item-desc {
          font-size: 0.85rem;
          opacity: 0.8;
          margin-bottom: 0.35rem;
        }

        .theme-installer-item-source {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          font-size: 0.75rem;
        }

        .theme-source-badge {
          background: var(--theme-accent, #3b82f6);
          color: white;
          padding: 0.15rem 0.4rem;
          border-radius: 4px;
          text-transform: uppercase;
          font-weight: 500;
        }

        .theme-source-location {
          opacity: 0.6;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .theme-installer-item-actions {
          display: flex;
          flex-direction: column;
          gap: 0.5rem;
          flex-shrink: 0;
        }

        .theme-installer-btn-small {
          padding: 0.4rem 0.75rem;
          border-radius: 6px;
          border: 1px solid var(--border-primary, rgba(255,255,255,0.2));
          background: transparent;
          font-size: 0.8rem;
          cursor: pointer;
          transition: all 0.2s;
        }

        .theme-installer-btn-small:hover {
          background: var(--bg-secondary-hover, rgba(255,255,255,0.1));
        }

        .theme-installer-btn-danger:hover {
          background: rgba(239, 68, 68, 0.15);
          border-color: rgba(239, 68, 68, 0.3);
          color: #f87171;
        }

        .theme-active-badge {
          padding: 0.4rem 0.75rem;
          border-radius: 6px;
          background: rgba(16, 185, 129, 0.15);
          border: 1px solid rgba(16, 185, 129, 0.3);
          color: #34d399;
          font-size: 0.8rem;
          text-align: center;
        }

        .theme-installer-footer {
          padding-top: 1rem;
          border-top: 1px solid var(--border-primary, rgba(255,255,255,0.1));
          text-align: center;
        }

        .theme-installer-footer p {
          font-size: 0.9rem;
          opacity: 0.7;
        }

        .theme-installer-footer a {
          color: var(--theme-accent, #3b82f6);
          text-decoration: none;
          margin-left: 0.5rem;
        }

        .theme-installer-footer a:hover {
          text-decoration: underline;
        }
      `}</style>
    </div>
  );
}

export default ThemeInstaller;

