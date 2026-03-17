export type LngLat = [number, number];

export interface TheaterCountryFeature {
  type: "Feature";
  properties: {
    iso3: string;
    name: string;
  };
  geometry: {
    type: "Polygon";
    coordinates: LngLat[][];
  };
}

export interface TheaterBorderCollection {
  type: "FeatureCollection";
  features: TheaterCountryFeature[];
}

function closeRing(points: LngLat[]): LngLat[] {
  if (points.length === 0) return points;
  const [fx, fy] = points[0];
  const [lx, ly] = points[points.length - 1];
  if (fx === lx && fy === ly) return points;
  return [...points, points[0]];
}

function polygon(iso3: string, name: string, points: LngLat[]): TheaterCountryFeature {
  return {
    type: "Feature",
    properties: { iso3, name },
    geometry: {
      type: "Polygon",
      coordinates: [closeRing(points)],
    },
  };
}

// Simplified regional border layer for the Iran-war theater.
// These polygons are intentionally coarse. They are used first for map
// visualization and later as a local geometry source for theater-level country
// lookup and access-control logic.
export const THEATER_BORDERS_GEOJSON: TheaterBorderCollection = {
  type: "FeatureCollection",
  features: [
    polygon("ISR", "Israel", [
      [34.25, 29.50],
      [35.72, 29.50],
      [35.88, 31.20],
      [35.70, 32.10],
      [35.55, 33.28],
      [34.95, 33.30],
      [34.25, 31.10],
    ]),
    polygon("LBN", "Lebanon", [
      [35.10, 33.05],
      [36.62, 33.05],
      [36.57, 34.69],
      [35.10, 34.69],
    ]),
    polygon("JOR", "Jordan", [
      [34.95, 29.10],
      [39.35, 29.10],
      [39.30, 33.40],
      [35.45, 33.40],
      [35.05, 32.30],
      [34.95, 30.90],
    ]),
    polygon("SYR", "Syria", [
      [35.70, 32.35],
      [42.35, 32.35],
      [42.35, 37.28],
      [36.00, 37.28],
      [35.70, 35.90],
    ]),
    polygon("IRQ", "Iraq", [
      [38.80, 29.00],
      [48.60, 29.00],
      [48.60, 37.40],
      [42.35, 37.40],
      [38.80, 33.10],
    ]),
    polygon("IRN", "Iran", [
      [44.00, 25.00],
      [50.50, 25.00],
      [56.90, 25.20],
      [61.30, 25.10],
      [63.30, 27.20],
      [63.40, 31.00],
      [61.80, 36.00],
      [60.20, 37.80],
      [55.60, 39.80],
      [47.00, 39.20],
      [44.20, 37.60],
      [44.00, 30.50],
    ]),
    polygon("KWT", "Kuwait", [
      [46.55, 28.50],
      [48.48, 28.50],
      [48.48, 30.10],
      [46.55, 30.10],
    ]),
    polygon("SAU", "Saudi Arabia", [
      [34.60, 16.20],
      [36.80, 31.90],
      [42.00, 32.20],
      [50.30, 31.80],
      [55.80, 26.00],
      [55.40, 20.50],
      [51.80, 16.50],
      [42.00, 16.20],
    ]),
    polygon("BHR", "Bahrain", [
      [50.28, 25.50],
      [50.82, 25.50],
      [50.82, 26.42],
      [50.28, 26.42],
    ]),
    polygon("QAT", "Qatar", [
      [50.70, 24.45],
      [51.75, 24.45],
      [51.75, 26.20],
      [50.70, 26.20],
    ]),
    polygon("ARE", "United Arab Emirates", [
      [51.45, 22.55],
      [56.45, 22.55],
      [56.45, 26.10],
      [54.90, 26.10],
      [53.00, 25.80],
      [51.45, 24.20],
    ]),
    polygon("OMN", "Oman", [
      [52.00, 16.60],
      [59.95, 16.60],
      [59.95, 26.40],
      [56.50, 26.40],
      [55.10, 24.90],
      [53.20, 24.20],
      [52.00, 21.80],
    ]),
    polygon("EGY", "Egypt", [
      [24.70, 22.00],
      [36.95, 22.00],
      [36.95, 31.70],
      [32.20, 31.70],
      [34.90, 29.10],
      [34.20, 28.70],
      [32.00, 29.70],
      [24.70, 31.70],
    ]),
    polygon("TUR", "Turkey", [
      [26.00, 35.80],
      [44.90, 35.80],
      [44.90, 42.20],
      [28.00, 42.20],
      [26.00, 40.50],
    ]),
  ],
};
