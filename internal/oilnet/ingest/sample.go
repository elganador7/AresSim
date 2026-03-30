package ingest

import (
	_ "embed"
	"strings"

	"github.com/aressim/internal/oilnet"
)

//go:embed sample/gem_assets.csv
var sampleGEMAssetsCSV string

//go:embed sample/jodi_balance.csv
var sampleJODIBalanceCSV string

//go:embed sample/comtrade_flows.csv
var sampleComtradeFlowsCSV string

// LoadSampleGraph builds a small graph from normalized source exports.
func LoadSampleGraph() (*oilnet.Graph, error) {
	assets, err := ParseGEMAssetsCSV(strings.NewReader(sampleGEMAssetsCSV))
	if err != nil {
		return nil, err
	}
	balances, err := ParseJODIBalanceCSV(strings.NewReader(sampleJODIBalanceCSV))
	if err != nil {
		return nil, err
	}
	flows, err := ParseComtradeFlowsCSV(strings.NewReader(sampleComtradeFlowsCSV))
	if err != nil {
		return nil, err
	}
	return BuildGraph(BuildInput{
		GraphID:     "oilnet-ingested-sample",
		Name:        "Oil Network Ingested Sample",
		Description: "Small graph built from normalized sample source exports.",
		Version:     "0.1.0",
		Assets:      assets,
		Balances:    balances,
		TradeFlows:  flows,
		SourceRefs: []oilnet.SourceRef{
			{Name: "GOGET sample", Organization: "Global Energy Monitor", URL: "local-sample", Confidence: 0.9},
			{Name: "JODI sample", Organization: "JODI", URL: "local-sample", Confidence: 0.85},
			{Name: "UN Comtrade sample", Organization: "United Nations", URL: "local-sample", Confidence: 0.8},
		},
	})
}
