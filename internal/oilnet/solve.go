package oilnet

import (
	"fmt"
	"slices"
)

// ShockRequest describes a supply-shock simulation against the baseline graph.
type ShockRequest struct {
	OfflineNodeIDs  []string `json:"offlineNodeIds,omitempty"`
	OfflineEdgeIDs  []string `json:"offlineEdgeIds,omitempty"`
	DegradedNodeIDs []string `json:"degradedNodeIds,omitempty"`
	DegradedEdgeIDs []string `json:"degradedEdgeIds,omitempty"`
}

// HorizonRequest extends a one-day outage with multi-day storage drawdown.
type HorizonRequest struct {
	ShockRequest
	Days int `json:"days"`
}

// CommoditySummary aggregates global loss and shortage by commodity.
type CommoditySummary struct {
	Commodity             Commodity `json:"commodity"`
	BaselineFlowBPD       float64   `json:"baselineFlowBpd"`
	CurrentFlowBPD        float64   `json:"currentFlowBpd"`
	LostFlowBPD           float64   `json:"lostFlowBpd"`
	BaselineDemandBPD     float64   `json:"baselineDemandBpd"`
	ServedDemandBPD       float64   `json:"servedDemandBpd"`
	UnmetDemandBPD        float64   `json:"unmetDemandBpd"`
	BaselineRefinedOutBPD float64   `json:"baselineRefinedOutputBpd"`
	CurrentRefinedOutBPD  float64   `json:"currentRefinedOutputBpd"`
}

// CountrySummary aggregates country-level supply and demand impacts.
type CountrySummary struct {
	CountryCode          string  `json:"countryCode"`
	BaselineFlowBPD      float64 `json:"baselineFlowBpd"`
	CurrentFlowBPD       float64 `json:"currentFlowBpd"`
	LostFlowBPD          float64 `json:"lostFlowBpd"`
	BaselineDemandBPD    float64 `json:"baselineDemandBpd"`
	ServedDemandBPD      float64 `json:"servedDemandBpd"`
	UnmetDemandBPD       float64 `json:"unmetDemandBpd"`
	RefinedOutputLossBPD float64 `json:"refinedOutputLossBpd"`
}

// ImpactedComponent describes a node or edge whose throughput changed materially.
type ImpactedComponent struct {
	ID              string           `json:"id"`
	Kind            string           `json:"kind"`
	Name            string           `json:"name"`
	BaselineFlowBPD float64          `json:"baselineFlowBpd"`
	CurrentFlowBPD  float64          `json:"currentFlowBpd"`
	LostFlowBPD     float64          `json:"lostFlowBpd"`
	State           OperationalState `json:"state"`
}

// ShockResult is the recomputed graph plus global summaries.
type ShockResult struct {
	Graph              *Graph              `json:"graph"`
	CommoditySummaries []CommoditySummary  `json:"commoditySummaries"`
	CountrySummaries   []CountrySummary    `json:"countrySummaries"`
	ImpactedComponents []ImpactedComponent `json:"impactedComponents"`
}

// HorizonDay captures one day in a multi-day outage simulation.
type HorizonDay struct {
	Day                int                 `json:"day"`
	CommoditySummaries []CommoditySummary  `json:"commoditySummaries"`
	CountrySummaries   []CountrySummary    `json:"countrySummaries"`
	ImpactedComponents []ImpactedComponent `json:"impactedComponents"`
}

// HorizonResult is a multi-day shock simulation.
type HorizonResult struct {
	Initial *ShockResult `json:"initial"`
	Days    []HorizonDay `json:"days"`
}

func stateFactor(state OperationalState) float64 {
	switch state {
	case StateOffline:
		return 0
	case StateDegraded:
		return 0.5
	default:
		return 1
	}
}

