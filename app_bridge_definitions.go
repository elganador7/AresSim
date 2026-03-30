package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/aressim/internal/library"
	"github.com/aressim/internal/scenario"
	"google.golang.org/protobuf/proto"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// ListWeaponDefinitions returns all weapon definitions for the frontend.
func (a *App) ListWeaponDefinitions() ([]map[string]any, error) {
	if a.weaponDefRepo == nil {
		return nil, fmt.Errorf("database not ready")
	}
	rows, err := a.weaponDefRepo.List(a.ctx)
	if err != nil {
		return nil, err
	}
	return normalizeRecordIDs(rows), nil
}

// listWeaponDefsProto converts DB weapon definition rows into proto messages.
func (a *App) listWeaponDefsProto() []*enginev1.WeaponDefinition {
	defaults, err := scenario.DefaultWeaponDefinitions()
	if err != nil {
		slog.Warn("listWeaponDefsProto: defaults", "err", err)
		defaults = nil
	}
	mergedByID := make(map[string]*enginev1.WeaponDefinition, len(defaults))
	for _, wd := range defaults {
		if wd == nil || strings.TrimSpace(wd.GetId()) == "" {
			continue
		}
		mergedByID[wd.GetId()] = proto.Clone(wd).(*enginev1.WeaponDefinition)
	}
	if a.weaponDefRepo == nil {
		return sortWeaponDefinitionRecords(mergedByID)
	}
	rows, err := a.weaponDefRepo.List(a.ctx)
	if err != nil {
		slog.Warn("listWeaponDefsProto: list", "err", err)
		return sortWeaponDefinitionRecords(mergedByID)
	}
	for _, row := range normalizeRecordIDs(rows) {
		id := toString(row["id"])
		if id == "" {
			continue
		}
		base := mergedByID[id]
		mergedByID[id] = mergeWeaponDefinitionWithRow(base, row)
	}
	return sortWeaponDefinitionRecords(mergedByID)
}

// ListUnitDefinitions returns all unit definitions for the palette/editor.
func (a *App) ListUnitDefinitions() ([]map[string]any, error) {
	defsByID := make(map[string]map[string]any, len(a.libDefsCache))

	for id, def := range a.libDefsCache {
		rec := def.ToRecord()
		rec["id"] = id
		rec["visual_model_id"] = inferVisualModelID(id, rec)
		defsByID[id] = rec
	}

	if a.unitDefRepo == nil {
		if len(defsByID) == 0 {
			return nil, fmt.Errorf("database not ready")
		}
		return sortDefinitionRecords(defsByID), nil
	}

	rows, err := a.unitDefRepo.List(a.ctx)
	if err != nil {
		if len(defsByID) == 0 {
			return nil, err
		}
		slog.Warn("ListUnitDefinitions: db list failed; falling back to library cache", "err", err)
		return sortDefinitionRecords(defsByID), nil
	}

	for _, row := range normalizeRecordIDs(rows) {
		id := toString(row["id"])
		if id == "" {
			continue
		}
		source := toString(row["definition_source"])
		if source == "" && defsByID[id] == nil {
			continue
		}
		if base, ok := defsByID[id]; ok {
			merged := make(map[string]any, len(base)+len(row))
			for key, value := range base {
				merged[key] = value
			}
			for key, value := range row {
				merged[key] = value
			}
			if toFloat64(merged["general_type"]) == 0 && toFloat64(base["general_type"]) != 0 {
				merged["general_type"] = base["general_type"]
			}
			if toFloat64(merged["domain"]) == 0 && toFloat64(base["domain"]) != 0 {
				merged["domain"] = base["domain"]
			}
			if toString(merged["short_name"]) == "" && toString(base["short_name"]) != "" {
				merged["short_name"] = base["short_name"]
			}
			if toString(merged["specific_type"]) == "" && toString(base["specific_type"]) != "" {
				merged["specific_type"] = base["specific_type"]
			}
			if toString(merged["name"]) == "" && toString(base["name"]) != "" {
				merged["name"] = base["name"]
			}
			if toString(merged["visual_model_id"]) == "" && toString(base["visual_model_id"]) != "" {
				merged["visual_model_id"] = base["visual_model_id"]
			}
			if toString(merged["nation_of_origin"]) == "" && toString(base["nation_of_origin"]) != "" {
				merged["nation_of_origin"] = base["nation_of_origin"]
			}
			if employedBy, ok := merged["employed_by"].([]any); ok && len(employedBy) == 0 {
				merged["employed_by"] = base["employed_by"]
			}
			defsByID[id] = merged
			continue
		}
		defsByID[id] = row
	}

	return sortDefinitionRecords(defsByID), nil
}

