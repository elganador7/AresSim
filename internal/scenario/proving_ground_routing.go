package scenario

import (
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func provingGroundDestroyerVsMissileBoat() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-destroyer-vs-missile-boat",
		Name:           "Proving Ground: Destroyer vs Missile Boat",
		Description:    "A modern U.S. surface combatant versus an Iranian fast-attack missile boat in open water. Used to calibrate surface combat lethality.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: clearWeatherMap(),
		Units: []*enginev1.Unit{
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-usa-mmsc", "MMSC", "U.S. Surface Combatant Test Ship", "USA", "COALITION_WEST", "al-jubail-mmsc-saudi", 19.80, 63.20, 0, 90, 12)
				u.Weapons = []*enginev1.WeaponState{
					{WeaponId: "asm-nsm", CurrentQty: 8, MaxQty: 8},
				}
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-irn-missile-boat", "Zulfiqar", "Iranian Zulfiqar Missile Boat Test Ship", "IRN", "COALITION_IRAN", "zulfiqar-missile-boat", 19.80, 64.30, 0, 270, 16)
				u.Weapons = []*enginev1.WeaponState{
					{WeaponId: "asm-noor", CurrentQty: 4, MaxQty: 4},
				}
				return u
			}(),
		},
	}
}

func provingGroundMaritimeTransitRules() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-maritime-transit-rules",
		Name:           "Proving Ground: Maritime Transit Rules",
		Description:    "Manual transit-permission scenario. Play as USA. Try moving the surface ship into Qatari territorial waters near 25.445, 51.669 and it should be blocked. Try moving the submarine to the same point and it should be allowed.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: clearWeatherMap(),
		Units: []*enginev1.Unit{
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-mtr-usa-ship", "MMSC", "U.S. Surface Combatant - Transit Rules", "USA", "COALITION_WEST", "al-jubail-mmsc-saudi", 25.30, 51.80, 0, 90, 12)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "asm-nsm", CurrentQty: 8, MaxQty: 8}}
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-mtr-usa-sub", "Virginia", "USS Virginia - Transit Rules", "USA", "COALITION_WEST", "virginia-block-v", 25.22, 51.92, 0, 90, 7)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "torp-mk48", CurrentQty: 6, MaxQty: 6}}
				return u
			}(),
		},
	}
}

func provingGroundAirspaceRoutingRules() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-airspace-routing-rules",
		Name:           "Proving Ground: Airspace Routing Rules",
		Description:    "Manual airspace-transit scenario. Play as USA. Try moving the fighter toward Bushehr at 28.950, 50.830 and it should be blocked. Try moving it east over the Gulf of Oman near 24.900, 56.000 and it should be allowed.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{{
			FromCountry:            "USA",
			ToCountry:              "QAT",
			AirspaceTransitAllowed: true,
		}},
		Units: []*enginev1.Unit{
			provingAircraft("pg-arr-usa-f35a", "F-35A", "USAF F-35A - Airspace Routing Rules", "USA", "COALITION_WEST", "f35a-lightning", 25.12, 51.31, 9_500, 90, 240),
		},
	}
}

func provingGroundLandBorderRoutingRules() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-land-border-routing-rules",
		Name:           "Proving Ground: Land Border Routing Rules",
		Description:    "Manual land-border scenario. Play as USA. Try moving the convoy into Iran near 30.400, 48.200 and it should be blocked. Try moving it within Iraq near 30.250, 47.500 and it should be allowed.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{{
			FromCountry:            "USA",
			ToCountry:              "IRQ",
			AirspaceTransitAllowed: true,
		}},
		Units: []*enginev1.Unit{
			provingGroundUnit("pg-lbr-usa-convoy", "U.S. Convoy", "U.S. Ground Convoy - Border Rules", "USA", "COALITION_WEST", "stryker-company", 30.50, 47.70, 0, 90, 15),
		},
	}
}

func provingGroundMaritimeRerouteAroundQatar() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-maritime-reroute-around-qatar",
		Name:           "Proving Ground: Maritime Reroute Around Qatar",
		Description:    "Manual reroute scenario. Play as USA. Move the ship from west of Qatar to 24.800, 52.300 and it should succeed with a curved route around Qatar instead of a straight line.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "SAU", MaritimeTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "ARE", MaritimeTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "BHR", MaritimeTransitAllowed: true},
		},
		Units: []*enginev1.Unit{
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-mraq-usa-ship", "MMSC", "U.S. Surface Combatant - Qatar Reroute", "USA", "COALITION_WEST", "al-jubail-mmsc-saudi", 24.90, 50.30, 0, 90, 12)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "asm-nsm", CurrentQty: 8, MaxQty: 8}}
				return u
			}(),
		},
	}
}

func provingGroundHormuzTransitPassage() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-hormuz-transit-passage",
		Name:           "Proving Ground: Hormuz Transit Passage",
		Description:    "Manual routing scenario. Play as USA. Move the ship east through the Strait of Hormuz toward 26.340, 56.820. It should route through the transit lane without crossing Musandam or nearby land.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: clearWeatherMap(),
		Units: []*enginev1.Unit{
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-htp-usa-ship", "MMSC", "U.S. Surface Combatant - Hormuz Transit", "USA", "COALITION_WEST", "al-jubail-mmsc-saudi", 25.86, 55.18, 0, 70, 12)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "asm-nsm", CurrentQty: 8, MaxQty: 8}}
				return u
			}(),
		},
	}
}

func provingGroundAirRerouteAroundQatar() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-air-reroute-around-qatar",
		Name:           "Proving Ground: Air Reroute Around Qatar",
		Description:    "Manual reroute scenario. Play as USA. Move the fighter east toward 24.800, 53.300 and it should succeed with a rerouted path that bends around denied Qatari airspace.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "SAU", AirspaceTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "ARE", AirspaceTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "BHR", AirspaceTransitAllowed: true},
		},
		Units: []*enginev1.Unit{
			provingAircraft("pg-arq-usa-f35a", "F-35A", "USAF F-35A - Qatar Air Reroute", "USA", "COALITION_WEST", "f35a-lightning", 24.90, 50.30, 9_500, 90, 240),
		},
	}
}

func provingGroundVirginiaVsFatehSub() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-virginia-vs-fateh-sub",
		Name:           "Proving Ground: Virginia vs Fateh Sub",
		Description:    "Virginia-class attack submarine against an Iranian Fateh-class submarine in Gulf littoral waters. Used to calibrate subsurface lethality.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: clearWeatherMap(),
		Units: []*enginev1.Unit{
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-usa-virginia", "Virginia", "USS Virginia Test Boat", "USA", "COALITION_WEST", "virginia-block-v", 26.35, 56.05, 0, 90, 7)
				u.Weapons = []*enginev1.WeaponState{
					{WeaponId: "torp-mk48", CurrentQty: 6, MaxQty: 6},
				}
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-irn-fateh-sub", "Fateh", "IRIN Fateh Test Boat", "IRN", "COALITION_IRAN", "fateh-submarine", 26.39, 56.38, 0, 250, 5)
				u.Weapons = []*enginev1.WeaponState{
					{WeaponId: "torp-valfajr", CurrentQty: 4, MaxQty: 4},
				}
				return u
			}(),
		},
	}
}
