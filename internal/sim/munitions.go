package sim

import (
	"fmt"
	"sync/atomic"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// InFlightMunition represents a weapon in transit between a shooter and a
// target. Kill resolution is deferred until the munition arrives at its
// destination (see ResolveArrivals).
//
// HitProbability is range-degraded at fire time and carried here so the kill
// roll can be made on arrival.
//
// Guidance determines whether the munition updates its destination each tick
// (tracking) or flies to the fixed point set at launch.
type InFlightMunition struct {
	ID             string
	WeaponID       string
	ShooterID      string
	ShooterSide    string // side of the firing unit; used for radar lock check
	TargetID       string // unit ID of the intended target
	HitProbability float64
	CurLat         float64
	CurLon         float64
	DestLat        float64
	DestLon        float64
	SpeedMps       float64             // metres per sim-second
	TargetDomains  []enginev1.UnitDomain
	Guidance       enginev1.GuidanceType
}

var munitionSeq atomic.Int64

// NextMunitionID returns a unique, compact ID for a new in-flight munition.
func NextMunitionID() string {
	return fmt.Sprintf("mun-%d", munitionSeq.Add(1))
}

// AdvanceMunitions moves each munition one tick toward its destination.
//
// For tracking guidance types (IR, wire, sonar), the destination is updated
// each tick to the target's current position. For radar-guided munitions, the
// destination is only updated while the shooter's side still detects the
// target (passed in via detections). GPS, laser, unguided, and unspecified
// munitions fly to a fixed point set at launch time.
//
// Munitions that have arrived (distance ≤ this tick's travel) are removed and
// returned as the second value.
func AdvanceMunitions(
	munitions []*InFlightMunition,
	timeScale float64,
	units []*enginev1.Unit,
	detections DetectionSet,
) (remaining, arrived []*InFlightMunition) {
	if len(munitions) == 0 {
		return
	}

	// Build O(1) lookup helpers only when there are tracking munitions.
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, u := range units {
		unitByID[u.Id] = u
	}
	detectSet := make(map[string]map[string]bool, len(detections))
	for side, ids := range detections {
		set := make(map[string]bool, len(ids))
		for _, id := range ids {
			set[id] = true
		}
		detectSet[side] = set
	}

	for _, m := range munitions {
		updateMunitionDestination(m, unitByID, detectSet)

		dist := haversineM(m.CurLat, m.CurLon, m.DestLat, m.DestLon)
		canMove := m.SpeedMps * timeScale
		if canMove >= dist {
			arrived = append(arrived, m)
			continue
		}
		brng := bearingRad(m.CurLat, m.CurLon, m.DestLat, m.DestLon)
		m.CurLat, m.CurLon = movePoint(m.CurLat, m.CurLon, brng, canMove)
		remaining = append(remaining, m)
	}
	return
}

// updateMunitionDestination updates the destination of a tracking munition.
//
//   - RADAR: tracks while shooter's side maintains detection of the target.
//     If lock is lost the munition continues to the last known position.
//   - IR, WIRE, SONAR: always update to the current target position (fire-and-forget
//     or self-guided seekers cannot be jammed by radar-off manoeuvres).
//   - GPS, LASER, UNGUIDED, UNSPECIFIED: fixed-point — no update.
func updateMunitionDestination(
	m *InFlightMunition,
	unitByID map[string]*enginev1.Unit,
	detectSet map[string]map[string]bool,
) {
	switch m.Guidance {
	case enginev1.GuidanceType_GUIDANCE_IR,
		enginev1.GuidanceType_GUIDANCE_WIRE,
		enginev1.GuidanceType_GUIDANCE_SONAR:
		// Always track — update to target's current position.
		if t := unitByID[m.TargetID]; t != nil && unitIsActive(t) {
			m.DestLat = t.GetPosition().GetLat()
			m.DestLon = t.GetPosition().GetLon()
		}

	case enginev1.GuidanceType_GUIDANCE_RADAR:
		// Track only while the shooter's side still has a detection on the target.
		if detectSet[m.ShooterSide][m.TargetID] {
			if t := unitByID[m.TargetID]; t != nil && unitIsActive(t) {
				m.DestLat = t.GetPosition().GetLat()
				m.DestLon = t.GetPosition().GetLon()
			}
		}
		// Lock lost → fly to last known position (no update).

	default:
		// UNSPECIFIED, UNGUIDED, GPS, LASER — fixed point, nothing to update.
	}
}

// DetectMunitions determines which in-flight munitions each side can currently
// detect. A platform can detect a munition when:
//  1. The platform's domain is one of the munition's target domains (sensors
//     appropriate for that type of threat), and
//  2. The munition is within the platform's detection range.
//
// Returns a map of side → deduplicated slice of detected munition IDs.
func DetectMunitions(units []*enginev1.Unit, defs map[string]DefStats, munitions []*InFlightMunition) map[string][]string {
	if len(munitions) == 0 {
		return nil
	}

	type sideSet = map[string]bool
	bySide := make(map[string]sideSet)

	for _, detector := range units {
		if !unitIsActive(detector) {
			continue
		}
		def := defs[detector.DefinitionId]
		if def.DetectionRangeM <= 0 {
			continue
		}
		if bySide[detector.Side] == nil {
			bySide[detector.Side] = make(sideSet)
		}
		detLat := detector.GetPosition().GetLat()
		detLon := detector.GetPosition().GetLon()

		for _, m := range munitions {
			// Sensor must be in a domain that the munition targets.
			if !canTargetDomain(m.TargetDomains, def.Domain) {
				continue
			}
			dist := haversineM(detLat, detLon, m.CurLat, m.CurLon)
			if dist <= def.DetectionRangeM {
				bySide[detector.Side][m.ID] = true
			}
		}
	}

	result := make(map[string][]string, len(bySide))
	for side, ids := range bySide {
		list := make([]string, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		result[side] = list
	}
	return result
}
