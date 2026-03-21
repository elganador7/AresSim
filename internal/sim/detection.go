package sim

import (
	"math"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

type detectionIndex map[string]map[string]bool

type trackPicture struct {
	BySide       DetectionSet
	ByGroup      detectionIndex
	ByDetector   detectionIndex
	GroupForUnit map[string]string
}

type SensorCapability struct {
	SensorType   enginev1.SensorType
	MaxRangeM    float64
	TargetStates []enginev1.SensorTargetState
	FireControl  bool
}

// DetectionSet maps each detecting team to the full set of enemy unit IDs
// currently visible to that team.
type DetectionSet map[string][]string

type fixedRng float64

func (f fixedRng) Float64() float64 { return float64(f) }

// SensorTick scans all operational units and builds the current detection picture.
// This wrapper stays deterministic for tests and non-stateful callers.
func SensorTick(units []*enginev1.Unit, defs map[string]DefStats, rules RelationshipRules) DetectionSet {
	return buildTrackPicture(units, defs, rules, nil, fixedRng(0)).BySide
}

func buildTrackPicture(units []*enginev1.Unit, defs map[string]DefStats, rules RelationshipRules, previous detectionIndex, rng Rng) trackPicture {
	groupForUnit := resolveTrackGroupIDs(units)
	bySide := make(map[string]map[string]bool)
	byGroup := make(detectionIndex)
	byDetector := make(detectionIndex)

	for _, u := range units {
		if !unitCanOperate(u) {
			continue
		}
		teamID := unitTeamID(u)
		if bySide[teamID] == nil {
			bySide[teamID] = make(map[string]bool)
		}
		groupID := groupForUnit[u.Id]
		if groupID != "" && byGroup[groupID] == nil {
			byGroup[groupID] = make(map[string]bool)
		}
		byDetector[u.Id] = make(map[string]bool)
	}

	for _, detector := range units {
		if !unitCanOperate(detector) {
			continue
		}
		detectorDef := defs[detector.DefinitionId]
		if detectorDef.DetectionRangeM <= 0 {
			continue
		}
		groupID := groupForUnit[detector.Id]
		for _, target := range units {
			if !unitIsAlive(target) {
				continue
			}
			if !unitsAreHostile(detector, target) && !isUnauthorizedOverflight(detector, target, defs, rules) {
				continue
			}
			targetDef := defs[target.DefinitionId]
			effectiveRange, canObserve := effectiveSensorRange(detectorDef, detector, targetDef, target)
			if !canObserve {
				continue
			}
			dist := haversineM(
				detector.GetPosition().GetLat(), detector.GetPosition().GetLon(),
				target.GetPosition().GetLat(), target.GetPosition().GetLon(),
			)
			if dist > effectiveRange {
				continue
			}
			retained := previous != nil && previous[detector.Id][target.Id]
			if rng.Float64() > detectionProbability(dist, effectiveRange, retained) {
				continue
			}
			byDetector[detector.Id][target.Id] = true
			bySide[unitTeamID(detector)][target.Id] = true
			if groupID != "" {
				byGroup[groupID][target.Id] = true
			}
		}
	}

	return trackPicture{
		BySide:       boolSetsToDetectionSet(bySide),
		ByGroup:      byGroup,
		ByDetector:   byDetector,
		GroupForUnit: groupForUnit,
	}
}

func detectionProbability(distanceM, effectiveRangeM float64, retained bool) float64 {
	if effectiveRangeM <= 0 || distanceM > effectiveRangeM {
		return 0
	}
	normalized := distanceM / effectiveRangeM
	pDetect := 0.95 - (0.75 * normalized)
	if pDetect < 0.20 {
		pDetect = 0.20
	}
	if retained {
		return math.Min(0.99, pDetect+0.25)
	}
	return pDetect
}

func detectorCanObserveTarget(detectorDef DefStats, detector *enginev1.Unit, targetDef DefStats, target *enginev1.Unit) bool {
	_, ok := effectiveSensorRange(detectorDef, detector, targetDef, target)
	return ok
}

func effectiveSensorRange(detectorDef DefStats, detector *enginev1.Unit, targetDef DefStats, target *enginev1.Unit) (float64, bool) {
	if len(detectorDef.SensorSuite) > 0 {
		targetState := inferredTargetState(targetDef, target)
		if targetState == "" {
			return effectiveDetectionRangeM(detectorDef, targetDef), true
		}
		bestRange := 0.0
		for _, sensor := range detectorDef.SensorSuite {
			if sensor.MaxRangeM <= 0 {
				continue
			}
			for _, state := range sensor.TargetStates {
				if state == sensorTargetStateEnum(targetState) {
					sensorRange := effectiveDetectionRangeM(DefStats{DetectionRangeM: sensor.MaxRangeM}, targetDef)
					if sensorRange > bestRange {
						bestRange = sensorRange
					}
				}
			}
		}
		return bestRange, bestRange > 0
	}
	allowedStates := inferredSensorTargetStates(detectorDef)
	if len(allowedStates) == 0 {
		return effectiveDetectionRangeM(detectorDef, targetDef), true
	}
	targetState := inferredTargetState(targetDef, target)
	if targetState == "" {
		return effectiveDetectionRangeM(detectorDef, targetDef), true
	}
	if !allowedStates[targetState] {
		return 0, false
	}
	return effectiveDetectionRangeM(detectorDef, targetDef), true
}

func sensorTargetStateEnum(state string) enginev1.SensorTargetState {
	switch state {
	case "airborne":
		return enginev1.SensorTargetState_SENSOR_TARGET_STATE_AIRBORNE
	case "land":
		return enginev1.SensorTargetState_SENSOR_TARGET_STATE_LAND
	case "surface":
		return enginev1.SensorTargetState_SENSOR_TARGET_STATE_SURFACE
	case "submerged":
		return enginev1.SensorTargetState_SENSOR_TARGET_STATE_SUBMERGED
	default:
		return enginev1.SensorTargetState_SENSOR_TARGET_STATE_UNSPECIFIED
	}
}

func inferredSensorTargetStates(detectorDef DefStats) map[string]bool {
	switch detectorDef.Domain {
	case enginev1.UnitDomain_DOMAIN_LAND:
		return inferredLandSensorStates(detectorDef)
	case enginev1.UnitDomain_DOMAIN_AIR:
		return inferredAirSensorStates(detectorDef)
	case enginev1.UnitDomain_DOMAIN_SEA:
		return inferredSeaSensorStates(detectorDef)
	case enginev1.UnitDomain_DOMAIN_SUBSURFACE:
		return map[string]bool{"surface": true, "submerged": true}
	default:
		return nil
	}
}

func inferredLandSensorStates(detectorDef DefStats) map[string]bool {
	switch enginev1.UnitGeneralType(detectorDef.GeneralType) {
	case enginev1.UnitGeneralType_GENERAL_TYPE_UNSPECIFIED:
		return map[string]bool{"land": true, "airborne": true, "surface": true}
	case enginev1.UnitGeneralType_GENERAL_TYPE_AIR_DEFENSE,
		enginev1.UnitGeneralType_GENERAL_TYPE_RADAR_EARLY_WARNING:
		return map[string]bool{"airborne": true}
	case enginev1.UnitGeneralType_GENERAL_TYPE_COASTAL_DEFENSE_MISSILE:
		return map[string]bool{"surface": true}
	default:
		if detectorDef.AssetClass == "radar_site" {
			return map[string]bool{"airborne": true}
		}
		return map[string]bool{"land": true}
	}
}

func inferredAirSensorStates(detectorDef DefStats) map[string]bool {
	switch enginev1.UnitGeneralType(detectorDef.GeneralType) {
	case enginev1.UnitGeneralType_GENERAL_TYPE_UNSPECIFIED:
		return map[string]bool{"airborne": true, "land": true, "surface": true}
	case enginev1.UnitGeneralType_GENERAL_TYPE_MARITIME_PATROL,
		enginev1.UnitGeneralType_GENERAL_TYPE_NAVAL_HELICOPTER,
		enginev1.UnitGeneralType_GENERAL_TYPE_ASW_HELICOPTER:
		return map[string]bool{"airborne": true, "surface": true, "submerged": true}
	case enginev1.UnitGeneralType_GENERAL_TYPE_ISR_FIXED_WING,
		enginev1.UnitGeneralType_GENERAL_TYPE_ISR_UAV,
		enginev1.UnitGeneralType_GENERAL_TYPE_STRIKE_UAV,
		enginev1.UnitGeneralType_GENERAL_TYPE_UCAV,
		enginev1.UnitGeneralType_GENERAL_TYPE_ATTACK_AIRCRAFT,
		enginev1.UnitGeneralType_GENERAL_TYPE_BOMBER:
		return map[string]bool{"airborne": true, "land": true, "surface": true}
	default:
		return map[string]bool{"airborne": true}
	}
}

func inferredSeaSensorStates(detectorDef DefStats) map[string]bool {
	states := map[string]bool{"airborne": true, "surface": true}
	switch enginev1.UnitGeneralType(detectorDef.GeneralType) {
	case enginev1.UnitGeneralType_GENERAL_TYPE_CRUISER,
		enginev1.UnitGeneralType_GENERAL_TYPE_DESTROYER,
		enginev1.UnitGeneralType_GENERAL_TYPE_FRIGATE,
		enginev1.UnitGeneralType_GENERAL_TYPE_CORVETTE,
		enginev1.UnitGeneralType_GENERAL_TYPE_AIRCRAFT_CARRIER:
		states["submerged"] = true
	}
	return states
}

func inferredTargetState(targetDef DefStats, target *enginev1.Unit) string {
	switch targetDef.Domain {
	case enginev1.UnitDomain_DOMAIN_AIR:
		return "airborne"
	case enginev1.UnitDomain_DOMAIN_LAND:
		return "land"
	case enginev1.UnitDomain_DOMAIN_SEA:
		return "surface"
	case enginev1.UnitDomain_DOMAIN_SUBSURFACE:
		return "submerged"
	default:
		return ""
	}
}

func (tp trackPicture) unitHasTrack(unitID, targetID string) bool {
	groupID := tp.GroupForUnit[unitID]
	if groupID == "" {
		return false
	}
	return tp.ByGroup[groupID][targetID]
}

func boolSetsToDetectionSet(bySide map[string]map[string]bool) DetectionSet {
	result := make(DetectionSet, len(bySide))
	for side, ids := range bySide {
		list := make([]string, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		result[side] = list
	}
	return result
}
