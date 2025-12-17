import { useState, useEffect, useCallback, useRef } from 'react';
import * as THREE from 'three';
import type { ServerState } from '../../hooks/useMetrics';
import { latLngToVector3, getCountryCoordinates, EARTH_RADIUS } from './geoUtils';

export interface PingAnimation {
  id: string;
  source: ServerState;
  target: ServerState;
  startPosition: THREE.Vector3;
  endPosition: THREE.Vector3;
  latencyMs: number | null;
  startTime: number;
}

interface UseGlobeAnimationOptions {
  servers: ServerState[];
  maxConcurrentPings?: number;
  pingInterval?: number;
  enabled?: boolean;
}

/**
 * Get the 3D position for a server
 */
function getServerPosition(server: ServerState): THREE.Vector3 {
  const geoip = server.config.geoip;
  if (geoip?.latitude !== undefined && geoip?.longitude !== undefined) {
    return latLngToVector3(geoip.latitude, geoip.longitude, EARTH_RADIUS * 1.02);
  }
  const coords = getCountryCoordinates(geoip?.country_code || server.config.location);
  return latLngToVector3(coords.lat, coords.lng, EARTH_RADIUS * 1.02);
}

/**
 * Get ping latency between two servers (from ping metrics if available)
 */
function getLatencyBetweenServers(source: ServerState, target: ServerState): number | null {
  // Check if source has ping data to target
  const pingTargets = source.metrics?.ping?.targets;
  if (pingTargets) {
    // Try to find a matching target by name or host
    const matchingTarget = pingTargets.find(
      (t) =>
        t.name.toLowerCase().includes(target.config.name.toLowerCase()) ||
        target.config.name.toLowerCase().includes(t.name.toLowerCase())
    );
    if (matchingTarget && matchingTarget.latency_ms !== null) {
      return matchingTarget.latency_ms;
    }
  }

  // Generate realistic fake latency based on distance
  const sourcePos = getServerPosition(source);
  const targetPos = getServerPosition(target);
  const distance = sourcePos.distanceTo(targetPos);
  
  // Approximate: 1 unit distance ≈ 50-150ms (with some randomness)
  const baseLatency = distance * 80;
  const variance = (Math.random() - 0.5) * 40;
  return Math.max(10, Math.round(baseLatency + variance));
}

/**
 * Hook to manage ping animations between servers
 */
export function useGlobeAnimation({
  servers,
  maxConcurrentPings = 3,
  pingInterval = 3000,
  enabled = true,
}: UseGlobeAnimationOptions) {
  const [activePings, setActivePings] = useState<PingAnimation[]>([]);
  const pingIdCounter = useRef(0);
  const lastPingTime = useRef(0);

  // Filter online servers with valid positions
  const onlineServers = servers.filter((s) => {
    if (!s.isConnected) return false;
    const geoip = s.config.geoip;
    // Must have either geoip coords or a location/country code
    return (
      (geoip?.latitude !== undefined && geoip?.longitude !== undefined) ||
      geoip?.country_code ||
      s.config.location
    );
  });

  /**
   * Create a new ping animation between two random servers
   */
  const createRandomPing = useCallback(() => {
    if (onlineServers.length < 2) return null;

    // Pick two random different servers
    const sourceIndex = Math.floor(Math.random() * onlineServers.length);
    let targetIndex = Math.floor(Math.random() * onlineServers.length);
    while (targetIndex === sourceIndex) {
      targetIndex = Math.floor(Math.random() * onlineServers.length);
    }

    const source = onlineServers[sourceIndex];
    const target = onlineServers[targetIndex];

    const startPosition = getServerPosition(source);
    const endPosition = getServerPosition(target);
    const latencyMs = getLatencyBetweenServers(source, target);

    pingIdCounter.current += 1;

    return {
      id: `ping-${pingIdCounter.current}`,
      source,
      target,
      startPosition,
      endPosition,
      latencyMs,
      startTime: Date.now(),
    };
  }, [onlineServers]);

  /**
   * Remove a completed ping animation
   */
  const removePing = useCallback((id: string) => {
    setActivePings((prev) => prev.filter((p) => p.id !== id));
  }, []);

  /**
   * Add a new ping animation
   */
  const addPing = useCallback(() => {
    if (!enabled) return;
    if (activePings.length >= maxConcurrentPings) return;

    const newPing = createRandomPing();
    if (newPing) {
      setActivePings((prev) => [...prev, newPing]);
    }
  }, [enabled, activePings.length, maxConcurrentPings, createRandomPing]);

  // Continuous ping generation - always try to maintain max concurrent pings
  useEffect(() => {
    if (!enabled || onlineServers.length < 2) return;

    // More aggressive interval for continuous pings
    const interval = setInterval(() => {
      // Always try to add pings up to max
      if (activePings.length < maxConcurrentPings) {
        addPing();
        lastPingTime.current = Date.now();
      }
    }, Math.max(200, pingInterval / 3)); // Check frequently

    // Initial pings - fill up to max immediately
    const initialPings = maxConcurrentPings - activePings.length;
    for (let i = 0; i < initialPings; i++) {
      setTimeout(() => addPing(), i * 100);
    }
    lastPingTime.current = Date.now();

    return () => clearInterval(interval);
  }, [enabled, onlineServers.length, pingInterval, addPing, activePings.length, maxConcurrentPings]);

  // Cleanup old pings (safety measure)
  useEffect(() => {
    const cleanup = setInterval(() => {
      const now = Date.now();
      setActivePings((prev) =>
        prev.filter((p) => now - p.startTime < 10000) // Remove pings older than 10s
      );
    }, 5000);

    return () => clearInterval(cleanup);
  }, []);

  return {
    activePings,
    removePing,
    addPing,
    onlineServersCount: onlineServers.length,
  };
}

export default useGlobeAnimation;
