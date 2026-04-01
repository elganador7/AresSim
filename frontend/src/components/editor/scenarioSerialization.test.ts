import { describe, expect, it } from "vitest";
import { fromBinary } from "@bufbuild/protobuf";
import { ScenarioSchema } from "@proto/engine/v1/scenario_pb";
import { base64ToBytes, draftPointsJSON, draftToProtoB64 } from "./scenarioSerialization";
import type { ScenarioDraft } from "../../store/editorStore";

describe("scenarioSerialization", () => {
  it("preserves H3 fields for unit positions and move-order waypoints", () => {
    const draft: ScenarioDraft = {
      id: "scen-1",
      name: "Scenario",
      description: "",
      classification: "UNCLASSIFIED",
      author: "",
      startTimeUnix: 0,
      version: "1.0.0",
      tickRateHz: 10,
      timeScale: 1,
      weatherState: 1,
      visibilityKm: 40,
      windSpeedMps: 5,
      temperatureC: 20,
      relationships: [],
      units: [
        {
          id: "unit-1",
          displayName: "Unit",
          fullName: "Unit",
          teamId: "USA",
          coalitionId: "COALITION_WEST",
          definitionId: "def-1",
          loadoutConfigurationId: "",
          natoSymbolSidc: "",
          lat: 25.2854,
          lon: 51.531,
          h3Cell: "8c2a100d291b5ff",
          h3ParentCell: "8b2a100d291ffff",
          altMsl: 1000,
          heading: 90,
          speed: 250,
          personnelStrength: 1,
          equipmentStrength: 1,
          combatEffectiveness: 1,
          fuelLevelLiters: 10000,
          morale: 1,
          fatigue: 0,
          damageState: 1,
          engagementBehavior: 1,
          engagementPkillThreshold: 0.5,
          moveOrder: {
            waypoints: [
              {
                lat: 26.566,
                lon: 56.25,
                h3Cell: "8c3b0e8e33a89ff",
                h3ParentCell: "8b3b0e8e33a9fff",
                altMsl: 1200,
              },
            ],
          },
        },
      ],
    };

    const encoded = draftToProtoB64(draft);
    const scenario = fromBinary(ScenarioSchema, base64ToBytes(encoded));
    expect(scenario.units[0].position?.h3Cell).toBe("8c2a100d291b5ff");
    expect(scenario.units[0].position?.h3ParentCell).toBe("8b2a100d291ffff");
    expect(scenario.units[0].moveOrder?.waypoints[0].h3Cell).toBe("8c3b0e8e33a89ff");
    expect(scenario.units[0].moveOrder?.waypoints[0].h3ParentCell).toBe("8b3b0e8e33a9fff");
  });

  it("includes H3 fields in draft preview point JSON", () => {
    const json = draftPointsJSON([
      { lat: 25.2854, lon: 51.531, h3Cell: "8c2a100d291b5ff", h3ParentCell: "8b2a100d291ffff" },
    ]);
    expect(json).toContain("\"h3Cell\":\"8c2a100d291b5ff\"");
    expect(json).toContain("\"h3ParentCell\":\"8b2a100d291ffff\"");
  });
});
