import { useRef, useMemo, forwardRef, useState, useEffect } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';
import { EARTH_RADIUS } from './geoUtils';

interface EarthProps {
  autoRotate?: boolean;
  rotationSpeed?: number;
  children?: React.ReactNode;
}

/**
 * Earth globe component with real texture and atmosphere glow effect
 * Children (like ServerNodes) will be attached to the Earth and rotate with it
 */
export const Earth = forwardRef<THREE.Group, EarthProps>(
  ({ autoRotate = true, rotationSpeed = 0.001, children }, ref) => {
    const groupRef = useRef<THREE.Group>(null);
    const [texture, setTexture] = useState<THREE.Texture | null>(null);

    // Load texture manually with error handling
    useEffect(() => {
      let disposed = false;
      let loadedTex: THREE.Texture | null = null;
      
      const loader = new THREE.TextureLoader();
      // Use Vite's BASE_URL so this works when deployed under a sub-path (e.g. GitHub Pages).
      const baseUrl = import.meta.env.BASE_URL ?? '/';
      const basePath = baseUrl.endsWith('/') ? baseUrl : `${baseUrl}/`;
      
      // Try digital earth texture first, fallback to blue marble
      const textureUrl = `${basePath}textures/earth-digital.jpg`;
      const fallbackTextureUrl = `${basePath}textures/earth-blue-marble.jpg`;
      
      loader.load(
        textureUrl,
        (tex) => {
          if (disposed) {
            tex.dispose();
            return;
          }
          tex.colorSpace = THREE.SRGBColorSpace;
          tex.anisotropy = 16;
          loadedTex = tex;
          setTexture(tex);
        },
        undefined,
        (error) => {
          // Try fallback texture if digital texture fails
          console.warn('Failed to load digital Earth texture, trying fallback:', error);
          loader.load(
            fallbackTextureUrl,
            (fallbackTex) => {
              if (disposed) {
                fallbackTex.dispose();
                return;
              }
              fallbackTex.colorSpace = THREE.SRGBColorSpace;
              fallbackTex.anisotropy = 16;
              loadedTex = fallbackTex;
              setTexture(fallbackTex);
            },
            undefined,
            (fallbackError) => {
              console.warn('Failed to load fallback Earth texture:', fallbackError);
            }
          );
        }
      );

      return () => {
        disposed = true;
        if (loadedTex) {
          loadedTex.dispose();
        }
      };
    }, []);

    // Auto rotation
    useFrame((_, delta) => {
      if (autoRotate && groupRef.current) {
        groupRef.current.rotation.y += rotationSpeed * delta * 60;
      }
    });

    // Earth material - only create when texture is loaded
    const earthMaterial = useMemo(() => {
      if (!texture) return null;
      
      return new THREE.MeshStandardMaterial({
        map: texture,
        metalness: 0.1,
        roughness: 0.6,
        emissive: new THREE.Color(0xffffff),
        emissiveMap: texture,
        emissiveIntensity: 0.4,
      });
    }, [texture]);

    // Enhanced atmosphere glow material with stronger sci-fi effect
    const atmosphereMaterial = useMemo(() => {
      return new THREE.ShaderMaterial({
        uniforms: {
          glowColor: { value: new THREE.Color(0x00ffff) }, // Brighter cyan for more sci-fi feel
          coefficient: { value: 0.5 },
          power: { value: 2.5 },
        },
        vertexShader: `
          varying vec3 vNormal;
          varying vec3 vPositionNormal;
          
          void main() {
            vNormal = normalize(normalMatrix * normal);
            vPositionNormal = normalize((modelViewMatrix * vec4(position, 1.0)).xyz);
            gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
          }
        `,
        fragmentShader: `
          uniform vec3 glowColor;
          uniform float coefficient;
          uniform float power;
          
          varying vec3 vNormal;
          varying vec3 vPositionNormal;
          
          void main() {
            float intensity = pow(coefficient - dot(vNormal, vPositionNormal), power);
            // Enhanced glow with pulsing effect approximation
            float pulse = sin(dot(vNormal, vec3(1.0)) * 3.14159) * 0.1 + 0.9;
            gl_FragColor = vec4(glowColor, intensity * 0.7 * pulse);
          }
        `,
        side: THREE.BackSide,
        blending: THREE.AdditiveBlending,
        transparent: true,
      });
    }, []);

    // Don't render until texture is loaded
    if (!earthMaterial) {
      return null;
    }

    return (
      <group ref={(node) => {
        groupRef.current = node;
        if (typeof ref === 'function') {
          ref(node);
        } else if (ref) {
          ref.current = node;
        }
      }}>
        {/* Main Earth sphere */}
        <mesh material={earthMaterial}>
          <sphereGeometry args={[EARTH_RADIUS, 64, 64]} />
        </mesh>

        {/* Atmosphere glow */}
        <mesh material={atmosphereMaterial}>
          <sphereGeometry args={[EARTH_RADIUS * 1.12, 64, 64]} />
        </mesh>

        {/* Children (ServerNodes) - will rotate with Earth */}
        {children}
      </group>
    );
  }
);

Earth.displayName = 'Earth';

export default Earth;
