import React from "react";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import OilPanel from "./OilPanel";
import { useSimStore } from "../../store/simStore";

function resetStore() {
  useSimStore.setState({
    scenarioName: "",
    scenarioState: "idle",
    timeScale: 1.0,
    simSeconds: 0,
    tickNumber: 0,
    relationships: [],
    scores: [],
    units: new Map(),
    weaponDefs: new Map(),
    munitions: new Map(),
    explosions: new Map(),
    munitionDetections: new Map(),
    activeView: "debug",
    humanControlledTeam: "",
    oilGraph: null,
    oilLayerVisible: true,
    oilFocusToken: 0,
    oilLoadError: null,
    detections: new Map(),
    detectionContacts: new Map(),
    selectedUnitId: null,
    selectedTargetId: null,
    selectedOilNodeId: null,
    selectedOilEdgeId: null,
    mapCommandMode: { type: "none", unitId: null },
    selectedRoutePreview: null,
    selectedStrikePreview: null,
    eventLog: [],
  });
}

describe("OilPanel", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    resetStore();
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(async () => {
    await act(async () => {
      root.unmount();
    });
    container.remove();
  });

  it("shows project production, reserves, and child field count", async () => {
    useSimStore.setState({
      selectedOilNodeId: "project-1",
      oilGraph: {
        id: "oil",
        name: "Oil",
        description: "",
        version: "1",
        view: "global",
        nodes: [{
          id: "project-1",
          name: "Ahvaz Oil Project",
          kind: "project",
          countryCode: "IRN",
          lat: 31.3,
          lon: 48.7,
          state: "operational",
          productionBpd: 1_250_000,
          reserveBbl: 65_000_000_000,
          childFieldIds: ["field-1", "field-2", "field-3"],
          primaryCommodity: "crude",
          sources: [
            { name: "GEM", organization: "Global Energy Monitor", url: "", confidence: 0.9 },
            { name: "GOGI", organization: "EDX", url: "", confidence: 0.78 },
          ],
        }],
        edges: [],
      } as any,
    });

    await act(async () => {
      root.render(<OilPanel />);
    });

    expect(container.textContent).toContain("Ahvaz Oil Project");
    expect(container.textContent).toContain("Production");
    expect(container.textContent).toContain("Reserves");
    expect(container.textContent).toContain("Fields");
    expect(container.textContent).toContain("3");
    expect(container.textContent).toContain("Source Blend");
    expect(container.textContent).toContain("Global Energy Monitor, EDX");
  });

  it("shows pipeline product label and chokepoint details for selected edges", async () => {
    useSimStore.setState({
      selectedOilEdgeId: "edge-1",
      oilGraph: {
        id: "oil",
        name: "Oil",
        description: "",
        version: "1",
        view: "global",
        nodes: [],
        edges: [{
          id: "edge-1",
          name: "East-West Pipeline",
          kind: "pipeline",
          fromNodeId: "a",
          toNodeId: "b",
          commodity: "refined_products",
          commodityLabel: "NGL, oil products",
          state: "operational",
          capacityBpd: 5_000_000,
          currentFlowBpd: 3_000_000,
          crossesChokepoint: "None",
        }],
      } as any,
    });

    await act(async () => {
      root.render(<OilPanel />);
    });

    expect(container.textContent).toContain("East-West Pipeline");
    expect(container.textContent).toContain("NGL, oil products");
    expect(container.textContent).toContain("Chokepoint");
  });
});
