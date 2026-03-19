import { describe, expect, it } from "vitest";
import { buildWarCostSummary, computeIranWarObjectiveProgress, getIranWarObjectiveSet } from "./iranWarObjectives";
import type { TeamScore, Unit } from "../store/simStore";

function makeUnit(id: string, teamId: string, coalitionId: string): Unit {
  return {
    id,
    displayName: id,
    fullName: id,
    side: coalitionId,
    teamId,
    coalitionId,
    natoPendingSymbol: "",
    definitionId: id,
    damageState: 1,
    position: { lat: 0, lon: 0, altMsl: 0, heading: 0, speed: 0 },
    status: {
      personnelStrength: 1,
      equipmentStrength: 1,
      combatEffectiveness: 1,
      fuelLevelLiters: 1,
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

describe("iran war objectives", () => {
  it("counts neutralized targets when mission-killed or destroyed", () => {
    const set = getIranWarObjectiveSet("USA");
    if (!set) {
      throw new Error("expected objective set");
    }
    const objective = set.objectives[0];
    const units = new Map<string, Unit>();
    objective.unitIds.forEach((id, index) => {
      const unit = makeUnit(id, "IRN", "RED");
      if (index < 2) {
        unit.damageState = 3;
      }
      units.set(id, unit);
    });
    expect(computeIranWarObjectiveProgress(objective, units)).toEqual({
      completed: 2,
      total: objective.unitIds.length,
      label: `2/${objective.unitIds.length}`,
    });
  });

  it("aggregates war cost by coalition", () => {
    const units = new Map<string, Unit>([
      ["usa-1", makeUnit("usa-1", "USA", "BLUE")],
      ["isr-1", makeUnit("isr-1", "ISR", "BLUE")],
      ["irn-1", makeUnit("irn-1", "IRN", "RED")],
    ]);
    const scores: TeamScore[] = [
      { teamId: "USA", replacementLossUsd: 0, strategicLossUsd: 0, economicLossUsd: 0, humanLossUsd: 0, totalLossUsd: 10 },
      { teamId: "ISR", replacementLossUsd: 0, strategicLossUsd: 0, economicLossUsd: 0, humanLossUsd: 0, totalLossUsd: 20 },
      { teamId: "IRN", replacementLossUsd: 0, strategicLossUsd: 0, economicLossUsd: 0, humanLossUsd: 0, totalLossUsd: 35 },
    ];
    expect(buildWarCostSummary("USA", units, scores)).toEqual({ ownLossUsd: 30, enemyLossUsd: 35 });
  });

  it("counts degraded airbases as disrupted", () => {
    const set = getIranWarObjectiveSet("IRN");
    if (!set) {
      throw new Error("expected objective set");
    }
    const objective = set.objectives[0];
    const units = new Map<string, Unit>();
    objective.unitIds.forEach((id, index) => {
      const unit = makeUnit(id, "USA", "BLUE");
      unit.baseOps = {
        state: index < 2 ? 2 : 1,
        nextLaunchAvailableSeconds: 0,
        nextRecoveryAvailableSeconds: 0,
      };
      units.set(id, unit);
    });
    expect(computeIranWarObjectiveProgress(objective, units)).toEqual({
      completed: 2,
      total: objective.unitIds.length,
      label: `2/${objective.unitIds.length}`,
    });
  });
});
