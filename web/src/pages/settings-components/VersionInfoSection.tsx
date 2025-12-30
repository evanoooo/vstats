/**
 * Version Info Section
 * Displays current version and handles upgrade functionality
 */

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { showToast } from '../../components/Toast';

export interface VersionInfoSectionProps {
  token: string | null;
}

export function VersionInfoSection({ token }: VersionInfoSectionProps) {
  const { t } = useTranslation();
  const [serverVersion, setServerVersion] = useState<string>('');
  const [latestVersion, setLatestVersion] = useState<string | null>(null);
  const [updateAvailable, setUpdateAvailable] = useState(false);
  const [checkingVersion, setCheckingVersion] = useState(false);
  const [upgrading, setUpgrading] = useState(false);

  useEffect(() => {
    fetchServerVersion();
    checkLatestVersion();
  }, []);

  const fetchServerVersion = async () => {
    try {
      const res = await fetch('/api/version');
      if (res.ok) {
        const data = await res.json();
        setServerVersion(data.version);
      }
    } catch (e) {
      console.error('Failed to fetch server version', e);
    }
  };

  const checkLatestVersion = async () => {
    setCheckingVersion(true);
    try {
      const res = await fetch('/api/version/check');
      if (res.ok) {
        const data = await res.json();
        setLatestVersion(data.latest);
        setUpdateAvailable(data.update_available);
      }
    } catch (e) {
      console.error('Failed to check latest version', e);
    } finally {
      setCheckingVersion(false);
    }
  };

  const upgradeServer = async (force: boolean = false) => {
    const message = force
      ? 'Are you sure you want to force reinstall the server? This will restart the service.'
      : 'Are you sure you want to upgrade the server? This will restart the service.';

    if (!confirm(message)) {
      return;
    }

    setUpgrading(true);

    try {
      const res = await fetch('/api/server/upgrade', {
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
          const successMsg = force
            ? 'Force reinstall executed successfully! The server will restart.'
            : 'Upgrade command executed successfully! The server will restart.';
          showToast(successMsg, 'success');
          // Refresh version after a delay
          setTimeout(() => {
            fetchServerVersion();
            checkLatestVersion();
          }, 3000);
        } else {
          showToast(`Upgrade failed: ${data.message}`, 'error');
        }
      } else {
        showToast('Failed to execute upgrade command', 'error');
      }
    } catch (e) {
      console.error('Failed to upgrade server', e);
      showToast('Failed to execute upgrade command', 'error');
    } finally {
      setUpgrading(false);
    }
  };

  return (
    <div className="nezha-card p-6 mb-6">
      <h2 className="text-lg font-bold text-white flex items-center gap-2 mb-4">
        <span className="w-2 h-2 rounded-full bg-cyan-500"></span>
        {t('settings.versionInfo')}
      </h2>

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div>
            <div className="text-sm text-gray-400">{t('settings.currentVersion')}</div>
            <div className="text-lg font-mono text-white">{serverVersion || t('common.loading')}</div>
          </div>
          <button
            onClick={checkLatestVersion}
            disabled={checkingVersion}
            className="px-4 py-2 rounded-lg bg-cyan-500/10 hover:bg-cyan-500/20 text-cyan-400 text-sm font-medium transition-colors disabled:opacity-50"
          >
            {checkingVersion ? t('settings.checkingUpdate') : t('settings.checkUpdate')}
          </button>
        </div>

        {latestVersion && (
          <div className="pt-3 border-t border-white/5">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-400">{t('settings.latestVersion')}</div>
                <div className="text-lg font-mono text-white">{latestVersion}</div>
              </div>
              {updateAvailable ? (
                <div className="flex items-center gap-2">
                  <span className="px-3 py-1 rounded-lg bg-emerald-500/10 text-emerald-400 text-xs font-medium">
                    {t('settings.updateAvailable')}
                  </span>
                  <button
                    onClick={() => upgradeServer(false)}
                    disabled={upgrading}
                    className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {upgrading ? t('settings.upgrading') : t('settings.executeUpgrade')}
                  </button>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <span className="px-3 py-1 rounded-lg bg-gray-500/10 text-gray-400 text-xs font-medium">
                    {t('settings.upToDate')}
                  </span>
                  <button
                    onClick={() => upgradeServer(true)}
                    disabled={upgrading}
                    className="px-4 py-2 rounded-lg bg-orange-500/10 hover:bg-orange-500/20 text-orange-400 text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    title={t('settings.forceReinstallTooltip')}
                  >
                    {upgrading ? t('settings.reinstalling') : t('settings.forceReinstall')}
                  </button>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

