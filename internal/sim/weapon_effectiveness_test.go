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
