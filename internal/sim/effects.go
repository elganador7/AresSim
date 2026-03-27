package sim

import enginev1 "github.com/aressim/internal/gen/engine/v1"

type impactOutcome int

const (
	outcomeNoEffect impactOutcome = iota
	outcomeLightDamage
	outcomeMobilityKill
	outcomeFirepowerLoss
	outcomeMissionKill
	outcomeRunwayCrater
	outcomeCatastrophicKill
)

type HitResult struct {
	Attacker      *enginev1.Unit
	Victim        *enginev1.Unit
	Outcome       impactOutcome
	Destroyed     bool
	PreviousState enginev1.DamageState
}

func resolveImpactOutcome(effectType enginev1.WeaponEffectType, targetClass string) impactOutcome {
	switch effectType {
	case enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_AIR, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_INTERCEPTOR:
		switch targetClass {
		case "aircraft":
			return outcomeCatastrophicKill
		default:
			return outcomeNoEffect
		}
	case enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_SHIP, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_TORPEDO:
		switch targetClass {
		case "surface_warship", "submarine":
			if effectType == enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_TORPEDO {
				return outcomeCatastrophicKill
			}
			return outcomeMissionKill
		case "soft_infrastructure", "civilian_energy":
			return outcomeLightDamage
		default:
			return outcomeNoEffect
		}
	case enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_ARMOR:
		switch targetClass {
		case "armor":
			return outcomeCatastrophicKill
		case "sam_battery":
			return outcomeMobilityKill
		default:
			return outcomeLightDamage
		}
	case enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_SEAD:
		switch targetClass {
		case "sam_battery":
			return outcomeMissionKill
		case "hardened_infrastructure":
			return outcomeLightDamage
		default:
			return outcomeNoEffect
		}
	case enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE:
		switch targetClass {
		case "runway":
			return outcomeRunwayCrater
		case "launcher":
			return outcomeMissionKill
		case "hardened_infrastructure", "civilian_energy", "civilian_water":
			return outcomeMissionKill
		case "soft_infrastructure":
			return outcomeCatastrophicKill
		case "surface_warship":
			return outcomeLightDamage
		default:
			return outcomeMissionKill
		}
	case enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_GUNFIRE:
		switch targetClass {
		case "aircraft":
			return outcomeCatastrophicKill
		case "armor":
			return outcomeLightDamage
		default:
			return outcomeLightDamage
		}
	case enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_UNSPECIFIED:
		switch targetClass {
		case "runway":
			return outcomeRunwayCrater
		case "launcher":
			return outcomeCatastrophicKill
		case "hardened_infrastructure", "civilian_energy", "civilian_water":
			return outcomeMissionKill
		case "sam_battery":
			return outcomeFirepowerLoss
		case "surface_warship":
			return outcomeLightDamage
		default:
			return outcomeLightDamage
		}
	default:
		return outcomeLightDamage
	}
}

func applyHitToUnit(target *enginev1.Unit, outcome impactOutcome) (destroyed bool, previous enginev1.DamageState) {
	previous = currentDamageState(target)
	switch outcome {
	case outcomeNoEffect:
		return false, previous
	case outcomeLightDamage:
		setOperationalFractions(target, 0.88, 0.9, 0.82)
		degradeFacilityOps(target)
		if previous == enginev1.DamageState_DAMAGE_STATE_DAMAGED {
			setDamageState(target, enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED)
			closeFacilityOps(target)
			return false, previous
		}
		if previous == enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED {
			return false, previous
		}
		setDamageState(target, enginev1.DamageState_DAMAGE_STATE_DAMAGED)
		return false, previous
	case outcomeMobilityKill, outcomeFirepowerLoss:
		setOperationalFractions(target, 0.72, 0.78, 0.55)
		degradeFacilityOps(target)
		if previous == enginev1.DamageState_DAMAGE_STATE_DAMAGED {
			setDamageState(target, enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED)
			closeFacilityOps(target)
			return false, previous
		}
		if previous == enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED {
			return false, previous
		}
		setDamageState(target, enginev1.DamageState_DAMAGE_STATE_DAMAGED)
		return false, previous
	case outcomeMissionKill, outcomeRunwayCrater:
		setOperationalFractions(target, 0.55, 0.65, 0.18)
		if previous == enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED {
			killUnit(target)
			return true, previous
		}
		setDamageState(target, enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED)
		closeFacilityOps(target)
		return false, previous
	case outcomeCatastrophicKill:
		closeFacilityOps(target)
		killUnit(target)
		return true, previous
	default:
		return false, previous
	}
}

func currentDamageState(u *enginev1.Unit) enginev1.DamageState {
	if u == nil || u.DamageState == enginev1.DamageState_DAMAGE_STATE_UNSPECIFIED {
		return enginev1.DamageState_DAMAGE_STATE_OPERATIONAL
	}
	return u.DamageState
}

func setDamageState(u *enginev1.Unit, state enginev1.DamageState) {
	if u != nil {
		u.DamageState = state
	}
}

func degradeFacilityOps(u *enginev1.Unit) {
	if u == nil || u.GetBaseOps() == nil {
		return
	}
	if u.GetBaseOps().GetState() == enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE {
		u.BaseOps.State = enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_DEGRADED
	}
}

func closeFacilityOps(u *enginev1.Unit) {
	if u == nil || u.GetBaseOps() == nil {
		return
	}
	u.BaseOps.State = enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_CLOSED
}

func setOperationalFractions(u *enginev1.Unit, personnel, equipment, combat float32) {
	if u == nil {
		return
	}
	if u.Status == nil {
		u.Status = &enginev1.OperationalStatus{IsActive: true}
	}
	u.Status.PersonnelStrength = minFloat32(u.Status.PersonnelStrength, personnel)
	u.Status.EquipmentStrength = minFloat32(u.Status.EquipmentStrength, equipment)
	u.Status.CombatEffectiveness = minFloat32(u.Status.CombatEffectiveness, combat)
}

func minFloat32(current, next float32) float32 {
	if current == 0 {
		return next
	}
	if next < current {
		return next
	}
	return current
}

func describeImpact(outcome impactOutcome) string {
	switch outcome {
	case outcomeLightDamage:
		return "damaged"
	case outcomeMobilityKill:
		return "mobility-killed"
	case outcomeFirepowerLoss:
		return "suppressed"
	case outcomeMissionKill:
		return "mission-killed"
	case outcomeRunwayCrater:
		return "runway-cratered"
	case outcomeCatastrophicKill:
		return "destroyed"
	default:
		return "unaffected"
	}
}
