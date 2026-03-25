package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"

	"github.com/aressim/internal/db"
	"github.com/aressim/internal/db/repository"
	"github.com/aressim/internal/geo"
	"github.com/aressim/internal/library"
	"github.com/aressim/internal/routing"
	"github.com/aressim/internal/scenario"
	"github.com/aressim/internal/sim"
	"google.golang.org/protobuf/proto"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// getSimTimeScale reads the current time scale atomically.
func (a *App) getSimTimeScale() float64 {
	bits := a.simTimeScale.Load()
	if bits == 0 {
		return 1.0
	}
	return math.Float64frombits(bits)
}

// setSimTimeScale stores a new time scale atomically.
func (a *App) setSimTimeScale(v float64) {
	a.simTimeScale.Store(math.Float64bits(v))
}

// getSimSeconds reads the accumulated sim time atomically.
func (a *App) getSimSeconds() float64 {
	return math.Float64frombits(a.simSecondsAtomic.Load())
}

// setSimSeconds stores the accumulated sim time atomically.
func (a *App) setSimSeconds(v float64) {
	a.simSecondsAtomic.Store(math.Float64bits(v))
}

// getCachedDefs returns the cached DefStats map, building it from the DB if
// the cache is empty. The cache is invalidated when definitions are modified.
func (a *App) getCachedDefs() map[string]sim.DefStats {
	a.defsCacheMu.RLock()
	if a.defsCache != nil {
		d := a.defsCache
		a.defsCacheMu.RUnlock()
		return d
	}
	a.defsCacheMu.RUnlock()

	d := a.buildDefs()
	a.defsCacheMu.Lock()
	a.defsCache = d
	a.defsCacheMu.Unlock()
	return d
}

func (a *App) getCachedWeaponCatalog() map[string]sim.WeaponStats {
	a.defsCacheMu.RLock()
	if a.weaponCatalogCache != nil {
		w := a.weaponCatalogCache
		a.defsCacheMu.RUnlock()
		return w
	}
	a.defsCacheMu.RUnlock()

	w := a.buildWeaponCatalog()
	a.defsCacheMu.Lock()
	a.weaponCatalogCache = w
	a.defsCacheMu.Unlock()
	return w
}

// invalidateDefsCache clears the merged combat caches so the next call to
// getCachedDefs/getCachedWeaponCatalog rebuilds them.
func (a *App) invalidateDefsCache() {
	a.defsCacheMu.Lock()
	a.defsCache = nil
	a.weaponCatalogCache = nil
	a.defsCacheMu.Unlock()
}

func (a *App) invalidateRouteCache() {
	a.routeCacheMu.Lock()
	a.routeCache = nil
	a.routeCacheMu.Unlock()
}

func routeCacheKey(req routing.Request) string {
	coalitionKeys := make([]string, 0, len(req.CountryCoalitions))
	for country, coalition := range req.CountryCoalitions {
		coalitionKeys = append(coalitionKeys, country+"="+coalition)
	}
	sort.Strings(coalitionKeys)

	ruleKeys := make([]string, 0)
	for from, recipients := range req.RelationshipRules {
		for to, rule := range recipients {
			ruleKeys = append(ruleKeys, fmt.Sprintf("%s>%s:%t:%t:%t:%t:%t:%t",
				from, to,
				rule.ShareIntel,
				rule.AirspaceTransitAllowed,
				rule.AirspaceStrikeAllowed,
				rule.DefensivePositioningAllowed,
				rule.MaritimeTransitAllowed,
				rule.MaritimeStrikeAllowed,
			))
		}
	}
	sort.Strings(ruleKeys)

	return fmt.Sprintf("%s|%d|%s|%.4f,%.4f,%.0f|%.4f,%.4f,%.0f|%s|%s",
		req.OwnerCountry,
		req.Domain,
		req.Purpose,
		req.Start.Lat, req.Start.Lon, req.Start.AltMsl,
		req.End.Lat, req.End.Lon, req.End.AltMsl,
		strings.Join(ruleKeys, ";"),
		strings.Join(coalitionKeys, ";"),
	)
}

func (a *App) cachedBuildRoute(req routing.Request) routing.Result {
	key := routeCacheKey(req)
	a.routeCacheMu.RLock()
	if cached, ok := a.routeCache[key]; ok {
		a.routeCacheMu.RUnlock()
		return cached
	}
	a.routeCacheMu.RUnlock()

	result := routing.BuildRoute(req)

	a.routeCacheMu.Lock()
	if a.routeCache == nil {
		a.routeCache = make(map[string]routing.Result, 512)
	}
	if len(a.routeCache) >= 2048 {
		a.routeCache = make(map[string]routing.Result, 512)
	}
	a.routeCache[key] = result
	a.routeCacheMu.Unlock()
	return result
}

// storeLastDetection records the most recent sensor result for one team so
// RequestSync can replay it to reconnecting clients.
func (a *App) storeLastDetection(teamID string, ids []string) {
	a.lastDetMu.Lock()
	if a.lastDetections == nil {
		a.lastDetections = make(map[string][]string)
	}
	cp := make([]string, len(ids))
	copy(cp, ids)
	a.lastDetections[teamID] = cp
	a.lastDetMu.Unlock()
}

// makeEmitFn returns an EmitFn that intercepts detection updates and checkpoints.
func (a *App) makeEmitFn() sim.EmitFn {
	var ticksSinceCheckpoint int
	return func(name string, msg proto.Message) {
		switch name {
		case "detection_update":
			if du, ok := msg.(*enginev1.DetectionUpdate); ok {
				a.storeLastDetection(du.DetectingTeam, du.DetectedUnitIds)
			}
		case "batch_update":
			if bu, ok := msg.(*enginev1.BatchUnitUpdate); ok {
				if batchNeedsScoreUpdate(bu) && a.currentScenario != nil {
					bu.Scores = buildTeamScores(a.currentScenario.GetUnits(), a.getCachedDefs())
				}
				if bu.SimTime != nil {
					ticksSinceCheckpoint++
					if ticksSinceCheckpoint >= db.CheckpointInterval {
						ticksSinceCheckpoint = 0
						go a.writeCheckpoint(bu.SimTime.TickNumber, bu.SimTime.SecondsElapsed)
					}
				}
			}
		}
		a.emitProtoEvent(name, msg)
	}
}

// writeCheckpoint persists the current in-memory unit positions and scenario progress.
func (a *App) writeCheckpoint(tick int64, simSeconds float64) {
	if a.checkpoint == nil || a.currentScenario == nil {
		return
	}
	units := make([]repository.UnitRecord, 0, len(a.currentScenario.Units))
	for _, u := range a.currentScenario.Units {
		if u.Status != nil && !u.Status.IsActive {
			continue
		}
		if row := unitRecord(u); row != nil {
			units = append(units, row)
		}
	}
	snap := db.Snapshot{
		ScenarioID: a.currentScenario.Id,
		Tick:       tick,
		SimSeconds: simSeconds,
		Units:      units,
	}
	if err := a.checkpoint.Write(a.ctx, snap); err != nil {
		slog.Warn("checkpoint write failed", "tick", tick, "err", err)
	}
}

