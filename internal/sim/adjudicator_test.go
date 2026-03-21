package sim

import (
	"math"
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func approxEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// ─── helpers ──────────────────────────────────────────────────────────────────

func makeUnit(id, teamID, defID string, lat, lon float64) *enginev1.Unit {
	return &enginev1.Unit{
		Id:           id,
		TeamId:       teamID,
		CoalitionId:  teamID,
		DefinitionId: defID,
		Position:     &enginev1.Position{Lat: lat, Lon: lon},
		Status:       &enginev1.OperationalStatus{IsActive: true},
	}
}

func makeChildUnit(id, teamID, defID, parentID string, lat, lon float64) *enginev1.Unit {
	u := makeUnit(id, teamID, defID, lat, lon)
	u.ParentUnitId = parentID
	return u
}

func makeDef(detectionRangeM, baseStrength float64) DefStats {
	return DefStats{
		CruiseSpeedMps:  10,
		BaseStrength:    baseStrength,
		DetectionRangeM: detectionRangeM,
	}
}

// constRng is a deterministic Rng that always returns a fixed value.
// Use constRng(0) for "always hit" and constRng(1) for "always miss".
type constRng float64

func (c constRng) Float64() float64 { return float64(c) }

type sequenceRng struct {
	values []float64
	index  int
}

func (s *sequenceRng) Float64() float64 {
	if len(s.values) == 0 {
		return 0
	}
	if s.index >= len(s.values) {
		return s.values[len(s.values)-1]
	}
	v := s.values[s.index]
	s.index++
	return v
}

// alwaysHit and alwaysMiss are convenience instances.
const alwaysHit = constRng(0.0)
const alwaysMiss = constRng(1.0)

func makeWeaponCatalog(id string, rangeM, prob float64, domains ...enginev1.UnitDomain) map[string]WeaponStats {
	return map[string]WeaponStats{
		id: {RangeM: rangeM, ProbabilityOfHit: prob, DomainTargets: domains},
	}
}

func makeWeaponCatalogWithEffect(id string, rangeM, prob float64, effectType enginev1.WeaponEffectType, domains ...enginev1.UnitDomain) map[string]WeaponStats {
	return map[string]WeaponStats{
		id: {RangeM: rangeM, ProbabilityOfHit: prob, DomainTargets: domains, EffectType: effectType},
	}
}

func addWeapons(u *enginev1.Unit, weaponID string, qty int32) {
	u.Weapons = append(u.Weapons, &enginev1.WeaponState{
		WeaponId: weaponID, CurrentQty: qty, MaxQty: qty,
	})
}

func repeatInFlight(targetID string, hitProbability float64, count int) []*InFlightMunition {
	inFlight := make([]*InFlightMunition, 0, count)
	for range count {
		inFlight = append(inFlight, &InFlightMunition{
			ID:             NextMunitionID(),
			TargetID:       targetID,
			HitProbability: hitProbability,
		})
	}
	return inFlight
}

// munitionFromShot builds an InFlightMunition from a FiredShot for use in
// ResolveArrivals tests. SpeedMps=0 means the munition has already arrived.
func munitionFromShot(shot FiredShot) *InFlightMunition {
	return &InFlightMunition{
		ID:             NextMunitionID(),
		WeaponID:       shot.WeaponID,
		ShooterID:      shot.Shooter.Id,
		TargetID:       shot.Target.Id,
		HitProbability: shot.HitProbability,
	}
}

// ─── unitIsActive ─────────────────────────────────────────────────────────────

func TestUnitIsActive_NilStatus(t *testing.T) {
	u := &enginev1.Unit{Id: "u1"}
	if !unitIsActive(u) {
		t.Error("unit with nil Status should be considered active")
	}
}

func TestUnitIsActive_ActiveStatus(t *testing.T) {
	u := &enginev1.Unit{Status: &enginev1.OperationalStatus{IsActive: true}}
	if !unitIsActive(u) {
		t.Error("unit with IsActive=true should be active")
	}
}

func TestUnitIsActive_InactiveStatus(t *testing.T) {
	u := &enginev1.Unit{Status: &enginev1.OperationalStatus{IsActive: false}}
	if unitIsActive(u) {
		t.Error("unit with IsActive=false should not be active")
	}
}

// ─── killUnit ─────────────────────────────────────────────────────────────────

func TestKillUnit_SetsInactive(t *testing.T) {
	u := makeUnit("u1", "Blue", "def", 0, 0)
	u.MoveOrder = &enginev1.MoveOrder{Waypoints: []*enginev1.Waypoint{{Lat: 1, Lon: 1}}}
	killUnit(u)

	if unitIsActive(u) {
		t.Error("killed unit should not be active")
	}
	if u.MoveOrder != nil {
		t.Error("killed unit should have MoveOrder cleared")
	}
}

func TestKillUnit_NilStatus_InitializesStatus(t *testing.T) {
	u := &enginev1.Unit{Id: "u1"}
	killUnit(u)
	if u.Status == nil {
		t.Fatal("Status should be initialized by killUnit")
	}
	if u.Status.IsActive {
		t.Error("IsActive should be false after killUnit")
	}
}

// ─── rangeDegradedPoh ─────────────────────────────────────────────────────────

func TestRangeDegradedPoh_AtZeroRange(t *testing.T) {
	// At distance 0, full basePoh applies (factor = 1.0).
	got := rangeDegradedPoh(0.8, 0, 100_000)
	if got != 0.8 {
		t.Errorf("expected 0.8 at dist=0, got %f", got)
	}
}

func TestRangeDegradedPoh_AtMaxRange(t *testing.T) {
	// At max range, factor = 1.0 - 0.7 = 0.3, so result = basePoh * 0.3.
	got := rangeDegradedPoh(1.0, 100_000, 100_000)
	if !approxEqual(got, 0.3) {
		t.Errorf("expected ~0.3 at max range, got %f", got)
	}
}

func TestRangeDegradedPoh_AtHalfRange(t *testing.T) {
	// At half range, factor = 1.0 - 0.35 = 0.65, so result = 0.8 * 0.65 = 0.52.
	got := rangeDegradedPoh(0.8, 50_000, 100_000)
	want := 0.8 * 0.65
	if got != want {
		t.Errorf("expected %f at half range, got %f", want, got)
	}
}

func TestRangeDegradedPoh_FloorAt30Pct(t *testing.T) {
	// Beyond max range the factor is clamped to 0.3 minimum.
	got := rangeDegradedPoh(1.0, 200_000, 100_000)
	if got != 0.3 {
		t.Errorf("expected 0.3 floor beyond max range, got %f", got)
	}
}

func TestRangeDegradedPoh_ZeroMaxRange(t *testing.T) {
	// Zero rangeM guard: returns basePoh unchanged.
	got := rangeDegradedPoh(0.75, 0, 0)
	if got != 0.75 {
		t.Errorf("expected 0.75 with zero rangeM, got %f", got)
	}
}

func TestEffectiveDetectionRangeM_StealthTargetShrinksRange(t *testing.T) {
	detector := DefStats{DetectionRangeM: 100_000}
	target := DefStats{RadarCrossSectionM2: 0.01}

	got := effectiveDetectionRangeM(detector, target)
	want := 100_000 * math.Pow(0.01, 0.25)
	if !approxEqual(got, want) {
		t.Fatalf("expected stealth-adjusted range %f, got %f", want, got)
	}
}

func TestEffectiveDetectionRangeM_LargeTargetExtendsRange(t *testing.T) {
	detector := DefStats{DetectionRangeM: 100_000}
	target := DefStats{RadarCrossSectionM2: 16}

	got := effectiveDetectionRangeM(detector, target)
	want := 200_000.0 // clamped at 2x
	if !approxEqual(got, want) {
		t.Fatalf("expected large-RCS range %f, got %f", want, got)
	}
}

// ─── AdjudicateTick ───────────────────────────────────────────────────────────

func TestAdjudicateTick_NoUnits_NoShots(t *testing.T) {
	adj := AdjudicateTick(nil, nil, nil, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Errorf("expected 0 shots with no units, got %d", len(adj.Shots))
	}
}

func TestAdjudicateTick_FriendlyFire_NoShots(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Blue", "def", 0, 0)
	addWeapons(a, "gun", 5)
	addWeapons(b, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Errorf("friendly fire: expected 0 shots, got %d", len(adj.Shots))
	}
}

func TestAdjudicateTick_OutOfRange_NoShots(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 1.8) // ~200 km at equator
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Errorf("out-of-range: expected 0 shots, got %d", len(adj.Shots))
	}
}