func cloneGraph(src *Graph) *Graph {
	if src == nil {
		return nil
	}
	out := *src
	out.Nodes = make([]Node, len(src.Nodes))
	copy(out.Nodes, src.Nodes)
	for i := range out.Nodes {
		if src.Nodes[i].ProductsHandled != nil {
			out.Nodes[i].ProductsHandled = append([]Commodity(nil), src.Nodes[i].ProductsHandled...)
		}
		if src.Nodes[i].ChildFieldIDs != nil {
			out.Nodes[i].ChildFieldIDs = append([]string(nil), src.Nodes[i].ChildFieldIDs...)
		}
		if src.Nodes[i].ProductOutputs != nil {
			out.Nodes[i].ProductOutputs = append([]ProductOutput(nil), src.Nodes[i].ProductOutputs...)
		}
		if src.Nodes[i].OutlineRings != nil {
			out.Nodes[i].OutlineRings = make([][]RoutePoint, len(src.Nodes[i].OutlineRings))
			for j := range src.Nodes[i].OutlineRings {
				out.Nodes[i].OutlineRings[j] = append([]RoutePoint(nil), src.Nodes[i].OutlineRings[j]...)
			}
		}
		if src.Nodes[i].DemandProfile != nil {
			out.Nodes[i].DemandProfile = append([]CommodityQuantity(nil), src.Nodes[i].DemandProfile...)
		}
		if src.Nodes[i].Inventory != nil {
			out.Nodes[i].Inventory = append([]CommodityQuantity(nil), src.Nodes[i].Inventory...)
		}
		if src.Nodes[i].Tags != nil {
			out.Nodes[i].Tags = append([]string(nil), src.Nodes[i].Tags...)
		}
		if src.Nodes[i].Sources != nil {
			out.Nodes[i].Sources = append([]SourceRef(nil), src.Nodes[i].Sources...)
		}
	}
	out.Edges = make([]Edge, len(src.Edges))
	copy(out.Edges, src.Edges)
	for i := range out.Edges {
		if src.Edges[i].Route != nil {
			out.Edges[i].Route = append([]RoutePoint(nil), src.Edges[i].Route...)
		}
		if src.Edges[i].Commodities != nil {
			out.Edges[i].Commodities = append([]Commodity(nil), src.Edges[i].Commodities...)
		}
		if src.Edges[i].Routes != nil {
			out.Edges[i].Routes = make([][]RoutePoint, len(src.Edges[i].Routes))
			for j := range src.Edges[i].Routes {
				out.Edges[i].Routes[j] = append([]RoutePoint(nil), src.Edges[i].Routes[j]...)
			}
		}
		if src.Edges[i].Sources != nil {
			out.Edges[i].Sources = append([]SourceRef(nil), src.Edges[i].Sources...)
		}
	}
	if src.Sources != nil {
		out.Sources = append([]SourceRef(nil), src.Sources...)
	}
	return &out
}

