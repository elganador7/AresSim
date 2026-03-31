package ingest

import (
	"archive/zip"
	"bytes"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/aressim/internal/oilnet"
)

func TestParseGEMAssetsCSV(t *testing.T) {
	rows, err := ParseGEMAssetsCSV(strings.NewReader("id,name,asset_type,country_code,operator,lat,lon,capacity_bpd,commodity\nsa-ghawar,Ghawar,field,SAU,Aramco,25.385,49.655,3800000,crude\n"))
	if err != nil {
		t.Fatalf("ParseGEMAssetsCSV() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Commodity != oilnet.CommodityCrude {
		t.Fatalf("expected crude commodity, got %q", rows[0].Commodity)
	}
}

func TestParseJODIBalanceCSV(t *testing.T) {
	rows, err := ParseJODIBalanceCSV(strings.NewReader("country_code,commodity,production_bpd,refinery_intake_bpd,demand_bpd,imports_bpd,exports_bpd\nJPN,jet_kerosene,0,0,3600000,380000,0\n"))
	if err != nil {
		t.Fatalf("ParseJODIBalanceCSV() error = %v", err)
	}
	if len(rows) != 1 || rows[0].DemandBPD != 3600000 {
		t.Fatalf("unexpected JODI parse result: %+v", rows)
	}
}

func TestParseComtradeFlowsCSV(t *testing.T) {
	rows, err := ParseComtradeFlowsCSV(strings.NewReader("reporter_code,partner_code,commodity,flow_bpd\nUSA,NLD,crude,690000\n"))
	if err != nil {
		t.Fatalf("ParseComtradeFlowsCSV() error = %v", err)
	}
	if len(rows) != 1 || rows[0].ReporterCode != "USA" || rows[0].FlowBPD != 690000 {
		t.Fatalf("unexpected Comtrade parse result: %+v", rows)
	}
}

func TestLoadSampleGraph(t *testing.T) {
	graph, err := LoadSampleGraph()
	if err != nil {
		t.Fatalf("LoadSampleGraph() error = %v", err)
	}
	if graph.ID != "oilnet-ingested-sample" {
		t.Fatalf("expected sample graph id, got %q", graph.ID)
	}
	if len(graph.Nodes) == 0 || len(graph.Edges) == 0 {
		t.Fatalf("expected sample graph to contain nodes and edges, got nodes=%d edges=%d", len(graph.Nodes), len(graph.Edges))
	}
}

func TestLoadGulfRegionalGraph(t *testing.T) {
	graph, err := LoadGulfRegionalGraph()
	if err != nil {
		t.Fatalf("LoadGulfRegionalGraph() error = %v", err)
	}
	if graph.ID != "oilnet-gulf-generated" {
		t.Fatalf("expected gulf graph id, got %q", graph.ID)
	}
	found := false
	for _, node := range graph.Nodes {
		if node.ID == "sa-abqaiq" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected gulf graph to contain sa-abqaiq")
	}
}

func TestLoadEuropeRegionalGraph(t *testing.T) {
	graph, err := LoadEuropeRegionalGraph()
	if err != nil {
		t.Fatalf("LoadEuropeRegionalGraph() error = %v", err)
	}
	if graph.ID != "oilnet-europe-generated" {
		t.Fatalf("expected europe graph id, got %q", graph.ID)
	}
	found := false
	for _, node := range graph.Nodes {
		if node.ID == "deu-demand" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected europe graph to contain generated Germany demand node")
	}
}

func TestLoadAsiaRegionalGraph(t *testing.T) {
	graph, err := LoadAsiaRegionalGraph()
	if err != nil {
		t.Fatalf("LoadAsiaRegionalGraph() error = %v", err)
	}
	if graph.ID != "oilnet-asia-generated" {
		t.Fatalf("expected asia graph id, got %q", graph.ID)
	}
	found := false
	for _, node := range graph.Nodes {
		if node.ID == "jpn-demand" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected asia graph to contain generated Japan demand node")
	}
}

func TestLoadPipelinesGeoJSON(t *testing.T) {
	graph, err := LoadPipelinesGeoJSON("../../../data/GEM-GOIT-Oil-NGL-Pipelines-2025-03.geojson")
	if err != nil {
		t.Fatalf("LoadPipelinesGeoJSON() error = %v", err)
	}
	if len(graph.Edges) == 0 {
		t.Fatal("expected pipeline graph to contain edges")
	}
	if graph.Edges[0].Kind != oilnet.EdgePipeline {
		t.Fatalf("expected pipeline edge kind, got %v", graph.Edges[0].Kind)
	}
	if len(graph.Edges[0].Route) < 2 {
		t.Fatalf("expected pipeline route geometry, got %d points", len(graph.Edges[0].Route))
	}
}

func TestParseExtractionWorkbook(t *testing.T) {
	rows, err := ParseExtractionWorkbook("../../../data/Global-Oil-and-Gas-Extraction-Tracker-March-2026.xlsx")
	if err != nil {
		t.Fatalf("ParseExtractionWorkbook() error = %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected extraction workbook rows")
	}
	found := false
	for _, row := range rows {
		if row.UnitID == "L100000321014" {
			found = true
			if row.Latitude == 0 || row.Longitude == 0 {
				t.Fatalf("expected parsed coordinates for known field row, got lat=%v lon=%v", row.Latitude, row.Longitude)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected known extraction workbook row L100000321014")
	}
}

func TestLoadExtractionWorkbookGraph(t *testing.T) {
	graph, err := LoadExtractionWorkbookGraph("../../../data/Global-Oil-and-Gas-Extraction-Tracker-March-2026.xlsx")
	if err != nil {
		t.Fatalf("LoadExtractionWorkbookGraph() error = %v", err)
	}
	if len(graph.Nodes) == 0 {
		t.Fatal("expected extraction overlay nodes")
	}
	foundProject := false
	foundFieldWithParent := false
	foundProjectOutline := false
	foundFieldProduction := false
	for _, node := range graph.Nodes {
		if node.Kind == oilnet.NodeProject {
			foundProject = true
			if len(node.OutlineRings) > 0 {
				foundProjectOutline = true
			}
		}
		if node.Kind == oilnet.NodeExtractionSite && node.ParentProjectID != "" {
			foundFieldWithParent = true
			if node.ProductionBPD > 0 {
				foundFieldProduction = true
			}
		}
	}
	if !foundProject {
		t.Fatal("expected project nodes in extraction overlay")
	}
	if !foundFieldWithParent {
		t.Fatal("expected extraction field nodes to link back to a project")
	}
	if !foundProjectOutline {
		t.Fatal("expected at least one project outline parsed from WKT")
	}
	if !foundFieldProduction {
		t.Fatal("expected at least one field node with production data")
	}
}

func TestPipelineCommoditiesDistinguishMixedProducts(t *testing.T) {
	commodities, primary := pipelineCommodities("Oil, NGL, naphtha")
	if primary != oilnet.CommodityCrude {
		t.Fatalf("expected crude as primary commodity, got %v", primary)
	}
	if len(commodities) != 3 {
		t.Fatalf("expected three commodity buckets, got %v", commodities)
	}
	if commodities[0] != oilnet.CommodityCrude || commodities[1] != oilnet.CommodityNGL || commodities[2] != oilnet.CommodityNaphtha {
		t.Fatalf("unexpected commodity mapping: %+v", commodities)
	}
	commodities, primary = pipelineCommodities("NGL, oil products")
	if primary != oilnet.CommodityNGL {
		t.Fatalf("expected NGL as primary commodity, got %v", primary)
	}
	if len(commodities) != 2 {
		t.Fatalf("expected NGL and refined products buckets, got %v", commodities)
	}
	if commodities[0] != oilnet.CommodityNGL || commodities[1] != oilnet.CommodityRefinedProducts {
		t.Fatalf("unexpected mixed-product mapping: %+v", commodities)
	}
}

func TestLoadPipelinesGeoJSONRejectsMissingRequiredProperties(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/bad-pipelines.geojson"
	data := `{"type":"FeatureCollection","features":[{"type":"Feature","properties":{"Fuel":"Oil","Countries":"USA"},"geometry":{"type":"LineString","coordinates":[[1,2],[3,4]]}}]}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := LoadPipelinesGeoJSON(path)
	if err == nil || !strings.Contains(err.Error(), "missing required properties") {
		t.Fatalf("expected missing properties error, got %v", err)
	}
}

func TestParseExtractionWorkbookRejectsMissingRequiredSheets(t *testing.T) {
	path := writeMinimalWorkbook(t, map[string]string{
		"Field-level main data": "Unit ID,Unit Name,Fuel type,Country/Area,Status,Latitude,Longitude\nu1,Field,Oil,USA,Operating,1,2\n",
	})
	_, err := ParseExtractionWorkbook(path)
	if err == nil || !strings.Contains(err.Error(), "missing required sheets") {
		t.Fatalf("expected missing sheets error, got %v", err)
	}
}

func TestParseExtractionWorkbookRejectsMissingRequiredColumns(t *testing.T) {
	path := writeMinimalWorkbook(t, map[string]string{
		"Field-level main data":           "Unit ID,Unit Name,Country/Area,Status,Latitude,Longitude\nu1,Field,USA,Operating,1,2\n",
		"Field-level production data":     "Unit ID,Fuel description,Quantity (converted),Units (converted),Data Year\nu1,Oil,1000,bpd,2024\n",
		"Field-level reserves data":       "Unit ID,Fuel description,Quantity (converted),Units (converted),Data Year\nu1,Oil,1000,bbl,2024\n",
		"Project-level main data":         "Project ID,Project Name,Fuel type,Country/Area,Status\np1,Project,Oil,USA,Operating\n",
		"Project-level production data":   "Project ID,Fuel description,Quantity (converted),Units (converted),Data Year\np1,Oil,1000,bpd,2024\n",
		"Project-level reserves data":     "Project ID,Fuel description,Quantity (converted),Units (converted),Data Year\np1,Oil,1000,bbl,2024\n",
	})
	_, err := ParseExtractionWorkbook(path)
	if err == nil || !strings.Contains(err.Error(), "missing required columns") {
		t.Fatalf("expected missing columns error, got %v", err)
	}
}

func writeMinimalWorkbook(t *testing.T, sheets map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	writeZipFile := func(name, contents string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create(%q) error = %v", name, err)
		}
		if _, err := w.Write([]byte(contents)); err != nil {
			t.Fatalf("Write(%q) error = %v", name, err)
		}
	}

	writeZipFile("[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/_rels/workbook.xml.rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
</Types>`)

	names := make([]string, 0, len(sheets))
	for name := range sheets {
		names = append(names, name)
	}
	sort.Strings(names)

	var workbook strings.Builder
	workbook.WriteString(`<?xml version="1.0" encoding="UTF-8"?><workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	var rels strings.Builder
	rels.WriteString(`<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	index := 1
	for _, name := range names {
		rid := "rId" + strconv.Itoa(index)
		sheetFile := "worksheets/sheet" + strconv.Itoa(index) + ".xml"
		workbook.WriteString(`<sheet name="` + name + `" sheetId="` + strconv.Itoa(index) + `" r:id="` + rid + `"/>`)
		rels.WriteString(`<Relationship Id="` + rid + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="` + sheetFile + `"/>`)
		index++
	}
	workbook.WriteString(`</sheets></workbook>`)
	rels.WriteString(`</Relationships>`)
	writeZipFile("xl/workbook.xml", workbook.String())
	writeZipFile("xl/_rels/workbook.xml.rels", rels.String())

	index = 1
	for _, name := range names {
		writeZipFile("xl/worksheets/sheet"+strconv.Itoa(index)+".xml", worksheetXMLFromCSV(sheets[name]))
		index++
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("zip.Close() error = %v", err)
	}

	path := t.TempDir() + "/minimal.xlsx"
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func worksheetXMLFromCSV(csv string) string {
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet><sheetData>`)
	for rowIndex, line := range lines {
		b.WriteString(`<row r="` + strconv.Itoa(rowIndex+1) + `">`)
		cols := strings.Split(line, ",")
		for colIndex, col := range cols {
			cellRef := string(rune('A'+colIndex)) + strconv.Itoa(rowIndex+1)
			b.WriteString(`<c r="` + cellRef + `" t="inlineStr"><is><t>`)
			b.WriteString(col)
			b.WriteString(`</t></is></c>`)
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}
