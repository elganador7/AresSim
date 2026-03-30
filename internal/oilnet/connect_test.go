package oilnet

import "testing"

func TestAttachNearbyPipelineConnectorsAddsExtractionConnector(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{
				ID:               "extract-1",
				Name:             "Extraction",
				Kind:             NodeExtractionSite,
				CountryCode:      "USA",
				Lat:              29.0,
				Lon:              -95.0,
				PrimaryCommodity: CommodityCrude,
				CapacityBPD:      100000,
			},
			{
				ID:          "pipe-node-p1-start",
				Name:        "Pipeline Start",
				Kind:        NodeGatheringHub,
				CountryCode: "USA",
				Lat:         29.1,
				Lon:         -95.0,
			},
			{
				ID:          "pipe-node-p1-end",
				Name:        "Pipeline End",
				Kind:        NodeGatheringHub,
				CountryCode: "USA",
				Lat:         29.2,
				Lon:         -95.0,
			},
		},
		Edges: []Edge{
			{
				ID:          "pipeline-1",
				Name:        "Pipeline 1",
				Kind:        EdgePipeline,
				FromNodeID:  "pipe-node-p1-start",
				ToNodeID:    "pipe-node-p1-end",
				Commodity:   CommodityCrude,
				CapacityBPD: 100000,
				Route: []RoutePoint{
					{Lat: 29.1, Lon: -95.0},
					{Lat: 29.2, Lon: -95.0},
				},
			},
		},
	}
	AttachNearbyPipelineConnectors(graph, 50)
	if len(graph.Edges) != 2 {
		t.Fatalf("expected pipeline plus 1 connector edge, got %d", len(graph.Edges))
	}
	if graph.Edges[1].Kind != EdgeInternalBus {
		t.Fatalf("expected internal connector edge, got %v", graph.Edges[1].Kind)
	}
	if graph.Edges[1].ToNodeID != "pipe-node-p1-start" {
		t.Fatalf("expected connector to attach to nearest pipeline terminal, got %q", graph.Edges[1].ToNodeID)
	}
}

func TestAttachNearbyPipelineConnectorsRespectsCountryFilter(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{
				ID:               "extract-1",
				Name:             "Extraction",
				Kind:             NodeExtractionSite,
				CountryCode:      "USA",
				Lat:              29.0,
				Lon:              -95.0,
				PrimaryCommodity: CommodityCrude,
			},
			{
				ID:          "pipe-node-p1-start",
				Name:        "Pipeline Start",
				Kind:        NodeGatheringHub,
				CountryCode: "CAN",
				Lat:         29.0,
				Lon:         -95.0,
			},
			{
				ID:          "pipe-node-p1-end",
				Name:        "Pipeline End",
				Kind:        NodeGatheringHub,
				CountryCode: "CAN",
				Lat:         29.1,
				Lon:         -95.0,
			},
		},
		Edges: []Edge{
			{
				ID:          "pipeline-1",
				Name:        "Foreign Pipeline",
				Kind:        EdgePipeline,
				FromNodeID:  "pipe-node-p1-start",
				ToNodeID:    "pipe-node-p1-end",
				Commodity:   CommodityCrude,
				CapacityBPD: 100000,
				Route: []RoutePoint{
					{Lat: 29.0, Lon: -95.0},
					{Lat: 29.1, Lon: -95.0},
				},
			},
		},
	}
	AttachNearbyPipelineConnectors(graph, 50)
	if len(graph.Edges) != 1 {
		t.Fatalf("expected no same-country connector edge, got %d edges", len(graph.Edges))
	}
}
