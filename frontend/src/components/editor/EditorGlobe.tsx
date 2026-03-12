/**
 * EditorGlobe.tsx
 *
 * Cesium globe for the scenario editor. Subscribed to editorStore (not simStore).
 * Supports:
 *  - Rendering unit draft entities colour-coded by side
 *  - Left-click on empty terrain → onMapClick(lat, lon)
 *  - Left-click on entity → onUnitClick(unitId)
 */

import { useEffect, useRef } from "react";
import {
  Viewer,
  Ion,
  OpenStreetMapImageryProvider,
  ImageryLayer,
  Color,
  Cartesian3,
  Cartographic,
  ScreenSpaceEventType,
  Math as CesiumMath,
  PointGraphics,
  LabelGraphics,
  HorizontalOrigin,
  VerticalOrigin,
  Cartesian2,
} from "cesium";
import "cesium/Build/Cesium/Widgets/widgets.css";
import { useEditorStore, type UnitDraft } from "../../store/editorStore";

interface Props {
  onMapClick: (lat: number, lon: number) => void;
  onUnitClick: (unitId: string) => void;
  /** When true, the cursor indicates placement mode */
  placementMode: boolean;
}

const SIDE_COLORS: Record<string, Color> = {
  Blue: Color.fromCssColorString("#3b82f6"),
  Red: Color.fromCssColorString("#ef4444"),
  Neutral: Color.fromCssColorString("#f59e0b"),
};

export default function EditorGlobe({ onMapClick, onUnitClick, placementMode }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewerRef = useRef<Viewer | null>(null);

  // Mount Cesium viewer once
  useEffect(() => {
    if (!containerRef.current || viewerRef.current) return;

    Ion.defaultAccessToken = "";

    const osmProvider = new OpenStreetMapImageryProvider({
      url: "https://tile.openstreetmap.org/",
    });

    const viewer = new Viewer(containerRef.current, {
      baseLayer: new ImageryLayer(osmProvider),
      timeline: false,
      animation: false,
      geocoder: false,
      homeButton: false,
      sceneModePicker: false,
      baseLayerPicker: false,
      navigationHelpButton: false,
      fullscreenButton: false,
      infoBox: false,
      selectionIndicator: false,
    });
    viewerRef.current = viewer;

    // Initial camera: Mediterranean
    viewer.camera.setView({
      destination: Cartesian3.fromDegrees(25.8, 35.8, 2_000_000),
    });

    // Click handler
    viewer.screenSpaceEventHandler.setInputAction((evt: { position: Cartesian2 }) => {
      const ray = viewer.camera.getPickRay(evt.position);
      if (!ray) return;

      // Check if we clicked an entity first
      const picked = viewer.scene.pick(evt.position);
      if (picked && picked.id && typeof picked.id.properties?.unitId?.getValue === "function") {
        const uid = picked.id.properties.unitId.getValue();
        onUnitClick(uid);
        return;
      }

      // Otherwise: terrain / globe click
      const globe = viewer.scene.globe;
      const pos = globe.pick(ray, viewer.scene);
      if (!pos) return;
      const carto = Cartographic.fromCartesian(pos);
      onMapClick(
        CesiumMath.toDegrees(carto.latitude),
        CesiumMath.toDegrees(carto.longitude),
      );
    }, ScreenSpaceEventType.LEFT_CLICK);

    return () => {
      viewer.destroy();
      viewerRef.current = null;
    };
  }, [onMapClick, onUnitClick]);

  // Sync units from editorStore
  useEffect(() => {
    const sync = (units: UnitDraft[]) => {
      const viewer = viewerRef.current;
      if (!viewer) return;

      const current = new Set(units.map((u) => u.id));

      // Remove stale entities
      viewer.entities.values
        .filter((e) => e.properties?.unitId !== undefined)
        .forEach((e) => {
          const uid = e.properties?.unitId?.getValue();
          if (!current.has(uid)) viewer.entities.remove(e);
        });

      for (const unit of units) {
        const color = SIDE_COLORS[unit.side] ?? Color.GRAY;
        const pos = Cartesian3.fromDegrees(unit.lon, unit.lat, unit.altMsl);
        const existing = viewer.entities.getById(`editor-${unit.id}`);

        if (existing) {
          (existing.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(pos);
          if (existing.point) {
            existing.point.color = new PointGraphics({ color }).color;
          }
          if (existing.label) {
            existing.label.text = new LabelGraphics({ text: unit.displayName }).text;
          }
        } else {
          viewer.entities.add({
            id: `editor-${unit.id}`,
            position: pos,
            point: {
              pixelSize: 12,
              color,
              outlineColor: Color.WHITE,
              outlineWidth: 1.5,
            },
            label: {
              text: unit.displayName,
              font: "11px Courier New",
              fillColor: Color.WHITE,
              outlineColor: Color.BLACK,
              outlineWidth: 2,
              style: 2, // FILL_AND_OUTLINE
              pixelOffset: new Cartesian2(0, -20),
              horizontalOrigin: HorizontalOrigin.CENTER,
              verticalOrigin: VerticalOrigin.BOTTOM,
              disableDepthTestDistance: Number.POSITIVE_INFINITY,
            },
            properties: { unitId: unit.id },
          });
        }
      }
    };

    sync(useEditorStore.getState().draft.units);
    const unsub = useEditorStore.subscribe((state, prev) => {
      if (state.draft.units !== prev.draft.units) {
        sync(state.draft.units);
      }
    });
    return unsub;
  }, []);

  // Highlight selected unit
  useEffect(() => {
    const highlight = (selectedId: string | null) => {
      const viewer = viewerRef.current;
      if (!viewer) return;
      for (const entity of viewer.entities.values) {
        const uid = entity.properties?.unitId?.getValue();
        if (!uid) continue;
        const selected = uid === selectedId;
        if (entity.point) {
          entity.point.pixelSize = selected
            ? new PointGraphics({ pixelSize: 18 }).pixelSize
            : new PointGraphics({ pixelSize: 12 }).pixelSize;
          entity.point.outlineWidth = selected
            ? new PointGraphics({ outlineWidth: 3 }).outlineWidth
            : new PointGraphics({ outlineWidth: 1.5 }).outlineWidth;
        }
      }
    };
    highlight(useEditorStore.getState().selectedUnitId);
    const unsub = useEditorStore.subscribe((s, p) => {
      if (s.selectedUnitId !== p.selectedUnitId) highlight(s.selectedUnitId);
    });
    return unsub;
  }, []);

  return (
    <div
      ref={containerRef}
      style={{
        width: "100%",
        height: "100%",
        cursor: placementMode ? "crosshair" : "default",
      }}
    />
  );
}
