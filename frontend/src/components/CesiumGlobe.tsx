/**
 * CesiumGlobe.tsx
 *
 * Mounts a CesiumJS Viewer and drives entities from the Zustand store.
 * Uses OpenStreetMap tiles — no Ion token required.
 *
 * CesiumJS is driven imperatively via store.subscribe() — NOT React hooks —
 * so position updates from the sim loop do not cause React re-renders.
 */

import { useEffect, useRef } from "react";
import {
  Viewer,
  Ion,
  OpenStreetMapImageryProvider,
  ImageryLayer,
  Cartesian3,
  Color,
  LabelStyle,
  VerticalOrigin,
  HorizontalOrigin,
  NearFarScalar,
  HeightReference,
  Entity,
} from "cesium";
import "cesium/Build/Cesium/Widgets/widgets.css";
import { useSimStore, Unit } from "../store/simStore";

const SIDE_COLOR: Record<string, Color> = {
  Blue: Color.fromCssColorString("#3b82f6"),
  Red: Color.fromCssColorString("#ef4444"),
  Neutral: Color.fromCssColorString("#f59e0b"),
};

const SIDE_OUTLINE: Record<string, Color> = {
  Blue: Color.fromCssColorString("#93c5fd"),
  Red: Color.fromCssColorString("#fca5a5"),
  Neutral: Color.fromCssColorString("#fde68a"),
};

const LABEL_COLOR: Record<string, Color> = {
  Blue: Color.fromCssColorString("#dbeafe"),
  Red: Color.fromCssColorString("#fee2e2"),
  Neutral: Color.fromCssColorString("#fef3c7"),
};

function makeEntity(unit: Unit): Entity {
  const color = SIDE_COLOR[unit.side] ?? Color.WHITE;
  const outline = SIDE_OUTLINE[unit.side] ?? Color.GRAY;
  const labelColor = LABEL_COLOR[unit.side] ?? Color.WHITE;

  return new Entity({
    id: unit.id,
    position: Cartesian3.fromDegrees(unit.position.lon, unit.position.lat, unit.position.altMsl),
    point: {
      pixelSize: 10,
      color,
      outlineColor: outline,
      outlineWidth: 2,
      heightReference: HeightReference.CLAMP_TO_GROUND,
      scaleByDistance: new NearFarScalar(1.5e5, 1.5, 8e6, 0.6),
    },
    label: {
      text: unit.displayName,
      font: "12px 'Courier New', monospace",
      style: LabelStyle.FILL_AND_OUTLINE,
      fillColor: labelColor,
      outlineColor: Color.fromCssColorString("#0f1115"),
      outlineWidth: 3,
      verticalOrigin: VerticalOrigin.BOTTOM,
      horizontalOrigin: HorizontalOrigin.CENTER,
      pixelOffset: { x: 0, y: -14 } as unknown as Cartesian3,
      scaleByDistance: new NearFarScalar(1.5e5, 1.2, 8e6, 0.5),
      disableDepthTestDistance: Number.POSITIVE_INFINITY,
    },
  });
}

export default function CesiumGlobe() {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewerRef = useRef<Viewer | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    Ion.defaultAccessToken = "";

    const osmProvider = new OpenStreetMapImageryProvider({
      url: "https://tile.openstreetmap.org/",
      credit: "© OpenStreetMap contributors",
    });

    const viewer = new Viewer(containerRef.current, {
      baseLayer: new ImageryLayer(osmProvider),
      terrainProvider: undefined,
      baseLayerPicker: false,
      geocoder: false,
      homeButton: false,
      sceneModePicker: false,
      navigationHelpButton: false,
      animation: false,
      timeline: false,
      fullscreenButton: false,
      vrButton: false,
      infoBox: false,
      selectionIndicator: false,
    });

    viewer.scene.globe.enableLighting = false;
    viewer.scene.backgroundColor = Color.fromCssColorString("#0f1115");
    viewerRef.current = viewer;

    // Initial camera — Eastern Mediterranean.
    viewer.camera.flyTo({
      destination: Cartesian3.fromDegrees(25.8, 35.8, 1_200_000),
      duration: 0,
    });

    // Render helper — syncs all active units from a units Map into Cesium entities.
    const syncUnits = (units: Map<string, import("../store/simStore").Unit>) => {
      units.forEach((unit) => {
        if (!unit.status.isActive) {
          viewer.entities.removeById(unit.id);
          return;
        }
        const pos = Cartesian3.fromDegrees(unit.position.lon, unit.position.lat, unit.position.altMsl);
        const existing = viewer.entities.getById(unit.id);
        if (existing) {
          (existing.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(pos);
        } else {
          viewer.entities.add(makeEntity(unit));
        }
      });
      // Remove stale entities.
      const storeIds = new Set(units.keys());
      Array.from(viewer.entities.values)
        .map((e) => e.id)
        .filter((id) => !storeIds.has(id))
        .forEach((id) => viewer.entities.removeById(id));
    };

    // Render any units already in the store at mount time (e.g. after RequestSync).
    syncUnits(useSimStore.getState().units);

    // Subscribe to store imperatively — no React re-renders on tick updates.
    const unsub = useSimStore.subscribe((state, prev) => {
      if (state.units === prev.units) return;
      syncUnits(state.units);
    });

    return () => {
      unsub();
      if (!viewer.isDestroyed()) viewer.destroy();
      viewerRef.current = null;
    };
  }, []);

  return <div ref={containerRef} style={{ position: "absolute", inset: 0 }} />;
}
