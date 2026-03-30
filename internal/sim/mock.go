// Package sim provides the movement engine and adjudicator.
// Drives unit positions based on their MoveOrder waypoints each tick,
// then resolves combat between opposing units within engagement range.
package sim

import (
	"context"
	"math"
	"math/rand"
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"google.golang.org/protobuf/proto"
)

// EmitFn is the function signature for emitting a proto event to the frontend.
type EmitFn func(eventName string, msg proto.Message)

const (
	tickInterval = time.Second // one tick per second
	earthRadiusM = 6_371_000.0
	// Snap-to-waypoint is handled by canMoveM >= distM (unit covers the
	// remaining distance in one tick). arrivalRadius was removed because the
	// movement logic is already correct without it.
)

// MockLoop runs the movement and adjudication engine.
// defs maps definitionId → DefStats (speed, range, strength, domain).
// weapons maps weaponId → WeaponStats (range, probability, domain targets).
// startSeconds is the accumulated sim time to resume from (pass 0 for a fresh
// scenario; pass the value from the previous run when resuming after pause).
// getScale is called each tick to read the current time-scale multiplier;
// at 2× the sim advances 2 seconds of game-time per real second.
// reportSeconds is called after each tick with the new accumulated sim time so
// the caller can persist it across pause/resume cycles.
// Returns when ctx is cancelled.
func MockLoop(ctx context.Context, units []*enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats, relationshipRules func() RelationshipRules, startSeconds float64, getScale func() float64, reportSeconds func(float64), planTick func(float64) []*enginev1.UnitDelta, emit EmitFn) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	tick := int64(0)
	simSeconds := startSeconds
	var inFlight []*InFlightMunition
	var previousTracks detectionIndex

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick++
			timeScale := getScale()
			simSeconds += timeScale
			reportSeconds(simSeconds)

			applyPlanTick(planTick, emit, simSeconds, tick)
			emitBatchUpdate(emit, processReplenishmentTick(units, defs, simSeconds), simSeconds, tick)
			emitBatchUpdate(emit, processBehaviorTick(units, defs, weapons, simSeconds), simSeconds, tick)
			emitBatchUpdateAlways(emit, processTick(units, defs, timeScale), simSeconds, tick)

			rules := relationshipRules()

			basePicture := buildTrackPicture(units, defs, rules, previousTracks, rng)
			previousTracks = basePicture.ByDetector
			baseDetections := basePicture.BySide
			detections := ApplyIntelSharing(baseDetections, rules)
			detectionContacts := BuildDetectionContacts(baseDetections, rules)

			adj := AdjudicateTick(units, defs, weapons, inFlight, rules, simSeconds)
			inFlight = append(inFlight, emitShooterWeaponUpdatesAndSpawnMunitions(adj.Shots, weapons, emit)...)
			munitionDetections := map[string][]string(nil)
			inFlight, munitionDetections = processInFlightIntercepts(units, defs, weapons, inFlight, rules, rng, emit)

			var arrived []*InFlightMunition
			inFlight, arrived = AdvanceMunitions(inFlight, timeScale, units, defs)
			hits := ResolveArrivals(arrived, units, defs, weapons, rng)
			emitDetectionUpdates(emit, detections, munitionDetections, detectionContacts)
			emitMunitionUpdate(emit, inFlight, simSeconds, tick)
			emitArrivalResults(emit, hits, simSeconds, tick)
		}
	}
}

// ─── GEO MATH ─────────────────────────────────────────────────────────────────

// haversineM returns the great-circle distance in metres between two lat/lon points.
func haversineM(lat1, lon1, lat2, lon2 float64) float64 {
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// bearingRad returns the initial bearing in radians from (lat1,lon1) to (lat2,lon2).
func bearingRad(lat1, lon1, lat2, lon2 float64) float64 {
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δλ := (lon2 - lon1) * math.Pi / 180
	y := math.Sin(Δλ) * math.Cos(φ2)
	x := math.Cos(φ1)*math.Sin(φ2) - math.Sin(φ1)*math.Cos(φ2)*math.Cos(Δλ)
	return math.Atan2(y, x)
}

// BearingDeg returns the initial bearing in degrees (0–360 true north) from point A to B.
// Exported so app.go can compute initial headings when issuing move orders.
func BearingDeg(lat1, lon1, lat2, lon2 float64) float64 {
	b := bearingRad(lat1, lon1, lat2, lon2) * 180 / math.Pi
	return math.Mod(b+360, 360)
}

// movePoint advances a position by distM metres along bearing brng (radians).
func movePoint(lat, lon, brngRad, distM float64) (newLat, newLon float64) {
	δ := distM / earthRadiusM
	φ1 := lat * math.Pi / 180
	λ1 := lon * math.Pi / 180
	φ2 := math.Asin(math.Sin(φ1)*math.Cos(δ) + math.Cos(φ1)*math.Sin(δ)*math.Cos(brngRad))
	λ2 := λ1 + math.Atan2(math.Sin(brngRad)*math.Sin(δ)*math.Cos(φ1),
		math.Cos(δ)-math.Sin(φ1)*math.Sin(φ2))
	return φ2 * 180 / math.Pi, λ2 * 180 / math.Pi
}