// loadScenario is the internal entry point for starting a scenario.
func (a *App) loadScenario(scen *enginev1.Scenario) {
	if a.scenRepo != nil {
		if err := a.scenRepo.Save(a.ctx, scen.Id, scenarioRecord(scen)); err != nil {
			slog.Warn("save scenario to db", "err", err)
		}
	}

	a.currentScenario = scen
	a.setHumanControlledTeam("")
	a.invalidateRouteCache()
	defs := a.prepareScenarioForSimulation(scen)
	if a.unitRepo != nil {
		if err := a.unitRepo.DeleteAll(a.ctx); err != nil {
			slog.Warn("clear unit table", "err", err)
		} else {
			rows := make([]repository.UnitRecord, 0, len(scen.Units))
			for _, u := range scen.Units {
				if row := unitRecord(u); row != nil {
					rows = append(rows, row)
				}
			}
			if err := a.unitRepo.UpsertBatch(a.ctx, rows); err != nil {
				slog.Warn("seed unit rows", "err", err)
			}
		}
	}

	a.emitProtoEvent("full_state_snapshot", &enginev1.FullStateSnapshot{
		Units:             scen.Units,
		SimTime:           &enginev1.SimTime{},
		Weather:           scen.GetMap().GetInitialWeather(),
		ScenarioName:      scen.Name,
		WeaponDefinitions: a.listWeaponDefsProto(),
		Relationships:     scen.GetRelationships(),
		Scores:            buildTeamScores(scen.GetUnits(), defs),
	})
	a.emitProtoEvent("scenario_state", &enginev1.ScenarioStateEvent{
		State:     enginev1.ScenarioPlayState_SCENARIO_RUNNING,
		TimeScale: scen.GetSettings().GetTimeScale(),
	})

	a.setSimTimeScale(1.0)
	a.setSimSeconds(0)
	a.lastDetMu.Lock()
	a.lastDetections = nil
	a.lastDetMu.Unlock()

	if a.simCancel != nil {
		a.simCancel()
	}
	simCtx, simCancel := context.WithCancel(a.ctx)
	a.simCancel = simCancel
	go sim.MockLoop(
		simCtx,
		scen.Units,
		defs,
		a.getCachedWeaponCatalog(),
		a.relationshipRules,
		0,
		a.getSimTimeScale,
		a.setSimSeconds,
		a.planMajorActorStrikes,
		a.makeEmitFn(),
	)
	slog.Info("scenario loaded", "name", scen.Name, "units", len(scen.Units))
}

func (a *App) prepareScenarioForSimulation(scen *enginev1.Scenario) map[string]sim.DefStats {
	a.defsCacheMu.Lock()
	a.defsCache = a.buildDefs()
	a.weaponCatalogCache = a.buildWeaponCatalog()
	a.defsCacheMu.Unlock()
	defs := a.getCachedDefs()
	for _, u := range scen.Units {
		ensureUnitOpsState(u, defs[u.DefinitionId])
	}

	generalTypeByDefID := map[string]int32{}
	if a.unitDefRepo != nil {
		unitDefs, _ := a.unitDefRepo.List(a.ctx)
		generalTypeByDefID = make(map[string]int32, len(unitDefs))
		for _, row := range unitDefs {
			id := extractRecordID(row["id"])
			if gt, ok := row["general_type"]; ok {
				generalTypeByDefID[id] = int32(toFloat64(gt))
			}
		}
	}

	for _, u := range scen.Units {
		if len(u.Weapons) != 0 {
			continue
		}
		defID := extractRecordID(u.DefinitionId)
		if def, ok := a.libDefsCache[defID]; ok {
			loadoutID, slots := selectWeaponConfiguration(def, u.GetLoadoutConfigurationId())
			if len(slots) > 0 {
				u.LoadoutConfigurationId = loadoutID
				u.Weapons = loadoutToWeaponStates(slots)
				continue
			}
		}
		if def, ok := a.libDefsCache[defID]; ok && len(def.DefaultLoadout) > 0 {
			u.LoadoutConfigurationId = "default"
			u.Weapons = loadoutToWeaponStates(def.DefaultLoadout)
			continue
		}
		gt := generalTypeByDefID[defID]
		u.Weapons = scenario.InitUnitWeapons(u, gt)
	}
	return defs
}

func ensureUnitOpsState(u *enginev1.Unit, def sim.DefStats) {
	if u == nil {
		return
	}
	if u.BaseOps == nil && canHostAircraftFromPlatform(def) {
		u.BaseOps = &enginev1.BaseOpsState{
			State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE,
		}
	}
	if u.NextSortieReadySeconds < 0 {
		u.NextSortieReadySeconds = 0
	}
}

func canHostAircraftFromPlatform(def sim.DefStats) bool {
	if def.AssetClass == "airbase" {
		return true
	}
	return def.EmbarkedFixedWingCapacity > 0 ||
		def.EmbarkedRotaryWingCapacity > 0 ||
		def.EmbarkedUAVCapacity > 0 ||
		def.LaunchCapacityPerInterval > 0 ||
		def.RecoveryCapacityPerInterval > 0
}

func shouldInitiateLaunch(u *enginev1.Unit, def sim.DefStats) bool {
	if u == nil {
		return false
	}
	if def.Domain != enginev1.UnitDomain_DOMAIN_AIR {
		return false
	}
	pos := u.GetPosition()
	return pos != nil && pos.GetAltMsl() <= 0
}

func findScenarioUnit(units []*enginev1.Unit, id string) *enginev1.Unit {
	for _, u := range units {
		if u.GetId() == id {
			return u
		}
	}
	return nil
}

func isPreplannedFixedTarget(def sim.DefStats) bool {
	if def.AssetClass == "airbase" || def.AssetClass == "port" {
		return true
	}
	switch def.TargetClass {
	case "runway", "hardened_infrastructure", "soft_infrastructure", "civilian_energy", "civilian_water", "sam_battery":
		return true
	default:
		return false
	}
}

func (a *App) teamHasCurrentDetection(teamID, targetUnitID string) bool {
	teamID = sim.CountryDisplayCode(teamID)
	if teamID == "" || strings.TrimSpace(targetUnitID) == "" {
		return false
	}
	a.lastDetMu.RLock()
	defer a.lastDetMu.RUnlock()
	for _, id := range a.lastDetections[teamID] {
		if id == targetUnitID {
			return true
		}
	}
	return false
}

func launchStateDeltas(u *enginev1.Unit, scen *enginev1.Scenario) []*enginev1.UnitDelta {
	if u == nil {
		return nil
	}
	deltas := []*enginev1.UnitDelta{{
		UnitId:                 u.GetId(),
		NextSortieReadySeconds: u.GetNextSortieReadySeconds(),
	}}
	hostBaseID := strings.TrimSpace(u.GetHostBaseId())
	if scen == nil || hostBaseID == "" {
		return deltas
	}
	if base := findScenarioUnit(scen.GetUnits(), hostBaseID); base != nil && base.GetBaseOps() != nil {
		deltas = append(deltas, &enginev1.UnitDelta{
			UnitId:  base.GetId(),
			BaseOps: base.GetBaseOps(),
		})
	}
	return deltas
}

func (a *App) validateAndConsumeLaunch(u *enginev1.Unit, def sim.DefStats) error {
	if u == nil || a.currentScenario == nil {
		return nil
	}
	if !shouldInitiateLaunch(u, def) {
		return nil
	}
	hostBaseID := strings.TrimSpace(u.GetHostBaseId())
	if hostBaseID == "" {
		return fmt.Errorf("%s has no host base assigned", u.GetDisplayName())
	}
	base := findScenarioUnit(a.currentScenario.GetUnits(), hostBaseID)
	if base == nil {
		return fmt.Errorf("host base %s not found", hostBaseID)
	}
	baseDef := a.getCachedDefs()[base.GetDefinitionId()]
	if base.GetBaseOps() == nil {
		return fmt.Errorf("host base %s has no operational state", base.GetDisplayName())
	}
	if base.GetBaseOps().GetState() == enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_CLOSED {
		return fmt.Errorf("%s is closed", base.GetDisplayName())
	}
	simSeconds := a.getSimSeconds()
	if u.GetNextSortieReadySeconds() > simSeconds {
		return fmt.Errorf("%s is not sortie-ready until T+%.0fs", u.GetDisplayName(), u.GetNextSortieReadySeconds())
	}
	if base.GetBaseOps().GetNextLaunchAvailableSeconds() > simSeconds {
		return fmt.Errorf("%s launch window unavailable until T+%.0fs", base.GetDisplayName(), base.GetBaseOps().GetNextLaunchAvailableSeconds())
	}
	syncUnitPositionToHostBase(u, base)
	u.Position.AltMsl = sim.DefaultAirborneAltitudeM(def)
	u.Position.Speed = 0
	spacingSeconds := 0.0
	if baseDef.LaunchCapacityPerInterval > 0 {
		spacingSeconds = 900.0 / float64(baseDef.LaunchCapacityPerInterval)
	}
	if spacingSeconds <= 0 {
		spacingSeconds = 60
	}
	if base.GetBaseOps().GetState() == enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_DEGRADED {
		spacingSeconds *= 2
	}
	base.BaseOps.NextLaunchAvailableSeconds = simSeconds + spacingSeconds
	sortieSeconds := 0.0
	if def.SortieIntervalMinutes > 0 {
		sortieSeconds = float64(def.SortieIntervalMinutes) * 60
	}
	if sortieSeconds <= 0 {
		sortieSeconds = 90 * 60
	}
	if base.GetBaseOps().GetState() == enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_DEGRADED {
		sortieSeconds *= 2
	}
	u.NextSortieReadySeconds = simSeconds + sortieSeconds
	return nil
}

