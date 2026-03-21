import { describe, expect, it } from "vitest";
import { WeaponEffectType } from "@proto/engine/v1/weapon_pb";
import { filterValidLiveTargets } from "./tasking";
import type { Unit, WeaponDef } from "../store/simStore";

function makeUnit(overrides: Partial<Unit>): Unit {
  return {
    id: "unit-1",
    displayName: "Unit",
    fullName: "Unit",
    teamId: "USA",
    coalitionId: "COALITION_WEST",
    natoPendingSymbol: "",
    definitionId: "generic",
    damageState: 1,
    position: { lat: 0, lon: 0, altMsl: 0, heading: 0, speed: 0 },
    status: {
      personnelStrength: 1,
      equipmentStrength: 1,
      combatEffectiveness: 1,
      fuelLevelLiters: 0,
      morale: 1,
      fatigue: 0,
      isActive: true,
      suppressed: false,
      disrupted: false,
      routing: false,
    },
    weapons: [],
    ...overrides,
  };
}

describe("filterValidLiveTargets", () => {
  it("allows an Ohio SSGN with Tomahawks to target hostile land and sea units", () => {
    const ohio = makeUnit({
      id: "usa-ohio-1",
      definitionId: "ohio-ssgn",
      teamId: "USA",
      coalitionId: "COALITION_WEST",
      weapons: [
        { weaponId: "ssm-tomahawk", currentQty: 40, maxQty: 40 },
        { weaponId: "torp-mk48", currentQty: 12, maxQty: 12 },
      ],
    });
    const hostileAirbase = makeUnit({
      id: "irn-airbase-1",
      definitionId: "iran-strategic-airbase",
      teamId: "IRN",
      coalitionId: "COALITION_IRAN",
    });
    const hostileShip = makeUnit({
      id: "irn-ship-1",
      definitionId: "jamaran-frigate",
      teamId: "IRN",
      coalitionId: "COALITION_IRAN",
    });

    const units = new Map<string, Unit>([
      [ohio.id, ohio],
      [hostileAirbase.id, hostileAirbase],
      [hostileShip.id, hostileShip],
    ]);
    const weaponDefs = new Map<string, WeaponDef>([
      ["ssm-tomahawk", {
        id: "ssm-tomahawk",
        name: "Tomahawk",
        description: "Land attack missile",
        domainTargets: [1, 3],
        speedMps: 245,
        rangeM: 1_600_000,
        probabilityOfHit: 0.84,
        effectType: WeaponEffectType.LAND_STRIKE,
      }],
      ["torp-mk48", {
        id: "torp-mk48",
        name: "Mk 48",
        description: "Torpedo",
        domainTargets: [3, 4],
        speedMps: 28,
        rangeM: 55_000,
        probabilityOfHit: 0.8,
        effectType: WeaponEffectType.TORPEDO,
      }],
    ]);
    const definitionMap = new Map([
      ["iran-strategic-airbase", {
        domain: 1,
        targetClass: "runway",
        stationary: true,
        assetClass: "airbase",
      }],
      ["jamaran-frigate", {
        domain: 3,
        stationary: false,
        assetClass: "combat_unit",
      }],
    ]);

    const targets = filterValidLiveTargets(ohio, units, weaponDefs, definitionMap);
    expect(targets.map((target) => target.id).sort()).toEqual(["irn-airbase-1", "irn-ship-1"]);
  });

  it("still allows sea targets when target_class metadata is missing", () => {
    const ohio = makeUnit({
      id: "usa-ohio-1",
      definitionId: "ohio-ssgn",
      teamId: "USA",
      coalitionId: "COALITION_WEST",
      weapons: [
        { weaponId: "torp-mk48", currentQty: 12, maxQty: 12 },
      ],
    });
    const hostileShip = makeUnit({
      id: "irn-ship-1",
      definitionId: "jamaran-frigate",
      teamId: "IRN",
      coalitionId: "COALITION_IRAN",
    });
    const units = new Map<string, Unit>([
      [ohio.id, ohio],
      [hostileShip.id, hostileShip],
    ]);
    const weaponDefs = new Map<string, WeaponDef>([
      ["torp-mk48", {
        id: "torp-mk48",
        name: "Mk 48",
        description: "Torpedo",
        domainTargets: [3, 4],
        speedMps: 28,
        rangeM: 55_000,
        probabilityOfHit: 0.8,
        effectType: WeaponEffectType.TORPEDO,
      }],
    ]);
    const definitionMap = new Map([
      ["jamaran-frigate", {
        domain: 3,
        stationary: false,
        assetClass: "combat_unit",
      }],
    ]);

    const targets = filterValidLiveTargets(ohio, units, weaponDefs, definitionMap);
    expect(targets.map((target) => target.id)).toEqual(["irn-ship-1"]);
  });
});