// SimulateShock recomputes a new graph after specified outages/degradations.
func SimulateShock(base *Graph, req ShockRequest) (*ShockResult, error) {
	if err := ValidateGraph(base); err != nil {
		return nil, err
	}
	graph := cloneGraph(base)
	nodeByID := make(map[string]*Node, len(graph.Nodes))
	baseNodeByID := make(map[string]Node, len(base.Nodes))
	for i := range graph.Nodes {
		nodeByID[graph.Nodes[i].ID] = &graph.Nodes[i]
		baseNodeByID[base.Nodes[i].ID] = base.Nodes[i]
	}
	edgeByID := make(map[string]*Edge, len(graph.Edges))
	baseEdgeByID := make(map[string]Edge, len(base.Edges))
	for i := range graph.Edges {
		edgeByID[graph.Edges[i].ID] = &graph.Edges[i]
		baseEdgeByID[graph.Edges[i].ID] = base.Edges[i]
	}

	for _, id := range req.DegradedNodeIDs {
		node, ok := nodeByID[id]
		if !ok {
			return nil, fmt.Errorf("unknown degraded node %q", id)
		}
		node.State = StateDegraded
	}
	for _, id := range req.OfflineNodeIDs {
		node, ok := nodeByID[id]
		if !ok {
			return nil, fmt.Errorf("unknown offline node %q", id)
		}
		node.State = StateOffline
	}
	for _, id := range req.DegradedEdgeIDs {
		edge, ok := edgeByID[id]
		if !ok {
			return nil, fmt.Errorf("unknown degraded edge %q", id)
		}
		edge.State = StateDegraded
	}
	for _, id := range req.OfflineEdgeIDs {
		edge, ok := edgeByID[id]
		if !ok {
			return nil, fmt.Errorf("unknown offline edge %q", id)
		}
		edge.State = StateOffline
	}

	chokepointFactor := make(map[string]float64)
	for _, node := range graph.Nodes {
		if node.Kind == NodeChokepoint {
			chokepointFactor[node.ID] = stateFactor(node.State)
		}
	}

	outgoingByNodeCommodity := map[string][]*Edge{}
	incomingByNodeCommodity := map[string][]*Edge{}
	edgeMaxFlow := make(map[string]float64, len(graph.Edges))
	keyFor := func(nodeID string, commodity Commodity) string {
		return nodeID + "|" + string(commodity)
	}
	for i := range graph.Edges {
		edge := &graph.Edges[i]
		factor := stateFactor(edge.State)
		fromFactor := stateFactor(nodeByID[edge.FromNodeID].State)
		toFactor := stateFactor(nodeByID[edge.ToNodeID].State)
		for _, chokepoint := range edge.CrossesChokepoints {
			factor *= chokepointFactor[chokepoint]
		}
		if len(edge.CrossesChokepoints) == 0 && edge.CrossesChokepoint != "" {
			factor *= chokepointFactor[edge.CrossesChokepoint]
		}
		factor = minFloat(factor, fromFactor, toFactor)
		edgeMaxFlow[edge.ID] = edge.CapacityBPD * factor
		edge.CurrentFlowBPD = minFloat(baseEdgeByID[edge.ID].CurrentFlowBPD, edgeMaxFlow[edge.ID])
		outgoingByNodeCommodity[keyFor(edge.FromNodeID, edge.Commodity)] = append(outgoingByNodeCommodity[keyFor(edge.FromNodeID, edge.Commodity)], edge)
		incomingByNodeCommodity[keyFor(edge.ToNodeID, edge.Commodity)] = append(incomingByNodeCommodity[keyFor(edge.ToNodeID, edge.Commodity)], edge)
	}

	for i := range graph.Nodes {
		node := &graph.Nodes[i]
		baseNode := baseNodeByID[node.ID]
		factor := stateFactor(node.State)
		switch node.Kind {
		case NodeExtractionSite, NodeGatheringHub, NodeExportTerminal, NodeImportTerminal, NodeStorageHub, NodeChokepoint:
			target := baseNode.CurrentFlowBPD * factor
			limitOutgoing(node, outgoingByNodeCommodity, edgeMaxFlow, target)
			node.CurrentFlowBPD = target
			node.SpareCapacityBPD = maxFloat(0, node.CapacityBPD-target)
		case NodeRefinery:
			inboundCrude := sumEdgeFlows(incomingByNodeCommodity[keyFor(node.ID, CommodityCrude)])
			maxIntake := minFloat(baseNode.CrudeIntakeBPD*factor, node.CapacityBPD*factor)
			node.CrudeIntakeBPD = minFloat(inboundCrude, maxIntake)
			ratio := 0.0
			if baseNode.CrudeIntakeBPD > 0 {
				ratio = node.CrudeIntakeBPD / baseNode.CrudeIntakeBPD
			}
			node.CurrentFlowBPD = node.CrudeIntakeBPD
			node.SpareCapacityBPD = maxFloat(0, node.CapacityBPD-node.CrudeIntakeBPD)
			for j := range node.ProductOutputs {
				node.ProductOutputs[j].BPD = baseNode.ProductOutputs[j].BPD * ratio
			}
			for _, product := range node.ProductOutputs {
				limitOutgoing(node, outgoingByNodeCommodity, edgeMaxFlow, product.BPD, product.Commodity)
			}
		case NodeDemandCenter:
			node.CurrentFlowBPD = sumAllIncoming(node.ID, incomingByNodeCommodity)
			node.SpareCapacityBPD = maxFloat(0, demandBPD(baseNode)-node.CurrentFlowBPD)
		default:
			node.CurrentFlowBPD = baseNode.CurrentFlowBPD * factor
		}
	}

	return buildShockResult(base, graph), nil
}

// SimulateShockHorizon models multi-day outages with storage drawdown.
func SimulateShockHorizon(base *Graph, req HorizonRequest) (*HorizonResult, error) {
	if req.Days <= 0 {
		req.Days = 7
	}
	initial, err := SimulateShock(base, req.ShockRequest)
	if err != nil {
		return nil, err
	}
	rolling := cloneGraph(initial.Graph)
	horizon := &HorizonResult{
		Initial: initial,
		Days:    make([]HorizonDay, 0, req.Days),
	}
	for day := 1; day <= req.Days; day++ {
		applyStorageDrawdown(base, rolling)
		dayResult := buildShockResult(base, rolling)
		horizon.Days = append(horizon.Days, HorizonDay{
			Day:                day,
			CommoditySummaries: dayResult.CommoditySummaries,
			CountrySummaries:   dayResult.CountrySummaries,
			ImpactedComponents: dayResult.ImpactedComponents,
		})
	}
	return horizon, nil
}

