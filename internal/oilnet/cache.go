package oilnet

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

const RenderableCacheSchemaVersion = "oil-renderable-cache/v1"

type CacheMetadata struct {
	SchemaVersion string `json:"schemaVersion"`
	BuiltAt       string `json:"builtAt"`
	GraphID       string `json:"graphId"`
	GraphVersion  string `json:"graphVersion"`
}

type GraphDiagnostics struct {
	NodeCount       int            `json:"nodeCount"`
	EdgeCount       int            `json:"edgeCount"`
	NodeKinds       map[string]int `json:"nodeKinds"`
	EdgeKinds       map[string]int `json:"edgeKinds"`
	NodeStates      map[string]int `json:"nodeStates"`
	EdgeStates      map[string]int `json:"edgeStates"`
	PrimaryProducts map[string]int `json:"primaryProducts"`
}

type GraphCacheFile struct {
	Metadata    CacheMetadata    `json:"metadata"`
	Diagnostics GraphDiagnostics `json:"diagnostics"`
	Graph       *Graph           `json:"graph"`
}

func BuildGraphDiagnostics(graph *Graph) GraphDiagnostics {
	diagnostics := GraphDiagnostics{
		NodeKinds:       map[string]int{},
		EdgeKinds:       map[string]int{},
		NodeStates:      map[string]int{},
		EdgeStates:      map[string]int{},
		PrimaryProducts: map[string]int{},
	}
	if graph == nil {
		return diagnostics
	}
	diagnostics.NodeCount = len(graph.Nodes)
	diagnostics.EdgeCount = len(graph.Edges)
	for _, node := range graph.Nodes {
		diagnostics.NodeKinds[string(node.Kind)]++
		diagnostics.NodeStates[string(node.State)]++
		if node.PrimaryCommodity != "" {
			diagnostics.PrimaryProducts[string(node.PrimaryCommodity)]++
		}
	}
	for _, edge := range graph.Edges {
		diagnostics.EdgeKinds[string(edge.Kind)]++
		diagnostics.EdgeStates[string(edge.State)]++
		if edge.Commodity != "" {
			diagnostics.PrimaryProducts[string(edge.Commodity)]++
		}
	}
	return diagnostics
}

func NewGraphCacheFile(graph *Graph) *GraphCacheFile {
	if graph == nil {
		return &GraphCacheFile{
			Metadata: CacheMetadata{
				SchemaVersion: RenderableCacheSchemaVersion,
				BuiltAt:       time.Now().UTC().Format(time.RFC3339),
			},
		}
	}
	return &GraphCacheFile{
		Metadata: CacheMetadata{
			SchemaVersion: RenderableCacheSchemaVersion,
			BuiltAt:       time.Now().UTC().Format(time.RFC3339),
			GraphID:       graph.ID,
			GraphVersion:  graph.Version,
		},
		Diagnostics: BuildGraphDiagnostics(graph),
		Graph:       graph,
	}
}

func LoadGraphJSON(path string) (*Graph, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapped GraphCacheFile
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Graph != nil && wrapped.Metadata.SchemaVersion != "" {
		return wrapped.Graph, nil
	}

	var graph Graph
	if err := json.Unmarshal(raw, &graph); err != nil {
		return nil, fmt.Errorf("decode oil graph json %s: %w", path, err)
	}
	return &graph, nil
}

func WriteGraphCacheJSON(path string, graph *Graph) error {
	payload := NewGraphCacheFile(graph)
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal oil graph cache: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write oil graph cache %s: %w", path, err)
	}
	return nil
}

func SortedDiagnosticKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
