/**
 * Security Section
 * Manages password change functionality
 */

import { useState } from 'react';
import { useTranslation } from 'react-i18next';

export interface SecuritySectionProps {
  token: string | null;
}

export function SecuritySection({ token }: SecuritySectionProps) {
  const { t } = useTranslation();
  const [showPasswordForm, setShowPasswordForm] = useState(false);
  const [passwords, setPasswords] = useState({ current: '', new: '', confirm: '' });
  const [passwordError, setPasswordError] = useState('');
  const [passwordSuccess, setPasswordSuccess] = useState(false);

  const changePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordError('');
    setPasswordSuccess(false);

    if (passwords.new !== passwords.confirm) {
      setPasswordError(t('settings.passwordMismatch'));
      return;
    }

    if (passwords.new.length < 4) {
      setPasswordError(t('settings.passwordTooShort'));
      return;
    }

    try {
      const res = await fetch('/api/auth/password', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          current_password: passwords.current,
          new_password: passwords.new,
        }),
      });

      if (res.ok) {
        setPasswordSuccess(true);
        setPasswords({ current: '', new: '', confirm: '' });
        setShowPasswordForm(false);
      } else {
        setPasswordError(t('settings.currentPasswordIncorrect'));
      }
    } catch (e) {
      setPasswordError(t('settings.changePasswordFailed'));
    }
  };

  return (
    <div className="nezha-card p-6">
      <h2 className="text-lg font-bold text-white flex items-center gap-2 mb-6">
        <span className="w-2 h-2 rounded-full bg-purple-500"></span>
        {t('settings.security')}
      </h2>

      {passwordSuccess && (
        <div className="mb-4 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm">
          {t('settings.passwordChanged')}
        </div>
      )}

      {showPasswordForm ? (
        <form onSubmit={changePassword} className="space-y-4">
          <div>
            <label className="block text-xs text-gray-500 mb-1">{t('settings.currentPassword')}</label>
            <input
              type="password"
              value={passwords.current}
              onChange={(e) => setPasswords({ ...passwords, current: e.target.value })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
              required
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">{t('settings.newPassword')}</label>
            <input
              type="password"
              value={passwords.new}
              onChange={(e) => setPasswords({ ...passwords, new: e.target.value })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
              required
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">{t('settings.confirmPassword')}</label>
            <input
              type="password"
              value={passwords.confirm}
              onChange={(e) => setPasswords({ ...passwords, confirm: e.target.value })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-purple-500/50"
              required
            />
          </div>
          {passwordError && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
              {passwordError}
            </div>
          )}
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => {
                setShowPasswordForm(false);
                setPasswords({ current: '', new: '', confirm: '' });
                setPasswordError('');
              }}
              className="px-4 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-gray-400 text-sm transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              className="px-4 py-2 rounded-lg bg-purple-500 hover:bg-purple-600 text-white text-sm font-medium transition-colors"
            >
              {t('settings.changePassword')}
            </button>
          </div>
        </form>
      ) : (
        <button
          onClick={() => setShowPasswordForm(true)}
          className="px-4 py-2 rounded-lg bg-purple-500/10 hover:bg-purple-500/20 text-purple-400 text-sm font-medium transition-colors"
        >
          {t('settings.changePassword')}
        </button>
      )}
    </div>
  );
}

