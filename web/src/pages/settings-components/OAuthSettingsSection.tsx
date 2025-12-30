/**
 * OAuth Settings Section
 * Manages OAuth/SSO configuration including GitHub, Google, OIDC, and Cloudflare Access
 */

import { useState, useEffect } from 'react';
import { showToast } from '../../components/Toast';
import type { OAuthForm, OIDCProviderForm, SSOBinding } from './types';

export interface OAuthSettingsSectionProps {
  token: string | null;
  isZh: boolean;
}

export function OAuthSettingsSection({ token, isZh }: OAuthSettingsSectionProps) {
  const [showOAuthSettings, setShowOAuthSettings] = useState(false);
  const [oauthForm, setOauthForm] = useState<OAuthForm>({
    use_centralized: true,
    allowed_users: '',
    github: { enabled: false, client_id: '', client_secret: '', allowed_users: '' },
    google: { enabled: false, client_id: '', client_secret: '', allowed_users: '' },
    oidc: [],
    cloudflare_access: { enabled: false, team_domain: '', aud: '', allowed_users: '' },
  });
  const [ssoBindings, setSsoBindings] = useState<SSOBinding[]>([]);
  const [oauthSaving, setOauthSaving] = useState(false);
  const [oauthSuccess, setOauthSuccess] = useState(false);
  const [oauthTab, setOauthTab] = useState<'basic' | 'oidc' | 'cloudflare' | 'bindings'>('basic');

  useEffect(() => {
    fetchOAuthSettings();
  }, [token]);

  const fetchOAuthSettings = async () => {
    try {
      const res = await fetch('/api/settings/oauth', {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setOauthForm({
          use_centralized: data.use_centralized ?? true,
          allowed_users: data.allowed_users?.join(', ') || '',
          github: {
            enabled: data.github?.enabled || false,
            client_id: data.github?.client_id || '',
            client_secret: '',
            allowed_users: data.github?.allowed_users?.join(', ') || '',
          },
          google: {
            enabled: data.google?.enabled || false,
            client_id: data.google?.client_id || '',
            client_secret: '',
            allowed_users: data.google?.allowed_users?.join(', ') || '',
          },
          oidc: (data.oidc || []).map((p: any) => ({
            id: p.id || '',
            enabled: p.enabled || false,
            name: p.name || '',
            issuer: p.issuer || '',
            client_id: p.client_id || '',
            client_secret: '',
            scopes: p.scopes?.join(', ') || 'openid, email, profile',
            allowed_users: p.allowed_users?.join(', ') || '',
            allowed_groups: p.allowed_groups?.join(', ') || '',
            username_claim: p.username_claim || 'email',
          })),
          cloudflare_access: {
            enabled: data.cloudflare_access?.enabled || false,
            team_domain: data.cloudflare_access?.team_domain || '',
            aud: data.cloudflare_access?.aud || '',
            allowed_users: data.cloudflare_access?.allowed_users?.join(', ') || '',
          },
        });
        setSsoBindings(data.bindings || []);
      }
    } catch (e) {
      console.error('Failed to fetch OAuth settings', e);
    }
  };

  const saveOAuthSettings = async () => {
    setOauthSaving(true);
    setOauthSuccess(false);

    try {
      const payload: Record<string, any> = {
        use_centralized: oauthForm.use_centralized,
        allowed_users: oauthForm.allowed_users
          .split(',')
          .map((u) => u.trim())
          .filter((u) => u.length > 0),
      };

      if (!oauthForm.use_centralized) {
        payload.github = {
          enabled: oauthForm.github.enabled,
          client_id: oauthForm.github.client_id,
          allowed_users: oauthForm.github.allowed_users
            .split(',')
            .map((u) => u.trim())
            .filter((u) => u.length > 0),
        };
        if (oauthForm.github.client_secret) {
          payload.github.client_secret = oauthForm.github.client_secret;
        }

        payload.google = {
          enabled: oauthForm.google.enabled,
          client_id: oauthForm.google.client_id,
          allowed_users: oauthForm.google.allowed_users
            .split(',')
            .map((u) => u.trim())
            .filter((u) => u.length > 0),
        };
        if (oauthForm.google.client_secret) {
          payload.google.client_secret = oauthForm.google.client_secret;
        }
      }

      if (oauthForm.oidc.length > 0) {
        payload.oidc = oauthForm.oidc.map((p) => ({
          id: p.id,
          enabled: p.enabled,
          name: p.name,
          issuer: p.issuer,
          client_id: p.client_id,
          client_secret: p.client_secret || undefined,
          scopes: p.scopes.split(',').map((s) => s.trim()).filter((s) => s.length > 0),
          allowed_users: p.allowed_users.split(',').map((u) => u.trim()).filter((u) => u.length > 0),
          allowed_groups: p.allowed_groups.split(',').map((g) => g.trim()).filter((g) => g.length > 0),
          username_claim: p.username_claim || 'email',
        }));
      }

      if (oauthForm.cloudflare_access.team_domain || oauthForm.cloudflare_access.enabled) {
        payload.cloudflare_access = {
          enabled: oauthForm.cloudflare_access.enabled,
          team_domain: oauthForm.cloudflare_access.team_domain,
          aud: oauthForm.cloudflare_access.aud,
          allowed_users: oauthForm.cloudflare_access.allowed_users
            .split(',')
            .map((u) => u.trim())
            .filter((u) => u.length > 0),
        };
      }

      const res = await fetch('/api/settings/oauth', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        setOauthSuccess(true);
        fetchOAuthSettings();
        setTimeout(() => setOauthSuccess(false), 3000);
      }
    } catch (e) {
      console.error('Failed to save OAuth settings', e);
    }

    setOauthSaving(false);
  };

  const addOIDCProvider = () => {
    const newProvider: OIDCProviderForm = {
      id: `oidc_${oauthForm.oidc.length}`,
      enabled: false,
      name: '',
      issuer: '',
      client_id: '',
      client_secret: '',
      scopes: 'openid, email, profile',
      allowed_users: '',
      allowed_groups: '',
      username_claim: 'email',
    };
    setOauthForm({ ...oauthForm, oidc: [...oauthForm.oidc, newProvider] });
  };

  const removeOIDCProvider = (index: number) => {
    setOauthForm({ ...oauthForm, oidc: oauthForm.oidc.filter((_, i) => i !== index) });
  };

  const updateOIDCProvider = (index: number, field: keyof OIDCProviderForm, value: string | boolean) => {
    const updated = [...oauthForm.oidc];
    updated[index] = { ...updated[index], [field]: value };
    setOauthForm({ ...oauthForm, oidc: updated });
  };

  const deleteSSOBinding = async (provider: string, identifier: string) => {
    try {
      const res = await fetch(`/api/sso/bindings/${provider}?identifier=${encodeURIComponent(identifier)}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` },
      });
      if (res.ok) {
        setSsoBindings(ssoBindings.filter((b) => !(b.provider === provider && b.identifier === identifier)));
        showToast(isZh ? 'SSO 绑定已删除' : 'SSO binding deleted', 'success');
      }
    } catch (e) {
      console.error('Failed to delete SSO binding', e);
      showToast(isZh ? '删除失败' : 'Delete failed', 'error');
    }
  };

  return (
    <div className="nezha-card p-6 mb-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-bold text-white flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-orange-500"></span>
          OAuth / SSO 登录
        </h2>
        <button
          onClick={() => setShowOAuthSettings(!showOAuthSettings)}
          className="px-4 py-2 rounded-lg bg-orange-500/10 hover:bg-orange-500/20 text-orange-400 text-sm font-medium transition-colors"
        >
          {showOAuthSettings ? '收起' : '配置'}
        </button>
      </div>

      <p className="text-gray-400 text-sm mb-4">
        支持 GitHub、Google、通用 OIDC Provider 和 Cloudflare Access 登录。
      </p>

      {oauthSuccess && (
        <div className="mb-4 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm">
          OAuth 设置已保存！
        </div>
      )}

      {showOAuthSettings && (
        <div className="space-y-6">
          {/* Tabs */}
          <div className="flex gap-2 border-b border-white/10 pb-2 overflow-x-auto">
            <button
              onClick={() => setOauthTab('basic')}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors whitespace-nowrap ${
                oauthTab === 'basic'
                  ? 'bg-orange-500/20 text-orange-400'
                  : 'bg-white/5 text-gray-400 hover:bg-white/10'
              }`}
            >
              基础设置
            </button>
            <button
              onClick={() => setOauthTab('oidc')}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors whitespace-nowrap ${
                oauthTab === 'oidc'
                  ? 'bg-purple-500/20 text-purple-400'
                  : 'bg-white/5 text-gray-400 hover:bg-white/10'
              }`}
            >
              OIDC Provider {oauthForm.oidc.length > 0 && `(${oauthForm.oidc.length})`}
            </button>
            <button
              onClick={() => setOauthTab('cloudflare')}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors whitespace-nowrap ${
                oauthTab === 'cloudflare'
                  ? 'bg-blue-500/20 text-blue-400'
                  : 'bg-white/5 text-gray-400 hover:bg-white/10'
              }`}
            >
              Cloudflare Access
            </button>
            <button
              onClick={() => setOauthTab('bindings')}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors whitespace-nowrap ${
                oauthTab === 'bindings'
                  ? 'bg-emerald-500/20 text-emerald-400'
                  : 'bg-white/5 text-gray-400 hover:bg-white/10'
              }`}
            >
              SSO 绑定 {ssoBindings.length > 0 && `(${ssoBindings.length})`}
            </button>
          </div>

          {/* Basic OAuth Tab */}
          {oauthTab === 'basic' && (
            <div className="space-y-4">
              <div className="p-4 rounded-xl bg-gradient-to-r from-orange-500/10 to-transparent border border-orange-500/20">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-orange-500/20 flex items-center justify-center">
                      <svg className="w-5 h-5 text-orange-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                      </svg>
                    </div>
                    <div>
                      <h3 className="font-medium text-white">启用 OAuth 登录（托管模式）</h3>
                      <p className="text-xs text-gray-500">使用 GitHub 或 Google 账号登录（推荐）</p>
                    </div>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={oauthForm.use_centralized}
                      onChange={(e) => setOauthForm({ ...oauthForm, use_centralized: e.target.checked })}
                      className="sr-only peer"
                    />
                    <div className="w-11 h-6 bg-gray-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-500"></div>
                  </label>
                </div>

                {oauthForm.use_centralized && (
                  <div className="space-y-4 pt-4 border-t border-orange-500/10">
                    <div className="flex items-center gap-2 text-sm text-emerald-400">
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                      已启用 GitHub 和 Google 登录
                    </div>

                    <div>
                      <label className="block text-xs text-gray-500 mb-2">
                        允许的用户 <span className="text-gray-600">（GitHub 用户名或 Google 邮箱，逗号分隔，留空所有人都不能登录）</span>
                      </label>
                      <input
                        type="text"
                        value={oauthForm.allowed_users}
                        onChange={(e) => setOauthForm({ ...oauthForm, allowed_users: e.target.value })}
                        className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-orange-500/50"
                        placeholder="github_user, user@gmail.com"
                      />
                    </div>

                    <div className="text-xs text-gray-500 bg-black/20 rounded-lg p-3">
                      <p className="font-medium text-gray-400 mb-1">💡 使用说明：</p>
                      <ul className="list-disc list-inside space-y-1">
                        <li>启用后，登录页面将显示 GitHub 和 Google 登录按钮</li>
                        <li>OAuth 认证由 vstats.zsoft.cc 统一处理，无需额外配置</li>
                        <li>设置允许的用户可以限制谁能登录</li>
                      </ul>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* OIDC Provider Tab */}
          {oauthTab === 'oidc' && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="font-medium text-white">通用 OIDC Provider</h3>
                  <p className="text-xs text-gray-500">支持 Authentik、Keycloak、Okta、Auth0 等 OIDC 兼容服务</p>
                </div>
                <button
                  onClick={addOIDCProvider}
                  className="px-4 py-2 rounded-lg bg-purple-500/10 hover:bg-purple-500/20 text-purple-400 text-sm font-medium transition-colors"
                >
                  添加 Provider
                </button>
              </div>

              {oauthForm.oidc.length === 0 ? (
                <div className="text-center py-8 border border-dashed border-white/10 rounded-xl">
                  <svg className="w-12 h-12 mx-auto mb-3 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                  </svg>
                  <p className="text-gray-500 text-sm">暂无 OIDC Provider 配置</p>
                  <p className="text-gray-600 text-xs mt-1">点击上方按钮添加</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {oauthForm.oidc.map((provider, index) => (
                    <div key={index} className="p-4 rounded-xl bg-white/[0.02] border border-white/10">
                      <div className="flex items-center justify-between mb-4">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-lg bg-purple-500/20 flex items-center justify-center">
                            <span className="text-purple-400 font-bold text-sm">{index + 1}</span>
                          </div>
                          <input
                            type="text"
                            value={provider.name}
                            onChange={(e) => updateOIDCProvider(index, 'name', e.target.value)}
                            placeholder="Provider 名称（如 Authentik）"
                            className="px-2 py-1 rounded bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                          />
                          <label className="flex items-center gap-2">
                            <input
                              type="checkbox"
                              checked={provider.enabled}
                              onChange={(e) => updateOIDCProvider(index, 'enabled', e.target.checked)}
                              className="w-4 h-4 rounded border-white/10 bg-white/5 text-purple-500"
                            />
                            <span className="text-xs text-gray-400">启用</span>
                          </label>
                        </div>
                        <button
                          onClick={() => removeOIDCProvider(index)}
                          className="p-2 rounded-lg hover:bg-red-500/10 text-gray-500 hover:text-red-400 transition-colors"
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <label className="block text-xs text-gray-500 mb-1">Issuer URL</label>
                          <input
                            type="text"
                            value={provider.issuer}
                            onChange={(e) => updateOIDCProvider(index, 'issuer', e.target.value)}
                            placeholder="https://auth.example.com"
                            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                          />
                        </div>
                        <div>
                          <label className="block text-xs text-gray-500 mb-1">Client ID</label>
                          <input
                            type="text"
                            value={provider.client_id}
                            onChange={(e) => updateOIDCProvider(index, 'client_id', e.target.value)}
                            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                          />
                        </div>
                        <div>
                          <label className="block text-xs text-gray-500 mb-1">Client Secret</label>
                          <input
                            type="password"
                            value={provider.client_secret}
                            onChange={(e) => updateOIDCProvider(index, 'client_secret', e.target.value)}
                            placeholder="留空则保持不变"
                            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                          />
                        </div>
                        <div>
                          <label className="block text-xs text-gray-500 mb-1">Scopes</label>
                          <input
                            type="text"
                            value={provider.scopes}
                            onChange={(e) => updateOIDCProvider(index, 'scopes', e.target.value)}
                            placeholder="openid, email, profile"
                            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                          />
                        </div>
                        <div>
                          <label className="block text-xs text-gray-500 mb-1">允许的用户（邮箱/用户名，逗号分隔）</label>
                          <input
                            type="text"
                            value={provider.allowed_users}
                            onChange={(e) => updateOIDCProvider(index, 'allowed_users', e.target.value)}
                            placeholder="user@example.com, admin"
                            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                          />
                        </div>
                        <div>
                          <label className="block text-xs text-gray-500 mb-1">允许的用户组（逗号分隔）</label>
                          <input
                            type="text"
                            value={provider.allowed_groups}
                            onChange={(e) => updateOIDCProvider(index, 'allowed_groups', e.target.value)}
                            placeholder="admins, operators"
                            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                          />
                        </div>
                        <div>
                          <label className="block text-xs text-gray-500 mb-1">用户名 Claim</label>
                          <select
                            value={provider.username_claim}
                            onChange={(e) => updateOIDCProvider(index, 'username_claim', e.target.value)}
                            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                          >
                            <option value="email">email</option>
                            <option value="preferred_username">preferred_username</option>
                            <option value="sub">sub</option>
                            <option value="name">name</option>
                          </select>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              <div className="text-xs text-gray-500 bg-black/20 rounded-lg p-3">
                <p className="font-medium text-gray-400 mb-1">💡 OIDC 配置说明：</p>
                <ul className="list-disc list-inside space-y-1">
                  <li>Issuer URL 是 OIDC Provider 的基础 URL，系统会自动获取 .well-known/openid-configuration</li>
                  <li>回调地址格式：<code className="text-purple-400">https://your-domain/api/auth/oauth/oidc/oidc_0/callback</code></li>
                  <li>支持基于用户邮箱或用户组的访问控制</li>
                </ul>
              </div>
            </div>
          )}

          {/* Cloudflare Access Tab */}
          {oauthTab === 'cloudflare' && (
            <div className="space-y-4">
              <div className="p-4 rounded-xl bg-gradient-to-r from-blue-500/10 to-transparent border border-blue-500/20">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-blue-500/20 flex items-center justify-center">
                      <svg className="w-5 h-5 text-blue-400" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M16.5 8.25a.75.75 0 0 0-.75-.75h-6a.75.75 0 0 0 0 1.5h4.19l-6.72 6.72a.75.75 0 1 0 1.06 1.06l6.72-6.72v4.19a.75.75 0 0 0 1.5 0v-6Z" />
                        <path d="M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25Z" fillOpacity=".2" />
                      </svg>
                    </div>
                    <div>
                      <h3 className="font-medium text-white">Cloudflare Access</h3>
                      <p className="text-xs text-gray-500">使用 Cloudflare Zero Trust 进行身份验证</p>
                    </div>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={oauthForm.cloudflare_access.enabled}
                      onChange={(e) =>
                        setOauthForm({
                          ...oauthForm,
                          cloudflare_access: { ...oauthForm.cloudflare_access, enabled: e.target.checked },
                        })
                      }
                      className="sr-only peer"
                    />
                    <div className="w-11 h-6 bg-gray-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-500"></div>
                  </label>
                </div>

                {oauthForm.cloudflare_access.enabled && (
                  <div className="space-y-4 pt-4 border-t border-blue-500/10">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs text-gray-500 mb-1">Team Domain</label>
                        <input
                          type="text"
                          value={oauthForm.cloudflare_access.team_domain}
                          onChange={(e) =>
                            setOauthForm({
                              ...oauthForm,
                              cloudflare_access: { ...oauthForm.cloudflare_access, team_domain: e.target.value },
                            })
                          }
                          placeholder="mycompany.cloudflareaccess.com"
                          className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-gray-500 mb-1">Application AUD</label>
                        <input
                          type="text"
                          value={oauthForm.cloudflare_access.aud}
                          onChange={(e) =>
                            setOauthForm({
                              ...oauthForm,
                              cloudflare_access: { ...oauthForm.cloudflare_access, aud: e.target.value },
                            })
                          }
                          placeholder="从 Cloudflare Access 控制台获取"
                          className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
                        />
                      </div>
                      <div className="col-span-2">
                        <label className="block text-xs text-gray-500 mb-1">允许的用户（邮箱，逗号分隔，留空则使用 Access 策略）</label>
                        <input
                          type="text"
                          value={oauthForm.cloudflare_access.allowed_users}
                          onChange={(e) =>
                            setOauthForm({
                              ...oauthForm,
                              cloudflare_access: { ...oauthForm.cloudflare_access, allowed_users: e.target.value },
                            })
                          }
                          placeholder="admin@example.com, user@example.com"
                          className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
                        />
                      </div>
                    </div>
                  </div>
                )}
              </div>

              <div className="text-xs text-gray-500 bg-black/20 rounded-lg p-3">
                <p className="font-medium text-gray-400 mb-1">💡 Cloudflare Access 配置说明：</p>
                <ul className="list-disc list-inside space-y-1">
                  <li>在 Cloudflare Zero Trust 控制台创建 Self-hosted 应用</li>
                  <li>Application Domain 设置为你的 vStats 域名</li>
                  <li>AUD 值可在应用详情页的 Overview 标签中找到</li>
                  <li>确保 Access 策略允许目标用户访问</li>
                </ul>
              </div>
            </div>
          )}

          {/* SSO Bindings Tab */}
          {oauthTab === 'bindings' && (
            <div className="space-y-4">
              <div>
                <h3 className="font-medium text-white mb-2">已绑定的 SSO 账号</h3>
                <p className="text-xs text-gray-500 mb-4">这些 SSO 账号已绑定到管理员账户，可用于登录</p>
              </div>

              {ssoBindings.length === 0 ? (
                <div className="text-center py-8 border border-dashed border-white/10 rounded-xl">
                  <svg className="w-12 h-12 mx-auto mb-3 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                  </svg>
                  <p className="text-gray-500 text-sm">暂无 SSO 绑定</p>
                  <p className="text-gray-600 text-xs mt-1">通过 OAuth 登录后自动绑定</p>
                </div>
              ) : (
                <div className="space-y-2">
                  {ssoBindings.map((binding, index) => (
                    <div key={index} className="flex items-center justify-between p-3 rounded-lg bg-white/[0.02] border border-white/10">
                      <div className="flex items-center gap-3">
                        <div
                          className={`w-8 h-8 rounded-lg flex items-center justify-center ${
                            binding.provider === 'github'
                              ? 'bg-gray-800'
                              : binding.provider === 'google'
                              ? 'bg-white'
                              : binding.provider.startsWith('oidc')
                              ? 'bg-purple-500/20'
                              : binding.provider === 'cloudflare'
                              ? 'bg-blue-500/20'
                              : 'bg-gray-500/20'
                          }`}
                        >
                          {binding.provider === 'github' && (
                            <svg className="w-4 h-4 text-white" viewBox="0 0 24 24" fill="currentColor">
                              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
                            </svg>
                          )}
                          {binding.provider === 'google' && (
                            <svg className="w-4 h-4" viewBox="0 0 24 24">
                              <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
                              <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
                              <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
                              <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
                            </svg>
                          )}
                          {binding.provider.startsWith('oidc') && (
                            <svg className="w-4 h-4 text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                            </svg>
                          )}
                          {binding.provider === 'cloudflare' && (
                            <svg className="w-4 h-4 text-blue-400" viewBox="0 0 24 24" fill="currentColor">
                              <path d="M16.5 8.25a.75.75 0 0 0-.75-.75h-6a.75.75 0 0 0 0 1.5h4.19l-6.72 6.72a.75.75 0 1 0 1.06 1.06l6.72-6.72v4.19a.75.75 0 0 0 1.5 0v-6Z" />
                            </svg>
                          )}
                        </div>
                        <div>
                          <p className="text-sm text-white">{binding.identifier}</p>
                          <p className="text-xs text-gray-500">
                            {binding.provider === 'github'
                              ? 'GitHub'
                              : binding.provider === 'google'
                              ? 'Google'
                              : binding.provider.startsWith('oidc')
                              ? 'OIDC'
                              : binding.provider === 'cloudflare'
                              ? 'Cloudflare Access'
                              : binding.provider}
                            {binding.bound_at && ` · 绑定于 ${new Date(binding.bound_at).toLocaleDateString()}`}
                          </p>
                        </div>
                      </div>
                      <button
                        onClick={() => deleteSSOBinding(binding.provider, binding.identifier)}
                        className="p-2 rounded-lg hover:bg-red-500/10 text-gray-500 hover:text-red-400 transition-colors"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  ))}
                </div>
              )}

              <div className="text-xs text-gray-500 bg-black/20 rounded-lg p-3">
                <p className="font-medium text-gray-400 mb-1">💡 SSO 绑定说明：</p>
                <ul className="list-disc list-inside space-y-1">
                  <li>绑定的 SSO 账号可以直接登录管理后台</li>
                  <li>删除绑定后该 SSO 账号将无法登录（除非在允许用户列表中）</li>
                  <li>可以绑定多个不同 Provider 的账号</li>
                </ul>
              </div>
            </div>
          )}

          <div className="flex justify-end">
            <button
              onClick={saveOAuthSettings}
              disabled={oauthSaving}
              className="px-4 py-2 rounded-lg bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium transition-colors disabled:opacity-50"
            >
              {oauthSaving ? '保存中...' : '保存设置'}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

