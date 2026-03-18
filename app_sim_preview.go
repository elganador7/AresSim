package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/aressim/internal/geo"
	"github.com/aressim/internal/sim"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

type PathViolationPreview struct {
	Blocked  bool   `json:"blocked"`
	Country  string `json:"country,omitempty"`
	LegIndex int    `json:"legIndex,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type EffectiveRelationshipPreview struct {
	FromCountry                 string `json:"fromCountry"`
	ToCountry                   string `json:"toCountry"`
	ShareIntel                  bool   `json:"shareIntel"`
	AirspaceTransitAllowed      bool   `json:"airspaceTransitAllowed"`
	AirspaceStrikeAllowed       bool   `json:"airspaceStrikeAllowed"`
	DefensivePositioningAllowed bool   `json:"defensivePositioningAllowed"`
	MaritimeTransitAllowed      bool   `json:"maritimeTransitAllowed"`
	MaritimeStrikeAllowed       bool   `json:"maritimeStrikeAllowed"`
}

func isMaritimeDomain(domain enginev1.UnitDomain) bool {
	return domain == enginev1.UnitDomain_DOMAIN_SEA || domain == enginev1.UnitDomain_DOMAIN_SUBSURFACE
}

type draftRelationshipInput struct {
	FromCountry                 string `json:"fromCountry"`
	ToCountry                   string `json:"toCountry"`
	ShareIntel                  bool   `json:"shareIntel"`
	AirspaceTransitAllowed      bool   `json:"airspaceTransitAllowed"`
	AirspaceStrikeAllowed       bool   `json:"airspaceStrikeAllowed"`
	DefensivePositioningAllowed bool   `json:"defensivePositioningAllowed"`
	MaritimeTransitAllowed      bool   `json:"maritimeTransitAllowed"`
	MaritimeStrikeAllowed       bool   `json:"maritimeStrikeAllowed"`
}

type draftPointInput struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func buildEffectiveRelationships(countries []string, rules sim.RelationshipRules, countryCoalitions map[string]string) []EffectiveRelationshipPreview {
	normalized := make([]string, 0, len(countries))
	seen := make(map[string]bool, len(countries))
	for _, country := range countries {
		code := geo.CountryCode(country)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		normalized = append(normalized, code)
	}
	slices.Sort(normalized)

	previews := make([]EffectiveRelationshipPreview, 0, len(normalized)*len(normalized))
	for _, from := range normalized {
		for _, to := range normalized {
			if from == to {
				continue
			}
			rule := sim.GetRelationshipRuleWithCoalitions(rules, countryCoalitions, from, to)
			previews = append(previews, EffectiveRelationshipPreview{
				FromCountry:                 from,
				ToCountry:                   to,
				ShareIntel:                  rule.ShareIntel,
				AirspaceTransitAllowed:      rule.AirspaceTransitAllowed,
				AirspaceStrikeAllowed:       rule.AirspaceStrikeAllowed,
				DefensivePositioningAllowed: rule.DefensivePositioningAllowed,
				MaritimeTransitAllowed:      rule.MaritimeTransitAllowed,
				MaritimeStrikeAllowed:       rule.MaritimeStrikeAllowed,
			})
		}
	}
	return previews
}

func currentScenarioCountries(scen *enginev1.Scenario) []string {
	if scen == nil {
		return nil
	}
	countries := make([]string, 0, len(scen.GetUnits())+len(scen.GetRelationships())*2)
	for _, unit := range scen.GetUnits() {
		if unit == nil {
			continue
		}
		if code := sim.CountryDisplayCode(unit.GetTeamId()); code != "" {
			countries = append(countries, code)
		}
	}
	for _, relationship := range scen.GetRelationships() {
		if relationship == nil {
			continue
		}
		if code := geo.CountryCode(relationship.GetFromCountry()); code != "" {
			countries = append(countries, code)
		}
		if code := geo.CountryCode(relationship.GetToCountry()); code != "" {
			countries = append(countries, code)
		}
	}
	return countries
}

func previewTransitPath(ownerCountry string, maritime bool, points []geo.Point, rules sim.RelationshipRules, countryCoalitions map[string]string) *PathViolationPreview {
	ownerCountry = sim.CountryDisplayCode(ownerCountry)
	if ownerCountry == "" || len(points) < 2 {
		return nil
	}
	for idx, segment := range geo.SamplePath(points) {
		var countries []string
		if maritime {
			countries = segment.SeaZoneOwners
		} else {
			countries = segment.AirspaceOwners
		}
		for _, country := range countries {
			if country == "" || country == ownerCountry {
				continue
			}
			rule := sim.GetRelationshipRuleWithCoalitions(rules, countryCoalitions, ownerCountry, country)
			allowed := rule.AirspaceTransitAllowed
			reasonFmt := "%s cannot transit %s airspace"
			if maritime {
				allowed = rule.MaritimeTransitAllowed
				reasonFmt = "%s cannot transit %s territorial waters"
			}
			if !allowed {
				return &PathViolationPreview{
					Blocked:  true,
					Country:  country,
					LegIndex: idx + 1,
					Reason:   fmt.Sprintf(reasonFmt, ownerCountry, country),
				}
			}
		}
	}
	return nil
}

func previewStrikePath(ownerCountry string, maritime bool, points []geo.Point, rules sim.RelationshipRules, countryCoalitions map[string]string) *PathViolationPreview {
	ownerCountry = sim.CountryDisplayCode(ownerCountry)
	if ownerCountry == "" || len(points) < 2 {
		return nil
	}
	for idx, segment := range geo.SamplePath(points) {
		var countries []string
		if maritime {
			countries = segment.SeaZoneOwners
		} else {
			countries = segment.AirspaceOwners
		}
		for _, country := range countries {
			if country == "" || country == ownerCountry {
				continue
			}
			rule := sim.GetRelationshipRuleWithCoalitions(rules, countryCoalitions, ownerCountry, country)
			transitAllowed := rule.AirspaceTransitAllowed
			strikeAllowed := rule.AirspaceStrikeAllowed
			transitFmt := "%s cannot transit %s airspace"
			strikeFmt := "%s cannot conduct strike operations in %s airspace"
			if maritime {
				transitAllowed = rule.MaritimeTransitAllowed
				strikeAllowed = rule.MaritimeStrikeAllowed
				transitFmt = "%s cannot transit %s territorial waters"
				strikeFmt = "%s cannot conduct strike operations in %s territorial waters"
			}
			if !transitAllowed {
				return &PathViolationPreview{
					Blocked:  true,
					Country:  country,
					LegIndex: idx + 1,
					Reason:   fmt.Sprintf(transitFmt, ownerCountry, country),
				}
			}
			if !strikeAllowed {
				return &PathViolationPreview{
					Blocked:  true,
					Country:  country,
					LegIndex: idx + 1,
					Reason:   fmt.Sprintf(strikeFmt, ownerCountry, country),
				}
			}
		}
	}
	return nil
}

func (a *App) validateTransit(ownerCountry string, domain enginev1.UnitDomain, startLat, startLon, endLat, endLon float64) error {
	if violation := previewTransitPath(
		ownerCountry,
		isMaritimeDomain(domain),
		[]geo.Point{{Lat: startLat, Lon: startLon}, {Lat: endLat, Lon: endLon}},
		a.relationshipRules(),
		a.countryCoalitions(),
	); violation != nil {
		return fmt.Errorf("%s", violation.Reason)
	}
	return nil
}

func (a *App) validateStrike(shooter, target *enginev1.Unit) error {
	if shooter == nil || target == nil {
		return nil
	}
	ownerCountry := unitCountryCode(shooter)
	if ownerCountry == "" {
		return nil
	}
	domain := a.getCachedDefs()[shooter.DefinitionId].Domain
	points := [][2]float64{{shooter.GetPosition().GetLat(), shooter.GetPosition().GetLon()}}
	for _, wp := range shooter.GetMoveOrder().GetWaypoints() {
		points = append(points, [2]float64{wp.GetLat(), wp.GetLon()})
	}
	points = append(points, [2]float64{target.GetPosition().GetLat(), target.GetPosition().GetLon()})
	pathPoints := make([]geo.Point, 0, len(points))
	for _, point := range points {
		pathPoints = append(pathPoints, geo.Point{Lat: point[0], Lon: point[1]})
	}
	if violation := previewStrikePath(ownerCountry, isMaritimeDomain(domain), pathPoints, a.relationshipRules(), a.countryCoalitions()); violation != nil {
		return fmt.Errorf("%s", violation.Reason)
	}
	return nil
}

func (a *App) PreviewCurrentTransitPath(unitID string) (*PathViolationPreview, error) {
	if a.currentScenario == nil {
		return nil, fmt.Errorf("no scenario loaded")
	}
	var unit *enginev1.Unit
	for _, candidate := range a.currentScenario.GetUnits() {
		if candidate.GetId() == unitID {
			unit = candidate
			break
		}
	}
	if unit == nil {
		return nil, fmt.Errorf("unit %s not found", unitID)
	}
	if unit.GetMoveOrder() == nil || len(unit.GetMoveOrder().GetWaypoints()) == 0 {
		return &PathViolationPreview{Blocked: false}, nil
	}
	domain := a.getCachedDefs()[unit.DefinitionId].Domain
	points := make([]geo.Point, 0, len(unit.GetMoveOrder().GetWaypoints())+1)
	points = append(points, geo.Point{Lat: unit.GetPosition().GetLat(), Lon: unit.GetPosition().GetLon()})
	for _, wp := range unit.GetMoveOrder().GetWaypoints() {
		points = append(points, geo.Point{Lat: wp.GetLat(), Lon: wp.GetLon()})
	}
	violation := previewTransitPath(unitCountryCode(unit), isMaritimeDomain(domain), points, a.relationshipRules(), a.countryCoalitions())
	if violation == nil {
		return &PathViolationPreview{Blocked: false}, nil
	}
	return violation, nil
}

func (a *App) PreviewCurrentStrikePath(unitID string) (*PathViolationPreview, error) {
	if a.currentScenario == nil {
		return nil, fmt.Errorf("no scenario loaded")
	}
	var unit *enginev1.Unit
	var target *enginev1.Unit
	for _, candidate := range a.currentScenario.GetUnits() {
		if candidate.GetId() == unitID {
			unit = candidate
		}
	}
	if unit == nil {
		return nil, fmt.Errorf("unit %s not found", unitID)
	}
	if unit.GetAttackOrder() == nil || unit.GetAttackOrder().GetTargetUnitId() == "" {
		return &PathViolationPreview{Blocked: false}, nil
	}
	domain := a.getCachedDefs()[unit.DefinitionId].Domain
	for _, candidate := range a.currentScenario.GetUnits() {
		if candidate.GetId() == unit.GetAttackOrder().GetTargetUnitId() {
			target = candidate
			break
		}
	}
	if target == nil {
		return &PathViolationPreview{
			Blocked: true,
			Reason:  "Assigned target is no longer available.",
		}, nil
	}
	points := make([]geo.Point, 0, len(unit.GetMoveOrder().GetWaypoints())+2)
	points = append(points, geo.Point{Lat: unit.GetPosition().GetLat(), Lon: unit.GetPosition().GetLon()})
	for _, wp := range unit.GetMoveOrder().GetWaypoints() {
		points = append(points, geo.Point{Lat: wp.GetLat(), Lon: wp.GetLon()})
	}
	points = append(points, geo.Point{Lat: target.GetPosition().GetLat(), Lon: target.GetPosition().GetLon()})
	violation := previewStrikePath(unitCountryCode(unit), isMaritimeDomain(domain), points, a.relationshipRules(), a.countryCoalitions())
	if violation == nil {
		return &PathViolationPreview{Blocked: false}, nil
	}
	return violation, nil
}

func parseDraftRelationshipRules(relationshipsJSON string) (sim.RelationshipRules, error) {
	if strings.TrimSpace(relationshipsJSON) == "" {
		return nil, nil
	}
	var rels []draftRelationshipInput
	if err := json.Unmarshal([]byte(relationshipsJSON), &rels); err != nil {
		return nil, fmt.Errorf("decode relationships: %w", err)
	}
	protoRels := make([]*enginev1.CountryRelationship, 0, len(rels))
	for _, rel := range rels {
		protoRels = append(protoRels, &enginev1.CountryRelationship{
			FromCountry:                 rel.FromCountry,
			ToCountry:                   rel.ToCountry,
			ShareIntel:                  rel.ShareIntel,
			AirspaceTransitAllowed:      rel.AirspaceTransitAllowed,
			AirspaceStrikeAllowed:       rel.AirspaceStrikeAllowed,
			DefensivePositioningAllowed: rel.DefensivePositioningAllowed,
			MaritimeTransitAllowed:      rel.MaritimeTransitAllowed,
			MaritimeStrikeAllowed:       rel.MaritimeStrikeAllowed,
		})
	}
	return sim.BuildRelationshipRules(protoRels), nil
}

func parseDraftCountryCoalitions(countryCoalitionsJSON string) (map[string]string, error) {
	if strings.TrimSpace(countryCoalitionsJSON) == "" {
		return nil, nil
	}
	var raw map[string]string
	if err := json.Unmarshal([]byte(countryCoalitionsJSON), &raw); err != nil {
		return nil, fmt.Errorf("decode country coalitions: %w", err)
	}
	normalized := make(map[string]string, len(raw))
	for country, coalition := range raw {
		country = geo.CountryCode(country)
		coalition = geo.CountryCode(coalition)
		if country == "" || coalition == "" {
			continue
		}
		normalized[country] = coalition
	}
	return normalized, nil
}

func parseDraftPoints(pointsJSON string) ([]geo.Point, error) {
	if strings.TrimSpace(pointsJSON) == "" {
		return nil, nil
	}
	var raw []draftPointInput
	if err := json.Unmarshal([]byte(pointsJSON), &raw); err != nil {
		return nil, fmt.Errorf("decode points: %w", err)
	}
	points := make([]geo.Point, 0, len(raw))
	for _, point := range raw {
		points = append(points, geo.Point{Lat: point.Lat, Lon: point.Lon})
	}
	return points, nil
}

func parseDraftCountries(countriesJSON string) ([]string, error) {
	if strings.TrimSpace(countriesJSON) == "" {
		return nil, nil
	}
	var countries []string
	if err := json.Unmarshal([]byte(countriesJSON), &countries); err != nil {
		return nil, fmt.Errorf("decode countries: %w", err)
	}
	return countries, nil
}

func (a *App) PreviewDraftTransitPath(ownerCountry string, maritime bool, relationshipsJSON, countryCoalitionsJSON, pointsJSON string) (*PathViolationPreview, error) {
	rules, err := parseDraftRelationshipRules(relationshipsJSON)
	if err != nil {
		return nil, err
	}
	countryCoalitions, err := parseDraftCountryCoalitions(countryCoalitionsJSON)
	if err != nil {
		return nil, err
	}
	points, err := parseDraftPoints(pointsJSON)
	if err != nil {
		return nil, err
	}
	violation := previewTransitPath(ownerCountry, maritime, points, rules, countryCoalitions)
	if violation == nil {
		return &PathViolationPreview{Blocked: false}, nil
	}
	return violation, nil
}

func (a *App) PreviewDraftStrikePath(ownerCountry string, maritime bool, relationshipsJSON, countryCoalitionsJSON, pointsJSON string) (*PathViolationPreview, error) {
	rules, err := parseDraftRelationshipRules(relationshipsJSON)
	if err != nil {
		return nil, err
	}
	countryCoalitions, err := parseDraftCountryCoalitions(countryCoalitionsJSON)
	if err != nil {
		return nil, err
	}
	points, err := parseDraftPoints(pointsJSON)
	if err != nil {
		return nil, err
	}
	violation := previewStrikePath(ownerCountry, maritime, points, rules, countryCoalitions)
	if violation == nil {
		return &PathViolationPreview{Blocked: false}, nil
	}
	return violation, nil
}

func (a *App) PreviewDraftPlacement(ownerCountry string, maritime bool, employmentRole, relationshipsJSON, countryCoalitionsJSON string, lat, lon float64) (*PathViolationPreview, error) {
	ownerCountry = geo.CountryCode(ownerCountry)
	if ownerCountry == "" {
		return &PathViolationPreview{Blocked: false}, nil
	}
	rules, err := parseDraftRelationshipRules(relationshipsJSON)
	if err != nil {
		return nil, err
	}
	countryCoalitions, err := parseDraftCountryCoalitions(countryCoalitionsJSON)
	if err != nil {
		return nil, err
	}
	ctx := geo.LookupPoint(geo.Point{Lat: lat, Lon: lon})
	hostCountry := ctx.AirspaceOwner
	hostLabel := "airspace"
	if maritime {
		hostCountry = ctx.SeaZoneOwner
		hostLabel = "territorial waters"
	}
	if hostCountry == "" || hostCountry == ownerCountry {
		return &PathViolationPreview{Blocked: false}, nil
	}
	rule := sim.GetRelationshipRuleWithCoalitions(rules, countryCoalitions, ownerCountry, hostCountry)
	role := strings.TrimSpace(strings.ToLower(employmentRole))
	if role == "defensive" {
		if !rule.DefensivePositioningAllowed {
			return &PathViolationPreview{
				Blocked: true,
				Country: hostCountry,
				Reason:  fmt.Sprintf("%s cannot position defensive assets inside %s %s", ownerCountry, hostCountry, hostLabel),
			}, nil
		}
		return &PathViolationPreview{Blocked: false}, nil
	}
	offensiveAllowed := rule.AirspaceStrikeAllowed
	if maritime {
		offensiveAllowed = rule.MaritimeStrikeAllowed
	}
	if !offensiveAllowed {
		return &PathViolationPreview{
			Blocked: true,
			Country: hostCountry,
			Reason:  fmt.Sprintf("%s cannot position offensive-capable assets inside %s %s", ownerCountry, hostCountry, hostLabel),
		}, nil
	}
	return &PathViolationPreview{Blocked: false}, nil
}

func (a *App) PreviewCurrentRelationships() ([]EffectiveRelationshipPreview, error) {
	if a.currentScenario == nil {
		return nil, fmt.Errorf("no scenario loaded")
	}
	return buildEffectiveRelationships(
		currentScenarioCountries(a.currentScenario),
		a.relationshipRules(),
		a.countryCoalitions(),
	), nil
}

func (a *App) PreviewDraftRelationships(relationshipsJSON, countryCoalitionsJSON, countriesJSON string) ([]EffectiveRelationshipPreview, error) {
	rules, err := parseDraftRelationshipRules(relationshipsJSON)
	if err != nil {
		return nil, err
	}
	countryCoalitions, err := parseDraftCountryCoalitions(countryCoalitionsJSON)
	if err != nil {
		return nil, err
	}
	countries, err := parseDraftCountries(countriesJSON)
	if err != nil {
		return nil, err
	}
	return buildEffectiveRelationships(countries, rules, countryCoalitions), nil
}
