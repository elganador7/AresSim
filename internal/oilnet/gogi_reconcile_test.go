package oilnet

import "testing"

func TestReconcileGOGIOverlayMergesNearbyMarineTerminal(t *testing.T) {
	base := &Graph{
		Nodes: []Node{
			{
				ID:                 "terminal-sa-ras-tanura",
				Name:               "Ras Tanura Marine Terminal",
				Kind:               NodeMarineTerminal,
				CountryCode:        "SAU",
				Lat:                26.640,
				Lon:                50.155,
				State:              StateOperational,
				ProductsHandled:    []Commodity{CommodityCrude},
				StorageCapacityBbl: 35_000_000,
			},
		},
	}
	overlay := &Graph{
		Nodes: []Node{
			{
				ID:              "gogi:marine_terminal:ras-tanura-port",
				Name:            "Ras Tanura Port",
				Kind:            NodeMarineTerminal,
				CountryCode:     "SAU",
				Lat:             26.638,
				Lon:             50.160,
				State:           StateOperational,
				Operator:        "Saudi Aramco",
				ProductsHandled: []Commodity{CommodityCrude, CommodityLPG},
				Tags:            []string{"source:gogi"},
			},
		},
	}

	got := ReconcileGOGIOverlay(base, overlay)
	if len(got.Nodes) != 1 {
		t.Fatalf("expected merged terminal count 1, got %d", len(got.Nodes))
	}
	if got.Nodes[0].Operator != "Saudi Aramco" {
		t.Fatalf("expected operator enrichment, got %q", got.Nodes[0].Operator)
	}
	if len(got.Nodes[0].ProductsHandled) != 2 {
		t.Fatalf("expected merged commodities, got %+v", got.Nodes[0].ProductsHandled)
	}
	if got.Nodes[0].Tags[0] != "source:gogi" && got.Nodes[0].Tags[len(got.Nodes[0].Tags)-1] != "source:gogi" {
		t.Fatalf("expected gogi tag after merge, got %+v", got.Nodes[0].Tags)
	}
}

func TestReconcileGOGIOverlayKeepsStandaloneInfrastructure(t *testing.T) {
	base := &Graph{
		Nodes: []Node{{ID: "terminal-base", Kind: NodeMarineTerminal, Lat: 0, Lon: 0, State: StateOperational}},
	}
	overlay := &Graph{
		Nodes: []Node{
			{ID: "gogi:refinery:1", Kind: NodeRefinery, Lat: 1, Lon: 1, State: StateOperational},
			{ID: "gogi:terminal:far", Kind: NodeMarineTerminal, Lat: 50, Lon: 50, State: StateOperational},
		},
	}
	got := ReconcileGOGIOverlay(base, overlay)
	if len(got.Nodes) != 3 {
		t.Fatalf("expected 3 nodes after reconciliation, got %d", len(got.Nodes))
	}
}
