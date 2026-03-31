package sim

import (
	"math"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// AdjudicateTick checks all pairs of enemy active units and fires a salvo for
// each unit that meets the engagement criteria. Each unit fires at most one
// salvo per tick (at its highest-priority target).
//
// Salvo sizing — the minimum number of rounds N such that the cumulative
// kill probability of all munitions (in-flight + new salvo) exceeds 70 %.
// Already in-flight munitions targeting the same unit are counted, so platforms
// do not keep firing after enough rounds are already on the way.
func AdjudicateTick(units []*enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats, inFlight []*InFlightMunition, rules RelationshipRules, simSeconds float64) AdjudicateResult {
	firedThisTick := make(map[string]bool)
	orderedUnits := make(map[string]bool)
	var result AdjudicateResult
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, u := range units {
		unitByID[u.Id] = u
	}

	for _, shooter := range units {
		if !unitCanOperate(shooter) {
			continue
		}
		order := shooter.GetAttackOrder()
		if order == nil || order.GetOrderType() == enginev1.AttackOrderType_ATTACK_ORDER_TYPE_UNSPECIFIED || order.GetTargetUnitId() == "" {
			continue
		}
		orderedUnits[shooter.Id] = true
		target := unitByID[order.GetTargetUnitId()]
		if target == nil || !unitIsAlive(target) || !unitsAreHostile(shooter, target) {
			continue
		}
		if order.GetOrderType() == enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT &&
			desiredEffectSatisfied(target, order.GetDesiredEffect()) {
			continue
		}
		if tryFireAtTarget(
			shooter,
			target,
			defs,
			weapons,
			inFlight,
			firedThisTick,
			order.GetPkillThreshold(),
			order.GetDesiredEffect(),
			order.GetOrderType() == enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT,
			simSeconds,
			&result,
		) {
			continue
		}
	}

	for i := 0; i < len(units); i++ {
		a := units[i]
		if !unitCanOperate(a) {
			continue
		}

		for j := i + 1; j < len(units); j++ {
			b := units[j]
			if !unitCanOperate(b) {
				continue
			}
			defA := defs[a.DefinitionId]
			defB := defs[b.DefinitionId]

			aCanEngageB := unitsAreHostile(a, b) || (isUnauthorizedOverflight(a, b, defs, rules) && canPerformSovereignAirDefense(a, defA))
			bCanEngageA := unitsAreHostile(a, b) || (isUnauthorizedOverflight(b, a, defs, rules) && canPerformSovereignAirDefense(b, defB))
			var decisionA EngagementDecision
			var decisionB EngagementDecision
			aCanAct := aCanEngageB && !firedThisTick[a.Id] && !orderedUnits[a.Id]
			bCanAct := bCanEngageA && !firedThisTick[b.Id] && !orderedUnits[b.Id]
			if aCanAct {
				decisionA = EvaluateAutonomousEngagementDecision(a, b, defs, weapons, simSeconds)
			}
			if bCanAct {
				decisionB = EvaluateAutonomousEngagementDecision(b, a, defs, weapons, simSeconds)
			}

			if !decisionA.CanFire && !decisionB.CanFire {
				continue
			}

			if decisionA.CanFire {
				miss := inFlightMissProb(inFlight, b.Id)
				salvo := salvoToAchieveKillProb(miss, decisionA.FireProbability, 0.30)
				salvo = capAtAmmo(a, decisionA.WeaponID, salvo)
				if salvo > 0 {
					decrementAmmo(a, decisionA.WeaponID, salvo)
					applyStrikeCooldown(a, decisionA.Weapon, simSeconds)
					result.Shots = append(result.Shots, FiredShot{
						Shooter:        a,
						Target:         b,
						WeaponID:       decisionA.WeaponID,
						HitProbability: decisionA.FireProbability,
						SalvoSize:      salvo,
					})
					firedThisTick[a.Id] = true
				}
			}

			if decisionB.CanFire {
				miss := inFlightMissProb(inFlight, a.Id)
				salvo := salvoToAchieveKillProb(miss, decisionB.FireProbability, 0.30)
				salvo = capAtAmmo(b, decisionB.WeaponID, salvo)
				if salvo > 0 {
					decrementAmmo(b, decisionB.WeaponID, salvo)
					applyStrikeCooldown(b, decisionB.Weapon, simSeconds)
					result.Shots = append(result.Shots, FiredShot{
						Shooter:        b,
						Target:         a,
						WeaponID:       decisionB.WeaponID,
						HitProbability: decisionB.FireProbability,
						SalvoSize:      salvo,
					})
					firedThisTick[b.Id] = true
				}
			}

			if firedThisTick[a.Id] {
				break
			}
		}
	}
	return result
}

func tryFireAtTarget(
	shooter, target *enginev1.Unit,
	defs map[string]DefStats,
	weapons map[string]WeaponStats,
	inFlight []*InFlightMunition,
	firedThisTick map[string]bool,
	pkillThreshold float32,
	desiredEffect enginev1.DesiredEffect,
	requireDesiredEffect bool,
	simSeconds float64,
	result *AdjudicateResult,
) bool {
	if firedThisTick[shooter.Id] || shooter == nil || target == nil {
		return false
	}
	decision := EvaluateEngagementDecision(
		shooter,
		target,
		defs,
		weapons,
		desiredEffect,
		requireDesiredEffect,
		simSeconds,
	)
	if !decision.CanFire {
		return false
	}
	targetMissProb := 0.30
	if pkillThreshold > 0 && pkillThreshold < 1 {
		targetMissProb = 1.0 - float64(pkillThreshold)
	}
	miss := inFlightMissProb(inFlight, target.Id)
	salvo := salvoToAchieveKillProb(miss, decision.FireProbability, targetMissProb)
	salvo = capAtAmmo(shooter, decision.WeaponID, salvo)
	if salvo <= 0 {
		return false
	}
	decrementAmmo(shooter, decision.WeaponID, salvo)
	applyStrikeCooldown(shooter, decision.Weapon, simSeconds)
	result.Shots = append(result.Shots, FiredShot{
		Shooter:        shooter,
		Target:         target,
		WeaponID:       decision.WeaponID,
		HitProbability: decision.FireProbability,
		SalvoSize:      salvo,
	})
	firedThisTick[shooter.Id] = true
	return true
}

func unitReadyToStrike(unit *enginev1.Unit, weapon WeaponStats, simSeconds float64) bool {
	if unit == nil || !weaponUsesStrikeCadence(unit, weapon) {
		return true
	}
	return unit.GetNextStrikeReadySeconds() <= simSeconds
}

func canExecutePreplannedStrategicStrike(shooter, target *enginev1.Unit, targetDef DefStats, weapon WeaponStats) bool {
	if shooter == nil || target == nil {
		return false
	}
	if !weaponUsesStrikeCadence(shooter, weapon) {
		return false
	}
	if !isFixedStrategicTarget(target, targetDef) {
		return false
	}
	return true
}

func weaponUsesStrikeCadence(unit *enginev1.Unit, weapon WeaponStats) bool {
	if unit == nil || unit.GetPosition() == nil {
		return false
	}
	if weapon.EffectType != enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE &&
		weapon.EffectType != enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE {
		return false
	}
	if weapon.RangeM < 100_000 {
		return false
	}
	return unit.GetPosition().GetAltMsl() <= 0
}

func isFixedStrategicTarget(target *enginev1.Unit, def DefStats) bool {
	if target == nil {
		return false
	}
	if def.AssetClass == "airbase" || def.AssetClass == "port" {
		return true
	}
	if def.TargetClass == "runway" ||
		def.TargetClass == "hardened_infrastructure" ||
		def.TargetClass == "soft_infrastructure" ||
		def.TargetClass == "civilian_energy" ||
		def.TargetClass == "civilian_water" {
		return true
	}
	return false
}

func IsFixedStrategicTargetForUI(target *enginev1.Unit, def DefStats) bool {
	return isFixedStrategicTarget(target, def)
}

func applyStrikeCooldown(unit *enginev1.Unit, weapon WeaponStats, simSeconds float64) {
	if unit == nil || !weaponUsesStrikeCadence(unit, weapon) {
		return
	}
	cooldown := 3600.0
	if currentDamageState(unit) == enginev1.DamageState_DAMAGE_STATE_DAMAGED {
		cooldown = 7200.0
	}
	unit.NextStrikeReadySeconds = simSeconds + cooldown
}

func shouldAutonomouslyEngage(unit *enginev1.Unit, prob float64, detectedByTarget bool) bool {
	switch unit.GetEngagementBehavior() {
	case enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_HOLD_FIRE,
		enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_ASSIGNED_TARGETS_ONLY,
		enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_SHADOW_CONTACT,
		enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_WITHDRAW_ON_DETECT:
		return false
	case enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_SELF_DEFENSE_ONLY:
		return detectedByTarget
	case enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_AUTO_ENGAGE,
		enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_UNSPECIFIED:
		return prob >= pkillThresholdForUnit(unit) || detectedByTarget
	default:
		return prob >= pkillThresholdForUnit(unit) || detectedByTarget
	}
}

func shouldExecuteManualAttack(unit *enginev1.Unit, prob float64, detectedByTarget bool) bool {
	behavior := unit.GetEngagementBehavior()
	if behavior == enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_HOLD_FIRE {
		return false
	}
	return prob >= pkillThresholdForUnit(unit) || detectedByTarget
}

func pkillThresholdForUnit(unit *enginev1.Unit) float64 {
	threshold := float64(unit.GetEngagementPkillThreshold())
	if threshold <= 0 {
		return 0.50
	}
	if threshold >= 1 {
		return 0.99
	}
	return threshold
}

func desiredEffectSatisfied(target *enginev1.Unit, desired enginev1.DesiredEffect) bool {
	damage := currentDamageState(target)
	switch desired {
	case enginev1.DesiredEffect_DESIRED_EFFECT_DAMAGE:
		return damage >= enginev1.DamageState_DAMAGE_STATE_DAMAGED
	case enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL:
		return damage >= enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED
	case enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY, enginev1.DesiredEffect_DESIRED_EFFECT_UNSPECIFIED:
		return !unitIsAlive(target)
	default:
		return false
	}
}

func impactOutcomeSupportsDesiredEffect(outcome impactOutcome, desired enginev1.DesiredEffect) bool {
	switch desired {
	case enginev1.DesiredEffect_DESIRED_EFFECT_DAMAGE:
		return outcome != outcomeNoEffect
	case enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL:
		return outcome == outcomeMissionKill || outcome == outcomeRunwayCrater || outcome == outcomeCatastrophicKill
	case enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY:
		return outcome == outcomeMissionKill || outcome == outcomeCatastrophicKill
	case enginev1.DesiredEffect_DESIRED_EFFECT_UNSPECIFIED:
		return outcome != outcomeNoEffect
	default:
		return false
	}
}

func inFlightMissProb(inFlight []*InFlightMunition, targetID string) float64 {
	p := 1.0
	for _, m := range inFlight {
		if m.TargetID == targetID {
			p *= 1.0 - m.HitProbability
		}
	}
	return p
}

func salvoToAchieveKillProb(existingMissProb, singleShotPoh, targetMissProb float64) int32 {
	if existingMissProb <= targetMissProb {
		return 0
	}
	if singleShotPoh <= 0 {
		return 0
	}
	if singleShotPoh >= 1.0 {
		return 1
	}
	n := math.Log(targetMissProb/existingMissProb) / math.Log(1.0-singleShotPoh)
	result := int32(math.Ceil(n))
	if result < 1 {
		result = 1
	}
	return result
}

func capAtAmmo(unit *enginev1.Unit, weaponID string, requested int32) int32 {
	for _, ws := range unit.Weapons {
		if ws.WeaponId == weaponID {
			if ws.CurrentQty <= 0 {
				return 0
			}
			if requested > ws.CurrentQty {
				return ws.CurrentQty
			}
			return requested
		}
	}
	return 0
}

func decrementAmmo(shooter *enginev1.Unit, weaponID string, amount int32) {
	if amount <= 0 {
		return
	}
	for _, ws := range shooter.Weapons {
		if ws.WeaponId == weaponID && ws.CurrentQty > 0 {
			if amount >= ws.CurrentQty {
				ws.CurrentQty = 0
				return
			}
			ws.CurrentQty -= amount
			return
		}
	}
}
