package main

import (
	"context"
	"log/slog"
	"math"

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

// storeLastDetection records the most recent sensor result for one side so
// RequestSync can replay it to reconnecting clients.
func (a *App) storeLastDetection(side string, ids []string) {
	a.lastDetMu.Lock()
	if a.lastDetections == nil {
		a.lastDetections = make(map[string][]string)
	}
	cp := make([]string, len(ids))
	copy(cp, ids)
	a.lastDetections[side] = cp
	a.lastDetMu.Unlock()
}

// makeEmitFn returns an EmitFn that intercepts detection updates and checkpoints.
func (a *App) makeEmitFn() sim.EmitFn {
	var ticksSinceCheckpoint int
	return func(name string, msg proto.Message) {
		switch name {
		case "detection_update":
			if du, ok := msg.(*enginev1.DetectionUpdate); ok {
				a.storeLastDetection(du.DetectingSide, du.DetectedUnitIds)
			}
		case "batch_update":
			if bu, ok := msg.(*enginev1.BatchUnitUpdate); ok && bu.SimTime != nil {
				ticksSinceCheckpoint++
				if ticksSinceCheckpoint >= db.CheckpointInterval {
					ticksSinceCheckpoint = 0
					go a.writeCheckpoint(bu.SimTime.TickNumber, bu.SimTime.SecondsElapsed)
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

	unitDefs, _ := a.unitDefRepo.List(a.ctx)
	generalTypeByDefID := make(map[string]int32, len(unitDefs))
	for _, row := range unitDefs {
		id := extractRecordID(row["id"])
		if gt, ok := row["general_type"]; ok {
			generalTypeByDefID[id] = int32(toFloat64(gt))
		}
	}
	for _, u := range scen.Units {
		if len(u.Weapons) != 0 {
			continue
		}
		defID := extractRecordID(u.DefinitionId)
		if def, ok := a.libDefsCache[defID]; ok && len(def.DefaultLoadout) > 0 {
			u.Weapons = loadoutToWeaponStates(def.DefaultLoadout)
			slog.Info("weapon loadout from library", "unit", u.DisplayName, "def", defID, "weapons", len(u.Weapons))
			continue
		}
		gt := generalTypeByDefID[defID]
		u.Weapons = scenario.InitUnitWeapons(u, gt)
		slog.Info("weapon loadout from defaults", "unit", u.DisplayName, "def", defID, "general_type", gt, "weapons", len(u.Weapons))
	}

	a.emitProtoEvent("full_state_snapshot", &enginev1.FullStateSnapshot{
		Units:             scen.Units,
		SimTime:           &enginev1.SimTime{},
		Weather:           scen.GetMap().GetInitialWeather(),
		ScenarioName:      scen.Name,
		WeaponDefinitions: a.listWeaponDefsProto(),
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
	go sim.MockLoop(simCtx, scen.Units, a.getCachedDefs(), a.buildWeaponCatalog(), 0, a.getSimTimeScale, a.setSimSeconds, a.makeEmitFn())
	slog.Info("scenario loaded", "name", scen.Name, "units", len(scen.Units))
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
	})
	a.emitProtoEvent("scenario_state", &enginev1.ScenarioStateEvent{
		State:     enginev1.ScenarioPlayState_SCENARIO_RUNNING,
		TimeScale: float32(a.getSimTimeScale()),
	})
	a.lastDetMu.RLock()
	for side, ids := range a.lastDetections {
		a.emitProtoEvent("detection_update", &enginev1.DetectionUpdate{
			DetectingSide:   side,
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
		go sim.MockLoop(simCtx, a.currentScenario.Units, a.getCachedDefs(), a.buildWeaponCatalog(), a.getSimSeconds(), a.getSimTimeScale, a.setSimSeconds, a.makeEmitFn())
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
			Waypoints: []*enginev1.Waypoint{{
				Lat: lat, Lon: lon, AltMsl: oldPos.GetAltMsl(),
			}},
		}
		u.Position = newPos
		u.MoveOrder = newOrder

		a.emitProtoEvent("batch_update", &enginev1.BatchUnitUpdate{
			Deltas: []*enginev1.UnitDelta{{
				UnitId:    unitID,
				Position:  newPos,
				MoveOrder: newOrder,
			}},
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

// buildDefs queries all unit definitions and returns a map for the sim engine.
func (a *App) buildDefs() map[string]sim.DefStats {
	defs := make(map[string]sim.DefStats)
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
			CruiseSpeedMps:      toFloat64(row["cruise_speed_mps"]),
			BaseStrength:        toFloat64(row["base_strength"]),
			DetectionRangeM:     toFloat64(row["detection_range_m"]),
			RadarCrossSectionM2: rcs,
			Domain:              domain,
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
