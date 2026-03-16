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

// WeaponConfiguration is a named mission loadout for a platform.
type WeaponConfiguration struct {
	ID          string        `yaml:"id"`
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Loadout     []LoadoutSlot `yaml:"loadout"`
}

// Definition is one unit definition record inside a library file.
// Field names use snake_case to match the DB schema and YAML convention.
type Definition struct {
	ID                               string                `yaml:"id"`
	Name                             string                `yaml:"name"`
	Description                      string                `yaml:"description"`
	Domain                           int                   `yaml:"domain"`
	Form                             int                   `yaml:"form"`
	GeneralType                      int                   `yaml:"general_type"`
	SpecificType                     string                `yaml:"specific_type"`
	ShortName                        string                `yaml:"short_name"`
	AssetClass                       string                `yaml:"asset_class"`
	TargetClass                      string                `yaml:"target_class"`
	Stationary                       bool                  `yaml:"stationary"`
	Affiliation                      string                `yaml:"affiliation"`
	EmploymentRole                   string                `yaml:"employment_role"`
	NationOfOrigin                   string                `yaml:"nation_of_origin"`
	Operators                        []string              `yaml:"operators"`
	EmployedBy                       []string              `yaml:"employed_by"`
	ServiceEntryYear                 int                   `yaml:"service_entry_year"`
	BaseStrength                     float32               `yaml:"base_strength"`
	Accuracy                         float32               `yaml:"accuracy"`
	MaxSpeedMps                      float32               `yaml:"max_speed_mps"`
	CruiseSpeedMps                   float32               `yaml:"cruise_speed_mps"`
	MaxRangeKm                       float32               `yaml:"max_range_km"`
	Survivability                    float32               `yaml:"survivability"`
	DetectionRangeM                  float32               `yaml:"detection_range_m"`
	RadarCrossSectionM2              float32               `yaml:"radar_cross_section_m2"`
	FuelCapacityLiters               float32               `yaml:"fuel_capacity_liters"`
	FuelBurnRateLph                  float32               `yaml:"fuel_burn_rate_lph"`
	EmbarkedFixedWingCapacity        int                   `yaml:"embarked_fixed_wing_capacity"`
	EmbarkedRotaryWingCapacity       int                   `yaml:"embarked_rotary_wing_capacity"`
	EmbarkedUavCapacity              int                   `yaml:"embarked_uav_capacity"`
	EmbarkedSurfaceConnectorCapacity int                   `yaml:"embarked_surface_connector_capacity"`
	LaunchCapacityPerInterval        int                   `yaml:"launch_capacity_per_interval"`
	RecoveryCapacityPerInterval      int                   `yaml:"recovery_capacity_per_interval"`
	SortieIntervalMinutes            int                   `yaml:"sortie_interval_minutes"`
	DefaultLoadout                   []LoadoutSlot         `yaml:"default_loadout"`
	DefaultWeaponConfiguration       string                `yaml:"default_weapon_configuration"`
	WeaponConfigurations             []WeaponConfiguration `yaml:"weapon_configurations"`
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
	defaultConfigID, configs := normalizeWeaponConfigurations(d)
	operators := normalizeCountryCodes(d.Operators, d.NationOfOrigin)
	employedBy := normalizeCountryCodes(d.EmployedBy, operators...)
	assetClass := normalizeAssetClass(d.AssetClass)
	targetClass := normalizeTargetClass(d.TargetClass, d.Domain, d.GeneralType, assetClass)
	affiliation := normalizeAffiliation(d.Affiliation, assetClass)
	employmentRole := normalizeEmploymentRole(d.EmploymentRole, assetClass, d.GeneralType)
	stationary := normalizeStationary(d.Stationary, d.Form, assetClass)
	return map[string]any{
		"name":                                d.Name,
		"description":                         d.Description,
		"domain":                              d.Domain,
		"form":                                d.Form,
		"general_type":                        d.GeneralType,
		"specific_type":                       d.SpecificType,
		"short_name":                          shortName,
		"asset_class":                         assetClass,
		"target_class":                        targetClass,
		"stationary":                          stationary,
		"affiliation":                         affiliation,
		"employment_role":                     employmentRole,
		"definition_source":                   "library",
		"nation_of_origin":                    d.NationOfOrigin,
		"operators":                           operators,
		"employed_by":                         employedBy,
		"service_entry_year":                  d.ServiceEntryYear,
		"base_strength":                       float64(d.BaseStrength),
		"accuracy":                            float64(d.Accuracy),
		"max_speed_mps":                       float64(d.MaxSpeedMps),
		"cruise_speed_mps":                    float64(d.CruiseSpeedMps),
		"max_range_km":                        float64(d.MaxRangeKm),
		"survivability":                       float64(d.Survivability),
		"detection_range_m":                   float64(d.DetectionRangeM),
		"radar_cross_section_m2":              float64(d.RadarCrossSectionM2),
		"fuel_capacity_liters":                float64(d.FuelCapacityLiters),
		"fuel_burn_rate_lph":                  float64(d.FuelBurnRateLph),
		"embarked_fixed_wing_capacity":        d.EmbarkedFixedWingCapacity,
		"embarked_rotary_wing_capacity":       d.EmbarkedRotaryWingCapacity,
		"embarked_uav_capacity":               d.EmbarkedUavCapacity,
		"embarked_surface_connector_capacity": d.EmbarkedSurfaceConnectorCapacity,
		"launch_capacity_per_interval":        d.LaunchCapacityPerInterval,
		"recovery_capacity_per_interval":      d.RecoveryCapacityPerInterval,
		"sortie_interval_minutes":             d.SortieIntervalMinutes,
		"default_weapon_configuration":        defaultConfigID,
		"weapon_configurations":               configs,
	}
}

