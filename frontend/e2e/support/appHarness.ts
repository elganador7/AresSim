import type { Page } from "@playwright/test";

type HarnessGraph = {
  id: string;
  name: string;
  description: string;
  version: string;
  view: string;
  nodes: Array<Record<string, unknown>>;
  edges: Array<Record<string, unknown>>;
};

type HarnessScenario = Record<string, unknown>;

type HarnessConfig = {
  oilGraph?: HarnessGraph;
  scenarios?: HarnessScenario[];
  unitDefinitions?: Record<string, unknown>[];
  weaponDefinitions?: Record<string, unknown>[];
};

const defaultOilGraph: HarnessGraph = {
  id: "oil-harness",
  name: "Oil Harness",
  description: "Browser harness oil graph",
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
      h3Cell: "8c3b0e8e33a89ff",
      h3ParentCell: "8b3b0e8e33a9fff",
      state: "operational",
      currentFlowBpd: 20_000_000,
      capacityBpd: 23_000_000,
    },
    {
      id: "sa-ras-tanura",
      name: "Ras Tanura",
      kind: "marine_terminal",
      countryCode: "SAU",
      lat: 26.647,
      lon: 50.163,
      h3Cell: "8c2f9743028d3ff",
      h3ParentCell: "8b2f9743028ffff",
      state: "operational",
      currentFlowBpd: 6_000_000,
      capacityBpd: 6_500_000,
    },
  ],
  edges: [
    {
      id: "pipe-1",
      name: "Export Corridor",
      kind: "pipeline",
      fromNodeId: "sa-ras-tanura",
      toNodeId: "om-hormuz",
      commodity: "crude",
      state: "operational",
      capacityBpd: 1_000_000,
      currentFlowBpd: 750_000,
    },
  ],
};

const defaultScenarios: HarnessScenario[] = [
  {
    id: "pg-package-strait-regional-control",
    name: "Proving Ground: Package Strait Regional Control",
    author: "Harness",
    description: "Harness proving ground",
    scenario_kind: "proving_ground",
    recommended_trials: 3,
  },
  {
    id: "default",
    name: "Default Scenario",
    author: "Harness",
    description: "Baseline scenario",
    scenario_kind: "scenario",
  },
  {
    id: "isr-air-defense",
    name: "Israel Air Defense Drill",
    author: "Harness",
    description: "Regional air-defense scenario",
    scenario_kind: "scenario",
  },
];

const defaultUnitDefinitions: Record<string, unknown>[] = [
  {
    id: "usa-f35a",
    name: "F-35A Lightning II",
    short_name: "F-35A",
    shortName: "F-35A",
    general_type: 2,
    generalType: 2,
    domain: 2,
    nation_of_origin: "USA",
    nationOfOrigin: "USA",
    employed_by: ["USA"],
    employedBy: ["USA"],
    employment_role: "strike",
    employmentRole: "strike",
    default_weapon_configuration: "",
    defaultWeaponConfiguration: "",
    weapon_configurations: [],
    weaponConfigurations: [],
  },
  {
    id: "usa-kc46",
    name: "KC-46 Pegasus",
    short_name: "KC-46",
    shortName: "KC-46",
    general_type: 2,
    generalType: 2,
    domain: 2,
    nation_of_origin: "USA",
    nationOfOrigin: "USA",
    employed_by: ["USA"],
    employedBy: ["USA"],
    employment_role: "support",
    employmentRole: "support",
    default_weapon_configuration: "",
    defaultWeaponConfiguration: "",
    weapon_configurations: [],
    weaponConfigurations: [],
  },
  {
    id: "isr-f35i",
    name: "F-35I Adir",
    short_name: "F-35I",
    shortName: "F-35I",
    general_type: 2,
    generalType: 2,
    domain: 2,
    nation_of_origin: "ISR",
    nationOfOrigin: "ISR",
    employed_by: ["ISR"],
    employedBy: ["ISR"],
    employment_role: "strike",
    employmentRole: "strike",
    default_weapon_configuration: "",
    defaultWeaponConfiguration: "",
    weapon_configurations: [],
    weaponConfigurations: [],
  },
];

const defaultWeaponDefinitions: Record<string, unknown>[] = [
  {
    id: "aim120",
    domain_targets: [2],
    effect_type: 1,
  },
];

