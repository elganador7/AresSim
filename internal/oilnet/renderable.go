package oilnet

import "strings"

func BuildRenderableGraph(graph *Graph) *Graph {
	if graph == nil {
		return nil
	}
	out := *graph
	nodeByID := make(map[string]Node, len(graph.Nodes))
	out.Nodes = make([]Node, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if !ShouldRenderNode(node) {
			continue
		}
		simplified := simplifyRenderableNode(node)
		out.Nodes = append(out.Nodes, simplified)
		nodeByID[simplified.ID] = simplified
	}
	out.Edges = make([]Edge, 0, len(graph.Edges))
	for _, edge := range graph.Edges {
		if !ShouldRenderEdge(edge) {
			continue
		}
		if _, ok := nodeByID[edge.FromNodeID]; !ok {
			continue
		}
		if _, ok := nodeByID[edge.ToNodeID]; !ok {
			continue
		}
		out.Edges = append(out.Edges, simplifyRenderableEdge(edge))
	}
	return &out
}

func ShouldRenderNode(node Node) bool {
	if isLatentOrInactiveNode(node) {
		return false
	}
	if node.Kind == NodeChokepoint || node.Kind == NodeRefinery || node.Kind == NodeProject || node.Kind == NodeExtractionSite {
		return true
	}
	return node.CurrentFlowBPD >= 150_000 ||
		node.CapacityBPD >= 250_000 ||
		node.Kind == NodeGatheringHub ||
		node.Kind == NodePipelineTerminal ||
		node.Kind == NodeExportTerminal ||
		node.Kind == NodeImportTerminal ||
		node.Kind == NodeDemandCenter
}

func ShouldRenderEdge(edge Edge) bool {
	if isLatentOrInactiveEdge(edge) {
		return false
	}
	return edge.Kind == EdgePipeline ||
		edge.CurrentFlowBPD >= 300_000 ||
		(edge.Kind == EdgeShipping && (edge.CrossesChokepoint != "" || len(edge.CrossesChokepoints) > 0))
}

func isLatentOrInactiveNode(node Node) bool {
	for _, tag := range node.Tags {
		switch strings.TrimSpace(strings.ToLower(tag)) {
		case "status:mothballed", "status:idle", "status:shelved", "status:cancelled", "status:canceled", "status:abandoned", "status:retired", "status:decommissioning", "status:closed", "status:underground gas storage":
			return true
		}
	}
	return node.State == StateOffline
}

func isLatentOrInactiveEdge(edge Edge) bool {
	switch strings.TrimSpace(strings.ToLower(edge.StatusDetail)) {
	case "mothballed", "idle", "shelved", "cancelled", "canceled", "retired", "proposed", "":
		return true
	}
	return edge.State == StateOffline
}

func simplifyRenderableNode(node Node) Node {
	node.Tags = nil
	node.Sources = nil
	node.DemandProfile = nil
	node.Inventory = nil
	if node.Kind != NodeRefinery {
		node.ProductOutputs = nil
	}
	return node
}

func simplifyRenderableEdge(edge Edge) Edge {
	edge.Sources = nil
	edge.Route = simplifyRoute(edge.Route, 24)
	return edge
}

func simplifyRoute(route []RoutePoint, maxPoints int) []RoutePoint {
	if len(route) <= maxPoints || maxPoints < 2 {
		return route
	}
	out := make([]RoutePoint, 0, maxPoints)
	lastIdx := len(route) - 1
	for i := 0; i < maxPoints; i++ {
		idx := int(float64(i) * float64(lastIdx) / float64(maxPoints-1))
		out = append(out, route[idx])
	}
	return out
}
