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
  BoundingSphere,
  Color,
  GeoJsonDataSource,
  ScreenSpaceEventType,
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
  const viewerRef = useRef<Viewer | null>(null);
  const borderDataSourceRef = useRef<GeoJsonDataSource | null>(null);
  const maritimeDataSourceRef = useRef<GeoJsonDataSource | null>(null);
  const mapCommandMode = useSimStore((s) => s.mapCommandMode);
  const oilGraph = useSimStore((s) => s.oilGraph);
  const oilFocusToken = useSimStore((s) => s.oilFocusToken);
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
    viewerRef.current = viewer;

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

    // Default Cesium double-click zoom is too aggressive for unit interaction.
    viewer.screenSpaceEventHandler.removeInputAction(ScreenSpaceEventType.LEFT_DOUBLE_CLICK);

    setupCesiumInteractions(viewer, defInfoRef, draggingWaypointRef, suppressClickRef);

    const stopSync = setupCesiumStoreSync({ viewer, containerRef, defInfoRef });

    return () => {
      stopSync();
      removeTheaterOverlays(viewer, {
        borderDataSource: borderDataSourceRef.current,
        maritimeDataSource: maritimeDataSourceRef.current,
      });
      viewerRef.current = null;
      borderDataSourceRef.current = null;
      maritimeDataSourceRef.current = null;
      if (!viewer.isDestroyed()) viewer.destroy();
    };
  }, []);

  useEffect(() => {
    if (!viewerRef.current || !oilGraph || oilGraph.nodes.length === 0 || oilFocusToken === 0) {
      return;
    }
    const positions = oilGraph.nodes
      .slice(0, 4000)
      .map((node) => Cartesian3.fromDegrees(node.lon, node.lat, 0));
    if (positions.length === 0) {
      return;
    }
    viewerRef.current.camera.flyToBoundingSphere(BoundingSphere.fromPoints(positions), {
      duration: 1.2,
    });
  }, [oilFocusToken, oilGraph]);

  return (
    <div
      ref={containerRef}
      style={{
        position: "absolute",
        inset: 0,
        cursor: mapCommandMode.type === "route" ? "copy" : "default",
      }}
    />
  );
}
