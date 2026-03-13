/**
 * EditorGlobe.tsx
 *
 * CesiumJS globe for the scenario editor.
 *
 * Supports:
 *  - Rendering unit draft entities colour-coded by side
 *  - Left-click on terrain → onMapClick(lat, lon)  [click-to-place fallback]
 *  - Left-click on entity → onUnitClick(unitId)
 *  - HTML5 dragover / drop on the container div → onUnitDrop(lat, lon, payload)
 *
 * All three callbacks are kept in refs to avoid remounting Cesium when
 * parent components create new function references.
 */

import { useEffect, useRef, useState } from "react";
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
  Cartesian2,
  ConstantProperty,
  VerticalOrigin,
  HorizontalOrigin,
  NearFarScalar,
  HeightReference,
} from "cesium";
import "cesium/Build/Cesium/Widgets/widgets.css";
import { useEditorStore, type UnitDraft } from "../../store/editorStore";
import type { DragPayload } from "./UnitPalette";
import { getUnitBillboardUrl } from "../../utils/unitBillboard";

// ─── HELPERS ──────────────────────────────────────────────────────────────────

function getDropLatLon(
  viewer: Viewer,
  container: HTMLDivElement,
  clientX: number,
  clientY: number,
): { lat: number; lon: number } | null {
  const rect = container.getBoundingClientRect();
  const canvasPos = new Cartesian2(clientX - rect.left, clientY - rect.top);
  const ray = viewer.camera.getPickRay(canvasPos);
  if (!ray) return null;
  const pos = viewer.scene.globe.pick(ray, viewer.scene);
  if (!pos) return null;
  const carto = Cartographic.fromCartesian(pos);
  return {
    lat: CesiumMath.toDegrees(carto.latitude),
    lon: CesiumMath.toDegrees(carto.longitude),
  };
}

const LABEL_COLORS: Record<string, Color> = {
  Blue:    Color.fromCssColorString("#dbeafe"),
  Red:     Color.fromCssColorString("#fee2e2"),
  Neutral: Color.fromCssColorString("#fef3c7"),
};

// ─── PROPS ────────────────────────────────────────────────────────────────────

interface Props {
  onMapClick: (lat: number, lon: number) => void;
  onUnitClick: (unitId: string) => void;
  onUnitDrop: (lat: number, lon: number, payload: DragPayload) => void;
  placementMode: boolean;
}

// ─── COMPONENT ────────────────────────────────────────────────────────────────

