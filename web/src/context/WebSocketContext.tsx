import { createContext, useContext, useEffect, useRef, useState, useCallback, type ReactNode } from 'react';
import type { SystemMetrics, SiteSettings, ServerGroup, GroupDimension, GeoIPData } from '../types';
import { sanitizeSiteSettings } from '../utils/security';

// Types
interface NetworkSpeed {
  rx_sec: number;
  tx_sec: number;
}

// Server label with color
export interface ServerLabel {
  name: string;
  color: string;
}

export interface ServerConfig {
  id: string;
  name: string;
  type: 'real' | 'local';
  location?: string;
  provider?: string;
  tag?: string;
  group_id?: string;
  group_values?: Record<string, string>;
  version?: string;
  price?: {
    amount: string;
    period: 'month' | 'quarter' | 'year';
    currency?: string;
  };
  purchase_date?: string;
  expiry_date?: string;
  auto_renew?: boolean;
  remaining_value?: string;
  tip_badge?: string;
  notes?: string;
  labels?: ServerLabel[];
  geoip?: GeoIPData;
  sale_status?: '' | 'rent' | 'sell';
  sale_contact_url?: string;
}

export interface ServerState {
  config: ServerConfig;
  metrics: SystemMetrics | null;
  speed: NetworkSpeed;
  isConnected: boolean;
  error: string | null;
}

export type LoadingState = 'idle' | 'loading' | 'ready' | 'error';

interface DashboardMessage {
  type: string;
  servers: ServerMetricsUpdate[];
  groups?: ServerGroup[];
  group_dimensions?: GroupDimension[];
  site_settings?: SiteSettings;
}

// Streaming messages for faster initial load
interface StreamInitMessage {
  type: 'stream_init';
  total_servers: number;
  groups?: ServerGroup[];
  group_dimensions?: GroupDimension[];
  site_settings?: SiteSettings;
}

interface StreamServerMessage {
  type: 'stream_server';
  index: number;
  total: number;
  server: ServerMetricsUpdate;
}

// StreamEndMessage is handled by type check only (no additional fields needed)
// type: 'stream_end'

interface DeltaMessage {
  type: 'delta';
  ts: number;
  d: CompactServerUpdate[];
}

interface CompactServerUpdate {
  id: string;
  on?: boolean;
  m?: CompactMetrics;
}

interface CompactMetrics {
  c?: number;
  m?: number;
  d?: number;
  rx?: number;
  tx?: number;
  up?: number;
}

interface ServerMetricsUpdate {
  server_id: string;
  server_name: string;
  location: string;
  provider: string;
  tag?: string;
  group_id?: string;
  group_values?: Record<string, string>;
  version?: string;
  online: boolean;
  metrics: SystemMetrics | null;
  price_amount?: string;
  price_period?: string;
  price_currency?: string;
  purchase_date?: string;
  expiry_date?: string;
  auto_renew?: boolean;
  tip_badge?: string;
  notes?: string;
  labels?: ServerLabel[];
  geoip?: GeoIPData;
  sale_status?: string;
  sale_contact_url?: string;
}

// Context interface
interface WebSocketContextValue {
  servers: ServerState[];
  groups: ServerGroup[];
  groupDimensions: GroupDimension[];
  siteSettings: SiteSettings;
  loadingState: LoadingState;
  isInitialLoad: boolean;
  getServerById: (id: string) => ServerState | undefined;
  isConnected: boolean;
}

const defaultSiteSettings: SiteSettings = {
  site_name: 'vStats Dashboard',
  site_description: 'Real-time Server Monitoring',
  social_links: []
};

const WebSocketContext = createContext<WebSocketContextValue | null>(null);

// LocalStorage key for caching server metrics
const METRICS_CACHE_KEY = 'vstats-metrics-cache';

const loadCachedMetrics = (): Map<string, SystemMetrics> => {
  try {
    const cached = localStorage.getItem(METRICS_CACHE_KEY);
    if (cached) {
      const parsed = JSON.parse(cached);
      return new Map(Object.entries(parsed));
    }
  } catch (e) {
    console.warn('Failed to load cached metrics', e);
  }
  return new Map();
};

