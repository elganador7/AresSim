package main

import (
	"encoding/base64"
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"google.golang.org/protobuf/proto"
)

// ── scenarioRecord ────────────────────────────────────────────────────────────

// TestScenarioRecordFields verifies that scenarioRecord returns all required
// SurrealDB fields and that numeric types are widened to int/float64.
// The original bug was adj_model missing (causing "Expected int but found NONE")
// and float32 values being rejected by TYPE float fields.
func TestScenarioRecordFields(t *testing.T) {
	scen := &enginev1.Scenario{
		Id:             "test-id",
		Name:           "Test Scenario",
		Description:    "desc",
		Classification: "UNCLASSIFIED",
		Author:         "tester",
		StartTimeUnix:  1_748_750_400.0,
		Version:        "1.0.0",
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  2.0,
		},
	}

	rec := scenarioRecord(scen)

	required := []string{
		"name", "description", "author", "classification",
		"start_time_unix", "schema_version",
		"tick_rate_hz", "time_scale", "adj_model",
		"scenario_pb", "last_tick", "last_sim_seconds",
	}
	for _, field := range required {
		if _, ok := rec[field]; !ok {
			t.Errorf("scenarioRecord missing field %q", field)
		}
	}

	// Verify numeric types — SurrealDB SCHEMAFULL rejects float32 for TYPE float
	// and rejects missing values for TYPE int.
	if _, ok := rec["tick_rate_hz"].(float64); !ok {
		t.Errorf("tick_rate_hz must be float64, got %T", rec["tick_rate_hz"])
	}
	if _, ok := rec["time_scale"].(float64); !ok {
		t.Errorf("time_scale must be float64, got %T", rec["time_scale"])
	}
	if _, ok := rec["start_time_unix"].(float64); !ok {
		t.Errorf("start_time_unix must be float64, got %T", rec["start_time_unix"])
	}
	if _, ok := rec["last_sim_seconds"].(float64); !ok {
		t.Errorf("last_sim_seconds must be float64, got %T", rec["last_sim_seconds"])
	}
	if _, ok := rec["adj_model"].(int); !ok {
		t.Errorf("adj_model must be int, got %T", rec["adj_model"])
	}
	if _, ok := rec["last_tick"].(int); !ok {
		t.Errorf("last_tick must be int, got %T", rec["last_tick"])
	}

	// scenario_pb must be a non-empty base64 string.
	pb, ok := rec["scenario_pb"].(string)
	if !ok || pb == "" {
		t.Errorf("scenario_pb must be a non-empty string, got %T %q", rec["scenario_pb"], pb)
	}
	if _, err := base64.StdEncoding.DecodeString(pb); err != nil {
		t.Errorf("scenario_pb is not valid base64: %v", err)
	}
}

func TestScenarioRecordValuesRoundTrip(t *testing.T) {
	scen := &enginev1.Scenario{
		Id:            "round-trip-id",
		Name:          "Round Trip",
		StartTimeUnix: 1_748_750_400.0,
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 5,
			TimeScale:  60.0,
		},
	}

	rec := scenarioRecord(scen)

	if rec["name"] != "Round Trip" {
		t.Errorf("name: want %q got %q", "Round Trip", rec["name"])
	}
	if rec["tick_rate_hz"].(float64) != 5.0 {
		t.Errorf("tick_rate_hz: want 5.0 got %v", rec["tick_rate_hz"])
	}
	if rec["time_scale"].(float64) != 60.0 {
		t.Errorf("time_scale: want 60.0 got %v", rec["time_scale"])
	}

	// The proto blob stored in scenario_pb must decode back to the same scenario.
	raw, _ := base64.StdEncoding.DecodeString(rec["scenario_pb"].(string))
	got := &enginev1.Scenario{}
	if err := proto.Unmarshal(raw, got); err != nil {
		t.Fatalf("unmarshal scenario_pb: %v", err)
	}
	if got.Name != scen.Name {
		t.Errorf("decoded name: want %q got %q", scen.Name, got.Name)
	}
	if got.Settings.TickRateHz != scen.Settings.TickRateHz {
		t.Errorf("decoded tick_rate_hz: want %v got %v", scen.Settings.TickRateHz, got.Settings.TickRateHz)
	}
}

// ── decodeScenarioB64 ─────────────────────────────────────────────────────────

func TestDecodeScenarioB64RoundTrip(t *testing.T) {
	original := &enginev1.Scenario{
		Id:   "decode-test",
		Name: "Decode Test",
		Units: []*enginev1.Unit{
			{Id: "u1", DisplayName: "UNIT-1", Side: "Blue", DefinitionId: "some-def"},
		},
	}

	raw, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(raw)

	decoded, err := decodeScenarioB64(b64)
	if err != nil {
		t.Fatalf("decodeScenarioB64: %v", err)
	}

	if decoded.Id != original.Id {
		t.Errorf("id: want %q got %q", original.Id, decoded.Id)
	}
	if decoded.Name != original.Name {
		t.Errorf("name: want %q got %q", original.Name, decoded.Name)
	}
	if len(decoded.Units) != 1 || decoded.Units[0].DefinitionId != "some-def" {
		t.Errorf("units not preserved: %+v", decoded.Units)
	}
}

