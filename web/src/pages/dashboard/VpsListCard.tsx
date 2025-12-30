import type { ReactElement } from 'react';
import { useTranslation } from 'react-i18next';
import { formatUptime, type ServerState } from '../../hooks/useMetrics';
import { getOsIcon } from '../../components/Icons';
import { getProviderLogo, getDistributionLogo, LogoImage } from '../../utils/logoUtils';
import {
  getFlag,
  formatPrice,
  formatLatency,
  calculateExpiryFromPurchase,
  calculateDaysUntilExpiry,
  getExpiryStatusClass,
  formatExpiryDisplay,
  calculateRemainingValue,
  getShortCpuBrand,
  formatDiskSize,
  getResourceState,
  formatSpeed,
  getTipBadgeClass,
  getTipBadgeLabel,
  getLabelColorClasses,
} from './utils';

export interface VpsListCardProps {
  server: ServerState;
  onClick: () => void;
  isDark: boolean;
}

export function VpsListCard({ server, onClick, isDark }: VpsListCardProps) {
  const { t } = useTranslation();
  const { metrics, speed, isConnected, config } = server;
  const themeClass = isDark ? 'dark' : 'light';
  
  // Check if metrics data is complete (has all required fields)
  const hasCompleteMetrics = metrics && metrics.os && metrics.os.name && metrics.cpu && metrics.memory;
  
  const OsIcon = hasCompleteMetrics ? getOsIcon(metrics.os.name) : null;
  const providerLogo = config.provider ? getProviderLogo(config.provider) : null;
  const distributionLogo = hasCompleteMetrics ? getDistributionLogo(metrics.os.name) : null;
  const flag = getFlag(config.location);

  if (!hasCompleteMetrics) {
    // Server has never reported data - show offline placeholder with server info
    return (
      <div className={`vps-list-card vps-list-card--${themeClass} cursor-pointer relative`} onClick={onClick}>
        {/* Offline indicator */}
        <div className="absolute left-0 top-0 bottom-0 w-1 bg-red-500 rounded-l-2xl" />
        <div className={`vps-card-avatar vps-card-avatar--${themeClass}`}>
          {providerLogo ? (
            <LogoImage src={providerLogo} alt={config.provider || ''} className="w-6 h-6 object-contain opacity-50" />
          ) : flag ? (
            <span className="text-lg opacity-50">{flag}</span>
          ) : (
            <span className="text-lg opacity-50">🖥️</span>
          )}
        </div>
        <div className="flex-1">
          <div className={`font-medium ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>{config.name}</div>
          <div className={`text-xs ${isDark ? 'text-gray-500' : 'text-gray-400'}`}>
            {flag && <span className="mr-1">{flag}</span>}
            {config.location}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span className="vps-chip-dot vps-chip-dot--stopped" />
          <span className={`text-sm font-medium text-red-500`}>{t('dashboard.offline') || '离线'}</span>
        </div>
      </div>
    );
  }

  const totalDisk = (metrics.disks || []).reduce((acc, d) => acc + d.total, 0);

  // Tip badge: use config.tip_badge if set, otherwise infer from tag
  const tipBadgeClass = config.tip_badge || getTipBadgeClass(config.tag);
  const tipBadgeLabel = config.tip_badge 
    ? t(`dashboard.tipBadge.${config.tip_badge}`)
    : getTipBadgeLabel(config.tag, t);
  const pingMetrics = metrics.ping;
  const remainingValue = calculateRemainingValue(config.price, config.purchase_date);
  
  // Calculate expiry date from purchase date and billing period if not explicitly set
  const effectiveExpiryDate = config.expiry_date || 
    calculateExpiryFromPurchase(config.purchase_date, config.price?.period);
  
  // Calculate metrics details (same as Grid card)
  const diskUsage = metrics.disks?.[0]?.usage_percent || 0;
  const diskDetail = `${metrics.disks?.[0]?.disk_type || 'SSD'} · ${formatDiskSize(totalDisk)} total`;
  const memoryDetail = `${formatDiskSize(metrics.memory.total)}`;
  const networkValue = Math.min(100, Math.round(((speed.rx_sec + speed.tx_sec) * 8) / 1_000_000));
  const networkSubtitle = `↑ ${formatSpeed(speed.tx_sec)} · ↓ ${formatSpeed(speed.rx_sec)}`;

  const listMetricIcons: Record<string, ReactElement> = {
    CPU: (
      <svg viewBox="0 0 24 24" stroke="currentColor" fill="none" className="w-3 h-3">
        <path strokeWidth={1.6} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9z" />
      </svg>
    ),
    RAM: (
      <svg viewBox="0 0 24 24" stroke="currentColor" fill="none" className="w-3 h-3">
        <path strokeWidth={1.6} d="M3 7a2 2 0 012-2h14a2 2 0 012 2v9H3z" />
        <path strokeWidth={1.6} d="M6 18v2m4-2v2m4-2v2m4-2v2M7 7v5m10-5v5" />
      </svg>
    ),
    Disk: (
      <svg viewBox="0 0 24 24" stroke="currentColor" fill="none" className="w-3 h-3">
        <path strokeWidth={1.6} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7c0 2.21-3.582 4-8 4S4 9.21 4 7z" />
        <path strokeWidth={1.6} d="M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
      </svg>
    ),
    Network: (
      <svg viewBox="0 0 24 24" stroke="currentColor" fill="none" className="w-3 h-3">
        <path strokeWidth={1.6} d="M5 12h14M12 5l7 7-7 7" />
      </svg>
    ),
  };

  const metricRows = [
    { label: 'CPU', subtitle: `${getShortCpuBrand(metrics.cpu.brand)} · ${metrics.cpu.cores} cores`, value: metrics.cpu.usage, thresholds: [50, 80] as [number, number] },
    { label: 'RAM', subtitle: memoryDetail, value: metrics.memory.usage_percent, thresholds: [50, 80] as [number, number] },
    { label: 'Disk', subtitle: diskDetail, value: diskUsage, thresholds: [70, 90] as [number, number] },
    { label: 'Network', subtitle: networkSubtitle, value: networkValue, thresholds: [40, 70] as [number, number] },
  ];

  return (
    <div 
      className={`vps-list-card vps-list-card--${themeClass} cursor-pointer group relative overflow-hidden`}
      onClick={onClick}
    >
      {/* Offline indicator - red left border */}
      {!isConnected && (
        <div className="absolute left-0 top-0 bottom-0 w-1 bg-red-500 rounded-l-2xl" />
      )}
      
      {/* List Tip Badge */}
      {tipBadgeClass && tipBadgeLabel && (
        <div className={`vps-list-tip-badge ${tipBadgeClass}`}>{tipBadgeLabel}</div>
      )}

      {/* Sale/Rent Badge */}
      {config.sale_status && config.sale_contact_url && (
        <a
          href={config.sale_contact_url}
          target="_blank"
          rel="noopener noreferrer"
          onClick={(e) => e.stopPropagation()}
          className={`absolute ${tipBadgeClass && tipBadgeLabel ? 'top-9' : 'top-2'} right-2 z-10 px-2 py-1 rounded-lg text-xs font-semibold transition-all hover:scale-105 ${
            config.sale_status === 'rent' 
              ? 'bg-emerald-500/90 text-white hover:bg-emerald-600' 
              : 'bg-amber-500/90 text-white hover:bg-amber-600'
          }`}
          title={config.sale_status === 'rent' ? '点击联系合租' : '点击联系购买'}
        >
          {config.sale_status === 'rent' ? '🏠 招租' : '💰 出售'}
        </a>
      )}

      {/* Column 1: Identity */}
      <div className="vps-list-identity">
        <div className={`vps-card-avatar vps-card-avatar--${themeClass}`}>
          {distributionLogo ? (
            <LogoImage src={distributionLogo} alt={metrics.os.name} className="w-6 h-6 object-contain" />
          ) : OsIcon ? (
            <OsIcon className="w-5 h-5 text-blue-500" />
          ) : null}
        </div>
        <div className="vps-list-info">
          <div className={`vps-list-title vps-list-title--${themeClass}`}>
            {config.name}
            <span className={`vps-chip-dot ${isConnected ? 'vps-chip-dot--running' : 'vps-chip-dot--stopped'}`} />
            {!isConnected && (
              <span className="ml-1 text-xs font-medium text-red-500">{t('dashboard.offline') || '离线'}</span>
            )}
          </div>
          <div className="vps-list-meta">
            {flag && (
              <span className={`vps-location vps-location--${themeClass}`}>
                <span className="text-xs">{flag}</span>
                <span>{config.location}</span>
              </span>
            )}
            {providerLogo && (
              <span className={`vps-provider-logo vps-provider-logo--${themeClass}`}>
                <LogoImage src={providerLogo} alt={config.provider || ''} className="w-4 h-4 object-contain" />
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Column 2: Resources (same style as Grid card) */}
      <div className="vps-list-specs">
        <div className="vps-list-resources">
          {metricRows.map(({ label, subtitle, value, thresholds }) => {
            const state = getResourceState(value, thresholds);
            return (
              <div key={label} className="vps-resource-row">
                <div className={`vps-resource-icon vps-resource-icon--${themeClass} vps-resource-icon--${state}`}>
                  {listMetricIcons[label]}
                </div>
                <div className="vps-resource-content">
                  <div className="vps-resource-info">
                    <div className="vps-resource-title-row">
                      <span className={`vps-resource-label vps-resource-label--${themeClass}`}>{label.toUpperCase()}</span>
                      <span className={`vps-resource-detail vps-resource-detail--${themeClass}`}>{subtitle}</span>
                    </div>
                    <span className={`vps-resource-percent vps-resource-percent--${state}-${themeClass}`}>
                      {Math.round(value)}%
                    </span>
                  </div>
                  <div className={`vps-resource-bar-track vps-resource-bar-track--${themeClass}`}>
                    <div 
                      className={`vps-resource-bar-fill vps-resource-bar-fill--${state}-${themeClass}`}
                      style={{ width: `${Math.min(100, Math.max(0, value))}%` }}
                    />
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Column 3: Footer */}
      <div className={`vps-list-footer vps-list-footer--${themeClass}`}>
        {(config.price || config.purchase_date || effectiveExpiryDate) && (
          <div className="vps-footer-row-price">
            {config.price && (
              <div className={`vps-price vps-price--${themeClass}`}>
                <span className="vps-price-amount">{formatPrice(config.price.amount)}</span>
                <span className="vps-price-period">{config.price.period === 'month' ? t('dashboard.perMonth') : config.price.period === 'quarter' ? t('dashboard.perQuarter') : t('dashboard.perYear')}</span>
                {config.auto_renew && <span className="ml-1 text-xs text-emerald-400" title={t('dashboard.autoRenew') || '自动续费'}>↻</span>}
              </div>
            )}
            {remainingValue && (
              <div className={`vps-footer-info-item vps-footer-info-item--${themeClass}`}>
                <span className="vps-footer-info-label">{t('dashboard.remaining')}</span>
                <span className="vps-footer-info-value">{remainingValue}</span>
              </div>
            )}
            {effectiveExpiryDate && (() => {
              const daysLeft = calculateDaysUntilExpiry(effectiveExpiryDate);
              const statusClass = getExpiryStatusClass(daysLeft);
              return (
                <div className={`vps-footer-info-item vps-footer-info-item--${themeClass}`}>
                  <span className="vps-footer-info-label">{t('dashboard.expiry') || '到期'}</span>
                  <span className={`vps-footer-info-value ${statusClass}`}>
                    {formatExpiryDisplay(daysLeft, config.auto_renew)}
                  </span>
                </div>
              );
            })()}
          </div>
        )}
        {/* Labels */}
        {config.labels && config.labels.length > 0 && (
          <div className="vps-footer-row-labels flex flex-wrap gap-1 mb-2">
            {config.labels.map((label, idx) => {
              const colorClasses = getLabelColorClasses(label.color);
              return (
                <span
                  key={idx}
                  className={`inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium ${colorClasses.bg} ${colorClasses.text} border ${colorClasses.border}`}
                >
                  {label.name}
                </span>
              );
            })}
          </div>
        )}
        <div className="vps-footer-row-status">
          <div className={`vps-uptime-item vps-uptime-item--${themeClass}`}>
            <span className="vps-uptime-label">{t('dashboard.running')}</span>
            <span className="vps-uptime-value">{formatUptime(metrics.uptime)}</span>
          </div>
          {pingMetrics && pingMetrics.targets && pingMetrics.targets.length > 0 && (
            <div className="vps-latency">
              {pingMetrics.targets.slice(0, 3).map((target, idx) => (
                <div key={idx} className={`vps-latency-item vps-latency-item--${themeClass}`}>
                  <span className="vps-latency-label">{target.name}</span>
                  <span className={`vps-latency-value vps-latency-value--${themeClass}`}>
                    {formatLatency(target.latency_ms)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

