import type { TimeRange, GridConfig, ChartTheme } from './types';

/**
 * Convert ISO 3166-1 alpha-2 country code to flag emoji
 * Each letter becomes a regional indicator symbol (A=🇦, B=🇧, etc.)
 */
export const getFlag = (code: string): string => {
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

/**
 * Format bytes to human readable string
 */
export const formatBytesLocal = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
};

/**
 * Format time for chart X-axis based on time range
 */
export const formatTimeForChart = (timestamp: string, range: TimeRange): string => {
  const date = new Date(timestamp);
  switch (range) {
    case '1h':
      // Per minute: "14:35"
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    case '24h':
      // Per hour: "14:00"
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    case '7d':
      // Per day+time: "12/10 14:00"
      return date.toLocaleDateString([], { month: 'numeric', day: 'numeric' }) + ' ' + 
             date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    case '30d':
      // Per day: "12/10" or "Dec 10"
      return date.toLocaleDateString([], { month: 'numeric', day: 'numeric' });
    case '1y':
      // Per month+day: "Dec 10"
      return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
    default:
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
};

/**
 * Format full time for tooltip display
 */
export const formatFullTime = (timestamp: string): string => {
  const date = new Date(timestamp);
  return date.toLocaleString([], { 
    month: 'short', 
    day: 'numeric',
    hour: '2-digit', 
    minute: '2-digit'
  });
};

/**
 * Get grid configuration for different time ranges
 * Matching backend bucket sizes
 */
export const getGridConfigForRange = (range: TimeRange): GridConfig => {
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
};

/**
 * Get chart theme colors based on light/dark mode
 */
export const getChartTheme = (isLight: boolean): ChartTheme => ({
  gridColor: isLight ? 'rgba(0,0,0,0.06)' : 'rgba(255,255,255,0.05)',
  tickColor: isLight ? '#374151' : '#6b7280',
  legendColor: isLight ? '#4b5563' : '#9ca3af',
});

/**
 * Calculate X-axis tick interval for chart
 * Show about 10-12 labels for better readability
 */
export const getXAxisInterval = (points: number): number => {
  if (points <= 12) return 0; // Show all labels if few points
  // For 720 points, show ~12 labels (interval = 60)
  return Math.floor(points / 12) - 1;
};

