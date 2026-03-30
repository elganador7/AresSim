package scenario

import (
	"fmt"
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func provingGroundBallisticVsAirbase() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-ballistic-vs-airbase",
		Name:           "Proving Ground: Ballistic vs Airbase",
		Description:    "Kheibar Shekan ballistic brigade against a fixed Israeli airbase target. Used to calibrate runway and airbase strike effects.",
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
				u := provingGroundUnit("pg-kheibar-brigade", "Kheibar Brigade", "Kheibar Shekan Test Brigade", "IRN", "COALITION_IRAN", "kheibar-shekan-brigade", 32.10, 42.50, 0, 90, 0)
				u.Weapons = []*enginev1.WeaponState{
					{WeaponId: "ssm-kheibar-shekan", CurrentQty: 4, MaxQty: 4},
				}
				return u
			}(),
			provingGroundUnit("pg-nevatim-airbase", "Nevatim AB", "Israeli Strategic Air Base - Test Template", "ISR", "COALITION_WEST", "israel-strategic-airbase", 31.21, 35.01, 0, 0, 0),
		},
	}
}

func provingGroundIsraelMissileDefenseSaturation() *enginev1.Scenario {
	units := []*enginev1.Unit{
		provingGroundUnit("pg-imds-nevatim", "Nevatim AB", "Israeli Strategic Air Base - Missile Defense", "ISR", "COALITION_WEST", "israel-strategic-airbase", 31.21, 35.01, 0, 0, 0),
		provingGroundUnit("pg-imds-hatzor", "Hatzor AB", "Israeli Strategic Air Base - Hatzor", "ISR", "COALITION_WEST", "israel-strategic-airbase", 31.73, 34.72, 0, 0, 0),
		provingGroundUnit("pg-imds-palmachim", "Palmachim AB", "Israeli Strategic Air Base - Palmachim", "ISR", "COALITION_WEST", "israel-strategic-airbase", 31.89, 34.69, 0, 0, 0),
		provingGroundUnit("pg-imds-telnof", "Tel Nof AB", "Israeli Strategic Air Base - Tel Nof", "ISR", "COALITION_WEST", "israel-strategic-airbase", 31.84, 34.82, 0, 0, 0),
		provingGroundUnit("pg-imds-ramon", "Ramon AB", "Israeli Strategic Air Base - Ramon", "ISR", "COALITION_WEST", "israel-strategic-airbase", 30.61, 34.78, 0, 0, 0),
		provingGroundUnit("pg-imds-arrow3-palmachim", "Arrow-3 Palmachim", "Arrow-3 Battery - Palmachim", "ISR", "COALITION_WEST", "arrow3-battery", 31.93, 34.69, 0, 0, 0),
		provingGroundUnit("pg-imds-arrow2-central", "Arrow-2 Central", "Arrow-2 Battery - Central", "ISR", "COALITION_WEST", "arrow2-battery", 31.95, 34.78, 0, 0, 0),
		provingGroundUnit("pg-imds-ds-dan", "David's Sling Dan", "David's Sling - Dan Region", "ISR", "COALITION_WEST", "davids-sling-battery", 32.08, 34.86, 0, 0, 0),
		provingGroundUnit("pg-imds-id-dan", "Iron Dome Dan", "Iron Dome - Dan Region", "ISR", "COALITION_WEST", "iron-dome-battery", 32.09, 34.82, 0, 0, 0),
		provingGroundUnit("pg-imds-id-negev", "Iron Dome Negev", "Iron Dome - Negev", "ISR", "COALITION_WEST", "iron-dome-battery", 31.04, 34.72, 0, 0, 0),
	}

	f15Targets := []string{"pg-imds-kheibar-1", "pg-imds-kheibar-2", "pg-imds-paveh-1", "pg-imds-paveh-2"}
	for i := 0; i < 4; i++ {
		u := provingAircraft(
			fmt.Sprintf("pg-imds-isr-f15i-%d", i+1),
			"F-15I",
			"F-15I Ra'am Missile Defense Counterstrike",
			"ISR",
			"COALITION_WEST",
			"f15i-raam",
			33.55+float64(i)*0.05,
			44.00+float64(i)*0.08,
			9800,
			80,
			240,
		)
		u.LoadoutConfigurationId = "deep_strike"
		u.AttackOrder = attackOrder(f15Targets[i], enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		units = append(units, u)
	}
	units = append(units, provingAircraft("pg-imds-isr-eitam", "Eitam", "G550 Eitam Missile Defense Orbit", "ISR", "COALITION_WEST", "g550-eitam", 31.95, 34.88, 9500, 90, 210))

	iranTargets := []string{"pg-imds-nevatim", "pg-imds-palmachim", "pg-imds-telnof", "pg-imds-ramon"}
	for i := 0; i < 12; i++ {
		u := provingGroundUnit(
			fmt.Sprintf("pg-imds-kheibar-%d", i+1),
			"Kheibar Brigade",
			"Iranian Kheibar Shekan Brigade - Saturation Strike",
			"IRN",
			"COALITION_IRAN",
			"kheibar-shekan-brigade",
			34.10+0.08*float64(i),
			45.45+0.12*float64(i),
			0,
			240,
			0,
		)
		u.Weapons = []*enginev1.WeaponState{{WeaponId: "ssm-kheibar-shekan", CurrentQty: 10, MaxQty: 10}}
		u.AttackOrder = attackOrder(iranTargets[i%len(iranTargets)], enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL)
		units = append(units, u)
	}
	for i := 0; i < 12; i++ {
		u := provingGroundUnit(
			fmt.Sprintf("pg-imds-paveh-%d", i+1),
			"Paveh Regiment",
			"Iranian Paveh Cruise Missile Regiment - Saturation Strike",
			"IRN",
			"COALITION_IRAN",
			"paveh-cruise-missile-regiment",
			33.70+0.07*float64(i),
			45.80+0.12*float64(i),
			0,
			240,
			0,
		)
		u.Weapons = []*enginev1.WeaponState{{WeaponId: "ssm-paveh", CurrentQty: 10, MaxQty: 10}}
		u.AttackOrder = attackOrder(iranTargets[(i+1)%len(iranTargets)], enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL)
		units = append(units, u)
	}

	return &enginev1.Scenario{
		Id:             "pg-israel-missile-defense-saturation",
		Name:           "Proving Ground: Israel Missile Defense Saturation",
		Description:    "Large Iranian ballistic and cruise-missile salvo against Israel with layered Arrow, David's Sling, and Iron Dome defenses plus Israeli counterstrikes against launchers.",
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
			{FromCountry: "ISR", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true},
			{FromCountry: "IRN", ToCountry: "ISR", AirspaceTransitAllowed: true},
			{FromCountry: "ISR", ToCountry: "JOR", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
			{FromCountry: "ISR", ToCountry: "IRQ", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
		},
		Units: units,
	}
}

func provingGroundFuelLimitedDeepStrike() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-fuel-limited-deep-strike",
		Name:           "Proving Ground: Fuel-Limited Deep Strike",
		Description:    "Low-fuel F-35A launching from Al Udeid against Esfahan. Used to validate simple fuel burn and bingo-fuel return-to-base logic.",
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
			{FromCountry: "USA", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("pg-fuel-al-udeid", "Al Udeid", "U.S. Expeditionary Air Base - Fuel Test", "USA", "COALITION_WEST", "expeditionary-air-base", 25.12, 51.31, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingAircraft("pg-fuel-usa-f35a", "F-35A", "USAF F-35A Low-Fuel Strike", "USA", "COALITION_WEST", "f35a-lightning", 25.12, 51.31, 0, 90, 0)
				u.HostBaseId = "pg-fuel-al-udeid"
				u.LoadoutConfigurationId = "strike"
				u.Status.FuelLevelLiters = 1200
				return u
			}(),
			provingGroundUnit("pg-fuel-esfahan-airbase", "Esfahan AB", "Iranian Interior Strategic Air Base - Fuel Test", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 32.75, 51.86, 0, 0, 0),
		},
	}
}

