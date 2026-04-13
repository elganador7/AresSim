package sim

import (
	"testing"

	"github.com/aressim/internal/geo"
	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func TestCandidateUnitsInRangePrefiltersFarUnits(t *testing.T) {
	detector := &enginev1.Unit{
		Id:           "detector",
		DefinitionId: "detector",
		TeamId:       "USA",
		Position:     geo.ProtoPosition(25.2854, 51.5310, 0, 0, 0),
	}
	near := &enginev1.Unit{
		Id:           "near",
		DefinitionId: "target",
		TeamId:       "IRN",
		Position:     geo.ProtoPosition(25.30, 51.55, 0, 0, 0),
	}
	far := &enginev1.Unit{
		Id:           "far",
		DefinitionId: "target",
		TeamId:       "IRN",
		Position:     geo.ProtoPosition(35.6895, 139.6917, 0, 0, 0),
	}

	units := []*enginev1.Unit{detector, near, far}
	index := buildUnitH3Index(units)
	candidates := candidateUnitsInRange(detector, 150_000, index, units)
	seen := map[string]bool{}
	for _, unit := range candidates {
		seen[unit.GetId()] = true
	}
	if !seen["near"] {
		t.Fatal("expected nearby unit to remain in candidate set")
	}
	if seen["far"] {
		t.Fatal("expected distant unit to be filtered out of candidate set")
	}
}
