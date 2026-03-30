package oilnet

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/global_baseline.json
var globalBaselineJSON []byte

// LoadGlobalBaseline returns the starter global oil-network graph.
func LoadGlobalBaseline() (*Graph, error) {
	var graph Graph
	if err := json.Unmarshal(globalBaselineJSON, &graph); err != nil {
		return nil, fmt.Errorf("decode global baseline oil graph: %w", err)
	}
	if err := ValidateGraph(&graph); err != nil {
		return nil, err
	}
	return &graph, nil
}
