import type { WeaponDef, Unit } from "../store/simStore";
import type { UnitDraft, UnitDefinitionDraft } from "../store/editorStore";
import { weaponCanEffectivelyAttackTarget, type UnitDefinitionTargetLite, type WeaponDefLite } from "./loadoutValidation";
import { areHostile } from "./allegiance";

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
): Unit[] {
  const loadedWeapons = unit.weapons
    .filter((weapon) => weapon.currentQty > 0)
    .map((weapon) => weaponDefs.get(weapon.weaponId))
    .filter((weapon): weapon is WeaponDef => Boolean(weapon));
  if (loadedWeapons.length === 0) {
    return [];
  }
  return Array.from(units.values()).filter((candidate) => {
    if (candidate.id === unit.id || !areHostile(unit, candidate)) {
      return false;
    }
    const targetDef = definitionMap.get(candidate.definitionId);
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
  const actingUnit = {
    teamId: allegianceOverride?.teamId ?? unit.teamId,
    coalitionId: allegianceOverride?.coalitionId ?? unit.coalitionId,
  };
  return units.filter((candidate) => {
    if (candidate.id === unit.id || !areHostile(actingUnit, candidate)) {
      return false;
    }
    const candidateDef = unitDefinitions.find((def) => def.id === candidate.definitionId);
    return loadedWeapons.some((weapon) => weaponCanEffectivelyAttackTarget(weapon, candidateDef));
  });
}
