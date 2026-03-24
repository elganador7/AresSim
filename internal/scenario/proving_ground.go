package scenario

import (
	"time"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

type ProvingGroundSpec struct {
	ScenarioID                string
	Category                  string
	Purpose                   string
	ExpectedSummary           string
	RecommendedTrials         int
	MaxSimSeconds             float64
	FocusTeam                 string
	OpposingTeam              string
	TrackedTargetUnitID       string
	EndOnTrackedTargetDisable bool
	SetupActions              []ProvingGroundSetupAction
	MinFocusWinRate           float64
	MaxFocusWinRate           float64
	MinTargetMissionKillRate  float64
	MaxTargetMissionKillRate  float64
}

type ProvingGroundSetupAction struct {
	Kind                  string
	TeamID                string
	UnitID                string
	TargetUnitID          string
	LoadoutConfiguration  string
	OrderType             enginev1.AttackOrderType
	DesiredEffect         enginev1.DesiredEffect
	ExpectedShooterUnitID string
	Lat                   float64
	Lon                   float64
}

func ProvingGroundSpecs() map[string]ProvingGroundSpec {
	return map[string]ProvingGroundSpec{
		"pg-modern-vs-legacy-air": {
			ScenarioID:                "pg-modern-vs-legacy-air",
			Category:                  "air",
			Purpose:                   "Baseline modern BVR fighter performance against a legacy Iranian fighter.",
			ExpectedSummary:           "USA should win most runs; modern BVR fighter should decisively outperform the legacy opponent.",
			RecommendedTrials:         25,
			MaxSimSeconds:             900,
			FocusTeam:                 "USA",
			OpposingTeam:              "IRN",
			TrackedTargetUnitID:       "pg-irn-f4e",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "seed_detection", TeamID: "USA", TargetUnitID: "pg-irn-f4e"},
				{Kind: "preview_target", TeamID: "USA", TargetUnitID: "pg-irn-f4e", ExpectedShooterUnitID: "pg-usa-f35a"},
				{Kind: "assign_attack", UnitID: "pg-usa-f35a", TargetUnitID: "pg-irn-f4e", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
			},
			MinFocusWinRate:          0.80,
			MaxFocusWinRate:          1.00,
			MinTargetMissionKillRate: 0.80,
		},
		"pg-ballistic-vs-airbase": {
			ScenarioID:                "pg-ballistic-vs-airbase",
			Category:                  "strike",
			Purpose:                   "Calibrate Iranian ballistic-missile effectiveness against a fixed Israeli airbase target.",
			ExpectedSummary:           "Ballistic missiles should frequently mission-kill the runway or airbase, but not produce guaranteed hard destruction.",
			RecommendedTrials:         25,
			MaxSimSeconds:             1200,
			FocusTeam:                 "IRN",
			OpposingTeam:              "ISR",
			TrackedTargetUnitID:       "pg-nevatim-airbase",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "IRN"},
				{Kind: "preview_target", TeamID: "IRN", TargetUnitID: "pg-nevatim-airbase", ExpectedShooterUnitID: "pg-kheibar-brigade"},
				{Kind: "assign_attack", UnitID: "pg-kheibar-brigade", TargetUnitID: "pg-nevatim-airbase", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL},
			},
			MinTargetMissionKillRate: 0.55,
			MaxTargetMissionKillRate: 1.00,
		},
		"pg-destroyer-vs-missile-boat": {
			ScenarioID:                "pg-destroyer-vs-missile-boat",
			Category:                  "naval",
			Purpose:                   "Check that a modern surface combatant generally outperforms a lighter fast-attack missile boat in an open-water missile duel.",
			ExpectedSummary:           "The better-armed U.S. surface combatant should win a clear majority of runs, but the lighter attacker should still impose some risk.",
			RecommendedTrials:         25,
			MaxSimSeconds:             900,
			FocusTeam:                 "USA",
			OpposingTeam:              "IRN",
			TrackedTargetUnitID:       "pg-irn-missile-boat",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "seed_detection", TeamID: "USA", TargetUnitID: "pg-irn-missile-boat"},
				{Kind: "seed_detection", TeamID: "IRN", TargetUnitID: "pg-usa-mmsc"},
				{Kind: "assign_attack", UnitID: "pg-usa-mmsc", TargetUnitID: "pg-irn-missile-boat", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-irn-missile-boat", TargetUnitID: "pg-usa-mmsc", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
			},
			MinFocusWinRate:          0.70,
			MaxFocusWinRate:          1.00,
			MinTargetMissionKillRate: 0.70,
		},
		"pg-hosted-strike-loadout": {
			ScenarioID:                "pg-hosted-strike-loadout",
			Category:                  "strike",
			Purpose:                   "Exercise hosted-aircraft loadout switching and strike assignment against a fixed Iranian airbase from a Gulf base.",
			ExpectedSummary:           "The U.S. strike package should be able to switch to a deep-strike loadout, launch from base, and mission-kill the target in a meaningful share of runs.",
			RecommendedTrials:         15,
			MaxSimSeconds:             1800,
			FocusTeam:                 "USA",
			OpposingTeam:              "IRN",
			TrackedTargetUnitID:       "pg-irn-airbase",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "set_loadout", UnitID: "pg-usa-f15e", LoadoutConfiguration: "deep_strike"},
				{Kind: "preview_target", TeamID: "USA", TargetUnitID: "pg-irn-airbase", ExpectedShooterUnitID: "pg-usa-f15e"},
				{Kind: "assign_attack", UnitID: "pg-usa-f15e", TargetUnitID: "pg-irn-airbase", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL},
			},
			MinTargetMissionKillRate: 0.35,
			MaxTargetMissionKillRate: 1.00,
		},
		"pg-defensive-intercept-third-party": {
			ScenarioID:                "pg-defensive-intercept-third-party",
			Category:                  "airspace",
			Purpose:                   "Validate defensive air intercepts over third-country airspace where transit and defensive positioning are allowed but strike permission is not.",
			ExpectedSummary:           "The U.S. defender should usually win the intercept, and the engagement should remain legal without strike permission.",
			RecommendedTrials:         20,
			MaxSimSeconds:             900,
			FocusTeam:                 "USA",
			OpposingTeam:              "IRN",
			TrackedTargetUnitID:       "pg-irn-f4e-third-party",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "seed_detection", TeamID: "USA", TargetUnitID: "pg-irn-f4e-third-party"},
				{Kind: "preview_target", TeamID: "USA", TargetUnitID: "pg-irn-f4e-third-party", ExpectedShooterUnitID: "pg-usa-f35a-third-party"},
				{Kind: "assign_attack", UnitID: "pg-usa-f35a-third-party", TargetUnitID: "pg-irn-f4e-third-party", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
			},
			MinFocusWinRate:          0.75,
			MaxFocusWinRate:          1.00,
			MinTargetMissionKillRate: 0.75,
		},
		"pg-sam-vs-striker-overland": {
			ScenarioID:                "pg-sam-vs-striker-overland",
			Category:                  "iads",
			Purpose:                   "Calibrate overland strike survivability against a modern Iranian long-range SAM battery protecting a strategic airbase.",
			ExpectedSummary:           "The Iranian air-defense layer should stop or seriously degrade a meaningful share of inbound strike attempts over defended interior airspace.",
			RecommendedTrials:         20,
			MaxSimSeconds:             1200,
			FocusTeam:                 "IRN",
			OpposingTeam:              "USA",
			TrackedTargetUnitID:       "pg-irn-overland-airbase",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "set_loadout", UnitID: "pg-usa-f35a-overland", LoadoutConfiguration: "deep_strike"},
				{Kind: "preview_target", TeamID: "USA", TargetUnitID: "pg-irn-overland-airbase", ExpectedShooterUnitID: "pg-usa-f35a-overland"},
				{Kind: "assign_attack", UnitID: "pg-usa-f35a-overland", TargetUnitID: "pg-irn-overland-airbase", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL},
			},
			MinFocusWinRate:          0.55,
			MaxFocusWinRate:          1.00,
			MaxTargetMissionKillRate: 0.45,
		},
		"pg-virginia-vs-fateh-sub": {
			ScenarioID:                "pg-virginia-vs-fateh-sub",
			Category:                  "subsurface",
			Purpose:                   "Calibrate U.S. attack-submarine advantage against an Iranian coastal submarine in a Gulf littoral engagement.",
			ExpectedSummary:           "The Virginia-class boat should win most runs, but the Iranian submarine should still be dangerous inside littoral waters.",
			RecommendedTrials:         20,
			MaxSimSeconds:             1800,
			FocusTeam:                 "USA",
			OpposingTeam:              "IRN",
			TrackedTargetUnitID:       "pg-irn-fateh-sub",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "seed_detection", TeamID: "USA", TargetUnitID: "pg-irn-fateh-sub"},
				{Kind: "seed_detection", TeamID: "IRN", TargetUnitID: "pg-usa-virginia"},
				{Kind: "preview_target", TeamID: "USA", TargetUnitID: "pg-irn-fateh-sub", ExpectedShooterUnitID: "pg-usa-virginia"},
				{Kind: "assign_attack", UnitID: "pg-usa-virginia", TargetUnitID: "pg-irn-fateh-sub", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-irn-fateh-sub", TargetUnitID: "pg-usa-virginia", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
			},
			MinFocusWinRate: 0.60,
			MaxFocusWinRate: 1.00,
		},
		"pg-ui-demo-gulf-skirmish": {
			ScenarioID:        "pg-ui-demo-gulf-skirmish",
			Category:          "ui demo",
			Purpose:           "Small manual-inspection scenario for the UI: one visible fixed site strike, one airborne intercept, and one hosted-aircraft launch from base.",
			ExpectedSummary:   "Open as USA. Click the Iranian fighter to assign the F-35 intercept, then click Bushehr AB to assign the grounded F-15E strike and watch launch, climb, movement, and impact.",
			RecommendedTrials: 5,
			MaxSimSeconds:     1800,
			FocusTeam:         "USA",
			OpposingTeam:      "IRN",
		},
	}
}

