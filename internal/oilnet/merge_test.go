package oilnet

import "testing"

func TestMergeGraphsOverlaysNodesAndEdgesByID(t *testing.T) {
	base := &Graph{
		ID: "base",
		Nodes: []Node{
			{ID: "n1", Name: "Base Node", Kind: NodeExtractionSite},
		},
		Edges: []Edge{
			{ID: "e1", Name: "Base Edge", FromNodeID: "n1", ToNodeID: "n1", Commodity: CommodityCrude},
		},
	}
	overlay := &Graph{
		ID: "overlay",
		Nodes: []Node{
			{ID: "n1", Name: "Overlay Node", Kind: NodeExportTerminal},
			{ID: "n2", Name: "New Node", Kind: NodeDemandCenter},
		},
		Edges: []Edge{
			{ID: "e1", Name: "Overlay Edge", FromNodeID: "n1", ToNodeID: "n2", Commodity: CommodityCrude},
			{ID: "e2", Name: "New Edge", FromNodeID: "n2", ToNodeID: "n1", Commodity: CommodityDiesel},
		},
	}
	merged := MergeGraphs(base, overlay)
	if len(merged.Nodes) != 2 {
		t.Fatalf("expected 2 merged nodes, got %d", len(merged.Nodes))
	}
	if merged.Nodes[0].Name != "Overlay Node" {
		t.Fatalf("expected overlay node to replace base node, got %q", merged.Nodes[0].Name)
	}
	if len(merged.Edges) != 2 {
		t.Fatalf("expected 2 merged edges, got %d", len(merged.Edges))
	}
	if merged.Edges[0].Name != "Overlay Edge" {
		t.Fatalf("expected overlay edge to replace base edge, got %q", merged.Edges[0].Name)
	}
}
