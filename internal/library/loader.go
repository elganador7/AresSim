// Package library loads unit definition libraries from embedded YAML files
// and from user-supplied YAML files in the app data directory.
//
// File format:
//
//	library:
//	  id: "my-library-v1"
//	  name: "My Library"
//	  ...
//	definitions:
//	  - id: "some-slug"
//	    name: "Some Unit"
//	    ...
//
// Users can create their own libraries by dropping .yaml files into
// <AppDataDir>/libraries/ and sharing those files with others.
package library

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data
var defaultData embed.FS

// Meta describes a library file's header.
type Meta struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
	Author      string `yaml:"author"`
}

// LoadoutSlot describes one weapon type in a definition's default loadout.
type LoadoutSlot struct {
	WeaponID   string `yaml:"weapon_id"`
	MaxQty     int32  `yaml:"max_qty"`
	InitialQty int32  `yaml:"initial_qty"`
}

// Definition is one unit definition record inside a library file.
// Field names use snake_case to match the DB schema and YAML convention.
type Definition struct {
	ID                  string        `yaml:"id"`
	Name                string        `yaml:"name"`
	Description         string        `yaml:"description"`
	Domain              int           `yaml:"domain"`
	Form                int           `yaml:"form"`
	GeneralType         int           `yaml:"general_type"`
	SpecificType        string        `yaml:"specific_type"`
	ShortName           string        `yaml:"short_name"`
	NationOfOrigin      string        `yaml:"nation_of_origin"`
	ServiceEntryYear    int           `yaml:"service_entry_year"`
	BaseStrength        float32       `yaml:"base_strength"`
	Accuracy            float32       `yaml:"accuracy"`
	MaxSpeedMps         float32       `yaml:"max_speed_mps"`
	CruiseSpeedMps      float32       `yaml:"cruise_speed_mps"`
	MaxRangeKm          float32       `yaml:"max_range_km"`
	Survivability       float32       `yaml:"survivability"`
	DetectionRangeM     float32       `yaml:"detection_range_m"`
	RadarCrossSectionM2 float32       `yaml:"radar_cross_section_m2"`
	FuelCapacityLiters  float32       `yaml:"fuel_capacity_liters"`
	FuelBurnRateLph     float32       `yaml:"fuel_burn_rate_lph"`
	DefaultLoadout      []LoadoutSlot `yaml:"default_loadout"`
}

// File is the top-level structure of a library YAML file.
type File struct {
	Library     Meta         `yaml:"library"`
	Definitions []Definition `yaml:"definitions"`
}

// ToRecord converts a Definition to the map shape expected by UnitDefRepo.Save.
// Numeric types are widened to int / float64 to satisfy SurrealDB's SCHEMAFULL
// TYPE int and TYPE float field definitions.
func (d Definition) ToRecord() map[string]any {
	shortName := d.ShortName
	if shortName == "" {
		shortName = inferShortName(d.Name, d.SpecificType)
	}
	return map[string]any{
		"name":                   d.Name,
		"description":            d.Description,
		"domain":                 d.Domain,
		"form":                   d.Form,
		"general_type":           d.GeneralType,
		"specific_type":          d.SpecificType,
		"short_name":             shortName,
		"nation_of_origin":       d.NationOfOrigin,
		"service_entry_year":     d.ServiceEntryYear,
		"base_strength":          float64(d.BaseStrength),
		"accuracy":               float64(d.Accuracy),
		"max_speed_mps":          float64(d.MaxSpeedMps),
		"cruise_speed_mps":       float64(d.CruiseSpeedMps),
		"max_range_km":           float64(d.MaxRangeKm),
		"survivability":          float64(d.Survivability),
		"detection_range_m":      float64(d.DetectionRangeM),
		"radar_cross_section_m2": float64(d.RadarCrossSectionM2),
		"fuel_capacity_liters":   float64(d.FuelCapacityLiters),
		"fuel_burn_rate_lph":     float64(d.FuelBurnRateLph),
	}
}

func inferShortName(name, specificType string) string {
	for _, candidate := range []string{specificType, name} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if parts := strings.Fields(candidate); len(parts) > 0 {
			return strings.ToUpper(parts[0])
		}
	}
	return "UNIT"
}

// LoadAll returns every Definition from:
//  1. Embedded default libraries (data/default/*.yaml)
//  2. User libraries in userLibDir (if the directory exists)
//
// Definitions from user libraries are appended after the defaults,
// so user entries with the same ID will shadow the defaults when the
// caller deduplicates by ID.
func LoadAll(userLibDir string) ([]Definition, error) {
	var all []Definition

	// ── Embedded defaults ─────────────────────────────────────────────────────
	if err := fs.WalkDir(defaultData, "data", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if filepath.Ext(d.Name()) != ".yaml" {
			return nil
		}
		raw, err := defaultData.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}
		defs, name, err := parseFile(raw)
		if err != nil {
			return fmt.Errorf("parse embedded %s: %w", path, err)
		}
		slog.Info("library loaded", "source", "embedded", "name", name, "count", len(defs))
		all = append(all, defs...)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk embedded libraries: %w", err)
	}

	// ── User libraries ─────────────────────────────────────────────────────────
	if userLibDir == "" {
		return all, nil
	}
	entries, err := os.ReadDir(userLibDir)
	if err != nil {
		if os.IsNotExist(err) {
			return all, nil
		}
		return all, fmt.Errorf("read user library dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(userLibDir, entry.Name()))
		if err != nil {
			slog.Warn("library skip", "file", entry.Name(), "err", err)
			continue
		}
		defs, name, err := parseFile(raw)
		if err != nil {
			slog.Warn("library parse error", "file", entry.Name(), "err", err)
			continue
		}
		slog.Info("library loaded", "source", "user", "name", name, "count", len(defs))
		all = append(all, defs...)
	}

	return all, nil
}

func parseFile(raw []byte) ([]Definition, string, error) {
	var lf File
	if err := yaml.Unmarshal(raw, &lf); err != nil {
		return nil, "", err
	}
	return lf.Definitions, lf.Library.Name, nil
}