func syncUnitPositionToHostBase(u, base *enginev1.Unit) {
	if u == nil || base == nil || u.GetPosition() == nil || base.GetPosition() == nil {
		return
	}
	u.Position.Lat = base.GetPosition().GetLat()
	u.Position.Lon = base.GetPosition().GetLon()
	u.Position.AltMsl = base.GetPosition().GetAltMsl()
}

func unitTravelAltitudeM(u *enginev1.Unit, def sim.DefStats) float64 {
	return sim.TravelAltitudeM(u, def)
}

func selectWeaponConfiguration(def library.Definition, preferredID string) (string, []library.LoadoutSlot) {
	preferredID = strings.TrimSpace(preferredID)
	defaultID := strings.TrimSpace(def.DefaultWeaponConfiguration)
	if preferredID != "" {
		for _, cfg := range def.WeaponConfigurations {
			if cfg.ID == preferredID {
				return cfg.ID, cfg.Loadout
			}
		}
	}
	if defaultID != "" {
		for _, cfg := range def.WeaponConfigurations {
			if cfg.ID == defaultID {
				return cfg.ID, cfg.Loadout
			}
		}
	}
	if len(def.WeaponConfigurations) > 0 {
		return def.WeaponConfigurations[0].ID, def.WeaponConfigurations[0].Loadout
	}
	return "", nil
}

// RequestSync re-emits the full state snapshot, scenario state, and last known detections.
func (a *App) RequestSync() BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	a.emitProtoEvent("full_state_snapshot", &enginev1.FullStateSnapshot{
		Units:             a.currentScenario.Units,
		SimTime:           &enginev1.SimTime{SecondsElapsed: a.getSimSeconds()},
		Weather:           a.currentScenario.GetMap().GetInitialWeather(),
		ScenarioName:      a.currentScenario.Name,
		WeaponDefinitions: a.listWeaponDefsProto(),
		Relationships:     a.currentScenario.GetRelationships(),
		Scores:            buildTeamScores(a.currentScenario.GetUnits(), a.getCachedDefs()),
	})
	a.emitProtoEvent("scenario_state", &enginev1.ScenarioStateEvent{
		State:     enginev1.ScenarioPlayState_SCENARIO_RUNNING,
		TimeScale: float32(a.getSimTimeScale()),
	})
	a.lastDetMu.RLock()
	for teamID, ids := range a.lastDetections {
		a.emitProtoEvent("detection_update", &enginev1.DetectionUpdate{
			DetectingTeam:   teamID,
			DetectedUnitIds: ids,
		})
	}
	a.lastDetMu.RUnlock()
	return ok()
}

// PauseSim pauses or resumes the simulation.
func (a *App) PauseSim(paused bool) BridgeResult {
	if paused {
		if a.simCancel != nil {
			a.simCancel()
			a.simCancel = nil
		}
	} else if a.simCancel == nil && a.currentScenario != nil {
		simCtx, simCancel := context.WithCancel(a.ctx)
		a.simCancel = simCancel
		go sim.MockLoop(
			simCtx,
			a.currentScenario.Units,
			a.getCachedDefs(),
			a.getCachedWeaponCatalog(),
			a.relationshipRules,
			a.getSimSeconds(),
			a.getSimTimeScale,
			a.setSimSeconds,
			a.planMajorActorStrikes,
			a.makeEmitFn(),
		)
	}

	state := enginev1.ScenarioPlayState_SCENARIO_PAUSED
	if !paused {
		state = enginev1.ScenarioPlayState_SCENARIO_RUNNING
	}
	a.emitProtoEvent("scenario_state", &enginev1.ScenarioStateEvent{
		State:     state,
		TimeScale: float32(a.getSimTimeScale()),
	})
	return ok()
}

// MoveUnit issues a movement order to a unit.
func (a *App) MoveUnit(unitID string, lat, lon float64) BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	defs := a.getCachedDefs()

	for _, u := range a.currentScenario.Units {
		if u.Id != unitID {
			continue
		}
		oldPos := u.GetPosition()
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			slog.Warn("MoveUnit: target coordinates out of range", "id", unitID, "lat", lat, "lon", lon)
			return failMsg("target coordinates out of range")
		}
		def := defs[u.DefinitionId]
		if def.CruiseSpeedMps <= 0 {
			return failMsg("unit is stationary and cannot be routed")
		}
		route := a.cachedBuildRoute(routing.Request{
			OwnerCountry:      unitCountryCode(u),
			Domain:            def.Domain,
			Purpose:           routing.PurposeMove,
			Start:             geo.Point{Lat: oldPos.GetLat(), Lon: oldPos.GetLon(), AltMsl: oldPos.GetAltMsl()},
			End:               geo.Point{Lat: lat, Lon: lon},
			RelationshipRules: a.relationshipRules(),
			CountryCoalitions: a.countryCoalitions(),
		})
		if route.Blocked {
			return failMsg(route.Reason)
		}
		if err := a.validateAndConsumeLaunch(u, def); err != nil {
			return failMsg(err.Error())
		}
		travelAlt := unitTravelAltitudeM(u, def)

		cruiseSpeed := def.CruiseSpeedMps
		if cruiseSpeed <= 0 {
			cruiseSpeed = 10
		}
		heading := sim.BearingDeg(oldPos.GetLat(), oldPos.GetLon(), lat, lon)
		newPos := &enginev1.Position{
			Lat:     oldPos.GetLat(),
			Lon:     oldPos.GetLon(),
			AltMsl:  travelAlt,
			Heading: heading,
			Speed:   cruiseSpeed,
		}
		newOrder := &enginev1.MoveOrder{AutoGenerated: false}
		for _, point := range route.Points {
			newOrder.Waypoints = append(newOrder.Waypoints, &enginev1.Waypoint{
				Lat: point.Lat, Lon: point.Lon, AltMsl: travelAlt,
			})
		}
		u.Position = newPos
		u.MoveOrder = newOrder

		a.emitProtoEvent("batch_update", &enginev1.BatchUnitUpdate{
			Deltas: append([]*enginev1.UnitDelta{{
				UnitId:                 unitID,
				Position:               newPos,
				MoveOrder:              newOrder,
				NextSortieReadySeconds: u.GetNextSortieReadySeconds(),
			}}, launchStateDeltas(u, a.currentScenario)[1:]...),
		})
		slog.Info("move order issued", "id", unitID, "dest_lat", lat, "dest_lon", lon, "speed_mps", cruiseSpeed, "heading", heading)
		return ok()
	}
	return failMsg("unit not found: " + unitID)
}

