package geo

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type RegionType string

const (
	RegionTypeNationalAirspace      RegionType = "national"
	RegionTypeInternationalAirspace RegionType = "international"
)

type SeaZoneType string

const (
	SeaZoneTypeNone           SeaZoneType = ""
	SeaZoneTypeTerritorialSea SeaZoneType = "territorial_sea"
	SeaZoneTypeHighSeas       SeaZoneType = "high_seas"
)

type Point struct {
	Lat    float64
	Lon    float64
	AltMsl float64
}

type GeoContext struct {
	AirspaceOwner           string
	AirspaceType            RegionType
	SeaZoneOwner            string
	SeaZoneType             SeaZoneType
	IsInternationalWaters   bool
	IsInternationalAirspace bool
}

type GeoSegmentContext struct {
	Start              Point
	End                Point
	AirspaceOwners     []string
	SeaZoneOwners      []string
	SeaZoneTypes       []SeaZoneType
	ContainsIntlAir    bool
	ContainsIntlWaters bool
}

type polygonBounds struct {
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

type polygonShape struct {
	Exterior [][2]float64
	Holes    [][][2]float64
	Bounds   polygonBounds
}

type airspacePolygon struct {
	owner string
	shape polygonShape
}

type maritimePolygon struct {
	owner    string
	zoneType SeaZoneType
	shape    polygonShape
}

//go:embed data/world_borders.json
var worldBordersJSON []byte

//go:embed data/world_12nm.json
var worldMaritimeJSON []byte

var (
	worldAirspacePolygons []airspacePolygon
	loadAirspaceOnce      sync.Once
	loadAirspaceErr       error
	worldMaritimePolygons []maritimePolygon
	loadMaritimeOnce      sync.Once
	loadMaritimeErr       error
)

type featureCollectionJSON struct {
	Features []featureJSON `json:"features"`
}

type featureJSON struct {
	Properties map[string]any `json:"properties"`
	Geometry   geometryJSON   `json:"geometry"`
}

type geometryJSON struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

func CountryCode(code string) string {
	return strings.TrimSpace(strings.ToUpper(code))
}

func worldAirspace() []airspacePolygon {
	loadAirspaceOnce.Do(func() {
		var raw featureCollectionJSON
		loadAirspaceErr = json.Unmarshal(worldBordersJSON, &raw)
		if loadAirspaceErr != nil {
			return
		}
		worldAirspacePolygons = make([]airspacePolygon, 0, len(raw.Features))
		for _, feature := range raw.Features {
			owner := CountryCode(fmt.Sprint(feature.Properties["iso3"]))
			if owner == "" {
				continue
			}
			shapes, err := parseGeometryShapes(feature.Geometry)
			if err != nil {
				loadAirspaceErr = err
				return
			}
			for _, shape := range shapes {
				worldAirspacePolygons = append(worldAirspacePolygons, airspacePolygon{
					owner: owner,
					shape: shape,
				})
			}
		}
	})
	if loadAirspaceErr != nil {
		return nil
	}
	return worldAirspacePolygons
}

func worldMaritime() []maritimePolygon {
	loadMaritimeOnce.Do(func() {
		var raw featureCollectionJSON
		loadMaritimeErr = json.Unmarshal(worldMaritimeJSON, &raw)
		if loadMaritimeErr != nil {
			return
		}
		worldMaritimePolygons = make([]maritimePolygon, 0, len(raw.Features))
		for _, feature := range raw.Features {
			owner := CountryCode(fmt.Sprint(feature.Properties["owner"]))
			if owner == "" {
				continue
			}
			zoneType := SeaZoneType(strings.TrimSpace(strings.ToLower(fmt.Sprint(feature.Properties["zoneType"]))))
			if zoneType == SeaZoneTypeNone {
				zoneType = SeaZoneTypeTerritorialSea
			}
			shapes, err := parseGeometryShapes(feature.Geometry)
			if err != nil {
				loadMaritimeErr = err
				return
			}
			for _, shape := range shapes {
				worldMaritimePolygons = append(worldMaritimePolygons, maritimePolygon{
					owner:    owner,
					zoneType: zoneType,
					shape:    shape,
				})
			}
		}
	})
	if loadMaritimeErr != nil {
		return nil
	}
	return worldMaritimePolygons
}

func LookupPoint(p Point) GeoContext {
	ctx := GeoContext{
		AirspaceType:            RegionTypeInternationalAirspace,
		SeaZoneType:             SeaZoneTypeHighSeas,
		IsInternationalWaters:   true,
		IsInternationalAirspace: true,
	}

	for _, poly := range worldAirspace() {
		if pointInPolygon(p.Lon, p.Lat, poly.shape) {
			ctx.AirspaceOwner = poly.owner
			ctx.AirspaceType = RegionTypeNationalAirspace
			ctx.IsInternationalAirspace = false
			break
		}
	}

	for _, poly := range worldMaritime() {
		if pointInPolygon(p.Lon, p.Lat, poly.shape) {
			ctx.SeaZoneOwner = poly.owner
			ctx.SeaZoneType = poly.zoneType
			ctx.IsInternationalWaters = false
			if ctx.IsInternationalAirspace && poly.zoneType == SeaZoneTypeTerritorialSea {
				ctx.AirspaceOwner = poly.owner
				ctx.AirspaceType = RegionTypeNationalAirspace
				ctx.IsInternationalAirspace = false
			}
			return ctx
		}
	}

	if !ctx.IsInternationalAirspace {
		ctx.SeaZoneType = SeaZoneTypeNone
		ctx.IsInternationalWaters = false
	}

	return ctx
}

func SamplePath(points []Point) []GeoSegmentContext {
	if len(points) < 2 {
		return nil
	}
	result := make([]GeoSegmentContext, 0, len(points)-1)
	for i := 0; i < len(points)-1; i++ {
		result = append(result, sampleSegment(points[i], points[i+1]))
	}
	return result
}

func sampleSegment(start, end Point) GeoSegmentContext {
	steps := 8
	if d := maxFloat(absFloat(end.Lat-start.Lat), absFloat(end.Lon-start.Lon)); d > 0 {
		if n := int(d / 0.2); n > steps {
			steps = n
		}
	}
	if steps > 96 {
		steps = 96
	}
	seenAir := make(map[string]bool)
	owners := make([]string, 0, 4)
	seenSea := make(map[string]bool)
	seaOwners := make([]string, 0, 4)
	seenTypes := make(map[SeaZoneType]bool)
	seaTypes := make([]SeaZoneType, 0, 2)
	hasIntlAir := false
	hasIntlWaters := false
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		pt := Point{
			Lat: start.Lat + (end.Lat-start.Lat)*t,
			Lon: start.Lon + (end.Lon-start.Lon)*t,
		}
		ctx := LookupPoint(pt)
		if ctx.IsInternationalAirspace {
			hasIntlAir = true
		} else {
			owner := CountryCode(ctx.AirspaceOwner)
			if owner != "" && !seenAir[owner] {
				seenAir[owner] = true
				owners = append(owners, owner)
			}
		}
		if ctx.IsInternationalWaters {
			hasIntlWaters = true
		} else {
			owner := CountryCode(ctx.SeaZoneOwner)
			if owner != "" && !seenSea[owner] {
				seenSea[owner] = true
				seaOwners = append(seaOwners, owner)
			}
			if ctx.SeaZoneType != SeaZoneTypeNone && !seenTypes[ctx.SeaZoneType] {
				seenTypes[ctx.SeaZoneType] = true
				seaTypes = append(seaTypes, ctx.SeaZoneType)
			}
		}
	}
	return GeoSegmentContext{
		Start:              start,
		End:                end,
		AirspaceOwners:     owners,
		SeaZoneOwners:      seaOwners,
		SeaZoneTypes:       seaTypes,
		ContainsIntlAir:    hasIntlAir,
		ContainsIntlWaters: hasIntlWaters,
	}
}

