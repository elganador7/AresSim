package oilnet

import (
	"github.com/aressim/internal/geo"
)

func PopulateNodeH3(graph *Graph) error {
	if graph == nil {
		return nil
	}
	for i := range graph.Nodes {
		node := &graph.Nodes[i]
		if !validNodeLatLon(node) {
			continue
		}
		cell, err := geo.H3CellForLatLon(node.Lat, node.Lon, geo.DefaultH3Resolution)
		if err != nil {
			return err
		}
		parent, err := geo.ParentH3Cell(cell, geo.AggregateH3Resolution)
		if err != nil {
			return err
		}
		node.H3Cell = string(cell)
		node.H3ParentCell = string(parent)
	}
	return nil
}

func validNodeLatLon(node *Node) bool {
	if node == nil {
		return false
	}
	return node.Lat >= -90 && node.Lat <= 90 && node.Lon >= -180 && node.Lon <= 180
}
