package scenario

import (
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"google.golang.org/protobuf/proto"
)

func selectUnitsByID(units []*enginev1.Unit, ids ...string) []*enginev1.Unit {
	needed := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		needed[id] = struct{}{}
	}
	selected := make([]*enginev1.Unit, 0, len(ids))
	for _, unit := range units {
		if unit == nil {
			continue
		}
		if _, ok := needed[unit.GetId()]; !ok {
			continue
		}
		selected = append(selected, proto.Clone(unit).(*enginev1.Unit))
	}
	return selected
}

func provingGroundPackageIronDomeDrill() *enginev1.Scenario {
	ironDome, _ := InstantiatePackage(PackageTemplates()["pkg-isr-iron-dome-battery"], PackageInstanceOptions{
		Prefix:    "pg-pkg-id",
		AnchorLat: 31.89,
		AnchorLon: 34.79,
	})
	target := provingGroundUnit("pg-pkg-id-ashdod-port", "Ashdod Port", "Israeli Port - Iron Dome Drill", "ISR", "COALITION_WEST", "israel-mediterranean-port", 31.82, 34.65, 0, 0, 0)
	missiles, _ := InstantiatePackage(PackageTemplates()["pkg-irn-western-missile-regiment"], PackageInstanceOptions{
		Prefix:    "pg-pkg-id-irn",
		AnchorLat: 33.90,
		AnchorLon: 44.90,
	})
	drillLaunchers := map[string]struct{}{
		"pg-pkg-id-irn-kheibar-1": {},
		"pg-pkg-id-irn-kheibar-2": {},
	}
	for _, unit := range missiles {
		if _, ok := drillLaunchers[unit.GetId()]; !ok {
			continue
		}
		unit.AttackOrder = attackOrder(target.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL)
	}
	units := append([]*enginev1.Unit{}, ironDome...)
	units = append(units, target)
	for _, unit := range missiles {
		if _, ok := drillLaunchers[unit.GetId()]; ok {
			units = append(units, unit)
		}
	}
	return &enginev1.Scenario{
		Id:             "pg-package-iron-dome-drill",
		Name:           "Proving Ground: Package Iron Dome Drill",
		Description:    "Single Iron Dome package defending a coastal Israeli target against a small Iranian launcher package.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings:       &enginev1.SimulationSettings{TickRateHz: 10, TimeScale: 1.0},
		Map:            clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "IRN", ToCountry: "ISR", AirspaceTransitAllowed: true},
		},
		Units: units,
	}
}

func provingGroundPackageAlUdeidSortie() *enginev1.Scenario {
	basePkg, _ := InstantiatePackage(PackageTemplates()["pkg-usa-al-udeid-airbase"], PackageInstanceOptions{
		Prefix:    "pg-pkg-au",
		AnchorLat: 25.12,
		AnchorLon: 51.31,
	})
	target := provingGroundUnit("pg-pkg-au-bushehr", "Bushehr AB", "Iranian Strategic Air Base - Package Sortie Drill", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 28.95, 50.83, 0, 0, 0)
	secondary := provingGroundUnit("pg-pkg-au-shiraz", "Shiraz AB", "Iranian Strategic Air Base - Package Sortie Drill Secondary Target", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 29.54, 52.59, 0, 0, 0)
	for _, unit := range basePkg {
		switch unit.GetId() {
		case "pg-pkg-au-f15e-wing":
			unit.AttackOrder = attackOrder(target.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL)
		case "pg-pkg-au-f35a-lead":
			unit.AttackOrder = attackOrder(secondary.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL)
		}
	}
	return &enginev1.Scenario{
		Id:             "pg-package-al-udeid-sortie",
		Name:           "Proving Ground: Package Al Udeid Sortie",
		Description:    "Reusable Al Udeid airbase package launching a small hosted strike element against two Iranian airbases to validate launch, recovery, and replenishment flow.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings:       &enginev1.SimulationSettings{TickRateHz: 10, TimeScale: 1.0},
		Map:            clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "QAT", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true},
			{FromCountry: "USA", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true},
		},
		Units: append(append(basePkg, target), secondary),
	}
}

