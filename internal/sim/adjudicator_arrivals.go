package sim

import (
	"math"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func ResolveArrivals(arrived []*InFlightMunition, units []*enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats, rng Rng) []HitResult {
	if len(arrived) == 0 {
		return nil
	}
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, u := range units {
		unitByID[u.Id] = u
	}

	var results []HitResult
	for _, m := range arrived {
		if m.TargetID == "" {
			continue
		}
		target := unitByID[m.TargetID]
		if target == nil || !unitIsAlive(target) {
			continue
		}
		if rng.Float64() < m.HitProbability {
			outcome := resolveImpactOutcome(weapons[m.WeaponID].EffectType, effectiveTargetClass(definitionStatsFor(defs, target.DefinitionId)))
			if outcome == outcomeNoEffect {
				continue
			}
			destroyed, previous := applyHitToUnit(target, outcome)
			results = append(results, HitResult{
				Attacker:      unitByID[m.ShooterID],
				Victim:        target,
				Outcome:       outcome,
				Destroyed:     destroyed,
				PreviousState: previous,
			})
		}
	}
	return results
}

func rangeDegradedPoh(basePoh, dist, rangeM float64) float64 {
	if rangeM <= 0 {
		return basePoh
	}
	factor := 1.0 - 0.7*(dist/rangeM)
	if factor < 0.3 {
		factor = 0.3
	}
	return basePoh * factor
}

func effectiveDetectionRangeM(detector, target DefStats) float64 {
	if detector.DetectionRangeM <= 0 {
		return 0
	}
	rcs := target.RadarCrossSectionM2
	if rcs <= 0 {
		return detector.DetectionRangeM
	}
	factor := math.Pow(rcs, 0.25)
	if factor < 0.25 {
		factor = 0.25
	}
	if factor > 2.0 {
		factor = 2.0
	}
	return detector.DetectionRangeM * factor
}

func selectBestWeapon(unit *enginev1.Unit, targetDomain enginev1.UnitDomain, catalog map[string]WeaponStats) (weaponID string, stats WeaponStats, found bool) {
	bestRange := -1.0
	for _, ws := range unit.Weapons {
		if ws.CurrentQty <= 0 {
			continue
		}
		wdef, ok := catalog[ws.WeaponId]
		if !ok {
			continue
		}
		if !canTargetDomain(wdef.DomainTargets, targetDomain) {
			continue
		}
		if wdef.RangeM > bestRange {
			bestRange = wdef.RangeM
			weaponID = ws.WeaponId
			stats = wdef
			found = true
		}
	}
	return
}

func canTargetDomain(targets []enginev1.UnitDomain, d enginev1.UnitDomain) bool {
	for _, t := range targets {
		if t == d {
			return true
		}
	}
	return false
}
