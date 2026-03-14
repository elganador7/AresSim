package db

import (
	"context"
	"fmt"

	surrealdb "github.com/surrealdb/surrealdb.go"
)

// schemaVersion is incremented whenever the schema changes in a
// backwards-incompatible way. On startup, if the stored version differs from
// this constant, the database is wiped and rebuilt from scratch. This is
// acceptable for a single-player desktop application where the database is
// purely derived state (the authoritative record lives in scenario files).
const schemaVersion = 9

// SchemaVersion returns the current schema version for logging.
func SchemaVersion() int { return schemaVersion }

// EnsureSchema checks the stored schema version. If it differs from
// schemaVersion (or no version record exists), it wipes all tables and
// rebuilds the schema from scratch, then writes the new version record.
// For a single-player app with derived-only DB state this is safe.
func EnsureSchema(ctx context.Context, db *surrealdb.DB) error {
	const versionKey = "schema_meta:version"

	type versionRecord struct {
		Value int `json:"value"`
	}

	// Read stored version (may not exist yet).
	results, err := surrealdb.Query[[]versionRecord](ctx, db,
		"SELECT value FROM schema_meta:version",
		nil,
	)
	stored := 0
	if err == nil {
		rows := flattenVersionResults(results)
		if len(rows) > 0 {
			stored = rows[0].Value
		}
	}

	if stored != schemaVersion {
		// Wipe all user tables and rebuild.
		for _, tbl := range []string{"unit", "scenario", "checkpoint", "unit_definition", "weapon_definition", "schema_meta"} {
			if _, err := surrealdb.Query[any](ctx, db,
				fmt.Sprintf("REMOVE TABLE IF EXISTS %s", tbl), nil); err != nil {
				return fmt.Errorf("wipe table %s: %w", tbl, err)
			}
		}
		if err := ApplySchema(ctx, db); err != nil {
			return err
		}
		// Persist the new version.
		if _, err := surrealdb.Query[any](ctx, db,
			fmt.Sprintf("UPSERT schema_meta:version SET value = %d", schemaVersion),
			nil,
		); err != nil {
			return fmt.Errorf("write schema version: %w", err)
		}
	}
	return nil
}

// ApplySchema runs all DEFINE statements against the connected database.
// Every statement is idempotent (IF NOT EXISTS), so this is safe to call on
// every application startup.
func ApplySchema(ctx context.Context, db *surrealdb.DB) error {
	for i, stmt := range schemaStatements {
		if _, err := surrealdb.Query[any](ctx, db, stmt, nil); err != nil {
			return fmt.Errorf("schema statement %d: %w\nSQL: %s", i, err, stmt)
		}
	}
	return nil
}

func flattenVersionResults[T any](results *[]surrealdb.QueryResult[[]T]) []T {
	if results == nil {
		return nil
	}
	var out []T
	for _, r := range *results {
		out = append(out, r.Result...)
	}
	return out
}