func applyStorageDrawdown(base, graph *Graph) {
	baseNodeByID := make(map[string]Node, len(base.Nodes))
	currentNodeByID := make(map[string]*Node, len(graph.Nodes))
	for _, node := range base.Nodes {
		baseNodeByID[node.ID] = node
	}
	for i := range graph.Nodes {
		currentNodeByID[graph.Nodes[i].ID] = &graph.Nodes[i]
	}
	outgoing := map[string][]*Edge{}
	incoming := map[string][]*Edge{}
	for i := range graph.Edges {
		edge := &graph.Edges[i]
		outgoing[edge.FromNodeID+"|"+string(edge.Commodity)] = append(outgoing[edge.FromNodeID+"|"+string(edge.Commodity)], edge)
		incoming[edge.ToNodeID+"|"+string(edge.Commodity)] = append(incoming[edge.ToNodeID+"|"+string(edge.Commodity)], edge)
	}
	for _, node := range graph.Nodes {
		if node.Kind != NodeDemandCenter || len(node.DemandProfile) == 0 {
			continue
		}
		currentDemand := currentNodeByID[node.ID]
		for _, demand := range node.DemandProfile {
			served := sumEdgeFlows(incoming[node.ID+"|"+string(demand.Commodity)])
			shortfall := maxFloat(0, demand.BPD-served)
			if shortfall <= 0 {
				continue
			}
			for i := range graph.Nodes {
				storage := &graph.Nodes[i]
				if storage.Kind != NodeStorageHub {
					continue
				}
				inv := inventoryFor(storage, demand.Commodity)
				if inv == nil || inv.Barrels <= 0 {
					continue
				}
				edges := outgoing[storage.ID+"|"+string(demand.Commodity)]
				if len(edges) == 0 {
					continue
				}
				dailyLimit := storage.DailyDrawLimitBPD
				if dailyLimit <= 0 {
					dailyLimit = shortfall
				}
				draw := minFloat(shortfall, inv.Barrels, dailyLimit)
				if draw <= 0 {
					continue
				}
				redistributeEdges(edges, draw)
				inv.Barrels -= draw
				currentDemand.CurrentFlowBPD += draw
				shortfall -= draw
				if shortfall <= 0 {
					break
				}
			}
		}
	}
	// keep refinery output and source flow baseline as-is; only inventories shift in horizon mode.
	for i := range graph.Nodes {
		baseNode := baseNodeByID[graph.Nodes[i].ID]
		if graph.Nodes[i].Kind == NodeStorageHub {
			graph.Nodes[i].CurrentFlowBPD = sumAllOutgoing(graph.Nodes[i].ID, graph.Edges)
			graph.Nodes[i].SpareCapacityBPD = maxFloat(0, baseNode.CapacityBPD-graph.Nodes[i].CurrentFlowBPD)
		}
	}
}

func inventoryFor(node *Node, commodity Commodity) *CommodityQuantity {
	for i := range node.Inventory {
		if node.Inventory[i].Commodity == commodity {
			return &node.Inventory[i]
		}
	}
	return nil
}

func limitOutgoing(node *Node, outgoing map[string][]*Edge, edgeMaxFlow map[string]float64, target float64, commodities ...Commodity) {
	if len(commodities) == 0 {
		if node.PrimaryCommodity == "" {
			return
		}
		commodities = []Commodity{node.PrimaryCommodity}
	}
	for _, commodity := range commodities {
		edges := outgoing[node.ID+"|"+string(commodity)]
		if len(edges) == 0 {
			continue
		}
		maxAllowed := 0.0
		for _, edge := range edges {
			maxAllowed += edgeMaxFlow[edge.ID]
		}
		redistributeEdgesWithCap(edges, minFloat(target, maxAllowed), edgeMaxFlow)
	}
}

func redistributeEdges(edges []*Edge, target float64) {
	edgeMax := make(map[string]float64, len(edges))
	for _, edge := range edges {
		edgeMax[edge.ID] = edge.CapacityBPD
	}
	redistributeEdgesWithCap(edges, target, edgeMax)
}

