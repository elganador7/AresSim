package geo

import "testing"

func TestLookupPoint(t *testing.T) {
	ctx := LookupPoint(Point{Lat: 31.5, Lon: 35.0})
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

func TestLookupPointTerritorialWaters(t *testing.T) {
	ctx := LookupPoint(Point{Lat: 25.70, Lon: 51.30})
	if ctx.SeaZoneOwner != "QAT" {
		t.Fatalf("expected QAT territorial waters, got %q", ctx.SeaZoneOwner)
	}
	if ctx.SeaZoneType != SeaZoneTypeTerritorialSea {
		t.Fatalf("expected territorial sea, got %q", ctx.SeaZoneType)
	}
	if ctx.IsInternationalWaters {
		t.Fatal("expected territorial waters, not international waters")
	}
}

func TestSamplePath(t *testing.T) {
	segments := SamplePath([]Point{
		{Lat: 31.50, Lon: 35.00},
		{Lat: 31.40, Lon: 36.00},
	})
	if len(segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segments))
	}
	foundISR := false
	foundJOR := false
	for _, owner := range segments[0].AirspaceOwners {
		if owner == "ISR" {
			foundISR = true
		}
		if owner == "JOR" {
			foundJOR = true
		}
	}
	if !foundISR || !foundJOR {
		t.Fatalf("expected path to cross ISR and JOR, got %#v", segments[0].AirspaceOwners)
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
