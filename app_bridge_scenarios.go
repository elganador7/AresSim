package main

import (
	"encoding/base64"
	"fmt"

	"github.com/aressim/internal/scenario"
	"github.com/aressim/internal/sim"
)

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
