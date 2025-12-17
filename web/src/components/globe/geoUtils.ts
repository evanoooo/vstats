import * as THREE from 'three';

/**
 * Earth radius constant for the 3D globe
 */
export const EARTH_RADIUS = 1;

/**
 * Convert latitude/longitude to 3D cartesian coordinates on a sphere
 * @param lat Latitude in degrees (-90 to 90)
 * @param lng Longitude in degrees (-180 to 180)
 * @param radius Sphere radius (default: EARTH_RADIUS)
 * @returns THREE.Vector3 position on the sphere surface
 */
export function latLngToVector3(
  lat: number,
  lng: number,
  radius: number = EARTH_RADIUS
): THREE.Vector3 {
  // Convert to radians
  const phi = (90 - lat) * (Math.PI / 180);
  const theta = (lng + 180) * (Math.PI / 180);

  // Spherical to cartesian conversion
  const x = -radius * Math.sin(phi) * Math.cos(theta);
  const y = radius * Math.cos(phi);
  const z = radius * Math.sin(phi) * Math.sin(theta);

  return new THREE.Vector3(x, y, z);
}

/**
 * Create a curved arc (geodesic) between two points on the sphere
 * @param start Starting position on sphere
 * @param end Ending position on sphere
 * @param radius Sphere radius
 * @param arcHeight Height of the arc above the sphere surface (multiplier)
 * @param segments Number of segments for the curve
 * @returns CatmullRomCurve3 representing the arc
 */
export function createArcCurve(
  start: THREE.Vector3,
  end: THREE.Vector3,
  radius: number = EARTH_RADIUS,
  arcHeight: number = 0.4,
  segments: number = 50
): THREE.CatmullRomCurve3 {
  const points: THREE.Vector3[] = [];
  
  // Calculate the angle between start and end
  const angle = start.angleTo(end);
  
  // Dynamic arc height based on distance
  const dynamicHeight = Math.min(arcHeight, angle * 0.3);
  
  for (let i = 0; i <= segments; i++) {
    const t = i / segments;
    
    // Spherical interpolation (slerp) for great circle path
    const point = new THREE.Vector3().lerpVectors(start, end, t);
    point.normalize();
    
    // Add height for arc effect (parabolic)
    const heightMultiplier = 1 + dynamicHeight * Math.sin(t * Math.PI);
    point.multiplyScalar(radius * heightMultiplier);
    
    points.push(point);
  }

  return new THREE.CatmullRomCurve3(points);
}

/**
 * Get a random position on the sphere surface
 * @param radius Sphere radius
 * @returns Random Vector3 on sphere surface
 */
export function getRandomSpherePosition(radius: number = EARTH_RADIUS): THREE.Vector3 {
  const lat = Math.random() * 180 - 90;
  const lng = Math.random() * 360 - 180;
  return latLngToVector3(lat, lng, radius);
}

/**
 * Calculate the great circle distance between two lat/lng points
 * @param lat1 Latitude of point 1
 * @param lng1 Longitude of point 1
 * @param lat2 Latitude of point 2
 * @param lng2 Longitude of point 2
 * @returns Distance in radians
 */
export function greatCircleDistance(
  lat1: number,
  lng1: number,
  lat2: number,
  lng2: number
): number {
  const toRad = (deg: number) => deg * (Math.PI / 180);
  
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  
  const a =
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) *
    Math.sin(dLng / 2) * Math.sin(dLng / 2);
  
  return 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

/**
 * Country code to approximate lat/lng mapping
 * Used as fallback when geoip data is not available
 */
export const COUNTRY_COORDINATES: Record<string, { lat: number; lng: number }> = {
  // North America
  US: { lat: 37.0902, lng: -95.7129 },
  CA: { lat: 56.1304, lng: -106.3468 },
  MX: { lat: 23.6345, lng: -102.5528 },
  
  // Europe
  GB: { lat: 55.3781, lng: -3.436 },
  DE: { lat: 51.1657, lng: 10.4515 },
  FR: { lat: 46.2276, lng: 2.2137 },
  NL: { lat: 52.1326, lng: 5.2913 },
  IT: { lat: 41.8719, lng: 12.5674 },
  ES: { lat: 40.4637, lng: -3.7492 },
  PL: { lat: 51.9194, lng: 19.1451 },
  SE: { lat: 60.1282, lng: 18.6435 },
  FI: { lat: 61.9241, lng: 25.7482 },
  NO: { lat: 60.472, lng: 8.4689 },
  DK: { lat: 56.2639, lng: 9.5018 },
  CH: { lat: 46.8182, lng: 8.2275 },
  AT: { lat: 47.5162, lng: 14.5501 },
  BE: { lat: 50.5039, lng: 4.4699 },
  IE: { lat: 53.1424, lng: -7.6921 },
  PT: { lat: 39.3999, lng: -8.2245 },
  CZ: { lat: 49.8175, lng: 15.473 },
  RO: { lat: 45.9432, lng: 24.9668 },
  UA: { lat: 48.3794, lng: 31.1656 },
  RU: { lat: 61.524, lng: 105.3188 },
  
  // Asia
  JP: { lat: 36.2048, lng: 138.2529 },
  CN: { lat: 35.8617, lng: 104.1954 },
  HK: { lat: 22.3193, lng: 114.1694 },
  TW: { lat: 23.6978, lng: 120.9605 },
  KR: { lat: 35.9078, lng: 127.7669 },
  SG: { lat: 1.3521, lng: 103.8198 },
  IN: { lat: 20.5937, lng: 78.9629 },
  ID: { lat: -0.7893, lng: 113.9213 },
  MY: { lat: 4.2105, lng: 101.9758 },
  TH: { lat: 15.87, lng: 100.9925 },
  VN: { lat: 14.0583, lng: 108.2772 },
  PH: { lat: 12.8797, lng: 121.774 },
  
  // Oceania
  AU: { lat: -25.2744, lng: 133.7751 },
  NZ: { lat: -40.9006, lng: 174.886 },
  
  // South America
  BR: { lat: -14.235, lng: -51.9253 },
  AR: { lat: -38.4161, lng: -63.6167 },
  CL: { lat: -35.6751, lng: -71.543 },
  CO: { lat: 4.5709, lng: -74.2973 },
  
  // Africa
  ZA: { lat: -30.5595, lng: 22.9375 },
  EG: { lat: 26.8206, lng: 30.8025 },
  NG: { lat: 9.082, lng: 8.6753 },
  KE: { lat: -0.0236, lng: 37.9062 },
  
  // Middle East
  AE: { lat: 23.4241, lng: 53.8478 },
  IL: { lat: 31.0461, lng: 34.8516 },
  TR: { lat: 38.9637, lng: 35.2433 },
  SA: { lat: 23.8859, lng: 45.0792 },
};

/**
 * Get coordinates for a country code, with fallback to random position
 * @param countryCode ISO 3166-1 alpha-2 country code
 * @returns Coordinates { lat, lng }
 */
export function getCountryCoordinates(countryCode?: string): { lat: number; lng: number } {
  if (countryCode && COUNTRY_COORDINATES[countryCode.toUpperCase()]) {
    return COUNTRY_COORDINATES[countryCode.toUpperCase()];
  }
  // Random fallback
  return {
    lat: Math.random() * 120 - 60,
    lng: Math.random() * 360 - 180,
  };
}
