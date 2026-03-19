package library

import "testing"

func TestDefinitionToRecordAppliesDefaultValuation(t *testing.T) {
	rec := (Definition{
		ID:             "airbase-test",
		Name:           "Test Airbase",
		Domain:         1,
		Form:           34,
		GeneralType:    93,
		AssetClass:     "airbase",
		TargetClass:    "runway",
		Affiliation:    "military",
		EmploymentRole: "defensive",
	}).ToRecord()

	if got := rec["replacement_cost_usd"].(float64); got <= 0 {
		t.Fatalf("expected replacement cost default, got %v", got)
	}
	if got := rec["strategic_value_usd"].(float64); got <= 0 {
		t.Fatalf("expected strategic value default, got %v", got)
	}
	if got := rec["economic_value_usd"].(float64); got <= 0 {
		t.Fatalf("expected economic value default, got %v", got)
	}
	if got := rec["authorized_personnel"].(int); got <= 0 {
		t.Fatalf("expected authorized personnel default, got %v", got)
	}
}