func TestAdjudicateTick_InRange_ShotFired(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01) // ~1.1 km
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected 1 shot, got %d", len(adj.Shots))
	}
	if adj.Shots[0].Shooter.Id != "a" || adj.Shots[0].Target.Id != "b" {
		t.Errorf("unexpected shot: %+v", adj.Shots[0])
	}
	if adj.Shots[0].SalvoSize != 1 {
		t.Errorf("expected single-round salvo, got %d", adj.Shots[0].SalvoSize)
	}
}

func TestAdjudicateTick_ShotHasHitProbability(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01)
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected 1 shot, got %d", len(adj.Shots))
	}
	if adj.Shots[0].HitProbability <= 0 || adj.Shots[0].HitProbability > 1.0 {
		t.Errorf("HitProbability should be in (0,1], got %f", adj.Shots[0].HitProbability)
	}
}

func TestAdjudicateTick_AmmoDecremented(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01)
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if a.Weapons[0].CurrentQty != 4 {
		t.Errorf("expected ammo to decrement to 4, got %d", a.Weapons[0].CurrentQty)
	}
}

func TestAdjudicateTick_OutOfAmmo_NoShots(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01)
	addWeapons(a, "gun", 0) // depleted
	addWeapons(b, "gun", 0) // depleted
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Errorf("depleted units should fire 0 shots, got %d", len(adj.Shots))
	}
}

func TestAdjudicateTick_WrongDomain_NoShots(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01)
	addWeapons(a, "land-gun", 10)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_AIR}, // air — land gun can't reach
	}
	catalog := makeWeaponCatalog("land-gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Errorf("wrong-domain weapon: expected 0 shots, got %d", len(adj.Shots))
	}
}

func TestAdjudicateTick_BothInRange_BothShoot(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 0.01)
	addWeapons(a, "gun", 5)
	addWeapons(b, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 2 {
		t.Errorf("expected 2 shots (both fire), got %d", len(adj.Shots))
	}
}

func TestAdjudicateTick_EachUnitFiresOnce(t *testing.T) {
	// Blue has massive range; three Red units all close. Blue should fire only once.
	blue := makeUnit("blue", "Blue", "long", 0, 0)
	r1 := makeUnit("r1", "Red", "short", 0, 0.01)
	r2 := makeUnit("r2", "Red", "short", 0, 0.02)
	r3 := makeUnit("r3", "Red", "short", 0, 0.03)
	addWeapons(blue, "missile", 10)
	defs := map[string]DefStats{
		"long":  {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 500_000},
		"short": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}
	catalog := makeWeaponCatalog("missile", 500_000, 0.9, enginev1.UnitDomain_DOMAIN_AIR)

	adj := AdjudicateTick([]*enginev1.Unit{blue, r1, r2, r3}, defs, catalog, nil, nil, 0)
	blueShots := 0
	for _, s := range adj.Shots {
		if s.Shooter.Id == "blue" {
			blueShots++
		}
	}
	if blueShots != 1 {
		t.Errorf("each unit should fire at most once per tick; blue fired %d times", blueShots)
	}
}

func TestAdjudicateTick_DestroyedUnit_NoShots(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	killUnit(a)
	b := makeUnit("b", "Red", "def", 0, 0)
	addWeapons(b, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	for _, s := range adj.Shots {
		if s.Shooter.Id == "a" {
			t.Error("destroyed unit should not fire")
		}
	}
}

func TestAdjudicateTick_LowPkillOutsideSensors_HoldsFire(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01)
	addWeapons(a, "missile", 5)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 0},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 0},
	}
	catalog := makeWeaponCatalog("missile", 50_000, 0.4, enginev1.UnitDomain_DOMAIN_AIR)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Fatalf("expected no shots when pkill <= 50%% outside sensors, got %d", len(adj.Shots))
	}
	if a.Weapons[0].CurrentQty != 5 {
		t.Errorf("ammo should be conserved, got %d remaining", a.Weapons[0].CurrentQty)
	}
}

func TestAdjudicateTick_LowPkillInsideEnemySensors_Fires(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01)
	addWeapons(a, "missile", 5)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 50_000},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 5_000},
	}
	catalog := makeWeaponCatalog("missile", 50_000, 0.4, enginev1.UnitDomain_DOMAIN_AIR)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected one firing decision when already detected, got %d", len(adj.Shots))
	}
	if adj.Shots[0].SalvoSize != 3 {
		t.Errorf("expected 3-round salvo to exceed 70%% cumulative pkill, got %d", adj.Shots[0].SalvoSize)
	}
	if a.Weapons[0].CurrentQty != 2 {
		t.Errorf("expected ammo to drop from 5 to 2, got %d", a.Weapons[0].CurrentQty)
	}
}