func parseGeometryShapes(geometry geometryJSON) ([]polygonShape, error) {
	switch strings.TrimSpace(geometry.Type) {
	case "Polygon":
		var coords [][][]float64
		if err := json.Unmarshal(geometry.Coordinates, &coords); err != nil {
			return nil, err
		}
		shape, ok := buildPolygonShape(coords)
		if !ok {
			return nil, nil
		}
		return []polygonShape{shape}, nil
	case "MultiPolygon":
		var coords [][][][]float64
		if err := json.Unmarshal(geometry.Coordinates, &coords); err != nil {
			return nil, err
		}
		shapes := make([]polygonShape, 0, len(coords))
		for _, polygonCoords := range coords {
			shape, ok := buildPolygonShape(polygonCoords)
			if ok {
				shapes = append(shapes, shape)
			}
		}
		return shapes, nil
	default:
		return nil, nil
	}
}

func buildPolygonShape(coords [][][]float64) (polygonShape, bool) {
	if len(coords) == 0 {
		return polygonShape{}, false
	}
	exterior := toRing(coords[0])
	if len(exterior) < 3 {
		return polygonShape{}, false
	}
	shape := polygonShape{
		Exterior: exterior,
		Holes:    make([][][2]float64, 0, maxInt(len(coords)-1, 0)),
		Bounds:   computeBounds(exterior),
	}
	for _, holeCoords := range coords[1:] {
		hole := toRing(holeCoords)
		if len(hole) >= 3 {
			shape.Holes = append(shape.Holes, hole)
		}
	}
	return shape, true
}

func toRing(coords [][]float64) [][2]float64 {
	ring := make([][2]float64, 0, len(coords))
	for _, coord := range coords {
		if len(coord) < 2 {
			continue
		}
		ring = append(ring, [2]float64{coord[0], coord[1]})
	}
	return ring
}

func computeBounds(ring [][2]float64) polygonBounds {
	bounds := polygonBounds{
		MinLon: ring[0][0],
		MinLat: ring[0][1],
		MaxLon: ring[0][0],
		MaxLat: ring[0][1],
	}
	for _, coord := range ring[1:] {
		if coord[0] < bounds.MinLon {
			bounds.MinLon = coord[0]
		}
		if coord[0] > bounds.MaxLon {
			bounds.MaxLon = coord[0]
		}
		if coord[1] < bounds.MinLat {
			bounds.MinLat = coord[1]
		}
		if coord[1] > bounds.MaxLat {
			bounds.MaxLat = coord[1]
		}
	}
	return bounds
}

func pointInPolygon(lon, lat float64, shape polygonShape) bool {
	if lon < shape.Bounds.MinLon || lon > shape.Bounds.MaxLon || lat < shape.Bounds.MinLat || lat > shape.Bounds.MaxLat {
		return false
	}
	if !pointInRing(lon, lat, shape.Exterior) {
		return false
	}
	for _, hole := range shape.Holes {
		if pointInRing(lon, lat, hole) {
			return false
		}
	}
	return true
}

func pointInRing(lon, lat float64, ring [][2]float64) bool {
	inside := false
	for i, j := 0, len(ring)-1; i < len(ring); j, i = i, i+1 {
		xi, yi := ring[i][0], ring[i][1]
		xj, yj := ring[j][0], ring[j][1]
		intersects := (yi > lat) != (yj > lat) && lon < ((xj-xi)*(lat-yi))/((yj-yi)+1e-12)+xi
		if intersects {
			inside = !inside
		}
	}
	return inside
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
