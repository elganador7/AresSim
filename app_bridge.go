package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/aressim/internal/library"
	"github.com/aressim/internal/scenario"
	"github.com/aressim/internal/sim"
	"google.golang.org/protobuf/proto"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// BridgeResult is the standard return type for all bridge calls.
type BridgeResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func ok() BridgeResult { return BridgeResult{Success: true} }

func fail(err error) BridgeResult {
	slog.Error("bridge error", "err", err)
	return BridgeResult{Error: err.Error()}
}

func failMsg(msg string) BridgeResult {
	slog.Error("bridge error", "msg", msg)
	return BridgeResult{Error: msg}
}

// GetVersion returns the application version string.
func (a *App) GetVersion() string {
	return "0.1.0-dev"
}

// ListScenarios returns metadata for all stored scenarios.
func (a *App) ListScenarios() ([]map[string]any, error) {
	if a.scenRepo == nil {
		return nil, fmt.Errorf("database not ready")
	}
	rows, err := a.scenRepo.List(a.ctx)
	if err != nil {
		return nil, err
	}
	normalized := normalizeRecordIDs(rows)
	specs := scenario.ProvingGroundSpecs()
	for _, row := range normalized {
		id := toString(row["id"])
		spec, ok := specs[id]
		if !ok {
			continue
		}
		row["scenario_kind"] = "proving_ground"
		row["proving_ground_category"] = spec.Category
		row["proving_ground_purpose"] = spec.Purpose
		row["proving_ground_expected"] = spec.ExpectedSummary
		row["recommended_trials"] = spec.RecommendedTrials
	}
	return normalized, nil
}

// LoadScenarioFromProto accepts a base64-encoded serialized Scenario proto.
func (a *App) LoadScenarioFromProto(protoB64 string) BridgeResult {
	scen, err := decodeScenarioB64(protoB64)
	if err != nil {
		return fail(err)
	}
	a.loadScenario(scen)
	return ok()
}

// SaveScenario persists an edited scenario proto without starting the simulation.
func (a *App) SaveScenario(protoB64 string) BridgeResult {
	scen, err := decodeScenarioB64(protoB64)
	if err != nil {
		return fail(err)
	}
	if a.scenRepo == nil {
		return failMsg("database not ready")
	}
	if err := a.scenRepo.Save(a.ctx, scen.Id, scenarioRecord(scen)); err != nil {
		return fail(err)
	}
	slog.Info("scenario saved", "id", scen.Id, "name", scen.Name)
	return ok()
}