func inferVisualModelID(id string, rec map[string]any) string {
	raw := strings.TrimSpace(strings.ToLower(id))
	shortName := strings.TrimSpace(strings.ToLower(toString(rec["short_name"])))
	specificType := strings.TrimSpace(strings.ToLower(toString(rec["specific_type"])))
	name := strings.TrimSpace(strings.ToLower(toString(rec["name"])))
	text := strings.Join([]string{raw, shortName, specificType, name}, " ")
	contains := func(parts ...string) bool {
		for _, part := range parts {
			if strings.Contains(text, strings.ToLower(part)) {
				return true
			}
		}
		return false
	}

	switch {
	case contains("ddg-51", "arleigh burke"):
		return "ddg51"
	case contains("f-35"):
		return "f35"
	case contains("f-22"):
		return "f22"
	case contains("f-15"):
		return "f15"
	case contains("f-16"):
		return "f16"
	case contains("e-3", "sentry", "eitam", "globaleye", "aew", "mainstay", "kj-500"):
		return "aew"
	case contains("kc-46", "kc-135", "re'em", "tanker", "yy-20", "il-78"):
		return "tanker"
	case contains("c-17", "c-130", "transport", "y-20"):
		return "transport"
	case contains("patriot"):
		return "patriot"
	case contains("thaad"):
		return "thaad"
	case contains("iron dome"):
		return "iron-dome"
	case contains("david's sling", "davids sling"):
		return "davids-sling"
	case contains("arrow-3"):
		return "arrow-3"
	case contains("arrow-2"):
		return "arrow-2"
	case contains("spyder"):
		return "spyder"
	case contains("barak"):
		return "barak-mx"
	case contains("s-300"):
		return "s300"
	case contains("s-400"):
		return "s400"
	case contains("bavar"):
		return "bavar373"
	case contains("khordad"):
		return "khordad15"
	case contains("tor"):
		return "tor"
	case contains("launcher", "brigade", "regiment", "missile battery", "missile brigade"):
		return "missile-launcher"
	case contains("radar"):
		return "radar-site"
	case contains("frigate"):
		return "frigate"
	case contains("destroyer"):
		return "destroyer"
	case contains("corvette"):
		return "corvette"
	case contains("patrol", "opv"):
		return "patrol-vessel"
	case contains("carrier"):
		return "carrier"
	case contains("littoral combat ship", "lcs"):
		return "lcs"
	case contains("submarine", "ssn", "ssgn", "ssbn"):
		return "submarine"
	case contains("air base", "airbase"):
		return "airbase"
	case contains("port"):
		return "port"
	case contains("operations center", "aoc", "adoc", "c2", "command"):
		return "command-site"
	default:
		return ""
	}
}

// SaveUnitDefinition persists a unit definition from a JSON map.
func (a *App) SaveUnitDefinition(jsonStr string) BridgeResult {
	var rec map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &rec); err != nil {
		return fail(fmt.Errorf("json decode: %w", err))
	}
	id, _ := rec["id"].(string)
	if id == "" {
		return failMsg("unit definition id is required")
	}
	if toString(rec["short_name"]) == "" {
		rec["short_name"] = inferUnitShortName(toString(rec["name"]), toString(rec["specific_type"]))
	}
	if toString(rec["asset_class"]) == "" {
		rec["asset_class"] = "combat_unit"
	}
	if toString(rec["target_class"]) == "" {
		rec["target_class"] = "soft_infrastructure"
	}
	if _, ok := rec["stationary"]; !ok {
		rec["stationary"] = int(toFloat64(rec["form"])) == 34
	}
	if toString(rec["affiliation"]) == "" {
		rec["affiliation"] = "military"
	}
	if toString(rec["employment_role"]) == "" {
		rec["employment_role"] = "dual_use"
	}
	assetClass := toString(rec["asset_class"])
	targetClass := toString(rec["target_class"])
	affiliation := toString(rec["affiliation"])
	employmentRole := toString(rec["employment_role"])
	domain := int(toFloat64(rec["domain"]))
	generalType := int(toFloat64(rec["general_type"]))
	if int(toFloat64(rec["authorized_personnel"])) <= 0 {
		rec["authorized_personnel"] = library.DefaultAuthorizedPersonnel(assetClass, domain, generalType)
	}
	if toFloat64(rec["replacement_cost_usd"]) <= 0 {
		rec["replacement_cost_usd"] = library.DefaultReplacementCostUSD(assetClass, domain, generalType)
	}
	if toFloat64(rec["strategic_value_usd"]) <= 0 {
		rec["strategic_value_usd"] = library.DefaultStrategicValueUSD(assetClass, targetClass, domain, generalType, employmentRole)
	}
	if toFloat64(rec["economic_value_usd"]) <= 0 {
		rec["economic_value_usd"] = library.DefaultEconomicValueUSD(assetClass, affiliation)
	}
	if _, ok := rec["operators"]; !ok {
		if origin := toString(rec["nation_of_origin"]); origin != "" {
			rec["operators"] = []string{origin}
		}
	}
	if _, ok := rec["employed_by"]; !ok {
		if operators, ok := rec["operators"]; ok {
			rec["employed_by"] = operators
		} else if origin := toString(rec["nation_of_origin"]); origin != "" {
			rec["employed_by"] = []string{origin}
		}
	}
	if toString(rec["definition_source"]) == "" {
		rec["definition_source"] = "editor"
	}
	if a.unitDefRepo == nil {
		return failMsg("database not ready")
	}
	if err := a.unitDefRepo.Save(a.ctx, id, rec); err != nil {
		return fail(err)
	}
	a.invalidateDefsCache()
	return ok()
}

