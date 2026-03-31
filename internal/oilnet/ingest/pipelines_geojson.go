package ingest

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aressim/internal/oilnet"
)

var requiredPipelineProperties = []string{
	"ProjectID",
	"Fuel",
}

type pipelineFeatureCollection struct {
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	Features []pipelineFeature `json:"features"`
}

type pipelineFeature struct {
	Type       string               `json:"type"`
	Properties pipelineFeatureProps `json:"properties"`
	Geometry   pipelineFeatureGeom  `json:"geometry"`
}

type pipelineFeatureProps struct {
	ProjectID      string `json:"ProjectID"`
	Fuel           string `json:"Fuel"`
	PipelineName   string `json:"PipelineName"`
	SegmentName    string `json:"SegmentName"`
	Status         string `json:"Status"`
	Owner          string `json:"Owner"`
	Countries      string `json:"Countries"`
	CapacityBOEd   string `json:"CapacityBOEd"`
	LengthMergedKm string `json:"LengthMergedKm"`
	StartLocation  string `json:"StartLocation"`
	StartCountry   string `json:"StartCountry"`
	EndLocation    string `json:"EndLocation"`
	EndCountry     string `json:"EndCountry"`
	LastUpdated    string `json:"LastUpdated"`
	Wiki           string `json:"Wiki"`
}

type pipelineFeatureGeom struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

// LoadPipelinesGeoJSON builds a pipeline-only overlay graph from the provided GeoJSON file.
func LoadPipelinesGeoJSON(path string) (*oilnet.Graph, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read pipelines geojson: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	name := "Pipelines Overlay"
	graph := &oilnet.Graph{
		ID:          "oilnet-pipelines-overlay",
		Name:        name,
		Description: "Pipeline infrastructure overlay parsed directly from the GEM GOIT oil/NGL pipelines GeoJSON.",
		Version:     "0.1.0",
		View:        "global",
		Sources: []oilnet.SourceRef{
			{
				Name:         name,
				Organization: "Global Energy Monitor",
				URL:          path,
				Confidence:   0.95,
				Notes:        "Parsed directly from local GeoJSON pipeline dataset.",
			},
		},
	}
	root, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode pipelines geojson root: %w", err)
	}
	if delim, ok := root.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("unexpected geojson root token %v", root)
	}
	collectionTypeValidated := false
	nodeIDs := map[string]struct{}{}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("decode pipelines geojson key: %w", err)
		}
		key, _ := keyToken.(string)
		switch key {
		case "type":
			var collectionType string
			if err := decoder.Decode(&collectionType); err != nil {
				return nil, fmt.Errorf("decode pipelines geojson type: %w", err)
			}
			if strings.TrimSpace(collectionType) != "FeatureCollection" {
				return nil, fmt.Errorf("unexpected geojson type %q", collectionType)
			}
			collectionTypeValidated = true
		case "name":
			if err := decoder.Decode(&graph.Name); err != nil {
				return nil, fmt.Errorf("decode pipelines geojson name: %w", err)
			}
			graph.Sources[0].Name = graph.Name
		case "features":
			if err := decodePipelineFeatures(decoder, graph, nodeIDs); err != nil {
				return nil, err
			}
		default:
			var discard any
			if err := decoder.Decode(&discard); err != nil {
				return nil, fmt.Errorf("decode pipelines geojson field %q: %w", key, err)
			}
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("decode pipelines geojson close: %w", err)
	}
	if !collectionTypeValidated {
		return nil, fmt.Errorf("pipelines geojson missing required type field")
	}
	if err := oilnet.ValidateGraph(graph); err != nil {
		return nil, err
	}
	return graph, nil
}

