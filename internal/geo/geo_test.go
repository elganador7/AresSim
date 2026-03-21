package geo

import "testing"

func TestLookupPoint(t *testing.T) {
	ctx := LookupPoint(Point{Lat: 32.08, Lon: 34.78})
	if ctx.AirspaceOwner != "ISR" {
		t.Fatalf("expected ISR, got %q", ctx.AirspaceOwner)
	}
	if ctx.IsInternationalAirspace {
		t.Fatal("expected national airspace")
	}
	if ctx.IsInternationalWaters {
		t.Fatal("expected land point not to be international waters")
	}
}

func TestLookupPointGlobalLandAirspace(t *testing.T) {
	ctx := LookupPoint(Point{Lat: 50.85, Lon: 4.35})
	if ctx.AirspaceOwner != "BEL" {
		t.Fatalf("expected BEL sovereign airspace, got %q", ctx.AirspaceOwner)
	}
	if ctx.IsInternationalAirspace {
		t.Fatal("expected Belgian land point to be sovereign airspace")
	}
}

func TestLookupPointTerritorialWaters(t *testing.T) {
	ctx := LookupPoint(Point{Lat: 25.44547, Lon: 51.66943})
	if ctx.SeaZoneOwner != "QAT" {
		t.Fatalf("expected QAT territorial waters, got %q", ctx.SeaZoneOwner)
	}
	if ctx.SeaZoneType != SeaZoneTypeTerritorialSea {
		t.Fatalf("expected territorial sea, got %q", ctx.SeaZoneType)
	}
	if ctx.IsInternationalWaters {
		t.Fatal("expected territorial waters, not international waters")
	}
	if ctx.AirspaceOwner != "QAT" || ctx.IsInternationalAirspace {
		t.Fatalf("expected QAT sovereign airspace above territorial waters, got owner=%q intl=%v", ctx.AirspaceOwner, ctx.IsInternationalAirspace)
	}
}

func TestLookupPointGlobalTerritorialWaters(t *testing.T) {
	ctx := LookupPoint(Point{Lat: 51.30, Lon: 2.80})
	if ctx.SeaZoneOwner != "BEL" {
		t.Fatalf("expected BEL territorial waters, got %q", ctx.SeaZoneOwner)
	}
	if ctx.SeaZoneType != SeaZoneTypeTerritorialSea {
		t.Fatalf("expected territorial sea, got %q", ctx.SeaZoneType)
	}
	if ctx.IsInternationalWaters {
		t.Fatal("expected Belgian territorial waters, not international waters")
	}
}

func TestSamplePath(t *testing.T) {
	segments := SamplePath([]Point{
		{Lat: 32.08, Lon: 34.78},
		{Lat: 31.40, Lon: 36.00},
	})
	if len(segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segments))
	}
	foundISR := false
	foundPSX := false
	foundJOR := false
	for _, owner := range segments[0].AirspaceOwners {
		if owner == "ISR" {
			foundISR = true
		}
		if owner == "PSX" {
			foundPSX = true
		}
		if owner == "JOR" {
			foundJOR = true
		}
	}
	if !foundISR || !foundPSX || !foundJOR {
		t.Fatalf("expected path to cross ISR, PSX, and JOR, got %#v", segments[0].AirspaceOwners)
	}
}

func TestLookupPointInternationalWaters(t *testing.T) {
	ctx := LookupPoint(Point{Lat: 34.50, Lon: 19.00})
	if !ctx.IsInternationalWaters {
		t.Fatal("expected point to be in international waters")
	}
	if ctx.SeaZoneType != SeaZoneTypeHighSeas {
		t.Fatalf("expected high seas, got %q", ctx.SeaZoneType)
	}
}

func TestIsLandPoint(t *testing.T) {
	if !IsLandPoint(Point{Lat: 32.08, Lon: 34.78}) {
		t.Fatal("expected Tel Aviv point to be land")
	}
	if IsLandPoint(Point{Lat: 25.44547, Lon: 51.66943}) {
		t.Fatal("expected territorial waters point not to be land")
	}
}

func TestSegmentCrossesLand(t *testing.T) {
	if !SegmentCrossesLand(
		Point{Lat: 29.5, Lon: 47.8},
		Point{Lat: 29.5, Lon: 49.2},
	) {
		t.Fatal("expected Gulf segment across Kuwait/Iraq coast to cross land")
	}
	if SegmentCrossesLand(
		Point{Lat: 25.30, Lon: 51.80},
		Point{Lat: 25.30, Lon: 52.20},
	) {
		t.Fatal("expected open-water segment not to cross land")
	}
}

func TestBuildMaritimeRouteRejectsLandDestination(t *testing.T) {
	if route, ok := BuildMaritimeRoute(
		Point{Lat: 25.8, Lon: 51.5},
		Point{Lat: 32.08, Lon: 34.78},
	); ok || route != nil {
		t.Fatal("expected land destination to be rejected")
	}
}
