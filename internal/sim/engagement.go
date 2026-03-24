package sim

import (
	"fmt"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

type EngagementReason string

const (
	EngagementReasonNoTarget              EngagementReason = "no_target"
	EngagementReasonNoWeapon              EngagementReason = "no_weapon"
	EngagementReasonOutOfRange            EngagementReason = "out_of_range"
	EngagementReasonStrikeCooldown        EngagementReason = "strike_cooldown"
	EngagementReasonDesiredEffectMismatch EngagementReason = "desired_effect_mismatch"
	EngagementReasonDoctrineThreshold     EngagementReason = "doctrine_threshold"
	EngagementReasonHoldFire              EngagementReason = "hold_fire"
	EngagementReasonReady                 EngagementReason = "ready"
)

type EngagementDecision struct {
	WeaponID             string
	Weapon               WeaponStats
	RangeToTargetM       float64
	WeaponRangeM         float64
	FireProbability      float64
	DesiredEffectSupport bool
	InStrikeCooldown     bool
	CanFire              bool
	Reason               EngagementReason
}

func (d EngagementDecision) ReasonText() string {
	switch d.Reason {
	case EngagementReasonNoTarget:
		return "No target assigned."
	case EngagementReasonNoWeapon:
		return "No loaded weapon can effectively engage this target."
	case EngagementReasonOutOfRange:
		return fmt.Sprintf("Target is outside weapon range (%.0f km / %.0f km).", d.RangeToTargetM/1000, d.WeaponRangeM/1000)
	case EngagementReasonStrikeCooldown:
		return "Unit is waiting on strike cooldown."
	case EngagementReasonDesiredEffectMismatch:
		return "Loaded weapons are a poor match for the requested effect."
	case EngagementReasonDoctrineThreshold:
		return "Shot withheld by current engagement threshold."
	case EngagementReasonHoldFire:
		return "Unit is set to hold fire."
	case EngagementReasonReady:
		return "Ready to fire."
	default:
		return ""
	}
}

type weaponCandidate struct {
	weaponID             string
	weapon               WeaponStats
	outcome              impactOutcome
	desiredEffectSupport bool
	score                float64
}

func scoreWeaponCandidate(weapon WeaponStats, outcome impactOutcome, desiredEffect enginev1.DesiredEffect) float64 {
	score := weapon.RangeM/1000 + weapon.ProbabilityOfHit*100
	if outcome == outcomeNoEffect {
		return -1
	}
	if impactOutcomeSupportsDesiredEffect(outcome, desiredEffect) {
		score += 500
	}
	switch outcome {
	case outcomeCatastrophicKill:
		score += 40
	case outcomeMissionKill, outcomeRunwayCrater:
		score += 24
	case outcomeFirepowerLoss, outcomeMobilityKill:
		score += 12
	case outcomeLightDamage:
		score += 4
	}
	return score
}

func selectWeaponForEngagement(unit *enginev1.Unit, targetDef DefStats, desiredEffect enginev1.DesiredEffect, catalog map[string]WeaponStats) (weaponID string, stats WeaponStats, desiredEffectSupport bool, found bool) {
	bestScore := -1.0
	bestEffectSupport := false
	for _, ws := range unit.Weapons {
		if ws.CurrentQty <= 0 {
			continue
		}
		wdef, ok := catalog[ws.WeaponId]
		if !ok || !canTargetDomain(wdef.DomainTargets, targetDef.Domain) {
			continue
		}
		outcome := resolveImpactOutcome(wdef.EffectType, targetDef.TargetClass)
		score := scoreWeaponCandidate(wdef, outcome, desiredEffect)
		if score < 0 {
			continue
		}
		supports := impactOutcomeSupportsDesiredEffect(outcome, desiredEffect)
		if score > bestScore || (score == bestScore && supports && !bestEffectSupport) {
			bestScore = score
			bestEffectSupport = supports
			weaponID = ws.WeaponId
			stats = wdef
			found = true
			desiredEffectSupport = supports
		}
	}
	return
}

func EvaluateEngagementDecision(
	shooter, target *enginev1.Unit,
	defs map[string]DefStats,
	weapons map[string]WeaponStats,
	desiredEffect enginev1.DesiredEffect,
	requireDesiredEffect bool,
	simSeconds float64,
) EngagementDecision {
	decision := EngagementDecision{Reason: EngagementReasonNoTarget}
	if shooter == nil || target == nil {
		return decision
	}
	targetDef := definitionStatsFor(defs, target.DefinitionId)
	weaponID, weapon, desiredEffectSupport, hasWeapon := selectWeaponForEngagement(shooter, targetDef, desiredEffect, weapons)
	if !hasWeapon {
		decision.Reason = EngagementReasonNoWeapon
		return decision
	}
	decision.WeaponID = weaponID
	decision.Weapon = weapon
	decision.WeaponRangeM = weapon.RangeM
	decision.DesiredEffectSupport = desiredEffectSupport
	if !unitReadyToStrike(shooter, weapon, simSeconds) {
		decision.InStrikeCooldown = true
		decision.Reason = EngagementReasonStrikeCooldown
		return decision
	}
	if requireDesiredEffect && !desiredEffectSupport {
		decision.Reason = EngagementReasonDesiredEffectMismatch
		return decision
	}
	dist := haversineM(
		shooter.GetPosition().GetLat(), shooter.GetPosition().GetLon(),
		target.GetPosition().GetLat(), target.GetPosition().GetLon(),
	)
	decision.RangeToTargetM = dist
	if dist > weapon.RangeM {
		decision.Reason = EngagementReasonOutOfRange
		return decision
	}
	decision.FireProbability = launchKillProbability(weapon, targetDef, dist)
	switch shooter.GetEngagementBehavior() {
	case enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_HOLD_FIRE:
		decision.Reason = EngagementReasonHoldFire
		return decision
	}
	decision.CanFire = true
	decision.Reason = EngagementReasonReady
	return decision
}

func EvaluateCurrentEngagement(
	shooter, target *enginev1.Unit,
	units []*enginev1.Unit,
	defs map[string]DefStats,
	weapons map[string]WeaponStats,
	rules RelationshipRules,
	desiredEffect enginev1.DesiredEffect,
	requireDesiredEffect bool,
	simSeconds float64,
) EngagementDecision {
	_ = units
	_ = rules
	return EvaluateEngagementDecision(shooter, target, defs, weapons, desiredEffect, requireDesiredEffect, simSeconds)
}

func EvaluateAutonomousEngagementDecision(
	shooter, target *enginev1.Unit,
	defs map[string]DefStats,
	weapons map[string]WeaponStats,
	simSeconds float64,
) EngagementDecision {
	decision := EngagementDecision{Reason: EngagementReasonNoTarget}
	if shooter == nil || target == nil {
		return decision
	}
	targetDef := definitionStatsFor(defs, target.DefinitionId)
	weaponID, weapon, _, hasWeapon := selectWeaponForEngagement(shooter, targetDef, enginev1.DesiredEffect_DESIRED_EFFECT_UNSPECIFIED, weapons)
	if !hasWeapon {
		decision.Reason = EngagementReasonNoWeapon
		return decision
	}
	decision.WeaponID = weaponID
	decision.Weapon = weapon
	decision.WeaponRangeM = weapon.RangeM
	if !unitReadyToStrike(shooter, weapon, simSeconds) {
		decision.InStrikeCooldown = true
		decision.Reason = EngagementReasonStrikeCooldown
		return decision
	}
	dist := haversineM(
		shooter.GetPosition().GetLat(), shooter.GetPosition().GetLon(),
		target.GetPosition().GetLat(), target.GetPosition().GetLon(),
	)
	decision.RangeToTargetM = dist
	if dist > weapon.RangeM {
		decision.Reason = EngagementReasonOutOfRange
		return decision
	}
	decision.FireProbability = launchKillProbability(weapon, targetDef, dist)
	switch shooter.GetEngagementBehavior() {
	case enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_HOLD_FIRE,
		enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_ASSIGNED_TARGETS_ONLY,
		enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_SHADOW_CONTACT,
		enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_WITHDRAW_ON_DETECT:
		decision.Reason = EngagementReasonHoldFire
		return decision
	}
	decision.CanFire = true
	decision.Reason = EngagementReasonReady
	return decision
}
