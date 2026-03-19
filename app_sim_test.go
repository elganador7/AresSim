package main

import (
	"context"
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/sim"
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
}

func TestApplyOpeningLoadoutSelections(t *testing.T) {
	scen := &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{Id: "strike-1", LoadoutConfigurationId: "default"},
		},
		OpeningStrikeActions: []*enginev1.OpeningStrikeAction{
			{UnitId: "strike-1", LoadoutConfigurationId: "air_superiority"},
		},
	}

	applyOpeningLoadoutSelections(scen)

	if got := scen.GetUnits()[0].GetLoadoutConfigurationId(); got != "air_superiority" {
		t.Fatalf("expected opening action to override loadout, got %q", got)
	}
}

func TestApplyOpeningStrikeActionsAssignsAttackOrders(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.setSimSeconds(0)
	app.defsCache = map[string]sim.DefStats{
		"airbase": {AssetClass: "airbase", LaunchCapacityPerInterval: 3},
	}
	scen := &enginev1.Scenario{
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
				Id:           "strike-1",
				DisplayName:  "Strike One",
				DefinitionId: "fighter",
				HostBaseId:   "base-1",
				Position:     &enginev1.Position{},
			},
			{
				Id:           "target-1",
				DisplayName:  "Target One",
				DefinitionId: "sam",
				Position:     &enginev1.Position{Lat: 1, Lon: 1},
			},
		},
		OpeningStrikeActions: []*enginev1.OpeningStrikeAction{
			{
				UnitId:        "strike-1",
				TargetUnitId:  "target-1",
				DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL,
			},
		},
	}
	app.currentScenario = scen

	app.applyOpeningStrikeActions(scen, map[string]sim.DefStats{
		"fighter": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
		"sam":     {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"airbase": {AssetClass: "airbase", LaunchCapacityPerInterval: 3},
	})

	order := scen.GetUnits()[1].GetAttackOrder()
	if order == nil {
		t.Fatal("expected opening strike to assign attack order")
	}
	if got := order.GetTargetUnitId(); got != "target-1" {
		t.Fatalf("expected target-1, got %q", got)
	}
	if scen.GetUnits()[0].GetBaseOps().GetNextLaunchAvailableSeconds() <= 0 {
		t.Fatal("expected opening strike to consume host-base launch availability")
	}
}

func TestApplyOpeningStrikeActionsSkipsBlockedLaunch(t *testing.T) {
	app := &App{ctx: context.Background()}
	app.setSimSeconds(0)
	app.defsCache = map[string]sim.DefStats{
		"airbase": {AssetClass: "airbase", LaunchCapacityPerInterval: 3},
	}
	scen := &enginev1.Scenario{
		Units: []*enginev1.Unit{
			{
				Id:           "base-1",
				DisplayName:  "Base One",
				DefinitionId: "airbase",
				BaseOps: &enginev1.BaseOpsState{
					State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_CLOSED,
				},
			},
			{
				Id:           "strike-1",
				DisplayName:  "Strike One",
				DefinitionId: "fighter",
				HostBaseId:   "base-1",
				Position:     &enginev1.Position{},
			},
			{
				Id:           "target-1",
				DisplayName:  "Target One",
				DefinitionId: "sam",
				Position:     &enginev1.Position{Lat: 1, Lon: 1},
			},
		},
		OpeningStrikeActions: []*enginev1.OpeningStrikeAction{
			{
				UnitId:        "strike-1",
				TargetUnitId:  "target-1",
				DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL,
			},
		},
	}
	app.currentScenario = scen

	app.applyOpeningStrikeActions(scen, map[string]sim.DefStats{
		"fighter": {Domain: enginev1.UnitDomain_DOMAIN_AIR},
		"sam":     {Domain: enginev1.UnitDomain_DOMAIN_LAND},
		"airbase": {AssetClass: "airbase", LaunchCapacityPerInterval: 3},
	})

	if scen.GetUnits()[1].GetAttackOrder() != nil {
		t.Fatal("expected blocked launch to leave no attack order")
	}
}
