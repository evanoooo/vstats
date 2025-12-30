import { useTranslation } from 'react-i18next';
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
import { CustomTooltip } from './CustomTooltip';
import type { TimeGridPoint, ChartTheme } from '../../types';
import { formatBytesLocal, formatFullTime } from '../../utils';

interface NetworkChartProps {
  data: TimeGridPoint[];
  chartTheme: ChartTheme;
  xAxisInterval: number;
  isLight: boolean;
}

export function NetworkChart({
  data,
  chartTheme,
  xAxisInterval,
  isLight,
}: NetworkChartProps) {
  const { t } = useTranslation();

  return (
    <div className="space-y-6">
      <div className="h-56 w-full">
        <ResponsiveContainer width="100%" height="100%" minWidth={0} minHeight={0}>
          <LineChart data={data} margin={{ top: 5, right: 5, left: -5, bottom: 5 }}>
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
            {formatBytesLocal(data.filter(d => d.hasData && d.net_tx !== null).reduce((a, b) => a + (b.net_tx || 0), 0))}
          </div>
        </div>
        <div className="p-3 rounded-lg bg-cyan-500/10 border border-cyan-500/20">
          <div className="text-[10px] text-gray-500 uppercase">{t('serverDetail.history.totalDownload')}</div>
          <div className="text-lg font-mono text-cyan-500">
            {formatBytesLocal(data.filter(d => d.hasData && d.net_rx !== null).reduce((a, b) => a + (b.net_rx || 0), 0))}
          </div>
        </div>
      </div>
    </div>
  );
}

