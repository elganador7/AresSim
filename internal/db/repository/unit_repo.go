// Package repository provides typed data-access operations against SurrealDB.
//
// Each repo method works with map[string]any records. The adjudicator layer
// is responsible for converting between proto messages and these maps.
// This keeps the repository agnostic of protobuf and easy to test with
// plain Go maps.
package repository

import (
	"context"
	"fmt"

	surrealdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// UnitRecord is the canonical map shape for a unit row in SurrealDB.
// Fields match the DEFINE FIELD statements in schema.go exactly.
// The "id" key, when present, must be a models.RecordID with Table="unit".
type UnitRecord = map[string]any

// UnitRepo provides CRUD and batch operations for the unit table.
type UnitRepo struct {
	db *surrealdb.DB
}

// NewUnitRepo creates a UnitRepo backed by the given connection.
func NewUnitRepo(db *surrealdb.DB) *UnitRepo {
	return &UnitRepo{db: db}
}

// Upsert writes a single unit record, creating or fully replacing it.
// The record must include an "id" field of type models.RecordID.
func (r *UnitRepo) Upsert(ctx context.Context, unitID string, record UnitRecord) error {
	rid := models.RecordID{Table: "unit", ID: unitID}
	if _, err := surrealdb.Upsert[UnitRecord](ctx, r.db, rid, sanitizeRecord(record)); err != nil {
		return fmt.Errorf("upsert unit %s: %w", unitID, err)
	}
	return nil
}

// UpsertBatch writes multiple unit records in a single transaction.
// This is the hot path for checkpoint writes (up to 1,000 units per call).
//
// The "id" key in each record must be a models.RecordID; the table name
// is populated automatically from the slice of (id, record) pairs.
func (r *UnitRepo) UpsertBatch(ctx context.Context, units []UnitRecord) error {
	if len(units) == 0 {
		return nil
	}

	// Build one multi-statement transaction so we get atomicity.
	// SurrealDB 2.x supports UPSERT inside a transaction.
	query := "BEGIN TRANSACTION;\n"
	params := make(map[string]any, len(units))

	for i, u := range units {
		rid, ok := u["id"].(models.RecordID)
		if !ok {
			return fmt.Errorf("unit record at index %d missing valid RecordID", i)
		}
		paramKey := fmt.Sprintf("u%d", i)
		idKey := fmt.Sprintf("id%d", i)
		params[paramKey] = u
		params[idKey] = rid.ID

		// Remove the id from the data payload — the UPSERT target is the rid.
		// We write a full canonical unit row on each checkpoint, so use CONTENT
		// rather than MERGE. This avoids SurrealDB retaining or synthesizing NULL
		// for omitted option<T> fields like attack_order.
		data := make(UnitRecord, len(u))
		for k, v := range u {
			if k != "id" {
				data[k] = v
			}
		}
		params[paramKey] = sanitizeRecord(data)
		query += fmt.Sprintf("UPSERT type::record('unit', $%s) CONTENT $%s;\n", idKey, paramKey)
	}
	query += "COMMIT TRANSACTION;"

	if _, err := surrealdb.Query[any](ctx, r.db, query, params); err != nil {
		return fmt.Errorf("upsert batch (%d units): %w", len(units), err)
	}
	return nil
}

// GetByID returns the raw record for a single unit.
func (r *UnitRepo) GetByID(ctx context.Context, unitID string) (UnitRecord, error) {
	rid := models.RecordID{Table: "unit", ID: unitID}
	result, err := surrealdb.Select[UnitRecord](ctx, r.db, rid)
	if err != nil {
		return nil, fmt.Errorf("get unit %s: %w", unitID, err)
	}
	return *result, nil
}

// GetAllActive returns all unit records where is_active = true.
// Called on scenario load to populate the adjudicator's in-memory cache.
func (r *UnitRepo) GetAllActive(ctx context.Context) ([]UnitRecord, error) {
	results, err := surrealdb.Query[[]UnitRecord](
		ctx, r.db,
		"SELECT * FROM unit WHERE is_active = true",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("get all active units: %w", err)
	}
	return flattenQueryResults(results), nil
}

// MarkDestroyed sets is_active = false for a unit.
// The record is retained for the event log and post-game analytics.
func (r *UnitRepo) MarkDestroyed(ctx context.Context, unitID string) error {
	_, err := surrealdb.Merge[UnitRecord](
		ctx, r.db,
		models.RecordID{Table: "unit", ID: unitID},
		map[string]any{"is_active": false},
	)
	if err != nil {
		return fmt.Errorf("mark unit %s destroyed: %w", unitID, err)
	}
	return nil
}

// UpdateStatus merges only the status scalar fields for a unit.
// Used when the adjudicator needs to persist a status change without
// a full checkpoint (e.g., unit just routed).
func (r *UnitRepo) UpdateStatus(ctx context.Context, unitID string, status map[string]any) error {
	_, err := surrealdb.Merge[UnitRecord](
		ctx, r.db,
		models.RecordID{Table: "unit", ID: unitID},
		sanitizeRecord(status),
	)
	if err != nil {
		return fmt.Errorf("update status for unit %s: %w", unitID, err)
	}
	return nil
}

// DeleteAll removes all unit records. Called when loading a new scenario.
func (r *UnitRepo) DeleteAll(ctx context.Context) error {
	_, err := surrealdb.Query[any](ctx, r.db, "DELETE unit", nil)
	if err != nil {
		return fmt.Errorf("delete all units: %w", err)
	}
	return nil
}
