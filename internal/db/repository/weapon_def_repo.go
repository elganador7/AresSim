package repository

import (
	"context"
	"fmt"

	surrealdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// WeaponDefRecord is the map shape for a weapon_definition row.
type WeaponDefRecord = map[string]any

// WeaponDefRepo handles weapon definition persistence.
type WeaponDefRepo struct {
	db *surrealdb.DB
}

// NewWeaponDefRepo creates a WeaponDefRepo backed by the given connection.
func NewWeaponDefRepo(db *surrealdb.DB) *WeaponDefRepo {
	return &WeaponDefRepo{db: db}
}

// Save writes or updates a weapon definition record.
func (r *WeaponDefRepo) Save(ctx context.Context, id string, record WeaponDefRecord) error {
	rid := models.RecordID{Table: "weapon_definition", ID: id}
	if _, err := surrealdb.Upsert[WeaponDefRecord](ctx, r.db, rid, record); err != nil {
		return fmt.Errorf("save weapon_definition %s: %w", id, err)
	}
	return nil
}

// Exists returns true if a weapon definition with the given id exists.
func (r *WeaponDefRepo) Exists(ctx context.Context, id string) bool {
	rid := models.RecordID{Table: "weapon_definition", ID: id}
	result, err := surrealdb.Select[WeaponDefRecord](ctx, r.db, rid)
	return err == nil && result != nil
}

// List returns all weapon definitions ordered by name.
func (r *WeaponDefRepo) List(ctx context.Context) ([]WeaponDefRecord, error) {
	results, err := surrealdb.Query[[]WeaponDefRecord](
		ctx, r.db,
		"SELECT * FROM weapon_definition ORDER BY name",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("list weapon_definitions: %w", err)
	}
	return flattenQueryResults(results), nil
}
