package main

import (
	"encoding/base64"
	"fmt"
	"log/slog"
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

// stripTablePrefix removes the "table:" prefix that may arrive on IDs.
func stripTablePrefix(id string) string {
	if idx := strings.LastIndex(id, ":"); idx >= 0 {
		return id[idx+1:]
	}
	return id
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
	data, err := proto.Marshal(msg)
	if err != nil {
		slog.Error("proto marshal for event", "event", eventName, "err", err)
		return
	}
	runtime.EventsEmit(a.ctx, "sim:"+eventName, base64.StdEncoding.EncodeToString(data))
}
