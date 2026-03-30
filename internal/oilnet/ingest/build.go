package ingest

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aressim/internal/oilnet"
)

// BuildInput bundles normalized source rows into one graph build request.
type BuildInput struct {
	GraphID     string
	Name        string
	Description string
	Version     string
	Assets      []GEMAssetRow
	Balances    []JODIBalanceRow
	TradeFlows  []ComtradeFlowRow
	SourceRefs  []oilnet.SourceRef
}

// BuildGraph creates a graph from normalized source rows.
func BuildGraph(input BuildInput) (*oilnet.Graph, error) {
	graph := &oilnet.Graph{
		ID:          defaultString(input.GraphID, "oilnet-ingested"),
		Name:        defaultString(input.Name, "Ingested Oil Network"),
		Description: input.Description,
		Version:     defaultString(input.Version, "0.1.0"),
		View:        "global",
		Sources:     append([]oilnet.SourceRef(nil), input.SourceRefs...),
	}

	nodeByID := make(map[string]oilnet.Node)
	for _, asset := range input.Assets {
		node, ok := nodeFromAsset(asset)
		if !ok {
			continue
		}
		nodeByID[node.ID] = node
	}
	ensureStandardChokepoints(nodeByID)
	countryAnchors := computeCountryAnchors(nodeByID)
	jodiByCountryCommodity := make(map[string]JODIBalanceRow)
	for _, balance := range input.Balances {
		jodiByCountryCommodity[balance.CountryCode+"|"+string(balance.Commodity)] = balance
	}

	refineriesByCountry := make(map[string][]string)
	exportNodesByCountryCommodity := make(map[string][]string)
	demandNodesByCountry := make(map[string]string)
	for id, node := range nodeByID {
		if node.Kind == oilnet.NodeRefinery {
			refineriesByCountry[node.CountryCode] = append(refineriesByCountry[node.CountryCode], id)
		}
		if node.Kind == oilnet.NodeExportTerminal || node.Kind == oilnet.NodeExtractionSite {
			key := node.CountryCode + "|" + string(node.PrimaryCommodity)
			exportNodesByCountryCommodity[key] = append(exportNodesByCountryCommodity[key], id)
		}
	}

	for _, balance := range input.Balances {
		if balance.DemandBPD > 0 {
			id := demandNodeID(balance.CountryCode)
			demandNodesByCountry[balance.CountryCode] = id
			node := nodeByID[id]
			if node.ID == "" {
				anchor := countryAnchors[balance.CountryCode]
				if anchor == (anchorPoint{}) {
					anchor = defaultCountryAnchor(balance.CountryCode)
				}
				node = oilnet.Node{
					ID:               id,
					Name:             balance.CountryCode + " Demand Center",
					Kind:             oilnet.NodeDemandCenter,
					CountryCode:      balance.CountryCode,
					State:            oilnet.StateOperational,
					Lat:              anchor.Lat,
					Lon:              anchor.Lon,
					PrimaryCommodity: balance.Commodity,
				}
			}
			node.DemandProfile = upsertDemand(node.DemandProfile, balance.Commodity, balance.DemandBPD)
			node.CurrentFlowBPD += balance.DemandBPD
			node.CapacityBPD += balance.DemandBPD
			nodeByID[id] = node
		}
	}

	graph.Nodes = make([]oilnet.Node, 0, len(nodeByID))
	for _, node := range nodeByID {
		graph.Nodes = append(graph.Nodes, node)
	}
	sort.Slice(graph.Nodes, func(i, j int) bool { return graph.Nodes[i].ID < graph.Nodes[j].ID })

	graph.Edges = buildEdges(input, nodeByID, refineriesByCountry, exportNodesByCountryCommodity, demandNodesByCountry, jodiByCountryCommodity)
	if err := oilnet.ValidateGraph(graph); err != nil {
		return nil, err
	}
	return graph, nil
}

