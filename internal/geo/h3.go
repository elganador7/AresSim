package geo

import (
	"fmt"
	"math"

	h3 "github.com/uber/h3-go/v4"
)

const (
	// DefaultH3Resolution is the canonical storage precision for new point features.
	DefaultH3Resolution = 12
	// AggregateH3Resolution is the preferred parent resolution for coarser bucketing.
	AggregateH3Resolution = 11
)

type H3Cell string

func NormalizeH3Resolution(resolution int) int {
	switch resolution {
	case AggregateH3Resolution, DefaultH3Resolution:
		return resolution
	case 0:
		return DefaultH3Resolution
	default:
		if resolution < AggregateH3Resolution {
			return AggregateH3Resolution
		}
		if resolution > DefaultH3Resolution {
			return DefaultH3Resolution
		}
		return resolution
	}
}

func validLatLon(lat, lon float64) bool {
	return !math.IsNaN(lat) &&
		!math.IsNaN(lon) &&
		lat >= -90 && lat <= 90 &&
		lon >= -180 && lon <= 180
}

func H3CellForLatLon(lat, lon float64, resolution int) (H3Cell, error) {
	if !validLatLon(lat, lon) {
		return "", fmt.Errorf("invalid lat/lon: lat=%v lon=%v", lat, lon)
	}
	cell := h3.LatLngToCell(h3.NewLatLng(lat, lon), NormalizeH3Resolution(resolution))
	return H3Cell(cell.String()), nil
}

func H3CellForPoint(point Point, resolution int) (H3Cell, error) {
	return H3CellForLatLon(point.Lat, point.Lon, resolution)
}

func ParseH3Cell(raw string) (H3Cell, error) {
	if raw == "" {
		return "", fmt.Errorf("empty h3 cell")
	}
	cell := h3.Cell(h3.IndexFromString(raw))
	if !cell.IsValid() {
		return "", fmt.Errorf("invalid h3 cell: %s", raw)
	}
	return H3Cell(cell.String()), nil
}

func (cell H3Cell) IsZero() bool {
	return cell == ""
}

func (cell H3Cell) Parse() (h3.Cell, error) {
	parsed := h3.Cell(h3.IndexFromString(string(cell)))
	if !parsed.IsValid() {
		return 0, fmt.Errorf("invalid h3 cell: %s", cell)
	}
	return parsed, nil
}

func (cell H3Cell) IsValid() bool {
	if cell == "" {
		return false
	}
	parsed, err := cell.Parse()
	return err == nil && parsed.IsValid()
}

func ParentH3Cell(cell H3Cell, resolution int) (H3Cell, error) {
	parsed, err := cell.Parse()
	if err != nil {
		return "", err
	}
	parent := parsed.Parent(NormalizeH3Resolution(resolution))
	if !parent.IsValid() {
		return "", fmt.Errorf("could not derive parent h3 cell for %s", cell)
	}
	return H3Cell(parent.String()), nil
}

func PointForH3Cell(cell H3Cell) (Point, error) {
	parsed, err := cell.Parse()
	if err != nil {
		return Point{}, err
	}
	latLng := parsed.LatLng()
	return Point{
		Lat: latLng.Lat,
		Lon: latLng.Lng,
	}, nil
}

func (p Point) CanonicalH3Cell() (H3Cell, error) {
	if p.H3Cell != "" {
		return ParseH3Cell(string(p.H3Cell))
	}
	return H3CellForPoint(p, DefaultH3Resolution)
}

func (p Point) H3CellAtResolution(resolution int) (H3Cell, error) {
	if p.H3Cell == "" {
		return H3CellForPoint(p, resolution)
	}
	if NormalizeH3Resolution(resolution) == DefaultH3Resolution {
		return ParseH3Cell(string(p.H3Cell))
	}
	return ParentH3Cell(p.H3Cell, resolution)
}

func (p *Point) EnsureH3Cell() error {
	if p == nil || p.H3Cell != "" {
		return nil
	}
	cell, err := H3CellForPoint(*p, DefaultH3Resolution)
	if err != nil {
		return err
	}
	p.H3Cell = cell
	return nil
}
