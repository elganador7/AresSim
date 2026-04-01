package geo

import enginev1 "github.com/aressim/internal/gen/engine/v1"

func ProtoPosition(lat, lon, altMsl, heading, speed float64) *enginev1.Position {
	point := Point{Lat: lat, Lon: lon, AltMsl: altMsl}
	_ = point.EnsureH3Cell()
	parent, _ := point.H3CellAtResolution(AggregateH3Resolution)
	return &enginev1.Position{
		Lat:          lat,
		Lon:          lon,
		AltMsl:       altMsl,
		Heading:      heading,
		Speed:        speed,
		H3Cell:       string(point.H3Cell),
		H3ParentCell: string(parent),
	}
}

func ProtoWaypoint(lat, lon, altMsl float64) *enginev1.Waypoint {
	point := Point{Lat: lat, Lon: lon, AltMsl: altMsl}
	_ = point.EnsureH3Cell()
	parent, _ := point.H3CellAtResolution(AggregateH3Resolution)
	return &enginev1.Waypoint{
		Lat:          lat,
		Lon:          lon,
		AltMsl:       altMsl,
		H3Cell:       string(point.H3Cell),
		H3ParentCell: string(parent),
	}
}

func PopulateProtoPosition(position *enginev1.Position) {
	if position == nil {
		return
	}
	if position.H3Cell != "" && position.H3ParentCell != "" {
		return
	}
	point := Point{Lat: position.Lat, Lon: position.Lon, AltMsl: position.AltMsl, H3Cell: H3Cell(position.H3Cell)}
	if point.H3Cell == "" {
		_ = point.EnsureH3Cell()
	}
	parent, _ := point.H3CellAtResolution(AggregateH3Resolution)
	position.H3Cell = string(point.H3Cell)
	position.H3ParentCell = string(parent)
}

func PopulateProtoWaypoint(waypoint *enginev1.Waypoint) {
	if waypoint == nil {
		return
	}
	if waypoint.H3Cell != "" && waypoint.H3ParentCell != "" {
		return
	}
	point := Point{Lat: waypoint.Lat, Lon: waypoint.Lon, AltMsl: waypoint.AltMsl, H3Cell: H3Cell(waypoint.H3Cell)}
	if point.H3Cell == "" {
		_ = point.EnsureH3Cell()
	}
	parent, _ := point.H3CellAtResolution(AggregateH3Resolution)
	waypoint.H3Cell = string(point.H3Cell)
	waypoint.H3ParentCell = string(parent)
}
