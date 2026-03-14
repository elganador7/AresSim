package sim

import (
	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// DefStats holds the per-definition values the sim loop needs each tick.
type DefStats struct {
	CruiseSpeedMps  float64
	BaseStrength    float64
	DetectionRangeM float64
	Domain          enginev1.UnitDomain // physical domain of this platform
}

// WeaponStats holds the per-weapon catalog data needed for engagement resolution
// and in-flight munition tracking.
type WeaponStats struct {
	RangeM           float64
	SpeedMps         float64 // projectile/missile speed; used for munition travel time
	ProbabilityOfHit float64
	DomainTargets    []enginev1.UnitDomain
}

// Rng is the minimal interface for probability rolls.
// *rand.Rand satisfies this interface; a deterministic stub is used in tests.
type Rng interface {
	Float64() float64
}

// Kill represents a single combat outcome resolved when a munition arrives.
// Attacker may be nil if the shooter was destroyed before the munition landed.
type Kill struct {
	Attacker *enginev1.Unit
	Victim   *enginev1.Unit
}

// FiredShot records a single weapon discharge during adjudication.
type FiredShot struct {
	Shooter        *enginev1.Unit
	Target         *enginev1.Unit
	WeaponID       string
	HitProbability float64 // range-degraded probability at fire time
}

// AdjudicateResult holds all shots fired in one tick of adjudication.
// Kills are NOT resolved here — they are deferred to when the in-flight
// munition arrives at its destination (see ResolveArrivals).
type AdjudicateResult struct {
	Shots []FiredShot
}

// AdjudicateTick checks all pairs of enemy active units and fires weapons for
// any unit within its best weapon's range. Each unit fires at most once per
// tick. Probability of hit is range-degraded at fire time and stored on the
// returned FiredShot for later resolution via ResolveArrivals.
//
// Units with no loadout, or with a loadout where no weapon can reach the
// target's domain, simply cannot engage — there is no fallback.
func AdjudicateTick(units []*enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats) AdjudicateResult {
	firedThisTick := make(map[string]bool)
	var result AdjudicateResult

	for i := 0; i < len(units); i++ {
		a := units[i]
		if !unitIsActive(a) {
			continue
		}

		for j := i + 1; j < len(units); j++ {
			b := units[j]
			if !unitIsActive(b) {
				continue
			}
			if a.Side == b.Side {
				continue
			}

			dist := haversineM(
				a.GetPosition().GetLat(), a.GetPosition().GetLon(),
				b.GetPosition().GetLat(), b.GetPosition().GetLon(),
			)

			defA := defs[a.DefinitionId]
			defB := defs[b.DefinitionId]

			wIDA, wA, hasWeapA := selectBestWeapon(a, defB.Domain, weapons)
			wIDB, wB, hasWeapB := selectBestWeapon(b, defA.Domain, weapons)

			aCanFire := hasWeapA && dist <= wA.RangeM && !firedThisTick[a.Id]
			bCanFire := hasWeapB && dist <= wB.RangeM && !firedThisTick[b.Id]

			if !aCanFire && !bCanFire {
				continue
			}

			if aCanFire {
				prob := rangeDegradedPoh(wA.ProbabilityOfHit, dist, wA.RangeM)
				decrementAmmo(a, wIDA)
				result.Shots = append(result.Shots, FiredShot{
					Shooter:        a,
					Target:         b,
					WeaponID:       wIDA,
					HitProbability: prob,
				})
				firedThisTick[a.Id] = true
			}

			if bCanFire {
				prob := rangeDegradedPoh(wB.ProbabilityOfHit, dist, wB.RangeM)
				decrementAmmo(b, wIDB)
				result.Shots = append(result.Shots, FiredShot{
					Shooter:        b,
					Target:         a,
					WeaponID:       wIDB,
					HitProbability: prob,
				})
				firedThisTick[b.Id] = true
			}

			if firedThisTick[a.Id] {
				break // A has fired; advance to the next outer unit
			}
		}
	}
	return result
}

// ResolveArrivals resolves kill outcomes for munitions that have reached their
// destinations. For each arrived munition, rng is rolled against its
// pre-computed HitProbability. Already-destroyed targets are safely skipped.
func ResolveArrivals(arrived []*InFlightMunition, units []*enginev1.Unit, rng Rng) []Kill {
	if len(arrived) == 0 {
		return nil
	}
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, u := range units {
		unitByID[u.Id] = u
	}

	var kills []Kill
	for _, m := range arrived {
		if m.TargetID == "" {
			continue
		}
		target := unitByID[m.TargetID]
		if target == nil || !unitIsActive(target) {
			continue // already destroyed before munition arrived
		}
		if rng.Float64() < m.HitProbability {
			killUnit(target)
			kills = append(kills, Kill{
				Attacker: unitByID[m.ShooterID],
				Victim:   target,
			})
		}
	}
	return kills
}

// rangeDegradedPoh returns the base probability of hit scaled by a linear
// range factor. At dist=0 the full basePoh applies; at dist=rangeM the
// probability is reduced to 30% of basePoh, reflecting the difficulty of
// engaging a target at maximum effective range.
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

// selectBestWeapon finds the highest-range weapon on unit that can target
// targetDomain and has ammo remaining.
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

// canTargetDomain returns true if the given domain is in the targets slice.
func canTargetDomain(targets []enginev1.UnitDomain, d enginev1.UnitDomain) bool {
	for _, t := range targets {
		if t == d {
			return true
		}
	}
	return false
}

// decrementAmmo reduces the current quantity of weaponID on shooter by 1.
func decrementAmmo(shooter *enginev1.Unit, weaponID string) {
	for _, ws := range shooter.Weapons {
		if ws.WeaponId == weaponID && ws.CurrentQty > 0 {
			ws.CurrentQty--
			return
		}
	}
}

// ─── SENSOR DETECTION ─────────────────────────────────────────────────────────

// DetectionSet maps each detecting side to the full set of enemy unit IDs
// currently within sensor range of at least one unit on that side.
type DetectionSet map[string][]string

// SensorTick scans all active units and builds the current detection picture.
func SensorTick(units []*enginev1.Unit, defs map[string]DefStats) DetectionSet {
	bySet := make(map[string]map[string]bool)

	for _, u := range units {
		if unitIsActive(u) {
			if bySet[u.Side] == nil {
				bySet[u.Side] = make(map[string]bool)
			}
		}
	}

	for _, detector := range units {
		if !unitIsActive(detector) {
			continue
		}
		rangeM := defs[detector.DefinitionId].DetectionRangeM
		if rangeM <= 0 {
			continue
		}
		for _, target := range units {
			if !unitIsActive(target) || target.Side == detector.Side {
				continue
			}
			dist := haversineM(
				detector.GetPosition().GetLat(), detector.GetPosition().GetLon(),
				target.GetPosition().GetLat(), target.GetPosition().GetLon(),
			)
			if dist <= rangeM {
				bySet[detector.Side][target.Id] = true
			}
		}
	}

	result := make(DetectionSet, len(bySet))
	for side, ids := range bySet {
		list := make([]string, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		result[side] = list
	}
	return result
}

// unitIsActive returns true if u is not yet destroyed.
func unitIsActive(u *enginev1.Unit) bool {
	if u.Status == nil {
		return true
	}
	return u.Status.IsActive
}

// killUnit marks u as destroyed and clears its move order in-place.
func killUnit(u *enginev1.Unit) {
	if u.Status == nil {
		u.Status = &enginev1.OperationalStatus{}
	}
	u.Status.IsActive = false
	u.MoveOrder = nil
}
