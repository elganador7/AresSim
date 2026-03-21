import { DesiredEffect } from "@proto/engine/v1/unit_pb";
import { WeaponEffectType } from "@proto/engine/v1/weapon_pb";
import type { UnitDefinitionDraft, WeaponConfigurationDraft } from "../store/editorStore";

export interface WeaponDefLite {
  id: string;
  domainTargets: number[];
  effectType: number;
}

export interface UnitDefinitionTargetLite {
  domain: number;
  targetClass?: string;
  stationary?: boolean;
  assetClass?: string;
}

export interface LoadoutAssessment {
  severity: "none" | "good" | "poor" | "invalid";
  message: string;
}

function resolveEffectOutcome(effectType: number, targetClass: string): "none" | "damage" | "mission_kill" | "destroy" {
  switch (effectType) {
    case WeaponEffectType.ANTI_AIR:
    case WeaponEffectType.INTERCEPTOR:
      return targetClass === "aircraft" ? "destroy" : "none";
    case WeaponEffectType.ANTI_SHIP:
    case WeaponEffectType.TORPEDO:
      if (targetClass === "surface_warship" || targetClass === "submarine") {
        return effectType === WeaponEffectType.TORPEDO ? "destroy" : "mission_kill";
      }
      return targetClass === "soft_infrastructure" || targetClass === "civilian_energy" ? "damage" : "none";
    case WeaponEffectType.ANTI_ARMOR:
      if (targetClass === "armor") return "destroy";
      if (targetClass === "sam_battery") return "damage";
      return "damage";
    case WeaponEffectType.SEAD:
      if (targetClass === "sam_battery") return "mission_kill";
      if (targetClass === "hardened_infrastructure") return "damage";
      return "none";
    case WeaponEffectType.BALLISTIC_STRIKE:
      if (targetClass === "soft_infrastructure") return "destroy";
      if (targetClass === "runway" || targetClass === "hardened_infrastructure" || targetClass === "civilian_energy" || targetClass === "civilian_water") {
        return "mission_kill";
      }
      return "damage";
    case WeaponEffectType.GUNFIRE:
      return targetClass === "aircraft" ? "destroy" : "damage";
    case WeaponEffectType.LAND_STRIKE:
    case WeaponEffectType.UNSPECIFIED:
    default:
      if (targetClass === "runway" || targetClass === "hardened_infrastructure" || targetClass === "civilian_energy" || targetClass === "civilian_water") {
        return "mission_kill";
      }
      return targetClass === "sam_battery" || targetClass === "surface_warship" ? "damage" : "damage";
  }
}

function resolveTargetClass(targetDef: UnitDefinitionTargetLite): string {
  const explicit = String(targetDef.targetClass ?? "").trim();
  if (explicit !== "") {
    return explicit;
  }

  if (targetDef.assetClass === "airbase") {
    return "runway";
  }

  switch (targetDef.domain) {
    case 2:
      return "aircraft";
    case 3:
      return "surface_warship";
    case 4:
      return "submarine";
    case 1:
      return targetDef.stationary ? "hardened_infrastructure" : "armor";
    default:
      return "soft_infrastructure";
  }
}

export function weaponCanEffectivelyAttackTarget(
  weapon: WeaponDefLite,
  targetDef: UnitDefinitionTargetLite | undefined,
): boolean {
  if (!targetDef) {
    return false;
  }
  if (!weapon.domainTargets.includes(targetDef.domain)) {
    return false;
  }
  const targetClass = resolveTargetClass(targetDef);

  // Ballistic strike weapons are strategic / fixed-target tools in this sim.
  // They should not be assignable against mobile units like aircraft or armor.
  if (weapon.effectType === WeaponEffectType.BALLISTIC_STRIKE) {
    const fixedTargetClass = new Set([
      "runway",
      "soft_infrastructure",
      "hardened_infrastructure",
      "civilian_energy",
      "civilian_water",
      "sam_battery",
    ]);
    if (!targetDef.stationary && !fixedTargetClass.has(targetClass)) {
      return false;
    }
  }

  return resolveEffectOutcome(weapon.effectType, targetClass) !== "none";
}

function outcomeSupportsDesiredEffect(outcome: "none" | "damage" | "mission_kill" | "destroy", desiredEffect: number): boolean {
  switch (desiredEffect) {
    case DesiredEffect.DAMAGE:
      return outcome !== "none";
    case DesiredEffect.MISSION_KILL:
      return outcome === "mission_kill" || outcome === "destroy";
    case DesiredEffect.DESTROY:
      return outcome === "mission_kill" || outcome === "destroy";
    default:
      return outcome !== "none";
  }
}

export function assessLoadoutAgainstTarget(
  configuration: WeaponConfigurationDraft | undefined,
  targetDef: UnitDefinitionDraft | undefined,
  weaponDefs: Map<string, WeaponDefLite>,
  desiredEffect: number,
): LoadoutAssessment {
  if (!configuration || !targetDef) {
    return { severity: "none", message: "" };
  }

  const loadedWeapons = configuration.loadout
    .filter((slot) => slot.initialQty > 0 || slot.maxQty > 0)
    .map((slot) => weaponDefs.get(slot.weaponId))
    .filter((weapon): weapon is WeaponDefLite => Boolean(weapon));

  if (loadedWeapons.length === 0) {
    return { severity: "invalid", message: "Selected loadout has no weapons." };
  }

  const domainMatches = loadedWeapons.filter((weapon) => weapon.domainTargets.includes(targetDef.domain));
  if (domainMatches.length === 0) {
    return { severity: "invalid", message: "Selected loadout cannot engage this target domain." };
  }

  const effectiveWeapons = domainMatches.filter((weapon) => weaponCanEffectivelyAttackTarget(weapon, targetDef));
  if (effectiveWeapons.length === 0) {
    return { severity: "invalid", message: "Selected loadout is not an effective match for this target." };
  }

  const effectMatches = effectiveWeapons.filter((weapon) =>
    outcomeSupportsDesiredEffect(resolveEffectOutcome(weapon.effectType, resolveTargetClass(targetDef)), desiredEffect),
  );
  if (effectMatches.length === 0) {
    return { severity: "poor", message: "Selected loadout can reach the target, but it is a poor fit for the requested effect." };
  }

  return { severity: "good", message: "Selected loadout is suitable for this attack task." };
}
