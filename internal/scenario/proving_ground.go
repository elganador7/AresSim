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
	}
}

func ProvingGroundBuiltins() []*enginev1.Scenario {
	return []*enginev1.Scenario{
		provingGroundModernVsLegacyAir(),
		provingGroundAEWSupportedIntercept(),
		provingGroundBallisticVsAirbase(),
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

func provingGroundMaritimeTransitRules() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-maritime-transit-rules",
		Name:           "Proving Ground: Maritime Transit Rules",
		Description:    "Manual transit-permission scenario. Play as USA. Try moving the surface ship into Qatari territorial waters near 25.445, 51.669 and it should be blocked. Try moving the submarine to the same point and it should be allowed.",
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
				u := provingGroundUnit("pg-mtr-usa-ship", "MMSC", "U.S. Surface Combatant - Transit Rules", "USA", "COALITION_WEST", "al-jubail-mmsc-saudi", 25.30, 51.80, 0, 90, 12)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "asm-nsm", CurrentQty: 8, MaxQty: 8}}
				return u
			}(),
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-mtr-usa-sub", "Virginia", "USS Virginia - Transit Rules", "USA", "COALITION_WEST", "virginia-block-v", 25.22, 51.92, 0, 90, 7)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "torp-mk48", CurrentQty: 6, MaxQty: 6}}
				return u
			}(),
		},
	}
}

func provingGroundAirspaceRoutingRules() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-airspace-routing-rules",
		Name:           "Proving Ground: Airspace Routing Rules",
		Description:    "Manual airspace-transit scenario. Play as USA. Try moving the fighter toward Bushehr at 28.950, 50.830 and it should be blocked. Try moving it east over the Gulf of Oman near 24.900, 56.000 and it should be allowed.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{{
			FromCountry:            "USA",
			ToCountry:              "QAT",
			AirspaceTransitAllowed: true,
		}},
		Units: []*enginev1.Unit{
			provingAircraft("pg-arr-usa-f35a", "F-35A", "USAF F-35A - Airspace Routing Rules", "USA", "COALITION_WEST", "f35a-lightning", 25.12, 51.31, 9_500, 90, 240),
		},
	}
}

func provingGroundLandBorderRoutingRules() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-land-border-routing-rules",
		Name:           "Proving Ground: Land Border Routing Rules",
		Description:    "Manual land-border scenario. Play as USA. Try moving the convoy into Iran near 30.400, 48.200 and it should be blocked. Try moving it within Iraq near 30.250, 47.500 and it should be allowed.",
		Classification: "PROVING GROUND",
		Author:         "AresSim Calibration",
		Version:        "1.0.0",
		StartTimeUnix:  float64(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Unix()),
		Settings: &enginev1.SimulationSettings{
			TickRateHz: 10,
			TimeScale:  1.0,
		},
		Map: clearWeatherMap(),
		Relationships: []*enginev1.CountryRelationship{{
			FromCountry:            "USA",
			ToCountry:              "IRQ",
			AirspaceTransitAllowed: true,
		}},
		Units: []*enginev1.Unit{
			provingGroundUnit("pg-lbr-usa-convoy", "U.S. Convoy", "U.S. Ground Convoy - Border Rules", "USA", "COALITION_WEST", "stryker-company", 30.50, 47.70, 0, 90, 15),
		},
	}
}

func provingGroundMaritimeRerouteAroundQatar() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-maritime-reroute-around-qatar",
		Name:           "Proving Ground: Maritime Reroute Around Qatar",
		Description:    "Manual reroute scenario. Play as USA. Move the ship from west of Qatar to 24.800, 52.300 and it should succeed with a curved route around Qatar instead of a straight line.",
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
			{FromCountry: "USA", ToCountry: "SAU", MaritimeTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "ARE", MaritimeTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "BHR", MaritimeTransitAllowed: true},
		},
		Units: []*enginev1.Unit{
			func() *enginev1.Unit {
				u := provingGroundUnit("pg-mraq-usa-ship", "MMSC", "U.S. Surface Combatant - Qatar Reroute", "USA", "COALITION_WEST", "al-jubail-mmsc-saudi", 24.90, 50.30, 0, 90, 12)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "asm-nsm", CurrentQty: 8, MaxQty: 8}}
				return u
			}(),
		},
	}
}

func provingGroundHormuzTransitPassage() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-hormuz-transit-passage",
		Name:           "Proving Ground: Hormuz Transit Passage",
		Description:    "Manual routing scenario. Play as USA. Move the ship east through the Strait of Hormuz toward 26.340, 56.820. It should route through the transit lane without crossing Musandam or nearby land.",
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
				u := provingGroundUnit("pg-htp-usa-ship", "MMSC", "U.S. Surface Combatant - Hormuz Transit", "USA", "COALITION_WEST", "al-jubail-mmsc-saudi", 25.86, 55.18, 0, 70, 12)
				u.Weapons = []*enginev1.WeaponState{{WeaponId: "asm-nsm", CurrentQty: 8, MaxQty: 8}}
				return u
			}(),
		},
	}
}

func provingGroundAirRerouteAroundQatar() *enginev1.Scenario {
	return &enginev1.Scenario{
		Id:             "pg-air-reroute-around-qatar",
		Name:           "Proving Ground: Air Reroute Around Qatar",
		Description:    "Manual reroute scenario. Play as USA. Move the fighter east toward 24.800, 53.300 and it should succeed with a rerouted path that bends around denied Qatari airspace.",
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
			{FromCountry: "USA", ToCountry: "SAU", AirspaceTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "ARE", AirspaceTransitAllowed: true},
			{FromCountry: "USA", ToCountry: "BHR", AirspaceTransitAllowed: true},
		},
		Units: []*enginev1.Unit{
			provingAircraft("pg-arq-usa-f35a", "F-35A", "USAF F-35A - Qatar Air Reroute", "USA", "COALITION_WEST", "f35a-lightning", 24.90, 50.30, 9_500, 90, 240),
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
