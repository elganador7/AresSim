package ingest

import (
	_ "embed"
	"strings"

	"github.com/aressim/internal/oilnet"
)

//go:embed sample/europe_gem_assets.csv
var europeGEMAssetsCSV string

//go:embed sample/europe_jodi_balance.csv
var europeJODIBalanceCSV string

//go:embed sample/europe_comtrade_flows.csv
var europeComtradeFlowsCSV string

// LoadEuropeRegionalGraph builds a generated Northwest Europe refining/import subgraph.
func LoadEuropeRegionalGraph() (*oilnet.Graph, error) {
	assets, err := ParseGEMAssetsCSV(strings.NewReader(europeGEMAssetsCSV))
	if err != nil {
		return nil, err
	}
	balances, err := ParseJODIBalanceCSV(strings.NewReader(europeJODIBalanceCSV))
	if err != nil {
		return nil, err
	}
	flows, err := ParseComtradeFlowsCSV(strings.NewReader(europeComtradeFlowsCSV))
	if err != nil {
		return nil, err
	}
	return BuildGraph(BuildInput{
		GraphID:     "oilnet-europe-generated",
		Name:        "Oil Network Europe Generated",
		Description: "Generated Northwest Europe refining and import subgraph from normalized sample source exports.",
		Version:     "0.1.0",
		Assets:      assets,
		Balances:    balances,
		TradeFlows:  flows,
		SourceRefs: []oilnet.SourceRef{
			{Name: "GOIT Europe sample", Organization: "Global Energy Monitor", URL: "local-sample", Confidence: 0.9},
			{Name: "JODI Europe sample", Organization: "JODI", URL: "local-sample", Confidence: 0.85},
			{Name: "UN Comtrade Europe sample", Organization: "United Nations", URL: "local-sample", Confidence: 0.8},
		},
	})
}
