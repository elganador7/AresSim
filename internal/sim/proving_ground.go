package sim

import (
	"math"
	"math/rand"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"google.golang.org/protobuf/proto"
)

type ProvingGroundTrialResult struct {
	ElapsedSeconds              float64
	WinnerTeam                  string
	FocusTeam                   string
	FocusWin                    bool
	TargetMissionKilled         bool
	TargetDestroyed             bool
	FirstShotSeconds            float64
	ShotsFired                  int
	HitsScored                  int
	FuelExhaustions             int
	FocusLosses                 int
	OpposingLosses              int
	TerminalReason              string
	TargetMissionKillTimeSecond float64
	TargetDestroyedTimeSecond   float64
	Events                      []ProvingGroundEvent
}

type ProvingGroundAggregate struct {
	Trials                int
	FocusTeam             string
	FocusWinRate          float64
	TargetMissionKillRate float64
	TargetDestroyedRate   float64
	MeanElapsedSeconds    float64
	MeanFirstShotSeconds  float64
	MeanShotsFired        float64
	MeanHitsScored        float64
	MeanFuelExhaustions   float64
	MeanReplenishments    float64
	MeanFocusLosses       float64
	MeanOpposingLosses    float64
	TerminalReasons       map[string]int
	SampleEvents          []ProvingGroundEvent
}

type ProvingGroundEvent struct {
	TimeSeconds  float64
	Type         string
	ActorUnitID  string
	TargetUnitID string
	WeaponID     string
	Detail       string
}

