package main

import (
	"encoding/json"
	"fmt"

	"github.com/aressim/internal/oilnet"
	oilruntime "github.com/aressim/internal/oilnet/runtime"
)

// GetGlobalOilNetwork returns the starter global oil trade graph for the map layer.
func (a *App) GetGlobalOilNetwork() (map[string]any, error) {
	graph, err := a.cachedGlobalOilGraph()
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		return nil, fmt.Errorf("marshal oil graph: %w", err)
	}
	out := make(map[string]any)
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal oil graph map: %w", err)
	}
	return out, nil
}

// GetRenderableOilNetwork returns a filtered, map-safe subset of the global oil graph.
func (a *App) GetRenderableOilNetwork() (map[string]any, error) {
	graph, err := a.cachedRenderableOilGraph()
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		return nil, fmt.Errorf("marshal renderable oil graph: %w", err)
	}
	out := make(map[string]any)
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal renderable oil graph map: %w", err)
	}
	return out, nil
}

// SimulateOilShock recomputes the global oil graph after outages or degradations.
func (a *App) SimulateOilShock(request map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal shock request: %w", err)
	}
	var req oilnet.ShockRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("decode shock request: %w", err)
	}
	graph, err := a.cachedGlobalOilGraph()
	if err != nil {
		return nil, err
	}
	result, err := oilnet.SimulateShock(graph, req)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal shock result: %w", err)
	}
	out := make(map[string]any)
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("unmarshal shock result map: %w", err)
	}
	return out, nil
}

// SimulateOilShockHorizon recomputes multi-day outage effects with storage drawdown.
func (a *App) SimulateOilShockHorizon(request map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal horizon request: %w", err)
	}
	var req oilnet.HorizonRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("decode horizon request: %w", err)
	}
	graph, err := a.cachedGlobalOilGraph()
	if err != nil {
		return nil, err
	}
	result, err := oilnet.SimulateShockHorizon(graph, req)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal horizon result: %w", err)
	}
	out := make(map[string]any)
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("unmarshal horizon result map: %w", err)
	}
	return out, nil
}

func loadGlobalOilGraph() (*oilnet.Graph, error) {
	return oilruntime.LoadRealDataGraph(
		"data/Global-Oil-and-Gas-Extraction-Tracker-March-2026.xlsx",
		"data/GEM-GOIT-Oil-NGL-Pipelines-2025-03.geojson",
	)
}

func loadRenderableOilGraph() (*oilnet.Graph, error) {
	const cachePath = "data/oil-renderable-cache-v2.json"
	if cached, err := oilnet.LoadGraphJSON(cachePath); err == nil {
		return cached, nil
	}
	graph, err := loadGlobalOilGraph()
	if err != nil {
		return nil, err
	}
	renderable := oilnet.BuildRenderableGraph(graph)
	if err := oilnet.ValidateGraph(renderable); err != nil {
		return nil, err
	}
	return renderable, nil
}

func (a *App) cachedGlobalOilGraph() (*oilnet.Graph, error) {
	a.oilGraphMu.RLock()
	if a.oilGraph != nil {
		graph := a.oilGraph
		a.oilGraphMu.RUnlock()
		return graph, nil
	}
	a.oilGraphMu.RUnlock()

	a.oilGraphMu.Lock()
	defer a.oilGraphMu.Unlock()
	if a.oilGraph != nil {
		return a.oilGraph, nil
	}
	graph, err := loadGlobalOilGraph()
	if err != nil {
		return nil, err
	}
	a.oilGraph = graph
	return a.oilGraph, nil
}

func (a *App) cachedRenderableOilGraph() (*oilnet.Graph, error) {
	a.oilRenderableGraphMu.RLock()
	if a.oilRenderableGraph != nil {
		graph := a.oilRenderableGraph
		a.oilRenderableGraphMu.RUnlock()
		return graph, nil
	}
	a.oilRenderableGraphMu.RUnlock()

	a.oilRenderableGraphMu.Lock()
	defer a.oilRenderableGraphMu.Unlock()
	if a.oilRenderableGraph != nil {
		return a.oilRenderableGraph, nil
	}
	graph, err := loadRenderableOilGraph()
	if err != nil {
		return nil, err
	}
	a.oilRenderableGraph = graph
	return a.oilRenderableGraph, nil
}
