package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aressim/internal/db/repository"
)

// CheckpointInterval is the number of adjudicator ticks between checkpoint
// writes. At 10 Hz this means a persist every 500ms.
const CheckpointInterval = 5

// Snapshot is a point-in-time view of the adjudicator state passed to the
// checkpoint writer. It contains pre-converted map records — the adjudicator
// is responsible for converting proto messages to maps before calling Write.
// This keeps the db package free of proto imports and allows the adjudicator
// to take the snapshot without holding its state lock during DB I/O.
type Snapshot struct {
	ScenarioID string
	Tick       int64
	SimSeconds float64

	// Units contains one UnitRecord per unit that is dirty (changed this tick
	// window). On the very first checkpoint all units are included.
	Units []repository.UnitRecord

	// Nodes contains one NodeRecord per logistics node that changed.
	Nodes []repository.NodeRecord
}

// CheckpointWriter coordinates a consistent batch write for one tick window.
// It is not safe for concurrent use — the adjudicator calls it from a single
// goroutine that is separate from the hot tick loop.
type CheckpointWriter struct {
	units    *repository.UnitRepo
	nodes    *repository.NodeRepo
	scenario *repository.ScenarioRepo
}

// NewCheckpointWriter creates a CheckpointWriter using the given repos.
func NewCheckpointWriter(
	units *repository.UnitRepo,
	nodes *repository.NodeRepo,
	scenario *repository.ScenarioRepo,
) *CheckpointWriter {
	return &CheckpointWriter{
		units:    units,
		nodes:    nodes,
		scenario: scenario,
	}
}

// Write persists the snapshot to SurrealDB.
//
// The write sequence is:
//  1. Batch-upsert all dirty units.
//  2. Batch-upsert all dirty logistics nodes.
//  3. Update scenario progress (last_tick, last_sim_seconds).
//  4. Write a checkpoint marker record.
//
// Steps 1–2 are issued as separate transactions for batching efficiency.
// If any step fails, the error is returned but the sim continues running —
// the next checkpoint will retry. SurrealDB data is advisory persistence;
// the adjudicator's sync.Map is the authoritative runtime state.
func (cw *CheckpointWriter) Write(ctx context.Context, snap Snapshot) error {
	wallTime := float64(time.Now().UnixMilli()) / 1000.0

	if err := cw.units.UpsertBatch(ctx, snap.Units); err != nil {
		return fmt.Errorf("checkpoint tick %d units: %w", snap.Tick, err)
	}

	if err := cw.nodes.UpsertBatch(ctx, snap.Nodes); err != nil {
		return fmt.Errorf("checkpoint tick %d nodes: %w", snap.Tick, err)
	}

	if err := cw.scenario.UpdateProgress(ctx, snap.ScenarioID, snap.Tick, snap.SimSeconds); err != nil {
		return fmt.Errorf("checkpoint tick %d scenario progress: %w", snap.Tick, err)
	}

	if err := cw.scenario.WriteCheckpointMarker(ctx, snap.ScenarioID, snap.Tick, snap.SimSeconds, wallTime); err != nil {
		return fmt.Errorf("checkpoint tick %d marker: %w", snap.Tick, err)
	}

	return nil
}
