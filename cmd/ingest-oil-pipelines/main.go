package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/aressim/internal/oilnet/ingest"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ingest oil pipelines: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	input := flag.String("input", "data/GEM-GOIT-Oil-NGL-Pipelines-2025-03.geojson", "path to oil pipeline GeoJSON")
	output := flag.String("output", "", "optional output JSON path")
	flag.Parse()

	graph, err := ingest.LoadPipelinesGeoJSON(*input)
	if err != nil {
		return fmt.Errorf("load pipelines geojson: %w", err)
	}

	encoded, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pipeline graph: %w", err)
	}

	if *output == "" {
		fmt.Println(string(encoded))
		return nil
	}
	if err := os.WriteFile(*output, encoded, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
