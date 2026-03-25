package main

import (
	"context"
	"math"
	"strings"
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/geo"
	"github.com/aressim/internal/library"
	"github.com/aressim/internal/routing"
	scenario "github.com/aressim/internal/scenario"
	"github.com/aressim/internal/sim"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

func TestEnsureUnitOpsStateInitializesAirbaseOps(t *testing.T) {
	unit := &enginev1.Unit{Id: "ab-1"}
	def := sim.DefStats{AssetClass: "airbase"}

	ensureUnitOpsState(unit, def)

	if unit.GetBaseOps() == nil {
		t.Fatal("expected airbase to get default base ops state")
	}
	if got := unit.GetBaseOps().GetState(); got != enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE {
		t.Fatalf("expected usable base state, got %v", got)
	}
	if got := unit.GetNextSortieReadySeconds(); got != 0 {
		t.Fatalf("expected non-negative sortie ready time default, got %v", got)
	}
}

func TestEnsureUnitOpsStateInitializesCarrierOps(t *testing.T) {
	unit := &enginev1.Unit{Id: "cvn-1"}
	def := sim.DefStats{
		Domain:                      enginev1.UnitDomain_DOMAIN_SEA,
		EmbarkedFixedWingCapacity:   70,
		EmbarkedRotaryWingCapacity:  12,
		LaunchCapacityPerInterval:   20,
		RecoveryCapacityPerInterval: 20,
	}

	ensureUnitOpsState(unit, def)

	if unit.GetBaseOps() == nil {
		t.Fatal("expected carrier host platform to get default base ops state")
	}
}

func TestEnsureUnitOpsStateLeavesNonAirbaseUntouched(t *testing.T) {
	unit := &enginev1.Unit{Id: "fighter-1"}
	def := sim.DefStats{AssetClass: "combat_unit"}

	ensureUnitOpsState(unit, def)

	if unit.GetBaseOps() != nil {
		t.Fatal("did not expect non-airbase to get base ops state")
	}
}

func TestValidateAndConsumeLaunchBlocksClosedBase(t *testing.T) {
	app := &App{}
	app.setSimSeconds(0)
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{
				Id:          "base-1",
				DisplayName: "Base One",
				BaseOps: &enginev1.BaseOpsState{
					State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_CLOSED,
				},
			},
			{
				Id:           "ac-1",
				DisplayName:  "Shooter",
				DefinitionId: "fighter",
				HostBaseId:   "base-1",
				Position:     &enginev1.Position{},
			},
		},
	}

	err := app.validateAndConsumeLaunch(app.currentScenario.Units[1], sim.DefStats{
		Domain: enginev1.UnitDomain_DOMAIN_AIR,
	})
	if err == nil {
		t.Fatal("expected closed base to block launch")
	}
}

func TestValidateAndConsumeLaunchAdvancesBaseWindow(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.setSimSeconds(120)
	app.defsCache = map[string]sim.DefStats{
		"airbase": {AssetClass: "airbase", LaunchCapacityPerInterval: 3},
	}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{
				Id:           "base-1",
				DisplayName:  "Base One",
				DefinitionId: "airbase",
				BaseOps: &enginev1.BaseOpsState{
					State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE,
				},
			},
			{
				Id:           "ac-1",
				DisplayName:  "Shooter",
				DefinitionId: "fighter",
				HostBaseId:   "base-1",
				Position:     &enginev1.Position{},
			},
		},
	}

	err := app.validateAndConsumeLaunch(app.currentScenario.Units[1], sim.DefStats{
		Domain: enginev1.UnitDomain_DOMAIN_AIR,
	})
	if err != nil {
		t.Fatalf("unexpected launch validation failure: %v", err)
	}
	got := app.currentScenario.Units[0].GetBaseOps().GetNextLaunchAvailableSeconds()
	want := 420.0
	if got != want {
		t.Fatalf("expected launch window %v, got %v", want, got)
	}
	if got := app.currentScenario.Units[1].GetNextSortieReadySeconds(); got != 5520 {
		t.Fatalf("expected sortie ready time 5520, got %v", got)
	}
}

func TestValidateAndConsumeLaunchSnapsAircraftToHostBase(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.setSimSeconds(120)
	app.defsCache = map[string]sim.DefStats{
		"carrier": {
			Domain:                    enginev1.UnitDomain_DOMAIN_SEA,
			EmbarkedFixedWingCapacity: 75,
			LaunchCapacityPerInterval: 24,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{
				Id:           "carrier-1",
				DisplayName:  "Carrier One",
				DefinitionId: "carrier",
				Position:     &enginev1.Position{Lat: 24.2, Lon: 36.9},
				BaseOps: &enginev1.BaseOpsState{
					State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE,
				},
			},
			{
				Id:           "ac-1",
				DisplayName:  "Shooter",
				DefinitionId: "fighter",
				HostBaseId:   "carrier-1",
				Position:     &enginev1.Position{Lat: 0, Lon: 0},
			},
		},
	}

	err := app.validateAndConsumeLaunch(app.currentScenario.Units[1], sim.DefStats{
		Domain: enginev1.UnitDomain_DOMAIN_AIR,
	})
	if err != nil {
		t.Fatalf("unexpected launch validation failure: %v", err)
	}
	if got := app.currentScenario.Units[1].GetPosition().GetLat(); got != 24.2 {
		t.Fatalf("expected aircraft lat to snap to host base, got %v", got)
	}
	if got := app.currentScenario.Units[1].GetPosition().GetLon(); got != 36.9 {
		t.Fatalf("expected aircraft lon to snap to host base, got %v", got)
	}
	if got := app.currentScenario.Units[1].GetPosition().GetAltMsl(); got <= 0 {
		t.Fatalf("expected launched aircraft to climb above ground level, got %v", got)
	}
}