func TestAdjudicateTick_EnemySensorRangeUsesTargetRCS(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.036) // ~4 km
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 50_000, RadarCrossSectionM2: 0.01},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 10_000},
	}
	catalog := makeWeaponCatalog("gun", 10_000, 0.2, enginev1.UnitDomain_DOMAIN_AIR)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Fatalf("expected no shot when stealth target stays outside adjusted sensor range, got %d", len(adj.Shots))
	}

	defs["def-a"] = DefStats{Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 50_000, RadarCrossSectionM2: 16}
	adj = AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected one shot when large-RCS target is inside adjusted sensor range, got %d", len(adj.Shots))
	}
}

func TestAdjudicateTick_SalvoSizedToThreshold(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01)
	addWeapons(a, "missile", 10)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 50_000},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}
	catalog := makeWeaponCatalog("missile", 50_000, 0.6, enginev1.UnitDomain_DOMAIN_AIR)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected one shot record, got %d", len(adj.Shots))
	}
	if adj.Shots[0].SalvoSize != 2 {
		t.Errorf("expected 2-round salvo for 60%% single-shot pkill, got %d", adj.Shots[0].SalvoSize)
	}
	if a.Weapons[0].CurrentQty != 8 {
		t.Errorf("expected ammo to drop from 10 to 8, got %d", a.Weapons[0].CurrentQty)
	}
}

func TestAdjudicateTick_InFlightRoundsReduceNewSalvo(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01)
	addWeapons(a, "missile", 10)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 50_000},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 5_000},
	}
	catalog := makeWeaponCatalog("missile", 50_000, 0.4, enginev1.UnitDomain_DOMAIN_AIR)
	inFlight := repeatInFlight("b", 0.4, 2)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, inFlight, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected one shot record, got %d", len(adj.Shots))
	}
	if adj.Shots[0].SalvoSize != 1 {
		t.Errorf("expected one additional round after two are already inbound, got %d", adj.Shots[0].SalvoSize)
	}
}

func TestAdjudicateTick_EnoughInFlight_NoAdditionalShot(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01)
	addWeapons(a, "missile", 10)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 50_000},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 5_000},
	}
	catalog := makeWeaponCatalog("missile", 50_000, 0.4, enginev1.UnitDomain_DOMAIN_AIR)
	inFlight := repeatInFlight("b", 0.4, 3)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, inFlight, nil, 0)
	if len(adj.Shots) != 0 {
		t.Fatalf("expected no new shot once in-flight salvo already exceeds 70%% pkill, got %d", len(adj.Shots))
	}
	if a.Weapons[0].CurrentQty != 10 {
		t.Errorf("ammo should be unchanged when existing salvo is sufficient, got %d", a.Weapons[0].CurrentQty)
	}
}

func TestAdjudicateTick_ConnectedSensorAllowsLauncherToFire(t *testing.T) {
	parent := makeUnit("battery", "Blue", "command", 0, 0)
	radar := makeChildUnit("radar", "Blue", "sensor", "battery", 0, 0)
	launcher := makeChildUnit("launcher", "Blue", "launcher", "battery", 0, 0.02)
	target := makeUnit("red", "Red", "target", 0, 0.03)
	addWeapons(launcher, "sam", 4)
	defs := map[string]DefStats{
		"command":  {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"sensor":   {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000},
		"launcher": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"target":   {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}
	catalog := makeWeaponCatalog("sam", 50_000, 0.8, enginev1.UnitDomain_DOMAIN_AIR)

	adj := AdjudicateTick([]*enginev1.Unit{parent, radar, launcher, target}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected connected launcher to fire off shared track, got %d shots", len(adj.Shots))
	}
	if adj.Shots[0].Shooter.Id != launcher.Id {
		t.Fatalf("expected launcher to fire, got shooter %s", adj.Shots[0].Shooter.Id)
	}
}

func TestAdjudicateTick_UnconnectedLauncherCannotFireWithoutTrack(t *testing.T) {
	radar := makeUnit("radar", "Blue", "sensor", 0, 0)
	launcher := makeUnit("launcher", "Blue", "launcher", 0, 0.02)
	target := makeUnit("red", "Red", "target", 0, 0.03)
	addWeapons(launcher, "sam", 4)
	defs := map[string]DefStats{
		"sensor":   {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000},
		"launcher": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"target":   {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}
	catalog := makeWeaponCatalog("sam", 50_000, 0.8, enginev1.UnitDomain_DOMAIN_AIR)

	adj := AdjudicateTick([]*enginev1.Unit{radar, launcher, target}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Fatalf("expected unconnected launcher to hold fire without a shared track, got %d shots", len(adj.Shots))
	}
}

func TestAdjudicateTick_AttackOrderOverridesAutonomousTargetChoice(t *testing.T) {
	shooter := makeUnit("shooter", "Blue", "air", 0, 0)
	nearTarget := makeUnit("near", "Red", "ground", 0, 0.05)
	orderedTarget := makeUnit("ordered", "Red", "ground", 0, 0.2)
	addWeapons(shooter, "strike", 6)
	shooter.AttackOrder = &enginev1.AttackOrder{
		OrderType:      enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET,
		TargetUnitId:   orderedTarget.Id,
		DesiredEffect:  enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY,
		PkillThreshold: 0.7,
	}
	defs := map[string]DefStats{
		"air":    {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 100_000, TargetClass: "aircraft"},
		"ground": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 10_000, TargetClass: "soft_infrastructure"},
	}
	catalog := makeWeaponCatalogWithEffect("strike", 100_000, 0.9, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{shooter, nearTarget, orderedTarget}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected 1 ordered shot, got %d", len(adj.Shots))
	}
	if adj.Shots[0].Target.Id != orderedTarget.Id {
		t.Fatalf("expected assigned target to be engaged, got %s", adj.Shots[0].Target.Id)
	}
}

func TestAdjudicateTick_StrikeUntilEffect_HoldsOnceSatisfied(t *testing.T) {
	shooter := makeUnit("shooter", "Blue", "air", 0, 0)
	target := makeUnit("target", "Red", "airbase", 0, 0.1)
	addWeapons(shooter, "strike", 6)
	shooter.AttackOrder = &enginev1.AttackOrder{
		OrderType:      enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT,
		TargetUnitId:   target.Id,
		DesiredEffect:  enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL,
		PkillThreshold: 0.7,
	}
	target.DamageState = enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED
	defs := map[string]DefStats{
		"air":     {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 100_000, TargetClass: "aircraft"},
		"airbase": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 10_000, TargetClass: "runway"},
	}
	catalog := makeWeaponCatalogWithEffect("strike", 100_000, 0.9, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{shooter, target}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Fatalf("expected no shots once desired effect is already met, got %d", len(adj.Shots))
	}
}

func TestAdjudicateTick_HoldFire_PreventsAutonomousEngagement(t *testing.T) {
	shooter := makeUnit("shooter", "Blue", "air", 0, 0)
	target := makeUnit("target", "Red", "ground", 0, 0.05)
	shooter.EngagementBehavior = enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_HOLD_FIRE
	addWeapons(shooter, "strike", 6)
	defs := map[string]DefStats{
		"air":    {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 100_000, TargetClass: "aircraft"},
		"ground": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 10_000, TargetClass: "soft_infrastructure"},
	}
	catalog := makeWeaponCatalogWithEffect("strike", 100_000, 0.9, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{shooter, target}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Fatalf("expected hold-fire unit not to engage autonomously, got %d shots", len(adj.Shots))
	}
}