func provingGroundPackageLayeredIsraelDefense() *enginev1.Scenario {
	defense, _ := InstantiatePackage(PackageTemplates()["pkg-isr-layered-defense-dan"], PackageInstanceOptions{
		Prefix:    "pg-pkg-lad-isr",
		AnchorLat: 31.95,
		AnchorLon: 34.78,
	})
	missiles, _ := InstantiatePackage(PackageTemplates()["pkg-irn-western-missile-regiment"], PackageInstanceOptions{
		Prefix:    "pg-pkg-lad-irn",
		AnchorLat: 34.20,
		AnchorLon: 45.55,
	})
	targets := []*enginev1.Unit{
		provingGroundUnit("pg-pkg-lad-nevatim", "Nevatim AB", "Israeli Strategic Air Base - Package Layered Defense", "ISR", "COALITION_WEST", "israel-strategic-airbase", 31.21, 35.01, 0, 0, 0),
		provingGroundUnit("pg-pkg-lad-palmachim", "Palmachim AB", "Israeli Strategic Air Base - Package Layered Defense", "ISR", "COALITION_WEST", "israel-strategic-airbase", 31.89, 34.69, 0, 0, 0),
	}
	counterPkg, _ := InstantiatePackage(PackageTemplates()["pkg-isr-counterstrike-cell"], PackageInstanceOptions{
		Prefix:    "pg-pkg-lad-isrstrike",
		AnchorLat: 31.89,
		AnchorLon: 34.69,
	})
	// Push the strike package into an already-launched posture for the proving ground.
	for _, unit := range counterPkg {
		if unit.GetId() == "pg-pkg-lad-isrstrike-airbase" {
			continue
		}
		if unit.GetDefinitionId() == "g550-eitam" || unit.GetDefinitionId() == "boeing707-reem" {
			if unit.GetPosition() != nil {
				unit.Position.AltMsl = 10200
				unit.Position.Speed = 230
				unit.Position.Heading = 90
				unit.Position.Lat = 32.70
				unit.Position.Lon = 40.90
			}
			unit.HostBaseId = ""
			continue
		}
		// Keep the F-15 standoff pair and F-35 penetrating pair airborne for the opening wave.
		if unit.GetDefinitionId() != "f15i-raam" && unit.GetDefinitionId() != "f35i-adir" {
			continue
		}
		if unit.GetPosition() != nil {
			unit.Position.AltMsl = 9800
			unit.Position.Speed = 240
			unit.Position.Heading = 90
			switch unit.GetId() {
			case "pg-pkg-lad-isrstrike-f15i-lead":
				unit.Position.Lat = 33.10
				unit.Position.Lon = 41.85
			case "pg-pkg-lad-isrstrike-f15i-wing":
				unit.Position.Lat = 33.22
				unit.Position.Lon = 42.00
			case "pg-pkg-lad-isrstrike-f35i-lead":
				unit.Position.Lat = 33.28
				unit.Position.Lon = 42.36
				unit.Position.AltMsl = 9200
				unit.Position.Speed = 235
			case "pg-pkg-lad-isrstrike-f35i-wing":
				unit.Position.Lat = 33.36
				unit.Position.Lon = 42.46
				unit.Position.AltMsl = 9200
				unit.Position.Speed = 235
			}
		}
		unit.HostBaseId = ""
	}
	// Only keep the aircraft needed for the first-wave counterstrike slice.
	aircraft := make([]*enginev1.Unit, 0, 4)
	for _, unit := range counterPkg {
		if unit.GetPosition().GetAltMsl() > 0 && (unit.GetDefinitionId() == "f15i-raam" || unit.GetDefinitionId() == "f35i-adir") {
			aircraft = append(aircraft, unit)
		}
	}
	launcherTargets := map[string]string{
		"pg-pkg-lad-irn-kheibar-1": "pg-pkg-lad-nevatim",
		"pg-pkg-lad-irn-kheibar-2": "pg-pkg-lad-palmachim",
		"pg-pkg-lad-irn-paveh-1":   "pg-pkg-lad-nevatim",
		"pg-pkg-lad-irn-paveh-2":   "pg-pkg-lad-nevatim",
	}
	for _, unit := range missiles {
		targetID, ok := launcherTargets[unit.GetId()]
		if !ok {
			continue
		}
		unit.AttackOrder = attackOrder(targetID, enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL)
	}
	counterTargets := map[string]string{
		"pg-pkg-lad-isrstrike-f15i-lead": "pg-pkg-lad-irn-kheibar-1",
		"pg-pkg-lad-isrstrike-f15i-wing": "pg-pkg-lad-irn-kheibar-2",
		"pg-pkg-lad-isrstrike-f35i-lead": "pg-pkg-lad-irn-paveh-1",
		"pg-pkg-lad-isrstrike-f35i-wing": "pg-pkg-lad-irn-paveh-2",
	}
	for _, unit := range aircraft {
		targetID, ok := counterTargets[unit.GetId()]
		if !ok {
			continue
		}
		unit.AttackOrder = attackOrder(targetID, enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
	}
	units := append([]*enginev1.Unit{}, defense...)
	units = append(units, targets...)
	units = append(units, missiles...)
	units = append(units, aircraft...)
	return &enginev1.Scenario{
		Id:             "pg-package-layered-israel-defense",
		Name:           "Proving Ground: Package Layered Israel Defense",
		Description:    "Composed scenario using reusable layered Israeli defense, Iranian launcher, and strike packages to validate package composition in a live missile-defense exchange.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings:       &enginev1.SimulationSettings{TickRateHz: 10, TimeScale: 1.0},
		Map:            clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "IRN", ToCountry: "ISR", AirspaceTransitAllowed: true},
			{FromCountry: "ISR", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true},
		},
		Units: units,
	}
}

