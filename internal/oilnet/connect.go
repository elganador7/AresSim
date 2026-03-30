package oilnet

import (
	"fmt"
	"math"
	"strings"
)

// AttachNearbyPipelineConnectors attaches infrastructure nodes to the nearest
// same-country pipeline terminal node. This intentionally uses only terminal
// nodes instead of route-geometry snapping so the real-data graph stays cheap
// enough to build interactively.
func AttachNearbyPipelineConnectors(graph *Graph, maxDistanceKM float64) {
	if graph == nil || len(graph.Nodes) == 0 || len(graph.Edges) == 0 {
		return
	}

	nodeByID := make(map[string]Node, len(graph.Nodes))
	pipelineTerminalIDsByCountry := make(map[string][]string)
	for _, node := range graph.Nodes {
		nodeByID[node.ID] = node
	}
	for _, edge := range graph.Edges {
		if edge.Kind != EdgePipeline {
			continue
		}
		if from, ok := nodeByID[edge.FromNodeID]; ok && isPipelineTerminalNode(from) {
			pipelineTerminalIDsByCountry[from.CountryCode] = appendIfMissing(pipelineTerminalIDsByCountry[from.CountryCode], from.ID)
		}
		if to, ok := nodeByID[edge.ToNodeID]; ok && isPipelineTerminalNode(to) {
			pipelineTerminalIDsByCountry[to.CountryCode] = appendIfMissing(pipelineTerminalIDsByCountry[to.CountryCode], to.ID)
		}
	}

	existingEdgeIDs := make(map[string]struct{}, len(graph.Edges))
	for _, edge := range graph.Edges {
		existingEdgeIDs[edge.ID] = struct{}{}
	}

	for _, node := range graph.Nodes {
		if !isConnectableInfrastructure(node) || node.CountryCode == "" {
			continue
		}
		candidates := pipelineTerminalIDsByCountry[node.CountryCode]
		if len(candidates) == 0 {
			continue
		}
		nearestID := ""
		nearestDistance := math.MaxFloat64
		for _, candidateID := range candidates {
			candidate, ok := nodeByID[candidateID]
			if !ok || candidate.ID == node.ID {
				continue
			}
			distance := haversineKM(node.Lat, node.Lon, candidate.Lat, candidate.Lon)
			if distance < nearestDistance {
				nearestDistance = distance
				nearestID = candidate.ID
			}
		}
		if nearestID == "" || nearestDistance > maxDistanceKM {
			continue
		}
		addDirectionalConnectors(graph, existingEdgeIDs, node, nodeByID[nearestID], nearestDistance)
	}
}

func isPipelineTerminalNode(node Node) bool {
	return strings.HasPrefix(node.ID, "pipe-node-")
}

func isConnectableInfrastructure(node Node) bool {
	if isPipelineTerminalNode(node) {
		return false
	}
	switch node.Kind {
	case NodeExtractionSite, NodeGatheringHub, NodeExportTerminal, NodeImportTerminal, NodeRefinery, NodeStorageHub:
		return true
	default:
		return false
	}
}

func addDirectionalConnectors(graph *Graph, existingEdgeIDs map[string]struct{}, node Node, pipelineNode Node, distanceKM float64) {
	commodity := node.PrimaryCommodity
	if commodity == "" {
		commodity = CommodityCrude
	}
	baseID := fmt.Sprintf("connector-%s-%s", node.ID, pipelineNode.ID)
	switch node.Kind {
	case NodeImportTerminal, NodeRefinery:
		addConnectorEdge(graph, existingEdgeIDs, baseID, pipelineNode.ID, node.ID, commodity, distanceKM, connectorCapacity(node))
	case NodeStorageHub:
		addConnectorEdge(graph, existingEdgeIDs, baseID+"-in", pipelineNode.ID, node.ID, commodity, distanceKM, connectorCapacity(node))
		addConnectorEdge(graph, existingEdgeIDs, baseID+"-out", node.ID, pipelineNode.ID, commodity, distanceKM, connectorCapacity(node))
	default:
		addConnectorEdge(graph, existingEdgeIDs, baseID, node.ID, pipelineNode.ID, commodity, distanceKM, connectorCapacity(node))
	}
}

func addConnectorEdge(graph *Graph, existingEdgeIDs map[string]struct{}, id, fromID, toID string, commodity Commodity, distanceKM, capacityBPD float64) {
	if _, exists := existingEdgeIDs[id]; exists {
		return
	}
	graph.Edges = append(graph.Edges, Edge{
		ID:             id,
		Name:           "Connector " + id,
		Kind:           EdgeInternalBus,
		FromNodeID:     fromID,
		ToNodeID:       toID,
		Commodity:      commodity,
		State:          StateOperational,
		CapacityBPD:    capacityBPD,
		CurrentFlowBPD: capacityBPD,
		LengthKM:       distanceKM,
	})
	existingEdgeIDs[id] = struct{}{}
}

func connectorCapacity(node Node) float64 {
	if node.CapacityBPD > 0 {
		return node.CapacityBPD
	}
	if node.CrudeIntakeBPD > 0 {
		return node.CrudeIntakeBPD
	}
	if node.CurrentFlowBPD > 0 {
		return node.CurrentFlowBPD
	}
	return 100_000
}

func appendIfMissing(items []string, id string) []string {
	for _, existing := range items {
		if existing == id {
			return items
		}
	}
	return append(items, id)
}

func haversineKM(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKM = 6371.0
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKM * c
}
