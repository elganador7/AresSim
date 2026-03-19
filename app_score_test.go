package main

import (
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/sim"
)

func TestBuildTeamScoresByDamageState(t *testing.T) {
	units := []*enginev1.Unit{
		{
			Id:           "a1",
			TeamId:       "ISR",
			DefinitionId: "fighter",
			DamageState:  enginev1.DamageState_DAMAGE_STATE_DAMAGED,
		},
		{
			Id:           "a2",
			TeamId:       "ISR",
			DefinitionId: "airbase",
			DamageState:  enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED,
		},
		{
			Id:           "b1",
			TeamId:       "IRN",
			DefinitionId: "sam",
			DamageState:  enginev1.DamageState_DAMAGE_STATE_DESTROYED,
		},
	}
	defs := map[string]sim.DefStats{
		"fighter": {AuthorizedPersonnel: 1, ReplacementCostUSD: 100, StrategicValueUSD: 200, EconomicValueUSD: 0},
		"airbase": {AuthorizedPersonnel: 100, AssetClass: "airbase", ReplacementCostUSD: 1000, StrategicValueUSD: 2000, EconomicValueUSD: 500},
		"sam":     {AuthorizedPersonnel: 10, ReplacementCostUSD: 300, StrategicValueUSD: 400, EconomicValueUSD: 0},
	}

	scores := buildTeamScores(units, defs)
	if len(scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(scores))
	}
	byTeam := map[string]*enginev1.TeamScore{}
	for _, score := range scores {
		byTeam[score.TeamId] = score
	}

	if byTeam["ISR"].ReplacementLossUsd != 625 {
		t.Fatalf("unexpected ISR replacement loss: got %v want 625", byTeam["ISR"].ReplacementLossUsd)
	}
	if byTeam["ISR"].StrategicLossUsd != 1250 {
		t.Fatalf("unexpected ISR strategic loss: got %v want 1250", byTeam["ISR"].StrategicLossUsd)
	}
	if byTeam["ISR"].EconomicLossUsd != 300 {
		t.Fatalf("unexpected ISR economic loss: got %v want 300", byTeam["ISR"].EconomicLossUsd)
	}
	wantHumanISR := 1*0.08*valueOfStatisticalLifeUSD + 100*0.0625*valueOfStatisticalLifeUSD
	if byTeam["ISR"].HumanLossUsd != wantHumanISR {
		t.Fatalf("unexpected ISR human loss: got %v want %v", byTeam["ISR"].HumanLossUsd, wantHumanISR)
	}
	if byTeam["ISR"].TotalLossUsd != 625+1250+300+wantHumanISR {
		t.Fatalf("unexpected ISR total loss: got %v", byTeam["ISR"].TotalLossUsd)
	}

	wantHumanIRN := 10 * 0.75 * valueOfStatisticalLifeUSD
	if byTeam["IRN"].HumanLossUsd != wantHumanIRN || byTeam["IRN"].TotalLossUsd != 700+wantHumanIRN {
		t.Fatalf("unexpected IRN score: %+v", byTeam["IRN"])
	}
}

func TestBatchNeedsScoreUpdate(t *testing.T) {
	if batchNeedsScoreUpdate(&enginev1.BatchUnitUpdate{}) {
		t.Fatal("empty batch should not require score update")
	}
	if !batchNeedsScoreUpdate(&enginev1.BatchUnitUpdate{
		Deltas: []*enginev1.UnitDelta{{UnitId: "u1", DamageState: enginev1.DamageState_DAMAGE_STATE_DAMAGED}},
	}) {
		t.Fatal("damage-state delta should require score update")
	}
}
