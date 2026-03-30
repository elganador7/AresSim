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
import type { SimStore } from "./simTypes";
export type {
  CountryRelationship,
  DetectionContact,
  EventLogEntry,
  ExplosionFx,
  MapCommandMode,
  MoveOrder,
  Munition,
  OilCommodityQuantity,
  OilEdge,
  OilGraph,
  OilNode,
  OilProductOutput,
  OilRoutePoint,
  OilSourceRef,
  PathViolationPreview,
  Position,
  ScenarioState,
  TeamScore,
  Unit,
  UnitStatus,
  Waypoint,
  WeaponDef,
  WeaponState,
} from "./simTypes";

const MAX_EVENT_LOG = 200;

export const useSimStore = create<SimStore>((set) => ({
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

  loadSnapshot: (units, scenarioName, weaponDefs, relationships, scores) =>
    // Replaces the entire units Map reference. Zustand notifies all subscribers
    // because the reference changed, so CesiumGlobe's subscription fires and
    // does a full syncUnits pass. This is correct: a snapshot is a full rebuild.
    set((state) => ({
      scenarioName,
      scenarioState: "paused",
      simSeconds: 0,
      tickNumber: 0,
      relationships: relationships ?? state.relationships,
      scores: scores ?? state.scores,
      units: new Map(units.map((u) => [u.id, u])),
      selectedUnitId: null,
      selectedTargetId: null,
      selectedOilNodeId: null,
      selectedOilEdgeId: null,
      mapCommandMode: { type: "none", unitId: null },
      selectedRoutePreview: null,
      selectedStrikePreview: null,
      munitions: new Map(),
      explosions: new Map(),
      detections: new Map(),
      detectionContacts: new Map(),
      eventLog: [],
      activeView: state.scenarioName === scenarioName ? state.activeView : "debug",
      weaponDefs: weaponDefs
        ? new Map(weaponDefs.map((d) => [d.id, d]))
        : state.weaponDefs,
      humanControlledTeam: state.scenarioName === scenarioName ? state.humanControlledTeam : "",
    })),

  applyUnitDelta: (id, delta) =>
    set((state) => {
      const existing = state.units.get(id);
      if (!existing) return {};
      const updated = new Map(state.units);
      const merged = { ...existing, ...delta };
      // When a weapons delta arrives, merge by weaponId rather than replacing
      // the entire array, so un-fired weapons retain their previous quantities.
      if (delta.weapons && delta.weapons.length > 0) {
        const weaponMap = new Map(existing.weapons.map((w) => [w.weaponId, w]));
        for (const w of delta.weapons) {
          weaponMap.set(w.weaponId, w);
        }
        merged.weapons = Array.from(weaponMap.values());
      }
      updated.set(id, merged);
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
        updated.set(id, { ...unit, damageState: 4, status: { ...unit.status, isActive: false } });
      }
      return { units: updated };
    }),

  setScenarioState: (scenarioState, timeScale) =>
    set({ scenarioState, timeScale }),

  setSimTime: (simSeconds, tickNumber) =>
    set({ simSeconds, tickNumber }),

  setRelationships: (relationships) => set({ relationships }),
  setScores: (scores) => set({ scores }),

  appendEventLog: (entry) =>
    set((state) => {
      const log = [...state.eventLog, entry];
      return { eventLog: log.length > MAX_EVENT_LOG ? log.slice(-MAX_EVENT_LOG) : log };
    }),

  selectUnit: (selectedUnitId) =>
    set((state) => ({
      selectedUnitId,
      selectedTargetId: selectedUnitId ? null : state.selectedTargetId,
      selectedOilNodeId: selectedUnitId ? null : state.selectedOilNodeId,
      selectedOilEdgeId: selectedUnitId ? null : state.selectedOilEdgeId,
    })),
  selectTarget: (selectedTargetId) =>
    set((state) => ({
      selectedTargetId,
      selectedUnitId: selectedTargetId ? null : state.selectedUnitId,
      selectedOilNodeId: selectedTargetId ? null : state.selectedOilNodeId,
      selectedOilEdgeId: selectedTargetId ? null : state.selectedOilEdgeId,
      mapCommandMode: selectedTargetId ? { type: "none", unitId: null } : state.mapCommandMode,
    })),
  startRouteEdit: (unitId) => set({ mapCommandMode: unitId ? { type: "route", unitId } : { type: "none", unitId: null } }),
  clearMapCommandMode: () => set({ mapCommandMode: { type: "none", unitId: null } }),
  setSelectedRoutePreview: (selectedRoutePreview) => set({ selectedRoutePreview }),
  setSelectedStrikePreview: (selectedStrikePreview) => set({ selectedStrikePreview }),
  setActiveView: (activeView) => set({ activeView }),
  setHumanControlledTeam: (humanControlledTeam) => set({ humanControlledTeam }),
  setOilGraph: (oilGraph) => set({ oilGraph, oilLoadError: null }),
  setOilLoadError: (oilLoadError) => set({ oilLoadError }),
  setOilLayerVisible: (oilLayerVisible) => set({ oilLayerVisible }),
  requestOilFocus: () => set((state) => ({ oilFocusToken: state.oilFocusToken + 1 })),
  selectOilNode: (selectedOilNodeId) =>
    set({
      selectedOilNodeId,
      selectedOilEdgeId: null,
      selectedUnitId: null,
      selectedTargetId: null,
      mapCommandMode: { type: "none", unitId: null },
    }),
  selectOilEdge: (selectedOilEdgeId) =>
    set({
      selectedOilEdgeId,
      selectedOilNodeId: null,
      selectedUnitId: null,
      selectedTargetId: null,
      mapCommandMode: { type: "none", unitId: null },
    }),

  setDetections: (teamId, ids, contacts = []) =>
    set((state) => {
      const updated = new Map(state.detections);
      updated.set(teamId, new Set(ids));
      const updatedContacts = new Map(state.detectionContacts);
      updatedContacts.set(teamId, new Map(contacts.map((contact) => [contact.unitId, contact])));
      return { detections: updated, detectionContacts: updatedContacts };
    }),

  setMunitions: (munitions) =>
    set({ munitions: new Map(munitions.map((m) => [m.id, m])) }),

  addExplosion: (explosion) =>
    set((state) => {
      const updated = new Map(state.explosions);
      updated.set(explosion.id, explosion);
      return { explosions: updated };
    }),

  removeExplosion: (id) =>
    set((state) => {
      if (!state.explosions.has(id)) return {};
      const updated = new Map(state.explosions);
      updated.delete(id);
      return { explosions: updated };
    }),

  setMunitionDetections: (teamId, ids) =>
    set((state) => {
      const updated = new Map(state.munitionDetections);
      updated.set(teamId, new Set(ids));
      return { munitionDetections: updated };
    }),
}));
