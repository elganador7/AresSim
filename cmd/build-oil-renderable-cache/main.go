package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aressim/internal/oilnet"
	"github.com/aressim/internal/oilnet/ingest"
)

func main() {
	extractionPath := flag.String("extraction", "data/Global-Oil-and-Gas-Extraction-Tracker-March-2026.xlsx", "path to extraction workbook")
	pipelinesPath := flag.String("pipelines", "data/GEM-GOIT-Oil-NGL-Pipelines-2025-03.geojson", "path to pipeline geojson")
	outputPath := flag.String("output", "data/oil-renderable-cache-v2.json", "output cache path")
	flag.Parse()

	extraction, err := ingest.LoadExtractionWorkbookGraph(*extractionPath)
	if err != nil {
		panic(fmt.Errorf("load extraction workbook: %w", err))
	}
	pipelines, err := ingest.LoadPipelinesGeoJSON(*pipelinesPath)
	if err != nil {
		panic(fmt.Errorf("load pipeline geojson: %w", err))
	}
	merged := oilnet.MergeGraphs(extraction, pipelines)
	renderable := renderableOilGraph(merged)
	if err := oilnet.ValidateGraph(renderable); err != nil {
		panic(fmt.Errorf("validate renderable graph: %w", err))
	}
	raw, err := json.MarshalIndent(renderable, "", "  ")
	if err != nil {
		panic(fmt.Errorf("marshal renderable graph: %w", err))
	}
	if err := os.WriteFile(*outputPath, raw, 0o644); err != nil {
		panic(fmt.Errorf("write renderable cache: %w", err))
	}
	fmt.Printf("wrote %s with %d nodes and %d edges\n", *outputPath, len(renderable.Nodes), len(renderable.Edges))
}

func renderableOilGraph(graph *oilnet.Graph) *oilnet.Graph {
	if graph == nil {
		return nil
	}
	out := *graph
	nodeByID := make(map[string]oilnet.Node, len(graph.Nodes))
	out.Nodes = make([]oilnet.Node, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if !shouldRenderOilNode(node) {
			continue
		}
		simplified := simplifyRenderableOilNode(node)
		out.Nodes = append(out.Nodes, simplified)
		nodeByID[simplified.ID] = simplified
	}
	out.Edges = make([]oilnet.Edge, 0, len(graph.Edges))
	for _, edge := range graph.Edges {
		if !shouldRenderOilEdge(edge) {
			continue
		}
		if _, ok := nodeByID[edge.FromNodeID]; !ok {
			continue
		}
		if _, ok := nodeByID[edge.ToNodeID]; !ok {
			continue
		}
		out.Edges = append(out.Edges, simplifyRenderableOilEdge(edge))
	}
	return &out
}

func shouldRenderOilNode(node oilnet.Node) bool {
	if isLatentOrInactiveOilNode(node) {
		return false
	}
	if node.Kind == oilnet.NodeChokepoint || node.Kind == oilnet.NodeRefinery || node.Kind == oilnet.NodeProject || node.Kind == oilnet.NodeExtractionSite {
		return true
	}
	return node.CurrentFlowBPD >= 150_000 ||
		node.CapacityBPD >= 250_000 ||
		node.Kind == oilnet.NodeGatheringHub ||
		node.Kind == oilnet.NodePipelineTerminal ||
		node.Kind == oilnet.NodeExportTerminal ||
		node.Kind == oilnet.NodeImportTerminal ||
		node.Kind == oilnet.NodeDemandCenter
}

func shouldRenderOilEdge(edge oilnet.Edge) bool {
	if isLatentOrInactiveOilEdge(edge) {
		return false
	}
	return edge.Kind == oilnet.EdgePipeline ||
		edge.CurrentFlowBPD >= 300_000 ||
		(edge.Kind == oilnet.EdgeShipping && (edge.CrossesChokepoint != "" || len(edge.CrossesChokepoints) > 0))
}

func isLatentOrInactiveOilNode(node oilnet.Node) bool {
	for _, tag := range node.Tags {
		switch strings.TrimSpace(strings.ToLower(tag)) {
		case "status:mothballed", "status:idle", "status:shelved", "status:cancelled", "status:canceled", "status:abandoned", "status:retired", "status:decommissioning", "status:closed", "status:underground gas storage":
			return true
		}
	}
	return node.State == oilnet.StateOffline
}

func isLatentOrInactiveOilEdge(edge oilnet.Edge) bool {
	switch strings.TrimSpace(strings.ToLower(edge.StatusDetail)) {
	case "mothballed", "idle", "shelved", "cancelled", "canceled", "retired", "proposed", "":
		return true
	}
	return edge.State == oilnet.StateOffline
}

func simplifyRenderableOilNode(node oilnet.Node) oilnet.Node {
	node.Tags = nil
	node.Sources = nil
	node.DemandProfile = nil
	node.Inventory = nil
	if node.Kind != oilnet.NodeRefinery {
		node.ProductOutputs = nil
	}
	return node
}

func simplifyRenderableOilEdge(edge oilnet.Edge) oilnet.Edge {
	edge.Sources = nil
	edge.Route = simplifyRoute(edge.Route, 24)
	return edge
}

func simplifyRoute(route []oilnet.RoutePoint, maxPoints int) []oilnet.RoutePoint {
	if len(route) <= maxPoints || maxPoints < 2 {
		return route
	}
	out := make([]oilnet.RoutePoint, 0, maxPoints)
	lastIdx := len(route) - 1
	for i := 0; i < maxPoints; i++ {
		idx := int(float64(i) * float64(lastIdx) / float64(maxPoints-1))
		out = append(out, route[idx])
	}
	return out
}
