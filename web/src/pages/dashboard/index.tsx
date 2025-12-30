import { useState, useEffect, lazy, Suspense } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useServerManager, formatSpeed } from '../../hooks/useMetrics';
import { useTheme } from '../../context/ThemeContext';
import { LanguageSwitcher } from '../../components/LanguageSwitcher';
import type { GroupOption } from '../../types';

import { SocialLinks } from './SocialLinks';
import { VpsGridCard } from './VpsGridCard';
import { VpsListCard } from './VpsListCard';
import { VpsDualCard } from './VpsDualCard';
import { VpsCompactCard, VpsCompactTableHeader } from './VpsCompactCard';
import { 
  VpsGridCardSkeleton, 
  VpsListCardSkeleton, 
  VpsDualCardSkeleton,
  VpsCompactRowSkeleton
} from './skeletons';

// Lazy load GlobeView for better initial load performance
const GlobeView = lazy(() => import('../../components/globe/GlobeView'));

type ViewMode = 'list' | 'grid' | 'dual' | 'compact' | 'globe';

export default function Dashboard() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { servers, groupDimensions, siteSettings, isInitialLoad } = useServerManager();
  const { isDark, backgroundUrl, background, themeId } = useTheme();
  const themeClass = themeId;
  
  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    return (localStorage.getItem('vstats-view-mode') as ViewMode) || 'grid';
  });
  const [serverVersion, setServerVersion] = useState<string>('');
  const [onlineUsers, setOnlineUsers] = useState<number>(0);
  
  // Selected dimension for grouping (null = no grouping, 'tag' = group by tag)
  const [selectedDimensionId, setSelectedDimensionId] = useState<string | null>(() => {
    return localStorage.getItem('vstats-group-dimension') || null;
  });
  
  // Get enabled dimensions only
  const enabledDimensions = groupDimensions.filter(d => d.enabled);

  useEffect(() => {
    const fetchServerVersion = async () => {
      try {
        const res = await fetch('/api/version');
        if (res.ok) {
          const data = await res.json();
          setServerVersion(data.version || '');
        }
      } catch (e) {
        console.error('Failed to fetch server version', e);
      }
    };
    fetchServerVersion();
  }, []);

  // Fetch online users count
  useEffect(() => {
    const fetchOnlineUsers = async () => {
      try {
        const res = await fetch('/api/online-users');
        if (res.ok) {
          const data = await res.json();
          setOnlineUsers(data.count || 0);
        }
      } catch (e) {
        console.error('Failed to fetch online users', e);
      }
    };
    
    // Fetch initially
    fetchOnlineUsers();
    
    // Refresh every 30 seconds
    const interval = setInterval(fetchOnlineUsers, 30000);
    return () => clearInterval(interval);
  }, []);

  // Prevent body scroll when in globe view (fullscreen)
  useEffect(() => {
    if (viewMode === 'globe') {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [viewMode]);

  const toggleViewMode = () => {
    const modes: ViewMode[] = ['grid', 'dual', 'list', 'compact'];
    const currentIndex = modes.indexOf(viewMode);
    // If current mode is globe or not in the list, start from grid
    const nextIndex = currentIndex === -1 ? 0 : (currentIndex + 1) % modes.length;
    const newMode = modes[nextIndex];
    setViewMode(newMode);
    localStorage.setItem('vstats-view-mode', newMode);
  };

  const setGlobeView = () => {
    setViewMode('globe');
    localStorage.setItem('vstats-view-mode', 'globe');
  };

  const onlineCount = servers.filter(s => s.isConnected).length;
  const totalBandwidthRx = servers.reduce((acc, s) => acc + s.speed.rx_sec, 0);
  const totalBandwidthTx = servers.reduce((acc, s) => acc + s.speed.tx_sec, 0);

  const showSkeleton = isInitialLoad && servers.length === 0;

  // Get the selected dimension (or 'tag' for tag grouping)
  const isGroupingByTag = selectedDimensionId === 'tag';
  const selectedDimension = isGroupingByTag ? null : enabledDimensions.find(d => d.id === selectedDimensionId) || null;
  
  // Handle dimension selection
  const handleDimensionSelect = (dimId: string | null) => {
    setSelectedDimensionId(dimId);
    if (dimId) {
      localStorage.setItem('vstats-group-dimension', dimId);
    } else {
      localStorage.removeItem('vstats-group-dimension');
    }
  };
  
  // Organize servers by selected dimension or tag
  const serversByOption = new Map<string | null, typeof servers>();
  
  if (isGroupingByTag) {
    // Group by tag
    const tagMap = new Map<string, typeof servers>();
    
    for (const server of servers) {
      const tag = server.config.tag || '';
      const tagKey = tag.trim() || t('dashboard.untagged');
      if (!tagMap.has(tagKey)) {
        tagMap.set(tagKey, []);
      }
      tagMap.get(tagKey)!.push(server);
    }
    
    // Convert tag map to serversByOption
    const sortedTags = Array.from(tagMap.keys()).sort();
    for (const tag of sortedTags) {
      serversByOption.set(tag, tagMap.get(tag)!);
    }
  } else if (selectedDimension) {
    // Group by dimension
    const sortedOptions: GroupOption[] = [...selectedDimension.options].sort((a, b) => a.sort_order - b.sort_order);
    
    // Initialize options
    for (const option of sortedOptions) {
      serversByOption.set(option.id, []);
    }
    serversByOption.set(null, []); // Ungrouped/Unassigned
    
    // Distribute servers to options based on selected dimension
    for (const server of servers) {
      const optionId = server.config.group_values?.[selectedDimension.id] || null;
      if (serversByOption.has(optionId)) {
        serversByOption.get(optionId)!.push(server);
      } else {
        // Option doesn't exist (shouldn't happen normally)
        serversByOption.get(null)!.push(server);
      }
    }
  } else {
    // No grouping, all servers go to ungrouped
    serversByOption.set(null, [...servers]);
  }
  
  // Check if we have any grouped servers
  const hasGroupedServers = (isGroupingByTag || selectedDimension) && Array.from(serversByOption.keys()).some(key => key !== null && (serversByOption.get(key)?.length || 0) > 0);
  const ungroupedServers = serversByOption.get(null) || [];
  
  // Get sorted options for display (for dimension grouping)
  const sortedOptions: GroupOption[] = selectedDimension 
    ? [...selectedDimension.options].sort((a, b) => a.sort_order - b.sort_order)
    : [];
  
  // Get sorted tags for display (for tag grouping)
  const sortedTags = isGroupingByTag 
    ? Array.from(serversByOption.keys()).filter(key => key !== null).sort() as string[]
    : [];

  // Render server cards based on view mode
  const renderServerCard = (server: typeof servers[0], index: number) => {
    const commonProps = {
      key: server.config.id,
      server,
      onClick: () => navigate(`/server/${server.config.id}`),
    };

    const wrapperStyle = { animationDelay: `${index * (viewMode === 'compact' ? 20 : 30)}ms` };

    if (viewMode === 'compact') {
      return (
        <div className="animate-fadeIn" style={wrapperStyle} key={server.config.id}>
          <VpsCompactCard {...commonProps} themeId={themeClass} />
        </div>
      );
    }

    if (viewMode === 'list') {
      return (
        <div className="animate-fadeIn" style={wrapperStyle} key={server.config.id}>
          <VpsListCard {...commonProps} isDark={isDark} />
        </div>
      );
    }

    if (viewMode === 'dual') {
      return (
        <div className="animate-fadeIn" style={wrapperStyle} key={server.config.id}>
          <VpsDualCard {...commonProps} isDark={isDark} />
        </div>
      );
    }

    // Default: grid
    return (
      <div className="animate-fadeIn" style={wrapperStyle} key={server.config.id}>
        <VpsGridCard {...commonProps} isDark={isDark} />
      </div>
    );
  };

  // Get grid classes based on view mode
  const getGridClasses = () => {
    if (viewMode === 'compact') return 'vps-compact-view';
    if (viewMode === 'list') return 'vps-list-view';
    if (viewMode === 'dual') return 'grid grid-cols-1 md:grid-cols-2 gap-4';
    return 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4';
  };

  return (
    <div className={`vps-page vps-page--${themeClass}${backgroundUrl ? ' has-bg-image' : ''}`}>
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
      
      {/* Background Blobs (only show if no background image) */}
      {!backgroundUrl && (
        <div className="vps-page-blobs">
          {isDark ? (
            <>
              <div className="vps-blobs-dark-1" />
              <div className="vps-blobs-dark-2" />
              <div className="vps-blobs-dark-3" />
            </>
          ) : (
            <>
              <div className="vps-blobs-light-1" />
              <div className="vps-blobs-light-2" />
              <div className="vps-blobs-light-3" />
            </>
          )}
        </div>
      )}

      <div className="vps-page-inner flex flex-col gap-6">
        {/* Header */}
        <header className={`flex items-center justify-between ${viewMode === 'globe' ? 'hidden' : ''}`}>
          <div className="flex items-center gap-4">
            <div>
              <h1 className={`text-xl md:text-2xl font-bold tracking-tight flex items-center gap-2 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                <span className="text-emerald-500">⚡</span> {siteSettings.site_name || t('dashboard.title')}
              </h1>
              <p className={`text-xs mt-0.5 font-mono ${isDark ? 'text-gray-500' : 'text-gray-400'}`}>
                {siteSettings.site_description || t('dashboard.subtitle')}
              </p>
            </div>
            <SocialLinks links={siteSettings.social_links} className="hidden sm:flex" isDark={isDark} />
          </div>
          <div className="flex items-center gap-3">
            {/* View Mode Toggle */}
            <button
              onClick={toggleViewMode}
              className={`vps-btn ${isDark ? 'vps-btn-outline-dark' : 'vps-btn-outline-light'} p-2.5`}
              title={`${t('dashboard.switchView')} (${viewMode === 'grid' ? t('dashboard.viewModeGrid') : viewMode === 'dual' ? t('dashboard.viewModeDual') || '双排' : viewMode === 'list' ? t('dashboard.viewModeList') : viewMode === 'compact' ? t('dashboard.viewModeCompact') : t('dashboard.viewModeGlobe') || 'Globe'})`}
            >
              {viewMode === 'grid' ? (
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
                </svg>
              ) : viewMode === 'dual' ? (
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 5a1 1 0 011-1h4a1 1 0 011 1v14a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM14 5a1 1 0 011-1h4a1 1 0 011 1v14a1 1 0 01-1 1h-4a1 1 0 01-1-1V5z" />
                </svg>
              ) : viewMode === 'list' ? (
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 6h16M4 12h16M4 18h16" />
                </svg>
              ) : viewMode === 'compact' ? (
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
                </svg>
              ) : (
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <circle cx="12" cy="12" r="9" strokeWidth={1.5} />
                  <ellipse cx="12" cy="12" rx="4" ry="9" strokeWidth={1.5} />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 12h18" />
                </svg>
              )}
            </button>
            {/* Globe View Button */}
            <button
              onClick={setGlobeView}
              className={`vps-btn ${viewMode === 'globe' ? (isDark ? 'vps-btn-accent-ok-dark' : 'vps-btn-accent-ok-light') : (isDark ? 'vps-btn-outline-dark' : 'vps-btn-outline-light')} p-2.5`}
              title={t('dashboard.viewModeGlobe') || '3D Globe View'}
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <circle cx="12" cy="12" r="9" strokeWidth={1.5} />
                <ellipse cx="12" cy="12" rx="4" ry="9" strokeWidth={1.5} />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 12h18" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 3c2.5 3 4 6 4 9s-1.5 6-4 9" />
              </svg>
            </button>
            {/* Language Switcher */}
            <LanguageSwitcher isDark={isDark} />
            <button
              onClick={() => navigate('/settings')}
              className={`vps-btn ${isDark ? 'vps-btn-outline-dark' : 'vps-btn-outline-light'} p-2.5`}
              title={t('dashboard.settings')}
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
            </button>
          </div>
        </header>

        {/* Overview Cards */}
        <div className={`grid grid-cols-2 md:grid-cols-4 gap-3 ${viewMode === 'globe' ? 'hidden' : ''}`}>
          <div className={`vps-overview-card vps-overview-card--online-${themeClass}`}>
            <div className="vps-overview-label vps-overview-label--online">{t('dashboard.online')}</div>
            <div className={`vps-overview-value vps-overview-value--${themeClass}`}>{onlineCount}</div>
          </div>
          <div className={`vps-overview-card vps-overview-card--offline-${themeClass}`}>
            <div className="vps-overview-label vps-overview-label--offline">{t('dashboard.offline')}</div>
            <div className={`vps-overview-value vps-overview-value--${themeClass}`}>{servers.length - onlineCount}</div>
          </div>
          <div className={`vps-overview-card vps-overview-card--download-${themeClass}`}>
            <div className="vps-overview-label vps-overview-label--download">↓ {t('dashboard.download')}</div>
            <div className={`vps-overview-value vps-overview-value--${themeClass} text-lg md:text-xl font-mono`}>{formatSpeed(totalBandwidthRx)}</div>
          </div>
          <div className={`vps-overview-card vps-overview-card--upload-${themeClass}`}>
            <div className="vps-overview-label vps-overview-label--upload">↑ {t('dashboard.upload')}</div>
            <div className={`vps-overview-value vps-overview-value--${themeClass} text-lg md:text-xl font-mono`}>{formatSpeed(totalBandwidthTx)}</div>
          </div>
        </div>

        {/* Dimension Selector */}
        {(enabledDimensions.length > 0 || servers.some(s => s.config.tag)) && (
          <div className={`flex items-center gap-2 flex-wrap ${viewMode === 'globe' ? 'hidden' : ''}`}>
            <span className={`text-xs font-medium ${isDark ? 'text-gray-500' : 'text-gray-400'}`}>{t('dashboard.groupBy')}</span>
            <button
              onClick={() => handleDimensionSelect(null)}
              className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${
                selectedDimensionId === null
                  ? isDark 
                    ? 'bg-emerald-500/20 text-emerald-400 ring-1 ring-emerald-500/30'
                    : 'bg-emerald-500/10 text-emerald-600 ring-1 ring-emerald-500/30'
                  : isDark
                    ? 'bg-gray-800/50 text-gray-400 hover:bg-gray-700/50'
                    : 'bg-gray-100 text-gray-500 hover:bg-gray-200'
              }`}
            >
              {t('common.all')}
            </button>
            {/* Tag grouping option */}
            {servers.some(s => s.config.tag) && (
              <button
                onClick={() => handleDimensionSelect('tag')}
                className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${
                  selectedDimensionId === 'tag'
                    ? isDark 
                      ? 'bg-emerald-500/20 text-emerald-400 ring-1 ring-emerald-500/30'
                      : 'bg-emerald-500/10 text-emerald-600 ring-1 ring-emerald-500/30'
                    : isDark
                      ? 'bg-gray-800/50 text-gray-400 hover:bg-gray-700/50'
                      : 'bg-gray-100 text-gray-500 hover:bg-gray-200'
                }`}
              >
                {t('dashboard.groupByTag') || '按标签分组'}
              </button>
            )}
            {enabledDimensions.map(dim => (
              <button
                key={dim.id}
                onClick={() => handleDimensionSelect(dim.id)}
                className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${
                  selectedDimensionId === dim.id
                    ? isDark 
                      ? 'bg-emerald-500/20 text-emerald-400 ring-1 ring-emerald-500/30'
                      : 'bg-emerald-500/10 text-emerald-600 ring-1 ring-emerald-500/30'
                    : isDark
                      ? 'bg-gray-800/50 text-gray-400 hover:bg-gray-700/50'
                      : 'bg-gray-100 text-gray-500 hover:bg-gray-200'
                }`}
              >
                {dim.name}
              </button>
            ))}
          </div>
        )}

        {/* Server List */}
        <div className="flex flex-col gap-3">
          {viewMode !== 'globe' && (
            <div className={`flex items-center justify-between px-1 text-[10px] font-bold uppercase tracking-wider ${isDark ? 'text-gray-600' : 'text-gray-400'}`}>
              <span>{t('dashboard.serverDetails')}</span>
              <span className={`font-mono ${isDark ? 'text-gray-700' : 'text-gray-400'}`}>{new Date().toLocaleTimeString()}</span>
            </div>
          )}
          
          {viewMode === 'globe' ? (
            <Suspense fallback={
              <div className="globe-container flex items-center justify-center">
                <div className={`text-center ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                  <svg className="animate-spin h-8 w-8 mx-auto mb-3" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  <span>{t('common.loading') || 'Loading 3D Globe...'}</span>
                </div>
              </div>
            }>
              <GlobeView 
                servers={servers} 
                onServerClick={(server) => navigate(`/server/${server.config.id}`)}
                onExitFullscreen={() => {
                  setViewMode('grid');
                  localStorage.setItem('vstats-view-mode', 'grid');
                }}
              />
            </Suspense>
          ) : showSkeleton ? (
            viewMode === 'list' ? (
              <div className="flex flex-col gap-3">
                {[1, 2, 3].map(i => <VpsListCardSkeleton key={i} isDark={isDark} />)}
              </div>
            ) : viewMode === 'compact' ? (
              <div className="vps-compact-table">
                <VpsCompactTableHeader themeId={themeClass} />
                <div className="vps-compact-body">
                  {[1, 2, 3, 4, 5].map(i => (
                    <VpsCompactRowSkeleton key={i} themeClass={themeClass} />
                  ))}
                </div>
              </div>
            ) : viewMode === 'dual' ? (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {[1, 2].map(i => <VpsDualCardSkeleton key={i} isDark={isDark} />)}
              </div>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                {[1, 2, 3, 4].map(i => <VpsGridCardSkeleton key={i} isDark={isDark} />)}
              </div>
            )
          ) : hasGroupedServers ? (
            // Display servers grouped by dimension options or tags
            <div className="space-y-6">
              {isGroupingByTag ? (
                // Group by tag
                sortedTags.map((tag) => {
                  const tagServers = serversByOption.get(tag) || [];
                  if (tagServers.length === 0) return null;
                  
                  return (
                    <div key={tag}>
                      {/* Tag Header */}
                      <div className={`flex items-center gap-2 mb-3 px-1`}>
                        <div className={`w-1.5 h-1.5 rounded-full ${isDark ? 'bg-orange-400' : 'bg-orange-500'}`} />
                        <span className={`text-sm font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>
                          {tag}
                        </span>
                        <span className={`text-xs ${isDark ? 'text-gray-600' : 'text-gray-400'}`}>
                          ({tagServers.length})
                        </span>
                      </div>
                      
                      {/* Tag Servers */}
                      <div className={getGridClasses()}>
                        {tagServers.map((server, index) => renderServerCard(server, index))}
                      </div>
                    </div>
                  );
                })
              ) : (
                // Group by dimension
                sortedOptions.map((option) => {
                  const optionServers = serversByOption.get(option.id) || [];
                  if (optionServers.length === 0) return null;
                  
                  return (
                    <div key={option.id}>
                      {/* Option Header */}
                      <div className={`flex items-center gap-2 mb-3 px-1`}>
                        <div className={`w-1.5 h-1.5 rounded-full ${isDark ? 'bg-orange-400' : 'bg-orange-500'}`} />
                        <span className={`text-sm font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>
                          {option.name}
                        </span>
                        <span className={`text-xs ${isDark ? 'text-gray-600' : 'text-gray-400'}`}>
                          ({optionServers.length})
                        </span>
                      </div>
                    
                      {/* Option Servers */}
                      <div className={getGridClasses()}>
                        {optionServers.map((server, index) => renderServerCard(server, index))}
                      </div>
                    </div>
                  );
                })
              )}
              
              {/* Unassigned Servers */}
              {ungroupedServers.length > 0 && (
                <div>
                  {/* Unassigned Header */}
                  <div className={`flex items-center gap-2 mb-3 px-1`}>
                    <div className={`w-1.5 h-1.5 rounded-full ${isDark ? 'bg-gray-500' : 'bg-gray-400'}`} />
                    <span className={`text-sm font-medium ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                      {t('common.unassigned')}
                    </span>
                    <span className={`text-xs ${isDark ? 'text-gray-600' : 'text-gray-400'}`}>
                      ({ungroupedServers.length})
                    </span>
                  </div>
                  
                  {/* Unassigned Servers */}
                  <div className={getGridClasses()}>
                    {ungroupedServers.map((server, index) => renderServerCard(server, index))}
                  </div>
                </div>
              )}
            </div>
          ) : viewMode === 'compact' ? (
            <div className="vps-compact-table">
              <VpsCompactTableHeader themeId={themeClass} />
              <div className="vps-compact-body">
                {servers.map((server, index) => renderServerCard(server, index))}
              </div>
            </div>
          ) : (
            <div className={getGridClasses()}>
              {servers.map((server, index) => renderServerCard(server, index))}
            </div>
          )}
        </div>

        {/* Footer */}
        <footer className="text-center mt-auto pt-6 pb-2">
          <p className={`text-[10px] font-mono ${isDark ? 'text-gray-600' : 'text-gray-400'}`}>
            vStats Monitor {serverVersion && `v${serverVersion}`}
            {serverVersion && ' · '}
            {onlineUsers > 0 && (
              <>
                <span className={`inline-flex items-center gap-1 ${isDark ? 'text-emerald-500' : 'text-emerald-600'}`}>
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                  {onlineUsers} {t('dashboard.onlineUsers')}
                </span>
                {' · '}
              </>
            )}
            {t('dashboard.madeWith')} <span className="text-red-500">❤️</span> {t('dashboard.by')}{' '}
            <a 
              href="https://vstats.zsoft.cc" 
              target="_blank" 
              rel="noopener noreferrer"
              className={`hover:underline ${isDark ? 'text-gray-500 hover:text-gray-400' : 'text-gray-400 hover:text-gray-500'}`}
            >
              vstats.zsoft.cc
            </a>
          </p>
        </footer>
      </div>
    </div>
  );
}