func TestValidateAndConsumeLaunch_DegradedBaseSlowsLaunchAndSortieWindows(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.setSimSeconds(300)
	app.defsCache = map[string]sim.DefStats{
		"airbase": {AssetClass: "airbase", LaunchCapacityPerInterval: 4},
	}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{
				Id:           "base-1",
				DisplayName:  "Base One",
				DefinitionId: "airbase",
				BaseOps: &enginev1.BaseOpsState{
					State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_DEGRADED,
				},
			},
			{
				Id:           "ac-1",
				DisplayName:  "Shooter",
				DefinitionId: "fighter",
				HostBaseId:   "base-1",
				Position:     &enginev1.Position{},
			},
		},
	}

	err := app.validateAndConsumeLaunch(app.currentScenario.Units[1], sim.DefStats{
		Domain:                enginev1.UnitDomain_DOMAIN_AIR,
		SortieIntervalMinutes: 60,
	})
	if err != nil {
		t.Fatalf("unexpected launch validation failure: %v", err)
	}
	if got := app.currentScenario.Units[0].GetBaseOps().GetNextLaunchAvailableSeconds(); got != 750 {
		t.Fatalf("expected degraded launch window 750, got %v", got)
	}
	if got := app.currentScenario.Units[1].GetNextSortieReadySeconds(); got != 7500 {
		t.Fatalf("expected degraded sortie ready time 7500, got %v", got)
	}
}

