# Scenario Feature Roadmap

Current baseline: scenario loading, YAML-backed unit libraries, named loadouts, and autonomous engagement all work. The items below define the next major feature series for building a modern Iran vs coalition campaign.

## Goal
- Support player-directed strike planning, stationary military and civilian targets, differentiated weapon effects, and configurable unit behavior.
- Keep the simulation at the operational / strategic level. Batteries, airbases, oil facilities, ports, and task groups should usually be modeled as one entity, not as every subordinate component.

## Shared Foundation
- Add target semantics before adding new UI behavior.
- Required new fields:
  - `asset_class`: `combat_unit`, `airbase`, `port`, `oil_field`, `pipeline_node`, `desalination_plant`, `power_plant`, `radar_site`, `c2_site`
  - `target_class`: `surface_warship`, `submarine`, `aircraft`, `sam_battery`, `armor`, `soft_infrastructure`, `hardened_infrastructure`, `runway`, `civilian_energy`, `civilian_water`
  - `stationary`: bool
  - `affiliation`: `military`, `civilian`, `dual_use`
  - `damage_state`: `operational`, `damaged`, `mission_killed`, `destroyed`
  - `engagement_behavior`
  - `attack_order`
- Apply these through proto, DB schema, YAML definitions, editor store, and sim state together.

## 1. Manual Attack Orders
- Add user-issued attack tasks for a selected unit against a selected target.
- Orders should persist in scenario state and override autonomous target selection.
- Initial order types:
  - `attack_assigned_target`
  - `strike_assigned_target_until_effect`
  - `cancel_order`
- Validation rules:
  - target must be known or user-designated
  - shooter must have suitable weapons / loadout
  - sim still enforces range, detection, ammo, and survivability limits
- UI:
  - select unit
  - click `Assign Attack`
  - click target
  - optionally choose preferred loadout / weapon configuration

## 2. Stationary Military and Civilian Assets
- Represent fixed sites as regular scenario entities with `stationary=true`.
- First-wave asset types:
  - airbases
  - naval bases / ports
  - oil export terminals
  - desalination plants
  - power plants
  - radar / C2 sites
  - missile garrisons
- Pipelines should initially be modeled as key nodes, not full map polylines.
- These assets should appear in the editor palette and support country filtering.

## 3. Loadouts as First-Class Gameplay State
- Keep multiple authored loadouts per definition and enforce them operationally.
- Loadout selection should affect:
  - what targets the unit can engage well
  - what attack orders are valid
  - how the AI chooses targets autonomously
- Requirements:
  - preserve selected loadout on placed units
  - expose loadout clearly in unit edit UI
  - warn when a unit is ordered against a target its current loadout cannot service effectively

## 4. Munition Effects Lookup
- Replace binary kill logic with an effects table keyed by `weapon_effect_type x target_class`.
- Example outputs:
  - `no_effect`
  - `light_damage`
  - `mobility_kill`
  - `mission_kill`
  - `runway_crater`
  - `firepower_loss`
  - `catastrophic_kill`
- Example: anti-ship missiles should often mission-kill or heavily damage a frigate before fully sinking it.
- Store this in authored data, not hardcoded conditionals.

## 5. Detection / Engagement Behavior
- Replace universal auto-fire with configurable behavior profiles.
- Initial behaviors:
  - `hold_fire`
  - `self_defense_only`
  - `auto_engage`
  - `auto_engage_pkill_threshold`
  - `assigned_targets_only`
  - `shadow_contact`
  - `withdraw_on_detect`
- Behavior should be a default policy. Manual attack orders should override it when appropriate.

## 6. OPFOR AI For First-24-Hour War Gaming
- Replace deterministic follow-on strike scripting with a bounded operational AI for adversary and non-player behavior.
- Keep scope limited to `USA`, `ISR`, and `IRN` as major actors plus simple defensive behavior for everyone else.

### 6.1 Major-Actor Strike AI
- Add a target-selection layer for `USA`, `ISR`, and `IRN` that chooses targets to cause maximum strategic pain within current sim limits.
- First target categories to score:
  - airbases
  - runway / base throughput nodes
  - missile brigades and drone strike groups
  - SAM / IADS nodes
  - ports / naval bases
  - oil / gas / desalination targets
- Required model inputs:
  - target strategic value
  - current target damage / usability
  - distance / access feasibility
  - weapon suitability
  - expected defensive risk
  - strike cooldown / sortie readiness
- Initial objective bias:
  - `IRN`: maximize coalition sortie disruption, regional base pain, infrastructure shock, and pressure in the Gulf
  - `USA` / `ISR`: maximize suppression of missile/drone forces, IADS degradation, and airbase denial

### 6.2 Lightweight Strike Planner
- Add a reusable planner that chooses:
  - shooter
  - target
  - desired effect
  - route / ingress point
  - launch timing
- Keep it small:
  - no full ATO
  - no tanker scheduling optimizer
  - no detailed package deconfliction
- Planner should support:
  - single-unit opportunistic strikes
  - grouped salvos against one target
  - prioritizing stationary strategic targets over mobile ones when target quality is weak

### 6.3 Third-Country Defensive Autopilot
- Non-player countries should remain defensive/local:
  - engage unauthorized overflight
  - intercept inbound missiles and drones threatening their territory or hosted assets
  - avoid initiating theater-wide offensive campaigns
- This is intentionally not a diplomacy engine. It is a local sovereign-defense behavior layer.

### 6.4 Scoring Inputs The AI Needs
- Add authored or derived strategic-value weights on targets and units:
  - `airbase_value`
  - `missile_force_value`
  - `iads_value`
  - `infrastructure_value`
  - `naval_chokepoint_value`
- The first AI should optimize against those weights rather than trying to infer doctrine from raw unit types alone.

### 6.5 Bounded AI Non-Goals
- No full logistics AI
- No dynamic diplomacy
- No operational research-grade package optimization
- No campaign-level reinforcement planning
- No political / escalation AI beyond target weights

## Recommended Milestones
1. Data-model foundation
2. Stationary assets in library + editor
3. Damage states + munition effects table
4. Manual attack orders
5. Loadout-aware order validation
6. Engagement behavior profiles
7. Base capacity and first-day operational effects
8. Major-actor OPFOR strike AI
9. Scenario-specific UX polish and scoring

## Validation Scenarios
- F-15I ordered to strike an S-300 site with a strike loadout
- Sa'ar 6 attacking an Iranian corvette and causing damage before destruction
- Iranian missile brigade striking a desalination plant and leaving it mission-killed
- Patriot battery set to `self_defense_only` while nearby fighters remain on `auto_engage`
- Tanker aircraft ordered to hold while escorts attack assigned targets
- Iranian AI choosing between Al Udeid, Al Dhafra, and Israeli airbases based on current operational impact
- Israeli / U.S. AI reprioritizing from air defenses to missile brigades after the first successful suppression wave

## Likely Refactors
- Separate `unit` from `asset` presentation in the editor without splitting the core entity model
- Move autonomous target selection logic out of `AdjudicateTick` into behavior-aware helpers
- Add a dedicated authored-data layer for weapon effects rather than growing `weapons.go` into behavior logic
- Split strike-planning heuristics from low-level engagement execution so AI target choice does not further bloat `AdjudicateTick`
