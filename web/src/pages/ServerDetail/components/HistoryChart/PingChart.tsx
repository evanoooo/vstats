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
import { chartColors, pingColorKeys } from './chartConfig';
import { CustomTooltip } from './CustomTooltip';
import type { TimeGridPoint, PingTimeGridPoint, ChartTheme, GridConfig } from '../../types';
import { formatTimeForChart, formatFullTime } from '../../utils';
import type { TimeRange } from '../../types';
import type { PingHistoryTarget } from '../../../../types';

interface PingChartProps {
  data: TimeGridPoint[];
  pingTargets: PingHistoryTarget[];
  chartTheme: ChartTheme;
  xAxisInterval: number;
  isLight: boolean;
  range: TimeRange;
  gridConfig: GridConfig;
}

export function PingChart({
  data,
  pingTargets,
  chartTheme,
  xAxisInterval,
  isLight,
  range,
  gridConfig,
}: PingChartProps) {
  const now = Date.now();
  const { points: pingPoints, interval: pingInterval, startTime: pingStartTime } = gridConfig;
  
  // Check if we have detailed ping target data
  if (pingTargets.length > 0) {
    // Create fixed time grid for ping
    const pingTimeGrid: PingTimeGridPoint[] = [];
    
    for (let i = 0; i < pingPoints; i++) {
      const gridTime = pingStartTime + i * pingInterval;
      const timestamp = new Date(gridTime).toISOString();
      const point: PingTimeGridPoint = {
        index: i,
        gridTime,
        timestamp,
        formattedTime: formatTimeForChart(timestamp, range),
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
      const gridTime = gridPoint.gridTime as number;
      
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
      <div>
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
  const pingData = data.filter(d => d.hasData && d.ping_ms !== null && d.ping_ms > 0);
  
  // If no valid ping data, still show the chart with the time grid (all zeros or nulls)
  const hasAnyPingData = pingData.length > 0;
  
  const pingValues = hasAnyPingData ? pingData.map(d => d.ping_ms!) : [0];
  const avgPing = hasAnyPingData ? pingValues.reduce((a, b) => a + b, 0) / pingValues.length : 0;
  const minPing = hasAnyPingData ? Math.min(...pingValues) : 0;
  const maxPing = hasAnyPingData ? Math.max(...pingValues) : 0;
  
  // Use sampledData which already has the fixed time grid
  const pingChartData = data.map((d, i) => ({
    ...d,
    index: i,
    ping: d.hasData ? (d.ping_ms ?? 0) : null,
  }));
  
  return (
    <div>
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