func provingGroundPackageBahrainMaritimePresence() *enginev1.Scenario {
	bahrainPkg, _ := InstantiatePackage(PackageTemplates()["pkg-bhr-naval-support-bahrain"], PackageInstanceOptions{
		Prefix:    "pg-pkg-bh",
		AnchorLat: 26.23,
		AnchorLon: 50.61,
	})
	raider := provingGroundUnit("pg-pkg-bh-irn-raider", "Zulfiqar", "Iranian Missile Boat - Bahrain Package Drill", "IRN", "COALITION_IRAN", "zulfiqar-missile-boat", 26.66, 51.48, 0, 240, 16)
	for _, unit := range bahrainPkg {
		switch unit.GetId() {
		case "pg-pkg-bh-lcs", "pg-pkg-bh-frigate":
			unit.AttackOrder = attackOrder(raider.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	raider.AttackOrder = attackOrder("pg-pkg-bh-lcs", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
	units := append([]*enginev1.Unit{}, bahrainPkg...)
	units = append(units, raider)
	return &enginev1.Scenario{
		Id:             "pg-package-bahrain-maritime-presence",
		Name:           "Proving Ground: Package Bahrain Maritime Presence",
		Description:    "Reusable Bahrain naval support package facing a small Iranian surface raider to validate Gulf littoral presence, sovereignty, and local maritime defense behavior.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings:       &enginev1.SimulationSettings{TickRateHz: 10, TimeScale: 1.0},
		Map:            clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "BHR", MaritimeTransitAllowed: true},
			{FromCountry: "IRN", ToCountry: "BHR", MaritimeTransitAllowed: false},
			{FromCountry: "BHR", ToCountry: "IRN", MaritimeTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "IRN", MaritimeTransitAllowed: true},
		},
		Units: units,
	}
}

func provingGroundPackageAlDhafraForwardStrike() *enginev1.Scenario {
	dhafraPkg, _ := InstantiatePackage(PackageTemplates()["pkg-are-al-dhafra-airbase"], PackageInstanceOptions{
		Prefix:    "pg-pkg-ad",
		AnchorLat: 24.25,
		AnchorLon: 54.55,
	})
	target := provingGroundUnit("pg-pkg-ad-bandar-abbas", "Bandar Abbas AB", "Iranian Strategic Air Base - Al Dhafra Forward Strike Drill", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 27.21, 56.37, 0, 0, 0)
	for _, unit := range dhafraPkg {
		switch unit.GetId() {
		case "pg-pkg-ad-f35a-lead":
			unit.AttackOrder = attackOrder(target.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL)
		case "pg-pkg-ad-f35a-wing":
			unit.AttackOrder = attackOrder(target.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL)
		}
	}
	units := append([]*enginev1.Unit{}, dhafraPkg...)
	units = append(units, target)
	return &enginev1.Scenario{
		Id:             "pg-package-al-dhafra-forward-strike",
		Name:           "Proving Ground: Package Al Dhafra Forward Strike",
		Description:    "Reusable Al Dhafra package validating sovereign/operator split, hosted F-35 launch, and forward Gulf strike behavior against Bandar Abbas.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings:       &enginev1.SimulationSettings{TickRateHz: 10, TimeScale: 1.0},
		Map:            clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "ARE", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true},
			{FromCountry: "USA", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true},
		},
		Units: units,
	}
}

