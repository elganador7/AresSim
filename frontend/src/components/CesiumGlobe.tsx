/**
 * CesiumGlobe.tsx
 *
 * Mounts a CesiumJS Viewer and drives entities from the Zustand store.
 * Uses OpenStreetMap tiles — no Ion token required.
 *
 * All state is driven imperatively via store.subscribe() — NOT React hooks —
 * so updates from the sim loop do not cause React re-renders.
 *
 * Interactions:
 *   Left-click entity  → select unit (highlights billboard, opens panel)
 *   Left-click terrain → move selected unit (if the active view owns that team)
 *
 * Fog-of-war:
 *   "debug" view  — all units visible
 *   nation view   — own-team units always; all other units only if detected
 *                   or explicitly shared into that team's picture
 */

import { useEffect, useRef } from "react";
import {
  Viewer,
  Ion,
  OpenStreetMapImageryProvider,
  ImageryLayer,
  Cartesian3,
  Color,
  GeoJsonDataSource,
} from "cesium";
import "cesium/Build/Cesium/Widgets/widgets.css";
import { useSimStore } from "../store/simStore";
import { type DefInfo } from "./cesium/helpers";
import { setupCesiumInteractions } from "./cesium/interactions";
import { loadTheaterOverlays, removeTheaterOverlays } from "./cesium/overlays";
import { setupCesiumStoreSync } from "./cesium/sync";

// ─── COMPONENT ────────────────────────────────────────────────────────────────

export default function CesiumGlobe() {
  const containerRef = useRef<HTMLDivElement>(null);
  const borderDataSourceRef = useRef<GeoJsonDataSource | null>(null);
  const maritimeDataSourceRef = useRef<GeoJsonDataSource | null>(null);
  const mapCommandMode = useSimStore((s) => s.mapCommandMode);
  const draggingWaypointRef = useRef<{ unitId: string; waypointIndex: number } | null>(null);
  const suppressClickRef = useRef(false);
  // definitionId → { generalType, combatRangeM }, populated from DB on mount
  const defInfoRef   = useRef<Record<string, DefInfo>>({});

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

    loadTheaterOverlays(viewer)
      .then(({ borderDataSource, maritimeDataSource }) => {
        borderDataSourceRef.current = borderDataSource;
        maritimeDataSourceRef.current = maritimeDataSource;
      })
      .catch(console.error);

    // Initial camera — Eastern Mediterranean.
    viewer.camera.flyTo({
      destination: Cartesian3.fromDegrees(25.8, 35.8, 1_200_000),
      duration: 0,
    });

    setupCesiumInteractions(viewer, defInfoRef, draggingWaypointRef, suppressClickRef);

    const stopSync = setupCesiumStoreSync({ viewer, containerRef, defInfoRef });

    return () => {
      stopSync();
      removeTheaterOverlays(viewer, {
        borderDataSource: borderDataSourceRef.current,
        maritimeDataSource: maritimeDataSourceRef.current,
      });
      borderDataSourceRef.current = null;
      maritimeDataSourceRef.current = null;
      if (!viewer.isDestroyed()) viewer.destroy();
    };
  }, []);

  return (
    <div
      ref={containerRef}
      style={{
        position: "absolute",
        inset: 0,
        cursor: mapCommandMode.type === "target_pick" ? "crosshair" : mapCommandMode.type === "route" ? "copy" : "default",
      }}
    />
  );
}