func decodePipelineFeatures(decoder *json.Decoder, graph *oilnet.Graph, nodeIDs map[string]struct{}) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode features start: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("unexpected features token %v", token)
	}
	for decoder.More() {
		var feature pipelineFeature
		if err := decoder.Decode(&feature); err != nil {
			return fmt.Errorf("decode pipeline feature: %w", err)
		}
		if err := validatePipelineFeature(feature); err != nil {
			return err
		}
		routes, err := featureRoutes(feature.Geometry)
		if err != nil || len(routes) == 0 {
			continue
		}
		route := primaryRoute(routes)
		if len(route) < 2 {
			continue
		}
		commodities, commodity := pipelineCommodities(feature.Properties.Fuel)
		rawStatus := strings.ToLower(strings.TrimSpace(feature.Properties.Status))
		state := pipelineOperationalState(rawStatus)
		startNode := pipelineTerminalNode(feature.Properties.ProjectID+"-start", fallbackName(feature.Properties.StartLocation, feature.Properties.PipelineName+" Start"), countryCodeFromName(feature.Properties.StartCountry), route[0])
		endNode := pipelineTerminalNode(feature.Properties.ProjectID+"-end", fallbackName(feature.Properties.EndLocation, feature.Properties.PipelineName+" End"), countryCodeFromName(feature.Properties.EndCountry), route[len(route)-1])
		if _, ok := nodeIDs[startNode.ID]; !ok {
			graph.Nodes = append(graph.Nodes, startNode)
			nodeIDs[startNode.ID] = struct{}{}
		}
		if _, ok := nodeIDs[endNode.ID]; !ok {
			graph.Nodes = append(graph.Nodes, endNode)
			nodeIDs[endNode.ID] = struct{}{}
		}
		capacity := parsePipelineFloat(feature.Properties.CapacityBOEd)
		graph.Edges = append(graph.Edges, oilnet.Edge{
			ID:             "pipeline-" + feature.Properties.ProjectID,
			Name:           fullPipelineName(feature.Properties),
			Kind:           oilnet.EdgePipeline,
			FromNodeID:     startNode.ID,
			ToNodeID:       endNode.ID,
			Commodity:      commodity,
			Commodities:    commodities,
			CommodityLabel: strings.TrimSpace(feature.Properties.Fuel),
			State:          state,
			StatusDetail:   rawStatus,
			CapacityBPD:    capacity,
			CurrentFlowBPD: flowForState(capacity, state),
			LengthKM:       parsePipelineFloat(feature.Properties.LengthMergedKm),
			Route:          route,
			Routes:         routes,
			Sources: []oilnet.SourceRef{
				{
					Name:         feature.Properties.PipelineName,
					Organization: "Global Energy Monitor",
					URL:          feature.Properties.Wiki,
					LastUpdated:  feature.Properties.LastUpdated,
					Confidence:   0.95,
				},
			},
		})
	}
	_, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("decode features close: %w", err)
	}
	return nil
}

func validatePipelineFeature(feature pipelineFeature) error {
	if strings.TrimSpace(feature.Type) != "Feature" {
		return fmt.Errorf("unexpected pipeline feature type %q", feature.Type)
	}
	missing := make([]string, 0)
	props := map[string]string{
		"ProjectID": feature.Properties.ProjectID,
		"Fuel":      feature.Properties.Fuel,
		"Status":    feature.Properties.Status,
	}
	for _, key := range requiredPipelineProperties {
		if strings.TrimSpace(props[key]) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("pipeline feature missing required properties: %s", strings.Join(missing, ", "))
	}
	return nil
}

func featureRoutes(geom pipelineFeatureGeom) ([][]oilnet.RoutePoint, error) {
	switch geom.Type {
	case "LineString":
		var coords [][]float64
		if err := json.Unmarshal(geom.Coordinates, &coords); err != nil {
			return nil, err
		}
		route := toRoute(coords)
		if len(route) < 2 {
			return nil, fmt.Errorf("short route")
		}
		return [][]oilnet.RoutePoint{route}, nil
	case "MultiLineString":
		var lines [][][]float64
		if err := json.Unmarshal(geom.Coordinates, &lines); err != nil {
			return nil, err
		}
		routes := make([][]oilnet.RoutePoint, 0, len(lines))
		for _, line := range lines {
			route := toRoute(line)
			if len(route) < 2 {
				continue
			}
			routes = append(routes, route)
		}
		return routes, nil
	default:
		return nil, fmt.Errorf("unsupported geometry type %q", geom.Type)
	}
}

func primaryRoute(routes [][]oilnet.RoutePoint) []oilnet.RoutePoint {
	longest := []oilnet.RoutePoint(nil)
	for _, route := range routes {
		if len(route) > len(longest) {
			longest = route
		}
	}
	return longest
}

