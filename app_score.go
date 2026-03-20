package main

import (
	"sort"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/library"
	"github.com/aressim/internal/sim"
)

const valueOfStatisticalLifeUSD = library.DefaultValueOfStatisticalLifeUSD

func currentTeamID(u *enginev1.Unit) string {
	if u == nil {
		return ""
	}
	return sim.CountryDisplayCode(u.GetTeamId())
}

func damageLossFraction(state enginev1.DamageState) float64 {
	switch state {
	case enginev1.DamageState_DAMAGE_STATE_DAMAGED:
		return 0.25
	case enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED:
		return 0.60
	case enginev1.DamageState_DAMAGE_STATE_DESTROYED:
		return 1.0
	default:
		return 0
	}
}

func casualtyFraction(def sim.DefStats, state enginev1.DamageState, personnelStrength float32) float64 {
	statusLoss := 0.0
	if personnelStrength > 0 {
		statusLoss = 1 - float64(personnelStrength)
		if statusLoss < 0 {
			statusLoss = 0
		}
	}
	base := 0.0
	switch state {
	case enginev1.DamageState_DAMAGE_STATE_DAMAGED:
		base = 0.08
	case enginev1.DamageState_DAMAGE_STATE_MISSION_KILLED:
		base = 0.25
	case enginev1.DamageState_DAMAGE_STATE_DESTROYED:
		base = 0.75
	}
	switch def.AssetClass {
	case "airbase", "port", "oil_field", "pipeline_node", "desalination_plant", "power_plant", "radar_site", "c2_site":
		base *= 0.25
	}
	if def.Domain == enginev1.UnitDomain_DOMAIN_AIR || def.Domain == enginev1.UnitDomain_DOMAIN_SEA || def.Domain == enginev1.UnitDomain_DOMAIN_SUBSURFACE {
		base *= 1.1
	}
	if statusLoss > base {
		return statusLoss
	}
	return base
}

func buildTeamScores(units []*enginev1.Unit, defs map[string]sim.DefStats) []*enginev1.TeamScore {
	if len(units) == 0 {
		return nil
	}
	byTeam := make(map[string]*enginev1.TeamScore)
	for _, u := range units {
		if u == nil {
			continue
		}
		def := defs[u.GetDefinitionId()]
		fraction := damageLossFraction(u.GetDamageState())
		if fraction <= 0 {
			continue
		}
		teamID := currentTeamID(u)
		if teamID == "" {
			continue
		}
		score := byTeam[teamID]
		if score == nil {
			score = &enginev1.TeamScore{TeamId: teamID}
			byTeam[teamID] = score
		}
		score.ReplacementLossUsd += def.ReplacementCostUSD * fraction
		score.StrategicLossUsd += def.StrategicValueUSD * fraction
		score.EconomicLossUsd += def.EconomicValueUSD * fraction
		score.HumanLossUsd += float64(def.AuthorizedPersonnel) * casualtyFraction(def, u.GetDamageState(), u.GetStatus().GetPersonnelStrength()) * valueOfStatisticalLifeUSD
	}
	out := make([]*enginev1.TeamScore, 0, len(byTeam))
	for _, score := range byTeam {
		score.TotalLossUsd = score.ReplacementLossUsd + score.StrategicLossUsd + score.EconomicLossUsd + score.HumanLossUsd
		out = append(out, score)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalLossUsd == out[j].TotalLossUsd {
			return out[i].TeamId < out[j].TeamId
		}
		return out[i].TotalLossUsd > out[j].TotalLossUsd
	})
	return out
}

func batchNeedsScoreUpdate(update *enginev1.BatchUnitUpdate) bool {
	if update == nil {
		return false
	}
	for _, delta := range update.GetDeltas() {
		if delta == nil {
			continue
		}
		if delta.GetDamageState() != enginev1.DamageState_DAMAGE_STATE_UNSPECIFIED {
			return true
		}
	}
	return false
}

func defaultIfZero(value, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

func maxInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
