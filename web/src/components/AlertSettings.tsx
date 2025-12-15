import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { showToast } from './Toast';
import type { 
  AlertConfig, 
  NotificationChannel, 
  AlertState, 
  AlertStats,
  AlertHistory,
  ThresholdConfig
} from '../types';

interface AlertSettingsProps {
  token: string | null;
  servers?: { id: string; name: string }[];
}

const CHANNEL_TYPES = [
  { value: 'telegram', label: 'Telegram', icon: '📱' },
  { value: 'discord', label: 'Discord', icon: '🎮' },
  { value: 'email', label: 'Email', icon: '📧' },
  { value: 'webhook', label: 'Webhook', icon: '🔗' },
  { value: 'bark', label: 'Bark (iOS)', icon: '🔔' },
  { value: 'serverchan', label: 'ServerChan', icon: '💬' },
];

const CHANNEL_CONFIG_FIELDS: Record<string, { key: string; label: string; labelZh: string; type?: string; placeholder?: string }[]> = {
  telegram: [
    { key: 'bot_token', label: 'Bot Token', labelZh: 'Bot Token', placeholder: '123456789:ABC...' },
    { key: 'chat_id', label: 'Chat ID', labelZh: '聊天 ID', placeholder: '-1001234567890' },
  ],
  discord: [
    { key: 'webhook_url', label: 'Webhook URL', labelZh: 'Webhook 地址', placeholder: 'https://discord.com/api/webhooks/...' },
    { key: 'username', label: 'Bot Name (optional)', labelZh: '机器人名称 (可选)', placeholder: 'vStats Alert' },
  ],
  email: [
    { key: 'smtp_host', label: 'SMTP Host', labelZh: 'SMTP 服务器', placeholder: 'smtp.gmail.com' },
    { key: 'smtp_port', label: 'SMTP Port', labelZh: 'SMTP 端口', placeholder: '587' },
    { key: 'username', label: 'Username', labelZh: '用户名', placeholder: 'your@email.com' },
    { key: 'password', label: 'Password', labelZh: '密码', type: 'password' },
    { key: 'from', label: 'From Address', labelZh: '发件人地址', placeholder: 'alerts@example.com' },
    { key: 'to', label: 'To Addresses', labelZh: '收件人地址', placeholder: 'admin@example.com, ops@example.com' },
  ],
  webhook: [
    { key: 'url', label: 'Webhook URL', labelZh: 'Webhook 地址', placeholder: 'https://your-server.com/webhook' },
    { key: 'method', label: 'HTTP Method', labelZh: 'HTTP 方法', placeholder: 'POST' },
  ],
  bark: [
    { key: 'device_key', label: 'Device Key', labelZh: '设备密钥', placeholder: 'Your Bark device key' },
    { key: 'server_url', label: 'Server URL (optional)', labelZh: '服务器地址 (可选)', placeholder: 'https://api.day.app' },
  ],
  serverchan: [
    { key: 'send_key', label: 'Send Key', labelZh: 'SendKey', placeholder: 'SCT...' },
    { key: 'channel', label: 'Channel (optional)', labelZh: '渠道 (可选)', placeholder: '9' },
  ],
};

