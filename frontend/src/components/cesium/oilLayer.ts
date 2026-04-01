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

export interface OilLayerSyncState {
  nodeEntityIds: Set<string>;
  edgeEntityIds: Set<string>;
  outlineEntityIds: Set<string>;
  lastCameraBucketKey: string | null;
}

export type OilZoomBand = "global" | "regional" | "local";

export function createOilLayerSyncState(): OilLayerSyncState {
  return {
    nodeEntityIds: new Set<string>(),
    edgeEntityIds: new Set<string>(),
    outlineEntityIds: new Set<string>(),
    lastCameraBucketKey: null,
  };
}

export function oilZoomBandForHeight(heightM: number): OilZoomBand {
  if (heightM >= 2_500_000) {
    return "global";
  }
  if (heightM >= 600_000) {
    return "regional";
  }
  return "local";
}

export function oilCameraBucketKey(
  heightM: number,
  viewRect: { west: number; south: number; east: number; north: number } | null,
): string {
  const zoomBand = oilZoomBandForHeight(heightM);
  if (!viewRect) {
    return `${zoomBand}:none`;
  }
  const gridDeg = zoomBand === "global" ? 10 : zoomBand === "regional" ? 3 : 1;
  const bucket = (value: number) => Math.round(value / gridDeg) * gridDeg;
  return [
    zoomBand,
    bucket(viewRect.west),
    bucket(viewRect.south),
    bucket(viewRect.east),
    bucket(viewRect.north),
  ].join(":");
}

export function hasGOGISource(node: OilNode): boolean {
  return (node.tags ?? []).includes("source:gogi")
    || (node.sources ?? []).some((source) => {
      const org = String(source.organization ?? "").toLowerCase();
      const name = String(source.name ?? "").toLowerCase();
      return org.includes("edx") || name.includes("gogi");
    });
}

