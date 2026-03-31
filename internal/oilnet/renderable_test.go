package oilnet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRenderableGraphFiltersAndSimplifies(t *testing.T) {
	graph := &Graph{
		ID: "test",
		Nodes: []Node{
			{
				ID:             "project-1",
				Name:           "Project",
				Kind:           NodeProject,
				State:          StateOperational,
				ProductionBPD:  500_000,
				Tags:           []string{"status:operating"},
				Sources:        []SourceRef{{Name: "src", Confidence: 0.9}},
				DemandProfile:  []CommodityQuantity{{Commodity: CommodityCrude, BPD: 1}},
				Inventory:      []CommodityQuantity{{Commodity: CommodityCrude, Barrels: 1}},
				ProductOutputs: []ProductOutput{{Commodity: CommodityDiesel, BPD: 1}},
			},
			{
				ID:          "project-2",
				Name:        "Offline",
				Kind:        NodeProject,
				State:       StateOffline,
				Tags:        []string{"status:retired"},
				Sources:     []SourceRef{{Name: "src", Confidence: 0.9}},
				CapacityBPD: 10,
			},
		},
		Edges: []Edge{
			{
				ID:             "edge-1",
				Name:           "Pipeline",
				Kind:           EdgePipeline,
				FromNodeID:     "project-1",
				ToNodeID:       "project-1",
				State:          StateOperational,
				StatusDetail:   "operating",
				CapacityBPD:    500_000,
				CurrentFlowBPD: 400_000,
				Route: []RoutePoint{
					{Lat: 1, Lon: 1},
					{Lat: 2, Lon: 2},
					{Lat: 3, Lon: 3},
					{Lat: 4, Lon: 4},
				},
				Sources: []SourceRef{{Name: "src", Confidence: 0.9}},
			},
			{
				ID:             "edge-2",
				Name:           "Retired",
				Kind:           EdgePipeline,
				FromNodeID:     "project-1",
				ToNodeID:       "project-1",
				State:          StateOperational,
				StatusDetail:   "retired",
				CapacityBPD:    500_000,
				CurrentFlowBPD: 400_000,
			},
		},
	}

	renderable := BuildRenderableGraph(graph)
	if len(renderable.Nodes) != 1 {
		t.Fatalf("expected 1 renderable node, got %d", len(renderable.Nodes))
	}
	if len(renderable.Edges) != 1 {
		t.Fatalf("expected 1 renderable edge, got %d", len(renderable.Edges))
	}
	if got := renderable.Nodes[0].Sources; got != nil {
		t.Fatalf("expected node sources to be stripped, got %+v", got)
	}
	if got := renderable.Nodes[0].DemandProfile; got != nil {
		t.Fatalf("expected node demand profile to be stripped, got %+v", got)
	}
	if got := renderable.Nodes[0].Inventory; got != nil {
		t.Fatalf("expected node inventory to be stripped, got %+v", got)
	}
	if got := renderable.Nodes[0].ProductOutputs; got != nil {
		t.Fatalf("expected non-refinery product outputs to be stripped, got %+v", got)
	}
	if got := renderable.Edges[0].Sources; got != nil {
		t.Fatalf("expected edge sources to be stripped, got %+v", got)
	}
}

func TestLoadGraphJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")
	raw, err := json.Marshal(&Graph{
		ID:    "json-graph",
		Nodes: []Node{{ID: "n1", Name: "Node", Kind: NodeProject, State: StateOperational}},
		Edges: []Edge{},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	graph, err := LoadGraphJSON(path)
	if err != nil {
		t.Fatalf("LoadGraphJSON() error = %v", err)
	}
	if graph.ID != "json-graph" {
		t.Fatalf("expected graph id json-graph, got %q", graph.ID)
	}
}

func TestLoadGraphJSONWrappedCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "graph-cache.json")
	graph := &Graph{
		ID:      "wrapped-graph",
		Version: "1.2.3",
		Nodes:   []Node{{ID: "n1", Name: "Node", Kind: NodeProject, State: StateOperational}},
		Edges:   []Edge{},
	}
	if err := WriteGraphCacheJSON(path, graph); err != nil {
		t.Fatalf("WriteGraphCacheJSON() error = %v", err)
	}
	loaded, err := LoadGraphJSON(path)
	if err != nil {
		t.Fatalf("LoadGraphJSON() error = %v", err)
	}
	if loaded.ID != "wrapped-graph" {
		t.Fatalf("expected wrapped graph id, got %q", loaded.ID)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var wrapped GraphCacheFile
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if wrapped.Metadata.SchemaVersion != RenderableCacheSchemaVersion {
		t.Fatalf("expected schema version %q, got %q", RenderableCacheSchemaVersion, wrapped.Metadata.SchemaVersion)
	}
	if wrapped.Diagnostics.NodeCount != 1 {
		t.Fatalf("expected one node in diagnostics, got %d", wrapped.Diagnostics.NodeCount)
	}
}
