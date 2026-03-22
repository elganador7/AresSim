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
  it("prevents maritime shooters from targeting hostile land units", () => {
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
      ["ohio-ssgn", {
        domain: 4,
        stationary: false,
        assetClass: "combat_unit",
      }],
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
    expect(targets.map((target) => target.id).sort()).toEqual(["irn-ship-1"]);
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
      ["ohio-ssgn", {
        domain: 4,
        stationary: false,
        assetClass: "combat_unit",
      }],
      ["jamaran-frigate", {
        domain: 3,
        stationary: false,
        assetClass: "combat_unit",
      }],
    ]);

    const targets = filterValidLiveTargets(ohio, units, weaponDefs, definitionMap);
    expect(targets.map((target) => target.id)).toEqual(["irn-ship-1"]);
  });

  it("requires a current track for mobile live targets", () => {
    const fighter = makeUnit({
      id: "usa-f35-1",
      definitionId: "f35",
      teamId: "USA",
      coalitionId: "COALITION_WEST",
      weapons: [{ weaponId: "aam", currentQty: 4, maxQty: 4 }],
    });
    const hostileFighter = makeUnit({
      id: "irn-f14-1",
      definitionId: "f14",
      teamId: "IRN",
      coalitionId: "COALITION_IRAN",
      position: { lat: 1, lon: 1, altMsl: 8_000, heading: 0, speed: 250 },
    });
    const units = new Map<string, Unit>([
      [fighter.id, fighter],
      [hostileFighter.id, hostileFighter],
    ]);
    const weaponDefs = new Map<string, WeaponDef>([
      ["aam", {
        id: "aam",
        name: "AAM",
        description: "air to air",
        domainTargets: [2],
        speedMps: 800,
        rangeM: 100_000,
        probabilityOfHit: 0.7,
        effectType: WeaponEffectType.ANTI_AIR,
      }],
    ]);
    const definitionMap = new Map([
      ["f35", { domain: 2, stationary: false, assetClass: "combat_unit", targetClass: "aircraft" }],
      ["f14", { domain: 2, stationary: false, assetClass: "combat_unit", targetClass: "aircraft" }],
    ]);

    expect(filterValidLiveTargets(fighter, units, weaponDefs, definitionMap, new Set()).map((target) => target.id)).toEqual([]);
    expect(filterValidLiveTargets(fighter, units, weaponDefs, definitionMap, new Set(["irn-f14-1"])).map((target) => target.id)).toEqual(["irn-f14-1"]);
  });

  it("allows fixed strategic targets even without a current track", () => {
    const bomber = makeUnit({
      id: "usa-b1-1",
      definitionId: "b1",
      teamId: "USA",
      coalitionId: "COALITION_WEST",
      weapons: [{ weaponId: "cruise", currentQty: 8, maxQty: 8 }],
    });
    const hostileAirbase = makeUnit({
      id: "irn-airbase-1",
      definitionId: "iran-strategic-airbase",
      teamId: "IRN",
      coalitionId: "COALITION_IRAN",
    });
    const units = new Map<string, Unit>([
      [bomber.id, bomber],
      [hostileAirbase.id, hostileAirbase],
    ]);
    const weaponDefs = new Map<string, WeaponDef>([
      ["cruise", {
        id: "cruise",
        name: "Cruise",
        description: "land attack",
        domainTargets: [1],
        speedMps: 240,
        rangeM: 1_500_000,
        probabilityOfHit: 0.8,
        effectType: WeaponEffectType.LAND_STRIKE,
      }],
    ]);
    const definitionMap = new Map([
      ["b1", { domain: 2, stationary: false, assetClass: "combat_unit" }],
      ["iran-strategic-airbase", { domain: 1, stationary: true, assetClass: "airbase", targetClass: "runway" }],
    ]);

    expect(filterValidLiveTargets(bomber, units, weaponDefs, definitionMap, new Set()).map((target) => target.id)).toEqual(["irn-airbase-1"]);
  });

  it("does not treat a naval combatant as a preplanned fixed target", () => {
    const bomber = makeUnit({
      id: "usa-b1-1",
      definitionId: "b1",
      teamId: "USA",
      coalitionId: "COALITION_WEST",
      weapons: [{ weaponId: "cruise", currentQty: 8, maxQty: 8 }],
    });
    const hostileDestroyer = makeUnit({
      id: "irn-ship-1",
      definitionId: "destroyer",
      teamId: "IRN",
      coalitionId: "COALITION_IRAN",
    });
    const units = new Map<string, Unit>([
      [bomber.id, bomber],
      [hostileDestroyer.id, hostileDestroyer],
    ]);
    const weaponDefs = new Map<string, WeaponDef>([
      ["cruise", {
        id: "cruise",
        name: "Cruise",
        description: "land attack",
        domainTargets: [1, 3],
        speedMps: 240,
        rangeM: 1_500_000,
        probabilityOfHit: 0.8,
        effectType: WeaponEffectType.LAND_STRIKE,
      }],
    ]);
    const definitionMap = new Map([
      ["b1", { domain: 2, stationary: false, assetClass: "combat_unit" }],
      // Even if bad metadata marks it stationary, a destroyer should not be treated like a fixed installation.
      ["destroyer", { domain: 3, stationary: true, assetClass: "combat_unit", targetClass: "surface_warship" }],
    ]);

    expect(filterValidLiveTargets(bomber, units, weaponDefs, definitionMap, new Set()).map((target) => target.id)).toEqual([]);
  });

  it("supports record-prefixed definition ids in live tasking", () => {
    const fighter = makeUnit({
      id: "usa-f35-1",
      definitionId: "unit_definition:f35",
      teamId: "USA",
      coalitionId: "COALITION_WEST",
      weapons: [{ weaponId: "aam", currentQty: 4, maxQty: 4 }],
    });
    const hostileFighter = makeUnit({
      id: "irn-f14-1",
      definitionId: "unit_definition:f14",
      teamId: "IRN",
      coalitionId: "COALITION_IRAN",
      position: { lat: 1, lon: 1, altMsl: 8_000, heading: 0, speed: 250 },
    });
    const units = new Map<string, Unit>([
      [fighter.id, fighter],
      [hostileFighter.id, hostileFighter],
    ]);
    const weaponDefs = new Map<string, WeaponDef>([
      ["aam", {
        id: "aam",
        name: "AAM",
        description: "air to air",
        domainTargets: [2],
        speedMps: 800,
        rangeM: 100_000,
        probabilityOfHit: 0.7,
        effectType: WeaponEffectType.ANTI_AIR,
      }],
    ]);
    const definitionMap = new Map([
      ["f35", { domain: 2, stationary: false, assetClass: "combat_unit", targetClass: "aircraft" }],
      ["f14", { domain: 2, stationary: false, assetClass: "combat_unit", targetClass: "aircraft" }],
    ]);

    expect(filterValidLiveTargets(fighter, units, weaponDefs, definitionMap, new Set(["irn-f14-1"])).map((target) => target.id)).toEqual(["irn-f14-1"]);
  });
});