// DeleteUnitDefinition removes a unit definition by id.
func (a *App) DeleteUnitDefinition(id string) BridgeResult {
	if a.unitDefRepo == nil {
		return failMsg("database not ready")
	}
	if err := a.unitDefRepo.Delete(a.ctx, stripTablePrefix(id)); err != nil {
		return fail(err)
	}
	a.invalidateDefsCache()
	return ok()
}

func (a *App) SetHumanControlledTeam(teamID string) BridgeResult {
	a.setHumanControlledTeam(teamID)
	return ok()
}

func mergeWeaponDefinitionWithRow(base *enginev1.WeaponDefinition, row map[string]any) *enginev1.WeaponDefinition {
	var merged *enginev1.WeaponDefinition
	if base != nil {
		merged = proto.Clone(base).(*enginev1.WeaponDefinition)
	} else {
		merged = &enginev1.WeaponDefinition{}
	}
	if id := extractRecordID(row["id"]); id != "" {
		merged.Id = id
	}
	if name := toString(row["name"]); name != "" {
		merged.Name = name
	}
	if description := toString(row["description"]); description != "" {
		merged.Description = description
	}
	if speed := float32(toFloat64(row["speed_mps"])); speed > 0 {
		merged.SpeedMps = speed
	}
	if rng := float32(toFloat64(row["range_m"])); rng > 0 {
		merged.RangeM = rng
	}
	if poh := float32(toFloat64(row["probability_of_hit"])); poh > 0 {
		merged.ProbabilityOfHit = poh
	}
	if guidance := enginev1.GuidanceType(int32(toFloat64(row["guidance"]))); guidance != 0 {
		merged.Guidance = guidance
	}
	if effect := enginev1.WeaponEffectType(int32(toFloat64(row["effect_type"]))); effect != enginev1.WeaponEffectType_WEAPON_EFFECT_TYPE_UNSPECIFIED {
		merged.EffectType = effect
	}
	if targets := weaponDomainTargetsFromRow(row["domain_targets"]); len(targets) > 0 {
		merged.DomainTargets = targets
	}
	return merged
}

func weaponDomainTargetsFromRow(raw any) []enginev1.UnitDomain {
	values, ok := raw.([]any)
	if !ok {
		return nil
	}
	targets := make([]enginev1.UnitDomain, 0, len(values))
	for _, item := range values {
		domain := enginev1.UnitDomain(int32(toFloat64(item)))
		if domain == enginev1.UnitDomain_DOMAIN_UNSPECIFIED {
			continue
		}
		targets = append(targets, domain)
	}
	return targets
}

func sortWeaponDefinitionRecords(defsByID map[string]*enginev1.WeaponDefinition) []*enginev1.WeaponDefinition {
	out := make([]*enginev1.WeaponDefinition, 0, len(defsByID))
	for _, wd := range defsByID {
		if wd != nil {
			out = append(out, wd)
		}
	}
	slices.SortFunc(out, func(a, b *enginev1.WeaponDefinition) int {
		return strings.Compare(a.GetName(), b.GetName())
	})
	return out
}
