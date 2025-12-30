import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useServerManager, formatBytes, formatSpeed, formatUptime } from '../../hooks/useMetrics';
import { getOsIcon, getProviderIcon } from '../../components/Icons';
import { getProviderLogo, getDistributionLogo, LogoImage } from '../../utils/logoUtils';
import { useTheme } from '../../context/ThemeContext';
import { getFlag } from './utils';
import { StatCard, HistoryChart } from './components';

export default function ServerDetail() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { servers, loadingState, isInitialLoad } = useServerManager();
  const { isDark, backgroundUrl, background } = useTheme();
  const [showContent, setShowContent] = useState(false);

  const server = servers.find(s => s.config.id === id);

  // Delay showing content for smooth transition
  useEffect(() => {
    if (server?.metrics) {
      const timer = setTimeout(() => setShowContent(true), 50);
      return () => clearTimeout(timer);
    }
  }, [server?.metrics]);

  // Show loading state during initial load or when server data is not yet available
  if (isInitialLoad || loadingState === 'loading') {
    return (
      <div className={`min-h-screen relative ${backgroundUrl ? 'has-bg-image' : ''}`}>
        {backgroundUrl && (
          <>
            <div 
              className="bg-image-layer"
              style={{ 
                backgroundImage: `url(${backgroundUrl})`,
                filter: `blur(${background.blur || 0}px)`,
                transform: 'scale(1.1)'
              }}
            />
            <div 
              className="bg-overlay"
              style={{ 
                backgroundColor: isDark ? '#000000' : '#ffffff',
                opacity: (100 - (background.opacity || 100)) / 100
              }}
            />
          </>
        )}
        <div className="relative z-10 min-h-screen flex items-center justify-center">
          <div className="flex flex-col items-center gap-4 animate-pulse">
            <div className="w-12 h-12 border-4 border-white/20 border-t-emerald-500 rounded-full animate-spin" />
            <div className="text-white/60 text-sm">{t('serverDetail.loadingServerData')}</div>
          </div>
        </div>
      </div>
    );
  }

  // If server not found after data is loaded, show a brief delay before showing error
  // This prevents flash of "not found" during navigation
  if (!server) {
    return (
      <div className={`min-h-screen relative ${backgroundUrl ? 'has-bg-image' : ''}`}>
        {backgroundUrl && (
          <>
            <div 
              className="bg-image-layer"
              style={{ 
                backgroundImage: `url(${backgroundUrl})`,
                filter: `blur(${background.blur || 0}px)`,
                transform: 'scale(1.1)'
              }}
            />
            <div 
              className="bg-overlay"
              style={{ 
                backgroundColor: isDark ? '#000000' : '#ffffff',
                opacity: (100 - (background.opacity || 100)) / 100
              }}
            />
          </>
        )}
        <div className="relative z-10 min-h-screen flex items-center justify-center animate-fadeIn">
          <div className="text-center">
            <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-white/5 border border-white/10 flex items-center justify-center">
              <svg className="w-8 h-8 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
              </svg>
            </div>
            <div className="text-gray-400 mb-2 text-lg font-medium">{t('serverDetail.serverNotAvailable')}</div>
            <div className="text-gray-600 text-sm mb-6">{t('serverDetail.serverNotAvailableDesc')}</div>
            <button 
              onClick={() => navigate('/')}
              className="px-6 py-2.5 rounded-xl bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/30 text-emerald-400 text-sm font-medium transition-all hover:scale-105"
            >
              ← {t('serverDetail.backToDashboard')}
            </button>
          </div>
        </div>
      </div>
    );
  }

  const { metrics, speed, isConnected, config } = server;

  // Show connecting state if metrics not yet available
  if (!metrics) {
    return (
      <div className={`min-h-screen relative ${backgroundUrl ? 'has-bg-image' : ''}`}>
        {backgroundUrl && (
          <>
            <div 
              className="bg-image-layer"
              style={{ 
                backgroundImage: `url(${backgroundUrl})`,
                filter: `blur(${background.blur || 0}px)`,
                transform: 'scale(1.1)'
              }}
            />
            <div 
              className="bg-overlay"
              style={{ 
                backgroundColor: isDark ? '#000000' : '#ffffff',
                opacity: (100 - (background.opacity || 100)) / 100
              }}
            />
          </>
        )}
        <div className="relative z-10 min-h-screen flex items-center justify-center">
          <div className="flex flex-col items-center gap-4">
            <div className="w-12 h-12 border-4 border-white/20 border-t-emerald-500 rounded-full animate-spin" />
            <div className="text-white/60 text-sm">{t('serverDetail.connectingTo', { name: config.name })}</div>
          </div>
        </div>
      </div>
    );
  }

  // Check if metrics data is incomplete (offline server with only partial data from database)
  const hasCompleteMetrics = metrics.os && metrics.os.name && metrics.cpu && metrics.memory;

  const OsIcon = hasCompleteMetrics ? getOsIcon(metrics.os.name) : null;
  const ProviderIcon = config.provider ? getProviderIcon(config.provider) : null;
  const providerLogo = config.provider ? getProviderLogo(config.provider) : null;
  const distributionLogo = hasCompleteMetrics ? getDistributionLogo(metrics.os.name) : null;
  const flag = getFlag(config.location || '');

  // If metrics data is incomplete (offline server), show a simplified view
  if (!hasCompleteMetrics) {
    return (
      <div className={`server-detail min-h-screen relative ${backgroundUrl ? 'has-bg-image' : ''}`}>
        {backgroundUrl && (
          <>
            <div 
              className="bg-image-layer"
              style={{ 
                backgroundImage: `url(${backgroundUrl})`,
                filter: `blur(${background.blur || 0}px)`,
                transform: 'scale(1.1)'
              }}
            />
            <div 
              className="bg-overlay"
              style={{ 
                backgroundColor: isDark ? '#000000' : '#ffffff',
                opacity: (100 - (background.opacity || 100)) / 100
              }}
            />
          </>
        )}
        <div className="relative z-10 p-4 md:p-6 lg:p-10 max-w-6xl mx-auto animate-fadeIn">
        {/* Back Button */}
        <button 
          onClick={() => navigate('/')}
          className={`mb-6 flex items-center gap-2 ${isDark ? 'text-gray-400 hover:text-white' : 'text-gray-600 hover:text-gray-900'} transition-colors group`}
        >
          <svg className="w-5 h-5 group-hover:-translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          <span className="text-sm">{t('serverDetail.backToDashboard')}</span>
        </button>

        {/* Offline Header */}
        <div className="nezha-card p-6 md:p-8 mb-6 relative overflow-hidden">
          {/* Offline Banner */}
          <div className="absolute top-0 left-0 right-0 bg-red-500/90 text-white text-sm font-medium py-2 px-4 flex items-center gap-2 z-20">
            <span className="w-2 h-2 rounded-full bg-white animate-pulse" />
            {t('dashboard.offline') || '离线'}
          </div>
          
          <div className="flex flex-col md:flex-row items-start md:items-center gap-6 relative z-10 pt-8">
            <div className="w-20 h-20 rounded-2xl bg-gradient-to-br from-red-500/20 to-red-500/5 border border-red-500/30 flex items-center justify-center">
              {providerLogo ? (
                <LogoImage src={providerLogo} alt={config.provider || ''} className="w-14 h-14 object-contain opacity-50" />
              ) : (
                <svg className="w-12 h-12 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
                </svg>
              )}
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-3 mb-3">
                <h1 className="text-3xl font-bold text-white">{config.name}</h1>
                <span className="w-3 h-3 rounded-full bg-red-500 animate-pulse" />
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {config.location && (
                  <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-cyan-500/10 border border-cyan-500/20">
                    <span className="text-sm">{flag}</span>
                    <span className="text-xs text-cyan-300 font-medium">{config.location}</span>
                  </div>
                )}
                {config.provider && (
                  <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-amber-500/10 border border-amber-500/20">
                    {providerLogo && <LogoImage src={providerLogo} alt={config.provider} className="w-4 h-4 object-contain" />}
                    <span className="text-xs text-amber-300 font-medium">{config.provider}</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Partial metrics display if available */}
        {metrics.cpu && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
            <div className="nezha-card p-4">
              <div className="text-xs text-gray-500 mb-1">CPU</div>
              <div className="text-2xl font-bold text-blue-400 font-mono">{metrics.cpu.usage?.toFixed(1) || 0}%</div>
            </div>
            {metrics.memory && (
              <div className="nezha-card p-4">
                <div className="text-xs text-gray-500 mb-1">Memory</div>
                <div className="text-2xl font-bold text-purple-400 font-mono">{metrics.memory.usage_percent?.toFixed(1) || 0}%</div>
              </div>
            )}
            {metrics.disks && metrics.disks[0] && (
              <div className="nezha-card p-4">
                <div className="text-xs text-gray-500 mb-1">Disk</div>
                <div className="text-2xl font-bold text-amber-400 font-mono">{metrics.disks[0].usage_percent?.toFixed(1) || 0}%</div>
              </div>
            )}
          </div>
        )}

        {/* History Charts - still available for offline servers */}
        <HistoryChart serverId={id!} />
        </div>
      </div>
    );
  }

  return (
    <div className={`server-detail min-h-screen relative ${backgroundUrl ? 'has-bg-image' : ''}`}>
      {/* Background Image Layer */}
      {backgroundUrl && (
        <>
          <div 
            className="bg-image-layer"
            style={{ 
              backgroundImage: `url(${backgroundUrl})`,
              filter: `blur(${background.blur || 0}px)`,
              transform: 'scale(1.1)' // Prevent blur edges from showing
            }}
          />
          <div 
            className="bg-overlay"
            style={{ 
              backgroundColor: isDark ? '#000000' : '#ffffff',
              opacity: (100 - (background.opacity || 100)) / 100
            }}
          />
        </>
      )}
      
      <div className={`relative z-10 p-4 md:p-6 lg:p-10 max-w-6xl mx-auto ${showContent ? 'animate-slideUp' : 'opacity-0'}`}>
      {/* Back Button */}
      <button 
        onClick={() => navigate('/')}
        className={`mb-6 flex items-center gap-2 ${isDark ? 'text-gray-400 hover:text-white' : 'text-gray-600 hover:text-gray-900'} transition-colors group`}
      >
        <svg className="w-5 h-5 group-hover:-translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
        </svg>
        <span className="text-sm">{t('serverDetail.backToDashboard')}</span>
      </button>

      {/* Header */}
      <div className="nezha-card p-6 md:p-8 mb-6 relative overflow-hidden">
        {/* Provider Logo Background */}
        {providerLogo && (
          <div className="absolute -right-4 -bottom-4 w-32 h-32 opacity-[0.06] pointer-events-none">
            <LogoImage 
              src={providerLogo} 
              alt="" 
              className="w-full h-full object-contain transform rotate-[-15deg]" 
            />
          </div>
        )}
        
        <div className="flex flex-col md:flex-row items-start md:items-center gap-6 relative z-10">
          {/* Main Icon: OS System Logo */}
          <div className="w-20 h-20 rounded-2xl bg-gradient-to-br from-blue-500/20 to-blue-500/5 border border-blue-500/30 flex items-center justify-center overflow-hidden">
            {distributionLogo ? (
              <LogoImage src={distributionLogo} alt={metrics.os.name} className="w-14 h-14 object-contain" />
            ) : OsIcon ? (
              <OsIcon className="w-12 h-12 text-blue-400" />
            ) : (
              <svg className="w-12 h-12 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
              </svg>
            )}
          </div>
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-3">
              <h1 className="text-3xl font-bold text-white">{config.name}</h1>
              <span className={`w-3 h-3 rounded-full ${isConnected ? 'bg-emerald-500 shadow-[0_0_12px_rgba(16,185,129,0.6)]' : 'bg-red-500'}`} />
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {/* Location with flag */}
              {config.location && (
                <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-cyan-500/10 border border-cyan-500/20">
                  <span className="text-sm">{flag}</span>
                  <span className="text-xs text-cyan-300 font-medium">{config.location}</span>
                </div>
              )}
              {/* Provider */}
              {providerLogo ? (
                <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-amber-500/10 border border-amber-500/20">
                  <LogoImage src={providerLogo} alt={config.provider || ''} className="w-4 h-4 object-contain" />
                  <span className="text-xs text-amber-300 font-medium">{config.provider}</span>
                </div>
              ) : ProviderIcon ? (
                <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-amber-500/10 border border-amber-500/20">
                  <ProviderIcon className="w-4 h-4 text-amber-400" />
                  <span className="text-xs text-amber-300 font-medium">{config.provider}</span>
                </div>
              ) : null}
              {/* OS Name */}
              <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-blue-500/10 border border-blue-500/20">
                <span className="text-xs text-blue-300 font-medium">{metrics.os.name}</span>
              </div>
              {/* Architecture */}
              <div className="px-2.5 py-1.5 rounded-lg bg-purple-500/10 border border-purple-500/20">
                <span className="text-xs text-purple-300 font-medium">{metrics.os.arch}</span>
              </div>
              {/* Cores */}
              <div className="px-2.5 py-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
                <span className="text-xs text-emerald-300 font-medium">{metrics.cpu.cores} Cores</span>
              </div>
            </div>
          </div>
          <div className="text-right">
            <div className="text-[10px] text-gray-500 uppercase tracking-wider">{t('dashboard.uptime')}</div>
            <div className="text-2xl font-bold text-emerald-400 font-mono">{formatUptime(metrics.uptime)}</div>
          </div>
        </div>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <StatCard label={t('serverDetail.kernel')} value={metrics.os.kernel} color="gray" />
        <StatCard label={t('serverDetail.load1m')} value={metrics.load_average.one.toFixed(2)} color="purple" />
        <StatCard label={t('serverDetail.load5m')} value={metrics.load_average.five.toFixed(2)} color="purple" />
        <StatCard label={t('serverDetail.load15m')} value={metrics.load_average.fifteen.toFixed(2)} color="purple" />
      </div>

      {/* Main Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        
        {/* CPU Section */}
        <div className="nezha-card p-6">
          <h2 className="text-sm font-bold text-gray-400 uppercase tracking-wider mb-4 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-blue-500"></span>
            {t('serverDetail.cpuSection')}
          </h2>
          <div className="flex items-center justify-between mb-4">
            <span className="text-sm text-gray-300 truncate flex-1 mr-4">{metrics.cpu.brand}</span>
            <span className="text-3xl font-bold text-blue-400 font-mono">{metrics.cpu.usage.toFixed(1)}%</span>
          </div>
          <div className="h-3 w-full bg-gray-700/50 rounded-full overflow-hidden mb-4">
            <div 
              className="h-full rounded-full bg-gradient-to-r from-blue-500 to-cyan-400 transition-all duration-500" 
              style={{ width: `${metrics.cpu.usage}%` }} 
            />
          </div>
          <div className="flex items-center justify-between text-xs text-gray-500 mb-6">
            <span>{t('serverDetail.cpuCores', { count: metrics.cpu.cores })}</span>
            <span>{(metrics.cpu.frequency / 1000).toFixed(2)} GHz</span>
          </div>

          {/* Per-core usage */}
          {metrics.cpu.per_core && metrics.cpu.per_core.length > 0 && (
            <div className="pt-4 border-t border-white/5">
              <div className="text-xs text-gray-500 mb-3">{t('serverDetail.perCoreUsage')}</div>
              <div className="grid grid-cols-5 gap-2">
                {metrics.cpu.per_core.map((usage, i) => (
                  <div key={i} className="relative h-16 rounded-lg bg-gray-800/50 overflow-hidden group" title={`Core ${i}: ${usage.toFixed(0)}%`}>
                    <div 
                      className={`absolute bottom-0 left-0 right-0 transition-all duration-500 ${usage > 80 ? 'bg-red-500' : usage > 50 ? 'bg-yellow-500' : 'bg-emerald-500'}`}
                      style={{ height: `${usage}%` }}
                    />
                    <div className="absolute inset-0 flex flex-col items-center justify-center">
                      <span className="text-[10px] font-mono text-white/50">{i}</span>
                      <span className="text-xs font-mono font-bold text-white">{usage.toFixed(0)}%</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Memory Section */}
        <div className="nezha-card p-6">
          <h2 className="text-sm font-bold text-gray-400 uppercase tracking-wider mb-4 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-purple-500"></span>
            {t('serverDetail.memorySection')}
          </h2>
          
          {/* RAM */}
          <div className="mb-6">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm text-gray-400">{t('serverDetail.ram')}</span>
              <span className="text-2xl font-bold text-purple-400 font-mono">{metrics.memory.usage_percent.toFixed(1)}%</span>
            </div>
            <div className="h-3 w-full bg-gray-700/50 rounded-full overflow-hidden mb-3">
              <div 
                className="h-full rounded-full bg-gradient-to-r from-purple-500 to-pink-400 transition-all duration-500" 
                style={{ width: `${metrics.memory.usage_percent}%` }} 
              />
            </div>
            <div className="grid grid-cols-3 gap-4 text-center">
              <div>
                <div className="text-xs text-gray-500">{t('serverDetail.used')}</div>
                <div className="text-sm font-mono text-white">{formatBytes(metrics.memory.used)}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">{t('serverDetail.available')}</div>
                <div className="text-sm font-mono text-emerald-400">{formatBytes(metrics.memory.available)}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">{t('serverDetail.total')}</div>
                <div className="text-sm font-mono text-white">{formatBytes(metrics.memory.total)}</div>
              </div>
            </div>
          </div>

          {/* Swap */}
          {metrics.memory.swap_total > 0 && (
            <div className="pt-4 border-t border-white/5">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-400">{t('serverDetail.swap')}</span>
                <span className="text-lg font-bold text-gray-400 font-mono">
                  {((metrics.memory.swap_used / metrics.memory.swap_total) * 100).toFixed(1)}%
                </span>
              </div>
              <div className="h-2 w-full bg-gray-700/50 rounded-full overflow-hidden mb-2">
                <div 
                  className="h-full rounded-full bg-gray-500 transition-all duration-500" 
                  style={{ width: `${(metrics.memory.swap_used / metrics.memory.swap_total) * 100}%` }} 
                />
              </div>
              <div className="flex justify-between text-xs text-gray-500">
                <span>{t('serverDetail.used')}: {formatBytes(metrics.memory.swap_used)}</span>
                <span>{t('serverDetail.total')}: {formatBytes(metrics.memory.swap_total)}</span>
              </div>
            </div>
          )}
        </div>

        {/* Storage Section */}
        <div className="nezha-card p-6">
          <h2 className="text-sm font-bold text-gray-400 uppercase tracking-wider mb-4 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-amber-500"></span>
            {t('serverDetail.storageSection')}
          </h2>
          <div className="space-y-5">
            {(metrics.disks || []).map((disk, i) => (
              <div key={i}>
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-gray-300 font-mono">{disk.model || disk.name}</span>
                    {disk.disk_type && (
                      <span className="text-[10px] text-gray-600 px-1.5 py-0.5 rounded bg-white/5">{disk.disk_type}</span>
                    )}
                  </div>
                  <span className={`text-lg font-bold font-mono ${disk.usage_percent > 90 ? 'text-red-400' : 'text-amber-400'}`}>
                    {disk.usage_percent.toFixed(1)}%
                  </span>
                </div>
                <div className="h-2 w-full bg-gray-700/50 rounded-full overflow-hidden mb-2">
                  <div 
                    className={`h-full rounded-full transition-all duration-500 ${disk.usage_percent > 90 ? 'bg-red-500' : 'bg-amber-500'}`}
                    style={{ width: `${disk.usage_percent}%` }} 
                  />
                </div>
                <div className="flex justify-between text-xs text-gray-500">
                  <span>{formatBytes(disk.used)} {t('serverDetail.used').toLowerCase()}</span>
                  <span>{formatBytes(disk.total - disk.used)} {t('serverDetail.free')}</span>
                  <span>{formatBytes(disk.total)} {t('serverDetail.total').toLowerCase()}</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* GPU Section */}
        {metrics.gpu && metrics.gpu.gpus && metrics.gpu.gpus.length > 0 && (
          <div className="nezha-card p-6">
            <h2 className="text-sm font-bold text-gray-400 uppercase tracking-wider mb-4 flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-green-500"></span>
              {t('serverDetail.gpuSection')}
            </h2>
            <div className="space-y-6">
              {metrics.gpu.gpus.map((gpu, i) => (
                <div key={i} className={i > 0 ? 'pt-4 border-t border-white/5' : ''}>
                  {/* GPU Header */}
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-gray-300 font-medium">{gpu.name}</span>
                      <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${
                        gpu.vendor === 'NVIDIA' ? 'bg-green-500/20 text-green-400' :
                        gpu.vendor === 'AMD' ? 'bg-red-500/20 text-red-400' :
                        'bg-blue-500/20 text-blue-400'
                      }`}>
                        {gpu.vendor}
                      </span>
                    </div>
                    {gpu.temperature !== undefined && gpu.temperature > 0 && (
                      <span className={`text-lg font-mono font-bold ${
                        gpu.temperature > 80 ? 'text-red-400' : gpu.temperature > 60 ? 'text-amber-400' : 'text-green-400'
                      }`}>
                        {gpu.temperature}°C
                      </span>
                    )}
                  </div>

                  {/* GPU Utilization */}
                  <div className="mb-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-xs text-gray-500">{t('serverDetail.gpuUsage')}</span>
                      <span className="text-xl font-bold text-green-400 font-mono">{gpu.utilization.toFixed(1)}%</span>
                    </div>
                    <div className="h-2 w-full bg-gray-700/50 rounded-full overflow-hidden">
                      <div 
                        className="h-full rounded-full bg-gradient-to-r from-green-500 to-emerald-400 transition-all duration-500" 
                        style={{ width: `${gpu.utilization}%` }} 
                      />
                    </div>
                  </div>

                  {/* VRAM Usage */}
                  {gpu.memory_total > 0 && (
                    <div className="mb-4">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-xs text-gray-500">{t('serverDetail.vram')}</span>
                        <span className="text-lg font-bold text-teal-400 font-mono">{gpu.memory_percent.toFixed(1)}%</span>
                      </div>
                      <div className="h-2 w-full bg-gray-700/50 rounded-full overflow-hidden mb-2">
                        <div 
                          className="h-full rounded-full bg-gradient-to-r from-teal-500 to-cyan-400 transition-all duration-500" 
                          style={{ width: `${gpu.memory_percent}%` }} 
                        />
                      </div>
                      <div className="flex justify-between text-xs text-gray-500">
                        <span>{formatBytes(gpu.memory_used)} {t('serverDetail.used').toLowerCase()}</span>
                        <span>{formatBytes(gpu.memory_total)} {t('serverDetail.total').toLowerCase()}</span>
                      </div>
                    </div>
                  )}

                  {/* GPU Stats Grid */}
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-center">
                    {gpu.power_draw !== undefined && gpu.power_draw > 0 && (
                      <div className="p-2 rounded-lg bg-white/[0.02] border border-white/5">
                        <div className="text-[10px] text-gray-500 uppercase">{t('serverDetail.power')}</div>
                        <div className="text-sm font-mono text-amber-400">
                          {gpu.power_draw.toFixed(0)}W
                          {gpu.power_limit ? ` / ${gpu.power_limit.toFixed(0)}W` : ''}
                        </div>
                      </div>
                    )}
                    {gpu.fan_speed !== undefined && gpu.fan_speed > 0 && (
                      <div className="p-2 rounded-lg bg-white/[0.02] border border-white/5">
                        <div className="text-[10px] text-gray-500 uppercase">{t('serverDetail.fan')}</div>
                        <div className="text-sm font-mono text-blue-400">{gpu.fan_speed}%</div>
                      </div>
                    )}
                    {gpu.clock_core !== undefined && gpu.clock_core > 0 && (
                      <div className="p-2 rounded-lg bg-white/[0.02] border border-white/5">
                        <div className="text-[10px] text-gray-500 uppercase">{t('serverDetail.coreClock')}</div>
                        <div className="text-sm font-mono text-purple-400">{gpu.clock_core} MHz</div>
                      </div>
                    )}
                    {gpu.clock_memory !== undefined && gpu.clock_memory > 0 && (
                      <div className="p-2 rounded-lg bg-white/[0.02] border border-white/5">
                        <div className="text-[10px] text-gray-500 uppercase">{t('serverDetail.memClock')}</div>
                        <div className="text-sm font-mono text-pink-400">{gpu.clock_memory} MHz</div>
                      </div>
                    )}
                  </div>

                  {/* Driver Info */}
                  {(gpu.driver_version || gpu.cuda_version) && (
                    <div className="mt-3 flex flex-wrap gap-2 text-xs">
                      {gpu.driver_version && (
                        <span className="px-2 py-1 rounded bg-white/5 text-gray-400">
                          Driver: {gpu.driver_version}
                        </span>
                      )}
                      {gpu.cuda_version && (
                        <span className="px-2 py-1 rounded bg-green-500/10 text-green-400">
                          CUDA: {gpu.cuda_version}
                        </span>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Network Section */}
        <div className="nezha-card p-6">
          <h2 className="text-sm font-bold text-gray-400 uppercase tracking-wider mb-4 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-cyan-500"></span>
            {t('serverDetail.networkSection')}
          </h2>

          {/* Current Speed */}
          <div className="grid grid-cols-2 gap-4 mb-6">
            <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20">
              <div className="text-xs text-emerald-400 mb-1">↑ {t('serverDetail.uploadSpeed')}</div>
              <div className="text-2xl font-bold font-mono text-emerald-300">{formatSpeed(speed.tx_sec)}</div>
            </div>
            <div className="p-4 rounded-xl bg-blue-500/10 border border-blue-500/20">
              <div className="text-xs text-blue-400 mb-1">↓ {t('serverDetail.downloadSpeed')}</div>
              <div className="text-2xl font-bold font-mono text-blue-300">{formatSpeed(speed.rx_sec)}</div>
            </div>
          </div>

          {/* Total Traffic */}
          <div className="grid grid-cols-2 gap-4 mb-6 p-4 rounded-xl bg-white/[0.02] border border-white/5">
            <div>
              <div className="text-xs text-gray-500 mb-1">{t('serverDetail.totalUploaded')}</div>
              <div className="text-lg font-bold font-mono text-white">{formatBytes(metrics.network.total_tx)}</div>
            </div>
            <div>
              <div className="text-xs text-gray-500 mb-1">{t('serverDetail.totalDownloaded')}</div>
              <div className="text-lg font-bold font-mono text-white">{formatBytes(metrics.network.total_rx)}</div>
            </div>
          </div>

          {/* Interfaces */}
          {metrics.network.interfaces && metrics.network.interfaces.length > 0 && (
          <div className="pt-4 border-t border-white/5">
            <div className="text-xs text-gray-500 mb-3">{t('serverDetail.networkInterfaces')}</div>
            <div className="space-y-2 max-h-48 overflow-y-auto">
              {[...(metrics.network.interfaces || [])]
                .filter(iface => iface.rx_bytes > 0 || iface.tx_bytes > 0)
                .sort((a, b) => a.name.localeCompare(b.name))
                .map((iface, i) => (
                  <div key={i} className="flex items-center justify-between py-2 px-3 rounded-lg bg-white/[0.02] hover:bg-white/[0.04] transition-colors">
                    <span className="font-mono text-gray-300 text-sm">{iface.name}</span>
                    <div className="flex gap-4 text-xs font-mono">
                      <span className="text-emerald-400">↑ {formatBytes(iface.tx_bytes)}</span>
                      <span className="text-blue-400">↓ {formatBytes(iface.rx_bytes)}</span>
                    </div>
                  </div>
                ))}
            </div>
          </div>
          )}

          {/* Ping Status */}
          {metrics.ping && metrics.ping.targets && metrics.ping.targets.length > 0 && (
            <div className="pt-4 border-t border-white/5 mt-4">
              <div className="text-xs text-gray-500 mb-3">{t('serverDetail.pingLatency')}</div>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                {metrics.ping.targets.map((target, i) => (
                  <div 
                    key={i} 
                    className={`p-3 rounded-lg border ${
                      target.status === 'ok' 
                        ? 'bg-emerald-500/5 border-emerald-500/20' 
                        : target.status === 'timeout'
                        ? 'bg-amber-500/5 border-amber-500/20'
                        : 'bg-red-500/5 border-red-500/20'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs text-gray-400">{target.name}</span>
                      <span className={`w-2 h-2 rounded-full ${
                        target.status === 'ok' ? 'bg-emerald-500' : target.status === 'timeout' ? 'bg-amber-500' : 'bg-red-500'
                      }`} />
                    </div>
                    <div className="text-lg font-mono font-bold text-white">
                      {target.latency_ms !== null ? `${target.latency_ms.toFixed(1)} ms` : '--'}
                    </div>
                    <div className="text-[10px] text-gray-500 mt-1 font-mono">{target.host}</div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* History Section - Full Width */}
      <div className="mt-6">
        <HistoryChart serverId={id!} />
      </div>

      {/* Footer */}
      <footer className="text-center mt-8 pt-6 border-t border-white/5">
        <p className="text-white/20 text-xs font-mono">
          {t('serverDetail.lastUpdated')} {new Date().toLocaleString()}
        </p>
      </footer>
      </div>
    </div>
  );
}

