import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRoot, type Root } from "react-dom/client";
import { act } from "react";
import { useSimStore } from "../store/simStore";

const appMocks = vi.hoisted(() => ({
  GetRenderableOilNetwork: vi.fn(),
  PauseSim: vi.fn(() => Promise.resolve({ success: true })),
  RequestSync: vi.fn(() => Promise.resolve({ success: true })),
  SetHumanControlledTeam: vi.fn(() => Promise.resolve({ success: true })),
  SetSimSpeed: vi.fn(() => Promise.resolve({ success: true })),
}));

vi.mock("../../wailsjs/go/main/App", () => appMocks);
vi.mock("../../wailsjs/runtime/runtime", () => ({
  EventsOn: vi.fn(),
}));

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

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
}

describe("oil bridge integration", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it("loads the renderable oil graph into the store on initBridge", async () => {
    appMocks.GetRenderableOilNetwork.mockResolvedValue({
      id: "oil-test",
      name: "Oil Test",
      description: "",
      version: "1",
      view: "global",
      nodes: [
        {
          id: "om-hormuz",
          name: "Strait of Hormuz",
          kind: "chokepoint",
          countryCode: "OMN",
          lat: 26.566,
          lon: 56.25,
          state: "operational",
          currentFlowBpd: 20_000_000,
          capacityBpd: 23_000_000,
        },
      ],
      edges: [],
    });

    const { initBridge } = await import("./bridge");
    initBridge();
    await flushPromises();

    const state = useSimStore.getState();
    expect(appMocks.GetRenderableOilNetwork).toHaveBeenCalledTimes(1);
    expect(state.oilGraph?.id).toBe("oil-test");
    expect(state.oilGraph?.nodes).toHaveLength(1);
    expect(state.oilLoadError).toBeNull();
  });

  it("stores an oil load error when the bridge request fails", async () => {
    appMocks.GetRenderableOilNetwork.mockRejectedValue(new Error("backend unavailable"));

    const { initBridge } = await import("./bridge");
    initBridge();
    await flushPromises();

    const state = useSimStore.getState();
    expect(state.oilGraph).toBeNull();
    expect(state.oilLoadError).toContain("backend unavailable");
  });
});

describe("TopBar oil menu integration", () => {
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

  it("shows oil node count when the oil graph is loaded", async () => {
    useSimStore.setState({
      oilGraph: {
        id: "oil-test",
        name: "Oil Test",
        description: "",
        version: "1",
        view: "global",
        nodes: [
          {
            id: "om-hormuz",
            name: "Strait of Hormuz",
            kind: "chokepoint",
            countryCode: "OMN",
            lat: 26.566,
            lon: 56.25,
            state: "operational",
          },
          {
            id: "sa-ras-tanura",
            name: "Ras Tanura",
            kind: "export_terminal",
            countryCode: "SAU",
            lat: 26.647,
            lon: 50.163,
            state: "operational",
          },
        ],
        edges: [],
      } as any,
    });

    const { default: TopBar } = await import("../components/hud/TopBar");
    await act(async () => {
      root.render(
        <TopBar
          onOpenEditor={() => {}}
          onOpenScenario={() => {}}
          debugViewMenuVisible={false}
        />,
      );
    });

    const menuButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent?.includes("MENU"));
    expect(menuButton).toBeTruthy();

    await act(async () => {
      menuButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(container.textContent).toContain("Focus Oil Network (2 nodes)");
  });

  it("shows load failure status when oil graph startup fails", async () => {
    useSimStore.setState({ oilLoadError: "backend unavailable" });

    const { default: TopBar } = await import("../components/hud/TopBar");
    await act(async () => {
      root.render(
        <TopBar
          onOpenEditor={() => {}}
          onOpenScenario={() => {}}
          debugViewMenuVisible={false}
        />,
      );
    });

    const menuButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent?.includes("MENU"));
    expect(menuButton).toBeTruthy();

    await act(async () => {
      menuButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(container.textContent).toContain("Focus Oil Network (Load Failed)");
  });
});
