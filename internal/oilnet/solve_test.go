package oilnet

import "testing"

func TestSimulateShock_NodeOutageReducesThroughput(t *testing.T) {
	base, err := LoadGlobalBaseline()
	if err != nil {
		t.Fatalf("LoadGlobalBaseline() error = %v", err)
	}
	result, err := SimulateShock(base, ShockRequest{
		OfflineNodeIDs: []string{"om-hormuz"},
	})
	if err != nil {
		t.Fatalf("SimulateShock() error = %v", err)
	}
	var found bool
	for _, impact := range result.ImpactedComponents {
		if impact.ID == "om-hormuz" {
			found = true
			if impact.CurrentFlowBPD != 0 {
				t.Fatalf("expected offline chokepoint flow 0, got %v", impact.CurrentFlowBPD)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected chokepoint outage to appear in impacted components")
	}
	var crudeLoss float64
	for _, summary := range result.CommoditySummaries {
		if summary.Commodity == CommodityCrude {
			crudeLoss = summary.LostFlowBPD
			break
		}
	}
	if crudeLoss <= 0 {
		t.Fatalf("expected crude loss after Hormuz outage, got %v", crudeLoss)
	}
}

func TestSimulateShock_RefineryOutageReducesDerivativeOutputs(t *testing.T) {
	base, err := LoadGlobalBaseline()
	if err != nil {
		t.Fatalf("LoadGlobalBaseline() error = %v", err)
	}
	result, err := SimulateShock(base, ShockRequest{
		OfflineNodeIDs: []string{"kw-mina-al-ahmadi"},
	})
	if err != nil {
		t.Fatalf("SimulateShock() error = %v", err)
	}
	var diesel CommoditySummary
	found := false
	for _, summary := range result.CommoditySummaries {
		if summary.Commodity == CommodityDiesel {
			diesel = summary
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected diesel summary")
	}
	if diesel.CurrentRefinedOutBPD >= diesel.BaselineRefinedOutBPD {
		t.Fatalf("expected refinery outage to reduce diesel output, baseline=%v current=%v", diesel.BaselineRefinedOutBPD, diesel.CurrentRefinedOutBPD)
	}
}

func TestSimulateShockHorizon_UsesStorageDrawdown(t *testing.T) {
	base, err := LoadGlobalBaseline()
	if err != nil {
		t.Fatalf("LoadGlobalBaseline() error = %v", err)
	}
	result, err := SimulateShockHorizon(base, HorizonRequest{
		ShockRequest: ShockRequest{
			OfflineNodeIDs: []string{"om-hormuz"},
		},
		Days: 2,
	})
	if err != nil {
		t.Fatalf("SimulateShockHorizon() error = %v", err)
	}
	if len(result.Days) != 2 {
		t.Fatalf("expected 2 horizon days, got %d", len(result.Days))
	}
	initialJetDemand := 0.0
	for _, summary := range result.Initial.CommoditySummaries {
		if summary.Commodity == CommodityJet {
			initialJetDemand = summary.ServedDemandBPD
			break
		}
	}
	dayOneJetDemand := 0.0
	for _, summary := range result.Days[0].CommoditySummaries {
		if summary.Commodity == CommodityJet {
			dayOneJetDemand = summary.ServedDemandBPD
			break
		}
	}
	if dayOneJetDemand < initialJetDemand {
		t.Fatalf("expected storage drawdown to preserve or improve served jet demand, initial=%v day1=%v", initialJetDemand, dayOneJetDemand)
	}
}

func TestSimulateShock_MultiChokepointEdgeHonorsAllConstraints(t *testing.T) {
	base := &Graph{
		ID: "test",
		Nodes: []Node{
			{ID: "from", Name: "From", Kind: NodeExportTerminal, CountryCode: "SAU", State: StateOperational, CurrentFlowBPD: 100, CapacityBPD: 100},
			{ID: "to", Name: "To", Kind: NodeDemandCenter, CountryCode: "NLD", State: StateOperational, CurrentFlowBPD: 100, CapacityBPD: 100, DemandProfile: []CommodityQuantity{{Commodity: CommodityCrude, BPD: 100}}},
			{ID: "om-hormuz", Name: "Hormuz", Kind: NodeChokepoint, CountryCode: "OMN", State: StateOperational},
			{ID: "eg-suez", Name: "Suez", Kind: NodeChokepoint, CountryCode: "EGY", State: StateOperational},
		},
		Edges: []Edge{
			{
				ID:                 "ship-1",
				Name:               "Ship",
				Kind:               EdgeShipping,
				FromNodeID:         "from",
				ToNodeID:           "to",
				Commodity:          CommodityCrude,
				State:              StateOperational,
				CapacityBPD:        100,
				CurrentFlowBPD:     100,
				CrossesChokepoints: []string{"om-hormuz", "eg-suez"},
			},
		},
	}
	result, err := SimulateShock(base, ShockRequest{OfflineNodeIDs: []string{"eg-suez"}})
	if err != nil {
		t.Fatalf("SimulateShock() error = %v", err)
	}
	if got := result.Graph.Edges[0].CurrentFlowBPD; got != 0 {
		t.Fatalf("expected Suez outage to zero multi-chokepoint edge, got %v", got)
	}
}
