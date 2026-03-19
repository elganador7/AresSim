import { describe, expect, it } from "vitest";
import { buildCountryCoalitionMap, collectRelationshipCountries } from "./countryRelationships";

describe("countryRelationships", () => {
  it("builds a country-to-coalition map from units", () => {
    expect(buildCountryCoalitionMap([
      { teamId: "ISR", coalitionId: "Blue" },
      { teamId: "IRN", coalitionId: "Red" },
      { teamId: "ISR", coalitionId: "OtherIgnored" },
    ])).toEqual({
      ISR: "BLUE",
      IRN: "RED",
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
