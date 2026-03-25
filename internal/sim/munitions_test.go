package sim

import (
	"strings"
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// ─── AdvanceMunitions ──────────────────────────────────────────────────────────

func makeMunition(id string, curLat, curLon, destLat, destLon, speedMps float64, domains ...enginev1.UnitDomain) *InFlightMunition {
	return &InFlightMunition{
		ID:            id,
		WeaponID:      "test-weapon",
		ShooterID:     "shooter",
		LaunchLat:     curLat,
		LaunchLon:     curLon,
		MaxRangeM:     500_000,
		CurLat:        curLat,
		CurLon:        curLon,
		DestLat:       destLat,
		DestLon:       destLon,
		SpeedMps:      speedMps,
		TargetDomains: domains,
	}
}

func TestAdvanceMunitions_MovesCloser(t *testing.T) {
	// Munition at origin, destination ~111 km north, speed 1000 m/s.
	// One tick (timeScale=1) it should move 1000 m north and remain in-flight.
	m := makeMunition("m1", 0, 0, 1.0, 0, 1000, enginev1.UnitDomain_DOMAIN_LAND)
	remaining, arrived := AdvanceMunitions([]*InFlightMunition{m}, 1.0, nil, nil)
	if len(arrived) != 0 {
		t.Errorf("expected no arrivals, got %d", len(arrived))
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining, got %d", len(remaining))
	}
	if remaining[0].CurLat <= 0 {
		t.Errorf("expected munition to move north, CurLat=%f", remaining[0].CurLat)
	}
}

func TestAdvanceMunitions_ArrivesWhenCanCoverDistance(t *testing.T) {
	// Destination is 100 m away, speed is 1000 m/s — should arrive in one tick.
	m := makeMunition("m1", 0, 0, 0.0009, 0, 1000, enginev1.UnitDomain_DOMAIN_LAND)
	// Distance ~100 m; 1000 m/s covers it in one tick.
	remaining, arrived := AdvanceMunitions([]*InFlightMunition{m}, 1.0, nil, nil)
	if len(arrived) != 1 {
		t.Errorf("expected 1 arrival, got %d", len(arrived))
	}
	if len(remaining) != 0 {
		t.Errorf("expected 0 remaining, got %d", len(remaining))
	}
}

func TestAdvanceMunitions_TimeScaleAffectsSpeed(t *testing.T) {
	// At 10x time scale, munition covers 10x distance per tick.
	slow := makeMunition("s", 0, 0, 1.0, 0, 100, enginev1.UnitDomain_DOMAIN_LAND)
	fast := makeMunition("f", 0, 0, 1.0, 0, 100, enginev1.UnitDomain_DOMAIN_LAND)

	remSlow, _ := AdvanceMunitions([]*InFlightMunition{slow}, 1.0, nil, nil)
	remFast, _ := AdvanceMunitions([]*InFlightMunition{fast}, 10.0, nil, nil)

	if len(remSlow) == 0 {
		t.Fatal("slow munition should not have arrived at 1x")
	}
	if len(remFast) == 0 {
		t.Fatal("fast munition should not have arrived (destination is 1° / ~111 km)")
	}
	// Fast should be 10x further than slow.
	latSlow := remSlow[0].CurLat
	latFast := remFast[0].CurLat
	ratio := latFast / latSlow
	if ratio < 9 || ratio > 11 {
		t.Errorf("expected 10x distance ratio, got %.2f (slow=%.6f fast=%.6f)", ratio, latSlow, latFast)
	}
}

func TestAdvanceMunitions_EmptyInput(t *testing.T) {
	rem, arr := AdvanceMunitions(nil, 1.0, nil, nil)
	if len(rem) != 0 || len(arr) != 0 {
		t.Error("expected nil/empty slices for empty input")
	}
}

func TestAdvanceMunitions_MultipleOneMayArrive(t *testing.T) {
	near := makeMunition("near", 0, 0, 0.0009, 0, 1000, enginev1.UnitDomain_DOMAIN_LAND) // arrives
	far := makeMunition("far", 0, 0, 5.0, 0, 100, enginev1.UnitDomain_DOMAIN_LAND)       // remains
	rem, arr := AdvanceMunitions([]*InFlightMunition{near, far}, 1.0, nil, nil)
	if len(arr) != 1 || arr[0].ID != "near" {
		t.Errorf("expected 'near' to arrive, got %v", arr)
	}
	if len(rem) != 1 || rem[0].ID != "far" {
		t.Errorf("expected 'far' to remain, got %v", rem)
	}
}

// ─── Direct chase / range escape ─────────────────────────────────────────────

func makeTrackingMunition() (*InFlightMunition, *enginev1.Unit) {
	target := makeUnit("tgt", "Red", "def", 1.0, 0.0)
	m := &InFlightMunition{
		ID:          "m1",
		WeaponID:    "test",
		ShooterID:   "shooter",
		ShooterTeam: "USA",
		TargetID:    "tgt",
		LaunchLat:   0,
		LaunchLon:   0,
		MaxRangeM:   500_000,
		CurLat:      0, CurLon: 0,
		DestLat: 1.0, DestLon: 0.0,
		SpeedMps: 100,
	}
	return m, target
}

func TestAdvanceMunitions_TracksMovingTargetDirectly(t *testing.T) {
	m, target := makeTrackingMunition()
	target.Position.Lon = 1.0
	units := []*enginev1.Unit{target}

	rem, _ := AdvanceMunitions([]*InFlightMunition{m}, 1.0, units, nil)
	if len(rem) == 0 {
		t.Fatal("munition should still be in flight")
	}
	if rem[0].DestLon != 1.0 {
		t.Errorf("munition should chase moving target: expected DestLon=1.0, got %.4f", rem[0].DestLon)
	}
}

func TestAdvanceMunitions_DropsWhenTargetEscapesRange(t *testing.T) {
	m, target := makeTrackingMunition()
	m.MaxRangeM = 50_000
	target.Position.Lat = 1.0
	units := []*enginev1.Unit{target}

	rem, _ := AdvanceMunitions([]*InFlightMunition{m}, 1.0, units, nil)
	if len(rem) != 0 {
		t.Fatalf("expected munition to fail once target escapes weapon range, got %d remaining", len(rem))
	}
}

// ─── DetectMunitions ──────────────────────────────────────────────────────────

func TestDetectMunitions_NoMunitions(t *testing.T) {
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	defs := map[string]DefStats{"sensor": {DetectionRangeM: 100_000, Domain: enginev1.UnitDomain_DOMAIN_LAND}}
	result := DetectMunitions([]*enginev1.Unit{blue}, defs, nil)
	if len(result) != 0 {
		t.Error("expected no detections with no munitions")
	}
}

func TestDetectMunitions_InRangeCorrectDomain(t *testing.T) {
	// Blue land unit detects a land-targeting munition close by.
	blue := makeUnit("blue", "Blue", "land-sensor", 0, 0)
	m := makeMunition("m1", 0, 0.05, 0, 1, 500, enginev1.UnitDomain_DOMAIN_LAND) // ~5.5 km away
	defs := map[string]DefStats{
		"land-sensor": {DetectionRangeM: 10_000, Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}

	result := DetectMunitions([]*enginev1.Unit{blue}, defs, []*InFlightMunition{m})
	ids, ok := result["BLUE"]
	if !ok || len(ids) == 0 {
		t.Fatal("BLUE should detect the munition")
	}
	if ids[0] != "m1" {
		t.Errorf("expected munition id 'm1', got %s", ids[0])
	}
}

func TestDetectMunitions_OutOfRange_NotDetected(t *testing.T) {
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	m := makeMunition("m1", 0, 5.0, 0, 10, 500, enginev1.UnitDomain_DOMAIN_LAND) // ~555 km
	defs := map[string]DefStats{
		"sensor": {DetectionRangeM: 10_000, Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}

	result := DetectMunitions([]*enginev1.Unit{blue}, defs, []*InFlightMunition{m})
	for _, id := range result["BLUE"] {
		if id == "m1" {
			t.Error("munition at 555 km should not be detected with 10 km sensor")
		}
	}
}

func TestDetectMunitions_WrongDomain_NotDetected(t *testing.T) {
	// Land unit cannot detect an air-targeting munition.
	blue := makeUnit("blue", "Blue", "land-sensor", 0, 0)
	m := makeMunition("m1", 0, 0.01, 0, 1, 500, enginev1.UnitDomain_DOMAIN_AIR) // air-targeting
	defs := map[string]DefStats{
		"land-sensor": {DetectionRangeM: 100_000, Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}

	result := DetectMunitions([]*enginev1.Unit{blue}, defs, []*InFlightMunition{m})
	for _, id := range result["BLUE"] {
		if id == "m1" {
			t.Error("land sensor should not detect air-targeting munition")
		}
	}
}

func TestDetectMunitions_CorrectDomain_Detected(t *testing.T) {
	// Air unit can detect an air-targeting munition (incoming AAM).
	blue := makeUnit("blue", "Blue", "air-sensor", 0, 0)
	m := makeMunition("m1", 0, 0.1, 0, 1, 1000, enginev1.UnitDomain_DOMAIN_AIR)
	defs := map[string]DefStats{
		"air-sensor": {DetectionRangeM: 100_000, Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}

	result := DetectMunitions([]*enginev1.Unit{blue}, defs, []*InFlightMunition{m})
	found := false
	for _, id := range result["BLUE"] {
		if id == "m1" {
			found = true
		}
	}
	if !found {
		t.Error("air sensor should detect air-targeting munition")
	}
}

func TestDetectMunitions_DestroyedUnit_NotDetector(t *testing.T) {
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	killUnit(blue)
	m := makeMunition("m1", 0, 0.01, 0, 1, 500, enginev1.UnitDomain_DOMAIN_LAND)
	defs := map[string]DefStats{
		"sensor": {DetectionRangeM: 100_000, Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}

	result := DetectMunitions([]*enginev1.Unit{blue}, defs, []*InFlightMunition{m})
	if len(result) != 0 {
		t.Error("destroyed unit should not contribute to munition detection")
	}
}

func TestDetectMunitions_BothSidesCanDetect(t *testing.T) {
	// A land-targeting munition fired at a Red unit; both Blue and Red land platforms
	// in range can detect it (e.g. the shooter can track their own missile if in range).
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	red := makeUnit("red", "Red", "sensor", 0, 0.02)
	m := makeMunition("m1", 0, 0.01, 0.5, 0, 500, enginev1.UnitDomain_DOMAIN_LAND)
	defs := map[string]DefStats{
		"sensor": {DetectionRangeM: 50_000, Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}

	result := DetectMunitions([]*enginev1.Unit{blue, red}, defs, []*InFlightMunition{m})
	blueDetects := false
	for _, id := range result["BLUE"] {
		if id == "m1" {
			blueDetects = true
		}
	}
	redDetects := false
	for _, id := range result["RED"] {
		if id == "m1" {
			redDetects = true
		}
	}

	if !blueDetects {
		t.Error("BLUE should detect the munition")
	}
	if !redDetects {
		t.Error("RED should detect the incoming munition")
	}
}

func TestNextMunitionID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NextMunitionID()
		if ids[id] {
			t.Errorf("duplicate munition ID: %s", id)
		}
		ids[id] = true
		if !strings.HasPrefix(id, "mun-") {
			t.Errorf("expected 'mun-' prefix, got %s", id)
		}
	}
}

func TestInterceptMunitionsTick_InterceptsThreatToSide(t *testing.T) {
	defender := makeUnit("qat-sam", "Blue", "sam", 25.12, 51.31)
	defender.TeamId = "QAT"
	addWeapons(defender, "sam-shot", 2)
	target := makeUnit("qat-base", "Blue", "base", 25.12, 51.31)
	target.TeamId = "QAT"
	m := &InFlightMunition{
		ID:            "m1",
		WeaponID:      "raid",
		ShooterID:     "irn-launcher",
		ShooterTeam:   "IRN",
		TargetID:      "qat-base",
		CurLat:        25.20,
		CurLon:        51.31,
		CurAltMsl:     1500,
		TargetDomains: []enginev1.UnitDomain{enginev1.UnitDomain_DOMAIN_LAND},
	}
	defs := map[string]DefStats{
		"sam":  {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"base": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
	}
	weapons := makeWeaponCatalogWithEffect("sam-shot", 100_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_INTERCEPTOR, enginev1.UnitDomain_DOMAIN_AIR)

	remaining, shots := InterceptMunitionsTick(
		[]*enginev1.Unit{defender, target},
		defs,
		weapons,
		[]*InFlightMunition{m},
		map[string][]string{"QAT": {"m1"}},
		alwaysHit,
	)
	if len(remaining) != 0 {
		t.Fatalf("expected munition to be intercepted, got %d remaining", len(remaining))
	}
	if len(shots) != 1 || shots[0].Defender.Id != "qat-sam" {
		t.Fatalf("expected one interceptor shot from qat-sam, got %+v", shots)
	}
	if !shots[0].Success {
		t.Fatal("expected interceptor shot to be marked successful")
	}
	if defender.Weapons[0].CurrentQty != 1 {
		t.Fatalf("expected interceptor ammo to decrement, got %d", defender.Weapons[0].CurrentQty)
	}
}

func TestInterceptMunitionsTick_AircraftDoNotInterceptLandTargetingMissiles(t *testing.T) {
	defender := makeUnit("fighter", "Blue", "fighter", 25.12, 51.31)
	defender.TeamId = "ISR"
	addWeapons(defender, "aam-shot", 2)
	m := &InFlightMunition{
		ID:            "m1",
		WeaponID:      "raid",
		ShooterID:     "irn-launcher",
		ShooterTeam:   "IRN",
		TargetID:      "isr-base",
		CurLat:        25.20,
		CurLon:        51.31,
		CurAltMsl:     1500,
		TargetDomains: []enginev1.UnitDomain{enginev1.UnitDomain_DOMAIN_LAND},
	}
	defs := map[string]DefStats{
		"fighter": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}
	weapons := makeWeaponCatalogWithEffect("aam-shot", 100_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_AIR, enginev1.UnitDomain_DOMAIN_AIR)

	remaining, shots := InterceptMunitionsTick(
		[]*enginev1.Unit{defender},
		defs,
		weapons,
		[]*InFlightMunition{m},
		map[string][]string{"ISR": {"m1"}},
		alwaysHit,
	)
	if len(shots) != 0 {
		t.Fatalf("expected no interceptor shots from aircraft against land-targeting missile, got %+v", shots)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected missile to remain in flight, got %d remaining", len(remaining))
	}
}

func TestInterceptMunitionsTick_AircraftDoNotInterceptAirTargetingMissilesWithoutInterceptorWeapons(t *testing.T) {
	defender := makeUnit("fighter", "Blue", "fighter", 25.12, 51.31)
	defender.TeamId = "USA"
	addWeapons(defender, "aam-shot", 2)
	m := &InFlightMunition{
		ID:            "m1",
		WeaponID:      "sam-raid",
		ShooterID:     "irn-sam",
		ShooterTeam:   "IRN",
		TargetID:      "usa-awacs",
		CurLat:        25.20,
		CurLon:        51.31,
		CurAltMsl:     9000,
		TargetDomains: []enginev1.UnitDomain{enginev1.UnitDomain_DOMAIN_AIR},
	}
	defs := map[string]DefStats{
		"fighter": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}
	weapons := makeWeaponCatalogWithEffect("aam-shot", 100_000, 1.0, enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_AIR, enginev1.UnitDomain_DOMAIN_AIR)

	remaining, shots := InterceptMunitionsTick(
		[]*enginev1.Unit{defender},
		defs,
		weapons,
		[]*InFlightMunition{m},
		map[string][]string{"USA": {"m1"}},
		alwaysHit,
	)
	if len(shots) != 0 {
		t.Fatalf("expected no interceptor shots from aircraft with AAMs, got %+v", shots)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected missile to remain in flight, got %d remaining", len(remaining))
	}
}