// CancelMoveOrder clears a unit's movement order, stopping it in place.
func (a *App) CancelMoveOrder(unitID string) BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	for _, u := range a.currentScenario.Units {
		if u.Id != unitID {
			continue
		}
		pos := u.GetPosition()
		stoppedPos := &enginev1.Position{
			Lat: pos.GetLat(), Lon: pos.GetLon(),
			AltMsl: pos.GetAltMsl(), Heading: pos.GetHeading(), Speed: 0,
		}
		u.Position = stoppedPos
		u.MoveOrder = nil
		a.emitProtoEvent("batch_update", &enginev1.BatchUnitUpdate{
			Deltas: []*enginev1.UnitDelta{{
				UnitId:    unitID,
				Position:  stoppedPos,
				MoveOrder: &enginev1.MoveOrder{},
			}},
		})
		slog.Info("move order cancelled", "id", unitID)
		return ok()
	}
	return failMsg("unit not found: " + unitID)
}

// ReturnUnitToBase clears attacking and issues an auto-generated route back to
// the unit's host base or carrier.
func (a *App) ReturnUnitToBase(unitID string) BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	defs := a.getCachedDefs()
	var unit *enginev1.Unit
	var host *enginev1.Unit
	for _, u := range a.currentScenario.Units {
		if u.Id == unitID {
			unit = u
		}
	}
	if unit == nil {
		return failMsg("unit not found: " + unitID)
	}
	if strings.TrimSpace(unit.GetHostBaseId()) == "" {
		return failMsg("unit has no host base")
	}
	for _, u := range a.currentScenario.Units {
		if u.Id == unit.GetHostBaseId() {
			host = u
			break
		}
	}
	if host == nil || host.GetPosition() == nil {
		return failMsg("host base not found")
	}
	pos := unit.GetPosition()
	if pos == nil {
		return failMsg("unit has no position")
	}
	def := defs[extractRecordID(unit.DefinitionId)]
	if def.CruiseSpeedMps <= 0 {
		return failMsg("unit is stationary and cannot return to base")
	}
	travelAlt := unitTravelAltitudeM(unit, def)
	route := a.cachedBuildRoute(routing.Request{
		OwnerCountry:      unitCountryCode(unit),
		Domain:            def.Domain,
		Purpose:           routing.PurposeMove,
		Start:             geo.Point{Lat: pos.GetLat(), Lon: pos.GetLon(), AltMsl: travelAlt},
		End:               geo.Point{Lat: host.GetPosition().GetLat(), Lon: host.GetPosition().GetLon(), AltMsl: 0},
		RelationshipRules: a.relationshipRules(),
		CountryCoalitions: a.countryCoalitions(),
	})
	if route.Blocked {
		return failMsg(route.Reason)
	}
	moveOrder := &enginev1.MoveOrder{AutoGenerated: true}
	for _, routePoint := range route.Points {
		alt := travelAlt
		if routePoint.Lat == host.GetPosition().GetLat() && routePoint.Lon == host.GetPosition().GetLon() {
			alt = 0
		}
		moveOrder.Waypoints = append(moveOrder.Waypoints, &enginev1.Waypoint{
			Lat: routePoint.Lat, Lon: routePoint.Lon, AltMsl: alt,
		})
	}
	unit.AttackOrder = nil
	unit.MoveOrder = moveOrder
	a.emitProtoEvent("batch_update", &enginev1.BatchUnitUpdate{
		Deltas: []*enginev1.UnitDelta{{
			UnitId:    unit.GetId(),
			MoveOrder: moveOrder,
		}},
	})
	return ok()
}

// AppendMoveWaypoint appends a waypoint to the unit's current route.
func (a *App) AppendMoveWaypoint(unitID string, lat, lon float64) BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return failMsg("target coordinates out of range")
	}
	defs := a.getCachedDefs()

	for _, u := range a.currentScenario.Units {
		if u.Id != unitID {
			continue
		}
		pos := u.GetPosition()
		if pos == nil {
			return failMsg("unit has no position")
		}
		startLat, startLon := pos.GetLat(), pos.GetLon()
		if current := u.GetMoveOrder(); current != nil && len(current.GetWaypoints()) > 0 {
			last := current.GetWaypoints()[len(current.GetWaypoints())-1]
			startLat, startLon = last.GetLat(), last.GetLon()
		}
		hadWaypoints := u.GetMoveOrder() != nil && len(u.GetMoveOrder().GetWaypoints()) > 0
		def := defs[u.DefinitionId]
		if def.CruiseSpeedMps <= 0 {
			return failMsg("unit is stationary and cannot be routed")
		}
		travelAlt := unitTravelAltitudeM(u, def)
		route := a.cachedBuildRoute(routing.Request{
			OwnerCountry:      unitCountryCode(u),
			Domain:            def.Domain,
			Purpose:           routing.PurposeMove,
			Start:             geo.Point{Lat: startLat, Lon: startLon, AltMsl: travelAlt},
			End:               geo.Point{Lat: lat, Lon: lon, AltMsl: travelAlt},
			RelationshipRules: a.relationshipRules(),
			CountryCoalitions: a.countryCoalitions(),
		})
		if route.Blocked {
			return failMsg(route.Reason)
		}
		if (u.GetMoveOrder() == nil || len(u.GetMoveOrder().GetWaypoints()) == 0) && shouldInitiateLaunch(u, def) {
			if err := a.validateAndConsumeLaunch(u, def); err != nil {
				return failMsg(err.Error())
			}
			travelAlt = unitTravelAltitudeM(u, def)
		}
		var newOrder *enginev1.MoveOrder
		if current := u.GetMoveOrder(); current != nil {
			newOrder = proto.Clone(current).(*enginev1.MoveOrder)
		} else {
			newOrder = &enginev1.MoveOrder{AutoGenerated: false}
		}
		newOrder.AutoGenerated = false
		for _, routePoint := range route.Points {
			newOrder.Waypoints = append(newOrder.Waypoints, &enginev1.Waypoint{
				Lat: routePoint.Lat, Lon: routePoint.Lon, AltMsl: travelAlt,
			})
		}
		u.MoveOrder = newOrder

		newPos := u.GetPosition()
		if !hadWaypoints && len(newOrder.Waypoints) > 0 {
			first := newOrder.Waypoints[0]
			cruiseSpeed := def.CruiseSpeedMps
			if cruiseSpeed <= 0 {
				cruiseSpeed = 10
			}
			heading := sim.BearingDeg(pos.GetLat(), pos.GetLon(), first.GetLat(), first.GetLon())
			newPos = &enginev1.Position{
				Lat:     pos.GetLat(),
				Lon:     pos.GetLon(),
				AltMsl:  travelAlt,
				Heading: heading,
				Speed:   cruiseSpeed,
			}
			u.Position = newPos
		}

		a.emitProtoEvent("batch_update", &enginev1.BatchUnitUpdate{
			Deltas: append([]*enginev1.UnitDelta{{
				UnitId:                 unitID,
				Position:               newPos,
				MoveOrder:              newOrder,
				NextSortieReadySeconds: u.GetNextSortieReadySeconds(),
			}}, launchStateDeltas(u, a.currentScenario)[1:]...),
		})
		return ok()
	}
	return failMsg("unit not found: " + unitID)
}

// RemoveMoveWaypoint removes a waypoint from the unit's current route by index.
func (a *App) RemoveMoveWaypoint(unitID string, waypointIndex int) BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	for _, u := range a.currentScenario.Units {
		if u.Id != unitID {
			continue
		}
		order := u.GetMoveOrder()
		if order == nil || waypointIndex < 0 || waypointIndex >= len(order.Waypoints) {
			return failMsg("waypoint not found")
		}
		newWaypoints := append([]*enginev1.Waypoint{}, order.Waypoints[:waypointIndex]...)
		newWaypoints = append(newWaypoints, order.Waypoints[waypointIndex+1:]...)

		var nextOrder *enginev1.MoveOrder
		if len(newWaypoints) > 0 {
			nextOrder = &enginev1.MoveOrder{Waypoints: newWaypoints, AutoGenerated: false}
		} else {
			nextOrder = &enginev1.MoveOrder{AutoGenerated: false}
		}
		u.MoveOrder = nextOrder

		a.emitProtoEvent("batch_update", &enginev1.BatchUnitUpdate{
			Deltas: []*enginev1.UnitDelta{{
				UnitId:    unitID,
				MoveOrder: nextOrder,
			}},
		})
		return ok()
	}
	return failMsg("unit not found: " + unitID)
}