func nodeFromAsset(asset GEMAssetRow) (oilnet.Node, bool) {
	id := strings.TrimSpace(asset.ID)
	if id == "" {
		return oilnet.Node{}, false
	}
	node := oilnet.Node{
		ID:               id,
		Name:             defaultString(asset.Name, id),
		CountryCode:      strings.TrimSpace(asset.CountryCode),
		Operator:         strings.TrimSpace(asset.Operator),
		Lat:              asset.Lat,
		Lon:              asset.Lon,
		State:            oilnet.StateOperational,
		PrimaryCommodity: asset.Commodity,
		CapacityBPD:      asset.CapacityBPD,
		CurrentFlowBPD:   asset.CapacityBPD,
	}
	switch strings.ToLower(strings.TrimSpace(asset.AssetType)) {
	case "field", "extraction_site", "oil_field":
		node.Kind = oilnet.NodeExtractionSite
	case "gathering_hub", "hub":
		node.Kind = oilnet.NodeGatheringHub
	case "export_terminal", "terminal":
		node.Kind = oilnet.NodeExportTerminal
	case "import_terminal":
		node.Kind = oilnet.NodeImportTerminal
	case "refinery":
		node.Kind = oilnet.NodeRefinery
		node.CrudeIntakeBPD = asset.CapacityBPD
		node.ProductOutputs = []oilnet.ProductOutput{
			{Commodity: oilnet.CommodityGasoline, BPD: asset.CapacityBPD * 0.18},
			{Commodity: oilnet.CommodityDiesel, BPD: asset.CapacityBPD * 0.34},
			{Commodity: oilnet.CommodityJet, BPD: asset.CapacityBPD * 0.12},
			{Commodity: oilnet.CommodityFuelOil, BPD: asset.CapacityBPD * 0.14},
			{Commodity: oilnet.CommodityLPG, BPD: asset.CapacityBPD * 0.05},
			{Commodity: oilnet.CommodityNaphtha, BPD: asset.CapacityBPD * 0.09},
		}
	case "storage", "storage_hub":
		node.Kind = oilnet.NodeStorageHub
	default:
		return oilnet.Node{}, false
	}
	return node, true
}

func buildEdges(
	input BuildInput,
	nodeByID map[string]oilnet.Node,
	refineriesByCountry map[string][]string,
	exportNodesByCountryCommodity map[string][]string,
	demandNodesByCountry map[string]string,
	jodiByCountryCommodity map[string]JODIBalanceRow,
) []oilnet.Edge {
	edges := make([]oilnet.Edge, 0)
	seen := make(map[string]struct{})

	for _, balance := range input.Balances {
		refineries := refineriesByCountry[balance.CountryCode]
		exporters := exportNodesByCountryCommodity[balance.CountryCode+"|"+string(balance.Commodity)]
		if balance.RefineryIntakeBPD > 0 && len(refineries) > 0 && len(exporters) > 0 {
			perRefinery := balance.RefineryIntakeBPD / float64(len(refineries))
			for _, exporter := range exporters {
				for _, refinery := range refineries {
					id := fmt.Sprintf("edge-%s-%s-%s", exporter, refinery, balance.Commodity)
					if _, ok := seen[id]; ok {
						continue
					}
					seen[id] = struct{}{}
					edges = append(edges, oilnet.Edge{
						ID:             id,
						Name:           nodeByID[exporter].Name + " to " + nodeByID[refinery].Name,
						Kind:           oilnet.EdgePipeline,
						FromNodeID:     exporter,
						ToNodeID:       refinery,
						Commodity:      balance.Commodity,
						State:          oilnet.StateOperational,
						CapacityBPD:    perRefinery,
						CurrentFlowBPD: perRefinery,
					})
				}
			}
		}
	}

	for _, flow := range input.TradeFlows {
		fromCandidates := exportNodesByCountryCommodity[flow.ReporterCode+"|"+string(flow.Commodity)]
		toDemand := demandNodesByCountry[flow.PartnerCode]
		if len(fromCandidates) == 0 || toDemand == "" {
			continue
		}
		flowBPD := reconcileTradeFlow(flow, jodiByCountryCommodity)
		perRoute := flowBPD / float64(len(fromCandidates))
		for _, from := range fromCandidates {
			id := fmt.Sprintf("trade-%s-%s-%s", from, toDemand, flow.Commodity)
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			route, chokepoints, transitDays, lengthKM := inferShippingPath(nodeByID[from], nodeByID[toDemand])
			edges = append(edges, oilnet.Edge{
				ID:                  id,
				Name:                nodeByID[from].Name + " exports to " + nodeByID[toDemand].Name,
				Kind:                oilnet.EdgeShipping,
				FromNodeID:          from,
				ToNodeID:            toDemand,
				Commodity:           flow.Commodity,
				State:               oilnet.StateOperational,
				CapacityBPD:         perRoute,
				CurrentFlowBPD:      perRoute,
				TransitDays:         transitDays,
				LengthKM:            lengthKM,
				Route:               route,
				CrossesChokepoints:  chokepoints,
				CrossesChokepoint:   firstString(chokepoints),
			})
		}
	}

	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })
	return edges
}

