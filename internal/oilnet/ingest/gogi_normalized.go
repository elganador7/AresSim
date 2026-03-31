package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aressim/internal/oilnet"
)

var gogiLayerKinds = map[string]oilnet.NodeKind{
	"refineries.ndjson":              oilnet.NodeRefinery,
	"ports.ndjson":                   oilnet.NodeMarineTerminal,
	"lng.ndjson":                     oilnet.NodeMarineTerminal,
	"storage.ndjson":                 oilnet.NodeStorageHub,
	"underground_storage.ndjson":     oilnet.NodeStorageHub,
	"processing_plants.ndjson":       oilnet.NodeGatheringHub,
	"platforms_and_well_pads.ndjson": oilnet.NodeGatheringHub,
	"stations.ndjson":                oilnet.NodeGatheringHub,
}

type gogiNormalizedRow struct {
	SourceLayer      string    `json:"sourceLayer"`
	Category         string    `json:"category"`
	SourceKey        string    `json:"sourceKey"`
	Name             string    `json:"name"`
	Country          string    `json:"country"`
	Status           string    `json:"status"`
	Type             string    `json:"type"`
	Commodity        string    `json:"commodity"`
	Capacity         any       `json:"capacity"`
	Throughput       any       `json:"throughput"`
	Operator         string    `json:"operator"`
	OnshoreOffshore  string    `json:"onshoreOffshore"`
	InstallationDate string    `json:"installationDate"`
	GeometryType     string    `json:"geometryType"`
	Centroid         []float64 `json:"centroid"`
	Bounds           []float64 `json:"bounds"`
}

func LoadGOGINormalizedGraph(dir string) (*oilnet.Graph, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, nil
	}
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat gogi normalized dir: %w", err)
	}
	graph := &oilnet.Graph{
		ID:          "oilnet-gogi-overlay",
		Name:        "GOGI Infrastructure Overlay",
		Description: "Normalized GOGI infrastructure overlay for refineries, ports, storage, LNG, and related oil and gas facilities.",
		Version:     "v1",
		View:        "global",
		Sources: []oilnet.SourceRef{
			{
				Name:         "GOGI",
				Organization: "EDX",
				Confidence:   0.78,
				Notes:        "Normalized overlay generated from the Global Oil and Gas Features geodatabase.",
			},
		},
	}
	for fileName, kind := range gogiLayerKinds {
		path := filepath.Join(dir, "layers", fileName)
		nodes, err := loadGOGINodes(path, kind)
		if err != nil {
			return nil, fmt.Errorf("load GOGI layer %s: %w", fileName, err)
		}
		graph.Nodes = append(graph.Nodes, nodes...)
	}
	return graph, nil
}

func loadGOGINodes(path string, kind oilnet.NodeKind) ([]oilnet.Node, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	nodes := make([]oilnet.Node, 0, 1024)
	scanner := bufio.NewScanner(file)
	const maxCapacity = 8 * 1024 * 1024
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, maxCapacity)
	for scanner.Scan() {
		var row gogiNormalizedRow
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, fmt.Errorf("decode ndjson row: %w", err)
		}
		node, ok := gogiRowToNode(row, kind)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

func gogiRowToNode(row gogiNormalizedRow, kind oilnet.NodeKind) (oilnet.Node, bool) {
	if len(row.Centroid) < 2 {
		return oilnet.Node{}, false
	}
	name := cleanGOGIString(row.Name)
	if name == "" {
		name = fallbackGOGIName(row, kind)
	}
	status := cleanGOGIString(row.Status)
	state := oilnet.StateOperational
	if isGOGIOffline(status) {
		state = oilnet.StateOffline
	}
	tags := []string{
		"source:gogi",
		"gogi:layer:" + strings.ToLower(strings.TrimSpace(row.SourceLayer)),
		"gogi:category:" + strings.ToLower(strings.TrimSpace(row.Category)),
	}
	if status != "" {
		tags = append(tags, "status:"+strings.ToLower(status))
	}
	if value := cleanGOGIString(row.Type); value != "" {
		tags = append(tags, "type:"+strings.ToLower(value))
	}
	if value := cleanGOGIString(row.OnshoreOffshore); value != "" {
		tags = append(tags, "onshore_offshore:"+strings.ToLower(value))
	}

	node := oilnet.Node{
		ID:               buildGOGINodeID(row, kind),
		Name:             name,
		Kind:             kind,
		CountryCode:      cleanGOGIString(row.Country),
		Operator:         cleanGOGIString(row.Operator),
		Lat:              row.Centroid[1],
		Lon:              row.Centroid[0],
		State:            state,
		PrimaryCommodity: mapGOGICommodity(row.Commodity),
		ProductsHandled:  mapGOGIProducts(row.Commodity),
		CapacityBPD:      parseGOGINumber(row.Capacity),
		CurrentFlowBPD:   parseGOGINumber(row.Throughput),
		Tags:             tags,
		Sources: []oilnet.SourceRef{
			{
				Name:         "GOGI",
				Organization: "EDX",
				Confidence:   0.78,
				Notes:        "Normalized GOGI infrastructure overlay.",
			},
		},
	}
	if kind == oilnet.NodeMarineTerminal {
		node.TerminalType = cleanGOGIString(row.Category)
	}
	if kind == oilnet.NodeRefinery {
		node.CrudeIntakeBPD = node.CapacityBPD
	}
	return node, true
}