func RunProvingGroundTrial(
	scen *enginev1.Scenario,
	defs map[string]DefStats,
	weapons map[string]WeaponStats,
	rules RelationshipRules,
	maxSimSeconds float64,
	focusTeam string,
	opposingTeam string,
	trackedTargetUnitID string,
	endOnTrackedTargetDisable bool,
	seed int64,
) ProvingGroundTrialResult {
	cloned := proto.Clone(scen).(*enginev1.Scenario)
	units := cloned.GetUnits()
	var inFlight []*InFlightMunition
	simSeconds := 0.0
	rng := rand.New(rand.NewSource(seed))
	result := ProvingGroundTrialResult{
		FocusTeam:                   focusTeam,
		FirstShotSeconds:            -1,
		TargetMissionKillTimeSecond: -1,
		TargetDestroyedTimeSecond:   -1,
	}
	fuelExhaustedSeen := map[string]bool{}
	replenishingState := map[string]bool{}
	for _, unit := range units {
		if unit == nil {
			continue
		}
		replenishingState[unit.GetId()] = unitIsReplenishing(unit, simSeconds)
	}

	for simSeconds < maxSimSeconds {
		processBehaviorTick(units, defs, weapons, simSeconds)
		moveDeltas := processTick(units, defs, 1.0)
		for _, delta := range moveDeltas {
			if delta == nil || delta.GetStatus() == nil {
				continue
			}
			if fuelExhaustedSeen[delta.GetUnitId()] {
				continue
			}
			if delta.GetStatus().GetFuelLevelLiters() > 0 {
				continue
			}
			if delta.GetDamageState() != enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED {
				continue
			}
			fuelExhaustedSeen[delta.GetUnitId()] = true
			result.FuelExhaustions++
			result.Events = append(result.Events, ProvingGroundEvent{
				TimeSeconds: simSeconds,
				Type:        "fuel_exhausted",
				ActorUnitID: delta.GetUnitId(),
				Detail:      "unit exhausted fuel and became mission-killed",
			})
		}
		for _, unit := range units {
			if unit == nil {
				continue
			}
			replenishingNow := unitIsReplenishing(unit, simSeconds)
			replenishingBefore := replenishingState[unit.GetId()]
			switch {
			case replenishingNow && !replenishingBefore:
				result.Events = append(result.Events, ProvingGroundEvent{
					TimeSeconds: simSeconds,
					Type:        "replenishment_started",
					ActorUnitID: unit.GetId(),
					Detail:      "unit entered refuel/rearm cycle",
				})
			case !replenishingNow && replenishingBefore:
				result.Events = append(result.Events, ProvingGroundEvent{
					TimeSeconds: simSeconds,
					Type:        "replenishment_complete",
					ActorUnitID: unit.GetId(),
					Detail:      "unit completed refuel/rearm cycle",
				})
			}
			replenishingState[unit.GetId()] = replenishingNow
		}

		adj := AdjudicateTick(units, defs, weapons, inFlight, rules, simSeconds)
		for _, shot := range adj.Shots {
			ws, ok := weapons[shot.WeaponID]
			if !ok || ws.SpeedMps <= 0 {
				continue
			}
			if result.FirstShotSeconds < 0 {
				result.FirstShotSeconds = simSeconds
			}
			result.ShotsFired += int(shot.SalvoSize)
			result.Events = append(result.Events, ProvingGroundEvent{
				TimeSeconds:  simSeconds,
				Type:         "shot_fired",
				ActorUnitID:  shot.Shooter.GetId(),
				TargetUnitID: shot.Target.GetId(),
				WeaponID:     shot.WeaponID,
				Detail:       "salvo launched",
			})
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

		var arrived []*InFlightMunition
		inFlight, arrived = AdvanceMunitions(inFlight, 1.0, units, defs)
		hits := ResolveArrivals(arrived, units, defs, weapons, rng)
		for _, hit := range hits {
			if hit.Victim == nil {
				continue
			}
			result.HitsScored++
			result.Events = append(result.Events, ProvingGroundEvent{
				TimeSeconds:  simSeconds,
				Type:         "target_hit",
				ActorUnitID:  safeUnitID(hit.Attacker),
				TargetUnitID: hit.Victim.GetId(),
				Detail:       outcomeLabel(hit.Outcome),
			})
			if hit.Destroyed {
				switch unitTeamID(hit.Victim) {
				case focusTeam:
					result.FocusLosses++
				case opposingTeam:
					result.OpposingLosses++
				}
				result.Events = append(result.Events, ProvingGroundEvent{
					TimeSeconds:  simSeconds,
					Type:         "unit_destroyed",
					ActorUnitID:  safeUnitID(hit.Attacker),
					TargetUnitID: hit.Victim.GetId(),
					Detail:       unitTeamID(hit.Victim),
				})
			}
			if hit.Victim.GetId() != trackedTargetUnitID {
				continue
			}
			if result.TargetMissionKillTimeSecond < 0 &&
				hit.Victim.GetDamageState() >= enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED &&
				hit.PreviousState < enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED {
				result.TargetMissionKillTimeSecond = simSeconds
				result.Events = append(result.Events, ProvingGroundEvent{
					TimeSeconds:  simSeconds,
					Type:         "target_mission_killed",
					ActorUnitID:  safeUnitID(hit.Attacker),
					TargetUnitID: hit.Victim.GetId(),
					Detail:       "tracked target mission-killed",
				})
			}
			if result.TargetDestroyedTimeSecond < 0 && hit.Destroyed {
				result.TargetDestroyedTimeSecond = simSeconds
				result.Events = append(result.Events, ProvingGroundEvent{
					TimeSeconds:  simSeconds,
					Type:         "target_destroyed",
					ActorUnitID:  safeUnitID(hit.Attacker),
					TargetUnitID: hit.Victim.GetId(),
					Detail:       "tracked target destroyed",
				})
			}
		}

		target := findUnitByID(units, trackedTargetUnitID)
		if endOnTrackedTargetDisable && target != nil &&
			target.GetDamageState() >= enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED &&
			len(inFlight) == 0 {
			result.TerminalReason = "tracked_target_neutralized"
			result.Events = append(result.Events, ProvingGroundEvent{
				TimeSeconds: simSeconds,
				Type:        "scenario_ended",
				Detail:      result.TerminalReason,
			})
			simSeconds += 1.0
			break
		}

		simSeconds += 1.0
		if terminalCombatState(units, focusTeam, opposingTeam) && len(inFlight) == 0 {
			result.TerminalReason = terminalReason(units, focusTeam, opposingTeam)
			result.Events = append(result.Events, ProvingGroundEvent{
				TimeSeconds: simSeconds,
				Type:        "scenario_ended",
				Detail:      result.TerminalReason,
			})
			break
		}
	}
	if result.TerminalReason == "" {
		result.TerminalReason = "timeout"
		result.Events = append(result.Events, ProvingGroundEvent{
			TimeSeconds: simSeconds,
			Type:        "scenario_ended",
			Detail:      result.TerminalReason,
		})
	}

	winner := winningTeamByCombatPower(units, focusTeam, opposingTeam)
	target := findUnitByID(units, trackedTargetUnitID)
	if result.TerminalReason == "tracked_target_neutralized" && target != nil && unitTeamID(target) == opposingTeam {
		winner = focusTeam
	}
	result.ElapsedSeconds = simSeconds
	result.WinnerTeam = winner
	result.FocusWin = winner == focusTeam
	result.TargetMissionKilled = target != nil && target.GetDamageState() >= enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED
	result.TargetDestroyed = target != nil && target.GetDamageState() == enginev1.DamageState_DAMAGE_STATE_DESTROYED
	return result
}

func AggregateProvingGroundResults(results []ProvingGroundTrialResult, focusTeam string) ProvingGroundAggregate {
	if len(results) == 0 {
		return ProvingGroundAggregate{FocusTeam: focusTeam}
	}
	var wins, missionKills, destroyed int
	var elapsed, firstShot, shotsFired, hitsScored, fuelExhaustions, replenishments, focusLosses, opposingLosses float64
	var firstShotObserved int
	terminalReasons := map[string]int{}
	var sampleEvents []ProvingGroundEvent
	for _, result := range results {
		if result.FocusWin {
			wins++
		}
		if result.TargetMissionKilled {
			missionKills++
		}
		if result.TargetDestroyed {
			destroyed++
		}
		elapsed += result.ElapsedSeconds
		shotsFired += float64(result.ShotsFired)
		hitsScored += float64(result.HitsScored)
		fuelExhaustions += float64(result.FuelExhaustions)
		replenishments += float64(countEventType(result.Events, "replenishment_started"))
		focusLosses += float64(result.FocusLosses)
		opposingLosses += float64(result.OpposingLosses)
		if result.FirstShotSeconds >= 0 {
			firstShot += result.FirstShotSeconds
			firstShotObserved++
		}
		if result.TerminalReason != "" {
			terminalReasons[result.TerminalReason]++
		}
		if len(sampleEvents) == 0 && len(result.Events) > 0 {
			sampleEvents = append(sampleEvents, result.Events...)
		}
	}
	n := float64(len(results))
	meanFirstShot := -1.0
	if firstShotObserved > 0 {
		meanFirstShot = firstShot / float64(firstShotObserved)
	}
	return ProvingGroundAggregate{
		Trials:                len(results),
		FocusTeam:             focusTeam,
		FocusWinRate:          float64(wins) / n,
		TargetMissionKillRate: float64(missionKills) / n,
		TargetDestroyedRate:   float64(destroyed) / n,
		MeanElapsedSeconds:    elapsed / n,
		MeanFirstShotSeconds:  meanFirstShot,
		MeanShotsFired:        shotsFired / n,
		MeanHitsScored:        hitsScored / n,
		MeanFuelExhaustions:   fuelExhaustions / n,
		MeanReplenishments:    replenishments / n,
		MeanFocusLosses:       focusLosses / n,
		MeanOpposingLosses:    opposingLosses / n,
		TerminalReasons:       terminalReasons,
		SampleEvents:          sampleEvents,
	}
}

func terminalReason(units []*enginev1.Unit, focusTeam, opposingTeam string) string {
	focusActive := false
	opposingActive := false
	for _, unit := range units {
		if unit == nil || !unitCanOperate(unit) {
			continue
		}
		switch unitTeamID(unit) {
		case focusTeam:
			focusActive = true
		case opposingTeam:
			opposingActive = true
		}
	}
	switch {
	case !focusActive && !opposingActive:
		return "mutual_attrition"
	case !opposingActive:
		return "opposing_eliminated"
	case !focusActive:
		return "focus_eliminated"
	default:
		return "ongoing"
	}
}

func terminalCombatState(units []*enginev1.Unit, teams ...string) bool {
	activeByTeam := make(map[string]bool, len(teams))
	for _, unit := range units {
		if unit == nil || !unitCanOperate(unit) {
			continue
		}
		team := unitTeamID(unit)
		for _, candidate := range teams {
			if team == candidate {
				activeByTeam[candidate] = true
			}
		}
	}
	for _, team := range teams {
		if !activeByTeam[team] {
			return true
		}
	}
	return false
}

func safeUnitID(unit *enginev1.Unit) string {
	if unit == nil {
		return ""
	}
	return unit.GetId()
}

func outcomeLabel(outcome impactOutcome) string {
	switch outcome {
	case outcomeNoEffect:
		return "no_effect"
	case outcomeLightDamage:
		return "light_damage"
	case outcomeMobilityKill:
		return "mobility_kill"
	case outcomeFirepowerLoss:
		return "firepower_loss"
	case outcomeMissionKill:
		return "mission_kill"
	case outcomeRunwayCrater:
		return "runway_crater"
	case outcomeCatastrophicKill:
		return "catastrophic_kill"
	default:
		return "unknown"
	}
}

func winningTeamByCombatPower(units []*enginev1.Unit, teams ...string) string {
	bestTeam := ""
	bestScore := -1.0
	for _, team := range teams {
		score := 0.0
		for _, unit := range units {
			if unit == nil || unitTeamID(unit) != team {
				continue
			}
			if unit.GetStatus() == nil {
				continue
			}
			if unit.GetDamageState() == enginev1.DamageState_DAMAGE_STATE_DESTROYED {
				continue
			}
			score += math.Max(0, float64(unit.GetStatus().GetCombatEffectiveness()))
		}
		if score > bestScore {
			bestScore = score
			bestTeam = team
		}
	}
	return bestTeam
}

func findUnitByID(units []*enginev1.Unit, id string) *enginev1.Unit {
	for _, unit := range units {
		if unit != nil && unit.GetId() == id {
			return unit
		}
	}
	return nil
}

func unitIsReplenishing(unit *enginev1.Unit, simSeconds float64) bool {
	if unit == nil || unit.GetStatus() == nil {
		return false
	}
	if unit.GetStatus().GetIsActive() {
		return false
	}
	return unit.GetNextSortieReadySeconds() > simSeconds
}

func countEventType(events []ProvingGroundEvent, eventType string) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}
