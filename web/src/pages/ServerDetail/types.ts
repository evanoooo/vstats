// Time range options for history charts
export type TimeRange = '1h' | '24h' | '7d' | '30d' | '1y';

// History chart tab options
export type HistoryTab = 'overview' | 'cpu' | 'memory' | 'disk' | 'network' | 'ping';

// Chart color configuration
export interface ChartColorSet {
  stroke: string;
  fill: string;
  gradient: [string, string];
}

// Time grid point for chart data
export interface TimeGridPoint {
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
}

// Ping time grid point
export interface PingTimeGridPoint {
  index: number;
  gridTime: number;
  timestamp: string;
  formattedTime: string;
  hasData: boolean;
  [key: string]: number | string | boolean | null;
}

// Grid configuration for different time ranges
export interface GridConfig {
  points: number;
  duration: number;
  interval: number;
  startTime: number;
}

// Chart theme colors
export interface ChartTheme {
  gridColor: string;
  tickColor: string;
  legendColor: string;
}

