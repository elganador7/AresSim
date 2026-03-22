package main

import (
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

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
