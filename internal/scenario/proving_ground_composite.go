package scenario

import (
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func provingGroundCompositeGulfRaid() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-composite-gulf-raid",
		Name:           "Proving Ground: Composite Gulf Raid",
		Description:    "Small composite scenario with hosted strike launch, fighter escort, opposing CAP, SAM defense, and a defended fixed target. Built for feature integration and manual inspection.",
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
			{FromCountry: "USA", ToCountry: "QAT", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
			{FromCountry: "USA", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true},
			{FromCountry: "IRN", ToCountry: "USA", AirspaceTransitAllowed: true, DefensivePositioningAllowed: true},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("pg-cg-al-udeid", "Al Udeid", "U.S. Expeditionary Air Base - Composite Raid", "USA", "COALITION_WEST", "expeditionary-air-base", 25.12, 51.31, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingAircraft("pg-cg-usa-f15e", "F-15E", "USAF F-15E Composite Strike", "USA", "COALITION_WEST", "f15e-strike-eagle", 25.12, 51.31, 0, 90, 0)
				u.HostBaseId = "pg-cg-al-udeid"
				u.LoadoutConfigurationId = "deep_strike"
				u.Status.FuelLevelLiters = 6200
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("pg-cg-usa-f35a", "F-35A", "USAF F-35A Composite Escort", "USA", "COALITION_WEST", "f35a-lightning", 27.90, 49.40, 9_400, 60, 235)
				u.Status.FuelLevelLiters = 5200
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("pg-cg-irn-f4e", "F-4E", "IRIAF F-4E Composite CAP", "IRN", "COALITION_IRAN", "f4e-phantom-iriaf", 29.90, 49.20, 9_000, 235, 220)
				u.Status.FuelLevelLiters = 4300
				return u
			}(),
			provingGroundUnit("pg-cg-irn-s300", "S-300", "Bushehr S-300 Test Battery", "IRN", "COALITION_IRAN", "s300pmu2-battery-iran", 28.98, 50.94, 0, 0, 0),
			provingGroundUnit("pg-cg-bushehr-airbase", "Bushehr AB", "Iranian Strategic Air Base - Composite Raid", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 28.95, 50.83, 0, 0, 0),
		},
	}
}

func provingGroundGulfEscalationCycle() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-gulf-escalation-cycle",
		Name:           "Proving Ground: Gulf Escalation Cycle",
		Description:    "Bidirectional Gulf opening exchange with U.S. hosted strike and escort, Iranian CAP and SAM defense, plus ballistic retaliation against the Gulf base. Built for complex feature integration and manual inspection.",
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
			{FromCountry: "USA", ToCountry: "QAT", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
			{FromCountry: "USA", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true},
			{FromCountry: "IRN", ToCountry: "QAT", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
			{FromCountry: "IRN", ToCountry: "USA", AirspaceTransitAllowed: true, DefensivePositioningAllowed: true},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("pg-ge-al-udeid", "Al Udeid", "U.S. Expeditionary Air Base - Gulf Escalation", "USA", "COALITION_WEST", "expeditionary-air-base", 25.12, 51.31, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingAircraft("pg-ge-usa-f15e", "F-15E", "USAF F-15E Gulf Escalation Strike", "USA", "COALITION_WEST", "f15e-strike-eagle", 25.12, 51.31, 0, 90, 0)
				u.HostBaseId = "pg-ge-al-udeid"
				u.LoadoutConfigurationId = "deep_strike"
				u.Status.FuelLevelLiters = 5600
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("pg-ge-usa-f35a", "F-35A", "USAF F-35A Gulf Escalation Escort", "USA", "COALITION_WEST", "f35a-lightning", 27.95, 49.30, 9_400, 60, 235)
				u.Status.FuelLevelLiters = 4600
				return u
			}(),
			func() *enginev1.Unit {
				u := provingAircraft("pg-ge-irn-f4e", "F-4E", "IRIAF F-4E Gulf Escalation CAP", "IRN", "COALITION_IRAN", "f4e-phantom-iriaf", 29.70, 49.15, 9_000, 235, 220)
				u.Status.FuelLevelLiters = 3800
				return u
			}(),
			provingGroundUnit("pg-ge-irn-s300", "S-300", "Bushehr S-300 Gulf Escalation Battery", "IRN", "COALITION_IRAN", "s300pmu2-battery-iran", 28.98, 50.94, 0, 0, 0),
			provingGroundUnit("pg-ge-bushehr-airbase", "Bushehr AB", "Iranian Strategic Air Base - Gulf Escalation", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 28.95, 50.83, 0, 0, 0),
			provingGroundUnit("pg-ge-kheibar-brigade", "Kheibar Brigade", "Iranian Ballistic Brigade - Gulf Escalation", "IRN", "COALITION_IRAN", "kheibar-shekan-brigade", 30.70, 50.20, 0, 180, 0),
		},
	}
}

func provingGroundHormuzMultiDomainSkirmish() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-hormuz-multi-domain-skirmish",
		Name:           "Proving Ground: Hormuz Multi-Domain Skirmish",
		Description:    "Compact Strait of Hormuz fight with simultaneous air, surface, and subsurface engagements. Built to inspect whether multiple combat layers still feel coherent when they happen together.",
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
			{FromCountry: "USA", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true},
			{FromCountry: "IRN", ToCountry: "USA", AirspaceTransitAllowed: true, DefensivePositioningAllowed: true},
		},
		Units: []*enginev1.Unit{
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-hm-usa-destroyer", "MMSC", "U.S. Surface Combatant - Hormuz Skirmish", "USA", "COALITION_WEST", "al-jubail-mmsc-saudi", 26.58, 56.22, 0, 90, 12)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "asm-nsm", CurrentQty: 8, MaxQty: 8}}
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-hm-usa-virginia", "Virginia", "USS Virginia - Hormuz Skirmish", "USA", "COALITION_WEST", "virginia-block-v", 26.48, 56.12, 0, 90, 7)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "torp-mk48", CurrentQty: 6, MaxQty: 6}}
				return u
			}(),
			provingAircraft("pg-hm-usa-f35a", "F-35A", "USAF F-35A Hormuz CAP", "USA", "COALITION_WEST", "f35a-lightning", 26.70, 56.00, 9_200, 75, 235),
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-hm-irn-missile-boat", "Zulfiqar", "Iranian Missile Boat - Hormuz Skirmish", "IRN", "COALITION_IRAN", "zulfiqar-missile-boat", 26.62, 56.74, 0, 250, 16)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "asm-noor", CurrentQty: 4, MaxQty: 4}}
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-hm-irn-fateh-sub", "Fateh", "IRIN Fateh - Hormuz Skirmish", "IRN", "COALITION_IRAN", "fateh-submarine", 26.42, 56.55, 0, 250, 5)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "torp-valfajr", CurrentQty: 4, MaxQty: 4}}
				return u
			}(),
			provingAircraft("pg-hm-irn-f4e", "F-4E", "IRIAF F-4E Hormuz CAP", "IRN", "COALITION_IRAN", "f4e-phantom-iriaf", 26.76, 56.82, 9_100, 255, 220),
		},
	}
}
