package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"

	"github.com/surrealdb/surrealdb.go/pkg/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"google.golang.org/protobuf/proto"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// toFloat64 converts a numeric any value (from SurrealDB row) to float64.
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	}
	return 0
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// extractRecordID converts a SurrealDB id field to a plain string.
func extractRecordID(v any) string {
	switch rid := v.(type) {
	case models.RecordID:
		return fmt.Sprintf("%v", rid.ID)
	case *models.RecordID:
		return fmt.Sprintf("%v", rid.ID)
	}
	s := fmt.Sprintf("%v", v)
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

var shortNamePattern = regexp.MustCompile(`(?i)([a-z]+[-/]?\d+[a-z0-9/-]*)`)

func inferUnitShortName(name, specificType string) string {
	for _, candidate := range []string{specificType, name} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if match := shortNamePattern.FindStringSubmatch(candidate); len(match) > 1 {
			return strings.ToUpper(match[1])
		}
		fields := strings.Fields(candidate)
		if len(fields) > 0 {
			return strings.ToUpper(fields[0])
		}
	}
	return "UNIT"
}

// stripTablePrefix removes the "table:" prefix that may arrive on IDs.
func stripTablePrefix(id string) string {
	if idx := strings.LastIndex(id, ":"); idx >= 0 {
		return id[idx+1:]
	}
	return id
}

func sortDefinitionRecords(defsByID map[string]map[string]any) []map[string]any {
	rows := make([]map[string]any, 0, len(defsByID))
	for _, row := range defsByID {
		rows = append(rows, row)
	}
	slices.SortFunc(rows, func(a, b map[string]any) int {
		if diff := cmpFloat64(toFloat64(a["domain"]), toFloat64(b["domain"])); diff != 0 {
			return diff
		}
		if diff := cmpFloat64(toFloat64(a["general_type"]), toFloat64(b["general_type"])); diff != 0 {
			return diff
		}
		return strings.Compare(toString(a["name"]), toString(b["name"]))
	})
	return rows
}

func cmpFloat64(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// scenarioRecord builds the SurrealDB map for a Scenario proto.
func scenarioRecord(scen *enginev1.Scenario) map[string]any {
	raw, _ := proto.Marshal(scen)
	return map[string]any{
		"name":             scen.Name,
		"description":      scen.Description,
		"author":           scen.Author,
		"classification":   scen.Classification,
		"start_time_unix":  scen.StartTimeUnix,
		"schema_version":   scen.Version,
		"tick_rate_hz":     float64(scen.GetSettings().GetTickRateHz()),
		"time_scale":       float64(scen.GetSettings().GetTimeScale()),
		"adj_model":        0,
		"scenario_pb":      base64.StdEncoding.EncodeToString(raw),
		"last_tick":        0,
		"last_sim_seconds": 0.0,
	}
}

func unitRecord(u *enginev1.Unit) map[string]any {
	if u == nil {
		return nil
	}
	pos := u.GetPosition()
	status := u.GetStatus()
	combatEffects := status.GetCombatEffects()
	record := map[string]any{
		"id":               models.RecordID{Table: "unit", ID: u.GetId()},
		"display_name":     u.GetDisplayName(),
		"full_name":        u.GetFullName(),
		"team_id":          strings.TrimSpace(u.GetTeamId()),
		"coalition_id":     strings.TrimSpace(u.GetCoalitionId()),
		"nato_symbol_sidc": u.GetNatoSymbolSidc(),
		"definition_id":    strings.TrimSpace(u.GetDefinitionId()),
		"posture":          int32(u.GetPosture()),
		"position": map[string]any{
			"type":        "Point",
			"coordinates": []float64{pos.GetLon(), pos.GetLat()},
		},
		"alt_msl":                    pos.GetAltMsl(),
		"heading":                    pos.GetHeading(),
		"speed":                      pos.GetSpeed(),
		"personnel_strength":         status.GetPersonnelStrength(),
		"equipment_strength":         status.GetEquipmentStrength(),
		"combat_effectiveness":       status.GetCombatEffectiveness(),
		"fuel_level_liters":          status.GetFuelLevelLiters(),
		"morale":                     status.GetMorale(),
		"fatigue":                    status.GetFatigue(),
		"is_active":                  status.GetIsActive(),
		"suppressed":                 combatEffects.GetSuppressed(),
		"disrupted":                  combatEffects.GetDisrupted(),
		"routing":                    combatEffects.GetRouting(),
		"parent_unit_id":             emptyToNil(u.GetParentUnitId()),
		"damage_state":               int32(u.GetDamageState()),
		"engagement_behavior":        int32(u.GetEngagementBehavior()),
		"engagement_pkill_threshold": u.GetEngagementPkillThreshold(),
		"attack_order":               attackOrderRecord(u.GetAttackOrder()),
	}
	return record
}

func attackOrderRecord(order *enginev1.AttackOrder) map[string]any {
	if order == nil {
		return nil
	}
	return map[string]any{
		"order_type":      int32(order.GetOrderType()),
		"target_unit_id":  order.GetTargetUnitId(),
		"desired_effect":  int32(order.GetDesiredEffect()),
		"pkill_threshold": order.GetPkillThreshold(),
	}
}

func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// decodeScenarioB64 decodes a base64-encoded proto Scenario binary.
func decodeScenarioB64(b64 string) (*enginev1.Scenario, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	scen := &enginev1.Scenario{}
	if err := proto.Unmarshal(raw, scen); err != nil {
		return nil, fmt.Errorf("proto unmarshal: %w", err)
	}
	return scen, nil
}

// emitProtoEvent marshals a proto message and emits it to the frontend.
func (a *App) emitProtoEvent(eventName string, msg proto.Message) {
	if a == nil || a.ctx == nil || a.ctx == context.Background() || a.ctx == context.TODO() {
		return
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		slog.Error("proto marshal for event", "event", eventName, "err", err)
		return
	}
	runtime.EventsEmit(a.ctx, "sim:"+eventName, base64.StdEncoding.EncodeToString(data))
}