func pipelineCommodities(rawFuel string) ([]oilnet.Commodity, oilnet.Commodity) {
	raw := strings.ToLower(strings.TrimSpace(rawFuel))
	if raw == "" {
		return []oilnet.Commodity{oilnet.CommodityCrude}, oilnet.CommodityCrude
	}
	parts := strings.Split(raw, ",")
	seen := map[oilnet.Commodity]struct{}{}
	out := make([]oilnet.Commodity, 0, len(parts))
	add := func(c oilnet.Commodity) {
		if _, exists := seen[c]; exists {
			return
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	for _, part := range parts {
		token := strings.TrimSpace(part)
		switch {
		case token == "oil":
			add(oilnet.CommodityCrude)
		case token == "ngl":
			add(oilnet.CommodityNGL)
		case token == "lpg":
			add(oilnet.CommodityLPG)
		case token == "naphtha":
			add(oilnet.CommodityNaphtha)
		case token == "oil products":
			add(oilnet.CommodityRefinedProducts)
		default:
			if strings.Contains(token, "naphtha") {
				add(oilnet.CommodityNaphtha)
			}
			if strings.Contains(token, "lpg") {
				add(oilnet.CommodityLPG)
			}
			if strings.Contains(token, "ngl") {
				add(oilnet.CommodityNGL)
			}
			if strings.Contains(token, "oil product") {
				add(oilnet.CommodityRefinedProducts)
			} else if strings.Contains(token, "oil") || strings.Contains(token, "condensate") {
				add(oilnet.CommodityCrude)
			}
		}
	}
	if len(out) == 0 {
		out = append(out, oilnet.CommodityCrude)
	}
	return out, out[0]
}

func toRoute(coords [][]float64) []oilnet.RoutePoint {
	route := make([]oilnet.RoutePoint, 0, len(coords))
	for _, coord := range coords {
		if len(coord) < 2 {
			continue
		}
		route = append(route, oilnet.RoutePoint{Lon: coord[0], Lat: coord[1]})
	}
	return route
}

func pipelineTerminalNode(id, name, countryCode string, point oilnet.RoutePoint) oilnet.Node {
	return oilnet.Node{
		ID:               "pipe-node-" + id,
		Name:             name,
		Kind:             oilnet.NodePipelineTerminal,
		CountryCode:      countryCode,
		Lat:              point.Lat,
		Lon:              point.Lon,
		State:            oilnet.StateOperational,
		PrimaryCommodity: oilnet.CommodityCrude,
	}
}

func fullPipelineName(props pipelineFeatureProps) string {
	if strings.TrimSpace(props.SegmentName) == "" {
		return props.PipelineName
	}
	return props.PipelineName + " / " + props.SegmentName
}

func parsePipelineFloat(raw string) float64 {
	cleaned := strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	if cleaned == "" || cleaned == "--" {
		return 0
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	return value
}

func flowForState(capacity float64, state oilnet.OperationalState) float64 {
	switch state {
	case oilnet.StateOffline:
		return 0
	case oilnet.StateDegraded:
		return 0
	default:
		return capacity
	}
}

func pipelineOperationalState(status string) oilnet.OperationalState {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "operating":
		return oilnet.StateOperational
	case "construction", "pre-construction", "idle", "mothballed":
		return oilnet.StateDegraded
	case "shelved", "cancelled", "canceled", "retired", "proposed":
		return oilnet.StateOffline
	case "":
		return oilnet.StateOffline
	default:
		return oilnet.StateOperational
	}
}

func fallbackName(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func countryCodeFromName(name string) string {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "canada":
		return "CAN"
	case "united states":
		return "USA"
	case "saudi arabia":
		return "SAU"
	case "united arab emirates":
		return "ARE"
	case "kuwait":
		return "KWT"
	case "iraq":
		return "IRQ"
	case "iran":
		return "IRN"
	case "netherlands":
		return "NLD"
	case "germany":
		return "DEU"
	case "japan":
		return "JPN"
	case "south korea":
		return "KOR"
	case "singapore":
		return "SGP"
	case "china":
		return "CHN"
	case "india":
		return "IND"
	case "belgium":
		return "BEL"
	case "united kingdom":
		return "GBR"
	default:
		return strings.ToUpper(strings.TrimSpace(name))
	}
}
