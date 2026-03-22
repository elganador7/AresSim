package main

import (
	"fmt"
	"strings"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/geo"
	"github.com/aressim/internal/sim"
)

var majorActorTeams = map[string]bool{
	"USA": true,
	"ISR": true,
	"IRN": true,
}

type aiTargetCandidate struct {
	target   *enginev1.Unit
	priority float64
}

type aiAssignedTargets map[string]map[string]int

func (a *App) setHumanControlledTeam(teamID string) {
	a.humanTeamMu.Lock()
	a.humanTeam = sim.CountryDisplayCode(teamID)
	a.humanTeamMu.Unlock()
}

func (a *App) getHumanControlledTeam() string {
	a.humanTeamMu.RLock()
	defer a.humanTeamMu.RUnlock()
	return a.humanTeam
}

func (a *App) randomFloat64() float64 {
	a.aiRandMu.Lock()
	defer a.aiRandMu.Unlock()
	if a.aiRand == nil {
		return 0.5
	}
	return a.aiRand.Float64()
}

func (a *App) randomBetween(min, max float64) float64 {
	if max <= min {
		return min
	}
	return min + a.randomFloat64()*(max-min)
}

func isMajorActorTeam(teamID string) bool {
	return majorActorTeams[sim.CountryDisplayCode(teamID)]
}

func (a *App) isPlannerControlled(unit *enginev1.Unit, def sim.DefStats, simSeconds float64) bool {
	if unit == nil || unit.GetStatus() == nil || !unit.GetStatus().GetIsActive() {
		return false
	}
	if a.getHumanControlledTeam() == "" {
		return false
	}
	teamID := sim.CountryDisplayCode(unit.GetTeamId())
	if !isMajorActorTeam(teamID) {
		return false
	}
	if teamID == a.getHumanControlledTeam() {
		return false
	}
	if strings.EqualFold(def.EmploymentRole, "defensive") {
		return false
	}
	if unit.GetAttackOrder() != nil && unit.GetAttackOrder().GetTargetUnitId() != "" {
		return false
	}
	if order := unit.GetMoveOrder(); order != nil && len(order.GetWaypoints()) > 0 {
		return false
	}
	if def.Domain == enginev1.UnitDomain_DOMAIN_AIR && unit.GetNextSortieReadySeconds() > simSeconds {
		return false
	}
	if unit.GetNextStrikeReadySeconds() > simSeconds {
		return false
	}
	return true
}

func isAIFixedStrategicTarget(unit *enginev1.Unit, def sim.DefStats) bool {
	if unit == nil {
		return false
	}
	if def.AssetClass == "airbase" || def.AssetClass == "port" {
		return true
	}
	if def.TargetClass == "runway" ||
		def.TargetClass == "hardened_infrastructure" ||
		def.TargetClass == "soft_infrastructure" ||
		def.TargetClass == "civilian_energy" ||
		def.TargetClass == "civilian_water" ||
		def.TargetClass == "sam_battery" {
		return true
	}
	if def.Domain == enginev1.UnitDomain_DOMAIN_LAND {
		switch def.GeneralType {
		case 72, 73:
			return true
		}
	}
	return false
}

func targetPainValue(def sim.DefStats) float64 {
	return def.ReplacementCostUSD + def.StrategicValueUSD + def.EconomicValueUSD + float64(def.AuthorizedPersonnel)*valueOfStatisticalLifeUSD
}

func isStrategicStrikeShooter(def sim.DefStats) bool {
	return def.GeneralType == 72 || def.GeneralType == 73
}

func isRaidWorthyTarget(def sim.DefStats) bool {
	if def.AssetClass == "airbase" || def.AssetClass == "port" {
		return true
	}
	return def.TargetClass == "runway" || def.TargetClass == "civilian_energy" || def.TargetClass == "civilian_water"
}

func actorTargetBias(shooterTeam string, def sim.DefStats) float64 {
	team := sim.CountryDisplayCode(shooterTeam)
	switch team {
	case "IRN":
		switch def.AssetClass {
		case "airbase":
			return 2.8
		case "power_plant", "desalination_plant", "oil_field", "pipeline_node", "port":
			return 1.6
		}
		if def.TargetClass == "runway" {
			return 3.0
		}
		if def.TargetClass == "sam_battery" {
			return 1.9
		}
	case "USA", "ISR":
		switch def.GeneralType {
		case 72:
			return 4.4
		case 73:
			return 3.4
		}
		if def.AssetClass == "airbase" {
			return 1.7
		}
		if def.TargetClass == "sam_battery" {
			return 2.1
		}
	}
	return 1.0
}

func estimatedTargetPriority(shooterTeam string, target *enginev1.Unit, def sim.DefStats) float64 {
	valueRemaining := targetPainValue(def) * (1 - damageLossFraction(target.GetDamageState()))
	if valueRemaining <= 0 {
		return 0
	}
	return valueRemaining * actorTargetBias(shooterTeam, def)
}

func (a aiAssignedTargets) count(teamID, targetID string) int {
	if a == nil {
		return 0
	}
	return a[teamID][targetID]
}