func redistributeEdgesWithCap(edges []*Edge, target float64, edgeMaxFlow map[string]float64) {
	if len(edges) == 0 {
		return
	}
	totalMax := 0.0
	totalCurrent := 0.0
	for _, edge := range edges {
		totalMax += edgeMaxFlow[edge.ID]
		totalCurrent += edge.CurrentFlowBPD
	}
	if totalMax <= 0 || target <= 0 {
		for _, edge := range edges {
			edge.CurrentFlowBPD = 0
		}
		return
	}
	if target >= totalMax {
		for _, edge := range edges {
			edge.CurrentFlowBPD = edgeMaxFlow[edge.ID]
		}
		return
	}
	remaining := target
	if totalCurrent > 0 {
		for _, edge := range edges {
			share := edge.CurrentFlowBPD / totalCurrent
			edge.CurrentFlowBPD = minFloat(edgeMaxFlow[edge.ID], target*share)
			remaining -= edge.CurrentFlowBPD
		}
	}
	for remaining > 1 {
		progress := false
		for _, edge := range edges {
			spare := edgeMaxFlow[edge.ID] - edge.CurrentFlowBPD
			if spare <= 0 {
				continue
			}
			add := minFloat(spare, remaining)
			edge.CurrentFlowBPD += add
			remaining -= add
			progress = true
			if remaining <= 1 {
				break
			}
		}
		if !progress {
			break
		}
	}
}

func sumEdgeFlows(edges []*Edge) float64 {
	total := 0.0
	for _, edge := range edges {
		total += edge.CurrentFlowBPD
	}
	return total
}

func sumAllIncoming(nodeID string, incoming map[string][]*Edge) float64 {
	total := 0.0
	for key, edges := range incoming {
		if len(key) < len(nodeID)+1 || key[:len(nodeID)] != nodeID || key[len(nodeID)] != '|' {
			continue
		}
		total += sumEdgeFlows(edges)
	}
	return total
}

func sumAllOutgoing(nodeID string, edges []Edge) float64 {
	total := 0.0
	for _, edge := range edges {
		if edge.FromNodeID == nodeID {
			total += edge.CurrentFlowBPD
		}
	}
	return total
}

func demandBPD(node Node) float64 {
	if len(node.DemandProfile) == 0 {
		return node.CurrentFlowBPD
	}
	total := 0.0
	for _, demand := range node.DemandProfile {
		total += demand.BPD
	}
	return total
}

