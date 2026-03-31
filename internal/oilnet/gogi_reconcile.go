package oilnet

import "math"

const (
	gogiTerminalMergeKM       = 50.0
	gogiStrongTerminalMergeKM = 15.0
)

// ReconcileGOGIOverlay merges high-confidence GOGI terminal duplicates into an
// existing graph while keeping non-terminal infrastructure as standalone nodes.
func ReconcileGOGIOverlay(base *Graph, overlay *Graph) *Graph {
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
		if node.Kind == NodeMarineTerminal {
			if idx, ok := findMatchingMarineTerminal(out.Nodes, node); ok {
				out.Nodes[idx] = mergeMarineTerminalNode(out.Nodes[idx], node)
				continue
			}
		}
		if idx, ok := nodeIndex[node.ID]; ok {
			out.Nodes[idx] = node
			continue
		}
		out.Nodes = append(out.Nodes, node)
		nodeIndex[node.ID] = len(out.Nodes) - 1
	}
	out.Sources = append(out.Sources, overlay.Sources...)
	return out
}

func findMatchingMarineTerminal(nodes []Node, overlay Node) (int, bool) {
	bestIdx := -1
	bestDistance := math.MaxFloat64
	for i, candidate := range nodes {
		if candidate.Kind != NodeMarineTerminal {
			continue
		}
		distance := haversineKM(candidate.Lat, candidate.Lon, overlay.Lat, overlay.Lon)
		if distance > gogiTerminalMergeKM {
			continue
		}
		if sameCountry(candidate.CountryCode, overlay.CountryCode) || distance <= gogiStrongTerminalMergeKM {
			if distance < bestDistance {
				bestDistance = distance
				bestIdx = i
			}
		}
	}
	return bestIdx, bestIdx >= 0
}

func mergeMarineTerminalNode(base Node, overlay Node) Node {
	if base.Operator == "" {
		base.Operator = overlay.Operator
	}
	if base.TerminalType == "" {
		base.TerminalType = overlay.TerminalType
	}
	if base.PrimaryCommodity == "" {
		base.PrimaryCommodity = overlay.PrimaryCommodity
	}
	if base.CapacityBPD == 0 {
		base.CapacityBPD = overlay.CapacityBPD
	}
	if base.CurrentFlowBPD == 0 {
		base.CurrentFlowBPD = overlay.CurrentFlowBPD
	}
	if base.StorageCapacityBbl == 0 {
		base.StorageCapacityBbl = overlay.StorageCapacityBbl
	}
	if base.BerthCount == 0 {
		base.BerthCount = overlay.BerthCount
	}
	if base.DraftClass == "" {
		base.DraftClass = overlay.DraftClass
	}
	base.ProductsHandled = unionCommodities(base.ProductsHandled, overlay.ProductsHandled)
	base.Tags = unionStrings(base.Tags, overlay.Tags)
	base.Sources = append(base.Sources, overlay.Sources...)
	return base
}

func unionCommodities(a []Commodity, b []Commodity) []Commodity {
	out := append([]Commodity(nil), a...)
	for _, item := range b {
		seen := false
		for _, existing := range out {
			if existing == item {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, item)
		}
	}
	return out
}

func unionStrings(a []string, b []string) []string {
	out := append([]string(nil), a...)
	for _, item := range b {
		seen := false
		for _, existing := range out {
			if existing == item {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, item)
		}
	}
	return out
}

func sameCountry(a string, b string) bool {
	return a != "" && b != "" && a == b
}
