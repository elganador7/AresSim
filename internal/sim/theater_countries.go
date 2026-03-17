package sim

import "strings"

type theaterCountryPolygon struct {
	iso3 string
	ring [][2]float64
}

var theaterCountryPolygons = []theaterCountryPolygon{
	{iso3: "ISR", ring: [][2]float64{{34.25, 29.50}, {35.72, 29.50}, {35.88, 31.20}, {35.70, 32.10}, {35.55, 33.28}, {34.95, 33.30}, {34.25, 31.10}, {34.25, 29.50}}},
	{iso3: "LBN", ring: [][2]float64{{35.10, 33.05}, {36.62, 33.05}, {36.57, 34.69}, {35.10, 34.69}, {35.10, 33.05}}},
	{iso3: "JOR", ring: [][2]float64{{34.95, 29.10}, {39.35, 29.10}, {39.30, 33.40}, {35.45, 33.40}, {35.05, 32.30}, {34.95, 30.90}, {34.95, 29.10}}},
	{iso3: "SYR", ring: [][2]float64{{35.70, 32.35}, {42.35, 32.35}, {42.35, 37.28}, {36.00, 37.28}, {35.70, 35.90}, {35.70, 32.35}}},
	{iso3: "IRQ", ring: [][2]float64{{38.80, 29.00}, {48.60, 29.00}, {48.60, 37.40}, {42.35, 37.40}, {38.80, 33.10}, {38.80, 29.00}}},
	{iso3: "IRN", ring: [][2]float64{{44.00, 25.00}, {50.50, 25.00}, {56.90, 25.20}, {61.30, 25.10}, {63.30, 27.20}, {63.40, 31.00}, {61.80, 36.00}, {60.20, 37.80}, {55.60, 39.80}, {47.00, 39.20}, {44.20, 37.60}, {44.00, 30.50}, {44.00, 25.00}}},
	{iso3: "KWT", ring: [][2]float64{{46.55, 28.50}, {48.48, 28.50}, {48.48, 30.10}, {46.55, 30.10}, {46.55, 28.50}}},
	{iso3: "SAU", ring: [][2]float64{{34.60, 16.20}, {36.80, 31.90}, {42.00, 32.20}, {50.30, 31.80}, {55.80, 26.00}, {55.40, 20.50}, {51.80, 16.50}, {42.00, 16.20}, {34.60, 16.20}}},
	{iso3: "BHR", ring: [][2]float64{{50.28, 25.50}, {50.82, 25.50}, {50.82, 26.42}, {50.28, 26.42}, {50.28, 25.50}}},
	{iso3: "QAT", ring: [][2]float64{{50.70, 24.45}, {51.75, 24.45}, {51.75, 26.20}, {50.70, 26.20}, {50.70, 24.45}}},
	{iso3: "ARE", ring: [][2]float64{{51.45, 22.55}, {56.45, 22.55}, {56.45, 26.10}, {54.90, 26.10}, {53.00, 25.80}, {51.45, 24.20}, {51.45, 22.55}}},
	{iso3: "OMN", ring: [][2]float64{{52.00, 16.60}, {59.95, 16.60}, {59.95, 26.40}, {56.50, 26.40}, {55.10, 24.90}, {53.20, 24.20}, {52.00, 21.80}, {52.00, 16.60}}},
	{iso3: "EGY", ring: [][2]float64{{24.70, 22.00}, {36.95, 22.00}, {36.95, 31.70}, {32.20, 31.70}, {34.90, 29.10}, {34.20, 28.70}, {32.00, 29.70}, {24.70, 31.70}, {24.70, 22.00}}},
	{iso3: "TUR", ring: [][2]float64{{26.00, 35.80}, {44.90, 35.80}, {44.90, 42.20}, {28.00, 42.20}, {26.00, 40.50}, {26.00, 35.80}}},
}

func CountryCodeForPoint(lat, lon float64) string {
	for _, poly := range theaterCountryPolygons {
		if pointInRing(lon, lat, poly.ring) {
			return poly.iso3
		}
	}
	return ""
}

func CountriesAlongSegment(startLat, startLon, endLat, endLon float64) []string {
	steps := 8
	if d := maxFloat(absFloat(endLat-startLat), absFloat(endLon-startLon)); d > 0 {
		if n := int(d / 0.2); n > steps {
			steps = n
		}
	}
	if steps > 96 {
		steps = 96
	}
	seen := make(map[string]bool)
	var result []string
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		lat := startLat + (endLat-startLat)*t
		lon := startLon + (endLon-startLon)*t
		code := CountryCodeForPoint(lat, lon)
		if code != "" && !seen[code] {
			seen[code] = true
			result = append(result, code)
		}
	}
	return result
}

func CountryDisplayCode(code string) string {
	return strings.TrimSpace(strings.ToUpper(code))
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
