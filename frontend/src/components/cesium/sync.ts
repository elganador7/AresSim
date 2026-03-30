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
import { syncExplosions, syncMunitions } from "./effectsLayer";
import { OIL_EDGE_PREFIX, OIL_NODE_PREFIX, syncOilGraph } from "./oilLayer";
import {
  type ActiveView,
  type DefInfo,
  type Detections,
  BLOCKED_ROUTE_COLOR,
  EXPLOSION_ENTITY_PREFIX,
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
      viewer.entities.removeById(unit.id);
      viewer.entities.removeById(routeId);
      viewer.entities.removeById(destId);
      viewer.entities.removeById(rangeId);
      viewer.entities.removeById(sensorId);
      viewer.entities.removeById(targetMarkerId);
      viewer.entities.removeById(assignedTargetMarkerId);
      Array.from(viewer.entities.values)
        .map((entity) => entity.id as string)
        .filter((id) => id.startsWith(waypointPrefix))
        .forEach((id) => viewer.entities.removeById(id));
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

      Array.from(viewer.entities.values)
        .map((entity) => entity.id as string)
        .filter((id) => id.startsWith(routeSegmentPrefix))
        .forEach((id) => viewer.entities.removeById(id));

      for (let idx = 0; idx < points.length - 1; idx += 1) {
        const start = points[idx];
        const end = points[idx + 1];
        const blocked = isSelected && selectedRoutePreview?.blocked && selectedRoutePreview.legIndex === idx + 1;
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
        renderedWaypoints.forEach((wp, idx) => {
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
        for (let idx = 0; idx < pathPoints.length - 1; idx += 1) {
          const start = pathPoints[idx];
          const end = pathPoints[idx + 1];
          const blocked = isSelected && selectedStrikePreview?.blocked && selectedStrikePreview.legIndex === idx + 1;
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
    Array.from(viewer.entities.values)
      .map((e) => e.id as string)
      .filter((id) =>
        !id.endsWith("_route") &&
        !id.endsWith("_dest") &&
        !id.endsWith("_range") &&
        !id.endsWith("_sensor") &&
        !id.endsWith("_target_marker") &&
        !id.endsWith("_assigned_target_marker") &&
        !id.includes("_wp_") &&
        !id.includes("_route_seg_") &&
        !id.includes("_strike_seg_") &&
        !id.startsWith(OIL_NODE_PREFIX) &&
        !id.startsWith(OIL_EDGE_PREFIX) &&
        !id.startsWith(MUNITION_ENTITY_PREFIX) &&
        !id.startsWith(EXPLOSION_ENTITY_PREFIX) &&
        !storeIds.has(id))
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
    syncMunitions(viewer, munitions, activeView, munitionDetections);
    syncExplosions(viewer, explosions);
    syncOilGraph(viewer, oilGraph, oilLayerVisible, selectedOilNodeId, selectedOilEdgeId);
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
      syncExplosions(viewer, state.explosions);
      return;
    }
    if (viewChanged) {
      syncUnits(state.units, state.activeView, state.selectedUnitId, state.detections);
      syncTrackLinks(state.units, state.selectedUnitId, state.activeView, state.detections, state.detectionContacts);
      syncMunitions(viewer, state.munitions, state.activeView, state.munitionDetections);
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
      syncMunitions(viewer, state.munitions, state.activeView, state.munitionDetections);
      return;
    }
    if (explosionsChanged) {
      syncExplosions(viewer, state.explosions);
      return;
    }
    if (oilGraphChanged || oilLayerChanged || oilSelectionChanged) {
      syncOilGraph(viewer, state.oilGraph, state.oilLayerVisible, state.selectedOilNodeId, state.selectedOilEdgeId);
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

  let oilCameraRerenderPending = false;
  const handleOilCameraChanged = () => {
    if (oilCameraRerenderPending) {
      return;
    }
    oilCameraRerenderPending = true;
    requestAnimationFrame(() => {
      oilCameraRerenderPending = false;
      const { oilGraph, oilLayerVisible, selectedOilNodeId, selectedOilEdgeId } = useSimStore.getState();
      syncOilGraph(viewer, oilGraph, oilLayerVisible, selectedOilNodeId, selectedOilEdgeId);
    });
  };
  viewer.camera.changed.addEventListener(handleOilCameraChanged);

  loadDefinitions();
  renderInitialState();

  return () => {
    viewer.camera.changed.removeEventListener(handleOilCameraChanged);
    unsubscribe();
  };
}
