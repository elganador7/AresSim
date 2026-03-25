package sim

import (
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func TestWeaponEffectivenessTableLoadsAuthoredData(t *testing.T) {
	got := weaponEffectivenessMultiplier(
		enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_BALLISTIC_STRIKE,
		DefStats{TargetClass: "airbase"},
	)
	if got != 0.95 {
		t.Fatalf("expected authored ballistic-vs-airbase multiplier, got %f", got)
	}
}

func TestEffectiveTargetClassFallsBackByDomain(t *testing.T) {
	got := effectiveTargetClass(DefStats{Domain: enginev1.UnitDomain_DOMAIN_SEA})
	if got != "surface_warship" {
		t.Fatalf("expected sea-domain fallback target class, got %q", got)
	}
}

func TestSelectWeaponForEngagementUsesFallbackTargetClass(t *testing.T) {
	shooter := &enginev1.Unit{
		Id: "shooter",
		Weapons: []*enginev1.WeaponState{{
			WeaponId:   "asm-nsm",
			CurrentQty: 1,
			MaxQty:     1,
		}},
	}
	weaponID, _, _, found := selectWeaponForEngagement(
		shooter,
		DefStats{Domain: enginev1.UnitDomain_DOMAIN_SEA},
		enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY,
		map[string]WeaponStats{
			"asm-nsm": {
				DomainTargets: []enginev1.UnitDomain{enginev1.UnitDomain_DOMAIN_SEA},
				EffectType:    enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_ANTI_SHIP,
				RangeM:        100_000,
			},
		},
	)
	if !found || weaponID != "asm-nsm" {
		t.Fatalf("expected anti-ship weapon to be selected with sea-domain fallback, got found=%v weapon=%q", found, weaponID)
	}
}

func TestLaunchKillProbability_StealthAircraftReduceAntiAirEffectiveness(t *testing.T) {
	shooter := DefStats{Accuracy: 0.62}
	weapon := WeaponStats{
		RangeM:           300_000,
		ProbabilityOfHit: 0.77,
		EffectType:       enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_INTERCEPTOR,
	}
	stealth := DefStats{
		Domain:              enginev1.UnitDomain_DOMAIN_AIR,
		TargetClass:         "aircraft",
		RadarCrossSectionM2: 0.0015,
	}
	conventional := DefStats{
		Domain:              enginev1.UnitDomain_DOMAIN_AIR,
		TargetClass:         "aircraft",
		RadarCrossSectionM2: 5,
	}

	stealthPk := launchKillProbability(shooter, weapon, stealth, 120_000)
	conventionalPk := launchKillProbability(shooter, weapon, conventional, 120_000)
	if stealthPk >= conventionalPk {
		t.Fatalf("expected stealth aircraft to reduce anti-air pk: stealth=%f conventional=%f", stealthPk, conventionalPk)
	}
	if stealthPk <= 0 {
		t.Fatalf("expected reduced but nonzero pk against stealth target, got %f", stealthPk)
	}
}

func TestLaunchKillProbability_LandStrikeIgnoresStealthFactor(t *testing.T) {
	shooter := DefStats{Accuracy: 0.7}
	weapon := WeaponStats{
		RangeM:           300_000,
		ProbabilityOfHit: 0.8,
		EffectType:       enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_LAND_STRIKE,
	}
	lowRCS := DefStats{
		Domain:              enginev1.UnitDomain_DOMAIN_AIR,
		TargetClass:         "aircraft",
		RadarCrossSectionM2: 0.0015,
	}
	highRCS := DefStats{
		Domain:              enginev1.UnitDomain_DOMAIN_AIR,
		TargetClass:         "aircraft",
		RadarCrossSectionM2: 10,
	}

	gotLow := launchKillProbability(shooter, weapon, lowRCS, 100_000)
	gotHigh := launchKillProbability(shooter, weapon, highRCS, 100_000)
	if gotLow != gotHigh {
		t.Fatalf("expected non-air-defense effects to ignore stealth factor: %f vs %f", gotLow, gotHigh)
	}
}
