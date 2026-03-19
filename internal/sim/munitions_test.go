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
		TrackGroupID:  "Blue|shooter",
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

// ─── Guidance tracking ────────────────────────────────────────────────────────

// makeTrackingMunition builds a munition with the given guidance type aimed at
// a target that starts 1° north and moves to 1° north + 0.1° east each tick.
func makeTrackingMunition(guidance enginev1.GuidanceType) (*InFlightMunition, *enginev1.Unit) {
	target := makeUnit("tgt", "Red", "def", 1.0, 0.0)
	m := &InFlightMunition{
		ID:           "m1",
		WeaponID:     "test",
		ShooterID:    "shooter",
		ShooterSide:  "Blue",
		TrackGroupID: "Blue|shooter",
		TargetID:     "tgt",
		CurLat:       0, CurLon: 0,
		DestLat: 1.0, DestLon: 0.0,
		SpeedMps: 100,
		Guidance: guidance,
	}
	return m, target
}

func TestAdvanceMunitions_IR_TracksMovingTarget(t *testing.T) {
	m, target := makeTrackingMunition(enginev1.GuidanceType_GUIDANCE_IR)
	// Move target to a new position not in the original bearing.
	target.Position.Lon = 1.0
	units := []*enginev1.Unit{target}

	rem, _ := AdvanceMunitions([]*InFlightMunition{m}, 1.0, units, nil)
	if len(rem) == 0 {
		t.Fatal("munition should still be in flight")
	}
	// Destination should have been updated to the target's new longitude.
	if rem[0].DestLon != 1.0 {
		t.Errorf("IR munition should track target: expected DestLon=1.0, got %.4f", rem[0].DestLon)
	}
}

func TestAdvanceMunitions_GPS_DoesNotTrack(t *testing.T) {
	m, target := makeTrackingMunition(enginev1.GuidanceType_GUIDANCE_GPS)
	originalDestLon := m.DestLon
	target.Position.Lon = 1.0 // target moves
	units := []*enginev1.Unit{target}

	rem, _ := AdvanceMunitions([]*InFlightMunition{m}, 1.0, units, nil)
	if len(rem) == 0 {
		t.Fatal("munition should still be in flight")
	}
	if rem[0].DestLon != originalDestLon {
		t.Errorf("GPS munition should not track: expected DestLon=%.4f, got %.4f", originalDestLon, rem[0].DestLon)
	}
}

func TestAdvanceMunitions_Radar_TracksWhenLockHeld(t *testing.T) {
	m, target := makeTrackingMunition(enginev1.GuidanceType_GUIDANCE_RADAR)
	shooter := makeUnit("shooter", "Blue", "sensor", 0, 0)
	target.Position.Lon = 1.0
	units := []*enginev1.Unit{shooter, target}
	defs := map[string]DefStats{
		"sensor": {DetectionRangeM: 500_000, Domain: enginev1.UnitDomain_DOMAIN_AIR},
		"def":    {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}

	rem, _ := AdvanceMunitions([]*InFlightMunition{m}, 1.0, units, defs)
	if len(rem) == 0 {
		t.Fatal("munition should still be in flight")
	}
	if rem[0].DestLon != 1.0 {
		t.Errorf("radar munition should track when lock held: expected DestLon=1.0, got %.4f", rem[0].DestLon)
	}
}

func TestAdvanceMunitions_Radar_FixedPointWhenLockLost(t *testing.T) {
	m, target := makeTrackingMunition(enginev1.GuidanceType_GUIDANCE_RADAR)
	originalDestLon := m.DestLon
	shooter := makeUnit("shooter", "Blue", "sensor", 0, 0)
	target.Position.Lon = 1.0
	units := []*enginev1.Unit{shooter, target}
	defs := map[string]DefStats{
		"sensor": {DetectionRangeM: 10_000, Domain: enginev1.UnitDomain_DOMAIN_AIR},
		"def":    {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}

	rem, _ := AdvanceMunitions([]*InFlightMunition{m}, 1.0, units, defs)
	if len(rem) == 0 {
		t.Fatal("munition should still be in flight")
	}
	if rem[0].DestLon != originalDestLon {
		t.Errorf("radar munition should hold last known pos when lock lost: expected DestLon=%.4f, got %.4f", originalDestLon, rem[0].DestLon)
	}
}

func TestAdvanceMunitions_Radar_TracksWhenSiblingSensorHoldsLock(t *testing.T) {
	m, target := makeTrackingMunition(enginev1.GuidanceType_GUIDANCE_RADAR)
	parent := makeUnit("battery", "Blue", "command", 0, 0)
	shooter := makeChildUnit("shooter", "Blue", "launcher", "battery", 0, 0)
	sensor := makeChildUnit("sensor", "Blue", "sensor", "battery", 0, 0)
	m.TrackGroupID = "Blue|battery"
	target.Position.Lon = 1.0
	units := []*enginev1.Unit{parent, shooter, sensor, target}
	defs := map[string]DefStats{
		"command":  {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"launcher": {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"sensor":   {DetectionRangeM: 500_000, Domain: enginev1.UnitDomain_DOMAIN_AIR},
		"def":      {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}

	rem, _ := AdvanceMunitions([]*InFlightMunition{m}, 1.0, units, defs)
	if len(rem) == 0 {
		t.Fatal("munition should still be in flight")
	}
	if rem[0].DestLon != 1.0 {
		t.Errorf("radar munition should track from sibling sensor lock: expected DestLon=1.0, got %.4f", rem[0].DestLon)
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
	ids, ok := result["Blue"]
	if !ok || len(ids) == 0 {
		t.Fatal("Blue should detect the munition")
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
	for _, id := range result["Blue"] {
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
	for _, id := range result["Blue"] {
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
	for _, id := range result["Blue"] {
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
	for _, id := range result["Blue"] {
		if id == "m1" {
			blueDetects = true
		}
	}
	redDetects := false
	for _, id := range result["Red"] {
		if id == "m1" {
			redDetects = true
		}
	}

	if !blueDetects {
		t.Error("Blue should detect the munition")
	}
	if !redDetects {
		t.Error("Red should detect the incoming munition")
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
		ShooterSide:   "Red",
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
	weapons := makeWeaponCatalog("sam-shot", 100_000, 1.0, enginev1.UnitDomain_DOMAIN_AIR)

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
	if defender.Weapons[0].CurrentQty != 1 {
		t.Fatalf("expected interceptor ammo to decrement, got %d", defender.Weapons[0].CurrentQty)
	}
}
