package scenario

import (
	"strings"
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// IranCoalitionWarSkeleton is a reviewable starting point for the current war.
// It is intentionally incomplete: the goal is to establish the major force
// packages, theater geometry, and side structure before filling out the full
// order of battle.
func IranCoalitionWarSkeleton() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "00000000-0000-0000-0000-000000000002",
		Name:           "Iran War 2026 Skeleton",
		Description:    "Operational skeleton for an Iran versus U.S./Israeli-led coalition conflict, starting at the opening coalition strike. Built to review theater geometry, force packages, and initial basing before full order-of-battle expansion.",
		Classification: "UNCLASSIFIED // SCENARIO DESIGN DRAFT",
		Author:         "AresSim Default",
		Version:        "0.1.0",
		StartTimeUnix:  float64(time.Date(2026, 2, 28, 1, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: &enginev1.MapSettings{
			InitialWeather: &enginev1.WeatherConditions{
				State:         enginev1.WeatherState_WEATHER_CLEAR,
				VisibilityKm:  55,
				WindSpeedMps:  6,
				TemperatureC:  23,
				CloudCeilingM: 6000,
			},
		},
		Relationships: iranWarDayOneRelationships(),
		OpeningStrikeActions: []*enginev1.OpeningStrikeAction{
			openingStrike("isr-f35i-nevatim", "irn-s300-tehran", "air_superiority", enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL, "Israeli F-35I package opens the war against Tehran air defenses."),
			openingStrike("isr-f15i-hatzor", "irn-qiam-central", "close_air_support", enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY, "Israeli F-15I strike package targets Iranian ballistic-missile forces."),
			openingStrike("isr-f16i-ramon", "irn-bavar-esfahan", "sead", enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL, "Israeli F-16I SEAD package suppresses Esfahan air defenses."),
			openingStrike("usa-f35a-al-udeid", "irn-khordad-bushehr", "air_superiority", enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL, "U.S. F-35A package attacks Bushehr-sector air defenses."),
			openingStrike("usa-f15e-al-dhafra", "irn-paveh-south", "close_air_support", enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY, "U.S. F-15E package strikes southern Iranian cruise-missile forces."),
			openingStrike("usa-b1b-diego-garcia", "irn-kheibar-west", "default", enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY, "U.S. B-1B long-range strike hits western Iranian missile brigades."),
		},
		Units: []*enginev1.Unit{
			// Core airbases used for first-wave sortie generation and later turnaround.
			scenarioUnit("isr-airbase-nevatim", "Nevatim AB", "Israeli Strategic Air Base - Nevatim", "Blue", "israel-strategic-airbase", 31.21, 35.01, 0, 0, 0),
			scenarioUnit("isr-airbase-hatzor", "Hatzor AB", "Israeli Strategic Air Base - Hatzor", "Blue", "israel-strategic-airbase", 31.73, 34.72, 0, 0, 0),
			scenarioUnit("isr-airbase-ramon", "Ramon AB", "Israeli Strategic Air Base - Ramon", "Blue", "israel-strategic-airbase", 30.61, 34.78, 0, 0, 0),
			scenarioUnit("qat-airbase-al-udeid", "Al Udeid AB", "Qatari Expeditionary Air Base - Al Udeid", "Blue", "qatari-expeditionary-airbase", 25.12, 51.31, 0, 0, 0),
			scenarioUnit("uae-airbase-al-dhafra", "Al Dhafra AB", "Emirati Strategic Air Base - Al Dhafra", "Blue", "emirati-strategic-airbase", 24.25, 54.55, 0, 0, 0),
			scenarioUnit("sau-airbase-khamis", "Khamis Mushait AB", "Saudi Strategic Air Base - Khamis Mushait", "Blue", "saudi-strategic-airbase", 18.30, 42.80, 0, 0, 0),
			scenarioUnit("omn-airbase-seeb", "Seeb AB", "Omani Maritime Air Base - Seeb", "Blue", "omani-maritime-airbase", 23.59, 58.28, 0, 0, 0),
			scenarioUnit("bhr-airbase-isa", "Isa AB", "Bahraini Air Base - Isa", "Blue", "bahraini-airbase", 26.27, 50.63, 0, 0, 0),
			scenarioUnit("jor-airbase-central", "Jordan AB", "Jordanian Air Base - Central Sector", "Blue", "jordanian-airbase", 31.72, 35.99, 0, 0, 0),
			scenarioUnit("usa-airbase-diego-garcia", "Diego Garcia AB", "U.S. Expeditionary Air Base - Diego Garcia", "Blue", "expeditionary-air-base", -7.31, 72.41, 0, 0, 0),
			scenarioUnit("irn-airbase-tehran", "Tehran AB", "Iranian Strategic Air Base - Tehran", "Red", "iran-strategic-airbase", 35.69, 51.31, 0, 0, 0),
			scenarioUnit("irn-airbase-bandar-abbas", "Bandar Abbas AB", "Iranian Strategic Air Base - Bandar Abbas", "Red", "iran-strategic-airbase", 27.22, 56.38, 0, 0, 0),

			// Israel: homeland air defense, long-range strike, ISR, and maritime screen.
			scenarioAircraft("isr-f35i-nevatim", "Adir 101", "F-35I Adir 101st Squadron", "Blue", "f35i-adir", "isr-airbase-nevatim", 31.21, 35.01, 0, 270, 0),
			scenarioAircraft("isr-f15i-hatzor", "Ra'am 69", "F-15I Ra'am 69 Squadron", "Blue", "f15i-raam", "isr-airbase-hatzor", 31.73, 34.72, 0, 270, 0),
			scenarioAircraft("isr-f16i-ramon", "Sufa 119", "F-16I Sufa 119 Squadron", "Blue", "f16i-sufa", "isr-airbase-ramon", 30.61, 34.78, 0, 270, 0),
			scenarioAircraft("isr-eitam-central", "Eitam", "G550 Eitam AEW&C", "Blue", "g550-eitam", "isr-airbase-nevatim", 31.99, 34.90, 9000, 90, 210),
			scenarioAircraft("isr-oron-central", "Oron", "G550 Oron ISR Aircraft", "Blue", "g550-oron", "isr-airbase-hatzor", 31.90, 34.65, 9000, 100, 210),
			scenarioAircraft("isr-reem-support", "Re'em", "Boeing 707 Re'em Tanker", "Blue", "boeing707-reem", "isr-airbase-nevatim", 31.60, 34.85, 8500, 110, 190),
			scenarioUnit("isr-arrow3-palmachim", "Arrow-3 Palmachim", "Arrow-3 Battery - Palmachim Sector", "Blue", "arrow3-battery", 31.93, 34.69, 0, 0, 0),
			scenarioUnit("isr-davids-sling-dan", "David's Sling Dan", "David's Sling Battery - Dan Region", "Blue", "davids-sling-battery", 32.08, 34.86, 0, 0, 0),
			scenarioUnit("isr-iron-dome-haifa", "Iron Dome Haifa", "Iron Dome Battery - Haifa", "Blue", "iron-dome-battery", 32.82, 35.02, 0, 0, 0),
			scenarioUnit("isr-saar6-eastern-med", "Sa'ar 6-1", "Sa'ar 6 Corvette - Eastern Mediterranean Screen", "Blue", "saar6-corvette", 33.40, 34.40, 0, 180, 8),

			// United States: carrier and long-range strike posture plus Gulf air and missile defense.
			scenarioUnit("usa-cvn78-redsea", "CVN-78", "USS Gerald R. Ford Carrier Strike Group", "Blue", "cvn78-ford", 24.20, 36.90, 0, 330, 10),
			scenarioAircraft("usa-f35a-al-udeid", "F-35A Al Udeid", "F-35A Lightning II Detachment - Al Udeid", "Blue", "f35a-lightning", "qat-airbase-al-udeid", 25.12, 51.31, 0, 0, 0),
			scenarioAircraft("usa-f15e-al-dhafra", "F-15E Dhafra", "F-15E Strike Eagle Detachment - Al Dhafra", "Blue", "f15e-strike-eagle", "uae-airbase-al-dhafra", 24.25, 54.55, 0, 0, 0),
			scenarioAircraft("usa-b1b-diego-garcia", "B-1B DG", "B-1B Lancer Detachment - Diego Garcia", "Blue", "b1b-lancer", "usa-airbase-diego-garcia", -7.31, 72.41, 0, 0, 0),
			scenarioAircraft("usa-kc46-gulf", "KC-46 Gulf", "KC-46A Pegasus Gulf Tanker Orbit", "Blue", "kc46a-pegasus", "qat-airbase-al-udeid", 26.20, 51.80, 8500, 315, 210),
			scenarioUnit("usa-thaad-uae", "THAAD UAE", "THAAD Battery - UAE", "Blue", "thaad-battery", 24.43, 54.65, 0, 0, 0),
			scenarioUnit("usa-patriot-kuwait", "Patriot KWT", "Patriot Battery - Kuwait", "Blue", "patriot-kuwait", 29.22, 47.98, 0, 0, 0),
			scenarioUnit("usa-patriot-qatar", "Patriot QAT", "Patriot PAC-3 Battery - Qatar", "Blue", "patriot-pac3-battery", 25.10, 51.36, 0, 0, 0),

			// Gulf coalition: key partner airpower, AEW, and sea control.
			scenarioAircraft("sau-f15sa-khamis", "F-15SA Khamis", "F-15SA Wing - King Khalid Air Base", "Blue", "f15sa-strike-eagle", "sau-airbase-khamis", 18.30, 42.80, 0, 45, 0),
			scenarioAircraft("sau-e3a-riyadh", "E-3A RSAF", "E-3A Sentry - Saudi AEW", "Blue", "e3a-sentry-saudi", "sau-airbase-khamis", 24.95, 46.72, 8500, 70, 200),
			scenarioAircraft("uae-f16-block60", "Desert Falcon", "F-16E/F Block 60 Desert Falcon Wing", "Blue", "f16e-desert-falcon", "uae-airbase-al-dhafra", 24.25, 54.55, 0, 0, 0),
			scenarioAircraft("uae-globaleye", "GlobalEye", "GlobalEye UAE AEW/ISR", "Blue", "globaleye-uae", "uae-airbase-al-dhafra", 24.30, 54.40, 9000, 60, 205),
			scenarioAircraft("qat-f15qa", "F-15QA", "F-15QA Ababil Wing", "Blue", "f15qa-ababil", "qat-airbase-al-udeid", 25.12, 51.31, 0, 0, 0),
			scenarioAircraft("omn-f16-seeb", "F-16 Oman", "F-16C/D Block 50 Wing - Oman", "Blue", "f16c-oman", "omn-airbase-seeb", 23.59, 58.28, 0, 330, 0),
			scenarioAircraft("bhr-f16v-isa", "F-16V Bahrain", "F-16V Viper Squadron - Bahrain", "Blue", "f16v-viper", "bhr-airbase-isa", 26.27, 50.63, 0, 0, 0),
			scenarioAircraft("jord-f16-central", "F-16 Jordan", "F-16AM/BM Jordanian Air Defense Squadron", "Blue", "f16-jordan", "jor-airbase-central", 31.72, 35.99, 0, 90, 0),

			// Iran: layered IADS, missile forces, strike aircraft, and Gulf denial.
			scenarioUnit("irn-s300-tehran", "S-300 Tehran", "S-300PMU-2 Battery - Tehran", "Red", "s300pmu2-battery-iran", 35.70, 51.40, 0, 0, 0),
			scenarioUnit("irn-bavar-esfahan", "Bavar Esfahan", "Bavar-373 Battery - Esfahan", "Red", "bavar373-battery", 32.65, 51.67, 0, 0, 0),
			scenarioUnit("irn-khordad-bushehr", "3rd Khordad", "3rd Khordad Battery - Bushehr Sector", "Red", "third-khordad-battery", 28.95, 50.84, 0, 0, 0),
			scenarioUnit("irn-tor-natanz", "Tor Natanz", "Tor-M1 Battery - Natanz", "Red", "tor-m1-battery-iran", 33.72, 51.72, 0, 0, 0),
			scenarioUnit("irn-qiam-central", "Qiam Brigade", "Qiam-1 Missile Brigade", "Red", "qiam1-missile-brigade", 34.10, 49.70, 0, 0, 0),
			scenarioUnit("irn-kheibar-west", "Kheibar Brigade", "Kheibar Shekan Brigade", "Red", "kheibar-shekan-brigade", 35.20, 46.98, 0, 0, 0),
			scenarioUnit("irn-paveh-south", "Paveh Regiment", "Paveh Cruise Missile Regiment", "Red", "paveh-cruise-missile-regiment", 27.90, 56.15, 0, 0, 0),
			scenarioUnit("irn-shahed-central", "Shahed Grp", "Shahed-136 Strike Group", "Red", "shahed136-strike-group", 31.40, 54.50, 0, 0, 0),
			scenarioUnit("irn-arash-west", "Arash-2", "Arash-2 Strike Group", "Red", "arash2-strike-group", 34.35, 47.20, 0, 0, 0),
			scenarioAircraft("irn-f14-tehran", "F-14A Tehran", "F-14A Tomcat Interceptor Detachment", "Red", "f14a-tomcat-iriaf", "irn-airbase-tehran", 35.69, 51.31, 0, 250, 0),
			scenarioAircraft("irn-f4-bandar-abbas", "F-4E Abbas", "F-4E Phantom Maritime Strike Detachment", "Red", "f4e-phantom-iriaf", "irn-airbase-bandar-abbas", 27.22, 56.38, 0, 130, 0),
			scenarioUnit("irn-soleimani-hormuz", "Shahid Soleimani", "Shahid Soleimani Corvette - Hormuz Patrol", "Red", "shahid-soleimani-corvette", 26.70, 56.05, 0, 110, 12),
			scenarioUnit("irn-swarm-qeshm", "IRGCN Swarm", "IRGCN Swarm Attack Group - Qeshm Axis", "Red", "irgcn-swarm-group", 26.88, 55.95, 0, 95, 14),
			scenarioUnit("irn-jamaran-bushehr", "Jamaran", "Jamaran Frigate - Bushehr Patrol", "Red", "jamaran-frigate", 28.75, 50.55, 0, 140, 10),
			scenarioUnit("irn-ghadir-sub", "Ghadir Sub", "Ghadir Midget Submarine - Strait Ambush Line", "Red", "ghadir-midget-submarine", 26.45, 56.30, -20, 90, 4),
		},
	}
}

func openingStrike(unitID, targetUnitID, loadoutID string, desiredEffect enginev1.DesiredEffect, narrative string) *enginev1.OpeningStrikeAction {
	return &enginev1.OpeningStrikeAction{
		UnitId:                 unitID,
		TargetUnitId:           targetUnitID,
		LoadoutConfigurationId: loadoutID,
		DesiredEffect:          desiredEffect,
		Narrative:              narrative,
	}
}

func scenarioAircraft(id, displayName, fullName, side, definitionID, hostBaseID string, lat, lon, altMsl, heading, speed float64) *enginev1.Unit {
	u := scenarioUnit(id, displayName, fullName, side, definitionID, lat, lon, altMsl, heading, speed)
	u.HostBaseId = hostBaseID
	return u
}

func iranWarDayOneRelationships() []*enginev1.CountryRelationship {
	return []*enginev1.CountryRelationship{
		relationship("USA", "ISR", true, true, true, true, true, true),
		relationship("ISR", "USA", true, true, true, true, true, true),

		// Day-one Gulf posture: host countries support transit and defensive
		// presence for U.S. forces, but do not openly grant strike access.
		relationship("USA", "BHR", false, true, false, true, true, false),
		relationship("BHR", "USA", false, true, false, true, true, false),
		relationship("USA", "QAT", false, true, false, true, true, false),
		relationship("QAT", "USA", false, true, false, true, true, false),
		relationship("USA", "ARE", false, true, false, true, true, false),
		relationship("ARE", "USA", false, true, false, true, true, false),
		relationship("USA", "SAU", false, true, false, true, true, false),
		relationship("SAU", "USA", false, true, false, true, true, false),
	}
}

func relationship(from, to string, shareIntel, transit, strike, defensive, maritimeTransit, maritimeStrike bool) *enginev1.CountryRelationship {
	return &enginev1.CountryRelationship{
		FromCountry:                 from,
		ToCountry:                   to,
		ShareIntel:                  shareIntel,
		AirspaceTransitAllowed:      transit,
		AirspaceStrikeAllowed:       strike,
		DefensivePositioningAllowed: defensive,
		MaritimeTransitAllowed:      maritimeTransit,
		MaritimeStrikeAllowed:       maritimeStrike,
	}
}

func scenarioUnit(id, displayName, fullName, side, definitionID string, lat, lon, altMsl, heading, speed float64) *enginev1.Unit {
	teamID := inferScenarioTeamID(id, side)
	return &enginev1.Unit{
		Id:           id,
		DisplayName:  displayName,
		FullName:     fullName,
		Side:         side,
		TeamId:       teamID,
		CoalitionId:  side,
		DefinitionId: definitionID,
		Position: &enginev1.Position{
			Lat:     lat,
			Lon:     lon,
			AltMsl:  altMsl,
			Heading: heading,
			Speed:   speed,
		},
		Status: &enginev1.OperationalStatus{
			PersonnelStrength:   1.0,
			EquipmentStrength:   1.0,
			CombatEffectiveness: 1.0,
			FuelLevelLiters:     500000,
			Morale:              0.92,
			Fatigue:             0.08,
			IsActive:            true,
		},
	}
}

func inferScenarioTeamID(id, side string) string {
	prefix := strings.ToUpper(strings.Split(id, "-")[0])
	if len(prefix) == 3 {
		return prefix
	}
	return strings.ToUpper(side)
}
