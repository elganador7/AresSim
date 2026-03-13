package sim

import (
	"context"
	"math"
	"testing"
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"google.golang.org/protobuf/proto"
)

// ─── Geo math ─────────────────────────────────────────────────────────────────

func TestHaversineM_SamePoint(t *testing.T) {
	d := haversineM(40.0, -74.0, 40.0, -74.0)
	if d != 0.0 {
		t.Errorf("same point: expected 0, got %f", d)
	}
}

func TestHaversineM_KnownDistance(t *testing.T) {
	// New York (40.7128°N, 74.0060°W) to London (51.5074°N, 0.1278°W)
	// Roughly 5,570 km — accept ±10 km tolerance.
	d := haversineM(40.7128, -74.0060, 51.5074, -0.1278)
	expected := 5_570_000.0
	tolerance := 10_000.0
	if math.Abs(d-expected) > tolerance {
		t.Errorf("NYC→London: expected ~%.0f m, got %.0f m", expected, d)
	}
}

func TestHaversineM_Symmetry(t *testing.T) {
	d1 := haversineM(0, 0, 1, 1)
	d2 := haversineM(1, 1, 0, 0)
	if math.Abs(d1-d2) > 0.001 {
		t.Errorf("haversineM should be symmetric: %.6f vs %.6f", d1, d2)
	}
}

func TestBearingDeg_North(t *testing.T) {
	// Bearing from (0,0) to (1,0) should be approximately 0° (north).
	b := BearingDeg(0, 0, 1, 0)
	if math.Abs(b) > 0.1 && math.Abs(b-360) > 0.1 {
		t.Errorf("north bearing: expected ~0°, got %f°", b)
	}
}

func TestBearingDeg_East(t *testing.T) {
	// Bearing from (0,0) to (0,1) should be approximately 90° (east).
	b := BearingDeg(0, 0, 0, 1)
	if math.Abs(b-90) > 0.1 {
		t.Errorf("east bearing: expected ~90°, got %f°", b)
	}
}

func TestBearingDeg_South(t *testing.T) {
	b := BearingDeg(0, 0, -1, 0)
	if math.Abs(b-180) > 0.1 {
		t.Errorf("south bearing: expected ~180°, got %f°", b)
	}
}

func TestBearingDeg_West(t *testing.T) {
	b := BearingDeg(0, 0, 0, -1)
	if math.Abs(b-270) > 0.1 {
		t.Errorf("west bearing: expected ~270°, got %f°", b)
	}
}

func TestBearingDeg_Range(t *testing.T) {
	// Bearing should always be in [0, 360).
	tests := [][4]float64{
		{0, 0, 1, 1},
		{0, 0, -1, 1},
		{0, 0, -1, -1},
		{0, 0, 1, -1},
		{45, 90, -45, -90},
	}
	for _, tt := range tests {
		b := BearingDeg(tt[0], tt[1], tt[2], tt[3])
		if b < 0 || b >= 360 {
			t.Errorf("bearing out of range [0,360): %f for (%v,%v)→(%v,%v)", b, tt[0], tt[1], tt[2], tt[3])
		}
	}
}

func TestMovePoint_MovesCorrectDistance(t *testing.T) {
	// Move 1000 m north from origin; resulting latitude should be ~0.009° north.
	brng := bearingRad(0, 0, 1, 0) // north
	newLat, newLon := movePoint(0, 0, brng, 1000)

	// After moving 1000 m north, we should be about 0.009° north, 0° east.
	if math.Abs(newLon) > 0.0001 {
		t.Errorf("moving north should not change longitude; got %f", newLon)
	}
	if newLat <= 0 {
		t.Errorf("moving north should increase latitude; got %f", newLat)
	}
	// 1 degree latitude ≈ 111,000 m → 1000 m ≈ 0.009°
	expectedLat := 1000.0 / 111_000.0
	if math.Abs(newLat-expectedLat) > 0.0002 {
		t.Errorf("new latitude: expected ~%.5f, got %.5f", expectedLat, newLat)
	}
}

