package ingest

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/aressim/internal/oilnet"
)

// GEMAssetRow is a normalized topology row from Global Energy Monitor exports.
type GEMAssetRow struct {
	ID          string
	Name        string
	AssetType   string
	CountryCode string
	Operator    string
	Lat         float64
	Lon         float64
	CapacityBPD float64
	Commodity   oilnet.Commodity
}

// JODIBalanceRow is a normalized country/day balance row.
type JODIBalanceRow struct {
	CountryCode       string
	Commodity         oilnet.Commodity
	ProductionBPD     float64
	RefineryIntakeBPD float64
	DemandBPD         float64
	ImportsBPD        float64
	ExportsBPD        float64
}

// ComtradeFlowRow is a normalized bilateral trade corridor row.
type ComtradeFlowRow struct {
	ReporterCode string
	PartnerCode  string
	Commodity    oilnet.Commodity
	FlowBPD      float64
}

func ParseGEMAssetsCSV(r io.Reader) ([]GEMAssetRow, error) {
	rows, err := parseCSV(r)
	if err != nil {
		return nil, err
	}
	out := make([]GEMAssetRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, GEMAssetRow{
			ID:          row["id"],
			Name:        row["name"],
			AssetType:   row["asset_type"],
			CountryCode: row["country_code"],
			Operator:    row["operator"],
			Lat:         parseFloat(row["lat"]),
			Lon:         parseFloat(row["lon"]),
			CapacityBPD: parseFloat(row["capacity_bpd"]),
			Commodity:   oilnet.Commodity(strings.TrimSpace(row["commodity"])),
		})
	}
	return out, nil
}

func ParseJODIBalanceCSV(r io.Reader) ([]JODIBalanceRow, error) {
	rows, err := parseCSV(r)
	if err != nil {
		return nil, err
	}
	out := make([]JODIBalanceRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, JODIBalanceRow{
			CountryCode:       row["country_code"],
			Commodity:         oilnet.Commodity(strings.TrimSpace(row["commodity"])),
			ProductionBPD:     parseFloat(row["production_bpd"]),
			RefineryIntakeBPD: parseFloat(row["refinery_intake_bpd"]),
			DemandBPD:         parseFloat(row["demand_bpd"]),
			ImportsBPD:        parseFloat(row["imports_bpd"]),
			ExportsBPD:        parseFloat(row["exports_bpd"]),
		})
	}
	return out, nil
}

func ParseComtradeFlowsCSV(r io.Reader) ([]ComtradeFlowRow, error) {
	rows, err := parseCSV(r)
	if err != nil {
		return nil, err
	}
	out := make([]ComtradeFlowRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, ComtradeFlowRow{
			ReporterCode: row["reporter_code"],
			PartnerCode:  row["partner_code"],
			Commodity:    oilnet.Commodity(strings.TrimSpace(row["commodity"])),
			FlowBPD:      parseFloat(row["flow_bpd"]),
		})
	}
	return out, nil
}

func parseCSV(r io.Reader) ([]map[string]string, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}
	for i := range header {
		header[i] = strings.TrimSpace(strings.ToLower(header[i]))
	}
	out := make([]map[string]string, 0)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read csv row: %w", err)
		}
		row := make(map[string]string, len(header))
		for i, key := range header {
			if i < len(record) {
				row[key] = strings.TrimSpace(record[i])
			}
		}
		out = append(out, row)
	}
}

func parseFloat(raw string) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0
	}
	return value
}