func TestAdjudicateTick_AssignedTargetsOnly_HoldsWithoutOrder(t *testing.T) {
	shooter := makeUnit("shooter", "Blue", "air", 0, 0)
	target := makeUnit("target", "Red", "ground", 0, 0.05)
	shooter.EngagementBehavior = enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_ASSIGNED_TARGETS_ONLY
	addWeapons(shooter, "strike", 6)
	defs := map[string]DefStats{
		"air":    {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 100_000, TargetClass: "aircraft"},
		"ground": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 10_000, TargetClass: "soft_infrastructure"},
	}
	catalog := makeWeaponCatalogWithEffect("strike", 100_000, 0.9, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{shooter, target}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Fatalf("expected assigned-targets-only unit to hold without orders, got %d shots", len(adj.Shots))
	}
}

func TestAdjudicateTick_SelfDefenseOnly_FiresWhenDetected(t *testing.T) {
	shooter := makeUnit("shooter", "Blue", "air", 0, 0)
	target := makeUnit("target", "Red", "ground", 0, 0.095)
	shooter.EngagementBehavior = enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_SELF_DEFENSE_ONLY
	addWeapons(shooter, "strike", 20)
	defs := map[string]DefStats{
		"air":    {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 20_000, TargetClass: "aircraft"},
		"ground": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 12_000, TargetClass: "soft_infrastructure"},
	}
	catalog := makeWeaponCatalogWithEffect("strike", 20_000, 0.2, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{shooter, target}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected self-defense unit to fire when detected, got %d shots", len(adj.Shots))
	}
}

func TestAdjudicateTick_AutoEngage_UsesConfiguredThreshold(t *testing.T) {
	shooter := makeUnit("shooter", "Blue", "air", 0, 0)
	target := makeUnit("target", "Red", "ground", 0, 0.05)
	shooter.EngagementBehavior = enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_AUTO_ENGAGE
	shooter.EngagementPkillThreshold = 0.8
	addWeapons(shooter, "strike", 6)
	defs := map[string]DefStats{
		"air":    {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 100_000, TargetClass: "aircraft"},
		"ground": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 1_000, TargetClass: "soft_infrastructure"},
	}
	catalog := makeWeaponCatalogWithEffect("strike", 100_000, 0.6, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{shooter, target}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Fatalf("expected high pkill threshold to suppress engagement, got %d shots", len(adj.Shots))
	}
}

func TestAdjudicateTick_AssignedTargetsOnly_AllowsManualOrder(t *testing.T) {
	shooter := makeUnit("shooter", "Blue", "air", 0, 0)
	target := makeUnit("target", "Red", "ground", 0, 0.05)
	shooter.EngagementBehavior = enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_ASSIGNED_TARGETS_ONLY
	shooter.AttackOrder = &enginev1.AttackOrder{
		OrderType:      enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET,
		TargetUnitId:   target.Id,
		DesiredEffect:  enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY,
		PkillThreshold: 0.7,
	}
	addWeapons(shooter, "strike", 6)
	defs := map[string]DefStats{
		"air":    {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 100_000, TargetClass: "aircraft"},
		"ground": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 10_000, TargetClass: "soft_infrastructure"},
	}
	catalog := makeWeaponCatalogWithEffect("strike", 100_000, 0.9, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{shooter, target}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected manual order to override assigned-targets-only hold, got %d shots", len(adj.Shots))
	}
}

// ─── ResolveArrivals ──────────────────────────────────────────────────────────

func TestResolveArrivals_EmptyArrived_NoHits(t *testing.T) {
	hits := ResolveArrivals(nil, nil, nil, nil, alwaysHit)
	if len(hits) != 0 {
		t.Errorf("expected 0 hits with no arrived munitions, got %d", len(hits))
	}
}

func TestResolveArrivals_AlwaysHit_Kill(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 0.01)
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000, TargetClass: "armor"},
	}
	catalog := makeWeaponCatalogWithEffect("gun", 50_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_ARMOR, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("setup: expected 1 shot, got %d", len(adj.Shots))
	}

	arrived := []*InFlightMunition{munitionFromShot(adj.Shots[0])}
	hits := ResolveArrivals(arrived, []*enginev1.Unit{a, b}, defs, catalog, alwaysHit)
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit on arrival with alwaysHit, got %d", len(hits))
	}
	if hits[0].Victim.Id != "b" {
		t.Errorf("expected victim b, got %s", hits[0].Victim.Id)
	}
	if !unitIsActive(a) {
		t.Error("attacker a should still be active")
	}
	if unitIsActive(b) {
		t.Error("victim b should be inactive after kill")
	}
}

func TestResolveArrivals_AlwaysMiss_NoKill(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 0.01)
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000, TargetClass: "armor"},
	}
	catalog := makeWeaponCatalogWithEffect("gun", 50_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_ARMOR, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	arrived := []*InFlightMunition{munitionFromShot(adj.Shots[0])}
	hits := ResolveArrivals(arrived, []*enginev1.Unit{a, b}, defs, catalog, alwaysMiss)
	if len(hits) != 0 {
		t.Errorf("expected 0 hits on miss, got %d", len(hits))
	}
	if !unitIsActive(b) {
		t.Error("b should still be active after miss")
	}
}

func TestResolveArrivals_DeadTarget_Skipped(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 0.01)
	killUnit(b) // already destroyed before munition arrives

	mun := &InFlightMunition{
		ID: NextMunitionID(), WeaponID: "gun", ShooterID: a.Id, TargetID: b.Id, HitProbability: 1.0,
	}
	hits := ResolveArrivals([]*InFlightMunition{mun}, []*enginev1.Unit{a, b}, map[string]DefStats{
		"def": {TargetClass: "armor"},
	}, makeWeaponCatalogWithEffect("gun", 50_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_ARMOR, enginev1.UnitDomain_DOMAIN_LAND), alwaysHit)
	if len(hits) != 0 {
		t.Errorf("already-dead target should not generate a hit, got %d", len(hits))
	}
}

