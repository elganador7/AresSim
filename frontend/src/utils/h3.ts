import { cellToParent, latLngToCell } from "h3-js";

export const DEFAULT_H3_RESOLUTION = 12;
export const AGGREGATE_H3_RESOLUTION = 11;

export interface H3Fields {
  h3Cell?: string;
  h3ParentCell?: string;
}

export function h3FieldsForLatLon(lat: number, lon: number): H3Fields {
  if (!Number.isFinite(lat) || !Number.isFinite(lon)) {
    return {};
  }
  try {
    const h3Cell = latLngToCell(lat, lon, DEFAULT_H3_RESOLUTION);
    const h3ParentCell = cellToParent(h3Cell, AGGREGATE_H3_RESOLUTION);
    return { h3Cell, h3ParentCell };
  } catch {
    return {};
  }
}

export const pointH3Fields = h3FieldsForLatLon;

export function withH3Fields<T extends { lat: number; lon: number }>(point: T): T & H3Fields {
  return { ...point, ...h3FieldsForLatLon(point.lat, point.lon) };
}
