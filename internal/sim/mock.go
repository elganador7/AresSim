// Package sim provides the movement engine and adjudicator.
// Drives unit positions based on their MoveOrder waypoints each tick,
// then resolves combat between opposing units within engagement range.
package sim

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
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
					DetectedMunitionIds: nil,
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

func processBehaviorTick(units []*enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats, simSeconds float64) []*enginev1.UnitDelta {
	tracks := buildTrackPicture(units, defs, nil, nil, fixedRng(0))
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, u := range units {
		unitByID[u.Id] = u
	}

	deltas := make([]*enginev1.UnitDelta, 0)
	for _, u := range units {
		if !unitCanOperate(u) || !unitCanMove(u, defs) {
			continue
		}
		def := defs[normalizeDefinitionID(u.GetDefinitionId())]

		if shouldWithdrawForFuel(u, def, unitByID) {
			waypoint := returnToHostBaseWaypoint(u, def, unitByID)
			if waypoint != nil {
				needsUpdate := u.GetAttackOrder() != nil || returnRouteNeedsUpdate(u, waypoint)
				u.AttackOrder = nil
				if needsUpdate {
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
			}
			continue
		}

		if shouldWithdrawForPrimaryWeaponDepletion(u, def) {
			waypoint := returnToHostBaseWaypoint(u, def, unitByID)
			if waypoint != nil {
				needsUpdate := u.GetAttackOrder() != nil || returnRouteNeedsUpdate(u, waypoint)
				u.AttackOrder = nil
				if needsUpdate {
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
			}
			continue
		}

		if hasExplicitAttackOrder(u) {
			order := u.GetAttackOrder()
			target := unitByID[order.GetTargetUnitId()]
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
				AltMsl: TravelAltitudeM(u, defs[u.DefinitionId]),
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
		targetDef, ok = defs[normalizeDefinitionID(target.DefinitionId)]
	}
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
	shooterDef := defs[normalizeDefinitionID(shooter.DefinitionId)]
	return &enginev1.Waypoint{
		Lat:    lat,
		Lon:    lon,
		AltMsl: TravelAltitudeM(shooter, shooterDef),
	}
}

func CanUnitAttackTarget(shooter, target *enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats) bool {
	if shooter == nil || target == nil || shooter.GetPosition() == nil || target.GetPosition() == nil {
		return false
	}
	targetDef, ok := defs[target.DefinitionId]
	if !ok {
		targetDef, ok = defs[normalizeDefinitionID(target.DefinitionId)]
	}
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

func returnRouteNeedsUpdate(unit *enginev1.Unit, waypoint *enginev1.Waypoint) bool {
	order := unit.GetMoveOrder()
	if order == nil || len(order.GetWaypoints()) == 0 {
		return true
	}
	if !order.GetAutoGenerated() || len(order.GetWaypoints()) != 1 {
		return true
	}
	current := order.GetWaypoints()[0]
	return haversineM(current.GetLat(), current.GetLon(), waypoint.GetLat(), waypoint.GetLon()) > 1_000
}

func canHostAircraft(def DefStats) bool {
	if def.AssetClass == "airbase" {
		return true
	}
	return def.EmbarkedFixedWingCapacity > 0 ||
		def.EmbarkedRotaryWingCapacity > 0 ||
		def.EmbarkedUAVCapacity > 0 ||
		def.LaunchCapacityPerInterval > 0 ||
		def.RecoveryCapacityPerInterval > 0
}

func hostedUnitShouldMirrorBase(unit *enginev1.Unit, def DefStats) bool {
	if unit == nil || unit.GetPosition() == nil {
		return false
	}
	if def.Domain != enginev1.UnitDomain_DOMAIN_AIR {
		return false
	}
	if unit.GetPosition().GetAltMsl() > 0 {
		return false
	}
	if order := unit.GetMoveOrder(); order != nil && len(order.GetWaypoints()) > 0 {
		return false
	}
	return unit.GetHostBaseId() != ""
}

func syncHostedAircraftToHostBases(units []*enginev1.Unit, defs map[string]DefStats) []*enginev1.UnitDelta {
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, unit := range units {
		unitByID[unit.GetId()] = unit
	}

	deltas := make([]*enginev1.UnitDelta, 0)
	for _, unit := range units {
		if !hostedUnitShouldMirrorBase(unit, defs[unit.GetDefinitionId()]) {
			continue
		}
		base := unitByID[unit.GetHostBaseId()]
		if base == nil || base.GetPosition() == nil || !canHostAircraft(defs[base.GetDefinitionId()]) {
			continue
		}
		if unit.GetPosition().GetLat() == base.GetPosition().GetLat() &&
			unit.GetPosition().GetLon() == base.GetPosition().GetLon() &&
			unit.GetPosition().GetAltMsl() == base.GetPosition().GetAltMsl() {
			continue
		}
		unit.Position = &enginev1.Position{
			Lat:     base.GetPosition().GetLat(),
			Lon:     base.GetPosition().GetLon(),
			AltMsl:  base.GetPosition().GetAltMsl(),
			Heading: base.GetPosition().GetHeading(),
			Speed:   base.GetPosition().GetSpeed(),
		}
		deltas = append(deltas, &enginev1.UnitDelta{
			UnitId:   unit.GetId(),
			Position: unit.GetPosition(),
		})
	}
	return deltas
}

func isAirborneAircraft(unit *enginev1.Unit, def DefStats) bool {
	if unit == nil || unit.GetPosition() == nil {
		return false
	}
	return def.Domain == enginev1.UnitDomain_DOMAIN_AIR && unit.GetPosition().GetAltMsl() > 0
}

func requiredFuelToReachHostBaseLiters(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit) float64 {
	if unit == nil || def.Domain != enginev1.UnitDomain_DOMAIN_AIR || def.CruiseSpeedMps <= 0 || def.FuelBurnRateLph <= 0 {
		return 0
	}
	hostBaseID := unit.GetHostBaseId()
	if hostBaseID == "" {
		return 0
	}
	host := unitByID[hostBaseID]
	if host == nil || host.GetPosition() == nil || unit.GetPosition() == nil {
		return 0
	}
	distM := haversineM(unit.GetPosition().GetLat(), unit.GetPosition().GetLon(), host.GetPosition().GetLat(), host.GetPosition().GetLon())
	flightHours := distM / def.CruiseSpeedMps / 3600.0
	return flightHours * def.FuelBurnRateLph
}

func shouldWithdrawForFuel(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit) bool {
	if !isAirborneAircraft(unit, def) {
		return false
	}
	requiredFuel := requiredFuelToReachHostBaseLiters(unit, def, unitByID)
	if requiredFuel <= 0 {
		return false
	}
	return float64(unit.GetStatus().GetFuelLevelLiters()) <= requiredFuel*1.10
}

func shouldWithdrawForPrimaryWeaponDepletion(unit *enginev1.Unit, def DefStats) bool {
	if !isAirborneAircraft(unit, def) {
		return false
	}
	if strings.TrimSpace(unit.GetHostBaseId()) == "" {
		return false
	}
	primary := primaryWeaponState(unit)
	if primary == nil {
		return false
	}
	return primary.GetMaxQty() > 0 && primary.GetCurrentQty() <= 0
}

func primaryWeaponState(unit *enginev1.Unit) *enginev1.WeaponState {
	for _, weapon := range unit.GetWeapons() {
		if weapon.GetMaxQty() > 0 {
			return weapon
		}
	}
	return nil
}

func returnToHostBaseWaypoint(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit) *enginev1.Waypoint {
	if unit == nil {
		return nil
	}
	host := unitByID[unit.GetHostBaseId()]
	if host == nil || host.GetPosition() == nil {
		return nil
	}
	return &enginev1.Waypoint{
		Lat:    host.GetPosition().GetLat(),
		Lon:    host.GetPosition().GetLon(),
		AltMsl: TravelAltitudeM(unit, def),
	}
}

func shouldLandAtHostBase(unit *enginev1.Unit, unitByID map[string]*enginev1.Unit, lat, lon float64) bool {
	if unit == nil || unit.GetHostBaseId() == "" {
		return false
	}
	host := unitByID[unit.GetHostBaseId()]
	if host == nil || host.GetPosition() == nil {
		return false
	}
	return haversineM(lat, lon, host.GetPosition().GetLat(), host.GetPosition().GetLon()) <= 1_000
}

// processTick advances all active units with move orders by one tick and
// returns a UnitDelta for every unit that changed position or order state.
// timeScale multiplies how many sim-seconds of movement occur per real second.
func processTick(units []*enginev1.Unit, defs map[string]DefStats, timeScale float64) []*enginev1.UnitDelta {
	deltas := make([]*enginev1.UnitDelta, 0, len(units))
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, unit := range units {
		unitByID[unit.GetId()] = unit
	}

	for _, u := range units {
		if !unitCanOperate(u) {
			continue // skip destroyed units
		}
		pos := u.GetPosition()
		if pos == nil {
			continue
		}
		status := u.GetStatus()
		if status == nil {
			status = &enginev1.OperationalStatus{IsActive: true}
			u.Status = status
		}

		def := defs[normalizeDefinitionID(u.GetDefinitionId())]
		order := u.GetMoveOrder()
		hasMoveOrder := order != nil && len(order.Waypoints) > 0
		airborne := isAirborneAircraft(u, def)
		if !hasMoveOrder && !airborne {
			continue // unit is stationary and not consuming airborne fuel
		}

		cruiseSpeed := def.CruiseSpeedMps
		if cruiseSpeed <= 0 {
			cruiseSpeed = 10 // m/s fallback
		}

		var wp *enginev1.Waypoint
		distM := 0.0
		canMoveM := 0.0
		if hasMoveOrder {
			wp = order.Waypoints[0]
			distM = haversineM(pos.Lat, pos.Lon, wp.Lat, wp.Lon)
			canMoveM = cruiseSpeed * tickInterval.Seconds() * timeScale
		}

		fuelBurnLiters := tickFuelBurnLiters(u, def, timeScale)
		if fuelBurnLiters > 0 {
			currentFuel := float64(status.GetFuelLevelLiters())
			if currentFuel <= 0 {
				canMoveM = 0
				fuelBurnLiters = 0
			} else if hasMoveOrder && currentFuel < fuelBurnLiters {
				canMoveM *= currentFuel / fuelBurnLiters
				fuelBurnLiters = currentFuel
			} else if currentFuel < fuelBurnLiters {
				fuelBurnLiters = currentFuel
			}
		}

		newLat, newLon := pos.Lat, pos.Lon
		var remainingWaypoints []*enginev1.Waypoint

		if hasMoveOrder {
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
		}

		// Compute new heading (toward current or next target).
		newHeading := pos.Heading
		if len(remainingWaypoints) > 0 {
			target := remainingWaypoints[0]
			newHeading = BearingDeg(newLat, newLon, target.Lat, target.Lon)
		}

		// Speed: cruising while moving, stopped when done.
		newSpeed := 0.0
		var newOrder *enginev1.MoveOrder
		if len(remainingWaypoints) > 0 {
			newSpeed = cruiseSpeed
			newOrder = &enginev1.MoveOrder{
				Waypoints:     remainingWaypoints,
				AutoGenerated: order.GetAutoGenerated(),
			}
		} else if hasMoveOrder {
			newOrder = &enginev1.MoveOrder{AutoGenerated: order.GetAutoGenerated()} // empty = cleared on frontend
		}

		altMsl := pos.GetAltMsl()
		if hasMoveOrder {
			altMsl = TravelAltitudeM(u, def)
		}
		if hasMoveOrder && len(remainingWaypoints) == 0 && shouldLandAtHostBase(u, unitByID, newLat, newLon) {
			host := unitByID[u.GetHostBaseId()]
			newLat = host.GetPosition().GetLat()
			newLon = host.GetPosition().GetLon()
			altMsl = host.GetPosition().GetAltMsl()
			newHeading = host.GetPosition().GetHeading()
			newSpeed = 0
		}

		// Mutate in-memory unit state so RequestSync stays accurate.
		u.Position = &enginev1.Position{
			Lat:     newLat,
			Lon:     newLon,
			AltMsl:  altMsl,
			Heading: newHeading,
			Speed:   newSpeed,
		}
		u.MoveOrder = newOrder
		if fuelBurnLiters > 0 {
			status.FuelLevelLiters = float32(math.Max(0, float64(status.GetFuelLevelLiters())-fuelBurnLiters))
		}
		fuelExhausted := status.GetFuelLevelLiters() <= 0 && def.FuelBurnRateLph > 0
		if fuelExhausted && airborne {
			status.IsActive = false
			status.CombatEffectiveness = 0
			u.DamageState = enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED
			u.MoveOrder = nil
			u.Position.Speed = 0
			newOrder = &enginev1.MoveOrder{}
		}

		delta := &enginev1.UnitDelta{
			UnitId:    u.Id,
			Position:  u.Position,
			MoveOrder: newOrder,
		}
		if fuelBurnLiters > 0 || fuelExhausted {
			delta.Status = status
			delta.DamageState = u.GetDamageState()
		}
		deltas = append(deltas, delta)
	}

	deltas = append(deltas, syncHostedAircraftToHostBases(units, defs)...)
	return deltas
}

func processReplenishmentTick(units []*enginev1.Unit, defs map[string]DefStats, simSeconds float64) []*enginev1.UnitDelta {
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, unit := range units {
		unitByID[unit.GetId()] = unit
	}

	deltas := make([]*enginev1.UnitDelta, 0)
	for _, unit := range units {
		def := defs[normalizeDefinitionID(unit.GetDefinitionId())]
		if shouldCompleteReplenishment(unit, simSeconds) {
			refillUnit(unit, def)
			deltas = append(deltas, &enginev1.UnitDelta{
				UnitId:                 unit.GetId(),
				Status:                 unit.GetStatus(),
				Weapons:                cloneWeaponStates(unit.GetWeapons()),
				NextSortieReadySeconds: unit.GetNextSortieReadySeconds(),
			})
			continue
		}
		if shouldStartReplenishment(unit, def, unitByID, defs, simSeconds) {
			beginReplenishment(unit, def, simSeconds)
			deltas = append(deltas, &enginev1.UnitDelta{
				UnitId:                 unit.GetId(),
				Status:                 unit.GetStatus(),
				MoveOrder:              &enginev1.MoveOrder{},
				NextSortieReadySeconds: unit.GetNextSortieReadySeconds(),
			})
		}
	}
	return deltas
}

func tickFuelBurnLiters(unit *enginev1.Unit, def DefStats, timeScale float64) float64 {
	if def.FuelBurnRateLph <= 0 || timeScale <= 0 {
		return 0
	}
	if isAirborneAircraft(unit, def) {
		return def.FuelBurnRateLph * (tickInterval.Seconds() * timeScale / 3600.0)
	}
	if unit.GetMoveOrder() == nil || len(unit.GetMoveOrder().GetWaypoints()) == 0 {
		return 0
	}
	return def.FuelBurnRateLph * (tickInterval.Seconds() * timeScale / 3600.0)
}

func shouldStartReplenishment(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit, defs map[string]DefStats, simSeconds float64) bool {
	if unit == nil || unit.GetStatus() == nil || !unit.GetStatus().GetIsActive() {
		return false
	}
	if currentDamageState(unit) != enginev1.DamageState_DAMAGE_STATE_OPERATIONAL {
		return false
	}
	if !unitNeedsReplenishment(unit, def) {
		return false
	}
	return isAtReplenishmentProvider(unit, def, unitByID, defs, simSeconds)
}

func shouldCompleteReplenishment(unit *enginev1.Unit, simSeconds float64) bool {
	if unit == nil || unit.GetStatus() == nil || unit.GetStatus().GetIsActive() {
		return false
	}
	ready := unit.GetNextSortieReadySeconds()
	return ready > 0 && ready <= simSeconds && currentDamageState(unit) == enginev1.DamageState_DAMAGE_STATE_OPERATIONAL
}

func unitNeedsReplenishment(unit *enginev1.Unit, def DefStats) bool {
	if unit == nil {
		return false
	}
	if def.FuelCapacityLiters > 0 && float64(unit.GetStatus().GetFuelLevelLiters()) < def.FuelCapacityLiters*0.98 {
		return true
	}
	for _, weapon := range unit.GetWeapons() {
		if weapon.GetCurrentQty() < weapon.GetMaxQty() {
			return true
		}
	}
	return false
}

func isAtReplenishmentProvider(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit, defs map[string]DefStats, simSeconds float64) bool {
	if unit == nil || unit.GetPosition() == nil {
		return false
	}
	switch def.Domain {
	case enginev1.UnitDomain_DOMAIN_AIR:
		return canAircraftReplenish(unit, def, unitByID, defs)
	case enginev1.UnitDomain_DOMAIN_SEA, enginev1.UnitDomain_DOMAIN_LAND, enginev1.UnitDomain_DOMAIN_SUBSURFACE:
		return nearbyFriendlyLogisticsProvider(unit, def, unitByID, defs) != nil
	default:
		return false
	}
}

func canAircraftReplenish(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit, defs map[string]DefStats) bool {
	if unit == nil || unit.GetPosition() == nil {
		return false
	}
	host := unitByID[unit.GetHostBaseId()]
	if host != nil && host.GetPosition() != nil &&
		haversineM(unit.GetPosition().GetLat(), unit.GetPosition().GetLon(), host.GetPosition().GetLat(), host.GetPosition().GetLon()) <= 1_000 &&
		unit.GetPosition().GetAltMsl() <= 100 {
		return true
	}
	return nearbyAirTanker(unit, unitByID, defs)
}

func nearbyAirTanker(unit *enginev1.Unit, unitByID map[string]*enginev1.Unit, defs map[string]DefStats) bool {
	for _, provider := range unitByID {
		if provider == nil || provider.GetId() == unit.GetId() || provider.GetPosition() == nil || unitTeamID(provider) != unitTeamID(unit) {
			continue
		}
		if provider.GetPosition().GetAltMsl() <= 100 {
			continue
		}
		if haversineM(unit.GetPosition().GetLat(), unit.GetPosition().GetLon(), provider.GetPosition().GetLat(), provider.GetPosition().GetLon()) > 5_000 {
			continue
		}
		if providerDef, ok := providerDefStats(provider, defs); ok && providerDef.GeneralType == int32(enginev1.UnitGeneralType_GENERAL_TYPE_TANKER) {
			return true
		}
	}
	return false
}

func nearbyFriendlyLogisticsProvider(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit, defs map[string]DefStats) *enginev1.Unit {
	if unit.GetPosition() == nil || unit.GetPosition().GetSpeed() > 1 {
		return nil
	}
	maxDist := 2_000.0
	if def.Domain == enginev1.UnitDomain_DOMAIN_SEA || def.Domain == enginev1.UnitDomain_DOMAIN_SUBSURFACE {
		maxDist = 5_000
	}
	for _, provider := range unitByID {
		if provider == nil || provider.GetId() == unit.GetId() || provider.GetPosition() == nil || unitTeamID(provider) != unitTeamID(unit) {
			continue
		}
		providerDef, ok := providerDefStats(provider, defs)
		if !ok {
			continue
		}
		if providerDef.AssetClass != "port" && providerDef.GeneralType != int32(enginev1.UnitGeneralType_GENERAL_TYPE_LOGISTICS) {
			continue
		}
		if haversineM(unit.GetPosition().GetLat(), unit.GetPosition().GetLon(), provider.GetPosition().GetLat(), provider.GetPosition().GetLon()) <= maxDist {
			return provider
		}
	}
	return nil
}

func providerDefStats(provider *enginev1.Unit, defs map[string]DefStats) (DefStats, bool) {
	if provider == nil {
		return DefStats{}, false
	}
	def, ok := defs[normalizeDefinitionID(provider.GetDefinitionId())]
	return def, ok
}

func beginReplenishment(unit *enginev1.Unit, def DefStats, simSeconds float64) {
	if unit.GetStatus() == nil {
		unit.Status = &enginev1.OperationalStatus{}
	}
	durationSeconds := replenishmentDurationSeconds(def)
	if durationSeconds <= 0 {
		durationSeconds = 1800
	}
	if unit.GetNextSortieReadySeconds() > simSeconds {
		unit.NextSortieReadySeconds = math.Max(unit.GetNextSortieReadySeconds(), simSeconds+durationSeconds)
	} else {
		unit.NextSortieReadySeconds = simSeconds + durationSeconds
	}
	unit.Status.IsActive = false
	unit.MoveOrder = nil
	if unit.GetPosition() != nil {
		unit.Position.Speed = 0
	}
}

func refillUnit(unit *enginev1.Unit, def DefStats) {
	if unit.GetStatus() == nil {
		unit.Status = &enginev1.OperationalStatus{}
	}
	unit.Status.IsActive = true
	if def.FuelCapacityLiters > 0 {
		unit.Status.FuelLevelLiters = float32(def.FuelCapacityLiters)
	}
	for _, weapon := range unit.GetWeapons() {
		weapon.CurrentQty = weapon.GetMaxQty()
	}
	unit.NextSortieReadySeconds = 0
}

func replenishmentDurationSeconds(def DefStats) float64 {
	minutes := def.SortieIntervalMinutes
	if minutes <= 0 {
		switch def.Domain {
		case enginev1.UnitDomain_DOMAIN_AIR:
			minutes = 45
		case enginev1.UnitDomain_DOMAIN_SEA:
			minutes = 120
		case enginev1.UnitDomain_DOMAIN_SUBSURFACE:
			minutes = 180
		default:
			minutes = 60
		}
	}
	return float64(minutes) * 60
}

func cloneWeaponStates(states []*enginev1.WeaponState) []*enginev1.WeaponState {
	cloned := make([]*enginev1.WeaponState, 0, len(states))
	for _, state := range states {
		if state == nil {
			continue
		}
		cloned = append(cloned, proto.Clone(state).(*enginev1.WeaponState))
	}
	return cloned
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
		AltMsl: TravelAltitudeM(unit, def),
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
