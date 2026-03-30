package scenario

import (
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func provingGroundModernVsLegacyAir() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-modern-vs-legacy-air",
		Name:           "Proving Ground: Modern vs Legacy Air",
		Description:    "F-35A versus F-4E in a controlled BVR intercept setup. Used to calibrate fighter advantage curves.",
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
			{
				FromCountry:                 "USA",
				ToCountry:                   "IRN",
				AirspaceTransitAllowed:      true,
				DefensivePositioningAllowed: true,
			},
			{
				FromCountry:                 "IRN",
				ToCountry:                   "USA",
				AirspaceTransitAllowed:      true,
				DefensivePositioningAllowed: true,
			},
		},
		Units: []*enginev1.Unit{
			provingAircraft("pg-usa-f35a", "F-35A", "F-35A Lightning II Test Flight", "USA", "COALITION_WEST", "f35a-lightning", 30.00, 50.00, 9_000, 90, 240),
			provingAircraft("pg-irn-f4e", "F-4E", "IRIAF F-4E Phantom II Test Flight", "IRN", "COALITION_IRAN", "f4e-phantom-iriaf", 30.00, 51.75, 9_000, 270, 240),
		},
	}
}

func provingGroundAEWSupportedIntercept() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-aew-supported-intercept",
		Name:           "Proving Ground: AEW-Supported Intercept",
		Description:    "Two U.S. F-15Cs with E-3A support against two Iranian F-4Es in the Gulf. Used to inspect whether the AEW-supported package shows a clear force-multiplier effect.",
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
			{
				FromCountry:                 "USA",
				ToCountry:                   "IRN",
				AirspaceTransitAllowed:      true,
				DefensivePositioningAllowed: true,
			},
			{
				FromCountry:                 "IRN",
				ToCountry:                   "USA",
				AirspaceTransitAllowed:      true,
				DefensivePositioningAllowed: true,
			},
		},
		Units: []*enginev1.Unit{
			provingAircraft("pg-aew-usa-e3a", "E-3A", "Saudi E-3A Test Support", "USA", "COALITION_WEST", "e3a-sentry-saudi", 28.80, 49.00, 10_000, 70, 220),
			provingAircraft("pg-aew-usa-lead", "F-15C", "USAF F-15C Lead", "USA", "COALITION_WEST", "f15c-eagle", 29.05, 49.45, 9_600, 70, 240),
			provingAircraft("pg-aew-usa-wing", "F-15C", "USAF F-15C Wing", "USA", "COALITION_WEST", "f15c-eagle", 28.95, 49.25, 9_600, 70, 240),
			provingAircraft("pg-aew-irn-lead", "F-4E", "IRIAF F-4E Lead", "IRN", "COALITION_IRAN", "f4e-phantom-iriaf", 29.25, 51.10, 9_100, 250, 220),
			provingAircraft("pg-aew-irn-wing", "F-4E", "IRIAF F-4E Wing", "IRN", "COALITION_IRAN", "f4e-phantom-iriaf", 29.10, 50.90, 9_100, 250, 220),
		},
	}
}

func provingGroundHostedStrikeLoadout() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-hosted-strike-loadout",
		Name:           "Proving Ground: Hosted Strike Loadout",
		Description:    "U.S. F-15E launched from a Gulf base against a fixed Iranian airbase. Used to validate hosted-aircraft loadout switching and attack-order assignment.",
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
			{
				FromCountry:            "USA",
				ToCountry:              "QAT",
				AirspaceTransitAllowed: true,
				AirspaceStrikeAllowed:  true,
			},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("pg-usa-home-airbase", "Al Udeid", "U.S. Expeditionary Air Base - Hosted Strike Test", "USA", "COALITION_WEST", "expeditionary-air-base", 25.12, 51.31, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingAircraft("pg-usa-f15e", "F-15E", "USAF F-15E Strike Eagle Test Flight", "USA", "COALITION_WEST", "f15e-strike-eagle", 25.12, 51.31, 0, 90, 0)
				u.HostBaseId = "pg-usa-home-airbase"
				u.LoadoutConfigurationId = "sead"
				return u
			}(),
			provingGroundUnit("pg-irn-airbase", "Bushehr AB", "Iranian Strategic Air Base - Strike Test Template", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 28.95, 50.83, 0, 0, 0),
		},
	}
}

