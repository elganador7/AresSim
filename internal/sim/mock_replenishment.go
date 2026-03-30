package sim

import (
	"math"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"google.golang.org/protobuf/proto"
)

func processReplenishmentTick(units []*enginev1.Unit, defs map[string]DefStats, simSeconds float64) []*enginev1.UnitDelta {
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, unit := range units {
		unitByID[unit.GetId()] = unit
	}

	deltas := make([]*enginev1.UnitDelta, 0)
	for _, unit := range units {
		def := defs[normalizeDefinitionID(unit.GetDefinitionId())]
		if shouldCompleteReplenishment(unit, simSeconds) {
			refillUnit(unit, def)
			deltas = append(deltas, &enginev1.UnitDelta{
				UnitId:                 unit.GetId(),
				Status:                 unit.GetStatus(),
				Weapons:                cloneWeaponStates(unit.GetWeapons()),
				NextSortieReadySeconds: unit.GetNextSortieReadySeconds(),
			})
			continue
		}
		if shouldStartReplenishment(unit, def, unitByID, defs, simSeconds) {
			beginReplenishment(unit, def, simSeconds)
			deltas = append(deltas, &enginev1.UnitDelta{
				UnitId:                 unit.GetId(),
				Status:                 unit.GetStatus(),
				MoveOrder:              &enginev1.MoveOrder{},
				NextSortieReadySeconds: unit.GetNextSortieReadySeconds(),
			})
		}
	}
	return deltas
}

func tickFuelBurnLiters(unit *enginev1.Unit, def DefStats, timeScale float64) float64 {
	if def.FuelBurnRateLph <= 0 || timeScale <= 0 {
		return 0
	}
	if isAirborneAircraft(unit, def) {
		return def.FuelBurnRateLph * (tickInterval.Seconds() * timeScale / 3600.0)
	}
	if unit.GetMoveOrder() == nil || len(unit.GetMoveOrder().GetWaypoints()) == 0 {
		return 0
	}
	return def.FuelBurnRateLph * (tickInterval.Seconds() * timeScale / 3600.0)
}

func shouldStartReplenishment(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit, defs map[string]DefStats, simSeconds float64) bool {
	if unit == nil || unit.GetStatus() == nil || !unit.GetStatus().GetIsActive() {
		return false
	}
	if currentDamageState(unit) != enginev1.DamageState_DAMAGE_STATE_OPERATIONAL {
		return false
	}
	if !unitNeedsReplenishment(unit, def) {
		return false
	}
	return isAtReplenishmentProvider(unit, def, unitByID, defs, simSeconds)
}

func shouldCompleteReplenishment(unit *enginev1.Unit, simSeconds float64) bool {
	if unit == nil || unit.GetStatus() == nil || unit.GetStatus().GetIsActive() {
		return false
	}
	ready := unit.GetNextSortieReadySeconds()
	return ready > 0 && ready <= simSeconds && currentDamageState(unit) == enginev1.DamageState_DAMAGE_STATE_OPERATIONAL
}

func unitNeedsReplenishment(unit *enginev1.Unit, def DefStats) bool {
	if unit == nil {
		return false
	}
	if def.FuelCapacityLiters > 0 && float64(unit.GetStatus().GetFuelLevelLiters()) < def.FuelCapacityLiters*0.98 {
		return true
	}
	for _, weapon := range unit.GetWeapons() {
		if weapon.GetCurrentQty() < weapon.GetMaxQty() {
			return true
		}
	}
	return false
}

func isAtReplenishmentProvider(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit, defs map[string]DefStats, simSeconds float64) bool {
	if unit == nil || unit.GetPosition() == nil {
		return false
	}
	switch def.Domain {
	case enginev1.UnitDomain_DOMAIN_AIR:
		return canAircraftReplenish(unit, def, unitByID, defs)
	case enginev1.UnitDomain_DOMAIN_SEA, enginev1.UnitDomain_DOMAIN_LAND, enginev1.UnitDomain_DOMAIN_SUBSURFACE:
		return nearbyFriendlyLogisticsProvider(unit, def, unitByID, defs) != nil
	default:
		return false
	}
}

func canAircraftReplenish(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit, defs map[string]DefStats) bool {
	if unit == nil || unit.GetPosition() == nil {
		return false
	}
	host := unitByID[unit.GetHostBaseId()]
	if host != nil && host.GetPosition() != nil &&
		haversineM(unit.GetPosition().GetLat(), unit.GetPosition().GetLon(), host.GetPosition().GetLat(), host.GetPosition().GetLon()) <= 1_000 &&
		unit.GetPosition().GetAltMsl() <= 100 {
		return true
	}
	return nearbyAirTanker(unit, unitByID, defs)
}

