package main

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aressim/internal/db"
	"github.com/aressim/internal/db/repository"
	"github.com/aressim/internal/library"
	"github.com/aressim/internal/sim"

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
	humanTeamMu     sync.RWMutex
	humanTeam       string
	aiRandMu        sync.Mutex
	aiRand          *rand.Rand

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
	return &App{
		aiRand: rand.New(rand.NewSource(time.Now().UnixNano())), //nolint:gosec
	}
}
