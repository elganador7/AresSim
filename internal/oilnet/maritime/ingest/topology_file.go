package ingest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/aressim/internal/oilnet/maritime"
)

// LoadTopologyFile loads maritime topology from an external JSON file.
func LoadTopologyFile(path string) (*TopologyCarrier, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read maritime topology file %s: %w", path, err)
	}
	return decodeTopology(raw, path)
}

// LoadTopologyFileOrEmbedded prefers a real topology file and falls back to the embedded seed.
func LoadTopologyFileOrEmbedded(path string) (*TopologyCarrier, error) {
	if path != "" {
		if topology, err := LoadTopologyFile(path); err == nil {
			return topology, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	topology, err := LoadEmbeddedTopology()
	if err != nil {
		return nil, err
	}
	return toCarrier(topology), nil
}

type TopologyCarrier struct {
	Topology *maritime.Topology
	Source   string
}

func decodeTopology(raw []byte, source string) (*TopologyCarrier, error) {
	var topology maritime.Topology
	if err := json.Unmarshal(raw, &topology); err != nil {
		return nil, fmt.Errorf("decode maritime topology %s: %w", source, err)
	}
	if topology.ID == "" {
		return nil, fmt.Errorf("maritime topology %s missing id", source)
	}
	return toCarrier(&topology).withSource(source), nil
}

func toCarrier(topology *maritime.Topology) *TopologyCarrier {
	return &TopologyCarrier{Topology: topology}
}

func (c *TopologyCarrier) withSource(source string) *TopologyCarrier {
	c.Source = source
	return c
}
