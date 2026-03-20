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

			// ── Behavior reactions ────────────────────────────────────────
			reactionDeltas := processBehaviorTick(units, defs, weapons)
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

			// ── Adjudication ──────────────────────────────────────────────
			adj := AdjudicateTick(units, defs, weapons, inFlight, rules, simSeconds)
			trackGroups := resolveTrackGroupIDs(units)

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
							TrackGroupID:   trackGroups[shot.Shooter.Id],
							TargetID:       shot.Target.Id,
							HitProbability: shot.HitProbability,
							CurLat:         shot.Shooter.GetPosition().GetLat(),
							CurLon:         shot.Shooter.GetPosition().GetLon(),
							CurAltMsl:      shot.Shooter.GetPosition().GetAltMsl(),
							DestLat:        shot.Target.GetPosition().GetLat(),
							DestLon:        shot.Target.GetPosition().GetLon(),
							DestAltMsl:     shot.Target.GetPosition().GetAltMsl(),
							SpeedMps:       ws.SpeedMps,
							TargetDomains:  ws.DomainTargets,
							Guidance:       ws.Guidance,
						})
					}
				}
			}

			// ── Sensor detection (unit-to-unit) ───────────────────────────
			// Run before AdvanceMunitions so radar-guided munitions can check
			// whether the shooter still holds a detection on their target.
			baseDetections := SensorTick(units, defs, rules)
			detections := ApplyIntelSharing(baseDetections, rules)
			detectionContacts := BuildDetectionContacts(baseDetections, rules)

			// ── Move in-flight munitions ───────────────────────────────────
			var arrived []*InFlightMunition
			inFlight, arrived = AdvanceMunitions(inFlight, timeScale, units, defs)

			// ── Munition detection / interception (post-advance positions) ─
			munitionDets := DetectMunitions(units, defs, inFlight)
			inFlight, interceptShots := InterceptMunitionsTick(units, defs, weapons, inFlight, munitionDets, rng)
			munitionDets = DetectMunitions(units, defs, inFlight)

			for _, shot := range interceptShots {
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

			// ── Resolve kills for munitions that arrived this tick ─────────
			hits := ResolveArrivals(arrived, units, defs, weapons, rng)

			// Emit per-side detection updates, merging unit and munition contacts.
			activeSides := make(map[string]bool, len(detections)+len(munitionDets))
			for side := range detections {
				activeSides[side] = true
			}
			for side := range munitionDets {
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
					DetectedMunitionIds: munitionDets[side],
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

func processBehaviorTick(units []*enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats) []*enginev1.UnitDelta {
	tracks := buildTrackPicture(units, defs, nil)
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, u := range units {
		unitByID[u.Id] = u
	}

	deltas := make([]*enginev1.UnitDelta, 0)
	for _, u := range units {
		if !unitCanOperate(u) || !unitCanMove(u, defs) {
			continue
		}

		if hasExplicitAttackOrder(u) {
			target := unitByID[u.GetAttackOrder().GetTargetUnitId()]
			if target == nil || !unitIsAlive(target) || (u.GetAttackOrder().GetOrderType() == enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT &&
				desiredEffectSatisfied(target, u.GetAttackOrder().GetDesiredEffect())) {
				u.AttackOrder = nil
				var moveOrderDelta *enginev1.MoveOrder
				if current := u.GetMoveOrder(); current != nil && current.GetAutoGenerated() {
					u.MoveOrder = nil
					moveOrderDelta = &enginev1.MoveOrder{}
				}
				deltas = append(deltas, &enginev1.UnitDelta{
					UnitId:    u.Id,
					MoveOrder: moveOrderDelta,
				})
				continue
			}
			waypoint := computeAttackWaypoint(u, target, defs, weapons)
			if waypoint != nil {
				if attackRouteNeedsUpdate(u, waypoint) {
					order := &enginev1.MoveOrder{
						Waypoints:     []*enginev1.Waypoint{waypoint},
						AutoGenerated: true,
					}
					u.MoveOrder = order
					deltas = append(deltas, &enginev1.UnitDelta{
						UnitId:    u.Id,
						MoveOrder: order,
					})
				}
				continue
			}
		}

		if hasExplicitMoveOrder(u) || hasExplicitAttackOrder(u) {
			continue
		}

		target := nearestTrackedEnemy(u, tracks, unitByID)
		if target == nil {
			continue
		}

		var waypoint *enginev1.Waypoint
		switch u.GetEngagementBehavior() {
		case enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_SHADOW_CONTACT:
			waypoint = &enginev1.Waypoint{
				Lat:    target.GetPosition().GetLat(),
				Lon:    target.GetPosition().GetLon(),
				AltMsl: target.GetPosition().GetAltMsl(),
			}
		case enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_WITHDRAW_ON_DETECT:
			waypoint = computeWithdrawWaypoint(u, target, defs[u.DefinitionId])
		default:
			continue
		}
		if waypoint == nil {
			continue
		}

		order := &enginev1.MoveOrder{Waypoints: []*enginev1.Waypoint{waypoint}}
		u.MoveOrder = order
		deltas = append(deltas, &enginev1.UnitDelta{
			UnitId:    u.Id,
			MoveOrder: order,
		})
	}
	return deltas
}

func ComputeAttackWaypointForOrder(shooter, target *enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats) *enginev1.Waypoint {
	if shooter == nil || target == nil || shooter.GetPosition() == nil || target.GetPosition() == nil {
		return nil
	}
	targetDef, ok := defs[target.DefinitionId]
	if !ok {
		return nil
	}
	_, weapon, hasWeapon := selectBestWeapon(shooter, targetDef.Domain, weapons)
	if !hasWeapon || weapon.RangeM <= 0 {
		return nil
	}
	dist := haversineM(
		shooter.GetPosition().GetLat(), shooter.GetPosition().GetLon(),
		target.GetPosition().GetLat(), target.GetPosition().GetLon(),
	)
	desiredStandOff := weapon.RangeM * 0.85
	if desiredStandOff <= 0 {
		return nil
	}
	if dist <= desiredStandOff {
		return nil
	}
	moveDist := dist - desiredStandOff
	brng := bearingRad(
		shooter.GetPosition().GetLat(), shooter.GetPosition().GetLon(),
		target.GetPosition().GetLat(), target.GetPosition().GetLon(),
	)
	lat, lon := movePoint(shooter.GetPosition().GetLat(), shooter.GetPosition().GetLon(), brng, moveDist)
	return &enginev1.Waypoint{
		Lat:    lat,
		Lon:    lon,
		AltMsl: shooter.GetPosition().GetAltMsl(),
	}
}

func CanUnitAttackTarget(shooter, target *enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats) bool {
	if shooter == nil || target == nil || shooter.GetPosition() == nil || target.GetPosition() == nil {
		return false
	}
	targetDef, ok := defs[target.DefinitionId]
	if !ok {
		return false
	}
	_, weapon, hasWeapon := selectBestWeapon(shooter, targetDef.Domain, weapons)
	if !hasWeapon || weapon.RangeM <= 0 {
		return false
	}
	dist := haversineM(
		shooter.GetPosition().GetLat(), shooter.GetPosition().GetLon(),
		target.GetPosition().GetLat(), target.GetPosition().GetLon(),
	)
	if dist <= weapon.RangeM {
		return true
	}
	return ComputeAttackWaypointForOrder(shooter, target, defs, weapons) != nil
}

func computeAttackWaypoint(shooter, target *enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats) *enginev1.Waypoint {
	return ComputeAttackWaypointForOrder(shooter, target, defs, weapons)
}

func attackRouteNeedsUpdate(unit *enginev1.Unit, waypoint *enginev1.Waypoint) bool {
	order := unit.GetMoveOrder()
	if order == nil || len(order.GetWaypoints()) == 0 {
		return true
	}
	if !order.GetAutoGenerated() {
		return false
	}
	if len(order.GetWaypoints()) != 1 {
		return true
	}
	current := order.GetWaypoints()[0]
	return haversineM(current.GetLat(), current.GetLon(), waypoint.GetLat(), waypoint.GetLon()) > 5_000
}

// processTick advances all active units with move orders by one tick and
// returns a UnitDelta for every unit that changed position or order state.
// timeScale multiplies how many sim-seconds of movement occur per real second.
func processTick(units []*enginev1.Unit, defs map[string]DefStats, timeScale float64) []*enginev1.UnitDelta {
	deltas := make([]*enginev1.UnitDelta, 0, len(units))

	for _, u := range units {
		if !unitCanOperate(u) {
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
			newOrder = &enginev1.MoveOrder{
				Waypoints:     remainingWaypoints,
				AutoGenerated: order.GetAutoGenerated(),
			}
		} else {
			newSpeed = 0
			newOrder = &enginev1.MoveOrder{AutoGenerated: order.GetAutoGenerated()} // empty = cleared on frontend
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

func hasExplicitMoveOrder(u *enginev1.Unit) bool {
	order := u.GetMoveOrder()
	return order != nil && len(order.Waypoints) > 0
}

func hasExplicitAttackOrder(u *enginev1.Unit) bool {
	order := u.GetAttackOrder()
	return order != nil &&
		order.GetOrderType() != enginev1.AttackOrderType_ATTACK_ORDER_TYPE_UNSPECIFIED &&
		order.GetTargetUnitId() != ""
}

func unitCanMove(u *enginev1.Unit, defs map[string]DefStats) bool {
	if u.GetPosition() == nil {
		return false
	}
	return defs[u.DefinitionId].CruiseSpeedMps > 0
}

func nearestTrackedEnemy(unit *enginev1.Unit, tracks trackPicture, unitByID map[string]*enginev1.Unit) *enginev1.Unit {
	groupID := tracks.GroupForUnit[unit.Id]
	if groupID == "" {
		return nil
	}
	targets := tracks.ByGroup[groupID]
	var nearest *enginev1.Unit
	bestDist := math.MaxFloat64
	for targetID := range targets {
		target := unitByID[targetID]
		if target == nil || !unitIsAlive(target) || !unitsAreHostile(unit, target) {
			continue
		}
		dist := haversineM(
			unit.GetPosition().GetLat(), unit.GetPosition().GetLon(),
			target.GetPosition().GetLat(), target.GetPosition().GetLon(),
		)
		if dist < bestDist {
			bestDist = dist
			nearest = target
		}
	}
	return nearest
}

func computeWithdrawWaypoint(unit, threat *enginev1.Unit, def DefStats) *enginev1.Waypoint {
	if unit.GetPosition() == nil || threat.GetPosition() == nil {
		return nil
	}
	distance := def.DetectionRangeM * 0.75
	if distance < 20_000 {
		distance = 20_000
	}
	if distance > 150_000 {
		distance = 150_000
	}
	brng := bearingRad(threat.GetPosition().GetLat(), threat.GetPosition().GetLon(), unit.GetPosition().GetLat(), unit.GetPosition().GetLon())
	lat, lon := movePoint(unit.GetPosition().GetLat(), unit.GetPosition().GetLon(), brng, distance)
	return &enginev1.Waypoint{
		Lat:    lat,
		Lon:    lon,
		AltMsl: unit.GetPosition().GetAltMsl(),
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
