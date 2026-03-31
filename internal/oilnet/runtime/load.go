package oilruntime

import (
	"fmt"

	"github.com/aressim/internal/oilnet"
	"github.com/aressim/internal/oilnet/ingest"
	maritimeingest "github.com/aressim/internal/oilnet/maritime/ingest"
)

const (
	DefaultExtractionWorkbookPath = "data/Global-Oil-and-Gas-Extraction-Tracker-March-2026.xlsx"
	DefaultPipelinesGeoJSONPath   = "data/GEM-GOIT-Oil-NGL-Pipelines-2025-03.geojson"
	DefaultMaritimeTopologyPath   = "data/oil-maritime-topology.json"
	DefaultGOGINormalizedPath     = "data/gogi-normalized-v1"
	DefaultRenderableCachePath    = "data/oil-renderable-cache-v2.json"
)

func LoadDefaultRealDataGraph() (*oilnet.Graph, error) {
	return LoadRealDataGraph(DefaultExtractionWorkbookPath, DefaultPipelinesGeoJSONPath, DefaultMaritimeTopologyPath, DefaultGOGINormalizedPath)
}

func LoadRealDataGraph(extractionPath string, pipelinesPath string, maritimePath string, gogiPath string) (*oilnet.Graph, error) {
	extraction, err := ingest.LoadExtractionWorkbookGraph(extractionPath)
	if err != nil {
		return nil, fmt.Errorf("load extraction workbook: %w", err)
	}
	pipelines, err := ingest.LoadPipelinesGeoJSON(pipelinesPath)
	if err != nil {
		return nil, fmt.Errorf("load pipeline geojson: %w", err)
	}
	maritimeCarrier, err := maritimeingest.LoadTopologyFileOrEmbedded(maritimePath)
	if err != nil {
		return nil, fmt.Errorf("load maritime topology: %w", err)
	}
	maritimeGraph, err := maritimeingest.TopologyToGraph(maritimeCarrier.Topology)
	if err != nil {
		return nil, fmt.Errorf("build maritime topology graph: %w", err)
	}
	gogiGraph, err := ingest.LoadGOGINormalizedGraph(gogiPath)
	if err != nil {
		return nil, fmt.Errorf("load gogi normalized overlay: %w", err)
	}
	merged := oilnet.MergeGraphs(extraction, pipelines)
	merged = oilnet.MergeGraphs(merged, maritimeGraph)
	merged = oilnet.ReconcileGOGIOverlay(merged, gogiGraph)
	merged.ID = "global-oil-network-realdata"
	merged.Name = "Global Oil Network Real Data"
	merged.Description = "Global oil network built from the provided extraction workbook, pipeline GeoJSON, seeded maritime petroleum topology, and optional GOGI infrastructure overlay."
	merged.View = "global"
	if err := oilnet.ValidateGraph(merged); err != nil {
		return nil, err
	}
	return merged, nil
}

func LoadRenderableGraph(cachePath string, extractionPath string, pipelinesPath string, maritimePath string, gogiPath string) (*oilnet.Graph, error) {
	if cached, err := oilnet.LoadRenderableCacheJSON(cachePath); err == nil {
		return cached, nil
	}
	graph, err := LoadRealDataGraph(extractionPath, pipelinesPath, maritimePath, gogiPath)
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
		DefaultMaritimeTopologyPath,
		DefaultGOGINormalizedPath,
	)
}
