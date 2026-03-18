import { describe, expect, it } from "vitest";
import { buildCountryCoalitionMap, collectRelationshipCountries, getRelationshipRule } from "./countryRelationships";

describe("countryRelationships", () => {
  it("allows transit and strike by default for hostile countries", () => {
    const coalitions = buildCountryCoalitionMap([
      { teamId: "ISR", coalitionId: "Blue" },
      { teamId: "IRN", coalitionId: "Red" },
    ]);
    expect(getRelationshipRule([], "ISR", "IRN", coalitions)).toMatchObject({
      shareIntel: false,
      airspaceTransitAllowed: true,
      airspaceStrikeAllowed: true,
      defensivePositioningAllowed: false,
      maritimeTransitAllowed: true,
      maritimeStrikeAllowed: true,
    });
  });

  it("keeps the fallback closed for non-hostile pairs", () => {
    const coalitions = buildCountryCoalitionMap([
      { teamId: "USA", coalitionId: "Blue" },
      { teamId: "QAT", coalitionId: "Blue" },
    ]);
    expect(getRelationshipRule([], "USA", "QAT", coalitions)).toMatchObject({
      shareIntel: false,
      airspaceTransitAllowed: false,
      airspaceStrikeAllowed: false,
      defensivePositioningAllowed: false,
      maritimeTransitAllowed: false,
      maritimeStrikeAllowed: false,
    });
  });

  it("collects countries from both units and explicit relationships", () => {
    expect(collectRelationshipCountries(
      [{ teamId: "ISR" }],
      [{
        fromCountry: "USA",
        toCountry: "SAU",
        shareIntel: false,
        airspaceTransitAllowed: false,
        airspaceStrikeAllowed: false,
        defensivePositioningAllowed: false,
        maritimeTransitAllowed: false,
        maritimeStrikeAllowed: false,
      }],
    )).toEqual(["ISR", "SAU", "USA"]);
  });
});
