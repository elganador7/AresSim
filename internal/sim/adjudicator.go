package sim

import (
	"strings"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/geo"
)

// DefStats holds the per-definition values the sim loop needs each tick.
type DefStats struct {
	CruiseSpeedMps              float64
	BaseStrength                float64
	Accuracy                    float64
	DetectionRangeM             float64
	RadarCrossSectionM2         float64
	FuelCapacityLiters          float64
	FuelBurnRateLph             float64
	SensorSuite                 []SensorCapability
	GeneralType                 int32
	EmploymentRole              string
	AuthorizedPersonnel         int
	ReplacementCostUSD          float64
	StrategicValueUSD           float64
	EconomicValueUSD            float64
	Domain                      enginev1.UnitDomain // physical domain of this platform
	TargetClass                 string
	AssetClass                  string
	EmbarkedFixedWingCapacity   int
	EmbarkedRotaryWingCapacity  int
	EmbarkedUAVCapacity         int
	LaunchCapacityPerInterval   int
	RecoveryCapacityPerInterval int
	SortieIntervalMinutes       int
}

// WeaponStats holds the per-weapon catalog data needed for engagement resolution
// and in-flight munition tracking.
type WeaponStats struct {
	RangeM           float64
	SpeedMps         float64 // projectile/missile speed; used for munition travel time
	ProbabilityOfHit float64
	DomainTargets    []enginev1.UnitDomain
	Guidance         enginev1.GuidanceType // homing behaviour for in-flight munitions
	EffectType       enginev1.WeaponEffectType
}

// Rng is the minimal interface for probability rolls.
// *rand.Rand satisfies this interface; a deterministic stub is used in tests.
type Rng interface {
	Float64() float64
}

// FiredShot records a salvo discharge during adjudication.
// SalvoSize is the number of rounds fired in this salvo; mock.go uses it to
// create that many in-flight munitions, each with the same HitProbability.
type FiredShot struct {
	Shooter        *enginev1.Unit
	Target         *enginev1.Unit
	WeaponID       string
	HitProbability float64 // range-degraded probability per round at fire time
	SalvoSize      int32   // rounds fired in this salvo (≥1)
}

// AdjudicateResult holds all shots fired in one tick of adjudication.
// Kills are NOT resolved here — they are deferred to when the in-flight
// munition arrives at its destination (see ResolveArrivals).
type AdjudicateResult struct {
	Shots []FiredShot
}

func unitTeamID(u *enginev1.Unit) string {
	if u == nil {
		return ""
	}
	return CountryDisplayCode(u.GetTeamId())
}

func unitCoalitionID(u *enginev1.Unit) string {
	if u == nil {
		return ""
	}
	if coalition := strings.TrimSpace(u.GetCoalitionId()); coalition != "" {
		return coalition
	}
	return unitTeamID(u)
}

func unitsAreHostile(a, b *enginev1.Unit) bool {
	if a == nil || b == nil {
		return false
	}
	aTeam := unitTeamID(a)
	bTeam := unitTeamID(b)
	if aTeam == "" || bTeam == "" {
		return false
	}
	if aTeam == bTeam {
		return false
	}
	aCoalition := unitCoalitionID(a)
	bCoalition := unitCoalitionID(b)
	if aCoalition != "" && bCoalition != "" {
		return aCoalition != bCoalition
	}
	return true
}

func UnitsAreHostileForUI(a, b *enginev1.Unit) bool {
	return unitsAreHostile(a, b)
}

func isUnauthorizedOverflight(defender, intruder *enginev1.Unit, defs map[string]DefStats, rules RelationshipRules) bool {
	if defender == nil || intruder == nil {
		return false
	}
	if unitTeamID(defender) == "" || unitTeamID(defender) == unitTeamID(intruder) {
		return false
	}
	intruderDef := defs[intruder.DefinitionId]
	if intruderDef.Domain != enginev1.UnitDomain_DOMAIN_AIR {
		return false
	}
	if intruder.GetPosition() == nil || intruder.GetPosition().GetAltMsl() <= 100 {
		return false
	}
	ctx := geo.LookupPoint(geo.Point{
		Lat: intruder.GetPosition().GetLat(),
		Lon: intruder.GetPosition().GetLon(),
	})
	defenderCountry := unitTeamID(defender)
	if geo.CountryCode(ctx.AirspaceOwner) != defenderCountry {
		return false
	}
	rule := GetRelationshipRule(rules, unitTeamID(intruder), defenderCountry)
	return !rule.AirspaceTransitAllowed
}

func canPerformSovereignAirDefense(unit *enginev1.Unit, def DefStats) bool {
	if unit == nil {
		return false
	}
	if def.Domain != enginev1.UnitDomain_DOMAIN_AIR {
		return true
	}
	return unit.GetPosition() != nil && unit.GetPosition().GetAltMsl() > 100
}

// ─── SENSOR DETECTION ─────────────────────────────────────────────────────────

func resolveTrackGroupIDs(units []*enginev1.Unit) map[string]string {
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, u := range units {
		if unitCanOperate(u) {
			unitByID[u.Id] = u
		}
	}

	resolved := make(map[string]string, len(unitByID))
	for _, u := range units {
		if !unitCanOperate(u) {
			continue
		}
		root := resolveTrackRoot(u, unitByID, resolved, map[string]bool{})
		resolved[u.Id] = unitTeamID(u) + "|" + root
	}
	return resolved
}

func resolveTrackRoot(unit *enginev1.Unit, unitByID map[string]*enginev1.Unit, resolved map[string]string, visiting map[string]bool) string {
	if groupID := resolved[unit.Id]; groupID != "" {
		if idx := len(unitTeamID(unit)) + 1; len(groupID) > idx {
			return groupID[idx:]
		}
		return unit.Id
	}
	if visiting[unit.Id] {
		return unit.Id
	}
	visiting[unit.Id] = true

	parentID := unit.GetParentUnitId()
	if parentID == "" {
		return unit.Id
	}
	parent, ok := unitByID[parentID]
	if !ok || unitTeamID(parent) != unitTeamID(unit) {
		return parentID
	}
	return resolveTrackRoot(parent, unitByID, resolved, visiting)
}

// unitIsActive returns true if u has not been destroyed in status terms.
func unitIsActive(u *enginev1.Unit) bool {
	if u.Status == nil {
		return true
	}
	return u.Status.IsActive
}

func unitIsAlive(u *enginev1.Unit) bool {
	return unitIsActive(u) && currentDamageState(u) != enginev1.DamageState_DAMAGE_STATE_DESTROYED
}

func unitCanOperate(u *enginev1.Unit) bool {
	return unitIsAlive(u) && currentDamageState(u) != enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED
}

func UnitCanOperateForUI(u *enginev1.Unit) bool {
	return unitCanOperate(u)
}

// killUnit marks u as destroyed and clears its move order in-place.
func killUnit(u *enginev1.Unit) {
	if u.Status == nil {
		u.Status = &enginev1.OperationalStatus{}
	}
	u.Status.IsActive = false
	u.Status.PersonnelStrength = 0
	u.Status.EquipmentStrength = 0
	u.Status.CombatEffectiveness = 0
	u.DamageState = enginev1.DamageState_DAMAGE_STATE_DESTROYED
	u.MoveOrder = nil
}
