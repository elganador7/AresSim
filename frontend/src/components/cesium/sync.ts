import type { MutableRefObject } from "react";
import {
  Cartesian2,
  Cartesian3,
  Color,
  ColorMaterialProperty,
  ConstantProperty,
  EllipseGraphics,
  Entity,
  HeightReference,
  HorizontalOrigin,
  LabelStyle,
  Math as CesiumMath,
  NearFarScalar,
  DistanceDisplayCondition,
  PolylineDashMaterialProperty,
  Viewer,
} from "cesium";
import { ListUnitDefinitions } from "../../../wailsjs/go/main/App";
import type { Unit } from "../../store/simStore";
import { useSimStore } from "../../store/simStore";
import { reportError } from "../../utils/errors";
import { getUnitBillboardUrl } from "../../utils/unitBillboard";
import {
  createEffectsLayerSyncState,
  syncExplosions,
  syncMunitions,
} from "./effectsLayer";
import {
  oilCameraBucketKey,
  createOilLayerSyncState,
  syncOilGraph,
} from "./oilLayer";
import {
  type ActiveView,
  type DefInfo,
  type Detections,
  BLOCKED_ROUTE_COLOR,
  MUNITION_ENTITY_PREFIX,
  SENSOR_COLOR,
  STRIKE_PATH_COLOR,
  TRACK_LINK_PREFIX,
  isMunitionVisible,
  isTrack,
  isVisible,
  applyUnitEntityVisual,
  makeUnitEntity,
  maxWeaponRangeM,
  relationshipColorHex,
  routeColorForUnit,
  teamForUnit,
  updateMapCursor,
  definitionInfoFor,
  normalizeDefinitionId,
} from "./helpers";

interface SetupCesiumStoreSyncOptions {
  viewer: Viewer;
  containerRef: MutableRefObject<HTMLDivElement | null>;
  defInfoRef: MutableRefObject<Record<string, DefInfo>>;
}

