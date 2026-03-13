package sim

import (
	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// DefStats holds the per-definition values the sim loop needs each tick.
type DefStats struct {
	CruiseSpeedMps  float64
	CombatRangeM    float64
	BaseStrength    float64
	DetectionRangeM float64
}

// Kill represents a single combat outcome from one engagement check.
// Attacker is nil for mutual annihilation (both units destroy each other).
type Kill struct {
	Attacker *enginev1.Unit // unit that won; nil = mutual
	Victim   *enginev1.Unit // unit destroyed
}

// AdjudicateTick checks all pairs of enemy active units. When a pair falls
// within the longer unit's combat range, the unit with the superior range
// destroys the other. If ranges are equal, higher base_strength wins; a tie
// annihilates both. Killed units are mutated in-place (IsActive = false,
// MoveOrder cleared) so subsequent ticks skip them automatically.
func AdjudicateTick(units []*enginev1.Unit, defs map[string]DefStats) []Kill {
	killed := make(map[string]bool)
	var results []Kill

	for i := 0; i < len(units); i++ {
		a := units[i]
		if !unitIsActive(a) || killed[a.Id] {
			continue
		}

		for j := i + 1; j < len(units); j++ {
			b := units[j]
			if !unitIsActive(b) || killed[b.Id] {
				continue
			}
			if a.Side == b.Side {
				continue // no friendly fire
			}

			dist := haversineM(
				a.GetPosition().GetLat(), a.GetPosition().GetLon(),
				b.GetPosition().GetLat(), b.GetPosition().GetLon(),
			)

			defA := defs[a.DefinitionId]
			defB := defs[b.DefinitionId]

			rangeA := defA.CombatRangeM
			rangeB := defB.CombatRangeM
			maxRange := max(rangeA, rangeB)

			if maxRange <= 0 || dist > maxRange {
				continue // out of engagement range
			}

			switch {
			case rangeA > rangeB:
				// A can engage; B is out of range — A wins.
				killUnit(b)
				killed[b.Id] = true
				results = append(results, Kill{Attacker: a, Victim: b})

			case rangeB > rangeA:
				// B can engage; A is out of range — B wins.
				killUnit(a)
				killed[a.Id] = true
				results = append(results, Kill{Attacker: b, Victim: a})

			default:
				// Equal range — resolve by base_strength; tie = mutual.
				switch {
				case defA.BaseStrength > defB.BaseStrength:
					killUnit(b)
					killed[b.Id] = true
					results = append(results, Kill{Attacker: a, Victim: b})
				case defB.BaseStrength > defA.BaseStrength:
					killUnit(a)
					killed[a.Id] = true
					results = append(results, Kill{Attacker: b, Victim: a})
				default:
					// Mutual annihilation.
					killUnit(a)
					killUnit(b)
					killed[a.Id] = true
					killed[b.Id] = true
					results = append(results,
						Kill{Attacker: nil, Victim: a},
						Kill{Attacker: nil, Victim: b},
					)
				}
			}
		}
	}
	return results
}

// ─── SENSOR DETECTION ─────────────────────────────────────────────────────────

// DetectionSet maps each detecting side to the full set of enemy unit IDs
// currently within sensor range of at least one unit on that side.
type DetectionSet map[string][]string

// SensorTick scans all active units and builds the current detection picture.
// For each detector unit, any enemy unit within its DetectionRangeM is added
// to the detector's side's contact set. Returns the full set for every side
// that has at least one active unit (empty slice = no contacts this tick).
func SensorTick(units []*enginev1.Unit, defs map[string]DefStats) DetectionSet {
	// Use intermediate sets to avoid duplicate entries.
	bySet := make(map[string]map[string]bool)

	// Collect all active sides so we can emit an empty update when a side
	// has no contacts (clearing stale frontend state).
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
// Treats nil Status as active (proto3 defaults bool to false, so we can't
// rely on GetIsActive() alone for units that haven't had status set yet).
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
