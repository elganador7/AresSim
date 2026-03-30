package scenario

import (
	"fmt"
	"math"
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func DestroyerVisualScaleScenario(count int, columns int, spacingKm float64) *enginev1.Scenario {
	if count <= 0 {
		count = 256
	}
	if columns <= 0 {
		columns = 16
	}
	if spacingKm <= 0 {
		spacingKm = 6
	}

	baseLat := 22.4
	baseLon := 60.2
	latStep := spacingKm / 111.0
	lonStep := spacingKm / (111.0 * math.Cos(baseLat*math.Pi/180.0))

	units := make([]*enginev1.Unit, 0, count)
	for i := 0; i < count; i++ {
		row := i / columns
		col := i % columns
		lat := baseLat + float64(row)*latStep
		lon := baseLon + float64(col)*lonStep
		units = append(units, &enginev1.Unit{
			Id:             fmt.Sprintf("vs-ddg51-%03d", i+1),
			DisplayName:    fmt.Sprintf("DDG-%03d", 101+i),
			FullName:       fmt.Sprintf("USS Arleigh Burke Visual %03d", i+1),
			TeamId:         "USA",
			CoalitionId:    "COALITION_WEST",
			DefinitionId:   "ddg51-flight-iia",
			NatoSymbolSidc: "SFSPCLDD--E----",
			Position: &enginev1.Position{
				Lat:     lat,
				Lon:     lon,
				AltMsl:  0,
				Heading: 90,
				Speed:   0,
			},
			Status: &enginev1.OperationalStatus{
				PersonnelStrength:   1.0,
				EquipmentStrength:   1.0,
				CombatEffectiveness: 1.0,
				FuelLevelLiters:     1_800_000,
				Morale:              0.98,
				Fatigue:             0.02,
				IsActive:            true,
			},
		})
	}

	return &enginev1.Scenario{
		Id:             "visual-showcase-ddg51-wall",
		Name:           "Visual Showcase: DDG-51 Wall",
		Description:    fmt.Sprintf("A large open-water visual scale scenario with %d Arleigh Burke-class destroyers for inspecting 3D ship rendering at density.", count),
		Classification: "UNCLASSIFIED // VISUAL SHOWCASE",
		Author:         "AresSim Visual Lab",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: &enginev1.MapSettings{
			InitialWeather: &enginev1.WeatherConditions{
				State:        enginev1.WeatherState_WEATHER_CLEAR,
				VisibilityKm: 50,
				WindSpeedMps: 4,
				TemperatureC: 28,
			},
		},
		Units: units,
	}
}

