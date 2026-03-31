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
		fmt.Fprintf(os.Stderr, "ingest oil extraction: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	input := flag.String("input", "data/Global-Oil-and-Gas-Extraction-Tracker-March-2026.xlsx", "path to extraction workbook")
	output := flag.String("output", "", "optional output JSON path")
	flag.Parse()

	graph, err := ingest.LoadExtractionWorkbookGraph(*input)
	if err != nil {
		return fmt.Errorf("load extraction workbook: %w", err)
	}

	encoded, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal extraction graph: %w", err)
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
