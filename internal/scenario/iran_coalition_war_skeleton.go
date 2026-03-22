package scenario

import (
	"strings"
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

// IranCoalitionWarSkeleton is a reviewable starting point for the current war.
// It is intentionally incomplete: the goal is to establish the major force
// packages, theater geometry, and country structure before filling out the full
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
		Units: []*enginev1.Unit{
			// Core airbases used for first-wave sortie generation and later turnaround.
			scenarioUnit("isr-airbase-nevatim", "Nevatim AB", "Israeli Strategic Air Base - Nevatim", "Blue", "israel-strategic-airbase", 31.21, 35.01, 0, 0, 0),
			scenarioUnit("isr-airbase-hatzor", "Hatzor AB", "Israeli Strategic Air Base - Hatzor", "Blue", "israel-strategic-airbase", 31.73, 34.72, 0, 0, 0),
			scenarioUnit("isr-airbase-ramon", "Ramon AB", "Israeli Strategic Air Base - Ramon", "Blue", "israel-strategic-airbase", 30.61, 34.78, 0, 0, 0),
			scenarioUnit("isr-airbase-palmachim", "Palmachim AB", "Israeli Strategic Air Base - Palmachim", "Blue", "israel-strategic-airbase", 31.89, 34.69, 0, 0, 0),
			scenarioUnit("isr-airbase-telnof", "Tel Nof AB", "Israeli Strategic Air Base - Tel Nof", "Blue", "israel-strategic-airbase", 31.84, 34.82, 0, 0, 0),
			scenarioUnit("isr-airbase-ramat-david", "Ramat David AB", "Israeli Strategic Air Base - Ramat David", "Blue", "israel-strategic-airbase", 32.67, 35.18, 0, 0, 0),
			scenarioUnit("qat-airbase-al-udeid", "Al Udeid AB", "Qatari Expeditionary Air Base - Al Udeid", "Blue", "qatari-expeditionary-airbase", 25.12, 51.31, 0, 0, 0),
			scenarioUnit("uae-airbase-al-dhafra", "Al Dhafra AB", "Emirati Strategic Air Base - Al Dhafra", "Blue", "emirati-strategic-airbase", 24.25, 54.55, 0, 0, 0),
			scenarioUnit("sau-airbase-khamis", "Khamis Mushait AB", "Saudi Strategic Air Base - Khamis Mushait", "Blue", "saudi-strategic-airbase", 18.30, 42.80, 0, 0, 0),
			scenarioUnit("sau-airbase-prince-sultan", "Prince Sultan AB", "Saudi Strategic Air Base - Prince Sultan", "Blue", "saudi-strategic-airbase", 24.06, 47.58, 0, 0, 0),
			scenarioUnit("omn-airbase-seeb", "Seeb AB", "Omani Maritime Air Base - Seeb", "Blue", "omani-maritime-airbase", 23.59, 58.28, 0, 0, 0),
			scenarioUnit("bhr-airbase-isa", "Isa AB", "Bahraini Air Base - Isa", "Blue", "bahraini-airbase", 26.27, 50.63, 0, 0, 0),
			scenarioUnit("kwt-airbase-ahmad", "Ahmad al-Jaber AB", "Kuwaiti Air Base - Ahmad al-Jaber", "Blue", "kuwaiti-airbase", 29.22, 47.98, 0, 0, 0),
			scenarioUnit("jor-airbase-central", "Jordan AB", "Jordanian Air Base - Central Sector", "Blue", "jordanian-airbase", 31.72, 35.99, 0, 0, 0),
			scenarioUnit("usa-airbase-diego-garcia", "Diego Garcia AB", "U.S. Expeditionary Air Base - Diego Garcia", "Blue", "expeditionary-air-base", -7.31, 72.41, 0, 0, 0),
			scenarioUnit("irn-airbase-tehran", "Tehran AB", "Iranian Strategic Air Base - Tehran", "Red", "iran-strategic-airbase", 35.69, 51.31, 0, 0, 0),
			scenarioUnit("irn-airbase-bandar-abbas", "Bandar Abbas AB", "Iranian Strategic Air Base - Bandar Abbas", "Red", "iran-strategic-airbase", 27.22, 56.38, 0, 0, 0),
			scenarioUnit("irn-airbase-esfahan", "Esfahan AB", "Iranian Strategic Air Base - Esfahan", "Red", "iran-strategic-airbase", 32.75, 51.86, 0, 0, 0),
			scenarioUnit("irn-airbase-tabriz", "Tabriz AB", "Iranian Strategic Air Base - Tabriz", "Red", "iran-strategic-airbase", 38.13, 46.24, 0, 0, 0),
			scenarioUnit("irn-airbase-shiraz", "Shiraz AB", "Iranian Strategic Air Base - Shiraz", "Red", "iran-strategic-airbase", 29.54, 52.59, 0, 0, 0),
			scenarioUnit("irn-airbase-bushehr", "Bushehr AB", "Iranian Strategic Air Base - Bushehr", "Red", "iran-strategic-airbase", 28.95, 50.84, 0, 0, 0),

			// Israel: long-range strike, ISR, layered homeland defense, and eastern Mediterranean sea control.
			scenarioAircraft("isr-f35i-nevatim", "Adir 101", "F-35I Adir 101st Squadron", "Blue", "f35i-adir", "isr-airbase-nevatim", 31.21, 35.01, 0, 270, 0),
			scenarioAircraft("isr-f15i-hatzor", "Ra'am 69", "F-15I Ra'am 69 Squadron", "Blue", "f15i-raam", "isr-airbase-hatzor", 31.73, 34.72, 0, 270, 0),
			scenarioAircraft("isr-f16i-ramon", "Sufa 119", "F-16I Sufa 119 Squadron", "Blue", "f16i-sufa", "isr-airbase-ramon", 30.61, 34.78, 0, 270, 0),
			scenarioAircraft("isr-f35i-nevatim-2", "Adir 140", "F-35I Adir Follow-on Strike Detachment", "Blue", "f35i-adir", "isr-airbase-nevatim", 31.22, 35.02, 0, 270, 0),
			scenarioAircraft("isr-f15i-hatzor-2", "Ra'am 133", "F-15I Ra'am Deep Strike Detachment", "Blue", "f15i-raam", "isr-airbase-hatzor", 31.74, 34.73, 0, 270, 0),
			scenarioAircraft("isr-f16i-ramon-2", "Sufa 201", "F-16I Sufa Follow-on Strike Squadron", "Blue", "f16i-sufa", "isr-airbase-ramon", 30.62, 34.79, 0, 270, 0),
			scenarioAircraft("isr-f35i-nevatim-3", "Adir 116", "F-35I Adir Second-Wave Strike Squadron", "Blue", "f35i-adir", "isr-airbase-nevatim", 31.23, 35.03, 0, 270, 0),
			scenarioAircraft("isr-f35i-nevatim-4", "Adir 117", "F-35I Adir Deep Penetration Strike Squadron", "Blue", "f35i-adir", "isr-airbase-nevatim", 31.24, 35.04, 0, 270, 0),
			scenarioAircraft("isr-f15i-hatzor-3", "Ra'am 201", "F-15I Ra'am Long-Range Escort Squadron", "Blue", "f15i-raam", "isr-airbase-hatzor", 31.75, 34.74, 0, 270, 0),
			scenarioAircraft("isr-f16i-ramon-3", "Sufa 253", "F-16I Sufa Follow-up Strike Squadron", "Blue", "f16i-sufa", "isr-airbase-ramon", 30.63, 34.80, 0, 270, 0),
			scenarioAircraft("isr-f16i-ramat-david", "Sufa North", "F-16I Sufa Northern Air Defense Squadron", "Blue", "f16i-sufa", "isr-airbase-ramat-david", 32.67, 35.18, 0, 250, 0),
			scenarioAircraft("isr-f16i-telnof", "Sufa Center", "F-16I Sufa Central Reserve Squadron", "Blue", "f16i-sufa", "isr-airbase-telnof", 31.84, 34.82, 0, 260, 0),
			scenarioAircraft("isr-eitam-central", "Eitam", "G550 Eitam AEW&C", "Blue", "g550-eitam", "isr-airbase-nevatim", 31.99, 34.90, 9000, 90, 210),
			scenarioAircraft("isr-oron-central", "Oron", "G550 Oron ISR Aircraft", "Blue", "g550-oron", "isr-airbase-hatzor", 31.90, 34.65, 9000, 100, 210),
			scenarioAircraft("isr-reem-support", "Re'em", "Boeing 707 Re'em Tanker", "Blue", "boeing707-reem", "isr-airbase-nevatim", 31.60, 34.85, 8500, 110, 190),
			scenarioAircraft("isr-eitam-central-2", "Eitam 2", "G550 Eitam AEW&C - National Reserve Orbit", "Blue", "g550-eitam", "isr-airbase-telnof", 31.95, 34.88, 9000, 85, 210),
			scenarioAircraft("isr-oron-central-2", "Oron 2", "G550 Oron ISR Aircraft - Persistent Recon Orbit", "Blue", "g550-oron", "isr-airbase-nevatim", 31.85, 34.95, 9000, 95, 210),
			scenarioAircraft("isr-reem-support-2", "Re'em 2", "Boeing 707 Re'em Secondary Tanker Track", "Blue", "boeing707-reem", "isr-airbase-nevatim", 31.58, 34.87, 8500, 115, 190),
			scenarioAircraft("isr-heron-nevatim", "Heron TP", "Heron TP Eitan Long-Endurance ISR Detachment", "Blue", "heron-tp-eitan", "isr-airbase-nevatim", 31.26, 35.05, 0, 270, 0),
			scenarioAircraft("isr-heron-palmachim", "Heron TP South", "Heron TP Eitan Southern ISR Detachment", "Blue", "heron-tp-eitan", "isr-airbase-palmachim", 31.89, 34.70, 0, 270, 0),
			scenarioAircraft("isr-hermes900-palmachim", "Hermes 900", "Hermes 900 Kochav ISR Squadron", "Blue", "hermes-900-kochav", "isr-airbase-palmachim", 31.88, 34.68, 0, 270, 0),
			scenarioAircraft("isr-hermes450-ramon", "Hermes 450", "Hermes 450 Tactical Surveillance Flight", "Blue", "hermes-450", "isr-airbase-ramon", 30.60, 34.77, 0, 270, 0),
			scenarioUnit("isr-arrow2-central", "Arrow-2 Central", "Arrow-2 Battery - Central Israel", "Blue", "arrow2-battery", 31.95, 34.78, 0, 0, 0),
			scenarioUnit("isr-arrow3-palmachim", "Arrow-3 Palmachim", "Arrow-3 Battery - Palmachim Sector", "Blue", "arrow3-battery", 31.93, 34.69, 0, 0, 0),
			scenarioUnit("isr-arrow3-north", "Arrow-3 North", "Arrow-3 Battery - Northern Israel", "Blue", "arrow3-battery", 32.73, 35.11, 0, 0, 0),
			scenarioUnit("isr-davids-sling-dan", "David's Sling Dan", "David's Sling Battery - Dan Region", "Blue", "davids-sling-battery", 32.08, 34.86, 0, 0, 0),
			scenarioUnit("isr-davids-sling-south", "David's Sling South", "David's Sling Battery - Negev Sector", "Blue", "davids-sling-battery", 31.24, 34.79, 0, 0, 0),
			scenarioUnit("isr-iron-dome-haifa", "Iron Dome Haifa", "Iron Dome Battery - Haifa", "Blue", "iron-dome-battery", 32.82, 35.02, 0, 0, 0),
			scenarioUnit("isr-iron-dome-dan", "Iron Dome Dan", "Iron Dome Battery - Dan Region", "Blue", "iron-dome-battery", 32.09, 34.82, 0, 0, 0),
			scenarioUnit("isr-iron-dome-negev", "Iron Dome Negev", "Iron Dome Battery - Negev Airbase Defense", "Blue", "iron-dome-battery", 31.04, 34.72, 0, 0, 0),
			scenarioUnit("isr-barak-haifa", "Barak MX Haifa", "Barak MX Battery - Haifa Maritime Shield", "Blue", "barak-mx-battery", 32.84, 35.00, 0, 0, 0),
			scenarioUnit("isr-spyder-nevatim", "SPYDER-MR Nevatim", "SPYDER-MR Battery - Nevatim Air Base Defense", "Blue", "spyder-mr-battery", 31.20, 35.02, 0, 0, 0),
			scenarioUnit("isr-spyder-hatzor", "SPYDER-SR Hatzor", "SPYDER-SR Battery - Hatzor Air Base Defense", "Blue", "spyder-sr-battery", 31.72, 34.74, 0, 0, 0),
			scenarioUnit("isr-c2-central", "National C2", "Israeli National C2 Site - Central Sector", "Blue", "israel-national-c2-site", 31.98, 34.91, 0, 0, 0),
			scenarioUnit("isr-radar-hermon", "Hermon Radar", "Israeli Early Warning Radar - Hermon Sector", "Blue", "israel-early-warning-radar-site", 33.31, 35.78, 0, 0, 0),
			scenarioUnit("isr-radar-negev", "Negev Radar", "Israeli Early Warning Radar - Negev Sector", "Blue", "israel-early-warning-radar-site", 30.98, 34.92, 0, 0, 0),
			scenarioUnit("isr-port-haifa", "Haifa Port", "Israeli Mediterranean Port - Haifa Naval Support Hub", "Blue", "israel-mediterranean-port", 32.83, 35.00, 0, 0, 0),
			scenarioUnit("isr-saar6-eastern-med", "Sa'ar 6-1", "Sa'ar 6 Corvette - Eastern Mediterranean Screen", "Blue", "saar6-corvette", 33.40, 34.40, 0, 180, 8),
			scenarioUnit("isr-saar6-eastern-med-2", "Sa'ar 6-2", "Sa'ar 6 Corvette - Offshore Gas Defense", "Blue", "saar6-corvette", 32.90, 34.05, 0, 180, 8),
			scenarioUnit("isr-saar6-eastern-med-3", "Sa'ar 6-3", "Sa'ar 6 Corvette - Northern Offshore Defense", "Blue", "saar6-corvette", 33.15, 34.18, 0, 180, 8),
			scenarioUnit("isr-saar45-haifa", "Sa'ar 4.5 Haifa", "Sa'ar 4.5 Missile Boat - Haifa Escort Screen", "Blue", "saar45-missile-boat", 32.95, 34.55, 0, 180, 11),
			scenarioUnit("isr-saar45-ashdod", "Sa'ar 4.5 Ashdod", "Sa'ar 4.5 Missile Boat - Southern Littoral Screen", "Blue", "saar45-missile-boat", 32.05, 34.25, 0, 180, 11),
			scenarioUnit("isr-shaldag-haifa", "Shaldag Haifa", "Shaldag Mk V - Haifa Coastal Security Patrol", "Blue", "shaldag-mkv", 32.88, 34.92, 0, 170, 13),
			scenarioUnit("isr-shaldag-ashdod", "Shaldag Ashdod", "Shaldag Mk V - Ashdod Coastal Security Patrol", "Blue", "shaldag-mkv", 31.88, 34.56, 0, 170, 13),
			scenarioUnit("isr-dolphin-med", "Dolphin II", "Dolphin II Submarine - Eastern Mediterranean Patrol", "Blue", "dolphin-ii-submarine", 33.05, 34.20, -30, 165, 5),
			scenarioUnit("isr-dolphin-med-2", "Dolphin II-2", "Dolphin II Submarine - Southern Mediterranean Patrol", "Blue", "dolphin-ii-submarine", 32.58, 34.05, -35, 160, 5),

			// United States: carrier and long-range strike posture plus Gulf air and missile defense.
			scenarioUnit("usa-cvn78-redsea", "CVN-78", "USS Gerald R. Ford Carrier Strike Group", "Blue", "cvn78-ford", 24.20, 36.90, 0, 330, 10),
			scenarioUnit("usa-cvn68-arabian-sea", "CVN-68", "USS Nimitz Carrier Strike Group", "Blue", "cvn68-nimitz", 18.40, 63.90, 0, 60, 10),
			scenarioUnit("usa-ddg51-redsea", "DDG-51", "Arleigh Burke Flight IIA Destroyer - Red Sea Escort", "Blue", "ddg51-flight-iia", 24.55, 36.60, 0, 330, 11),
			scenarioUnit("usa-ddg51-gulf", "DDG-99", "Arleigh Burke Flight IIA Destroyer - Gulf BMD Patrol", "Blue", "ddg51-flight-iia", 26.15, 51.95, 0, 120, 10),
			scenarioUnit("usa-ddg51-arabian-sea", "DDG-112", "Arleigh Burke Flight IIA Destroyer - Arabian Sea Escort", "Blue", "ddg51-flight-iia", 18.10, 64.20, 0, 55, 10),
			scenarioUnit("usa-cg47-redsea", "CG-61", "Ticonderoga-class Cruiser - Red Sea Air Defense Commander", "Blue", "cg47-ticonderoga", 24.35, 36.75, 0, 330, 10),
			scenarioUnit("usa-cg47-gulf", "CG-52", "Ticonderoga-class Cruiser - Gulf BMD Screen", "Blue", "cg47-ticonderoga", 26.40, 52.05, 0, 110, 10),
			scenarioUnit("usa-ohio-arabian-sea", "Ohio SSGN", "Ohio-class SSGN - Arabian Sea Tomahawk Ambush", "Blue", "ohio-ssgn", 17.60, 64.80, -45, 55, 4),
			scenarioUnit("usa-virginia-gulf", "Virginia SSN", "Virginia Block V - Gulf Outer Screen", "Blue", "virginia-block-v", 25.80, 57.20, -40, 105, 4),
			scenarioAircraft("usa-f35c-ford", "F-35C Ford", "F-35C Lightning II - Ford Air Wing", "Blue", "f35c-lightning", "usa-cvn78-redsea", 24.20, 36.90, 0, 0, 0),
			scenarioAircraft("usa-fa18e-ford", "F/A-18E Ford", "F/A-18E Super Hornet - Ford Strike Fighter Squadron", "Blue", "fa18e-super-hornet", "usa-cvn78-redsea", 24.20, 36.90, 0, 0, 0),
			scenarioAircraft("usa-fa18f-ford", "F/A-18F Ford", "F/A-18F Super Hornet - Ford Two-Seat Strike Fighter Squadron", "Blue", "fa18f-super-hornet", "usa-cvn78-redsea", 24.20, 36.90, 0, 0, 0),
			scenarioAircraft("usa-ea18g-ford", "EA-18G Ford", "EA-18G Growler - Ford Electronic Attack Squadron", "Blue", "ea18g-growler", "usa-cvn78-redsea", 24.20, 36.90, 0, 0, 0),
			scenarioAircraft("usa-e2d-ford", "E-2D Ford", "E-2D Advanced Hawkeye - Ford Air Wing", "Blue", "e2d-advanced-hawkeye", "usa-cvn78-redsea", 24.20, 36.90, 0, 0, 0),
			scenarioAircraft("usa-mh60r-ford", "MH-60R Ford", "MH-60R Seahawk - Ford Maritime Helicopter Detachment", "Blue", "mh60r-seahawk", "usa-cvn78-redsea", 24.20, 36.90, 0, 0, 0),
			scenarioAircraft("usa-fa18e-nimitz", "F/A-18E Nimitz", "F/A-18E Super Hornet - Nimitz Strike Fighter Squadron", "Blue", "fa18e-super-hornet", "usa-cvn68-arabian-sea", 18.40, 63.90, 0, 0, 0),
			scenarioAircraft("usa-ea18g-nimitz", "EA-18G Nimitz", "EA-18G Growler - Nimitz Electronic Attack Squadron", "Blue", "ea18g-growler", "usa-cvn68-arabian-sea", 18.40, 63.90, 0, 0, 0),
			scenarioAircraft("usa-e2d-nimitz", "E-2D Nimitz", "E-2D Advanced Hawkeye - Nimitz Air Wing", "Blue", "e2d-advanced-hawkeye", "usa-cvn68-arabian-sea", 18.40, 63.90, 0, 0, 0),
			scenarioAircraft("usa-f35a-al-udeid", "F-35A Al Udeid", "F-35A Lightning II Detachment - Al Udeid", "Blue", "f35a-lightning", "qat-airbase-al-udeid", 25.12, 51.31, 0, 0, 0),
			scenarioAircraft("usa-f35a-dhafra", "F-35A Dhafra", "F-35A Lightning II Detachment - Al Dhafra", "Blue", "f35a-lightning", "uae-airbase-al-dhafra", 24.24, 54.54, 0, 0, 0),
			scenarioAircraft("usa-f15e-al-dhafra", "F-15E Dhafra", "F-15E Strike Eagle Detachment - Al Dhafra", "Blue", "f15e-strike-eagle", "uae-airbase-al-dhafra", 24.25, 54.55, 0, 0, 0),
			scenarioAircraft("usa-f15ex-prince-sultan", "F-15EX Saudi", "F-15EX Eagle II Forward Counter-Air Detachment", "Blue", "f15ex-eagle-ii", "sau-airbase-prince-sultan", 24.06, 47.58, 0, 0, 0),
			scenarioAircraft("usa-f22a-al-udeid", "F-22A Qatar", "F-22A Raptor Air Dominance Detachment", "Blue", "f22a-raptor", "qat-airbase-al-udeid", 25.13, 51.30, 0, 0, 0),
			scenarioAircraft("usa-f22a-al-udeid-2", "F-22A Qatar 2", "F-22A Raptor Follow-on Air Dominance Detachment", "Blue", "f22a-raptor", "qat-airbase-al-udeid", 25.14, 51.32, 0, 0, 0),
			scenarioAircraft("usa-p8a-gulf", "P-8A Gulf", "P-8A Poseidon Gulf Maritime Patrol", "Blue", "p8a-poseidon", "qat-airbase-al-udeid", 25.15, 51.28, 0, 0, 0),
			scenarioAircraft("usa-p8a-dhafra", "P-8A Dhafra", "P-8A Poseidon Arabian Gulf ASW Patrol", "Blue", "p8a-poseidon", "uae-airbase-al-dhafra", 24.27, 54.57, 0, 0, 0),
			scenarioAircraft("usa-c17a-diego-garcia", "C-17A DG", "C-17A Globemaster III Airlift Detachment", "Blue", "c17a-globemaster-iii", "usa-airbase-diego-garcia", -7.30, 72.42, 0, 0, 0),
			scenarioAircraft("usa-c17a-qatar", "C-17A Qatar", "C-17A Globemaster III Theater Lift Detachment", "Blue", "c17a-globemaster-iii", "qat-airbase-al-udeid", 25.10, 51.29, 0, 0, 0),
			scenarioAircraft("usa-b1b-diego-garcia", "B-1B DG", "B-1B Lancer Detachment - Diego Garcia", "Blue", "b1b-lancer", "usa-airbase-diego-garcia", -7.31, 72.41, 0, 0, 0),
			scenarioAircraft("usa-kc46-gulf", "KC-46 Gulf", "KC-46A Pegasus Gulf Tanker Orbit", "Blue", "kc46a-pegasus", "qat-airbase-al-udeid", 26.20, 51.80, 8500, 315, 210),
			scenarioAircraft("usa-kc46-dhafra", "KC-46 Dhafra", "KC-46A Pegasus Southern Gulf Tanker Orbit", "Blue", "kc46a-pegasus", "uae-airbase-al-dhafra", 24.90, 54.90, 8500, 300, 210),
			scenarioAircraft("usa-rq4-prince-sultan", "RQ-4 Gulf", "RQ-4B Global Hawk Strategic ISR Orbit", "Blue", "rq4b-global-hawk", "sau-airbase-prince-sultan", 24.07, 47.57, 0, 0, 0),
			scenarioAircraft("usa-mq9-dhafra", "MQ-9 Dhafra", "MQ-9A Reaper Persistent Strike-ISR Detachment", "Blue", "mq9a-reaper", "uae-airbase-al-dhafra", 24.23, 54.53, 0, 0, 0),
			scenarioUnit("usa-thaad-uae", "THAAD UAE", "THAAD Battery - UAE", "Blue", "thaad-battery", 24.43, 54.65, 0, 0, 0),
			scenarioUnit("usa-patriot-kuwait", "Patriot KWT", "Patriot Battery - Kuwait", "Blue", "patriot-kuwait", 29.22, 47.98, 0, 0, 0),
			scenarioUnit("usa-patriot-qatar", "Patriot QAT", "Patriot PAC-3 Battery - Qatar", "Blue", "patriot-pac3-battery", 25.10, 51.36, 0, 0, 0),
			scenarioUnit("usa-patriot-bahrain", "Patriot BHR", "Patriot PAC-3 Battery - Bahrain", "Blue", "patriot-pac3-battery", 26.14, 50.58, 0, 0, 0),
			scenarioUnit("usa-patriot-saudi", "Patriot SAU", "Patriot PAC-3 Battery - Prince Sultan Sector", "Blue", "patriot-pac3-battery", 24.04, 47.60, 0, 0, 0),

			// Gulf coalition: key partner airpower, AEW, and sea control.
			scenarioAircraft("sau-f15sa-khamis", "F-15SA Khamis", "F-15SA Wing - King Khalid Air Base", "Blue", "f15sa-strike-eagle", "sau-airbase-khamis", 18.30, 42.80, 0, 45, 0),
			scenarioAircraft("sau-f15sa-khamis-2", "F-15SA South", "F-15SA Follow-on Counter-Air Wing", "Blue", "f15sa-strike-eagle", "sau-airbase-khamis", 18.31, 42.81, 0, 45, 0),
			scenarioAircraft("sau-e3a-riyadh", "E-3A RSAF", "E-3A Sentry - Saudi AEW", "Blue", "e3a-sentry-saudi", "sau-airbase-khamis", 24.95, 46.72, 8500, 70, 200),
			scenarioUnit("sau-alriyadh-gulf", "Al Riyadh", "Al Riyadh Frigate - Gulf Air Defense Patrol", "Blue", "al-riyadh-frigate-saudi", 26.95, 50.05, 0, 120, 9),
			scenarioUnit("sau-avante-gulf", "Avante 2200", "Avante 2200 Corvette - Eastern Province Patrol", "Blue", "avante-2200-corvette-saudi", 27.10, 49.95, 0, 120, 11),
			scenarioAircraft("uae-f16-block60", "Desert Falcon", "F-16E/F Block 60 Desert Falcon Wing", "Blue", "f16e-desert-falcon", "uae-airbase-al-dhafra", 24.25, 54.55, 0, 0, 0),
			scenarioAircraft("uae-f16-block60-2", "Desert Falcon 2", "F-16E/F Block 60 Follow-on Strike Wing", "Blue", "f16e-desert-falcon", "uae-airbase-al-dhafra", 24.26, 54.56, 0, 0, 0),
			scenarioAircraft("uae-globaleye", "GlobalEye", "GlobalEye UAE AEW/ISR", "Blue", "globaleye-uae", "uae-airbase-al-dhafra", 24.30, 54.40, 9000, 60, 205),
			scenarioAircraft("uae-dash8-mpa", "Dash-8 MPA", "Dash-8 Maritime Patrol Aircraft - UAE", "Blue", "dash8-mpa-uae", "uae-airbase-al-dhafra", 24.24, 54.54, 0, 0, 0),
			scenarioUnit("uae-baynunah-hormuz", "Baynunah", "Baynunah Corvette - Strait of Hormuz Patrol", "Blue", "baynunah-corvette-uae", 25.85, 55.10, 0, 95, 11),
			scenarioAircraft("qat-f15qa", "F-15QA", "F-15QA Ababil Wing", "Blue", "f15qa-ababil", "qat-airbase-al-udeid", 25.12, 51.31, 0, 0, 0),
			scenarioAircraft("qat-f15qa-2", "F-15QA North", "F-15QA Air Defense Detachment", "Blue", "f15qa-ababil", "qat-airbase-al-udeid", 25.11, 51.30, 0, 0, 0),
			scenarioUnit("qat-doha-corvette", "Doha", "Doha Corvette - Qatari Maritime Screen", "Blue", "doha-corvette-qatar", 25.75, 51.95, 0, 115, 10),
			scenarioAircraft("omn-f16-seeb", "F-16 Oman", "F-16C/D Block 50 Wing - Oman", "Blue", "f16c-oman", "omn-airbase-seeb", 23.59, 58.28, 0, 330, 0),
			scenarioAircraft("omn-cn235-mpa", "CN-235 Oman", "CN-235 Maritime Patrol Aircraft - Oman", "Blue", "cn235-mpa-oman", "omn-airbase-seeb", 23.58, 58.27, 0, 330, 0),
			scenarioAircraft("omn-super-lynx", "Super Lynx", "Super Lynx 300 - Omani Naval Aviation", "Blue", "super-lynx-300-oman", "omn-airbase-seeb", 23.60, 58.29, 0, 330, 0),
			scenarioAircraft("bhr-f16v-isa", "F-16V Bahrain", "F-16V Viper Squadron - Bahrain", "Blue", "f16v-viper", "bhr-airbase-isa", 26.27, 50.63, 0, 0, 0),
			scenarioAircraft("bhr-bell412", "Bell 412", "Bell 412 Utility Helicopter - Bahrain", "Blue", "bell412-bahrain", "bhr-airbase-isa", 26.26, 50.62, 0, 0, 0),
			scenarioAircraft("kwt-fa18-ahmad", "F/A-18 Kuwait", "F/A-18E/F Super Hornet Wing - Kuwait", "Blue", "fa18-super-hornet-kuwait", "kwt-airbase-ahmad", 29.21, 47.97, 0, 45, 0),
			scenarioAircraft("kwt-kc130j-ahmad", "KC-130J Kuwait", "KC-130J Tanker/Transport - Kuwait", "Blue", "kc130j-kuwait", "kwt-airbase-ahmad", 29.20, 47.96, 0, 45, 0),
			scenarioAircraft("jord-f16-central", "F-16 Jordan", "F-16AM/BM Jordanian Air Defense Squadron", "Blue", "f16-jordan", "jor-airbase-central", 31.72, 35.99, 0, 90, 0),

			// Iran: layered IADS, missile forces, strike aircraft, and Gulf denial.
			scenarioUnit("irn-s300-tehran", "S-300 Tehran", "S-300PMU-2 Battery - Tehran", "Red", "s300pmu2-battery-iran", 35.70, 51.40, 0, 0, 0),
			scenarioUnit("irn-bavar-tehran-south", "Bavar Tehran South", "Bavar-373 Battery - Tehran Southern Sector", "Red", "bavar373-battery", 35.45, 51.15, 0, 0, 0),
			scenarioUnit("irn-bavar-esfahan", "Bavar Esfahan", "Bavar-373 Battery - Esfahan", "Red", "bavar373-battery", 32.65, 51.67, 0, 0, 0),
			scenarioUnit("irn-khordad-bushehr", "3rd Khordad", "3rd Khordad Battery - Bushehr Sector", "Red", "third-khordad-battery", 28.95, 50.84, 0, 0, 0),
			scenarioUnit("irn-khordad-tabriz", "3rd Khordad Tabriz", "3rd Khordad Battery - Tabriz Sector", "Red", "third-khordad-battery", 38.02, 46.45, 0, 0, 0),
			scenarioUnit("irn-tor-natanz", "Tor Natanz", "Tor-M1 Battery - Natanz", "Red", "tor-m1-battery-iran", 33.72, 51.72, 0, 0, 0),
			scenarioUnit("irn-tor-bandar-abbas", "Tor Abbas", "Tor-M1 Battery - Bandar Abbas Air Defense", "Red", "tor-m1-battery-iran", 27.21, 56.31, 0, 0, 0),
			scenarioUnit("irn-s300-bandar-abbas", "S-300 Abbas", "S-300PMU-2 Battery - Bandar Abbas", "Red", "s300pmu2-battery-iran", 27.18, 56.25, 0, 0, 0),
			scenarioUnit("irn-khordad-hormuz", "3rd Khordad Hormuz", "3rd Khordad Battery - Strait Sector", "Red", "third-khordad-battery", 27.05, 56.10, 0, 0, 0),
			scenarioUnit("irn-raad-qeshm", "Raad Qeshm", "Raad Coastal Defense Battery - Qeshm Axis", "Red", "raad-coastal-battery", 26.80, 55.85, 0, 0, 0),
			scenarioStrikeUnit("irn-qiam-central", "Qiam Brigade", "Qiam-1 Missile Brigade", "Red", "qiam1-missile-brigade", 34.10, 49.70, 0, 0, 0, 0),
			scenarioStrikeUnit("irn-qiam-northwest", "Qiam NW", "Qiam-1 Missile Brigade - Northwestern Sector", "Red", "qiam1-missile-brigade", 36.15, 47.65, 0, 0, 0, 0),
			scenarioStrikeUnit("irn-kheibar-west", "Kheibar Brigade", "Kheibar Shekan Brigade", "Red", "kheibar-shekan-brigade", 35.20, 46.98, 0, 0, 0, 0),
			scenarioStrikeUnit("irn-fateh-south", "Fateh Brigade", "Fateh-110 Brigade - Southern Theater", "Red", "fateh110-brigade", 29.30, 52.65, 0, 0, 0, 0),
			scenarioStrikeUnit("irn-paveh-south", "Paveh Regiment", "Paveh Cruise Missile Regiment", "Red", "paveh-cruise-missile-regiment", 27.90, 56.15, 0, 0, 0, 0),
			scenarioStrikeUnit("irn-sejjil-central", "Sejjil Brigade", "Sejjil Missile Brigade", "Red", "sejjil-missile-brigade", 33.05, 50.55, 0, 0, 0, 0),
			scenarioStrikeUnit("irn-shahed-central", "Shahed Grp", "Shahed-136 Strike Group", "Red", "shahed136-strike-group", 31.40, 54.50, 0, 0, 0, 0),
			scenarioStrikeUnit("irn-shahed-east", "Shahed East", "Shahed-136 Follow-on Strike Group", "Red", "shahed136-strike-group", 32.20, 55.10, 0, 0, 0, 0),
			scenarioStrikeUnit("irn-arash-west", "Arash-2", "Arash-2 Strike Group", "Red", "arash2-strike-group", 34.35, 47.20, 0, 0, 0, 0),
			scenarioStrikeUnit("irn-shahed129-central", "Shahed-129", "Shahed-129 Armed UAV Squadron", "Red", "shahed129-uav-squadron", 31.60, 54.20, 0, 0, 0, 0),
			scenarioStrikeUnit("irn-mohajer6-central", "Mohajer-6", "Mohajer-6 UAV Squadron", "Red", "mohajer6-uav-squadron", 31.58, 54.18, 0, 0, 0, 0),
			scenarioAircraft("irn-f14-tehran", "F-14A Tehran", "F-14A Tomcat Interceptor Detachment", "Red", "f14a-tomcat-iriaf", "irn-airbase-tehran", 35.69, 51.31, 0, 250, 0),
			scenarioAircraft("irn-f14-tehran-2", "F-14A Tehran 2", "F-14A Tomcat CAP Detachment", "Red", "f14a-tomcat-iriaf", "irn-airbase-tehran", 35.68, 51.30, 0, 250, 0),
			scenarioAircraft("irn-f14-tabriz", "F-14A Tabriz", "F-14A Tomcat Northwestern Interceptor Detachment", "Red", "f14a-tomcat-iriaf", "irn-airbase-tabriz", 38.14, 46.25, 0, 110, 0),
			scenarioAircraft("irn-f14-esfahan", "F-14A Esfahan", "F-14A Tomcat Central Reserve Interceptor", "Red", "f14a-tomcat-iriaf", "irn-airbase-esfahan", 32.74, 51.85, 0, 160, 0),
			scenarioAircraft("irn-f4-bandar-abbas", "F-4E Abbas", "F-4E Phantom Maritime Strike Detachment", "Red", "f4e-phantom-iriaf", "irn-airbase-bandar-abbas", 27.22, 56.38, 0, 130, 0),
			scenarioAircraft("irn-f4-bushehr", "F-4E Bushehr", "F-4E Phantom Gulf Maritime Strike Squadron", "Red", "f4e-phantom-iriaf", "irn-airbase-bushehr", 28.95, 50.84, 0, 160, 0),
			scenarioAircraft("irn-su24-tehran", "Su-24MK Tehran", "Su-24MK Strike Squadron - Central Iran", "Red", "su24mk-strike-squadron-iran", "irn-airbase-tehran", 35.67, 51.29, 0, 250, 0),
			scenarioAircraft("irn-su24-shiraz", "Su-24MK Shiraz", "Su-24MK Southern Strike Squadron", "Red", "su24mk-strike-squadron-iran", "irn-airbase-shiraz", 29.55, 52.60, 0, 140, 0),
			scenarioAircraft("irn-p3f-bandar-abbas", "P-3F Orion", "P-3F Orion Maritime Patrol - Bandar Abbas", "Red", "p3f-orion-mpa-iran", "irn-airbase-bandar-abbas", 27.23, 56.37, 0, 130, 0),
			scenarioAircraft("irn-p3f-bushehr", "P-3F Bushehr", "P-3F Orion Gulf Maritime Patrol - Bushehr", "Red", "p3f-orion-mpa-iran", "irn-airbase-bushehr", 28.96, 50.85, 0, 140, 0),
			scenarioAircraft("irn-707-tehran", "707 Tanker", "Boeing 707 Tanker - Tehran", "Red", "boeing707-tanker-iran", "irn-airbase-tehran", 35.66, 51.28, 0, 250, 0),
			scenarioAircraft("irn-707-esfahan", "707 Esfahan", "Boeing 707 Tanker - Esfahan Reserve", "Red", "boeing707-tanker-iran", "irn-airbase-esfahan", 32.73, 51.84, 0, 160, 0),
			scenarioUnit("irn-soleimani-hormuz", "Shahid Soleimani", "Shahid Soleimani Corvette - Hormuz Patrol", "Red", "shahid-soleimani-corvette", 26.70, 56.05, 0, 110, 12),
			scenarioUnit("irn-soleimani-abbas", "Shahid Soleimani 2", "Shahid Soleimani Corvette - Bandar Abbas Screen", "Red", "shahid-soleimani-corvette", 27.05, 56.18, 0, 120, 12),
			scenarioUnit("irn-swarm-qeshm", "IRGCN Swarm", "IRGCN Swarm Attack Group - Qeshm Axis", "Red", "irgcn-swarm-group", 26.88, 55.95, 0, 95, 14),
			scenarioUnit("irn-swarm-larak", "IRGCN Swarm 2", "IRGCN Swarm Attack Group - Larak Axis", "Red", "irgcn-swarm-group", 26.55, 56.35, 0, 105, 14),
			scenarioUnit("irn-jamaran-bushehr", "Jamaran", "Jamaran Frigate - Bushehr Patrol", "Red", "jamaran-frigate", 28.75, 50.55, 0, 140, 10),
			scenarioUnit("irn-jamaran-oman", "Jamaran 2", "Jamaran Frigate - Gulf of Oman Patrol", "Red", "jamaran-frigate", 25.90, 58.20, 0, 110, 10),
			scenarioUnit("irn-tareq-gulf", "Tareq Kilo", "Tareq Kilo Submarine - Gulf Ambush Patrol", "Red", "tareq-kilo-submarine", 26.60, 55.80, -25, 95, 5),
			scenarioUnit("irn-ghadir-sub", "Ghadir Sub", "Ghadir Midget Submarine - Strait Ambush Line", "Red", "ghadir-midget-submarine", 26.45, 56.30, -20, 90, 4),
			scenarioUnit("irn-ghadir-sub-2", "Ghadir Sub 2", "Ghadir Midget Submarine - Qeshm Ambush Line", "Red", "ghadir-midget-submarine", 26.78, 55.88, -20, 95, 4),
			scenarioUnit("irn-fateh-sub-gulf", "Fateh Sub", "Fateh Coastal Submarine - Gulf of Oman Patrol", "Red", "fateh-submarine", 25.40, 58.65, -30, 100, 4),
		},
	}
}

