package sim

import (
	"math"
	"strings"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/geo"
)

// DefStats holds the per-definition values the sim loop needs each tick.
type DefStats struct {
	CruiseSpeedMps              float64
	BaseStrength                float64
	DetectionRangeM             float64
	RadarCrossSectionM2         float64
	AuthorizedPersonnel         int
	ReplacementCostUSD          float64
	StrategicValueUSD           float64
	EconomicValueUSD            float64
	Domain                      enginev1.UnitDomain // physical domain of this platform
	TargetClass                 string
	AssetClass                  string
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

type detectionIndex map[string]map[string]bool

type trackPicture struct {
	BySide       DetectionSet
	ByGroup      detectionIndex
	GroupForUnit map[string]string
}

func unitTeamID(u *enginev1.Unit) string {
	if u == nil {
		return ""
	}
	if team := strings.TrimSpace(u.GetTeamId()); team != "" {
		return team
	}
	return strings.TrimSpace(u.GetSide())
}

func unitCoalitionID(u *enginev1.Unit) string {
	if u == nil {
		return ""
	}
	if coalition := strings.TrimSpace(u.GetCoalitionId()); coalition != "" {
		return coalition
	}
	return strings.TrimSpace(u.GetSide())
}

func unitsAreHostile(a, b *enginev1.Unit) bool {
	if a == nil || b == nil {
		return false
	}
	aCoalition := unitCoalitionID(a)
	bCoalition := unitCoalitionID(b)
	if aCoalition == "" || bCoalition == "" {
		return a.GetSide() != b.GetSide()
	}
	return aCoalition != bCoalition
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

// AdjudicateTick checks all pairs of enemy active units and fires a salvo for
// each unit that meets the engagement criteria. Each unit fires at most one
// salvo per tick (at its highest-priority target).
//
// Fire conditions — a unit fires only when EITHER:
//   - The range-degraded probability of hit exceeds 50 %, OR
//   - The shooter is already within the target's detection range (stealth is
//     already compromised, so there is nothing to gain by holding fire).
//
// Salvo sizing — the minimum number of rounds N such that the cumulative
// kill probability of all munitions (in-flight + new salvo) exceeds 70 %.
// Already in-flight munitions targeting the same unit are counted, so platforms
// do not keep firing after enough rounds are already on the way.
func AdjudicateTick(units []*enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats, inFlight []*InFlightMunition, rules RelationshipRules, simSeconds float64) AdjudicateResult {
	firedThisTick := make(map[string]bool)
	orderedUnits := make(map[string]bool)
	var result AdjudicateResult
	tracks := buildTrackPicture(units, defs, rules)
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
			tracks,
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
			dist := haversineM(
				a.GetPosition().GetLat(), a.GetPosition().GetLon(),
				b.GetPosition().GetLat(), b.GetPosition().GetLon(),
			)

			defA := defs[a.DefinitionId]
			defB := defs[b.DefinitionId]

			wIDA, wA, hasWeapA := selectBestWeapon(a, defB.Domain, weapons)
			wIDB, wB, hasWeapB := selectBestWeapon(b, defA.Domain, weapons)

			aHasTrack := tracks.unitHasTrack(a.Id, b.Id)
			bHasTrack := tracks.unitHasTrack(b.Id, a.Id)
			aCanEngageB := unitsAreHostile(a, b) || (isUnauthorizedOverflight(a, b, defs, rules) && canPerformSovereignAirDefense(a, defA))
			bCanEngageA := unitsAreHostile(a, b) || (isUnauthorizedOverflight(b, a, defs, rules) && canPerformSovereignAirDefense(b, defB))

			aInRange := aCanEngageB && hasWeapA && aHasTrack && dist <= wA.RangeM && !firedThisTick[a.Id] && !orderedUnits[a.Id] && unitReadyToStrike(a, wA, simSeconds)
			bInRange := bCanEngageA && hasWeapB && bHasTrack && dist <= wB.RangeM && !firedThisTick[b.Id] && !orderedUnits[b.Id] && unitReadyToStrike(b, wB, simSeconds)

			if !aInRange && !bInRange {
				continue
			}

			if aInRange {
				prob := rangeDegradedPoh(wA.ProbabilityOfHit, dist, wA.RangeM)
				aDetectedByB := dist <= effectiveDetectionRangeM(defB, defA)
				if shouldAutonomouslyEngage(a, prob, aDetectedByB) {
					miss := inFlightMissProb(inFlight, b.Id)
					salvo := salvoToAchieveKillProb(miss, prob, 0.30)
					salvo = capAtAmmo(a, wIDA, salvo)
					if salvo > 0 {
						decrementAmmo(a, wIDA, salvo)
						applyStrikeCooldown(a, wA, simSeconds)
						result.Shots = append(result.Shots, FiredShot{
							Shooter:        a,
							Target:         b,
							WeaponID:       wIDA,
							HitProbability: prob,
							SalvoSize:      salvo,
						})
						firedThisTick[a.Id] = true
					}
				}
			}

			if bInRange {
				prob := rangeDegradedPoh(wB.ProbabilityOfHit, dist, wB.RangeM)
				bDetectedByA := dist <= effectiveDetectionRangeM(defA, defB)
				if shouldAutonomouslyEngage(b, prob, bDetectedByA) {
					miss := inFlightMissProb(inFlight, a.Id)
					salvo := salvoToAchieveKillProb(miss, prob, 0.30)
					salvo = capAtAmmo(b, wIDB, salvo)
					if salvo > 0 {
						decrementAmmo(b, wIDB, salvo)
						applyStrikeCooldown(b, wB, simSeconds)
						result.Shots = append(result.Shots, FiredShot{
							Shooter:        b,
							Target:         a,
							WeaponID:       wIDB,
							HitProbability: prob,
							SalvoSize:      salvo,
						})
						firedThisTick[b.Id] = true
					}
				}
			}

			if firedThisTick[a.Id] {
				break // A has fired; advance to the next outer unit
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
	tracks trackPicture,
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
	targetDef := defs[target.DefinitionId]
	shooterDef := defs[shooter.DefinitionId]
	weaponID, weapon, hasWeapon := selectBestWeapon(shooter, targetDef.Domain, weapons)
	if !hasWeapon {
		return false
	}
	if !tracks.unitHasTrack(shooter.Id, target.Id) && !canExecutePreplannedStrategicStrike(shooter, target, targetDef, weapon) {
		return false
	}
	if !unitReadyToStrike(shooter, weapon, simSeconds) {
		return false
	}
	dist := haversineM(
		shooter.GetPosition().GetLat(), shooter.GetPosition().GetLon(),
		target.GetPosition().GetLat(), target.GetPosition().GetLon(),
	)
	if dist > weapon.RangeM {
		return false
	}
	prob := rangeDegradedPoh(weapon.ProbabilityOfHit, dist, weapon.RangeM)
	detectedByTarget := dist <= effectiveDetectionRangeM(targetDef, shooterDef)
	if !shouldExecuteManualAttack(shooter, prob, detectedByTarget) {
		return false
	}
	outcome := resolveImpactOutcome(weapon.EffectType, targetDef.TargetClass)
	if outcome == outcomeNoEffect {
		return false
	}
	if requireDesiredEffect && !impactOutcomeSupportsDesiredEffect(outcome, desiredEffect) {
		return false
	}
	targetMissProb := 0.30
	if pkillThreshold > 0 && pkillThreshold < 1 {
		targetMissProb = 1.0 - float64(pkillThreshold)
	}
	miss := inFlightMissProb(inFlight, target.Id)
	salvo := salvoToAchieveKillProb(miss, prob, targetMissProb)
	salvo = capAtAmmo(shooter, weaponID, salvo)
	if salvo <= 0 {
		return false
	}
	decrementAmmo(shooter, weaponID, salvo)
	applyStrikeCooldown(shooter, weapon, simSeconds)
	result.Shots = append(result.Shots, FiredShot{
		Shooter:        shooter,
		Target:         target,
		WeaponID:       weaponID,
		HitProbability: prob,
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

// inFlightMissProb returns the combined probability that ALL currently
// in-flight munitions targeting targetID miss. The complement is the
// cumulative kill probability of the existing salvo:
//
//	cumKillProb = 1 − inFlightMissProb(...)
func inFlightMissProb(inFlight []*InFlightMunition, targetID string) float64 {
	p := 1.0
	for _, m := range inFlight {
		if m.TargetID == targetID {
			p *= 1.0 - m.HitProbability
		}
	}
	return p
}

// salvoToAchieveKillProb returns the minimum number of additional rounds at
// singleShotPoh needed so that the combined miss probability drops to or below
// targetMissProb (default 0.30 → cumulative kill ≥ 70 %).
//
//	existingMissProb × (1−p)^N ≤ targetMissProb
//	N ≥ log(targetMissProb / existingMissProb) / log(1−p)
//
// Returns 0 if the threshold is already met by existing in-flight munitions.
func salvoToAchieveKillProb(existingMissProb, singleShotPoh, targetMissProb float64) int32 {
	if existingMissProb <= targetMissProb {
		return 0 // enough in flight already
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

// capAtAmmo returns the minimum of requested and available rounds for weaponID
// on the unit. Returns 0 if the weapon is not found or has no ammo.
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

// ResolveArrivals resolves kill outcomes for munitions that have reached their
// destinations. For each arrived munition, rng is rolled against its
// pre-computed HitProbability. Already-destroyed targets are safely skipped.
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
			continue // already destroyed before munition arrived
		}
		if rng.Float64() < m.HitProbability {
			outcome := resolveImpactOutcome(weapons[m.WeaponID].EffectType, defs[target.DefinitionId].TargetClass)
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

// effectiveDetectionRangeM adjusts the detector's nominal range based on the
// target's radar cross section. A 1 m^2 target uses the nominal range.
// The fourth-root scaling is a simple approximation of the radar equation:
//
//	range ∝ sigma^(1/4)
//
// Very small RCS values are clamped so stealth remains meaningful without
// making targets practically invisible; very large signatures are also capped
// to avoid runaway detection bonuses.
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

// decrementAmmo reduces the current quantity of weaponID on shooter by amount.
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

// ─── SENSOR DETECTION ─────────────────────────────────────────────────────────

// DetectionSet maps each detecting side to the full set of enemy unit IDs
// currently within sensor range of at least one unit on that side.
type DetectionSet map[string][]string

// SensorTick scans all operational units and builds the current detection picture.
func SensorTick(units []*enginev1.Unit, defs map[string]DefStats, rules RelationshipRules) DetectionSet {
	return buildTrackPicture(units, defs, rules).BySide
}

func buildTrackPicture(units []*enginev1.Unit, defs map[string]DefStats, rules RelationshipRules) trackPicture {
	groupForUnit := resolveTrackGroupIDs(units)
	bySide := make(map[string]map[string]bool)
	byGroup := make(detectionIndex)

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
			dist := haversineM(
				detector.GetPosition().GetLat(), detector.GetPosition().GetLon(),
				target.GetPosition().GetLat(), target.GetPosition().GetLon(),
			)
			if dist > effectiveDetectionRangeM(detectorDef, defs[target.DefinitionId]) {
				continue
			}
			bySide[unitTeamID(detector)][target.Id] = true
			if groupID != "" {
				byGroup[groupID][target.Id] = true
			}
		}
	}

	return trackPicture{
		BySide:       boolSetsToDetectionSet(bySide),
		ByGroup:      byGroup,
		GroupForUnit: groupForUnit,
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
