import { useTranslation } from 'react-i18next';
import type { ServerState } from '../../hooks/useMetrics';
import { formatSpeed } from '../../hooks/useMetrics';

interface GlobeOverlayProps {
  servers: ServerState[];
  selectedServer: ServerState | null;
  hoveredServer: ServerState | null;
  onClose: () => void;
  onExitFullscreen?: () => void;
  isDark: boolean;
}

/**
 * Overlay UI for the globe view (stats, tooltips, modals)
 */
export function GlobeOverlay({
  servers,
  selectedServer,
  hoveredServer,
  onClose,
  onExitFullscreen,
  isDark,
}: GlobeOverlayProps) {
  const { t } = useTranslation();

  const onlineCount = servers.filter((s) => s.isConnected).length;
  const offlineCount = servers.length - onlineCount;
  const totalRx = servers.reduce((acc, s) => acc + s.speed.rx_sec, 0);
  const totalTx = servers.reduce((acc, s) => acc + s.speed.tx_sec, 0);

  const themeClass = isDark ? 'dark' : 'light';

  return (
    <>
      {/* Exit Fullscreen Button - Top Right */}
      {onExitFullscreen && (
        <button
          onClick={onExitFullscreen}
          className={`globe-exit-btn globe-exit-btn--${themeClass}`}
          title="退出全屏"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}

      {/* Stats Panel - Bottom Left */}
      <div className={`globe-stats globe-stats--${themeClass}`}>
        <div className="globe-stats-grid">
          <div className="globe-stat">
            <span className="globe-stat-value text-emerald-400">{onlineCount}</span>
            <span className="globe-stat-label">{t('dashboard.online')}</span>
          </div>
          <div className="globe-stat">
            <span className="globe-stat-value text-red-400">{offlineCount}</span>
            <span className="globe-stat-label">{t('dashboard.offline')}</span>
          </div>
          <div className="globe-stat">
            <span className="globe-stat-value text-cyan-400">{formatSpeed(totalRx)}</span>
            <span className="globe-stat-label">↓ {t('dashboard.download')}</span>
          </div>
          <div className="globe-stat">
            <span className="globe-stat-value text-purple-400">{formatSpeed(totalTx)}</span>
            <span className="globe-stat-label">↑ {t('dashboard.upload')}</span>
          </div>
        </div>
      </div>

      {/* Hover Tooltip - follows mouse */}
      {hoveredServer && !selectedServer && (
        <div className={`globe-tooltip globe-tooltip--${themeClass}`}>
          <div className="globe-tooltip-header">
            <span className={`globe-tooltip-status ${hoveredServer.isConnected ? 'online' : 'offline'}`} />
            <span className="globe-tooltip-name">{hoveredServer.config.name}</span>
          </div>
          <div className="globe-tooltip-location">
            {hoveredServer.config.geoip?.country_name || hoveredServer.config.location || 'Unknown'}
          </div>
          {hoveredServer.isConnected && hoveredServer.metrics && (
            <div className="globe-tooltip-metrics">
              <span>CPU: {Math.round(hoveredServer.metrics.cpu?.usage || 0)}%</span>
              <span>RAM: {Math.round(hoveredServer.metrics.memory?.usage_percent || 0)}%</span>
            </div>
          )}
        </div>
      )}

      {/* Selected Server Modal */}
      {selectedServer && (
        <div className={`globe-modal globe-modal--${themeClass}`}>
          <div className="globe-modal-content">
            <button className="globe-modal-close" onClick={onClose}>
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>

            <div className="globe-modal-header">
              <span className={`globe-modal-status ${selectedServer.isConnected ? 'online' : 'offline'}`} />
              <h3 className="globe-modal-title">{selectedServer.config.name}</h3>
            </div>

            <div className="globe-modal-info">
              <div className="globe-modal-row">
                <span className="globe-modal-label">{t('serverDetail.location')}</span>
                <span className="globe-modal-value">
                  {selectedServer.config.geoip?.city && `${selectedServer.config.geoip.city}, `}
                  {selectedServer.config.geoip?.country_name || selectedServer.config.location || 'Unknown'}
                </span>
              </div>

              {selectedServer.config.provider && (
                <div className="globe-modal-row">
                  <span className="globe-modal-label">{t('settings.provider')}</span>
                  <span className="globe-modal-value">{selectedServer.config.provider}</span>
                </div>
              )}

              {selectedServer.isConnected && selectedServer.metrics && (
                <>
                  <div className="globe-modal-divider" />

                  <div className="globe-modal-row">
                    <span className="globe-modal-label">CPU</span>
                    <div className="globe-modal-meter">
                      <div
                        className="globe-modal-meter-fill"
                        style={{
                          width: `${selectedServer.metrics.cpu?.usage || 0}%`,
                          backgroundColor: getUsageColor(selectedServer.metrics.cpu?.usage || 0),
                        }}
                      />
                    </div>
                    <span className="globe-modal-value">{Math.round(selectedServer.metrics.cpu?.usage || 0)}%</span>
                  </div>

                  <div className="globe-modal-row">
                    <span className="globe-modal-label">RAM</span>
                    <div className="globe-modal-meter">
                      <div
                        className="globe-modal-meter-fill"
                        style={{
                          width: `${selectedServer.metrics.memory?.usage_percent || 0}%`,
                          backgroundColor: getUsageColor(selectedServer.metrics.memory?.usage_percent || 0),
                        }}
                      />
                    </div>
                    <span className="globe-modal-value">{Math.round(selectedServer.metrics.memory?.usage_percent || 0)}%</span>
                  </div>

                  <div className="globe-modal-row">
                    <span className="globe-modal-label">{t('dashboard.network')}</span>
                    <span className="globe-modal-value">
                      ↓ {formatSpeed(selectedServer.speed.rx_sec)} / ↑ {formatSpeed(selectedServer.speed.tx_sec)}
                    </span>
                  </div>

                  {selectedServer.metrics.ping?.targets && selectedServer.metrics.ping.targets.length > 0 && (
                    <>
                      <div className="globe-modal-divider" />
                      <div className="globe-modal-pings">
                        <span className="globe-modal-label">Ping Targets</span>
                        {selectedServer.metrics.ping.targets.slice(0, 3).map((target, idx) => (
                          <div key={idx} className="globe-modal-ping-item">
                            <span>{target.name}</span>
                            <span className={target.latency_ms !== null ? 'text-emerald-400' : 'text-red-400'}>
                              {target.latency_ms !== null ? `${target.latency_ms.toFixed(1)}ms` : 'N/A'}
                            </span>
                          </div>
                        ))}
                      </div>
                    </>
                  )}
                </>
              )}

              {!selectedServer.isConnected && (
                <div className="globe-modal-offline">
                  <span className="text-red-400">{t('dashboard.offline')}</span>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Instructions */}
      <div className={`globe-instructions globe-instructions--${themeClass}`}>
        <span>🖱️ {t('globe.dragToRotate') || 'Drag to rotate'}</span>
        <span>🔍 {t('globe.scrollToZoom') || 'Scroll to zoom'}</span>
        <span>👆 {t('globe.clickForDetails') || 'Click node for details'}</span>
      </div>
    </>
  );
}

function getUsageColor(usage: number): string {
  if (usage > 80) return '#ef4444';
  if (usage > 50) return '#f59e0b';
  return '#10b981';
}

export default GlobeOverlay;
