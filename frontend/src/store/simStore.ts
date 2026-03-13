/**
 * simStore.ts
 *
 * Central Zustand store for simulation state. This is the single source of
 * truth for the frontend. The store is updated by the event bridge (bridge.ts)
 * and read by React components and the CesiumJS renderer.
 *
 * Design principle: CesiumJS does NOT subscribe to this store via React hooks.
 * Instead, the CesiumJS renderer calls store.subscribe() to get raw state
 * diffs and updates entities imperatively. This avoids triggering a React
 * re-render on every tick (which would be 10x/sec with 1000+ units).
 *
 * React components (HUD, event log, unit panel) use useSimStore() normally —
 * they only re-render when their specific slice changes.
 */

import { create } from "zustand";

// ─── TYPES ────────────────────────────────────────────────────────────────────
// These mirror the proto message shapes after JSON deserialization.
// The full proto types (from @proto/engine/v1/*_pb.ts) are used in bridge.ts
// for deserialization; the store uses plain objects for simpler rendering.

export interface Position {
  lat: number;
  lon: number;
  altMsl: number;
  heading: number;
  speed: number;
}

export interface UnitStatus {
  personnelStrength: number;
  equipmentStrength: number;
  combatEffectiveness: number;
  fuelLevelLiters: number;
  morale: number;
  fatigue: number;
  isActive: boolean;
  suppressed: boolean;
  disrupted: boolean;
  routing: boolean;
}

export interface Unit {
  id: string;
  displayName: string;
  fullName: string;
  side: string;  // "Blue" | "Red" | "Neutral"
  natoPendingSymbol: string;  // NATO APP-6D SIDC code
  domain: number;
  unitType: number;
  position: Position;
  status: UnitStatus;
  parentUnitId?: string;
}

export type ScenarioState = "idle" | "paused" | "running" | "ended";

export interface SimStore {
  // ── Scenario ────────────────────────────────────────────────────────────
  scenarioName: string;
  scenarioState: ScenarioState;
  timeScale: number;
  simSeconds: number;
  tickNumber: number;

  // ── Units ────────────────────────────────────────────────────────────────
  // Map keyed by unit ID for O(1) lookup and incremental delta merges.
  units: Map<string, Unit>;

  // ── Selection ────────────────────────────────────────────────────────────
  selectedUnitId: string | null;

  // ── Event log ────────────────────────────────────────────────────────────
  // Capped at 200 entries; oldest dropped when full.
  eventLog: EventLogEntry[];

  // ── Actions ──────────────────────────────────────────────────────────────
  // Called by bridge.ts on incoming sim events.
  loadSnapshot: (units: Unit[], scenarioName: string) => void;
  applyUnitDelta: (id: string, delta: Partial<Unit>) => void;
  spawnUnit: (unit: Unit) => void;
  destroyUnit: (id: string) => void;
  setScenarioState: (state: ScenarioState, timeScale: number) => void;
  setSimTime: (seconds: number, tick: number) => void;
  appendEventLog: (entry: EventLogEntry) => void;
  selectUnit: (id: string | null) => void;
}

export interface EventLogEntry {
  id: string;        // UUID from NarrativeEvent
  text: string;
  category: string;  // "combat" | "logistics" | "c2" | "intelligence" | "scenario"
  unitId: string;
  side: string;
  simSeconds: number;
}

const MAX_EVENT_LOG = 200;

export const useSimStore = create<SimStore>((set) => ({
  scenarioName: "",
  scenarioState: "idle",
  timeScale: 1.0,
  simSeconds: 0,
  tickNumber: 0,
  units: new Map(),
  selectedUnitId: null,
  eventLog: [],

  loadSnapshot: (units, scenarioName) =>
    set({
      scenarioName,
      scenarioState: "paused",
      simSeconds: 0,
      tickNumber: 0,
      units: new Map(units.map((u) => [u.id, u])),
      eventLog: [],
    }),

  applyUnitDelta: (id, delta) =>
    set((state) => {
      const existing = state.units.get(id);
      if (!existing) return {};
      const updated = new Map(state.units);
      updated.set(id, { ...existing, ...delta });
      return { units: updated };
    }),

  spawnUnit: (unit) =>
    set((state) => {
      const updated = new Map(state.units);
      updated.set(unit.id, unit);
      return { units: updated };
    }),

  destroyUnit: (id) =>
    set((state) => {
      const updated = new Map(state.units);
      const unit = updated.get(id);
      if (unit) {
        updated.set(id, { ...unit, status: { ...unit.status, isActive: false } });
      }
      return { units: updated };
    }),

  setScenarioState: (scenarioState, timeScale) =>
    set({ scenarioState, timeScale }),

  setSimTime: (simSeconds, tickNumber) =>
    set({ simSeconds, tickNumber }),

  appendEventLog: (entry) =>
    set((state) => {
      const log = [...state.eventLog, entry];
      return { eventLog: log.length > MAX_EVENT_LOG ? log.slice(-MAX_EVENT_LOG) : log };
    }),

  selectUnit: (selectedUnitId) => set({ selectedUnitId }),
}));
