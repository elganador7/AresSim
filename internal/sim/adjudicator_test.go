package sim

import (
	"math"
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func approxEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// ─── helpers ──────────────────────────────────────────────────────────────────

func makeUnit(id, side, defID string, lat, lon float64) *enginev1.Unit {
	return &enginev1.Unit{
		Id:           id,
		Side:         side,
		DefinitionId: defID,
		Position:     &enginev1.Position{Lat: lat, Lon: lon},
		Status:       &enginev1.OperationalStatus{IsActive: true},
	}
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

// alwaysHit and alwaysMiss are convenience instances.
const alwaysHit = constRng(0.0)
const alwaysMiss = constRng(1.0)

func makeWeaponCatalog(id string, rangeM, prob float64, domains ...enginev1.UnitDomain) map[string]WeaponStats {
	return map[string]WeaponStats{
		id: {RangeM: rangeM, ProbabilityOfHit: prob, DomainTargets: domains},
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

// ─── AdjudicateTick ───────────────────────────────────────────────────────────

func TestAdjudicateTick_NoUnits_NoShots(t *testing.T) {
	adj := AdjudicateTick(nil, nil, nil, nil)
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
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
	if len(adj.Shots) != 0 {
		t.Errorf("friendly fire: expected 0 shots, got %d", len(adj.Shots))
	}
}

func TestAdjudicateTick_OutOfRange_NoShots(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 1.8) // ~200 km at equator
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
	if len(adj.Shots) != 0 {
		t.Errorf("out-of-range: expected 0 shots, got %d", len(adj.Shots))
	}
}

func TestAdjudicateTick_InRange_ShotFired(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01) // ~1.1 km
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
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
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
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
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
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

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
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

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
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
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
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
		"long":  {Domain: enginev1.UnitDomain_DOMAIN_AIR},
		"short": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}
	catalog := makeWeaponCatalog("missile", 500_000, 0.9, enginev1.UnitDomain_DOMAIN_AIR)

	adj := AdjudicateTick([]*enginev1.Unit{blue, r1, r2, r3}, defs, catalog, nil)
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
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
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

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
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
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 0},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 5_000},
	}
	catalog := makeWeaponCatalog("missile", 50_000, 0.4, enginev1.UnitDomain_DOMAIN_AIR)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
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

func TestAdjudicateTick_SalvoSizedToThreshold(t *testing.T) {
	a := makeUnit("a", "Blue", "def-a", 0, 0)
	b := makeUnit("b", "Red", "def-b", 0, 0.01)
	addWeapons(a, "missile", 10)
	defs := map[string]DefStats{
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}
	catalog := makeWeaponCatalog("missile", 50_000, 0.6, enginev1.UnitDomain_DOMAIN_AIR)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
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
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 5_000},
	}
	catalog := makeWeaponCatalog("missile", 50_000, 0.4, enginev1.UnitDomain_DOMAIN_AIR)
	inFlight := repeatInFlight("b", 0.4, 2)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, inFlight)
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
		"def-a": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
		"def-b": {Domain: enginev1.UnitDomain_DOMAIN_AIR, DetectionRangeM: 5_000},
	}
	catalog := makeWeaponCatalog("missile", 50_000, 0.4, enginev1.UnitDomain_DOMAIN_AIR)
	inFlight := repeatInFlight("b", 0.4, 3)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, inFlight)
	if len(adj.Shots) != 0 {
		t.Fatalf("expected no new shot once in-flight salvo already exceeds 70%% pkill, got %d", len(adj.Shots))
	}
	if a.Weapons[0].CurrentQty != 10 {
		t.Errorf("ammo should be unchanged when existing salvo is sufficient, got %d", a.Weapons[0].CurrentQty)
	}
}

// ─── ResolveArrivals ──────────────────────────────────────────────────────────

func TestResolveArrivals_EmptyArrived_NoKills(t *testing.T) {
	kills := ResolveArrivals(nil, nil, alwaysHit)
	if len(kills) != 0 {
		t.Errorf("expected 0 kills with no arrived munitions, got %d", len(kills))
	}
}

