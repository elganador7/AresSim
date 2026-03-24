package sim

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

//go:embed data/weapon_effectiveness.json
var weaponEffectivenessJSON []byte

type targetClassEffectivenessKey struct {
	effectType  enginev1.WeaponEffectType
	targetClass string
}

type weaponEffectivenessEntry struct {
	EffectType  string  `json:"effectType"`
	TargetClass string  `json:"targetClass"`
	Multiplier  float64 `json:"multiplier"`
}

var (
	weaponTargetEffectiveness     map[targetClassEffectivenessKey]float64
	weaponTargetEffectivenessOnce sync.Once
)

func loadWeaponTargetEffectiveness() map[targetClassEffectivenessKey]float64 {
	weaponTargetEffectivenessOnce.Do(func() {
		var entries []weaponEffectivenessEntry
		if err := json.Unmarshal(weaponEffectivenessJSON, &entries); err != nil {
			panic(err)
		}
		weaponTargetEffectiveness = make(map[targetClassEffectivenessKey]float64, len(entries))
		for _, entry := range entries {
			effectType, err := parseWeaponEffectType(entry.EffectType)
			if err != nil {
				panic(err)
			}
			targetClass := strings.TrimSpace(strings.ToLower(entry.TargetClass))
			if targetClass == "" {
				panic("weapon effectiveness entry has empty targetClass")
			}
			weaponTargetEffectiveness[targetClassEffectivenessKey{
				effectType:  effectType,
				targetClass: targetClass,
			}] = entry.Multiplier
		}
	})
	return weaponTargetEffectiveness
}

func parseWeaponEffectType(raw string) (enginev1.WeaponEffectType, error) {
	key := strings.TrimSpace(strings.ToUpper(raw))
	if key == "" {
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_UNSPECIFIED, fmt.Errorf("empty weapon effect type")
	}
	if !strings.HasPrefix(key, "WEAPON_EFFECT_TYPE_") {
		key = "WEAPON_EFFECT_TYPE_" + key
	}
	value, ok := enginev1.WeaponEffectType_value[key]
	if !ok {
		return enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_UNSPECIFIED, fmt.Errorf("unknown weapon effect type %q", raw)
	}
	return enginev1.WeaponEffectType(value), nil
}

func launchKillProbability(shooterDef DefStats, weapon WeaponStats, targetDef DefStats, distM float64) float64 {
	if weapon.RangeM <= 0 || distM > weapon.RangeM {
		return 0
	}
	base := weapon.ProbabilityOfHit * shooterAccuracyFactor(shooterDef) * weaponEffectivenessMultiplier(weapon.EffectType, targetDef)
	if base <= 0 {
		return 0
	}
	rangeFactor := 1.0 - 0.75*(distM/weapon.RangeM)
	if rangeFactor < 0.25 {
		rangeFactor = 0.25
	}
	return base * rangeFactor
}

func shooterAccuracyFactor(def DefStats) float64 {
	if def.Accuracy <= 0 {
		return 1.0
	}
	factor := 0.35 + 0.65*def.Accuracy
	if factor < 0.4 {
		return 0.4
	}
	if factor > 1.0 {
		return 1.0
	}
	return factor
}

func effectiveTargetClass(targetDef DefStats) string {
	targetClass := strings.TrimSpace(targetDef.TargetClass)
	if targetClass == "" {
		targetClass = strings.TrimSpace(targetDef.AssetClass)
	}
	if targetClass == "" {
		switch targetDef.Domain {
		case enginev1.UnitDomain_DOMAIN_AIR:
			targetClass = "aircraft"
		case enginev1.UnitDomain_DOMAIN_SEA:
			targetClass = "surface_warship"
		case enginev1.UnitDomain_DOMAIN_SUBSURFACE:
			targetClass = "submarine"
		case enginev1.UnitDomain_DOMAIN_LAND:
			targetClass = "soft_infrastructure"
		default:
			targetClass = strings.ToLower(targetDef.Domain.String())
		}
	}
	return strings.ToLower(targetClass)
}

func weaponEffectivenessMultiplier(effectType enginev1.WeaponEffectType, targetDef DefStats) float64 {
	targetClass := effectiveTargetClass(targetDef)
	if v, ok := loadWeaponTargetEffectiveness()[targetClassEffectivenessKey{effectType: effectType, targetClass: targetClass}]; ok {
		return v
	}
	if outcome := resolveImpactOutcome(effectType, targetClass); outcome != outcomeNoEffect {
		return 0.75
	}
	return 0
}
