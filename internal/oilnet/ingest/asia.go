package ingest

import (
	_ "embed"
	"strings"

	"github.com/aressim/internal/oilnet"
)

//go:embed sample/asia_gem_assets.csv
var asiaGEMAssetsCSV string

//go:embed sample/asia_jodi_balance.csv
var asiaJODIBalanceCSV string

//go:embed sample/asia_comtrade_flows.csv
var asiaComtradeFlowsCSV string

// LoadAsiaRegionalGraph builds a generated Northeast Asia demand/import subgraph.
func LoadAsiaRegionalGraph() (*oilnet.Graph, error) {
	assets, err := ParseGEMAssetsCSV(strings.NewReader(asiaGEMAssetsCSV))
	if err != nil {
		return nil, err
	}
	balances, err := ParseJODIBalanceCSV(strings.NewReader(asiaJODIBalanceCSV))
	if err != nil {
		return nil, err
	}
	flows, err := ParseComtradeFlowsCSV(strings.NewReader(asiaComtradeFlowsCSV))
	if err != nil {
		return nil, err
	}
	return BuildGraph(BuildInput{
		GraphID:     "oilnet-asia-generated",
		Name:        "Oil Network Northeast Asia Generated",
		Description: "Generated Northeast Asia demand and import corridor subgraph from normalized sample source exports.",
		Version:     "0.1.0",
		Assets:      assets,
		Balances:    balances,
		TradeFlows:  flows,
		SourceRefs: []oilnet.SourceRef{
			{Name: "GOIT Asia sample", Organization: "Global Energy Monitor", URL: "local-sample", Confidence: 0.9},
			{Name: "JODI Asia sample", Organization: "JODI", URL: "local-sample", Confidence: 0.85},
			{Name: "UN Comtrade Asia sample", Organization: "United Nations", URL: "local-sample", Confidence: 0.8},
		},
	})
}
