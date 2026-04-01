package oilnet

import "testing"

func TestPopulateNodeH3(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{ID: "n1", Name: "Doha", Kind: NodeProject, Lat: 25.2854, Lon: 51.5310, State: StateOperational},
			{ID: "n2", Name: "Hormuz", Kind: NodeChokepoint, Lat: 26.566, Lon: 56.25, State: StateOperational},
		},
	}
	if err := PopulateNodeH3(graph); err != nil {
		t.Fatalf("PopulateNodeH3 returned error: %v", err)
	}
	for _, node := range graph.Nodes {
		if node.H3Cell == "" {
			t.Fatalf("expected h3 cell for %s", node.ID)
		}
		if node.H3ParentCell == "" {
			t.Fatalf("expected h3 parent cell for %s", node.ID)
		}
		if node.H3Cell == node.H3ParentCell {
			t.Fatalf("expected parent cell to differ for %s", node.ID)
		}
	}
}
