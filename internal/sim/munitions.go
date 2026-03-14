package sim

import (
	"fmt"
	"sync/atomic"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// InFlightMunition represents a weapon in transit between a shooter and a
// target. Kill resolution is deferred until the munition arrives at its
// destination (see ResolveArrivals). HitProbability is range-degraded at fire
// time and carried here so the roll can be made on arrival.
type InFlightMunition struct {
	ID             string
	WeaponID       string
	ShooterID      string
	TargetID       string  // unit ID of the intended target
	HitProbability float64 // range-degraded PoH computed at fire time
	CurLat         float64
	CurLon         float64
	DestLat        float64
	DestLon        float64
	SpeedMps       float64 // metres per sim-second
	TargetDomains  []enginev1.UnitDomain
}

var munitionSeq atomic.Int64

// NextMunitionID returns a unique, compact ID for a new in-flight munition.
func NextMunitionID() string {
	return fmt.Sprintf("mun-%d", munitionSeq.Add(1))
}

// AdvanceMunitions moves each munition one tick toward its destination.
// Munitions that have arrived are removed and returned as the second value.
func AdvanceMunitions(munitions []*InFlightMunition, timeScale float64) (remaining, arrived []*InFlightMunition) {
	for _, m := range munitions {
		dist := haversineM(m.CurLat, m.CurLon, m.DestLat, m.DestLon)
		canMove := m.SpeedMps * timeScale // metres this tick
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