// schemaStatements contains every DEFINE statement in dependency order.
// Indexes are defined after their tables. Relation tables after their endpoint tables.
var schemaStatements = []string{

	// ── unit ──────────────────────────────────────────────────────────────────
	// Core simulation entity. Indexed fields support the adjudicator's hot-path
	// queries (all units by side, all active units). Capabilities and orders are
	// stored as proto binary blobs because they are deeply nested and not queried.

	`DEFINE TABLE IF NOT EXISTS unit SCHEMAFULL`,

	// Identity
	`DEFINE FIELD IF NOT EXISTS display_name      ON unit TYPE string`,
	`DEFINE FIELD IF NOT EXISTS full_name         ON unit TYPE string`,
	`DEFINE FIELD IF NOT EXISTS side              ON unit TYPE string`,
	`DEFINE FIELD IF NOT EXISTS nation_id         ON unit TYPE option<string>`,
	`DEFINE FIELD IF NOT EXISTS nato_symbol_sidc  ON unit TYPE string`,
	`DEFINE FIELD IF NOT EXISTS definition_id     ON unit TYPE option<string>`,

	// Posture (stored as int32 proto enum value)
	`DEFINE FIELD IF NOT EXISTS posture           ON unit TYPE int`,

	// Geospatial — geometry<point> enables spatial range queries in Phase 2.
	// SurrealDB geometry uses GeoJSON coordinate order: [longitude, latitude].
	`DEFINE FIELD IF NOT EXISTS position          ON unit TYPE geometry<point>`,
	`DEFINE FIELD IF NOT EXISTS alt_msl           ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS heading           ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS speed             ON unit TYPE float`,

	// Status: mutable scalar fields; updated every checkpoint.
	`DEFINE FIELD IF NOT EXISTS personnel_strength    ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS equipment_strength    ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS combat_effectiveness  ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS fuel_level_liters     ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS morale                ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS fatigue               ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS is_active             ON unit TYPE bool`,

	// Combat effect flags
	`DEFINE FIELD IF NOT EXISTS suppressed        ON unit TYPE bool`,
	`DEFINE FIELD IF NOT EXISTS disrupted         ON unit TYPE bool`,
	`DEFINE FIELD IF NOT EXISTS routing           ON unit TYPE bool`,

	// C2 hierarchy
	`DEFINE FIELD IF NOT EXISTS parent_unit_id    ON unit TYPE option<string>`,

	// Indexes
	`DEFINE INDEX IF NOT EXISTS idx_unit_side     ON unit FIELDS side`,
	`DEFINE INDEX IF NOT EXISTS idx_unit_active   ON unit FIELDS is_active`,
	`DEFINE INDEX IF NOT EXISTS idx_unit_pos      ON unit FIELDS position`,

	// ── unit_definition ───────────────────────────────────────────────────────
	// Canonical platform templates. Units reference these by definition_id slug.
	// Stored as flat scalar fields for queryability.

	`DEFINE TABLE IF NOT EXISTS unit_definition SCHEMAFULL`,

	`DEFINE FIELD IF NOT EXISTS name               ON unit_definition TYPE string`,
	`DEFINE FIELD IF NOT EXISTS description        ON unit_definition TYPE string`,
	`DEFINE FIELD IF NOT EXISTS domain             ON unit_definition TYPE int`,
	`DEFINE FIELD IF NOT EXISTS form               ON unit_definition TYPE int`,
	`DEFINE FIELD IF NOT EXISTS general_type       ON unit_definition TYPE int`,
	`DEFINE FIELD IF NOT EXISTS specific_type      ON unit_definition TYPE string`,
	`DEFINE FIELD IF NOT EXISTS nation_of_origin   ON unit_definition TYPE string`,
	`DEFINE FIELD IF NOT EXISTS service_entry_year ON unit_definition TYPE int`,
	`DEFINE FIELD IF NOT EXISTS base_strength      ON unit_definition TYPE float`,
	`DEFINE FIELD IF NOT EXISTS accuracy           ON unit_definition TYPE float`,
	`DEFINE FIELD IF NOT EXISTS max_speed_mps      ON unit_definition TYPE float`,
	`DEFINE FIELD IF NOT EXISTS cruise_speed_mps   ON unit_definition TYPE float`,
	`DEFINE FIELD IF NOT EXISTS max_range_km       ON unit_definition TYPE float`,
	`DEFINE FIELD IF NOT EXISTS survivability      ON unit_definition TYPE float`,
	`DEFINE FIELD IF NOT EXISTS detection_range_m  ON unit_definition TYPE float`,
	`DEFINE FIELD IF NOT EXISTS fuel_capacity_liters ON unit_definition TYPE float`,
	`DEFINE FIELD IF NOT EXISTS fuel_burn_rate_lph ON unit_definition TYPE float`,

	`DEFINE INDEX IF NOT EXISTS idx_unitdef_domain ON unit_definition FIELDS domain`,

	// ── scenario ──────────────────────────────────────────────────────────────
	// Scenario metadata + proto blob. The unit table holds live checkpointed
	// state; scenario_pb holds the immutable initial order of battle.

	`DEFINE TABLE IF NOT EXISTS scenario SCHEMAFULL`,

	`DEFINE FIELD IF NOT EXISTS name              ON scenario TYPE string`,
	`DEFINE FIELD IF NOT EXISTS description       ON scenario TYPE string`,
	`DEFINE FIELD IF NOT EXISTS author            ON scenario TYPE string`,
	`DEFINE FIELD IF NOT EXISTS classification    ON scenario TYPE string`,
	`DEFINE FIELD IF NOT EXISTS start_time_unix   ON scenario TYPE float`,
	`DEFINE FIELD IF NOT EXISTS schema_version    ON scenario TYPE string`,
	`DEFINE FIELD IF NOT EXISTS tick_rate_hz      ON scenario TYPE float`,
	`DEFINE FIELD IF NOT EXISTS time_scale        ON scenario TYPE float`,
	`DEFINE FIELD IF NOT EXISTS adj_model         ON scenario TYPE option<int>`,
	`DEFINE FIELD IF NOT EXISTS scenario_pb       ON scenario TYPE string`,
	`DEFINE FIELD IF NOT EXISTS last_tick         ON scenario TYPE int`,
	`DEFINE FIELD IF NOT EXISTS last_sim_seconds  ON scenario TYPE float`,

	`DEFINE INDEX IF NOT EXISTS idx_scenario_name ON scenario FIELDS name`,

	// ── checkpoint ────────────────────────────────────────────────────────────
	// Lightweight tick markers written every N ticks. The unit/node tables
	// ARE the checkpoint state; this table just records when it happened.

	`DEFINE TABLE IF NOT EXISTS checkpoint SCHEMAFULL`,

	`DEFINE FIELD IF NOT EXISTS scenario_id      ON checkpoint TYPE string`,
	`DEFINE FIELD IF NOT EXISTS tick_number      ON checkpoint TYPE int`,
	`DEFINE FIELD IF NOT EXISTS sim_seconds      ON checkpoint TYPE float`,
	`DEFINE FIELD IF NOT EXISTS wall_time        ON checkpoint TYPE float`,

	`DEFINE INDEX IF NOT EXISTS idx_ckpt_scenario ON checkpoint FIELDS scenario_id`,
	`DEFINE INDEX IF NOT EXISTS idx_ckpt_tick      ON checkpoint FIELDS tick_number`,

	// ── weapon_definition ─────────────────────────────────────────────────────
	// Global munition catalog seeded from DefaultWeaponDefinitions() on startup.
	// domain_targets stored as a JSON array of int (UnitDomain enum values).

	`DEFINE TABLE IF NOT EXISTS weapon_definition SCHEMAFULL`,

	`DEFINE FIELD IF NOT EXISTS name               ON weapon_definition TYPE string`,
	`DEFINE FIELD IF NOT EXISTS description        ON weapon_definition TYPE string`,
	`DEFINE FIELD IF NOT EXISTS domain_targets     ON weapon_definition TYPE array<int>`,
	`DEFINE FIELD IF NOT EXISTS speed_mps          ON weapon_definition TYPE float`,
	`DEFINE FIELD IF NOT EXISTS range_m            ON weapon_definition TYPE float`,
	`DEFINE FIELD IF NOT EXISTS probability_of_hit ON weapon_definition TYPE float`,

	`DEFINE INDEX IF NOT EXISTS idx_weapondef_name ON weapon_definition FIELDS name`,
}
