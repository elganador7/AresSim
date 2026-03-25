package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
	"github.com/aressim/internal/library"
	"github.com/aressim/internal/scenario"
	"github.com/aressim/internal/sim"
	"google.golang.org/protobuf/proto"
)

type runSummary struct {
	ScenarioID               string                   `json:"scenarioId"`
	Name                     string                   `json:"name"`
	Category                 string                   `json:"category"`
	Purpose                  string                   `json:"purpose"`
	ExpectedSummary          string                   `json:"expectedSummary"`
	Trials                   int                      `json:"trials"`
	FocusTeam                string                   `json:"focusTeam"`
	FocusWinRate             float64                  `json:"focusWinRate"`
	TargetMissionKillRate    float64                  `json:"targetMissionKillRate"`
	TargetDestroyedRate      float64                  `json:"targetDestroyedRate"`
	MeanElapsedSeconds       float64                  `json:"meanElapsedSeconds"`
	MeanFirstShotSeconds     float64                  `json:"meanFirstShotSeconds"`
	MeanShotsFired           float64                  `json:"meanShotsFired"`
	MeanHitsScored           float64                  `json:"meanHitsScored"`
	MeanFuelExhaustions      float64                  `json:"meanFuelExhaustions"`
	MeanReplenishments       float64                  `json:"meanReplenishments"`
	MeanFocusLosses          float64                  `json:"meanFocusLosses"`
	MeanOpposingLosses       float64                  `json:"meanOpposingLosses"`
	Pass                     bool                     `json:"pass"`
	TerminalReasons          map[string]int           `json:"terminalReasons,omitempty"`
	SampleEvents             []sim.ProvingGroundEvent `json:"sampleEvents,omitempty"`
	MinFocusWinRate          float64                  `json:"minFocusWinRate,omitempty"`
	MaxFocusWinRate          float64                  `json:"maxFocusWinRate,omitempty"`
	MinTargetMissionKillRate float64                  `json:"minTargetMissionKillRate,omitempty"`
	MaxTargetMissionKillRate float64                  `json:"maxTargetMissionKillRate,omitempty"`
}