export async function installAppHarness(page: Page, config: HarnessConfig = {}) {
  const payload = {
    oilGraph: config.oilGraph ?? defaultOilGraph,
    scenarios: config.scenarios ?? defaultScenarios,
    unitDefinitions: config.unitDefinitions ?? defaultUnitDefinitions,
    weaponDefinitions: config.weaponDefinitions ?? defaultWeaponDefinitions,
  };

  await page.addInitScript((seed) => {
    const listeners = new Map<string, Array<(...args: any[]) => void>>();
    const appState = {
      oilGraph: seed.oilGraph,
      scenarios: seed.scenarios,
      unitDefinitions: seed.unitDefinitions,
      weaponDefinitions: seed.weaponDefinitions,
    };

    const ok = () => Promise.resolve({ success: true });

    const runtime = {
      LogPrint() {},
      LogTrace() {},
      LogDebug() {},
      LogInfo() {},
      LogWarning() {},
      LogError() {},
      LogFatal() {},
      EventsOnMultiple(eventName: string, callback: (...args: any[]) => void) {
        const current = listeners.get(eventName) ?? [];
        current.push(callback);
        listeners.set(eventName, current);
        return () => {
          const next = (listeners.get(eventName) ?? []).filter((entry) => entry !== callback);
          listeners.set(eventName, next);
        };
      },
      EventsOff(eventName: string) {
        listeners.delete(eventName);
      },
      EventsOffAll() {
        listeners.clear();
      },
      EventsEmit(eventName: string, ...args: any[]) {
        for (const callback of listeners.get(eventName) ?? []) {
          callback(...args);
        }
      },
    };

    const app = {
      AppendMoveWaypoint: ok,
      CancelMoveOrder: ok,
      DeleteScenario: ok,
      DeleteUnitDefinition: ok,
      GetGlobalOilNetwork: () => Promise.resolve(appState.oilGraph),
      GetRenderableOilNetwork: () => Promise.resolve(appState.oilGraph),
      GetScenario: () => Promise.resolve("HARNESS_PROTO"),
      GetVersion: () => Promise.resolve("harness"),
      ListScenarios: () => Promise.resolve(appState.scenarios),
      ListUnitDefinitions: () => Promise.resolve(appState.unitDefinitions),
      ListWeaponDefinitions: () => Promise.resolve(appState.weaponDefinitions),
      LoadScenarioFromProto: ok,
      MoveUnit: ok,
      PauseSim: ok,
      PreviewCurrentEngagement: () => Promise.resolve({}),
      PreviewCurrentRelationships: () => Promise.resolve([]),
      PreviewCurrentStrikePath: () => Promise.resolve({ blocked: false, routePoints: [] }),
      PreviewCurrentTransitPath: () => Promise.resolve({ blocked: false, routePoints: [] }),
      PreviewDraftPlacement: () => Promise.resolve({ blocked: false }),
      PreviewDraftRelationships: () => Promise.resolve([]),
      PreviewDraftStrikePath: () => Promise.resolve({ blocked: false, routePoints: [] }),
      PreviewDraftTransitPath: () => Promise.resolve({ blocked: false, routePoints: [] }),
      PreviewEngagementOptions: () => Promise.resolve([]),
      PreviewEngagementOptionsForLoadout: () => Promise.resolve([]),
      PreviewTargetEngagementOptions: () => Promise.resolve([]),
      PreviewTargetEngagementSummary: () => Promise.resolve({}),
      RemoveMoveWaypoint: ok,
      RequestSync: ok,
      ReturnUnitToBase: ok,
      RunProvingGroundScenario: () => Promise.resolve({
        pass: true,
        trials: 3,
        focusTeam: "USA",
        focusWinRate: 1,
      }),
      SaveScenario: ok,
      SaveUnitDefinition: ok,
      SetCountryRelationship: ok,
      SetHumanControlledTeam: ok,
      SetIntelSharing: ok,
      SetSimSpeed: ok,
      SetUnitAttackOrder: ok,
      SetUnitEngagement: ok,
      SetUnitLoadoutConfiguration: ok,
      SimulateOilShock: () => Promise.resolve({}),
      SimulateOilShockHorizon: () => Promise.resolve({}),
      UpdateMoveWaypoint: ok,
    };

    (window as any).runtime = runtime;
    (window as any).go = { main: { App: app } };
    (window as any).__ARES_E2E__ = {
      emit(eventName: string, ...args: any[]) {
        runtime.EventsEmit(eventName, ...args);
      },
      setOilGraph(graph: any) {
        appState.oilGraph = graph;
      },
      setScenarios(scenarios: any[]) {
        appState.scenarios = scenarios;
      },
    };
  }, payload);

  await page.route("https://tile.openstreetmap.org/**", async (route) => {
    await route.fulfill({ status: 204, body: "" });
  });
}
