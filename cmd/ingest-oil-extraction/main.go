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
	input := flag.String("input", "data/Global-Oil-and-Gas-Extraction-Tracker-March-2026.xlsx", "path to extraction workbook")
	output := flag.String("output", "", "optional output JSON path")
	flag.Parse()

	graph, err := ingest.LoadExtractionWorkbookGraph(*input)
	if err != nil {
		log.Fatalf("load extraction workbook: %v", err)
	}

	encoded, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		log.Fatalf("marshal extraction graph: %v", err)
	}

	if *output == "" {
		fmt.Println(string(encoded))
		return
	}
	if err := os.WriteFile(*output, encoded, 0o644); err != nil {
		log.Fatalf("write output: %v", err)
	}
}
