package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"path/filepath"

	"github.com/aressim/internal/db"
	"github.com/aressim/internal/db/repository"
	"github.com/aressim/internal/library"
	"github.com/surrealdb/surrealdb.go/pkg/models"
	"github.com/aressim/internal/scenario"
	"github.com/aressim/internal/sim"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"google.golang.org/protobuf/proto"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// App is the Wails application object. All exported methods are automatically
// bound to the frontend as window.go.main.App.MethodName().
//
// The App owns the SurrealDB manager and the checkpoint writer. The full
// adjudicator will be wired in once the sim loop is implemented.
type App struct {
	// ctx is the Wails lifecycle context — required by runtime.EventsEmit.
	// It must never be replaced after startup.
	ctx    context.Context
	cancel context.CancelFunc

	// simCancel cancels the currently running sim loop goroutine.
	// Separate from ctx so we can restart the loop without touching the Wails context.
	simCancel context.CancelFunc

	// simTimeScale stores math.Float64bits(timeScale) atomically so SetSimSpeed
	// can update it from the Wails thread while MockLoop reads it each tick.
	simTimeScale atomic.Uint64

	// simSecondsAtomic stores math.Float64bits(simSeconds) atomically.
	// MockLoop updates this each tick via reportSeconds; PauseSim reads it
	// before restarting the loop so simSeconds survives pause/resume cycles.
	simSecondsAtomic atomic.Uint64

	// currentScenario is the last loaded scenario, kept so RequestSync can
	// re-emit the full state snapshot on demand.
	currentScenario *enginev1.Scenario

	// defsCache caches the result of buildDefs() so we don't re-query SurrealDB
	// on every MoveUnit call. Invalidated when a unit definition is saved or deleted.
	defsCacheMu sync.RWMutex
	defsCache   map[string]sim.DefStats

	// lastDetections stores the most recent DetectionSet emitted by SensorTick.
	// RequestSync re-emits these so reconnecting frontends see current contacts.
	lastDetMu      sync.RWMutex
	lastDetections map[string][]string

	dbMgr         *db.Manager
	dbClient      *db.Client
	unitRepo      *repository.UnitRepo
	scenRepo      *repository.ScenarioRepo
	unitDefRepo   *repository.UnitDefRepo
	weaponDefRepo *repository.WeaponDefRepo
	checkpoint    *db.CheckpointWriter

	// libDefsCache caches all library.Definition records keyed by ID so
	// loadScenario can apply per-definition weapon loadouts without re-parsing YAML.
	libDefsCache map[string]library.Definition

	// shutdownOnce ensures the shutdown path runs at most once, whether
	// triggered by Wails OnShutdown or by our OS signal handler.
	shutdownOnce sync.Once
}

// NewApp creates a new App instance. Called from main.go before wails.Run.
func NewApp() *App {
	return &App{}
}

// startup is called by Wails when the application starts.
// It initialises SurrealDB and wires up the repository layer.
func (a *App) startup(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)

	cfg, err := db.DefaultConfig()
	if err != nil {
		slog.Error("db config", "err", err)
		runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Startup Error",
			Message: fmt.Sprintf("Failed to resolve app data directory: %v", err),
		})
		return
	}

	a.dbMgr = db.NewManager(cfg)
	if err := a.dbMgr.Start(ctx); err != nil {
		slog.Error("surreal start", "err", err)
		runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Database Error",
			Message: fmt.Sprintf("Failed to start SurrealDB: %v\n\nInstall with: curl -sSf https://install.surrealdb.com | sh", err),
		})
		return
	}
	slog.Info("surrealdb ready", "addr", a.dbMgr.Addr())

	a.dbClient, err = db.Connect(ctx, cfg)
	if err != nil {
		slog.Error("surreal connect", "err", err)
		return
	}

	if err := db.EnsureSchema(ctx, a.dbClient.DB()); err != nil {
		slog.Error("schema ensure", "err", err)
		return
	}
	slog.Info("schema ready", "version", db.SchemaVersion())

	rawDB := a.dbClient.DB()
	a.unitRepo = repository.NewUnitRepo(rawDB)
	a.scenRepo = repository.NewScenarioRepo(rawDB)
	a.unitDefRepo = repository.NewUnitDefRepo(rawDB)
	a.weaponDefRepo = repository.NewWeaponDefRepo(rawDB)
	a.checkpoint = db.NewCheckpointWriter(a.unitRepo, a.scenRepo)

	// Seed default data synchronously so it is present before the frontend loads.
	a.seedDefaults(ctx, cfg)

	// Register an OS-level signal handler so that Ctrl-C (SIGINT) and SIGTERM
	// both trigger a graceful SurrealDB shutdown before the process exits.
	// Wails's OnShutdown hook is NOT called when the process receives a signal,
	// so this is the only reliable place to stop the DB subprocess.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		sig := <-sigCh
		slog.Info("received signal — shutting down", "signal", sig)
		a.shutdown(ctx)
		os.Exit(0)
	}()
}