const saveCachedMetrics = (metricsMap: Map<string, SystemMetrics>) => {
  try {
    const obj: Record<string, SystemMetrics> = {};
    metricsMap.forEach((value, key) => {
      obj[key] = value;
    });
    const data = JSON.stringify(obj);
    // Only cache if data is reasonably sized (< 1MB)
    if (data.length < 1024 * 1024) {
      localStorage.setItem(METRICS_CACHE_KEY, data);
    }
  } catch (e) {
    // If quota exceeded, clear the cache and don't retry
    if (e instanceof DOMException && e.name === 'QuotaExceededError') {
      try {
        localStorage.removeItem(METRICS_CACHE_KEY);
      } catch {
        // Ignore removal errors
      }
    }
    // Don't log - this is a non-critical optimization
  }
};

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const [servers, setServers] = useState<ServerState[]>([]);
  const [groups, setGroups] = useState<ServerGroup[]>([]);
  const [groupDimensions, setGroupDimensions] = useState<GroupDimension[]>([]);
  const [siteSettings, setSiteSettings] = useState<SiteSettings>(defaultSiteSettings);
  const [loadingState, setLoadingState] = useState<LoadingState>('loading');
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  const [isConnected, setIsConnected] = useState(false);
  
  const lastMetricsMap = useRef<Map<string, { metrics: SystemMetrics, time: number }>>(new Map());
  const serversCache = useRef<Map<string, ServerState>>(new Map());
  const cachedMetricsRef = useRef<Map<string, SystemMetrics>>(loadCachedMetrics());
  const wsRef = useRef<WebSocket | null>(null);
  const initialDataReceived = useRef(false);
  const reconnectTimeoutRef = useRef<number | null>(null);
  
  // Streaming state
  const streamingServersRef = useRef<ServerState[]>([]);
  const streamingTotalRef = useRef(0);

  const applyDelta = useCallback((delta: CompactServerUpdate) => {
    const cached = serversCache.current.get(delta.id);
    if (!cached) return null;
    
    const updated = { ...cached };
    
    if (delta.on !== undefined) {
      updated.isConnected = delta.on;
    }
    
    if (delta.m && updated.metrics) {
      const m = delta.m;
      updated.metrics = { ...updated.metrics };
      
      if (m.c !== undefined) {
        updated.metrics.cpu = { ...updated.metrics.cpu, usage: m.c };
      }
      if (m.m !== undefined) {
        updated.metrics.memory = { ...updated.metrics.memory, usage_percent: m.m };
      }
      if (m.d !== undefined && updated.metrics.disks?.[0]) {
        updated.metrics.disks = [{ ...updated.metrics.disks[0], usage_percent: m.d }];
      }
      if (m.rx !== undefined || m.tx !== undefined) {
        updated.metrics.network = { 
          ...updated.metrics.network,
          rx_speed: m.rx ?? updated.metrics.network.rx_speed,
          tx_speed: m.tx ?? updated.metrics.network.tx_speed,
        };
        updated.speed = {
          rx_sec: m.rx ?? updated.speed.rx_sec,
          tx_sec: m.tx ?? updated.speed.tx_sec,
        };
      }
      if (m.up !== undefined) {
        updated.metrics.uptime = m.up;
      }
    }
    
    return updated;
  }, []);

  // Global WebSocket connection - persists across page navigations
  useEffect(() => {
    const connect = () => {
      try {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;

        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onopen = () => {
          console.log('[WebSocket] Connected (global)');
          setIsConnected(true);
        };

        ws.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data);
            
            if (data.type === 'metrics' && data.servers) {
              const fullData = data as DashboardMessage;
              
              if (fullData.site_settings) {
                const sanitized = sanitizeSiteSettings(fullData.site_settings);
                setSiteSettings(sanitized);
                // Dispatch for ThemeContext
                window.dispatchEvent(new CustomEvent('vstats-site-settings', { detail: sanitized }));
              }
              
              if (fullData.groups) {
                setGroups(fullData.groups);
              }
              
              if (fullData.group_dimensions) {
                setGroupDimensions(fullData.group_dimensions.sort((a, b) => a.sort_order - b.sort_order));
              }
              
              if (!initialDataReceived.current) {
                initialDataReceived.current = true;
                setLoadingState('ready');
                setIsInitialLoad(false);
              }
              
              const now = Date.now();
              
              const allServers: ServerState[] = fullData.servers.map(serverUpdate => {
                const lastData = lastMetricsMap.current.get(serverUpdate.server_id);
                
                let newSpeed = { rx_sec: 0, tx_sec: 0 };

                if (serverUpdate.metrics?.network.rx_speed !== undefined && 
                    serverUpdate.metrics?.network.tx_speed !== undefined) {
                  newSpeed = {
                    rx_sec: serverUpdate.metrics.network.rx_speed,
                    tx_sec: serverUpdate.metrics.network.tx_speed
                  };
                } else if (lastData && serverUpdate.metrics) {
                  const timeDiff = (now - lastData.time) / 1000;
                  if (timeDiff > 0) {
                    const rxDiff = serverUpdate.metrics.network.total_rx - lastData.metrics.network.total_rx;
                    const txDiff = serverUpdate.metrics.network.total_tx - lastData.metrics.network.total_tx;
                    newSpeed = {
                      rx_sec: Math.max(0, rxDiff / timeDiff),
                      tx_sec: Math.max(0, txDiff / timeDiff)
                    };
                  }
                }

                let metricsToUse = serverUpdate.metrics;
                if (serverUpdate.metrics) {
                  lastMetricsMap.current.set(serverUpdate.server_id, { 
                    metrics: serverUpdate.metrics, 
                    time: now 
                  });
                  cachedMetricsRef.current.set(serverUpdate.server_id, serverUpdate.metrics);
                } else if (!serverUpdate.online) {
                  const cachedMetrics = cachedMetricsRef.current.get(serverUpdate.server_id);
                  if (cachedMetrics) {
                    metricsToUse = cachedMetrics;
                    newSpeed = { rx_sec: 0, tx_sec: 0 };
                  }
                }

                const isLocal = serverUpdate.server_id === 'local';

                const serverState: ServerState = {
                  config: {
                    id: serverUpdate.server_id,
                    name: serverUpdate.server_name,
                    type: isLocal ? 'local' as const : 'real' as const,
                    location: serverUpdate.location,
                    provider: serverUpdate.provider,
                    tag: serverUpdate.tag,
                    group_id: serverUpdate.group_id,
                    group_values: serverUpdate.group_values,
                    version: serverUpdate.version || serverUpdate.metrics?.version,
                    price: serverUpdate.price_amount ? {
                      amount: serverUpdate.price_amount,
                      period: (serverUpdate.price_period as 'month' | 'year') || 'month',
                      currency: serverUpdate.price_currency,
                    } : undefined,
                    purchase_date: serverUpdate.purchase_date,
                    expiry_date: serverUpdate.expiry_date,
                    auto_renew: serverUpdate.auto_renew,
                    tip_badge: serverUpdate.tip_badge,
                    notes: serverUpdate.notes,
                    labels: serverUpdate.labels,
                    geoip: serverUpdate.geoip,
                    sale_status: (serverUpdate.sale_status as '' | 'rent' | 'sell') || '',
                    sale_contact_url: serverUpdate.sale_contact_url,
                  },
                  metrics: metricsToUse,
                  speed: newSpeed,
                  isConnected: serverUpdate.online,
                  error: null
                };
                
                serversCache.current.set(serverUpdate.server_id, serverState);
                
                return serverState;
              });

              saveCachedMetrics(cachedMetricsRef.current);
              setServers(allServers);
            }
            else if (data.type === 'site_settings' && data.site_settings) {
              const sanitized = sanitizeSiteSettings(data.site_settings);
              setSiteSettings(sanitized);
              window.dispatchEvent(new CustomEvent('vstats-site-settings', { detail: sanitized }));
            }
            else if (data.type === 'delta') {
              const deltaData = data as DeltaMessage;
              
              if (deltaData.d && deltaData.d.length > 0) {
                setServers(prev => {
                  let hasChanges = false;
                  const updated = prev.map(server => {
                    const delta = deltaData.d.find(d => d.id === server.config.id);
                    if (delta) {
                      const newState = applyDelta(delta);
                      if (newState) {
                        hasChanges = true;
                        serversCache.current.set(server.config.id, newState);
                        if (newState.metrics && newState.isConnected) {
                          cachedMetricsRef.current.set(server.config.id, newState.metrics);
                        }
                        return newState;
                      }
                    }
                    return server;
                  });
                  if (hasChanges) {
                    saveCachedMetrics(cachedMetricsRef.current);
                  }
                  return hasChanges ? updated : prev;
                });
              }
            }
            // Handle streaming init message (metadata first)
            else if (data.type === 'stream_init') {
              const initData = data as StreamInitMessage;
              streamingServersRef.current = [];
              streamingTotalRef.current = initData.total_servers;
              
              // Apply metadata immediately
              if (initData.site_settings) {
                const sanitized = sanitizeSiteSettings(initData.site_settings);
                setSiteSettings(sanitized);
                window.dispatchEvent(new CustomEvent('vstats-site-settings', { detail: sanitized }));
              }
              if (initData.groups) {
                setGroups(initData.groups);
              }
              if (initData.group_dimensions) {
                setGroupDimensions(initData.group_dimensions.sort((a, b) => a.sort_order - b.sort_order));
              }
              
              // Mark as ready immediately (UI can start rendering)
              if (!initialDataReceived.current) {
                initialDataReceived.current = true;
                setLoadingState('ready');
                setIsInitialLoad(false);
              }
            }
            // Handle individual server in stream
            else if (data.type === 'stream_server') {
              const serverMsg = data as StreamServerMessage;
              const serverUpdate = serverMsg.server;
              const now = Date.now();
              
              const lastData = lastMetricsMap.current.get(serverUpdate.server_id);
              let newSpeed = { rx_sec: 0, tx_sec: 0 };

              if (serverUpdate.metrics?.network.rx_speed !== undefined && 
                  serverUpdate.metrics?.network.tx_speed !== undefined) {
                newSpeed = {
                  rx_sec: serverUpdate.metrics.network.rx_speed,
                  tx_sec: serverUpdate.metrics.network.tx_speed
                };
              } else if (lastData && serverUpdate.metrics) {
                const timeDiff = (now - lastData.time) / 1000;
                if (timeDiff > 0) {
                  const rxDiff = serverUpdate.metrics.network.total_rx - lastData.metrics.network.total_rx;
                  const txDiff = serverUpdate.metrics.network.total_tx - lastData.metrics.network.total_tx;
                  newSpeed = {
                    rx_sec: Math.max(0, rxDiff / timeDiff),
                    tx_sec: Math.max(0, txDiff / timeDiff)
                  };
                }
              }

              let metricsToUse = serverUpdate.metrics;
              if (serverUpdate.metrics) {
                lastMetricsMap.current.set(serverUpdate.server_id, { 
                  metrics: serverUpdate.metrics, 
                  time: now 
                });
                cachedMetricsRef.current.set(serverUpdate.server_id, serverUpdate.metrics);
              } else if (!serverUpdate.online) {
                const cachedMetrics = cachedMetricsRef.current.get(serverUpdate.server_id);
                if (cachedMetrics) {
                  metricsToUse = cachedMetrics;
                  newSpeed = { rx_sec: 0, tx_sec: 0 };
                }
              }

              const isLocal = serverUpdate.server_id === 'local';
              const serverState: ServerState = {
                config: {
                  id: serverUpdate.server_id,
                  name: serverUpdate.server_name,
                  type: isLocal ? 'local' as const : 'real' as const,
                  location: serverUpdate.location,
                  provider: serverUpdate.provider,
                  tag: serverUpdate.tag,
                  group_id: serverUpdate.group_id,
                  group_values: serverUpdate.group_values,
                  version: serverUpdate.version || serverUpdate.metrics?.version,
                  price: serverUpdate.price_amount ? {
                    amount: serverUpdate.price_amount,
                    period: (serverUpdate.price_period as 'month' | 'year') || 'month',
                    currency: serverUpdate.price_currency,
                  } : undefined,
                  purchase_date: serverUpdate.purchase_date,
                  expiry_date: serverUpdate.expiry_date,
                  auto_renew: serverUpdate.auto_renew,
                  tip_badge: serverUpdate.tip_badge,
                  notes: serverUpdate.notes,
                  labels: serverUpdate.labels,
                  geoip: serverUpdate.geoip,
                  sale_status: (serverUpdate.sale_status as '' | 'rent' | 'sell') || '',
                  sale_contact_url: serverUpdate.sale_contact_url,
                },
                metrics: metricsToUse,
                speed: newSpeed,
                isConnected: serverUpdate.online,
                error: null
              };
              
              serversCache.current.set(serverUpdate.server_id, serverState);
              streamingServersRef.current.push(serverState);
              
              // Update servers incrementally for real-time display
              setServers([...streamingServersRef.current]);
            }
            // Handle stream end
            else if (data.type === 'stream_end') {
              saveCachedMetrics(cachedMetricsRef.current);
              // Ensure final state is set
              if (streamingServersRef.current.length > 0) {
                setServers([...streamingServersRef.current]);
              }
            }
          } catch (e) {
            console.error('[WebSocket] Parse error', e);
          }
        };

        ws.onclose = () => {
          console.log('[WebSocket] Disconnected, reconnecting...');
          wsRef.current = null;
          setIsConnected(false);
          if (!initialDataReceived.current) {
            setLoadingState('loading');
          }
          // Fast reconnect - use shorter delay for better UX
          reconnectTimeoutRef.current = window.setTimeout(connect, 1000);
        };

        ws.onerror = (err) => {
          console.error('[WebSocket] Error', err);
        };
      } catch (e) {
        console.error('[WebSocket] Connection error', e);
        reconnectTimeoutRef.current = window.setTimeout(connect, 1000);
      }
    };

    // Connect immediately
    connect();

    // Cleanup only when the entire app unmounts (not on page navigation)
    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [applyDelta]);

  const getServerById = useCallback((id: string): ServerState | undefined => {
    return servers.find(s => s.config.id === id);
  }, [servers]);

  return (
    <WebSocketContext.Provider value={{
      servers,
      groups,
      groupDimensions,
      siteSettings,
      loadingState,
      isInitialLoad,
      getServerById,
      isConnected,
    }}>
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
}

// Compatibility export - same interface as old useServerManager
export function useServerManager() {
  return useWebSocket();
}