export default function AlertSettings({ token }: AlertSettingsProps) {
  const { i18n } = useTranslation();
  const isZh = i18n.language.startsWith('zh');
  
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState<AlertConfig | null>(null);
  const [activeAlerts, setActiveAlerts] = useState<AlertState[]>([]);
  const [alertStats, setAlertStats] = useState<AlertStats | null>(null);
  const [alertHistory, setAlertHistory] = useState<AlertHistory[]>([]);
  const [activeTab, setActiveTab] = useState<'overview' | 'channels' | 'rules' | 'history'>('overview');
  
  // Channel editing
  const [editingChannel, setEditingChannel] = useState<NotificationChannel | null>(null);
  const [newChannel, setNewChannel] = useState<Partial<NotificationChannel> | null>(null);
  const [testingChannel, setTestingChannel] = useState<string | null>(null);

  // Load alert configuration
  const loadConfig = useCallback(async () => {
    if (!token) return;
    
    try {
      const res = await fetch('/api/settings/alerts', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setConfig(data);
      }
    } catch (err) {
      console.error('Failed to load alert config:', err);
    } finally {
      setLoading(false);
    }
  }, [token]);

  // Load active alerts
  const loadAlerts = useCallback(async () => {
    if (!token) return;
    
    try {
      const res = await fetch('/api/alerts', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setActiveAlerts(data.alerts || []);
        setAlertStats(data.stats);
      }
    } catch (err) {
      console.error('Failed to load alerts:', err);
    }
  }, [token]);

  // Load alert history
  const loadHistory = useCallback(async () => {
    if (!token) return;
    
    try {
      const res = await fetch('/api/alerts/history?limit=50', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setAlertHistory(data || []);
      }
    } catch (err) {
      console.error('Failed to load alert history:', err);
    }
  }, [token]);

  useEffect(() => {
    loadConfig();
    loadAlerts();
  }, [loadConfig, loadAlerts]);

  useEffect(() => {
    if (activeTab === 'history') {
      loadHistory();
    }
  }, [activeTab, loadHistory]);

  // Save configuration
  const saveConfig = async (updates: Partial<AlertConfig>) => {
    if (!token) return;
    
    setSaving(true);
    try {
      const res = await fetch('/api/settings/alerts', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(updates),
      });
      
      if (res.ok) {
        showToast(isZh ? '保存成功' : 'Saved successfully', 'success');
        loadConfig();
      } else {
        const err = await res.json();
        showToast(err.error || (isZh ? '保存失败' : 'Save failed'), 'error');
      }
    } catch (err) {
      showToast(isZh ? '保存失败' : 'Save failed', 'error');
    } finally {
      setSaving(false);
    }
  };

  // Toggle alerts enabled
  const toggleEnabled = async () => {
    if (!config) return;
    await saveConfig({ enabled: !config.enabled });
  };

  // Add channel
  const addChannel = async () => {
    if (!newChannel || !token) return;
    
    setSaving(true);
    try {
      const res = await fetch('/api/alerts/channels', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          type: newChannel.type,
          name: newChannel.name,
          enabled: true,
          config: newChannel.config || {},
        }),
      });
      
      if (res.ok) {
        showToast(isZh ? '渠道添加成功' : 'Channel added', 'success');
        setNewChannel(null);
        loadConfig();
      } else {
        const err = await res.json();
        showToast(err.error || (isZh ? '添加失败' : 'Add failed'), 'error');
      }
    } catch (err) {
      showToast(isZh ? '添加失败' : 'Add failed', 'error');
    } finally {
      setSaving(false);
    }
  };

  // Update channel
  const updateChannel = async (channel: NotificationChannel) => {
    if (!token) return;
    
    setSaving(true);
    try {
      const res = await fetch(`/api/alerts/channels/${channel.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          type: channel.type,
          name: channel.name,
          enabled: channel.enabled,
          config: channel.config,
        }),
      });
      
      if (res.ok) {
        showToast(isZh ? '保存成功' : 'Saved', 'success');
        setEditingChannel(null);
        loadConfig();
      } else {
        const err = await res.json();
        showToast(err.error || (isZh ? '保存失败' : 'Save failed'), 'error');
      }
    } catch (err) {
      showToast(isZh ? '保存失败' : 'Save failed', 'error');
    } finally {
      setSaving(false);
    }
  };

  // Delete channel
  const deleteChannel = async (channelId: string) => {
    if (!token || !confirm(isZh ? '确定删除此渠道？' : 'Delete this channel?')) return;
    
    try {
      const res = await fetch(`/api/alerts/channels/${channelId}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      
      if (res.ok) {
        showToast(isZh ? '删除成功' : 'Deleted', 'success');
        loadConfig();
      }
    } catch (err) {
      showToast(isZh ? '删除失败' : 'Delete failed', 'error');
    }
  };

  // Test channel
  const testChannel = async (channelId: string) => {
    if (!token) return;
    
    setTestingChannel(channelId);
    try {
      const res = await fetch('/api/alerts/channels/test', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ channel_id: channelId }),
      });
      
      if (res.ok) {
        showToast(isZh ? '测试通知已发送' : 'Test notification sent', 'success');
      } else {
        const err = await res.json();
        showToast(err.error || (isZh ? '测试失败' : 'Test failed'), 'error');
      }
    } catch (err) {
      showToast(isZh ? '测试失败' : 'Test failed', 'error');
    } finally {
      setTestingChannel(null);
    }
  };

  // Save rules
  const saveRules = async (ruleType: 'offline' | 'load' | 'traffic', rule: unknown) => {
    if (!token) return;
    
    setSaving(true);
    try {
      const res = await fetch(`/api/alerts/rules/${ruleType}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(rule),
      });
      
      if (res.ok) {
        showToast(isZh ? '规则已保存' : 'Rule saved', 'success');
        loadConfig();
      } else {
        const err = await res.json();
        showToast(err.error || (isZh ? '保存失败' : 'Save failed'), 'error');
      }
    } catch (err) {
      showToast(isZh ? '保存失败' : 'Save failed', 'error');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin w-8 h-8 border-2 border-cyan-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-bold text-white flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-red-500"></span>
          {isZh ? '告警通知' : 'Alert Notifications'}
        </h2>
        <label className="flex items-center gap-3 cursor-pointer">
          <span className="text-sm text-gray-400">
            {config?.enabled ? (isZh ? '已启用' : 'Enabled') : (isZh ? '已禁用' : 'Disabled')}
          </span>
          <div 
            onClick={toggleEnabled}
            className={`relative w-12 h-6 rounded-full transition-colors ${
              config?.enabled ? 'bg-emerald-500' : 'bg-gray-600'
            }`}
          >
            <div className={`absolute top-1 w-4 h-4 bg-white rounded-full transition-transform ${
              config?.enabled ? 'translate-x-7' : 'translate-x-1'
            }`} />
          </div>
        </label>
      </div>

      {/* Stats Overview */}
      {alertStats && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="nezha-card p-4">
            <div className="text-2xl font-bold text-white">{alertStats.total_firing}</div>
            <div className="text-xs text-gray-400">{isZh ? '活跃告警' : 'Active Alerts'}</div>
          </div>
          <div className="nezha-card p-4">
            <div className="text-2xl font-bold text-red-400">{alertStats.critical}</div>
            <div className="text-xs text-gray-400">{isZh ? '严重' : 'Critical'}</div>
          </div>
          <div className="nezha-card p-4">
            <div className="text-2xl font-bold text-yellow-400">{alertStats.warning}</div>
            <div className="text-xs text-gray-400">{isZh ? '警告' : 'Warning'}</div>
          </div>
          <div className="nezha-card p-4">
            <div className="text-2xl font-bold text-emerald-400">
              {alertStats.servers_online}/{alertStats.servers_total}
            </div>
            <div className="text-xs text-gray-400">{isZh ? '服务器在线' : 'Servers Online'}</div>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-2 border-b border-white/10 pb-2">
        {(['overview', 'channels', 'rules', 'history'] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              activeTab === tab
                ? 'bg-cyan-500/20 text-cyan-400'
                : 'text-gray-400 hover:text-white hover:bg-white/5'
            }`}
          >
            {tab === 'overview' && (isZh ? '概览' : 'Overview')}
            {tab === 'channels' && (isZh ? '通知渠道' : 'Channels')}
            {tab === 'rules' && (isZh ? '告警规则' : 'Rules')}
            {tab === 'history' && (isZh ? '历史记录' : 'History')}
          </button>
        ))}
      </div>

      {/* Overview Tab */}
      {activeTab === 'overview' && (
        <div className="space-y-4">
          {activeAlerts.length === 0 ? (
            <div className="nezha-card p-8 text-center">
              <div className="text-4xl mb-4">✅</div>
              <div className="text-gray-400">{isZh ? '当前没有活跃告警' : 'No active alerts'}</div>
            </div>
          ) : (
            <div className="space-y-3">
              {activeAlerts.map((alert) => (
                <div key={alert.id} className={`nezha-card p-4 border-l-4 ${
                  alert.severity === 'critical' ? 'border-red-500' : 'border-yellow-500'
                }`}>
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="flex items-center gap-2 mb-1">
                        <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                          alert.severity === 'critical' 
                            ? 'bg-red-500/20 text-red-400' 
                            : 'bg-yellow-500/20 text-yellow-400'
                        }`}>
                          {alert.severity === 'critical' ? (isZh ? '严重' : 'Critical') : (isZh ? '警告' : 'Warning')}
                        </span>
                        <span className="text-sm text-gray-400">{alert.server_name}</span>
                      </div>
                      <div className="text-white">{alert.message}</div>
                      <div className="text-xs text-gray-500 mt-1">
                        {isZh ? '开始时间' : 'Started'}: {new Date(alert.started_at).toLocaleString()}
                      </div>
                    </div>
                    {!alert.muted && (
                      <button
                        onClick={async () => {
                          if (!token) return;
                          await fetch(`/api/alerts/${alert.id}/mute`, {
                            method: 'POST',
                            headers: { Authorization: `Bearer ${token}` },
                          });
                          loadAlerts();
                        }}
                        className="px-3 py-1 rounded text-xs bg-white/5 hover:bg-white/10 text-gray-400"
                      >
                        {isZh ? '静音' : 'Mute'}
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Channels Tab */}
      {activeTab === 'channels' && (
        <div className="space-y-4">
          {/* Existing channels */}
          {config?.channels.map((channel) => (
            <div key={channel.id} className="nezha-card p-4">
              {editingChannel?.id === channel.id ? (
                <ChannelForm
                  channel={editingChannel}
                  onChange={setEditingChannel}
                  onSave={() => updateChannel(editingChannel)}
                  onCancel={() => setEditingChannel(null)}
                  saving={saving}
                  isZh={isZh}
                />
              ) : (
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <span className="text-2xl">
                      {CHANNEL_TYPES.find(t => t.value === channel.type)?.icon || '📢'}
                    </span>
                    <div>
                      <div className="font-medium text-white">{channel.name}</div>
                      <div className="text-xs text-gray-500">
                        {CHANNEL_TYPES.find(t => t.value === channel.type)?.label}
                      </div>
                    </div>
                    <div className={`px-2 py-0.5 rounded text-xs ${
                      channel.enabled 
                        ? 'bg-emerald-500/20 text-emerald-400' 
                        : 'bg-gray-500/20 text-gray-400'
                    }`}>
                      {channel.enabled ? (isZh ? '启用' : 'Enabled') : (isZh ? '禁用' : 'Disabled')}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => testChannel(channel.id)}
                      disabled={testingChannel === channel.id}
                      className="px-3 py-1.5 rounded text-xs bg-cyan-500/10 hover:bg-cyan-500/20 text-cyan-400 disabled:opacity-50"
                    >
                      {testingChannel === channel.id 
                        ? (isZh ? '发送中...' : 'Sending...') 
                        : (isZh ? '测试' : 'Test')}
                    </button>
                    <button
                      onClick={() => setEditingChannel({ ...channel })}
                      className="px-3 py-1.5 rounded text-xs bg-white/5 hover:bg-white/10 text-gray-400"
                    >
                      {isZh ? '编辑' : 'Edit'}
                    </button>
                    <button
                      onClick={() => deleteChannel(channel.id)}
                      className="px-3 py-1.5 rounded text-xs bg-red-500/10 hover:bg-red-500/20 text-red-400"
                    >
                      {isZh ? '删除' : 'Delete'}
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))}

          {/* Add new channel */}
          {newChannel ? (
            <div className="nezha-card p-4">
              <ChannelForm
                channel={newChannel as NotificationChannel}
                onChange={(c) => setNewChannel(c)}
                onSave={addChannel}
                onCancel={() => setNewChannel(null)}
                saving={saving}
                isNew
                isZh={isZh}
              />
            </div>
          ) : (
            <button
              onClick={() => setNewChannel({ type: 'telegram', name: '', enabled: true, config: {} })}
              className="w-full p-4 rounded-xl border-2 border-dashed border-white/10 hover:border-cyan-500/30 text-gray-400 hover:text-cyan-400 transition-colors"
            >
              + {isZh ? '添加通知渠道' : 'Add Notification Channel'}
            </button>
          )}
        </div>
      )}

      {/* Rules Tab */}
      {activeTab === 'rules' && config && (
        <div className="space-y-6">
          {/* Offline Alert Rule */}
          <div className="nezha-card p-4">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <span className="text-2xl">🔌</span>
                <div>
                  <div className="font-medium text-white">{isZh ? '离线告警' : 'Offline Alert'}</div>
                  <div className="text-xs text-gray-500">
                    {isZh ? '服务器离线时发送通知' : 'Notify when server goes offline'}
                  </div>
                </div>
              </div>
              <label className="flex items-center gap-2 cursor-pointer">
                <div 
                  onClick={() => saveRules('offline', { 
                    ...config.rules.offline, 
                    enabled: !config.rules.offline.enabled 
                  })}
                  className={`relative w-10 h-5 rounded-full transition-colors ${
                    config.rules.offline.enabled ? 'bg-emerald-500' : 'bg-gray-600'
                  }`}
                >
                  <div className={`absolute top-0.5 w-4 h-4 bg-white rounded-full transition-transform ${
                    config.rules.offline.enabled ? 'translate-x-5' : 'translate-x-0.5'
                  }`} />
                </div>
              </label>
            </div>
            
            {config.rules.offline.enabled && (
              <div className="space-y-4 pt-4 border-t border-white/10">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">
                    {isZh ? '宽限期（秒）' : 'Grace Period (seconds)'}
                  </label>
                  <input
                    type="number"
                    value={config.rules.offline.grace_period}
                    onChange={(e) => setConfig({
                      ...config,
                      rules: {
                        ...config.rules,
                        offline: { ...config.rules.offline, grace_period: parseInt(e.target.value) || 60 }
                      }
                    })}
                    className="w-32 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm"
                  />
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => saveRules('offline', config.rules.offline)}
                    disabled={saving}
                    className="px-4 py-2 rounded-lg bg-cyan-500 hover:bg-cyan-600 text-white text-sm disabled:opacity-50"
                  >
                    {saving ? (isZh ? '保存中...' : 'Saving...') : (isZh ? '保存' : 'Save')}
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Load Alert Rule */}
          <div className="nezha-card p-4">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <span className="text-2xl">📊</span>
                <div>
                  <div className="font-medium text-white">{isZh ? '负载告警' : 'Load Alert'}</div>
                  <div className="text-xs text-gray-500">
                    {isZh ? 'CPU/内存/磁盘超过阈值时发送通知' : 'Notify when CPU/Memory/Disk exceeds threshold'}
                  </div>
                </div>
              </div>
              <label className="flex items-center gap-2 cursor-pointer">
                <div 
                  onClick={() => saveRules('load', { 
                    ...config.rules.load, 
                    enabled: !config.rules.load.enabled 
                  })}
                  className={`relative w-10 h-5 rounded-full transition-colors ${
                    config.rules.load.enabled ? 'bg-emerald-500' : 'bg-gray-600'
                  }`}
                >
                  <div className={`absolute top-0.5 w-4 h-4 bg-white rounded-full transition-transform ${
                    config.rules.load.enabled ? 'translate-x-5' : 'translate-x-0.5'
                  }`} />
                </div>
              </label>
            </div>
            
            {config.rules.load.enabled && (
              <div className="space-y-4 pt-4 border-t border-white/10">
                <ThresholdEditor
                  label={isZh ? 'CPU' : 'CPU'}
                  config={config.rules.load.cpu}
                  onChange={(cpu) => setConfig({
                    ...config,
                    rules: {
                      ...config.rules,
                      load: { ...config.rules.load, cpu }
                    }
                  })}
                  isZh={isZh}
                />
                <ThresholdEditor
                  label={isZh ? '内存' : 'Memory'}
                  config={config.rules.load.memory}
                  onChange={(memory) => setConfig({
                    ...config,
                    rules: {
                      ...config.rules,
                      load: { ...config.rules.load, memory }
                    }
                  })}
                  isZh={isZh}
                />
                <ThresholdEditor
                  label={isZh ? '磁盘' : 'Disk'}
                  config={config.rules.load.disk}
                  onChange={(disk) => setConfig({
                    ...config,
                    rules: {
                      ...config.rules,
                      load: { ...config.rules.load, disk }
                    }
                  })}
                  isZh={isZh}
                />
                <div className="flex gap-2">
                  <button
                    onClick={() => saveRules('load', config.rules.load)}
                    disabled={saving}
                    className="px-4 py-2 rounded-lg bg-cyan-500 hover:bg-cyan-600 text-white text-sm disabled:opacity-50"
                  >
                    {saving ? (isZh ? '保存中...' : 'Saving...') : (isZh ? '保存' : 'Save')}
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Traffic Alert Rule */}
          <div className="nezha-card p-4">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <span className="text-2xl">📈</span>
                <div>
                  <div className="font-medium text-white">{isZh ? '流量告警' : 'Traffic Alert'}</div>
                  <div className="text-xs text-gray-500">
                    {isZh ? '月度流量超过限额时发送通知' : 'Notify when monthly traffic exceeds limit'}
                  </div>
                </div>
              </div>
              <label className="flex items-center gap-2 cursor-pointer">
                <div 
                  onClick={() => saveRules('traffic', { 
                    ...config.rules.traffic, 
                    enabled: !config.rules.traffic.enabled 
                  })}
                  className={`relative w-10 h-5 rounded-full transition-colors ${
                    config.rules.traffic.enabled ? 'bg-emerald-500' : 'bg-gray-600'
                  }`}
                >
                  <div className={`absolute top-0.5 w-4 h-4 bg-white rounded-full transition-transform ${
                    config.rules.traffic.enabled ? 'translate-x-5' : 'translate-x-0.5'
                  }`} />
                </div>
              </label>
            </div>
            
            {config.rules.traffic.enabled && (
              <div className="pt-4 border-t border-white/10">
                <div className="text-sm text-gray-400 mb-2">
                  {isZh ? '在服务器管理中为每个服务器设置流量限额' : 'Set traffic limits for each server in Server Management'}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* History Tab */}
      {activeTab === 'history' && (
        <div className="space-y-3">
          {alertHistory.length === 0 ? (
            <div className="nezha-card p-8 text-center">
              <div className="text-4xl mb-4">📜</div>
              <div className="text-gray-400">{isZh ? '暂无历史记录' : 'No history'}</div>
            </div>
          ) : (
            alertHistory.map((h) => (
              <div key={h.id} className="nezha-card p-4">
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2 mb-1">
                      <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                        h.severity === 'critical' 
                          ? 'bg-red-500/20 text-red-400' 
                          : 'bg-yellow-500/20 text-yellow-400'
                      }`}>
                        {h.type}
                      </span>
                      <span className="text-sm text-gray-400">{h.server_name}</span>
                      {h.resolved_at && (
                        <span className="px-2 py-0.5 rounded text-xs bg-emerald-500/20 text-emerald-400">
                          {isZh ? '已恢复' : 'Resolved'}
                        </span>
                      )}
                    </div>
                    <div className="text-white text-sm">{h.message}</div>
                    <div className="flex gap-4 text-xs text-gray-500 mt-1">
                      <span>{isZh ? '开始' : 'Started'}: {new Date(h.started_at).toLocaleString()}</span>
                      {h.duration > 0 && (
                        <span>{isZh ? '持续' : 'Duration'}: {formatDuration(h.duration)}</span>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}

// Channel Form Component
function ChannelForm({
  channel,
  onChange,
  onSave,
  onCancel,
  saving,
  isNew,
  isZh,
}: {
  channel: NotificationChannel;
  onChange: (channel: NotificationChannel) => void;
  onSave: () => void;
  onCancel: () => void;
  saving: boolean;
  isNew?: boolean;
  isZh: boolean;
}) {
  const fields = CHANNEL_CONFIG_FIELDS[channel.type] || [];

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-xs text-gray-500 mb-1">{isZh ? '渠道类型' : 'Channel Type'}</label>
          <select
            value={channel.type}
            onChange={(e) => onChange({ ...channel, type: e.target.value as NotificationChannel['type'], config: {} })}
            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm"
          >
            {CHANNEL_TYPES.map((t) => (
              <option key={t.value} value={t.value}>{t.icon} {t.label}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">{isZh ? '名称' : 'Name'}</label>
          <input
            type="text"
            value={channel.name}
            onChange={(e) => onChange({ ...channel, name: e.target.value })}
            placeholder={isZh ? '例如: 运维通知' : 'e.g., Ops Alerts'}
            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm"
          />
        </div>
      </div>

      {fields.map((field) => (
        <div key={field.key}>
          <label className="block text-xs text-gray-500 mb-1">
            {isZh ? field.labelZh : field.label}
          </label>
          <input
            type={field.type || 'text'}
            value={channel.config[field.key] || ''}
            onChange={(e) => onChange({
              ...channel,
              config: { ...channel.config, [field.key]: e.target.value }
            })}
            placeholder={field.placeholder}
            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm"
          />
        </div>
      ))}

      <div className="flex items-center gap-4">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={channel.enabled}
            onChange={(e) => onChange({ ...channel, enabled: e.target.checked })}
            className="rounded"
          />
          <span className="text-sm text-gray-400">{isZh ? '启用' : 'Enabled'}</span>
        </label>
      </div>

      <div className="flex gap-2">
        <button
          onClick={onCancel}
          className="px-4 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-gray-400 text-sm"
        >
          {isZh ? '取消' : 'Cancel'}
        </button>
        <button
          onClick={onSave}
          disabled={saving || !channel.name}
          className="px-4 py-2 rounded-lg bg-cyan-500 hover:bg-cyan-600 text-white text-sm disabled:opacity-50"
        >
          {saving 
            ? (isZh ? '保存中...' : 'Saving...') 
            : (isNew ? (isZh ? '添加' : 'Add') : (isZh ? '保存' : 'Save'))
          }
        </button>
      </div>
    </div>
  );
}

// Threshold Editor Component
function ThresholdEditor({
  label,
  config,
  onChange,
  isZh,
}: {
  label: string;
  config?: ThresholdConfig;
  onChange: (config: ThresholdConfig) => void;
  isZh: boolean;
}) {
  const value = config || { warning: 80, critical: 95, duration: 60 };

  return (
    <div className="grid grid-cols-3 gap-4">
      <div>
        <label className="block text-xs text-gray-500 mb-1">
          {label} {isZh ? '警告阈值 (%)' : 'Warning (%)'}
        </label>
        <input
          type="number"
          value={value.warning || ''}
          onChange={(e) => onChange({ ...value, warning: parseInt(e.target.value) || undefined })}
          placeholder="80"
          className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm"
        />
      </div>
      <div>
        <label className="block text-xs text-gray-500 mb-1">
          {label} {isZh ? '严重阈值 (%)' : 'Critical (%)'}
        </label>
        <input
          type="number"
          value={value.critical || ''}
          onChange={(e) => onChange({ ...value, critical: parseInt(e.target.value) || undefined })}
          placeholder="95"
          className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm"
        />
      </div>
      <div>
        <label className="block text-xs text-gray-500 mb-1">
          {isZh ? '持续时间 (秒)' : 'Duration (s)'}
        </label>
        <input
          type="number"
          value={value.duration || ''}
          onChange={(e) => onChange({ ...value, duration: parseInt(e.target.value) || undefined })}
          placeholder="60"
          className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm"
        />
      </div>
    </div>
  );
}

// Helper function
function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
  return `${Math.floor(seconds / 86400)}d ${Math.floor((seconds % 86400) / 3600)}h`;
}
