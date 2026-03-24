package main

import (
	"fmt"

	"github.com/aressim/internal/scenario"
	"github.com/aressim/internal/sim"
	"google.golang.org/protobuf/proto"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func (a *App) prepareProvingGroundScenario(id string) (*enginev1.Scenario, scenario.ProvingGroundSpec, error) {
	spec, ok := scenario.ProvingGroundSpecs()[stripTablePrefix(id)]
	if !ok {
		return nil, scenario.ProvingGroundSpec{}, fmt.Errorf("scenario %s is not a proving-ground scenario", id)
	}
	builtin := scenario.BuiltinByID(spec.ScenarioID)
	if builtin == nil {
		return nil, scenario.ProvingGroundSpec{}, fmt.Errorf("built-in proving-ground scenario %s not found", spec.ScenarioID)
	}
	scen := proto.Clone(builtin).(*enginev1.Scenario)
	a.currentScenario = scen
	a.setHumanControlledTeam("")
	a.setSimSeconds(0)
	a.lastDetMu.Lock()
	a.lastDetections = nil
	a.lastDetMu.Unlock()
	a.prepareScenarioForSimulation(scen)
	return scen, spec, nil
}

func (a *App) applyProvingGroundSetup(spec scenario.ProvingGroundSpec) error {
	for _, action := range spec.SetupActions {
		switch action.Kind {
		case "set_player":
			if res := a.SetHumanControlledTeam(action.TeamID); !res.Success {
				return fmt.Errorf("set player %s: %s", action.TeamID, res.Error)
			}
		case "set_loadout":
			if res := a.SetUnitLoadoutConfiguration(action.UnitID, action.LoadoutConfiguration); !res.Success {
				return fmt.Errorf("set loadout %s on %s: %s", action.LoadoutConfiguration, action.UnitID, res.Error)
			}
		case "seed_detection":
			a.storeLastDetection(action.TeamID, []string{action.TargetUnitID})
		case "move":
			if res := a.MoveUnit(action.UnitID, action.Lat, action.Lon); !res.Success {
				return fmt.Errorf("move %s: %s", action.UnitID, res.Error)
			}
		case "append_waypoint":
			if res := a.AppendMoveWaypoint(action.UnitID, action.Lat, action.Lon); !res.Success {
				return fmt.Errorf("append waypoint for %s: %s", action.UnitID, res.Error)
			}
		case "preview_target":
			playerTeam := action.TeamID
			if playerTeam == "" {
				playerTeam = a.getHumanControlledTeam()
			}
			options, err := a.PreviewTargetEngagementOptions(action.TargetUnitID, playerTeam)
			if err != nil {
				return fmt.Errorf("preview target %s: %w", action.TargetUnitID, err)
			}
			if action.ExpectedShooterUnitID == "" {
				continue
			}
			found := false
			for _, option := range options {
				if option.ShooterUnitId != action.ExpectedShooterUnitID {
					continue
				}
				found = true
				if !option.ReadyToFire && !option.CanAssign {
					return fmt.Errorf("preview target %s found shooter %s but it was blocked: %s", action.TargetUnitID, action.ExpectedShooterUnitID, option.Reason)
				}
				break
			}
			if !found {
				return fmt.Errorf("preview target %s did not include expected shooter %s", action.TargetUnitID, action.ExpectedShooterUnitID)
			}
		case "assign_attack":
			res := a.SetUnitAttackOrder(
				action.UnitID,
				int32(action.OrderType),
				action.TargetUnitID,
				int32(action.DesiredEffect),
				0.7,
			)
			if !res.Success {
				return fmt.Errorf("assign attack %s -> %s: %s", action.UnitID, action.TargetUnitID, res.Error)
			}
		default:
			return fmt.Errorf("unknown proving-ground action %q", action.Kind)
		}
	}
	return nil
}

func (a *App) runPreparedProvingGroundTrial(spec scenario.ProvingGroundSpec, seed int64) sim.ProvingGroundTrialResult {
	return sim.RunProvingGroundTrial(
		a.currentScenario,
		a.getCachedDefs(),
		a.getCachedWeaponCatalog(),
		a.relationshipRules(),
		spec.MaxSimSeconds,
		spec.FocusTeam,
		spec.OpposingTeam,
		spec.TrackedTargetUnitID,
		spec.EndOnTrackedTargetDisable,
		seed,
	)
}
