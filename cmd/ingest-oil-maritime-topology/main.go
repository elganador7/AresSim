package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/aressim/internal/oilnet"
	maritimeingest "github.com/aressim/internal/oilnet/maritime/ingest"
	oilruntime "github.com/aressim/internal/oilnet/runtime"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ingest oil maritime topology: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	inputPath := flag.String("input", oilruntime.DefaultMaritimeTopologyPath, "path to maritime topology json")
	outputPath := flag.String("output", "", "optional output path for graph json")
	flag.Parse()

	carrier, err := maritimeingest.LoadTopologyFileOrEmbedded(*inputPath)
	if err != nil {
		return fmt.Errorf("load maritime topology: %w", err)
	}
	graph, err := maritimeingest.TopologyToGraph(carrier.Topology)
	if err != nil {
		return fmt.Errorf("build maritime graph: %w", err)
	}
	if err := oilnet.ValidateGraph(graph); err != nil {
		return fmt.Errorf("validate maritime graph: %w", err)
	}
	diagnostics := oilnet.BuildGraphDiagnostics(graph)
	fmt.Printf(
		"loaded maritime topology from %s with %d nodes and %d edges\n",
		sourceLabel(carrier.Source),
		diagnostics.NodeCount,
		diagnostics.EdgeCount,
	)
	if *outputPath == "" {
		return nil
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		return fmt.Errorf("marshal maritime graph: %w", err)
	}
	if err := os.WriteFile(*outputPath, raw, 0o644); err != nil {
		return fmt.Errorf("write maritime graph %s: %w", *outputPath, err)
	}
	return nil
}

func sourceLabel(value string) string {
	if value == "" {
		return "embedded seed"
	}
	return value
}