func (a aiAssignedTargets) add(teamID, targetID string) {
	if a[teamID] == nil {
		a[teamID] = make(map[string]int)
	}
	a[teamID][targetID]++
}

func buildAssignedTargets(units []*enginev1.Unit) aiAssignedTargets {
	out := make(aiAssignedTargets)
	for _, unit := range units {
		if unit == nil || unit.GetAttackOrder() == nil || unit.GetAttackOrder().GetTargetUnitId() == "" {
			continue
		}
		out.add(sim.CountryDisplayCode(unit.GetTeamId()), unit.GetAttackOrder().GetTargetUnitId())
	}
	return out
}

func adjustedTargetPriority(base float64, shooterTeam string, shooterDef, targetDef sim.DefStats, targetID string, assigned aiAssignedTargets) float64 {
	if base <= 0 {
		return 0
	}
	count := assigned.count(sim.CountryDisplayCode(shooterTeam), targetID)
	if count == 0 {
		return base
	}
	if isStrategicStrikeShooter(shooterDef) && isRaidWorthyTarget(targetDef) && count < 3 {
		return base * (1 + 0.2/float64(count))
	}
	return base / (1 + float64(count)*1.35)
}

func (a *App) applyPlannerTimingJitter(shooter *enginev1.Unit, def sim.DefStats, simSeconds float64) {
	if shooter == nil || !isStrategicStrikeShooter(def) {
		return
	}
	delaySeconds := a.randomBetween(180, 900)
	readyAt := simSeconds + delaySeconds
	if shooter.GetNextStrikeReadySeconds() < readyAt {
		shooter.NextStrikeReadySeconds = readyAt
	}
}

func unitsHostileForPlanning(shooter, target *enginev1.Unit) bool {
	if shooter == nil || target == nil {
		return false
	}
	shooterTeam := sim.CountryDisplayCode(shooter.GetTeamId())
	targetTeam := sim.CountryDisplayCode(target.GetTeamId())
	if shooterTeam == "" || targetTeam == "" {
		return false
	}
	if shooterTeam == targetTeam {
		return false
	}
	shooterCoalition := strings.TrimSpace(strings.ToUpper(shooter.GetCoalitionId()))
	targetCoalition := strings.TrimSpace(strings.ToUpper(target.GetCoalitionId()))
	if shooterCoalition != "" && targetCoalition != "" {
		return shooterCoalition != targetCoalition
	}
	return true
}

func (a *App) aiStrikePathAllowed(shooter, target *enginev1.Unit, defs map[string]sim.DefStats, weapons map[string]sim.WeaponStats) bool {
	if shooter == nil || target == nil {
		return false
	}
	shooterDef := defs[shooter.GetDefinitionId()]
	if isMaritimeDomain(shooterDef.Domain) && geo.IsLandPoint(geo.Point{
		Lat: target.GetPosition().GetLat(),
		Lon: target.GetPosition().GetLon(),
	}) {
		return false
	}
	points := []geo.Point{{
		Lat: shooter.GetPosition().GetLat(),
		Lon: shooter.GetPosition().GetLon(),
	}}
	if waypoint := sim.ComputeAttackWaypointForOrder(shooter, target, defs, weapons); waypoint != nil {
		if isMaritimeDomain(shooterDef.Domain) {
			rerouted, ok := geo.BuildMaritimeRoute(
				points[0],
				geo.Point{Lat: waypoint.GetLat(), Lon: waypoint.GetLon()},
			)
			if !ok {
				return false
			}
			points = append(points, rerouted...)
		} else {
			points = append(points, geo.Point{Lat: waypoint.GetLat(), Lon: waypoint.GetLon()})
		}
	}
	points = append(points, geo.Point{
		Lat: target.GetPosition().GetLat(),
		Lon: target.GetPosition().GetLon(),
	})
	return previewStrikePath(unitCountryCode(shooter), isMaritimeDomain(shooterDef.Domain), points, a.relationshipRules(), a.countryCoalitions()) == nil
}