func TestResolveArrivals_AttackerLookup(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 0.01)
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000, TargetClass: "armor"},
	}
	catalog := makeWeaponCatalogWithEffect("gun", 50_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_ARMOR, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	arrived := []*InFlightMunition{munitionFromShot(adj.Shots[0])}
	hits := ResolveArrivals(arrived, []*enginev1.Unit{a, b}, defs, catalog, alwaysHit)
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].Attacker == nil || hits[0].Attacker.Id != "a" {
		t.Errorf("expected attacker a, got %v", hits[0].Attacker)
	}
}

func TestResolveArrivals_MultipleMunitions(t *testing.T) {
	// Both sides fire; both munitions arrive; alwaysHit → both die.
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 0.01)
	addWeapons(a, "gun", 5)
	addWeapons(b, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 50_000, TargetClass: "armor"},
	}
	catalog := makeWeaponCatalogWithEffect("gun", 50_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_ARMOR, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil, nil, 0)
	var arrived []*InFlightMunition
	for _, s := range adj.Shots {
		arrived = append(arrived, munitionFromShot(s))
	}
	hits := ResolveArrivals(arrived, []*enginev1.Unit{a, b}, defs, catalog, alwaysHit)
	if len(hits) != 2 {
		t.Errorf("expected 2 hits (both die), got %d", len(hits))
	}
}

func TestResolveArrivals_LandStrikeAgainstRunway_MissionKill(t *testing.T) {
	attacker := makeUnit("a", "Blue", "shooter", 0, 0)
	target := makeUnit("b", "Red", "airbase", 0, 0.01)
	target.BaseOps = &enginev1.BaseOpsState{
		State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE,
	}
	defs := map[string]DefStats{
		"shooter": {Domain: enginev1.UnitDomain_DOMAIN_AIR, TargetClass: "aircraft"},
		"airbase": {Domain: enginev1.UnitDomain_DOMAIN_LAND, TargetClass: "runway"},
	}
	catalog := makeWeaponCatalogWithEffect("strike", 100_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	hits := ResolveArrivals([]*InFlightMunition{{
		ID:             NextMunitionID(),
		WeaponID:       "strike",
		ShooterID:      attacker.Id,
		TargetID:       target.Id,
		HitProbability: 1.0,
	}}, []*enginev1.Unit{attacker, target}, defs, catalog, alwaysHit)

	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if target.DamageState != enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED {
		t.Fatalf("expected runway target to be mission-killed, got %v", target.DamageState)
	}
	if target.GetBaseOps().GetState() != enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_CLOSED {
		t.Fatalf("expected runway hit to close airbase operations, got %v", target.GetBaseOps().GetState())
	}
	if !unitIsActive(target) {
		t.Fatal("runway target should remain present after mission kill")
	}
}

func TestResolveArrivals_LightDamageDegradesAirbaseOps(t *testing.T) {
	attacker := makeUnit("a", "Blue", "shooter", 0, 0)
	target := makeUnit("b", "Red", "airbase", 0, 0.01)
	target.BaseOps = &enginev1.BaseOpsState{
		State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE,
	}
	defs := map[string]DefStats{
		"shooter": {Domain: enginev1.UnitDomain_DOMAIN_AIR, TargetClass: "aircraft"},
		"airbase": {Domain: enginev1.UnitDomain_DOMAIN_LAND, TargetClass: "soft_infrastructure"},
	}
	catalog := makeWeaponCatalogWithEffect("strike", 100_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	hits := ResolveArrivals([]*InFlightMunition{{
		ID:             NextMunitionID(),
		WeaponID:       "strike",
		ShooterID:      attacker.Id,
		TargetID:       target.Id,
		HitProbability: 1.0,
	}}, []*enginev1.Unit{attacker, target}, defs, catalog, alwaysHit)

	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if target.DamageState != enginev1.DamageState_DAMAGE_STATE_DAMAGED {
		t.Fatalf("expected airbase target to be damaged, got %v", target.DamageState)
	}
	if target.GetBaseOps().GetState() != enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_DEGRADED {
		t.Fatalf("expected light damage to degrade airbase operations, got %v", target.GetBaseOps().GetState())
	}
}

func TestResolveArrivals_AntiShipSecondHit_DestroysTarget(t *testing.T) {
	attacker := makeUnit("a", "Blue", "ship", 0, 0)
	target := makeUnit("b", "Red", "ship", 0, 0.01)
	defs := map[string]DefStats{
		"ship": {Domain: enginev1.UnitDomain_DOMAIN_SEA, TargetClass: "surface_warship"},
	}
	catalog := makeWeaponCatalogWithEffect("asm", 100_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_SHIP, enginev1.UnitDomain_DOMAIN_SEA)
	arrived := []*InFlightMunition{
		{ID: NextMunitionID(), WeaponID: "asm", ShooterID: attacker.Id, TargetID: target.Id, HitProbability: 1.0},
		{ID: NextMunitionID(), WeaponID: "asm", ShooterID: attacker.Id, TargetID: target.Id, HitProbability: 1.0},
	}

	hits := ResolveArrivals(arrived, []*enginev1.Unit{attacker, target}, defs, catalog, alwaysHit)
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}
	if unitIsActive(target) {
		t.Fatal("expected second anti-ship hit to destroy target")
	}
}

// ─── SensorTick ───────────────────────────────────────────────────────────────

func TestSensorTick_NoUnits(t *testing.T) {
	result := SensorTick(nil, nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty DetectionSet with no units, got %v", result)
	}
}

func TestSensorTick_InRange_Detected(t *testing.T) {
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	red := makeUnit("red", "Red", "no_sensor", 0, 0.05) // ~5.5 km
	defs := map[string]DefStats{
		"sensor":    makeDef(10_000, 0),
		"no_sensor": makeDef(0, 0),
	}

	result := SensorTick([]*enginev1.Unit{blue, red}, defs, nil)
	ids, ok := result["BLUE"]
	if !ok {
		t.Fatal("BLUE team should have a detection entry")
	}
	found := false
	for _, id := range ids {
		if id == "red" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'red' in BLUE detections, got %v", ids)
	}
}

func TestSensorTick_OutOfRange_NotDetected(t *testing.T) {
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	red := makeUnit("red", "Red", "no_sensor", 0, 1.0) // ~111 km
	defs := map[string]DefStats{
		"sensor":    makeDef(10_000, 0),
		"no_sensor": makeDef(0, 0),
	}

	result := SensorTick([]*enginev1.Unit{blue, red}, defs, nil)
	ids := result["BLUE"]
	for _, id := range ids {
		if id == "red" {
			t.Error("red should not be detected at 111 km with 10 km sensor range")
		}
	}
}

func TestSensorTick_FriendlyNotDetected(t *testing.T) {
	b1 := makeUnit("b1", "Blue", "sensor", 0, 0)
	b2 := makeUnit("b2", "Blue", "sensor", 0, 0)
	defs := map[string]DefStats{"sensor": makeDef(100_000, 0)}

	result := SensorTick([]*enginev1.Unit{b1, b2}, defs, nil)
	for _, id := range result["BLUE"] {
		if id == "b1" || id == "b2" {
			t.Errorf("friendly unit %s should not appear in BLUE detections", id)
		}
	}
}

func TestSensorTick_DestroyedUnitSkipped(t *testing.T) {
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	red := makeUnit("red", "Red", "no_sensor", 0, 0.01)
	killUnit(red)
	defs := map[string]DefStats{
		"sensor":    makeDef(100_000, 0),
		"no_sensor": makeDef(0, 0),
	}

	result := SensorTick([]*enginev1.Unit{blue, red}, defs, nil)
	for _, id := range result["BLUE"] {
		if id == "red" {
			t.Error("destroyed unit should not be detected")
		}
	}
}

func TestSensorTick_EmptySliceForSideWithNoContacts(t *testing.T) {
	// Blue has sensor range but Red is too far; Blue's entry should still exist
	// (with an empty slice) so the frontend clears stale contacts.
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	red := makeUnit("red", "Red", "no_sensor", 0, 10.0) // very far
	defs := map[string]DefStats{
		"sensor":    makeDef(1_000, 0),
		"no_sensor": makeDef(0, 0),
	}

	result := SensorTick([]*enginev1.Unit{blue, red}, defs, nil)
	if _, ok := result["BLUE"]; !ok {
		t.Error("BLUE should have an entry even with zero contacts (to clear stale state)")
	}
	if len(result["BLUE"]) != 0 {
		t.Errorf("BLUE should have 0 detections, got %v", result["BLUE"])
	}
}

func TestSensorTick_BothSidesDetectEachOther(t *testing.T) {
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	red := makeUnit("red", "Red", "sensor", 0, 0.05) // ~5.5 km
	defs := map[string]DefStats{"sensor": makeDef(10_000, 0)}

	result := SensorTick([]*enginev1.Unit{blue, red}, defs, nil)

	foundRed := false
	for _, id := range result["BLUE"] {
		if id == "red" {
			foundRed = true
		}
	}
	foundBlue := false
	for _, id := range result["RED"] {
		if id == "blue" {
			foundBlue = true
		}
	}

	if !foundRed {
		t.Error("BLUE should detect RED")
	}
	if !foundBlue {
		t.Error("RED should detect BLUE")
	}
}

func TestSensorTick_StealthTargetCanAvoidDetection(t *testing.T) {
	blue := makeUnit("blue", "Blue", "def-blue", 0, 0)
	red := makeUnit("red", "Red", "def-red", 0, 0.036) // ~4 km
	defs := map[string]DefStats{
		"def-blue": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 10_000},
		"def-red":  {Domain: enginev1.UnitDomain_DOMAIN_AIR, RadarCrossSectionM2: 0.01},
	}

	result := SensorTick([]*enginev1.Unit{blue, red}, defs, nil)
	if len(result["BLUE"]) != 0 {
		t.Fatalf("expected stealth target to remain undetected, got %v", result["BLUE"])
	}
}

