package sim

import enginev1 "github.com/aressim/internal/gen/engine/v1"

// DefaultAirborneAltitudeM returns a simple mission altitude band for airborne
// platforms. This keeps aircraft visibly airborne without modeling detailed
// climb profiles or mission-specific altitude planning yet.
func DefaultAirborneAltitudeM(def DefStats) float64 {
	if def.Domain != enginev1.UnitDomain_DOMAIN_AIR {
		return 0
	}
	switch def.GeneralType {
	case 13: // bomber
		return 11_500
	case 14, 15, 16, 17, 18, 19: // tankers / AEW / ISR / transport / EW
		return 10_500
	case 20, 21, 22, 23: // helicopters
		return 1_500
	case 30, 31, 32, 33, 34, 35: // UAV families
		return 6_000
	default: // fighters / multirole / attack aircraft
		return 9_000
	}
}

func TravelAltitudeM(unit *enginev1.Unit, def DefStats) float64 {
	if unit == nil || unit.GetPosition() == nil {
		return 0
	}
	if def.Domain != enginev1.UnitDomain_DOMAIN_AIR {
		return unit.GetPosition().GetAltMsl()
	}
	alt := unit.GetPosition().GetAltMsl()
	if alt > 0 {
		return alt
	}
	return DefaultAirborneAltitudeM(def)
}
