package library

import "testing"

func TestLoadAllHasNoDuplicateDefinitionIDs(t *testing.T) {
	defs, err := LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	seen := make(map[string]string)
	for _, def := range defs {
		if def.ID == "" {
			t.Fatalf("definition with empty id: %+v", def)
		}
		if prev, ok := seen[def.ID]; ok {
			t.Fatalf("duplicate definition id %q seen in %q and later entry %q", def.ID, prev, def.Name)
		}
		seen[def.ID] = def.Name
	}
}

func TestCuratedMajorPowerDefinitionsHaveSourceLinks(t *testing.T) {
	defs, err := LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	required := map[string]struct{}{
		"e2d-advanced-hawkeye": {},
		"ea18g-growler":        {},
		"f15k-slam-eagle":      {},
		"kf21-boramae":         {},
		"hermes-900-kochav":    {},
		"heron-tp-eitan":       {},
		"arrow3-battery":       {},
		"davids-sling-battery": {},
		"spyder-mr-battery":    {},
		"f14a-tomcat-iriaf":    {},
	}

	for _, def := range defs {
		if _, ok := required[def.ID]; !ok {
			continue
		}
		if len(def.SourceLinks) == 0 {
			t.Fatalf("definition %q is missing source_links", def.ID)
		}
		delete(required, def.ID)
	}

	for id := range required {
		t.Fatalf("required curated definition %q not found", id)
	}
}

func TestDefinitionToRecordNormalizesSensorSuite(t *testing.T) {
	rec := (Definition{
		Name: "Test Sensor Platform",
		SensorSuite: []SensorDefinition{
			{
				SensorType:   " air_search ",
				MaxRangeM:    120000,
				TargetStates: []string{"airborne", "airborne", "surface"},
				FireControl:  true,
			},
			{
				SensorType:   "unknown",
				MaxRangeM:    5000,
				TargetStates: []string{"land"},
			},
		},
	}).ToRecord()

	suite, ok := rec["sensor_suite"].([]map[string]any)
	if !ok {
		t.Fatalf("expected normalized sensor_suite, got %T", rec["sensor_suite"])
	}
	if len(suite) != 1 {
		t.Fatalf("expected one valid sensor entry, got %d", len(suite))
	}
	if suite[0]["sensor_type"] != "air_search" {
		t.Fatalf("expected normalized sensor type, got %v", suite[0]["sensor_type"])
	}
	targetStates, ok := suite[0]["target_states"].([]string)
	if !ok {
		t.Fatalf("expected normalized target states slice, got %T", suite[0]["target_states"])
	}
	if len(targetStates) != 2 || targetStates[0] != "airborne" || targetStates[1] != "surface" {
		t.Fatalf("unexpected normalized target states: %v", targetStates)
	}
}

func TestIranWarAnchorPlatformsHaveSensorSuites(t *testing.T) {
	defs, err := LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	required := map[string]struct{}{
		"f35i-adir":              {},
		"g550-eitam":             {},
		"saar6-corvette":         {},
		"dolphin-ii-submarine":   {},
		"f35a-lightning":         {},
		"f22a-raptor":            {},
		"e2d-advanced-hawkeye":   {},
		"ddg51-flight-iia":       {},
		"virginia-block-v":       {},
		"patriot-pac3-battery":   {},
		"globaleye-uae":          {},
		"doha-corvette-qatar":    {},
		"s300pmu2-battery-iran":  {},
		"bavar373-battery":       {},
		"third-khordad-battery":  {},
		"f14a-tomcat-iriaf":      {},
		"p3f-orion-mpa-iran":     {},
		"ghadir-midget-submarine": {},
	}

	for _, def := range defs {
		if _, ok := required[def.ID]; !ok {
			continue
		}
		rec := def.ToRecord()
		suite, ok := rec["sensor_suite"].([]map[string]any)
		if !ok || len(suite) == 0 {
			t.Fatalf("definition %q is missing curated sensor_suite", def.ID)
		}
		delete(required, def.ID)
	}

	for id := range required {
		t.Fatalf("required curated sensor-suite definition %q not found", id)
	}
}