func TestMovePoint_RoundTrip(t *testing.T) {
	// Move 50 km northeast then back southwest — should approximately return to origin.
	lat0, lon0 := 40.0, -74.0
	dist := 50_000.0

	brngNE := bearingRad(0, 0, 1, 1)
	lat1, lon1 := movePoint(lat0, lon0, brngNE, dist)

	brngSW := bearingRad(lat1, lon1, lat0, lon0)
	lat2, lon2 := movePoint(lat1, lon1, brngSW, dist)

	if math.Abs(lat2-lat0) > 0.001 {
		t.Errorf("round-trip latitude: expected ~%.4f, got %.4f", lat0, lat2)
	}
	if math.Abs(lon2-lon0) > 0.001 {
		t.Errorf("round-trip longitude: expected ~%.4f, got %.4f", lon0, lon2)
	}
}

// ─── processTick ──────────────────────────────────────────────────────────────

func makeMovingUnit(id, side, defID string, lat, lon, destLat, destLon float64) *enginev1.Unit {
	return &enginev1.Unit{
		Id:           id,
		Side:         side,
		DefinitionId: defID,
		Position:     &enginev1.Position{Lat: lat, Lon: lon},
		Status:       &enginev1.OperationalStatus{IsActive: true},
		MoveOrder: &enginev1.MoveOrder{
			Waypoints: []*enginev1.Waypoint{{Lat: destLat, Lon: destLon}},
		},
	}
}

func TestProcessTick_UnitMovesCloser(t *testing.T) {
	u := makeMovingUnit("u1", "Blue", "fast", 0, 0, 1, 0)
	defs := map[string]DefStats{"fast": {CruiseSpeedMps: 100}}

	before := haversineM(0, 0, 1, 0)
	processTick([]*enginev1.Unit{u}, defs, 1.0)
	after := haversineM(u.Position.Lat, u.Position.Lon, 1, 0)

	if after >= before {
		t.Errorf("unit should be closer to destination after tick; before=%.0f after=%.0f", before, after)
	}
}

func TestProcessTick_UnitSnapsToWaypoint(t *testing.T) {
	// Unit is 10 m away; cruiseSpeed = 100 m/s → should snap to waypoint.
	u := makeMovingUnit("u1", "Blue", "fast", 0, 0, 0, 0.0001)
	defs := map[string]DefStats{"fast": {CruiseSpeedMps: 100}}

	deltas := processTick([]*enginev1.Unit{u}, defs, 1.0)

	if len(deltas) == 0 {
		t.Fatal("expected a delta for the moving unit")
	}
	// Waypoints should be cleared after arriving at final waypoint.
	if u.MoveOrder != nil && len(u.MoveOrder.Waypoints) > 0 {
		t.Errorf("waypoints should be cleared after reaching final destination")
	}
	if u.Position.Speed != 0 {
		t.Errorf("speed should be 0 after reaching destination, got %f", u.Position.Speed)
	}
}

func TestProcessTick_StationaryUnit_NoMoveOrder(t *testing.T) {
	u := makeUnit("u1", "Blue", "def", 0, 0)
	defs := map[string]DefStats{"def": {CruiseSpeedMps: 10}}

	deltas := processTick([]*enginev1.Unit{u}, defs, 1.0)
	if len(deltas) != 0 {
		t.Errorf("stationary unit should produce no deltas; got %d", len(deltas))
	}
}

func TestProcessTick_DestroyedUnit_Skipped(t *testing.T) {
	u := makeMovingUnit("u1", "Blue", "fast", 0, 0, 1, 0)
	killUnit(u)
	defs := map[string]DefStats{"fast": {CruiseSpeedMps: 100}}

	deltas := processTick([]*enginev1.Unit{u}, defs, 1.0)
	if len(deltas) != 0 {
		t.Errorf("destroyed unit should produce no deltas; got %d", len(deltas))
	}
}

