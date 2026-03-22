import type { WeaponDef, Unit } from "../store/simStore";
import type { UnitDraft, UnitDefinitionDraft } from "../store/editorStore";
import { weaponCanEffectivelyAttackTarget, type UnitDefinitionTargetLite, type WeaponDefLite } from "./loadoutValidation";
import { areHostile } from "./allegiance";

function normalizeDefinitionId(definitionId: string | undefined): string {
  const raw = String(definitionId ?? "").trim();
  const idx = raw.lastIndexOf(":");
  return idx >= 0 ? raw.slice(idx + 1) : raw;
}

export const ENGAGEMENT_BEHAVIORS = [
  { value: 1, label: "Auto Engage" },
  { value: 2, label: "Self-Defense Only" },
  { value: 3, label: "Hold Fire" },
  { value: 4, label: "Assigned Targets Only" },
  { value: 5, label: "Shadow Contact" },
  { value: 6, label: "Withdraw On Detect" },
];

export const ATTACK_ORDER_TYPES = [
  { value: 0, label: "No Attack Task" },
  { value: 1, label: "Attack Assigned Target" },
  { value: 2, label: "Strike Until Effect" },
];

export const DESIRED_EFFECTS = [
  { value: 1, label: "Damage" },
  { value: 2, label: "Mission Kill" },
  { value: 3, label: "Destroy" },
];

export function filterValidLiveTargets(
  unit: Unit,
  units: Map<string, Unit>,
  weaponDefs: Map<string, WeaponDef>,
  definitionMap: Map<string, UnitDefinitionTargetLite>,
  visibleTargetIds?: Set<string>,
): Unit[] {
  const loadedWeapons = unit.weapons
    .filter((weapon) => weapon.currentQty > 0)
    .map((weapon) => weaponDefs.get(weapon.weaponId))
    .filter((weapon): weapon is WeaponDef => Boolean(weapon));
  return filterValidLiveTargetsForWeapons(unit, units, loadedWeapons, definitionMap, visibleTargetIds);
}

function isPreplannedFixedTarget(targetDef: UnitDefinitionTargetLite | undefined): boolean {
  if (!targetDef) {
    return false;
  }
  if (targetDef.assetClass === "airbase" || targetDef.assetClass === "port") {
    return true;
  }
  const targetClass = String(targetDef.targetClass ?? "").trim();
  if (
    targetClass === "runway" ||
    targetClass === "hardened_infrastructure" ||
    targetClass === "soft_infrastructure" ||
    targetClass === "civilian_energy" ||
    targetClass === "civilian_water" ||
    targetClass === "sam_battery"
  ) {
    return true;
  }
  return false;
}

export function filterValidLiveTargetsForWeapons(
  unit: Pick<Unit, "id" | "teamId" | "coalitionId" | "definitionId">,
  units: Map<string, Unit>,
  loadedWeapons: WeaponDefLite[],
  definitionMap: Map<string, UnitDefinitionTargetLite>,
  visibleTargetIds?: Set<string>,
): Unit[] {
  const shooterDef = definitionMap.get(normalizeDefinitionId(unit.definitionId));
  if (loadedWeapons.length === 0) {
    return [];
  }
  return Array.from(units.values()).filter((candidate) => {
    if (candidate.id === unit.id || !areHostile(unit, candidate)) {
      return false;
    }
    const targetDef = definitionMap.get(normalizeDefinitionId(candidate.definitionId));
    if ((shooterDef?.domain === 3 || shooterDef?.domain === 4) && targetDef?.domain === 1) {
      return false;
    }
    const currentlyVisible = visibleTargetIds?.has(candidate.id) ?? true;
    if (!currentlyVisible && !isPreplannedFixedTarget(targetDef)) {
      return false;
    }
    return loadedWeapons.some((weapon) => weaponCanEffectivelyAttackTarget(weapon, targetDef));
  });
}

export function filterValidEditorTargets(
  unit: UnitDraft,
  units: UnitDraft[],
  loadedWeapons: WeaponDefLite[],
  unitDefinitions: UnitDefinitionDraft[],
  allegianceOverride?: { teamId?: string; coalitionId?: string },
): UnitDraft[] {
  if (loadedWeapons.length === 0) {
    return [];
  }
  const shooterDef = unitDefinitions.find((def) => def.id === normalizeDefinitionId(unit.definitionId));
  const actingUnit = {
    teamId: allegianceOverride?.teamId ?? unit.teamId,
    coalitionId: allegianceOverride?.coalitionId ?? unit.coalitionId,
  };
  return units.filter((candidate) => {
    if (candidate.id === unit.id || !areHostile(actingUnit, candidate)) {
      return false;
    }
    const candidateDef = unitDefinitions.find((def) => def.id === normalizeDefinitionId(candidate.definitionId));
    if ((shooterDef?.domain === 3 || shooterDef?.domain === 4) && candidateDef?.domain === 1) {
      return false;
    }
    return loadedWeapons.some((weapon) => weaponCanEffectivelyAttackTarget(weapon, candidateDef));
  });
}