// seedDefaults writes the default scenario and all built-in unit definitions
// into the database. Uses UPSERT semantics so it is safe to call on every
// startup — existing records are overwritten, ensuring library updates are
// always applied.
func (a *App) seedDefaults(ctx context.Context, cfg db.Config) {
	slog.Info("seeding defaults")

	// ── Default scenario ───────────────────────────────────────────────────
	def := scenario.Default()
	if err := a.scenRepo.Save(ctx, def.Id, scenarioRecord(def)); err != nil {
		slog.Warn("seed default scenario", "err", err)
	}

	// ── Scaffold unit definitions (land/sea/support/transport) ────────────
	for _, unitDef := range scenario.DefaultUnitDefinitions() {
		if err := a.unitDefRepo.Save(ctx, unitDef.Id, map[string]any{
			"name":                 unitDef.Name,
			"description":          unitDef.Description,
			"domain":               int(unitDef.Domain),
			"form":                 int(unitDef.Form),
			"general_type":         int(unitDef.GeneralType),
			"specific_type":        unitDef.SpecificType,
			"nation_of_origin":     unitDef.NationOfOrigin,
			"service_entry_year":   int(unitDef.ServiceEntryYear),
			"base_strength":        float64(unitDef.BaseStrength),
			"accuracy":             float64(unitDef.Accuracy),
			"max_speed_mps":        float64(unitDef.MaxSpeedMps),
			"cruise_speed_mps":     float64(unitDef.CruiseSpeedMps),
			"max_range_km":         float64(unitDef.MaxRangeKm),
			"survivability":        float64(unitDef.Survivability),
			"detection_range_m":    float64(unitDef.DetectionRangeM),
			"fuel_capacity_liters": float64(unitDef.FuelCapacityLiters),
			"fuel_burn_rate_lph":   float64(unitDef.FuelBurnRateLph),
		}); err != nil {
			slog.Warn("seed scaffold definition", "id", unitDef.Id, "err", err)
		}
	}

	// ── Library unit definitions (embedded YAML + user libraries) ─────────
	userLibDir := filepath.Join(cfg.DataDir, "libraries")
	libDefs, err := library.LoadAll(userLibDir)
	if err != nil {
		slog.Warn("load libraries", "err", err)
	}
	a.libDefsCache = make(map[string]library.Definition, len(libDefs))
	for _, d := range libDefs {
		a.libDefsCache[d.ID] = d
		if err := a.unitDefRepo.Save(ctx, d.ID, d.ToRecord()); err != nil {
			slog.Warn("seed library definition", "id", d.ID, "err", err)
		}
	}
	// ── Weapon definitions ────────────────────────────────────────────────
	for _, wd := range scenario.DefaultWeaponDefinitions() {
		targets := make([]int, len(wd.DomainTargets))
		for i, d := range wd.DomainTargets {
			targets[i] = int(d)
		}
		if err := a.weaponDefRepo.Save(ctx, wd.Id, map[string]any{
			"name":               wd.Name,
			"description":        wd.Description,
			"domain_targets":     targets,
			"speed_mps":          float64(wd.SpeedMps),
			"range_m":            float64(wd.RangeM),
			"probability_of_hit": float64(wd.ProbabilityOfHit),
		}); err != nil {
			slog.Warn("seed weapon definition", "id", wd.Id, "err", err)
		}
	}

	slog.Info("seeding complete",
		"scaffold", len(scenario.DefaultUnitDefinitions()),
		"library", len(libDefs),
		"weapons", len(scenario.DefaultWeaponDefinitions()),
	)
}

