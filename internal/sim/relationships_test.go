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
	if !rule.DefensivePositioningAllowed {
		t.Fatal("expected hostile-country fallback to allow defensive positioning")
	}
	if rule.ShareIntel {
		t.Fatal("expected hostile-country fallback to keep intel sharing disabled")
	}
}

func TestGetRelationshipRuleWithCoalitionsOverridesExplicitRestrictionsForHostileCountries(t *testing.T) {
	rule := GetRelationshipRuleWithCoalitions(
		RelationshipRules{
			"USA": {
				"IRN": {
					ShareIntel:                  false,
					AirspaceTransitAllowed:      false,
					AirspaceStrikeAllowed:       false,
					DefensivePositioningAllowed: false,
					MaritimeTransitAllowed:      false,
					MaritimeStrikeAllowed:       false,
				},
			},
		},
		map[string]string{
			"USA": "BLUE",
			"IRN": "RED",
		},
		"USA",
		"IRN",
	)
	if !rule.AirspaceTransitAllowed || !rule.AirspaceStrikeAllowed {
		t.Fatal("expected hostile-country override to allow air transit and strike")
	}
	if !rule.DefensivePositioningAllowed {
		t.Fatal("expected hostile-country override to allow defensive positioning")
	}
	if !rule.MaritimeTransitAllowed || !rule.MaritimeStrikeAllowed {
		t.Fatal("expected hostile-country override to allow maritime transit and strike")
	}
	if rule.ShareIntel {
		t.Fatal("expected hostile-country override to preserve non-shared intel state")
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
