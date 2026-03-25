package routing

import (
	"strings"
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/geo"
	"github.com/aressim/internal/sim"
)

func TestBuildRoute_SurfaceShipBlockedByForeignTerritorialWaters(t *testing.T) {
	t.Parallel()

	result := BuildRoute(Request{
		OwnerCountry: "USA",
		Domain:       enginev1.UnitDomain_DOMAIN_SEA,
		Purpose:      PurposeMove,
		Start:        geo.Point{Lat: 25.30, Lon: 51.80},
		End:          geo.Point{Lat: 25.445, Lon: 51.669},
	})

	if !result.Blocked {
		t.Fatalf("expected surface route to be blocked")
	}
	if !strings.Contains(result.Reason, "territorial waters") {
		t.Fatalf("expected territorial-waters reason, got %q", result.Reason)
	}
}

func TestBuildRoute_SubsurfaceCanTransitForeignWaters(t *testing.T) {
	t.Parallel()

	result := BuildRoute(Request{
		OwnerCountry: "USA",
		Domain:       enginev1.UnitDomain_DOMAIN_SUBSURFACE,
		Purpose:      PurposeMove,
		Start:        geo.Point{Lat: 25.22, Lon: 51.92},
		End:          geo.Point{Lat: 25.445, Lon: 51.669},
	})

	if result.Blocked {
		t.Fatalf("expected subsurface route to be allowed, got %q", result.Reason)
	}
	if len(result.Points) == 0 {
		t.Fatalf("expected routed points")
	}
}

func TestBuildRoute_DefensiveAirRequiresDefensivePermission(t *testing.T) {
	t.Parallel()

	rules := sim.RelationshipRules{
		"USA": {
			"IRN": {
				AirspaceTransitAllowed:      true,
				DefensivePositioningAllowed: false,
			},
		},
	}

	result := BuildRoute(Request{
		OwnerCountry:      "USA",
		Domain:            enginev1.UnitDomain_DOMAIN_AIR,
		Purpose:           PurposeDefensiveAir,
		Start:             geo.Point{Lat: 26.70, Lon: 56.00},
		End:               geo.Point{Lat: 27.20, Lon: 56.60},
		RelationshipRules: rules,
	})

	if !result.Blocked {
		t.Fatalf("expected defensive air route to be blocked")
	}
	if !strings.Contains(result.Reason, "defensive air operations") {
		t.Fatalf("expected defensive-air reason, got %q", result.Reason)
	}
}

func TestBuildRoute_AirStrikeAllowedWithPermission(t *testing.T) {
	t.Parallel()

	rules := sim.RelationshipRules{
		"USA": {
			"IRN": {
				AirspaceTransitAllowed: true,
				AirspaceStrikeAllowed:  true,
			},
		},
	}

	result := BuildRoute(Request{
		OwnerCountry:      "USA",
		Domain:            enginev1.UnitDomain_DOMAIN_AIR,
		Purpose:           PurposeStrike,
		Start:             geo.Point{Lat: 27.20, Lon: 56.60},
		End:               geo.Point{Lat: 28.95, Lon: 50.83},
		RelationshipRules: rules,
	})

	if result.Blocked {
		t.Fatalf("expected strike route to be allowed, got %q", result.Reason)
	}
	if len(result.Points) == 0 {
		t.Fatalf("expected routed points")
	}
}

func TestBuildRoute_SeaReroutesAroundQatarLandmass(t *testing.T) {
	t.Parallel()

	rules := sim.RelationshipRules{
		"USA": {
			"SAU": {MaritimeTransitAllowed: true},
			"ARE": {MaritimeTransitAllowed: true},
			"BHR": {MaritimeTransitAllowed: true},
		},
	}

	result := BuildRoute(Request{
		OwnerCountry:      "USA",
		Domain:            enginev1.UnitDomain_DOMAIN_SEA,
		Purpose:           PurposeMove,
		Start:             geo.Point{Lat: 24.90, Lon: 50.30},
		End:               geo.Point{Lat: 24.80, Lon: 52.30},
		RelationshipRules: rules,
	})

	if result.Blocked {
		t.Fatalf("expected maritime reroute to succeed, got %q", result.Reason)
	}
	if len(result.Points) < 2 {
		t.Fatalf("expected rerouted maritime path with intermediate points, got %+v", result.Points)
	}
}

func TestBuildRoute_AirMoveReroutesAroundDeniedQatariAirspace(t *testing.T) {
	t.Parallel()

	rules := sim.RelationshipRules{
		"USA": {
			"SAU": {AirspaceTransitAllowed: true},
			"ARE": {AirspaceTransitAllowed: true},
			"BHR": {AirspaceTransitAllowed: true},
		},
	}

	result := BuildRoute(Request{
		OwnerCountry:      "USA",
		Domain:            enginev1.UnitDomain_DOMAIN_AIR,
		Purpose:           PurposeMove,
		Start:             geo.Point{Lat: 24.90, Lon: 50.30},
		End:               geo.Point{Lat: 24.80, Lon: 53.30},
		RelationshipRules: rules,
	})

	if result.Blocked {
		t.Fatalf("expected air reroute to succeed, got %q", result.Reason)
	}
	if len(result.Points) < 2 {
		t.Fatalf("expected rerouted air path with intermediate points, got %+v", result.Points)
	}
}

func TestBuildRoute_SurfaceShipCanTransitHormuzPassage(t *testing.T) {
	t.Parallel()

	result := BuildRoute(Request{
		OwnerCountry: "USA",
		Domain:       enginev1.UnitDomain_DOMAIN_SEA,
		Purpose:      PurposeMove,
		Start:        geo.Point{Lat: 25.86, Lon: 55.18},
		End:          geo.Point{Lat: 26.34, Lon: 56.82},
	})

	if result.Blocked {
		t.Fatalf("expected Hormuz transit passage to allow routing, got %q", result.Reason)
	}
	if len(result.Points) == 0 {
		t.Fatalf("expected routed points through Hormuz")
	}
}

func TestBuildRoute_SurfaceShipRerouteAvoidsLandPoints(t *testing.T) {
	t.Parallel()

	rules := sim.RelationshipRules{
		"USA": {
			"SAU": {MaritimeTransitAllowed: true},
			"ARE": {MaritimeTransitAllowed: true},
			"BHR": {MaritimeTransitAllowed: true},
		},
	}

	result := BuildRoute(Request{
		OwnerCountry:      "USA",
		Domain:            enginev1.UnitDomain_DOMAIN_SEA,
		Purpose:           PurposeMove,
		Start:             geo.Point{Lat: 24.90, Lon: 50.30},
		End:               geo.Point{Lat: 24.80, Lon: 52.30},
		RelationshipRules: rules,
	})

	if result.Blocked {
		t.Fatalf("expected maritime reroute to succeed, got %q", result.Reason)
	}
	for _, point := range result.Points {
		if geo.IsLandPoint(point) {
			t.Fatalf("expected routed maritime point %+v to remain water", point)
		}
	}
}
