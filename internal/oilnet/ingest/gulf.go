package ingest

import (
	_ "embed"
	"strings"

	"github.com/aressim/internal/oilnet"
)

//go:embed sample/gulf_gem_assets.csv
var gulfGEMAssetsCSV string

//go:embed sample/gulf_jodi_balance.csv
var gulfJODIBalanceCSV string

//go:embed sample/gulf_comtrade_flows.csv
var gulfComtradeFlowsCSV string

// LoadGulfRegionalGraph builds a richer Gulf subgraph from normalized source exports.
func LoadGulfRegionalGraph() (*oilnet.Graph, error) {
	assets, err := ParseGEMAssetsCSV(strings.NewReader(gulfGEMAssetsCSV))
	if err != nil {
		return nil, err
	}
	balances, err := ParseJODIBalanceCSV(strings.NewReader(gulfJODIBalanceCSV))
	if err != nil {
		return nil, err
	}
	flows, err := ParseComtradeFlowsCSV(strings.NewReader(gulfComtradeFlowsCSV))
	if err != nil {
		return nil, err
	}
	return BuildGraph(BuildInput{
		GraphID:     "oilnet-gulf-generated",
		Name:        "Oil Network Gulf Generated",
		Description: "Generated Gulf petroleum subgraph from normalized sample source exports.",
		Version:     "0.1.0",
		Assets:      assets,
		Balances:    balances,
		TradeFlows:  flows,
		SourceRefs: []oilnet.SourceRef{
			{Name: "GOGET/GOIT Gulf sample", Organization: "Global Energy Monitor", URL: "local-sample", Confidence: 0.9},
			{Name: "JODI Gulf sample", Organization: "JODI", URL: "local-sample", Confidence: 0.85},
			{Name: "UN Comtrade Gulf sample", Organization: "United Nations", URL: "local-sample", Confidence: 0.8},
		},
	})
}
