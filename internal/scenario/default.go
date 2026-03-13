// Package scenario provides built-in scenarios for AresSim.
package scenario

import (
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// Default returns the built-in Mediterranean Exercise scenario.
// Two US Navy destroyers operating in the Eastern Mediterranean.
// Used as the startup scenario when no saved scenario exists.
func Default() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "00000000-0000-0000-0000-000000000001",
		Name:           "Mediterranean Exercise Alpha",
		Description:    "Two Arleigh Burke-class destroyers conducting freedom of navigation operations in the Eastern Mediterranean.",
		Classification: "UNCLASSIFIED // EXERCISE USE ONLY",
		Author:         "AresSim Default",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2025, 6, 1, 6, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: &enginev1.MapSettings{
			InitialWeather: &enginev1.WeatherConditions{
				State:        enginev1.WeatherState_WEATHER_CLEAR,
				VisibilityKm: 40.0,
				WindSpeedMps: 5.0,
				TemperatureC: 22.0,
			},
		},
		Units: []*enginev1.Unit{
			{
				Id:             "ddg-51-burke",
				DisplayName:    "DDG-51",
				FullName:       "USS Arleigh Burke (DDG-51)",
				Side:           "Blue",
				Domain:         enginev1.UnitDomain_DOMAIN_SEA,
				Type:           enginev1.UnitType_UNIT_TYPE_DESTROYER,
				NatoSymbolSidc: "SFSPCLDD--E----",
				Position: &enginev1.Position{
					Lat:     36.20,
					Lon:     28.10,
					AltMsl:  0,
					Heading: 270,
					Speed:   8.2, // ~16 knots in m/s
				},
				Status: &enginev1.OperationalStatus{
					PersonnelStrength:   1.0,
					EquipmentStrength:   1.0,
					CombatEffectiveness: 1.0,
					FuelLevelLiters:     1_800_000,
					Morale:              0.97,
					Fatigue:             0.03,
					IsActive:            true,
				},
			},
			{
				Id:             "ddg-67-cole",
				DisplayName:    "DDG-67",
				FullName:       "USS Cole (DDG-67)",
				Side:           "Blue",
				Domain:         enginev1.UnitDomain_DOMAIN_SEA,
				Type:           enginev1.UnitType_UNIT_TYPE_DESTROYER,
				NatoSymbolSidc: "SFSPCLDD--E----",
				Position: &enginev1.Position{
					Lat:     35.40,
					Lon:     23.50,
					AltMsl:  0,
					Heading: 90,
					Speed:   7.7, // ~15 knots in m/s
				},
				Status: &enginev1.OperationalStatus{
					PersonnelStrength:   0.95,
					EquipmentStrength:   0.92,
					CombatEffectiveness: 0.88,
					FuelLevelLiters:     1_620_000,
					Morale:              0.91,
					Fatigue:             0.09,
					IsActive:            true,
				},
			},
		},
	}
}
