package main

import enginev1 "github.com/aressim/internal/gen/engine/v1"

func normalizeRecordIDs(rows []map[string]any) []map[string]any {
	for _, row := range rows {
		row["id"] = extractRecordID(row["id"])
	}
	return rows
}

func unitDefinitionRecord(unitDef *enginev1.UnitDefinition) map[string]any {
	return map[string]any{
		"name":                 unitDef.Name,
		"description":          unitDef.Description,
		"domain":               int(unitDef.Domain),
		"form":                 int(unitDef.Form),
		"general_type":         int(unitDef.GeneralType),
		"specific_type":        unitDef.SpecificType,
		"nation_of_origin":     unitDef.NationOfOrigin,
		"service_entry_year":   int(unitDef.ServiceEntryYear),
		"base_strength":        float64(unitDef.BaseStrength),
		"accuracy":             float64(unitDef.Accuracy),
		"max_speed_mps":        float64(unitDef.MaxSpeedMps),
		"cruise_speed_mps":     float64(unitDef.CruiseSpeedMps),
		"max_range_km":         float64(unitDef.MaxRangeKm),
		"survivability":        float64(unitDef.Survivability),
		"detection_range_m":    float64(unitDef.DetectionRangeM),
		"fuel_capacity_liters": float64(unitDef.FuelCapacityLiters),
		"fuel_burn_rate_lph":   float64(unitDef.FuelBurnRateLph),
	}
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
	}
	if targets, ok := row["domain_targets"].([]any); ok {
		for _, item := range targets {
			wd.DomainTargets = append(wd.DomainTargets, enginev1.UnitDomain(int32(toFloat64(item))))
		}
	}
	return wd
}
