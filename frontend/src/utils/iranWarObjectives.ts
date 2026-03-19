import type { TeamScore, Unit } from "../store/simStore";

export type IranWarObjectiveMode = "neutralize" | "preserve" | "usable_airbase" | "disrupt_airbase";

export interface IranWarObjective {
  id: string;
  title: string;
  detail: string;
  mode: IranWarObjectiveMode;
  unitIds: string[];
}

export interface IranWarObjectiveSet {
  title: string;
  summary: string;
  objectives: IranWarObjective[];
}

export interface IranWarObjectiveProgress {
  completed: number;
  total: number;
  label: string;
}

const IRAN_MISSILE_FORCE = [
  "irn-qiam-central",
  "irn-kheibar-west",
  "irn-paveh-south",
  "irn-shahed-central",
  "irn-arash-west",
];

const IRAN_IADS = [
  "irn-s300-tehran",
  "irn-bavar-esfahan",
  "irn-khordad-bushehr",
  "irn-tor-natanz",
];

const COALITION_AIRBASES = [
  "isr-airbase-nevatim",
  "isr-airbase-hatzor",
  "isr-airbase-ramon",
  "qat-airbase-al-udeid",
  "uae-airbase-al-dhafra",
  "usa-airbase-diego-garcia",
];

const ISRAELI_DEFENSES = [
  "isr-arrow3-palmachim",
  "isr-davids-sling-dan",
  "isr-iron-dome-haifa",
];

const IRANIAN_BASES = [
  "irn-airbase-tehran",
  "irn-airbase-bandar-abbas",
];

export function isIranWarScenario(name: string): boolean {
  return name.trim().toUpperCase().includes("IRAN WAR 2026");
}

export function getIranWarObjectiveSet(teamId: string): IranWarObjectiveSet | null {
  switch (teamId.trim().toUpperCase()) {
    case "USA":
      return {
        title: "Coalition Air Campaign",
        summary: "Suppress Iranian long-range strike systems while keeping Gulf basing usable for follow-on sorties.",
        objectives: [
          {
            id: "usa-missiles",
            title: "Neutralize Iranian Missile Forces",
            detail: "Reduce ballistic, cruise, and one-way attack systems before major retaliation lands on Gulf bases.",
            mode: "neutralize",
            unitIds: IRAN_MISSILE_FORCE,
          },
          {
            id: "usa-iads",
            title: "Break Iranian Air Defenses",
            detail: "Mission-kill key IADS nodes so later waves can strike with lower risk.",
            mode: "neutralize",
            unitIds: IRAN_IADS,
          },
          {
            id: "usa-bases",
            title: "Keep Coalition Airbases Usable",
            detail: "Protect Gulf and Israeli launch/recovery hubs from closure and sustained throughput loss.",
            mode: "usable_airbase",
            unitIds: COALITION_AIRBASES,
          },
        ],
      };
    case "ISR":
      return {
        title: "Israeli Opening Strike",
        summary: "Disrupt Iranian retaliatory capacity and preserve homeland air defense through the opening day.",
        objectives: [
          {
            id: "isr-missiles",
            title: "Suppress Iranian Retaliatory Fires",
            detail: "Destroy or mission-kill missile and strike-drone units threatening Israel in the first hours.",
            mode: "neutralize",
            unitIds: IRAN_MISSILE_FORCE,
          },
          {
            id: "isr-iads",
            title: "Open the Air Corridor",
            detail: "Knock down enough Iranian SAM coverage to preserve repeated strike access.",
            mode: "neutralize",
            unitIds: IRAN_IADS,
          },
          {
            id: "isr-homeland",
            title: "Preserve Homeland Defense",
            detail: "Keep Israel’s strategic missile-defense batteries alive and functioning through retaliation.",
            mode: "preserve",
            unitIds: ISRAELI_DEFENSES,
          },
        ],
      };
    case "IRN":
      return {
        title: "Iranian Retaliation Campaign",
        summary: "Inflict maximum operational pain on coalition basing while preserving enough retaliatory force for follow-on raids.",
        objectives: [
          {
            id: "irn-airbases",
            title: "Damage Coalition Airbases",
            detail: "Close or degrade the main launch hubs driving the coalition’s first-day strike tempo.",
            mode: "disrupt_airbase",
            unitIds: COALITION_AIRBASES,
          },
          {
            id: "irn-survive",
            title: "Preserve Retaliatory Forces",
            detail: "Keep enough missile and drone units alive to sustain repeated salvos after the opening blows.",
            mode: "preserve",
            unitIds: IRAN_MISSILE_FORCE,
          },
          {
            id: "irn-bases",
            title: "Keep Iranian Bases Open",
            detail: "Prevent Tehran and Bandar Abbas from becoming closed launch/recovery bottlenecks.",
            mode: "usable_airbase",
            unitIds: IRANIAN_BASES,
          },
        ],
      };
    default:
      return null;
  }
}

function isNeutralized(unit: Unit | undefined): boolean {
  if (!unit) {
    return true;
  }
  return unit.damageState >= 3 || unit.status.isActive === false;
}

function isPreserved(unit: Unit | undefined): boolean {
  if (!unit) {
    return false;
  }
  return unit.damageState < 3 && unit.status.isActive !== false;
}

function isUsableAirbase(unit: Unit | undefined): boolean {
  if (!unit) {
    return false;
  }
  return unit.baseOps?.state === 1;
}

function isDisruptedAirbase(unit: Unit | undefined): boolean {
  if (!unit) {
    return false;
  }
  if (unit.baseOps) {
    return unit.baseOps.state !== 1;
  }
  return unit.damageState >= 2 || unit.status.isActive === false;
}

export function computeIranWarObjectiveProgress(objective: IranWarObjective, units: Map<string, Unit>): IranWarObjectiveProgress {
  let completed = 0;
  for (const unitId of objective.unitIds) {
    const unit = units.get(unitId);
    switch (objective.mode) {
      case "neutralize":
        if (isNeutralized(unit)) {
          completed++;
        }
        break;
      case "preserve":
        if (isPreserved(unit)) {
          completed++;
        }
        break;
      case "usable_airbase":
        if (isUsableAirbase(unit)) {
          completed++;
        }
        break;
      case "disrupt_airbase":
        if (isDisruptedAirbase(unit)) {
          completed++;
        }
        break;
    }
  }
  return {
    completed,
    total: objective.unitIds.length,
    label: `${completed}/${objective.unitIds.length}`,
  };
}

export function buildWarCostSummary(playerTeam: string, units: Map<string, Unit>, scores: TeamScore[]): { ownLossUsd: number; enemyLossUsd: number } {
  const teamCoalitions = new Map<string, string>();
  units.forEach((unit) => {
    const teamId = unit.teamId?.trim().toUpperCase();
    const coalitionId = unit.coalitionId?.trim().toUpperCase();
    if (teamId && coalitionId && !teamCoalitions.has(teamId)) {
      teamCoalitions.set(teamId, coalitionId);
    }
  });

  const playerCoalition = teamCoalitions.get(playerTeam.trim().toUpperCase()) ?? "";
  let ownLossUsd = 0;
  let enemyLossUsd = 0;
  for (const score of scores) {
    const coalitionId = teamCoalitions.get(score.teamId.trim().toUpperCase()) ?? "";
    if (playerCoalition && coalitionId === playerCoalition) {
      ownLossUsd += score.totalLossUsd;
    } else if (playerCoalition && coalitionId && coalitionId !== playerCoalition) {
      enemyLossUsd += score.totalLossUsd;
    }
  }
  return { ownLossUsd, enemyLossUsd };
}
