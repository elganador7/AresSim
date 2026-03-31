package sim

import (
	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/geo"
)

func isUnauthorizedOverflight(defender, intruder *enginev1.Unit, defs map[string]DefStats, rules RelationshipRules) bool {
	if defender == nil || intruder == nil {
		return false
	}
	if unitTeamID(defender) == "" || unitTeamID(defender) == unitTeamID(intruder) {
		return false
	}
	intruderDef := defs[intruder.DefinitionId]
	if intruderDef.Domain != enginev1.UnitDomain_DOMAIN_AIR {
		return false
	}
	if intruder.GetPosition() == nil || intruder.GetPosition().GetAltMsl() <= 100 {
		return false
	}
	ctx := geo.LookupPoint(geo.Point{
		Lat: intruder.GetPosition().GetLat(),
		Lon: intruder.GetPosition().GetLon(),
	})
	defenderCountry := unitTeamID(defender)
	if geo.CountryCode(ctx.AirspaceOwner) != defenderCountry {
		return false
	}
	rule := GetRelationshipRule(rules, unitTeamID(intruder), defenderCountry)
	return !rule.AirspaceTransitAllowed
}

func canPerformSovereignAirDefense(unit *enginev1.Unit, def DefStats) bool {
	if unit == nil {
		return false
	}
	if def.Domain != enginev1.UnitDomain_DOMAIN_AIR {
		return true
	}
	return unit.GetPosition() != nil && unit.GetPosition().GetAltMsl() > 100
}