func ProvingGroundBuiltins() []*enginev1.Scenario {
	return []*enginev1.Scenario{
		provingGroundModernVsLegacyAir(),
		provingGroundBallisticVsAirbase(),
		provingGroundDestroyerVsMissileBoat(),
		provingGroundHostedStrikeLoadout(),
		provingGroundDefensiveInterceptThirdParty(),
		provingGroundSamVsStrikerOverland(),
		provingGroundVirginiaVsFatehSub(),
		provingGroundUIDemoGulfSkirmish(),
	}
}

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
			provingGroundUnit("pg-kheibar-brigade", "Kheibar Brigade", "Kheibar Shekan Test Brigade", "IRN", "COALITION_IRAN", "kheibar-shekan-brigade", 31.90, 45.90, 0, 90, 0),
			provingGroundUnit("pg-nevatim-airbase", "Nevatim AB", "Israeli Strategic Air Base - Test Template", "ISR", "COALITION_WEST", "israel-strategic-airbase", 31.21, 35.01, 0, 0, 0),
		},
	}
}

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
				FromCountry:            "USA",
				ToCountry:              "IRN",
				AirspaceTransitAllowed: true,
				AirspaceStrikeAllowed:  true,
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

func provingGroundUIDemoGulfSkirmish() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:   "pg-ui-demo-gulf-skirmish",
		Name: "Proving Ground: UI Demo Gulf Skirmish",
		Description: "Manual demo scenario. Play as USA. First click the Iranian F-4 and assign the airborne F-35A to intercept. Then click Bushehr AB and assign the grounded F-15E strike aircraft. This should show visible fixed sites, enemy-card shooter selection, hosted-aircraft launch, aircraft altitude, movement, and impact resolution without larger-scenario clutter.",
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

func clearWeatherMap() *enginev1.MapSettings {
	return &enginev1.MapSettings{
		InitialWeather: &enginev1.WeatherConditions{
			State:        enginev1.WeatherState_WEATHER_CLEAR,
			VisibilityKm: 60,
			WindSpeedMps: 2,
			TemperatureC: 20,
		},
	}
}

func provingGroundUnit(id, displayName, fullName, teamID, coalitionID, definitionID string, lat, lon, altMsl, heading, speed float64) *enginev1.Unit {
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
			PersonnelStrength:   1,
			EquipmentStrength:   1,
			CombatEffectiveness: 1,
			FuelLevelLiters:     1_000_000,
			Morale:              1,
			Fatigue:             0,
			IsActive:            true,
		},
		EngagementBehavior: enginev1.EngagementBehavior_ENGAGEMENT_BEHAVIOR_AUTO_ENGAGE,
	}
}

func provingAircraft(id, displayName, fullName, teamID, coalitionID, definitionID string, lat, lon, altMsl, heading, speed float64) *enginev1.Unit {
	return provingGroundUnit(id, displayName, fullName, teamID, coalitionID, definitionID, lat, lon, altMsl, heading, speed)
}
