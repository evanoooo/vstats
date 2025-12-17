import { Suspense, useState, useCallback, useMemo } from 'react';
import { Canvas } from '@react-three/fiber';
import { OrbitControls, Stars, Preload } from '@react-three/drei';
import * as THREE from 'three';
import type { ServerState } from '../../hooks/useMetrics';
import { useTheme } from '../../context/ThemeContext';
import { Earth } from './Earth';
import { ServerNode } from './ServerNode';
import { PingBeam } from './PingBeam';
import { GlobeOverlay } from './GlobeOverlay';
import { useGlobeAnimation } from './useGlobeAnimation';

interface GlobeViewProps {
  servers: ServerState[];
  onServerClick?: (server: ServerState) => void;
  onExitFullscreen?: () => void;
}

/**
 * Generate mock servers for demo/preview when no real servers are available
 */
function generateMockServers(): ServerState[] {
  const mockLocations = [
    { name: 'Tokyo', lat: 35.6762, lng: 139.6503, country: 'JP' },
    { name: 'Singapore', lat: 1.3521, lng: 103.8198, country: 'SG' },
    { name: 'Frankfurt', lat: 50.1109, lng: 8.6821, country: 'DE' },
    { name: 'New York', lat: 40.7128, lng: -74.0060, country: 'US' },
    { name: 'San Francisco', lat: 37.7749, lng: -122.4194, country: 'US' },
    { name: 'Sydney', lat: -33.8688, lng: 151.2093, country: 'AU' },
    { name: 'London', lat: 51.5074, lng: -0.1278, country: 'GB' },
    { name: 'São Paulo', lat: -23.5505, lng: -46.6333, country: 'BR' },
  ];

  return mockLocations.map((loc, idx) => ({
    config: {
      id: `mock-${idx}`,
      name: `${loc.name} Server`,
      location: loc.country,
      geoip: {
        latitude: loc.lat,
        longitude: loc.lng,
        country_code: loc.country,
        city: loc.name,
      },
    },
    isConnected: true,
    lastUpdate: Date.now(),
    speed: { download: 0, upload: 0 },
    error: null,
    metrics: {
      cpu: { usage: Math.random() * 60 + 10 },
      memory: { usage_percent: Math.random() * 50 + 20 },
    },
  } as unknown as ServerState));
}

/**
 * Loading fallback for the 3D scene
 */
function LoadingFallback() {
  return (
    <mesh>
      <sphereGeometry args={[1, 32, 32]} />
      <meshBasicMaterial color="#1a1a2e" wireframe />
    </mesh>
  );
}

/**
 * Main 3D scene containing the globe and all elements
 */
function GlobeScene({
  servers,
  onServerClick,
  onServerHover,
  selectedServer,
}: {
  servers: ServerState[];
  onServerClick: (server: ServerState) => void;
  onServerHover: (server: ServerState | null) => void;
  selectedServer: ServerState | null;
}) {
  // Use mock servers if no real servers available (for demo/preview)
  const mockServers = useMemo(() => generateMockServers(), []);
  const effectiveServers = servers.length >= 2 ? servers : mockServers;
  const isMockMode = servers.length < 2;

  // Ping animations - continuous non-stop pings
  const { activePings, removePing } = useGlobeAnimation({
    servers: effectiveServers,
    maxConcurrentPings: isMockMode ? 10 : 6,
    pingInterval: isMockMode ? 500 : 1000,
    enabled: true,
  });

  // Handle background click to deselect
  const handleBackgroundClick = () => {
    if (selectedServer) {
      onServerClick(null as any);
    }
  };

  return (
    <>
      {/* Camera and controls */}
      <OrbitControls
        enablePan={false}
        enableZoom={true}
        minDistance={2.0}
        maxDistance={6}
        autoRotate={false}
        dampingFactor={0.05}
        rotateSpeed={0.5}
      />

      {/* Enhanced lighting for sci-fi atmosphere */}
      <ambientLight intensity={0.5} />
      <directionalLight position={[5, 3, 5]} intensity={1.2} />
      <pointLight position={[-10, -10, -10]} intensity={0.4} color="#00ffff" />
      <pointLight position={[10, 10, 10]} intensity={0.2} color="#00d4ff" />

      {/* Starfield background */}
      <Stars
        radius={100}
        depth={50}
        count={5000}
        factor={4}
        saturation={0}
        fade
        speed={0.5}
      />

      {/* Earth globe with server nodes and ping beams as children */}
      <group onClick={handleBackgroundClick}>
        <Earth autoRotate={!selectedServer} rotationSpeed={0.0005}>
          {/* Server nodes - children of Earth, will rotate with it */}
          {effectiveServers.map((server) => (
            <ServerNode
              key={server.config.id}
              server={server}
              onClick={isMockMode ? undefined : onServerClick}
              onHover={isMockMode ? undefined : onServerHover}
              isSelected={selectedServer?.config.id === server.config.id}
            />
          ))}
          
          {/* Ping beams - also children of Earth, rotate with it */}
          {activePings.map((ping) => (
            <PingBeam
              key={ping.id}
              startPosition={ping.startPosition}
              endPosition={ping.endPosition}
              latencyMs={ping.latencyMs}
              sourceName={ping.source.config.name}
              targetName={ping.target.config.name}
              onComplete={() => removePing(ping.id)}
              duration={2000}
              color={new THREE.Color(0x00d4ff)}
            />
          ))}
        </Earth>
        
      </group>

      {/* Preload assets */}
      <Preload all />
    </>
  );
}

/**
 * Main GlobeView component with Canvas and overlay
 */
export function GlobeView({ servers, onServerClick, onExitFullscreen }: GlobeViewProps) {
  const { isDark } = useTheme();
  const [selectedServer, setSelectedServer] = useState<ServerState | null>(null);
  const [hoveredServer, setHoveredServer] = useState<ServerState | null>(null);

  const handleServerClick = useCallback((server: ServerState) => {
    setSelectedServer((prev) => (prev?.config.id === server?.config.id ? null : server));
    onServerClick?.(server);
  }, [onServerClick]);

  const handleServerHover = useCallback((server: ServerState | null) => {
    setHoveredServer(server);
  }, []);

  const handleCloseModal = useCallback(() => {
    setSelectedServer(null);
  }, []);

  return (
    <div className="globe-container">
      {/* WebGL Canvas */}
      <Canvas
        camera={{
          position: [0, 0, 3.5],
          fov: 45,
          near: 0.1,
          far: 1000,
        }}
        gl={{
          antialias: true,
          alpha: true,
          powerPreference: 'high-performance',
        }}
        dpr={[1, 2]}
        style={{ background: 'transparent' }}
      >
        <Suspense fallback={<LoadingFallback />}>
          <GlobeScene
            servers={servers}
            onServerClick={handleServerClick}
            onServerHover={handleServerHover}
            selectedServer={selectedServer}
          />
        </Suspense>
      </Canvas>

      {/* UI Overlay */}
      <GlobeOverlay
        servers={servers}
        selectedServer={selectedServer}
        hoveredServer={hoveredServer}
        onClose={handleCloseModal}
        onExitFullscreen={onExitFullscreen}
        isDark={isDark}
      />
    </div>
  );
}

export default GlobeView;