func nearbyAirTanker(unit *enginev1.Unit, unitByID map[string]*enginev1.Unit, defs map[string]DefStats) bool {
	for _, provider := range unitByID {
		if provider == nil || provider.GetId() == unit.GetId() || provider.GetPosition() == nil || unitTeamID(provider) != unitTeamID(unit) {
			continue
		}
		if provider.GetPosition().GetAltMsl() <= 100 {
			continue
		}
		if haversineM(unit.GetPosition().GetLat(), unit.GetPosition().GetLon(), provider.GetPosition().GetLat(), provider.GetPosition().GetLon()) > 5_000 {
			continue
		}
		if providerDef, ok := providerDefStats(provider, defs); ok && providerDef.GeneralType == int32(enginev1.UnitGeneralType_GENERAL_TYPE_TANKER) {
			return true
		}
	}
	return false
}

func nearbyFriendlyLogisticsProvider(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit, defs map[string]DefStats) *enginev1.Unit {
	if unit.GetPosition() == nil || unit.GetPosition().GetSpeed() > 1 {
		return nil
	}
	maxDist := 2_000.0
	if def.Domain == enginev1.UnitDomain_DOMAIN_SEA || def.Domain == enginev1.UnitDomain_DOMAIN_SUBSURFACE {
		maxDist = 5_000
	}
	for _, provider := range unitByID {
		if provider == nil || provider.GetId() == unit.GetId() || provider.GetPosition() == nil || unitTeamID(provider) != unitTeamID(unit) {
			continue
		}
		providerDef, ok := providerDefStats(provider, defs)
		if !ok {
			continue
		}
		if providerDef.AssetClass != "port" && providerDef.GeneralType != int32(enginev1.UnitGeneralType_GENERAL_TYPE_LOGISTICS) {
			continue
		}
		if haversineM(unit.GetPosition().GetLat(), unit.GetPosition().GetLon(), provider.GetPosition().GetLat(), provider.GetPosition().GetLon()) <= maxDist {
			return provider
		}
	}
	return nil
}

func providerDefStats(provider *enginev1.Unit, defs map[string]DefStats) (DefStats, bool) {
	if provider == nil {
		return DefStats{}, false
	}
	def, ok := defs[normalizeDefinitionID(provider.GetDefinitionId())]
	return def, ok
}

func beginReplenishment(unit *enginev1.Unit, def DefStats, simSeconds float64) {
	if unit.GetStatus() == nil {
		unit.Status = &enginev1.OperationalStatus{}
	}
	durationSeconds := replenishmentDurationSeconds(def)
	if durationSeconds <= 0 {
		durationSeconds = 1800
	}
	if unit.GetNextSortieReadySeconds() > simSeconds {
		unit.NextSortieReadySeconds = math.Max(unit.GetNextSortieReadySeconds(), simSeconds+durationSeconds)
	} else {
		unit.NextSortieReadySeconds = simSeconds + durationSeconds
	}
	unit.Status.IsActive = false
	unit.MoveOrder = nil
	if unit.GetPosition() != nil {
		unit.Position.Speed = 0
	}
}

func refillUnit(unit *enginev1.Unit, def DefStats) {
	if unit.GetStatus() == nil {
		unit.Status = &enginev1.OperationalStatus{}
	}
	unit.Status.IsActive = true
	if def.FuelCapacityLiters > 0 {
		unit.Status.FuelLevelLiters = float32(def.FuelCapacityLiters)
	}
	for _, weapon := range unit.GetWeapons() {
		weapon.CurrentQty = weapon.GetMaxQty()
	}
	unit.NextSortieReadySeconds = 0
}

func replenishmentDurationSeconds(def DefStats) float64 {
	minutes := def.SortieIntervalMinutes
	if minutes <= 0 {
		switch def.Domain {
		case enginev1.UnitDomain_DOMAIN_AIR:
			minutes = 45
		case enginev1.UnitDomain_DOMAIN_SEA:
			minutes = 120
		case enginev1.UnitDomain_DOMAIN_SUBSURFACE:
			minutes = 180
		default:
			minutes = 60
		}
	}
	return float64(minutes) * 60
}

func cloneWeaponStates(states []*enginev1.WeaponState) []*enginev1.WeaponState {
	cloned := make([]*enginev1.WeaponState, 0, len(states))
	for _, state := range states {
		if state == nil {
			continue
		}
		cloned = append(cloned, proto.Clone(state).(*enginev1.WeaponState))
	}
	return cloned
}
