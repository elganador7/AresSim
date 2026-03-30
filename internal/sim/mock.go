// Package sim provides the movement engine and adjudicator.
// Drives unit positions based on their MoveOrder waypoints each tick,
// then resolves combat between opposing units within engagement range.
package sim

import (
	"context"
	"fmt"
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

			if planTick != nil {
				planDeltas := planTick(simSeconds)
				if len(planDeltas) > 0 {
					emit("batch_update", &enginev1.BatchUnitUpdate{
						Deltas:  planDeltas,
						SimTime: &enginev1.SimTime{SecondsElapsed: simSeconds, TickNumber: tick},
					})
				}
			}

			replenishmentDeltas := processReplenishmentTick(units, defs, simSeconds)
			if len(replenishmentDeltas) > 0 {
				emit("batch_update", &enginev1.BatchUnitUpdate{
					Deltas:  replenishmentDeltas,
					SimTime: &enginev1.SimTime{SecondsElapsed: simSeconds, TickNumber: tick},
				})
			}

			// ── Behavior reactions ────────────────────────────────────────
			reactionDeltas := processBehaviorTick(units, defs, weapons, simSeconds)
			if len(reactionDeltas) > 0 {
				emit("batch_update", &enginev1.BatchUnitUpdate{
					Deltas:  reactionDeltas,
					SimTime: &enginev1.SimTime{SecondsElapsed: simSeconds, TickNumber: tick},
				})
			}

			// ── Movement ──────────────────────────────────────────────────
			deltas := processTick(units, defs, timeScale)
			emit("batch_update", &enginev1.BatchUnitUpdate{
				Deltas:  deltas,
				SimTime: &enginev1.SimTime{SecondsElapsed: simSeconds, TickNumber: tick},
			})

			rules := relationshipRules()

			// ── Detection / adjudication ─────────────────────────────────
			basePicture := buildTrackPicture(units, defs, rules, previousTracks, rng)
			previousTracks = basePicture.ByDetector
			baseDetections := basePicture.BySide
			detections := ApplyIntelSharing(baseDetections, rules)
			detectionContacts := BuildDetectionContacts(baseDetections, rules)

			adj := AdjudicateTick(units, defs, weapons, inFlight, rules, simSeconds)
			// Emit a weapon-state delta for every unit that fired this tick
			// so the frontend can update ammo counters in real time.
			fired := make(map[string]bool)
			for _, shot := range adj.Shots {
				if !fired[shot.Shooter.Id] {
					fired[shot.Shooter.Id] = true
					states := make([]*enginev1.WeaponState, len(shot.Shooter.Weapons))
					for i, ws := range shot.Shooter.Weapons {
						states[i] = &enginev1.WeaponState{
							WeaponId:   ws.WeaponId,
							CurrentQty: ws.CurrentQty,
							MaxQty:     ws.MaxQty,
						}
					}
					emit("batch_update", &enginev1.BatchUnitUpdate{
						Deltas: []*enginev1.UnitDelta{{
							UnitId:                 shot.Shooter.Id,
							Weapons:                states,
							NextStrikeReadySeconds: shot.Shooter.GetNextStrikeReadySeconds(),
						}},
					})
				}

				// Create one in-flight munition per round in the salvo.
				// Carry the target ID and pre-computed PoH so kill resolution
				// can be deferred until the munition arrives.
				ws, hasStats := weapons[shot.WeaponID]
				if hasStats && ws.SpeedMps > 0 {
					for range shot.SalvoSize {
						inFlight = append(inFlight, &InFlightMunition{
							ID:             NextMunitionID(),
							WeaponID:       shot.WeaponID,
							ShooterID:      shot.Shooter.Id,
							ShooterTeam:    unitTeamID(shot.Shooter),
							TargetID:       shot.Target.Id,
							HitProbability: shot.HitProbability,
							LaunchLat:      shot.Shooter.GetPosition().GetLat(),
							LaunchLon:      shot.Shooter.GetPosition().GetLon(),
							MaxRangeM:      ws.RangeM,
							CurLat:         shot.Shooter.GetPosition().GetLat(),
							CurLon:         shot.Shooter.GetPosition().GetLon(),
							CurAltMsl:      shot.Shooter.GetPosition().GetAltMsl(),
							DestLat:        shot.Target.GetPosition().GetLat(),
							DestLon:        shot.Target.GetPosition().GetLon(),
							DestAltMsl:     shot.Target.GetPosition().GetAltMsl(),
							SpeedMps:       ws.SpeedMps,
							TargetDomains:  ws.DomainTargets,
						})
					}
				}
			}

			// ── Move in-flight munitions ───────────────────────────────────
			munitionDetections := ApplyIntelSharing(DetectMunitions(units, defs, inFlight), rules)
			var interceptShots []InterceptShot
			inFlight, interceptShots = InterceptMunitionsTick(units, defs, weapons, inFlight, munitionDetections, rng)
			interceptorFired := make(map[string]bool)
			for _, shot := range interceptShots {
				if shot.Defender == nil {
					continue
				}
				if !interceptorFired[shot.Defender.Id] {
					interceptorFired[shot.Defender.Id] = true
					states := make([]*enginev1.WeaponState, len(shot.Defender.Weapons))
					for i, ws := range shot.Defender.Weapons {
						states[i] = &enginev1.WeaponState{
							WeaponId:   ws.WeaponId,
							CurrentQty: ws.CurrentQty,
							MaxQty:     ws.MaxQty,
						}
					}
					emit("batch_update", &enginev1.BatchUnitUpdate{
						Deltas: []*enginev1.UnitDelta{{
							UnitId:  shot.Defender.Id,
							Weapons: states,
						}},
					})
				}
				if !shot.Success {
					continue
				}
				targetID := shot.Munition.TargetID
				narrative := fmt.Sprintf("%s intercepted %s inbound to %s", shot.Defender.DisplayName, shot.Munition.WeaponID, targetID)
				if target := findUnitByID(units, targetID); target != nil {
					narrative = fmt.Sprintf("%s intercepted %s inbound to %s", shot.Defender.DisplayName, shot.Munition.WeaponID, target.DisplayName)
				}
				emit("narrative", &enginev1.NarrativeEvent{
					Text:     narrative,
					Category: "air_defense",
					UnitId:   shot.Defender.Id,
					TeamId:   unitTeamID(shot.Defender),
				})
			}

			var arrived []*InFlightMunition
			inFlight, arrived = AdvanceMunitions(inFlight, timeScale, units, defs)

			// ── Resolve kills for munitions that arrived this tick ─────────
			hits := ResolveArrivals(arrived, units, defs, weapons, rng)

			// Emit per-side detection updates, merging unit and munition contacts.
			activeSides := make(map[string]bool, len(detections))
			for side := range detections {
				activeSides[side] = true
			}
			for side := range activeSides {
				contacts := detectionContacts[side]
				protoContacts := make([]*enginev1.DetectionContact, 0, len(contacts))
				for _, contact := range contacts {
					protoContacts = append(protoContacts, &enginev1.DetectionContact{
						UnitId:     contact.UnitID,
						SourceTeam: contact.SourceTeam,
						Shared:     contact.Shared,
					})
				}
				emit("detection_update", &enginev1.DetectionUpdate{
					DetectingTeam:       side,
					DetectedUnitIds:     detections[side],
					DetectedMunitionIds: munitionDetections[side],
					UnitContacts:        protoContacts,
				})
			}

			// ── Emit full in-flight munition state ─────────────────────────
			munProtos := make([]*enginev1.InFlightMunition, len(inFlight))
			for i, m := range inFlight {
				munProtos[i] = &enginev1.InFlightMunition{
					Id:        m.ID,
					WeaponId:  m.WeaponID,
					ShooterId: m.ShooterID,
					Position:  &enginev1.Position{Lat: m.CurLat, Lon: m.CurLon, AltMsl: m.CurAltMsl},
				}
			}
			emit("munition_update", &enginev1.MunitionUpdate{
				Munitions: munProtos,
				SimTime:   &enginev1.SimTime{SecondsElapsed: simSeconds, TickNumber: tick},
			})

			// ── Emit damage/destruction results for arrived munitions ──────
			for _, hit := range hits {
				emit("batch_update", &enginev1.BatchUnitUpdate{
					Deltas: []*enginev1.UnitDelta{{
						UnitId:                 hit.Victim.Id,
						Status:                 hit.Victim.Status,
						DamageState:            hit.Victim.DamageState,
						BaseOps:                hit.Victim.GetBaseOps(),
						NextSortieReadySeconds: hit.Victim.GetNextSortieReadySeconds(),
					}},
				})

				attackerID := ""
				if hit.Attacker != nil {
					attackerID = hit.Attacker.Id
				}
				if hit.Destroyed {
					emit("unit_destroyed", &enginev1.UnitDestroyedEvent{
						UnitId:     hit.Victim.Id,
						Cause:      "combat",
						AttackerId: attackerID,
						SimTime:    &enginev1.SimTime{SecondsElapsed: simSeconds, TickNumber: tick},
					})
				}

				var narrative string
				if hit.Attacker != nil {
					narrative = fmt.Sprintf("%s %s %s", hit.Attacker.DisplayName, describeImpact(hit.Outcome), hit.Victim.DisplayName)
				} else {
					narrative = fmt.Sprintf("%s was %s in a mutual engagement", hit.Victim.DisplayName, describeImpact(hit.Outcome))
				}
				side := unitTeamID(hit.Victim)
				unitID := hit.Victim.Id
				if hit.Attacker != nil {
					side = unitTeamID(hit.Attacker)
					unitID = hit.Attacker.Id
				}
				emit("narrative", &enginev1.NarrativeEvent{
					Text:     narrative,
					Category: "combat",
					UnitId:   unitID,
					TeamId:   side,
				})
			}
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
