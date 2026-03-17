package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/aressim/internal/db"
	"github.com/aressim/internal/db/repository"
	"github.com/aressim/internal/library"
	"github.com/aressim/internal/scenario"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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

// seedDefaults writes the default scenario plus the YAML-backed unit and weapon
// libraries into the database. Uses UPSERT semantics so it is safe to call on
// every startup — existing records are overwritten, ensuring library updates
// are always applied.
func (a *App) seedDefaults(ctx context.Context, cfg db.Config) {
	slog.Info("seeding defaults")

	for _, scen := range scenario.Builtins() {
		if err := a.scenRepo.Save(ctx, scen.Id, scenarioRecord(scen)); err != nil {
			slog.Warn("seed built-in scenario", "id", scen.Id, "name", scen.Name, "err", err)
		}
	}

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

	for _, wd := range scenario.DefaultWeaponDefinitions() {
		if err := a.weaponDefRepo.Save(ctx, wd.Id, weaponDefinitionRecord(wd)); err != nil {
			slog.Warn("seed weapon definition", "id", wd.Id, "err", err)
		}
	}

	slog.Info("seeding complete",
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
