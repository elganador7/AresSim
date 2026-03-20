package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/aressim/internal/db"
	"github.com/aressim/internal/db/repository"
	"github.com/aressim/internal/library"
	"github.com/aressim/internal/scenario"
	"github.com/aressim/internal/sim"
	"github.com/surrealdb/surrealdb.go/pkg/models"
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

// invalidateDefsCache clears the DefStats cache so the next call to
// getCachedDefs re-queries the database.
func (a *App) invalidateDefsCache() {
	a.defsCacheMu.Lock()
	a.defsCache = nil
	a.defsCacheMu.Unlock()
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
		pos := u.GetPosition()
		units = append(units, repository.UnitRecord{
			"id": models.RecordID{Table: "unit", ID: u.Id},
			"position": map[string]any{
				"type":        "Point",
				"coordinates": []float64{pos.GetLon(), pos.GetLat()},
			},
			"alt_msl": pos.GetAltMsl(),
			"heading": pos.GetHeading(),
			"speed":   pos.GetSpeed(),
		})
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
	defs := a.getCachedDefs()
	for _, u := range scen.Units {
		ensureUnitOpsState(u, defs[u.DefinitionId])
	}

	unitDefs, _ := a.unitDefRepo.List(a.ctx)
	generalTypeByDefID := make(map[string]int32, len(unitDefs))
	for _, row := range unitDefs {
		id := extractRecordID(row["id"])
		if gt, ok := row["general_type"]; ok {
			generalTypeByDefID[id] = int32(toFloat64(gt))
		}
	}

	applyOpeningLoadoutSelections(scen)
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
				slog.Info("weapon loadout from library", "unit", u.DisplayName, "def", defID, "config", loadoutID, "weapons", len(u.Weapons))
				continue
			}
		}
		if def, ok := a.libDefsCache[defID]; ok && len(def.DefaultLoadout) > 0 {
			u.LoadoutConfigurationId = "default"
			u.Weapons = loadoutToWeaponStates(def.DefaultLoadout)
			slog.Info("weapon loadout from legacy default", "unit", u.DisplayName, "def", defID, "weapons", len(u.Weapons))
			continue
		}
		gt := generalTypeByDefID[defID]
		u.Weapons = scenario.InitUnitWeapons(u, gt)
		slog.Info("weapon loadout from defaults", "unit", u.DisplayName, "def", defID, "general_type", gt, "weapons", len(u.Weapons))
	}
	a.applyOpeningStrikeActions(scen, defs)

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
		a.buildWeaponCatalog(),
		a.relationshipRules,
		0,
		a.getSimTimeScale,
		a.setSimSeconds,
		a.planMajorActorStrikes,
		a.makeEmitFn(),
	)
	slog.Info("scenario loaded", "name", scen.Name, "units", len(scen.Units))
}

