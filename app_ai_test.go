package main

import (
	"context"
	"math/rand"
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/sim"
)

func TestPlanMajorActorStrikesChoosesHigherPainTarget(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.setSimSeconds(0)
	app.setHumanControlledTeam("ISR")
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
				Position:               &enginev1.Position{Lat: 25, Lon: 51, AltMsl: 5000},
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

func TestPlanMajorActorStrikesSkipsHumanControlledTeam(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.setSimSeconds(0)
	app.setHumanControlledTeam("USA")
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "ISR", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
			{FromCountry: "USA", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
		},
		Units: []*enginev1.Unit{
			{
				Id:           "isr-airbase",
				DisplayName:  "Hatzerim",
				TeamId:       "ISR",
				CoalitionId:  "Blue",
				DefinitionId: "airbase",
				BaseOps:      &enginev1.BaseOpsState{State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE},
			},
			{
				Id:           "usa-airbase",
				DisplayName:  "Al Udeid",
				TeamId:       "USA",
				CoalitionId:  "Blue",
				DefinitionId: "airbase",
				BaseOps:      &enginev1.BaseOpsState{State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE},
			},
			{
				Id:                     "isr-striker",
				DisplayName:            "Israeli Strike Jet",
				TeamId:                 "ISR",
				CoalitionId:            "Blue",
				DefinitionId:           "fighter",
				HostBaseId:             "isr-airbase",
				Position:               &enginev1.Position{Lat: 31.3, Lon: 34.7, AltMsl: 0},
				Status:                 &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
				Weapons:                []*enginev1.WeaponState{{WeaponId: "agm-158-jassm-er", CurrentQty: 2, MaxQty: 2}},
				NextSortieReadySeconds: 0,
			},
			{
				Id:                     "usa-striker",
				DisplayName:            "American Strike Jet",
				TeamId:                 "USA",
				CoalitionId:            "Blue",
				DefinitionId:           "fighter",
				HostBaseId:             "usa-airbase",
				Position:               &enginev1.Position{Lat: 25, Lon: 51, AltMsl: 5000},
				Status:                 &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
				Weapons:                []*enginev1.WeaponState{{WeaponId: "agm-158-jassm-er", CurrentQty: 2, MaxQty: 2}},
				NextSortieReadySeconds: 0,
			},
			{
				Id:           "irn-target",
				DisplayName:  "Iranian Airbase",
				TeamId:       "IRN",
				CoalitionId:  "Red",
				DefinitionId: "target",
				Position:     &enginev1.Position{Lat: 31.5, Lon: 34.9},
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
		"target": {
			Domain:              enginev1.UnitDomain_DOMAIN_LAND,
			AssetClass:          "airbase",
			TargetClass:         "runway",
			ReplacementCostUSD:  100_000_000,
			StrategicValueUSD:   200_000_000,
			AuthorizedPersonnel: 50,
		},
	}

	app.planMajorActorStrikes(0)

	if app.currentScenario.GetUnits()[2].GetAttackOrder() == nil {
		t.Fatal("expected non-human major actor to receive a strike order")
	}
	if app.currentScenario.GetUnits()[3].GetAttackOrder() != nil {
		t.Fatal("expected human-controlled team to remain unassigned by AI")
	}
}

func TestPlanMajorActorStrikesWaitsForHumanTeamSelection(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.setSimSeconds(0)
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "ISR", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
			{FromCountry: "USA", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
			{FromCountry: "IRN", ToCountry: "ISR", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
		},
		Units: []*enginev1.Unit{
			{
				Id:           "isr-airbase",
				DisplayName:  "Hatzerim",
				TeamId:       "ISR",
				CoalitionId:  "Blue",
				DefinitionId: "airbase",
				BaseOps:      &enginev1.BaseOpsState{State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE},
			},
			{
				Id:           "irn-airbase",
				DisplayName:  "Esfahan",
				TeamId:       "IRN",
				CoalitionId:  "Red",
				DefinitionId: "airbase",
				BaseOps:      &enginev1.BaseOpsState{State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE},
			},
			{
				Id:                     "isr-striker",
				DisplayName:            "Israeli Strike Jet",
				TeamId:                 "ISR",
				CoalitionId:            "Blue",
				DefinitionId:           "fighter",
				HostBaseId:             "isr-airbase",
				Position:               &enginev1.Position{Lat: 31.3, Lon: 34.7, AltMsl: 0},
				Status:                 &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
				Weapons:                []*enginev1.WeaponState{{WeaponId: "agm-158-jassm-er", CurrentQty: 2, MaxQty: 2}},
				NextSortieReadySeconds: 0,
			},
			{
				Id:           "irn-missile",
				DisplayName:  "Kheibar Brigade",
				TeamId:       "IRN",
				CoalitionId:  "Red",
				DefinitionId: "missile",
				Position:     &enginev1.Position{Lat: 35.2, Lon: 46.98, AltMsl: 0},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
				Weapons:      []*enginev1.WeaponState{{WeaponId: "ssm-kheibar-shekan", CurrentQty: 2, MaxQty: 2}},
			},
			{
				Id:           "isr-target",
				DisplayName:  "Nevatim",
				TeamId:       "ISR",
				CoalitionId:  "Blue",
				DefinitionId: "target",
				Position:     &enginev1.Position{Lat: 31.21, Lon: 35.01},
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
			EmploymentRole:      "offensive",
			GeneralType:         72,
			AuthorizedPersonnel: 20,
		},
		"target": {
			Domain:              enginev1.UnitDomain_DOMAIN_LAND,
			AssetClass:          "airbase",
			TargetClass:         "runway",
			ReplacementCostUSD:  100_000_000,
			StrategicValueUSD:   200_000_000,
			AuthorizedPersonnel: 50,
		},
	}

	app.planMajorActorStrikes(0)

	for _, unit := range app.currentScenario.GetUnits() {
		if unit.GetAttackOrder() != nil {
			t.Fatalf("expected no AI strike orders before player team selection, got %s -> %s", unit.GetId(), unit.GetAttackOrder().GetTargetUnitId())
		}
	}
}

func TestPlanMajorActorStrikesDeconflictsAirAttackers(t *testing.T) {
	app := &App{ctx: context.Background(), aiRand: rand.New(rand.NewSource(1))} //nolint:gosec
	app.setSimSeconds(0)
	app.setHumanControlledTeam("ISR")
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "QAT", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
			{FromCountry: "USA", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
		},
		Units: []*enginev1.Unit{
			{
				Id:           "base-1",
				DisplayName:  "Al Udeid 1",
				TeamId:       "USA",
				CoalitionId:  "Blue",
				DefinitionId: "airbase",
				BaseOps:      &enginev1.BaseOpsState{State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE},
			},
			{
				Id:           "base-2",
				DisplayName:  "Al Udeid 2",
				TeamId:       "USA",
				CoalitionId:  "Blue",
				DefinitionId: "airbase",
				BaseOps:      &enginev1.BaseOpsState{State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE},
			},
			{
				Id:                     "striker-1",
				DisplayName:            "Strike 1",
				TeamId:                 "USA",
				CoalitionId:            "Blue",
				DefinitionId:           "fighter",
				HostBaseId:             "base-1",
				Position:               &enginev1.Position{Lat: 25, Lon: 51, AltMsl: 0},
				Status:                 &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
				Weapons:                []*enginev1.WeaponState{{WeaponId: "agm-158-jassm-er", CurrentQty: 2, MaxQty: 2}},
				NextSortieReadySeconds: 0,
			},
			{
				Id:                     "striker-2",
				DisplayName:            "Strike 2",
				TeamId:                 "USA",
				CoalitionId:            "Blue",
				DefinitionId:           "fighter",
				HostBaseId:             "base-2",
				Position:               &enginev1.Position{Lat: 25, Lon: 51, AltMsl: 0},
				Status:                 &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
				Weapons:                []*enginev1.WeaponState{{WeaponId: "agm-158-jassm-er", CurrentQty: 2, MaxQty: 2}},
				NextSortieReadySeconds: 0,
			},
			{
				Id:           "target-a",
				DisplayName:  "Target A",
				TeamId:       "IRN",
				CoalitionId:  "Red",
				DefinitionId: "airbase-target",
				Position:     &enginev1.Position{Lat: 27.1, Lon: 52.1},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
			},
			{
				Id:           "target-b",
				DisplayName:  "Target B",
				TeamId:       "IRN",
				CoalitionId:  "Red",
				DefinitionId: "airbase-target",
				Position:     &enginev1.Position{Lat: 27.15, Lon: 52.15},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"airbase":        {AssetClass: "airbase", LaunchCapacityPerInterval: 4},
		"fighter":        {Domain: enginev1.UnitDomain_DOMAIN_AIR, EmploymentRole: "dual_use", CruiseSpeedMps: 250, AuthorizedPersonnel: 1},
		"airbase-target": {Domain: enginev1.UnitDomain_DOMAIN_LAND, AssetClass: "airbase", TargetClass: "runway", ReplacementCostUSD: 100_000_000, StrategicValueUSD: 200_000_000, AuthorizedPersonnel: 50},
	}

	app.planMajorActorStrikes(0)

	first := app.currentScenario.GetUnits()[2].GetAttackOrder()
	second := app.currentScenario.GetUnits()[3].GetAttackOrder()
	if first == nil || second == nil {
		t.Fatal("expected both AI-controlled aircraft to receive strike orders")
	}
	if first.GetTargetUnitId() == second.GetTargetUnitId() {
		t.Fatalf("expected deconflicted targets, both aircraft assigned %q", first.GetTargetUnitId())
	}
}

func TestPlanMajorActorStrikesAllowsStrategicRaidBatching(t *testing.T) {
	app := &App{ctx: context.Background(), aiRand: rand.New(rand.NewSource(1))} //nolint:gosec
	app.setSimSeconds(0)
	app.setHumanControlledTeam("USA")
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "IRN", ToCountry: "ISR", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
		},
		Units: []*enginev1.Unit{
			{
				Id:           "missile-1",
				DisplayName:  "Missile 1",
				TeamId:       "IRN",
				CoalitionId:  "Red",
				DefinitionId: "missile",
				Position:     &enginev1.Position{Lat: 29, Lon: 51.5, AltMsl: 0},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
				Weapons:      []*enginev1.WeaponState{{WeaponId: "ssm-kheibar-shekan", CurrentQty: 2, MaxQty: 2}},
			},
			{
				Id:           "missile-2",
				DisplayName:  "Missile 2",
				TeamId:       "IRN",
				CoalitionId:  "Red",
				DefinitionId: "missile",
				Position:     &enginev1.Position{Lat: 29.1, Lon: 51.6, AltMsl: 0},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
				Weapons:      []*enginev1.WeaponState{{WeaponId: "ssm-kheibar-shekan", CurrentQty: 2, MaxQty: 2}},
			},
			{
				Id:           "isr-airbase",
				DisplayName:  "Israeli Airbase",
				TeamId:       "ISR",
				CoalitionId:  "Blue",
				DefinitionId: "high-value-airbase",
				Position:     &enginev1.Position{Lat: 29.2, Lon: 51.7},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
			},
			{
				Id:           "isr-power",
				DisplayName:  "Israeli Power",
				TeamId:       "ISR",
				CoalitionId:  "Blue",
				DefinitionId: "mid-value-power",
				Position:     &enginev1.Position{Lat: 29.25, Lon: 51.75},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"missile":            {Domain: enginev1.UnitDomain_DOMAIN_LAND, EmploymentRole: "offensive", GeneralType: 72, AuthorizedPersonnel: 20},
		"high-value-airbase": {Domain: enginev1.UnitDomain_DOMAIN_LAND, AssetClass: "airbase", TargetClass: "runway", ReplacementCostUSD: 100_000_000, StrategicValueUSD: 900_000_000, AuthorizedPersonnel: 50},
		"mid-value-power":    {Domain: enginev1.UnitDomain_DOMAIN_LAND, AssetClass: "power_plant", TargetClass: "civilian_energy", ReplacementCostUSD: 100_000_000, StrategicValueUSD: 500_000_000, EconomicValueUSD: 200_000_000, AuthorizedPersonnel: 50},
	}

	app.planMajorActorStrikes(0)

	first := app.currentScenario.GetUnits()[0].GetAttackOrder()
	second := app.currentScenario.GetUnits()[1].GetAttackOrder()
	if first == nil || second == nil {
		t.Fatal("expected both strategic shooters to receive strike orders")
	}
	if first.GetTargetUnitId() != "isr-airbase" || second.GetTargetUnitId() != "isr-airbase" {
		t.Fatalf("expected raid batching on high-value airbase, got %q and %q", first.GetTargetUnitId(), second.GetTargetUnitId())
	}
	for _, unit := range app.currentScenario.GetUnits()[:2] {
		if unit.GetNextStrikeReadySeconds() <= 0 {
			t.Fatalf("expected strategic shooter %s to receive timing jitter", unit.GetId())
		}
	}
}

func TestPlanMajorActorStrikesIranPrioritizesAirbaseClosure(t *testing.T) {
	app := &App{ctx: context.Background(), aiRand: rand.New(rand.NewSource(1))} //nolint:gosec
	app.setSimSeconds(0)
	app.setHumanControlledTeam("USA")
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "IRN", ToCountry: "ISR", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
		},
		Units: []*enginev1.Unit{
			{
				Id:           "irn-missile",
				DisplayName:  "Missile Brigade",
				TeamId:       "IRN",
				CoalitionId:  "Red",
				DefinitionId: "missile",
				Position:     &enginev1.Position{Lat: 29, Lon: 51.5, AltMsl: 0},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
				Weapons:      []*enginev1.WeaponState{{WeaponId: "ssm-kheibar-shekan", CurrentQty: 2, MaxQty: 2}},
			},
			{
				Id:           "coalition-airbase",
				DisplayName:  "Coalition Airbase",
				TeamId:       "ISR",
				CoalitionId:  "Blue",
				DefinitionId: "airbase-target",
				Position:     &enginev1.Position{Lat: 29.2, Lon: 51.7},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
			},
			{
				Id:           "coalition-power",
				DisplayName:  "Coalition Power",
				TeamId:       "ISR",
				CoalitionId:  "Blue",
				DefinitionId: "power-target",
				Position:     &enginev1.Position{Lat: 29.25, Lon: 51.75},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"missile":        {Domain: enginev1.UnitDomain_DOMAIN_LAND, EmploymentRole: "offensive", GeneralType: 72, AuthorizedPersonnel: 20},
		"airbase-target": {Domain: enginev1.UnitDomain_DOMAIN_LAND, AssetClass: "airbase", TargetClass: "runway", ReplacementCostUSD: 120_000_000, StrategicValueUSD: 700_000_000, AuthorizedPersonnel: 50},
		"power-target":   {Domain: enginev1.UnitDomain_DOMAIN_LAND, AssetClass: "power_plant", TargetClass: "civilian_energy", ReplacementCostUSD: 300_000_000, StrategicValueUSD: 600_000_000, EconomicValueUSD: 300_000_000, AuthorizedPersonnel: 50},
	}

	app.planMajorActorStrikes(0)

	order := app.currentScenario.GetUnits()[0].GetAttackOrder()
	if order == nil {
		t.Fatal("expected iranian strike order to be assigned")
	}
	if got := order.GetTargetUnitId(); got != "coalition-airbase" {
		t.Fatalf("expected iranian planner to favor airbase closure, got %q", got)
	}
}

func TestPlanMajorActorStrikesSkipsNavalLandTargets(t *testing.T) {
	app := &App{ctx: context.Background(), aiRand: rand.New(rand.NewSource(1))} //nolint:gosec
	app.setSimSeconds(0)
	app.currentScenario = &enginev1.Scenario{
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "IRN", ToCountry: "ISR", MaritimeTransitAllowed: true, MaritimeStrikeAllowed: true},
		},
		Units: []*enginev1.Unit{
			{
				Id:           "irn-sub",
				DisplayName:  "Iranian Sub",
				TeamId:       "IRN",
				CoalitionId:  "Red",
				DefinitionId: "sub",
				Position:     &enginev1.Position{Lat: 26.0, Lon: 56.0, AltMsl: -30},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
				Weapons:      []*enginev1.WeaponState{{WeaponId: "ssm-tomahawk", CurrentQty: 2, MaxQty: 2}},
			},
			{
				Id:           "isr-airbase",
				DisplayName:  "Israeli Airbase",
				TeamId:       "ISR",
				CoalitionId:  "Blue",
				DefinitionId: "airbase-target",
				Position:     &enginev1.Position{Lat: 31.21, Lon: 35.01},
				Status:       &enginev1.OperationalStatus{IsActive: true, PersonnelStrength: 1},
			},
		},
	}
	app.defsCache = map[string]sim.DefStats{
		"sub":            {Domain: enginev1.UnitDomain_DOMAIN_SUBSURFACE, EmploymentRole: "offensive", GeneralType: 50, AuthorizedPersonnel: 20},
		"airbase-target": {Domain: enginev1.UnitDomain_DOMAIN_LAND, AssetClass: "airbase", TargetClass: "runway", ReplacementCostUSD: 120_000_000, StrategicValueUSD: 700_000_000, AuthorizedPersonnel: 50},
	}

	app.planMajorActorStrikes(0)

	if app.currentScenario.GetUnits()[0].GetAttackOrder() != nil {
		t.Fatal("expected maritime AI shooter to skip land target")
	}
}