export function setupCesiumStoreSync({
  viewer,
  containerRef,
  defInfoRef,
}: SetupCesiumStoreSyncOptions) {
  const oilLayerSyncState = createOilLayerSyncState();
  const effectsLayerSyncState = createEffectsLayerSyncState();
  const trackedUnitEntityIds = new Set<string>();
  const trackedTrackLinkIds = new Set<string>();
  const trackedWaypointIds = new Map<string, Set<string>>();
  const trackedRouteSegmentIds = new Map<string, Set<string>>();
  const trackedStrikeSegmentIds = new Map<string, Set<string>>();
  const removeTrackedIds = (ids: Set<string>) => {
    ids.forEach((id) => viewer.entities.removeById(id));
    ids.clear();
  };
  const replaceTrackedIds = (map: Map<string, Set<string>>, key: string, nextIds: string[]) => {
    const currentIds = map.get(key);
    if (currentIds) {
      removeTrackedIds(currentIds);
    }
    if (nextIds.length === 0) {
      map.delete(key);
      return;
    }
    map.set(key, new Set<string>(nextIds));
  };
  const clearTrackedForUnit = (unitId: string) => {
    const waypointIds = trackedWaypointIds.get(unitId);
    if (waypointIds) {
      removeTrackedIds(waypointIds);
      trackedWaypointIds.delete(unitId);
    }
    const routeSegmentIds = trackedRouteSegmentIds.get(unitId);
    if (routeSegmentIds) {
      removeTrackedIds(routeSegmentIds);
      trackedRouteSegmentIds.delete(unitId);
    }
    const strikeSegmentIds = trackedStrikeSegmentIds.get(unitId);
    if (strikeSegmentIds) {
      removeTrackedIds(strikeSegmentIds);
      trackedStrikeSegmentIds.delete(unitId);
    }
  };
  const syncOilGraphNow = (
    oilGraph: ReturnType<typeof useSimStore.getState>["oilGraph"],
    oilLayerVisible: boolean,
    selectedOilNodeId: string | null,
    selectedOilEdgeId: string | null,
  ) => {
    const rect = viewer.camera.computeViewRectangle(viewer.scene.globe.ellipsoid);
    const viewRect = rect ? {
      west: CesiumMath.toDegrees(rect.west),
      south: CesiumMath.toDegrees(rect.south),
      east: CesiumMath.toDegrees(rect.east),
      north: CesiumMath.toDegrees(rect.north),
    } : null;
    const cameraHeight = viewer.camera.positionCartographic?.height ?? 3_000_000;
    oilLayerSyncState.lastCameraBucketKey = oilCameraBucketKey(cameraHeight, viewRect);
    syncOilGraph(viewer, oilGraph, oilLayerVisible, selectedOilNodeId, selectedOilEdgeId, oilLayerSyncState);
  };
  const activeViewContact = (view: ActiveView, unitId: string) => {
    if (view === "debug") {
      return undefined;
    }
    return useSimStore.getState().detectionContacts.get(view)?.get(unitId);
  };

  const syncUnit = (
    unit: Unit,
    view: ActiveView,
    selectedId: string | null,
    detections: Detections,
  ) => {
    const routeId = `${unit.id}_route`;
    const destId = `${unit.id}_dest`;
    const rangeId = `${unit.id}_range`;
    const sensorId = `${unit.id}_sensor`;
    const waypointPrefix = `${unit.id}_wp_`;
    const routeSegmentPrefix = `${unit.id}_route_seg_`;
    const strikeSegmentPrefix = `${unit.id}_strike_seg_`;
    const targetMarkerId = `${unit.id}_target_marker`;
    const assignedTargetMarkerId = `${unit.id}_assigned_target_marker`;

    if (!unit.status.isActive) {
      trackedUnitEntityIds.delete(unit.id);
      viewer.entities.removeById(unit.id);
      viewer.entities.removeById(routeId);
      viewer.entities.removeById(destId);
      viewer.entities.removeById(rangeId);
      viewer.entities.removeById(sensorId);
      viewer.entities.removeById(targetMarkerId);
      viewer.entities.removeById(assignedTargetMarkerId);
      clearTrackedForUnit(unit.id);
      return;
    }
    trackedUnitEntityIds.add(unit.id);

    const visible = isVisible(unit, view, detections, defInfoRef.current);
    const track = isTrack(unit, view, defInfoRef.current);
    const trackAlpha = track ? 0.55 : 1.0;
    const pos = Cartesian3.fromDegrees(
      unit.position.lon, unit.position.lat, unit.position.altMsl,
    );
    const isSelected = unit.id === selectedId;

    const existing = viewer.entities.getById(unit.id);
    if (existing) {
      (existing.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(pos);
      existing.show = visible;
      const def = definitionInfoFor(defInfoRef.current, unit.definitionId);
      applyUnitEntityVisual(
        existing,
        unit,
        def?.generalType ?? 0,
        def?.shortName ?? unit.displayName,
        def?.domain,
        def?.stationary,
        def?.assetClass,
        def?.visualModelId,
        isSelected,
        trackAlpha,
      );
    } else {
      const def = definitionInfoFor(defInfoRef.current, unit.definitionId);
      const entity = makeUnitEntity(
        unit,
        def?.generalType ?? 0,
        def?.shortName ?? unit.displayName,
        def?.domain,
        def?.stationary,
        def?.assetClass,
        def?.visualModelId,
      );
      entity.show = visible;
      viewer.entities.add(entity);
      applyUnitEntityVisual(
        entity,
        unit,
        def?.generalType ?? 0,
        def?.shortName ?? unit.displayName,
        def?.domain,
        def?.stationary,
        def?.assetClass,
        def?.visualModelId,
        isSelected,
        trackAlpha,
      );
    }

    const order = unit.moveOrder;
    const { humanControlledTeam, units, selectedRoutePreview } = useSimStore.getState();
    const renderedWaypoints = order?.waypoints ?? [];
    if (!track && renderedWaypoints.length > 0) {
      const routeColor = routeColorForUnit(unit, humanControlledTeam, units);
      const positions: Cartesian3[] = [
        Cartesian3.fromDegrees(unit.position.lon, unit.position.lat),
        ...renderedWaypoints.map((wp) => Cartesian3.fromDegrees(wp.lon, wp.lat)),
      ];
      const points = [
        { lat: unit.position.lat, lon: unit.position.lon },
        ...renderedWaypoints.map((wp) => ({ lat: wp.lat, lon: wp.lon })),
      ];
      const last = renderedWaypoints[renderedWaypoints.length - 1];
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

      const routeSegmentIds: string[] = [];
      for (let idx = 0; idx < points.length - 1; idx += 1) {
        const start = points[idx];
        const end = points[idx + 1];
        const blocked = isSelected && selectedRoutePreview?.blocked && selectedRoutePreview.legIndex === idx + 1;
        const segmentId = `${routeSegmentPrefix}${idx}`;
        routeSegmentIds.push(segmentId);
        viewer.entities.add(new Entity({
          id: segmentId,
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
      replaceTrackedIds(trackedRouteSegmentIds, unit.id, routeSegmentIds);

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

      const waypointIds: string[] = [];
      if (isSelected) {
        renderedWaypoints.forEach((wp, idx) => {
          const waypointId = `${waypointPrefix}${idx}`;
          waypointIds.push(waypointId);
          viewer.entities.add(new Entity({
            id: waypointId,
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
      replaceTrackedIds(trackedWaypointIds, unit.id, waypointIds);
    } else {
      viewer.entities.removeById(routeId);
      viewer.entities.removeById(destId);
      replaceTrackedIds(trackedRouteSegmentIds, unit.id, []);
      replaceTrackedIds(trackedWaypointIds, unit.id, []);
    }

    replaceTrackedIds(trackedStrikeSegmentIds, unit.id, []);
    viewer.entities.removeById(targetMarkerId);
    viewer.entities.removeById(assignedTargetMarkerId);

    if (isSelected && visible && !track && unit.attackOrder?.targetUnitId) {
      const selectedStrikePreview = useSimStore.getState().selectedStrikePreview;
      const target = useSimStore.getState().units.get(unit.attackOrder.targetUnitId);
      const visibleTarget = target && isVisible(target, view, detections, defInfoRef.current) ? target : null;
      const sharedContact = activeViewContact(view, unit.attackOrder.targetUnitId);
      const targetLabel = sharedContact?.shared
        ? `Assigned Target · shared by ${sharedContact.sourceTeam}`
        : (visibleTarget ? "Assigned Target" : "Assigned Target Area");
      if (visibleTarget) {
        const routedStrikePoints = selectedStrikePreview?.routePoints && selectedStrikePreview.routePoints.length > 0
          ? selectedStrikePreview.routePoints
          : [
            ...(unit.moveOrder?.waypoints ?? []).map((wp) => ({ lat: wp.lat, lon: wp.lon })),
            { lat: visibleTarget.position.lat, lon: visibleTarget.position.lon },
          ];
        const pathPoints = [
          { lat: unit.position.lat, lon: unit.position.lon },
          ...routedStrikePoints,
        ];
        const strikeSegmentIds: string[] = [];
        for (let idx = 0; idx < pathPoints.length - 1; idx += 1) {
          const start = pathPoints[idx];
          const end = pathPoints[idx + 1];
          const blocked = isSelected && selectedStrikePreview?.blocked && selectedStrikePreview.legIndex === idx + 1;
          const segmentId = `${strikeSegmentPrefix}${idx}`;
          strikeSegmentIds.push(segmentId);
          viewer.entities.add(new Entity({
            id: segmentId,
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
        replaceTrackedIds(trackedStrikeSegmentIds, unit.id, strikeSegmentIds);
        viewer.entities.add(new Entity({
          id: targetMarkerId,
          show: true,
          position: Cartesian3.fromDegrees(visibleTarget.position.lon, visibleTarget.position.lat, visibleTarget.position.altMsl),
          point: {
            pixelSize: 12,
            color: STRIKE_PATH_COLOR.withAlpha(0.95),
            outlineColor: Color.WHITE,
            outlineWidth: 2,
            disableDepthTestDistance: Number.POSITIVE_INFINITY,
          },
          label: {
            text: targetLabel,
            fillColor: Color.WHITE,
            outlineColor: Color.BLACK,
            outlineWidth: 2,
            style: LabelStyle.FILL_AND_OUTLINE,
            font: "12px sans-serif",
            pixelOffset: new Cartesian2(0, -18),
            disableDepthTestDistance: Number.POSITIVE_INFINITY,
          },
        }));
      }
    }

    viewer.entities.removeById(rangeId);
    if (isSelected && visible) {
      const { humanControlledTeam, units } = useSimStore.getState();
      const ringColor = routeColorForUnit(unit, humanControlledTeam, units);
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

    const sensorRangeM = definitionInfoFor(defInfoRef.current, unit.definitionId)?.detectionRangeM ?? 0;
    if (!visible || !isSelected || sensorRangeM <= 0) {
      viewer.entities.removeById(sensorId);
    } else {
      const sensorEntity = viewer.entities.getById(sensorId);
      const outlineAlpha = 0.75;
      const fillAlpha = 0.1;
      if (sensorEntity) {
        (sensorEntity.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(pos);
        sensorEntity.show = true;
        if (sensorEntity.ellipse) {
          sensorEntity.ellipse.semiMajorAxis = new ConstantProperty(sensorRangeM);
          sensorEntity.ellipse.semiMinorAxis = new ConstantProperty(sensorRangeM);
          sensorEntity.ellipse.material = new ColorMaterialProperty(SENSOR_COLOR.withAlpha(fillAlpha));
          sensorEntity.ellipse.outlineColor = new ConstantProperty(SENSOR_COLOR.withAlpha(outlineAlpha));
          sensorEntity.ellipse.outlineWidth = new ConstantProperty(2);
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
        }));
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

    const storeIds = new Set(units.keys());
    Array.from(trackedUnitEntityIds).forEach((id) => {
      if (!storeIds.has(id)) {
        viewer.entities.removeById(id);
        viewer.entities.removeById(`${id}_route`);
        viewer.entities.removeById(`${id}_dest`);
        viewer.entities.removeById(`${id}_range`);
        viewer.entities.removeById(`${id}_sensor`);
        viewer.entities.removeById(`${id}_target_marker`);
        viewer.entities.removeById(`${id}_assigned_target_marker`);
        clearTrackedForUnit(id);
        trackedUnitEntityIds.delete(id);
      }
    });
  };

  const syncTrackLinks = (
    units: Map<string, Unit>,
    selectedId: string | null,
    view: ActiveView,
    detections: Detections,
    detectionContacts: Map<string, Map<string, { unitId: string; sourceTeam: string; shared: boolean }>>,
  ) => {
    removeTrackedIds(trackedTrackLinkIds);

    if (!selectedId || view === "debug") return;
    const selectedUnit = units.get(selectedId);
    if (!selectedUnit || teamForUnit(selectedUnit, defInfoRef.current) !== view) return;
    const visibleTracks = detections.get(view);
    if (!visibleTracks || visibleTracks.size === 0) return;
    const contactMeta = detectionContacts.get(view) ?? new Map();
    for (const targetId of visibleTracks) {
      const target = units.get(targetId);
      if (!target) continue;
      const meta = contactMeta.get(targetId);
      const shared = !!meta?.shared;
      const trackLinkId = `${TRACK_LINK_PREFIX}${selectedId}_${targetId}`;
      trackedTrackLinkIds.add(trackLinkId);
      viewer.entities.add(new Entity({
        id: trackLinkId,
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

  const loadDefinitions = () =>
    ListUnitDefinitions()
      .then((rows) => {
        const map: Record<string, DefInfo> = {};
        const numeric = (value: unknown) => {
          const parsed = Number(value);
          return Number.isFinite(parsed) ? parsed : 0;
        };
        rows.forEach((r) => {
          const id = normalizeDefinitionId(String(r["id"] ?? ""));
          if (!id) {
            return;
          }
          const shortName = String(r["short_name"] ?? "").trim()
            || String(r["specific_type"] ?? "").trim()
            || String(r["name"] ?? "").trim();
          map[id] = {
            generalType: numeric(r["general_type"]),
            domain: numeric(r["domain"]),
            detectionRangeM: numeric(r["detection_range_m"]),
            shortName,
            teamCode: Array.isArray(r["employed_by"]) && r["employed_by"].length > 0
              ? String(r["employed_by"][0]).trim().toUpperCase()
              : String(r["nation_of_origin"] ?? "").trim().toUpperCase(),
            stationary: Boolean(r["stationary"]),
            assetClass: String(r["asset_class"] ?? "").trim(),
            visualModelId: String(r["visual_model_id"] ?? "").trim(),
          };
        });
        defInfoRef.current = map;
        const { units, activeView, selectedUnitId, detections } = useSimStore.getState();
        syncUnits(units, activeView, selectedUnitId, detections);
      })
      .catch((error) => reportError("CesiumSync:ListUnitDefinitions", error));

  const renderInitialState = () => {
    const {
      units,
      activeView,
      selectedUnitId,
      detections,
      detectionContacts,
      munitions,
      explosions,
      munitionDetections,
      oilGraph,
      oilLayerVisible,
      selectedOilNodeId,
      selectedOilEdgeId,
    } = useSimStore.getState();
    syncUnits(units, activeView, selectedUnitId, detections);
    syncTrackLinks(units, selectedUnitId, activeView, detections, detectionContacts);
    syncMunitions(viewer, munitions, activeView, munitionDetections, effectsLayerSyncState);
    syncExplosions(viewer, explosions, effectsLayerSyncState);
    syncOilGraphNow(oilGraph, oilLayerVisible, selectedOilNodeId, selectedOilEdgeId);
  };

  const unsubscribe = useSimStore.subscribe((state, prev) => {
    const unitsChanged = state.units !== prev.units;
    const scenarioChanged = state.scenarioName !== prev.scenarioName;
    const viewChanged = state.activeView !== prev.activeView;
    const selectionChanged = state.selectedUnitId !== prev.selectedUnitId;
    const detectionsChanged = state.detections !== prev.detections;
    const detectionMetaChanged = state.detectionContacts !== prev.detectionContacts;
    const relationshipsChanged = state.relationships !== prev.relationships;
    const routePreviewChanged = state.selectedRoutePreview !== prev.selectedRoutePreview;
    const strikePreviewChanged = state.selectedStrikePreview !== prev.selectedStrikePreview;
    const munitionsChanged = state.munitions !== prev.munitions;
    const explosionsChanged = state.explosions !== prev.explosions;
    const munitionDetectChanged = state.munitionDetections !== prev.munitionDetections;
    const oilGraphChanged = state.oilGraph !== prev.oilGraph;
    const oilLayerChanged = state.oilLayerVisible !== prev.oilLayerVisible;
    const oilSelectionChanged = state.selectedOilNodeId !== prev.selectedOilNodeId
      || state.selectedOilEdgeId !== prev.selectedOilEdgeId;

    if (scenarioChanged) {
      loadDefinitions();
      syncUnits(state.units, state.activeView, state.selectedUnitId, state.detections);
      syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
      return;
    }
    if (unitsChanged) {
      syncUnits(state.units, state.activeView, state.selectedUnitId, state.detections);
      syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
      syncExplosions(viewer, state.explosions, effectsLayerSyncState);
      return;
    }
    if (viewChanged) {
      syncUnits(state.units, state.activeView, state.selectedUnitId, state.detections);
      syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
      syncMunitions(viewer, state.munitions, state.activeView, state.munitionDetections, effectsLayerSyncState);
      return;
    }
    if (relationshipsChanged) {
      syncUnits(state.units, state.activeView, state.selectedUnitId, state.detections);
      syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
      return;
    }
    if (routePreviewChanged || strikePreviewChanged) {
      if (state.selectedUnitId) {
        const unit = state.units.get(state.selectedUnitId);
        if (unit) {
          syncUnit(unit, state.activeView, state.selectedUnitId, state.detections);
        }
      }
      return;
    }
    if (detectionsChanged || detectionMetaChanged) {
      state.units.forEach((unit) => {
        const wasVisible = isVisible(unit, state.activeView, prev.detections, defInfoRef.current);
        const nowVisible = isVisible(unit, state.activeView, state.detections, defInfoRef.current);
        const wasTrack = isTrack(unit, prev.activeView, defInfoRef.current);
        const nowTrack = isTrack(unit, state.activeView, defInfoRef.current);
        if (wasVisible !== nowVisible || wasTrack !== nowTrack) {
          syncUnit(unit, state.activeView, state.selectedUnitId, state.detections);
        }
      });
      syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
      return;
    }
    if (munitionsChanged) {
      syncMunitions(viewer, state.munitions, state.activeView, state.munitionDetections, effectsLayerSyncState);
      return;
    }
    if (explosionsChanged) {
      syncExplosions(viewer, state.explosions, effectsLayerSyncState);
      return;
    }
    if (oilGraphChanged || oilLayerChanged || oilSelectionChanged) {
      syncOilGraphNow(
        state.oilGraph,
        state.oilLayerVisible,
        state.selectedOilNodeId,
        state.selectedOilEdgeId,
      );
      return;
    }
    if (munitionDetectChanged) {
      state.munitions.forEach((m) => {
        const entityId = `${MUNITION_ENTITY_PREFIX}${m.id}`;
        const e = viewer.entities.getById(entityId);
        if (e) e.show = isMunitionVisible(m, state.activeView, state.munitionDetections);
      });
      return;
    }
    if (selectionChanged) {
      [prev.selectedUnitId, state.selectedUnitId].forEach((id) => {
        if (!id) return;
        const unit = state.units.get(id);
        if (unit) syncUnit(unit, state.activeView, state.selectedUnitId, state.detections);
      });
      syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
      updateMapCursor(
        containerRef.current,
        state.mapCommandMode,
        state.units,
        state.selectedUnitId,
        state.activeView,
        defInfoRef.current,
      );
    }
  });

  const handleOilCameraMoveEnd = () => {
    const rect = viewer.camera.computeViewRectangle(viewer.scene.globe.ellipsoid);
    const viewRect = rect ? {
      west: CesiumMath.toDegrees(rect.west),
      south: CesiumMath.toDegrees(rect.south),
      east: CesiumMath.toDegrees(rect.east),
      north: CesiumMath.toDegrees(rect.north),
    } : null;
    const cameraHeight = viewer.camera.positionCartographic?.height ?? 3_000_000;
    const nextBucketKey = oilCameraBucketKey(cameraHeight, viewRect);
    if (oilLayerSyncState.lastCameraBucketKey === nextBucketKey) {
      return;
    }
    oilLayerSyncState.lastCameraBucketKey = nextBucketKey;
    const { oilGraph, oilLayerVisible, selectedOilNodeId, selectedOilEdgeId } = useSimStore.getState();
    syncOilGraphNow(
      oilGraph,
      oilLayerVisible,
      selectedOilNodeId,
      selectedOilEdgeId,
    );
  };
  viewer.camera.moveEnd.addEventListener(handleOilCameraMoveEnd);

  loadDefinitions();
  renderInitialState();

  return () => {
    viewer.camera.moveEnd.removeEventListener(handleOilCameraMoveEnd);
    unsubscribe();
  };
}