func scenarioAircraft(id, displayName, fullName, coalitionHint, definitionID, hostBaseID string, lat, lon, altMsl, heading, speed float64) *enginev1.Unit {
	u := scenarioUnit(id, displayName, fullName, coalitionHint, definitionID, lat, lon, altMsl, heading, speed)
	u.HostBaseId = hostBaseID
	return u
}

func scenarioStrikeUnit(id, displayName, fullName, coalitionHint, definitionID string, lat, lon, altMsl, heading, speed, nextStrikeReadySeconds float64) *enginev1.Unit {
	u := scenarioUnit(id, displayName, fullName, coalitionHint, definitionID, lat, lon, altMsl, heading, speed)
	u.NextStrikeReadySeconds = nextStrikeReadySeconds
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

func scenarioUnit(id, displayName, fullName, coalitionHint, definitionID string, lat, lon, altMsl, heading, speed float64) *enginev1.Unit {
	teamID := inferScenarioTeamID(id, coalitionHint)
	coalitionID := inferScenarioCoalition(coalitionHint, teamID)
	return &enginev1.Unit{
		Id:           id,
		DisplayName:  displayName,
		FullName:     fullName,
		TeamId:       teamID,
		CoalitionId:  coalitionID,
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

func inferScenarioCoalition(coalitionHint, teamID string) string {
	switch strings.ToUpper(coalitionHint) {
	case "BLUE":
		return "COALITION_WEST"
	case "RED":
		return "COALITION_IRAN"
	case "NEUTRAL":
		return "NON_ALIGNED"
	}
	if strings.TrimSpace(coalitionHint) != "" {
		return strings.ToUpper(coalitionHint)
	}
	return strings.ToUpper(teamID)
}