// shutdown is called by Wails when the application is about to quit, and also
// by our OS signal handler. shutdownOnce ensures the body runs exactly once
// regardless of which path fires first.
func (a *App) shutdown(ctx context.Context) {
	a.shutdownOnce.Do(func() {
		slog.Info("shutdown started")
		if a.simCancel != nil {
			a.simCancel()
		}
		if a.cancel != nil {
			a.cancel()
		}
		if a.dbClient != nil {
			_ = a.dbClient.Close(ctx)
		}
		if a.dbMgr != nil {
			a.dbMgr.Stop()
		}
		slog.Info("shutdown complete")
	})
}

// domReady is called after the frontend DOM is loaded.
// Seeding is done in startup so all data is available before the frontend renders.
func (a *App) domReady(_ context.Context) {
	slog.Info("dom ready")
}

// getSimTimeScale reads the current time scale atomically.
func (a *App) getSimTimeScale() float64 {
	bits := a.simTimeScale.Load()
	if bits == 0 {
		return 1.0 // zero value before first store — default to real-time
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
// Called by MockLoop via the reportSeconds callback each tick.
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

	// Cache miss — build from DB.
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

// makeEmitFn returns an EmitFn that:
//   - intercepts detection_update to keep lastDetections current
//   - intercepts batch_update to trigger periodic checkpoint writes
//   - forwards everything to emitProtoEvent for the frontend
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

// writeCheckpoint persists the current in-memory unit positions and scenario
// progress to SurrealDB. Called asynchronously every CheckpointInterval ticks.
// Errors are non-fatal — the sim continues and the next checkpoint will retry.
func (a *App) writeCheckpoint(tick int64, simSeconds float64) {
	if a.checkpoint == nil || a.currentScenario == nil {
		return
	}
	units := make([]repository.UnitRecord, 0, len(a.currentScenario.Units))
	for _, u := range a.currentScenario.Units {
		if u.Status != nil && !u.Status.IsActive {
			continue // don't persist destroyed units
		}
		pos := u.GetPosition()
		units = append(units, repository.UnitRecord{
			"id": models.RecordID{Table: "unit", ID: u.Id},
			// GeoJSON coordinate order for geometry<point>: [longitude, latitude]
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

// ─── BRIDGE METHODS ───────────────────────────────────────────────────────────
// These are the Go methods exposed to the TypeScript frontend.
// Parameter and return types must be JSON-serializable (Wails uses JSON IPC).
//
// For proto messages, the convention is:
//   - Inbound (TS → Go): base64-encoded proto binary string
//   - Outbound events (Go → TS): base64-encoded proto binary via EventsEmit
//   - Bridge return values: plain JSON structs for simplicity

// BridgeResult is the standard return type for all bridge calls.
// Matches OperationResult in common.proto but avoids a proto round-trip
// for simple success/error signalling.
type BridgeResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func ok() BridgeResult { return BridgeResult{Success: true} }
func fail(err error) BridgeResult {
	slog.Error("bridge error", "err", err)
	return BridgeResult{Error: err.Error()}
}
func failMsg(msg string) BridgeResult {
	slog.Error("bridge error", "msg", msg)
	return BridgeResult{Error: msg}
}

// GetVersion returns the application version string.
// Used by the frontend's about panel.
func (a *App) GetVersion() string {
	return "0.1.0-dev"
}

// ListScenarios returns metadata for all stored scenarios.
// The frontend uses this to populate the scenario picker.
// The "id" field is normalized to a plain string (table prefix stripped).
func (a *App) ListScenarios() ([]map[string]any, error) {
	if a.scenRepo == nil {
		return nil, fmt.Errorf("database not ready")
	}
	rows, err := a.scenRepo.List(a.ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row["id"] = extractRecordID(row["id"])
	}
	return rows, nil
}

// LoadScenarioFromProto accepts a base64-encoded serialized Scenario proto,
// stores it in SurrealDB, and starts the simulation.
func (a *App) LoadScenarioFromProto(protoB64 string) BridgeResult {
	scen, err := decodeScenarioB64(protoB64)
	if err != nil {
		return fail(err)
	}
	a.loadScenario(scen)
	return ok()
}

// loadScenario is the internal entry point for starting a scenario.
// It persists to SurrealDB (if available), emits FullStateSnapshot, and
// starts the mock sim loop.
func (a *App) loadScenario(scen *enginev1.Scenario) {
	// Persist to SurrealDB if the database is ready.
	if a.scenRepo != nil {
		if err := a.scenRepo.Save(a.ctx, scen.Id, scenarioRecord(scen)); err != nil {
			slog.Warn("save scenario to db", "err", err)
		}
	}

	a.currentScenario = scen

	// Initialize weapon loadouts for units that have none yet.
	unitDefs, _ := a.unitDefRepo.List(a.ctx)
	generalTypeByDefID := make(map[string]int32, len(unitDefs))
	for _, row := range unitDefs {
		id := extractRecordID(row["id"])
		if gt, ok := row["general_type"]; ok {
			generalTypeByDefID[id] = int32(toFloat64(gt))
		}
	}
	for _, u := range scen.Units {
		if len(u.Weapons) == 0 {
			// Normalize the definition ID — strip any residual table prefix that
			// may have been stored when extractRecordID used the old fmt path.
			defID := extractRecordID(u.DefinitionId)
			if def, ok := a.libDefsCache[defID]; ok && len(def.DefaultLoadout) > 0 {
				u.Weapons = loadoutToWeaponStates(def.DefaultLoadout)
				slog.Info("weapon loadout from library", "unit", u.DisplayName, "def", defID, "weapons", len(u.Weapons))
			} else {
				gt := generalTypeByDefID[defID]
				u.Weapons = scenario.InitUnitWeapons(u, gt)
				slog.Info("weapon loadout from defaults", "unit", u.DisplayName, "def", defID, "general_type", gt, "weapons", len(u.Weapons))
			}
		}
	}

	// Fetch weapon definitions for the snapshot.
	weaponDefs := a.listWeaponDefsProto()

	// Tell the frontend to rebuild from scratch.
	a.emitProtoEvent("full_state_snapshot", &enginev1.FullStateSnapshot{
		Units:             scen.Units,
		SimTime:           &enginev1.SimTime{},
		Weather:           scen.GetMap().GetInitialWeather(),
		ScenarioName:      scen.Name,
		WeaponDefinitions: weaponDefs,
	})
	a.emitProtoEvent("scenario_state", &enginev1.ScenarioStateEvent{
		State:     enginev1.ScenarioPlayState_SCENARIO_RUNNING,
		TimeScale: scen.GetSettings().GetTimeScale(),
	})

	// Reset sim clock and time scale on every new scenario load.
	a.setSimTimeScale(1.0)
	a.setSimSeconds(0)
	// Clear stale detections from a previous scenario.
	a.lastDetMu.Lock()
	a.lastDetections = nil
	a.lastDetMu.Unlock()

	// Cancel any previous sim loop, then start a new one.
	if a.simCancel != nil {
		a.simCancel()
	}
	simCtx, simCancel := context.WithCancel(a.ctx)
	a.simCancel = simCancel
	go sim.MockLoop(simCtx, scen.Units, a.getCachedDefs(), a.buildWeaponCatalog(), 0, a.getSimTimeScale, a.setSimSeconds, a.makeEmitFn())
	slog.Info("scenario loaded", "name", scen.Name, "units", len(scen.Units))
}

// RequestSync re-emits the full state snapshot, scenario state, and last known
// sensor detections. The frontend calls this after registering its event
// listeners to guarantee it receives the initial state even if domReady fired
// first. Re-emitting detections ensures reconnecting clients don't see a blank
// fog-of-war until the next SensorTick fires.
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
	// Re-emit each side's current detection set so contacts aren't lost on reconnect.
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
// Pause actually stops the mock loop so manual unit moves take effect.
// Resume restarts the mock loop from the current scenario unit positions.
func (a *App) PauseSim(paused bool) BridgeResult {
	if paused {
		if a.simCancel != nil {
			a.simCancel()
			a.simCancel = nil
		}
	} else {
		if a.simCancel == nil && a.currentScenario != nil {
			simCtx, simCancel := context.WithCancel(a.ctx)
			a.simCancel = simCancel
			// Resume from the accumulated sim time so the clock doesn't reset.
			go sim.MockLoop(simCtx, a.currentScenario.Units, a.getCachedDefs(), a.buildWeaponCatalog(), a.getSimSeconds(), a.getSimTimeScale, a.setSimSeconds, a.makeEmitFn())
		}
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

// MoveUnit issues a movement order to a unit, sending it toward the given
// lat/lon at its cruise speed. The sim loop processes one step per tick.
//
// Note: heading is set to the bearing toward the destination at order time.
// It remains at this value for up to one tick until processTick updates it.
// This is a known one-tick visual stutter and is intentional.
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

		// Warn if a land unit is being sent to an implausible location.
		// Full domain validation (e.g. ocean tile detection) is future work.
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			slog.Warn("MoveUnit: target coordinates out of range", "id", unitID, "lat", lat, "lon", lon)
			return failMsg("target coordinates out of range")
		}

		cruiseSpeed := defs[u.DefinitionId].CruiseSpeedMps
		if cruiseSpeed <= 0 {
			cruiseSpeed = 10 // m/s fallback
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
		slog.Info("move order issued", "id", unitID,
			"dest_lat", lat, "dest_lon", lon,
			"speed_mps", cruiseSpeed, "heading", heading)
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
				MoveOrder: &enginev1.MoveOrder{}, // empty = clear
			}},
		})
		slog.Info("move order cancelled", "id", unitID)
		return ok()
	}
	return failMsg("unit not found: " + unitID)
}

// buildDefs queries all unit definitions and returns a map of
// definitionId → DefStats for use by the movement and adjudication engine.
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
		defs[id] = sim.DefStats{
			CruiseSpeedMps:  toFloat64(row["cruise_speed_mps"]),
			BaseStrength:    toFloat64(row["base_strength"]),
			DetectionRangeM: toFloat64(row["detection_range_m"]),
			Domain:          enginev1.UnitDomain(int32(toFloat64(row["domain"]))),
		}
	}
	return defs
}

