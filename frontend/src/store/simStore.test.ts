/**
 * simStore.test.ts
 *
 * Unit tests for the Zustand simulation store.
 * Tests all actions that bridge.ts can call.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { useSimStore } from "./simStore";
import type { Unit } from "./simStore";

// ─── helpers ──────────────────────────────────────────────────────────────────

function makeUnit(id: string, side: "Blue" | "Red" | "Neutral" = "Blue"): Unit {
  return {
    id,
    displayName: `Unit-${id}`,
    fullName: `Full Name ${id}`,
    side,
    natoPendingSymbol: "",
    definitionId: "def-1",
    position: { lat: 0, lon: 0, altMsl: 0, heading: 0, speed: 0 },
    status: {
      personnelStrength: 100,
      equipmentStrength: 100,
      combatEffectiveness: 1,
      fuelLevelLiters: 500,
      morale: 1,
      fatigue: 0,
      isActive: true,
      suppressed: false,
      disrupted: false,
      routing: false,
    },
  };
}

function resetStore() {
  useSimStore.setState({
    scenarioName: "",
    scenarioState: "idle",
    timeScale: 1.0,
    simSeconds: 0,
    tickNumber: 0,
    units: new Map(),
    activeView: "debug",
    detections: new Map(),
    selectedUnitId: null,
    eventLog: [],
  });
}

// ─── loadSnapshot ─────────────────────────────────────────────────────────────

describe("loadSnapshot", () => {
  beforeEach(resetStore);

  it("populates units map from array", () => {
    const store = useSimStore.getState();
    store.loadSnapshot([makeUnit("a"), makeUnit("b")], "Test Scenario");
    const s = useSimStore.getState();
    expect(s.units.size).toBe(2);
    expect(s.units.get("a")?.displayName).toBe("Unit-a");
    expect(s.units.get("b")?.side).toBe("Blue");
  });

  it("sets scenarioName", () => {
    useSimStore.getState().loadSnapshot([], "My Scenario");
    expect(useSimStore.getState().scenarioName).toBe("My Scenario");
  });

  it("resets simSeconds and tickNumber to 0", () => {
    useSimStore.setState({ simSeconds: 999, tickNumber: 42 });
    useSimStore.getState().loadSnapshot([], "S");
    const s = useSimStore.getState();
    expect(s.simSeconds).toBe(0);
    expect(s.tickNumber).toBe(0);
  });

  it("clears eventLog", () => {
    useSimStore.setState({
      eventLog: [{ id: "1", text: "hi", category: "combat", unitId: "", side: "", simSeconds: 0 }],
    });
    useSimStore.getState().loadSnapshot([], "S");
    expect(useSimStore.getState().eventLog).toHaveLength(0);
  });

  it("replaces previous units entirely", () => {
    useSimStore.getState().loadSnapshot([makeUnit("old")], "S");
    useSimStore.getState().loadSnapshot([makeUnit("new1"), makeUnit("new2")], "S");
    const units = useSimStore.getState().units;
    expect(units.has("old")).toBe(false);
    expect(units.has("new1")).toBe(true);
    expect(units.size).toBe(2);
  });
});

// ─── applyUnitDelta ───────────────────────────────────────────────────────────

describe("applyUnitDelta", () => {
  beforeEach(resetStore);

  it("merges position onto existing unit", () => {
    useSimStore.getState().loadSnapshot([makeUnit("u1")], "S");
    useSimStore.getState().applyUnitDelta("u1", {
      position: { lat: 10, lon: 20, altMsl: 100, heading: 90, speed: 5 },
    });
    const u = useSimStore.getState().units.get("u1")!;
    expect(u.position.lat).toBe(10);
    expect(u.position.lon).toBe(20);
    expect(u.displayName).toBe("Unit-u1"); // unchanged fields preserved
  });

  it("does nothing for unknown unit id", () => {
    useSimStore.getState().loadSnapshot([makeUnit("u1")], "S");
    const before = useSimStore.getState().units.get("u1")!.position.lat;
    useSimStore.getState().applyUnitDelta("unknown", {
      position: { lat: 99, lon: 99, altMsl: 0, heading: 0, speed: 0 },
    });
    expect(useSimStore.getState().units.get("u1")!.position.lat).toBe(before);
  });

  it("merges moveOrder onto unit", () => {
    useSimStore.getState().loadSnapshot([makeUnit("u1")], "S");
    useSimStore.getState().applyUnitDelta("u1", {
      moveOrder: { waypoints: [{ lat: 5, lon: 6, altMsl: 0 }] },
    });
    expect(useSimStore.getState().units.get("u1")!.moveOrder?.waypoints[0].lat).toBe(5);
  });

  it("clears moveOrder when set to undefined", () => {
    useSimStore.getState().loadSnapshot([
      { ...makeUnit("u1"), moveOrder: { waypoints: [{ lat: 1, lon: 1, altMsl: 0 }] } },
    ], "S");
    useSimStore.getState().applyUnitDelta("u1", { moveOrder: undefined });
    expect(useSimStore.getState().units.get("u1")!.moveOrder).toBeUndefined();
  });
});

// ─── spawnUnit ────────────────────────────────────────────────────────────────

describe("spawnUnit", () => {
  beforeEach(resetStore);

  it("adds a new unit to the map", () => {
    useSimStore.getState().spawnUnit(makeUnit("new"));
    expect(useSimStore.getState().units.get("new")?.id).toBe("new");
  });

  it("overwrites existing unit with same id", () => {
    useSimStore.getState().loadSnapshot([makeUnit("u")], "S");
    const replacement = { ...makeUnit("u"), displayName: "Replaced" };
    useSimStore.getState().spawnUnit(replacement);
    expect(useSimStore.getState().units.get("u")!.displayName).toBe("Replaced");
  });
});

// ─── destroyUnit ──────────────────────────────────────────────────────────────

describe("destroyUnit", () => {
  beforeEach(resetStore);

  it("marks unit isActive=false", () => {
    useSimStore.getState().loadSnapshot([makeUnit("u1")], "S");
    useSimStore.getState().destroyUnit("u1");
    expect(useSimStore.getState().units.get("u1")!.status.isActive).toBe(false);
  });

  it("preserves all other unit fields on destroy", () => {
    useSimStore.getState().loadSnapshot([makeUnit("u1")], "S");
    useSimStore.getState().destroyUnit("u1");
    const u = useSimStore.getState().units.get("u1")!;
    expect(u.displayName).toBe("Unit-u1");
    expect(u.side).toBe("Blue");
  });

  it("does not throw for unknown unit id", () => {
    expect(() => useSimStore.getState().destroyUnit("ghost")).not.toThrow();
  });

  it("unit stays in Map after destroy (needed for Cesium cleanup)", () => {
    useSimStore.getState().loadSnapshot([makeUnit("u1")], "S");
    useSimStore.getState().destroyUnit("u1");
    expect(useSimStore.getState().units.has("u1")).toBe(true);
  });
});

// ─── setScenarioState ─────────────────────────────────────────────────────────

describe("setScenarioState", () => {
  beforeEach(resetStore);

  it("updates scenarioState and timeScale", () => {
    useSimStore.getState().setScenarioState("running", 2.0);
    const s = useSimStore.getState();
    expect(s.scenarioState).toBe("running");
    expect(s.timeScale).toBe(2.0);
  });

  it("accepts paused state", () => {
    useSimStore.getState().setScenarioState("paused", 1.0);
    expect(useSimStore.getState().scenarioState).toBe("paused");
  });
});

// ─── setSimTime ───────────────────────────────────────────────────────────────

describe("setSimTime", () => {
  beforeEach(resetStore);

  it("updates simSeconds and tickNumber", () => {
    useSimStore.getState().setSimTime(3600, 42);
    const s = useSimStore.getState();
    expect(s.simSeconds).toBe(3600);
    expect(s.tickNumber).toBe(42);
  });
});

// ─── appendEventLog ───────────────────────────────────────────────────────────

describe("appendEventLog", () => {
  beforeEach(resetStore);

  it("appends entries to the log", () => {
    const store = useSimStore.getState();
    store.appendEventLog({ id: "1", text: "Hello", category: "combat", unitId: "u1", side: "Blue", simSeconds: 10 });
    store.appendEventLog({ id: "2", text: "World", category: "scenario", unitId: "", side: "", simSeconds: 20 });
    expect(useSimStore.getState().eventLog).toHaveLength(2);
    expect(useSimStore.getState().eventLog[1].text).toBe("World");
  });

  it("caps log at 200 entries, dropping oldest", () => {
    const store = useSimStore.getState();
    for (let i = 0; i < 210; i++) {
      store.appendEventLog({ id: String(i), text: `msg-${i}`, category: "scenario", unitId: "", side: "", simSeconds: i });
    }
    const log = useSimStore.getState().eventLog;
    expect(log).toHaveLength(200);
    // Oldest entries should have been dropped.
    expect(log[0].text).toBe("msg-10");
    expect(log[199].text).toBe("msg-209");
  });
});

// ─── selectUnit ───────────────────────────────────────────────────────────────

describe("selectUnit", () => {
  beforeEach(resetStore);

  it("sets selectedUnitId", () => {
    useSimStore.getState().selectUnit("u1");
    expect(useSimStore.getState().selectedUnitId).toBe("u1");
  });

  it("clears selectedUnitId when null passed", () => {
    useSimStore.getState().selectUnit("u1");
    useSimStore.getState().selectUnit(null);
    expect(useSimStore.getState().selectedUnitId).toBeNull();
  });
});

// ─── setActiveView ────────────────────────────────────────────────────────────

describe("setActiveView", () => {
  beforeEach(resetStore);

  it("switches to blue view", () => {
    useSimStore.getState().setActiveView("blue");
    expect(useSimStore.getState().activeView).toBe("blue");
  });

  it("switches back to debug view", () => {
    useSimStore.getState().setActiveView("red");
    useSimStore.getState().setActiveView("debug");
    expect(useSimStore.getState().activeView).toBe("debug");
  });
});

// ─── setDetections ────────────────────────────────────────────────────────────

describe("setDetections", () => {
  beforeEach(resetStore);

  it("stores detected IDs as a Set for a given side", () => {
    useSimStore.getState().setDetections("Blue", ["red1", "red2"]);
    const d = useSimStore.getState().detections.get("Blue")!;
    expect(d.has("red1")).toBe(true);
    expect(d.has("red2")).toBe(true);
    expect(d.size).toBe(2);
  });

  it("replaces previous detections for the same side", () => {
    useSimStore.getState().setDetections("Blue", ["r1", "r2"]);
    useSimStore.getState().setDetections("Blue", ["r3"]);
    const d = useSimStore.getState().detections.get("Blue")!;
    expect(d.has("r1")).toBe(false);
    expect(d.has("r3")).toBe(true);
    expect(d.size).toBe(1);
  });

  it("clears detections for a side when empty array passed", () => {
    useSimStore.getState().setDetections("Red", ["b1"]);
    useSimStore.getState().setDetections("Red", []);
    expect(useSimStore.getState().detections.get("Red")!.size).toBe(0);
  });

  it("tracks multiple sides independently", () => {
    useSimStore.getState().setDetections("Blue", ["r1"]);
    useSimStore.getState().setDetections("Red", ["b1", "b2"]);
    expect(useSimStore.getState().detections.get("Blue")!.size).toBe(1);
    expect(useSimStore.getState().detections.get("Red")!.size).toBe(2);
  });
});