func provingGroundSortieRegenerationCycle() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-sortie-regeneration-cycle",
		Name:           "Proving Ground: Sortie Regeneration Cycle",
		Description:    "Single hosted strike aircraft launches from Al Udeid, attacks Bushehr, returns home, and has enough scenario time to refuel/rearm and become ready again. A second Iranian fixed target remains available for manual re-tasking after replenishment.",
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
			{FromCountry: "USA", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true},
		},
		Units: []*enginev1.Unit{
			provingGroundUnit("pg-src-al-udeid", "Al Udeid", "U.S. Expeditionary Air Base - Sortie Cycle", "USA", "COALITION_WEST", "expeditionary-air-base", 25.12, 51.31, 0, 0, 0),
			func() *enginev1.Unit {
				u := provingAircraft("pg-src-usa-f15e", "F-15E", "USAF F-15E Sortie Cycle", "USA", "COALITION_WEST", "f15e-strike-eagle", 25.12, 51.31, 0, 90, 0)
				u.HostBaseId = "pg-src-al-udeid"
				u.LoadoutConfigurationId = "deep_strike"
				u.Status.FuelLevelLiters = 6500
				return u
			}(),
			provingGroundUnit("pg-src-bushehr-airbase", "Bushehr AB", "Iranian Strategic Air Base - Sortie Cycle Primary", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 28.95, 50.83, 0, 0, 0),
			provingGroundUnit("pg-src-shiraz-airbase", "Shiraz AB", "Iranian Strategic Air Base - Sortie Cycle Secondary", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 29.54, 52.59, 0, 0, 0),
		},
	}
}
