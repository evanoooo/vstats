import { useRef, useState, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import { Billboard, Text } from '@react-three/drei';
import * as THREE from 'three';
import { latLngToVector3, getCountryCoordinates, EARTH_RADIUS } from './geoUtils';
import type { ServerState } from '../../hooks/useMetrics';

interface ServerNodeProps {
  server: ServerState;
  onClick?: (server: ServerState) => void;
  onHover?: (server: ServerState | null) => void;
  isSelected?: boolean;
}

/**
 * Get color based on server status and resource usage
 * Uses digital/cyberpunk style colors matching the earth texture
 */
function getNodeColor(server: ServerState): THREE.Color {
  if (!server.isConnected) {
    return new THREE.Color(0xff3366); // Neon pink/red for offline
  }

  const cpuUsage = server.metrics?.cpu?.usage ?? 0;
  const memUsage = server.metrics?.memory?.usage_percent ?? 0;
  const maxUsage = Math.max(cpuUsage, memUsage);

  if (maxUsage > 80) {
    return new THREE.Color(0xff6b9d); // Pink for high load
  } else if (maxUsage > 50) {
    return new THREE.Color(0x00d4ff); // Light cyan for medium load
  }
  return new THREE.Color(0x00ffff); // Bright cyan for healthy
}

/**
 * Single server node on the globe
 */
export function ServerNode({ server, onClick, onHover, isSelected }: ServerNodeProps) {
  const groupRef = useRef<THREE.Group>(null);
  const glowRef = useRef<THREE.Mesh>(null);
  const [hovered, setHovered] = useState(false);
  
  // Calculate position from geoip data or country code
  const position = useMemo(() => {
    const geoip = server.config.geoip;
    if (geoip?.latitude !== undefined && geoip?.longitude !== undefined) {
      return latLngToVector3(geoip.latitude, geoip.longitude, EARTH_RADIUS * 1.02);
    }
    // Fallback to country code estimation
    const coords = getCountryCoordinates(geoip?.country_code || server.config.location);
    // Add slight random offset to prevent overlapping
    const offset = {
      lat: (Math.random() - 0.5) * 5,
      lng: (Math.random() - 0.5) * 5,
    };
    return latLngToVector3(
      coords.lat + offset.lat,
      coords.lng + offset.lng,
      EARTH_RADIUS * 1.02
    );
  }, [server.config.geoip, server.config.location]);

  const color = useMemo(() => getNodeColor(server), [server.isConnected, server.metrics]);

  // Pulse animation
  useFrame((state) => {
    if (glowRef.current) {
      const scale = 1 + Math.sin(state.clock.elapsedTime * 3 + position.x * 10) * 0.2;
      glowRef.current.scale.setScalar(hovered || isSelected ? scale * 1.5 : scale);
    }
  });

  const handlePointerOver = () => {
    setHovered(true);
    document.body.style.cursor = 'pointer';
    onHover?.(server);
  };

  const handlePointerOut = () => {
    setHovered(false);
    document.body.style.cursor = 'auto';
    onHover?.(null);
  };

  const handleClick = (e: { stopPropagation?: () => void }) => {
    e.stopPropagation?.();
    onClick?.(server);
  };

  // Enhanced glow material for digital/cyberpunk style
  const glowMaterial = useMemo(() => {
    return new THREE.ShaderMaterial({
      uniforms: {
        color: { value: color },
        intensity: { value: hovered || isSelected ? 3.0 : 1.5 },
      },
      vertexShader: `
        varying vec3 vNormal;
        varying vec3 vPosition;
        void main() {
          vNormal = normalize(normalMatrix * normal);
          vPosition = position;
          gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
        }
      `,
      fragmentShader: `
        uniform vec3 color;
        uniform float intensity;
        varying vec3 vNormal;
        varying vec3 vPosition;
        
        void main() {
          float dist = length(vPosition);
          float glow = 1.0 - smoothstep(0.0, 1.0, dist * 15.0);
          glow = pow(glow, 1.5);
          vec3 glowColor = color * intensity * 1.5;
          gl_FragColor = vec4(glowColor, glow * 0.6);
        }
      `,
      transparent: true,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
    });
  }, [color, hovered, isSelected]);

  // Update uniform when color changes
  useMemo(() => {
    if (glowMaterial.uniforms) {
      glowMaterial.uniforms.color.value = color;
      glowMaterial.uniforms.intensity.value = hovered || isSelected ? 2.0 : 1.0;
    }
  }, [color, hovered, isSelected, glowMaterial]);

  const nodeSize = hovered || isSelected ? 0.025 : 0.018;

  // Core material with emissive glow for digital look
  const coreMaterial = useMemo(() => {
    return new THREE.MeshBasicMaterial({
      color: color,
      transparent: true,
      opacity: 1.0,
    });
  }, [color]);

  return (
    <group ref={groupRef} position={position}>
      {/* Core point - cube style for digital look */}
      <mesh
        onPointerOver={handlePointerOver}
        onPointerOut={handlePointerOut}
        onClick={handleClick}
        material={coreMaterial}
      >
        <boxGeometry args={[nodeSize, nodeSize, nodeSize]} />
      </mesh>

      {/* Inner glow - slightly larger cube */}
      <mesh>
        <boxGeometry args={[nodeSize * 1.5, nodeSize * 1.5, nodeSize * 1.5]} />
        <meshBasicMaterial 
          color={color} 
          transparent 
          opacity={0.4}
        />
      </mesh>

      {/* Outer glow effect */}
      <mesh ref={glowRef} material={glowMaterial}>
        <sphereGeometry args={[nodeSize * 3, 16, 16]} />
      </mesh>

      {/* Digital pulse rings - square style */}
      {server.isConnected && (
        <>
          <mesh rotation={[Math.PI / 2, 0, 0]}>
            <ringGeometry args={[nodeSize * 2.5, nodeSize * 3, 4]} />
            <meshBasicMaterial
              color={color}
              transparent
              opacity={0.4}
              side={THREE.DoubleSide}
            />
          </mesh>
          <mesh rotation={[Math.PI / 2, 0, Math.PI / 4]}>
            <ringGeometry args={[nodeSize * 3.5, nodeSize * 4, 4]} />
            <meshBasicMaterial
              color={color}
              transparent
              opacity={0.2}
              side={THREE.DoubleSide}
            />
          </mesh>
        </>
      )}

      {/* Label on hover - digital style */}
      {(hovered || isSelected) && (
        <Billboard follow={true} lockX={false} lockY={false} lockZ={false}>
          <Text
            position={[0, 0.07, 0]}
            fontSize={0.022}
            color="#00ffff"
            anchorX="center"
            anchorY="bottom"
            outlineWidth={0.002}
            outlineColor="#001a1a"
          >
            {server.config.name}
          </Text>
          <Text
            position={[0, 0.045, 0]}
            fontSize={0.014}
            color={server.isConnected ? '#00ffff' : '#ff3366'}
            anchorX="center"
            anchorY="bottom"
          >
            {server.isConnected ? '● ONLINE' : '○ OFFLINE'}
          </Text>
        </Billboard>
      )}
    </group>
  );
}

export default ServerNode;
