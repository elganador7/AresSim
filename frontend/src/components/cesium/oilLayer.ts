import {
  Cartesian3,
  Color,
  ColorMaterialProperty,
  ConstantProperty,
  DistanceDisplayCondition,
  Entity,
  HeightReference,
  Math as CesiumMath,
  PolylineDashMaterialProperty,
  Viewer,
} from "cesium";
import type { OilEdge, OilGraph, OilNode } from "../../store/simStore";

export const OIL_NODE_PREFIX = "oil_node_";
export const OIL_EDGE_PREFIX = "oil_edge_";
export const OIL_OUTLINE_PREFIX = "oil_outline_";

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

export function oilNodePixelSize(node: OilNode, isSelected: boolean): number {
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

export function oilEdgeWidth(edge: OilEdge, isSelected: boolean): number {
  if (isSelected) {
    return 6;
  }
  return Math.max(2.5, Math.min(7, 2.5 + (edge.currentFlowBpd ?? edge.capacityBpd ?? 0) / 500_000));
}

export function lonInRange(lon: number, west: number, east: number): boolean {
  if (west <= east) {
    return lon >= west && lon <= east;
  }
  return lon >= west || lon <= east;
}

export function isOilPointVisible(
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

export function shouldRenderOilNode(node: OilNode, selectedProjectId: string | null, selectedProjectChildIDs: Set<string>): boolean {
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

export function shouldRenderOilEdge(edge: OilEdge): boolean {
  return edge.kind === "pipeline"
    || edge.kind === "internal_transfer"
    || (edge.currentFlowBpd ?? 0) >= 200_000
    || edge.kind === "shipping_lane" && !!edge.crossesChokepoint
    || edge.kind === "shipping_lane" && (edge.crossesChokepoints?.length ?? 0) > 0;
}

export function syncOilGraph(
  viewer: Viewer,
  oilGraph: OilGraph | null,
  visible: boolean,
  selectedNodeId: string | null,
  selectedEdgeId: string | null,
) {
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
}
