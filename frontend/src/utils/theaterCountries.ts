import { THEATER_BORDERS_GEOJSON } from "../data/theaterBorders";

function pointInRing(lon: number, lat: number, ring: [number, number][]): boolean {
  let inside = false;
  for (let i = 0, j = ring.length - 1; i < ring.length; j = i++) {
    const [xi, yi] = ring[i];
    const [xj, yj] = ring[j];
    const intersects =
      yi > lat !== yj > lat &&
      lon < ((xj - xi) * (lat - yi)) / ((yj - yi) || Number.EPSILON) + xi;
    if (intersects) {
      inside = !inside;
    }
  }
  return inside;
}

export function getCountryCodeForPoint(lat: number, lon: number): string {
  for (const feature of THEATER_BORDERS_GEOJSON.features) {
    const ring = feature.geometry.coordinates[0];
    if (pointInRing(lon, lat, ring)) {
      return feature.properties.iso3;
    }
  }
  return "";
}

export function getCountriesAlongSegment(
  start: { lat: number; lon: number },
  end: { lat: number; lon: number },
): string[] {
  const steps = Math.max(
    8,
    Math.min(
      96,
      Math.ceil(Math.max(Math.abs(end.lat - start.lat), Math.abs(end.lon - start.lon)) / 0.2),
    ),
  );
  const seen = new Set<string>();
  for (let i = 0; i <= steps; i += 1) {
    const t = i / steps;
    const lat = start.lat + (end.lat - start.lat) * t;
    const lon = start.lon + (end.lon - start.lon) * t;
    const code = getCountryCodeForPoint(lat, lon);
    if (code) {
      seen.add(code);
    }
  }
  return Array.from(seen);
}

export function getCountriesAlongRoute(
  origin: { lat: number; lon: number },
  waypoints: { lat: number; lon: number }[],
): string[] {
  const seen = new Set<string>();
  let current = origin;
  for (const waypoint of waypoints) {
    for (const code of getCountriesAlongSegment(current, waypoint)) {
      seen.add(code);
    }
    current = waypoint;
  }
  return Array.from(seen);
}
