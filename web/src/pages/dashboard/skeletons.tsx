export interface SkeletonProps {
  isDark: boolean;
}

export function VpsDualCardSkeleton({ isDark }: SkeletonProps) {
  const themeClass = isDark ? 'dark' : 'light';
  return (
    <div className={`vps-dual-card vps-dual-card--${themeClass} animate-pulse`}>
      <div className="vps-dual-header">
        <div className="w-2 h-2 rounded-full skeleton-bg" />
        <div className="h-4 skeleton-bg rounded w-32" />
      </div>
      <div className="vps-dual-metrics">
        {[1, 2, 3, 4, 5, 6].map(i => (
          <div key={i} className="vps-dual-metric">
            <div className="h-3 skeleton-bg rounded w-8 mb-1" />
            <div className="h-4 skeleton-bg rounded w-12" />
          </div>
        ))}
      </div>
      <div className="vps-dual-footer">
        <div className="h-6 skeleton-bg rounded flex-1" />
        <div className="h-6 skeleton-bg rounded flex-1" />
      </div>
    </div>
  );
}

export function VpsListCardSkeleton({ isDark }: SkeletonProps) {
  const themeClass = isDark ? 'dark' : 'light';
  return (
    <div className={`vps-card vps-card--${themeClass} p-4 md:p-5 flex flex-col lg:flex-row items-start lg:items-center gap-4 lg:gap-6 animate-pulse`}>
      <div className="w-full lg:w-56 shrink-0 flex items-center gap-3">
        <div className={`vps-card-avatar vps-card-avatar--${themeClass}`} />
        <div className="flex-1">
          <div className="h-4 skeleton-bg rounded w-32 mb-2" />
          <div className="h-3 skeleton-bg rounded w-24" />
        </div>
      </div>
      <div className="flex-1 w-full grid grid-cols-3 gap-3 lg:gap-6">
        {[1, 2, 3].map(i => (
          <div key={i} className="space-y-1">
            <div className="h-3 skeleton-bg rounded w-12" />
            <div className={`vps-resource-bar-track vps-resource-bar-track--${themeClass}`} />
          </div>
        ))}
      </div>
      <div className="w-full lg:w-40 flex flex-row lg:flex-col justify-between lg:justify-center items-end lg:items-end gap-1 shrink-0">
        <div className="h-4 skeleton-bg rounded w-16" />
        <div className="h-4 skeleton-bg rounded w-16" />
      </div>
    </div>
  );
}

export function VpsGridCardSkeleton({ isDark }: SkeletonProps) {
  const themeClass = isDark ? 'dark' : 'light';
  return (
    <div className={`vps-card vps-card--${themeClass} animate-pulse`}>
      <div className="vps-card-header">
        <div className="vps-card-identity">
          <div className={`vps-card-avatar vps-card-avatar--${themeClass}`} />
          <div className="vps-card-info space-y-2">
            <div className="h-4 skeleton-bg rounded w-3/4" />
            <div className="h-3 skeleton-bg rounded w-1/2" />
          </div>
        </div>
      </div>
      <div className="space-y-3 mt-4">
        {[1, 2, 3, 4].map(i => (
          <div key={i} className="vps-resource-row">
            <div className="flex justify-between mb-1">
              <div className="h-3 skeleton-bg rounded w-16" />
              <div className="h-3 skeleton-bg rounded w-10" />
            </div>
            <div className={`vps-resource-bar-track vps-resource-bar-track--${themeClass}`} />
          </div>
        ))}
      </div>
      <div className={`vps-divider vps-divider--${themeClass}`} />
      <div className="flex justify-between">
        <div className="h-3 skeleton-bg rounded w-24" />
        <div className="h-3 skeleton-bg rounded w-20" />
      </div>
    </div>
  );
}

export interface CompactSkeletonProps {
  themeClass: string;
}

export function VpsCompactRowSkeleton({ themeClass }: CompactSkeletonProps) {
  return (
    <div className={`vps-compact-row vps-compact-row--${themeClass} animate-pulse`}>
      <div className="vps-compact-col vps-compact-col--node">
        <div className="flex items-center gap-2">
          <div className="w-5 h-5 skeleton-bg rounded" />
          <div className="h-3 skeleton-bg rounded w-20" />
        </div>
      </div>
      <div className="vps-compact-col vps-compact-col--type"><div className="h-3 skeleton-bg rounded w-16" /></div>
      <div className="vps-compact-col vps-compact-col--uptime"><div className="h-3 skeleton-bg rounded w-14" /></div>
      <div className="vps-compact-col vps-compact-col--network"><div className="h-3 skeleton-bg rounded w-20" /></div>
      <div className="vps-compact-col vps-compact-col--traffic"><div className="h-3 skeleton-bg rounded w-24" /></div>
      <div className="vps-compact-col vps-compact-col--cpu"><div className="h-3 skeleton-bg rounded w-12" /></div>
      <div className="vps-compact-col vps-compact-col--mem"><div className="h-3 skeleton-bg rounded w-12" /></div>
      <div className="vps-compact-col vps-compact-col--hdd"><div className="h-3 skeleton-bg rounded w-12" /></div>
    </div>
  );
}