func main() {
	scenarioID := flag.String("scenario", "all", "proving-ground scenario id or 'all'")
	trials := flag.Int("trials", 0, "number of trials to run; 0 uses scenario default")
	pretty := flag.Bool("pretty", true, "pretty-print JSON output")
	flag.Parse()

	defs, err := library.LoadAll("")
	if err != nil {
		log.Fatalf("load library definitions: %v", err)
	}
	defsByID := make(map[string]library.Definition, len(defs))
	simDefs := make(map[string]sim.DefStats, len(defs))
	for _, def := range defs {
		defsByID[def.ID] = def
		simDefs[def.ID] = defStatsFromLibraryDefinition(def)
	}
	weapons := buildWeaponCatalog()
	specs := scenario.ProvingGroundSpecs()

	ids := make([]string, 0, len(specs))
	if strings.EqualFold(strings.TrimSpace(*scenarioID), "all") {
		for id := range specs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
	} else {
		id := strings.TrimSpace(*scenarioID)
		if _, ok := specs[id]; !ok {
			log.Fatalf("unknown proving-ground scenario %q", id)
		}
		ids = append(ids, id)
	}

	results := make([]runSummary, 0, len(ids))
	for _, id := range ids {
		spec := specs[id]
		builtin := scenario.BuiltinByID(id)
		if builtin == nil {
			log.Fatalf("built-in scenario %q not found", id)
		}
		trialCount := *trials
		if trialCount <= 0 {
			trialCount = spec.RecommendedTrials
		}
		trialResults := make([]sim.ProvingGroundTrialResult, 0, trialCount)
		for i := 0; i < trialCount; i++ {
			scen := proto.Clone(builtin).(*enginev1.Scenario)
			prepareScenarioForHeadlessRun(scen, defsByID)
			applyScenarioSetupActions(scen, defsByID, spec)
			rules := sim.BuildRelationshipRules(scen.GetRelationships())
			trialResults = append(trialResults, sim.RunProvingGroundTrial(
				scen,
				simDefs,
				weapons,
				rules,
				spec.MaxSimSeconds,
				spec.FocusTeam,
				spec.OpposingTeam,
				spec.TrackedTargetUnitID,
				spec.EndOnTrackedTargetDisable,
				int64(i+1),
			))
		}
		aggregate := sim.AggregateProvingGroundResults(trialResults, spec.FocusTeam)
		results = append(results, runSummary{
			ScenarioID:               id,
			Name:                     builtin.GetName(),
			Category:                 spec.Category,
			Purpose:                  spec.Purpose,
			ExpectedSummary:          spec.ExpectedSummary,
			Trials:                   aggregate.Trials,
			FocusTeam:                aggregate.FocusTeam,
			FocusWinRate:             aggregate.FocusWinRate,
			TargetMissionKillRate:    aggregate.TargetMissionKillRate,
			TargetDestroyedRate:      aggregate.TargetDestroyedRate,
			MeanElapsedSeconds:       aggregate.MeanElapsedSeconds,
			MeanFirstShotSeconds:     aggregate.MeanFirstShotSeconds,
			MeanShotsFired:           aggregate.MeanShotsFired,
			MeanHitsScored:           aggregate.MeanHitsScored,
			MeanFuelExhaustions:      aggregate.MeanFuelExhaustions,
			MeanReplenishments:       aggregate.MeanReplenishments,
			MeanFocusLosses:          aggregate.MeanFocusLosses,
			MeanOpposingLosses:       aggregate.MeanOpposingLosses,
			Pass:                     passesExpectedBands(aggregate, spec),
			TerminalReasons:          aggregate.TerminalReasons,
			SampleEvents:             aggregate.SampleEvents,
			MinFocusWinRate:          spec.MinFocusWinRate,
			MaxFocusWinRate:          spec.MaxFocusWinRate,
			MinTargetMissionKillRate: spec.MinTargetMissionKillRate,
			MaxTargetMissionKillRate: spec.MaxTargetMissionKillRate,
		})
	}

	var data []byte
	if *pretty {
		data, err = json.MarshalIndent(results, "", "  ")
	} else {
		data, err = json.Marshal(results)
	}
	if err != nil {
		log.Fatalf("marshal results: %v", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
}

func passesExpectedBands(aggregate sim.ProvingGroundAggregate, spec scenario.ProvingGroundSpec) bool {
	pass := true
	if spec.MinFocusWinRate > 0 && aggregate.FocusWinRate < spec.MinFocusWinRate {
		pass = false
	}
	if spec.MaxFocusWinRate > 0 && aggregate.FocusWinRate > spec.MaxFocusWinRate {
		pass = false
	}
	if spec.MinTargetMissionKillRate > 0 && aggregate.TargetMissionKillRate < spec.MinTargetMissionKillRate {
		pass = false
	}
	if spec.MaxTargetMissionKillRate > 0 && aggregate.TargetMissionKillRate > spec.MaxTargetMissionKillRate {
		pass = false
	}
	return pass
}

func prepareScenarioForHeadlessRun(scen *enginev1.Scenario, defsByID map[string]library.Definition) {
	for _, unit := range scen.GetUnits() {
		if unit.GetBaseOps() == nil {
			unit.BaseOps = &enginev1.BaseOpsState{
				State: enginev1.FacilityOperationalState_FACILITY_OPERATIONAL_STATE_USABLE,
			}
		}
		if len(unit.GetWeapons()) > 0 {
			continue
		}
		defID := strings.TrimSpace(unit.GetDefinitionId())
		def, ok := defsByID[defID]
		if !ok {
			continue
		}
		if loadoutID, slots := selectWeaponConfiguration(def, unit.GetLoadoutConfigurationId()); len(slots) > 0 {
			unit.LoadoutConfigurationId = loadoutID
			unit.Weapons = loadoutToWeaponStates(slots)
			continue
		}
		if len(def.DefaultLoadout) > 0 {
			unit.LoadoutConfigurationId = "default"
			unit.Weapons = loadoutToWeaponStates(def.DefaultLoadout)
			continue
		}
		unit.Weapons = scenario.InitUnitWeapons(unit, int32(def.GeneralType))
	}
}

func applyScenarioSetupActions(scen *enginev1.Scenario, defsByID map[string]library.Definition, spec scenario.ProvingGroundSpec) {
	for _, action := range spec.SetupActions {
		switch action.Kind {
		case "set_loadout":
			unit := findUnitByID(scen.GetUnits(), action.UnitID)
			if unit == nil {
				continue
			}
			def, ok := defsByID[strings.TrimSpace(unit.GetDefinitionId())]
			if !ok {
				continue
			}
			loadoutID, slots := selectWeaponConfiguration(def, action.LoadoutConfiguration)
			if len(slots) == 0 {
				continue
			}
			unit.LoadoutConfigurationId = loadoutID
			unit.Weapons = loadoutToWeaponStates(slots)
		case "move":
			unit := findUnitByID(scen.GetUnits(), action.UnitID)
			if unit == nil {
				continue
			}
			unit.MoveOrder = &enginev1.MoveOrder{
				Waypoints: []*enginev1.Waypoint{{
					Lat: action.Lat,
					Lon: action.Lon,
				}},
				AutoGenerated: false,
			}
		case "append_waypoint":
			unit := findUnitByID(scen.GetUnits(), action.UnitID)
			if unit == nil {
				continue
			}
			if unit.MoveOrder == nil {
				unit.MoveOrder = &enginev1.MoveOrder{}
			}
			unit.MoveOrder.Waypoints = append(unit.MoveOrder.Waypoints, &enginev1.Waypoint{
				Lat: action.Lat,
				Lon: action.Lon,
			})
		case "assign_attack":
			unit := findUnitByID(scen.GetUnits(), action.UnitID)
			if unit == nil {
				continue
			}
			unit.AttackOrder = &enginev1.AttackOrder{
				OrderType:      action.OrderType,
				TargetUnitId:   action.TargetUnitID,
				DesiredEffect:  action.DesiredEffect,
				PkillThreshold: 0.7,
			}
		}
	}
}

func findUnitByID(units []*enginev1.Unit, id string) *enginev1.Unit {
	for _, unit := range units {
		if unit != nil && unit.GetId() == id {
			return unit
		}
	}
	return nil
}

func selectWeaponConfiguration(def library.Definition, preferredID string) (string, []library.LoadoutSlot) {
	preferredID = strings.TrimSpace(preferredID)
	defaultID := strings.TrimSpace(def.DefaultWeaponConfiguration)
	if preferredID != "" {
		for _, cfg := range def.WeaponConfigurations {
			if cfg.ID == preferredID {
				return cfg.ID, cfg.Loadout
			}
		}
	}
	if defaultID != "" {
		for _, cfg := range def.WeaponConfigurations {
			if cfg.ID == defaultID {
				return cfg.ID, cfg.Loadout
			}
		}
	}
	if len(def.WeaponConfigurations) > 0 {
		return def.WeaponConfigurations[0].ID, def.WeaponConfigurations[0].Loadout
	}
	return "", nil
}

func loadoutToWeaponStates(slots []library.LoadoutSlot) []*enginev1.WeaponState {
	states := make([]*enginev1.WeaponState, 0, len(slots))
	for _, slot := range slots {
		states = append(states, &enginev1.WeaponState{
			WeaponId:   slot.WeaponID,
			CurrentQty: slot.InitialQty,
			MaxQty:     slot.MaxQty,
		})
	}
	return states
}

func defStatsFromLibraryDefinition(def library.Definition) sim.DefStats {
	return sim.DefStats{
		CruiseSpeedMps:              float64(def.CruiseSpeedMps),
		BaseStrength:                float64(def.BaseStrength),
		Accuracy:                    float64(def.Accuracy),
		DetectionRangeM:             float64(def.DetectionRangeM),
		RadarCrossSectionM2:         float64(def.RadarCrossSectionM2),
		FuelCapacityLiters:          float64(def.FuelCapacityLiters),
		FuelBurnRateLph:             float64(def.FuelBurnRateLph),
		GeneralType:                 int32(def.GeneralType),
		EmploymentRole:              strings.TrimSpace(def.EmploymentRole),
		ReplacementCostUSD:          def.ReplacementCostUSD,
		StrategicValueUSD:           def.StrategicValueUSD,
		EconomicValueUSD:            def.EconomicValueUSD,
		Domain:                      enginev1.UnitDomain(def.Domain),
		TargetClass:                 strings.TrimSpace(def.TargetClass),
		AssetClass:                  strings.TrimSpace(def.AssetClass),
		EmbarkedFixedWingCapacity:   def.EmbarkedFixedWingCapacity,
		EmbarkedRotaryWingCapacity:  def.EmbarkedRotaryWingCapacity,
		EmbarkedUAVCapacity:         def.EmbarkedUavCapacity,
		LaunchCapacityPerInterval:   def.LaunchCapacityPerInterval,
		RecoveryCapacityPerInterval: def.RecoveryCapacityPerInterval,
		SortieIntervalMinutes:       def.SortieIntervalMinutes,
	}
}

func buildWeaponCatalog() map[string]sim.WeaponStats {
	catalog := make(map[string]sim.WeaponStats)
	for _, wd := range scenario.DefaultWeaponDefinitions() {
		if wd == nil {
			continue
		}
		catalog[wd.GetId()] = sim.WeaponStats{
			RangeM:           float64(wd.GetRangeM()),
			SpeedMps:         float64(wd.GetSpeedMps()),
			ProbabilityOfHit: float64(wd.GetProbabilityOfHit()),
			DomainTargets:    wd.GetDomainTargets(),
			Guidance:         wd.GetGuidance(),
			EffectType:       wd.GetEffectType(),
		}
	}
	return catalog
}