func (a *App) planMajorActorStrikes(simSeconds float64) []*enginev1.UnitDelta {
	if a.currentScenario == nil {
		return nil
	}
	defs := a.getCachedDefs()
	weapons := a.buildWeaponCatalog()
	assignedTargets := buildAssignedTargets(a.currentScenario.GetUnits())
	targets := make([]*enginev1.Unit, 0, len(a.currentScenario.GetUnits()))
	for _, candidate := range a.currentScenario.GetUnits() {
		if candidate == nil || !candidate.GetStatus().GetIsActive() {
			continue
		}
		def := defs[candidate.GetDefinitionId()]
		if !isAIFixedStrategicTarget(candidate, def) {
			continue
		}
		targets = append(targets, candidate)
	}

	deltas := make([]*enginev1.UnitDelta, 0)
	for _, shooter := range a.currentScenario.GetUnits() {
		if shooter == nil {
			continue
		}
		def := defs[shooter.GetDefinitionId()]
		if !a.isPlannerControlled(shooter, def, simSeconds) {
			continue
		}
		var bestTarget *enginev1.Unit
		bestPriority := 0.0
		candidates := make([]aiTargetCandidate, 0)
		for _, target := range targets {
			if !unitsHostileForPlanning(shooter, target) {
				continue
			}
			targetDef := defs[target.GetDefinitionId()]
			if !sim.CanUnitAttackTarget(shooter, target, defs, weapons) {
				continue
			}
			if !a.aiStrikePathAllowed(shooter, target, defs, weapons) {
				continue
			}
			priority := adjustedTargetPriority(
				estimatedTargetPriority(shooter.GetTeamId(), target, targetDef),
				shooter.GetTeamId(),
				def,
				targetDef,
				target.GetId(),
				assignedTargets,
			)
			if priority > 0 {
				candidates = append(candidates, aiTargetCandidate{target: target, priority: priority})
			}
			if priority > bestPriority {
				bestPriority = priority
			}
		}
		bestTarget = a.selectPlannerTarget(candidates, bestPriority)
		if bestTarget == nil {
			continue
		}
		assignedTargets.add(sim.CountryDisplayCode(shooter.GetTeamId()), bestTarget.GetId())
		if current := shooter.GetMoveOrder(); current == nil || len(current.GetWaypoints()) == 0 {
			if err := a.validateAndConsumeLaunch(shooter, def); err != nil {
				continue
			}
		}
		shooter.AttackOrder = &enginev1.AttackOrder{
			OrderType:      enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT,
			TargetUnitId:   bestTarget.GetId(),
			DesiredEffect:  plannedDesiredEffect(bestTarget, defs[bestTarget.GetDefinitionId()]),
			PkillThreshold: 0.7,
		}
		if waypoint := sim.ComputeAttackWaypointForOrder(shooter, bestTarget, defs, weapons); waypoint != nil {
			waypoints := []*enginev1.Waypoint{waypoint}
			if isMaritimeDomain(def.Domain) {
				rerouted, ok := geo.BuildMaritimeRoute(
					geo.Point{Lat: shooter.GetPosition().GetLat(), Lon: shooter.GetPosition().GetLon(), AltMsl: shooter.GetPosition().GetAltMsl()},
					geo.Point{Lat: waypoint.GetLat(), Lon: waypoint.GetLon(), AltMsl: waypoint.GetAltMsl()},
				)
				if !ok {
					shooter.AttackOrder = nil
					continue
				}
				waypoints = make([]*enginev1.Waypoint, 0, len(rerouted))
				for _, routePoint := range rerouted {
					waypoints = append(waypoints, &enginev1.Waypoint{
						Lat: routePoint.Lat, Lon: routePoint.Lon, AltMsl: waypoint.GetAltMsl(),
					})
				}
			}
			shooter.MoveOrder = &enginev1.MoveOrder{
				Waypoints:     waypoints,
				AutoGenerated: true,
			}
		}
		a.applyPlannerTimingJitter(shooter, def, simSeconds)
		deltas = append(deltas, launchStateDeltas(shooter, a.currentScenario)...)
		deltas = append(deltas, &enginev1.UnitDelta{
			UnitId:                 shooter.GetId(),
			MoveOrder:              shooter.GetMoveOrder(),
			NextStrikeReadySeconds: shooter.GetNextStrikeReadySeconds(),
		})
		a.emitProtoEvent("narrative", &enginev1.NarrativeEvent{
			Text:     fmt.Sprintf("%s AI assigns %s to strike %s", sim.CountryDisplayCode(shooter.GetTeamId()), shooter.GetDisplayName(), bestTarget.GetDisplayName()),
			Category: "c2",
			UnitId:   shooter.GetId(),
			TeamId:   sim.CountryDisplayCode(shooter.GetTeamId()),
		})
	}
	return deltas
}

func (a *App) selectPlannerTarget(candidates []aiTargetCandidate, bestPriority float64) *enginev1.Unit {
	if len(candidates) == 0 || bestPriority <= 0 {
		return nil
	}
	shortlist := make([]aiTargetCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.priority >= bestPriority*0.8 {
			shortlist = append(shortlist, candidate)
		}
	}
	if len(shortlist) == 0 {
		shortlist = candidates
	}
	if len(shortlist) == 1 {
		return shortlist[0].target
	}
	totalWeight := 0.0
	for _, candidate := range shortlist {
		totalWeight += candidate.priority
	}
	if totalWeight <= 0 {
		return shortlist[0].target
	}
	roll := a.randomFloat64() * totalWeight
	accum := 0.0
	for _, candidate := range shortlist {
		accum += candidate.priority
		if roll <= accum {
			return candidate.target
		}
	}
	return shortlist[len(shortlist)-1].target
}

func plannedDesiredEffect(target *enginev1.Unit, def sim.DefStats) enginev1.DesiredEffect {
	if def.AssetClass == "airbase" || def.TargetClass == "runway" || def.TargetClass == "sam_battery" {
		return enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL
	}
	if def.TargetClass == "civilian_energy" || def.TargetClass == "civilian_water" {
		return enginev1.DesiredEffect_DESIRED_EFFECT_DAMAGE
	}
	if def.GeneralType == 72 {
		return enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY
	}
	return enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL
}