func TestMoveUnit_SurfaceShipBlockedByForeignTerritorialWaters(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"surface-ship": {
			Domain:         enginev1.UnitDomain_DOMAIN_SEA,
			CruiseSpeedMps: 12,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{{
			Id:           "ship-1",
			TeamId:       "USA",
			DefinitionId: "surface-ship",
			Position:     &enginev1.Position{Lat: 25.30, Lon: 51.80, Speed: 12},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.MoveUnit("ship-1", 25.44547, 51.66943)
	if result.Success {
		t.Fatal("expected surface ship move into foreign territorial waters to be blocked")
	}
	if !strings.Contains(result.Error, "territorial waters") {
		t.Fatalf("expected territorial waters error, got %q", result.Error)
	}
}

func TestMoveUnit_SurfaceShipAllowedWithMaritimeTransitPermission(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"surface-ship": {
			Domain:         enginev1.UnitDomain_DOMAIN_SEA,
			CruiseSpeedMps: 12,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{{
			FromCountry:            "USA",
			ToCountry:              "QAT",
			MaritimeTransitAllowed: true,
		}},
		Units: []*enginev1.Unit{{
			Id:           "ship-1",
			TeamId:       "USA",
			DefinitionId: "surface-ship",
			Position:     &enginev1.Position{Lat: 25.30, Lon: 51.80, Speed: 12},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.MoveUnit("ship-1", 25.44547, 51.66943)
	if !result.Success {
		t.Fatalf("expected surface ship move with maritime transit permission to succeed, got %q", result.Error)
	}
}

func TestAppendMoveWaypoint_SubsurfaceCanTransitForeignTerritorialWaters(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"attack-sub": {
			Domain:         enginev1.UnitDomain_DOMAIN_SUBSURFACE,
			CruiseSpeedMps: 7,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{{
			Id:           "sub-1",
			TeamId:       "USA",
			DefinitionId: "attack-sub",
			Position:     &enginev1.Position{Lat: 25.30, Lon: 51.80, Speed: 7},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.AppendMoveWaypoint("sub-1", 25.44547, 51.66943)
	if !result.Success {
		t.Fatalf("expected subsurface waypoint into foreign territorial waters to succeed, got %q", result.Error)
	}
}

func TestMoveUnit_AirUnitBlockedByForeignAirspaceWithoutTransitPermission(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"fighter": {
			Domain:         enginev1.UnitDomain_DOMAIN_AIR,
			CruiseSpeedMps: 250,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{{
			FromCountry:            "USA",
			ToCountry:              "QAT",
			AirspaceTransitAllowed: true,
		}},
		Units: []*enginev1.Unit{{
			Id:           "air-1",
			TeamId:       "USA",
			DefinitionId: "fighter",
			Position:     &enginev1.Position{Lat: 25.12, Lon: 51.31, AltMsl: 9000, Speed: 250},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.MoveUnit("air-1", 28.95, 50.83)
	if result.Success {
		t.Fatal("expected air move into foreign airspace without transit permission to be blocked")
	}
	if !strings.Contains(result.Error, "airspace") {
		t.Fatalf("expected airspace error, got %q", result.Error)
	}
}

func TestMoveUnit_AirUnitAllowedWithTransitPermission(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"fighter": {
			Domain:         enginev1.UnitDomain_DOMAIN_AIR,
			CruiseSpeedMps: 250,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{
				FromCountry:            "USA",
				ToCountry:              "QAT",
				AirspaceTransitAllowed: true,
			},
			{
				FromCountry:            "USA",
				ToCountry:              "IRN",
				AirspaceTransitAllowed: true,
			},
		},
		Units: []*enginev1.Unit{{
			Id:           "air-1",
			TeamId:       "USA",
			DefinitionId: "fighter",
			Position:     &enginev1.Position{Lat: 25.12, Lon: 51.31, AltMsl: 9000, Speed: 250},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.MoveUnit("air-1", 28.95, 50.83)
	if !result.Success {
		t.Fatalf("expected air move with transit permission to succeed, got %q", result.Error)
	}
}

func TestMoveUnit_LandUnitBlockedByForeignBorder(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"ground-unit": {
			Domain:         enginev1.UnitDomain_DOMAIN_LAND,
			CruiseSpeedMps: 15,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{{
			FromCountry:            "USA",
			ToCountry:              "IRQ",
			AirspaceTransitAllowed: true,
		}},
		Units: []*enginev1.Unit{{
			Id:           "land-1",
			TeamId:       "USA",
			DefinitionId: "ground-unit",
			Position:     &enginev1.Position{Lat: 30.50, Lon: 47.70, Speed: 15},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.MoveUnit("land-1", 30.40, 48.20)
	if result.Success {
		t.Fatal("expected land move across closed foreign border to be blocked")
	}
	if !strings.Contains(result.Error, "land border") {
		t.Fatalf("expected land border error, got %q", result.Error)
	}
}

func TestMoveUnit_LandUnitAllowedWithBorderPermission(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"ground-unit": {
			Domain:         enginev1.UnitDomain_DOMAIN_LAND,
			CruiseSpeedMps: 15,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{
				FromCountry:            "USA",
				ToCountry:              "IRQ",
				AirspaceTransitAllowed: true,
			},
			{
				FromCountry:            "USA",
				ToCountry:              "IRN",
				AirspaceTransitAllowed: true,
			},
		},
		Units: []*enginev1.Unit{{
			Id:           "land-1",
			TeamId:       "USA",
			DefinitionId: "ground-unit",
			Position:     &enginev1.Position{Lat: 30.50, Lon: 47.70, Speed: 15},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.MoveUnit("land-1", 30.40, 48.20)
	if !result.Success {
		t.Fatalf("expected land move with border permission to succeed, got %q", result.Error)
	}
}

func TestMoveUnit_StationaryUnitRejected(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"airbase": {
			Domain:         enginev1.UnitDomain_DOMAIN_LAND,
			CruiseSpeedMps: 0,
			AssetClass:     "airbase",
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{{
			Id:           "base-1",
			TeamId:       "USA",
			DefinitionId: "airbase",
			Position:     &enginev1.Position{Lat: 25.12, Lon: 51.31},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.MoveUnit("base-1", 25.20, 51.40)
	if result.Success {
		t.Fatal("expected stationary unit move to be rejected")
	}
}

func TestAppendMoveWaypoint_StationaryUnitRejected(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"airbase": {
			Domain:         enginev1.UnitDomain_DOMAIN_LAND,
			CruiseSpeedMps: 0,
			AssetClass:     "airbase",
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{{
			Id:           "base-1",
			TeamId:       "USA",
			DefinitionId: "airbase",
			Position:     &enginev1.Position{Lat: 25.12, Lon: 51.31},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.AppendMoveWaypoint("base-1", 25.20, 51.40)
	if result.Success {
		t.Fatal("expected stationary unit waypoint append to be rejected")
	}
}

func TestMoveUnit_SurfaceShipReroutesAroundQatar(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"surface-ship": {
			Domain:         enginev1.UnitDomain_DOMAIN_SEA,
			CruiseSpeedMps: 12,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "SAU", MaritimeTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "ARE", MaritimeTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "BHR", MaritimeTransitAllowed: true},
		},
		Units: []*enginev1.Unit{{
			Id:           "ship-1",
			TeamId:       "USA",
			DefinitionId: "surface-ship",
			Position:     &enginev1.Position{Lat: 24.90, Lon: 50.30, Speed: 12},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.MoveUnit("ship-1", 24.80, 52.30)
	if !result.Success {
		t.Fatalf("expected rerouted surface move to succeed, got %q", result.Error)
	}
	waypoints := app.currentScenario.GetUnits()[0].GetMoveOrder().GetWaypoints()
	if len(waypoints) < 2 {
		t.Fatalf("expected rerouted maritime move order with intermediate waypoints, got %d", len(waypoints))
	}
}

func TestMoveUnit_SurfaceShipCanTransitHormuzPassage(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"surface-ship": {
			Domain:         enginev1.UnitDomain_DOMAIN_SEA,
			CruiseSpeedMps: 12,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{{
			Id:           "ship-1",
			TeamId:       "USA",
			DefinitionId: "surface-ship",
			Position:     &enginev1.Position{Lat: 25.86, Lon: 55.18, Speed: 12},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.MoveUnit("ship-1", 26.34, 56.82)
	if !result.Success {
		t.Fatalf("expected Hormuz transit move to succeed, got %q", result.Error)
	}
	waypoints := app.currentScenario.GetUnits()[0].GetMoveOrder().GetWaypoints()
	if len(waypoints) == 0 {
		t.Fatal("expected routed Hormuz move order")
	}
}

func TestMoveUnit_AirUnitReroutesAroundDeniedQatariAirspace(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"fighter": {
			Domain:         enginev1.UnitDomain_DOMAIN_AIR,
			CruiseSpeedMps: 250,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "SAU", AirspaceTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "ARE", AirspaceTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "BHR", AirspaceTransitAllowed: true},
		},
		Units: []*enginev1.Unit{{
			Id:           "air-1",
			TeamId:       "USA",
			DefinitionId: "fighter",
			Position:     &enginev1.Position{Lat: 24.90, Lon: 50.30, AltMsl: 9000, Speed: 250},
			Status:       &enginev1.OperationalStatus{IsActive: true},
		}},
	}

	result := app.MoveUnit("air-1", 24.80, 53.30)
	if !result.Success {
		t.Fatalf("expected rerouted air move to succeed, got %q", result.Error)
	}
	waypoints := app.currentScenario.GetUnits()[0].GetMoveOrder().GetWaypoints()
	if len(waypoints) < 2 {
		t.Fatalf("expected rerouted air move order with intermediate waypoints, got %d", len(waypoints))
	}
}

func TestReturnUnitToBase_ClearsAttackAndIssuesHomeRoute(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"fighter": {
			Domain:         enginev1.UnitDomain_DOMAIN_AIR,
			CruiseSpeedMps: 250,
		},
		"airbase": {
			Domain: enginev1.UnitDomain_DOMAIN_LAND,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "QAT", AirspaceTransitAllowed: true},
		},
		Units: []*enginev1.Unit{
			{
				Id:           "base-1",
				TeamId:       "USA",
				DefinitionId: "airbase",
				Position:     &enginev1.Position{Lat: 25.12, Lon: 51.31},
				Status:       &enginev1.OperationalStatus{IsActive: true},
			},
			{
				Id:           "air-1",
				TeamId:       "USA",
				DefinitionId: "fighter",
				HostBaseId:   "base-1",
				Position:     &enginev1.Position{Lat: 25.80, Lon: 52.20, AltMsl: 9000, Speed: 250},
				Status:       &enginev1.OperationalStatus{IsActive: true},
				AttackOrder: &enginev1.AttackOrder{
					OrderType:    enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET,
					TargetUnitId: "target-1",
				},
			},
		},
	}

	result := app.ReturnUnitToBase("air-1")
	if !result.Success {
		t.Fatalf("expected return-to-base to succeed, got %q", result.Error)
	}
	aircraft := app.currentScenario.GetUnits()[1]
	if aircraft.GetAttackOrder() != nil {
		t.Fatal("expected return-to-base to clear attack order")
	}
	if aircraft.GetMoveOrder() == nil || len(aircraft.GetMoveOrder().GetWaypoints()) == 0 {
		t.Fatal("expected return-to-base to issue a route home")
	}
	last := aircraft.GetMoveOrder().GetWaypoints()[len(aircraft.GetMoveOrder().GetWaypoints())-1]
	if math.Abs(last.GetLat()-25.12) > 0.0001 || math.Abs(last.GetLon()-51.31) > 0.0001 {
		t.Fatalf("expected final waypoint at host base, got (%f,%f)", last.GetLat(), last.GetLon())
	}
}

func TestPreviewCurrentStrikePath_AirStrikeUsesRoutedPreview(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"strike-aircraft": {
			Domain:         enginev1.UnitDomain_DOMAIN_AIR,
			CruiseSpeedMps: 250,
		},
		"target-site": {
			Domain:      enginev1.UnitDomain_DOMAIN_LAND,
			AssetClass:  "airbase",
			TargetClass: "runway",
		},
	}
	app.weaponCatalogCache = map[string]sim.WeaponStats{
		"agm-158-jassm-er": {
			RangeM:           900_000,
			SpeedMps:         250,
			ProbabilityOfHit: 0.8,
			DomainTargets:    []enginev1.UnitDomain{enginev1.UnitDomain_DOMAIN_LAND},
			EffectType:       enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE,
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "SAU", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
			{FromCountry: "USA", ToCountry: "QAT", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
			{FromCountry: "USA", ToCountry: "ARE", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
			{FromCountry: "USA", ToCountry: "BHR", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
		},
		Units: []*enginev1.Unit{
			{
				Id:           "air-1",
				TeamId:       "USA",
				DefinitionId: "strike-aircraft",
				Position:     &enginev1.Position{Lat: 24.90, Lon: 50.30, AltMsl: 9000, Speed: 250},
				Status:       &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
				Weapons: []*enginev1.WeaponState{
					{WeaponId: "agm-158-jassm-er", CurrentQty: 2, MaxQty: 2},
				},
			},
			{
				Id:           "target-1",
				TeamId:       "IRN",
				DefinitionId: "target-site",
				Position:     &enginev1.Position{Lat: 24.80, Lon: 53.30},
				Status:       &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
			},
		},
	}
	app.lastDetections = map[string][]string{
		"USA": {"target-1"},
	}

	app.currentScenario.GetUnits()[0].AttackOrder = &enginev1.AttackOrder{
		OrderType:     enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT,
		TargetUnitId:  "target-1",
		DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL,
	}

	preview, err := app.PreviewCurrentStrikePath("air-1")
	if err != nil {
		t.Fatalf("PreviewCurrentStrikePath failed: %v", err)
	}
	if preview.Blocked {
		t.Fatalf("expected routed strike preview to succeed, got %q", preview.Reason)
	}
	if len(preview.RoutePoints) == 0 {
		t.Fatal("expected routed strike preview points")
	}
}

func TestRouteCacheInvalidatesAfterRelationshipChange(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{},
	}

	blocked := app.cachedBuildRoute(routing.Request{
		OwnerCountry:      "USA",
		Domain:            enginev1.UnitDomain_DOMAIN_SEA,
		Purpose:           routing.PurposeMove,
		Start:             geo.Point{Lat: 25.30, Lon: 51.80},
		End:               geo.Point{Lat: 25.44547, Lon: 51.66943},
		RelationshipRules: app.relationshipRules(),
		CountryCoalitions: app.countryCoalitions(),
	})
	if !blocked.Blocked {
		t.Fatal("expected initial route to be blocked")
	}

	result := app.SetCountryRelationship("USA", "QAT", false, false, false, false, true, false)
	if !result.Success {
		t.Fatalf("SetCountryRelationship failed: %q", result.Error)
	}

	allowed := app.cachedBuildRoute(routing.Request{
		OwnerCountry:      "USA",
		Domain:            enginev1.UnitDomain_DOMAIN_SEA,
		Purpose:           routing.PurposeMove,
		Start:             geo.Point{Lat: 25.30, Lon: 51.80},
		End:               geo.Point{Lat: 25.44547, Lon: 51.66943},
		RelationshipRules: app.relationshipRules(),
		CountryCoalitions: app.countryCoalitions(),
	})
	if allowed.Blocked {
		t.Fatalf("expected cached route to be invalidated after relationship change, got %q", allowed.Reason)
	}
}

func TestMergeDefStatsWithRowPreservesLibraryTargetMetadataWhenDbRowIsPartial(t *testing.T) {
	base := sim.DefStats{
		Domain:            enginev1.UnitDomain_DOMAIN_LAND,
		AssetClass:        "airbase",
		TargetClass:       "runway",
		EmploymentRole:    "defensive",
		GeneralType:       34,
		StrategicValueUSD: 100_000_000,
	}

	merged := mergeDefStatsWithRow(base, map[string]any{
		"id":                  "israel-strategic-airbase",
		"strategic_value_usd": 200_000_000,
	})

	if got := merged.AssetClass; got != "airbase" {
		t.Fatalf("expected asset class airbase to survive partial DB row, got %q", got)
	}
	if got := merged.TargetClass; got != "runway" {
		t.Fatalf("expected target class runway to survive partial DB row, got %q", got)
	}
	if got := merged.Domain; got != enginev1.UnitDomain_DOMAIN_LAND {
		t.Fatalf("expected domain LAND to survive partial DB row, got %v", got)
	}
	if got := merged.StrategicValueUSD; got != 200_000_000 {
		t.Fatalf("expected DB row to override strategic value, got %v", got)
	}
}

func TestSetUnitLoadoutConfiguration_GroundedHostedAircraftCanSwitch(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{
				Id:           "base-1",
				DisplayName:  "Base One",
				DefinitionId: "airbase",
				BaseOps: &enginev1.BaseOpsState{
					State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE,
				},
			},
			{
				Id:                     "strike-1",
				DisplayName:            "Strike One",
				DefinitionId:           "fighter",
				HostBaseId:             "base-1",
				LoadoutConfigurationId: "cap",
				Position:               &enginev1.Position{},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"fighter": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
	}
	app.libDefsCache = map[string]library.Definition{
		"fighter": {
			ID:                         "fighter",
			DefaultWeaponConfiguration: "cap",
			WeaponConfigurations: []library.WeaponConfiguration{
				{ID: "cap", Loadout: []library.LoadoutSlot{{WeaponID: "aam", MaxQty: 2, InitialQty: 2}}},
				{ID: "strike", Loadout: []library.LoadoutSlot{{WeaponID: "bomb", MaxQty: 4, InitialQty: 4}}},
			},
		},
	}

	result := app.SetUnitLoadoutConfiguration("strike-1", "strike")
	if !result.Success {
		t.Fatalf("expected loadout switch to succeed, got %q", result.Error)
	}
	unit := app.currentScenario.GetUnits()[1]
	if got := unit.GetLoadoutConfigurationId(); got != "strike" {
		t.Fatalf("expected loadout id strike, got %q", got)
	}
	if len(unit.GetWeapons()) != 1 || unit.GetWeapons()[0].GetWeaponId() != "bomb" {
		t.Fatalf("expected switched weapon state, got %+v", unit.GetWeapons())
	}
}

func TestSetUnitAttackOrder_RejectsUndetectedMobileTarget(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{
				Id:           "strike-1",
				DisplayName:  "Strike One",
				DefinitionId: "fighter",
				TeamId:       "USA",
				CoalitionId:  "COALITION_WEST",
				Position:     &enginev1.Position{Lat: 0, Lon: 0, AltMsl: 10_000},
				Status:       &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
			},
			{
				Id:           "mobile-1",
				DisplayName:  "Mobile One",
				DefinitionId: "fighter",
				TeamId:       "IRN",
				CoalitionId:  "COALITION_IRAN",
				Position:     &enginev1.Position{Lat: 0.1, Lon: 0.1, AltMsl: 10_000},
				Status:       &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"fighter": {Domain: enginev1.UnitDomain_DOMAIN_AIR, TargetClass: "aircraft"},
	}

	result := app.SetUnitAttackOrder("strike-1", int32(enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET), "mobile-1", int32(enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY), 0.7)
	if result.Success {
		t.Fatal("expected undetected mobile target assignment to fail")
	}
	if !strings.Contains(result.Error, "current track") {
		t.Fatalf("expected current track error, got %q", result.Error)
	}
}

func TestSetUnitAttackOrder_AllowsUndetectedFixedStrategicTarget(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{
				Id:           "strike-1",
				DisplayName:  "Strike One",
				DefinitionId: "fighter",
				TeamId:       "USA",
				CoalitionId:  "COALITION_WEST",
				Position:     &enginev1.Position{Lat: 0, Lon: 0, AltMsl: 10_000},
				Status:       &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
				Weapons: []*enginev1.WeaponState{
					{WeaponId: "agm-158-jassm-er", CurrentQty: 2, MaxQty: 2},
				},
			},
			{
				Id:           "airbase-1",
				DisplayName:  "Airbase One",
				DefinitionId: "airbase",
				TeamId:       "IRN",
				CoalitionId:  "COALITION_IRAN",
				Position:     &enginev1.Position{Lat: 0.1, Lon: 0.1},
				Status:       &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"fighter": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
		"airbase": {Domain: enginev1.UnitDomain_DOMAIN_LAND, AssetClass: "airbase", TargetClass: "runway"},
	}

	result := app.SetUnitAttackOrder("strike-1", int32(enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET), "airbase-1", int32(enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL), 0.7)
	if !result.Success {
		t.Fatalf("expected fixed strategic target assignment to succeed, got %q", result.Error)
	}
	order := app.currentScenario.GetUnits()[0].GetAttackOrder()
	if order == nil || order.GetTargetUnitId() != "airbase-1" {
		t.Fatalf("expected airbase attack order, got %+v", order)
	}
}

func TestValidateStrikeWithWeapon_AllowsDefensiveAirInterceptWithoutStrikePermission(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{
				FromCountry:                 "USA",
				ToCountry:                   "JOR",
				AirspaceTransitAllowed:      true,
				AirspaceStrikeAllowed:       false,
				DefensivePositioningAllowed: true,
			},
		},
		Units: []*enginev1.Unit{
			{
				Id:           "usa-fighter",
				DisplayName:  "USA Fighter",
				DefinitionId: "fighter",
				TeamId:       "USA",
				Position:     &enginev1.Position{Lat: 31.8, Lon: 36.0, AltMsl: 9_000},
				Weapons:      []*enginev1.WeaponState{{WeaponId: "aam-aim120c", CurrentQty: 2, MaxQty: 2}},
			},
			{
				Id:           "irn-fighter",
				DisplayName:  "Iran Fighter",
				DefinitionId: "fighter",
				TeamId:       "IRN",
				Position:     &enginev1.Position{Lat: 31.9, Lon: 36.4, AltMsl: 9_000},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"fighter": {Domain: enginev1.UnitDomain_DOMAIN_AIR, TargetClass: "aircraft"},
	}
	app.weaponCatalogCache = map[string]sim.WeaponStats{
		"aam-aim120c": {
			RangeM:           100_000,
			ProbabilityOfHit: 0.7,
			DomainTargets:    []enginev1.UnitDomain{enginev1.UnitDomain_DOMAIN_AIR},
			EffectType:       enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_AIR,
		},
	}

	err := app.validateStrikeWithWeapon(app.currentScenario.Units[0], app.currentScenario.Units[1], "aam-aim120c")
	if err != nil {
		t.Fatalf("expected defensive intercept to be allowed, got %v", err)
	}
}

func TestValidateStrikeWithWeapon_BlocksOffensiveStrikeWithoutStrikePermission(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{
				FromCountry:                 "USA",
				ToCountry:                   "JOR",
				AirspaceTransitAllowed:      true,
				AirspaceStrikeAllowed:       false,
				DefensivePositioningAllowed: true,
			},
		},
		Units: []*enginev1.Unit{
			{
				Id:           "usa-striker",
				DisplayName:  "USA Striker",
				DefinitionId: "fighter",
				TeamId:       "USA",
				Position:     &enginev1.Position{Lat: 31.8, Lon: 36.0, AltMsl: 9_000},
				Weapons:      []*enginev1.WeaponState{{WeaponId: "agm-158-jassm-er", CurrentQty: 2, MaxQty: 2}},
			},
			{
				Id:           "irn-airbase",
				DisplayName:  "Iran Airbase",
				DefinitionId: "airbase",
				TeamId:       "IRN",
				Position:     &enginev1.Position{Lat: 31.9, Lon: 36.4, AltMsl: 0},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"fighter": {Domain: enginev1.UnitDomain_DOMAIN_AIR, TargetClass: "aircraft"},
		"airbase": {Domain: enginev1.UnitDomain_DOMAIN_LAND, AssetClass: "airbase", TargetClass: "runway"},
	}
	app.weaponCatalogCache = map[string]sim.WeaponStats{
		"agm-158-jassm-er": {
			RangeM:           400_000,
			ProbabilityOfHit: 0.7,
			DomainTargets:    []enginev1.UnitDomain{enginev1.UnitDomain_DOMAIN_LAND},
			EffectType:       enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE,
		},
	}

	err := app.validateStrikeWithWeapon(app.currentScenario.Units[0], app.currentScenario.Units[1], "agm-158-jassm-er")
	if err == nil {
		t.Fatal("expected offensive strike to be blocked without strike permission")
	}
	if !strings.Contains(err.Error(), "strike operations") {
		t.Fatalf("expected strike permission error, got %v", err)
	}
}

func TestLoadScenarioSeedsOhioSSGNWeaponsFromLibrary(t *testing.T) {
	app := &App{
		ctx: context.Background(),
		libDefsCache: map[string]library.Definition{
			"ohio-ssgn": {
				ID:   "ohio-ssgn",
				Name: "Ohio-class SSGN",
				DefaultLoadout: []library.LoadoutSlot{
					{WeaponID: "ssm-tomahawk", MaxQty: 40, InitialQty: 40},
					{WeaponID: "torp-mk48", MaxQty: 12, InitialQty: 12},
				},
			},
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Id:   "scen-1",
		Name: "Test Scenario",
		Units: []*enginev1.Unit{
			{
				Id:           "usa-ohio-1",
				DisplayName:  "Ohio",
				DefinitionId: "ohio-ssgn",
				TeamId:       "USA",
				CoalitionId:  "COALITION_WEST",
				Position:     &enginev1.Position{Lat: 10, Lon: 10, AltMsl: -30},
				Status: &enginev1.OperationalStatus{
					PersonnelStrength:   1,
					EquipmentStrength:   1,
					CombatEffectiveness: 1,
					FuelLevelLiters:     1,
					Morale:              1,
					Fatigue:             0,
					IsActive:            true,
				},
			},
		},
	}

	app.loadScenario(app.currentScenario)

	if len(app.currentScenario.GetUnits()[0].GetWeapons()) == 0 {
		t.Fatal("expected Ohio SSGN to receive weapons on scenario load")
	}
	weaponIDs := make([]string, 0, len(app.currentScenario.GetUnits()[0].GetWeapons()))
	for _, w := range app.currentScenario.GetUnits()[0].GetWeapons() {
		weaponIDs = append(weaponIDs, w.GetWeaponId())
	}
	got := strings.Join(weaponIDs, ",")
	if !strings.Contains(got, "ssm-tomahawk") {
		t.Fatalf("expected Ohio SSGN loadout to include ssm-tomahawk, got %q", got)
	}
}

func TestLoadScenarioSeedsOhioSSGNWeaponsInIranWarScenario(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
	}

	app := &App{
		ctx:          context.Background(),
		libDefsCache: defsByID,
	}
	scen := scenario.IranCoalitionWarSkeleton()

	app.loadScenario(scen)

	var ohio *enginev1.Unit
	for _, unit := range scen.GetUnits() {
		if unit.GetId() == "usa-ohio-arabian-sea" {
			ohio = unit
			break
		}
	}
	if ohio == nil {
		t.Fatal("expected usa-ohio-arabian-sea in Iran war scenario")
	}
	if len(ohio.GetWeapons()) == 0 {
		t.Fatal("expected Ohio SSGN in Iran war scenario to have seeded weapons")
	}
	weaponIDs := make([]string, 0, len(ohio.GetWeapons()))
	for _, w := range ohio.GetWeapons() {
		weaponIDs = append(weaponIDs, w.GetWeaponId())
	}
	got := strings.Join(weaponIDs, ",")
	if !strings.Contains(got, "ssm-tomahawk") {
		t.Fatalf("expected Ohio SSGN in Iran war scenario to include ssm-tomahawk, got %q", got)
	}
}

func TestPreviewEngagementOptionsReturnsCandidatesForScenarioStrikeAircraft(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
	}

	app := &App{
		ctx:          context.Background(),
		libDefsCache: defsByID,
	}
	scen := scenario.IranCoalitionWarSkeleton()
	app.loadScenario(scen)

	options, err := app.PreviewEngagementOptions("isr-f35i-nevatim")
	if err != nil {
		t.Fatalf("PreviewEngagementOptions failed: %v", err)
	}
	if len(options) == 0 {
		t.Fatal("expected engagement options for isr-f35i-nevatim")
	}
}

func TestPreviewTargetEngagementOptionsReturnsIranianShootersForScenarioEnemyTarget(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
	}

	app := &App{
		ctx:          context.Background(),
		libDefsCache: defsByID,
	}
	scen := scenario.IranCoalitionWarSkeleton()
	app.loadScenario(scen)

	options, err := app.PreviewTargetEngagementOptions("usa-f35a-al-udeid", "IRN")
	if err != nil {
		t.Fatalf("PreviewTargetEngagementOptions failed: %v", err)
	}
	if len(options) == 0 {
		t.Fatal("expected Iranian engagement options for a U.S. target")
	}
	for _, option := range options {
		if option.ShooterTeamId != "IRN" {
			t.Fatalf("expected only Iranian shooters, got %q", option.ShooterTeamId)
		}
	}
}

func TestPreviewTargetEngagementOptions_AllowsPrefixedDefinitionIdsForFixedTargets(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{
				Id:                     "strike-1",
				DisplayName:            "Strike One",
				DefinitionId:           "unit_definition:f35i-adir",
				TeamId:                 "ISR",
				CoalitionId:            "COALITION_WEST",
				LoadoutConfigurationId: "deep_strike",
				Position:               &enginev1.Position{Lat: 0, Lon: 0, AltMsl: 0},
				Status:                 &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
				Weapons: []*enginev1.WeaponState{
					{WeaponId: "agm-158-jassm-er", CurrentQty: 2, MaxQty: 2},
				},
			},
			{
				Id:           "airbase-1",
				DisplayName:  "Target Airbase",
				DefinitionId: "unit_definition:iran-strategic-airbase",
				TeamId:       "IRN",
				CoalitionId:  "COALITION_IRAN",
				Position:     &enginev1.Position{Lat: 0.1, Lon: 0.1},
				Status:       &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"f35i-adir":              {Domain: enginev1.UnitDomain_DOMAIN_AIR},
		"iran-strategic-airbase": {Domain: enginev1.UnitDomain_DOMAIN_LAND, AssetClass: "airbase", TargetClass: "runway"},
	}
	app.lastDetections = map[string][]string{}

	options, err := app.PreviewTargetEngagementOptions("airbase-1", "ISR")
	if err != nil {
		t.Fatalf("PreviewTargetEngagementOptions failed: %v", err)
	}
	if len(options) != 1 {
		t.Fatalf("expected one shooter option, got %d", len(options))
	}
	if !options[0].CanAssign {
		t.Fatalf("expected fixed airbase target to be assignable with prefixed definition ids, got %+v", options[0])
	}
	if options[0].WeaponId != "agm-158-jassm-er" {
		t.Fatalf("expected jassm-er to be selected, got %q", options[0].WeaponId)
	}
}

func TestPreviewTargetEngagementOptions_IncludesIranianMissileShootersForCoalitionAirbase(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
	}

	app := &App{
		ctx:          context.Background(),
		libDefsCache: defsByID,
	}
	scen := scenario.IranCoalitionWarSkeleton()
	app.loadScenario(scen)

	options, err := app.PreviewTargetEngagementOptions("uae-airbase-al-dhafra", "IRN")
	if err != nil {
		t.Fatalf("PreviewTargetEngagementOptions failed: %v", err)
	}
	if len(options) == 0 {
		t.Fatal("expected Iranian shooters for Al Dhafra airbase")
	}
	foundMissileShooter := false
	for _, option := range options {
		if option.ShooterUnitId == "irn-fateh-south" || option.ShooterUnitId == "irn-sejjil-central" || option.ShooterUnitId == "irn-qiam-central" {
			foundMissileShooter = true
			if option.WeaponId == "" {
				t.Fatalf("expected Iranian missile shooter %s to have a selected weapon", option.ShooterUnitId)
			}
			if !option.CanAssign && !option.ReadyToFire {
				t.Fatalf("expected Iranian missile shooter %s to be assignable against a fixed airbase, got %+v", option.ShooterUnitId, option)
			}
		}
	}
	if !foundMissileShooter {
		t.Fatalf("expected at least one Iranian missile shooter in options, got %+v", options)
	}
}

func TestPreviewTargetEngagementOptions_IncludesKheibarBrigadeForFixedAirbaseTarget(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
	}

	app := &App{
		ctx:          context.Background(),
		libDefsCache: defsByID,
	}
	scen := scenario.IranCoalitionWarSkeleton()
	app.loadScenario(scen)

	options, err := app.PreviewTargetEngagementOptions("qat-airbase-al-udeid", "IRN")
	if err != nil {
		t.Fatalf("PreviewTargetEngagementOptions failed: %v", err)
	}
	for _, option := range options {
		if option.ShooterUnitId != "irn-kheibar-west" {
			continue
		}
		if option.WeaponId != "ssm-kheibar-shekan" {
			t.Fatalf("expected Kheibar Brigade to use ssm-kheibar-shekan, got %q", option.WeaponId)
		}
		if !option.CanAssign && !option.ReadyToFire {
			t.Fatalf("expected Kheibar Brigade to be eligible against fixed airbase target, got %+v", option)
		}
		return
	}
	t.Fatal("expected Kheibar Brigade to appear in target engagement options")
}

func TestSetUnitAttackOrder_AllowsKheibarBrigadeAgainstFixedAirbaseTarget(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
	}

	app := &App{
		ctx:          context.Background(),
		libDefsCache: defsByID,
	}
	scen := scenario.IranCoalitionWarSkeleton()
	app.loadScenario(scen)

	result := app.SetUnitAttackOrder(
		"irn-kheibar-west",
		int32(enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET),
		"qat-airbase-al-udeid",
		int32(enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL),
		0.7,
	)
	if !result.Success {
		t.Fatalf("expected Kheibar Brigade fixed-target assignment to succeed, got %q", result.Error)
	}
}

func TestPreviewTargetEngagementOptions_IncludesKheibarBrigadeForNevatimAirbase(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
	}

	app := &App{
		ctx:          context.Background(),
		libDefsCache: defsByID,
	}
	scen := scenario.IranCoalitionWarSkeleton()
	app.loadScenario(scen)

	options, err := app.PreviewTargetEngagementOptions("isr-airbase-nevatim", "IRN")
	if err != nil {
		t.Fatalf("PreviewTargetEngagementOptions failed: %v", err)
	}
	for _, option := range options {
		if option.ShooterUnitId != "irn-kheibar-west" {
			continue
		}
		if option.WeaponId != "ssm-kheibar-shekan" {
			t.Fatalf("expected Kheibar Brigade to use ssm-kheibar-shekan against Nevatim, got %q", option.WeaponId)
		}
		if !option.CanAssign && !option.ReadyToFire {
			t.Fatalf("expected Kheibar Brigade to be eligible against Nevatim, got %+v", option)
		}
		return
	}
	t.Fatal("expected Kheibar Brigade to appear in Nevatim target engagement options")
}

func TestPreviewTargetEngagementOptions_UsesSelectedHumanTeamAsAuthority(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
	}

	app := &App{
		ctx:          context.Background(),
		libDefsCache: defsByID,
	}
	scen := scenario.IranCoalitionWarSkeleton()
	app.loadScenario(scen)
	app.setHumanControlledTeam("IRN")

	options, err := app.PreviewTargetEngagementOptions("isr-airbase-nevatim", "USA")
	if err != nil {
		t.Fatalf("PreviewTargetEngagementOptions failed: %v", err)
	}
	for _, option := range options {
		if option.ShooterUnitId == "irn-kheibar-west" {
			return
		}
	}
	t.Fatal("expected selected human team IRN to override mismatched explicit preview team")
}

func TestPreviewTargetEngagementSummary_NevatimForIran(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
	}

	app := &App{
		ctx:          context.Background(),
		libDefsCache: defsByID,
	}
	scen := scenario.IranCoalitionWarSkeleton()
	app.loadScenario(scen)
	app.setHumanControlledTeam("IRN")

	summary, err := app.PreviewTargetEngagementSummary("isr-airbase-nevatim", "")
	if err != nil {
		t.Fatalf("PreviewTargetEngagementSummary failed: %v", err)
	}
	if summary.PlayerTeam != "IRN" {
		t.Fatalf("expected IRN player team, got %q", summary.PlayerTeam)
	}
	if summary.FriendlyUnitCount == 0 {
		t.Fatal("expected at least one Iranian friendly unit to be evaluated for Nevatim")
	}
	if summary.ReadyShooterCount+summary.AssignableShooterCount == 0 {
		t.Fatalf("expected at least one Iranian shooter to be able to fire or pursue Nevatim, got %+v", summary)
	}
}

func TestUnitRecord_OmitsNilAttackOrder(t *testing.T) {
	record := unitRecord(&enginev1.Unit{
		Id:           "u-1",
		DisplayName:  "Unit",
		FullName:     "Unit Full",
		DefinitionId: "fighter",
		TeamId:       "USA",
		Position:     &enginev1.Position{},
		Status:       &enginev1.OperationalStatus{},
	})
	if _, ok := record["attack_order"]; ok {
		t.Fatal("expected nil attack order to be omitted from unit record")
	}
}

func TestUnitRecord_IncludesAttackOrderWhenPresent(t *testing.T) {
	record := unitRecord(&enginev1.Unit{
		Id:           "u-1",
		DisplayName:  "Unit",
		FullName:     "Unit Full",
		DefinitionId: "fighter",
		TeamId:       "USA",
		Position:     &enginev1.Position{},
		Status:       &enginev1.OperationalStatus{},
		AttackOrder: &enginev1.AttackOrder{
			OrderType:      enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET,
			TargetUnitId:   "target-1",
			DesiredEffect:  enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY,
			PkillThreshold: 0.7,
		},
	})
	if _, ok := record["attack_order"]; !ok {
		t.Fatal("expected present attack order to be included in unit record")
	}
}

func TestUnitRecord_UsesGeometryPointForPosition(t *testing.T) {
	record := unitRecord(&enginev1.Unit{
		Id:           "u-1",
		DisplayName:  "Unit",
		FullName:     "Unit Full",
		DefinitionId: "fighter",
		TeamId:       "USA",
		Position: &enginev1.Position{
			Lat: 25.12,
			Lon: 51.31,
		},
		Status: &enginev1.OperationalStatus{},
	})
	position, ok := record["position"].(models.GeometryPoint)
	if !ok {
		t.Fatalf("expected geometry point position, got %T", record["position"])
	}
	if position.Latitude != 25.12 || position.Longitude != 51.31 {
		t.Fatalf("unexpected geometry point coordinates: %+v", position)
	}
}

func TestPreviewTargetEngagementOptions_IncludesNonOperationalFriendlyShooterWithReason(t *testing.T) {
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
	}

	app := &App{
		ctx:          context.Background(),
		libDefsCache: defsByID,
	}
	scen := scenario.IranCoalitionWarSkeleton()
	app.loadScenario(scen)

	for _, unit := range scen.GetUnits() {
		if unit.GetId() == "irn-kheibar-west" {
			unit.DamageState = enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED
			break
		}
	}

	options, err := app.PreviewTargetEngagementOptions("isr-airbase-nevatim", "IRN")
	if err != nil {
		t.Fatalf("PreviewTargetEngagementOptions failed: %v", err)
	}
	for _, option := range options {
		if option.ShooterUnitId != "irn-kheibar-west" {
			continue
		}
		if got := option.ReasonCode; got != targetEngagementReasonNotOperational {
			t.Fatalf("expected Kheibar Brigade to surface as non-operational, got %q", got)
		}
		return
	}
	t.Fatal("expected non-operational Kheibar Brigade to still appear in target engagement options")
}

func TestPreviewTargetEngagementOptions_IncludesNonHostileFriendlyShooterWithReason(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{
				Id:           "irn-friendly",
				DisplayName:  "Iranian Friendly",
				TeamId:       "IRN",
				CoalitionId:  "COALITION_IRAN",
				DefinitionId: "shooter",
				Position:     &enginev1.Position{Lat: 30, Lon: 50},
				Status:       &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
				Weapons:      []*enginev1.WeaponState{{WeaponId: "ssm-kheibar-shekan", CurrentQty: 1, MaxQty: 1}},
			},
			{
				Id:           "irn-target",
				DisplayName:  "Iranian Airbase",
				TeamId:       "IRN",
				CoalitionId:  "COALITION_IRAN",
				DefinitionId: "target",
				Position:     &enginev1.Position{Lat: 31, Lon: 51},
				Status:       &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"shooter": {Domain: enginev1.UnitDomain_DOMAIN_LAND, GeneralType: 72},
		"target":  {Domain: enginev1.UnitDomain_DOMAIN_LAND, AssetClass: "airbase", TargetClass: "runway"},
	}

	options, err := app.PreviewTargetEngagementOptions("irn-target", "IRN")
	if err != nil {
		t.Fatalf("PreviewTargetEngagementOptions failed: %v", err)
	}
	for _, option := range options {
		if option.ShooterUnitId != "irn-friendly" {
			continue
		}
		if got := option.ReasonCode; got != targetEngagementReasonNotHostile {
			t.Fatalf("expected non-hostile reason, got %q", got)
		}
		return
	}
	t.Fatal("expected non-hostile friendly shooter to appear in target engagement options")
}

func TestPreviewEngagementOptionsTreatsSharedTeamDetectionAsPursuable(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.defsCache = map[string]sim.DefStats{
		"shooter": {
			Domain:          enginev1.UnitDomain_DOMAIN_AIR,
			TargetClass:     "aircraft",
			AssetClass:      "combat_unit",
			DetectionRangeM: 0,
		},
		"target": {
			Domain:      enginev1.UnitDomain_DOMAIN_LAND,
			TargetClass: "runway",
			AssetClass:  "airbase",
		},
	}
	app.currentScenario = &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{
				Id:           "shooter-1",
				TeamId:       "USA",
				CoalitionId:  "BLUE",
				DefinitionId: "shooter",
				Position:     &enginev1.Position{Lat: 25.12, Lon: 51.31},
				Status:       &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
				Weapons: []*enginev1.WeaponState{
					{WeaponId: "agm-158-jassm-er", CurrentQty: 2, MaxQty: 2},
				},
			},
			{
				Id:           "target-1",
				TeamId:       "IRN",
				CoalitionId:  "RED",
				DefinitionId: "target",
				DisplayName:  "Target Airbase",
				Position:     &enginev1.Position{Lat: 28.95, Lon: 50.84},
				Status:       &enginev1.OperationalStatus{IsActive: true, CombatEffectiveness: 1},
			},
		},
	}
	app.storeLastDetection("USA", []string{"target-1"})

	options, err := app.PreviewEngagementOptions("shooter-1")
	if err != nil {
		t.Fatalf("PreviewEngagementOptions failed: %v", err)
	}
	if len(options) != 1 {
		t.Fatalf("expected 1 engagement option, got %d", len(options))
	}
	if !options[0].CanAssign {
		t.Fatal("expected target to remain assignable")
	}
}
