package ingest

import (
	"testing"

	"github.com/aressim/internal/oilnet"
)

func TestBuildGraphCreatesNodesAndEdges(t *testing.T) {
	graph, err := BuildGraph(BuildInput{
		GraphID: "test-oilnet",
		Assets: []GEMAssetRow{
			{ID: "sa-field", Name: "Saudi Field", AssetType: "field", CountryCode: "SAU", CapacityBPD: 1000000, Commodity: oilnet.CommodityCrude},
			{ID: "sa-refinery", Name: "Saudi Refinery", AssetType: "refinery", CountryCode: "SAU", CapacityBPD: 300000, Commodity: oilnet.CommodityCrude},
		},
		Balances: []JODIBalanceRow{
			{CountryCode: "SAU", Commodity: oilnet.CommodityCrude, RefineryIntakeBPD: 300000},
			{CountryCode: "JPN", Commodity: oilnet.CommodityJet, DemandBPD: 500000},
		},
		TradeFlows: []ComtradeFlowRow{
			{ReporterCode: "SAU", PartnerCode: "JPN", Commodity: oilnet.CommodityCrude, FlowBPD: 400000},
		},
	})
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}
	if graph.ID != "test-oilnet" {
		t.Fatalf("expected graph id, got %q", graph.ID)
	}
	if len(graph.Nodes) < 3 {
		t.Fatalf("expected generated nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) < 2 {
		t.Fatalf("expected generated edges, got %d", len(graph.Edges))
	}

	var jpnDemand oilnet.Node
	for _, node := range graph.Nodes {
		if node.ID == "jpn-demand" {
			jpnDemand = node
			break
		}
	}
	if jpnDemand.ID == "" || jpnDemand.Lat == 0 || jpnDemand.Lon == 0 {
		t.Fatalf("expected anchored demand node for JPN, got %+v", jpnDemand)
	}

	var trade oilnet.Edge
	for _, edge := range graph.Edges {
		if edge.ID == "trade-sa-field-jpn-demand-crude" {
			trade = edge
			break
		}
	}
	if trade.ID == "" {
		t.Fatal("expected generated trade edge")
	}
	if len(trade.Route) < 3 {
		t.Fatalf("expected routed shipping edge, got %+v", trade.Route)
	}
	if len(trade.CrossesChokepoints) == 0 || trade.CrossesChokepoints[0] != "om-hormuz" {
		t.Fatalf("expected Hormuz chokepoint on Saudi export route, got %+v", trade.CrossesChokepoints)
	}
}
