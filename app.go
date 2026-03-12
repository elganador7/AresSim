package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/aressim/internal/db"
	"github.com/aressim/internal/db/repository"
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

	dbMgr      *db.Manager
	dbClient   *db.Client
	unitRepo   *repository.UnitRepo
	nodeRepo   *repository.NodeRepo
	linkRepo   *repository.LinkRepo
	scenRepo   *repository.ScenarioRepo
	checkpoint *db.CheckpointWriter
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

	if err := db.ApplySchema(ctx, a.dbClient.DB()); err != nil {
		slog.Error("schema apply", "err", err)
		return
	}
	slog.Info("schema applied")

	rawDB := a.dbClient.DB()
	a.unitRepo = repository.NewUnitRepo(rawDB)
	a.nodeRepo = repository.NewNodeRepo(rawDB)
	a.linkRepo = repository.NewLinkRepo(rawDB)
	a.scenRepo = repository.NewScenarioRepo(rawDB)
	a.checkpoint = db.NewCheckpointWriter(a.unitRepo, a.nodeRepo, a.scenRepo)
}

// shutdown is called by Wails when the application is about to quit.
func (a *App) shutdown(ctx context.Context) {
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
}

// domReady is called after the frontend DOM is loaded.
// Loads the default scenario so the map is populated immediately on startup.
func (a *App) domReady(ctx context.Context) {
	slog.Info("dom ready — loading default scenario")
	a.loadScenario(scenario.Default())
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

func ok() BridgeResult                 { return BridgeResult{Success: true} }
func fail(err error) BridgeResult      { return BridgeResult{Error: err.Error()} }
func failMsg(msg string) BridgeResult  { return BridgeResult{Error: msg} }

// GetVersion returns the application version string.
// Used by the frontend's about panel.
func (a *App) GetVersion() string {
	return "0.1.0-dev"
}

// ListScenarios returns metadata for all stored scenarios.
// The frontend uses this to populate the scenario picker.
func (a *App) ListScenarios() ([]map[string]any, error) {
	if a.scenRepo == nil {
		return nil, fmt.Errorf("database not ready")
	}
	return a.scenRepo.List(a.ctx)
}

// LoadScenarioFromProto accepts a base64-encoded serialized Scenario proto,
// stores it in SurrealDB, and starts the simulation.
func (a *App) LoadScenarioFromProto(protoB64 string) BridgeResult {
	raw, err := base64.StdEncoding.DecodeString(protoB64)
	if err != nil {
		return fail(fmt.Errorf("base64 decode: %w", err))
	}
	scen := &enginev1.Scenario{}
	if err := proto.Unmarshal(raw, scen); err != nil {
		return fail(fmt.Errorf("proto unmarshal: %w", err))
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
		raw, _ := proto.Marshal(scen)
		if err := a.scenRepo.Save(a.ctx, scen.Id, map[string]any{
			"name":             scen.Name,
			"description":      scen.Description,
			"author":           scen.Author,
			"classification":   scen.Classification,
			"start_time_unix":  scen.StartTimeUnix,
			"schema_version":   scen.Version,
			"tick_rate_hz":     scen.GetSettings().GetTickRateHz(),
			"time_scale":       scen.GetSettings().GetTimeScale(),
			"adj_model":        int(scen.GetSettings().GetAdjModel()),
			"scenario_pb":      raw,
			"last_tick":        0,
			"last_sim_seconds": 0.0,
		}); err != nil {
			slog.Warn("save scenario to db", "err", err)
		}
	}

	a.currentScenario = scen

	// Tell the frontend to rebuild from scratch.
	a.emitProtoEvent("full_state_snapshot", &enginev1.FullStateSnapshot{
		Units:        scen.Units,
		Nodes:        scen.Nodes,
		SimTime:      &enginev1.SimTime{},
		Weather:      scen.GetMap().GetInitialWeather(),
		ScenarioName: scen.Name,
	})
	a.emitProtoEvent("scenario_state", &enginev1.ScenarioStateEvent{
		State:     enginev1.ScenarioPlayState_SCENARIO_RUNNING,
		TimeScale: scen.GetSettings().GetTimeScale(),
	})

	// Cancel any previous sim loop, then start a new one.
	// Use a child of the Wails context so the loop stops on app shutdown too.
	if a.simCancel != nil {
		a.simCancel()
	}
	simCtx, simCancel := context.WithCancel(a.ctx)
	a.simCancel = simCancel
	go sim.MockLoop(simCtx, scen.Units, a.emitProtoEvent)
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
		Nodes:        a.currentScenario.Nodes,
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
func (a *App) PauseSim(paused bool) BridgeResult {
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

// ─── EVENT EMISSION ───────────────────────────────────────────────────────────

// emitProtoEvent marshals a proto message to binary, base64-encodes it,
// and emits it to the frontend via the Wails event bus.
//
// The frontend listens with: runtime.EventsOn("sim:event_name", handler)
// and decodes: const msg = proto.fromBinary(EventClass, base64ToBytes(data))
func (a *App) emitProtoEvent(eventName string, msg proto.Message) {
	data, err := proto.Marshal(msg)
	if err != nil {
		slog.Error("proto marshal for event", "event", eventName, "err", err)
		return
	}
	runtime.EventsEmit(a.ctx, "sim:"+eventName, base64.StdEncoding.EncodeToString(data))
}
