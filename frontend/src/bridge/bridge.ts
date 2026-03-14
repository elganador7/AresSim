/**
 * bridge.ts
 *
 * Wails event bridge: subscribes to sim:* events emitted by the Go backend,
 * decodes base64-encoded proto binaries, and dispatches into the Zustand store.
 */

import { EventsOn } from "../../wailsjs/runtime/runtime";
import { useSimStore, Unit, MoveOrder, EventLogEntry, WeaponState, WeaponDef, Munition } from "../store/simStore";
import { fromBinary } from "@bufbuild/protobuf";

import {
  FullStateSnapshotSchema,
  BatchUnitUpdateSchema,
  ScenarioStateEventSchema,
  UnitSpawnedEventSchema,
  UnitDestroyedEventSchema,
  NarrativeEventSchema,
  DetectionUpdateSchema,
  MunitionUpdateSchema,
} from "@proto/engine/v1/events_pb";
import type {
  FullStateSnapshot,
  UnitSpawnedEvent,
  UnitDestroyedEvent,
  NarrativeEvent,
  UnitDelta,
} from "@proto/engine/v1/events_pb";
import type { Unit as ProtoUnit } from "@proto/engine/v1/unit_pb";
import type { WeaponDefinition as ProtoWeaponDef } from "@proto/engine/v1/weapon_pb";
import type { MoveOrder as ProtoMoveOrder } from "@proto/engine/v1/common_pb";
import type { OperationalStatus } from "@proto/engine/v1/status_pb";
import { ScenarioPlayState } from "@proto/engine/v1/events_pb";

// ─── HELPERS ─────────────────────────────────────────────────────────────────

