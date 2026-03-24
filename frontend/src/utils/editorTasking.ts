import type { UnitDraft, UnitDefinitionDraft } from "../store/editorStore";
import type { UnitDefinitionTargetLite, WeaponDefLite } from "./loadoutValidation";
import { weaponCanEffectivelyAttackTarget } from "./loadoutValidation";
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
    return loadedWeapons.some((weapon) => weaponCanEffectivelyAttackTarget(weapon, candidateDef as UnitDefinitionTargetLite | undefined));
  });
}
