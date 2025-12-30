/**
 * Server Management Section
 * Manages servers, agents, and quick install commands
 */

import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { showToast } from '../../components/Toast';
import type { GroupDimension } from '../../types';
import type { RemoteServer } from './types';
import type { EditForm, ServerManagementSectionProps } from './serverManagementTypes';
import { DEFAULT_EDIT_FORM } from './serverManagementTypes';
import { copyTextToClipboard } from './utils';
import { BatchUpdateHeader } from './BatchUpdateHeader';
import { ServerCard } from './ServerCard';

export function ServerManagementSection({ token, isZh }: ServerManagementSectionProps) {
  const { t } = useTranslation();

  // State
  const [servers, setServers] = useState<RemoteServer[]>([]);
  const [dimensions, setDimensions] = useState<GroupDimension[]>([]);
  const [agentStatus, setAgentStatus] = useState<Record<string, boolean>>({});
  const [updatingAgents, setUpdatingAgents] = useState<Record<string, boolean>>({});

  // Batch update
  const [batchUpdating, setBatchUpdating] = useState(false);
  const [batchUpdateProgress, setBatchUpdateProgress] = useState<{ current: number; total: number; currentServer: string }>({ current: 0, total: 0, currentServer: '' });

  // Forms
  const [showAddForm, setShowAddForm] = useState(false);
  const [newServer, setNewServer] = useState({ name: '', url: '', location: '', provider: '', tag: '', group_values: {} as Record<string, string> });
  const [addLoading, setAddLoading] = useState(false);

  // Install command
  const [showInstallCommand, setShowInstallCommand] = useState(false);
  const [installCommand, setInstallCommand] = useState('');
  const [windowsInstallCommand, setWindowsInstallCommand] = useState('');
  const [copied, setCopied] = useState(false);
  const [installPlatform, setInstallPlatform] = useState<'linux' | 'windows'>('linux');

  // Edit server
  const [editingServer, setEditingServer] = useState<string | null>(null);
  const [editForm, setEditForm] = useState<EditForm>(DEFAULT_EDIT_FORM);
  const [editLoading, setEditLoading] = useState(false);
  const [editSuccess, setEditSuccess] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);

  // Copy token
  const [copiedToken, setCopiedToken] = useState<string | null>(null);

  useEffect(() => {
    fetchServers();
    fetchDimensions();
    generateInstallCommand();
    fetchAgentStatus();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  useEffect(() => {
    const interval = setInterval(fetchAgentStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  const fetchServers = async () => {
    try {
      const res = await fetch('/api/servers');
      if (res.ok) {
        const data = await res.json();
        setServers(data);
      }
    } catch (e) {
      console.error('Failed to fetch servers', e);
    }
  };

  const fetchDimensions = async () => {
    try {
      const res = await fetch('/api/dimensions', {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setDimensions(data || []);
      }
    } catch (e) {
      console.error('Failed to fetch dimensions', e);
    }
  };

  const fetchAgentStatus = async () => {
    try {
      const res = await fetch('/api/metrics/all');
      if (res.ok) {
        const data = await res.json();
        const status: Record<string, boolean> = {};
        if (Array.isArray(data)) {
          data.forEach((s: { server_id: string; online: boolean }) => {
            status[s.server_id] = s.online;
          });
          setAgentStatus(status);
        }
      }
    } catch (e) {
      console.error('Failed to fetch agent status', e);
    }
  };

  const generateInstallCommand = () => {
    const host = window.location.host;
    const protocol = window.location.protocol === 'https:' ? 'https' : 'http';
    const baseUrl = `${protocol}://${host}`;
    const scriptUrl = 'https://vstats.zsoft.cc';

    const linuxCommand = `curl -fsSL ${scriptUrl}/agent.sh | sudo bash -s -- \\
  --server ${baseUrl} \\
  --token "${token}" \\
  --name "$(hostname)"`;

    const windowsCommand = `[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; iex (irm ${scriptUrl}/agent.ps1); Install-VStatsAgent -Server "${baseUrl}" -Token "${token}"`;

    setInstallCommand(linuxCommand);
    setWindowsInstallCommand(windowsCommand);
  };

  const copyToClipboard = useCallback(async () => {
    const commandToCopy = installPlatform === 'windows' ? windowsInstallCommand : installCommand;
    const success = await copyTextToClipboard(commandToCopy);
    if (success) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [installCommand, windowsInstallCommand, installPlatform]);

  const copyToken = useCallback(async (token: string) => {
    const success = await copyTextToClipboard(token);
    if (success) {
      setCopiedToken(token);
      setTimeout(() => setCopiedToken(null), 2000);
    }
  }, []);

  // Auto-sync tag to group_values
  const syncTagToGroupValues = async (tag: string): Promise<Record<string, string>> => {
    if (!tag || !tag.trim()) return {};

    const tagValue = tag.trim();
    const tagDimensionKey = 'tag';

    let tagDimension = dimensions.find((d) => d.key === tagDimensionKey);

    if (!tagDimension) {
      try {
        const res = await fetch('/api/dimensions', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
          body: JSON.stringify({
            name: '标签',
            key: tagDimensionKey,
            enabled: true,
            sort_order: dimensions.length,
          }),
        });

        if (res.ok) {
          const newDimension = await res.json();
          tagDimension = newDimension;
          setDimensions((prev) => [...prev, newDimension]);
        } else {
          return {};
        }
      } catch (e) {
        console.error('Failed to create tag dimension', e);
        return {};
      }
    }

    if (!tagDimension) return {};

    let tagOption = tagDimension.options.find((o) => o.name === tagValue);

    if (!tagOption) {
      try {
        const res = await fetch(`/api/dimensions/${tagDimension.id}/options`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
          body: JSON.stringify({
            name: tagValue,
            sort_order: tagDimension.options.length,
          }),
        });

        if (res.ok) {
          const newOption = await res.json();
          tagOption = newOption;
          setDimensions((prev) =>
            prev.map((d) => (d.id === tagDimension!.id ? { ...d, options: [...d.options, newOption] } : d))
          );
        } else {
          return {};
        }
      } catch (e) {
        console.error('Failed to create tag option', e);
        return {};
      }
    }

    if (!tagOption) return {};

    return { [tagDimension.id]: tagOption.id };
  };

  const updateAgent = async (serverId: string, force: boolean = false) => {
    setUpdatingAgents((prev) => ({ ...prev, [serverId]: true }));

    try {
      const res = await fetch(`/api/servers/${serverId}/update`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ force }),
      });

      if (res.ok) {
        const data = await res.json();
        if (data.success) {
          showToast(force ? '强制升级命令已发送' : t('settings.updateSent'), 'success');
        } else {
          showToast(`${t('settings.updateFailed')}: ${data.message}`, 'error');
        }
      } else {
        showToast(t('settings.updateFailed'), 'error');
      }
    } catch (e) {
      console.error('Failed to update agent', e);
      showToast(t('settings.updateFailed'), 'error');
    }

    setUpdatingAgents((prev) => ({ ...prev, [serverId]: false }));
  };

  const updateAllAgents = async (force: boolean = false) => {
    const onlineServers = servers.filter((server) => agentStatus[server.id]);

    if (onlineServers.length === 0) {
      showToast('没有在线的 Agent 可以升级', 'error');
      return;
    }

    setBatchUpdating(true);
    setBatchUpdateProgress({ current: 0, total: onlineServers.length, currentServer: '' });

    let successCount = 0;
    let failCount = 0;

    for (let i = 0; i < onlineServers.length; i++) {
      const server = onlineServers[i];
      setBatchUpdateProgress({ current: i + 1, total: onlineServers.length, currentServer: server.name });
      setUpdatingAgents((prev) => ({ ...prev, [server.id]: true }));

      try {
        const res = await fetch(`/api/servers/${server.id}/update`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
          body: JSON.stringify({ force }),
        });

        if (res.ok) {
          const data = await res.json();
          if (data.success) {
            successCount++;
          } else {
            failCount++;
          }
        } else {
          failCount++;
        }
      } catch (e) {
        console.error(`Failed to update agent ${server.name}`, e);
        failCount++;
      }

      setUpdatingAgents((prev) => ({ ...prev, [server.id]: false }));

      if (i < onlineServers.length - 1) {
        await new Promise((resolve) => setTimeout(resolve, 2000));
      }
    }

    setBatchUpdating(false);
    setBatchUpdateProgress({ current: 0, total: 0, currentServer: '' });

    if (failCount === 0) {
      showToast(`成功发送升级命令给 ${successCount} 个 Agent`, 'success');
    } else {
      showToast(`升级完成：${successCount} 成功，${failCount} 失败`, failCount > 0 ? 'error' : 'success');
    }
  };

  const startEditServer = (server: RemoteServer) => {
    setEditingServer(server.id);
    setEditForm({
      name: server.name,
      location: server.location,
      provider: server.provider,
      tag: server.tag || '',
      price_amount: server.price_amount || '',
      price_period: (server.price_period as 'month' | 'quarter' | 'year') || 'month',
      price_currency: server.price_currency || 'USD',
      purchase_date: server.purchase_date || '',
      auto_renew: server.auto_renew || false,
      tip_badge: server.tip_badge || '',
      notes: server.notes || '',
      group_values: server.group_values ? { ...server.group_values } : {},
      labels: server.labels ? [...server.labels] : [],
      sale_status: (server.sale_status as '' | 'rent' | 'sell') || '',
      sale_contact_url: server.sale_contact_url || '',
      traffic_limit_gb: server.traffic_limit_gb?.toString() || '',
      traffic_threshold_type: server.traffic_threshold_type || 'sum',
      traffic_reset_day: server.traffic_reset_day || 1,
    });
    // Fetch traffic stats for this server
    fetchTrafficStats(server.id);
  };

  const fetchTrafficStats = async (serverId: string) => {
    try {
      const res = await fetch(`/api/traffic/stats/${serverId}`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      if (res.ok) {
        const stats = await res.json();
        if (stats) {
          setEditForm((prev) => ({
            ...prev,
            traffic_limit_gb: stats.monthly_limit_gb?.toString() || prev.traffic_limit_gb,
            traffic_threshold_type: stats.threshold_type || prev.traffic_threshold_type,
            traffic_reset_day: stats.reset_day || prev.traffic_reset_day,
          }));
        }
      }
    } catch (e) {
      console.error('Failed to fetch traffic stats', e);
    }
  };

  const saveEditServer = async () => {
    if (!editingServer) return;

    if (!editForm.name.trim()) {
      setEditError('Server name is required');
      return;
    }

    setEditLoading(true);
    setEditSuccess(false);
    setEditError(null);

    try {
      const syncedGroupValues = await syncTagToGroupValues(editForm.tag);

      const updateData: Record<string, unknown> = {
        name: editForm.name.trim(),
        location: editForm.location.trim(),
        provider: editForm.provider.trim(),
        tag: editForm.tag.trim(),
      };

      if (editForm.price_amount.trim()) {
        updateData.price_amount = editForm.price_amount.trim();
        updateData.price_period = editForm.price_period;
        updateData.price_currency = editForm.price_currency;
      }

      if (editForm.purchase_date.trim()) {
        updateData.purchase_date = editForm.purchase_date.trim();
      }
      updateData.auto_renew = editForm.auto_renew;
      if (editForm.notes.trim()) {
        updateData.notes = editForm.notes.trim();
      }
      if (editForm.tip_badge.trim()) {
        updateData.tip_badge = editForm.tip_badge.trim();
      }

      updateData.group_values = { ...editForm.group_values, ...syncedGroupValues };
      updateData.labels = editForm.labels;
      updateData.sale_status = editForm.sale_status;
      updateData.sale_contact_url = editForm.sale_contact_url;

      // Update traffic settings if configured
      const trafficLimitGB = parseFloat(editForm.traffic_limit_gb);
      if (!isNaN(trafficLimitGB) && trafficLimitGB >= 0) {
        try {
          await fetch('/api/traffic/limit', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${token}`,
            },
            body: JSON.stringify({
              server_id: editingServer,
              monthly_limit_gb: trafficLimitGB,
              threshold_type: editForm.traffic_threshold_type,
              reset_day: editForm.traffic_reset_day,
            }),
          });
        } catch (e) {
          console.error('Failed to update traffic settings', e);
        }
      }

      const res = await fetch(`/api/servers/${editingServer}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify(updateData),
      });

      if (res.ok) {
        const updated = await res.json();
        setServers(servers.map((s) => (s.id === editingServer ? updated : s)));
        setEditSuccess(true);
        setTimeout(() => {
          setEditingServer(null);
          setEditForm(DEFAULT_EDIT_FORM);
          setEditSuccess(false);
        }, 1500);
      } else {
        const errorData = await res.json().catch(() => ({ message: 'Failed to update server' }));
        setEditError(errorData.message || 'Failed to update server');
      }
    } catch (e) {
      console.error('Failed to update server', e);
      setEditError('Network error: Failed to update server');
    } finally {
      setEditLoading(false);
    }
  };

  const addServer = async (e: React.FormEvent) => {
    e.preventDefault();
    setAddLoading(true);

    try {
      const syncedGroupValues = await syncTagToGroupValues(newServer.tag);
      const serverData = {
        ...newServer,
        group_values: { ...newServer.group_values, ...syncedGroupValues },
      };

      const res = await fetch('/api/servers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify(serverData),
      });

      if (res.ok) {
        const server = await res.json();
        setServers([...servers, server]);
        setNewServer({ name: '', url: '', location: '', provider: '', tag: '', group_values: {} });
        setShowAddForm(false);
      }
    } catch (e) {
      console.error('Failed to add server', e);
    }

    setAddLoading(false);
  };

  const deleteServer = async (id: string) => {
    if (!confirm(t('settings.deleteServerConfirm'))) return;

    try {
      const res = await fetch(`/api/servers/${id}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` },
      });

      if (res.ok) {
        setServers(servers.filter((s) => s.id !== id));
      }
    } catch (e) {
      console.error('Failed to delete server', e);
    }
  };

  return (
    <>
      {/* Quick Install Section */}
      <div className="nezha-card p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-cyan-500"></span>
            {t('settings.quickInstallTitle')}
          </h2>
          <button
            onClick={() => setShowInstallCommand(!showInstallCommand)}
            className="px-4 py-2 rounded-lg bg-cyan-500/10 hover:bg-cyan-500/20 text-cyan-400 text-sm font-medium transition-colors flex items-center gap-2"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            {showInstallCommand ? t('settings.hideCommand') : t('settings.showCommand')}
          </button>
        </div>

        <p className="text-gray-400 text-sm mb-4">{t('settings.installInstruction')}</p>

        {showInstallCommand && (
          <div>
            {/* Platform Tabs */}
            <div className="flex gap-2 mb-4">
              <button
                onClick={() => setInstallPlatform('linux')}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center gap-2 ${
                  installPlatform === 'linux'
                    ? 'bg-cyan-500/20 text-cyan-400 border border-cyan-500/30'
                    : 'bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white border border-transparent'
                }`}
              >
                <svg className="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12.504 0c-.155 0-.315.008-.48.021-4.226.333-3.105 4.807-3.17 6.298-.076 1.092-.3 1.953-1.05 3.02-.885 1.051-2.127 2.75-2.716 4.521-.278.832-.41 1.684-.287 2.489a.424.424 0 00-.11.135c-.26.268-.45.6-.663.839-.199.199-.485.267-.797.4-.313.136-.658.269-.864.68-.09.189-.136.394-.132.602 0 .199.027.4.055.536.058.399.116.728.04.97-.249.68-.28 1.145-.106 1.484.174.334.535.47.94.601.81.2 1.91.135 2.774.6.926.466 1.866.67 2.616.47.526-.116.97-.464 1.208-.946.587-.003 1.23-.269 2.26-.334.699-.058 1.574.267 2.577.2.025.134.063.198.114.333l.003.003c.391.778 1.113 1.132 1.884 1.071.771-.06 1.592-.536 2.257-1.306.631-.765 1.683-1.084 2.378-1.503.348-.199.629-.469.649-.853.023-.4-.2-.811-.714-1.376v-.097l-.003-.003c-.17-.2-.25-.535-.338-.926-.085-.401-.182-.786-.492-1.046h-.003c-.059-.054-.123-.067-.188-.135a.357.357 0 00-.19-.064c.431-1.278.264-2.55-.173-3.694-.533-1.41-1.465-2.638-2.175-3.483-.796-1.005-1.576-1.957-1.56-3.368.026-2.152.236-6.133-3.544-6.139z" />
                </svg>
                {t('settings.linuxInstall')}
              </button>
              <button
                onClick={() => setInstallPlatform('windows')}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center gap-2 ${
                  installPlatform === 'windows'
                    ? 'bg-cyan-500/20 text-cyan-400 border border-cyan-500/30'
                    : 'bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white border border-transparent'
                }`}
              >
                <svg className="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801" />
                </svg>
                {t('settings.windowsInstall')}
              </button>
            </div>

            <div className="relative">
              <pre className="p-4 rounded-xl bg-black/40 border border-white/10 text-sm text-emerald-400 font-mono overflow-x-auto whitespace-pre-wrap break-all">
                {installPlatform === 'windows' ? windowsInstallCommand : installCommand}
              </pre>
              <button
                onClick={copyToClipboard}
                className={`absolute top-3 right-3 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                  copied
                    ? 'bg-emerald-500/20 text-emerald-400'
                    : 'bg-white/10 hover:bg-white/20 text-gray-400 hover:text-white'
                }`}
              >
                {copied ? t('common.copied') : t('common.copy')}
              </button>
            </div>

            {installPlatform === 'windows' && (
              <p className="mt-3 text-xs text-gray-500">
                💡 命令已包含 TLS 1.2 设置，并自动处理执行策略。请以管理员身份运行 PowerShell。
              </p>
            )}
          </div>
        )}
      </div>

      {/* Server Management Section */}
      <div className="nezha-card p-6 mb-6">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-emerald-500"></span>
            {t('settings.serverManagement')}
          </h2>
          <button
            onClick={() => setShowAddForm(true)}
            className="px-4 py-2 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 text-sm font-medium transition-colors flex items-center gap-2"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            {t('settings.addAgent')}
          </button>
        </div>

        {/* Add Server Form */}
        {showAddForm && (
          <form onSubmit={addServer} className="mb-6 p-4 rounded-xl bg-white/[0.02] border border-white/10">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
              <div>
                <label className="block text-xs text-gray-500 mb-1">{t('settings.serverName')}</label>
                <input
                  type="text"
                  value={newServer.name}
                  onChange={(e) => setNewServer({ ...newServer, name: e.target.value })}
                  className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-emerald-500/50"
                  placeholder={isZh ? '例：US-West-1' : 'e.g., US-West-1'}
                  required
                />
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">{t('settings.locationCode')}</label>
                <input
                  type="text"
                  value={newServer.location}
                  onChange={(e) => setNewServer({ ...newServer, location: e.target.value })}
                  className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-emerald-500/50"
                  placeholder={isZh ? '例：US、HK、JP' : 'e.g., US, HK, JP'}
                />
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">{t('settings.provider')}</label>
                <input
                  type="text"
                  value={newServer.provider}
                  onChange={(e) => setNewServer({ ...newServer, provider: e.target.value })}
                  className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-emerald-500/50"
                  placeholder={isZh ? '例：阿里云、AWS、Vultr' : 'e.g., AWS, Vultr'}
                />
              </div>
            </div>

            {/* Group Dimensions Selection */}
            {dimensions.length > 0 && (
              <div className="mb-4 pt-4 border-t border-white/10">
                <label className="block text-xs text-gray-500 mb-3">分组标签</label>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {dimensions
                    .filter((dim) => dim.enabled)
                    .sort((a, b) => a.sort_order - b.sort_order)
                    .map((dimension) => (
                      <div key={dimension.id}>
                        <label className="block text-xs text-gray-400 mb-1">{dimension.name}</label>
                        <select
                          value={newServer.group_values[dimension.id] || ''}
                          onChange={(e) => {
                            const newGroupValues = { ...newServer.group_values };
                            if (e.target.value) {
                              newGroupValues[dimension.id] = e.target.value;
                            } else {
                              delete newGroupValues[dimension.id];
                            }
                            setNewServer({ ...newServer, group_values: newGroupValues });
                          }}
                          className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-emerald-500/50"
                        >
                          <option value="">-- 未选择 --</option>
                          {dimension.options
                            .sort((a, b) => a.sort_order - b.sort_order)
                            .map((option) => (
                              <option key={option.id} value={option.id}>
                                {option.name}
                              </option>
                            ))}
                        </select>
                      </div>
                    ))}
                </div>
              </div>
            )}

            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setShowAddForm(false)}
                className="px-4 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-gray-400 text-sm transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={addLoading}
                className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-sm font-medium transition-colors disabled:opacity-50"
              >
                {addLoading ? 'Adding...' : 'Add Server'}
              </button>
            </div>
          </form>
        )}

        {/* Server List */}
        <div className="space-y-3">
          {/* Remote Agents */}
          {servers.length === 0 ? (
            <div className="text-center py-6 text-gray-500 border border-dashed border-white/10 rounded-xl">
              <p>{t('settings.noRemoteAgents')}</p>
              <p className="text-sm mt-1">{t('settings.installAgentTip')}</p>
            </div>
          ) : (
            <>
              {/* Batch Update Header */}
              <BatchUpdateHeader
                servers={servers}
                agentStatus={agentStatus}
                batchUpdating={batchUpdating}
                batchUpdateProgress={batchUpdateProgress}
                onUpdateAll={updateAllAgents}
              />

              {/* Server List */}
              {servers.map((server) => (
                <ServerCard
                  key={server.id}
                  server={server}
                  isOnline={agentStatus[server.id] || false}
                  isUpdating={updatingAgents[server.id] || false}
                  editingServer={editingServer}
                  editForm={editForm}
                  editLoading={editLoading}
                  editSuccess={editSuccess}
                  editError={editError}
                  copiedToken={copiedToken}
                  dimensions={dimensions}
                  onEdit={startEditServer}
                  onSaveEdit={saveEditServer}
                  onCancelEdit={() => {
                    setEditingServer(null);
                    setEditForm(DEFAULT_EDIT_FORM);
                  }}
                  onEditFormChange={setEditForm}
                  onUpdate={updateAgent}
                  onDelete={deleteServer}
                  onCopyToken={copyToken}
                />
              ))}
            </>
          )}
        </div>
      </div>
    </>
  );
}