// UpdateMoveWaypoint updates the coordinates of an existing waypoint by index.
func (a *App) UpdateMoveWaypoint(unitID string, waypointIndex int, lat, lon float64) BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return failMsg("target coordinates out of range")
	}
	for _, u := range a.currentScenario.Units {
		if u.Id != unitID {
			continue
		}
		order := u.GetMoveOrder()
		if order == nil || waypointIndex < 0 || waypointIndex >= len(order.Waypoints) {
			return failMsg("waypoint not found")
		}
		var startLat, startLon float64
		if waypointIndex == 0 {
			startLat, startLon = u.GetPosition().GetLat(), u.GetPosition().GetLon()
		} else {
			prev := order.Waypoints[waypointIndex-1]
			startLat, startLon = prev.GetLat(), prev.GetLon()
		}
		def := a.getCachedDefs()[u.DefinitionId]
		if def.CruiseSpeedMps <= 0 {
			return failMsg("unit is stationary and cannot be routed")
		}
		travelAlt := unitTravelAltitudeM(u, def)
		firstLeg := a.cachedBuildRoute(routing.Request{
			OwnerCountry:      unitCountryCode(u),
			Domain:            def.Domain,
			Purpose:           routing.PurposeMove,
			Start:             geo.Point{Lat: startLat, Lon: startLon, AltMsl: travelAlt},
			End:               geo.Point{Lat: lat, Lon: lon, AltMsl: travelAlt},
			RelationshipRules: a.relationshipRules(),
			CountryCoalitions: a.countryCoalitions(),
		})
		if firstLeg.Blocked {
			return failMsg(firstLeg.Reason)
		}
		secondPoints := []geo.Point{}
		if waypointIndex < len(order.Waypoints)-1 {
			next := order.Waypoints[waypointIndex+1]
			secondLeg := a.cachedBuildRoute(routing.Request{
				OwnerCountry:      unitCountryCode(u),
				Domain:            def.Domain,
				Purpose:           routing.PurposeMove,
				Start:             geo.Point{Lat: lat, Lon: lon, AltMsl: travelAlt},
				End:               geo.Point{Lat: next.GetLat(), Lon: next.GetLon(), AltMsl: next.GetAltMsl()},
				RelationshipRules: a.relationshipRules(),
				CountryCoalitions: a.countryCoalitions(),
			})
			if secondLeg.Blocked {
				return failMsg(secondLeg.Reason)
			}
			secondPoints = secondLeg.Points
		}
		newOrder := &enginev1.MoveOrder{AutoGenerated: false}
		newOrder.Waypoints = append(newOrder.Waypoints, order.GetWaypoints()[:waypointIndex]...)
		for _, point := range firstLeg.Points {
			newOrder.Waypoints = append(newOrder.Waypoints, &enginev1.Waypoint{
				Lat: point.Lat, Lon: point.Lon, AltMsl: travelAlt,
			})
		}
		for _, point := range secondPoints {
			newOrder.Waypoints = append(newOrder.Waypoints, &enginev1.Waypoint{
				Lat: point.Lat, Lon: point.Lon, AltMsl: travelAlt,
			})
		}
		if waypointIndex+2 <= len(order.GetWaypoints()) {
			newOrder.Waypoints = append(newOrder.Waypoints, order.GetWaypoints()[waypointIndex+2:]...)
		}
		u.MoveOrder = newOrder
		a.emitProtoEvent("batch_update", &enginev1.BatchUnitUpdate{
			Deltas: []*enginev1.UnitDelta{{
				UnitId:    unitID,
				MoveOrder: newOrder,
			}},
		})
		return ok()
	}
	return failMsg("unit not found: " + unitID)
}

// SetUnitEngagement updates the live engagement policy on a unit.
func (a *App) SetUnitEngagement(unitID string, behavior int32, pkillThreshold float32) BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	for _, u := range a.currentScenario.Units {
		if u.Id != unitID {
			continue
		}
		u.EngagementBehavior = enginev1.EngagementBehavior(behavior)
		u.EngagementPkillThreshold = pkillThreshold
		return ok()
	}
	return failMsg("unit not found: " + unitID)
}

// SetUnitAttackOrder updates or clears the live attack order on a unit.
func (a *App) SetUnitAttackOrder(unitID string, orderType int32, targetUnitID string, desiredEffect int32, pkillThreshold float32) BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	defs := a.getCachedDefs()
	var shooter *enginev1.Unit
	var target *enginev1.Unit
	for _, u := range a.currentScenario.Units {
		if u.Id == unitID {
			shooter = u
		}
		if u.Id == targetUnitID {
			target = u
		}
	}
	for _, u := range a.currentScenario.Units {
		if u.Id != unitID {
			continue
		}
		if orderType == 0 || strings.TrimSpace(targetUnitID) == "" {
			var moveOrderDelta *enginev1.MoveOrder
			u.AttackOrder = nil
			if current := u.GetMoveOrder(); current != nil && current.GetAutoGenerated() {
				u.MoveOrder = nil
				moveOrderDelta = &enginev1.MoveOrder{}
			}
			a.emitProtoEvent("batch_update", &enginev1.BatchUnitUpdate{
				Deltas: []*enginev1.UnitDelta{{
					UnitId:    u.GetId(),
					MoveOrder: moveOrderDelta,
				}},
			})
			return ok()
		}
		if target == nil {
			return failMsg("target unit not found: " + targetUnitID)
		}
		targetDef := defs[extractRecordID(target.DefinitionId)]
		if !a.teamHasCurrentDetection(u.GetTeamId(), targetUnitID) && !isPreplannedFixedTarget(targetDef) {
			return failMsg("mobile targets require a current track before assignment")
		}
		if isMaritimeDomain(defs[extractRecordID(u.DefinitionId)].Domain) && geo.IsLandPoint(geo.Point{
			Lat: target.GetPosition().GetLat(),
			Lon: target.GetPosition().GetLon(),
		}) {
			return failMsg("naval units cannot attack land targets")
		}
		weaponID := ""
		options, err := a.buildTargetEngagementOptions(
			target,
			sim.CountryDisplayCode(u.GetTeamId()),
			enginev1.DesiredEffect(desiredEffect),
			enginev1.AttackOrderType(orderType) == enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT,
		)
		if err != nil {
			return fail(err)
		}
		foundShooter := false
		for _, option := range options {
			if option.ShooterUnitId != u.GetId() {
				continue
			}
			foundShooter = true
			if option.ReadyToFire || option.CanAssign {
				weaponID = option.WeaponId
				break
			}
			if option.Reason != "" {
				return failMsg(option.Reason)
			}
			break
		}
		if !foundShooter {
			return failMsg("unit is not a friendly shooter for this target")
		}
		if err := a.validateStrikeWithWeapon(shooter, target, weaponID); err != nil {
			return failMsg(err.Error())
		}
		shooterDef := defs[extractRecordID(u.DefinitionId)]
		if current := u.GetMoveOrder(); current == nil || len(current.GetWaypoints()) == 0 {
			if err := a.validateAndConsumeLaunch(u, shooterDef); err != nil {
				return failMsg(err.Error())
			}
		}
		u.AttackOrder = &enginev1.AttackOrder{
			OrderType:      enginev1.AttackOrderType(orderType),
			TargetUnitId:   targetUnitID,
			DesiredEffect:  enginev1.DesiredEffect(desiredEffect),
			PkillThreshold: pkillThreshold,
		}
		if current := u.GetMoveOrder(); current == nil || len(current.GetWaypoints()) == 0 || current.GetAutoGenerated() {
			if waypoint := sim.ComputeAttackWaypointForOrder(u, target, a.getCachedDefs(), a.getCachedWeaponCatalog()); waypoint != nil {
				waypoints := []*enginev1.Waypoint{waypoint}
				purpose := routing.PurposeStrike
				if weapon, ok := weaponStatsForID(a.getCachedWeaponCatalog(), weaponID); ok {
					purpose = routePurposeForStrikeMission(targetDef, weapon)
				}
				route := a.cachedBuildRoute(routing.Request{
					OwnerCountry:      unitCountryCode(u),
					Domain:            shooterDef.Domain,
					Purpose:           purpose,
					Start:             geo.Point{Lat: u.GetPosition().GetLat(), Lon: u.GetPosition().GetLon(), AltMsl: u.GetPosition().GetAltMsl()},
					End:               geo.Point{Lat: waypoint.GetLat(), Lon: waypoint.GetLon(), AltMsl: waypoint.GetAltMsl()},
					RelationshipRules: a.relationshipRules(),
					CountryCoalitions: a.countryCoalitions(),
				})
				if route.Blocked {
					return failMsg(route.Reason)
				}
				if len(route.Points) > 0 {
					waypoints = make([]*enginev1.Waypoint, 0, len(route.Points))
					for _, routePoint := range route.Points {
						waypoints = append(waypoints, &enginev1.Waypoint{
							Lat: routePoint.Lat, Lon: routePoint.Lon, AltMsl: waypoint.GetAltMsl(),
						})
					}
				}
				u.MoveOrder = &enginev1.MoveOrder{
					Waypoints:     waypoints,
					AutoGenerated: true,
				}
			}
		}
		a.emitProtoEvent("batch_update", &enginev1.BatchUnitUpdate{
			Deltas: append([]*enginev1.UnitDelta{{
				UnitId:                 u.GetId(),
				MoveOrder:              u.GetMoveOrder(),
				NextSortieReadySeconds: u.GetNextSortieReadySeconds(),
			}}, launchStateDeltas(u, a.currentScenario)[1:]...),
		})
		return ok()
	}
	return failMsg("unit not found: " + unitID)
}

