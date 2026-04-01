package main

import (
	"testing"

	"github.com/aressim/internal/geo"
)

func TestPointsToDraftInputsPopulateH3(t *testing.T) {
	points := []geo.Point{
		{Lat: 25.2854, Lon: 51.5310},
		{Lat: 26.5660, Lon: 56.2500},
	}

	got := pointsToDraftInputs(points)
	if len(got) != len(points) {
		t.Fatalf("expected %d draft points, got %d", len(points), len(got))
	}
	for i, point := range got {
		if point.H3Cell == "" {
			t.Fatalf("expected H3 cell at index %d", i)
		}
		if point.H3ParentCell == "" {
			t.Fatalf("expected H3 parent cell at index %d", i)
		}
	}
}
