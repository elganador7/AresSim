package oilruntime

import (
	"fmt"

	"github.com/aressim/internal/oilnet"
	"github.com/aressim/internal/oilnet/ingest"
)

func LoadRealDataGraph(extractionPath string, pipelinesPath string) (*oilnet.Graph, error) {
	extraction, err := ingest.LoadExtractionWorkbookGraph(extractionPath)
	if err != nil {
		return nil, fmt.Errorf("load extraction workbook: %w", err)
	}
	pipelines, err := ingest.LoadPipelinesGeoJSON(pipelinesPath)
	if err != nil {
		return nil, fmt.Errorf("load pipeline geojson: %w", err)
	}
	merged := oilnet.MergeGraphs(extraction, pipelines)
	merged.ID = "global-oil-network-realdata"
	merged.Name = "Global Oil Network Real Data"
	merged.Description = "Global oil network built only from the provided extraction workbook and pipeline GeoJSON."
	merged.View = "global"
	if err := oilnet.ValidateGraph(merged); err != nil {
		return nil, err
	}
	return merged, nil
}
