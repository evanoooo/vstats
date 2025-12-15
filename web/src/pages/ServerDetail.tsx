import { useState, useEffect, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useServerManager, formatBytes, formatSpeed, formatUptime } from '../hooks/useMetrics';
import { getOsIcon, getProviderIcon } from '../components/Icons';
import { getProviderLogo, getDistributionLogo, LogoImage } from '../utils/logoUtils';
import { useTheme } from '../context/ThemeContext';
import type { HistoryPoint, HistoryResponse, PingHistoryTarget } from '../types';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';

// Convert ISO 3166-1 alpha-2 country code to flag emoji
// Each letter becomes a regional indicator symbol (A=🇦, B=🇧, etc.)
const getFlag = (code: string): string => {
  if (!code || code.length !== 2) return '🌍';
  const upper = code.toUpperCase();
  // Regional indicator symbols start at 0x1F1E6 for 'A'
  const offset = 0x1F1E6 - 65; // 65 is char code for 'A'
  try {
    return String.fromCodePoint(
      upper.charCodeAt(0) + offset,
      upper.charCodeAt(1) + offset
    );
  } catch {
    return '🌍';
  }
};

function StatCard({ label, value, subValue, color = 'gray' }: { label: string; value: string; subValue?: string; color?: string }) {
  const colorClasses: Record<string, string> = {
    emerald: 'from-emerald-500/20 border-emerald-500/30',
    blue: 'from-blue-500/20 border-blue-500/30',
    purple: 'from-purple-500/20 border-purple-500/30',
    amber: 'from-amber-500/20 border-amber-500/30',
    gray: 'from-white/5 border-white/10',
  };

  return (
    <div className={`nezha-card p-4 bg-gradient-to-br ${colorClasses[color]} to-transparent`}>
      <div className="text-[10px] uppercase tracking-wider text-gray-400 mb-1">{label}</div>
      <div className="text-xl font-bold text-white font-mono">{value}</div>
      {subValue && <div className="text-xs text-gray-500 mt-1">{subValue}</div>}
    </div>
  );
}

// History Chart Component
type TimeRange = '1h' | '24h' | '7d' | '30d' | '1y';
type HistoryTab = 'overview' | 'cpu' | 'memory' | 'disk' | 'network' | 'ping';

// Color palette for charts
const chartColors = {
  blue: { stroke: '#3b82f6', fill: '#3b82f6', gradient: ['#3b82f6', '#06b6d4'] },
  purple: { stroke: '#a855f7', fill: '#a855f7', gradient: ['#a855f7', '#ec4899'] },
  amber: { stroke: '#f59e0b', fill: '#f59e0b', gradient: ['#f59e0b', '#ef4444'] },
  cyan: { stroke: '#06b6d4', fill: '#06b6d4', gradient: ['#06b6d4', '#3b82f6'] },
  rose: { stroke: '#f43f5e', fill: '#f43f5e', gradient: ['#f43f5e', '#a855f7'] },
  emerald: { stroke: '#10b981', fill: '#10b981', gradient: ['#10b981', '#06b6d4'] },
};

// Custom tooltip component
const CustomTooltip = ({ 
  active, 
  payload, 
  label, 
  formatValue,
  labelFormatter,
  isLight,
}: {
  active?: boolean;
  payload?: Array<{ value: number; name: string; color: string }>;
  label?: string;
  formatValue?: (v: number) => string;
  labelFormatter?: (label: string) => string;
  isLight?: boolean;
}) => {
  if (!active || !payload?.length) return null;
  
  return (
    <div className={`backdrop-blur-sm rounded-lg p-3 shadow-xl ${
      isLight 
        ? 'bg-white/95 border border-gray-200' 
        : 'bg-gray-900/95 border border-white/10'
    }`}>
      <p className={`text-xs mb-2 font-mono ${isLight ? 'text-gray-500' : 'text-gray-400'}`}>
        {labelFormatter ? labelFormatter(label || '') : label}
      </p>
      {payload.map((entry, index) => (
        <div key={index} className="flex items-center gap-2">
          <span 
            className="w-2 h-2 rounded-full" 
            style={{ backgroundColor: entry.color }}
          />
          <span className={`text-xs ${isLight ? 'text-gray-600' : 'text-gray-300'}`}>{entry.name}:</span>
          <span className={`text-sm font-mono font-semibold ${isLight ? 'text-gray-900' : 'text-white'}`}>
            {formatValue ? formatValue(entry.value) : entry.value}
          </span>
        </div>
      ))}
    </div>
  );
};

