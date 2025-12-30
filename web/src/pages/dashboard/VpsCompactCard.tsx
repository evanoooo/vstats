import { useTranslation } from 'react-i18next';
import { type ServerState } from '../../hooks/useMetrics';
import { getFlag, formatSpeed, formatTraffic, formatUptimeDays } from './utils';

export interface VpsCompactTableHeaderProps {
  themeId: string;
}

// VPS Compact Table Header
export function VpsCompactTableHeader({ themeId }: VpsCompactTableHeaderProps) {
  const { t } = useTranslation();
  return (
    <div className={`vps-compact-header vps-compact-header--${themeId}`}>
      <div className="vps-compact-col vps-compact-col--node">{t('dashboard.node')}</div>
      <div className="vps-compact-col vps-compact-col--type">{t('dashboard.type')}</div>
      <div className="vps-compact-col vps-compact-col--uptime">{t('dashboard.uptime')}</div>
      <div className="vps-compact-col vps-compact-col--network">{t('dashboard.network')}</div>
      <div className="vps-compact-col vps-compact-col--traffic">{t('dashboard.traffic')}</div>
      <div className="vps-compact-col vps-compact-col--cpu">{t('dashboard.cpu')}</div>
      <div className="vps-compact-col vps-compact-col--mem">{t('dashboard.mem')}</div>
      <div className="vps-compact-col vps-compact-col--hdd">{t('dashboard.hdd')}</div>
    </div>
  );
}

export interface VpsCompactCardProps {
  server: ServerState;
  onClick: () => void;
  themeId: string;
}