func normalizeEmploymentRole(v, assetClass string, generalType int) string {
	v = strings.TrimSpace(v)
	if v != "" {
		return v
	}
	switch assetClass {
	case "airbase", "port", "oil_field", "pipeline_node", "desalination_plant", "power_plant", "radar_site", "c2_site":
		return "defensive"
	}
	switch generalType {
	case 73:
		return "defensive"
	case 31, 32, 51, 52, 72:
		return "offensive"
	default:
		return "dual_use"
	}
}

func normalizeAssetClass(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "combat_unit"
	}
	return v
}

func normalizeAffiliation(v, assetClass string) string {
	v = strings.TrimSpace(v)
	if v != "" {
		return v
	}
	switch assetClass {
	case "oil_field", "pipeline_node", "desalination_plant", "power_plant":
		return "civilian"
	case "port":
		return "dual_use"
	default:
		return "military"
	}
}

func normalizeTargetClass(v string, domain, generalType int, assetClass string) string {
	v = strings.TrimSpace(v)
	if v != "" {
		return v
	}
	switch assetClass {
	case "airbase":
		return "runway"
	case "port":
		return "soft_infrastructure"
	case "oil_field", "pipeline_node", "power_plant":
		return "civilian_energy"
	case "desalination_plant":
		return "civilian_water"
	case "radar_site":
		return "sam_battery"
	case "c2_site":
		return "hardened_infrastructure"
	}
	switch domain {
	case 2:
		return "aircraft"
	case 3:
		return "surface_warship"
	case 4:
		return "submarine"
	case 1:
		switch generalType {
		case 60, 61, 62, 63:
			return "armor"
		case 73:
			return "sam_battery"
		default:
			return "soft_infrastructure"
		}
	default:
		return "soft_infrastructure"
	}
}

func normalizeStationary(v bool, form int, assetClass string) bool {
	if v {
		return true
	}
	if form == 34 {
		return true
	}
	switch assetClass {
	case "airbase", "port", "oil_field", "pipeline_node", "desalination_plant", "power_plant", "radar_site", "c2_site":
		return true
	default:
		return false
	}
}

func normalizeCountryCodes(codes []string, fallbacks ...string) []string {
	seen := make(map[string]bool)
	var normalized []string
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		normalized = append(normalized, code)
	}
	for _, fallback := range fallbacks {
		fallback = strings.TrimSpace(fallback)
		if fallback == "" || seen[fallback] {
			continue
		}
		seen[fallback] = true
		normalized = append(normalized, fallback)
	}
	return normalized
}

func normalizeWeaponConfigurations(d Definition) (string, []map[string]any) {
	configs := d.WeaponConfigurations
	if len(configs) == 0 && len(d.DefaultLoadout) > 0 {
		configs = []WeaponConfiguration{{
			ID:          "default",
			Name:        "Default",
			Description: "Standard baseline loadout.",
			Loadout:     d.DefaultLoadout,
		}}
	}

	normalized := make([]map[string]any, 0, len(configs))
	for i, cfg := range configs {
		id := strings.TrimSpace(cfg.ID)
		if id == "" {
			if i == 0 {
				id = "default"
			} else {
				id = fmt.Sprintf("config_%d", i+1)
			}
		}
		name := strings.TrimSpace(cfg.Name)
		if name == "" {
			name = humanizeConfigID(id)
		}
		loadout := make([]map[string]any, 0, len(cfg.Loadout))
		for _, slot := range cfg.Loadout {
			loadout = append(loadout, map[string]any{
				"weapon_id":   slot.WeaponID,
				"max_qty":     int(slot.MaxQty),
				"initial_qty": int(slot.InitialQty),
			})
		}
		normalized = append(normalized, map[string]any{
			"id":          id,
			"name":        name,
			"description": cfg.Description,
			"loadout":     loadout,
		})
	}

	defaultID := strings.TrimSpace(d.DefaultWeaponConfiguration)
	if defaultID == "" && len(normalized) > 0 {
		defaultID = fmt.Sprintf("%v", normalized[0]["id"])
	}
	return defaultID, normalized
}

func humanizeConfigID(id string) string {
	parts := strings.Fields(strings.NewReplacer("_", " ", "-", " ").Replace(id))
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
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
//  1. Embedded default libraries (data/default/**/*.yaml)
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
	if err := filepath.WalkDir(userLibDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".yaml" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("library skip", "file", path, "err", err)
			return nil
		}
		defs, name, err := parseFile(raw)
		if err != nil {
			slog.Warn("library parse error", "file", path, "err", err)
			return nil
		}
		slog.Info("library loaded", "source", "user", "name", name, "count", len(defs))
		all = append(all, defs...)
		return nil
	}); err != nil {
		if os.IsNotExist(err) {
			return all, nil
		}
		return all, fmt.Errorf("walk user library dir: %w", err)
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