func ensureUnitOpsState(u *enginev1.Unit, def sim.DefStats) {
	if u == nil {
		return
	}
	if u.BaseOps == nil && def.AssetClass == "airbase" {
		u.BaseOps = &enginev1.BaseOpsState{
			State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE,
		}
	}
	if u.NextSortieReadySeconds < 0 {
		u.NextSortieReadySeconds = 0
	}
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

func applyOpeningLoadoutSelections(scen *enginev1.Scenario) {
	if scen == nil {
		return
	}
	unitByID := make(map[string]*enginev1.Unit, len(scen.GetUnits()))
	for _, unit := range scen.GetUnits() {
		if unit != nil {
			unitByID[unit.GetId()] = unit
		}
	}
	for _, action := range scen.GetOpeningStrikeActions() {
		if action == nil {
			continue
		}
		unit := unitByID[action.GetUnitId()]
		if unit == nil {
			continue
		}
		if loadoutID := strings.TrimSpace(action.GetLoadoutConfigurationId()); loadoutID != "" {
			unit.LoadoutConfigurationId = loadoutID
			unit.Weapons = nil
		}
	}
}

func (a *App) applyOpeningStrikeActions(scen *enginev1.Scenario, defs map[string]sim.DefStats) {
	if scen == nil || len(scen.GetOpeningStrikeActions()) == 0 {
		return
	}
	unitByID := make(map[string]*enginev1.Unit, len(scen.GetUnits()))
	for _, unit := range scen.GetUnits() {
		if unit != nil {
			unitByID[unit.GetId()] = unit
		}
	}
	for _, action := range scen.GetOpeningStrikeActions() {
		if action == nil {
			continue
		}
		shooter := unitByID[action.GetUnitId()]
		target := unitByID[action.GetTargetUnitId()]
		if shooter == nil || target == nil {
			slog.Warn("opening strike action references missing unit", "shooter", action.GetUnitId(), "target", action.GetTargetUnitId())
			continue
		}
		if err := a.validateAndConsumeLaunch(shooter, defs[shooter.GetDefinitionId()]); err != nil {
			slog.Warn("opening strike launch blocked", "unit", shooter.GetId(), "err", err)
			continue
		}
		shooter.AttackOrder = &enginev1.AttackOrder{
			OrderType:      enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT,
			TargetUnitId:   target.GetId(),
			DesiredEffect:  action.GetDesiredEffect(),
			PkillThreshold: 0.7,
		}
		if waypoint := sim.ComputeAttackWaypointForOrder(shooter, target, defs, a.buildWeaponCatalog()); waypoint != nil {
			shooter.MoveOrder = &enginev1.MoveOrder{
				Waypoints:     []*enginev1.Waypoint{waypoint},
				AutoGenerated: true,
			}
		}
		if text := strings.TrimSpace(action.GetNarrative()); text != "" {
			teamID := sim.CountryDisplayCode(shooter.GetTeamId())
			a.emitProtoEvent("narrative", &enginev1.NarrativeEvent{
				Text:     text,
				Category: "scenario",
				UnitId:   shooter.GetId(),
				TeamId:   teamID,
			})
		}
	}
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
			a.buildWeaponCatalog(),
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
		if err := a.validateTransit(unitCountryCode(u), defs[u.DefinitionId].Domain, oldPos.GetLat(), oldPos.GetLon(), lat, lon); err != nil {
			return failMsg(err.Error())
		}
		if err := a.validateAndConsumeLaunch(u, defs[u.DefinitionId]); err != nil {
			return failMsg(err.Error())
		}

		cruiseSpeed := defs[u.DefinitionId].CruiseSpeedMps
		if cruiseSpeed <= 0 {
			cruiseSpeed = 10
		}
		heading := sim.BearingDeg(oldPos.GetLat(), oldPos.GetLon(), lat, lon)
		newPos := &enginev1.Position{
			Lat:     oldPos.GetLat(),
			Lon:     oldPos.GetLon(),
			AltMsl:  oldPos.GetAltMsl(),
			Heading: heading,
			Speed:   cruiseSpeed,
		}
		newOrder := &enginev1.MoveOrder{
			AutoGenerated: false,
			Waypoints: []*enginev1.Waypoint{{
				Lat: lat, Lon: lon, AltMsl: oldPos.GetAltMsl(),
			}},
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
		if err := a.validateTransit(unitCountryCode(u), defs[u.DefinitionId].Domain, startLat, startLon, lat, lon); err != nil {
			return failMsg(err.Error())
		}
		if (u.GetMoveOrder() == nil || len(u.GetMoveOrder().GetWaypoints()) == 0) && shouldInitiateLaunch(u, defs[u.DefinitionId]) {
			if err := a.validateAndConsumeLaunch(u, defs[u.DefinitionId]); err != nil {
				return failMsg(err.Error())
			}
		}
		wp := &enginev1.Waypoint{Lat: lat, Lon: lon, AltMsl: pos.GetAltMsl()}
		var newOrder *enginev1.MoveOrder
		if current := u.GetMoveOrder(); current != nil {
			newOrder = proto.Clone(current).(*enginev1.MoveOrder)
		} else {
			newOrder = &enginev1.MoveOrder{AutoGenerated: false}
		}
		newOrder.AutoGenerated = false
		newOrder.Waypoints = append(newOrder.Waypoints, wp)
		u.MoveOrder = newOrder

		newPos := u.GetPosition()
		if len(newOrder.Waypoints) == 1 {
			cruiseSpeed := defs[u.DefinitionId].CruiseSpeedMps
			if cruiseSpeed <= 0 {
				cruiseSpeed = 10
			}
			heading := sim.BearingDeg(pos.GetLat(), pos.GetLon(), lat, lon)
			newPos = &enginev1.Position{
				Lat:     pos.GetLat(),
				Lon:     pos.GetLon(),
				AltMsl:  pos.GetAltMsl(),
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
		if err := a.validateTransit(unitCountryCode(u), a.getCachedDefs()[u.DefinitionId].Domain, startLat, startLon, lat, lon); err != nil {
			return failMsg(err.Error())
		}
		if waypointIndex < len(order.Waypoints)-1 {
			next := order.Waypoints[waypointIndex+1]
			if err := a.validateTransit(unitCountryCode(u), a.getCachedDefs()[u.DefinitionId].Domain, lat, lon, next.GetLat(), next.GetLon()); err != nil {
				return failMsg(err.Error())
			}
		}
		newOrder := proto.Clone(order).(*enginev1.MoveOrder)
		newOrder.AutoGenerated = false
		newOrder.Waypoints[waypointIndex].Lat = lat
		newOrder.Waypoints[waypointIndex].Lon = lon
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
		if err := a.validateStrike(shooter, target); err != nil {
			return failMsg(err.Error())
		}
		if current := u.GetMoveOrder(); current == nil || len(current.GetWaypoints()) == 0 {
			if err := a.validateAndConsumeLaunch(u, defs[u.DefinitionId]); err != nil {
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
			if waypoint := sim.ComputeAttackWaypointForOrder(u, target, a.buildDefs(), a.buildWeaponCatalog()); waypoint != nil {
				u.MoveOrder = &enginev1.MoveOrder{
					Waypoints:     []*enginev1.Waypoint{waypoint},
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
		domain := enginev1.UnitDomain(def.Domain)
		generalType := int32(def.GeneralType)
		rcs := float64(def.RadarCrossSectionM2)
		if rcs <= 0 {
			rcs = defaultRadarCrossSectionM2(domain, generalType)
		}
		defs[id] = sim.DefStats{
			CruiseSpeedMps:              float64(def.CruiseSpeedMps),
			BaseStrength:                float64(def.BaseStrength),
			DetectionRangeM:             float64(def.DetectionRangeM),
			RadarCrossSectionM2:         rcs,
			GeneralType:                 int32(def.GeneralType),
			EmploymentRole:              strings.TrimSpace(def.EmploymentRole),
			AuthorizedPersonnel:         maxInt(def.AuthorizedPersonnel, library.DefaultAuthorizedPersonnel(def.AssetClass, def.Domain, def.GeneralType)),
			ReplacementCostUSD:          defaultIfZero(def.ReplacementCostUSD, library.DefaultReplacementCostUSD(def.AssetClass, def.Domain, def.GeneralType)),
			StrategicValueUSD:           defaultIfZero(def.StrategicValueUSD, library.DefaultStrategicValueUSD(def.AssetClass, def.TargetClass, def.Domain, def.GeneralType, def.EmploymentRole)),
			EconomicValueUSD:            defaultIfZero(def.EconomicValueUSD, library.DefaultEconomicValueUSD(def.AssetClass, def.Affiliation)),
			Domain:                      domain,
			TargetClass:                 strings.TrimSpace(def.TargetClass),
			AssetClass:                  strings.TrimSpace(def.AssetClass),
			LaunchCapacityPerInterval:   def.LaunchCapacityPerInterval,
			RecoveryCapacityPerInterval: def.RecoveryCapacityPerInterval,
			SortieIntervalMinutes:       def.SortieIntervalMinutes,
		}
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
		domain := enginev1.UnitDomain(int32(toFloat64(row["domain"])))
		generalType := int32(toFloat64(row["general_type"]))
		rcs := toFloat64(row["radar_cross_section_m2"])
		if rcs <= 0 {
			rcs = defaultRadarCrossSectionM2(domain, generalType)
		}
		defs[id] = sim.DefStats{
			CruiseSpeedMps:              toFloat64(row["cruise_speed_mps"]),
			BaseStrength:                toFloat64(row["base_strength"]),
			DetectionRangeM:             toFloat64(row["detection_range_m"]),
			RadarCrossSectionM2:         rcs,
			GeneralType:                 int32(toFloat64(row["general_type"])),
			EmploymentRole:              toString(row["employment_role"]),
			AuthorizedPersonnel:         maxInt(int(toFloat64(row["authorized_personnel"])), library.DefaultAuthorizedPersonnel(toString(row["asset_class"]), int(toFloat64(row["domain"])), int(toFloat64(row["general_type"])))),
			ReplacementCostUSD:          defaultIfZero(toFloat64(row["replacement_cost_usd"]), library.DefaultReplacementCostUSD(toString(row["asset_class"]), int(toFloat64(row["domain"])), int(toFloat64(row["general_type"])))),
			StrategicValueUSD:           defaultIfZero(toFloat64(row["strategic_value_usd"]), library.DefaultStrategicValueUSD(toString(row["asset_class"]), toString(row["target_class"]), int(toFloat64(row["domain"])), int(toFloat64(row["general_type"])), toString(row["employment_role"]))),
			EconomicValueUSD:            defaultIfZero(toFloat64(row["economic_value_usd"]), library.DefaultEconomicValueUSD(toString(row["asset_class"]), toString(row["affiliation"]))),
			Domain:                      domain,
			TargetClass:                 toString(row["target_class"]),
			AssetClass:                  toString(row["asset_class"]),
			LaunchCapacityPerInterval:   int(toFloat64(row["launch_capacity_per_interval"])),
			RecoveryCapacityPerInterval: int(toFloat64(row["recovery_capacity_per_interval"])),
			SortieIntervalMinutes:       int(toFloat64(row["sortie_interval_minutes"])),
		}
	}
	return defs
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
