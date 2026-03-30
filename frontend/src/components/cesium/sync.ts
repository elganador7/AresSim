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
  VerticalOrigin,
  Viewer,
} from "cesium";
import { ListUnitDefinitions } from "../../../wailsjs/go/main/App";
import type { ExplosionFx, Munition, OilEdge, OilGraph, OilNode, Unit } from "../../store/simStore";
import { useSimStore } from "../../store/simStore";
import { getUnitBillboardUrl } from "../../utils/unitBillboard";
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
  SENSOR_COLOR,
  STRIKE_PATH_COLOR,
  TRACK_LINK_PREFIX,
  getExplosionBillboard,
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

const OIL_NODE_PREFIX = "oil_node_";
const OIL_EDGE_PREFIX = "oil_edge_";
const OIL_OUTLINE_PREFIX = "oil_outline_";

function oilNodeColor(kind: string): Color {
  switch (kind) {
    case "project":
      return Color.fromCssColorString("#f59e0b");
    case "extraction_site":
      return Color.fromCssColorString("#f97316");
    case "refinery":
      return Color.fromCssColorString("#38bdf8");
    case "export_terminal":
    case "import_terminal":
      return Color.fromCssColorString("#facc15");
    case "storage_hub":
      return Color.fromCssColorString("#a78bfa");
    case "chokepoint":
      return Color.fromCssColorString("#ef4444");
    case "demand_center":
      return Color.fromCssColorString("#34d399");
    default:
      return Color.fromCssColorString("#94a3b8");
  }
}

function oilEdgeColor(edge: OilEdge): Color {
  if (edge.kind === "pipeline") {
    switch (edge.commodity) {
      case "crude":
        return Color.fromCssColorString("#fb923c");
      case "ngl":
        return Color.fromCssColorString("#8b5cf6");
      case "lpg":
        return Color.fromCssColorString("#60a5fa");
      case "naphtha":
        return Color.fromCssColorString("#f472b6");
      case "refined_products":
        return Color.fromCssColorString("#22c55e");
      default:
        return Color.fromCssColorString("#60a5fa");
    }
  }
  if (edge.kind === "shipping_lane") {
    return Color.fromCssColorString("#2dd4bf");
  }
  return Color.fromCssColorString("#64748b");
}

function oilNodePixelSize(node: OilNode, isSelected: boolean): number {
  if (isSelected) {
    return 16;
  }
  const production = node.productionBpd ?? 0;
  if (node.kind === "project") {
    return Math.max(9, Math.min(20, 9 + production / 300_000));
  }
  if (node.kind === "extraction_site") {
    return Math.max(6, Math.min(15, 6 + production / 150_000));
  }
  if (node.kind === "chokepoint") {
    return 13;
  }
  const flow = node.currentFlowBpd ?? node.capacityBpd ?? 0;
  return Math.max(8, Math.min(15, 8 + flow / 500_000));
}

function oilEdgeWidth(edge: OilEdge, isSelected: boolean): number {
  if (isSelected) {
    return 6;
  }
  return Math.max(2.5, Math.min(7, 2.5 + (edge.currentFlowBpd ?? edge.capacityBpd ?? 0) / 500_000));
}

function lonInRange(lon: number, west: number, east: number): boolean {
  if (west <= east) {
    return lon >= west && lon <= east;
  }
  return lon >= west || lon <= east;
}

function isOilPointVisible(
  lat: number,
  lon: number,
  viewRect: { west: number; south: number; east: number; north: number } | null,
): boolean {
  if (!viewRect) {
    return true;
  }
  return lat >= viewRect.south
    && lat <= viewRect.north
    && lonInRange(lon, viewRect.west, viewRect.east);
}

function isOilRouteVisible(
  route: { lat: number; lon: number }[],
  viewRect: { west: number; south: number; east: number; north: number } | null,
): boolean {
  for (const point of route) {
    if (isOilPointVisible(point.lat, point.lon, viewRect)) {
      return true;
    }
  }
  return false;
}

function shouldRenderOilNode(node: OilNode, selectedProjectId: string | null, selectedProjectChildIDs: Set<string>): boolean {
  if (node.kind === "project") {
    return true;
  }
  if (node.kind === "extraction_site") {
    return !!selectedProjectId && (node.parentProjectId === selectedProjectId || selectedProjectChildIDs.has(node.id));
  }
  if (node.kind === "chokepoint") {
    return true;
  }
  return (node.currentFlowBpd ?? 0) >= 150_000
    || (node.capacityBpd ?? 0) >= 250_000
    || node.kind === "gathering_hub"
    || node.kind === "pipeline_terminal"
    || node.kind === "refinery"
    || node.kind === "export_terminal"
    || node.kind === "import_terminal"
    || node.kind === "demand_center";
}