function oilNodeColor(kind: string): Color {
  switch (kind) {
    case "project":
      return Color.fromCssColorString("#f59e0b");
    case "extraction_site":
      return Color.fromCssColorString("#f97316");
    case "refinery":
      return Color.fromCssColorString("#38bdf8");
    case "marine_terminal":
      return Color.fromCssColorString("#facc15");
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

export function oilNodeOutlineColor(node: OilNode, isSelected: boolean): Color {
  if (isSelected) {
    return Color.WHITE.withAlpha(0.95);
  }
  if (hasGOGISource(node)) {
    return Color.fromCssColorString("#22c55e").withAlpha(0.95);
  }
  return Color.WHITE.withAlpha(0.9);
}

function oilNodeCellKey(node: OilNode, zoomBand: OilZoomBand): string {
  if (zoomBand === "global") {
    return node.h3ParentCell || node.h3Cell || node.id;
  }
  if (zoomBand === "regional") {
    return node.h3Cell || node.h3ParentCell || node.id;
  }
  return node.id;
}

export function oilEdgeWidth(edge: OilEdge, isSelected: boolean): number {
  if (isSelected) {
    return 6;
  }
  return Math.max(2.5, Math.min(7, 2.5 + (edge.currentFlowBpd ?? edge.capacityBpd ?? 0) / 500_000));
}

function oilNodeImportance(node: OilNode): number {
  let score = node.productionBpd ?? node.currentFlowBpd ?? node.capacityBpd ?? 0;
  switch (node.kind) {
    case "project":
      score += 3_000_000;
      break;
    case "chokepoint":
      score += 2_500_000;
      break;
    case "refinery":
      score += 2_000_000;
      break;
    case "marine_terminal":
    case "export_terminal":
    case "import_terminal":
      score += 1_600_000;
      break;
    case "storage_hub":
      score += 900_000;
      break;
    case "demand_center":
      score += 800_000;
      break;
    case "extraction_site":
      score += 500_000;
      break;
    default:
      break;
  }
  if (hasGOGISource(node)) {
    score += 150_000;
  }
  if (node.state === "offline") {
    score -= 200_000;
  }
  return score;
}

function oilEdgeImportance(edge: OilEdge): number {
  let score = edge.currentFlowBpd ?? edge.capacityBpd ?? 0;
  if (edge.kind === "pipeline") {
    score += 1_000_000;
  }
  if (edge.kind === "internal_transfer") {
    score += 400_000;
  }
  if (edge.kind === "shipping_lane" && ((edge.crossesChokepoints?.length ?? 0) > 0 || edge.crossesChokepoint)) {
    score += 600_000;
  }
  if (edge.state === "offline") {
    score -= 200_000;
  }
  return score;
}

function oilNodeLimit(zoomBand: OilZoomBand): number {
  switch (zoomBand) {
    case "global":
      return 400;
    case "regional":
      return 1_000;
    case "local":
      return 2_000;
  }
}

function oilEdgeLimit(zoomBand: OilZoomBand): number {
  switch (zoomBand) {
    case "global":
      return 250;
    case "regional":
      return 700;
    case "local":
      return 1_500;
  }
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

export function shouldRenderOilNode(
  node: OilNode,
  selectedProjectId: string | null,
  selectedProjectChildIDs: Set<string>,
  zoomBand: OilZoomBand,
  selectedNodeId: string | null,
): boolean {
  if (selectedNodeId === node.id) {
    return true;
  }
  if (node.kind === "project") {
    return true;
  }
  if (node.kind === "extraction_site") {
    return !!selectedProjectId && (node.parentProjectId === selectedProjectId || selectedProjectChildIDs.has(node.id));
  }
  if (node.kind === "chokepoint") {
    return true;
  }
  if (zoomBand === "global") {
    return node.kind === "marine_terminal"
      || node.kind === "refinery"
      || node.kind === "export_terminal"
      || node.kind === "import_terminal"
      || ((node.currentFlowBpd ?? 0) >= 500_000)
      || ((node.capacityBpd ?? 0) >= 750_000);
  }
  if (zoomBand === "regional") {
    if (node.kind === "marine_terminal"
      || node.kind === "refinery"
      || node.kind === "export_terminal"
      || node.kind === "import_terminal"
      || node.kind === "storage_hub"
      || node.kind === "demand_center") {
      return true;
    }
    if (hasGOGISource(node)) {
      return node.kind !== "gathering_hub" || (node.currentFlowBpd ?? node.capacityBpd ?? 0) >= 150_000;
    }
    return (node.currentFlowBpd ?? 0) >= 200_000 || (node.capacityBpd ?? 0) >= 350_000;
  }
  if (hasGOGISource(node)) {
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

export function selectOilNodesForRender(
  nodes: OilNode[],
  selectedProjectId: string | null,
  selectedProjectChildIDs: Set<string>,
  zoomBand: OilZoomBand,
  selectedNodeId: string | null,
): OilNode[] {
  const selectedNodes = nodes.filter((node) => selectedNodeId === node.id);
  const selectedIds = new Set(selectedNodes.map((node) => node.id));
  const ranked = nodes
    .filter((node) => shouldRenderOilNode(node, selectedProjectId, selectedProjectChildIDs, zoomBand, selectedNodeId))
    .sort((a, b) => oilNodeImportance(b) - oilNodeImportance(a));
  const limit = oilNodeLimit(zoomBand);
  if (zoomBand === "local") {
    const out = [...selectedNodes];
    for (const node of ranked) {
      if (out.length >= limit) {
        break;
      }
      if (!selectedIds.has(node.id)) {
        out.push(node);
      }
    }
    return out;
  }

  const out = [...selectedNodes];
  const seenIds = new Set(selectedIds);
  const usedCells = new Set<string>();
  for (const node of selectedNodes) {
    usedCells.add(oilNodeCellKey(node, zoomBand));
  }
  for (const node of ranked) {
    if (out.length >= limit) {
      break;
    }
    if (seenIds.has(node.id)) {
      continue;
    }
    const cellKey = oilNodeCellKey(node, zoomBand);
    if (usedCells.has(cellKey)) {
      continue;
    }
    out.push(node);
    seenIds.add(node.id);
    usedCells.add(cellKey);
  }
  return out;
}

export function shouldRenderOilEdge(edge: OilEdge, zoomBand: OilZoomBand, selectedEdgeId: string | null): boolean {
  if (selectedEdgeId === edge.id) {
    return true;
  }
  if (edge.kind === "seaborne_corridor") {
    return false;
  }
  if (zoomBand === "global") {
    return edge.kind === "pipeline"
      && ((edge.currentFlowBpd ?? 0) >= 500_000 || (edge.capacityBpd ?? 0) >= 750_000);
  }
  if (zoomBand === "regional") {
    return edge.kind === "pipeline"
      || edge.kind === "internal_transfer"
      || (edge.currentFlowBpd ?? 0) >= 250_000;
  }
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
  syncState: OilLayerSyncState,
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
  const cameraHeight = viewer.camera.positionCartographic?.height ?? 3_000_000;
  const zoomBand = oilZoomBandForHeight(cameraHeight);
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
  const visibleNodes = selectOilNodesForRender(
    oilGraph?.nodes ?? [],
    selectedProjectId,
    selectedProjectChildIDs,
    zoomBand,
    selectedNodeId,
  );
  for (const node of visibleNodes) {
    const pointVisible = visible && isOilPointVisible(node.lat, node.lon, viewRect);
    nodesById.set(node.id, node);
    const entityId = `${OIL_NODE_PREFIX}${node.id}`;
    nodeIds.add(entityId);
    const existing = viewer.entities.getById(entityId);
    const isSelected = selectedNodeId === node.id;
    const color = oilNodeColor(node.kind);
    const outlineColor = oilNodeOutlineColor(node, isSelected);
    const position = Cartesian3.fromDegrees(node.lon, node.lat, 0);
    if (existing) {
      existing.show = pointVisible;
      (existing.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(position);
      if (existing.point) {
        existing.point.pixelSize = new ConstantProperty(oilNodePixelSize(node, isSelected));
        existing.point.color = new ConstantProperty(color.withAlpha(node.state === "offline" ? 0.45 : 0.95));
        existing.point.outlineColor = new ConstantProperty(outlineColor);
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
          outlineColor,
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
    .filter((edge) => shouldRenderOilEdge(edge, zoomBand, selectedEdgeId))
    .sort((a, b) => oilEdgeImportance(b) - oilEdgeImportance(a))
    .slice(0, oilEdgeLimit(zoomBand));

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

  for (const id of syncState.nodeEntityIds) {
    if (!nodeIds.has(id)) {
      viewer.entities.removeById(id);
    }
  }
  for (const id of syncState.edgeEntityIds) {
    if (!edgeIds.has(id)) {
      viewer.entities.removeById(id);
    }
  }
  for (const id of syncState.outlineEntityIds) {
    if (!outlineIds.has(id)) {
      viewer.entities.removeById(id);
    }
  }
  syncState.nodeEntityIds = nodeIds;
  syncState.edgeEntityIds = edgeIds;
  syncState.outlineEntityIds = outlineIds;
}
