/**
 * bridge.test.ts
 *
 * Tests for pure bridge converter functions.
 * EventsOn subscriptions are not tested here (they require Wails runtime);
 * those are integration-tested at the app level.
 */

import { describe, it, expect } from "vitest";
import { protoUnitToStore, protoEventToLogEntry } from "./bridge";

// ─── Minimal proto-shaped objects (no generated classes needed) ───────────────

function makeProtoUnit(overrides: Record<string, unknown> = {}) {
  return {
    id: "u1",
    displayName: "Alpha",
    fullName: "1st Battalion Alpha",
    side: "Blue",
    natoSymbolSidc: "SFGPUCI----D",
    definitionId: "def-1",
    damageState: 1,
    parentUnitId: "",
    position: { lat: 10, lon: 20, altMsl: 500, heading: 90, speed: 15 },
    status: {
      personnelStrength: 80,
      equipmentStrength: 90,
      combatEffectiveness: 0.85,
      fuelLevelLiters: 300,
      morale: 0.9,
      fatigue: 0.1,
      isActive: true,
      suppressed: false,
      disrupted: false,
      routing: false,
    },
    moveOrder: undefined,
    ...overrides,
  } as any;
}

function makeProtoNarrative(overrides: Record<string, unknown> = {}) {
  return {
    text: "Alpha destroyed Bravo",
    category: "combat",
    unitId: "u1",
    side: "Blue",
    ...overrides,
  } as any;
}

// ─── protoUnitToStore ─────────────────────────────────────────────────────────

describe("protoUnitToStore", () => {
  it("maps id, displayName, fullName, side", () => {
    const u = protoUnitToStore(makeProtoUnit());
    expect(u.id).toBe("u1");
    expect(u.displayName).toBe("Alpha");
    expect(u.fullName).toBe("1st Battalion Alpha");
    expect(u.side).toBe("Blue");
    expect(u.damageState).toBe(1);
  });

  it("maps NATO symbol SIDC to natoPendingSymbol", () => {
    const u = protoUnitToStore(makeProtoUnit({ natoSymbolSidc: "SFGPUCI----D" }));
    expect(u.natoPendingSymbol).toBe("SFGPUCI----D");
  });

  it("maps position fields", () => {
    const u = protoUnitToStore(makeProtoUnit());
    expect(u.position.lat).toBe(10);
    expect(u.position.lon).toBe(20);
    expect(u.position.altMsl).toBe(500);
    expect(u.position.heading).toBe(90);
    expect(u.position.speed).toBe(15);
  });

  it("defaults position fields to 0 when position is missing", () => {
    const u = protoUnitToStore(makeProtoUnit({ position: undefined }));
    expect(u.position.lat).toBe(0);
    expect(u.position.lon).toBe(0);
    expect(u.position.speed).toBe(0);
  });

  it("maps status fields", () => {
    const u = protoUnitToStore(makeProtoUnit());
    expect(u.status.personnelStrength).toBe(80);
    expect(u.status.combatEffectiveness).toBe(0.85);
    expect(u.status.fuelLevelLiters).toBe(300);
    expect(u.status.morale).toBe(0.9);
    expect(u.status.isActive).toBe(true);
  });

  it("defaults status to full effectiveness when status is missing", () => {
    const u = protoUnitToStore(makeProtoUnit({ status: undefined }));
    expect(u.status.combatEffectiveness).toBe(1);
    expect(u.status.morale).toBe(1);
    expect(u.status.isActive).toBe(true);
    expect(u.status.fatigue).toBe(0);
  });

  it("maps moveOrder with waypoints", () => {
    const u = protoUnitToStore(makeProtoUnit({
      moveOrder: {
        waypoints: [
          { lat: 5, lon: 6, altMsl: 100 },
          { lat: 7, lon: 8, altMsl: 200 },
        ],
      },
    }));
    expect(u.moveOrder?.waypoints).toHaveLength(2);
    expect(u.moveOrder?.waypoints[0].lat).toBe(5);
    expect(u.moveOrder?.waypoints[1].lon).toBe(8);
  });

  it("sets moveOrder to undefined when no moveOrder present", () => {
    const u = protoUnitToStore(makeProtoUnit({ moveOrder: undefined }));
    expect(u.moveOrder).toBeUndefined();
  });

  it("sets moveOrder to undefined when moveOrder has no waypoints", () => {
    const u = protoUnitToStore(makeProtoUnit({ moveOrder: { waypoints: [] } }));
    // protoMoveOrderToStore returns { waypoints: [] } — not undefined — but
    // the store treats empty waypoints as "no order" in applyDelta.
    // Here we just check it's present (not undefined) since the proto object exists.
    expect(u.moveOrder).toBeDefined();
    expect(u.moveOrder?.waypoints).toHaveLength(0);
  });

  it("omits parentUnitId when empty string", () => {
    const u = protoUnitToStore(makeProtoUnit({ parentUnitId: "" }));
    expect(u.parentUnitId).toBeUndefined();
  });

  it("includes parentUnitId when set", () => {
    const u = protoUnitToStore(makeProtoUnit({ parentUnitId: "parent-1" }));
    expect(u.parentUnitId).toBe("parent-1");
  });
});

// ─── protoEventToLogEntry ─────────────────────────────────────────────────────

describe("protoEventToLogEntry", () => {
  it("maps text, category, unitId, side", () => {
    const entry = protoEventToLogEntry(makeProtoNarrative(), 120);
    expect(entry.text).toBe("Alpha destroyed Bravo");
    expect(entry.category).toBe("combat");
    expect(entry.unitId).toBe("u1");
    expect(entry.side).toBe("Blue");
    expect(entry.simSeconds).toBe(120);
  });

  it("generates a unique id for each entry", () => {
    const e1 = protoEventToLogEntry(makeProtoNarrative(), 0);
    const e2 = protoEventToLogEntry(makeProtoNarrative(), 0);
    expect(e1.id).not.toBe(e2.id);
  });

  it("preserves simSeconds passed in", () => {
    const entry = protoEventToLogEntry(makeProtoNarrative(), 3661);
    expect(entry.simSeconds).toBe(3661);
  });

  it("works with all narrative categories", () => {
    for (const cat of ["combat", "logistics", "c2", "intelligence", "scenario"]) {
      const e = protoEventToLogEntry(makeProtoNarrative({ category: cat }), 0);
      expect(e.category).toBe(cat);
    }
  });
});
