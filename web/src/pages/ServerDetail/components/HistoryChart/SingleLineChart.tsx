import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { chartColors } from './chartConfig';
import { CustomTooltip } from './CustomTooltip';
import type { TimeGridPoint, ChartTheme } from '../../types';
import { formatFullTime } from '../../utils';

interface SingleLineChartProps {
  data: TimeGridPoint[];
  dataKey: string;
  color: keyof typeof chartColors;
  label: string;
  formatValue: (v: number) => string;
  maxValue?: number;
  chartTheme: ChartTheme;
  xAxisInterval: number;
  isLight: boolean;
}

export function SingleLineChart({ 
  data,
  dataKey, 
  color,
  label,
  formatValue,
  maxValue,
  chartTheme,
  xAxisInterval,
  isLight,
}: SingleLineChartProps) {
  const colorSet = chartColors[color];
  
  // Only include points with actual data (hasData=true and value is not null) for statistics
  const validValues = data
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
          <LineChart data={data} margin={{ top: 5, right: 5, left: -15, bottom: 5 }}>
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
                  labelFormatter={(labelStr) => {
                    const point = data.find(d => d.formattedTime === labelStr);
                    return point ? formatFullTime(point.timestamp) : labelStr;
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
}

