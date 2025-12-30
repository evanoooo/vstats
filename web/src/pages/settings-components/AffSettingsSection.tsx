/**
 * Affiliate Provider Settings Section
 * Manage affiliate links for VPS providers
 */

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import type { AffProvider } from '../../types';

interface AffSettingsSectionProps {
  token: string | null;
  isZh: boolean;
}

export function AffSettingsSection({ token, isZh }: AffSettingsSectionProps) {
  const { t } = useTranslation();
  const [providers, setProviders] = useState<AffProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [showSettings, setShowSettings] = useState(false);

  useEffect(() => {
    fetchProviders();
  }, [token]);

  const fetchProviders = async () => {
    try {
      const res = await fetch('/api/settings/aff-providers', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setProviders(data || []);
      }
    } catch (e) {
      console.error('Failed to fetch aff providers', e);
    } finally {
      setLoading(false);
    }
  };

  const saveProviders = async () => {
    setSaving(true);
    try {
      const res = await fetch('/api/settings/aff-providers', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(providers),
      });
      if (res.ok) {
        setMessage({ type: 'success', text: isZh ? '保存成功' : 'Saved successfully' });
        setTimeout(() => setMessage(null), 3000);
      } else {
        setMessage({ type: 'error', text: isZh ? '保存失败' : 'Failed to save' });
      }
    } catch (e) {
      console.error('Failed to save aff providers', e);
      setMessage({ type: 'error', text: isZh ? '保存失败' : 'Failed to save' });
    } finally {
      setSaving(false);
    }
  };

  const addProvider = () => {
    const id = `aff_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    setProviders([...providers, { id, name: '', aff_link: '', logo_url: '', enabled: true }]);
  };

  const updateProvider = (index: number, field: keyof AffProvider, value: string | boolean) => {
    const updated = [...providers];
    updated[index] = { ...updated[index], [field]: value };
    setProviders(updated);
  };

  const removeProvider = (index: number) => {
    setProviders(providers.filter((_, i) => i !== index));
  };

  if (loading) {
    return (
      <div className="nezha-card p-6">
        <div className="animate-pulse space-y-4">
          <div className="h-6 bg-gray-700/50 rounded w-1/3" />
          <div className="h-4 bg-gray-700/50 rounded w-2/3" />
          <div className="h-20 bg-gray-700/50 rounded" />
        </div>
      </div>
    );
  }

  return (
    <div className="nezha-card p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold text-white flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-amber-500" />
            {isZh ? '推广链接设置' : 'Affiliate Links'}
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            {isZh 
              ? '设置 VPS 提供商的推广链接，在 Dashboard 上显示购买按钮'
              : 'Set affiliate links for VPS providers to show buy buttons on Dashboard'}
          </p>
        </div>
        <button
          onClick={() => setShowSettings(!showSettings)}
          className="px-4 py-2 rounded-lg bg-amber-500/10 hover:bg-amber-500/20 text-amber-400 text-sm font-medium transition-colors"
        >
          {showSettings ? t('common.close') : t('common.edit')}
        </button>
      </div>

      {/* Message */}
      {message && (
        <div
          className={`mb-4 p-3 rounded-lg text-sm ${
            message.type === 'success'
              ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
              : 'bg-red-500/20 text-red-400 border border-red-500/30'
          }`}
        >
          {message.text}
        </div>
      )}

      {showSettings && (
        <div className="space-y-4">
          {/* Add button */}
          <div className="flex items-center justify-between">
            <label className="text-xs text-gray-500 uppercase tracking-wider">
              {isZh ? '推广链接列表' : 'Affiliate Links'}
            </label>
            <button
              type="button"
              onClick={addProvider}
              className="text-xs text-amber-400 hover:text-amber-300 transition-colors"
            >
              + {isZh ? '添加提供商' : 'Add Provider'}
            </button>
          </div>

          {/* Provider List */}
          {providers.length === 0 ? (
            <div className="text-center py-8 border border-dashed border-white/10 rounded-xl">
              <svg className="w-12 h-12 mx-auto mb-3 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
              </svg>
              <p className="text-gray-500 text-sm">{isZh ? '暂无推广链接' : 'No affiliate links yet'}</p>
              <p className="text-gray-600 text-xs mt-1">{isZh ? '点击上方按钮添加' : 'Click the button above to add one'}</p>
            </div>
          ) : (
            <div className="space-y-4">
              {providers.map((provider, index) => (
                <div key={provider.id} className="p-4 rounded-xl bg-white/[0.02] border border-white/10">
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-lg bg-amber-500/20 flex items-center justify-center">
                        <span className="text-amber-400 font-bold text-sm">{index + 1}</span>
                      </div>
                      <input
                        type="text"
                        value={provider.name}
                        onChange={(e) => updateProvider(index, 'name', e.target.value)}
                        placeholder={isZh ? '提供商名称（如 Vultr）' : 'Provider name (e.g. Vultr)'}
                        className="px-2 py-1 rounded bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-amber-500/50"
                      />
                      <label className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={provider.enabled}
                          onChange={(e) => updateProvider(index, 'enabled', e.target.checked)}
                          className="w-4 h-4 rounded border-white/10 bg-white/5 text-amber-500"
                        />
                        <span className="text-xs text-gray-400">{isZh ? '启用' : 'Enable'}</span>
                      </label>
                    </div>
                    <button
                      onClick={() => removeProvider(index)}
                      className="p-2 rounded-lg hover:bg-red-500/10 text-gray-500 hover:text-red-400 transition-colors"
                    >
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="col-span-2">
                      <label className="block text-xs text-gray-500 mb-1">
                        {isZh ? '推广链接' : 'Affiliate Link'} *
                      </label>
                      <input
                        type="url"
                        value={provider.aff_link}
                        onChange={(e) => updateProvider(index, 'aff_link', e.target.value)}
                        placeholder="https://..."
                        className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-amber-500/50 font-mono"
                      />
                    </div>
                    <div className="col-span-2">
                      <label className="block text-xs text-gray-500 mb-1">
                        {isZh ? '自定义 Logo URL' : 'Custom Logo URL'}
                      </label>
                      <input
                        type="url"
                        value={provider.logo_url || ''}
                        onChange={(e) => updateProvider(index, 'logo_url', e.target.value)}
                        placeholder={isZh ? '留空使用默认 Logo' : 'Leave empty for default logo'}
                        className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-amber-500/50"
                      />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          <div className="pt-4 border-t border-white/5 text-xs text-gray-500">
            <p>💡 {isZh ? '提供商名称需要与服务器的 Provider 字段匹配才能显示购买按钮' : 'Provider name should match the server\'s Provider field to show the buy button'}</p>
          </div>

          <div className="flex justify-end pt-4">
            <button
              onClick={saveProviders}
              disabled={saving}
              className="px-4 py-2 rounded-lg bg-amber-500 hover:bg-amber-600 text-white text-sm font-medium transition-colors disabled:opacity-50"
            >
              {saving ? (isZh ? '保存中...' : 'Saving...') : (isZh ? '保存设置' : 'Save Settings')}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

