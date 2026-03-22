import { beforeEach, describe, expect, it } from "vitest";
import { definitionInfoFor, isTrack, isVisible, normalizeDefinitionId, type DefInfo } from "./helpers";
import { useSimStore, type Unit } from "../../store/simStore";

function makeUnit(id: string, teamId: string): Unit {
  return {
    id,
    displayName: id,
    fullName: id,
    teamId,
    coalitionId: teamId === "USA" || teamId === "ISR" ? "BLUE" : "RED",
    natoPendingSymbol: "",
    definitionId: `${id}-def`,
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
  };
}

const defInfo: Record<string, DefInfo> = {
  "usa-def": { generalType: 0, detectionRangeM: 0, shortName: "USA", teamCode: "USA", stationary: false, assetClass: "combat_unit" },
  "isr-def": { generalType: 0, detectionRangeM: 0, shortName: "ISR", teamCode: "ISR", stationary: false, assetClass: "combat_unit" },
  "irn-def": { generalType: 0, detectionRangeM: 0, shortName: "IRN", teamCode: "IRN", stationary: false, assetClass: "combat_unit" },
  "irn-airbase-def": { generalType: 0, detectionRangeM: 0, shortName: "AB", teamCode: "IRN", stationary: true, assetClass: "airbase" },
};

describe("cesium visibility helpers", () => {
  beforeEach(() => {
    useSimStore.setState({
      relationships: [],
      humanControlledTeam: "",
      activeView: "debug",
    });
  });

  it("shows allied shared units in a nation view", () => {
    useSimStore.setState({
      relationships: [
        {
          fromCountry: "ISR",
          toCountry: "USA",
          shareIntel: true,
          airspaceTransitAllowed: false,
          airspaceStrikeAllowed: false,
          defensivePositioningAllowed: false,
          maritimeTransitAllowed: false,
          maritimeStrikeAllowed: false,
        },
      ],
    });
    const detections = new Map<string, Set<string>>();
    const israeliUnit = makeUnit("isr", "ISR");
    expect(isVisible(israeliUnit, "USA", detections, defInfo)).toBe(true);
    expect(isTrack(israeliUnit, "USA", defInfo)).toBe(false);
  });

  it("still requires detections for enemy tracks", () => {
    useSimStore.setState({
      relationships: [
        {
          fromCountry: "ISR",
          toCountry: "USA",
          shareIntel: true,
          airspaceTransitAllowed: false,
          airspaceStrikeAllowed: false,
          defensivePositioningAllowed: false,
          maritimeTransitAllowed: false,
          maritimeStrikeAllowed: false,
        },
      ],
    });
    const iranianUnit = makeUnit("irn", "IRN");
    const detections = new Map<string, Set<string>>();
    expect(isVisible(iranianUnit, "USA", detections, defInfo)).toBe(false);
    detections.set("USA", new Set(["irn"]));
    expect(isVisible(iranianUnit, "USA", detections, defInfo)).toBe(true);
    expect(isTrack(iranianUnit, "USA", defInfo)).toBe(true);
  });

  it("always shows stationary fixed sites to all players", () => {
    const fixedSite = {
      ...makeUnit("irn-airbase", "IRN"),
      definitionId: "irn-airbase-def",
    };
    const detections = new Map<string, Set<string>>();
    expect(isVisible(fixedSite, "USA", detections, defInfo)).toBe(true);
    expect(isTrack(fixedSite, "USA", defInfo)).toBe(false);
  });

  it("resolves definition ids with record prefixes", () => {
    expect(normalizeDefinitionId("unit_definition:f35i")).toBe("f35i");
    expect(definitionInfoFor({ f35i: defInfo["usa-def"] }, "unit_definition:f35i")).toEqual(defInfo["usa-def"]);
  });
});