func TestProcessTick_TimeScale_AffectsMovement(t *testing.T) {
	dest := [2]float64{10, 0} // far away — unit won't arrive in one tick

	u1 := makeMovingUnit("u1", "Blue", "def", 0, 0, dest[0], dest[1])
	u2 := makeMovingUnit("u2", "Blue", "def", 0, 0, dest[0], dest[1])
	defs := map[string]DefStats{"def": {CruiseSpeedMps: 100}}

	processTick([]*enginev1.Unit{u1}, defs, 1.0)
	processTick([]*enginev1.Unit{u2}, defs, 10.0)

	dist1 := haversineM(u1.Position.Lat, u1.Position.Lon, dest[0], dest[1])
	dist2 := haversineM(u2.Position.Lat, u2.Position.Lon, dest[0], dest[1])

	if dist2 >= dist1 {
		t.Errorf("10× time scale should move unit farther; dist1=%.0f dist2=%.0f", dist1, dist2)
	}

	// Distance at 10× should be ~10× the distance covered at 1×.
	covered1 := haversineM(0, 0, dest[0], dest[1]) - dist1
	covered2 := haversineM(0, 0, dest[0], dest[1]) - dist2
	ratio := covered2 / covered1
	if math.Abs(ratio-10.0) > 0.5 {
		t.Errorf("expected ~10× more distance at 10× speed; ratio=%.2f", ratio)
	}
}

func TestProcessTick_FallbackSpeed_UsedWhenDefMissing(t *testing.T) {
	u := makeMovingUnit("u1", "Blue", "unknown_def", 0, 0, 1, 0)
	// empty defs — CruiseSpeedMps will be 0, triggering the 10 m/s fallback
	defs := map[string]DefStats{}

	before := haversineM(0, 0, 1, 0)
	processTick([]*enginev1.Unit{u}, defs, 1.0)
	after := haversineM(u.Position.Lat, u.Position.Lon, 1, 0)

	if after >= before {
		t.Error("unit should still move using fallback speed when def is missing")
	}
}

func TestProcessTick_MultipleWaypoints_AdvancesThrough(t *testing.T) {
	// Speed fast enough to reach first waypoint in one tick.
	u := &enginev1.Unit{
		Id:           "u1",
		Side:         "Blue",
		DefinitionId: "fast",
		Position:     &enginev1.Position{Lat: 0, Lon: 0},
		Status:       &enginev1.OperationalStatus{IsActive: true},
		MoveOrder: &enginev1.MoveOrder{
			Waypoints: []*enginev1.Waypoint{
				{Lat: 0, Lon: 0.0001}, // ~11 m — will snap
				{Lat: 0, Lon: 1},      // far, remaining waypoint
			},
		},
	}
	defs := map[string]DefStats{"fast": {CruiseSpeedMps: 100}}

	processTick([]*enginev1.Unit{u}, defs, 1.0)

	// After tick: first waypoint consumed, second should remain.
	if u.MoveOrder == nil || len(u.MoveOrder.Waypoints) == 0 {
		t.Error("second waypoint should still remain after passing first")
	}
}

// ─── MockLoop integration ─────────────────────────────────────────────────────

func TestMockLoop_ExitsOnContextCancel(t *testing.T) {
	u := makeUnit("u1", "Blue", "def", 0, 0)
	defs := map[string]DefStats{"def": {CruiseSpeedMps: 0}}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		MockLoop(ctx, []*enginev1.Unit{u}, defs, 0, func() float64 { return 1.0 },
			func(_ float64) {}, func(_ string, _ proto.Message) {})
		close(done)
	}()

	// Cancel immediately — loop should exit within a short window.
	cancel()
	select {
	case <-done:
		// Good.
	case <-time.After(2 * time.Second):
		t.Error("MockLoop did not exit after context cancellation")
	}
}

func TestMockLoop_EmitsBatchUpdate(t *testing.T) {
	u := makeUnit("u1", "Blue", "def", 0, 0)
	defs := map[string]DefStats{"def": {CruiseSpeedMps: 0}}

	emitted := make(chan string, 64)
	var emitFn EmitFn = func(name string, _ proto.Message) {
		select {
		case emitted <- name:
		default:
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	go MockLoop(ctx, []*enginev1.Unit{u}, defs, 0, func() float64 { return 1.0 }, func(_ float64) {}, emitFn)

	<-ctx.Done()

	// Should have at least one batch_update within 1.5 s (ticker fires at 1 s).
	gotBatch := false
	for {
		select {
		case name := <-emitted:
			if name == "batch_update" {
				gotBatch = true
			}
		default:
			if !gotBatch {
				t.Error("expected at least one batch_update event within 1.5 s")
			}
			return
		}
	}
}
