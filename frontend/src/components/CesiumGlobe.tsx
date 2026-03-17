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
  Cartesian2,
  Color,
  VerticalOrigin,
  HorizontalOrigin,
  NearFarScalar,
  Entity,
  EllipseGraphics,
  ConstantProperty,
  ColorMaterialProperty,
  PolylineDashMaterialProperty,
  GeoJsonDataSource,
  ScreenSpaceEventType,
  Math as CesiumMath,
  Cartographic,
  LabelStyle,
  HeightReference,
} from "cesium";
import "cesium/Build/Cesium/Widgets/widgets.css";
import { useSimStore, Unit, Munition, ExplosionFx } from "../store/simStore";
import { AppendMoveWaypoint, ListUnitDefinitions, MoveUnit, UpdateMoveWaypoint } from "../../wailsjs/go/main/App";
import { THEATER_BORDERS_GEOJSON } from "../data/theaterBorders";
import { getUnitBillboardUrl } from "../utils/unitBillboard";
import {
  type ActiveView,
  type DefInfo,
  type Detections,
  type MunitionDetections,
  BLOCKED_ROUTE_COLOR,
  EXPLOSION_ENTITY_PREFIX,
  IMPACT_COLOR,
  KILL_COLOR,
  MUNITION_COLOR,
  MUNITION_ENTITY_PREFIX,
  ROUTE_COLOR,
  SENSOR_COLOR,
  STRIKE_PATH_COLOR,
  TRACK_LINK_PREFIX,
  canMove,
  ensureBridgeSuccess,
  getExplosionBillboard,
  isMunitionVisible,
  isTrack,
  isVisible,
  makeUnitEntity,
  maxWeaponRangeM,
  routeSegmentBlocked,
  strikeSegmentBlocked,
  teamForUnit,
  updateMapCursor,
} from "./cesium/helpers";

// ─── COMPONENT ────────────────────────────────────────────────────────────────

