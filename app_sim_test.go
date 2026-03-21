package main

import (
	"context"
	"strings"
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/library"
	scenario "github.com/aressim/internal/scenario"
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
		ctx:         context.Background(),
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
