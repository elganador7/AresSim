// Package sim provides a mock simulation loop for development.
// Drives unit positions from a loaded scenario until the real adjudicator
// is implemented in Phase 2.
package sim

import (
	"context"
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"google.golang.org/protobuf/proto"
)

// EmitFn is the function signature for emitting a proto event to the frontend.
type EmitFn func(eventName string, msg proto.Message)

// MockLoop runs the mock simulation in the calling goroutine.
// It emits BatchUnitUpdate events every second, slowly advancing each unit
// along its initial heading. Returns when ctx is cancelled.
func MockLoop(ctx context.Context, units []*enginev1.Unit, emit EmitFn) {
	// Deep-copy positions so we can mutate them without touching the originals.
	type shipState struct {
		id      string
		lat     float64
		lon     float64
		heading float64
		speed   float64 // m/s
	}

	ships := make([]shipState, len(units))
	for i, u := range units {
		pos := u.GetPosition()
		ships[i] = shipState{
			id:      u.Id,
			lat:     pos.GetLat(),
			lon:     pos.GetLon(),
			heading: pos.GetHeading(),
			speed:   pos.GetSpeed(),
		}
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	tick := int64(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick++

			deltas := make([]*enginev1.UnitDelta, len(ships))
			for i := range ships {
				// Advance position along heading. 1 degree lat ≈ 111 km.
				// Convert m/s → degrees/s for a rough dead-reckoning step.
				headingRad := ships[i].heading * (3.14159265358979 / 180.0)
				dLat := ships[i].speed * cosApprox(headingRad) / 111_000.0
				dLon := ships[i].speed * sinApprox(headingRad) / (111_000.0 * cosApprox(ships[i].lat*(3.14159265358979/180.0)))

				ships[i].lat += dLat
				ships[i].lon += dLon

				deltas[i] = &enginev1.UnitDelta{
					UnitId: ships[i].id,
					Position: &enginev1.Position{
						Lat:     ships[i].lat,
						Lon:     ships[i].lon,
						Heading: ships[i].heading,
						Speed:   ships[i].speed,
					},
				}
			}

			emit("batch_update", &enginev1.BatchUnitUpdate{
				Deltas:  deltas,
				SimTime: &enginev1.SimTime{SecondsElapsed: float64(tick), TickNumber: tick},
			})
		}
	}
}

// sinApprox and cosApprox avoid importing math to keep the package lean.
// Replace with math.Sin/Cos if more precision is ever needed here.
func sinApprox(r float64) float64 {
	// Taylor series good enough for our tiny angular steps.
	r = r - float64(int(r/(2*3.14159265358979)))*2*3.14159265358979
	return r - r*r*r/6 + r*r*r*r*r/120
}

func cosApprox(r float64) float64 {
	return sinApprox(r + 3.14159265358979/2)
}