func provingGroundPackageGulfRegionalSupportPosture() *enginev1.Scenario {
	alUdeidPkg, _ := InstantiatePackage(PackageTemplates()["pkg-usa-al-udeid-airbase"], PackageInstanceOptions{
		Prefix:    "pg-pkg-grs-au",
		AnchorLat: 25.12,
		AnchorLon: 51.31,
	})
	dhafraPkg, _ := InstantiatePackage(PackageTemplates()["pkg-are-al-dhafra-airbase"], PackageInstanceOptions{
		Prefix:    "pg-pkg-grs-ad",
		AnchorLat: 24.25,
		AnchorLon: 54.55,
	})
	bahrainPkg, _ := InstantiatePackage(PackageTemplates()["pkg-bhr-naval-support-bahrain"], PackageInstanceOptions{
		Prefix:    "pg-pkg-grs-bh",
		AnchorLat: 26.23,
		AnchorLon: 50.61,
	})
	bushehr := provingGroundUnit("pg-pkg-grs-bushehr", "Bushehr AB", "Iranian Strategic Air Base - Gulf Regional Support Posture", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 28.95, 50.83, 0, 0, 0)
	bandarAbbas := provingGroundUnit("pg-pkg-grs-bandar-abbas", "Bandar Abbas AB", "Iranian Strategic Air Base - Gulf Regional Support Posture Eastern Target", "IRN", "COALITION_IRAN", "iran-strategic-airbase", 27.21, 56.37, 0, 0, 0)
	raider := provingGroundUnit("pg-pkg-grs-raider", "Zulfiqar", "Iranian Missile Boat - Gulf Regional Support Posture", "IRN", "COALITION_IRAN", "zulfiqar-missile-boat", 26.70, 51.42, 0, 235, 16)

	for _, unit := range alUdeidPkg {
		switch unit.GetId() {
		case "pg-pkg-grs-au-f15e-lead":
			unit.AttackOrder = attackOrder(bushehr.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL)
		case "pg-pkg-grs-au-f35a-lead":
			unit.AttackOrder = attackOrder(bushehr.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	for _, unit := range dhafraPkg {
		switch unit.GetId() {
		case "pg-pkg-grs-ad-f35a-lead", "pg-pkg-grs-ad-f35a-wing":
			unit.AttackOrder = attackOrder(bandarAbbas.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL)
		}
	}
	for _, unit := range bahrainPkg {
		switch unit.GetId() {
		case "pg-pkg-grs-bh-lcs", "pg-pkg-grs-bh-frigate":
			unit.AttackOrder = attackOrder(raider.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	raider.AttackOrder = attackOrder("pg-pkg-grs-bh-lcs", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)

	units := append([]*enginev1.Unit{}, alUdeidPkg...)
	units = append(units, dhafraPkg...)
	units = append(units, bahrainPkg...)
	units = append(units, bushehr, bandarAbbas, raider)
	return &enginev1.Scenario{
		Id:             "pg-package-gulf-regional-support-posture",
		Name:           "Proving Ground: Package Gulf Regional Support Posture",
		Description:    "Composed Gulf scenario combining reusable Al Udeid, Al Dhafra, and Bahrain packages against Iranian air and littoral threats to validate package interoperability.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings:       &enginev1.SimulationSettings{TickRateHz: 10, TimeScale: 1.0},
		Map:            clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "USA", ToCountry: "QAT", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true},
			{FromCountry: "USA", ToCountry: "ARE", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true},
			{FromCountry: "USA", ToCountry: "BHR", MaritimeTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "IRN", AirspaceTransitAllowed: true, AirspaceStrikeAllowed: true, DefensivePositioningAllowed: true, MaritimeTransitAllowed: true},
			{FromCountry: "BHR", ToCountry: "IRN", MaritimeTransitAllowed: true},
			{FromCountry: "IRN", ToCountry: "ARE", AirspaceTransitAllowed: true},
			{FromCountry: "IRN", ToCountry: "QAT", AirspaceTransitAllowed: true},
			{FromCountry: "IRN", ToCountry: "BHR", MaritimeTransitAllowed: false},
		},
		Units: units,
	}
}

func provingGroundPackageHormuzCoastalDenial() *enginev1.Scenario {
	denialPkg, _ := InstantiatePackage(PackageTemplates()["pkg-irn-hormuz-coastal-denial"], PackageInstanceOptions{
		Prefix:    "pg-pkg-hcd",
		AnchorLat: 27.03,
		AnchorLon: 56.24,
	})
	baynunah := provingGroundUnit("pg-pkg-hcd-baynunah", "Baynunah", "UAE Baynunah Corvette - Hormuz Coastal Denial Drill", "ARE", "COALITION_WEST", "baynunah-corvette-uae", 25.84, 56.58, 0, 40, 10)
	lcs := provingGroundUnit("pg-pkg-hcd-lcs", "USS Freedom", "U.S. LCS - Hormuz Coastal Denial Drill", "USA", "COALITION_WEST", "freedom-lcs", 25.78, 56.48, 0, 40, 10)
	for _, unit := range denialPkg {
		switch unit.GetId() {
		case "pg-pkg-hcd-coastal-battery", "pg-pkg-hcd-swarm", "pg-pkg-hcd-soleimani":
			unit.AttackOrder = attackOrder(lcs.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		case "pg-pkg-hcd-ghadir-sub":
			unit.AttackOrder = attackOrder(baynunah.GetId(), enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	baynunah.AttackOrder = attackOrder("pg-pkg-hcd-soleimani", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
	lcs.AttackOrder = attackOrder("pg-pkg-hcd-swarm", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
	units := append([]*enginev1.Unit{}, denialPkg...)
	units = append(units, baynunah, lcs)
	return &enginev1.Scenario{
		Id:             "pg-package-hormuz-coastal-denial",
		Name:           "Proving Ground: Package Hormuz Coastal Denial",
		Description:    "Reusable Iranian Hormuz coastal-denial package engaging a mixed UAE/U.S. transit group to validate layered littoral threat behavior.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings:       &enginev1.SimulationSettings{TickRateHz: 10, TimeScale: 1.0},
		Map:            clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "IRN", ToCountry: "ARE", AirspaceTransitAllowed: true, MaritimeTransitAllowed: false},
			{FromCountry: "IRN", ToCountry: "USA", AirspaceTransitAllowed: true, MaritimeTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "IRN", MaritimeTransitAllowed: true},
			{FromCountry: "ARE", ToCountry: "IRN", MaritimeTransitAllowed: true},
		},
		Units: units,
	}
}

func provingGroundPackageUAECoastalDefense() *enginev1.Scenario {
	uaePkg, _ := InstantiatePackage(PackageTemplates()["pkg-are-abu-dhabi-coastal-defense"], PackageInstanceOptions{
		Prefix:    "pg-pkg-uacd",
		AnchorLat: 24.52,
		AnchorLon: 54.37,
	})
	denialPkg, _ := InstantiatePackage(PackageTemplates()["pkg-irn-hormuz-coastal-denial"], PackageInstanceOptions{
		Prefix:    "pg-pkg-uacd-irn",
		AnchorLat: 26.30,
		AnchorLon: 55.35,
	})
	for _, unit := range uaePkg {
		switch unit.GetId() {
		case "pg-pkg-uacd-baynunah":
			unit.AttackOrder = attackOrder("pg-pkg-uacd-irn-soleimani", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		case "pg-pkg-uacd-baynunah-wing":
			unit.AttackOrder = attackOrder("pg-pkg-uacd-irn-swarm", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	for _, unit := range denialPkg {
		switch unit.GetId() {
		case "pg-pkg-uacd-irn-coastal-battery", "pg-pkg-uacd-irn-swarm", "pg-pkg-uacd-irn-soleimani":
			unit.AttackOrder = attackOrder("pg-pkg-uacd-baynunah", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	units := append([]*enginev1.Unit{}, uaePkg...)
	units = append(units, denialPkg...)
	return &enginev1.Scenario{
		Id:             "pg-package-uae-coastal-defense",
		Name:           "Proving Ground: Package UAE Coastal Defense",
		Description:    "Reusable Emirati coastal-defense package facing the Iranian Hormuz coastal-denial package to validate native western-Gulf littoral defense behavior.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings:       &enginev1.SimulationSettings{TickRateHz: 10, TimeScale: 1.0},
		Map:            clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "ARE", ToCountry: "IRN", MaritimeTransitAllowed: true},
			{FromCountry: "IRN", ToCountry: "ARE", MaritimeTransitAllowed: false, AirspaceTransitAllowed: true},
		},
		Units: units,
	}
}

func provingGroundPackageOMNMusandamStraitGuard() *enginev1.Scenario {
	omanPkg, _ := InstantiatePackage(PackageTemplates()["pkg-omn-musandam-strait-guard"], PackageInstanceOptions{
		Prefix:    "pg-pkg-omsg",
		AnchorLat: 26.20,
		AnchorLon: 56.30,
	})
	denialPkg, _ := InstantiatePackage(PackageTemplates()["pkg-irn-hormuz-coastal-denial"], PackageInstanceOptions{
		Prefix:    "pg-pkg-omsg-irn",
		AnchorLat: 26.74,
		AnchorLon: 56.68,
	})
	for _, unit := range omanPkg {
		switch unit.GetId() {
		case "pg-pkg-omsg-khareef":
			unit.AttackOrder = attackOrder("pg-pkg-omsg-irn-soleimani", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		case "pg-pkg-omsg-musandam":
			unit.AttackOrder = attackOrder("pg-pkg-omsg-irn-swarm", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	for _, unit := range denialPkg {
		switch unit.GetId() {
		case "pg-pkg-omsg-irn-coastal-battery", "pg-pkg-omsg-irn-swarm", "pg-pkg-omsg-irn-soleimani":
			unit.AttackOrder = attackOrder("pg-pkg-omsg-khareef", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	units := append([]*enginev1.Unit{}, omanPkg...)
	units = append(units, denialPkg...)
	return &enginev1.Scenario{
		Id:             "pg-package-oman-musandam-strait-guard",
		Name:           "Proving Ground: Package Oman Musandam Strait Guard",
		Description:    "Reusable Omani Musandam package facing the Iranian Hormuz coastal-denial package to validate southern-Strait escort and local sea-control behavior.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings:       &enginev1.SimulationSettings{TickRateHz: 10, TimeScale: 1.0},
		Map:            clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "OMN", ToCountry: "IRN", MaritimeTransitAllowed: true},
			{FromCountry: "IRN", ToCountry: "OMN", MaritimeTransitAllowed: false, AirspaceTransitAllowed: true},
		},
		Units: units,
	}
}

func provingGroundPackageStraitRegionalControl() *enginev1.Scenario {
	uaePkg, _ := InstantiatePackage(PackageTemplates()["pkg-are-abu-dhabi-coastal-defense"], PackageInstanceOptions{
		Prefix:    "pg-pkg-src-uacd",
		AnchorLat: 24.52,
		AnchorLon: 54.37,
	})
	omanPkg, _ := InstantiatePackage(PackageTemplates()["pkg-omn-musandam-strait-guard"], PackageInstanceOptions{
		Prefix:    "pg-pkg-src-omsg",
		AnchorLat: 26.20,
		AnchorLon: 56.30,
	})
	bahrainPkg, _ := InstantiatePackage(PackageTemplates()["pkg-bhr-naval-support-bahrain"], PackageInstanceOptions{
		Prefix:    "pg-pkg-src-bh",
		AnchorLat: 26.23,
		AnchorLon: 50.61,
	})
	dhafraPkg, _ := InstantiatePackage(PackageTemplates()["pkg-are-al-dhafra-airbase"], PackageInstanceOptions{
		Prefix:    "pg-pkg-src-ad",
		AnchorLat: 24.25,
		AnchorLon: 54.55,
	})
	denialPkg, _ := InstantiatePackage(PackageTemplates()["pkg-irn-hormuz-coastal-denial"], PackageInstanceOptions{
		Prefix:    "pg-pkg-src-irn",
		AnchorLat: 26.70,
		AnchorLon: 56.20,
	})
	uaeCombat := selectUnitsByID(uaePkg, "pg-pkg-src-uacd-baynunah", "pg-pkg-src-uacd-baynunah-wing")
	omanCombat := selectUnitsByID(omanPkg, "pg-pkg-src-omsg-radar", "pg-pkg-src-omsg-khareef", "pg-pkg-src-omsg-musandam")
	bahrainCombat := selectUnitsByID(bahrainPkg, "pg-pkg-src-bh-frigate", "pg-pkg-src-bh-lcs", "pg-pkg-src-bh-patrol")
	usAir := selectUnitsByID(dhafraPkg, "pg-pkg-src-ad-e3", "pg-pkg-src-ad-f22-lead", "pg-pkg-src-ad-f22-wing", "pg-pkg-src-ad-f35a-lead", "pg-pkg-src-ad-f35a-wing")
	iranCombat := selectUnitsByID(denialPkg, "pg-pkg-src-irn-coastal-battery", "pg-pkg-src-irn-sam", "pg-pkg-src-irn-area-sam", "pg-pkg-src-irn-swarm", "pg-pkg-src-irn-soleimani", "pg-pkg-src-irn-ghadir-sub")

	for _, unit := range uaeCombat {
		switch unit.GetId() {
		case "pg-pkg-src-uacd-baynunah":
			unit.AttackOrder = attackOrder("pg-pkg-src-irn-soleimani", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		case "pg-pkg-src-uacd-baynunah-wing":
			unit.AttackOrder = attackOrder("pg-pkg-src-irn-swarm", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	for _, unit := range omanCombat {
		switch unit.GetId() {
		case "pg-pkg-src-omsg-khareef":
			unit.AttackOrder = attackOrder("pg-pkg-src-irn-soleimani", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		case "pg-pkg-src-omsg-musandam":
			unit.AttackOrder = attackOrder("pg-pkg-src-irn-swarm", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	for _, unit := range bahrainCombat {
		switch unit.GetId() {
		case "pg-pkg-src-bh-lcs", "pg-pkg-src-bh-frigate":
			unit.AttackOrder = attackOrder("pg-pkg-src-irn-soleimani", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		case "pg-pkg-src-bh-patrol":
			unit.AttackOrder = attackOrder("pg-pkg-src-irn-swarm", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	for _, unit := range usAir {
		switch unit.GetId() {
		case "pg-pkg-src-ad-e3":
			unit.Position.AltMsl = 10200
			unit.Position.Speed = 220
			unit.Position.Heading = 45
			unit.Position.Lat = 25.40
			unit.Position.Lon = 55.15
			unit.HostBaseId = ""
		case "pg-pkg-src-ad-f22-lead":
			unit.Position.AltMsl = 9800
			unit.Position.Speed = 240
			unit.Position.Heading = 55
			unit.Position.Lat = 25.65
			unit.Position.Lon = 55.55
			unit.HostBaseId = ""
			unit.AttackOrder = attackOrder("pg-pkg-src-irn-soleimani", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		case "pg-pkg-src-ad-f22-wing":
			unit.Position.AltMsl = 9800
			unit.Position.Speed = 240
			unit.Position.Heading = 55
			unit.Position.Lat = 25.55
			unit.Position.Lon = 55.40
			unit.HostBaseId = ""
			unit.AttackOrder = attackOrder("pg-pkg-src-irn-swarm", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		case "pg-pkg-src-ad-f35a-lead":
			unit.Position.AltMsl = 9300
			unit.Position.Speed = 235
			unit.Position.Heading = 60
			unit.Position.Lat = 25.10
			unit.Position.Lon = 55.00
			unit.HostBaseId = ""
			unit.AttackOrder = attackOrder("pg-pkg-src-irn-coastal-battery", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		case "pg-pkg-src-ad-f35a-wing":
			unit.Position.AltMsl = 9300
			unit.Position.Speed = 235
			unit.Position.Heading = 60
			unit.Position.Lat = 25.00
			unit.Position.Lon = 54.90
			unit.HostBaseId = ""
			unit.AttackOrder = attackOrder("pg-pkg-src-irn-area-sam", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}
	for _, unit := range iranCombat {
		switch unit.GetId() {
		case "pg-pkg-src-irn-coastal-battery":
			unit.AttackOrder = attackOrder("pg-pkg-src-bh-lcs", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		case "pg-pkg-src-irn-swarm", "pg-pkg-src-irn-soleimani":
			unit.AttackOrder = attackOrder("pg-pkg-src-uacd-baynunah", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		case "pg-pkg-src-irn-ghadir-sub":
			unit.AttackOrder = attackOrder("pg-pkg-src-bh-lcs", enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY)
		}
	}

	units := append([]*enginev1.Unit{}, uaeCombat...)
	units = append(units, omanCombat...)
	units = append(units, bahrainCombat...)
	units = append(units, usAir...)
	units = append(units, iranCombat...)
	return &enginev1.Scenario{
		Id:             "pg-package-strait-regional-control",
		Name:           "Proving Ground: Package Strait Regional Control",
		Description:    "Composed Strait of Hormuz scenario using coalition surface combatants, visible U.S. air support, and fixed Iranian shore defenses plus mobile littoral attackers to validate regional package interoperability.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings:       &enginev1.SimulationSettings{TickRateHz: 10, TimeScale: 1.0},
		Map:            clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{
			{FromCountry: "ARE", ToCountry: "IRN", MaritimeTransitAllowed: true},
			{FromCountry: "OMN", ToCountry: "IRN", MaritimeTransitAllowed: true},
			{FromCountry: "BHR", ToCountry: "IRN", MaritimeTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "IRN", MaritimeTransitAllowed: true},
			{FromCountry: "IRN", ToCountry: "ARE", MaritimeTransitAllowed: false, AirspaceTransitAllowed: true},
			{FromCountry: "IRN", ToCountry: "OMN", MaritimeTransitAllowed: false, AirspaceTransitAllowed: true},
			{FromCountry: "IRN", ToCountry: "BHR", MaritimeTransitAllowed: false},
			{FromCountry: "USA", ToCountry: "BHR", MaritimeTransitAllowed: true},
		},
		Units: units,
	}
}
