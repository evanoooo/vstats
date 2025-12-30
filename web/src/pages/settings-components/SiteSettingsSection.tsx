/**
 * Site Settings Section
 * Manages site name, description, and social links
 */

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { SiteSettings, SocialLink } from '../../types';
import { sanitizeSiteSettings } from '../../utils/security';
import { PLATFORM_OPTIONS } from './constants';

export interface SiteSettingsSectionProps {
  token: string | null;
  siteSettings: SiteSettings;
  onSiteSettingsChange: (settings: SiteSettings) => void;
}

export function SiteSettingsSection({
  token,
  siteSettings,
  onSiteSettingsChange,
}: SiteSettingsSectionProps) {
  const { t } = useTranslation();
  const [showSiteSettings, setShowSiteSettings] = useState(false);
  const [siteSettingsSaving, setSiteSettingsSaving] = useState(false);
  const [siteSettingsSuccess, setSiteSettingsSuccess] = useState(false);

  const saveSiteSettings = async () => {
    setSiteSettingsSaving(true);
    setSiteSettingsSuccess(false);

    const sanitizedSettings = sanitizeSiteSettings(siteSettings);

    try {
      const res = await fetch('/api/settings/site', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify(sanitizedSettings),
      });

      if (res.ok) {
        onSiteSettingsChange(sanitizedSettings);
        setSiteSettingsSuccess(true);
        setTimeout(() => setSiteSettingsSuccess(false), 3000);
      }
    } catch (e) {
      console.error('Failed to save site settings', e);
    }

    setSiteSettingsSaving(false);
  };

  const addSocialLink = () => {
    onSiteSettingsChange({
      ...siteSettings,
      social_links: [...siteSettings.social_links, { platform: 'github', url: '', label: '' }],
    });
  };

  const removeSocialLink = (index: number) => {
    onSiteSettingsChange({
      ...siteSettings,
      social_links: siteSettings.social_links.filter((_, i) => i !== index),
    });
  };

  const updateSocialLink = (index: number, field: keyof SocialLink, value: string) => {
    const updated = [...siteSettings.social_links];
    updated[index] = { ...updated[index], [field]: value };
    onSiteSettingsChange({ ...siteSettings, social_links: updated });
  };

  return (
    <div className="nezha-card p-6 mb-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-bold text-white flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-blue-500"></span>
          {t('settings.siteSettings')}
        </h2>
        <button
          onClick={() => setShowSiteSettings(!showSiteSettings)}
          className="px-4 py-2 rounded-lg bg-blue-500/10 hover:bg-blue-500/20 text-blue-400 text-sm font-medium transition-colors"
        >
          {showSiteSettings ? t('common.close') : t('common.edit')}
        </button>
      </div>

      {siteSettingsSuccess && (
        <div className="mb-4 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm">
          {t('settings.saved')}
        </div>
      )}

      {showSiteSettings && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs text-gray-500 mb-1">{t('settings.siteName')}</label>
              <input
                type="text"
                value={siteSettings.site_name}
                onChange={(e) => onSiteSettingsChange({ ...siteSettings, site_name: e.target.value })}
                className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
                placeholder="vStats Dashboard"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">{t('settings.siteDescription')}</label>
              <input
                type="text"
                value={siteSettings.site_description}
                onChange={(e) => onSiteSettingsChange({ ...siteSettings, site_description: e.target.value })}
                className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
                placeholder="Real-time Server Monitoring"
              />
            </div>
          </div>

          {/* Social Links */}
          <div className="pt-4 border-t border-white/5">
            <div className="flex items-center justify-between mb-3">
              <label className="text-xs text-gray-500 uppercase tracking-wider">{t('settings.socialLinks')}</label>
              <button
                type="button"
                onClick={addSocialLink}
                className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
              >
                + {t('settings.addSocialLink')}
              </button>
            </div>

            {siteSettings.social_links.length === 0 ? (
              <p className="text-gray-600 text-sm text-center py-4">{t('common.none')}</p>
            ) : (
              <div className="space-y-3">
                {siteSettings.social_links.map((link, index) => (
                  <div key={index} className="flex items-center gap-2">
                    <select
                      value={link.platform}
                      onChange={(e) => updateSocialLink(index, 'platform', e.target.value)}
                      className="px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
                    >
                      {PLATFORM_OPTIONS.map((opt) => (
                        <option key={opt.value} value={opt.value}>
                          {opt.label}
                        </option>
                      ))}
                    </select>
                    <input
                      type="text"
                      value={link.url}
                      onChange={(e) => updateSocialLink(index, 'url', e.target.value)}
                      className="flex-1 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
                      placeholder="https://..."
                    />
                    <input
                      type="text"
                      value={link.label || ''}
                      onChange={(e) => updateSocialLink(index, 'label', e.target.value)}
                      className="w-24 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
                      placeholder="Label"
                    />
                    <button
                      type="button"
                      onClick={() => removeSocialLink(index)}
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
          </div>

          <div className="flex justify-end pt-4">
            <button
              onClick={saveSiteSettings}
              disabled={siteSettingsSaving}
              className="px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-sm font-medium transition-colors disabled:opacity-50"
            >
              {siteSettingsSaving ? t('common.loading') : t('common.save')}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

