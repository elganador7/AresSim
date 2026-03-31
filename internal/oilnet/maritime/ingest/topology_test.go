package ingest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aressim/internal/oilnet"
)

func TestLoadEmbeddedTopology(t *testing.T) {
	topology, err := LoadEmbeddedTopology()
	if err != nil {
		t.Fatalf("LoadEmbeddedTopology() error = %v", err)
	}
	if topology.ID == "" {
		t.Fatal("expected embedded maritime topology id")
	}
	if len(topology.Terminals) == 0 || len(topology.Corridors) == 0 {
		t.Fatalf("expected terminals and corridors, got terminals=%d corridors=%d", len(topology.Terminals), len(topology.Corridors))
	}
}

func TestTopologyToGraph(t *testing.T) {
	topology, err := LoadEmbeddedTopology()
	if err != nil {
		t.Fatalf("LoadEmbeddedTopology() error = %v", err)
	}
	graph, err := TopologyToGraph(topology)
	if err != nil {
		t.Fatalf("TopologyToGraph() error = %v", err)
	}
	if graph.ID != topology.ID {
		t.Fatalf("expected graph id %q, got %q", topology.ID, graph.ID)
	}
	foundTerminal := false
	foundCorridor := false
	foundCanal := false
	for _, node := range graph.Nodes {
		if node.Kind == oilnet.NodeMarineTerminal {
			foundTerminal = true
		}
		if node.Kind == oilnet.NodeCanal {
			foundCanal = true
		}
	}
	for _, edge := range graph.Edges {
		if edge.Kind == oilnet.EdgeSeaborneCorridor {
			foundCorridor = true
			if edge.VesselClass == "" {
				t.Fatal("expected seaborne corridor vessel class")
			}
		}
	}
	if !foundTerminal {
		t.Fatal("expected marine terminal nodes in maritime graph")
	}
	if !foundCanal {
		t.Fatal("expected canal nodes in maritime graph")
	}
	if !foundCorridor {
		t.Fatal("expected seaborne corridor edges in maritime graph")
	}
}

func TestLoadTopologyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "maritime.json")
	raw := []byte(`{"id":"test-maritime","name":"Test Maritime","description":"test","version":"1","terminals":[{"id":"terminal-a","name":"Terminal A","countryCode":"SAU","lat":25,"lon":50,"state":"operational","terminalType":"crude_export","productsHandled":["crude"]}],"canals":[],"corridors":[]}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	carrier, err := LoadTopologyFile(path)
	if err != nil {
		t.Fatalf("LoadTopologyFile() error = %v", err)
	}
	if carrier.Source != path {
		t.Fatalf("expected source %q, got %q", path, carrier.Source)
	}
	if carrier.Topology.ID != "test-maritime" {
		t.Fatalf("expected topology id test-maritime, got %q", carrier.Topology.ID)
	}
	if len(carrier.Topology.Terminals) != 1 {
		t.Fatalf("expected one terminal, got %d", len(carrier.Topology.Terminals))
	}
}

func TestLoadTopologyFileOrEmbeddedFallsBack(t *testing.T) {
	carrier, err := LoadTopologyFileOrEmbedded(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadTopologyFileOrEmbedded() error = %v", err)
	}
	if carrier.Topology == nil || carrier.Topology.ID == "" {
		t.Fatal("expected embedded fallback topology")
	}
	if carrier.Source != "" {
		t.Fatalf("expected empty source for embedded fallback, got %q", carrier.Source)
	}
}
