package sim

import (
	"fmt"
	"math/rand"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func emitBatchUpdate(emit EmitFn, deltas []*enginev1.UnitDelta, simSeconds float64, tick int64) {
	if len(deltas) == 0 {
		return
	}
	emitBatchUpdateAlways(emit, deltas, simSeconds, tick)
}

func emitBatchUpdateAlways(emit EmitFn, deltas []*enginev1.UnitDelta, simSeconds float64, tick int64) {
	emit("batch_update", &enginev1.BatchUnitUpdate{
		Deltas:  deltas,
		SimTime: &enginev1.SimTime{SecondsElapsed: simSeconds, TickNumber: tick},
	})
}

func applyPlanTick(planTick func(float64) []*enginev1.UnitDelta, emit EmitFn, simSeconds float64, tick int64) {
	if planTick == nil {
		return
	}
	emitBatchUpdate(emit, planTick(simSeconds), simSeconds, tick)
}

func emitShooterWeaponUpdatesAndSpawnMunitions(shots []FiredShot, weapons map[string]WeaponStats, emit EmitFn) []*InFlightMunition {
	fired := make(map[string]bool)
	spawned := make([]*InFlightMunition, 0)
	for _, shot := range shots {
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

		ws, hasStats := weapons[shot.WeaponID]
		if !hasStats || ws.SpeedMps <= 0 {
			continue
		}
		for range shot.SalvoSize {
			spawned = append(spawned, &InFlightMunition{
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
	return spawned
}

func processInFlightIntercepts(units []*enginev1.Unit, defs map[string]DefStats, weapons map[string]WeaponStats, inFlight []*InFlightMunition, rules RelationshipRules, rng *rand.Rand, emit EmitFn) ([]*InFlightMunition, map[string][]string) {
	munitionDetections := ApplyIntelSharing(DetectMunitions(units, defs, inFlight), rules)
	var interceptShots []InterceptShot
	inFlight, interceptShots = InterceptMunitionsTick(units, defs, weapons, inFlight, munitionDetections, rng)
	emitInterceptorUpdates(interceptShots, units, emit)
	return inFlight, munitionDetections
}

func emitInterceptorUpdates(interceptShots []InterceptShot, units []*enginev1.Unit, emit EmitFn) {
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
}

func emitDetectionUpdates(emit EmitFn, detections, munitionDetections map[string][]string, detectionContacts map[string][]DetectionContactInfo) {
	activeSides := make(map[string]bool, len(detections)+len(munitionDetections)+len(detectionContacts))
	for side := range detections {
		activeSides[side] = true
	}
	for side := range munitionDetections {
		activeSides[side] = true
	}
	for side := range detectionContacts {
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
}

func emitMunitionUpdate(emit EmitFn, inFlight []*InFlightMunition, simSeconds float64, tick int64) {
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
}

func emitArrivalResults(emit EmitFn, hits []HitResult, simSeconds float64, tick int64) {
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
