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
		"ea18g-growler": {},
		"f15k-slam-eagle": {},
		"kf21-boramae": {},
		"hermes-900-kochav": {},
		"heron-tp-eitan": {},
		"arrow3-battery": {},
		"davids-sling-battery": {},
		"spyder-mr-battery": {},
		"f14a-tomcat-iriaf": {},
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