func (a *App) SetUnitLoadoutConfiguration(unitID string, loadoutConfigurationID string) BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	loadoutConfigurationID = strings.TrimSpace(loadoutConfigurationID)
	defs := a.getCachedDefs()
	for _, u := range a.currentScenario.Units {
		if u.GetId() != unitID {
			continue
		}
		if defs[u.GetDefinitionId()].Domain != enginev1.UnitDomain_DOMAIN_AIR {
			return failMsg("only aircraft can change mission loadout")
		}
		if strings.TrimSpace(u.GetHostBaseId()) == "" {
			return failMsg("unit has no host base")
		}
		if u.GetPosition() != nil && u.GetPosition().GetAltMsl() > 0 {
			return failMsg("aircraft must be grounded to change loadout")
		}
		host := findScenarioUnit(a.currentScenario.Units, u.GetHostBaseId())
		if host == nil {
			return failMsg("host base not found")
		}
		if host.GetBaseOps() != nil && host.GetBaseOps().GetState() == enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_CLOSED {
			return failMsg("host base is closed")
		}
		def, found := a.libDefsCache[extractRecordID(u.GetDefinitionId())]
		if !found {
			return failMsg("unit definition not found")
		}
		selectedID, slots := selectWeaponConfiguration(def, loadoutConfigurationID)
		if len(slots) == 0 {
			return failMsg("selected loadout has no weapons")
		}
		u.LoadoutConfigurationId = selectedID
		u.Weapons = loadoutToWeaponStates(slots)
		return ok()
	}
	return failMsg("unit not found: " + unitID)
}

// SetIntelSharing updates the live scenario relationship matrix for one directed
// country-to-country intel-sharing edge.
func (a *App) SetIntelSharing(fromCountry, toCountry string, enabled bool) BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	fromCountry = sim.CountryDisplayCode(fromCountry)
	toCountry = sim.CountryDisplayCode(toCountry)
	if fromCountry == "" || toCountry == "" {
		return failMsg("fromCountry and toCountry are required")
	}
	if fromCountry == toCountry {
		return ok()
	}
	rule, _ := a.getOrCreateRelationship(fromCountry, toCountry)
	rule.ShareIntel = enabled
	if a.scenRepo != nil {
		if err := a.scenRepo.Save(a.ctx, a.currentScenario.Id, scenarioRecord(a.currentScenario)); err != nil {
			slog.Warn("save scenario relationship update", "err", err)
		}
	}
	return ok()
}

func (a *App) getOrCreateRelationship(fromCountry, toCountry string) (*enginev1.CountryRelationship, BridgeResult) {
	if a.currentScenario == nil {
		return nil, failMsg("no scenario loaded")
	}
	fromCountry = sim.CountryDisplayCode(fromCountry)
	toCountry = sim.CountryDisplayCode(toCountry)
	if fromCountry == "" || toCountry == "" {
		return nil, failMsg("fromCountry and toCountry are required")
	}
	if fromCountry == toCountry {
		return nil, ok()
	}
	var rel *enginev1.CountryRelationship
	for _, candidate := range a.currentScenario.GetRelationships() {
		if sim.CountryDisplayCode(candidate.GetFromCountry()) == fromCountry &&
			sim.CountryDisplayCode(candidate.GetToCountry()) == toCountry {
			rel = candidate
			break
		}
	}
	if rel == nil {
		rel = &enginev1.CountryRelationship{
			FromCountry:                 fromCountry,
			ToCountry:                   toCountry,
			AirspaceTransitAllowed:      false,
			AirspaceStrikeAllowed:       false,
			DefensivePositioningAllowed: false,
			MaritimeTransitAllowed:      false,
			MaritimeStrikeAllowed:       false,
		}
		a.currentScenario.Relationships = append(a.currentScenario.Relationships, rel)
	}
	return rel, ok()
}

// SetCountryRelationship updates the live scenario relationship matrix for one
// directed country-to-country rule set.
func (a *App) SetCountryRelationship(
	fromCountry string,
	toCountry string,
	shareIntel bool,
	airspaceTransitAllowed bool,
	airspaceStrikeAllowed bool,
	defensivePositioningAllowed bool,
	maritimeTransitAllowed bool,
	maritimeStrikeAllowed bool,
) BridgeResult {
	rel, result := a.getOrCreateRelationship(fromCountry, toCountry)
	if !result.Success || rel == nil {
		return result
	}
	rel.ShareIntel = shareIntel
	rel.AirspaceTransitAllowed = airspaceTransitAllowed
	rel.AirspaceStrikeAllowed = airspaceStrikeAllowed
	rel.DefensivePositioningAllowed = defensivePositioningAllowed
	rel.MaritimeTransitAllowed = maritimeTransitAllowed
	rel.MaritimeStrikeAllowed = maritimeStrikeAllowed
	a.invalidateRouteCache()
	if a.scenRepo != nil {
		if err := a.scenRepo.Save(a.ctx, a.currentScenario.Id, scenarioRecord(a.currentScenario)); err != nil {
			slog.Warn("save scenario relationship update", "err", err)
		}
	}
	return ok()
}

// buildDefs queries all unit definitions and returns a map for the sim engine.
func (a *App) buildDefs() map[string]sim.DefStats {
	defs := make(map[string]sim.DefStats, len(a.libDefsCache))
	for id, def := range a.libDefsCache {
		defs[id] = defStatsFromLibraryDefinition(def)
	}

	if a.unitDefRepo == nil {
		return defs
	}
	rows, err := a.unitDefRepo.List(a.ctx)
	if err != nil {
		slog.Warn("buildDefs: list definitions", "err", err)
		return defs
	}
	for _, row := range rows {
		id := extractRecordID(row["id"])
		defs[id] = mergeDefStatsWithRow(defs[id], row)
	}
	return defs
}

