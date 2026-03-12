package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aressim/internal/db/repository"
)

// CheckpointInterval is the number of adjudicator ticks between checkpoint writes.
// At 10 Hz this means a persist every 500ms.
const CheckpointInterval = 5

// Snapshot is a point-in-time view of the adjudicator state passed to the
// checkpoint writer. Units are pre-converted to map records so the db package
// stays free of proto imports.
type Snapshot struct {
	ScenarioID string
	Tick       int64
	SimSeconds float64
	Units      []repository.UnitRecord
}

// CheckpointWriter coordinates a consistent batch write for one tick window.
type CheckpointWriter struct {
	units    *repository.UnitRepo
	scenario *repository.ScenarioRepo
}

func NewCheckpointWriter(units *repository.UnitRepo, scenario *repository.ScenarioRepo) *CheckpointWriter {
	return &CheckpointWriter{units: units, scenario: scenario}
}

// Write persists the snapshot to SurrealDB. Errors are non-fatal — the sim
// continues and the next checkpoint will retry.
func (cw *CheckpointWriter) Write(ctx context.Context, snap Snapshot) error {
	wallTime := float64(time.Now().UnixMilli()) / 1000.0

	if err := cw.units.UpsertBatch(ctx, snap.Units); err != nil {
		return fmt.Errorf("checkpoint tick %d units: %w", snap.Tick, err)
	}
	if err := cw.scenario.UpdateProgress(ctx, snap.ScenarioID, snap.Tick, snap.SimSeconds); err != nil {
		return fmt.Errorf("checkpoint tick %d scenario progress: %w", snap.Tick, err)
	}
	if err := cw.scenario.WriteCheckpointMarker(ctx, snap.ScenarioID, snap.Tick, snap.SimSeconds, wallTime); err != nil {
		return fmt.Errorf("checkpoint tick %d marker: %w", snap.Tick, err)
	}
	return nil
}
