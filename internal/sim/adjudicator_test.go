package sim

import (
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

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

func makeDef(combatRangeM, detectionRangeM, baseStrength float64) DefStats {
	return DefStats{
		CruiseSpeedMps:  10,
		CombatRangeM:    combatRangeM,
		BaseStrength:    baseStrength,
		DetectionRangeM: detectionRangeM,
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

// ─── AdjudicateTick ───────────────────────────────────────────────────────────

func TestAdjudicate_NoUnits(t *testing.T) {
	kills := AdjudicateTick(nil, nil)
	if len(kills) != 0 {
		t.Errorf("expected 0 kills with no units, got %d", len(kills))
	}
}

func TestAdjudicate_FriendlyFire_NotAllowed(t *testing.T) {
	// Two Blue units within range of each other — no combat.
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Blue", "def", 0, 0.001)
	defs := map[string]DefStats{"def": makeDef(50_000, 0, 1)}

	kills := AdjudicateTick([]*enginev1.Unit{a, b}, defs)
	if len(kills) != 0 {
		t.Errorf("friendly fire: expected 0 kills, got %d", len(kills))
	}
}

func TestAdjudicate_OutOfRange_NoKill(t *testing.T) {
	// Units 200 km apart, combat range 50 km — no engagement.
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 1.8) // ~200 km at equator
	defs := map[string]DefStats{"def": makeDef(50_000, 0, 1)}

	kills := AdjudicateTick([]*enginev1.Unit{a, b}, defs)
	if len(kills) != 0 {
		t.Errorf("out-of-range: expected 0 kills, got %d", len(kills))
	}
}

func TestAdjudicate_LongerRangeWins(t *testing.T) {
	// a (Blue, 100 km range) vs b (Red, 10 km range), both within 100 km.
	// a should destroy b.
	a := makeUnit("a", "Blue", "long_range", 0, 0)
	b := makeUnit("b", "Red", "short_range", 0, 0.1) // ~11 km
	defs := map[string]DefStats{
		"long_range":  makeDef(100_000, 0, 1),
		"short_range": makeDef(10_000, 0, 1),
	}

	kills := AdjudicateTick([]*enginev1.Unit{a, b}, defs)
	if len(kills) != 1 {
		t.Fatalf("expected 1 kill, got %d", len(kills))
	}
	if kills[0].Victim.Id != "b" {
		t.Errorf("expected b to be killed, got %s", kills[0].Victim.Id)
	}
	if kills[0].Attacker.Id != "a" {
		t.Errorf("expected a to be attacker, got %s", kills[0].Attacker.Id)
	}
	if unitIsActive(b) {
		t.Error("b should be marked inactive after being killed")
	}
}

func TestAdjudicate_ShorterRangeLoses(t *testing.T) {
	// Reverse: Red has longer range.
	a := makeUnit("a", "Blue", "short_range", 0, 0)
	b := makeUnit("b", "Red", "long_range", 0, 0.1)
	defs := map[string]DefStats{
		"short_range": makeDef(10_000, 0, 1),
		"long_range":  makeDef(100_000, 0, 1),
	}

	kills := AdjudicateTick([]*enginev1.Unit{a, b}, defs)
	if len(kills) != 1 {
		t.Fatalf("expected 1 kill, got %d", len(kills))
	}
	if kills[0].Victim.Id != "a" {
		t.Errorf("expected a to be killed, got %s", kills[0].Victim.Id)
	}
}

func TestAdjudicate_EqualRange_HigherStrengthWins(t *testing.T) {
	a := makeUnit("a", "Blue", "strong", 0, 0)
	b := makeUnit("b", "Red", "weak", 0, 0.01) // close range
	defs := map[string]DefStats{
		"strong": makeDef(50_000, 0, 10),
		"weak":   makeDef(50_000, 0, 1),
	}

	kills := AdjudicateTick([]*enginev1.Unit{a, b}, defs)
	if len(kills) != 1 {
		t.Fatalf("expected 1 kill, got %d", len(kills))
	}
	if kills[0].Victim.Id != "b" {
		t.Errorf("expected b (weak) to be killed, got %s", kills[0].Victim.Id)
	}
}

func TestAdjudicate_EqualRange_EqualStrength_MutualAnnihilation(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 0.01)
	defs := map[string]DefStats{"def": makeDef(50_000, 0, 5)}

	kills := AdjudicateTick([]*enginev1.Unit{a, b}, defs)
	if len(kills) != 2 {
		t.Fatalf("expected 2 kills (mutual annihilation), got %d", len(kills))
	}
	for _, k := range kills {
		if k.Attacker != nil {
			t.Error("mutual annihilation kills should have nil Attacker")
		}
	}
	if unitIsActive(a) || unitIsActive(b) {
		t.Error("both units should be inactive after mutual annihilation")
	}
}

func TestAdjudicate_ZeroCombatRange_NoEngagement(t *testing.T) {
	a := makeUnit("a", "Blue", "def", 0, 0)
	b := makeUnit("b", "Red", "def", 0, 0)
	defs := map[string]DefStats{"def": makeDef(0, 0, 1)}

	kills := AdjudicateTick([]*enginev1.Unit{a, b}, defs)
	if len(kills) != 0 {
		t.Errorf("zero combat range: expected no kills, got %d", len(kills))
	}
}

func TestAdjudicate_DestroyedUnitSkipped(t *testing.T) {
	// a is already destroyed; it should not engage b.
	a := makeUnit("a", "Blue", "def", 0, 0)
	killUnit(a)
	b := makeUnit("b", "Red", "def", 0, 0)
	defs := map[string]DefStats{"def": makeDef(50_000, 0, 1)}

	kills := AdjudicateTick([]*enginev1.Unit{a, b}, defs)
	if len(kills) != 0 {
		t.Errorf("destroyed unit should not engage; got %d kills", len(kills))
	}
}

func TestAdjudicate_MultipleEnemies_ChainKills(t *testing.T) {
	// Blue has massive range; three Red units all close — all three should die.
	blue := makeUnit("blue", "Blue", "long", 0, 0)
	r1 := makeUnit("r1", "Red", "short", 0, 0.01)
	r2 := makeUnit("r2", "Red", "short", 0, 0.02)
	r3 := makeUnit("r3", "Red", "short", 0, 0.03)
	defs := map[string]DefStats{
		"long":  makeDef(500_000, 0, 1),
		"short": makeDef(1_000, 0, 1),
	}

	kills := AdjudicateTick([]*enginev1.Unit{blue, r1, r2, r3}, defs)
	if len(kills) != 3 {
		t.Errorf("expected 3 kills, got %d", len(kills))
	}
	for _, k := range kills {
		if k.Victim.Side != "Red" {
			t.Errorf("expected Red victims, got Side=%s", k.Victim.Side)
		}
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
		"sensor":    makeDef(0, 10_000, 0),
		"no_sensor": makeDef(0, 0, 0),
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
		"sensor":    makeDef(0, 10_000, 0),
		"no_sensor": makeDef(0, 0, 0),
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
	// Two Blue units next to each other — sensor should not report friendlies.
	b1 := makeUnit("b1", "Blue", "sensor", 0, 0)
	b2 := makeUnit("b2", "Blue", "sensor", 0, 0)
	defs := map[string]DefStats{"sensor": makeDef(0, 100_000, 0)}

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
		"sensor":    makeDef(0, 100_000, 0),
		"no_sensor": makeDef(0, 0, 0),
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
		"sensor":    makeDef(0, 1_000, 0),
		"no_sensor": makeDef(0, 0, 0),
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
	// Both units have sensor ranges that cover each other.
	blue := makeUnit("blue", "Blue", "sensor", 0, 0)
	red := makeUnit("red", "Red", "sensor", 0, 0.05) // ~5.5 km
	defs := map[string]DefStats{"sensor": makeDef(0, 10_000, 0)}

	result := SensorTick([]*enginev1.Unit{blue, red}, defs)

	blueDetects := result["Blue"]
	redDetects := result["Red"]

	foundRed := false
	for _, id := range blueDetects {
		if id == "red" {
			foundRed = true
		}
	}
	foundBlue := false
	for _, id := range redDetects {
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
