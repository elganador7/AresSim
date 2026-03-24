package repository

import (
	"context"
	"fmt"

	surrealdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// ScenarioRecord is the map shape for a scenario row.
type ScenarioRecord = map[string]any

// ScenarioRepo handles scenario metadata and checkpoint marker persistence.
type ScenarioRepo struct {
	db *surrealdb.DB
}

// NewScenarioRepo creates a ScenarioRepo backed by the given connection.
func NewScenarioRepo(db *surrealdb.DB) *ScenarioRepo {
	return &ScenarioRepo{db: db}
}

// Save writes or updates the scenario record.
// record must include metadata fields plus a "scenario_pb" bytes field
// containing the proto-marshaled Scenario message.
func (r *ScenarioRepo) Save(ctx context.Context, scenarioID string, record ScenarioRecord) error {
	rid := models.RecordID{Table: "scenario", ID: scenarioID}
	if _, err := surrealdb.Upsert[ScenarioRecord](ctx, r.db, rid, sanitizeRecord(record)); err != nil {
		return fmt.Errorf("save scenario %s: %w", scenarioID, err)
	}
	return nil
}

// Get retrieves a scenario record by ID.
func (r *ScenarioRepo) Get(ctx context.Context, scenarioID string) (ScenarioRecord, error) {
	rid := models.RecordID{Table: "scenario", ID: scenarioID}
	result, err := surrealdb.Select[ScenarioRecord](ctx, r.db, rid)
	if err != nil {
		return nil, fmt.Errorf("get scenario %s: %w", scenarioID, err)
	}
	return *result, nil
}

// List returns metadata for all stored scenarios (without the proto blob).
// Used to populate the scenario picker in the UI.
func (r *ScenarioRepo) List(ctx context.Context) ([]ScenarioRecord, error) {
	results, err := surrealdb.Query[[]ScenarioRecord](
		ctx, r.db,
		"SELECT id, name, description, author, classification, last_tick, last_sim_seconds FROM scenario ORDER BY name",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("list scenarios: %w", err)
	}
	return flattenQueryResults(results), nil
}

// UpdateProgress updates the last_tick and last_sim_seconds fields on the
// scenario record. Called every checkpoint write.
func (r *ScenarioRepo) UpdateProgress(ctx context.Context, scenarioID string, tick int64, simSeconds float64) error {
	_, err := surrealdb.Merge[ScenarioRecord](
		ctx, r.db,
		models.RecordID{Table: "scenario", ID: scenarioID},
		sanitizeRecord(map[string]any{
			"last_tick":        tick,
			"last_sim_seconds": simSeconds,
		}),
	)
	if err != nil {
		return fmt.Errorf("update scenario progress %s: %w", scenarioID, err)
	}
	return nil
}

// WriteCheckpointMarker appends a checkpoint record for the given tick.
func (r *ScenarioRepo) WriteCheckpointMarker(ctx context.Context, scenarioID string, tick int64, simSeconds, wallTime float64) error {
	markerID := fmt.Sprintf("%s_tick_%d", scenarioID, tick)
	rid := models.RecordID{Table: "checkpoint", ID: markerID}
	_, err := surrealdb.Create[ScenarioRecord](ctx, r.db, rid, sanitizeRecord(map[string]any{
		"scenario_id": scenarioID,
		"tick_number": tick,
		"sim_seconds": simSeconds,
		"wall_time":   wallTime,
	}))
	if err != nil {
		return fmt.Errorf("write checkpoint marker tick %d: %w", tick, err)
	}
	return nil
}

// Delete removes a scenario and all its checkpoint markers from the database.
func (r *ScenarioRepo) Delete(ctx context.Context, scenarioID string) error {
	rid := models.RecordID{Table: "scenario", ID: scenarioID}
	if _, err := surrealdb.Delete[ScenarioRecord](ctx, r.db, rid); err != nil {
		return fmt.Errorf("delete scenario %s: %w", scenarioID, err)
	}
	if _, err := surrealdb.Query[any](ctx, r.db,
		"DELETE checkpoint WHERE scenario_id = $sid",
		map[string]any{"sid": scenarioID},
	); err != nil {
		return fmt.Errorf("delete checkpoints for scenario %s: %w", scenarioID, err)
	}
	return nil
}

// LatestCheckpointTick returns the highest tick_number recorded for a scenario.
// Returns 0 if no checkpoints exist (new scenario).
func (r *ScenarioRepo) LatestCheckpointTick(ctx context.Context, scenarioID string) (int64, error) {
	results, err := surrealdb.Query[[]map[string]any](
		ctx, r.db,
		"SELECT tick_number FROM checkpoint WHERE scenario_id = $sid ORDER BY tick_number DESC LIMIT 1",
		map[string]any{"sid": scenarioID},
	)
	if err != nil {
		return 0, fmt.Errorf("get latest checkpoint for scenario %s: %w", scenarioID, err)
	}
	rows := flattenQueryResults(results)
	if len(rows) == 0 {
		return 0, nil
	}
	tick, ok := rows[0]["tick_number"].(int64)
	if !ok {
		// SurrealDB may return numeric values as float64
		if f, ok2 := rows[0]["tick_number"].(float64); ok2 {
			return int64(f), nil
		}
	}
	return tick, nil
}