function HistoryChart({ serverId }: { serverId: string }) {
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
  const chartTheme = {
    gridColor: isLight ? 'rgba(0,0,0,0.06)' : 'rgba(255,255,255,0.05)',
    tickColor: isLight ? '#374151' : '#6b7280',
    legendColor: isLight ? '#4b5563' : '#9ca3af',
  };

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

  const ranges: { value: TimeRange; label: string }[] = [
    { value: '1h', label: '1H' },
    { value: '24h', label: '24H' },
    { value: '7d', label: '7D' },
    { value: '30d', label: '30D' },
    { value: '1y', label: '1Y' },
  ];

  const tabs: { value: HistoryTab; label: string; color: string }[] = [
    { value: 'ping', label: t('serverDetail.history.ping'), color: 'rose' },
    { value: 'overview', label: t('serverDetail.history.overview'), color: 'emerald' },
    { value: 'cpu', label: t('serverDetail.history.cpu'), color: 'blue' },
    { value: 'memory', label: t('serverDetail.history.memory'), color: 'purple' },
    { value: 'disk', label: t('serverDetail.history.disk'), color: 'amber' },
    { value: 'network', label: t('serverDetail.history.network'), color: 'cyan' },
  ];

  // Number of points for each time range (matching backend bucket sizes)
  const getGridConfig = useMemo(() => {
    const now = Date.now();
    switch (range) {
      case '1h': 
        return { 
          points: 720, 
          duration: 60 * 60 * 1000, // 1 hour in ms
          interval: 5 * 1000, // 5 seconds per point (720 points)
          startTime: now - 60 * 60 * 1000
        };
      case '24h': 
        return { 
          points: 720, 
          duration: 24 * 60 * 60 * 1000, // 24 hours in ms
          interval: 2 * 60 * 1000, // 2 minutes per point (720 points)
          startTime: now - 24 * 60 * 60 * 1000
        };
      case '7d': 
        return { 
          points: 672, 
          duration: 7 * 24 * 60 * 60 * 1000, // 7 days in ms
          interval: 15 * 60 * 1000, // 15 minutes per point (672 points)
          startTime: now - 7 * 24 * 60 * 60 * 1000
        };
      case '30d': 
        return { 
          points: 720, 
          duration: 30 * 24 * 60 * 60 * 1000, // 30 days in ms
          interval: 60 * 60 * 1000, // 1 hour per point (720 points)
          startTime: now - 30 * 24 * 60 * 60 * 1000
        };
      case '1y': 
        return { 
          points: 730, 
          duration: 365 * 24 * 60 * 60 * 1000, // 1 year in ms
          interval: 12 * 60 * 60 * 1000, // 12 hours per point (730 points)
          startTime: now - 365 * 24 * 60 * 60 * 1000
        };
      default: 
        return { 
          points: 720, 
          duration: 24 * 60 * 60 * 1000,
          interval: 2 * 60 * 1000,
          startTime: now - 24 * 60 * 60 * 1000
        };
    }
  }, [range]);

  // Generate fixed time grid and merge with fetched data
  // Left = old, Right = new (current time)
  const sampledData = useMemo(() => {
    const { points, interval, startTime } = getGridConfig;
    const now = Date.now();
    
    // Create time grid from startTime to now
    const timeGrid: Array<{
      timestamp: string;
      gridTime: number;
      cpu: number | null;
      memory: number | null;
      disk: number | null;
      net_rx: number | null;
      net_tx: number | null;
      ping_ms: number | null;
      index: number;
      formattedTime: string;
      hasData: boolean;
    }> = [];
    
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
        formattedTime: formatTimeForChart(timestamp),
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
  }, [data, getGridConfig]);

  function formatTimeForChart(timestamp: string) {
    const date = new Date(timestamp);
    switch (range) {
      case '1h':
        // 按分钟: "14:35"
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
      case '24h':
        // 按小时: "14:00"
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
      case '7d':
        // 按天+时间: "12/10 14:00"
        return date.toLocaleDateString([], { month: 'numeric', day: 'numeric' }) + ' ' + 
               date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
      case '30d':
        // 按天: "12/10" 或 "Dec 10"
        return date.toLocaleDateString([], { month: 'numeric', day: 'numeric' });
      case '1y':
        // 按月+日: "Dec 10"
        return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
      default:
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }
  }

  function formatFullTime(timestamp: string) {
    const date = new Date(timestamp);
    return date.toLocaleString([], { 
      month: 'short', 
      day: 'numeric',
      hour: '2-digit', 
      minute: '2-digit'
    });
  }

  const formatBytesLocal = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
  };

  // Calculate tick interval to show evenly spaced labels on X axis
  // Show about 10-12 labels for better readability
  const xAxisInterval = useMemo(() => {
    const { points } = getGridConfig;
    if (points <= 12) return 0; // Show all labels if few points
    // For 720 points, show ~12 labels (interval = 60)
    return Math.floor(points / 12) - 1;
  }, [getGridConfig]);

  // Single Line Chart Component (no animation, no fill)
  const SingleLineChart = ({ 
    dataKey, 
    color,
    label,
    formatValue,
    maxValue,
  }: { 
    dataKey: string;
    color: keyof typeof chartColors;
    label: string;
    formatValue: (v: number) => string;
    maxValue?: number;
  }) => {
    const colorSet = chartColors[color];
    // Only include points with actual data (hasData=true and value is not null) for statistics
    const validValues = sampledData
      .filter(d => d.hasData && d[dataKey as keyof typeof d] !== null)
      .map(d => d[dataKey as keyof typeof d] as number);
    const avg = validValues.length > 0 ? validValues.reduce((a, b) => a + b, 0) / validValues.length : 0;
    const min = validValues.length > 0 ? Math.min(...validValues) : 0;
    const max = validValues.length > 0 ? Math.max(...validValues) : 0;

    return (
      <div className="relative">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <span 
              className="w-3 h-3 rounded-full" 
              style={{ backgroundColor: colorSet.stroke }}
            />
            <span className={`text-sm font-medium ${isLight ? 'text-gray-900' : 'text-white'}`}>{label}</span>
          </div>
          <div className="flex gap-4 text-xs">
            <span className={isLight ? 'text-gray-500' : 'text-gray-500'}>min: <span className="text-emerald-500 font-mono">{formatValue(min)}</span></span>
            <span className={isLight ? 'text-gray-500' : 'text-gray-500'}>avg: <span style={{ color: colorSet.stroke }} className="font-mono">{formatValue(avg)}</span></span>
            <span className={isLight ? 'text-gray-500' : 'text-gray-500'}>max: <span className="text-amber-500 font-mono">{formatValue(max)}</span></span>
          </div>
        </div>
        <div className="h-48 w-full">
          <ResponsiveContainer width="100%" height="100%" minWidth={0} minHeight={0}>
            <LineChart data={sampledData} margin={{ top: 5, right: 5, left: -15, bottom: 5 }}>
              <CartesianGrid 
                strokeDasharray="3 3" 
                stroke={chartTheme.gridColor} 
                vertical={false}
              />
              <XAxis 
                dataKey="formattedTime"
                axisLine={false}
                tickLine={false}
                tick={{ fill: chartTheme.tickColor, fontSize: 10 }}
                interval={xAxisInterval}
                tickFormatter={(value) => value || ''}
              />
              <YAxis
                domain={maxValue ? [0, maxValue] : ['auto', 'auto']}
                axisLine={false}
                tickLine={false}
                tick={{ fill: chartTheme.tickColor, fontSize: 10 }}
                tickFormatter={formatValue}
                width={55}
              />
              <Tooltip
                content={
                  <CustomTooltip 
                    formatValue={formatValue}
                    isLight={isLight}
                    labelFormatter={(label) => {
                      const point = sampledData.find(d => d.formattedTime === label);
                      return point ? formatFullTime(point.timestamp) : label;
                    }}
                  />
                }
              />
              <Line
                type="monotone"
                dataKey={dataKey}
                stroke={colorSet.stroke}
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, strokeWidth: 2, stroke: colorSet.stroke, fill: isLight ? '#ffffff' : '#1f2937' }}
                name={label}
                isAnimationActive={false}
                connectNulls={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    );
  };

  // Multi-line Chart Component for Overview (no animation)
  const MultiLineChart = ({ 
    lines,
    formatValue,
    maxValue,
  }: { 
    lines: Array<{ dataKey: string; color: keyof typeof chartColors; label: string }>;
    formatValue: (v: number) => string;
    maxValue?: number;
  }) => {
    return (
      <div className="h-72 w-full">
        <ResponsiveContainer width="100%" height="100%" minWidth={0} minHeight={0}>
          <LineChart data={sampledData} margin={{ top: 5, right: 5, left: -15, bottom: 5 }}>
            <CartesianGrid 
              strokeDasharray="3 3" 
              stroke={chartTheme.gridColor} 
              vertical={false}
            />
            <XAxis 
              dataKey="formattedTime"
              axisLine={false}
              tickLine={false}
              tick={{ fill: chartTheme.tickColor, fontSize: 10 }}
              interval={xAxisInterval}
              tickFormatter={(value) => value || ''}
            />
            <YAxis
              domain={maxValue ? [0, maxValue] : ['auto', 'auto']}
              axisLine={false}
              tickLine={false}
              tick={{ fill: chartTheme.tickColor, fontSize: 10 }}
              tickFormatter={formatValue}
              width={50}
            />
            <Tooltip
              content={
                <CustomTooltip 
                  formatValue={formatValue}
                  isLight={isLight}
                  labelFormatter={(label) => {
                    const point = sampledData.find(d => d.formattedTime === label);
                    return point ? formatFullTime(point.timestamp) : label;
                  }}
                />
              }
            />
            <Legend 
              verticalAlign="top" 
              height={36}
              iconType="circle"
              iconSize={8}
              formatter={(value) => <span className="text-xs" style={{ color: chartTheme.legendColor }}>{value}</span>}
            />
            {lines.map(({ dataKey, color, label }) => (
              <Line
                key={dataKey}
                type="monotone"
                dataKey={dataKey}
                stroke={chartColors[color].stroke}
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, strokeWidth: 2, stroke: chartColors[color].stroke, fill: isLight ? '#ffffff' : '#1f2937' }}
                name={label}
                isAnimationActive={false}
                connectNulls={false}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>
    );
  };

  // Network Chart with dual lines (no animation, no fill)
  const NetworkChart = () => {
    return (
      <div className="space-y-6">
        <div className="h-56 w-full">
          <ResponsiveContainer width="100%" height="100%" minWidth={0} minHeight={0}>
            <LineChart data={sampledData} margin={{ top: 5, right: 5, left: -5, bottom: 5 }}>
              <CartesianGrid 
                strokeDasharray="3 3" 
                stroke={chartTheme.gridColor} 
                vertical={false}
              />
              <XAxis 
                dataKey="formattedTime"
                axisLine={false}
                tickLine={false}
                tick={{ fill: chartTheme.tickColor, fontSize: 10 }}
                interval={xAxisInterval}
                tickFormatter={(value) => value || ''}
              />
              <YAxis
                axisLine={false}
                tickLine={false}
                tick={{ fill: chartTheme.tickColor, fontSize: 10 }}
                tickFormatter={formatBytesLocal}
                width={60}
              />
              <Tooltip
                content={
                  <CustomTooltip 
                    formatValue={formatBytesLocal}
                    isLight={isLight}
                    labelFormatter={(label) => {
                      const point = sampledData.find(d => d.formattedTime === label);
                      return point ? formatFullTime(point.timestamp) : label;
                    }}
                  />
                }
              />
              <Legend 
                verticalAlign="top" 
                height={36}
                iconType="circle"
                iconSize={8}
                formatter={(value) => <span className="text-xs" style={{ color: chartTheme.legendColor }}>{value}</span>}
              />
              <Line
                type="monotone"
                dataKey="net_tx"
                stroke="#10b981"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, strokeWidth: 2, stroke: '#10b981', fill: isLight ? '#ffffff' : '#1f2937' }}
                name="Upload (TX)"
                isAnimationActive={false}
                connectNulls={false}
              />
              <Line
                type="monotone"
                dataKey="net_rx"
                stroke="#06b6d4"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, strokeWidth: 2, stroke: '#06b6d4', fill: isLight ? '#ffffff' : '#1f2937' }}
                name="Download (RX)"
                isAnimationActive={false}
                connectNulls={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
        <div className="grid grid-cols-2 gap-4 text-center">
          <div className="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
            <div className="text-[10px] text-gray-500 uppercase">{t('serverDetail.history.totalUpload')}</div>
            <div className="text-lg font-mono text-emerald-500">
              {formatBytesLocal(sampledData.filter(d => d.hasData && d.net_tx !== null).reduce((a, b) => a + (b.net_tx || 0), 0))}
            </div>
          </div>
          <div className="p-3 rounded-lg bg-cyan-500/10 border border-cyan-500/20">
            <div className="text-[10px] text-gray-500 uppercase">{t('serverDetail.history.totalDownload')}</div>
            <div className="text-lg font-mono text-cyan-500">
              {formatBytesLocal(sampledData.filter(d => d.hasData && d.net_rx !== null).reduce((a, b) => a + (b.net_rx || 0), 0))}
            </div>
          </div>
        </div>
      </div>
    );
  };

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
              lines={[
                { dataKey: 'cpu', color: 'blue', label: 'CPU %' },
                { dataKey: 'memory', color: 'purple', label: 'Memory %' },
                { dataKey: 'disk', color: 'amber', label: 'Disk %' },
              ]}
              formatValue={v => `${v.toFixed(0)}%`}
              maxValue={100}
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
              dataKey="cpu"
              color="blue" 
              label="CPU Usage"
              formatValue={v => `${v.toFixed(1)}%`}
              maxValue={100}
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
              dataKey="memory"
              color="purple" 
              label="Memory Usage"
              formatValue={v => `${v.toFixed(1)}%`}
              maxValue={100}
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
              dataKey="disk"
              color="amber" 
              label="Disk Usage"
              formatValue={v => `${v.toFixed(1)}%`}
              maxValue={100}
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
            <NetworkChart />
          </div>
        );

      case 'ping': {
        // Colors for different ping targets
        const pingColorKeys: (keyof typeof chartColors)[] = ['rose', 'cyan', 'amber', 'purple', 'emerald', 'blue'];
        
        // Generate fixed time grid for ping data
        const { points: pingPoints, interval: pingInterval, startTime: pingStartTime } = getGridConfig;
        const now = Date.now();
        
        // Check if we have detailed ping target data
        if (pingTargets.length > 0) {
          // Create fixed time grid for ping
          const pingTimeGrid: Array<{
            index: number;
            gridTime: number;
            timestamp: string;
            formattedTime: string;
            hasData: boolean;
            [key: string]: number | string | boolean | null;
          }> = [];
          
          for (let i = 0; i < pingPoints; i++) {
            const gridTime = pingStartTime + i * pingInterval;
            const timestamp = new Date(gridTime).toISOString();
            const point: typeof pingTimeGrid[0] = {
              index: i,
              gridTime,
              timestamp,
              formattedTime: formatTimeForChart(timestamp),
              hasData: false,
            };
            // Initialize all ping targets as null
            pingTargets.forEach((_, targetIdx) => {
              point[`ping_${targetIdx}`] = null;
            });
            pingTimeGrid.push(point);
          }
          
          // Find earliest data time from all targets
          let earliestPingTime = Infinity;
          pingTargets.forEach(target => {
            if (target.data.length > 0) {
              const firstTime = new Date(target.data[0].timestamp).getTime();
              if (firstTime < earliestPingTime) earliestPingTime = firstTime;
            }
          });
          
          // Merge ping data into time grid
          const halfInterval = pingInterval / 2;
          
          for (let i = 0; i < pingTimeGrid.length; i++) {
            const gridPoint = pingTimeGrid[i];
            const gridTime = gridPoint.gridTime;
            
            // If before data collection or after now, leave as null (blank)
            if (gridTime < earliestPingTime - halfInterval || gridTime > now + halfInterval) {
              continue;
            }
            
            // For each target, find closest data point
            pingTargets.forEach((target, targetIdx) => {
              let closestValue: number | null = null;
              let minDistance = Infinity;
              
              for (const dataPoint of target.data) {
                const dataTime = new Date(dataPoint.timestamp).getTime();
                const distance = Math.abs(dataTime - gridTime);
                
                if (distance < minDistance && distance <= halfInterval) {
                  minDistance = distance;
                  closestValue = dataPoint.latency_ms;
                }
              }
              
              if (closestValue !== null) {
                pingTimeGrid[i][`ping_${targetIdx}`] = closestValue;
                pingTimeGrid[i].hasData = true;
              } else if (gridTime >= earliestPingTime) {
                // Within collection period but no data - set to 0
                pingTimeGrid[i][`ping_${targetIdx}`] = 0;
                pingTimeGrid[i].hasData = true;
              }
            });
          }

          // Calculate stats for each target
          const targetStats = pingTargets.map(target => {
            const validData = target.data.filter(d => d.latency_ms !== null);
            if (validData.length === 0) return { min: 0, avg: 0, max: 0 };
            const values = validData.map(d => d.latency_ms!);
            return {
              min: Math.min(...values),
              avg: values.reduce((a, b) => a + b, 0) / values.length,
              max: Math.max(...values),
            };
          });

          // Custom tooltip for combined ping chart
          const PingCombinedTooltip = ({ 
            active, 
            payload, 
            label 
          }: {
            active?: boolean;
            payload?: Array<{ value: number; name: string; color: string; dataKey: string }>;
            label?: string;
          }) => {
            if (!active || !payload?.length) return null;
            
            const point = pingTimeGrid.find(d => d.formattedTime === label);
            const timeLabel = point ? formatFullTime(point.timestamp as string) : label;
            
            return (
              <div className={`backdrop-blur-sm rounded-lg p-3 shadow-xl min-w-[180px] ${
                isLight 
                  ? 'bg-white/95 border border-gray-200' 
                  : 'bg-gray-900/95 border border-white/10'
              }`}>
                <p className={`text-xs mb-2 font-mono border-b pb-2 ${
                  isLight ? 'text-gray-500 border-gray-200' : 'text-gray-400 border-white/10'
                }`}>
                  {timeLabel}
                </p>
                <div className="space-y-1.5">
                  {payload.map((entry, index) => {
                    const targetIdx = parseInt(entry.dataKey.replace('ping_', ''));
                    const target = pingTargets[targetIdx];
                    return (
                      <div key={index} className="flex items-center justify-between gap-4">
                        <div className="flex items-center gap-2">
                          <span 
                            className="w-2 h-2 rounded-full" 
                            style={{ backgroundColor: entry.color }}
                          />
                          <span className={`text-xs ${isLight ? 'text-gray-600' : 'text-gray-300'}`}>{target?.name || entry.name}</span>
                        </div>
                        <span className={`text-sm font-mono font-semibold ${isLight ? 'text-gray-900' : 'text-white'}`}>
                          {entry.value !== null ? `${entry.value.toFixed(1)}ms` : 'N/A'}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>
            );
          };
          
          return (
            <div className={`transition-opacity ${opacity}`}>
              {/* Combined chart */}
              <div className="h-56 w-full">
                <ResponsiveContainer width="100%" height="100%" minWidth={0} minHeight={0}>
                  <LineChart data={pingTimeGrid} margin={{ top: 5, right: 5, left: -15, bottom: 5 }}>
                    <CartesianGrid 
                      strokeDasharray="3 3" 
                      stroke={chartTheme.gridColor} 
                      vertical={false}
                    />
                    <XAxis 
                      dataKey="formattedTime"
                      axisLine={false}
                      tickLine={false}
                      tick={{ fill: chartTheme.tickColor, fontSize: 10 }}
                      interval={xAxisInterval}
                      tickFormatter={(value) => value || ''}
                    />
                    <YAxis
                      axisLine={false}
                      tickLine={false}
                      tick={{ fill: chartTheme.tickColor, fontSize: 10 }}
                      tickFormatter={v => `${v}ms`}
                      width={55}
                    />
                    <Tooltip content={<PingCombinedTooltip />} />
                    <Legend 
                      verticalAlign="top" 
                      height={36}
                      iconType="circle"
                      iconSize={8}
                      formatter={(value, entry) => {
                        const idx = parseInt((entry.dataKey as string).replace('ping_', ''));
                        return <span className="text-xs" style={{ color: chartTheme.legendColor }}>{pingTargets[idx]?.name || value}</span>;
                      }}
                    />
                    {pingTargets.map((target, idx) => {
                      const colorKey = pingColorKeys[idx % pingColorKeys.length];
                      const colorSet = chartColors[colorKey];
                      return (
                        <Line
                          key={target.name}
                          type="monotone"
                          dataKey={`ping_${idx}`}
                          stroke={colorSet.stroke}
                          strokeWidth={2}
                          dot={false}
                          activeDot={{ r: 4, strokeWidth: 2, stroke: colorSet.stroke, fill: isLight ? '#ffffff' : '#1f2937' }}
                          name={target.name}
                          connectNulls={false}
                          isAnimationActive={false}
                        />
                      );
                    })}
                  </LineChart>
                </ResponsiveContainer>
              </div>

              {/* Stats cards for each target */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mt-4">
                {pingTargets.map((target, idx) => {
                  const colorKey = pingColorKeys[idx % pingColorKeys.length];
                  const colorSet = chartColors[colorKey];
                  const stats = targetStats[idx];
                  
                  return (
                    <div 
                      key={target.name} 
                      className="p-3 rounded-lg border"
                      style={{ 
                        backgroundColor: `${colorSet.stroke}08`,
                        borderColor: `${colorSet.stroke}30`
                      }}
                    >
                      <div className="flex items-center gap-2 mb-2">
                        <span 
                          className="w-2 h-2 rounded-full" 
                          style={{ backgroundColor: colorSet.stroke }}
                        />
                        <span className={`text-xs font-medium truncate ${isLight ? 'text-gray-900' : 'text-white'}`}>{target.name}</span>
                        <span className="text-[10px] text-gray-500 font-mono truncate">({target.host})</span>
                      </div>
                      <div className="flex justify-between text-xs">
                        <span className="text-gray-500">
                          <span className="text-emerald-400 font-mono">{stats.min.toFixed(0)}</span>
                          <span className="mx-1">/</span>
                          <span style={{ color: colorSet.stroke }} className="font-mono">{stats.avg.toFixed(0)}</span>
                          <span className="mx-1">/</span>
                          <span className="text-amber-400 font-mono">{stats.max.toFixed(0)}</span>
                          <span className="ml-1 text-gray-600">ms</span>
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          );
        }
        
        // Fallback to aggregated ping_ms data from sampledData (which already uses fixed time grid)
        // Check for points that have actual ping data (non-zero, non-null values)
        const pingData = sampledData.filter(d => d.hasData && d.ping_ms !== null && d.ping_ms > 0);
        
        // If no valid ping data, still show the chart with the time grid (all zeros or nulls)
        const hasAnyPingData = pingData.length > 0;
        
        const pingValues = hasAnyPingData ? pingData.map(d => d.ping_ms!) : [0];
        const avgPing = hasAnyPingData ? pingValues.reduce((a, b) => a + b, 0) / pingValues.length : 0;
        const minPing = hasAnyPingData ? Math.min(...pingValues) : 0;
        const maxPing = hasAnyPingData ? Math.max(...pingValues) : 0;
        
        // Use sampledData which already has the fixed time grid
        const pingChartData = sampledData.map((d, i) => ({
          ...d,
          index: i,
          ping: d.hasData ? (d.ping_ms ?? 0) : null,
        }));
        
        return (
          <div className={`transition-opacity ${opacity}`}>
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <span className="w-3 h-3 rounded-full bg-rose-500" />
                <span className={`text-sm font-medium ${isLight ? 'text-gray-900' : 'text-white'}`}>Ping Latency (Average)</span>
              </div>
              {hasAnyPingData ? (
                <div className="flex gap-4 text-xs">
                  <span className="text-gray-500">min: <span className="text-emerald-500 font-mono">{minPing.toFixed(1)}ms</span></span>
                  <span className="text-gray-500">avg: <span className="text-rose-500 font-mono">{avgPing.toFixed(1)}ms</span></span>
                  <span className="text-gray-500">max: <span className="text-amber-500 font-mono">{maxPing.toFixed(1)}ms</span></span>
                </div>
              ) : (
                <div className="text-xs text-gray-500">No ping data in this period</div>
              )}
            </div>
            <div className="h-48 w-full">
              <ResponsiveContainer width="100%" height="100%" minWidth={0} minHeight={0}>
                <LineChart data={pingChartData} margin={{ top: 5, right: 5, left: -15, bottom: 5 }}>
                  <CartesianGrid 
                    strokeDasharray="3 3" 
                    stroke={chartTheme.gridColor} 
                    vertical={false}
                  />
                  <XAxis 
                    dataKey="formattedTime"
                    axisLine={false}
                    tickLine={false}
                    tick={{ fill: chartTheme.tickColor, fontSize: 10 }}
                    interval={xAxisInterval}
                    tickFormatter={(value) => value || ''}
                  />
                  <YAxis
                    axisLine={false}
                    tickLine={false}
                    tick={{ fill: chartTheme.tickColor, fontSize: 10 }}
                    tickFormatter={v => `${v}ms`}
                    width={55}
                  />
                  <Tooltip
                    content={
                      <CustomTooltip 
                        formatValue={v => `${v.toFixed(1)} ms`}
                        isLight={isLight}
                        labelFormatter={(label) => {
                          const point = pingChartData.find(d => d.formattedTime === label);
                          return point ? formatFullTime(point.timestamp) : label;
                        }}
                      />
                    }
                  />
                  <Line
                    type="monotone"
                    dataKey="ping"
                    stroke="#f43f5e"
                    strokeWidth={2}
                    dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: '#f43f5e', fill: isLight ? '#ffffff' : '#1f2937' }}
                    name="Latency"
                    isAnimationActive={false}
                    connectNulls={false}
                  />
                </LineChart>
              </ResponsiveContainer>
            </div>
            {hasAnyPingData && (
              <div className="mt-6 grid grid-cols-3 gap-4 text-center">
                <div className="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
                  <div className="text-[10px] text-gray-500 uppercase">Min</div>
                  <div className="text-lg font-mono text-emerald-400">
                    {minPing.toFixed(1)} ms
                  </div>
                </div>
                <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/20">
                  <div className="text-[10px] text-gray-500 uppercase">Avg</div>
                  <div className="text-lg font-mono text-rose-400">
                    {avgPing.toFixed(1)} ms
                  </div>
                </div>
                <div className="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
                  <div className="text-[10px] text-gray-500 uppercase">Max</div>
                  <div className="text-lg font-mono text-amber-400">
                    {maxPing.toFixed(1)} ms
                  </div>
                </div>
              </div>
            )}
          </div>
        );
      }

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
          {ranges.map(r => (
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

export default function ServerDetail() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { servers, loadingState, isInitialLoad } = useServerManager();
  const { isDark } = useTheme();
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
      <div className="min-h-screen flex items-center justify-center">
        <div className="flex flex-col items-center gap-4 animate-pulse">
          <div className="w-12 h-12 border-4 border-white/20 border-t-emerald-500 rounded-full animate-spin" />
          <div className="text-white/60 text-sm">{t('serverDetail.loadingServerData')}</div>
        </div>
      </div>
    );
  }

  // If server not found after data is loaded, show a brief delay before showing error
  // This prevents flash of "not found" during navigation
  if (!server) {
    return (
      <div className="min-h-screen flex items-center justify-center animate-fadeIn">
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
    );
  }

  const { metrics, speed, isConnected, config } = server;

  // Show connecting state if metrics not yet available
  if (!metrics) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <div className="w-12 h-12 border-4 border-white/20 border-t-emerald-500 rounded-full animate-spin" />
          <div className="text-white/60 text-sm">{t('serverDetail.connectingTo', { name: config.name })}</div>
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
      <div className={`server-detail min-h-screen p-4 md:p-6 lg:p-10 max-w-6xl mx-auto animate-fadeIn`}>
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
    );
  }

  return (
    <div className={`server-detail min-h-screen p-4 md:p-6 lg:p-10 max-w-6xl mx-auto ${showContent ? 'animate-slideUp' : 'opacity-0'}`}>
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
  );
}

