import { describe, expect, it } from "vitest";
import type { OilEdge, OilNode } from "../../store/simTypes";
import {
  hasGOGISource,
  isOilPointVisible,
  oilEdgeWidth,
  oilNodeOutlineColor,
  oilNodePixelSize,
  shouldRenderOilEdge,
  shouldRenderOilNode,
} from "./oilLayer";

function makeNode(overrides: Partial<OilNode> = {}): OilNode {
  return {
    id: "node-1",
    name: "Node",
    kind: "project",
    countryCode: "USA",
    lat: 0,
    lon: 0,
    state: "operational",
    ...overrides,
  };
}

function makeEdge(overrides: Partial<OilEdge> = {}): OilEdge {
  return {
    id: "edge-1",
    name: "Edge",
    kind: "pipeline",
    fromNodeId: "a",
    toNodeId: "b",
    commodity: "crude",
    state: "operational",
    capacityBpd: 100_000,
    currentFlowBpd: 100_000,
    ...overrides,
  };
}

describe("oilLayer helpers", () => {
  it("always renders project nodes even without selection", () => {
    expect(shouldRenderOilNode(makeNode({ kind: "project" }), null, new Set())).toBe(true);
  });

  it("only renders extraction sites when their project is selected", () => {
    const field = makeNode({ kind: "extraction_site", id: "field-1", parentProjectId: "project-1" });
    expect(shouldRenderOilNode(field, null, new Set())).toBe(false);
    expect(shouldRenderOilNode(field, "project-1", new Set())).toBe(true);
    expect(shouldRenderOilNode(field, "project-2", new Set(["field-1"]))).toBe(true);
  });

  it("renders important infrastructure nodes without project selection", () => {
    expect(shouldRenderOilNode(makeNode({ kind: "refinery", currentFlowBpd: 10_000 }), null, new Set())).toBe(true);
    expect(shouldRenderOilNode(makeNode({ kind: "pipeline_terminal", currentFlowBpd: 10_000 }), null, new Set())).toBe(true);
    expect(shouldRenderOilNode(makeNode({ kind: "marine_terminal", currentFlowBpd: 10_000 }), null, new Set())).toBe(true);
    expect(shouldRenderOilNode(makeNode({ kind: "storage_hub", currentFlowBpd: 10_000 }), null, new Set())).toBe(false);
    expect(shouldRenderOilNode(makeNode({ kind: "storage_hub", currentFlowBpd: 200_000 }), null, new Set())).toBe(true);
  });

  it("renders pipelines and hides placeholder seaborne corridors", () => {
    expect(shouldRenderOilEdge(makeEdge({ kind: "pipeline" }))).toBe(true);
    expect(shouldRenderOilEdge(makeEdge({ kind: "internal_transfer", currentFlowBpd: 10_000 }))).toBe(true);
    expect(shouldRenderOilEdge(makeEdge({ kind: "shipping_lane", currentFlowBpd: 50_000, crossesChokepoints: ["om-hormuz"] }))).toBe(true);
    expect(shouldRenderOilEdge(makeEdge({ kind: "seaborne_corridor", currentFlowBpd: 500_000 } as any))).toBe(false);
    expect(shouldRenderOilEdge(makeEdge({ kind: "shipping_lane", currentFlowBpd: 50_000 }))).toBe(false);
  });

  it("culls points outside the current view rectangle including wrapped longitude ranges", () => {
    const rect = { west: 170, south: -10, east: -170, north: 10 };
    expect(isOilPointVisible(0, 175, rect)).toBe(true);
    expect(isOilPointVisible(0, -175, rect)).toBe(true);
    expect(isOilPointVisible(0, -120, rect)).toBe(false);
    expect(isOilPointVisible(30, 175, rect)).toBe(false);
  });

  it("scales project and field nodes by production and selection", () => {
    const project = makeNode({ kind: "project", productionBpd: 3_000_000 });
    const field = makeNode({ kind: "extraction_site", productionBpd: 900_000 });
    expect(oilNodePixelSize(project, false)).toBeGreaterThan(oilNodePixelSize(makeNode({ kind: "project", productionBpd: 100_000 }), false));
    expect(oilNodePixelSize(field, false)).toBeGreaterThan(oilNodePixelSize(makeNode({ kind: "extraction_site", productionBpd: 50_000 }), false));
    expect(oilNodePixelSize(field, true)).toBe(16);
  });

  it("widens selected or high-flow edges", () => {
    const low = oilEdgeWidth(makeEdge({ currentFlowBpd: 50_000 }), false);
    const high = oilEdgeWidth(makeEdge({ currentFlowBpd: 2_000_000 }), false);
    const selected = oilEdgeWidth(makeEdge({ currentFlowBpd: 50_000 }), true);
    expect(high).toBeGreaterThan(low);
    expect(selected).toBe(6);
  });

  it("detects GOGI-backed nodes and gives them a distinct outline", () => {
    const gogiNode = makeNode({
      kind: "refinery",
      tags: ["source:gogi"],
      sources: [{ organization: "EDX", name: "GOGI", confidence: 0.78 }],
    } as any);
    expect(hasGOGISource(gogiNode)).toBe(true);
    expect(oilNodeOutlineColor(gogiNode, false).toCssColorString()).toBe("rgba(34,197,94,0.95)");
    expect(shouldRenderOilNode(gogiNode, null, new Set())).toBe(true);
  });
});