function b64ToBytes(b64: string): Uint8Array {
  const binary = atob(b64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

// ─── PROTO → STORE CONVERTERS ─────────────────────────────────────────────────

function protoMoveOrderToStore(m: ProtoMoveOrder | undefined): MoveOrder | undefined {
  if (!m) return undefined;
  return {
    waypoints: m.waypoints.map((wp) => ({
      lat: wp.lat,
      lon: wp.lon,
      altMsl: wp.altMsl,
    })),
  };
}

function protoStatusToStore(s?: OperationalStatus): Unit["status"] {
  return {
    personnelStrength: s?.personnelStrength ?? 1,
    equipmentStrength: s?.equipmentStrength ?? 1,
    combatEffectiveness: s?.combatEffectiveness ?? 1,
    fuelLevelLiters: s?.fuelLevelLiters ?? 0,
    morale: s?.morale ?? 1,
    fatigue: s?.fatigue ?? 0,
    isActive: s?.isActive ?? true,
    suppressed: false,
    disrupted: false,
    routing: false,
  };
}

export function protoUnitToStore(u: ProtoUnit): Unit {
  return {
    id: u.id,
    displayName: u.displayName,
    fullName: u.fullName,
    side: u.side,
    natoPendingSymbol: u.natoSymbolSidc,
    definitionId: u.definitionId,
    position: {
      lat: u.position?.lat ?? 0,
      lon: u.position?.lon ?? 0,
      altMsl: u.position?.altMsl ?? 0,
      heading: u.position?.heading ?? 0,
      speed: u.position?.speed ?? 0,
    },
    status: protoStatusToStore(u.status),
    parentUnitId: u.parentUnitId || undefined,
    moveOrder: protoMoveOrderToStore(u.moveOrder),
    weapons: (u.weapons ?? []).map((w): WeaponState => ({
      weaponId: w.weaponId,
      currentQty: w.currentQty,
      maxQty: w.maxQty,
    })),
  };
}

function protoWeaponDefToStore(w: ProtoWeaponDef): WeaponDef {
  return {
    id: w.id,
    name: w.name,
    description: w.description,
    domainTargets: w.domainTargets,
    speedMps: w.speedMps,
    rangeM: w.rangeM,
    probabilityOfHit: w.probabilityOfHit,
  };
}

function applyDelta(delta: UnitDelta): void {
  const store = useSimStore.getState();
  const patch: Partial<Unit> = {};

  if (delta.position) {
    patch.position = {
      lat: delta.position.lat,
      lon: delta.position.lon,
      altMsl: delta.position.altMsl,
      heading: delta.position.heading,
      speed: delta.position.speed,
    };
  }
  if (delta.status) {
    patch.status = protoStatusToStore(delta.status);
  }
  // moveOrder present in delta = update (empty waypoints = clear order)
  if (delta.moveOrder !== undefined) {
    patch.moveOrder = delta.moveOrder.waypoints.length > 0
      ? protoMoveOrderToStore(delta.moveOrder)
      : undefined;
  }
  // weapon states present = ammo changed (adjudicator fired a weapon)
  if (delta.weapons && delta.weapons.length > 0) {
    patch.weapons = delta.weapons.map((w): WeaponState => ({
      weaponId: w.weaponId,
      currentQty: w.currentQty,
      maxQty: w.maxQty,
    }));
  }

  store.applyUnitDelta(delta.unitId, patch);
}

export function protoEventToLogEntry(e: NarrativeEvent, simSeconds: number): EventLogEntry {
  return {
    id: crypto.randomUUID(),
    text: e.text,
    category: e.category,
    unitId: e.unitId,
    side: e.side,
    simSeconds,
  };
}

// ─── EVENT SUBSCRIPTIONS ─────────────────────────────────────────────────────

export function initBridge(): void {
  const store = useSimStore.getState();

  // ── Full state snapshot ────────────────────────────────────────────────────
  EventsOn("sim:full_state_snapshot", (b64: string) => {
    let snap: FullStateSnapshot;
    try {
      snap = fromBinary(FullStateSnapshotSchema, b64ToBytes(b64));
    } catch (e) {
      console.error("[bridge] full_state_snapshot decode failed", e);
      return;
    }
    const units = snap.units.map(protoUnitToStore);
    const weaponDefs = snap.weaponDefinitions.map(protoWeaponDefToStore);
    store.loadSnapshot(units, snap.scenarioName, weaponDefs);
    if (snap.simTime) {
      store.setSimTime(snap.simTime.secondsElapsed, Number(snap.simTime.tickNumber));
    }
    console.log(`[bridge] snapshot loaded: ${units.length} units, ${weaponDefs.length} weapon defs, scenario="${snap.scenarioName}"`);
  });

  // ── Per-tick batch update ──────────────────────────────────────────────────
  EventsOn("sim:batch_update", (b64: string) => {
    let msg;
    try {
      msg = fromBinary(BatchUnitUpdateSchema, b64ToBytes(b64));
    } catch (e) {
      console.error("[bridge] batch_update decode failed", e);
      return;
    }
    for (const delta of msg.deltas) {
      applyDelta(delta);
    }
    if (msg.simTime) {
      store.setSimTime(msg.simTime.secondsElapsed, Number(msg.simTime.tickNumber));
    }
  });

  // ── Unit lifecycle ─────────────────────────────────────────────────────────
  EventsOn("sim:unit_spawned", (b64: string) => {
    let ev: UnitSpawnedEvent;
    try {
      ev = fromBinary(UnitSpawnedEventSchema, b64ToBytes(b64));
    } catch (e) {
      console.error("[bridge] unit_spawned decode failed", e);
      return;
    }
    if (ev.unit) store.spawnUnit(protoUnitToStore(ev.unit));
  });

  EventsOn("sim:unit_destroyed", (b64: string) => {
    let ev: UnitDestroyedEvent;
    try {
      ev = fromBinary(UnitDestroyedEventSchema, b64ToBytes(b64));
    } catch (e) {
      console.error("[bridge] unit_destroyed decode failed", e);
      return;
    }
    store.destroyUnit(ev.unitId);
  });

  // ── Scenario state ─────────────────────────────────────────────────────────
  EventsOn("sim:scenario_state", (b64: string) => {
    let ev;
    try {
      ev = fromBinary(ScenarioStateEventSchema, b64ToBytes(b64));
    } catch (e) {
      console.error("[bridge] scenario_state decode failed", e);
      return;
    }
    const stateMap = {
      [ScenarioPlayState.SCENARIO_UNSPECIFIED]: "idle",
      [ScenarioPlayState.SCENARIO_PAUSED]: "paused",
      [ScenarioPlayState.SCENARIO_RUNNING]: "running",
      [ScenarioPlayState.SCENARIO_ENDED]: "ended",
    } as const;
    store.setScenarioState(stateMap[ev.state] ?? "idle", ev.timeScale);
  });

  // ── Narrative event log ────────────────────────────────────────────────────
  EventsOn("sim:narrative", (b64: string) => {
    let ev: NarrativeEvent;
    try {
      ev = fromBinary(NarrativeEventSchema, b64ToBytes(b64));
    } catch (e) {
      console.error("[bridge] narrative decode failed", e);
      return;
    }
    const { simSeconds } = useSimStore.getState();
    store.appendEventLog(protoEventToLogEntry(ev, simSeconds));
  });

  // ── Sensor detection updates ───────────────────────────────────────────
  EventsOn("sim:detection_update", (b64: string) => {
    let ev;
    try {
      ev = fromBinary(DetectionUpdateSchema, b64ToBytes(b64));
    } catch (e) {
      console.error("[bridge] detection_update decode failed", e);
      return;
    }
    store.setDetections(ev.detectingSide, ev.detectedUnitIds);
    store.setMunitionDetections(ev.detectingSide, ev.detectedMunitionIds);
  });

  // ── In-flight munition update ──────────────────────────────────────────
  EventsOn("sim:munition_update", (b64: string) => {
    let ev;
    try {
      ev = fromBinary(MunitionUpdateSchema, b64ToBytes(b64));
    } catch (e) {
      console.error("[bridge] munition_update decode failed", e);
      return;
    }
    const munitions: Munition[] = ev.munitions.map((m) => ({
      id: m.id,
      weaponId: m.weaponId,
      shooterId: m.shooterId,
      lat: m.position?.lat ?? 0,
      lon: m.position?.lon ?? 0,
    }));
    store.setMunitions(munitions);
  });

  console.log("[bridge] event listeners registered");
}
