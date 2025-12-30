/**
 * Settings Sidebar Component
 * Navigation sidebar for settings page
 */

import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

export type SettingsSection =
  | 'site-settings'
  | 'probe-settings'
  | 'oauth-settings'
  | 'group-dimensions'
  | 'server-management'
  | 'theme-settings'
  | 'alerts'
  | 'aff-settings'
  | 'security'
  | 'audit-logs';

export interface SettingsSidebarProps {
  activeSection: SettingsSection;
  onSectionChange: (section: SettingsSection) => void;
  onLogout: () => void;
  isZh: boolean;
}

interface NavItem {
  id: SettingsSection;
  color: string;
  labelEn: string;
  labelZh: string;
}

const NAV_ITEMS: NavItem[] = [
  { id: 'site-settings', color: 'blue', labelEn: 'Site Settings', labelZh: '站点设置' },
  { id: 'probe-settings', color: 'purple', labelEn: 'Probe Settings', labelZh: '探测设置' },
  { id: 'oauth-settings', color: 'orange', labelEn: 'OAuth 2.0 Login', labelZh: 'OAuth 2.0 登录' },
  { id: 'group-dimensions', color: 'orange', labelEn: 'Group Dimensions', labelZh: '分组维度' },
  { id: 'server-management', color: 'emerald', labelEn: 'Server Management', labelZh: '管理服务器' },
  { id: 'theme-settings', color: 'pink', labelEn: 'Theme Settings', labelZh: '主题设置' },
  { id: 'alerts', color: 'red', labelEn: 'Alerts', labelZh: '告警通知' },
  { id: 'aff-settings', color: 'amber', labelEn: 'Affiliate Links', labelZh: '推广链接' },
  { id: 'security', color: 'purple', labelEn: 'Security', labelZh: '安全' },
  { id: 'audit-logs', color: 'slate', labelEn: 'Audit Logs', labelZh: '审计日志' },
];

const getColorClasses = (color: string, _isActive: boolean) => {
  const colors: Record<string, { active: string; inactive: string; dot: string }> = {
    blue: {
      active: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
      inactive: 'bg-white/5 text-gray-300 border-white/10 hover:bg-white/10 hover:border-white/20',
      dot: 'bg-blue-500',
    },
    purple: {
      active: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
      inactive: 'bg-white/5 text-gray-300 border-white/10 hover:bg-white/10 hover:border-white/20',
      dot: 'bg-purple-500',
    },
    orange: {
      active: 'bg-orange-500/20 text-orange-400 border-orange-500/30',
      inactive: 'bg-white/5 text-gray-300 border-white/10 hover:bg-white/10 hover:border-white/20',
      dot: 'bg-orange-500',
    },
    emerald: {
      active: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30',
      inactive: 'bg-white/5 text-gray-300 border-white/10 hover:bg-white/10 hover:border-white/20',
      dot: 'bg-emerald-500',
    },
    pink: {
      active: 'bg-pink-500/20 text-pink-400 border-pink-500/30',
      inactive: 'bg-white/5 text-gray-300 border-white/10 hover:bg-white/10 hover:border-white/20',
      dot: 'bg-pink-500',
    },
    red: {
      active: 'bg-red-500/20 text-red-400 border-red-500/30',
      inactive: 'bg-white/5 text-gray-300 border-white/10 hover:bg-white/10 hover:border-white/20',
      dot: 'bg-red-500',
    },
    amber: {
      active: 'bg-amber-500/20 text-amber-400 border-amber-500/30',
      inactive: 'bg-white/5 text-gray-300 border-white/10 hover:bg-white/10 hover:border-white/20',
      dot: 'bg-amber-500',
    },
    slate: {
      active: 'bg-slate-500/20 text-slate-400 border-slate-500/30',
      inactive: 'bg-white/5 text-gray-300 border-white/10 hover:bg-white/10 hover:border-white/20',
      dot: 'bg-slate-500',
    },
  };
  return colors[color] || colors.blue;
};

export function SettingsSidebar({ activeSection, onSectionChange, onLogout, isZh }: SettingsSidebarProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <aside className="settings-sidebar hidden lg:flex flex-col w-72 shrink-0 nezha-card self-start sticky top-10">
      <div className="p-6 border-b border-white/10">
        <div className="flex items-center gap-3 mb-4">
          <button
            onClick={() => navigate('/')}
            className="p-2 rounded-lg hover:bg-white/10 text-gray-400 hover:text-white transition-colors"
            title={t('settings.backToDashboard')}
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <div>
            <h1 className="text-xl font-bold text-white">{t('settings.title')}</h1>
            <p className="text-gray-500 text-xs">{t('settings.serverManagement')}</p>
          </div>
        </div>
        <button
          onClick={onLogout}
          className="w-full px-4 py-2 rounded-lg bg-red-500/10 hover:bg-red-500/20 text-red-400 text-sm font-medium transition-colors"
        >
          {t('settings.logout')}
        </button>
      </div>

      <nav className="flex-1 p-4 space-y-2 overflow-y-auto">
        {NAV_ITEMS.map((item) => {
          const isActive = activeSection === item.id;
          const colorClasses = getColorClasses(item.color, isActive);

          return (
            <button
              key={item.id}
              onClick={() => onSectionChange(item.id)}
              className={`w-full text-left p-4 rounded-xl transition-all border ${
                isActive ? `${colorClasses.active} shadow-lg` : colorClasses.inactive
              }`}
            >
              <div className="flex items-center gap-3 mb-1">
                <span className={`w-2 h-2 rounded-full ${colorClasses.dot}`}></span>
                <span className="text-sm font-bold">{isZh ? item.labelZh : item.labelEn}</span>
              </div>
            </button>
          );
        })}
      </nav>
    </aside>
  );
}

export function MobileHeader({ onLogout }: { onLogout: () => void }) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <div className="lg:hidden flex items-center justify-between mb-8">
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate('/')}
          className="p-2 rounded-lg hover:bg-white/10 text-gray-400 hover:text-white transition-colors"
          title={t('settings.backToDashboard')}
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <div>
          <h1 className="text-2xl font-bold text-white">{t('settings.title')}</h1>
          <p className="text-gray-500 text-sm">{t('settings.serverManagement')}</p>
        </div>
      </div>
      <button
        onClick={onLogout}
        className="px-4 py-2 rounded-lg bg-red-500/10 hover:bg-red-500/20 text-red-400 text-sm font-medium transition-colors"
      >
        {t('settings.logout')}
      </button>
    </div>
  );
}

