package sim

import (
	"strings"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

type CountryRelationshipRule struct {
	ShareIntel                 bool
	AirspaceTransitAllowed     bool
	AirspaceStrikeAllowed      bool
	DefensivePositioningAllowed bool
	MaritimeTransitAllowed     bool
	MaritimeStrikeAllowed      bool
}

type RelationshipRules map[string]map[string]CountryRelationshipRule

type DetectionContactInfo struct {
	UnitID     string
	SourceTeam string
	Shared     bool
}

func BuildCountryCoalitions(units []*enginev1.Unit) map[string]string {
	coalitions := make(map[string]string)
	for _, unit := range units {
		if unit == nil {
			continue
		}
		country := CountryDisplayCode(unit.GetTeamId())
		coalition := strings.TrimSpace(strings.ToUpper(unit.GetCoalitionId()))
		if country == "" || coalition == "" {
			continue
		}
		if _, exists := coalitions[country]; !exists {
			coalitions[country] = coalition
		}
	}
	return coalitions
}

func BuildRelationshipRules(relationships []*enginev1.CountryRelationship) RelationshipRules {
	rules := make(RelationshipRules)
	for _, rel := range relationships {
		if rel == nil {
			continue
		}
		from := strings.TrimSpace(strings.ToUpper(rel.GetFromCountry()))
		to := strings.TrimSpace(strings.ToUpper(rel.GetToCountry()))
		if from == "" || to == "" {
			continue
		}
		if rules[from] == nil {
			rules[from] = make(map[string]CountryRelationshipRule)
		}
		rules[from][to] = CountryRelationshipRule{
			ShareIntel:                  rel.GetShareIntel(),
			AirspaceTransitAllowed:      rel.GetAirspaceTransitAllowed(),
			AirspaceStrikeAllowed:       rel.GetAirspaceStrikeAllowed(),
			DefensivePositioningAllowed: rel.GetDefensivePositioningAllowed(),
			MaritimeTransitAllowed:      rel.GetMaritimeTransitAllowed(),
			MaritimeStrikeAllowed:       rel.GetMaritimeStrikeAllowed(),
		}
	}
	return rules
}

func ApplyIntelSharing(base DetectionSet, rules RelationshipRules) DetectionSet {
	if len(base) == 0 || len(rules) == 0 {
		return base
	}

	shared := make(map[string]map[string]bool, len(base))
	for team, ids := range base {
		if shared[team] == nil {
			shared[team] = make(map[string]bool)
		}
		for _, id := range ids {
			shared[team][id] = true
		}
	}

	for from, recipients := range rules {
		sourceIDs := base[from]
		if len(sourceIDs) == 0 {
			continue
		}
		for to, rule := range recipients {
			if !rule.ShareIntel {
				continue
			}
			if shared[to] == nil {
				shared[to] = make(map[string]bool)
			}
			for _, id := range sourceIDs {
				shared[to][id] = true
			}
		}
	}

	return boolSetsToDetectionSet(shared)
}

func BuildDetectionContacts(base DetectionSet, rules RelationshipRules) map[string][]DetectionContactInfo {
	contacts := make(map[string]map[string]DetectionContactInfo)
	for team, ids := range base {
		if contacts[team] == nil {
			contacts[team] = make(map[string]DetectionContactInfo)
		}
		for _, id := range ids {
			contacts[team][id] = DetectionContactInfo{
				UnitID:     id,
				SourceTeam: team,
				Shared:     false,
			}
		}
	}

	for from, recipients := range rules {
		sourceIDs := base[from]
		if len(sourceIDs) == 0 {
			continue
		}
		for to, rule := range recipients {
			if !rule.ShareIntel {
				continue
			}
			if contacts[to] == nil {
				contacts[to] = make(map[string]DetectionContactInfo)
			}
			for _, id := range sourceIDs {
				if _, exists := contacts[to][id]; exists {
					continue
				}
				contacts[to][id] = DetectionContactInfo{
					UnitID:     id,
					SourceTeam: from,
					Shared:     true,
				}
			}
		}
	}

	result := make(map[string][]DetectionContactInfo, len(contacts))
	for team, unitMap := range contacts {
		result[team] = make([]DetectionContactInfo, 0, len(unitMap))
		for _, contact := range unitMap {
			result[team] = append(result[team], contact)
		}
	}
	return result
}

func GetRelationshipRule(rules RelationshipRules, from, to string) CountryRelationshipRule {
	return GetRelationshipRuleWithCoalitions(rules, nil, from, to)
}

func GetRelationshipRuleWithCoalitions(rules RelationshipRules, countryCoalitions map[string]string, from, to string) CountryRelationshipRule {
	from = strings.TrimSpace(strings.ToUpper(from))
	to = strings.TrimSpace(strings.ToUpper(to))
	if from == "" || to == "" || from == to {
		return CountryRelationshipRule{
			ShareIntel:                  true,
			AirspaceTransitAllowed:      true,
			AirspaceStrikeAllowed:       true,
			DefensivePositioningAllowed: true,
			MaritimeTransitAllowed:      true,
			MaritimeStrikeAllowed:       true,
		}
	}
	if recipients, ok := rules[from]; ok {
		if rule, ok := recipients[to]; ok {
			return rule
		}
	}
	if countryCoalitions != nil {
		fromCoalition := strings.TrimSpace(strings.ToUpper(countryCoalitions[from]))
		toCoalition := strings.TrimSpace(strings.ToUpper(countryCoalitions[to]))
		if fromCoalition != "" && toCoalition != "" && fromCoalition != toCoalition {
			return CountryRelationshipRule{
				ShareIntel:                  false,
				AirspaceTransitAllowed:      true,
				AirspaceStrikeAllowed:       true,
				DefensivePositioningAllowed: false,
				MaritimeTransitAllowed:      true,
				MaritimeStrikeAllowed:       true,
			}
		}
	}
	return CountryRelationshipRule{}
}