func upsertDemand(profile []oilnet.CommodityQuantity, commodity oilnet.Commodity, bpd float64) []oilnet.CommodityQuantity {
	for i := range profile {
		if profile[i].Commodity == commodity {
			profile[i].BPD += bpd
			return profile
		}
	}
	return append(profile, oilnet.CommodityQuantity{Commodity: commodity, BPD: bpd})
}

func demandNodeID(countryCode string) string {
	return strings.ToLower(countryCode) + "-demand"
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

type anchorPoint struct {
	Lat float64
	Lon float64
}

func computeCountryAnchors(nodeByID map[string]oilnet.Node) map[string]anchorPoint {
	sumLat := make(map[string]float64)
	sumLon := make(map[string]float64)
	count := make(map[string]float64)
	for _, node := range nodeByID {
		if node.Kind == oilnet.NodeChokepoint || strings.TrimSpace(node.CountryCode) == "" {
			continue
		}
		sumLat[node.CountryCode] += node.Lat
		sumLon[node.CountryCode] += node.Lon
		count[node.CountryCode] += 1
	}
	out := make(map[string]anchorPoint, len(count))
	for country, n := range count {
		out[country] = anchorPoint{
			Lat: sumLat[country] / n,
			Lon: sumLon[country] / n,
		}
	}
	return out
}

func defaultCountryAnchor(country string) anchorPoint {
	switch strings.TrimSpace(strings.ToUpper(country)) {
	case "JPN":
		return anchorPoint{Lat: 35.676, Lon: 139.65}
	case "KOR":
		return anchorPoint{Lat: 37.566, Lon: 126.978}
	case "CHN":
		return anchorPoint{Lat: 31.23, Lon: 121.473}
	case "NLD":
		return anchorPoint{Lat: 51.922, Lon: 4.479}
	case "DEU":
		return anchorPoint{Lat: 53.551, Lon: 9.993}
	case "USA":
		return anchorPoint{Lat: 29.76, Lon: -95.369}
	case "SGP":
		return anchorPoint{Lat: 1.29, Lon: 103.85}
	case "IND":
		return anchorPoint{Lat: 19.076, Lon: 72.877}
	default:
		return anchorPoint{}
	}
}

func reconcileTradeFlow(flow ComtradeFlowRow, jodiByCountryCommodity map[string]JODIBalanceRow) float64 {
	reporter := jodiByCountryCommodity[flow.ReporterCode+"|"+string(flow.Commodity)]
	partner := jodiByCountryCommodity[flow.PartnerCode+"|"+string(flow.Commodity)]
	value := flow.FlowBPD
	if reporter.ExportsBPD > 0 && reporter.ExportsBPD < value {
		value = reporter.ExportsBPD
	}
	if partner.ImportsBPD > 0 && partner.ImportsBPD < value {
		value = partner.ImportsBPD
	}
	return value
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
