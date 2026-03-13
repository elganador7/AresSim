// Package sim provides the movement engine and adjudicator.
// Drives unit positions based on their MoveOrder waypoints each tick,
// then resolves combat between opposing units within engagement range.
package sim

import (
	"context"
	"fmt"
	"math"
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"google.golang.org/protobuf/proto"
)

// EmitFn is the function signature for emitting a proto event to the frontend.
type EmitFn func(eventName string, msg proto.Message)

const (
	tickInterval = time.Second  // one tick per second
	earthRadiusM = 6_371_000.0
	// Snap-to-waypoint is handled by canMoveM >= distM (unit covers the
	// remaining distance in one tick). arrivalRadius was removed because the
	// movement logic is already correct without it.
)

// MockLoop runs the movement and adjudication engine.
// defs maps definitionId → DefStats (speed, range, strength).
// startSeconds is the accumulated sim time to resume from (pass 0 for a fresh
// scenario; pass the value from the previous run when resuming after pause).
// getScale is called each tick to read the current time-scale multiplier;
// at 2× the sim advances 2 seconds of game-time per real second.
// reportSeconds is called after each tick with the new accumulated sim time so
// the caller can persist it across pause/resume cycles.
// Returns when ctx is cancelled.
func MockLoop(ctx context.Context, units []*enginev1.Unit, defs map[string]DefStats, startSeconds float64, getScale func() float64, reportSeconds func(float64), emit EmitFn) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	tick       := int64(0)
	simSeconds := startSeconds

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick++
			timeScale := getScale()
			simSeconds += timeScale
			reportSeconds(simSeconds)

			// ── Movement ──────────────────────────────────────────────────
			deltas := processTick(units, defs, timeScale)
			emit("batch_update", &enginev1.BatchUnitUpdate{
				Deltas:  deltas,
				SimTime: &enginev1.SimTime{SecondsElapsed: simSeconds, TickNumber: tick},
			})

			// ── Sensor detection ──────────────────────────────────────────
			detections := SensorTick(units, defs)
			for side, ids := range detections {
				emit("detection_update", &enginev1.DetectionUpdate{
					DetectingSide:   side,
					DetectedUnitIds: ids,
				})
			}

			// ── Adjudication ──────────────────────────────────────────────
			kills := AdjudicateTick(units, defs)
			for _, k := range kills {
				attackerID := ""
				if k.Attacker != nil {
					attackerID = k.Attacker.Id
				}
				emit("unit_destroyed", &enginev1.UnitDestroyedEvent{
					UnitId:     k.Victim.Id,
					Cause:      "combat",
					AttackerId: attackerID,
					SimTime:    &enginev1.SimTime{SecondsElapsed: simSeconds, TickNumber: tick},
				})

				var narrative string
				if k.Attacker != nil {
					narrative = fmt.Sprintf("%s destroyed %s", k.Attacker.DisplayName, k.Victim.DisplayName)
				} else {
					narrative = fmt.Sprintf("%s was destroyed in a mutual engagement", k.Victim.DisplayName)
				}
				side := k.Victim.Side
				unitID := k.Victim.Id
				if k.Attacker != nil {
					side = k.Attacker.Side
					unitID = k.Attacker.Id
				}
				emit("narrative", &enginev1.NarrativeEvent{
					Text:     narrative,
					Category: "combat",
					UnitId:   unitID,
					Side:     side,
				})
			}
		}
	}
}

// processTick advances all active units with move orders by one tick and
// returns a UnitDelta for every unit that changed position or order state.
// timeScale multiplies how many sim-seconds of movement occur per real second.
func processTick(units []*enginev1.Unit, defs map[string]DefStats, timeScale float64) []*enginev1.UnitDelta {
	deltas := make([]*enginev1.UnitDelta, 0, len(units))

	for _, u := range units {
		if !unitIsActive(u) {
			continue // skip destroyed units
		}
		order := u.GetMoveOrder()
		if order == nil || len(order.Waypoints) == 0 {
			continue // unit is stationary
		}

		pos := u.GetPosition()
		wp := order.Waypoints[0]

		cruiseSpeed := defs[u.DefinitionId].CruiseSpeedMps
		if cruiseSpeed <= 0 {
			cruiseSpeed = 10 // m/s fallback
		}

		distM := haversineM(pos.Lat, pos.Lon, wp.Lat, wp.Lon)
		canMoveM := cruiseSpeed * tickInterval.Seconds() * timeScale

		var newLat, newLon float64
		var remainingWaypoints []*enginev1.Waypoint

		if canMoveM >= distM {
			// Snap to waypoint and advance to the next one.
			newLat = wp.Lat
			newLon = wp.Lon
			remainingWaypoints = order.Waypoints[1:]
		} else {
			// Move toward waypoint by canMoveM metres.
			brng := bearingRad(pos.Lat, pos.Lon, wp.Lat, wp.Lon)
			newLat, newLon = movePoint(pos.Lat, pos.Lon, brng, canMoveM)
			remainingWaypoints = order.Waypoints
		}

		// Compute new heading (toward current or next target).
		var newHeading float64
		if len(remainingWaypoints) > 0 {
			target := remainingWaypoints[0]
			newHeading = BearingDeg(newLat, newLon, target.Lat, target.Lon)
		} else {
			newHeading = pos.Heading // keep heading at final destination
		}

		// Speed: cruising while moving, stopped when done.
		var newSpeed float64
		var newOrder *enginev1.MoveOrder
		if len(remainingWaypoints) > 0 {
			newSpeed = cruiseSpeed
			newOrder = &enginev1.MoveOrder{Waypoints: remainingWaypoints}
		} else {
			newSpeed = 0
			newOrder = &enginev1.MoveOrder{} // empty = cleared on frontend
		}

		// Mutate in-memory unit state so RequestSync stays accurate.
		u.Position = &enginev1.Position{
			Lat:     newLat,
			Lon:     newLon,
			AltMsl:  pos.AltMsl,
			Heading: newHeading,
			Speed:   newSpeed,
		}
		u.MoveOrder = newOrder

		deltas = append(deltas, &enginev1.UnitDelta{
			UnitId:    u.Id,
			Position:  u.Position,
			MoveOrder: newOrder,
		})
	}

	return deltas
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
