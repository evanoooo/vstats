import type { ChartColorSet } from '../../types';

// Color palette for charts
export const chartColors: Record<string, ChartColorSet> = {
  blue: { stroke: '#3b82f6', fill: '#3b82f6', gradient: ['#3b82f6', '#06b6d4'] },
  purple: { stroke: '#a855f7', fill: '#a855f7', gradient: ['#a855f7', '#ec4899'] },
  amber: { stroke: '#f59e0b', fill: '#f59e0b', gradient: ['#f59e0b', '#ef4444'] },
  cyan: { stroke: '#06b6d4', fill: '#06b6d4', gradient: ['#06b6d4', '#3b82f6'] },
  rose: { stroke: '#f43f5e', fill: '#f43f5e', gradient: ['#f43f5e', '#a855f7'] },
  emerald: { stroke: '#10b981', fill: '#10b981', gradient: ['#10b981', '#06b6d4'] },
};

// Color keys for ping targets
export const pingColorKeys = ['rose', 'cyan', 'amber', 'purple', 'emerald', 'blue'] as const;

// Time range options
export const timeRanges = [
  { value: '1h' as const, label: '1H' },
  { value: '24h' as const, label: '24H' },
  { value: '7d' as const, label: '7D' },
  { value: '30d' as const, label: '30D' },
  { value: '1y' as const, label: '1Y' },
];

