package main

import (
	"context"
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/sim"
)

func TestPlanMajorActorStrikesChoosesHigherPainTarget(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.setSimSeconds(0)
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "QAT", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
		},
		Units: []*enginev1.Unit{
			{
				Id:           "usa-airbase",
				DisplayName:  "Al Udeid",
				TeamId:       "USA",
				CoalitionId:  "Blue",
				DefinitionId: "airbase",
				BaseOps:      &enginev1.BaseOpsState{State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE},
			},
			{
				Id:                     "usa-striker",
				DisplayName:            "Strike Eagle 1",
				TeamId:                 "USA",
				CoalitionId:            "Blue",
				DefinitionId:           "fighter",
				HostBaseId:             "usa-airbase",
				Position:               &enginev1.Position{Lat: 25, Lon: 51, AltMsl: 0},
				Status:                 &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
				Weapons:                []*enginev1.WeaponState{{WeaponId: "agm-158-jassm-er", CurrentQty: 2, MaxQty: 2}},
				NextSortieReadySeconds: 0,
			},
			{
				Id:           "irn-missiles",
				DisplayName:  "Missile Brigade",
				TeamId:       "IRN",
				CoalitionId:  "Red",
				DefinitionId: "missile",
				Position:     &enginev1.Position{Lat: 27, Lon: 52.2},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
			},
			{
				Id:           "irn-power",
				DisplayName:  "Power Plant",
				TeamId:       "IRN",
				CoalitionId:  "Red",
				DefinitionId: "power",
				Position:     &enginev1.Position{Lat: 27.1, Lon: 52.1},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"airbase": {AssetClass: "airbase", LaunchCapacityPerInterval: 3},
		"fighter": {
			Domain:              enginev1.UnitDomain_DOMAIN_AIR,
			EmploymentRole:      "dual_use",
			CruiseSpeedMps:      250,
			AuthorizedPersonnel: 1,
		},
		"missile": {
			Domain:              enginev1.UnitDomain_DOMAIN_LAND,
			GeneralType:         72,
			AssetClass:          "combat_unit",
			TargetClass:         "soft_infrastructure",
			ReplacementCostUSD:  100_000_000,
			StrategicValueUSD:   1_200_000_000,
			AuthorizedPersonnel: 50,
		},
		"power": {
			Domain:              enginev1.UnitDomain_DOMAIN_LAND,
			AssetClass:          "power_plant",
			TargetClass:         "civilian_energy",
			ReplacementCostUSD:  900_000_000,
			StrategicValueUSD:   1_200_000_000,
			EconomicValueUSD:    3_000_000_000,
			AuthorizedPersonnel: 120,
		},
	}

	deltas := app.planMajorActorStrikes(0)
	if len(deltas) == 0 {
		t.Fatal("expected AI to assign a strike")
	}
	order := app.currentScenario.GetUnits()[1].GetAttackOrder()
	if order == nil {
		t.Fatal("expected strike order to be assigned")
	}
	if got := order.GetTargetUnitId(); got != "irn-missiles" {
		t.Fatalf("expected higher-priority missile brigade target, got %q", got)
	}
}