// loadoutToWeaponStates converts library.LoadoutSlot entries (from YAML) into
// the proto WeaponState slice expected by the adjudicator and frontend.
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

// buildWeaponCatalog converts the stored weapon definitions into a
// weaponId → WeaponStats map for the adjudication engine.
func (a *App) buildWeaponCatalog() map[string]sim.WeaponStats {
	catalog := make(map[string]sim.WeaponStats)
	protoWeapons := a.listWeaponDefsProto()
	for _, wd := range protoWeapons {
		catalog[wd.Id] = sim.WeaponStats{
			RangeM:           float64(wd.RangeM),
			SpeedMps:         float64(wd.SpeedMps),
			ProbabilityOfHit: float64(wd.ProbabilityOfHit),
			DomainTargets:    wd.DomainTargets,
		}
	}
	return catalog
}

// SetSimSpeed sets the simulation time scale multiplier.
// timeScale = 1.0 is real-time; 2.0 = 2× speed (2 sim-seconds per tick), etc.
// The running MockLoop reads this atomically at the start of each tick.
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

// ─── SCENARIO EDITOR BRIDGE ───────────────────────────────────────────────────

// SaveScenario persists an edited scenario proto to SurrealDB without
// starting the simulation. Used by the scenario editor's Save button.
func (a *App) SaveScenario(protoB64 string) BridgeResult {
	scen, err := decodeScenarioB64(protoB64)
	if err != nil {
		return fail(err)
	}
	if a.scenRepo == nil {
		return failMsg("database not ready")
	}
	if err := a.scenRepo.Save(a.ctx, scen.Id, scenarioRecord(scen)); err != nil {
		return fail(err)
	}
	slog.Info("scenario saved", "id", scen.Id, "name", scen.Name)
	return ok()
}