// GetScenario fetches a stored scenario by ID and returns it as base64.
func (a *App) GetScenario(id string) (string, error) {
	if a.scenRepo == nil {
		return "", fmt.Errorf("database not ready")
	}
	rec, err := a.scenRepo.Get(a.ctx, stripTablePrefix(id))
	if err != nil {
		return "", err
	}
	rawAny, ok := rec["scenario_pb"]
	if !ok {
		return "", fmt.Errorf("scenario %s has no proto blob", id)
	}
	var raw []byte
	switch v := rawAny.(type) {
	case []byte:
		raw = v
	case string:
		raw, err = base64.StdEncoding.DecodeString(v)
		if err != nil {
			return "", fmt.Errorf("decode stored proto: %w", err)
		}
	default:
		return "", fmt.Errorf("unexpected scenario_pb type %T", rawAny)
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func (a *App) RunProvingGroundScenario(id string, trials int) (map[string]any, error) {
	_, spec, err := a.prepareProvingGroundScenario(id)
	if err != nil {
		return nil, err
	}
	if trials <= 0 {
		trials = spec.RecommendedTrials
	}
	results := make([]sim.ProvingGroundTrialResult, 0, trials)
	for i := 0; i < trials; i++ {
		if _, _, err := a.prepareProvingGroundScenario(spec.ScenarioID); err != nil {
			return nil, err
		}
		if err := a.applyProvingGroundSetup(spec); err != nil {
			return nil, err
		}
		results = append(results, a.runPreparedProvingGroundTrial(spec, int64(i+1)))
	}
	aggregate := sim.AggregateProvingGroundResults(results, spec.FocusTeam)
	pass := true
	if spec.MinFocusWinRate > 0 && aggregate.FocusWinRate < spec.MinFocusWinRate {
		pass = false
	}
	if spec.MaxFocusWinRate > 0 && aggregate.FocusWinRate > spec.MaxFocusWinRate {
		pass = false
	}
	if spec.MinTargetMissionKillRate > 0 && aggregate.TargetMissionKillRate < spec.MinTargetMissionKillRate {
		pass = false
	}
	if spec.MaxTargetMissionKillRate > 0 && aggregate.TargetMissionKillRate > spec.MaxTargetMissionKillRate {
		pass = false
	}
	if spec.MinInterceptionRate > 0 && aggregate.InterceptionRate < spec.MinInterceptionRate {
		pass = false
	}
	if spec.MaxInterceptionRate > 0 && aggregate.InterceptionRate > spec.MaxInterceptionRate {
		pass = false
	}
	if spec.MinMeanFocusHitsTaken > 0 && aggregate.MeanFocusHitsTaken < spec.MinMeanFocusHitsTaken {
		pass = false
	}
	if spec.MaxMeanFocusHitsTaken > 0 && aggregate.MeanFocusHitsTaken > spec.MaxMeanFocusHitsTaken {
		pass = false
	}
	if spec.MinMeanOpposingLosses > 0 && aggregate.MeanOpposingLosses < spec.MinMeanOpposingLosses {
		pass = false
	}
	if spec.MaxMeanOpposingLosses > 0 && aggregate.MeanOpposingLosses > spec.MaxMeanOpposingLosses {
		pass = false
	}
	return map[string]any{
		"scenarioId":               spec.ScenarioID,
		"category":                 spec.Category,
		"purpose":                  spec.Purpose,
		"expectedSummary":          spec.ExpectedSummary,
		"trials":                   aggregate.Trials,
		"focusTeam":                aggregate.FocusTeam,
		"focusWinRate":             aggregate.FocusWinRate,
		"targetMissionKillRate":    aggregate.TargetMissionKillRate,
		"targetDestroyedRate":      aggregate.TargetDestroyedRate,
		"meanElapsedSeconds":       aggregate.MeanElapsedSeconds,
		"meanFirstShotSeconds":     aggregate.MeanFirstShotSeconds,
		"meanShotsFired":           aggregate.MeanShotsFired,
		"meanHitsScored":           aggregate.MeanHitsScored,
		"meanInterceptions":        aggregate.MeanInterceptions,
		"interceptionRate":         aggregate.InterceptionRate,
		"meanFuelExhaustions":      aggregate.MeanFuelExhaustions,
		"meanReplenishments":       aggregate.MeanReplenishments,
		"meanFocusLosses":          aggregate.MeanFocusLosses,
		"meanOpposingLosses":       aggregate.MeanOpposingLosses,
		"meanFocusHitsTaken":       aggregate.MeanFocusHitsTaken,
		"meanOpposingHitsTaken":    aggregate.MeanOpposingHitsTaken,
		"terminalReasons":          aggregate.TerminalReasons,
		"sampleEvents":             aggregate.SampleEvents,
		"pass":                     pass,
		"minFocusWinRate":          spec.MinFocusWinRate,
		"maxFocusWinRate":          spec.MaxFocusWinRate,
		"minTargetMissionKillRate": spec.MinTargetMissionKillRate,
		"maxTargetMissionKillRate": spec.MaxTargetMissionKillRate,
		"minInterceptionRate":      spec.MinInterceptionRate,
		"maxInterceptionRate":      spec.MaxInterceptionRate,
		"minMeanFocusHitsTaken":    spec.MinMeanFocusHitsTaken,
		"maxMeanFocusHitsTaken":    spec.MaxMeanFocusHitsTaken,
		"minMeanOpposingLosses":    spec.MinMeanOpposingLosses,
		"maxMeanOpposingLosses":    spec.MaxMeanOpposingLosses,
	}, nil
}

// DeleteScenario removes a scenario and its checkpoint history from the database.
func (a *App) DeleteScenario(id string) BridgeResult {
	if a.scenRepo == nil {
		return failMsg("database not ready")
	}
	if err := a.scenRepo.Delete(a.ctx, stripTablePrefix(id)); err != nil {
		return fail(err)
	}
	return ok()
}

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
	defaults := scenario.DefaultWeaponDefinitions()
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
