/**
 * Settings Page
 * Main settings page that orchestrates all settings sections
 */

import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';
import AlertSettings from '../components/AlertSettings';
import type { SiteSettings } from '../types';
import { sanitizeSiteSettings } from '../utils/security';

// Import from settings modules
import {
  ThemeSettingsSection,
  AuditLogsSection,
  SiteSettingsSection,
  ProbeSettingsSection,
  OAuthSettingsSection,
  DimensionsSection,
  ServerManagementSection,
  SecuritySection,
  VersionInfoSection,
  AffSettingsSection,
  SettingsSidebar,
  MobileHeader,
  type SettingsSection,
  type ProbeSettings,
  type RemoteServer,
} from './settings-components';

export default function Settings() {
  const { i18n } = useTranslation();
  const { isAuthenticated, token, logout, isLoading: authLoading } = useAuth();
  const navigate = useNavigate();
  const isZh = i18n.language.startsWith('zh');

  // Active section state
  const [activeSection, setActiveSection] = useState<SettingsSection>('site-settings');

  // Loading state
  const [loading, setLoading] = useState(true);

  // Site settings (shared between sections)
  const [siteSettings, setSiteSettings] = useState<SiteSettings>({
    site_name: '',
    site_description: '',
    social_links: [],
  });

  // Probe settings
  const [probeSettings, setProbeSettings] = useState<ProbeSettings>({ ping_targets: [] });

  // Servers list (for alerts and dimensions)
  const [servers, setServers] = useState<RemoteServer[]>([]);

  useEffect(() => {
    if (authLoading) return;

    if (!isAuthenticated) {
      navigate('/login', { replace: true });
      return;
    }

    // Fetch initial data
    Promise.all([
      fetchSiteSettings(),
      fetchProbeSettings(),
      fetchServers(),
    ]).finally(() => setLoading(false));
  }, [isAuthenticated, authLoading, navigate, token]);

  const fetchSiteSettings = async () => {
    try {
      const res = await fetch('/api/settings/site');
      if (res.ok) {
        const data = await res.json();
        setSiteSettings(sanitizeSiteSettings(data));
      }
    } catch (e) {
      console.error('Failed to fetch site settings', e);
    }
  };

  const fetchProbeSettings = async () => {
    try {
      const res = await fetch('/api/settings/probe', {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setProbeSettings(data);
      }
    } catch (e) {
      console.error('Failed to fetch probe settings', e);
    }
  };

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

  if (authLoading || loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="w-8 h-8 border-4 border-white/20 border-t-white rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <div className="min-h-screen flex justify-center p-4 md:p-6 lg:p-10">
      <div className="w-full max-w-7xl flex gap-6">
        {/* Fixed Sidebar */}
        <SettingsSidebar
          activeSection={activeSection}
          onSectionChange={setActiveSection}
          onLogout={logout}
          isZh={isZh}
        />

        {/* Main Content Area */}
        <div className="flex-1 min-w-0">
          <div className="">
            {/* Mobile Header */}
            <MobileHeader onLogout={logout} />

            {/* Site Settings Section */}
            {activeSection === 'site-settings' && (
              <>
                <SiteSettingsSection
                  token={token}
                  siteSettings={siteSettings}
                  onSiteSettingsChange={setSiteSettings}
                />
                <VersionInfoSection token={token} />
              </>
            )}

            {/* Probe Settings Section */}
            {activeSection === 'probe-settings' && (
              <ProbeSettingsSection
                token={token}
                probeSettings={probeSettings}
                onProbeSettingsChange={setProbeSettings}
              />
            )}

            {/* OAuth Settings Section */}
            {activeSection === 'oauth-settings' && (
              <OAuthSettingsSection token={token} isZh={isZh} />
            )}

            {/* Group Dimensions Section */}
            {activeSection === 'group-dimensions' && (
              <DimensionsSection token={token} servers={servers} />
            )}

            {/* Server Management Section */}
            {activeSection === 'server-management' && (
              <ServerManagementSection token={token} isZh={isZh} />
            )}

            {/* Theme Settings Section */}
            {activeSection === 'theme-settings' && (
              <div className="mb-6">
                <ThemeSettingsSection
                  isAuthenticated={isAuthenticated}
                  token={token}
                  siteSettings={siteSettings}
                  onSiteSettingsChange={setSiteSettings}
                />
              </div>
            )}

            {/* Alerts Section */}
            {activeSection === 'alerts' && (
              <div className="nezha-card p-6">
                <AlertSettings
                  token={token}
                  servers={servers.map((s) => ({ id: s.id, name: s.name }))}
                />
              </div>
            )}

            {/* Affiliate Links Section */}
            {activeSection === 'aff-settings' && (
              <AffSettingsSection token={token} isZh={isZh} />
            )}

            {/* Security Section */}
            {activeSection === 'security' && <SecuritySection token={token} />}

            {/* Audit Logs Section */}
            {activeSection === 'audit-logs' && (
              <AuditLogsSection token={token} isZh={isZh} />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
