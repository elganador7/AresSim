package ingest

import (
	"math"
	"strings"

	"github.com/aressim/internal/oilnet"
)

type chokepointDef struct {
	ID      string
	Name    string
	Country string
	Lat     float64
	Lon     float64
}

var standardChokepoints = []chokepointDef{
	{ID: "om-hormuz", Name: "Strait of Hormuz", Country: "OMN", Lat: 26.566, Lon: 56.25},
	{ID: "ye-bab-el-mandeb", Name: "Bab el-Mandeb", Country: "YEM", Lat: 12.585, Lon: 43.33},
	{ID: "eg-suez", Name: "Suez Chokepoint", Country: "EGY", Lat: 29.966, Lon: 32.55},
	{ID: "sg-malacca", Name: "Strait of Malacca", Country: "SGP", Lat: 1.264, Lon: 103.83},
}

func ensureStandardChokepoints(nodeByID map[string]oilnet.Node) {
	for _, def := range standardChokepoints {
		if _, ok := nodeByID[def.ID]; ok {
			continue
		}
		nodeByID[def.ID] = oilnet.Node{
			ID:               def.ID,
			Name:             def.Name,
			Kind:             oilnet.NodeChokepoint,
			CountryCode:      def.Country,
			Lat:              def.Lat,
			Lon:              def.Lon,
			State:            oilnet.StateOperational,
			PrimaryCommodity: oilnet.CommodityCrude,
		}
	}
}

func inferShippingPath(from, to oilnet.Node) ([]oilnet.RoutePoint, []string, float64, float64) {
	chokepoints := make([]string, 0, 3)
	route := []oilnet.RoutePoint{{Lat: from.Lat, Lon: from.Lon}}

	if requiresHormuz(from, to) {
		chokepoints = append(chokepoints, "om-hormuz")
		route = append(route, routePointForChokepoint("om-hormuz"))
	}
	if requiresBabElMandeb(from, to) {
		chokepoints = append(chokepoints, "ye-bab-el-mandeb")
		route = append(route, routePointForChokepoint("ye-bab-el-mandeb"))
	}
	if requiresSuez(from, to) {
		chokepoints = append(chokepoints, "eg-suez")
		route = append(route, routePointForChokepoint("eg-suez"))
	}
	if requiresMalacca(from, to) {
		chokepoints = append(chokepoints, "sg-malacca")
		route = append(route, routePointForChokepoint("sg-malacca"))
	}
	route = append(route, oilnet.RoutePoint{Lat: to.Lat, Lon: to.Lon})

	lengthKM := 0.0
	for i := 0; i < len(route)-1; i++ {
		lengthKM += haversineKM(route[i], route[i+1])
	}
	// Coarse tanker transit estimate at ~550 km/day including canal/chokepoint drag.
	transitDays := 0.0
	if lengthKM > 0 {
		transitDays = lengthKM / 550.0
	}
	return route, dedupeStrings(chokepoints), transitDays, lengthKM
}

func routePointForChokepoint(id string) oilnet.RoutePoint {
	for _, def := range standardChokepoints {
		if def.ID == id {
			return oilnet.RoutePoint{Lat: def.Lat, Lon: def.Lon}
		}
	}
	return oilnet.RoutePoint{}
}

func requiresHormuz(from, to oilnet.Node) bool {
	if from.Lon >= 56 {
		return false
	}
	return isGulfExporter(from.CountryCode) && !sameBasinDestination(to.CountryCode, "gulf")
}

func requiresBabElMandeb(from, to oilnet.Node) bool {
	return requiresSuez(from, to)
}

func requiresSuez(from, to oilnet.Node) bool {
	fromRegion := shippingRegion(from.CountryCode)
	toRegion := shippingRegion(to.CountryCode)
	return (fromRegion == "gulf" || fromRegion == "asia") && toRegion == "europe"
}

func requiresMalacca(from, to oilnet.Node) bool {
	fromRegion := shippingRegion(from.CountryCode)
	toRegion := shippingRegion(to.CountryCode)
	return (fromRegion == "gulf" || fromRegion == "europe") && toRegion == "east_asia"
}

func sameBasinDestination(country, basin string) bool {
	return shippingRegion(country) == basin
}

func shippingRegion(country string) string {
	switch strings.TrimSpace(strings.ToUpper(country)) {
	case "SAU", "ARE", "KWT", "IRQ", "IRN", "QAT", "BHR", "OMN":
		return "gulf"
	case "NLD", "DEU", "BEL", "FRA", "ITA", "ESP", "GBR", "NOR":
		return "europe"
	case "JPN", "KOR", "CHN", "TWN", "SGP":
		return "east_asia"
	case "IND":
		return "asia"
	case "USA", "CAN", "MEX", "BRA":
		return "americas"
	default:
		return "other"
	}
}

func isGulfExporter(country string) bool {
	return shippingRegion(country) == "gulf"
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func haversineKM(a, b oilnet.RoutePoint) float64 {
	const earthRadiusKM = 6371.0
	lat1 := a.Lat * math.Pi / 180.0
	lat2 := b.Lat * math.Pi / 180.0
	dLat := (b.Lat - a.Lat) * math.Pi / 180.0
	dLon := (b.Lon - a.Lon) * math.Pi / 180.0
	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return 2 * earthRadiusKM * math.Asin(math.Sqrt(h))
}
