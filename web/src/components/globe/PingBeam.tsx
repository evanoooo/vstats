import { useRef, useMemo, useState, useEffect } from 'react';
import { useFrame } from '@react-three/fiber';
import { Billboard, Text } from '@react-three/drei';
import * as THREE from 'three';
import { createArcCurve, EARTH_RADIUS } from './geoUtils';

interface PingBeamProps {
  startPosition: THREE.Vector3;
  endPosition: THREE.Vector3;
  latencyMs: number | null;
  sourceName?: string;
  targetName?: string;
  onComplete?: () => void;
  duration?: number;
  color?: THREE.Color;
}

/**
 * Animated ping beam between two points on the globe
 */
export function PingBeam({
  startPosition,
  endPosition,
  latencyMs,
  sourceName: _sourceName,
  targetName: _targetName,
  onComplete,
  duration = 2000,
  color = new THREE.Color(0x00d4ff),
}: PingBeamProps) {
  // Unused but kept for potential future use
  void _sourceName;
  void _targetName;
  
  const lineRef = useRef<THREE.Line | null>(null);
  const particleRef = useRef<THREE.Mesh>(null);
  const progressRef = useRef(0);
  const [showLatency, setShowLatency] = useState(false);
  const [latencyPosition, setLatencyPosition] = useState(endPosition.clone());

  // Create the arc curve
  const curve = useMemo(
    () => createArcCurve(startPosition, endPosition, EARTH_RADIUS),
    [startPosition, endPosition]
  );

  // Create the trail geometry
  const trailGeometry = useMemo(() => {
    const points = curve.getPoints(100);
    return new THREE.BufferGeometry().setFromPoints(points);
  }, [curve]);

  // Trail material with gradient
  const trailMaterial = useMemo(() => {
    return new THREE.ShaderMaterial({
      uniforms: {
        color: { value: color },
        progress: { value: 0 },
        trailLength: { value: 0.3 },
      },
      vertexShader: `
        attribute float lineDistance;
        varying float vLineDistance;
        
        void main() {
          vLineDistance = lineDistance;
          gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
        }
      `,
      fragmentShader: `
        uniform vec3 color;
        uniform float progress;
        uniform float trailLength;
        varying float vLineDistance;
        
        void main() {
          float totalLength = 1.0;
          float normalizedDist = vLineDistance / totalLength;
          
          // Trail fades behind the particle
          float trailStart = max(0.0, progress - trailLength);
          float alpha = 0.0;
          
          if (normalizedDist >= trailStart && normalizedDist <= progress) {
            alpha = (normalizedDist - trailStart) / trailLength;
            alpha = pow(alpha, 0.5) * 0.8;
          }
          
          // Glow effect
          vec3 glowColor = color * 1.5;
          
          gl_FragColor = vec4(glowColor, alpha);
        }
      `,
      transparent: true,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
    });
  }, [color]);

  // Add line distances to geometry
  useEffect(() => {
    if (trailGeometry) {
      const positions = trailGeometry.attributes.position;
      const distances = new Float32Array(positions.count);
      let totalDistance = 0;

      for (let i = 0; i < positions.count; i++) {
        if (i > 0) {
          const prev = new THREE.Vector3(
            positions.getX(i - 1),
            positions.getY(i - 1),
            positions.getZ(i - 1)
          );
          const curr = new THREE.Vector3(
            positions.getX(i),
            positions.getY(i),
            positions.getZ(i)
          );
          totalDistance += prev.distanceTo(curr);
        }
        distances[i] = totalDistance;
      }

      // Normalize distances
      for (let i = 0; i < distances.length; i++) {
        distances[i] /= totalDistance || 1;
      }

      trailGeometry.setAttribute('lineDistance', new THREE.BufferAttribute(distances, 1));
    }
  }, [trailGeometry]);

  // Particle (ping ball) material
  const particleMaterial = useMemo(() => {
    return new THREE.ShaderMaterial({
      uniforms: {
        color: { value: color },
        time: { value: 0 },
      },
      vertexShader: `
        varying vec3 vNormal;
        void main() {
          vNormal = normalize(normalMatrix * normal);
          gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
        }
      `,
      fragmentShader: `
        uniform vec3 color;
        uniform float time;
        varying vec3 vNormal;
        
        void main() {
          float rim = 1.0 - max(dot(vNormal, vec3(0.0, 0.0, 1.0)), 0.0);
          float pulse = 0.8 + sin(time * 10.0) * 0.2;
          vec3 glowColor = color * 2.0 * pulse;
          float alpha = 0.6 + rim * 0.4;
          gl_FragColor = vec4(glowColor, alpha);
        }
      `,
      transparent: true,
      blending: THREE.AdditiveBlending,
    });
  }, [color]);

  // Animation
  useFrame((state, delta) => {
    const speed = 1 / (duration / 1000);
    progressRef.current += delta * speed;

    if (progressRef.current >= 1) {
      setShowLatency(true);
      setLatencyPosition(endPosition.clone());
      
      // Keep showing latency for a moment, then complete
      if (progressRef.current >= 1.5) {
        onComplete?.();
        return;
      }
    }

    // Update trail material progress
    if (trailMaterial.uniforms) {
      trailMaterial.uniforms.progress.value = Math.min(progressRef.current, 1);
    }

    // Update particle position along curve
    if (particleRef.current && progressRef.current < 1) {
      const point = curve.getPoint(progressRef.current);
      particleRef.current.position.copy(point);
      
      // Update particle material time
      if (particleMaterial.uniforms) {
        particleMaterial.uniforms.time.value = state.clock.elapsedTime;
      }
    }
  });

  return (
    <group>
      {/* Trail line */}
      <primitive object={new THREE.Line(trailGeometry, trailMaterial)} ref={lineRef} />

      {/* Moving particle - cube style for digital look */}
      {progressRef.current < 1 && (
        <mesh ref={particleRef} material={particleMaterial}>
          <boxGeometry args={[0.018, 0.018, 0.018]} />
        </mesh>
      )}

      {/* Latency display at end point - digital style */}
      {showLatency && latencyMs !== null && (
        <Billboard position={latencyPosition.clone().multiplyScalar(1.08)}>
          <Text
            fontSize={0.022}
            color="#00ffff"
            anchorX="center"
            anchorY="middle"
            outlineWidth={0.002}
            outlineColor="#001a1a"
          >
            {`${latencyMs.toFixed(1)}ms`}
          </Text>
        </Billboard>
      )}

      {/* Start point marker - cube style */}
      <mesh position={startPosition}>
        <boxGeometry args={[0.01, 0.01, 0.01]} />
        <meshBasicMaterial color={color} transparent opacity={0.6} />
      </mesh>

      {/* End point marker - cube style */}
      <mesh position={endPosition}>
        <boxGeometry args={[0.01, 0.01, 0.01]} />
        <meshBasicMaterial color={color} transparent opacity={0.6} />
      </mesh>
    </group>
  );
}

export default PingBeam;
