package ingest

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/aressim/internal/oilnet"
	"github.com/aressim/internal/oilnet/maritime"
)

//go:embed data/topology_seed.json
var topologySeedJSON []byte

func LoadEmbeddedTopology() (*maritime.Topology, error) {
	var topology maritime.Topology
	if err := json.Unmarshal(topologySeedJSON, &topology); err != nil {
		return nil, fmt.Errorf("decode embedded maritime topology: %w", err)
	}
	if topology.ID == "" {
		return nil, fmt.Errorf("embedded maritime topology missing id")
	}
	return &topology, nil
}

func TopologyToGraph(topology *maritime.Topology) (*oilnet.Graph, error) {
	if topology == nil {
		return nil, fmt.Errorf("maritime topology is nil")
	}
	graph := &oilnet.Graph{
		ID:          topology.ID,
		Name:        topology.Name,
		Description: topology.Description,
		Version:     topology.Version,
		View:        "global",
		Sources:     append([]oilnet.SourceRef(nil), topology.Sources...),
	}

	for _, terminal := range topology.Terminals {
		graph.Nodes = append(graph.Nodes, oilnet.Node{
			ID:                 terminal.ID,
			Name:               terminal.Name,
			Kind:               oilnet.NodeMarineTerminal,
			CountryCode:        terminal.CountryCode,
			Operator:           terminal.Operator,
			Lat:                terminal.Lat,
			Lon:                terminal.Lon,
			State:              terminal.State,
			PrimaryCommodity:   firstCommodity(terminal.ProductsHandled),
			ProductsHandled:    append([]oilnet.Commodity(nil), terminal.ProductsHandled...),
			TerminalType:       string(terminal.TerminalType),
			DraftClass:         terminal.DraftClass,
			BerthCount:         terminal.BerthCount,
			CapacityBPD:        terminal.CapacityBPD,
			CurrentFlowBPD:     terminal.CapacityBPD,
			StorageCapacityBbl: terminal.StorageCapacityBbl,
			Tags:               append([]string(nil), terminal.Tags...),
			Sources:            append([]oilnet.SourceRef(nil), terminal.Sources...),
		})
	}

	for _, canal := range topology.Canals {
		graph.Nodes = append(graph.Nodes, oilnet.Node{
			ID:               canal.NodeID,
			Name:             canal.Name,
			Kind:             oilnet.NodeCanal,
			CountryCode:      canal.CountryCode,
			Lat:              canal.Lat,
			Lon:              canal.Lon,
			State:            canal.State,
			CapacityBPD:      canal.CapacityBPD,
			CurrentFlowBPD:   canal.CapacityBPD,
			PrimaryCommodity: oilnet.CommodityCrude,
			Sources:          append([]oilnet.SourceRef(nil), canal.Sources...),
		})
	}

	for _, corridor := range topology.Corridors {
		graph.Edges = append(graph.Edges, oilnet.Edge{
			ID:                 corridor.ID,
			Name:               corridor.Name,
			Kind:               oilnet.EdgeSeaborneCorridor,
			FromNodeID:         corridor.FromTerminalID,
			ToNodeID:           corridor.ToTerminalID,
			Commodity:          corridor.Commodity,
			Commodities:        append([]oilnet.Commodity(nil), corridor.ProductsHandled...),
			State:              corridor.State,
			VesselClass:        string(corridor.VesselClass),
			CapacityBPD:        corridor.CapacityBPD,
			CurrentFlowBPD:     corridor.CurrentFlowBPD,
			TransitDays:        corridor.TransitDays,
			LengthKM:           corridor.LengthKM,
			CrossesChokepoints: append([]string(nil), corridor.CrossesChokepoints...),
			CrossesChokepoint:  firstString(corridor.CrossesChokepoints),
			Route:              append([]oilnet.RoutePoint(nil), corridor.Route...),
			Sources:            append([]oilnet.SourceRef(nil), corridor.Sources...),
		})
	}

	if err := oilnet.ValidateGraph(graph); err != nil {
		return nil, err
	}
	return graph, nil
}

func firstCommodity(values []oilnet.Commodity) oilnet.Commodity {
	if len(values) == 0 {
		return oilnet.CommodityCrude
	}
	return values[0]
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
