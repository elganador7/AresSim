package scenario

import (
	"github.com/aressim/internal/geo"
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
	MinInterceptionRate       float64
	MaxInterceptionRate       float64
	MinMeanFocusHitsTaken     float64
	MaxMeanFocusHitsTaken     float64
	MinMeanOpposingLosses     float64
	MaxMeanOpposingLosses     float64
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
	specs := packageProvingGroundSpecs()
	for id, spec := range map[string]ProvingGroundSpec{
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
		"pg-aew-supported-intercept": {
			ScenarioID:                "pg-aew-supported-intercept",
			Category:                  "air",
			Purpose:                   "Calibrate whether an AEW-supported U.S. fighter pair reliably outperforms a comparable unsupported Iranian pair in a clean Gulf intercept.",
			ExpectedSummary:           "The U.S. package should usually win the intercept with at least one fighter surviving, showing a meaningful force-multiplier effect from AEW support.",
			RecommendedTrials:         20,
			MaxSimSeconds:             1200,
			FocusTeam:                 "USA",
			OpposingTeam:              "IRN",
			TrackedTargetUnitID:       "pg-aew-irn-lead",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "seed_detection", TeamID: "USA", TargetUnitID: "pg-aew-irn-lead"},
				{Kind: "seed_detection", TeamID: "USA", TargetUnitID: "pg-aew-irn-wing"},
				{Kind: "seed_detection", TeamID: "IRN", TargetUnitID: "pg-aew-usa-lead"},
				{Kind: "seed_detection", TeamID: "IRN", TargetUnitID: "pg-aew-usa-wing"},
				{Kind: "assign_attack", UnitID: "pg-aew-usa-lead", TargetUnitID: "pg-aew-irn-lead", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-aew-usa-wing", TargetUnitID: "pg-aew-irn-wing", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-aew-irn-lead", TargetUnitID: "pg-aew-usa-lead", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-aew-irn-wing", TargetUnitID: "pg-aew-usa-wing", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
			},
			MinFocusWinRate:          0.70,
			MaxFocusWinRate:          1.00,
			MinTargetMissionKillRate: 0.70,
		},
		"pg-ballistic-vs-airbase": {
			ScenarioID:                "pg-ballistic-vs-airbase",
			Category:                  "strike",
			Purpose:                   "Calibrate Iranian ballistic-missile effectiveness against a fixed Israeli airbase target.",
			ExpectedSummary:           "Ballistic missiles should frequently mission-kill the runway or airbase, but not produce guaranteed hard destruction.",
			RecommendedTrials:         25,
			MaxSimSeconds:             1800,
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
		"pg-israel-missile-defense-saturation": {
			ScenarioID:            "pg-israel-missile-defense-saturation",
			Category:              "missile_defense",
			Purpose:               "Stress Israel's layered missile defense against a mass Iranian ballistic and cruise-missile salvo while also testing Israeli counterstrikes against launcher units.",
			ExpectedSummary:       "Israel should intercept most inbound projectiles, absorb some leakage, and destroy a meaningful share of Iranian launchers while Iran still lands at least 10 impacts in Israel.",
			RecommendedTrials:     10,
			MaxSimSeconds:         5400,
			FocusTeam:             "ISR",
			OpposingTeam:          "IRN",
			SetupActions:          israelMissileDefenseSetupActions(),
			MinInterceptionRate:   0.75,
			MinMeanFocusHitsTaken: 10,
			MaxMeanFocusHitsTaken: 120,
			MinMeanOpposingLosses: 3,
			MaxMeanOpposingLosses: 12,
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
		"pg-maritime-transit-rules": {
			ScenarioID:        "pg-maritime-transit-rules",
			Category:          "maritime",
			Purpose:           "Manual transit-permission scenario for surface and subsurface pathing through Gulf territorial waters.",
			ExpectedSummary:   "Use this in the UI: try moving the surface ship into Qatari territorial waters and it should be blocked; try the same with the submarine and it should be allowed.",
			RecommendedTrials: 1,
			MaxSimSeconds:     1800,
			FocusTeam:         "USA",
			OpposingTeam:      "IRN",
		},
		"pg-airspace-routing-rules": {
			ScenarioID:        "pg-airspace-routing-rules",
			Category:          "routing",
			Purpose:           "Manual airspace-transit scenario for validating foreign-airspace movement permissions with an airborne U.S. fighter.",
			ExpectedSummary:   "Use this in the UI: move the fighter toward Bushehr at 28.950, 50.830 and it should be blocked; move it east over the Gulf of Oman and it should be allowed.",
			RecommendedTrials: 1,
			MaxSimSeconds:     1800,
			FocusTeam:         "USA",
			OpposingTeam:      "IRN",
		},
		"pg-land-border-routing-rules": {
			ScenarioID:        "pg-land-border-routing-rules",
			Category:          "routing",
			Purpose:           "Manual land-border scenario for validating closed-border movement with a U.S. ground convoy in Iraq.",
			ExpectedSummary:   "Use this in the UI: move the convoy east into Iran and it should be blocked; move it within Iraq and it should be allowed.",
			RecommendedTrials: 1,
			MaxSimSeconds:     1800,
			FocusTeam:         "USA",
			OpposingTeam:      "IRN",
		},
		"pg-maritime-reroute-around-qatar": {
			ScenarioID:        "pg-maritime-reroute-around-qatar",
			Category:          "routing",
			Purpose:           "Manual reroute scenario for validating that surface ships route around Qatar instead of cutting through land or denied waters.",
			ExpectedSummary:   "Use this in the UI: move the ship from west of Qatar to 24.800, 52.300 and it should succeed with a curved route around Qatar instead of a straight line.",
			RecommendedTrials: 1,
			MaxSimSeconds:     1800,
			FocusTeam:         "USA",
			OpposingTeam:      "IRN",
		},
		"pg-hormuz-transit-passage": {
			ScenarioID:        "pg-hormuz-transit-passage",
			Category:          "routing",
			Purpose:           "Manual transit-passage scenario for validating that ships can legally navigate the Strait of Hormuz while still respecting nearby land and coasts.",
			ExpectedSummary:   "Use this in the UI: move the ship east through Hormuz toward 26.340, 56.820 and it should succeed through the transit lane without cutting across Musandam or nearby shoreline.",
			RecommendedTrials: 1,
			MaxSimSeconds:     1800,
			FocusTeam:         "USA",
			OpposingTeam:      "IRN",
		},
		"pg-air-reroute-around-qatar": {
			ScenarioID:        "pg-air-reroute-around-qatar",
			Category:          "routing",
			Purpose:           "Manual reroute scenario for validating that aircraft avoid denied Qatari airspace when a legal Saudi/UAE path exists.",
			ExpectedSummary:   "Use this in the UI: move the fighter east toward 24.800, 53.300 and it should succeed with a rerouted path that bends around Qatar.",
			RecommendedTrials: 1,
			MaxSimSeconds:     1800,
			FocusTeam:         "USA",
			OpposingTeam:      "IRN",
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
		"pg-composite-gulf-raid": {
			ScenarioID:                "pg-composite-gulf-raid",
			Category:                  "composite",
			Purpose:                   "Composite Gulf raid with hosted launch, escort, opposing CAP, SAM defense, and a fixed-site strike target.",
			ExpectedSummary:           "The U.S. package should usually get the interceptor into the fight and give the strike aircraft a credible path to attack, but this is mainly for feature integration and manual inspection.",
			RecommendedTrials:         10,
			MaxSimSeconds:             2400,
			FocusTeam:                 "USA",
			OpposingTeam:              "IRN",
			TrackedTargetUnitID:       "pg-cg-bushehr-airbase",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "set_loadout", UnitID: "pg-cg-usa-f15e", LoadoutConfiguration: "deep_strike"},
				{Kind: "seed_detection", TeamID: "USA", TargetUnitID: "pg-cg-irn-f4e"},
				{Kind: "preview_target", TeamID: "USA", TargetUnitID: "pg-cg-bushehr-airbase", ExpectedShooterUnitID: "pg-cg-usa-f15e"},
				{Kind: "assign_attack", UnitID: "pg-cg-usa-f35a", TargetUnitID: "pg-cg-irn-f4e", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-cg-usa-f15e", TargetUnitID: "pg-cg-bushehr-airbase", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL},
			},
		},
		"pg-fuel-limited-deep-strike": {
			ScenarioID:                "pg-fuel-limited-deep-strike",
			Category:                  "fuel",
			Purpose:                   "Validate that a low-fuel strike aircraft aborts and returns home before it can press a deep inland strike, exposing simple bingo-fuel behavior.",
			ExpectedSummary:           "The strike should usually fail because the aircraft turns back for home before reaching a viable launch position.",
			RecommendedTrials:         10,
			MaxSimSeconds:             3600,
			FocusTeam:                 "USA",
			OpposingTeam:              "IRN",
			TrackedTargetUnitID:       "pg-fuel-esfahan-airbase",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "set_loadout", UnitID: "pg-fuel-usa-f35a", LoadoutConfiguration: "strike"},
				{Kind: "preview_target", TeamID: "USA", TargetUnitID: "pg-fuel-esfahan-airbase", ExpectedShooterUnitID: "pg-fuel-usa-f35a"},
				{Kind: "assign_attack", UnitID: "pg-fuel-usa-f35a", TargetUnitID: "pg-fuel-esfahan-airbase", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL},
			},
			MaxTargetMissionKillRate: 0.20,
		},
		"pg-sortie-regeneration-cycle": {
			ScenarioID:          "pg-sortie-regeneration-cycle",
			Category:            "replenishment",
			Purpose:             "Exercise launch, strike, return, landing, refuel/rearm, and second-sortie readiness in one compact scenario.",
			ExpectedSummary:     "Use this to inspect whether a strike aircraft launches, returns to Al Udeid, enters replenishment, and becomes ready again while a second target remains available for re-tasking.",
			RecommendedTrials:   5,
			MaxSimSeconds:       5400,
			FocusTeam:           "USA",
			OpposingTeam:        "IRN",
			TrackedTargetUnitID: "pg-src-bushehr-airbase",
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "set_loadout", UnitID: "pg-src-usa-f15e", LoadoutConfiguration: "deep_strike"},
				{Kind: "assign_attack", UnitID: "pg-src-usa-f15e", TargetUnitID: "pg-src-bushehr-airbase", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL},
			},
			MinTargetMissionKillRate: 0.35,
			MaxTargetMissionKillRate: 1.00,
		},
		"pg-gulf-escalation-cycle": {
			ScenarioID:                "pg-gulf-escalation-cycle",
			Category:                  "composite",
			Purpose:                   "Bidirectional Gulf escalation with U.S. escort and hosted strike, Iranian CAP, SAM defense, and retaliatory ballistic attack on the Gulf base.",
			ExpectedSummary:           "Use this to inspect a more realistic opening exchange: counterair, fixed-site strike, ballistic retaliation, fuel return, and replenishment in one scenario.",
			RecommendedTrials:         10,
			MaxSimSeconds:             3600,
			FocusTeam:                 "USA",
			OpposingTeam:              "IRN",
			TrackedTargetUnitID:       "pg-ge-bushehr-airbase",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "set_loadout", UnitID: "pg-ge-usa-f15e", LoadoutConfiguration: "deep_strike"},
				{Kind: "seed_detection", TeamID: "USA", TargetUnitID: "pg-ge-irn-f4e"},
				{Kind: "preview_target", TeamID: "USA", TargetUnitID: "pg-ge-bushehr-airbase", ExpectedShooterUnitID: "pg-ge-usa-f15e"},
				{Kind: "assign_attack", UnitID: "pg-ge-usa-f35a", TargetUnitID: "pg-ge-irn-f4e", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-ge-usa-f15e", TargetUnitID: "pg-ge-bushehr-airbase", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL},
				{Kind: "assign_attack", UnitID: "pg-ge-kheibar-brigade", TargetUnitID: "pg-ge-al-udeid", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_STRIKE_UNTIL_EFFECT, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_MISSION_KILL},
			},
		},
		"pg-hormuz-multi-domain-skirmish": {
			ScenarioID:                "pg-hormuz-multi-domain-skirmish",
			Category:                  "composite",
			Purpose:                   "Multi-domain Strait of Hormuz skirmish with simultaneous air, surface, and subsurface fights in one compact scenario.",
			ExpectedSummary:           "Use this to inspect whether air, naval, and subsurface combat feel proportionate when they unfold together in the same battlespace.",
			RecommendedTrials:         10,
			MaxSimSeconds:             2400,
			FocusTeam:                 "USA",
			OpposingTeam:              "IRN",
			TrackedTargetUnitID:       "pg-hm-irn-missile-boat",
			EndOnTrackedTargetDisable: true,
			SetupActions: []ProvingGroundSetupAction{
				{Kind: "set_player", TeamID: "USA"},
				{Kind: "seed_detection", TeamID: "USA", TargetUnitID: "pg-hm-irn-missile-boat"},
				{Kind: "seed_detection", TeamID: "USA", TargetUnitID: "pg-hm-irn-f4e"},
				{Kind: "seed_detection", TeamID: "USA", TargetUnitID: "pg-hm-irn-fateh-sub"},
				{Kind: "seed_detection", TeamID: "IRN", TargetUnitID: "pg-hm-usa-destroyer"},
				{Kind: "seed_detection", TeamID: "IRN", TargetUnitID: "pg-hm-usa-virginia"},
				{Kind: "seed_detection", TeamID: "IRN", TargetUnitID: "pg-hm-usa-f35a"},
				{Kind: "assign_attack", UnitID: "pg-hm-usa-destroyer", TargetUnitID: "pg-hm-irn-missile-boat", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-hm-usa-virginia", TargetUnitID: "pg-hm-irn-fateh-sub", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-hm-usa-f35a", TargetUnitID: "pg-hm-irn-f4e", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-hm-irn-missile-boat", TargetUnitID: "pg-hm-usa-destroyer", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-hm-irn-fateh-sub", TargetUnitID: "pg-hm-usa-virginia", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
				{Kind: "assign_attack", UnitID: "pg-hm-irn-f4e", TargetUnitID: "pg-hm-usa-f35a", OrderType: enginev1.AttackOrderType_ATTACK_ORDER_TYPE_ATTACK_ASSIGNED_TARGET, DesiredEffect: enginev1.DesiredEffect_DESIRED_EFFECT_DESTROY},
			},
		},
	} {
		specs[id] = spec
	}
	return specs
}

