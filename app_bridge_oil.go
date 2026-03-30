package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aressim/internal/oilnet"
	"github.com/aressim/internal/oilnet/ingest"
)

// GetGlobalOilNetwork returns the starter global oil trade graph for the map layer.
func (a *App) GetGlobalOilNetwork() (map[string]any, error) {
	graph, err := a.cachedGlobalOilGraph()
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		return nil, fmt.Errorf("marshal oil graph: %w", err)
	}
	out := make(map[string]any)
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal oil graph map: %w", err)
	}
	return out, nil
}

// GetRenderableOilNetwork returns a filtered, map-safe subset of the global oil graph.
func (a *App) GetRenderableOilNetwork() (map[string]any, error) {
	graph, err := a.cachedRenderableOilGraph()
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		return nil, fmt.Errorf("marshal renderable oil graph: %w", err)
	}
	out := make(map[string]any)
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal renderable oil graph map: %w", err)
	}
	return out, nil
}

// SimulateOilShock recomputes the global oil graph after outages or degradations.
func (a *App) SimulateOilShock(request map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal shock request: %w", err)
	}
	var req oilnet.ShockRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("decode shock request: %w", err)
	}
	graph, err := a.cachedGlobalOilGraph()
	if err != nil {
		return nil, err
	}
	result, err := oilnet.SimulateShock(graph, req)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal shock result: %w", err)
	}
	out := make(map[string]any)
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("unmarshal shock result map: %w", err)
	}
	return out, nil
}

// SimulateOilShockHorizon recomputes multi-day outage effects with storage drawdown.
func (a *App) SimulateOilShockHorizon(request map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal horizon request: %w", err)
	}
	var req oilnet.HorizonRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("decode horizon request: %w", err)
	}
	graph, err := a.cachedGlobalOilGraph()
	if err != nil {
		return nil, err
	}
	result, err := oilnet.SimulateShockHorizon(graph, req)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal horizon result: %w", err)
	}
	out := make(map[string]any)
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("unmarshal horizon result map: %w", err)
	}
	return out, nil
}

func loadGlobalOilGraph() (*oilnet.Graph, error) {
	extraction, err := ingest.LoadExtractionWorkbookGraph("data/Global-Oil-and-Gas-Extraction-Tracker-March-2026.xlsx")
	if err != nil {
		return nil, fmt.Errorf("load extraction workbook: %w", err)
	}
	pipelines, err := ingest.LoadPipelinesGeoJSON("data/GEM-GOIT-Oil-NGL-Pipelines-2025-03.geojson")
	if err != nil {
		return nil, fmt.Errorf("load pipeline geojson: %w", err)
	}
	merged := oilnet.MergeGraphs(extraction, pipelines)
	merged.ID = "global-oil-network-realdata"
	merged.Name = "Global Oil Network Real Data"
	merged.Description = "Global oil network built only from the provided extraction workbook and pipeline GeoJSON."
	merged.View = "global"
	if err := oilnet.ValidateGraph(merged); err != nil {
		return nil, err
	}
	return merged, nil
}

func loadRenderableOilGraph() (*oilnet.Graph, error) {
	const cachePath = "data/oil-renderable-cache-v2.json"
	if cached, err := loadOilGraphJSON(cachePath); err == nil {
		return cached, nil
	}
	graph, err := loadGlobalOilGraph()
	if err != nil {
		return nil, err
	}
	renderable := renderableOilGraph(graph)
	if err := oilnet.ValidateGraph(renderable); err != nil {
		return nil, err
	}
	return renderable, nil
}

func loadOilGraphJSON(path string) (*oilnet.Graph, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var graph oilnet.Graph
	if err := json.Unmarshal(raw, &graph); err != nil {
		return nil, fmt.Errorf("decode oil graph json %s: %w", path, err)
	}
	return &graph, nil
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

func (a *App) cachedGlobalOilGraph() (*oilnet.Graph, error) {
	a.oilGraphMu.RLock()
	if a.oilGraph != nil {
		graph := a.oilGraph
		a.oilGraphMu.RUnlock()
		return graph, nil
	}
	a.oilGraphMu.RUnlock()

	a.oilGraphMu.Lock()
	defer a.oilGraphMu.Unlock()
	if a.oilGraph != nil {
		return a.oilGraph, nil
	}
	graph, err := loadGlobalOilGraph()
	if err != nil {
		return nil, err
	}
	a.oilGraph = graph
	return a.oilGraph, nil
}

func (a *App) cachedRenderableOilGraph() (*oilnet.Graph, error) {
	a.oilRenderableGraphMu.RLock()
	if a.oilRenderableGraph != nil {
		graph := a.oilRenderableGraph
		a.oilRenderableGraphMu.RUnlock()
		return graph, nil
	}
	a.oilRenderableGraphMu.RUnlock()

	a.oilRenderableGraphMu.Lock()
	defer a.oilRenderableGraphMu.Unlock()
	if a.oilRenderableGraph != nil {
		return a.oilRenderableGraph, nil
	}
	graph, err := loadRenderableOilGraph()
	if err != nil {
		return nil, err
	}
	a.oilRenderableGraph = graph
	return a.oilRenderableGraph, nil
}