func TestSensorTick_LargeTargetDetectedEarlier(t *testing.T) {
	blue := makeUnit("blue", "Blue", "def-blue", 0, 0)
	red := makeUnit("red", "Red", "def-red", 0, 0.036) // ~4 km
	defs := map[string]DefStats{
		"def-blue": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 10_000},
		"def-red":  {Domain: enginev1.UnitDomain_DOMAIN_AIR, RadarCrossSectionM2: 16},
	}

	result := SensorTick([]*enginev1.Unit{blue, red}, defs, nil)
	if len(result["BLUE"]) != 1 || result["BLUE"][0] != "red" {
		t.Fatalf("expected large-RCS target to be detected, got %v", result["BLUE"])
	}
}

func TestSensorTick_DetectsUnauthorizedOverflight(t *testing.T) {
	defender := makeUnit("qat-sam", "Blue", "sam", 25.12, 51.31)
	defender.TeamId = "QAT"
	defender.CoalitionId = "Blue"
	intruder := makeUnit("isr-f35", "Blue", "fighter", 25.13, 51.31)
	intruder.TeamId = "ISR"
	intruder.CoalitionId = "Blue"
	intruder.Position.AltMsl = 5000

	defs := map[string]DefStats{
		"sam":     {DetectionRangeM: 100_000, Domain: enginev1.UnitDomain_DOMAIN_LAND, GeneralType: int32(enginev1.UnitGeneralType_GENERAL_TYPE_AIR_DEFENSE)},
		"fighter": {Domain: enginev1.UnitDomain_DOMAIN_AIR, RadarCrossSectionM2: 1},
	}
	rules := BuildRelationshipRules([]*enginev1.CountryRelationship{{
		FromCountry:            "ISR",
		ToCountry:              "QAT",
		AirspaceTransitAllowed: false,
	}})

	result := SensorTick([]*enginev1.Unit{defender, intruder}, defs, rules)
	if len(result["QAT"]) != 1 || result["QAT"][0] != "isr-f35" {
		t.Fatalf("expected QAT to detect unauthorized overflight, got %v", result["QAT"])
	}
}

func TestAdjudicateTick_EngagesUnauthorizedOverflight(t *testing.T) {
	defender := makeUnit("qat-sam", "Blue", "sam", 25.12, 51.31)
	defender.TeamId = "QAT"
	defender.CoalitionId = "Blue"
	defender.EngagementBehavior = enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_AUTO_ENGAGE
	addWeapons(defender, "sam-shot", 2)

	intruder := makeUnit("isr-f35", "Blue", "fighter", 25.13, 51.31)
	intruder.TeamId = "ISR"
	intruder.CoalitionId = "Blue"
	intruder.Position.AltMsl = 5000

	defs := map[string]DefStats{
		"sam":     {DetectionRangeM: 100_000, Domain: enginev1.UnitDomain_DOMAIN_LAND, GeneralType: int32(enginev1.UnitGeneralType_GENERAL_TYPE_AIR_DEFENSE)},
		"fighter": {Domain: enginev1.UnitDomain_DOMAIN_AIR, RadarCrossSectionM2: 1},
	}
	catalog := makeWeaponCatalog("sam-shot", 100_000, 1.0, enginev1.UnitDomain_DOMAIN_AIR)
	rules := BuildRelationshipRules([]*enginev1.CountryRelationship{{
		FromCountry:            "ISR",
		ToCountry:              "QAT",
		AirspaceTransitAllowed: false,
	}})

	adj := AdjudicateTick([]*enginev1.Unit{defender, intruder}, defs, catalog, nil, rules, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected sovereign air-defense shot, got %d", len(adj.Shots))
	}
	if adj.Shots[0].Shooter.Id != "qat-sam" || adj.Shots[0].Target.Id != "isr-f35" {
		t.Fatalf("unexpected sovereign defense shot: %+v", adj.Shots[0])
	}
}