export default function CesiumGlobe() {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewerRef    = useRef<Viewer | null>(null);
  const borderDataSourceRef = useRef<GeoJsonDataSource | null>(null);
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
    viewerRef.current = viewer;

    GeoJsonDataSource.load(THEATER_BORDERS_GEOJSON as never, {
      stroke: Color.fromCssColorString("#9eb0c2"),
      fill: Color.fromCssColorString("#000000").withAlpha(0.01),
      strokeWidth: 1.2,
      clampToGround: true,
    })
      .then((dataSource) => {
        borderDataSourceRef.current = dataSource;
        viewer.dataSources.add(dataSource);
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
      })
      .catch(console.error);

    // Initial camera — Eastern Mediterranean.
    viewer.camera.flyTo({
      destination: Cartesian3.fromDegrees(25.8, 35.8, 1_200_000),
      duration: 0,
    });

    const pickLatLon = (position: Cartesian2): { lat: number; lon: number } | null => {
      const ray = viewer.camera.getPickRay(position);
      if (!ray) return null;
      const pos = viewer.scene.globe.pick(ray, viewer.scene);
      if (!pos) return null;
      const carto = Cartographic.fromCartesian(pos);
      return {
        lat: CesiumMath.toDegrees(carto.latitude),
        lon: CesiumMath.toDegrees(carto.longitude),
      };
    };

    const previewDraggedWaypoint = (unitID: string, waypointIndex: number, lat: number, lon: number) => {
      const waypointEntity = viewer.entities.getById(`${unitID}_wp_${waypointIndex}`);
      if (waypointEntity) {
        (waypointEntity.position as unknown as { setValue: (p: Cartesian3) => void })
          .setValue(Cartesian3.fromDegrees(lon, lat));
      }

      const unit = useSimStore.getState().units.get(unitID);
      if (!unit?.moveOrder) return;
      const positions: Cartesian3[] = [
        Cartesian3.fromDegrees(unit.position.lon, unit.position.lat),
        ...unit.moveOrder.waypoints.map((wp, idx) =>
          idx === waypointIndex
            ? Cartesian3.fromDegrees(lon, lat)
            : Cartesian3.fromDegrees(wp.lon, wp.lat),
        ),
      ];
      const routeEntity = viewer.entities.getById(`${unitID}_route`);
      if (routeEntity?.polyline) {
        (routeEntity.polyline.positions as unknown as { setValue: (p: Cartesian3[]) => void }).setValue(positions);
      }
      const destEntity = viewer.entities.getById(`${unitID}_dest`);
      if (destEntity && waypointIndex === unit.moveOrder.waypoints.length - 1) {
        (destEntity.position as unknown as { setValue: (p: Cartesian3) => void })
          .setValue(Cartesian3.fromDegrees(lon, lat));
      }
    };

    viewer.screenSpaceEventHandler.setInputAction(
      (evt: { position: Cartesian2 }) => {
        const { mapCommandMode } = useSimStore.getState();
        if (mapCommandMode.type !== "route" || !mapCommandMode.unitId) return;
        const picked = viewer.scene.pick(evt.position);
        if (!(picked?.id instanceof Entity)) return;
        const pickedEntity = picked.id as Entity;
        const waypointUnitId = pickedEntity.properties?.waypointUnitId?.getValue?.();
        const waypointIndex = pickedEntity.properties?.waypointIndex?.getValue?.();
        if (typeof waypointUnitId === "string" && typeof waypointIndex === "number" && waypointUnitId === mapCommandMode.unitId) {
          draggingWaypointRef.current = { unitId: waypointUnitId, waypointIndex };
          suppressClickRef.current = true;
          viewer.scene.screenSpaceCameraController.enableRotate = false;
        }
      },
      ScreenSpaceEventType.LEFT_DOWN,
    );

    viewer.screenSpaceEventHandler.setInputAction(
      (evt: { endPosition: Cartesian2 }) => {
        const drag = draggingWaypointRef.current;
        if (!drag) return;
        const next = pickLatLon(evt.endPosition);
        if (!next) return;
        previewDraggedWaypoint(drag.unitId, drag.waypointIndex, next.lat, next.lon);
      },
      ScreenSpaceEventType.MOUSE_MOVE,
    );

    viewer.screenSpaceEventHandler.setInputAction(
      (evt: { position: Cartesian2 }) => {
        const drag = draggingWaypointRef.current;
        if (!drag) return;
        const next = pickLatLon(evt.position);
        draggingWaypointRef.current = null;
        viewer.scene.screenSpaceCameraController.enableRotate = true;
        if (!next) return;
        previewDraggedWaypoint(drag.unitId, drag.waypointIndex, next.lat, next.lon);
        UpdateMoveWaypoint(drag.unitId, drag.waypointIndex, next.lat, next.lon)
          .then(ensureBridgeSuccess)
          .catch((error) => {
            console.error(error);
            alert(error instanceof Error ? error.message : String(error));
          });
      },
      ScreenSpaceEventType.LEFT_UP,
    );

    // ── Click handler ──────────────────────────────────────────────────────
    viewer.screenSpaceEventHandler.setInputAction(
      (evt: { position: Cartesian2 }) => {
        if (suppressClickRef.current) {
          suppressClickRef.current = false;
          return;
        }
        if (draggingWaypointRef.current) {
          return;
        }
        const {
          units,
          selectedUnitId,
          mapCommandMode,
          activeView,
          selectUnit,
          startRouteEdit,
          clearMapCommandMode,
        } =
          useSimStore.getState();

        // Check for entity click first.
        const picked = viewer.scene.pick(evt.position);
        if (picked?.id instanceof Entity) {
          const clickedId = (picked.id as Entity).id;
          if (units.has(clickedId)) {
            if (mapCommandMode.type === "target_pick" && mapCommandMode.unitId && selectedUnitId) {
              const shooter = units.get(mapCommandMode.unitId);
              const clickedUnit = units.get(clickedId);
              if (shooter && clickedUnit && shooter.id !== clickedUnit.id && shooter.side !== clickedUnit.side) {
                selectUnit(mapCommandMode.unitId);
                clearMapCommandMode();
                window.dispatchEvent(new CustomEvent("sim:target-picked", {
                  detail: {
                    shooterId: mapCommandMode.unitId,
                    targetUnitId: clickedId,
                  },
                }));
                return;
              }
            }
            const clickedUnit = units.get(clickedId);
            const nextSelectedId = selectedUnitId === clickedId ? null : clickedId;
            selectUnit(nextSelectedId);
            clearMapCommandMode();
            if (nextSelectedId && clickedUnit && canMove(clickedUnit, activeView, defInfoRef.current)) {
              startRouteEdit(nextSelectedId);
            }
            return;
          }
        }

        // Terrain click → move selected unit if the view permits.
        if (!selectedUnitId) return;
        const unit = units.get(selectedUnitId);
        if (!unit || !canMove(unit, activeView, defInfoRef.current)) return;

        const next = pickLatLon(evt.position);
        if (!next) return;
        const { lat, lon } = next;

        if (mapCommandMode.type === "route" && mapCommandMode.unitId === selectedUnitId) {
          AppendMoveWaypoint(selectedUnitId, lat, lon)
            .then(ensureBridgeSuccess)
            .catch((error) => {
              console.error(error);
              alert(error instanceof Error ? error.message : String(error));
            });
          return;
        }

        if (mapCommandMode.type === "target_pick" && mapCommandMode.unitId === selectedUnitId) {
          clearMapCommandMode();
          return;
        }

        MoveUnit(selectedUnitId, lat, lon)
          .then(ensureBridgeSuccess)
          .then(() => {
            selectUnit(null);
          })
          .catch((error) => {
            console.error(error);
            alert(error instanceof Error ? error.message : String(error));
          });
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
      const waypointPrefix = `${unit.id}_wp_`;
      const routeSegmentPrefix = `${unit.id}_route_seg_`;
      const strikeSegmentPrefix = `${unit.id}_strike_seg_`;

      if (!unit.status.isActive) {
        viewer.entities.removeById(unit.id);
        viewer.entities.removeById(routeId);
        viewer.entities.removeById(destId);
        viewer.entities.removeById(rangeId);
        viewer.entities.removeById(sensorId);
        Array.from(viewer.entities.values)
          .map((entity) => entity.id as string)
          .filter((id) => id.startsWith(routeSegmentPrefix))
          .forEach((id) => viewer.entities.removeById(id));
        Array.from(viewer.entities.values)
          .map((entity) => entity.id as string)
          .filter((id) => id.startsWith(strikeSegmentPrefix))
          .forEach((id) => viewer.entities.removeById(id));
        return;
      }

      const visible    = isVisible(unit, view, detections, defInfoRef.current);
      const track      = isTrack(unit, view, defInfoRef.current);
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
        const entity = makeUnitEntity(unit, def?.generalType ?? 0, def?.shortName ?? unit.displayName);
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
        const points = [
          { lat: unit.position.lat, lon: unit.position.lon },
          ...order.waypoints.map((wp) => ({ lat: wp.lat, lon: wp.lon })),
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
              width: 1,
              material: new PolylineDashMaterialProperty({ color: routeColor.withAlpha(0.12), dashLength: 16 }),
              clampToGround: false,
            },
          }));
        }

        Array.from(viewer.entities.values)
          .map((entity) => entity.id as string)
          .filter((id) => id.startsWith(routeSegmentPrefix))
          .forEach((id) => viewer.entities.removeById(id));

        for (let idx = 0; idx < points.length - 1; idx += 1) {
          const start = points[idx];
          const end = points[idx + 1];
          const blocked = routeSegmentBlocked(unit, start, end);
          viewer.entities.add(new Entity({
            id: `${routeSegmentPrefix}${idx}`,
            show: visible,
            polyline: {
              positions: new ConstantProperty([
                Cartesian3.fromDegrees(start.lon, start.lat),
                Cartesian3.fromDegrees(end.lon, end.lat),
              ]),
              width: blocked ? 2.5 : 2,
              material: blocked
                ? new PolylineDashMaterialProperty({ color: BLOCKED_ROUTE_COLOR.withAlpha(0.65), dashLength: 14 })
                : new PolylineDashMaterialProperty({ color: routeColor.withAlpha(0.75), dashLength: 16 }),
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

        Array.from(viewer.entities.values)
          .map((entity) => entity.id as string)
          .filter((id) => id.startsWith(waypointPrefix))
          .forEach((id) => viewer.entities.removeById(id));

        if (isSelected) {
          order.waypoints.forEach((wp, idx) => {
            viewer.entities.add(new Entity({
              id: `${waypointPrefix}${idx}`,
              show: visible,
              position: Cartesian3.fromDegrees(wp.lon, wp.lat),
              point: {
                pixelSize: 14,
                color: routeColor.withAlpha(0.95),
                outlineColor: Color.WHITE,
                outlineWidth: 2,
                disableDepthTestDistance: Number.POSITIVE_INFINITY,
              },
              label: {
                text: `${idx + 1}`,
                fillColor: Color.WHITE,
                outlineColor: Color.BLACK,
                outlineWidth: 2,
                style: LabelStyle.FILL_AND_OUTLINE,
                font: "12px sans-serif",
                pixelOffset: new Cartesian2(0, -18),
                disableDepthTestDistance: Number.POSITIVE_INFINITY,
              },
              properties: {
                waypointUnitId: unit.id,
                waypointIndex: idx,
              },
            }));
          });
        }
      } else {
        viewer.entities.removeById(routeId);
        viewer.entities.removeById(destId);
        Array.from(viewer.entities.values)
          .map((entity) => entity.id as string)
          .filter((id) => id.startsWith(routeSegmentPrefix))
          .forEach((id) => viewer.entities.removeById(id));
        Array.from(viewer.entities.values)
          .map((entity) => entity.id as string)
          .filter((id) => id.startsWith(waypointPrefix))
          .forEach((id) => viewer.entities.removeById(id));
      }

      Array.from(viewer.entities.values)
        .map((entity) => entity.id as string)
        .filter((id) => id.startsWith(strikeSegmentPrefix))
        .forEach((id) => viewer.entities.removeById(id));

      if (isSelected && visible && !track && unit.attackOrder?.targetUnitId) {
        const target = useSimStore.getState().units.get(unit.attackOrder.targetUnitId);
        if (target && isVisible(target, view, detections, defInfoRef.current)) {
          const pathPoints = [
            { lat: unit.position.lat, lon: unit.position.lon },
            ...(unit.moveOrder?.waypoints ?? []).map((wp) => ({ lat: wp.lat, lon: wp.lon })),
            { lat: target.position.lat, lon: target.position.lon },
          ];
          for (let idx = 0; idx < pathPoints.length - 1; idx += 1) {
            const start = pathPoints[idx];
            const end = pathPoints[idx + 1];
            const blocked = strikeSegmentBlocked(unit, start, end);
            viewer.entities.add(new Entity({
              id: `${strikeSegmentPrefix}${idx}`,
              show: true,
              polyline: {
                positions: new ConstantProperty([
                  Cartesian3.fromDegrees(start.lon, start.lat),
                  Cartesian3.fromDegrees(end.lon, end.lat),
                ]),
                width: blocked ? 3 : 2,
                material: blocked
                  ? new PolylineDashMaterialProperty({ color: BLOCKED_ROUTE_COLOR.withAlpha(0.78), dashLength: 10 })
                  : new PolylineDashMaterialProperty({ color: STRIKE_PATH_COLOR.withAlpha(0.45), dashLength: 12 }),
                clampToGround: false,
              },
            }));
          }
        }
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
          !id.includes("_route_seg_") &&
          !id.includes("_strike_seg_") &&
          !id.startsWith(MUNITION_ENTITY_PREFIX) && // managed by syncMunitions
          !id.startsWith(EXPLOSION_ENTITY_PREFIX) && // managed by syncExplosions
          !storeIds.has(id),
        )
        .forEach((id) => viewer.entities.removeById(id));
    };

    const syncTrackLinks = (
      units: Map<string, Unit>,
      selectedId: string | null,
      view: ActiveView,
      detections: Detections,
      detectionContacts: Map<string, Map<string, { unitId: string; sourceTeam: string; shared: boolean }>>,
    ) => {
      Array.from(viewer.entities.values)
        .map((entity) => entity.id as string)
        .filter((id) => id.startsWith(TRACK_LINK_PREFIX))
        .forEach((id) => viewer.entities.removeById(id));

      if (!selectedId || view === "debug") {
        return;
      }
      const selectedUnit = units.get(selectedId);
      if (!selectedUnit || teamForUnit(selectedUnit, defInfoRef.current) !== view) {
        return;
      }
      const visibleTracks = detections.get(view);
      if (!visibleTracks || visibleTracks.size === 0) {
        return;
      }
      const contactMeta = detectionContacts.get(view) ?? new Map();
      for (const targetId of visibleTracks) {
        const target = units.get(targetId);
        if (!target) {
          continue;
        }
        const meta = contactMeta.get(targetId);
        const shared = !!meta?.shared;
        viewer.entities.add(new Entity({
          id: `${TRACK_LINK_PREFIX}${selectedId}_${targetId}`,
          polyline: {
            positions: new ConstantProperty([
              Cartesian3.fromDegrees(selectedUnit.position.lon, selectedUnit.position.lat, selectedUnit.position.altMsl),
              Cartesian3.fromDegrees(target.position.lon, target.position.lat, target.position.altMsl),
            ]),
            width: shared ? 1 : 1.25,
            material: shared
              ? new PolylineDashMaterialProperty({
                  color: Color.fromCssColorString("#cbd5e1").withAlpha(0.12),
                  dashLength: 10,
                })
              : new ColorMaterialProperty(Color.fromCssColorString("#e2e8f0").withAlpha(0.16)),
            clampToGround: false,
          },
        }));
      }
    };

    // ── Sync a single in-flight munition entity ────────────────────────────
    const syncMunition = (
      munition: Munition,
      view: ActiveView,
      munitionDetections: MunitionDetections,
    ) => {
      const entityId = `${MUNITION_ENTITY_PREFIX}${munition.id}`;
      const visible = isMunitionVisible(munition, view, munitionDetections);
      const pos = Cartesian3.fromDegrees(munition.lon, munition.lat, munition.altMsl);

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

    const syncExplosion = (explosion: ExplosionFx) => {
      const entityId = `${EXPLOSION_ENTITY_PREFIX}${explosion.id}`;
      const pos = Cartesian3.fromDegrees(explosion.lon, explosion.lat, explosion.altMsl);
      const pixelSize = explosion.kind === "kill" ? 16 : 10;
      const color = explosion.kind === "kill" ? KILL_COLOR : IMPACT_COLOR;
      const existing = viewer.entities.getById(entityId);
      if (existing) {
        (existing.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(pos);
        existing.show = true;
        if (existing.billboard) {
          existing.billboard.image = new ConstantProperty(getExplosionBillboard(explosion.kind));
        }
        if (existing.point) {
          existing.point.pixelSize = new ConstantProperty(pixelSize);
          existing.point.color = new ConstantProperty(color.withAlpha(0.95));
        }
      } else {
        viewer.entities.add(new Entity({
          id: entityId,
          show: true,
          position: pos,
          billboard: {
            image: getExplosionBillboard(explosion.kind),
            width: explosion.kind === "kill" ? 58 : 42,
            height: explosion.kind === "kill" ? 58 : 42,
            verticalOrigin: VerticalOrigin.CENTER,
            horizontalOrigin: HorizontalOrigin.CENTER,
            disableDepthTestDistance: Number.POSITIVE_INFINITY,
            scaleByDistance: new NearFarScalar(2e5, 1.0, 8e6, 0.5),
          },
          point: {
            pixelSize,
            color: color.withAlpha(0.95),
            outlineColor: Color.WHITE.withAlpha(0.85),
            outlineWidth: 1,
            disableDepthTestDistance: Number.POSITIVE_INFINITY,
          },
        }));
      }
    };

    const syncExplosions = (explosions: Map<string, ExplosionFx>) => {
      explosions.forEach((explosion) => syncExplosion(explosion));
      const liveIds = new Set(
        Array.from(explosions.keys()).map((id) => `${EXPLOSION_ENTITY_PREFIX}${id}`),
      );
      Array.from(viewer.entities.values)
        .map((e) => e.id as string)
        .filter((id) => id.startsWith(EXPLOSION_ENTITY_PREFIX) && !liveIds.has(id))
        .forEach((id) => viewer.entities.removeById(id));
    };

    // ── Reapply fog-of-war when view or detections change ──────────────────
    const applyView = (units: Map<string, Unit>, view: ActiveView, detections: Detections) => {
      units.forEach((unit) => {
        const visible = isVisible(unit, view, detections, defInfoRef.current);
        const e = viewer.entities.getById(unit.id);
        if (e) e.show = visible;
        const r = viewer.entities.getById(`${unit.id}_route`);
        if (r) r.show = visible;
        Array.from(viewer.entities.values)
          .filter((entity) => String(entity.id).startsWith(`${unit.id}_route_seg_`))
          .forEach((entity) => { entity.show = visible; });
        const d = viewer.entities.getById(`${unit.id}_dest`);
        if (d) d.show = visible;
        Array.from(viewer.entities.values)
          .filter((entity) => String(entity.id).startsWith(`${unit.id}_wp_`))
          .forEach((entity) => { entity.show = visible; });
        const rng = viewer.entities.getById(`${unit.id}_range`);
        if (rng) rng.show = visible;
        const sen = viewer.entities.getById(`${unit.id}_sensor`);
        if (sen) sen.show = visible;
      });
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
            teamCode: Array.isArray(r["employed_by"]) && r["employed_by"].length > 0
              ? String(r["employed_by"][0]).trim().toUpperCase()
              : String(r["nation_of_origin"] ?? "").trim().toUpperCase(),
          };
        });
        defInfoRef.current = map;
        const { units, activeView, selectedUnitId, detections } = useSimStore.getState();
        syncUnits(units, activeView, selectedUnitId, detections);
      })
      .catch(console.error);

    // Initial render with whatever is already in the store.
    const { units: initUnits, activeView: initView, selectedUnitId: initSel,
            detections: initDet, detectionContacts: initContactMeta, munitions: initMun, explosions: initExplosions,
            munitionDetections: initMunDet } = useSimStore.getState();
    syncUnits(initUnits, initView, initSel, initDet);
    syncTrackLinks(initUnits, initSel, initView, initDet, initContactMeta);
    syncMunitions(initMun, initView, initMunDet);
    syncExplosions(initExplosions);

    // ── Store subscriptions (imperatively driven, no React re-renders) ─────
    const unsub = useSimStore.subscribe((state, prev) => {
      const unitsChanged           = state.units              !== prev.units;
      const viewChanged            = state.activeView         !== prev.activeView;
      const selectionChanged       = state.selectedUnitId     !== prev.selectedUnitId;
      const detectionsChanged      = state.detections         !== prev.detections;
      const detectionMetaChanged   = state.detectionContacts  !== prev.detectionContacts;
      const relationshipsChanged   = state.relationships      !== prev.relationships;
      const munitionsChanged       = state.munitions          !== prev.munitions;
      const explosionsChanged      = state.explosions         !== prev.explosions;
      const munitionDetectChanged  = state.munitionDetections !== prev.munitionDetections;

      if (unitsChanged) {
        syncUnits(state.units, state.activeView, state.selectedUnitId, state.detections);
        syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
        syncExplosions(state.explosions);
        return; // syncUnits covers everything
      }
      if (viewChanged) {
        // View change affects all units and munitions — full passes required.
        syncUnits(state.units, state.activeView, state.selectedUnitId, state.detections);
        syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
        syncMunitions(state.munitions, state.activeView, state.munitionDetections);
        return;
      }
      if (relationshipsChanged) {
        syncUnits(state.units, state.activeView, state.selectedUnitId, state.detections);
        syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
        return;
      }
      if (detectionsChanged || detectionMetaChanged) {
        // Sensor tick fires every real second. Instead of a full syncUnits pass
        // over all units, only re-sync units whose visibility or track-status
        // actually changed. At 100+ units this avoids rebuilding every entity.
        state.units.forEach((unit) => {
          const wasVisible = isVisible(unit, state.activeView, prev.detections, defInfoRef.current);
          const nowVisible = isVisible(unit, state.activeView, state.detections, defInfoRef.current);
          const wasTrack   = isTrack(unit, prev.activeView, defInfoRef.current);
          const nowTrack   = isTrack(unit, state.activeView, defInfoRef.current);
          if (wasVisible !== nowVisible || wasTrack !== nowTrack) {
            syncUnit(unit, state.activeView, state.selectedUnitId, state.detections);
          }
        });
        syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
        return;
      }
      if (munitionsChanged) {
        syncMunitions(state.munitions, state.activeView, state.munitionDetections);
        return;
      }
      if (explosionsChanged) {
        syncExplosions(state.explosions);
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
        syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
        updateMapCursor(containerRef.current, state.mapCommandMode, state.units, state.selectedUnitId, state.activeView, defInfoRef.current);
      }
    });

    return () => {
      unsub();
      if (borderDataSourceRef.current) {
        viewer.dataSources.remove(borderDataSourceRef.current, true);
        borderDataSourceRef.current = null;
      }
      if (!viewer.isDestroyed()) viewer.destroy();
      viewerRef.current = null;
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