function shouldRenderOilEdge(edge: OilEdge): boolean {
  return edge.kind === "pipeline"
    || edge.kind === "internal_transfer"
    || (edge.currentFlowBpd ?? 0) >= 200_000
    || edge.kind === "shipping_lane" && !!edge.crossesChokepoint
    || edge.kind === "shipping_lane" && (edge.crossesChokepoints?.length ?? 0) > 0;
}

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

  const syncOilGraph = (
    oilGraph: OilGraph | null,
    visible: boolean,
    selectedNodeId: string | null,
    selectedEdgeId: string | null,
  ) => {
    const nodeIds = new Set<string>();
    const edgeIds = new Set<string>();
    const outlineIds = new Set<string>();
    const nodesById = new Map<string, OilNode>();
    const rect = viewer.camera.computeViewRectangle(viewer.scene.globe.ellipsoid);
    const viewRect = rect ? {
      west: CesiumMath.toDegrees(rect.west),
      south: CesiumMath.toDegrees(rect.south),
      east: CesiumMath.toDegrees(rect.east),
      north: CesiumMath.toDegrees(rect.north),
    } : null;
    const selectedProjectId = (() => {
      if (!selectedNodeId || !oilGraph) {
        return null;
      }
      const selected = oilGraph.nodes.find((node) => node.id === selectedNodeId);
      return selected?.kind === "project" ? selected.id : null;
    })();
    const selectedProjectChildIDs = (() => {
      if (!selectedProjectId || !oilGraph) {
        return new Set<string>();
      }
      const selected = oilGraph.nodes.find((node) => node.id === selectedProjectId);
      return new Set(selected?.childFieldIds ?? []);
    })();
    const visibleNodes = (oilGraph?.nodes ?? [])
      .filter((node) => shouldRenderOilNode(node, selectedProjectId, selectedProjectChildIDs));
    for (const node of visibleNodes) {
      const pointVisible = visible && isOilPointVisible(node.lat, node.lon, viewRect);
      nodesById.set(node.id, node);
      const entityId = `${OIL_NODE_PREFIX}${node.id}`;
      nodeIds.add(entityId);
      const existing = viewer.entities.getById(entityId);
      const isSelected = selectedNodeId === node.id;
      const color = oilNodeColor(node.kind);
      const position = Cartesian3.fromDegrees(node.lon, node.lat, 0);
      if (existing) {
        existing.show = pointVisible;
        (existing.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(position);
        if (existing.point) {
          existing.point.pixelSize = new ConstantProperty(oilNodePixelSize(node, isSelected));
          existing.point.color = new ConstantProperty(color.withAlpha(node.state === "offline" ? 0.45 : 0.95));
          existing.point.outlineWidth = new ConstantProperty(isSelected ? 3 : 2);
          existing.point.disableDepthTestDistance = undefined;
        }
      } else {
        viewer.entities.add(new Entity({
          id: entityId,
          show: pointVisible,
          position,
          point: {
            pixelSize: oilNodePixelSize(node, isSelected),
            color: color.withAlpha(node.state === "offline" ? 0.45 : 0.95),
            outlineColor: Color.WHITE.withAlpha(0.9),
            outlineWidth: isSelected ? 3 : 2,
            heightReference: HeightReference.CLAMP_TO_GROUND,
            distanceDisplayCondition: new DistanceDisplayCondition(0, 1.4e7),
          },
        }));
      }
    }

    const selectedProject = selectedProjectId
      ? oilGraph?.nodes.find((node) => node.id === selectedProjectId)
      : null;
    for (const [index, ring] of (selectedProject?.outlineRings ?? []).entries()) {
      if (!Array.isArray(ring) || ring.length < 2) {
        continue;
      }
      const entityId = `${OIL_OUTLINE_PREFIX}${selectedProject!.id}_${index}`;
      outlineIds.add(entityId);
      const positions = ring.map((point) => Cartesian3.fromDegrees(point.lon, point.lat, 0));
      const ringVisible = visible;
      const existing = viewer.entities.getById(entityId);
      if (existing?.polyline) {
        existing.show = ringVisible;
        existing.polyline.positions = new ConstantProperty(positions);
      } else {
        viewer.entities.add(new Entity({
          id: entityId,
          show: ringVisible,
          polyline: {
            positions: new ConstantProperty(positions),
            width: 3,
            material: new PolylineDashMaterialProperty({
              color: Color.fromCssColorString("#fbbf24").withAlpha(0.9),
              dashLength: 12,
            }),
            clampToGround: true,
          },
        }));
      }
    }

    const visibleEdges = (oilGraph?.edges ?? [])
      .filter((edge) => shouldRenderOilEdge(edge))
      .slice(0, 7000);

    for (const edge of visibleEdges) {
      const isSelected = selectedEdgeId === edge.id;
      const color = oilEdgeColor(edge);
      const routes = edge.routes && edge.routes.length > 0
        ? edge.routes
        : edge.route && edge.route.length > 1
          ? [edge.route]
          : (() => {
              const from = nodesById.get(edge.fromNodeId);
              const to = nodesById.get(edge.toNodeId);
              if (!from || !to) return [];
              return [[
                { lat: from.lat, lon: from.lon },
                { lat: to.lat, lon: to.lon },
              ]];
            })();
      routes.forEach((route, partIndex) => {
        const entityId = `${OIL_EDGE_PREFIX}${edge.id}__part_${partIndex}`;
        edgeIds.add(entityId);
        const existing = viewer.entities.getById(entityId);
        if (!route || route.length < 2) {
          viewer.entities.removeById(entityId);
          return;
        }
      const routeVisible = visible;
        const positions = route.map((point) => Cartesian3.fromDegrees(point.lon, point.lat, 0));
        if (existing) {
          existing.show = routeVisible;
          if (existing.polyline) {
            existing.polyline.positions = new ConstantProperty(positions);
            existing.polyline.width = new ConstantProperty(oilEdgeWidth(edge, isSelected));
            existing.polyline.material = edge.kind === "internal_transfer"
              ? new PolylineDashMaterialProperty({
                  color: color.withAlpha(edge.state === "offline" ? 0.28 : (isSelected ? 0.98 : 0.72)),
                  dashLength: 8,
                })
              : new ColorMaterialProperty(color.withAlpha(edge.state === "offline" ? 0.28 : (isSelected ? 0.98 : 0.8)));
            existing.polyline.clampToGround = new ConstantProperty(true);
            existing.polyline.zIndex = new ConstantProperty(isSelected ? 20 : 10);
          }
        } else {
          viewer.entities.add(new Entity({
            id: entityId,
            show: routeVisible,
            polyline: {
              positions: new ConstantProperty(positions),
              width: oilEdgeWidth(edge, isSelected),
              material: edge.kind === "internal_transfer"
                ? new PolylineDashMaterialProperty({
                    color: color.withAlpha(edge.state === "offline" ? 0.28 : (isSelected ? 0.98 : 0.72)),
                    dashLength: 8,
                  })
                : color.withAlpha(edge.state === "offline" ? 0.28 : (isSelected ? 0.98 : 0.8)),
              clampToGround: true,
              zIndex: isSelected ? 20 : 10,
            },
          }));
        }
      });
    }

    Array.from(viewer.entities.values)
      .map((entity) => entity.id as string)
      .filter((id) => id.startsWith(OIL_NODE_PREFIX) && !nodeIds.has(id))
      .forEach((id) => viewer.entities.removeById(id));
    Array.from(viewer.entities.values)
      .map((entity) => entity.id as string)
      .filter((id) => id.startsWith(OIL_EDGE_PREFIX) && !edgeIds.has(id))
      .forEach((id) => viewer.entities.removeById(id));
    Array.from(viewer.entities.values)
      .map((entity) => entity.id as string)
      .filter((id) => id.startsWith(OIL_OUTLINE_PREFIX) && !outlineIds.has(id))
      .forEach((id) => viewer.entities.removeById(id));
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
      .catch(console.error);

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
    syncMunitions(munitions, activeView, munitionDetections);
    syncExplosions(explosions);
    syncOilGraph(oilGraph, oilLayerVisible, selectedOilNodeId, selectedOilEdgeId);
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
      syncExplosions(state.explosions);
      return;
    }
    if (viewChanged) {
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
      syncMunitions(state.munitions, state.activeView, state.munitionDetections);
      return;
    }
    if (explosionsChanged) {
      syncExplosions(state.explosions);
      return;
    }
    if (oilGraphChanged || oilLayerChanged || oilSelectionChanged) {
      syncOilGraph(state.oilGraph, state.oilLayerVisible, state.selectedOilNodeId, state.selectedOilEdgeId);
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
      syncOilGraph(oilGraph, oilLayerVisible, selectedOilNodeId, selectedOilEdgeId);
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
