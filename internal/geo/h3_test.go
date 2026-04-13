package geo

import "testing"

func TestNormalizeH3Resolution(t *testing.T) {
	if got := NormalizeH3Resolution(0); got != DefaultH3Resolution {
		t.Fatalf("expected default resolution %d, got %d", DefaultH3Resolution, got)
	}
	if got := NormalizeH3Resolution(10); got != AggregateH3Resolution {
		t.Fatalf("expected low resolutions to clamp to %d, got %d", AggregateH3Resolution, got)
	}
	if got := NormalizeH3Resolution(13); got != DefaultH3Resolution {
		t.Fatalf("expected high resolutions to clamp to %d, got %d", DefaultH3Resolution, got)
	}
}

func TestPointEnsureH3CellAndParent(t *testing.T) {
	point := Point{Lat: 25.2854, Lon: 51.5310}
	if err := point.EnsureH3Cell(); err != nil {
		t.Fatalf("EnsureH3Cell returned error: %v", err)
	}
	if !point.H3Cell.IsValid() {
		t.Fatalf("expected valid h3 cell, got %q", point.H3Cell)
	}
	parent, err := ParentH3Cell(point.H3Cell, AggregateH3Resolution)
	if err != nil {
		t.Fatalf("ParentH3Cell returned error: %v", err)
	}
	if !parent.IsValid() {
		t.Fatalf("expected valid parent h3 cell, got %q", parent)
	}
	if parent == point.H3Cell {
		t.Fatalf("expected parent cell to differ from canonical cell")
	}
}

func TestPointForH3CellReturnsCentroid(t *testing.T) {
	cell, err := H3CellForLatLon(35.6895, 139.6917, DefaultH3Resolution)
	if err != nil {
		t.Fatalf("H3CellForLatLon returned error: %v", err)
	}
	point, err := PointForH3Cell(cell)
	if err != nil {
		t.Fatalf("PointForH3Cell returned error: %v", err)
	}
	if point.Lat < -90 || point.Lat > 90 || point.Lon < -180 || point.Lon > 180 {
		t.Fatalf("expected centroid to be a valid point, got %+v", point)
	}
}

func TestCanonicalH3CellUsesExistingCell(t *testing.T) {
	seed, err := H3CellForLatLon(32.0853, 34.7818, DefaultH3Resolution)
	if err != nil {
		t.Fatalf("seed cell: %v", err)
	}
	point := Point{Lat: 0, Lon: 0, H3Cell: seed}
	got, err := point.CanonicalH3Cell()
	if err != nil {
		t.Fatalf("CanonicalH3Cell returned error: %v", err)
	}
	if got != seed {
		t.Fatalf("expected canonical cell %q, got %q", seed, got)
	}
}

func TestGridDiskForRangeIncludesOriginCell(t *testing.T) {
	cell, err := H3CellForLatLon(25.2854, 51.5310, DefaultH3Resolution)
	if err != nil {
		t.Fatalf("H3CellForLatLon returned error: %v", err)
	}
	cells, err := GridDiskForRange(cell, 50_000, DefaultH3Resolution)
	if err != nil {
		t.Fatalf("GridDiskForRange returned error: %v", err)
	}
	if len(cells) == 0 {
		t.Fatal("expected non-empty grid disk")
	}
	found := false
	for _, candidate := range cells {
		if candidate == cell {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected origin cell %q in grid disk", cell)
	}
}