func buildShockResult(base, current *Graph) *ShockResult {
	commodityMap := map[Commodity]*CommoditySummary{}
	countryMap := map[string]*CountrySummary{}
	impact := make([]ImpactedComponent, 0)
	baseNodeByID := make(map[string]Node, len(base.Nodes))
	for _, node := range base.Nodes {
		baseNodeByID[node.ID] = node
	}
	for _, edge := range current.Edges {
		baseEdge := findEdge(base.Edges, edge.ID)
		summary := ensureCommoditySummary(commodityMap, edge.Commodity)
		summary.BaselineFlowBPD += baseEdge.CurrentFlowBPD
		summary.CurrentFlowBPD += edge.CurrentFlowBPD
		fromCountry := baseNodeByID[edge.FromNodeID].CountryCode
		country := ensureCountrySummary(countryMap, fromCountry)
		country.BaselineFlowBPD += baseEdge.CurrentFlowBPD
		country.CurrentFlowBPD += edge.CurrentFlowBPD
		if lost := baseEdge.CurrentFlowBPD - edge.CurrentFlowBPD; lost > 1 {
			impact = append(impact, ImpactedComponent{
				ID:              edge.ID,
				Kind:            string(edge.Kind),
				Name:            edge.Name,
				BaselineFlowBPD: baseEdge.CurrentFlowBPD,
				CurrentFlowBPD:  edge.CurrentFlowBPD,
				LostFlowBPD:     lost,
				State:           edge.State,
			})
		}
	}
	for _, node := range current.Nodes {
		baseNode := baseNodeByID[node.ID]
		country := ensureCountrySummary(countryMap, node.CountryCode)
		if node.Kind == NodeDemandCenter {
			if len(baseNode.DemandProfile) == 0 {
				commodity := ensureCommoditySummary(commodityMap, node.PrimaryCommodity)
				commodity.BaselineDemandBPD += baseNode.CurrentFlowBPD
				commodity.ServedDemandBPD += node.CurrentFlowBPD
				country.BaselineDemandBPD += baseNode.CurrentFlowBPD
				country.ServedDemandBPD += node.CurrentFlowBPD
			} else {
				for _, demand := range baseNode.DemandProfile {
					commodity := ensureCommoditySummary(commodityMap, demand.Commodity)
					commodity.BaselineDemandBPD += demand.BPD
					served := servedDemandForNode(current, node.ID, demand.Commodity)
					commodity.ServedDemandBPD += served
					country.BaselineDemandBPD += demand.BPD
					country.ServedDemandBPD += served
				}
			}
		}
		if node.Kind == NodeRefinery {
			for _, output := range node.ProductOutputs {
				baseOut := productOutputFor(baseNode, output.Commodity)
				commodity := ensureCommoditySummary(commodityMap, output.Commodity)
				commodity.BaselineRefinedOutBPD += baseOut
				commodity.CurrentRefinedOutBPD += output.BPD
				country.RefinedOutputLossBPD += maxFloat(0, baseOut-output.BPD)
			}
		}
		if lost := baseNode.CurrentFlowBPD - node.CurrentFlowBPD; lost > 1 || node.State != baseNode.State {
			impact = append(impact, ImpactedComponent{
				ID:              node.ID,
				Kind:            string(node.Kind),
				Name:            node.Name,
				BaselineFlowBPD: baseNode.CurrentFlowBPD,
				CurrentFlowBPD:  node.CurrentFlowBPD,
				LostFlowBPD:     maxFloat(0, baseNode.CurrentFlowBPD-node.CurrentFlowBPD),
				State:           node.State,
			})
		}
	}
	commodities := make([]CommoditySummary, 0, len(commodityMap))
	for _, summary := range commodityMap {
		summary.LostFlowBPD = maxFloat(0, summary.BaselineFlowBPD-summary.CurrentFlowBPD)
		summary.UnmetDemandBPD = maxFloat(0, summary.BaselineDemandBPD-summary.ServedDemandBPD)
		commodities = append(commodities, *summary)
	}
	slices.SortFunc(commodities, func(a, b CommoditySummary) int { return cmpString(string(a.Commodity), string(b.Commodity)) })
	countries := make([]CountrySummary, 0, len(countryMap))
	for _, summary := range countryMap {
		summary.LostFlowBPD = maxFloat(0, summary.BaselineFlowBPD-summary.CurrentFlowBPD)
		summary.UnmetDemandBPD = maxFloat(0, summary.BaselineDemandBPD-summary.ServedDemandBPD)
		countries = append(countries, *summary)
	}
	slices.SortFunc(countries, func(a, b CountrySummary) int { return cmpString(a.CountryCode, b.CountryCode) })
	slices.SortFunc(impact, func(a, b ImpactedComponent) int {
		if a.LostFlowBPD == b.LostFlowBPD {
			return cmpString(a.ID, b.ID)
		}
		if a.LostFlowBPD > b.LostFlowBPD {
			return -1
		}
		return 1
	})
	return &ShockResult{
		Graph:              current,
		CommoditySummaries: commodities,
		CountrySummaries:   countries,
		ImpactedComponents: impact,
	}
}

func productOutputFor(node Node, commodity Commodity) float64 {
	for _, output := range node.ProductOutputs {
		if output.Commodity == commodity {
			return output.BPD
		}
	}
	return 0
}

func servedDemandForNode(graph *Graph, nodeID string, commodity Commodity) float64 {
	total := 0.0
	for _, edge := range graph.Edges {
		if edge.ToNodeID == nodeID && edge.Commodity == commodity {
			total += edge.CurrentFlowBPD
		}
	}
	return total
}

func ensureCommoditySummary(m map[Commodity]*CommoditySummary, commodity Commodity) *CommoditySummary {
	summary, ok := m[commodity]
	if !ok {
		summary = &CommoditySummary{Commodity: commodity}
		m[commodity] = summary
	}
	return summary
}

func ensureCountrySummary(m map[string]*CountrySummary, countryCode string) *CountrySummary {
	summary, ok := m[countryCode]
	if !ok {
		summary = &CountrySummary{CountryCode: countryCode}
		m[countryCode] = summary
	}
	return summary
}

func findEdge(edges []Edge, id string) Edge {
	for _, edge := range edges {
		if edge.ID == id {
			return edge
		}
	}
	return Edge{}
}

func minFloat(first float64, rest ...float64) float64 {
	min := first
	for _, value := range rest {
		if value < min {
			min = value
		}
	}
	return min
}

func maxFloat(first float64, rest ...float64) float64 {
	max := first
	for _, value := range rest {
		if value > max {
			max = value
		}
	}
	return max
}

func cmpString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
