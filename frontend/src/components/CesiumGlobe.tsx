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
 *   Left-click terrain → move selected unit (if the active view owns that side)
 *
 * Fog-of-war:
 *   "debug" view  — all units visible
 *   "blue"  view  — Blue units always; others only if detected (future)
 *   "red"   view  — Red  units always; others only if detected (future)
 */

import { useEffect, useRef } from "react";
import {
  Viewer,
  Ion,
  OpenStreetMapImageryProvider,
  ImageryLayer,
  Cartesian3,
  Cartesian2,
  Color,
  VerticalOrigin,
  HorizontalOrigin,
  NearFarScalar,
  HeightReference,
  Entity,
  EllipseGraphics,
  ConstantProperty,
  ColorMaterialProperty,
  PolylineDashMaterialProperty,
  ScreenSpaceEventType,
  Math as CesiumMath,
  Cartographic,
} from "cesium";
import "cesium/Build/Cesium/Widgets/widgets.css";
import { useSimStore, Unit, Munition, WeaponDef } from "../store/simStore";
import { ListUnitDefinitions, MoveUnit } from "../../wailsjs/go/main/App";
import { getUnitBillboardUrl } from "../utils/unitBillboard";

// ─── TYPES ────────────────────────────────────────────────────────────────────

type ActiveView = "debug" | "blue" | "red";

interface DefInfo {
  generalType: number;
  detectionRangeM: number;
  shortName: string;
}

// ─── HELPERS ──────────────────────────────────────────────────────────────────

const ROUTE_COLOR: Record<string, Color> = {
  Blue:    Color.fromCssColorString("#60a5fa"),
  Red:     Color.fromCssColorString("#f87171"),
  Neutral: Color.fromCssColorString("#fcd34d"),
};

type Detections = Map<string, Set<string>>;
type MunitionDetections = Map<string, Set<string>>;

const MUNITION_ENTITY_PREFIX = "mun_";
const SENSOR_COLOR = Color.fromCssColorString("#0f9fb8");

/**
 * Returns the longest weapon range (metres) among weapons with ammo remaining.
 * Returns 0 if the unit has no weapons or all are depleted.
 */
function maxWeaponRangeM(unit: Unit, weaponDefs: Map<string, WeaponDef>): number {
  let best = 0;
  for (const ws of unit.weapons) {
    if (ws.currentQty <= 0) continue;
    const def = weaponDefs.get(ws.weaponId);
    if (def && def.rangeM > best) best = def.rangeM;
  }
  return best;
}

/**
 * Returns true if the munition should be shown in the given view.
 * In debug mode all munitions are visible. In blue/red mode only munitions
 * detected by that side's sensors are shown.
 */
function isMunitionVisible(
  munition: Munition,
  view: ActiveView,
  munitionDetections: MunitionDetections,
): boolean {
  if (view === "debug") return true;
  return munitionDetections.get(view === "blue" ? "Blue" : "Red")?.has(munition.id) ?? false;
}

/**
 * Returns true if the unit should be shown in the given view.
 * Own-side units are always visible. Enemy units are visible only if
 * detected by at least one sensor on the viewing side.
 */
function isVisible(unit: Unit, view: ActiveView, detections: Detections): boolean {
  if (view === "debug") return true;
  if (view === "blue") {
    return unit.side === "Blue" || (detections.get("Blue")?.has(unit.id) ?? false);
  }
  if (view === "red") {
    return unit.side === "Red" || (detections.get("Red")?.has(unit.id) ?? false);
  }
  return false;
}

/**
 * Returns true if this unit is a sensor track (detected enemy) rather than
 * an own-side unit in the current view. Tracks get a different visual style.
 */
function isTrack(unit: Unit, view: ActiveView): boolean {
  if (view === "debug") return false;
  return view === "blue" ? unit.side !== "Blue" : unit.side !== "Red";
}

/** Returns true if the active view is allowed to issue move orders to a unit. */
function canMove(unit: Unit, view: ActiveView): boolean {
  if (view === "debug") return true;
  if (view === "blue")  return unit.side === "Blue";
  if (view === "red")   return unit.side === "Red";
  return false;
}

