package oilnet

// MergeGraphs overlays one graph onto another by node and edge ID.
func MergeGraphs(base *Graph, overlay *Graph) *Graph {
	if base == nil {
		return cloneGraph(overlay)
	}
	if overlay == nil {
		return cloneGraph(base)
	}
	out := cloneGraph(base)
	nodeIndex := make(map[string]int, len(out.Nodes))
	for i, node := range out.Nodes {
		nodeIndex[node.ID] = i
	}
	for _, node := range overlay.Nodes {
		if idx, ok := nodeIndex[node.ID]; ok {
			out.Nodes[idx] = node
			continue
		}
		out.Nodes = append(out.Nodes, node)
		nodeIndex[node.ID] = len(out.Nodes) - 1
	}
	edgeIndex := make(map[string]int, len(out.Edges))
	for i, edge := range out.Edges {
		edgeIndex[edge.ID] = i
	}
	for _, edge := range overlay.Edges {
		if idx, ok := edgeIndex[edge.ID]; ok {
			out.Edges[idx] = edge
			continue
		}
		out.Edges = append(out.Edges, edge)
		edgeIndex[edge.ID] = len(out.Edges) - 1
	}
	out.Sources = append(out.Sources, overlay.Sources...)
	return out
}
