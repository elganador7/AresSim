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
const schemaVersion = 1

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

	// Classification (stored as int32 proto enum values)
	`DEFINE FIELD IF NOT EXISTS echelon           ON unit TYPE int`,
	`DEFINE FIELD IF NOT EXISTS domain            ON unit TYPE int`,
	`DEFINE FIELD IF NOT EXISTS unit_function     ON unit TYPE int`,
	`DEFINE FIELD IF NOT EXISTS unit_type         ON unit TYPE int`,
	`DEFINE FIELD IF NOT EXISTS posture           ON unit TYPE int`,

	// Geospatial — geometry<point> enables spatial range queries in Phase 2.
	// SurrealDB geometry uses GeoJSON coordinate order: [longitude, latitude].
	`DEFINE FIELD IF NOT EXISTS position          ON unit TYPE geometry<point>`,
	`DEFINE FIELD IF NOT EXISTS alt_msl           ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS heading           ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS speed             ON unit TYPE float`,

	// Capabilities: immutable blob; written once at spawn, never updated.
	`DEFINE FIELD IF NOT EXISTS capabilities_pb   ON unit TYPE bytes`,

	// Status: mutable scalar fields; updated every checkpoint.
	`DEFINE FIELD IF NOT EXISTS personnel_strength    ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS equipment_strength    ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS combat_effectiveness  ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS fuel_level_liters     ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS ammo_primary_pct      ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS ammo_secondary_pct    ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS ammo_missile_pct      ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS morale                ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS fatigue               ON unit TYPE float`,
	`DEFINE FIELD IF NOT EXISTS readiness             ON unit TYPE int`,
	`DEFINE FIELD IF NOT EXISTS is_active             ON unit TYPE bool`,

	// Combat effect flags (bool; fast for casualty reports and analytics).
	`DEFINE FIELD IF NOT EXISTS suppressed        ON unit TYPE bool`,
	`DEFINE FIELD IF NOT EXISTS disrupted         ON unit TYPE bool`,
	`DEFINE FIELD IF NOT EXISTS routing           ON unit TYPE bool`,
	`DEFINE FIELD IF NOT EXISTS exhausted         ON unit TYPE bool`,
	`DEFINE FIELD IF NOT EXISTS mobility_kill     ON unit TYPE bool`,

	// C2 hierarchy (denormalized IDs; graph edges are in c2_link table).
	`DEFINE FIELD IF NOT EXISTS parent_unit_id    ON unit TYPE option<string>`,

	// Orders blob: written when orders change, not every tick.
	`DEFINE FIELD IF NOT EXISTS orders_pb         ON unit TYPE bytes`,

	// Indexes
	`DEFINE INDEX IF NOT EXISTS idx_unit_side     ON unit FIELDS side`,
	`DEFINE INDEX IF NOT EXISTS idx_unit_active   ON unit FIELDS is_active`,
	`DEFINE INDEX IF NOT EXISTS idx_unit_domain   ON unit FIELDS domain`,
	`DEFINE INDEX IF NOT EXISTS idx_unit_pos      ON unit FIELDS position`,

	// ── logistics_node ────────────────────────────────────────────────────────

	`DEFINE TABLE IF NOT EXISTS logistics_node SCHEMAFULL`,

	`DEFINE FIELD IF NOT EXISTS name              ON logistics_node TYPE string`,
	`DEFINE FIELD IF NOT EXISTS side              ON logistics_node TYPE string`,
	`DEFINE FIELD IF NOT EXISTS nation_id         ON logistics_node TYPE option<string>`,
	`DEFINE FIELD IF NOT EXISTS node_type         ON logistics_node TYPE int`,
	`DEFINE FIELD IF NOT EXISTS position          ON logistics_node TYPE geometry<point>`,
	`DEFINE FIELD IF NOT EXISTS alt_msl           ON logistics_node TYPE float`,

	// Current stock levels (flat for fast update queries)
	`DEFINE FIELD IF NOT EXISTS fuel_liters       ON logistics_node TYPE float`,
	`DEFINE FIELD IF NOT EXISTS ammo_units        ON logistics_node TYPE float`,
	`DEFINE FIELD IF NOT EXISTS food_rations      ON logistics_node TYPE float`,
	`DEFINE FIELD IF NOT EXISTS spare_parts       ON logistics_node TYPE float`,
	`DEFINE FIELD IF NOT EXISTS medical_supplies  ON logistics_node TYPE float`,

	// Capacity ceiling (written once, not updated)
	`DEFINE FIELD IF NOT EXISTS cap_fuel_liters   ON logistics_node TYPE float`,
	`DEFINE FIELD IF NOT EXISTS cap_ammo_units    ON logistics_node TYPE float`,

	`DEFINE FIELD IF NOT EXISTS replenishment_rate    ON logistics_node TYPE float`,
	`DEFINE FIELD IF NOT EXISTS parent_node_id        ON logistics_node TYPE option<string>`,
	`DEFINE FIELD IF NOT EXISTS health                ON logistics_node TYPE float`,
	`DEFINE FIELD IF NOT EXISTS is_active             ON logistics_node TYPE bool`,
	`DEFINE FIELD IF NOT EXISTS is_supplied           ON logistics_node TYPE bool`,
	`DEFINE FIELD IF NOT EXISTS strategic_source_id   ON logistics_node TYPE option<string>`,

	`DEFINE INDEX IF NOT EXISTS idx_node_side     ON logistics_node FIELDS side`,
	`DEFINE INDEX IF NOT EXISTS idx_node_active   ON logistics_node FIELDS is_active`,
	`DEFINE INDEX IF NOT EXISTS idx_node_pos      ON logistics_node FIELDS position`,

	// ── supply_link ───────────────────────────────────────────────────────────
	// Graph relation. The adjudicator queries this with SurrealDB path syntax:
	//   SELECT <-supply_link<-logistics_node FROM logistics_node:depot_id
	// In Phase 2, traversal: SELECT ->supply_link->logistics_node FROM unit:id

	`DEFINE TABLE IF NOT EXISTS supply_link TYPE RELATION SCHEMAFULL`,

	`DEFINE FIELD IF NOT EXISTS link_type         ON supply_link TYPE int`,
	`DEFINE FIELD IF NOT EXISTS bandwidth         ON supply_link TYPE float`,
	`DEFINE FIELD IF NOT EXISTS length_m          ON supply_link TYPE float`,
	`DEFINE FIELD IF NOT EXISTS is_active         ON supply_link TYPE bool`,
	`DEFINE FIELD IF NOT EXISTS is_interdicted    ON supply_link TYPE bool`,
	`DEFINE FIELD IF NOT EXISTS vulnerability     ON supply_link TYPE float`,
	`DEFINE FIELD IF NOT EXISTS commodity_id      ON supply_link TYPE option<string>`,

	`DEFINE INDEX IF NOT EXISTS idx_link_active   ON supply_link FIELDS is_active`,

	// ── c2_link ───────────────────────────────────────────────────────────────
	// C2 hierarchy graph. Parent unit → child unit.
	// Queried to determine command radius effects and disruption propagation.

	`DEFINE TABLE IF NOT EXISTS c2_link TYPE RELATION SCHEMAFULL`,

	// "organic" | "opcon" | "attached"
	`DEFINE FIELD IF NOT EXISTS link_kind         ON c2_link TYPE string`,
	`DEFINE FIELD IF NOT EXISTS established_at    ON c2_link TYPE float`,

	`DEFINE INDEX IF NOT EXISTS idx_c2_in         ON c2_link FIELDS in`,
	`DEFINE INDEX IF NOT EXISTS idx_c2_out        ON c2_link FIELDS out`,

	// ── scenario ──────────────────────────────────────────────────────────────
	// Scenario metadata + full proto blob for round-trip fidelity.
	// The unit and logistics_node tables hold the live (checkpointed) state;
	// scenario_pb holds the immutable initial conditions (order of battle).

	`DEFINE TABLE IF NOT EXISTS scenario SCHEMAFULL`,

	`DEFINE FIELD IF NOT EXISTS name              ON scenario TYPE string`,
	`DEFINE FIELD IF NOT EXISTS description       ON scenario TYPE string`,
	`DEFINE FIELD IF NOT EXISTS author            ON scenario TYPE string`,
	`DEFINE FIELD IF NOT EXISTS classification    ON scenario TYPE string`,
	`DEFINE FIELD IF NOT EXISTS start_time_unix   ON scenario TYPE float`,
	`DEFINE FIELD IF NOT EXISTS schema_version    ON scenario TYPE string`,
	`DEFINE FIELD IF NOT EXISTS tick_rate_hz      ON scenario TYPE float`,
	`DEFINE FIELD IF NOT EXISTS time_scale        ON scenario TYPE float`,
	`DEFINE FIELD IF NOT EXISTS adj_model         ON scenario TYPE int`,
	`DEFINE FIELD IF NOT EXISTS scenario_pb       ON scenario TYPE bytes`,
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
}