func TestResolveArrivals_AlwaysHit_Kill(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 0.01)
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
	if len(adj.Shots) != 1 {
		t.Fatalf("setup: expected 1 shot, got %d", len(adj.Shots))
	}

	arrived := []*InFlightMunition{munitionFromShot(adj.Shots[0])}
	kills := ResolveArrivals(arrived, []*enginev1.Unit{a, b}, alwaysHit)
	if len(kills) != 1 {
		t.Fatalf("expected 1 kill on arrival with alwaysHit, got %d", len(kills))
	}
	if kills[0].Victim.Id != "b" {
		t.Errorf("expected victim b, got %s", kills[0].Victim.Id)
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
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
	arrived := []*InFlightMunition{munitionFromShot(adj.Shots[0])}
	kills := ResolveArrivals(arrived, []*enginev1.Unit{a, b}, alwaysMiss)
	if len(kills) != 0 {
		t.Errorf("expected 0 kills on miss, got %d", len(kills))
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
		ID: NextMunitionID(), ShooterID: a.Id, TargetID: b.Id, HitProbability: 1.0,
	}
	kills := ResolveArrivals([]*InFlightMunition{mun}, []*enginev1.Unit{a, b}, alwaysHit)
	if len(kills) != 0 {
		t.Errorf("already-dead target should not generate a kill, got %d", len(kills))
	}
}

func TestResolveArrivals_AttackerLookup(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 0.01)
	addWeapons(a, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
	arrived := []*InFlightMunition{munitionFromShot(adj.Shots[0])}
	kills := ResolveArrivals(arrived, []*enginev1.Unit{a, b}, alwaysHit)
	if len(kills) != 1 {
		t.Fatalf("expected 1 kill, got %d", len(kills))
	}
	if kills[0].Attacker == nil || kills[0].Attacker.Id != "a" {
		t.Errorf("expected attacker a, got %v", kills[0].Attacker)
	}
}

func TestResolveArrivals_MultipleMunitions(t *testing.T) {
	// Both sides fire; both munitions arrive; alwaysHit → both die.
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 0.01)
	addWeapons(a, "gun", 5)
	addWeapons(b, "gun", 5)
	defs := map[string]DefStats{
		"def": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	catalog := makeWeaponCatalog("gun", 50_000, 1.0, enginev1.UnitDomain_DOMAIN_LAND)

	adj := AdjudicateTick([]*enginev1.Unit{a, b}, defs, catalog, nil)
	var arrived []*InFlightMunition
	for _, s := range adj.Shots {
		arrived = append(arrived, munitionFromShot(s))
	}
	kills := ResolveArrivals(arrived, []*enginev1.Unit{a, b}, alwaysHit)
	if len(kills) != 2 {
		t.Errorf("expected 2 kills (both die), got %d", len(kills))
	}
}

// ─── SensorTick ───────────────────────────────────────────────────────────────

func TestSensorTick_NoUnits(t *testing.T) {
	result := SensorTick(nil, nil)
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

	result := SensorTick([]*enginev1.Unit{blue, red}, defs)
	ids, ok := result["Blue"]
	if !ok {
		t.Fatal("Blue side should have a detection entry")
	}
	found := false
	for _, id := range ids {
		if id == "red" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'red' in Blue detections, got %v", ids)
	}
}

func TestSensorTick_OutOfRange_NotDetected(t *testing.T) {
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	red := makeUnit("red", "Red", "no_sensor", 0, 1.0) // ~111 km
	defs := map[string]DefStats{
		"sensor":    makeDef(10_000, 0),
		"no_sensor": makeDef(0, 0),
	}

	result := SensorTick([]*enginev1.Unit{blue, red}, defs)
	ids := result["Blue"]
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

	result := SensorTick([]*enginev1.Unit{b1, b2}, defs)
	for _, id := range result["Blue"] {
		if id == "b1" || id == "b2" {
			t.Errorf("friendly unit %s should not appear in Blue detections", id)
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

	result := SensorTick([]*enginev1.Unit{blue, red}, defs)
	for _, id := range result["Blue"] {
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

	result := SensorTick([]*enginev1.Unit{blue, red}, defs)
	if _, ok := result["Blue"]; !ok {
		t.Error("Blue should have an entry even with zero contacts (to clear stale state)")
	}
	if len(result["Blue"]) != 0 {
		t.Errorf("Blue should have 0 detections, got %v", result["Blue"])
	}
}

func TestSensorTick_BothSidesDetectEachOther(t *testing.T) {
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	red := makeUnit("red", "Red", "sensor", 0, 0.05) // ~5.5 km
	defs := map[string]DefStats{"sensor": makeDef(10_000, 0)}

	result := SensorTick([]*enginev1.Unit{blue, red}, defs)

	foundRed := false
	for _, id := range result["Blue"] {
		if id == "red" {
			foundRed = true
		}
	}
	foundBlue := false
	for _, id := range result["Red"] {
		if id == "blue" {
			foundBlue = true
		}
	}

	if !foundRed {
		t.Error("Blue should detect Red")
	}
	if !foundBlue {
		t.Error("Red should detect Blue")
	}
}
