import theaterBorders from "../../../internal/geo/data/theater_borders.json";

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

export const THEATER_BORDERS_GEOJSON = theaterBorders as TheaterBorderCollection;