func defStatsFromLibraryDefinition(def library.Definition) sim.DefStats {
	domain := enginev1.UnitDomain(def.Domain)
	generalType := int32(def.GeneralType)
	rcs := float64(def.RadarCrossSectionM2)
	if rcs <= 0 {
		rcs = defaultRadarCrossSectionM2(domain, generalType)
	}
	return sim.DefStats{
		CruiseSpeedMps:              float64(def.CruiseSpeedMps),
		BaseStrength:                float64(def.BaseStrength),
		Accuracy:                    float64(def.Accuracy),
		DetectionRangeM:             float64(def.DetectionRangeM),
		RadarCrossSectionM2:         rcs,
		FuelCapacityLiters:          float64(def.FuelCapacityLiters),
		FuelBurnRateLph:             float64(def.FuelBurnRateLph),
		SensorSuite:                 sensorSuiteFromRecord(def.ToRecord()["sensor_suite"]),
		GeneralType:                 int32(def.GeneralType),
		EmploymentRole:              strings.TrimSpace(def.EmploymentRole),
		AuthorizedPersonnel:         maxInt(def.AuthorizedPersonnel, library.DefaultAuthorizedPersonnel(def.AssetClass, def.Domain, def.GeneralType)),
		ReplacementCostUSD:          defaultIfZero(def.ReplacementCostUSD, library.DefaultReplacementCostUSD(def.AssetClass, def.Domain, def.GeneralType)),
		StrategicValueUSD:           defaultIfZero(def.StrategicValueUSD, library.DefaultStrategicValueUSD(def.AssetClass, def.TargetClass, def.Domain, def.GeneralType, def.EmploymentRole)),
		EconomicValueUSD:            defaultIfZero(def.EconomicValueUSD, library.DefaultEconomicValueUSD(def.AssetClass, def.Affiliation)),
		Domain:                      domain,
		TargetClass:                 strings.TrimSpace(def.TargetClass),
		AssetClass:                  strings.TrimSpace(def.AssetClass),
		EmbarkedFixedWingCapacity:   def.EmbarkedFixedWingCapacity,
		EmbarkedRotaryWingCapacity:  def.EmbarkedRotaryWingCapacity,
		EmbarkedUAVCapacity:         def.EmbarkedUavCapacity,
		LaunchCapacityPerInterval:   def.LaunchCapacityPerInterval,
		RecoveryCapacityPerInterval: def.RecoveryCapacityPerInterval,
		SortieIntervalMinutes:       def.SortieIntervalMinutes,
	}
}

func mergeDefStatsWithRow(base sim.DefStats, row map[string]any) sim.DefStats {
	assetClass := toString(row["asset_class"])
	if assetClass == "" {
		assetClass = base.AssetClass
	}
	targetClass := toString(row["target_class"])
	if targetClass == "" {
		targetClass = base.TargetClass
	}
	employmentRole := toString(row["employment_role"])
	if employmentRole == "" {
		employmentRole = base.EmploymentRole
	}
	domain := enginev1.UnitDomain(int32(toFloat64(row["domain"])))
	if domain == enginev1.UnitDomain_DOMAIN_UNSPECIFIED {
		domain = base.Domain
	}
	generalType := int32(toFloat64(row["general_type"]))
	if generalType == 0 {
		generalType = base.GeneralType
	}
	rcs := toFloat64(row["radar_cross_section_m2"])
	if rcs <= 0 {
		rcs = base.RadarCrossSectionM2
	}
	if rcs <= 0 {
		rcs = defaultRadarCrossSectionM2(domain, generalType)
	}
	sensorSuite := sensorSuiteFromRecord(row["sensor_suite"])
	if len(sensorSuite) == 0 {
		sensorSuite = base.SensorSuite
	}
	cruiseSpeed := toFloat64(row["cruise_speed_mps"])
	if cruiseSpeed <= 0 {
		cruiseSpeed = base.CruiseSpeedMps
	}
	baseStrength := toFloat64(row["base_strength"])
	if baseStrength <= 0 {
		baseStrength = base.BaseStrength
	}
	detectionRange := toFloat64(row["detection_range_m"])
	if detectionRange <= 0 {
		detectionRange = base.DetectionRangeM
	}
	fuelCapacity := toFloat64(row["fuel_capacity_liters"])
	if fuelCapacity <= 0 {
		fuelCapacity = base.FuelCapacityLiters
	}
	fuelBurnRate := toFloat64(row["fuel_burn_rate_lph"])
	if fuelBurnRate <= 0 {
		fuelBurnRate = base.FuelBurnRateLph
	}
	authorizedPersonnel := maxInt(int(toFloat64(row["authorized_personnel"])), base.AuthorizedPersonnel)
	if authorizedPersonnel <= 0 {
		authorizedPersonnel = library.DefaultAuthorizedPersonnel(assetClass, int(domain), int(generalType))
	}
	replacementCost := defaultIfZero(toFloat64(row["replacement_cost_usd"]), base.ReplacementCostUSD)
	if replacementCost <= 0 {
		replacementCost = library.DefaultReplacementCostUSD(assetClass, int(domain), int(generalType))
	}
	strategicValue := defaultIfZero(toFloat64(row["strategic_value_usd"]), base.StrategicValueUSD)
	if strategicValue <= 0 {
		strategicValue = library.DefaultStrategicValueUSD(assetClass, targetClass, int(domain), int(generalType), employmentRole)
	}
	economicValue := defaultIfZero(toFloat64(row["economic_value_usd"]), base.EconomicValueUSD)
	if economicValue <= 0 {
		economicValue = library.DefaultEconomicValueUSD(assetClass, toString(row["affiliation"]))
	}
	embarkedFixedWing := int(toFloat64(row["embarked_fixed_wing_capacity"]))
	if embarkedFixedWing == 0 {
		embarkedFixedWing = base.EmbarkedFixedWingCapacity
	}
	embarkedRotaryWing := int(toFloat64(row["embarked_rotary_wing_capacity"]))
	if embarkedRotaryWing == 0 {
		embarkedRotaryWing = base.EmbarkedRotaryWingCapacity
	}
	embarkedUAV := int(toFloat64(row["embarked_uav_capacity"]))
	if embarkedUAV == 0 {
		embarkedUAV = base.EmbarkedUAVCapacity
	}
	launchCapacity := int(toFloat64(row["launch_capacity_per_interval"]))
	if launchCapacity == 0 {
		launchCapacity = base.LaunchCapacityPerInterval
	}
	recoveryCapacity := int(toFloat64(row["recovery_capacity_per_interval"]))
	if recoveryCapacity == 0 {
		recoveryCapacity = base.RecoveryCapacityPerInterval
	}
	sortieInterval := int(toFloat64(row["sortie_interval_minutes"]))
	if sortieInterval == 0 {
		sortieInterval = base.SortieIntervalMinutes
	}
	return sim.DefStats{
		CruiseSpeedMps:              cruiseSpeed,
		BaseStrength:                baseStrength,
		Accuracy:                    defaultIfZero(toFloat64(row["accuracy"]), base.Accuracy),
		DetectionRangeM:             detectionRange,
		RadarCrossSectionM2:         rcs,
		FuelCapacityLiters:          fuelCapacity,
		FuelBurnRateLph:             fuelBurnRate,
		SensorSuite:                 sensorSuite,
		GeneralType:                 generalType,
		EmploymentRole:              employmentRole,
		AuthorizedPersonnel:         authorizedPersonnel,
		ReplacementCostUSD:          replacementCost,
		StrategicValueUSD:           strategicValue,
		EconomicValueUSD:            economicValue,
		Domain:                      domain,
		TargetClass:                 targetClass,
		AssetClass:                  assetClass,
		EmbarkedFixedWingCapacity:   embarkedFixedWing,
		EmbarkedRotaryWingCapacity:  embarkedRotaryWing,
		EmbarkedUAVCapacity:         embarkedUAV,
		LaunchCapacityPerInterval:   launchCapacity,
		RecoveryCapacityPerInterval: recoveryCapacity,
		SortieIntervalMinutes:       sortieInterval,
	}
}

