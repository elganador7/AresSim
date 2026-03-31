package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aressim/internal/oilnet"
	oilruntime "github.com/aressim/internal/oilnet/runtime"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "build oil renderable cache: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	extractionPath := flag.String("extraction", oilruntime.DefaultExtractionWorkbookPath, "path to extraction workbook")
	pipelinesPath := flag.String("pipelines", oilruntime.DefaultPipelinesGeoJSONPath, "path to pipeline geojson")
	outputPath := flag.String("output", oilruntime.DefaultRenderableCachePath, "output cache path")
	flag.Parse()

	graph, err := oilruntime.LoadRealDataGraph(*extractionPath, *pipelinesPath)
	if err != nil {
		return fmt.Errorf("load real-data oil graph: %w", err)
	}
	renderable := oilnet.BuildRenderableGraph(graph)
	if err := oilnet.ValidateGraph(renderable); err != nil {
		return fmt.Errorf("validate renderable graph: %w", err)
	}
	if err := oilnet.WriteGraphCacheJSON(*outputPath, renderable); err != nil {
		return err
	}
	diagnostics := oilnet.BuildGraphDiagnostics(renderable)
	fmt.Printf("wrote %s with %d nodes and %d edges\n", *outputPath, diagnostics.NodeCount, diagnostics.EdgeCount)
	return nil
}
