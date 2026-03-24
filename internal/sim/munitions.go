package sim

import (
	"fmt"
	"sync/atomic"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/geo"
)

// InFlightMunition represents a weapon in transit between a shooter and a
// target. Kill resolution is deferred until the munition arrives at its
// destination (see ResolveArrivals).
//
// HitProbability is range-degraded at fire time and carried here so the kill
// roll can be made on arrival.
//
// Guidance determines whether the munition updates its destination each tick
// or flies to the target's current position each tick.
type InFlightMunition struct {
	ID             string
	WeaponID       string
	ShooterID      string
	ShooterTeam    string // firing team / country code
	TargetID       string // unit ID of the intended target
	HitProbability float64
	LaunchLat      float64
	LaunchLon      float64
	MaxRangeM      float64
	CurLat         float64
	CurLon         float64
	CurAltMsl      float64
	DestLat        float64
	DestLon        float64
	DestAltMsl     float64
	SpeedMps       float64 // metres per sim-second
	TargetDomains  []enginev1.UnitDomain
}

type InterceptShot struct {
	Defender *enginev1.Unit
	Munition *InFlightMunition
	WeaponID string
}

var munitionSeq atomic.Int64

// NextMunitionID returns a unique, compact ID for a new in-flight munition.
func NextMunitionID() string {
	return fmt.Sprintf("mun-%d", munitionSeq.Add(1))
}

// AdvanceMunitions moves each munition one tick toward its destination.
//
// All munitions directly chase the target's current position while it remains
// within the weapon's maximum range from the launch point. This intentionally
// avoids simulating seeker-management complexity.
//
// Munitions that have arrived (distance ≤ this tick's travel) are removed and
// returned as the second value.
func AdvanceMunitions(
	munitions []*InFlightMunition,
	timeScale float64,
	units []*enginev1.Unit,
	defs map[string]DefStats,
) (remaining, arrived []*InFlightMunition) {
	if len(munitions) == 0 {
		return
	}

	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, u := range units {
		unitByID[u.Id] = u
	}

	for _, m := range munitions {
		if target := unitByID[m.TargetID]; target != nil && unitIsAlive(target) {
			targetPos := target.GetPosition()
			m.DestLat = targetPos.GetLat()
			m.DestLon = targetPos.GetLon()
			m.DestAltMsl = targetPos.GetAltMsl()
			if m.MaxRangeM > 0 {
				targetFromLaunch := haversineM(m.LaunchLat, m.LaunchLon, targetPos.GetLat(), targetPos.GetLon())
				if targetFromLaunch > m.MaxRangeM {
					continue
				}
			}
		}

		dist := haversineM(m.CurLat, m.CurLon, m.DestLat, m.DestLon)
		canMove := m.SpeedMps * timeScale
		if canMove >= dist {
			m.CurLat = m.DestLat
			m.CurLon = m.DestLon
			m.CurAltMsl = m.DestAltMsl
			arrived = append(arrived, m)
			continue
		}
		brng := bearingRad(m.CurLat, m.CurLon, m.DestLat, m.DestLon)
		fraction := canMove / dist
		m.CurLat, m.CurLon = movePoint(m.CurLat, m.CurLon, brng, canMove)
		m.CurAltMsl += (m.DestAltMsl - m.CurAltMsl) * fraction
		remaining = append(remaining, m)
	}
	return
}

// DetectMunitions determines which in-flight munitions each team can currently
// detect. A platform can detect a munition when:
//  1. The platform's domain is one of the munition's target domains (sensors
//     appropriate for that type of threat), and
//  2. The munition is within the platform's defensive engagement range.
//
// Returns a map of team → deduplicated slice of detected munition IDs.
func DetectMunitions(units []*enginev1.Unit, defs map[string]DefStats, munitions []*InFlightMunition) map[string][]string {
	if len(munitions) == 0 {
		return nil
	}

	type sideSet = map[string]bool
	bySide := make(map[string]sideSet)

	for _, detector := range units {
		if !unitCanOperate(detector) {
			continue
		}
		def := defs[detector.DefinitionId]
		if def.DetectionRangeM <= 0 {
			continue
		}
		teamID := unitTeamID(detector)
		if bySide[teamID] == nil {
			bySide[teamID] = make(sideSet)
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
				bySide[teamID][m.ID] = true
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

func InterceptMunitionsTick(units []*enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats, munitions []*InFlightMunition, detections map[string][]string, rng Rng) (remaining []*InFlightMunition, shots []InterceptShot) {
	if len(munitions) == 0 {
		return munitions, nil
	}
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, u := range units {
		unitByID[u.GetId()] = u
	}
	detectedBySide := make(map[string]map[string]bool, len(detections))
	for side, ids := range detections {
		detectedBySide[side] = make(map[string]bool, len(ids))
		for _, id := range ids {
			detectedBySide[side][id] = true
		}
	}

	for _, m := range munitions {
		intercepted := false
		target := unitByID[m.TargetID]
		for _, defender := range units {
			if !unitCanOperate(defender) {
				continue
			}
			side := unitTeamID(defender)
			if side == "" {
				continue
			}
			if !canPerformSovereignAirDefense(defender, defs[defender.DefinitionId]) {
				continue
			}
			if side == unitTeamID(unitByID[m.ShooterID]) {
				continue
			}
			if !detectedBySide[side][m.ID] {
				continue
			}
			if !munitionThreatensSide(side, m, target) {
				continue
			}
			weaponID, weapon, ok := selectBestWeapon(defender, enginev1.UnitDomain_DOMAIN_AIR, weapons)
			if !ok {
				continue
			}
			dist := haversineM(defender.GetPosition().GetLat(), defender.GetPosition().GetLon(), m.CurLat, m.CurLon)
			if dist > weapon.RangeM {
				continue
			}
			salvo := capAtAmmo(defender, weaponID, 1)
			if salvo <= 0 {
				continue
			}
			decrementAmmo(defender, weaponID, salvo)
			shots = append(shots, InterceptShot{
				Defender: defender,
				Munition: m,
				WeaponID: weaponID,
			})
			prob := rangeDegradedPoh(weapon.ProbabilityOfHit, dist, weapon.RangeM)
			if rng.Float64() < prob {
				intercepted = true
			}
			break
		}
		if !intercepted {
			remaining = append(remaining, m)
		}
	}
	return remaining, shots
}

func munitionThreatensSide(side string, m *InFlightMunition, target *enginev1.Unit) bool {
	if side == "" || m == nil {
		return false
	}
	if target != nil && unitTeamID(target) == side {
		return true
	}
	ctx := geo.LookupPoint(geo.Point{Lat: m.CurLat, Lon: m.CurLon, AltMsl: m.CurAltMsl})
	return geo.CountryCode(ctx.AirspaceOwner) == side
}
