package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"path/filepath"

	"github.com/aressim/internal/db"
	"github.com/aressim/internal/db/repository"
	"github.com/aressim/internal/library"
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

	// currentScenario is the last loaded scenario, kept so RequestSync can
	// re-emit the full state snapshot on demand.
	currentScenario *enginev1.Scenario

	dbMgr       *db.Manager
	dbClient    *db.Client
	unitRepo    *repository.UnitRepo
	scenRepo    *repository.ScenarioRepo
	unitDefRepo *repository.UnitDefRepo
	checkpoint  *db.CheckpointWriter

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
			"combat_range_m":       float64(unitDef.CombatRangeM),
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
	for _, d := range libDefs {
		if err := a.unitDefRepo.Save(ctx, d.ID, d.ToRecord()); err != nil {
			slog.Warn("seed library definition", "id", d.ID, "err", err)
		}
	}
	slog.Info("seeding complete",
		"scaffold", len(scenario.DefaultUnitDefinitions()),
		"library", len(libDefs),
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

	// Tell the frontend to rebuild from scratch.
	a.emitProtoEvent("full_state_snapshot", &enginev1.FullStateSnapshot{
		Units:        scen.Units,
		SimTime:      &enginev1.SimTime{},
		Weather:      scen.GetMap().GetInitialWeather(),
		ScenarioName: scen.Name,
	})
	a.emitProtoEvent("scenario_state", &enginev1.ScenarioStateEvent{
		State:     enginev1.ScenarioPlayState_SCENARIO_RUNNING,
		TimeScale: scen.GetSettings().GetTimeScale(),
	})

	// Cancel any previous sim loop, then start a new one.
	if a.simCancel != nil {
		a.simCancel()
	}
	simCtx, simCancel := context.WithCancel(a.ctx)
	a.simCancel = simCancel
	go sim.MockLoop(simCtx, scen.Units, a.buildDefs(), a.emitProtoEvent)
	slog.Info("scenario loaded", "name", scen.Name, "units", len(scen.Units))
}

// RequestSync re-emits the full state snapshot and current scenario state.
// The frontend calls this after registering its event listeners to guarantee
// it receives the initial state even if domReady fired first.
func (a *App) RequestSync() BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	a.emitProtoEvent("full_state_snapshot", &enginev1.FullStateSnapshot{
		Units:        a.currentScenario.Units,
		SimTime:      &enginev1.SimTime{},
		Weather:      a.currentScenario.GetMap().GetInitialWeather(),
		ScenarioName: a.currentScenario.Name,
	})
	a.emitProtoEvent("scenario_state", &enginev1.ScenarioStateEvent{
		State:     enginev1.ScenarioPlayState_SCENARIO_RUNNING,
		TimeScale: a.currentScenario.GetSettings().GetTimeScale(),
	})
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
			go sim.MockLoop(simCtx, a.currentScenario.Units, a.buildDefs(), a.emitProtoEvent)
		}
	}
	state := enginev1.ScenarioPlayState_SCENARIO_PAUSED
	if !paused {
		state = enginev1.ScenarioPlayState_SCENARIO_RUNNING
	}
	a.emitProtoEvent("scenario_state", &enginev1.ScenarioStateEvent{
		State:     state,
		TimeScale: 1.0,
	})
	return ok()
}

// MoveUnit issues a movement order to a unit, sending it toward the given
// lat/lon at its cruise speed. The sim loop processes one step per tick.
func (a *App) MoveUnit(unitID string, lat, lon float64) BridgeResult {
	if a.currentScenario == nil {
		return failMsg("no scenario loaded")
	}
	defs := a.buildDefs()

	for _, u := range a.currentScenario.Units {
		if u.Id != unitID {
			continue
		}
		oldPos := u.GetPosition()

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
			CruiseSpeedMps: toFloat64(row["cruise_speed_mps"]),
			CombatRangeM:   toFloat64(row["combat_range_m"]),
			BaseStrength:   toFloat64(row["base_strength"]),
		}
	}
	return defs
}

// SetSimSpeed sets the simulation time scale multiplier.
// timeScale = 1.0 is real-time, 60.0 is 1 minute per second.
func (a *App) SetSimSpeed(timeScale float32) BridgeResult {
	if timeScale <= 0 || timeScale > 3600 {
		return failMsg("timeScale must be between 0 and 3600")
	}
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

func extractRecordID(v any) string {
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
