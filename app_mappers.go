package main

import enginev1 "github.com/aressim/internal/gen/engine/v1"

func normalizeRecordIDs(rows []map[string]any) []map[string]any {
	for _, row := range rows {
		row["id"] = extractRecordID(row["id"])
	}
	return rows
}

func weaponDefinitionRecord(wd *enginev1.WeaponDefinition) map[string]any {
	targets := make([]int, len(wd.DomainTargets))
	for i, d := range wd.DomainTargets {
		targets[i] = int(d)
	}
	return map[string]any{
		"name":               wd.Name,
		"description":        wd.Description,
		"domain_targets":     targets,
		"speed_mps":          float64(wd.SpeedMps),
		"range_m":            float64(wd.RangeM),
		"probability_of_hit": float64(wd.ProbabilityOfHit),
		"guidance":           int(wd.Guidance),
		"effect_type":        int(wd.EffectType),
	}
}

func weaponDefinitionFromRow(row map[string]any) *enginev1.WeaponDefinition {
	wd := &enginev1.WeaponDefinition{
		Id:               extractRecordID(row["id"]),
		Name:             toString(row["name"]),
		Description:      toString(row["description"]),
		SpeedMps:         float32(toFloat64(row["speed_mps"])),
		RangeM:           float32(toFloat64(row["range_m"])),
		ProbabilityOfHit: float32(toFloat64(row["probability_of_hit"])),
		Guidance:         enginev1.GuidanceType(int32(toFloat64(row["guidance"]))),
		EffectType:       enginev1.WeaponEffectType(int32(toFloat64(row["effect_type"]))),
	}
	if targets, ok := row["domain_targets"].([]any); ok {
		for _, item := range targets {
			wd.DomainTargets = append(wd.DomainTargets, enginev1.UnitDomain(int32(toFloat64(item))))
		}
	}
	return wd
}
