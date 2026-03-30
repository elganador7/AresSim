package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aressim/internal/oilnet/ingest"
)

func main() {
	input := flag.String("input", "data/GEM-GOIT-Oil-NGL-Pipelines-2025-03.geojson", "path to oil pipeline GeoJSON")
	output := flag.String("output", "", "optional output JSON path")
	flag.Parse()

	graph, err := ingest.LoadPipelinesGeoJSON(*input)
	if err != nil {
		log.Fatalf("load pipelines geojson: %v", err)
	}

	encoded, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		log.Fatalf("marshal pipeline graph: %v", err)
	}

	if *output == "" {
		fmt.Println(string(encoded))
		return
	}
	if err := os.WriteFile(*output, encoded, 0o644); err != nil {
		log.Fatalf("write output: %v", err)
	}
}