export default function EditorGlobe({
  onMapClick,
  onUnitClick,
  onUnitDrop,
  placementMode,
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewerRef = useRef<Viewer | null>(null);
  const [isDragOver, setIsDragOver] = useState(false);

  // Stable callback refs — updated every render, no need to remount Cesium
  const onMapClickRef = useRef(onMapClick);
  const onUnitClickRef = useRef(onUnitClick);
  const onUnitDropRef = useRef(onUnitDrop);
  useEffect(() => { onMapClickRef.current = onMapClick; }, [onMapClick]);
  useEffect(() => { onUnitClickRef.current = onUnitClick; }, [onUnitClick]);
  useEffect(() => { onUnitDropRef.current = onUnitDrop; }, [onUnitDrop]);

  // ── Mount Cesium once ──────────────────────────────────────────────────────
  useEffect(() => {
    const container = containerRef.current;
    if (!container || viewerRef.current) return;

    Ion.defaultAccessToken = "";

    const viewer = new Viewer(container, {
      baseLayer: new ImageryLayer(
        new OpenStreetMapImageryProvider({ url: "https://tile.openstreetmap.org/" }),
      ),
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

    viewer.camera.setView({
      destination: Cartesian3.fromDegrees(25.8, 35.8, 2_000_000),
    });

    // ── Cesium click handler ───────────────────────────────────────────────
    viewer.screenSpaceEventHandler.setInputAction(
      (evt: { position: Cartesian2 }) => {
        // Check for entity click first
        const picked = viewer.scene.pick(evt.position);
        if (picked?.id) {
          const uid = picked.id.properties?.unitId?.getValue?.();
          if (uid) {
            onUnitClickRef.current(uid);
            return;
          }
        }
        // Globe click → placement
        const ray = viewer.camera.getPickRay(evt.position);
        if (!ray) return;
        const pos = viewer.scene.globe.pick(ray, viewer.scene);
        if (!pos) return;
        const carto = Cartographic.fromCartesian(pos);
        onMapClickRef.current(
          CesiumMath.toDegrees(carto.latitude),
          CesiumMath.toDegrees(carto.longitude),
        );
      },
      ScreenSpaceEventType.LEFT_CLICK,
    );

    // ── HTML5 drag-and-drop (native DOM, not React synthetic) ─────────────
    const handleDragOver = (e: DragEvent) => {
      e.preventDefault();
      if (e.dataTransfer) e.dataTransfer.dropEffect = "copy";
    };
    const handleDragEnter = (e: DragEvent) => {
      e.preventDefault();
      setIsDragOver(true);
    };
    const handleDragLeave = (e: DragEvent) => {
      if (!container.contains(e.relatedTarget as Node | null)) {
        setIsDragOver(false);
      }
    };
    const handleDrop = (e: DragEvent) => {
      e.preventDefault();
      setIsDragOver(false);
      const raw = e.dataTransfer?.getData("text/plain");
      if (!raw || !viewerRef.current) return;
      let payload: DragPayload;
      try {
        payload = JSON.parse(raw) as DragPayload;
      } catch {
        return;
      }
      const coords = getDropLatLon(viewerRef.current, container, e.clientX, e.clientY);
      if (!coords) return;
      onUnitDropRef.current(coords.lat, coords.lon, payload);
    };

    container.addEventListener("dragover", handleDragOver);
    container.addEventListener("dragenter", handleDragEnter);
    container.addEventListener("dragleave", handleDragLeave);
    container.addEventListener("drop", handleDrop);

    return () => {
      container.removeEventListener("dragover", handleDragOver);
      container.removeEventListener("dragenter", handleDragEnter);
      container.removeEventListener("dragleave", handleDragLeave);
      container.removeEventListener("drop", handleDrop);
      viewer.destroy();
      viewerRef.current = null;
    };
  }, []); // empty — Cesium mounts once, callbacks use refs

  // ── Sync units from editorStore ───────────────────────────────────────────
  useEffect(() => {
    const syncUnits = (units: UnitDraft[]) => {
      const viewer = viewerRef.current;
      if (!viewer) return;

      // Build definitionId → generalType lookup from current editor definitions.
      const defs = useEditorStore.getState().unitDefinitions;
      const defMap: Record<string, number> = {};
      defs.forEach((d) => { defMap[d.id] = d.generalType; });

      const currentIds = new Set(units.map((u) => `editor-${u.id}`));

      // Remove stale entities
      for (const entity of [...viewer.entities.values]) {
        if (entity.properties?.unitId !== undefined && !currentIds.has(entity.id as string)) {
          viewer.entities.remove(entity);
        }
      }

      for (const unit of units) {
        const generalType = defMap[unit.definitionId] ?? 0;
        const labelColor  = LABEL_COLORS[unit.side] ?? Color.WHITE;
        const pos         = Cartesian3.fromDegrees(unit.lon, unit.lat, unit.altMsl);
        const entityId    = `editor-${unit.id}`;
        const existing    = viewer.entities.getById(entityId);

        if (existing) {
          (existing.position as unknown as { setValue(p: Cartesian3): void }).setValue(pos);
          if (existing.billboard) {
            existing.billboard.image = new ConstantProperty(
              getUnitBillboardUrl(generalType, unit.side),
            );
          }
          if (existing.label) {
            existing.label.text = new ConstantProperty(unit.displayName);
          }
        } else {
          viewer.entities.add({
            id: entityId,
            position: pos,
            billboard: {
              image: getUnitBillboardUrl(generalType, unit.side),
              width: 36,
              height: 36,
              verticalOrigin: VerticalOrigin.CENTER,
              horizontalOrigin: HorizontalOrigin.CENTER,
              scaleByDistance: new NearFarScalar(1.5e5, 1.2, 8e6, 0.4),
              disableDepthTestDistance: Number.POSITIVE_INFINITY,
              heightReference: HeightReference.CLAMP_TO_GROUND,
            },
            label: {
              text: unit.displayName,
              font: "bold 11px 'Courier New'",
              fillColor: labelColor,
              outlineColor: Color.BLACK,
              outlineWidth: 2,
              style: 2, // FILL_AND_OUTLINE
              verticalOrigin: VerticalOrigin.BOTTOM,
              horizontalOrigin: HorizontalOrigin.CENTER,
              pixelOffset: new Cartesian2(0, -22),
              disableDepthTestDistance: Number.POSITIVE_INFINITY,
            },
            properties: { unitId: unit.id },
          });
        }
      }
    };

    syncUnits(useEditorStore.getState().draft.units);
    return useEditorStore.subscribe((state, prev) => {
      if (state.draft.units !== prev.draft.units) syncUnits(state.draft.units);
    });
  }, []);

  // ── Highlight selected unit ───────────────────────────────────────────────
  useEffect(() => {
    const highlight = (selectedId: string | null) => {
      const viewer = viewerRef.current;
      if (!viewer) return;
      for (const entity of viewer.entities.values) {
        const uid = entity.properties?.unitId?.getValue?.();
        if (!uid || !entity.billboard) continue;
        const isSelected = uid === selectedId;
        entity.billboard.scale = new ConstantProperty(isSelected ? 1.4 : 1.0);
      }
    };
    highlight(useEditorStore.getState().selectedUnitId);
    return useEditorStore.subscribe((s, p) => {
      if (s.selectedUnitId !== p.selectedUnitId) highlight(s.selectedUnitId);
    });
  }, []);

  return (
    <div
      ref={containerRef}
      style={{
        width: "100%",
        height: "100%",
        cursor: placementMode ? "crosshair" : "default",
        outline: isDragOver ? "2px inset rgba(99,102,241,0.55)" : "none",
      }}
    />
  );
}
