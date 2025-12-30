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

export interface VpsGridCardProps {
  server: ServerState;
  onClick: () => void;
  isDark: boolean;
}

export function VpsGridCard({ server, onClick, isDark }: VpsGridCardProps) {
  const { t } = useTranslation();
  const { metrics, speed, isConnected, config } = server;
  const themeClass = isDark ? 'dark' : 'light';
  
  // Check if metrics data is complete (has all required fields)
  const hasCompleteMetrics = metrics && metrics.os && metrics.os.name && metrics.cpu && metrics.memory;
  
  const OsIcon = hasCompleteMetrics ? getOsIcon(metrics.os.name) : null;
  const distributionLogo = hasCompleteMetrics ? getDistributionLogo(metrics.os.name) : null;
  const providerLogo = config.provider ? getProviderLogo(config.provider) : null;
  const flag = getFlag(config.location);

  if (!hasCompleteMetrics) {
    // Server has never reported data - show offline placeholder with server info
    return (
      <div className={`vps-card vps-card--${themeClass} cursor-pointer relative`} onClick={onClick}>
        {/* Offline Banner */}
        <div className="absolute top-0 left-0 right-0 bg-red-500/90 text-white text-xs font-medium py-1 px-3 rounded-t-2xl flex items-center gap-1.5">
          <span className="w-2 h-2 rounded-full bg-white animate-pulse" />
          {t('dashboard.offline') || '离线'}
        </div>
        <div className="vps-card-header pt-6">
          <div className="vps-card-identity">
            <div className={`vps-card-avatar vps-card-avatar--${themeClass}`}>
              {providerLogo ? (
                <LogoImage src={providerLogo} alt={config.provider || ''} className="w-6 h-6 object-contain opacity-50" />
              ) : flag ? (
                <span className="text-lg opacity-50">{flag}</span>
              ) : (
                <span className="text-lg opacity-50">🖥️</span>
              )}
            </div>
            <div className="vps-card-info">
              <div className={`vps-card-title vps-card-title--${themeClass}`}>
                {config.name}
              </div>
              <div className="vps-card-meta">
                {flag && (
                  <span className={`vps-location vps-location--${themeClass}`}>
                    <span className="text-base">{flag}</span>
                    <span>{config.location}</span>
                  </span>
                )}
              </div>
            </div>
          </div>
          <span className={`vps-chip vps-chip--stopped-${themeClass}`}>
            <span className="vps-chip-dot vps-chip-dot--stopped" />
          </span>
        </div>
        <div className={`flex items-center justify-center py-8 text-sm ${isDark ? 'text-gray-500' : 'text-gray-400'}`}>
          {t('dashboard.noData') || '暂无数据'}
        </div>
      </div>
    );
  }

  const diskUsage = metrics.disks?.[0]?.usage_percent || 0;
  const totalDisk = (metrics.disks || []).reduce((acc, d) => acc + d.total, 0);
  const memoryModules = metrics.memory.modules;
  const memoryType = memoryModules?.[0]?.mem_type;
  const memorySpeed = memoryModules?.[0]?.speed;
  const memoryDetail = `${formatDiskSize(metrics.memory.total)}${memoryType ? ` · ${memoryType}` : ''}${memorySpeed ? `-${memorySpeed}MHz` : ''}`;
  const diskDetail = `${metrics.disks?.[0]?.disk_type || 'Storage'} · ${formatDiskSize(totalDisk)} total`;
  
  const networkMbps = ((speed.rx_sec + speed.tx_sec) * 8) / 1_000_000;
  const networkValue = Math.min(100, Math.round(networkMbps));
  const networkSubtitle = `↑ ${formatSpeed(speed.tx_sec)} · ↓ ${formatSpeed(speed.rx_sec)}`;

  const metricIcons: Record<string, ReactElement> = {
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

  return (
    <div className={`vps-card vps-card--${themeClass} group cursor-pointer relative`} onClick={onClick}>
      {/* Offline Banner - shown when server has metrics but is offline */}
      {!isConnected && (
        <div className="absolute top-0 left-0 right-0 bg-red-500/90 text-white text-xs font-medium py-1 px-3 rounded-t-2xl flex items-center gap-1.5 z-10">
          <span className="w-2 h-2 rounded-full bg-white animate-pulse" />
          {t('dashboard.offline') || '离线'}
        </div>
      )}
      
      {/* Tip Badge */}
      {tipBadgeClass && tipBadgeLabel && (
        <div className={`vps-tip-badge ${tipBadgeClass}`}>{tipBadgeLabel}</div>
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

      {/* Header */}
      <div className={`vps-card-header ${!isConnected ? 'pt-6' : ''}`}>
        <div className="vps-card-identity">
          <div className={`vps-card-avatar vps-card-avatar--${themeClass}`}>
            {distributionLogo ? (
              <LogoImage src={distributionLogo} alt={metrics.os.name} className="w-6 h-6 object-contain" />
            ) : OsIcon ? (
              <OsIcon className="w-5 h-5 text-blue-500" />
            ) : null}
          </div>
          <div className="vps-card-info">
            <div className={`vps-card-title vps-card-title--${themeClass}`}>
              {config.name}
            </div>
            <div className="vps-card-meta">
              {flag && (
                <span className={`vps-location vps-location--${themeClass}`}>
                  <span className="text-base">{flag}</span>
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
        <span className={`vps-chip ${isConnected ? `vps-chip--running-${themeClass}` : `vps-chip--stopped-${themeClass}`}`}>
          <span className={`vps-chip-dot ${isConnected ? 'vps-chip-dot--running' : 'vps-chip-dot--stopped'}`} />
        </span>
      </div>

      {/* Resource Metrics */}
      <div className="vps-resources">
        {metricRows.map(({ label, subtitle, value, thresholds }) => {
          const state = getResourceState(value, thresholds);
          return (
            <div key={label} className="vps-resource-row">
              <div className={`vps-resource-icon vps-resource-icon--${themeClass} vps-resource-icon--${state}`}>
                {metricIcons[label]}
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

      {/* Footer */}
      <div className={`vps-card-footer vps-card-footer--${themeClass}`}>
        {(config.price || config.purchase_date || effectiveExpiryDate) && (
          <div className="vps-footer-row-price">
            {config.price && (
              <div className={`vps-price vps-price--${themeClass}`}>
                <span className="vps-price-amount">{formatPrice(config.price.amount)}</span>
                <span className="vps-price-period">/{config.price.period === 'month' ? '月' : config.price.period === 'quarter' ? '季' : '年'}</span>
                {config.auto_renew && <span className="ml-1 text-xs text-emerald-400" title="自动续费">↻</span>}
              </div>
            )}
            {remainingValue && (
              <div className={`vps-footer-info-item vps-footer-info-item--${themeClass}`}>
                <span className="vps-footer-info-label">剩余</span>
                <span className="vps-footer-info-value">{remainingValue}</span>
              </div>
            )}
            {effectiveExpiryDate && (() => {
              const daysLeft = calculateDaysUntilExpiry(effectiveExpiryDate);
              const statusClass = getExpiryStatusClass(daysLeft);
              return (
                <div className={`vps-footer-info-item vps-footer-info-item--${themeClass}`}>
                  <span className="vps-footer-info-label">到期</span>
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
            <span className="vps-uptime-label">运行</span>
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

