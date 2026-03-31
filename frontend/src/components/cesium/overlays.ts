import {
  Color,
  ColorMaterialProperty,
  ConstantProperty,
  GeoJsonDataSource,
  Viewer,
} from "cesium";

type TheaterOverlayGeoJSON = {
  type: "FeatureCollection";
  features: unknown[];
};

let theaterBordersPromise: Promise<TheaterOverlayGeoJSON> | null = null;
let theaterMaritimePromise: Promise<TheaterOverlayGeoJSON> | null = null;

async function fetchJson<T>(url: string): Promise<T> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`failed to load ${url}: ${response.status} ${response.statusText}`);
  }
  return response.json() as Promise<T>;
}

function loadTheaterBorders() {
  if (!theaterBordersPromise) {
    theaterBordersPromise = fetchJson<TheaterOverlayGeoJSON>("/theater/theater_borders.json");
  }
  return theaterBordersPromise;
}

function loadTheaterMaritime() {
  if (!theaterMaritimePromise) {
    theaterMaritimePromise = fetchJson<TheaterOverlayGeoJSON>("/theater/theater_maritime.json");
  }
  return theaterMaritimePromise;
}

function styleBorderOverlay(dataSource: GeoJsonDataSource) {
  dataSource.entities.values.forEach((entity) => {
    if (entity.polygon) {
      entity.polygon.fill = new ConstantProperty(false);
      entity.polygon.outline = new ConstantProperty(true);
      entity.polygon.outlineColor = new ConstantProperty(Color.fromCssColorString("#90a3b8").withAlpha(0.5));
      entity.polygon.outlineWidth = new ConstantProperty(1.1);
    }
    if (entity.polyline) {
      entity.polyline.material = new ColorMaterialProperty(Color.fromCssColorString("#90a3b8").withAlpha(0.55));
      entity.polyline.width = new ConstantProperty(1.1);
    }
  });
}

function styleMaritimeOverlay(dataSource: GeoJsonDataSource) {
  dataSource.entities.values.forEach((entity) => {
    if (entity.polygon) {
      entity.polygon.fill = new ConstantProperty(true);
      entity.polygon.material = new ColorMaterialProperty(Color.fromCssColorString("#155e75").withAlpha(0.06));
      entity.polygon.outline = new ConstantProperty(true);
      entity.polygon.outlineColor = new ConstantProperty(Color.fromCssColorString("#67e8f9").withAlpha(0.18));
      entity.polygon.outlineWidth = new ConstantProperty(0.9);
    }
    if (entity.polyline) {
      entity.polyline.material = new ColorMaterialProperty(Color.fromCssColorString("#67e8f9").withAlpha(0.18));
      entity.polyline.width = new ConstantProperty(0.9);
    }
  });
}

export async function loadTheaterOverlays(viewer: Viewer) {
  const [theaterBorders, theaterMaritime] = await Promise.all([
    loadTheaterBorders(),
    loadTheaterMaritime(),
  ]);
  const [borderDataSource, maritimeDataSource] = await Promise.all([
    GeoJsonDataSource.load(theaterBorders as never, {
      stroke: Color.fromCssColorString("#9eb0c2"),
      fill: Color.fromCssColorString("#000000").withAlpha(0.01),
      strokeWidth: 1.2,
      clampToGround: true,
    }),
    GeoJsonDataSource.load(theaterMaritime as never, {
      stroke: Color.fromCssColorString("#67e8f9"),
      fill: Color.fromCssColorString("#155e75").withAlpha(0.06),
      strokeWidth: 0.9,
      clampToGround: true,
    }),
  ]);

  styleBorderOverlay(borderDataSource);
  styleMaritimeOverlay(maritimeDataSource);

  viewer.dataSources.add(borderDataSource);
  viewer.dataSources.add(maritimeDataSource);

  return { borderDataSource, maritimeDataSource };
}

export function removeTheaterOverlays(viewer: Viewer, overlays: {
  borderDataSource: GeoJsonDataSource | null;
  maritimeDataSource: GeoJsonDataSource | null;
}) {
  if (overlays.borderDataSource) {
    viewer.dataSources.remove(overlays.borderDataSource, true);
  }
  if (overlays.maritimeDataSource) {
    viewer.dataSources.remove(overlays.maritimeDataSource, true);
  }
}
