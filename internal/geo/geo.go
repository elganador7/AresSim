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
	SeaZoneTypeNone            SeaZoneType = ""
	SeaZoneTypeTerritorialSea  SeaZoneType = "territorial_sea"
	SeaZoneTypeHighSeas        SeaZoneType = "high_seas"
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

type airspacePolygon struct {
	owner string
	ring  [][2]float64
}

type maritimePolygon struct {
	owner    string
	zoneType SeaZoneType
	ring     [][2]float64
}

//go:embed data/theater_borders.json
var theaterBordersJSON []byte

//go:embed data/theater_maritime.json
var theaterMaritimeJSON []byte

var (
	theaterAirspacePolygons []airspacePolygon
	loadAirspaceOnce        sync.Once
	loadAirspaceErr         error
	theaterMaritimePolygons []maritimePolygon
	loadMaritimeOnce        sync.Once
	loadMaritimeErr         error
)

type theaterFeatureCollectionJSON struct {
	Features []theaterFeatureJSON `json:"features"`
}

type theaterFeatureJSON struct {
	Properties map[string]any    `json:"properties"`
	Geometry   theaterGeometryJSON `json:"geometry"`
}

type theaterGeometryJSON struct {
	Type        string          `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

func CountryCode(code string) string {
	return strings.TrimSpace(strings.ToUpper(code))
}

func theaterAirspace() []airspacePolygon {
	loadAirspaceOnce.Do(func() {
		var raw theaterFeatureCollectionJSON
		loadAirspaceErr = json.Unmarshal(theaterBordersJSON, &raw)
		if loadAirspaceErr != nil {
			return
		}
		theaterAirspacePolygons = make([]airspacePolygon, 0, len(raw.Features))
		for _, feature := range raw.Features {
			if strings.TrimSpace(feature.Geometry.Type) != "Polygon" || len(feature.Geometry.Coordinates) == 0 {
				continue
			}
			owner := CountryCode(fmt.Sprint(feature.Properties["iso3"]))
			if owner == "" {
				continue
			}
			ring := make([][2]float64, 0, len(feature.Geometry.Coordinates[0]))
			for _, coord := range feature.Geometry.Coordinates[0] {
				if len(coord) < 2 {
					continue
				}
				ring = append(ring, [2]float64{coord[0], coord[1]})
			}
			theaterAirspacePolygons = append(theaterAirspacePolygons, airspacePolygon{
				owner: owner,
				ring:  ring,
			})
		}
	})
	if loadAirspaceErr != nil {
		return nil
	}
	return theaterAirspacePolygons
}

func theaterMaritime() []maritimePolygon {
	loadMaritimeOnce.Do(func() {
		var raw theaterFeatureCollectionJSON
		loadMaritimeErr = json.Unmarshal(theaterMaritimeJSON, &raw)
		if loadMaritimeErr != nil {
			return
		}
		theaterMaritimePolygons = make([]maritimePolygon, 0, len(raw.Features))
		for _, feature := range raw.Features {
			if strings.TrimSpace(feature.Geometry.Type) != "Polygon" || len(feature.Geometry.Coordinates) == 0 {
				continue
			}
			owner := CountryCode(fmt.Sprint(feature.Properties["owner"]))
			if owner == "" {
				continue
			}
			ring := make([][2]float64, 0, len(feature.Geometry.Coordinates[0]))
			for _, coord := range feature.Geometry.Coordinates[0] {
				if len(coord) < 2 {
					continue
				}
				ring = append(ring, [2]float64{coord[0], coord[1]})
			}
			theaterMaritimePolygons = append(theaterMaritimePolygons, maritimePolygon{
				owner:    owner,
				zoneType: SeaZoneType(strings.TrimSpace(strings.ToLower(fmt.Sprint(feature.Properties["zoneType"])))),
				ring:     ring,
			})
		}
	})
	if loadMaritimeErr != nil {
		return nil
	}
	return theaterMaritimePolygons
}

func LookupPoint(p Point) GeoContext {
	ctx := GeoContext{
		AirspaceType:            RegionTypeInternationalAirspace,
		SeaZoneType:             SeaZoneTypeHighSeas,
		IsInternationalWaters:   true,
		IsInternationalAirspace: true,
	}
	for _, poly := range theaterAirspace() {
		if pointInRing(p.Lon, p.Lat, poly.ring) {
			ctx.AirspaceOwner = poly.owner
			ctx.AirspaceType = RegionTypeNationalAirspace
			ctx.IsInternationalAirspace = false
			break
		}
	}
	for _, poly := range theaterMaritime() {
		if pointInRing(p.Lon, p.Lat, poly.ring) {
			ctx.SeaZoneOwner = poly.owner
			ctx.SeaZoneType = poly.zoneType
			ctx.IsInternationalWaters = false
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
