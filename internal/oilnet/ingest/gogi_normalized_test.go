package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGOGINormalizedGraph(t *testing.T) {
	dir := t.TempDir()
	layersDir := filepath.Join(dir, "layers")
	if err := os.MkdirAll(layersDir, 0o755); err != nil {
		t.Fatalf("mkdir layers: %v", err)
	}
	write := func(name string, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(layersDir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("refineries.ndjson", `{"sourceLayer":"Refineries","category":"refinery","sourceKey":"R-1","name":"Example Refinery","country":"USA","status":"Operating Fully","commodity":"Gasoline; Diesel","capacity":"250000","operator":"Operator A","centroid":[-95.12,29.73]}`+"\n")
	write("ports.ndjson", `{"sourceLayer":"Ports","category":"port","name":"Example Port","country":"USA","status":"NA","centroid":[-90.1,29.9]}`+"\n")
	write("storage.ndjson", `{"sourceLayer":"Storage","category":"storage","name":"Example Tank Farm","country":"USA","status":"Operational","capacity":"125000","centroid":[-91.0,30.0]}`+"\n")
	write("lng.ndjson", `{"sourceLayer":"LNG","category":"lng_terminal","name":"Canaport","country":"Canada","status":"Operating Fully","commodity":"Natural Gas","capacity":"0.46","operator":"Repsol","centroid":[-65.97,45.21]}`+"\n")

	graph, err := LoadGOGINormalizedGraph(dir)
	if err != nil {
		t.Fatalf("LoadGOGINormalizedGraph failed: %v", err)
	}
	if graph == nil {
		t.Fatal("expected graph")
	}
	if len(graph.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(graph.Nodes))
	}
	var foundRefinery, foundPort, foundStorage, foundLNG bool
	for _, node := range graph.Nodes {
		switch node.Kind {
		case "refinery":
			foundRefinery = true
			if node.CrudeIntakeBPD != 250000 {
				t.Fatalf("expected refinery crude intake 250000, got %v", node.CrudeIntakeBPD)
			}
			if len(node.ProductsHandled) == 0 {
				t.Fatal("expected refinery products handled")
			}
		case "marine_terminal":
			if node.Name == "Example Port" {
				foundPort = true
			}
			if node.Name == "Canaport" {
				foundLNG = true
			}
		case "storage_hub":
			foundStorage = true
		}
	}
	if !foundRefinery || !foundPort || !foundStorage || !foundLNG {
		t.Fatalf("missing expected nodes: refinery=%v port=%v storage=%v lng=%v", foundRefinery, foundPort, foundStorage, foundLNG)
	}
}

func TestLoadGOGINormalizedGraphMissingDirIsOptional(t *testing.T) {
	graph, err := LoadGOGINormalizedGraph(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("expected missing dir to be optional, got %v", err)
	}
	if graph != nil {
		t.Fatal("expected nil graph for missing dir")
	}
}
