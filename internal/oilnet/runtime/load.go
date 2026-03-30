package oilruntime

import (
	"fmt"

	"github.com/aressim/internal/oilnet"
	"github.com/aressim/internal/oilnet/ingest"
)

const (
	DefaultExtractionWorkbookPath = "data/Global-Oil-and-Gas-Extraction-Tracker-March-2026.xlsx"
	DefaultPipelinesGeoJSONPath   = "data/GEM-GOIT-Oil-NGL-Pipelines-2025-03.geojson"
	DefaultRenderableCachePath    = "data/oil-renderable-cache-v2.json"
)

func LoadDefaultRealDataGraph() (*oilnet.Graph, error) {
	return LoadRealDataGraph(DefaultExtractionWorkbookPath, DefaultPipelinesGeoJSONPath)
}

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

func LoadRenderableGraph(cachePath string, extractionPath string, pipelinesPath string) (*oilnet.Graph, error) {
	if cached, err := oilnet.LoadGraphJSON(cachePath); err == nil {
		return cached, nil
	}
	graph, err := LoadRealDataGraph(extractionPath, pipelinesPath)
	if err != nil {
		return nil, err
	}
	renderable := oilnet.BuildRenderableGraph(graph)
	if err := oilnet.ValidateGraph(renderable); err != nil {
		return nil, err
	}
	return renderable, nil
}

func LoadDefaultRenderableGraph() (*oilnet.Graph, error) {
	return LoadRenderableGraph(
		DefaultRenderableCachePath,
		DefaultExtractionWorkbookPath,
		DefaultPipelinesGeoJSONPath,
	)
}