// VPS Compact Row Component
export function VpsCompactCard({ server, onClick, themeId }: VpsCompactCardProps) {
  const { t } = useTranslation();
  const { metrics, speed, isConnected, config } = server;
  
  const flag = getFlag(config.location);

  // Check if metrics data is complete (has all required fields)
  const hasCompleteMetrics = metrics && metrics.os && metrics.os.name && metrics.cpu && metrics.memory;

  if (!hasCompleteMetrics) {
    // Server has never reported data - show offline row with server info
    return (
      <div className={`vps-compact-row vps-compact-row--${themeId}`} onClick={onClick}>
        {/* NODE */}
        <div className="vps-compact-col vps-compact-col--node">
          <span className="vps-compact-status is-offline" style={{ background: '#ef4444' }} />
          <div className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 bg-white/5 border border-red-500/30">
            {flag ? (
              <span className="text-xl opacity-50">{flag}</span>
            ) : (
              <span className="text-xl opacity-50">🖥️</span>
            )}
          </div>
          <div className="vps-compact-node-info">
            <span className={`vps-compact-node-name vps-compact-node-name--${themeId}`}>
              {config.name}
            </span>
            <span className="text-xs text-red-500 font-medium">
              {t('dashboard.offline') || '离线'}
            </span>
          </div>
        </div>
        {/* TYPE */}
        <div className={`vps-compact-col vps-compact-col--type text-gray-500`}>
          {config.tag || '-'}
        </div>
        {/* UPTIME */}
        <div className={`vps-compact-col vps-compact-col--uptime text-gray-500`}>
          -
        </div>
        {/* NETWORK */}
        <div className={`vps-compact-col vps-compact-col--network text-gray-500`}>
          -
        </div>
        {/* TRAFFIC */}
        <div className={`vps-compact-col vps-compact-col--traffic text-gray-500`}>
          -
        </div>
        {/* CPU */}
        <div className="vps-compact-col vps-compact-col--cpu">
          <div className={`vps-compact-meter vps-compact-meter--${themeId}`}>
            <div className="vps-compact-meter-fill" style={{ width: '0%' }} />
          </div>
          <span className="vps-compact-meter-text text-gray-500">-</span>
        </div>
        {/* MEM */}
        <div className="vps-compact-col vps-compact-col--mem">
          <div className={`vps-compact-meter vps-compact-meter--${themeId}`}>
            <div className="vps-compact-meter-fill" style={{ width: '0%' }} />
          </div>
          <span className="vps-compact-meter-text text-gray-500">-</span>
        </div>
        {/* HDD */}
        <div className="vps-compact-col vps-compact-col--hdd">
          <div className={`vps-compact-meter vps-compact-meter--${themeId}`}>
            <div className="vps-compact-meter-fill" style={{ width: '0%' }} />
          </div>
          <span className="vps-compact-meter-text text-gray-500">-</span>
        </div>
      </div>
    );
  }

  const diskUsage = metrics.disks?.[0]?.usage_percent || 0;
  
  // Get virtualization type from config tag or default
  const getVirtType = () => {
    if (config.tag) return config.tag;
    // Check kernel for hints about virtualization
    const kernel = metrics.os.kernel?.toLowerCase() || '';
    if (kernel.includes('kvm')) return 'KVM';
    if (kernel.includes('vmware')) return 'VMware';
    if (kernel.includes('xen')) return 'Xen';
    if (kernel.includes('hyper-v')) return 'Hyper-V';
    if (kernel.includes('lxc')) return 'LXC';
    if (kernel.includes('openvz')) return 'OpenVZ';
    return 'VPS';
  };

  // Calculate total traffic from network metrics
  const totalTxTraffic = metrics.network?.total_tx || 0;
  const totalRxTraffic = metrics.network?.total_rx || 0;

  const getBarColor = (value: number, thresholds: [number, number]) => {
    if (value > thresholds[1]) return 'var(--compact-bar-bad)';
    if (value > thresholds[0]) return 'var(--compact-bar-warn)';
    return 'var(--compact-bar-ok)';
  };

  return (
    <div className={`vps-compact-row vps-compact-row--${themeId}`} onClick={onClick}>
      {/* NODE */}
      <div className="vps-compact-col vps-compact-col--node">
        <span className={`vps-compact-status ${isConnected ? 'is-online' : 'is-offline'}`} />
        {/* Country Flag as main icon */}
        <div className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 bg-white/5 border border-white/10">
          {flag ? (
            <span className="text-xl">{flag}</span>
          ) : (
            <span className="text-xl">🌍</span>
          )}
        </div>
        <div className="vps-compact-node-info">
          <span className={`vps-compact-node-name vps-compact-node-name--${themeId}`}>
            {config.name}
          </span>
          <span className={`vps-compact-node-location vps-compact-node-location--${themeId}`}>
            {config.location || 'Unknown'}
          </span>
        </div>
      </div>

      {/* TYPE */}
      <div className={`vps-compact-col vps-compact-col--type vps-compact-text--${themeId}`}>
        {getVirtType()}
      </div>

      {/* UPTIME */}
      <div className={`vps-compact-col vps-compact-col--uptime vps-compact-text--${themeId}`}>
        {formatUptimeDays(metrics.uptime, t)}
      </div>

      {/* NETWORK */}
      <div className={`vps-compact-col vps-compact-col--network vps-compact-text--${themeId}`}>
        <span>{formatSpeed(speed.tx_sec)}↑</span>
        <span>{formatSpeed(speed.rx_sec)}↓</span>
      </div>

      {/* TRAFFIC */}
      <div className={`vps-compact-col vps-compact-col--traffic vps-compact-text--${themeId}`}>
        <span>{formatTraffic(totalTxTraffic)}↑</span>
        <span>{formatTraffic(totalRxTraffic)}↓</span>
      </div>

      {/* CPU */}
      <div className="vps-compact-col vps-compact-col--cpu">
        <div className={`vps-compact-meter vps-compact-meter--${themeId}`}>
          <div 
            className="vps-compact-meter-fill"
            style={{ 
              width: `${Math.min(100, metrics.cpu.usage)}%`,
              backgroundColor: getBarColor(metrics.cpu.usage, [50, 80])
            }}
          />
        </div>
        <span className={`vps-compact-meter-text vps-compact-meter-text--${themeId}`}>
          {Math.round(metrics.cpu.usage)}%
        </span>
      </div>

      {/* MEM */}
      <div className="vps-compact-col vps-compact-col--mem">
        <div className={`vps-compact-meter vps-compact-meter--${themeId}`}>
          <div 
            className="vps-compact-meter-fill"
            style={{ 
              width: `${Math.min(100, metrics.memory.usage_percent)}%`,
              backgroundColor: getBarColor(metrics.memory.usage_percent, [50, 80])
            }}
          />
        </div>
        <span className={`vps-compact-meter-text vps-compact-meter-text--${themeId}`}>
          {Math.round(metrics.memory.usage_percent)}%
        </span>
      </div>

      {/* HDD */}
      <div className="vps-compact-col vps-compact-col--hdd">
        <div className={`vps-compact-meter vps-compact-meter--${themeId}`}>
          <div 
            className="vps-compact-meter-fill"
            style={{ 
              width: `${Math.min(100, diskUsage)}%`,
              backgroundColor: getBarColor(diskUsage, [70, 90])
            }}
          />
        </div>
        <span className={`vps-compact-meter-text vps-compact-meter-text--${themeId}`}>
          {Math.round(diskUsage)}%
        </span>
      </div>
    </div>
  );
}

