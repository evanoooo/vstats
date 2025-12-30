import { useTranslation } from 'react-i18next';
import { type ServerState } from '../../hooks/useMetrics';
import { getDistributionLogo, LogoImage } from '../../utils/logoUtils';
import { getFlag, formatSpeed, getShortOsName } from './utils';

export interface VpsDualCardProps {
  server: ServerState;
  onClick: () => void;
  isDark: boolean;
}

export function VpsDualCard({ server, onClick, isDark }: VpsDualCardProps) {
  const { t } = useTranslation();
  const { metrics, speed, isConnected, config } = server;
  const themeClass = isDark ? 'dark' : 'light';
  
  const hasCompleteMetrics = metrics && metrics.os && metrics.os.name && metrics.cpu && metrics.memory;
  const flag = getFlag(config.location);
  const distributionLogo = hasCompleteMetrics ? getDistributionLogo(metrics.os.name) : null;

  const getBarColor = (value: number, thresholds: [number, number]) => {
    if (value > thresholds[1]) return isDark ? '#ef4444' : '#dc2626';
    if (value > thresholds[0]) return isDark ? '#f59e0b' : '#d97706';
    return isDark ? '#22c55e' : '#16a34a';
  };

  if (!hasCompleteMetrics) {
    return (
      <div 
        className={`vps-dual-card vps-dual-card--${themeClass} cursor-pointer`}
        onClick={onClick}
      >
        <div className="vps-dual-header">
          <span className="vps-dual-status vps-dual-status--offline" />
          {flag && <span className="vps-dual-flag">{flag}</span>}
          <span className={`vps-dual-name vps-dual-name--${themeClass}`}>{config.name}</span>
        </div>
        <div className={`vps-dual-offline ${isDark ? 'text-gray-500' : 'text-gray-400'}`}>
          {t('dashboard.offline')}
        </div>
      </div>
    );
  }

  const diskUsage = metrics.disks?.[0]?.usage_percent || 0;
  const totalTxTraffic = metrics.network?.total_tx || 0;
  const totalRxTraffic = metrics.network?.total_rx || 0;

  // Format traffic in GiB
  const formatTrafficGiB = (bytes: number): string => {
    const gib = bytes / (1024 * 1024 * 1024);
    return `${gib.toFixed(2)} GiB`;
  };

  return (
    <div 
      className={`vps-dual-card vps-dual-card--${themeClass} cursor-pointer`}
      onClick={onClick}
    >
      {/* Header */}
      <div className="vps-dual-header">
        <span className={`vps-dual-status ${isConnected ? 'vps-dual-status--online' : 'vps-dual-status--offline'}`} />
        {flag && <span className="vps-dual-flag">{flag}</span>}
        <span className={`vps-dual-name vps-dual-name--${themeClass}`}>{config.name}</span>
      </div>

      {/* Metrics Row */}
      <div className="vps-dual-metrics">
        {/* OS */}
        <div className="vps-dual-metric">
          <div className={`vps-dual-metric-label vps-dual-metric-label--${themeClass}`}>
            {distributionLogo ? (
              <LogoImage src={distributionLogo} alt={metrics.os.name} className="w-4 h-4 object-contain" />
            ) : null}
            <span>{t('dashboard.system') || '系统'}</span>
          </div>
          <div className={`vps-dual-metric-value vps-dual-metric-value--${themeClass}`}>
            {getShortOsName(metrics.os.name)}
          </div>
        </div>

        {/* CPU */}
        <div className="vps-dual-metric">
          <div className={`vps-dual-metric-label vps-dual-metric-label--${themeClass}`}>CPU</div>
          <div className={`vps-dual-metric-value vps-dual-metric-value--${themeClass}`}>
            {metrics.cpu.usage.toFixed(2)}%
          </div>
          <div className={`vps-dual-bar vps-dual-bar--${themeClass}`}>
            <div 
              className="vps-dual-bar-fill"
              style={{ 
                width: `${Math.min(100, metrics.cpu.usage)}%`,
                backgroundColor: getBarColor(metrics.cpu.usage, [50, 80])
              }}
            />
          </div>
        </div>

        {/* Memory */}
        <div className="vps-dual-metric">
          <div className={`vps-dual-metric-label vps-dual-metric-label--${themeClass}`}>{t('dashboard.mem') || '内存'}</div>
          <div className={`vps-dual-metric-value vps-dual-metric-value--${themeClass}`}>
            {metrics.memory.usage_percent.toFixed(2)}%
          </div>
          <div className={`vps-dual-bar vps-dual-bar--${themeClass}`}>
            <div 
              className="vps-dual-bar-fill"
              style={{ 
                width: `${Math.min(100, metrics.memory.usage_percent)}%`,
                backgroundColor: getBarColor(metrics.memory.usage_percent, [50, 80])
              }}
            />
          </div>
        </div>

        {/* Storage */}
        <div className="vps-dual-metric">
          <div className={`vps-dual-metric-label vps-dual-metric-label--${themeClass}`}>{t('dashboard.storage') || '存储'}</div>
          <div className={`vps-dual-metric-value vps-dual-metric-value--${themeClass}`}>
            {diskUsage.toFixed(2)}%
          </div>
          <div className={`vps-dual-bar vps-dual-bar--${themeClass}`}>
            <div 
              className="vps-dual-bar-fill"
              style={{ 
                width: `${Math.min(100, diskUsage)}%`,
                backgroundColor: getBarColor(diskUsage, [70, 90])
              }}
            />
          </div>
        </div>

        {/* Upload Speed */}
        <div className="vps-dual-metric">
          <div className={`vps-dual-metric-label vps-dual-metric-label--${themeClass}`}>{t('dashboard.upload') || '上传'}</div>
          <div className={`vps-dual-metric-value vps-dual-metric-value--${themeClass}`}>
            {formatSpeed(speed.tx_sec)}
          </div>
        </div>

        {/* Download Speed */}
        <div className="vps-dual-metric">
          <div className={`vps-dual-metric-label vps-dual-metric-label--${themeClass}`}>{t('dashboard.download') || '下载'}</div>
          <div className={`vps-dual-metric-value vps-dual-metric-value--${themeClass}`}>
            {formatSpeed(speed.rx_sec)}
          </div>
        </div>
      </div>

      {/* Traffic Footer */}
      <div className="vps-dual-footer">
        <div className={`vps-dual-traffic vps-dual-traffic--${themeClass}`}>
          <span>{t('dashboard.upload') || '上传'}:{formatTrafficGiB(totalTxTraffic)}</span>
        </div>
        <div className={`vps-dual-traffic vps-dual-traffic--${themeClass}`}>
          <span>{t('dashboard.download') || '下载'}:{formatTrafficGiB(totalRxTraffic)}</span>
        </div>
      </div>
    </div>
  );
}

