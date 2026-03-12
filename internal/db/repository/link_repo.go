package repository

import (
	"context"
	"fmt"

	surrealdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// LinkRepo manages the supply_link and c2_link graph relation tables.
type LinkRepo struct {
	db *surrealdb.DB
}

// NewLinkRepo creates a LinkRepo backed by the given connection.
func NewLinkRepo(db *surrealdb.DB) *LinkRepo {
	return &LinkRepo{db: db}
}

// CreateSupplyLink creates a directed supply_link relation from one node/unit to another.
// fromTable and toTable must be "logistics_node" or "unit".
func (r *LinkRepo) CreateSupplyLink(ctx context.Context, linkID, fromTable, fromID, toTable, toID string, fields map[string]any) error {
	rel := &surrealdb.Relationship{
		ID: &models.RecordID{Table: "supply_link", ID: linkID},
		In: models.RecordID{Table: fromTable, ID: fromID},
		Out: models.RecordID{Table: toTable, ID: toID},
		Data: fields,
	}
	if _, err := surrealdb.Relate[map[string]any](ctx, r.db, rel); err != nil {
		return fmt.Errorf("create supply_link %s: %w", linkID, err)
	}
	return nil
}

// CreateC2Link creates a directed c2_link relation from parent unit to child unit.
// kind must be "organic", "opcon", or "attached".
func (r *LinkRepo) CreateC2Link(ctx context.Context, parentID, childID, kind string, simSeconds float64) error {
	// Use a deterministic ID based on parent+child so duplicate creates are idempotent.
	linkID := fmt.Sprintf("%s_%s", parentID, childID)
	rel := &surrealdb.Relationship{
		ID:  &models.RecordID{Table: "c2_link", ID: linkID},
		In:  models.RecordID{Table: "unit", ID: parentID},
		Out: models.RecordID{Table: "unit", ID: childID},
		Data: map[string]any{
			"link_kind":      kind,
			"established_at": simSeconds,
		},
	}
	if _, err := surrealdb.Relate[map[string]any](ctx, r.db, rel); err != nil {
		return fmt.Errorf("create c2_link %s→%s: %w", parentID, childID, err)
	}
	return nil
}

// DeleteC2Link removes the c2_link between parent and child (used when ATTACH order executes).
func (r *LinkRepo) DeleteC2Link(ctx context.Context, parentID, childID string) error {
	linkID := fmt.Sprintf("%s_%s", parentID, childID)
	rid := models.RecordID{Table: "c2_link", ID: linkID}
	if _, err := surrealdb.Delete[map[string]any](ctx, r.db, rid); err != nil {
		return fmt.Errorf("delete c2_link %s→%s: %w", parentID, childID, err)
	}
	return nil
}

// GetSupplyLinksForSide returns all supply_link relations where the source node
// belongs to the given side. Used by the Phase 2 supply graph traversal.
func (r *LinkRepo) GetSupplyLinksForSide(ctx context.Context, side string) ([]map[string]any, error) {
	results, err := surrealdb.Query[[]map[string]any](
		ctx, r.db,
		`SELECT *, in.side AS from_side, out.side AS to_side
		 FROM supply_link
		 WHERE in.side = $side AND is_active = true
		 FETCH in, out`,
		map[string]any{"side": side},
	)
	if err != nil {
		return nil, fmt.Errorf("get supply links for side %q: %w", side, err)
	}
	return flattenQueryResults(results), nil
}

// GetSubordinates returns all unit IDs that are direct organic subordinates
// of the given parent unit, via c2_link traversal.
func (r *LinkRepo) GetSubordinates(ctx context.Context, parentID string) ([]string, error) {
	results, err := surrealdb.Query[[]map[string]any](
		ctx, r.db,
		`SELECT out.id AS child_id
		 FROM c2_link
		 WHERE in = unit:$parent_id AND link_kind = 'organic'`,
		map[string]any{"parent_id": parentID},
	)
	if err != nil {
		return nil, fmt.Errorf("get subordinates of unit %s: %w", parentID, err)
	}
	rows := flattenQueryResults(results)
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if id, ok := row["child_id"].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// DeleteAllLinks removes all supply_link and c2_link records.
// Called when loading a new scenario.
func (r *LinkRepo) DeleteAllLinks(ctx context.Context) error {
	if _, err := surrealdb.Query[any](ctx, r.db, "DELETE supply_link", nil); err != nil {
		return fmt.Errorf("delete supply links: %w", err)
	}
	if _, err := surrealdb.Query[any](ctx, r.db, "DELETE c2_link", nil); err != nil {
		return fmt.Errorf("delete c2 links: %w", err)
	}
	return nil
}
