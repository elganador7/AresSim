package main

import (
	"context"
	"strings"
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

func TestRunProvingGroundScenario_IsraelMissileDefenseProducesInterceptions(t *testing.T) {
	app := loadTestAppWithLibrary(t)

	result, err := app.RunProvingGroundScenario("pg-israel-missile-defense-saturation", 3)
	if err != nil {
		t.Fatalf("RunProvingGroundScenario failed: %v", err)
	}
	interceptionRate, ok := result["interceptionRate"].(float64)
	if !ok {
		t.Fatalf("expected interceptionRate float64, got %T", result["interceptionRate"])
	}
	if interceptionRate <= 0 {
		t.Fatalf("expected missile-defense proving ground to intercept something, got %v", interceptionRate)
	}
	if _, ok := result["meanFocusHitsTaken"].(float64); !ok {
		t.Fatalf("expected meanFocusHitsTaken float64, got %T", result["meanFocusHitsTaken"])
	}
	opposingLosses, ok := result["meanOpposingLosses"].(float64)
	if !ok {
		t.Fatalf("expected meanOpposingLosses float64, got %T", result["meanOpposingLosses"])
	}
	if opposingLosses <= 0 {
		t.Fatalf("expected missile-defense proving ground to destroy some launchers, got %v", opposingLosses)
	}
}

func TestRunProvingGroundScenario_PackageAlUdeidSortieProducesStrike(t *testing.T) {
	app := loadTestAppWithLibrary(t)

	result, err := app.RunProvingGroundScenario("pg-package-al-udeid-sortie", 3)
	if err != nil {
		t.Fatalf("RunProvingGroundScenario failed: %v", err)
	}
	rate, ok := result["targetMissionKillRate"].(float64)
	if !ok {
		t.Fatalf("expected targetMissionKillRate float64, got %T", result["targetMissionKillRate"])
	}
	if rate <= 0 {
		t.Fatalf("expected package sortie proving ground to produce mission kills, got %v", rate)
	}
	if fuelOuts, ok := result["meanFuelExhaustions"].(float64); !ok {
		t.Fatalf("expected meanFuelExhaustions float64, got %T", result["meanFuelExhaustions"])
	} else if fuelOuts != 0 {
		t.Fatalf("expected package sortie proving ground to avoid fuel exhaustion, got %v", fuelOuts)
	}
}

func TestRunProvingGroundScenario_PackageBahrainMaritimePresenceProducesDefense(t *testing.T) {
	app := loadTestAppWithLibrary(t)

	result, err := app.RunProvingGroundScenario("pg-package-bahrain-maritime-presence", 3)
	if err != nil {
		t.Fatalf("RunProvingGroundScenario failed: %v", err)
	}
	rate, ok := result["targetMissionKillRate"].(float64)
	if !ok {
		t.Fatalf("expected targetMissionKillRate float64, got %T", result["targetMissionKillRate"])
	}
	if rate <= 0 {
		t.Fatalf("expected Bahrain package proving ground to disable the raider in some runs, got %v", rate)
	}
}

func TestRunProvingGroundScenario_PackageAlDhafraForwardStrikeProducesMissionEffects(t *testing.T) {
	app := loadTestAppWithLibrary(t)

	result, err := app.RunProvingGroundScenario("pg-package-al-dhafra-forward-strike", 3)
	if err != nil {
		t.Fatalf("RunProvingGroundScenario failed: %v", err)
	}
	rate, ok := result["targetMissionKillRate"].(float64)
	if !ok {
		t.Fatalf("expected targetMissionKillRate float64, got %T", result["targetMissionKillRate"])
	}
	if rate <= 0 {
		t.Fatalf("expected Al Dhafra package proving ground to produce mission effects, got %v", rate)
	}
}

func TestRunProvingGroundScenario_PackageGulfRegionalSupportPostureProducesCombinedEffects(t *testing.T) {
	app := loadTestAppWithLibrary(t)

	result, err := app.RunProvingGroundScenario("pg-package-gulf-regional-support-posture", 3)
	if err != nil {
		t.Fatalf("RunProvingGroundScenario failed: %v", err)
	}
	rate, ok := result["targetMissionKillRate"].(float64)
	if !ok {
		t.Fatalf("expected targetMissionKillRate float64, got %T", result["targetMissionKillRate"])
	}
	if rate <= 0 {
		t.Fatalf("expected Gulf regional support posture to produce mission effects against Bushehr, got %v", rate)
	}
	if losses, ok := result["meanOpposingLosses"].(float64); !ok {
		t.Fatalf("expected meanOpposingLosses float64, got %T", result["meanOpposingLosses"])
	} else if losses <= 0 {
		t.Fatalf("expected Gulf regional support posture to inflict some opposing losses, got %v", losses)
	}
}

func TestRunProvingGroundScenario_PackageHormuzCoastalDenialProducesLittoralEffects(t *testing.T) {
	app := loadTestAppWithLibrary(t)

	result, err := app.RunProvingGroundScenario("pg-package-hormuz-coastal-denial", 3)
	if err != nil {
		t.Fatalf("RunProvingGroundScenario failed: %v", err)
	}
	rate, ok := result["targetMissionKillRate"].(float64)
	if !ok {
		t.Fatalf("expected targetMissionKillRate float64, got %T", result["targetMissionKillRate"])
	}
	if rate <= 0 {
		t.Fatalf("expected Hormuz coastal-denial package to produce mission effects on the transit group, got %v", rate)
	}
}

func TestRunProvingGroundScenario_PackageUAECoastalDefenseProducesSomeAttrition(t *testing.T) {
	app := loadTestAppWithLibrary(t)

	result, err := app.RunProvingGroundScenario("pg-package-uae-coastal-defense", 3)
	if err != nil {
		t.Fatalf("RunProvingGroundScenario failed: %v", err)
	}
	losses, ok := result["meanOpposingLosses"].(float64)
	if !ok {
		t.Fatalf("expected meanOpposingLosses float64, got %T", result["meanOpposingLosses"])
	}
	if losses <= 0 {
		t.Fatalf("expected UAE coastal-defense package to inflict some losses, got %v", losses)
	}
}

func TestRunProvingGroundScenario_PackageOMNMusandamStraitGuardProducesSomeAttrition(t *testing.T) {
	app := loadTestAppWithLibrary(t)

	result, err := app.RunProvingGroundScenario("pg-package-oman-musandam-strait-guard", 3)
	if err != nil {
		t.Fatalf("RunProvingGroundScenario failed: %v", err)
	}
	losses, ok := result["meanOpposingLosses"].(float64)
	if !ok {
		t.Fatalf("expected meanOpposingLosses float64, got %T", result["meanOpposingLosses"])
	}
	if losses <= 0 {
		t.Fatalf("expected Omani strait-guard package to inflict some losses, got %v", losses)
	}
}

func TestRunProvingGroundScenario_PackageStraitRegionalControlProducesSomeAttrition(t *testing.T) {
	app := loadTestAppWithLibrary(t)

	result, err := app.RunProvingGroundScenario("pg-package-strait-regional-control", 3)
	if err != nil {
		t.Fatalf("RunProvingGroundScenario failed: %v", err)
	}
	losses, ok := result["meanOpposingLosses"].(float64)
	if !ok {
		t.Fatalf("expected meanOpposingLosses float64, got %T", result["meanOpposingLosses"])
	}
	if losses <= 0 {
		t.Fatalf("expected Strait regional control package composition to inflict some losses, got %v", losses)
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

func TestPrepareProvingGroundScenario_PackageAlUdeidSortieUsesStrikeElement(t *testing.T) {
	app := loadTestAppWithLibrary(t)
	_, spec, err := app.prepareProvingGroundScenario("pg-package-al-udeid-sortie")
	if err != nil {
		t.Fatalf("prepareProvingGroundScenario failed: %v", err)
	}
	var leadAssignFound bool
	for _, action := range spec.SetupActions {
		if action.Kind == "assign_attack" && action.UnitID == "pg-pkg-au-f15e-lead" && action.TargetUnitID == "pg-pkg-au-bushehr" {
			leadAssignFound = true
			break
		}
	}
	if !leadAssignFound {
		t.Fatal("expected proving-ground setup to assign F-15E lead against Bushehr")
	}
	f15 := findScenarioUnit(app.currentScenario.GetUnits(), "pg-pkg-au-f15e-lead")
	if f15 == nil {
		t.Fatal("expected F-15E lead to exist")
	}
	if got := f15.GetLoadoutConfigurationId(); got != "deep_strike" {
		t.Fatalf("expected deep_strike loadout, got %q", got)
	}
	f15Wing := findScenarioUnit(app.currentScenario.GetUnits(), "pg-pkg-au-f15e-wing")
	if f15Wing == nil {
		t.Fatal("expected F-15E wing to exist")
	}
	if f15Wing.GetAttackOrder() == nil || f15Wing.GetAttackOrder().GetTargetUnitId() != "pg-pkg-au-bushehr" {
		t.Fatalf("expected F-15E wing built-in attack order against Bushehr, got %+v", f15Wing.GetAttackOrder())
	}
	f35 := findScenarioUnit(app.currentScenario.GetUnits(), "pg-pkg-au-f35a-lead")
	if f35 == nil {
		t.Fatal("expected F-35A lead to exist")
	}
	if got := f35.GetLoadoutConfigurationId(); got != "internal_strike" {
		t.Fatalf("expected internal_strike loadout, got %q", got)
	}
	if f35.GetAttackOrder() == nil || f35.GetAttackOrder().GetTargetUnitId() != "pg-pkg-au-shiraz" {
		t.Fatalf("expected F-35A lead built-in attack order against Shiraz, got %+v", f35.GetAttackOrder())
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

func TestPrepareProvingGroundScenario_PackageLayeredIsraelDefenseUsesCounterstrikeWave(t *testing.T) {
	app := loadTestAppWithLibrary(t)
	_, _, err := app.prepareProvingGroundScenario("pg-package-layered-israel-defense")
	if err != nil {
		t.Fatalf("prepareProvingGroundScenario failed: %v", err)
	}
	cases := []struct {
		unitID        string
		loadoutID     string
		targetID      string
		desiredEffect enginev1.DesiredEffect
	}{
		{"pg-pkg-lad-isrstrike-f15i-lead", "deep_strike", "pg-pkg-lad-irn-kheibar-1", enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
		{"pg-pkg-lad-isrstrike-f15i-wing", "deep_strike", "pg-pkg-lad-irn-kheibar-2", enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
		{"pg-pkg-lad-isrstrike-f35i-lead", "internal_strike", "pg-pkg-lad-irn-paveh-1", enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
		{"pg-pkg-lad-isrstrike-f35i-wing", "internal_strike", "pg-pkg-lad-irn-paveh-2", enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
	}
	for _, tc := range cases {
		unit := findScenarioUnit(app.currentScenario.GetUnits(), tc.unitID)
		if unit == nil {
			t.Fatalf("expected %s to exist", tc.unitID)
		}
		if got := unit.GetLoadoutConfigurationId(); got != tc.loadoutID {
			t.Fatalf("expected %s loadout %q, got %q", tc.unitID, tc.loadoutID, got)
		}
		if unit.GetAttackOrder() == nil || unit.GetAttackOrder().GetTargetUnitId() != tc.targetID {
			t.Fatalf("expected %s attack order against %s, got %+v", tc.unitID, tc.targetID, unit.GetAttackOrder())
		}
		if got := unit.GetAttackOrder().GetDesiredEffect(); got != tc.desiredEffect {
			t.Fatalf("expected %s desired effect %v, got %v", tc.unitID, tc.desiredEffect, got)
		}
		target := findScenarioUnit(app.currentScenario.GetUnits(), tc.targetID)
		if target == nil {
			t.Fatalf("expected target %s to exist", tc.targetID)
		}
		decision := sim.EvaluateEngagementDecision(
			unit,
			target,
			app.getCachedDefs(),
			app.getCachedWeaponCatalog(),
			tc.desiredEffect,
			true,
			0,
		)
		if decision.Reason == sim.EngagementReasonNoWeapon || decision.Reason == sim.EngagementReasonDesiredEffectMismatch {
			t.Fatalf("expected %s to have a viable strike setup on %s, got reason=%s", tc.unitID, tc.targetID, decision.Reason)
		}
		if strings.HasPrefix(tc.unitID, "pg-pkg-lad-isrstrike-f15i") && !decision.CanFire {
			t.Fatalf("expected %s to be able to fire on %s, got reason=%s", tc.unitID, tc.targetID, decision.Reason)
		}
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
