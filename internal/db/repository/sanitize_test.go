package repository

import "testing"

func TestSanitizeRecord_RemovesNilRecursively(t *testing.T) {
	record := map[string]any{
		"attack_order": nil,
		"parent_unit_id": nil,
		"status": map[string]any{
			"target_unit_id": nil,
			"heading": 90.0,
		},
		"sensor_suite": []any{
			nil,
			map[string]any{
				"sensor_type": "air_search",
				"fire_control": nil,
			},
		},
	}

	got := sanitizeRecord(record)
	if _, ok := got["attack_order"]; ok {
		t.Fatal("expected nil top-level field to be removed")
	}
	if _, ok := got["parent_unit_id"]; ok {
		t.Fatal("expected nil option field to be removed")
	}
	status, ok := got["status"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested map, got %#v", got["status"])
	}
	if _, ok := status["target_unit_id"]; ok {
		t.Fatal("expected nested nil field to be removed")
	}
	suite, ok := got["sensor_suite"].([]any)
	if !ok || len(suite) != 1 {
		t.Fatalf("expected sanitized slice, got %#v", got["sensor_suite"])
	}
}