func buildGOGINodeID(row gogiNormalizedRow, kind oilnet.NodeKind) string {
	if key := cleanGOGIString(row.SourceKey); key != "" {
		return "gogi:" + string(kind) + ":" + slugifyGOGI(key)
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(strings.Join([]string{
		string(kind),
		cleanGOGIString(row.SourceLayer),
		cleanGOGIString(row.Name),
		cleanGOGIString(row.Country),
		cleanGOGIString(row.Type),
		fmt.Sprintf("%.6f", safeCoord(row.Centroid, 0)),
		fmt.Sprintf("%.6f", safeCoord(row.Centroid, 1)),
	}, "|")))
	return fmt.Sprintf("gogi:%s:%x", kind, hash.Sum64())
}

func safeCoord(coords []float64, idx int) float64 {
	if idx < len(coords) {
		return coords[idx]
	}
	return 0
}

func cleanGOGIString(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	if strings.EqualFold(value, "NA") || value == "" {
		return ""
	}
	return strings.Join(strings.Fields(value), " ")
}

func fallbackGOGIName(row gogiNormalizedRow, kind oilnet.NodeKind) string {
	switch kind {
	case oilnet.NodeRefinery:
		return "Unnamed Refinery"
	case oilnet.NodeMarineTerminal:
		return "Unnamed Terminal"
	case oilnet.NodeStorageHub:
		return "Unnamed Storage Site"
	default:
		return "Unnamed " + strings.Title(strings.ReplaceAll(string(kind), "_", " "))
	}
}

func parseGOGINumber(value any) float64 {
	switch raw := value.(type) {
	case float64:
		return raw
	case string:
		clean := strings.TrimSpace(raw)
		if clean == "" || strings.EqualFold(clean, "NA") {
			return 0
		}
		clean = strings.ReplaceAll(clean, ",", "")
		fields := strings.FieldsFunc(clean, func(r rune) bool {
			return !(r == '.' || r == '-' || (r >= '0' && r <= '9'))
		})
		if len(fields) == 0 {
			return 0
		}
		parsed, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func mapGOGICommodity(raw string) oilnet.Commodity {
	text := strings.ToLower(cleanGOGIString(raw))
	switch {
	case strings.Contains(text, "natural gas"):
		return oilnet.CommodityNGL
	case strings.Contains(text, "lpg"):
		return oilnet.CommodityLPG
	case strings.Contains(text, "naphtha"):
		return oilnet.CommodityNaphtha
	case strings.Contains(text, "gasoline"):
		return oilnet.CommodityGasoline
	case strings.Contains(text, "kerosene"), strings.Contains(text, "jet"):
		return oilnet.CommodityJet
	case strings.Contains(text, "gasoil"), strings.Contains(text, "diesel"):
		return oilnet.CommodityDiesel
	case strings.Contains(text, "fuel oil"), strings.Contains(text, "mazut"):
		return oilnet.CommodityFuelOil
	case strings.Contains(text, "oil"), strings.Contains(text, "crude"):
		return oilnet.CommodityCrude
	default:
		return ""
	}
}

func mapGOGIProducts(raw string) []oilnet.Commodity {
	text := strings.ToLower(cleanGOGIString(raw))
	if text == "" {
		return nil
	}
	var out []oilnet.Commodity
	add := func(c oilnet.Commodity) {
		if c == "" {
			return
		}
		for _, existing := range out {
			if existing == c {
				return
			}
		}
		out = append(out, c)
	}
	if strings.Contains(text, "natural gas") {
		add(oilnet.CommodityNGL)
	}
	if strings.Contains(text, "lpg") {
		add(oilnet.CommodityLPG)
	}
	if strings.Contains(text, "naphtha") {
		add(oilnet.CommodityNaphtha)
	}
	if strings.Contains(text, "gasoline") {
		add(oilnet.CommodityGasoline)
	}
	if strings.Contains(text, "kerosene") || strings.Contains(text, "jet") {
		add(oilnet.CommodityJet)
	}
	if strings.Contains(text, "gasoil") || strings.Contains(text, "diesel") {
		add(oilnet.CommodityDiesel)
	}
	if strings.Contains(text, "fuel oil") || strings.Contains(text, "mazut") {
		add(oilnet.CommodityFuelOil)
	}
	if strings.Contains(text, "oil") || strings.Contains(text, "crude") {
		add(oilnet.CommodityCrude)
	}
	if len(out) == 0 && text != "" {
		add(oilnet.CommodityRefinedProducts)
	}
	return out
}

func isGOGIOffline(status string) bool {
	status = strings.ToLower(cleanGOGIString(status))
	switch status {
	case "mothballed full", "mothballed partial", "closed down", "decommissioned", "terminated", "hold", "proposed":
		return true
	default:
		return false
	}
}

func slugifyGOGI(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
