package sim

import "testing"

func TestGetRelationshipRuleWithCoalitionsDefaultsHostileCountriesToTransitAndStrike(t *testing.T) {
	rule := GetRelationshipRuleWithCoalitions(
		nil,
		map[string]string{
			"ISR": "BLUE",
			"IRN": "RED",
		},
		"ISR",
		"IRN",
	)
	if !rule.AirspaceTransitAllowed {
		t.Fatal("expected hostile-country fallback to allow transit")
	}
	if !rule.AirspaceStrikeAllowed {
		t.Fatal("expected hostile-country fallback to allow strike")
	}
	if !rule.MaritimeTransitAllowed || !rule.MaritimeStrikeAllowed {
		t.Fatal("expected hostile-country fallback to allow maritime transit and strike")
	}
	if rule.DefensivePositioningAllowed {
		t.Fatal("expected hostile-country fallback to keep defensive positioning disabled")
	}
	if rule.ShareIntel {
		t.Fatal("expected hostile-country fallback to keep intel sharing disabled")
	}
}

func TestGetRelationshipRuleWithCoalitionsKeepsNeutralFallbackClosed(t *testing.T) {
	rule := GetRelationshipRuleWithCoalitions(
		nil,
		map[string]string{
			"USA": "BLUE",
			"QAT": "BLUE",
		},
		"USA",
		"QAT",
	)
	if rule.AirspaceTransitAllowed || rule.AirspaceStrikeAllowed || rule.DefensivePositioningAllowed || rule.ShareIntel || rule.MaritimeTransitAllowed || rule.MaritimeStrikeAllowed {
		t.Fatal("expected same-coalition fallback with no explicit relationship to remain closed")
	}
}
