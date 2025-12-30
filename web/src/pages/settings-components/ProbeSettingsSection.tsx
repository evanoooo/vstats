/**
 * Probe Settings Section
 * Manages ping targets for latency monitoring
 */

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { ProbeSettings } from './types';

export interface ProbeSettingsSectionProps {
  token: string | null;
  probeSettings: ProbeSettings;
  onProbeSettingsChange: (settings: ProbeSettings) => void;
}

export function ProbeSettingsSection({
  token,
  probeSettings,
  onProbeSettingsChange,
}: ProbeSettingsSectionProps) {
  const { t } = useTranslation();
  const [showProbeSettings, setShowProbeSettings] = useState(false);
  const [probeSaving, setProbeSaving] = useState(false);
  const [probeSuccess, setProbeSuccess] = useState(false);

  const saveProbeSettings = async () => {
    setProbeSaving(true);
    setProbeSuccess(false);

    try {
      const res = await fetch('/api/settings/probe', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify(probeSettings),
      });

      if (res.ok) {
        setProbeSuccess(true);
        setTimeout(() => setProbeSuccess(false), 2000);
      }
    } catch (e) {
      console.error('Failed to save probe settings', e);
    }

    setProbeSaving(false);
  };

  const addPingTarget = () => {
    onProbeSettingsChange({
      ...probeSettings,
      ping_targets: [...probeSettings.ping_targets, { name: '', host: '', type: 'icmp', port: 80 }],
    });
  };

  const removePingTarget = (index: number) => {
    onProbeSettingsChange({
      ...probeSettings,
      ping_targets: probeSettings.ping_targets.filter((_, i) => i !== index),
    });
  };

  const updatePingTarget = (index: number, field: 'name' | 'host' | 'type' | 'port', value: string | number) => {
    const newTargets = [...probeSettings.ping_targets];
    if (field === 'port') {
      newTargets[index] = { ...newTargets[index], [field]: typeof value === 'number' ? value : parseInt(value as string) || 80 };
    } else {
      newTargets[index] = { ...newTargets[index], [field]: value };
    }
    onProbeSettingsChange({ ...probeSettings, ping_targets: newTargets });
  };

  return (
    <div className="nezha-card p-6 mb-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-bold text-white flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-purple-500"></span>
          {t('settings.probeSettings')}
        </h2>
        <button
          onClick={() => setShowProbeSettings(!showProbeSettings)}
          className="px-4 py-2 rounded-lg bg-purple-500/10 hover:bg-purple-500/20 text-purple-400 text-sm font-medium transition-colors"
        >
          {showProbeSettings ? t('common.close') : t('common.edit')}
        </button>
      </div>

      <p className="text-gray-400 text-sm mb-4">
        Configure ping targets for latency monitoring. Agents will test connectivity to these targets and report latency.
        Supports both ICMP ping and TCP connection tests. TCP is useful when ICMP is blocked or you want to test actual service connectivity.
      </p>

      {probeSuccess && (
        <div className="mb-4 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm">
          Probe settings saved! Agents will update on next connection.
        </div>
      )}

      {showProbeSettings && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <label className="text-xs text-gray-500 uppercase tracking-wider">Ping Targets</label>
            <button
              type="button"
              onClick={addPingTarget}
              className="text-xs text-purple-400 hover:text-purple-300 transition-colors"
            >
              + Add Target
            </button>
          </div>

          {probeSettings.ping_targets.length === 0 ? (
            <div className="text-gray-600 text-sm text-center py-4 border border-dashed border-white/10 rounded-lg">
              No ping targets configured. Using defaults (Google DNS, Cloudflare).
            </div>
          ) : (
            <div className="space-y-3">
              {probeSettings.ping_targets.map((target, index) => (
                <div key={index} className="flex items-center gap-2 flex-wrap">
                  <input
                    type="text"
                    value={target.name}
                    onChange={(e) => updatePingTarget(index, 'name', e.target.value)}
                    className="w-32 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                    placeholder="Name"
                  />
                  <input
                    type="text"
                    value={target.host}
                    onChange={(e) => updatePingTarget(index, 'host', e.target.value)}
                    className="flex-1 min-w-[150px] px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50 font-mono"
                    placeholder="Host/IP"
                  />
                  <select
                    value={target.type || 'icmp'}
                    onChange={(e) => updatePingTarget(index, 'type', e.target.value)}
                    className="w-24 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                  >
                    <option value="icmp">ICMP</option>
                    <option value="tcp">TCP</option>
                  </select>
                  {target.type === 'tcp' && (
                    <input
                      type="number"
                      value={target.port || 80}
                      onChange={(e) => updatePingTarget(index, 'port', parseInt(e.target.value) || 80)}
                      className="w-20 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
                      placeholder="Port"
                      min="1"
                      max="65535"
                    />
                  )}
                  <button
                    type="button"
                    onClick={() => removePingTarget(index)}
                    className="p-2 rounded-lg hover:bg-red-500/10 text-gray-500 hover:text-red-400 transition-colors"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              ))}
            </div>
          )}

          <div className="pt-4 border-t border-white/5 text-xs text-gray-500">
            <p className="mb-2">Common China carrier IPs for reference:</p>
            <div className="grid grid-cols-3 gap-2 font-mono text-gray-400">
              <span>CT: 202.97.1.1</span>
              <span>CU: 219.158.1.1</span>
              <span>CM: 223.120.2.1</span>
            </div>
          </div>

          <div className="flex justify-end pt-4">
            <button
              onClick={saveProbeSettings}
              disabled={probeSaving}
              className="px-4 py-2 rounded-lg bg-purple-500 hover:bg-purple-600 text-white text-sm font-medium transition-colors disabled:opacity-50"
            >
              {probeSaving ? 'Saving...' : 'Save Probe Settings'}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