func ProvingGroundBuiltins() []*enginev1.Scenario {
	return append(packageProvingGroundBuiltins(), []*enginev1.Scenario{
		provingGroundModernVsLegacyAir(),
		provingGroundAEWSupportedIntercept(),
		provingGroundBallisticVsAirbase(),
		provingGroundIsraelMissileDefenseSaturation(),
		provingGroundDestroyerVsMissileBoat(),
		provingGroundMaritimeTransitRules(),
		provingGroundAirspaceRoutingRules(),
		provingGroundLandBorderRoutingRules(),
		provingGroundMaritimeRerouteAroundQatar(),
		provingGroundHormuzTransitPassage(),
		provingGroundAirRerouteAroundQatar(),
		provingGroundHostedStrikeLoadout(),
		provingGroundDefensiveInterceptThirdParty(),
		provingGroundSamVsStrikerOverland(),
		provingGroundVirginiaVsFatehSub(),
		provingGroundUIDemoGulfSkirmish(),
		provingGroundCompositeGulfRaid(),
		provingGroundFuelLimitedDeepStrike(),
		provingGroundSortieRegenerationCycle(),
		provingGroundGulfEscalationCycle(),
		provingGroundHormuzMultiDomainSkirmish(),
	}...)
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
		Position: geo.ProtoPosition(lat, lon, altMsl, heading, speed),
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