func provingGroundDefensiveInterceptThirdParty() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-defensive-intercept-third-party",
		Name:           "Proving Ground: Defensive Intercept Third-Party Airspace",
		Description:    "U.S. fighter intercept against an Iranian intruder over Iraqi airspace. Used to validate defensive air operations without strike permission.",
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
			{
				FromCountry:                 "USA",
				ToCountry:                   "IRQ",
				AirspaceTransitAllowed:      true,
				DefensivePositioningAllowed: true,
			},
			{
				FromCountry:            "USA",
				ToCountry:              "IRN",
				AirspaceTransitAllowed: true,
			},
			{
				FromCountry:            "IRN",
				ToCountry:              "IRQ",
				AirspaceTransitAllowed: true,
			},
		},
		Units: []*enginev1.Unit{
			provingAircraft("pg-usa-f35a-third-party", "F-35A", "USAF F-35A Third-Party Intercept", "USA", "COALITION_WEST", "f35a-lightning", 33.10, 44.20, 9_500, 80, 235),
			provingAircraft("pg-irn-f4e-third-party", "F-4E", "IRIAF F-4E Third-Party Transit", "IRN", "COALITION_IRAN", "f4e-phantom-iriaf", 33.12, 45.85, 9_200, 260, 220),
		},
	}
}

func provingGroundSamVsStrikerOverland() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-sam-vs-striker-overland",
		Name:           "Proving Ground: SAM vs Striker Overland",
		Description:    "F-15E strike against an Iranian interior airbase protected by a long-range SAM battery. Used to tune overland defended-strike survivability.",
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
			{
				FromCountry:                 "USA",
				ToCountry:                   "IRN",
				AirspaceTransitAllowed:      true,
				AirspaceStrikeAllowed:       true,
				DefensivePositioningAllowed: true,
			},
		},
		Units: []*enginev1.Unit{
			func() *enginev1.Unit {
				u := provingAircraft("pg-usa-f35a-overland", "F-35A", "USAF F-35A Overland Strike", "USA", "COALITION_WEST", "f35a-lightning", 33.60, 49.90, 9_000, 90, 240)
				u.LoadoutConfigurationId = "deep_strike"
				return u
			}(),
			provingGroundUnit("pg-irn-s300", "S-300", "Iranian S-300PMU2 Test Battery", "IRN", "COALITION_IRAN", "s300pmu2-battery-iran", 32.70, 51.68, 0, 0, 0),
			provingGroundUnit("pg-irn-overland-airbase", "Esfahan AB", "Iranian Interior Strategic Air Base - Test Template", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 32.75, 51.86, 0, 0, 0),
		},
	}
}

func provingGroundUIDemoGulfSkirmish() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-ui-demo-gulf-skirmish",
		Name:           "Proving Ground: UI Demo Gulf Skirmish",
		Description:    "Manual demo scenario. Play as USA. First click the Iranian F-4 and assign the airborne F-35A to intercept. Then click Bushehr AB and assign the grounded F-15E strike aircraft. This should show visible fixed sites, enemy-card shooter selection, hosted-aircraft launch, aircraft altitude, movement, and impact resolution without larger-scenario clutter.",
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
			{
				FromCountry:            "USA",
				ToCountry:              "QAT",
				AirspaceTransitAllowed: true,
				AirspaceStrikeAllowed:  true,
			},
			{
				FromCountry:            "USA",
				ToCountry:              "IRN",
				AirspaceTransitAllowed: true,
				AirspaceStrikeAllowed:  true,
			},
			{
				FromCountry:                 "USA",
				ToCountry:                   "IRQ",
				AirspaceTransitAllowed:      true,
				DefensivePositioningAllowed: true,
			},
			{
				FromCountry:            "IRN",
				ToCountry:              "IRQ",
				AirspaceTransitAllowed: true,
			},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("pg-ui-al-udeid", "Al Udeid", "U.S. Expeditionary Air Base - UI Demo", "USA", "COALITION_WEST", "expeditionary-air-base", 25.12, 51.31, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingAircraft("pg-ui-usa-f15e", "F-15E", "USAF F-15E Demo Strike Flight", "USA", "COALITION_WEST", "f15e-strike-eagle", 25.12, 51.31, 0, 90, 0)
				u.HostBaseId = "pg-ui-al-udeid"
				u.LoadoutConfigurationId = "deep_strike"
				return u
			}(),
			provingAircraft("pg-ui-usa-f35a", "F-35A", "USAF F-35A Demo Intercept Flight", "USA", "COALITION_WEST", "f35a-lightning", 28.90, 49.40, 9_500, 55, 235),
			provingAircraft("pg-ui-irn-f4e", "F-4E", "IRIAF F-4E Demo Patrol", "IRN", "COALITION_IRAN", "f4e-phantom-iriaf", 30.50, 48.20, 9_200, 235, 220),
			provingGroundUnit("pg-ui-bushehr-airbase", "Bushehr AB", "Iranian Strategic Air Base - UI Demo", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 28.95, 50.83, 0, 0, 0),
		},
	}
}

func israelMissileDefenseSetupActions() []ProvingGroundSetupAction {
	return []ProvingGroundSetupAction{{Kind: "set_player", TeamID: "ISR"}}
}

func attackOrder(targetID string, orderType enginev1.AttackOrderType, effect enginev1.DesiredEffect) *enginev1.AttackOrder {
	return &enginev1.AttackOrder{
		TargetUnitId:  targetID,
		OrderType:     orderType,
		DesiredEffect: effect,
	}
}
