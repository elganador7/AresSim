import theaterMaritime from "../../../internal/geo/data/theater_maritime.json";

export interface TheaterMaritimeFeature {
  type: "Feature";
  properties: {
    owner: string;
    zoneType: string;
  };
  geometry: {
    type: "Polygon" | "MultiPolygon";
    coordinates: [number, number][][] | [number, number][][][];
  };
}

export interface TheaterMaritimeCollection {
  type: "FeatureCollection";
  features: TheaterMaritimeFeature[];
}

export const THEATER_MARITIME_GEOJSON = theaterMaritime as TheaterMaritimeCollection;
