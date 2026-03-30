package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/aressim/internal/oilnet"
	oilruntime "github.com/aressim/internal/oilnet/runtime"
)

func main() {
	extractionPath := flag.String("extraction", "data/Global-Oil-and-Gas-Extraction-Tracker-March-2026.xlsx", "path to extraction workbook")
	pipelinesPath := flag.String("pipelines", "data/GEM-GOIT-Oil-NGL-Pipelines-2025-03.geojson", "path to pipeline geojson")
	outputPath := flag.String("output", "data/oil-renderable-cache-v2.json", "output cache path")
	flag.Parse()

	graph, err := oilruntime.LoadRealDataGraph(*extractionPath, *pipelinesPath)
	if err != nil {
		panic(fmt.Errorf("load real-data oil graph: %w", err))
	}
	renderable := oilnet.BuildRenderableGraph(graph)
	if err := oilnet.ValidateGraph(renderable); err != nil {
		panic(fmt.Errorf("validate renderable graph: %w", err))
	}
	raw, err := json.MarshalIndent(renderable, "", "  ")
	if err != nil {
		panic(fmt.Errorf("marshal renderable graph: %w", err))
	}
	if err := os.WriteFile(*outputPath, raw, 0o644); err != nil {
		panic(fmt.Errorf("write renderable cache: %w", err))
	}
	fmt.Printf("wrote %s with %d nodes and %d edges\n", *outputPath, len(renderable.Nodes), len(renderable.Edges))
}
