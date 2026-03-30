package oilnet

import "fmt"

// ValidateGraph ensures the graph is self-consistent enough for rendering and flow work.
func ValidateGraph(graph *Graph) error {
	if graph == nil {
		return fmt.Errorf("oil graph is nil")
	}
	nodeIDs := make(map[string]struct{}, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if node.ID == "" {
			return fmt.Errorf("oil graph contains node with empty id")
		}
		if _, exists := nodeIDs[node.ID]; exists {
			return fmt.Errorf("oil graph contains duplicate node id %q", node.ID)
		}
		nodeIDs[node.ID] = struct{}{}
	}
	for _, edge := range graph.Edges {
		if edge.ID == "" {
			return fmt.Errorf("oil graph contains edge with empty id")
		}
		if _, ok := nodeIDs[edge.FromNodeID]; !ok {
			return fmt.Errorf("edge %q references unknown from node %q", edge.ID, edge.FromNodeID)
		}
		if _, ok := nodeIDs[edge.ToNodeID]; !ok {
			return fmt.Errorf("edge %q references unknown to node %q", edge.ID, edge.ToNodeID)
		}
	}
	return nil
}
