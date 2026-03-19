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

export interface IranWarScoreboardItem {
  label: string;
  value: string;
  severity: "good" | "warning" | "bad";
}

export interface IranWarStrikeForceSummary {
  ready: number;
  delayed: number;
  spentOrLost: number;
}

export interface IranWarAirOpsStatus {
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

function isTrackedAirbase(unitId: string): boolean {
  return COALITION_AIRBASES.includes(unitId) || IRANIAN_BASES.includes(unitId);
}

function isTrackedStrikeUnit(unitId: string): boolean {
  return IRAN_MISSILE_FORCE.includes(unitId);
}

export function getIranWarStrikeForceSummary(teamId: string, units: Map<string, Unit>): IranWarStrikeForceSummary {
  let ready = 0;
  let delayed = 0;
  let spentOrLost = 0;
  units.forEach((unit) => {
    if (unit.teamId?.trim().toUpperCase() !== teamId.trim().toUpperCase()) {
      return;
    }
    if (!isTrackedStrikeUnit(unit.id)) {
      return;
    }
    if (unit.status.isActive === false || unit.damageState >= 4) {
      spentOrLost++;
      return;
    }
    const totalWeapons = unit.weapons.reduce((sum, weapon) => sum + weapon.currentQty, 0);
    if (totalWeapons <= 0) {
      spentOrLost++;
      return;
    }
    if ((unit.nextStrikeReadySeconds ?? 0) > 0 || unit.damageState >= 2) {
      delayed++;
      return;
    }
    ready++;
  });
  return { ready, delayed, spentOrLost };
}

function evaluateAirbaseBottleneck(unit: Unit | undefined): "good" | "warning" | "bad" {
  if (!unit || unit.status.isActive === false || unit.damageState >= 4) {
    return "bad";
  }
  if (!unit.baseOps) {
    return unit.damageState >= 2 ? "warning" : "good";
  }
  if (unit.baseOps.state !== 1) {
    return unit.baseOps.state === 2 ? "warning" : "bad";
  }
  if ((unit.baseOps.nextLaunchAvailableSeconds ?? 0) > 0 || (unit.baseOps.nextRecoveryAvailableSeconds ?? 0) > 0) {
    return "warning";
  }
  return "good";
}

function severityLabel(severity: "good" | "warning" | "bad"): string {
  switch (severity) {
    case "good":
      return "Stable";
    case "warning":
      return "Constrained";
    case "bad":
      return "Critical";
  }
}

export function getIranWarScoreboard(teamId: string, units: Map<string, Unit>): IranWarScoreboardItem[] {
  const team = teamId.trim().toUpperCase();
  const ownCoalition = Array.from(units.values()).find((unit) => unit.teamId?.trim().toUpperCase() === team)?.coalitionId?.trim().toUpperCase() ?? "";
  const ownAirbases = Array.from(units.values()).filter((unit) => unit.teamId?.trim().toUpperCase() === team && isTrackedAirbase(unit.id));
  const usableAirbases = ownAirbases.filter((unit) => unit.baseOps?.state === 1).length;
  const constrainedAirbases = ownAirbases.filter((unit) => evaluateAirbaseBottleneck(unit) !== "good").length;
  const strikeForce = getIranWarStrikeForceSummary(team, units);

  let enemyMissileSuppression = 0;
  const enemyStrikeUnits = Array.from(units.values()).filter((unit) => {
    if (!isTrackedStrikeUnit(unit.id)) {
      return false;
    }
    const otherCoalition = unit.coalitionId?.trim().toUpperCase() ?? "";
    return ownCoalition !== "" && otherCoalition !== "" && otherCoalition !== ownCoalition;
  });
  if (enemyStrikeUnits.length > 0) {
    enemyMissileSuppression = enemyStrikeUnits.filter((unit) => unit.damageState >= 3 || unit.status.isActive === false).length;
  }

  return [
    {
      label: "Usable Airbases",
      value: `${usableAirbases}/${ownAirbases.length || 0}`,
      severity: usableAirbases === ownAirbases.length ? "good" : usableAirbases > 0 ? "warning" : "bad",
    },
    {
      label: "Airbase Bottlenecks",
      value: constrainedAirbases === 0 ? "None" : `${constrainedAirbases} constrained`,
      severity: constrainedAirbases === 0 ? "good" : constrainedAirbases < ownAirbases.length ? "warning" : "bad",
    },
    {
      label: "Strike Force",
      value: `${strikeForce.ready} ready / ${strikeForce.delayed} delayed`,
      severity: strikeForce.ready > 0 ? (strikeForce.delayed > 0 ? "warning" : "good") : "bad",
    },
    {
      label: "Enemy Suppressed",
      value: `${enemyMissileSuppression}/${enemyStrikeUnits.length || 0}`,
      severity: enemyStrikeUnits.length === 0 ? "good" : enemyMissileSuppression === 0 ? "bad" : enemyMissileSuppression < enemyStrikeUnits.length ? "warning" : "good",
    },
  ];
}

function isOwnAirUnit(team: string, unit: Unit): boolean {
  return unit.teamId?.trim().toUpperCase() === team && unit.hostBaseId !== undefined;
}

function isGrounded(unit: Unit): boolean {
  return unit.position.altMsl <= 100;
}

function findUnit(units: Map<string, Unit>, unitId: string | undefined): Unit | undefined {
  return unitId ? units.get(unitId) : undefined;
}

function describeGrounding(unit: Unit, hostBase: Unit | undefined): Pick<IranWarAirOpsStatus, "status" | "severity"> | null {
  if (!isGrounded(unit) || unit.status.isActive === false) {
    return null;
  }
  if (hostBase?.baseOps?.state === 3) {
    return { status: "Grounded: Host Base Closed", severity: "bad" };
  }
  if (hostBase?.baseOps?.state === 2) {
    return { status: "Grounded: Base Degraded", severity: "warning" };
  }
  if ((unit.nextSortieReadySeconds ?? 0) > 0) {
    return { status: "Grounded: Turnaround Delay", severity: "warning" };
  }
  if ((hostBase?.baseOps?.nextLaunchAvailableSeconds ?? 0) > 0) {
    return { status: "Grounded: Launch Queue", severity: "warning" };
  }
  return null;
}

export function getIranWarGroundedAircraft(teamId: string, units: Map<string, Unit>): IranWarAirOpsStatus[] {
  const team = teamId.trim().toUpperCase();
  return Array.from(units.values())
    .filter((unit) => isOwnAirUnit(team, unit))
    .map((unit): IranWarAirOpsStatus | null => {
      const hostBase = findUnit(units, unit.hostBaseId);
      const state = describeGrounding(unit, hostBase);
      if (!state) {
        return null;
      }
      return {
        unitId: unit.id,
        label: unit.displayName,
        ...state,
      };
    })
    .filter((item): item is IranWarAirOpsStatus => item !== null)
    .slice(0, 6);
}

export function getIranWarAirbaseConstraints(teamId: string, units: Map<string, Unit>): IranWarAirOpsStatus[] {
  const team = teamId.trim().toUpperCase();
  return Array.from(units.values())
    .filter((unit) => unit.teamId?.trim().toUpperCase() === team && isTrackedAirbase(unit.id))
    .map((unit): IranWarAirOpsStatus | null => {
      if (unit.status.isActive === false || unit.damageState >= 4) {
        return {
          unitId: unit.id,
          label: unit.displayName,
          status: "Destroyed",
          severity: "bad",
        };
      }
      if (unit.baseOps?.state === 3) {
        return {
          unitId: unit.id,
          label: unit.displayName,
          status: "Closed",
          severity: "bad",
        };
      }
      if (unit.baseOps?.state === 2) {
        return {
          unitId: unit.id,
          label: unit.displayName,
          status: "Degraded",
          severity: "warning",
        };
      }
      if ((unit.baseOps?.nextLaunchAvailableSeconds ?? 0) > 0 || (unit.baseOps?.nextRecoveryAvailableSeconds ?? 0) > 0) {
        return {
          unitId: unit.id,
          label: unit.displayName,
          status: "Queueing",
          severity: "warning",
        };
      }
      return null;
    })
    .filter((item): item is IranWarAirOpsStatus => item !== null)
    .slice(0, 6);
}

function describeStrikeUnit(unit: Unit | undefined): Pick<IranWarAirOpsStatus, "status" | "severity"> {
  if (!unit || unit.status.isActive === false || unit.damageState >= 4) {
    return { status: "Destroyed", severity: "bad" };
  }
  const totalWeapons = unit.weapons.reduce((sum, weapon) => sum + weapon.currentQty, 0);
  if (totalWeapons <= 0) {
    return { status: "Out of Shots", severity: "bad" };
  }
  if (unit.damageState >= 2) {
    return { status: "Damaged", severity: "warning" };
  }
  if ((unit.nextStrikeReadySeconds ?? 0) > 0) {
    return { status: "Delayed", severity: "warning" };
  }
  return { status: "Ready", severity: "good" };
}

export function getIranWarStrikeUnitStatuses(teamId: string, units: Map<string, Unit>): IranWarAirOpsStatus[] {
  const team = teamId.trim().toUpperCase();
  return Array.from(units.values())
    .filter((unit) => unit.teamId?.trim().toUpperCase() === team && isTrackedStrikeUnit(unit.id))
    .map((unit) => ({
      unitId: unit.id,
      label: unit.displayName,
      ...describeStrikeUnit(unit),
    }))
    .slice(0, 6);
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
