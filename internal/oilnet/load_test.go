package oilnet

import "testing"

func TestLoadGlobalBaseline(t *testing.T) {
	graph, err := LoadGlobalBaseline()
	if err != nil {
		t.Fatalf("LoadGlobalBaseline() error = %v", err)
	}
	if graph.ID == "" {
		t.Fatal("expected graph id")
	}
	if len(graph.Nodes) < 10 {
		t.Fatalf("expected meaningful global graph, got %d nodes", len(graph.Nodes))
	}
	if len(graph.Edges) < 10 {
		t.Fatalf("expected meaningful global graph, got %d edges", len(graph.Edges))
	}
}

func TestGlobalBaselineIncludesRefineryDerivativeOutputs(t *testing.T) {
	graph, err := LoadGlobalBaseline()
	if err != nil {
		t.Fatalf("LoadGlobalBaseline() error = %v", err)
	}
	foundRefinery := false
	foundDerivatives := false
	for _, node := range graph.Nodes {
		if node.Kind != NodeRefinery {
			continue
		}
		foundRefinery = true
		if len(node.ProductOutputs) > 0 {
			foundDerivatives = true
			break
		}
	}
	if !foundRefinery {
		t.Fatal("expected at least one refinery node")
	}
	if !foundDerivatives {
		t.Fatal("expected refinery derivative outputs for click-through UI")
	}
}
