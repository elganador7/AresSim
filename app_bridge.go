package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aressim/internal/scenario"

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
	return normalizeRecordIDs(rows), nil
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
	if a.weaponDefRepo == nil {
		return scenario.DefaultWeaponDefinitions()
	}
	rows, err := a.weaponDefRepo.List(a.ctx)
	if err != nil {
		slog.Warn("listWeaponDefsProto: list", "err", err)
		return scenario.DefaultWeaponDefinitions()
	}
	out := make([]*enginev1.WeaponDefinition, 0, len(rows))
	for _, row := range rows {
		out = append(out, weaponDefinitionFromRow(row))
	}
	return out
}

// ListUnitDefinitions returns all unit definitions for the palette/editor.
func (a *App) ListUnitDefinitions() ([]map[string]any, error) {
	if a.unitDefRepo == nil {
		return nil, fmt.Errorf("database not ready")
	}
	rows, err := a.unitDefRepo.List(a.ctx)
	if err != nil {
		return nil, err
	}
	return normalizeRecordIDs(rows), nil
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
