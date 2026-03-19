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

export interface IranWarOpeningWaveItem {
  shooterId: string;
  shooterLabel: string;
  targetId: string;
  targetLabel: string;
}

export interface IranWarOpeningWaveStatus extends IranWarOpeningWaveItem {
  status: "ready" | "launched" | "spent" | "lost";
}

export interface IranWarKeyTargetStatus {
  unitId: string;
  label: string;
  status: string;
  severity: "good" | "warning" | "bad";
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

const OPENING_WAVE_BY_TEAM: Record<string, IranWarOpeningWaveItem[]> = {
  USA: [
    {
      shooterId: "usa-f35a-al-udeid",
      shooterLabel: "F-35A Al Udeid",
      targetId: "irn-khordad-bushehr",
      targetLabel: "Bushehr 3rd Khordad",
    },
    {
      shooterId: "usa-f15e-al-dhafra",
      shooterLabel: "F-15E Al Dhafra",
      targetId: "irn-paveh-south",
      targetLabel: "Southern Paveh Regiment",
    },
    {
      shooterId: "usa-b1b-diego-garcia",
      shooterLabel: "B-1B Diego Garcia",
      targetId: "irn-kheibar-west",
      targetLabel: "Western Kheibar Brigade",
    },
  ],
  ISR: [
    {
      shooterId: "isr-f35i-nevatim",
      shooterLabel: "F-35I Nevatim",
      targetId: "irn-s300-tehran",
      targetLabel: "Tehran S-300",
    },
    {
      shooterId: "isr-f15i-hatzor",
      shooterLabel: "F-15I Hatzor",
      targetId: "irn-qiam-central",
      targetLabel: "Central Qiam Brigade",
    },
    {
      shooterId: "isr-f16i-ramon",
      shooterLabel: "F-16I Ramon",
      targetId: "irn-bavar-esfahan",
      targetLabel: "Esfahan Bavar-373",
    },
  ],
  IRN: [
    {
      shooterId: "irn-qiam-central",
      shooterLabel: "Central Qiam Brigade",
      targetId: "isr-airbase-nevatim",
      targetLabel: "Nevatim AB",
    },
    {
      shooterId: "irn-kheibar-west",
      shooterLabel: "Western Kheibar Brigade",
      targetId: "qat-airbase-al-udeid",
      targetLabel: "Al Udeid AB",
    },
    {
      shooterId: "irn-paveh-south",
      shooterLabel: "Southern Paveh Regiment",
      targetId: "uae-airbase-al-dhafra",
      targetLabel: "Al Dhafra AB",
    },
  ],
};

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

function openingWaveStatusForUnit(unit: Unit | undefined): IranWarOpeningWaveStatus["status"] {
  if (!unit || unit.status.isActive === false || unit.damageState >= 4) {
    return "lost";
  }
  if (unit.attackOrder?.targetUnitId || (unit.moveOrder?.waypoints.length ?? 0) > 0 || unit.position.altMsl > 100) {
    return "launched";
  }
  const hasSpentMagazine = unit.weapons.length > 0 && unit.weapons.every((weapon) => weapon.currentQty < weapon.maxQty);
  if (hasSpentMagazine || (unit.nextStrikeReadySeconds ?? 0) > 0) {
    return "spent";
  }
  return "ready";
}

export function getIranWarOpeningWaveStatus(teamId: string, units: Map<string, Unit>): IranWarOpeningWaveStatus[] {
  const items = OPENING_WAVE_BY_TEAM[teamId.trim().toUpperCase()] ?? [];
  return items.map((item) => ({
    ...item,
    status: openingWaveStatusForUnit(units.get(item.shooterId)),
  }));
}

function describeKeyTarget(unit: Unit | undefined): Pick<IranWarKeyTargetStatus, "status" | "severity"> {
  if (!unit || unit.status.isActive === false || unit.damageState >= 4) {
    return { status: "Destroyed", severity: "bad" };
  }
  if (unit.baseOps) {
    if (unit.baseOps.state === 1) {
      return { status: "Usable", severity: "good" };
    }
    if (unit.baseOps.state === 2) {
      return { status: "Degraded", severity: "warning" };
    }
    return { status: "Closed", severity: "bad" };
  }
  if (unit.damageState >= 3) {
    return { status: "Mission Killed", severity: "bad" };
  }
  if (unit.damageState >= 2) {
    return { status: "Damaged", severity: "warning" };
  }
  return { status: "Operational", severity: "good" };
}

export function getIranWarKeyTargetStatuses(teamId: string, units: Map<string, Unit>): IranWarKeyTargetStatus[] {
  const team = teamId.trim().toUpperCase();
  let keys: { unitId: string; label: string }[] = [];
  switch (team) {
    case "USA":
      keys = [
        { unitId: "qat-airbase-al-udeid", label: "Al Udeid AB" },
        { unitId: "uae-airbase-al-dhafra", label: "Al Dhafra AB" },
        { unitId: "irn-qiam-central", label: "Qiam Brigade" },
        { unitId: "irn-kheibar-west", label: "Kheibar Brigade" },
      ];
      break;
    case "ISR":
      keys = [
        { unitId: "isr-airbase-nevatim", label: "Nevatim AB" },
        { unitId: "isr-arrow3-palmachim", label: "Arrow-3 Palmachim" },
        { unitId: "irn-s300-tehran", label: "Tehran S-300" },
        { unitId: "irn-qiam-central", label: "Qiam Brigade" },
      ];
      break;
    case "IRN":
      keys = [
        { unitId: "irn-airbase-tehran", label: "Tehran AB" },
        { unitId: "irn-kheibar-west", label: "Kheibar Brigade" },
        { unitId: "isr-airbase-nevatim", label: "Nevatim AB" },
        { unitId: "qat-airbase-al-udeid", label: "Al Udeid AB" },
      ];
      break;
    default:
      keys = [];
  }
  return keys.map((key) => ({
    unitId: key.unitId,
    label: key.label,
    ...describeKeyTarget(units.get(key.unitId)),
  }));
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
