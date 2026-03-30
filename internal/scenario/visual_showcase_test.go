package scenario

import "testing"

func TestDestroyerVisualScaleScenarioBuildsRequestedGrid(t *testing.T) {
	scen := DestroyerVisualScaleScenario(100, 10, 5)
	if scen.GetId() != "visual-showcase-ddg51-wall" {
		t.Fatalf("unexpected scenario id %q", scen.GetId())
	}
	if got := len(scen.GetUnits()); got != 100 {
		t.Fatalf("expected 100 units, got %d", got)
	}
	if scen.GetUnits()[0].GetDefinitionId() != "ddg51-flight-iia" {
		t.Fatalf("expected DDG-51 definition, got %q", scen.GetUnits()[0].GetDefinitionId())
	}
	if scen.GetUnits()[0].GetPosition().GetSpeed() != 0 {
		t.Fatalf("expected showcase ships to be stationary")
	}
}