// GetScenario fetches a stored scenario by ID and returns it as a
// base64-encoded proto binary string. Used by the scenario editor to
// load an existing scenario for editing.
func (a *App) GetScenario(id string) (string, error) {
	if a.scenRepo == nil {
		return "", fmt.Errorf("database not ready")
	}
	rec, err := a.scenRepo.Get(a.ctx, stripTablePrefix(id))
	if err != nil {
		return "", err
	}
	rawAny, ok := rec["scenario_pb"]
	if !ok {
		return "", fmt.Errorf("scenario %s has no proto blob", id)
	}
	// SurrealDB returns bytes as []byte or []uint8.
	var raw []byte
	switch v := rawAny.(type) {
	case []byte:
		raw = v
	case string:
		raw, err = base64.StdEncoding.DecodeString(v)
		if err != nil {
			return "", fmt.Errorf("decode stored proto: %w", err)
		}
	default:
		return "", fmt.Errorf("unexpected scenario_pb type %T", rawAny)
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

// DeleteScenario removes a scenario and its checkpoint history from the database.
func (a *App) DeleteScenario(id string) BridgeResult {
	if a.scenRepo == nil {
		return failMsg("database not ready")
	}
	if err := a.scenRepo.Delete(a.ctx, stripTablePrefix(id)); err != nil {
		return fail(err)
	}
	return ok()
}

// ─── WEAPON DEFINITION BRIDGE ─────────────────────────────────────────────────

// ListWeaponDefinitions returns all weapon definitions for the frontend.
func (a *App) ListWeaponDefinitions() ([]map[string]any, error) {
	if a.weaponDefRepo == nil {
		return nil, fmt.Errorf("database not ready")
	}
	rows, err := a.weaponDefRepo.List(a.ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row["id"] = extractRecordID(row["id"])
	}
	return rows, nil
}

// listWeaponDefsProto converts DB weapon definition rows into proto messages.
// Used internally to populate FullStateSnapshot.
func (a *App) listWeaponDefsProto() []*enginev1.WeaponDefinition {
	if a.weaponDefRepo == nil {
		return scenario.DefaultWeaponDefinitions()
	}
	rows, err := a.weaponDefRepo.List(a.ctx)
	if err != nil {
		slog.Warn("listWeaponDefsProto: list", "err", err)
		return scenario.DefaultWeaponDefinitions()
	}
	out := make([]*enginev1.WeaponDefinition, 0, len(rows))
	for _, row := range rows {
		wd := &enginev1.WeaponDefinition{
			Id:              extractRecordID(row["id"]),
			Name:            toString(row["name"]),
			Description:     toString(row["description"]),
			SpeedMps:        float32(toFloat64(row["speed_mps"])),
			RangeM:          float32(toFloat64(row["range_m"])),
			ProbabilityOfHit: float32(toFloat64(row["probability_of_hit"])),
		}
		if targets, ok := row["domain_targets"]; ok {
			switch v := targets.(type) {
			case []any:
				for _, item := range v {
					wd.DomainTargets = append(wd.DomainTargets, enginev1.UnitDomain(int32(toFloat64(item))))
				}
			}
		}
		out = append(out, wd)
	}
	return out
}

// ─── UNIT DEFINITION BRIDGE ───────────────────────────────────────────────────

// ListUnitDefinitions returns all unit definitions for the palette/editor.
// The "id" field is normalized to a plain string.
func (a *App) ListUnitDefinitions() ([]map[string]any, error) {
	if a.unitDefRepo == nil {
		return nil, fmt.Errorf("database not ready")
	}
	rows, err := a.unitDefRepo.List(a.ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row["id"] = extractRecordID(row["id"])
	}
	return rows, nil
}

// SaveUnitDefinition persists a unit definition from a JSON map.
// Expects all UnitDefinition fields as a JSON object string.
func (a *App) SaveUnitDefinition(jsonStr string) BridgeResult {
	var rec map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &rec); err != nil {
		return fail(fmt.Errorf("json decode: %w", err))
	}
	id, _ := rec["id"].(string)
	if id == "" {
		return failMsg("unit definition id is required")
	}
	if a.unitDefRepo == nil {
		return failMsg("database not ready")
	}
	if err := a.unitDefRepo.Save(a.ctx, id, rec); err != nil {
		return fail(err)
	}
	a.invalidateDefsCache()
	return ok()
}

// DeleteUnitDefinition removes a unit definition by id.
func (a *App) DeleteUnitDefinition(id string) BridgeResult {
	if a.unitDefRepo == nil {
		return failMsg("database not ready")
	}
	if err := a.unitDefRepo.Delete(a.ctx, stripTablePrefix(id)); err != nil {
		return fail(err)
	}
	a.invalidateDefsCache()
	return ok()
}

// ─── HELPERS ──────────────────────────────────────────────────────────────────

// extractRecordID converts whatever SurrealDB puts in the "id" field of a
// query result into a plain string suitable for use as a Go map key or URL
// parameter. SurrealDB 2/3 may return the id as:
//   - a models.RecordID struct  → fmt.Sprintf gives "table:id", so we strip the prefix
//   - a plain string            → used as-is (strip prefix if present)
//   - any other fmt.Stringer    → convert and strip prefix
// toFloat64 converts a numeric any value (from SurrealDB row) to float64.
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	}
	return 0
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func extractRecordID(v any) string {
	// Fast path: SurrealDB CBOR decoder puts RecordID values (not pointers) into
	// map[string]any, so String() (pointer receiver) is never called by fmt.Sprintf.
	// Access .ID directly to get the plain string identifier.
	switch rid := v.(type) {
	case models.RecordID:
		return fmt.Sprintf("%v", rid.ID)
	case *models.RecordID:
		return fmt.Sprintf("%v", rid.ID)
	}
	// Fallback for plain strings like "unit_definition:f22a-raptor".
	s := fmt.Sprintf("%v", v)
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

// stripTablePrefix removes the "table:" prefix that may arrive on IDs
// passed back from the frontend (which received them from ListScenarios).
func stripTablePrefix(id string) string {
	if idx := strings.LastIndex(id, ":"); idx >= 0 {
		return id[idx+1:]
	}
	return id
}

// ─── EVENT EMISSION ───────────────────────────────────────────────────────────

// emitProtoEvent marshals a proto message to binary, base64-encodes it,
// and emits it to the frontend via the Wails event bus.
//
// The frontend listens with: runtime.EventsOn("sim:event_name", handler)
// and decodes: const msg = proto.fromBinary(EventClass, base64ToBytes(data))
// scenarioRecord builds the SurrealDB map for a Scenario proto.
// All numeric types are widened to int/float64 to satisfy SCHEMAFULL TYPE
// constraints — the CBOR transport used by the Go SDK distinguishes float32
// from float64, and SurrealDB rejects the narrower type for TYPE float fields.
// The proto blob is stored as a base64 string (TYPE string) to avoid CBOR
// byte-array encoding issues with the TYPE bytes constraint.
func scenarioRecord(scen *enginev1.Scenario) map[string]any {
	raw, _ := proto.Marshal(scen)
	return map[string]any{
		"name":             scen.Name,
		"description":      scen.Description,
		"author":           scen.Author,
		"classification":   scen.Classification,
		"start_time_unix":  scen.StartTimeUnix, // already float64 (proto double)
		"schema_version":   scen.Version,
		"tick_rate_hz":     float64(scen.GetSettings().GetTickRateHz()),
		"time_scale":       float64(scen.GetSettings().GetTimeScale()),
		"adj_model":        0,
		"scenario_pb":      base64.StdEncoding.EncodeToString(raw),
		"last_tick":        0,
		"last_sim_seconds": 0.0,
	}
}

// decodeScenarioB64 decodes a base64-encoded proto Scenario binary.
func decodeScenarioB64(b64 string) (*enginev1.Scenario, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	scen := &enginev1.Scenario{}
	if err := proto.Unmarshal(raw, scen); err != nil {
		return nil, fmt.Errorf("proto unmarshal: %w", err)
	}
	return scen, nil
}

func (a *App) emitProtoEvent(eventName string, msg proto.Message) {
	data, err := proto.Marshal(msg)
	if err != nil {
		slog.Error("proto marshal for event", "event", eventName, "err", err)
		return
	}
	runtime.EventsEmit(a.ctx, "sim:"+eventName, base64.StdEncoding.EncodeToString(data))
}
