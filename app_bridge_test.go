package main

import (
	"context"
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/library"
	"github.com/aressim/internal/scenario"
	"github.com/aressim/internal/sim"
)

func loadTestAppWithLibrary(t *testing.T) *App {
	t.Helper()
	defs, err := library.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
	}
	return &App{
		ctx:          context.Background(),
		libDefsCache: defsByID,
	}
}

func TestMergeWeaponDefinitionWithRowPreservesDefaultTargetsAndEffectForPartialDbRow(t *testing.T) {
	base := &enginev1.WeaponDefinition{
		Id:               "ssm-kheibar-shekan",
		Name:             "Kheibar Shekan",
		DomainTargets:    []enginev1.UnitDomain{enginev1.UnitDomain_DOMAIN_LAND},
		EffectType:       enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE,
		RangeM:           1_450_000,
		ProbabilityOfHit: 0.72,
	}

	merged := mergeWeaponDefinitionWithRow(base, map[string]any{
		"id":                 "ssm-kheibar-shekan",
		"description":        "partial db row",
		"probability_of_hit": 0.8,
	})

	if len(merged.GetDomainTargets()) != 1 || merged.GetDomainTargets()[0] != enginev1.UnitDomain_DOMAIN_LAND {
		t.Fatalf("expected LAND domain target to survive partial DB row, got %+v", merged.GetDomainTargets())
	}
	if got := merged.GetEffectType(); got != enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE {
		t.Fatalf("expected ballistic effect to survive partial DB row, got %v", got)
	}
	if got := merged.GetProbabilityOfHit(); got != 0.8 {
		t.Fatalf("expected DB row to override probability of hit, got %v", got)
	}
}

func TestRunProvingGroundScenarioReturnsSummary(t *testing.T) {
	app := loadTestAppWithLibrary(t)

	result, err := app.RunProvingGroundScenario("pg-modern-vs-legacy-air", 3)
	if err != nil {
		t.Fatalf("RunProvingGroundScenario failed: %v", err)
	}
	if got := result["scenarioId"]; got != "pg-modern-vs-legacy-air" {
		t.Fatalf("expected proving-ground scenario id, got %v", got)
	}
	if got := result["trials"]; got != 3 {
		t.Fatalf("expected 3 trials, got %v", got)
	}
	if _, ok := result["terminalReasons"].(map[string]int); !ok {
		t.Fatalf("expected terminalReasons map[string]int, got %T", result["terminalReasons"])
	}
	if _, ok := result["sampleEvents"].([]sim.ProvingGroundEvent); !ok {
		t.Fatalf("expected sampleEvents []sim.ProvingGroundEvent, got %T", result["sampleEvents"])
	}
	if meanShots, ok := result["meanShotsFired"].(float64); !ok || meanShots <= 0 {
		t.Fatalf("expected positive meanShotsFired, got %v (%T)", result["meanShotsFired"], result["meanShotsFired"])
	}
}

func TestRunProvingGroundScenario_BallisticVsAirbaseProducesMissionKills(t *testing.T) {
	app := loadTestAppWithLibrary(t)

	result, err := app.RunProvingGroundScenario("pg-ballistic-vs-airbase", 10)
	if err != nil {
		t.Fatalf("RunProvingGroundScenario failed: %v", err)
	}
	rate, ok := result["targetMissionKillRate"].(float64)
	if !ok {
		t.Fatalf("expected targetMissionKillRate float64, got %T", result["targetMissionKillRate"])
	}
	if rate <= 0 {
		t.Fatalf("expected ballistic proving ground to produce at least some mission kills, got %v", rate)
	}
}

func TestApplyProvingGroundSetup_BallisticScenarioUsesPreviewAndAttackAPIs(t *testing.T) {
	app := loadTestAppWithLibrary(t)
	_, spec, err := app.prepareProvingGroundScenario("pg-ballistic-vs-airbase")
	if err != nil {
		t.Fatalf("prepareProvingGroundScenario failed: %v", err)
	}
	if err := app.applyProvingGroundSetup(spec); err != nil {
		t.Fatalf("applyProvingGroundSetup failed: %v", err)
	}
	if got := app.getHumanControlledTeam(); got != "IRN" {
		t.Fatalf("expected human team IRN, got %q", got)
	}
	shooter := findScenarioUnit(app.currentScenario.GetUnits(), "pg-kheibar-brigade")
	if shooter == nil || shooter.GetAttackOrder() == nil {
		t.Fatalf("expected Kheibar brigade attack order, got %+v", shooter)
	}
	if got := shooter.GetAttackOrder().GetTargetUnitId(); got != "pg-nevatim-airbase" {
		t.Fatalf("expected Nevatim target, got %q", got)
	}
}

func TestApplyProvingGroundSetup_HostedStrikeUsesLoadoutAndAttackAPIs(t *testing.T) {
	app := loadTestAppWithLibrary(t)
	_, spec, err := app.prepareProvingGroundScenario("pg-hosted-strike-loadout")
	if err != nil {
		t.Fatalf("prepareProvingGroundScenario failed: %v", err)
	}
	if err := app.applyProvingGroundSetup(spec); err != nil {
		t.Fatalf("applyProvingGroundSetup failed: %v", err)
	}
	shooter := findScenarioUnit(app.currentScenario.GetUnits(), "pg-usa-f15e")
	if shooter == nil {
		t.Fatal("expected hosted strike aircraft to exist")
	}
	if got := shooter.GetLoadoutConfigurationId(); got != "deep_strike" {
		t.Fatalf("expected deep_strike loadout, got %q", got)
	}
	if shooter.GetAttackOrder() == nil || shooter.GetAttackOrder().GetTargetUnitId() != "pg-irn-airbase" {
		t.Fatalf("expected hosted strike aircraft attack order against pg-irn-airbase, got %+v", shooter.GetAttackOrder())
	}
}

func TestApplyProvingGroundSetup_DestroyerScenarioUsesAttackAPI(t *testing.T) {
	app := loadTestAppWithLibrary(t)
	_, spec, err := app.prepareProvingGroundScenario("pg-destroyer-vs-missile-boat")
	if err != nil {
		t.Fatalf("prepareProvingGroundScenario failed: %v", err)
	}
	if err := app.applyProvingGroundSetup(spec); err != nil {
		t.Fatalf("applyProvingGroundSetup failed: %v", err)
	}
	shooter := findScenarioUnit(app.currentScenario.GetUnits(), "pg-usa-mmsc")
	if shooter == nil {
		t.Fatal("expected destroyer to exist")
	}
	if shooter.GetAttackOrder() == nil || shooter.GetAttackOrder().GetTargetUnitId() != "pg-irn-missile-boat" {
		t.Fatalf("expected destroyer attack order against pg-irn-missile-boat, got %+v", shooter.GetAttackOrder())
	}
}

func TestBuiltinByIDFindsProvingGroundScenario(t *testing.T) {
	scen := scenario.BuiltinByID("pg-ballistic-vs-airbase")
	if scen == nil {
		t.Fatal("expected proving-ground scenario to be available as a built-in")
	}
	if got := scen.GetClassification(); got != "PROVING GROUND" {
		t.Fatalf("expected proving-ground classification, got %q", got)
	}
}