func TestDecodeScenarioB64InvalidBase64(t *testing.T) {
	_, err := decodeScenarioB64("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}

func TestDecodeScenarioB64InvalidProto(t *testing.T) {
	garbage := base64.StdEncoding.EncodeToString([]byte{0xFF, 0xFE, 0x00, 0x01})
	_, err := decodeScenarioB64(garbage)
	if err == nil {
		t.Error("expected error for invalid proto bytes, got nil")
	}
}

func TestValidateTransitDeniedByRelationship(t *testing.T) {
	app := &App{
		currentScenario: &enginev1.Scenario{
			Relationships: []*enginev1.CountryRelationship{
				{
					FromCountry:            "ISR",
					ToCountry:              "JOR",
					AirspaceTransitAllowed: false,
				},
			},
		},
	}

	err := app.validateTransit("ISR", enginev1.UnitDomain_DOMAIN_AIR, 31.5, 35.0, 31.4, 36.0)
	if err == nil {
		t.Fatal("expected transit validation error, got nil")
	}
}

func TestValidateStrikeDeniedByRelationship(t *testing.T) {
	app := &App{
		currentScenario: &enginev1.Scenario{
			Relationships: []*enginev1.CountryRelationship{
				{
					FromCountry:            "ISR",
					ToCountry:              "JOR",
					AirspaceTransitAllowed: true,
					AirspaceStrikeAllowed:  false,
				},
			},
		},
	}

	shooter := &enginev1.Unit{
		Id:     "isr-f16",
		TeamId: "ISR",
		Position: &enginev1.Position{
			Lat: 31.5,
			Lon: 35.0,
		},
	}
	target := &enginev1.Unit{
		Id:     "irn-target",
		TeamId: "IRN",
		Position: &enginev1.Position{
			Lat: 31.2,
			Lon: 36.2,
		},
	}

	err := app.validateStrike(shooter, target)
	if err == nil {
		t.Fatal("expected strike validation error, got nil")
	}
}

func TestValidateTransitDeniedByMaritimeRelationship(t *testing.T) {
	app := &App{
		currentScenario: &enginev1.Scenario{
			Relationships: []*enginev1.CountryRelationship{
				{
					FromCountry:            "IRN",
					ToCountry:              "QAT",
					MaritimeTransitAllowed: false,
				},
			},
		},
	}

	err := app.validateTransit("IRN", enginev1.UnitDomain_DOMAIN_SEA, 28.03887, 50.17645, 25.44547, 51.66943)
	if err == nil {
		t.Fatal("expected maritime transit validation error, got nil")
	}
}

func TestSetUnitAttackOrder_ClearsAutoGeneratedMoveOrder(t *testing.T) {
	unit := &enginev1.Unit{
		Id: "isr-f16",
		MoveOrder: &enginev1.MoveOrder{
			Waypoints:     []*enginev1.Waypoint{{Lat: 31.5, Lon: 35.5}},
			AutoGenerated: true,
		},
		AttackOrder: &enginev1.AttackOrder{
			OrderType:    enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET,
			TargetUnitId: "irn-target",
		},
	}
	app := &App{
		currentScenario: &enginev1.Scenario{
			Units: []*enginev1.Unit{unit},
		},
	}

	result := app.SetUnitAttackOrder(unit.Id, 0, "", int32(enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY), 0.7)
	if !result.Success {
		t.Fatalf("expected clear order to succeed, got error %q", result.Error)
	}
	if unit.AttackOrder != nil {
		t.Fatal("expected attack order to be cleared")
	}
	if unit.MoveOrder != nil {
		t.Fatal("expected auto-generated move order to be cleared with attack order")
	}
}

func TestSetUnitAttackOrder_PreservesManualMoveOrderWhenCleared(t *testing.T) {
	unit := &enginev1.Unit{
		Id: "isr-f16",
		MoveOrder: &enginev1.MoveOrder{
			Waypoints:     []*enginev1.Waypoint{{Lat: 31.5, Lon: 35.5}},
			AutoGenerated: false,
		},
		AttackOrder: &enginev1.AttackOrder{
			OrderType:    enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET,
			TargetUnitId: "irn-target",
		},
	}
	app := &App{
		currentScenario: &enginev1.Scenario{
			Units: []*enginev1.Unit{unit},
		},
	}

	result := app.SetUnitAttackOrder(unit.Id, 0, "", int32(enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY), 0.7)
	if !result.Success {
		t.Fatalf("expected clear order to succeed, got error %q", result.Error)
	}
	if unit.MoveOrder == nil || len(unit.MoveOrder.GetWaypoints()) != 1 {
		t.Fatal("expected manual move order to remain in place")
	}
	if unit.MoveOrder.GetAutoGenerated() {
		t.Fatal("expected preserved move order to remain manual")
	}
}

func TestPreviewDraftPlacementDeniedByMaritimeRelationship(t *testing.T) {
	app := &App{}

	preview, err := app.PreviewDraftPlacement(
		"IRN",
		true,
		"offensive",
		`[{"fromCountry":"IRN","toCountry":"QAT","shareIntel":false,"airspaceTransitAllowed":false,"airspaceStrikeAllowed":false,"defensivePositioningAllowed":false,"maritimeTransitAllowed":false,"maritimeStrikeAllowed":false}]`,
		"",
		25.44547,
		51.66943,
	)
	if err != nil {
		t.Fatalf("preview draft placement: %v", err)
	}
	if preview == nil || !preview.Blocked {
		t.Fatal("expected maritime placement to be blocked")
	}
	if preview.Country != "QAT" {
		t.Fatalf("expected QAT maritime owner, got %q", preview.Country)
	}
}

func TestPreviewDraftPlacementAllowsInternationalWaters(t *testing.T) {
	app := &App{}

	preview, err := app.PreviewDraftPlacement(
		"IRN",
		true,
		"offensive",
		"",
		"",
		34.50,
		19.00,
	)
	if err != nil {
		t.Fatalf("preview draft placement: %v", err)
	}
	if preview == nil || preview.Blocked {
		t.Fatal("expected international waters placement to be allowed")
	}
}
