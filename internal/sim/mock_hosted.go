package sim

import enginev1 "github.com/aressim/internal/gen/engine/v1"

func canHostAircraft(def DefStats) bool {
	if def.AssetClass == "airbase" {
		return true
	}
	return def.EmbarkedFixedWingCapacity > 0 ||
		def.EmbarkedRotaryWingCapacity > 0 ||
		def.EmbarkedUAVCapacity > 0 ||
		def.LaunchCapacityPerInterval > 0 ||
		def.RecoveryCapacityPerInterval > 0
}

func hostedUnitShouldMirrorBase(unit *enginev1.Unit, def DefStats) bool {
	if unit == nil || unit.GetPosition() == nil {
		return false
	}
	if def.Domain != enginev1.UnitDomain_DOMAIN_AIR {
		return false
	}
	if unit.GetPosition().GetAltMsl() > 0 {
		return false
	}
	if order := unit.GetMoveOrder(); order != nil && len(order.GetWaypoints()) > 0 {
		return false
	}
	return unit.GetHostBaseId() != ""
}

func syncHostedAircraftToHostBases(units []*enginev1.Unit, defs map[string]DefStats) []*enginev1.UnitDelta {
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, unit := range units {
		unitByID[unit.GetId()] = unit
	}

	deltas := make([]*enginev1.UnitDelta, 0)
	for _, unit := range units {
		if !hostedUnitShouldMirrorBase(unit, defs[unit.GetDefinitionId()]) {
			continue
		}
		base := unitByID[unit.GetHostBaseId()]
		if base == nil || base.GetPosition() == nil || !canHostAircraft(defs[base.GetDefinitionId()]) {
			continue
		}
		if unit.GetPosition().GetLat() == base.GetPosition().GetLat() &&
			unit.GetPosition().GetLon() == base.GetPosition().GetLon() &&
			unit.GetPosition().GetAltMsl() == base.GetPosition().GetAltMsl() {
			continue
		}
		unit.Position = &enginev1.Position{
			Lat:     base.GetPosition().GetLat(),
			Lon:     base.GetPosition().GetLon(),
			AltMsl:  base.GetPosition().GetAltMsl(),
			Heading: base.GetPosition().GetHeading(),
			Speed:   base.GetPosition().GetSpeed(),
		}
		deltas = append(deltas, &enginev1.UnitDelta{
			UnitId:   unit.GetId(),
			Position: unit.GetPosition(),
		})
	}
	return deltas
}

func isAirborneAircraft(unit *enginev1.Unit, def DefStats) bool {
	if unit == nil || unit.GetPosition() == nil {
		return false
	}
	return def.Domain == enginev1.UnitDomain_DOMAIN_AIR && unit.GetPosition().GetAltMsl() > 0
}

func requiredFuelToReachHostBaseLiters(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit) float64 {
	if unit == nil || def.Domain != enginev1.UnitDomain_DOMAIN_AIR || def.CruiseSpeedMps <= 0 || def.FuelBurnRateLph <= 0 {
		return 0
	}
	hostBaseID := unit.GetHostBaseId()
	if hostBaseID == "" {
		return 0
	}
	host := unitByID[hostBaseID]
	if host == nil || host.GetPosition() == nil || unit.GetPosition() == nil {
		return 0
	}
	distM := haversineM(unit.GetPosition().GetLat(), unit.GetPosition().GetLon(), host.GetPosition().GetLat(), host.GetPosition().GetLon())
	flightHours := distM / def.CruiseSpeedMps / 3600.0
	return flightHours * def.FuelBurnRateLph
}

func primaryWeaponState(unit *enginev1.Unit) *enginev1.WeaponState {
	for _, weapon := range unit.GetWeapons() {
		if weapon.GetMaxQty() > 0 {
			return weapon
		}
	}
	return nil
}

func returnToHostBaseWaypoint(unit *enginev1.Unit, def DefStats, unitByID map[string]*enginev1.Unit) *enginev1.Waypoint {
	if unit == nil {
		return nil
	}
	host := unitByID[unit.GetHostBaseId()]
	if host == nil || host.GetPosition() == nil {
		return nil
	}
	return &enginev1.Waypoint{
		Lat:    host.GetPosition().GetLat(),
		Lon:    host.GetPosition().GetLon(),
		AltMsl: TravelAltitudeM(unit, def),
	}
}

func shouldLandAtHostBase(unit *enginev1.Unit, unitByID map[string]*enginev1.Unit, lat, lon float64) bool {
	if unit == nil || unit.GetHostBaseId() == "" {
		return false
	}
	host := unitByID[unit.GetHostBaseId()]
	if host == nil || host.GetPosition() == nil {
		return false
	}
	return haversineM(lat, lon, host.GetPosition().GetLat(), host.GetPosition().GetLon()) <= 1_000
}
