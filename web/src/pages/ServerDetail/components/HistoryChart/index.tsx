import { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useTheme } from '../../../../context/ThemeContext';
import type { HistoryPoint, HistoryResponse, PingHistoryTarget } from '../../../../types';
import type { TimeRange, HistoryTab, TimeGridPoint } from '../../types';
import { 
  formatTimeForChart, 
  getGridConfigForRange, 
  getChartTheme, 
  getXAxisInterval 
} from '../../utils';
import { timeRanges, chartColors } from './chartConfig';
import { SingleLineChart } from './SingleLineChart';
import { MultiLineChart } from './MultiLineChart';
import { NetworkChart } from './NetworkChart';
import { PingChart } from './PingChart';

interface HistoryChartProps {
  serverId: string;
}

export function HistoryChart({ serverId }: HistoryChartProps) {
  const { t } = useTranslation();
  const { isDark } = useTheme();
  const isLight = !isDark;
  const [range, setRange] = useState<TimeRange>('24h');
  const [tab, setTab] = useState<HistoryTab>('ping'); // Default to ping tab
  const [data, setData] = useState<HistoryPoint[]>([]);
  const [pingTargets, setPingTargets] = useState<PingHistoryTarget[]>([]);
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  const [isFetching, setIsFetching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Track which data types have been loaded for current range
  const [loadedData, setLoadedData] = useState<{ range: string; ping: boolean; metrics: boolean }>({
    range: '',
    ping: false,
    metrics: false,
  });
  
  // Theme-aware colors for chart elements
  const chartTheme = getChartTheme(isLight);

  // Determine what data type is needed for current tab
  // Ping tab needs ping data for 1h/24h, but needs metrics data for 7d/30d/1y (fallback to ping_ms)
  const pingRangeSupported = range === '1h' || range === '24h';
  const needsPingData = tab === 'ping' && pingRangeSupported;
  const needsMetricsData = tab !== 'ping' || (tab === 'ping' && !pingRangeSupported);

  useEffect(() => {
    // Reset loaded data state when range changes
    if (loadedData.range !== range) {
      setLoadedData({ range, ping: false, metrics: false });
      setData([]);
      setPingTargets([]);
    }
  }, [range, loadedData.range]);

  // Fetch history data function
  const fetchHistory = async (isRefresh = false) => {
    // Determine what to fetch based on current tab and what's already loaded
    const shouldFetchPing = needsPingData && (!loadedData.ping || isRefresh) && loadedData.range === range;
    const shouldFetchMetrics = needsMetricsData && (!loadedData.metrics || isRefresh) && loadedData.range === range;
    
    // Skip if we already have the data we need (and not a refresh)
    if (!shouldFetchPing && !shouldFetchMetrics && loadedData.range === range && !isRefresh) {
      return;
    }

    // Determine the type parameter
    let dataType = 'all';
    if (shouldFetchPing && !shouldFetchMetrics) {
      dataType = 'ping';
    } else if (shouldFetchMetrics && !shouldFetchPing) {
      dataType = 'metrics';
    }

    // Only show loading spinner on initial load, not on refresh
    if (isInitialLoad || (!isRefresh && (shouldFetchPing || shouldFetchMetrics))) {
      setIsFetching(true);
    }
    setError(null);
    
    try {
      const res = await fetch(`/api/history/${serverId}?range=${range}&type=${dataType}`);
      if (!res.ok) throw new Error('Failed to fetch history');
      const json: HistoryResponse = await res.json();
      
      // Update data based on what was fetched
      if (dataType === 'ping' || dataType === 'all') {
        setPingTargets(json.ping_targets || []);
        setLoadedData(prev => ({ ...prev, ping: true }));
      }
      if (dataType === 'metrics' || dataType === 'all') {
        setData(json.data || []);
        setLoadedData(prev => ({ ...prev, metrics: true }));
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setIsFetching(false);
      setIsInitialLoad(false);
    }
  };

  // Initial fetch when dependencies change
  useEffect(() => {
    fetchHistory(false);
  }, [serverId, range, tab, needsPingData, needsMetricsData, loadedData.range]);

  // Auto-refresh interval based on range
  // 1h: refresh every 30 seconds
  // 24h: refresh every 2 minutes
  // 7d+: refresh every 5 minutes
  useEffect(() => {
    const refreshInterval = range === '1h' ? 30 * 1000 : range === '24h' ? 2 * 60 * 1000 : 5 * 60 * 1000;
    
    const intervalId = setInterval(() => {
      fetchHistory(true);
    }, refreshInterval);
    
    return () => clearInterval(intervalId);
  }, [serverId, range, tab, needsPingData, needsMetricsData]);

  const tabs: { value: HistoryTab; label: string; color: string }[] = [
    { value: 'ping', label: t('serverDetail.history.ping'), color: 'rose' },
    { value: 'overview', label: t('serverDetail.history.overview'), color: 'emerald' },
    { value: 'cpu', label: t('serverDetail.history.cpu'), color: 'blue' },
    { value: 'memory', label: t('serverDetail.history.memory'), color: 'purple' },
    { value: 'disk', label: t('serverDetail.history.disk'), color: 'amber' },
    { value: 'network', label: t('serverDetail.history.network'), color: 'cyan' },
  ];

  // Get grid configuration for current range
  const gridConfig = useMemo(() => getGridConfigForRange(range), [range]);

  // Calculate X-axis interval
  const xAxisInterval = useMemo(() => getXAxisInterval(gridConfig.points), [gridConfig.points]);

  // Generate fixed time grid and merge with fetched data
  // Left = old, Right = new (current time)
  const sampledData = useMemo(() => {
    const { points, interval, startTime } = gridConfig;
    const now = Date.now();
    
    // Create time grid from startTime to now
    const timeGrid: TimeGridPoint[] = [];
    
    for (let i = 0; i < points; i++) {
      const gridTime = startTime + i * interval;
      const timestamp = new Date(gridTime).toISOString();
      timeGrid.push({
        timestamp,
        gridTime,
        cpu: null,
        memory: null,
        disk: null,
        net_rx: null,
        net_tx: null,
        ping_ms: null,
        index: i,
        formattedTime: formatTimeForChart(timestamp, range),
        hasData: false,
      });
    }
    
    // If no data fetched, return the empty grid with null values
    if (data.length === 0) {
      return timeGrid;
    }
    
    // Find the earliest data point to know when data collection started
    const sortedData = [...data].sort((a, b) => 
      new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    );
    const earliestDataTime = new Date(sortedData[0].timestamp).getTime();
    
    // Merge fetched data into time grid
    // For each grid point, find the closest data point within half the interval
    const halfInterval = interval / 2;
    
    for (let i = 0; i < timeGrid.length; i++) {
      const gridPoint = timeGrid[i];
      const gridTime = gridPoint.gridTime;
      
      // If this grid point is before data collection started, leave as null (blank)
      if (gridTime < earliestDataTime - halfInterval) {
        continue;
      }
      
      // If this grid point is in the future (beyond now), leave as null (blank)
      if (gridTime > now + halfInterval) {
        continue;
      }
      
      // Find the closest data point to this grid time
      let closestPoint: HistoryPoint | null = null;
      let minDistance = Infinity;
      
      for (const dataPoint of data) {
        const dataTime = new Date(dataPoint.timestamp).getTime();
        const distance = Math.abs(dataTime - gridTime);
        
        if (distance < minDistance && distance <= halfInterval) {
          minDistance = distance;
          closestPoint = dataPoint;
        }
      }
      
      if (closestPoint) {
        // Data exists for this time slot
        timeGrid[i] = {
          ...gridPoint,
          cpu: closestPoint.cpu,
          memory: closestPoint.memory,
          disk: closestPoint.disk,
          net_rx: closestPoint.net_rx,
          net_tx: closestPoint.net_tx,
          ping_ms: closestPoint.ping_ms ?? null,
          hasData: true,
        };
      } else {
        // No data for this time slot but within collection period - set to 0
        timeGrid[i] = {
          ...gridPoint,
          cpu: 0,
          memory: 0,
          disk: 0,
          net_rx: 0,
          net_tx: 0,
          ping_ms: 0,
          hasData: true,
        };
      }
    }
    
    return timeGrid;
  }, [data, gridConfig, range]);

  // Render content based on tab
  const renderContent = () => {
    // Show loading spinner when fetching data for current tab
    const isLoadingCurrentTab = isFetching && (
      (needsPingData && !loadedData.ping) || 
      (needsMetricsData && !loadedData.metrics)
    );

    if (isLoadingCurrentTab) {
      return (
        <div className="h-64 flex items-center justify-center">
          <div className="w-8 h-8 border-2 border-white/20 border-t-emerald-500 rounded-full animate-spin" />
        </div>
      );
    }

    if (error) {
      return (
        <div className="h-64 flex items-center justify-center text-gray-500 text-sm">
          {error}
        </div>
      );
    }

    // Check for no data based on current tab
    // For ping tab with unsupported range, check metrics data instead
    const hasNoData = tab === 'ping' 
      ? (pingRangeSupported ? pingTargets.length === 0 : data.length === 0)
      : data.length === 0;

    if (hasNoData && !isFetching) {
      return (
        <div className="h-64 flex items-center justify-center text-gray-500 text-sm">
          {t('serverDetail.history.noData')}
        </div>
      );
    }

    const opacity = isFetching ? 'opacity-50' : 'opacity-100';

    switch (tab) {
      case 'overview':
        return (
          <div className={`transition-opacity ${opacity}`}>
            <MultiLineChart 
              data={sampledData}
              lines={[
                { dataKey: 'cpu', color: 'blue', label: 'CPU %' },
                { dataKey: 'memory', color: 'purple', label: 'Memory %' },
                { dataKey: 'disk', color: 'amber', label: 'Disk %' },
              ]}
              formatValue={v => `${v.toFixed(0)}%`}
              maxValue={100}
              chartTheme={chartTheme}
              xAxisInterval={xAxisInterval}
              isLight={isLight}
            />
            <div className="mt-4 grid grid-cols-3 gap-4">
              {[
                { key: 'cpu', color: 'blue', label: 'CPU' },
                { key: 'memory', color: 'purple', label: 'Memory' },
                { key: 'disk', color: 'amber', label: 'Disk' },
              ].map(({ key, color, label }) => {
                const validValues = sampledData
                  .filter(d => d.hasData && d[key as keyof typeof d] !== null)
                  .map(d => d[key as keyof typeof d] as number);
                const avg = validValues.length > 0 ? validValues.reduce((a, b) => a + b, 0) / validValues.length : 0;
                return (
                  <div key={key} className={`p-3 rounded-lg bg-${color}-500/10 border border-${color}-500/20`}>
                    <div className="text-[10px] text-gray-500 uppercase mb-1">{label} Avg</div>
                    <div className={`text-xl font-mono font-bold`} style={{ color: chartColors[color as keyof typeof chartColors].stroke }}>
                      {avg.toFixed(1)}%
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        );

      case 'cpu': {
        const cpuValues = sampledData.filter(d => d.hasData && d.cpu !== null).map(d => d.cpu as number);
        const cpuMin = cpuValues.length > 0 ? Math.min(...cpuValues) : 0;
        const cpuAvg = cpuValues.length > 0 ? cpuValues.reduce((a, b) => a + b, 0) / cpuValues.length : 0;
        const cpuMax = cpuValues.length > 0 ? Math.max(...cpuValues) : 0;
        return (
          <div className={`transition-opacity ${opacity}`}>
            <SingleLineChart 
              data={sampledData}
              dataKey="cpu"
              color="blue" 
              label="CPU Usage"
              formatValue={v => `${v.toFixed(1)}%`}
              maxValue={100}
              chartTheme={chartTheme}
              xAxisInterval={xAxisInterval}
              isLight={isLight}
            />
            <div className="mt-6 grid grid-cols-3 gap-4 text-center">
              <div className="p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
                <div className="text-[10px] text-gray-500 uppercase">Min</div>
                <div className="text-lg font-mono text-blue-400">
                  {cpuMin.toFixed(1)}%
                </div>
              </div>
              <div className="p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
                <div className="text-[10px] text-gray-500 uppercase">Avg</div>
                <div className="text-lg font-mono text-blue-400">
                  {cpuAvg.toFixed(1)}%
                </div>
              </div>
              <div className="p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
                <div className="text-[10px] text-gray-500 uppercase">Max</div>
                <div className="text-lg font-mono text-blue-400">
                  {cpuMax.toFixed(1)}%
                </div>
              </div>
            </div>
          </div>
        );
      }

      case 'memory': {
        const memValues = sampledData.filter(d => d.hasData && d.memory !== null).map(d => d.memory as number);
        const memMin = memValues.length > 0 ? Math.min(...memValues) : 0;
        const memAvg = memValues.length > 0 ? memValues.reduce((a, b) => a + b, 0) / memValues.length : 0;
        const memMax = memValues.length > 0 ? Math.max(...memValues) : 0;
        return (
          <div className={`transition-opacity ${opacity}`}>
            <SingleLineChart 
              data={sampledData}
              dataKey="memory"
              color="purple" 
              label="Memory Usage"
              formatValue={v => `${v.toFixed(1)}%`}
              maxValue={100}
              chartTheme={chartTheme}
              xAxisInterval={xAxisInterval}
              isLight={isLight}
            />
            <div className="mt-6 grid grid-cols-3 gap-4 text-center">
              <div className="p-3 rounded-lg bg-purple-500/10 border border-purple-500/20">
                <div className="text-[10px] text-gray-500 uppercase">Min</div>
                <div className="text-lg font-mono text-purple-400">
                  {memMin.toFixed(1)}%
                </div>
              </div>
              <div className="p-3 rounded-lg bg-purple-500/10 border border-purple-500/20">
                <div className="text-[10px] text-gray-500 uppercase">Avg</div>
                <div className="text-lg font-mono text-purple-400">
                  {memAvg.toFixed(1)}%
                </div>
              </div>
              <div className="p-3 rounded-lg bg-purple-500/10 border border-purple-500/20">
                <div className="text-[10px] text-gray-500 uppercase">Max</div>
                <div className="text-lg font-mono text-purple-400">
                  {memMax.toFixed(1)}%
                </div>
              </div>
            </div>
          </div>
        );
      }

      case 'disk': {
        const diskValues = sampledData.filter(d => d.hasData && d.disk !== null).map(d => d.disk as number);
        const diskMin = diskValues.length > 0 ? Math.min(...diskValues) : 0;
        const diskAvg = diskValues.length > 0 ? diskValues.reduce((a, b) => a + b, 0) / diskValues.length : 0;
        const diskMax = diskValues.length > 0 ? Math.max(...diskValues) : 0;
        return (
          <div className={`transition-opacity ${opacity}`}>
            <SingleLineChart 
              data={sampledData}
              dataKey="disk"
              color="amber" 
              label="Disk Usage"
              formatValue={v => `${v.toFixed(1)}%`}
              maxValue={100}
              chartTheme={chartTheme}
              xAxisInterval={xAxisInterval}
              isLight={isLight}
            />
            <div className="mt-6 grid grid-cols-3 gap-4 text-center">
              <div className="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
                <div className="text-[10px] text-gray-500 uppercase">Min</div>
                <div className="text-lg font-mono text-amber-400">
                  {diskMin.toFixed(1)}%
                </div>
              </div>
              <div className="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
                <div className="text-[10px] text-gray-500 uppercase">Avg</div>
                <div className="text-lg font-mono text-amber-400">
                  {diskAvg.toFixed(1)}%
                </div>
              </div>
              <div className="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
                <div className="text-[10px] text-gray-500 uppercase">Max</div>
                <div className="text-lg font-mono text-amber-400">
                  {diskMax.toFixed(1)}%
                </div>
              </div>
            </div>
          </div>
        );
      }

      case 'network':
        return (
          <div className={`transition-opacity ${opacity}`}>
            <NetworkChart 
              data={sampledData}
              chartTheme={chartTheme}
              xAxisInterval={xAxisInterval}
              isLight={isLight}
            />
          </div>
        );

      case 'ping':
        return (
          <div className={`transition-opacity ${opacity}`}>
            <PingChart
              data={sampledData}
              pingTargets={pingTargets}
              chartTheme={chartTheme}
              xAxisInterval={xAxisInterval}
              isLight={isLight}
              range={range}
              gridConfig={gridConfig}
            />
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div className="nezha-card p-6">
      {/* Header with Tabs */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-6">
        <div className="flex flex-wrap gap-1 p-1 bg-white/5 rounded-lg">
          {tabs.map(t => (
            <button
              key={t.value}
              onClick={() => setTab(t.value)}
              className={`px-3 py-1.5 text-xs font-medium rounded-md transition-all ${
                tab === t.value
                  ? `bg-${t.color}-500 text-white`
                  : 'text-gray-400 hover:text-white hover:bg-white/10'
              }`}
              style={tab === t.value ? { backgroundColor: `var(--${t.color}-500, #10b981)` } : {}}
            >
              {t.label}
            </button>
          ))}
        </div>
        <div className="flex gap-1 p-1 bg-white/5 rounded-lg">
          {timeRanges.map(r => (
            <button
              key={r.value}
              onClick={() => setRange(r.value)}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-all ${
                range === r.value
                  ? 'bg-emerald-500 text-white'
                  : 'text-gray-400 hover:text-white hover:bg-white/10'
              }`}
            >
              {r.label}
            </button>
          ))}
        </div>
      </div>

      {renderContent()}
    </div>
  );
}

