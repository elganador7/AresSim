package repository

import (
	"context"
	"fmt"

	surrealdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// UnitDefRecord is the map shape for a unit_definition row.
type UnitDefRecord = map[string]any

// UnitDefRepo handles unit definition persistence.
type UnitDefRepo struct {
	db *surrealdb.DB
}

// NewUnitDefRepo creates a UnitDefRepo backed by the given connection.
func NewUnitDefRepo(db *surrealdb.DB) *UnitDefRepo {
	return &UnitDefRepo{db: db}
}

// Save writes or updates a unit definition record.
func (r *UnitDefRepo) Save(ctx context.Context, id string, record UnitDefRecord) error {
	rid := models.RecordID{Table: "unit_definition", ID: id}
	if _, err := surrealdb.Upsert[UnitDefRecord](ctx, r.db, rid, sanitizeRecord(record)); err != nil {
		return fmt.Errorf("save unit_definition %s: %w", id, err)
	}
	return nil
}

// Exists returns true if a unit definition with the given id exists.
func (r *UnitDefRepo) Exists(ctx context.Context, id string) bool {
	rid := models.RecordID{Table: "unit_definition", ID: id}
	result, err := surrealdb.Select[UnitDefRecord](ctx, r.db, rid)
	return err == nil && result != nil
}

// Get returns a single unit definition by id.
func (r *UnitDefRepo) Get(ctx context.Context, id string) (UnitDefRecord, error) {
	rid := models.RecordID{Table: "unit_definition", ID: id}
	result, err := surrealdb.Select[UnitDefRecord](ctx, r.db, rid)
	if err != nil {
		return nil, fmt.Errorf("get unit_definition %s: %w", id, err)
	}
	return *result, nil
}

// List returns all unit definitions ordered by domain, then general_type, then name.
func (r *UnitDefRepo) List(ctx context.Context) ([]UnitDefRecord, error) {
	results, err := surrealdb.Query[[]UnitDefRecord](
		ctx, r.db,
		"SELECT * FROM unit_definition ORDER BY domain, general_type, name",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("list unit_definitions: %w", err)
	}
	return flattenQueryResults(results), nil
}

// Delete removes a unit definition by id.
func (r *UnitDefRepo) Delete(ctx context.Context, id string) error {
	rid := models.RecordID{Table: "unit_definition", ID: id}
	if _, err := surrealdb.Delete[UnitDefRecord](ctx, r.db, rid); err != nil {
		return fmt.Errorf("delete unit_definition %s: %w", id, err)
	}
	return nil
}