function makeEntity(unit: Unit, generalType: number, shortName: string): Entity {
  return new Entity({
    id: unit.id,
    position: Cartesian3.fromDegrees(unit.position.lon, unit.position.lat, unit.position.altMsl),
    show: true,
    billboard: {
      image: getUnitBillboardUrl(generalType, unit.side, shortName),
      width: 62,
      height: 62,
      verticalOrigin: VerticalOrigin.CENTER,
      horizontalOrigin: HorizontalOrigin.CENTER,
      scaleByDistance: new NearFarScalar(1.5e5, 1.2, 8e6, 0.4),
      disableDepthTestDistance: Number.POSITIVE_INFINITY,
      heightReference: HeightReference.CLAMP_TO_GROUND,
    },
  });
}

// Munitions are rendered as small bright orange points.
const MUNITION_COLOR = Color.fromCssColorString("#f97316"); // tailwind orange-500

// ─── COMPONENT ────────────────────────────────────────────────────────────────

export default function CesiumGlobe() {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewerRef    = useRef<Viewer | null>(null);
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

    // Initial camera — Eastern Mediterranean.
    viewer.camera.flyTo({
      destination: Cartesian3.fromDegrees(25.8, 35.8, 1_200_000),
      duration: 0,
    });

    // ── Click handler ──────────────────────────────────────────────────────
    viewer.screenSpaceEventHandler.setInputAction(
      (evt: { position: Cartesian2 }) => {
        const { units, selectedUnitId, activeView, selectUnit } =
          useSimStore.getState();

        // Check for entity click first.
        const picked = viewer.scene.pick(evt.position);
        if (picked?.id instanceof Entity) {
          const clickedId = (picked.id as Entity).id;
          if (units.has(clickedId)) {
            selectUnit(selectedUnitId === clickedId ? null : clickedId);
            return;
          }
        }

        // Terrain click → move selected unit if the view permits.
        if (!selectedUnitId) return;
        const unit = units.get(selectedUnitId);
        if (!unit || !canMove(unit, activeView)) return;

        const ray = viewer.camera.getPickRay(evt.position);
        if (!ray) return;
        const pos = viewer.scene.globe.pick(ray, viewer.scene);
        if (!pos) return;
        const carto = Cartographic.fromCartesian(pos);
        const lat = CesiumMath.toDegrees(carto.latitude);
        const lon = CesiumMath.toDegrees(carto.longitude);

        MoveUnit(selectedUnitId, lat, lon).catch(console.error);
        selectUnit(null);
      },
      ScreenSpaceEventType.LEFT_CLICK,
    );

    // ── Sync a single unit and its associated entities ─────────────────────
    const syncUnit = (
      unit: Unit,
      view: ActiveView,
      selectedId: string | null,
      detections: Detections,
    ) => {
      const routeId  = `${unit.id}_route`;
      const destId   = `${unit.id}_dest`;
      const rangeId  = `${unit.id}_range`;
      const sensorId = `${unit.id}_sensor`;

      if (!unit.status.isActive) {
        viewer.entities.removeById(unit.id);
        viewer.entities.removeById(routeId);
        viewer.entities.removeById(destId);
        viewer.entities.removeById(rangeId);
        viewer.entities.removeById(sensorId);
        return;
      }

      const visible    = isVisible(unit, view, detections);
      const track      = isTrack(unit, view);
      // Tracks: dimmer billboard + no route/dest arrows (we don't know their orders)
      const trackAlpha = track ? 0.55 : 1.0;
      const pos = Cartesian3.fromDegrees(
        unit.position.lon, unit.position.lat, unit.position.altMsl,
      );
      const isSelected = unit.id === selectedId;

      // Unit billboard entity.
      const existing = viewer.entities.getById(unit.id);
      if (existing) {
        (existing.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(pos);
        existing.show = visible;
        if (existing.billboard) {
          const def = defInfoRef.current[unit.definitionId];
          existing.billboard.image = new ConstantProperty(
            getUnitBillboardUrl(def?.generalType ?? 0, unit.side, def?.shortName ?? unit.displayName),
          );
          existing.billboard.scale = new ConstantProperty(isSelected ? 1.4 : 1.0);
          existing.billboard.color = new ConstantProperty(Color.WHITE.withAlpha(trackAlpha));
        }
      } else {
        const def = defInfoRef.current[unit.definitionId];
        const entity = makeEntity(unit, def?.generalType ?? 0, def?.shortName ?? unit.displayName);
        entity.show = visible;
        viewer.entities.add(entity);
        // Apply initial track alpha after adding.
        if (entity.billboard) {
          entity.billboard.color = new ConstantProperty(Color.WHITE.withAlpha(trackAlpha));
        }
      }

      // Route / destination entities — only for own-side units (not tracks).
      const order = unit.moveOrder;
      if (!track && order && order.waypoints.length > 0) {
        const routeColor = ROUTE_COLOR[unit.side] ?? Color.YELLOW;
        const positions: Cartesian3[] = [
          Cartesian3.fromDegrees(unit.position.lon, unit.position.lat),
          ...order.waypoints.map((wp) => Cartesian3.fromDegrees(wp.lon, wp.lat)),
        ];
        const last = order.waypoints[order.waypoints.length - 1];
        const destPos = Cartesian3.fromDegrees(last.lon, last.lat);

        const routeEntity = viewer.entities.getById(routeId);
        if (routeEntity) {
          (routeEntity.polyline!.positions as unknown as { setValue: (p: Cartesian3[]) => void })
            .setValue(positions);
          routeEntity.show = visible;
        } else {
          viewer.entities.add(new Entity({
            id: routeId,
            show: visible,
            polyline: {
              positions: new ConstantProperty(positions),
              width: 2,
              material: new PolylineDashMaterialProperty({ color: routeColor, dashLength: 16 }),
              clampToGround: false,
            },
          }));
        }

        const destEntity = viewer.entities.getById(destId);
        if (destEntity) {
          (destEntity.position as unknown as { setValue: (p: Cartesian3) => void })
            .setValue(destPos);
          destEntity.show = visible;
        } else {
          viewer.entities.add(new Entity({
            id: destId,
            show: visible,
            position: destPos,
            point: {
              pixelSize: 10,
              color: routeColor,
              outlineColor: Color.WHITE,
              outlineWidth: 2,
              disableDepthTestDistance: Number.POSITIVE_INFINITY,
            },
          }));
        }
      } else {
        viewer.entities.removeById(routeId);
        viewer.entities.removeById(destId);
      }

      // Weapon range ring — shown only when this unit is selected and visible.
      // Always remove and recreate so radius stays correct as ammo depletes.
      viewer.entities.removeById(rangeId);

      if (isSelected && visible) {
        const ringColor = ROUTE_COLOR[unit.side] ?? Color.WHITE;

        // Weapon range ring — longest range among weapons with ammo remaining.
        const { weaponDefs } = useSimStore.getState();
        const weaponRangeM = maxWeaponRangeM(unit, weaponDefs);

        if (weaponRangeM > 0) {
          viewer.entities.add(new Entity({
            id: rangeId,
            show: true,
            position: pos,
            ellipse: new EllipseGraphics({
              semiMajorAxis: new ConstantProperty(weaponRangeM),
              semiMinorAxis: new ConstantProperty(weaponRangeM),
              material: new ColorMaterialProperty(ringColor.withAlpha(0.12)),
              outline: true,
              outlineColor: ringColor.withAlpha(0.95),
              outlineWidth: new ConstantProperty(2),
              heightReference: new ConstantProperty(HeightReference.CLAMP_TO_GROUND),
            }),
          }));
        }

      }

      // Sensor range ring — shown only for the selected visible platform.
      const sensorRangeM = defInfoRef.current[unit.definitionId]?.detectionRangeM ?? 0;
      if (!visible || !isSelected || sensorRangeM <= 0) {
        viewer.entities.removeById(sensorId)
      } else {
        const sensorEntity = viewer.entities.getById(sensorId)
        const outlineAlpha = 0.75
        const fillAlpha = 0.1
        if (sensorEntity) {
          (sensorEntity.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(pos)
          sensorEntity.show = true
          if (sensorEntity.ellipse) {
            sensorEntity.ellipse.semiMajorAxis = new ConstantProperty(sensorRangeM)
            sensorEntity.ellipse.semiMinorAxis = new ConstantProperty(sensorRangeM)
            sensorEntity.ellipse.material = new ColorMaterialProperty(SENSOR_COLOR.withAlpha(fillAlpha))
            sensorEntity.ellipse.outlineColor = new ConstantProperty(SENSOR_COLOR.withAlpha(outlineAlpha))
            sensorEntity.ellipse.outlineWidth = new ConstantProperty(2)
          }
        } else {
          viewer.entities.add(new Entity({
            id: sensorId,
            show: true,
            position: pos,
            ellipse: new EllipseGraphics({
              semiMajorAxis: new ConstantProperty(sensorRangeM),
              semiMinorAxis: new ConstantProperty(sensorRangeM),
              material: new ColorMaterialProperty(SENSOR_COLOR.withAlpha(fillAlpha)),
              outline: true,
              outlineColor: SENSOR_COLOR.withAlpha(outlineAlpha),
              outlineWidth: new ConstantProperty(2),
              heightReference: new ConstantProperty(HeightReference.CLAMP_TO_GROUND),
            }),
          }))
        }
      }
    };

    const syncUnits = (
      units: Map<string, Unit>,
      view: ActiveView,
      selectedId: string | null,
      detections: Detections,
    ) => {
      units.forEach((unit) => syncUnit(unit, view, selectedId, detections));

      // Remove stale unit entities (route/dest/range are managed by syncUnit).
      const storeIds = new Set(units.keys());
      Array.from(viewer.entities.values)
        .map((e) => e.id as string)
        .filter((id) =>
          !id.endsWith("_route") &&
          !id.endsWith("_dest") &&
          !id.endsWith("_range") &&
          !id.endsWith("_sensor") &&
          !id.startsWith(MUNITION_ENTITY_PREFIX) && // managed by syncMunitions
          !storeIds.has(id),
        )
        .forEach((id) => viewer.entities.removeById(id));
    };

    // ── Sync a single in-flight munition entity ────────────────────────────
    const syncMunition = (
      munition: Munition,
      view: ActiveView,
      munitionDetections: MunitionDetections,
    ) => {
      const entityId = `${MUNITION_ENTITY_PREFIX}${munition.id}`;
      const visible = isMunitionVisible(munition, view, munitionDetections);
      const pos = Cartesian3.fromDegrees(munition.lon, munition.lat, 100);

      const existing = viewer.entities.getById(entityId);
      if (existing) {
        (existing.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(pos);
        existing.show = visible;
      } else {
        viewer.entities.add(new Entity({
          id: entityId,
          show: visible,
          position: pos,
          point: {
            pixelSize: 6,
            color: MUNITION_COLOR,
            outlineColor: Color.WHITE,
            outlineWidth: 1,
            disableDepthTestDistance: Number.POSITIVE_INFINITY,
          },
        }));
      }
    };

    const syncMunitions = (
      munitions: Map<string, Munition>,
      view: ActiveView,
      munitionDetections: MunitionDetections,
    ) => {
      munitions.forEach((m) => syncMunition(m, view, munitionDetections));

      // Remove entities for munitions no longer in-flight.
      const liveIds = new Set(
        Array.from(munitions.keys()).map((id) => `${MUNITION_ENTITY_PREFIX}${id}`),
      );
      Array.from(viewer.entities.values)
        .map((e) => e.id as string)
        .filter((id) => id.startsWith(MUNITION_ENTITY_PREFIX) && !liveIds.has(id))
        .forEach((id) => viewer.entities.removeById(id));
    };

    // ── Reapply fog-of-war when view or detections change ──────────────────
    const applyView = (units: Map<string, Unit>, view: ActiveView, detections: Detections) => {
      units.forEach((unit) => {
        const visible = isVisible(unit, view, detections);
        const e = viewer.entities.getById(unit.id);
        if (e) e.show = visible;
        const r = viewer.entities.getById(`${unit.id}_route`);
        if (r) r.show = visible;
        const d = viewer.entities.getById(`${unit.id}_dest`);
        if (d) d.show = visible;
        const rng = viewer.entities.getById(`${unit.id}_range`);
        if (rng) rng.show = visible;
        const sen = viewer.entities.getById(`${unit.id}_sensor`);
        if (sen) sen.show = visible;
      });
    };

    // ── Update cursor based on selection ──────────────────────────────────
    const updateCursor = (
      units: Map<string, Unit>,
      selectedId: string | null,
      view: ActiveView,
    ) => {
      if (!containerRef.current) return;
      const unit = selectedId ? units.get(selectedId) : null;
      const moveable = unit ? canMove(unit, view) : false;
      containerRef.current.style.cursor = moveable ? "crosshair" : "default";
    };

    // ── Load definition info, then initial render ──────────────────────────
    ListUnitDefinitions()
      .then((rows) => {
        const map: Record<string, DefInfo> = {};
        rows.forEach((r) => {
          const shortName = String(r["short_name"] ?? "").trim()
            || String(r["specific_type"] ?? "").trim()
            || String(r["name"] ?? "").trim();
          map[String(r["id"])] = {
            generalType:    Number(r["general_type"]),
            detectionRangeM: Number(r["detection_range_m"]) || 0,
            shortName,
          };
        });
        defInfoRef.current = map;
        const { units, activeView, selectedUnitId, detections } = useSimStore.getState();
        syncUnits(units, activeView, selectedUnitId, detections);
      })
      .catch(console.error);

    // Initial render with whatever is already in the store.
    const { units: initUnits, activeView: initView, selectedUnitId: initSel,
            detections: initDet, munitions: initMun,
            munitionDetections: initMunDet } = useSimStore.getState();
    syncUnits(initUnits, initView, initSel, initDet);
    syncMunitions(initMun, initView, initMunDet);

    // ── Store subscriptions (imperatively driven, no React re-renders) ─────
    const unsub = useSimStore.subscribe((state, prev) => {
      const unitsChanged           = state.units              !== prev.units;
      const viewChanged            = state.activeView         !== prev.activeView;
      const selectionChanged       = state.selectedUnitId     !== prev.selectedUnitId;
      const detectionsChanged      = state.detections         !== prev.detections;
      const munitionsChanged       = state.munitions          !== prev.munitions;
      const munitionDetectChanged  = state.munitionDetections !== prev.munitionDetections;

      if (unitsChanged) {
        syncUnits(state.units, state.activeView, state.selectedUnitId, state.detections);
        return; // syncUnits covers everything
      }
      if (viewChanged) {
        // View change affects all units and munitions — full passes required.
        syncUnits(state.units, state.activeView, state.selectedUnitId, state.detections);
        syncMunitions(state.munitions, state.activeView, state.munitionDetections);
        return;
      }
      if (detectionsChanged) {
        // Sensor tick fires every real second. Instead of a full syncUnits pass
        // over all units, only re-sync units whose visibility or track-status
        // actually changed. At 100+ units this avoids rebuilding every entity.
        state.units.forEach((unit) => {
          const wasVisible = isVisible(unit, state.activeView, prev.detections);
          const nowVisible = isVisible(unit, state.activeView, state.detections);
          const wasTrack   = isTrack(unit, prev.activeView);
          const nowTrack   = isTrack(unit, state.activeView);
          if (wasVisible !== nowVisible || wasTrack !== nowTrack) {
            syncUnit(unit, state.activeView, state.selectedUnitId, state.detections);
          }
        });
        return;
      }
      if (munitionsChanged) {
        syncMunitions(state.munitions, state.activeView, state.munitionDetections);
        return;
      }
      if (munitionDetectChanged) {
        // Re-evaluate visibility of every in-flight munition.
        state.munitions.forEach((m) => {
          const entityId = `${MUNITION_ENTITY_PREFIX}${m.id}`;
          const e = viewer.entities.getById(entityId);
          if (e) e.show = isMunitionVisible(m, state.activeView, state.munitionDetections);
        });
        return;
      }
      if (selectionChanged) {
        // Re-sync old and new selected units so billboard scale and range
        // ring are both updated without a full syncUnits pass.
        [prev.selectedUnitId, state.selectedUnitId].forEach((id) => {
          if (!id) return;
          const unit = state.units.get(id);
          if (unit) syncUnit(unit, state.activeView, state.selectedUnitId, state.detections);
        });
        updateCursor(state.units, state.selectedUnitId, state.activeView);
      }
    });

    return () => {
      unsub();
      if (!viewer.isDestroyed()) viewer.destroy();
      viewerRef.current = null;
    };
  }, []);

  return (
    <div
      ref={containerRef}
      style={{ position: "absolute", inset: 0 }}
    />
  );
}