func TestAdjudicateTick_GroundedFighterDoesNotEngageUnauthorizedOverflight(t *testing.T) {
	defender := makeUnit("qat-f16", "Blue", "fighter", 25.12, 51.31)
	defender.TeamId = "QAT"
	defender.CoalitionId = "Blue"
	defender.EngagementBehavior = enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_AUTO_ENGAGE
	addWeapons(defender, "aam", 2)
	defender.Position.AltMsl = 0

	intruder := makeUnit("isr-f35", "Blue", "fighter", 25.13, 51.31)
	intruder.TeamId = "ISR"
	intruder.CoalitionId = "Blue"
	intruder.Position.AltMsl = 5000

	defs := map[string]DefStats{
		"fighter": {DetectionRangeM: 100_000, Domain: enginev1.UnitDomain_DOMAIN_AIR, GeneralType: int32(enginev1.UnitGeneralType_GENERAL_TYPE_FIGHTER), RadarCrossSectionM2: 1},
	}
	catalog := makeWeaponCatalog("aam", 100_000, 1.0, enginev1.UnitDomain_DOMAIN_AIR)
	rules := BuildRelationshipRules([]*enginev1.CountryRelationship{{
		FromCountry:            "ISR",
		ToCountry:              "QAT",
		AirspaceTransitAllowed: false,
	}})

	adj := AdjudicateTick([]*enginev1.Unit{defender, intruder}, defs, catalog, nil, rules, 0)
	if len(adj.Shots) != 0 {
		t.Fatalf("expected grounded fighter to hold fire on unauthorized overflight, got %d shots", len(adj.Shots))
	}
}

func TestSensorTick_LandUnitWithoutSurfaceSearchCannotDetectShip(t *testing.T) {
	detector := makeUnit("brigade", "ISR", "brigade", 0, 0)
	target := makeUnit("ship", "IRN", "ship", 0, 0.05)
	defs := map[string]DefStats{
		"brigade": {DetectionRangeM: 100_000, Domain: enginev1.UnitDomain_DOMAIN_LAND, GeneralType: int32(enginev1.UnitGeneralType_GENERAL_TYPE_LIGHT_INFANTRY)},
		"ship":    {Domain: enginev1.UnitDomain_DOMAIN_SEA, RadarCrossSectionM2: 10},
	}

	result := SensorTick([]*enginev1.Unit{detector, target}, defs, nil)
	if len(result["ISR"]) != 0 {
		t.Fatalf("expected land unit without maritime sensor to miss ship, got %v", result["ISR"])
	}
}

func TestSensorTick_CoastalBatteryCanDetectShip(t *testing.T) {
	detector := makeUnit("coastal", "ISR", "battery", 0, 0)
	target := makeUnit("ship", "IRN", "ship", 0, 0.05)
	defs := map[string]DefStats{
		"battery": {
			DetectionRangeM: 100_000,
			Domain:          enginev1.UnitDomain_DOMAIN_LAND,
			GeneralType:     int32(enginev1.UnitGeneralType_GENERAL_TYPE_COASTAL_DEFENSE_MISSILE),
		},
		"ship": {Domain: enginev1.UnitDomain_DOMAIN_SEA, RadarCrossSectionM2: 10},
	}

	result := SensorTick([]*enginev1.Unit{detector, target}, defs, nil)
	if len(result["ISR"]) != 1 || result["ISR"][0] != "ship" {
		t.Fatalf("expected coastal battery to detect ship, got %v", result["ISR"])
	}
}

func TestSensorTick_AuthoredSensorSuiteOverridesInference(t *testing.T) {
	detector := makeUnit("custom-radar", "ISR", "radar", 0, 0)
	target := makeUnit("ship", "IRN", "ship", 0, 0.05)
	defs := map[string]DefStats{
		"radar": {
			DetectionRangeM: 10_000,
			Domain:          enginev1.UnitDomain_DOMAIN_LAND,
			GeneralType:     int32(enginev1.UnitGeneralType_GENERAL_TYPE_LIGHT_INFANTRY),
			SensorSuite: []SensorCapability{{
				SensorType:   enginev1.SensorType_SENSOR_TYPE_SURFACE_SEARCH,
				MaxRangeM:    100_000,
				TargetStates: []enginev1.SensorTargetState{enginev1.SensorTargetState_SENSOR_TARGET_STATE_SURFACE},
			}},
		},
		"ship": {Domain: enginev1.UnitDomain_DOMAIN_SEA, RadarCrossSectionM2: 10},
	}

	result := SensorTick([]*enginev1.Unit{detector, target}, defs, nil)
	if len(result["ISR"]) != 1 || result["ISR"][0] != "ship" {
		t.Fatalf("expected authored sensor suite to enable surface detection, got %v", result["ISR"])
	}
}

func TestSensorTick_FighterCannotDetectSubmarine(t *testing.T) {
	fighter := makeUnit("fighter", "ISR", "fighter", 0, 0)
	fighter.Position.AltMsl = 5_000
	sub := makeUnit("sub", "IRN", "sub", 0, 0.05)
	defs := map[string]DefStats{
		"fighter": {
			DetectionRangeM: 150_000,
			Domain:          enginev1.UnitDomain_DOMAIN_AIR,
			GeneralType:     int32(enginev1.UnitGeneralType_GENERAL_TYPE_FIGHTER),
		},
		"sub": {Domain: enginev1.UnitDomain_DOMAIN_SUBSURFACE, RadarCrossSectionM2: 1},
	}

	result := SensorTick([]*enginev1.Unit{fighter, sub}, defs, nil)
	if len(result["ISR"]) != 0 {
		t.Fatalf("expected fighter not to detect submarine, got %v", result["ISR"])
	}
}