func sensorSuiteFromRecord(raw any) []sim.SensorCapability {
	switch values := raw.(type) {
	case []map[string]any:
		sensors := make([]sim.SensorCapability, 0, len(values))
		for _, value := range values {
			if sensor, ok := sensorCapabilityFromMap(value); ok {
				sensors = append(sensors, sensor)
			}
		}
		return sensors
	case []any:
		sensors := make([]sim.SensorCapability, 0, len(values))
		for _, value := range values {
			row, ok := value.(map[string]any)
			if !ok {
				continue
			}
			if sensor, ok := sensorCapabilityFromMap(row); ok {
				sensors = append(sensors, sensor)
			}
		}
		return sensors
	default:
		return nil
	}
}

func sensorCapabilityFromMap(row map[string]any) (sim.SensorCapability, bool) {
	sensorType := parseSensorType(toString(row["sensor_type"]))
	maxRange := toFloat64(row["max_range_m"])
	targetStates := parseSensorTargetStates(row["target_states"])
	if sensorType == enginev1.SensorType_SENSOR_TYPE_UNSPECIFIED || maxRange <= 0 || len(targetStates) == 0 {
		return sim.SensorCapability{}, false
	}
	return sim.SensorCapability{
		SensorType:   sensorType,
		MaxRangeM:    maxRange,
		TargetStates: targetStates,
		FireControl:  toBool(row["fire_control"]),
	}, true
}

func parseSensorType(v string) enginev1.SensorType {
	switch strings.TrimSpace(v) {
	case "air_search":
		return enginev1.SensorType_SENSOR_TYPE_AIR_SEARCH
	case "surface_search":
		return enginev1.SensorType_SENSOR_TYPE_SURFACE_SEARCH
	case "ground_search":
		return enginev1.SensorType_SENSOR_TYPE_GROUND_SEARCH
	case "sonar":
		return enginev1.SensorType_SENSOR_TYPE_SONAR
	case "eo_ir":
		return enginev1.SensorType_SENSOR_TYPE_EO_IR
	case "passive_esm":
		return enginev1.SensorType_SENSOR_TYPE_PASSIVE_ESM
	case "visual":
		return enginev1.SensorType_SENSOR_TYPE_VISUAL
	default:
		return enginev1.SensorType_SENSOR_TYPE_UNSPECIFIED
	}
}

func parseSensorTargetStates(raw any) []enginev1.SensorTargetState {
	values, ok := raw.([]any)
	if !ok {
		if stringsValues, ok := raw.([]string); ok {
			values = make([]any, len(stringsValues))
			for i, value := range stringsValues {
				values[i] = value
			}
		} else {
			return nil
		}
	}
	states := make([]enginev1.SensorTargetState, 0, len(values))
	for _, value := range values {
		switch strings.TrimSpace(toString(value)) {
		case "airborne":
			states = append(states, enginev1.SensorTargetState_SENSOR_TARGET_STATE_AIRBORNE)
		case "land":
			states = append(states, enginev1.SensorTargetState_SENSOR_TARGET_STATE_LAND)
		case "surface":
			states = append(states, enginev1.SensorTargetState_SENSOR_TARGET_STATE_SURFACE)
		case "submerged":
			states = append(states, enginev1.SensorTargetState_SENSOR_TARGET_STATE_SUBMERGED)
		}
	}
	return states
}

func toBool(v any) bool {
	switch value := v.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true")
	case float64:
		return value != 0
	case int:
		return value != 0
	default:
		return false
	}
}

func defaultRadarCrossSectionM2(domain enginev1.UnitDomain, generalType int32) float64 {
	switch domain {
	case enginev1.UnitDomain_DOMAIN_AIR:
		switch generalType {
		case int32(enginev1.UnitGeneralType_GENERAL_TYPE_BOMBER):
			return 20
		case int32(enginev1.UnitGeneralType_GENERAL_TYPE_TRANSPORT_AIRCRAFT),
			int32(enginev1.UnitGeneralType_GENERAL_TYPE_MARITIME_PATROL),
			int32(enginev1.UnitGeneralType_GENERAL_TYPE_AEW),
			int32(enginev1.UnitGeneralType_GENERAL_TYPE_TANKER):
			return 25
		case int32(enginev1.UnitGeneralType_GENERAL_TYPE_ISR_UAV),
			int32(enginev1.UnitGeneralType_GENERAL_TYPE_UCAV),
			int32(enginev1.UnitGeneralType_GENERAL_TYPE_LOITERING_MUNITION):
			return 0.5
		default:
			return 5
		}
	case enginev1.UnitDomain_DOMAIN_SEA:
		return 1_000
	case enginev1.UnitDomain_DOMAIN_SUBSURFACE:
		return 0.2
	case enginev1.UnitDomain_DOMAIN_LAND:
		switch generalType {
		case int32(enginev1.UnitGeneralType_GENERAL_TYPE_LIGHT_INFANTRY),
			int32(enginev1.UnitGeneralType_GENERAL_TYPE_AIRBORNE_INFANTRY),
			int32(enginev1.UnitGeneralType_GENERAL_TYPE_MARINE_INFANTRY),
			int32(enginev1.UnitGeneralType_GENERAL_TYPE_SPECIAL_FORCES):
			return 0.5
		case int32(enginev1.UnitGeneralType_GENERAL_TYPE_MAIN_BATTLE_TANK),
			int32(enginev1.UnitGeneralType_GENERAL_TYPE_INFANTRY_FIGHTING_VEHICLE),
			int32(enginev1.UnitGeneralType_GENERAL_TYPE_ARMORED_PERSONNEL_CARRIER),
			int32(enginev1.UnitGeneralType_GENERAL_TYPE_RECONNAISSANCE_VEHICLE):
			return 10
		default:
			return 5
		}
	default:
		return 1
	}
}

// loadoutToWeaponStates converts library.LoadoutSlot entries into proto weapon state.
func loadoutToWeaponStates(slots []library.LoadoutSlot) []*enginev1.WeaponState {
	states := make([]*enginev1.WeaponState, 0, len(slots))
	for _, s := range slots {
		states = append(states, &enginev1.WeaponState{
			WeaponId:   s.WeaponID,
			CurrentQty: s.InitialQty,
			MaxQty:     s.MaxQty,
		})
	}
	return states
}

// buildWeaponCatalog converts stored weapon definitions into sim stats.
func (a *App) buildWeaponCatalog() map[string]sim.WeaponStats {
	catalog := make(map[string]sim.WeaponStats)
	for _, wd := range a.listWeaponDefsProto() {
		catalog[wd.Id] = sim.WeaponStats{
			RangeM:           float64(wd.RangeM),
			SpeedMps:         float64(wd.SpeedMps),
			ProbabilityOfHit: float64(wd.ProbabilityOfHit),
			DomainTargets:    wd.DomainTargets,
			Guidance:         wd.Guidance,
			EffectType:       wd.EffectType,
		}
	}
	return catalog
}

// SetSimSpeed sets the simulation time scale multiplier.
func (a *App) SetSimSpeed(timeScale float32) BridgeResult {
	if timeScale <= 0 || timeScale > 3600 {
		return failMsg("timeScale must be between 0 and 3600")
	}
	a.setSimTimeScale(float64(timeScale))
	a.emitProtoEvent("scenario_state", &enginev1.ScenarioStateEvent{
		State:     enginev1.ScenarioPlayState_SCENARIO_RUNNING,
		TimeScale: timeScale,
	})
	return ok()
}

func (a *App) relationshipRules() sim.RelationshipRules {
	if a.currentScenario == nil {
		return nil
	}
	return sim.BuildRelationshipRules(a.currentScenario.GetRelationships())
}

func (a *App) countryCoalitions() map[string]string {
	if a.currentScenario == nil {
		return nil
	}
	return sim.BuildCountryCoalitions(a.currentScenario.GetUnits())
}

func unitCountryCode(u *enginev1.Unit) string {
	return sim.CountryDisplayCode(u.GetTeamId())
}
