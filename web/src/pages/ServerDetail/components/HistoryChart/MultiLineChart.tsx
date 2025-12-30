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
import { chartColors } from './chartConfig';
import { CustomTooltip } from './CustomTooltip';
import type { TimeGridPoint, ChartTheme } from '../../types';
import { formatFullTime } from '../../utils';

interface MultiLineChartProps {
  data: TimeGridPoint[];
  lines: Array<{ dataKey: string; color: keyof typeof chartColors; label: string }>;
  formatValue: (v: number) => string;
  maxValue?: number;
  chartTheme: ChartTheme;
  xAxisInterval: number;
  isLight: boolean;
}

export function MultiLineChart({ 
  data,
  lines,
  formatValue,
  maxValue,
  chartTheme,
  xAxisInterval,
  isLight,
}: MultiLineChartProps) {
  return (
    <div className="h-72 w-full">
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
            width={50}
          />
          <Tooltip
            content={
              <CustomTooltip 
                formatValue={formatValue}
                isLight={isLight}
                labelFormatter={(label) => {
                  const point = data.find(d => d.formattedTime === label);
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
}