func TestSensorTick_MaritimePatrolCanDetectSubmarine(t *testing.T) {
	mpa := makeUnit("mpa", "USA", "mpa", 0, 0)
	mpa.Position.AltMsl = 2_000
	sub := makeUnit("sub", "IRN", "sub", 0, 0.05)
	defs := map[string]DefStats{
		"mpa": {
			DetectionRangeM: 150_000,
			Domain:          enginev1.UnitDomain_DOMAIN_AIR,
			GeneralType:     int32(enginev1.UnitGeneralType_GENERAL_TYPE_MARITIME_PATROL),
		},
		"sub": {Domain: enginev1.UnitDomain_DOMAIN_SUBSURFACE, RadarCrossSectionM2: 1},
	}

	result := SensorTick([]*enginev1.Unit{mpa, sub}, defs, nil)
	if len(result["USA"]) != 1 || result["USA"][0] != "sub" {
		t.Fatalf("expected maritime patrol aircraft to detect submarine, got %v", result["USA"])
	}
}

func TestBuildTrackPicture_RetainProbabilityExceedsAcquireProbability(t *testing.T) {
	detector := makeUnit("fighter", "ISR", "fighter", 0, 0)
	detector.Position.AltMsl = 5_000
	target := makeUnit("target", "IRN", "target", 0, 0.72)
	target.Position.AltMsl = 5_000
	defs := map[string]DefStats{
		"fighter": {
			DetectionRangeM: 100_000,
			Domain:          enginev1.UnitDomain_DOMAIN_AIR,
			GeneralType:     int32(enginev1.UnitGeneralType_GENERAL_TYPE_FIGHTER),
		},
		"target": {
			Domain:              enginev1.UnitDomain_DOMAIN_AIR,
			GeneralType:         int32(enginev1.UnitGeneralType_GENERAL_TYPE_FIGHTER),
			RadarCrossSectionM2: 1,
		},
	}

	noTrack := buildTrackPicture([]*enginev1.Unit{detector, target}, defs, nil, nil, &sequenceRng{values: []float64{0.50}})
	if len(noTrack.BySide["ISR"]) != 0 {
		t.Fatalf("expected initial acquire roll to fail, got %v", noTrack.BySide["ISR"])
	}

	retained := buildTrackPicture([]*enginev1.Unit{detector, target}, defs, nil, detectionIndex{
		"fighter": {"target": true},
	}, &sequenceRng{values: []float64{0.50}})
	if len(retained.BySide["ISR"]) != 1 || retained.BySide["ISR"][0] != "target" {
		t.Fatalf("expected retained track to survive higher retain probability, got %v", retained.BySide["ISR"])
	}
}

func TestAdjudicateTick_StrategicStrikeUnitGetsCooldownAfterFiring(t *testing.T) {
	shooter := makeUnit("brigade", "Red", "brigade", 0, 0)
	shooter.Position.AltMsl = 0
	addWeapons(shooter, "ssm", 4)
	target := makeUnit("target", "Blue", "base", 0, 0.5)

	defs := map[string]DefStats{
		"brigade": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 100_000, TargetClass: "soft_infrastructure"},
		"base":    {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 10_000, TargetClass: "runway"},
	}
	catalog := makeWeaponCatalogWithEffect("ssm", 1_000_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{shooter, target}, defs, catalog, nil, nil, 600)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected one strike shot, got %d", len(adj.Shots))
	}
	if shooter.GetNextStrikeReadySeconds() != 4200 {
		t.Fatalf("expected next strike ready at 4200, got %v", shooter.GetNextStrikeReadySeconds())
	}
}

func TestAdjudicateTick_DamagedStrategicStrikeUnitHasLongerCooldown(t *testing.T) {
	shooter := makeUnit("brigade", "Red", "brigade", 0, 0)
	shooter.Position.AltMsl = 0
	shooter.DamageState = enginev1.DamageState_DAMAGE_STATE_DAMAGED
	addWeapons(shooter, "ssm", 4)
	target := makeUnit("target", "Blue", "base", 0, 0.5)

	defs := map[string]DefStats{
		"brigade": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 100_000, TargetClass: "soft_infrastructure"},
		"base":    {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 10_000, TargetClass: "runway"},
	}
	catalog := makeWeaponCatalogWithEffect("ssm", 1_000_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{shooter, target}, defs, catalog, nil, nil, 600)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected one strike shot, got %d", len(adj.Shots))
	}
	if shooter.GetNextStrikeReadySeconds() != 7800 {
		t.Fatalf("expected damaged strike ready at 7800, got %v", shooter.GetNextStrikeReadySeconds())
	}
}

func TestAdjudicateTick_ManualStrategicStrikeCanFireWithoutTrackAgainstFixedTarget(t *testing.T) {
	shooter := makeUnit("brigade", "Red", "brigade", 0, 0)
	shooter.Position.AltMsl = 0
	shooter.EngagementPkillThreshold = 0.1
	shooter.AttackOrder = &enginev1.AttackOrder{
		OrderType:      enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT,
		TargetUnitId:   "target",
		DesiredEffect:  enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL,
		PkillThreshold: 0.7,
	}
	addWeapons(shooter, "ssm", 4)
	target := makeUnit("target", "Blue", "base", 0, 1.0)

	defs := map[string]DefStats{
		"brigade": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 500, TargetClass: "soft_infrastructure"},
		"base":    {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 2_000_000, TargetClass: "runway", AssetClass: "airbase"},
	}
	catalog := makeWeaponCatalogWithEffect("ssm", 1_000_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{shooter, target}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 1 {
		t.Fatalf("expected strategic strike to fire without track against fixed target, got %d shots", len(adj.Shots))
	}
}

func TestAdjudicateTick_ManualStrategicStrikeStillNeedsTrackAgainstMobileTarget(t *testing.T) {
	shooter := makeUnit("brigade", "Red", "brigade", 0, 0)
	shooter.Position.AltMsl = 0
	shooter.EngagementPkillThreshold = 0.1
	shooter.AttackOrder = &enginev1.AttackOrder{
		OrderType:      enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT,
		TargetUnitId:   "target",
		DesiredEffect:  enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY,
		PkillThreshold: 0.7,
	}
	addWeapons(shooter, "ssm", 4)
	target := makeUnit("target", "Blue", "armor", 0, 1.0)

	defs := map[string]DefStats{
		"brigade": {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 500, TargetClass: "soft_infrastructure"},
		"armor":   {Domain: enginev1.UnitDomain_DOMAIN_LAND, DetectionRangeM: 2_000_000, TargetClass: "armor"},
	}
	catalog := makeWeaponCatalogWithEffect("ssm", 1_000_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{shooter, target}, defs, catalog, nil, nil, 0)
	if len(adj.Shots) != 0 {
		t.Fatalf("expected no strategic strike without track against mobile target, got %d shots", len(adj.Shots))
	}
}
