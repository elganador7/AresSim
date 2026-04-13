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

func PopulateProtoMoveOrder(order *enginev1.MoveOrder) {
	if order == nil {
		return
	}
	for _, waypoint := range order.GetWaypoints() {
		PopulateProtoWaypoint(waypoint)
	}
}

func PopulateProtoUnit(unit *enginev1.Unit) {
	if unit == nil {
		return
	}
	PopulateProtoPosition(unit.Position)
	PopulateProtoMoveOrder(unit.MoveOrder)
}

func PopulateProtoScenario(scenario *enginev1.Scenario) {
	if scenario == nil {
		return
	}
	for _, unit := range scenario.GetUnits() {
		PopulateProtoUnit(unit)
	}
}

func PopulateProtoBatchUpdate(update *enginev1.BatchUnitUpdate) {
	if update == nil {
		return
	}
	for _, delta := range update.GetDeltas() {
		if delta == nil {
			continue
		}
		PopulateProtoPosition(delta.Position)
		PopulateProtoMoveOrder(delta.MoveOrder)
	}
}

func PopulateProtoFullStateSnapshot(snapshot *enginev1.FullStateSnapshot) {
	if snapshot == nil {
		return
	}
	for _, unit := range snapshot.GetUnits() {
		PopulateProtoUnit(unit)
	}
}

func PopulateProtoUnitSpawned(event *enginev1.UnitSpawnedEvent) {
	if event == nil {
		return
	}
	PopulateProtoUnit(event.Unit)
}

func PopulateProtoMunitionUpdate(update *enginev1.MunitionUpdate) {
	if update == nil {
		return
	}
	for _, munition := range update.GetMunitions() {
		if munition == nil {
			continue
		}
		PopulateProtoPosition(munition.Position)
	}
}
