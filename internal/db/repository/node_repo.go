package repository

import (
	"context"
	"fmt"

	surrealdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// NodeRecord is the canonical map shape for a logistics_node row.
type NodeRecord = map[string]any

// NodeRepo provides CRUD operations for the logistics_node table.
type NodeRepo struct {
	db *surrealdb.DB
}

// NewNodeRepo creates a NodeRepo backed by the given connection.
func NewNodeRepo(db *surrealdb.DB) *NodeRepo {
	return &NodeRepo{db: db}
}

// Upsert writes a single logistics node record.
func (r *NodeRepo) Upsert(ctx context.Context, nodeID string, record NodeRecord) error {
	rid := models.RecordID{Table: "logistics_node", ID: nodeID}
	if _, err := surrealdb.Upsert[NodeRecord](ctx, r.db, rid, record); err != nil {
		return fmt.Errorf("upsert node %s: %w", nodeID, err)
	}
	return nil
}

// UpsertBatch writes multiple node records in one transaction.
func (r *NodeRepo) UpsertBatch(ctx context.Context, nodes []NodeRecord) error {
	if len(nodes) == 0 {
		return nil
	}

	query := "BEGIN TRANSACTION;\n"
	params := make(map[string]any, len(nodes))

	for i, n := range nodes {
		rid, ok := n["id"].(models.RecordID)
		if !ok {
			return fmt.Errorf("node record at index %d missing valid RecordID", i)
		}
		paramKey := fmt.Sprintf("n%d", i)
		data := make(NodeRecord, len(n))
		for k, v := range n {
			if k != "id" {
				data[k] = v
			}
		}
		params[paramKey] = data
		query += fmt.Sprintf("UPSERT logistics_node:%s MERGE $%s;\n", rid.ID, paramKey)
	}
	query += "COMMIT TRANSACTION;"

	if _, err := surrealdb.Query[any](ctx, r.db, query, params); err != nil {
		return fmt.Errorf("upsert batch (%d nodes): %w", len(nodes), err)
	}
	return nil
}

// GetAllActive returns all active logistics nodes.
func (r *NodeRepo) GetAllActive(ctx context.Context) ([]NodeRecord, error) {
	results, err := surrealdb.Query[[]NodeRecord](
		ctx, r.db,
		"SELECT * FROM logistics_node WHERE is_active = true",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("get all active nodes: %w", err)
	}
	return flattenQueryResults(results), nil
}

// GetBySide returns all active nodes for the given side.
func (r *NodeRepo) GetBySide(ctx context.Context, side string) ([]NodeRecord, error) {
	results, err := surrealdb.Query[[]NodeRecord](
		ctx, r.db,
		"SELECT * FROM logistics_node WHERE side = $side AND is_active = true",
		map[string]any{"side": side},
	)
	if err != nil {
		return nil, fmt.Errorf("get nodes by side %q: %w", side, err)
	}
	return flattenQueryResults(results), nil
}

// UpdateStock merges stock-level fields for a node. Called during supply resolution.
func (r *NodeRepo) UpdateStock(ctx context.Context, nodeID string, stock map[string]any) error {
	_, err := surrealdb.Merge[NodeRecord](
		ctx, r.db,
		models.RecordID{Table: "logistics_node", ID: nodeID},
		stock,
	)
	if err != nil {
		return fmt.Errorf("update stock for node %s: %w", nodeID, err)
	}
	return nil
}

// DeleteAll removes all logistics_node records. Called when loading a new scenario.
func (r *NodeRepo) DeleteAll(ctx context.Context) error {
	_, err := surrealdb.Query[any](ctx, r.db, "DELETE logistics_node", nil)
	if err != nil {
		return fmt.Errorf("delete all nodes: %w", err)
	}
	return nil
}
