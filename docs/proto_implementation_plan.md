# Ares Sim — Protobuf Implementation Plan

## Design Philosophy

The proto schema is the **single source of truth** for the entire simulation. Every system — the adjudicator loop, the persistence layer, the frontend renderer, and the event stream — speaks in these types. Changes to the simulation model always start here.

The schema is designed around a central tension: **operational fidelity now, strategic scale later**. Every MVP message is authored so that its Phase 2/3 extensions are obvious additions, never breaking renames. Fields that will matter at the strategic level (nation ownership, commodity costs, production linkage) are defined as reserved or stub fields from day one so the graph of dependencies is never a surprise.

### Core Principles

1. **Effects over hitpoints.** Combat resolution produces `CombatEffects` (attrition, suppression, disruption, exhaustion), not a single `health` float. A unit at 40% strength that is suppressed and disrupted behaves differently than a unit at 40% strength that is simply understrength. The adjudicator must distinguish these states.

2. **Echelon-aware hierarchy.** Units exist in a C2 graph. A battalion's combat effectiveness is a function of its subordinate companies' states AND whether it has comms with its parent brigade HQ. This graph is native to the schema from day one.

3. **Domain-complete from MVP.** All four warfare domains — Land, Air, Sea, Cyber — are enumerated even if only Land units are implemented in Phase 1. This prevents the frontend and adjudicator from ever hardcoding domain assumptions.

4. **Real-world unit fidelity.** Unit capabilities are modeled after actual military parameters: move rates by terrain type, direct fire ranges in meters matching real weapons, sensor envelopes reflecting actual detection systems. The game is only as credible as the underlying data.

5. **Strategic extensibility as a first-class constraint.** Every `Unit` carries a `nation_id`. Every capability has a `supply_consumption_rate`. Every logistics node has a `commodity_type`. These fields are unused in Phase 1 but their presence ensures the strategic layer is grafted on, not bolted on.

---

## Package Structure

The schema is split into focused files compiled together. All files share `package engine`.

```
proto/
├── common.proto         # Shared primitives: Position, Timestamp, Geometry
├── unit.proto           # Unit identity, echelon, domain, C2 hierarchy
├── capabilities.proto   # Firepower, mobility, protection, sensors, EW
├── status.proto         # Operational state: strength, ammo, fuel, effects
├── orders.proto         # Order types, tactical tasks, fire missions
├── combat.proto         # Engagement resolution inputs/outputs
├── logistics.proto      # Supply nodes, links, commodity flows
├── intelligence.proto   # Fog of war, detection, SIGINT (Phase 2)
├── scenario.proto       # Scenario setup, environment, objectives
├── events.proto         # All backend→frontend streaming event types
└── strategic.proto      # Nations, production, commodities, infrastructure (Phase 3)
```

---

## `common.proto`

Shared primitives used across all other files.

```protobuf
syntax = "proto3";
package engine;
option go_package = "github.com/yourorg/aressim/internal/gen/engine";

import "google/protobuf/timestamp.proto";

// WGS84 position. The canonical coordinate type for all geospatial fields.
message Position {
  double lat     = 1;  // Decimal degrees, -90 to +90
  double lon     = 2;  // Decimal degrees, -180 to +180
  double alt_msl = 3;  // Meters above mean sea level
  double heading = 4;  // True north bearing, 0-360 degrees
  double speed   = 5;  // Current speed, meters per second
}

// A bounding polygon for area-of-operations, objectives, engagement zones, etc.
message Polygon {
  repeated Position vertices = 1;
  string label               = 2;
  PolygonType type           = 3;
}

enum PolygonType {
  POLYGON_TYPE_UNKNOWN          = 0;
  POLYGON_TYPE_AREA_OF_OPS      = 1;
  POLYGON_TYPE_OBJECTIVE        = 2;
  POLYGON_TYPE_ENGAGEMENT_ZONE  = 3;
  POLYGON_TYPE_NO_FIRE_ZONE     = 4;
  POLYGON_TYPE_MINEFIELD        = 5;
}

// Simulation-internal time reference. Separate from wall-clock time.
message SimTime {
  double seconds_elapsed = 1;  // Seconds since scenario start
  int64  tick_number     = 2;  // Absolute tick counter
}

// Standardized result wrapper for adjudicator operations.
message OperationResult {
  bool   success = 1;
  string error   = 2;  // Empty on success
}
```

---

## `unit.proto`

The core unit identity model. Describes what a unit *is*, not what it can do (that's `capabilities.proto`) or how it currently is (that's `status.proto`).

```protobuf
syntax = "proto3";
package engine;
option go_package = "github.com/yourorg/aressim/internal/gen/engine";

import "capabilities.proto";
import "status.proto";
import "orders.proto";

message Unit {
  // Identity
  string id              = 1;   // UUID
  string display_name    = 2;   // e.g. "1-64 AR" (1st Battalion, 64th Armor)
  string full_name       = 3;   // e.g. "1st Battalion, 64th Armor Regiment"
  string side            = 4;   // "Blue", "Red", "Neutral", or custom faction name
  string nation_id       = 5;   // Strategic layer: owning nation (Phase 3)

  // Classification
  UnitEchelon  echelon   = 6;
  UnitDomain   domain    = 7;
  UnitFunction function  = 8;
  UnitType     type      = 9;

  // Geospatial
  Position     position  = 10;

  // Capabilities: what this unit type can do at 100% strength
  Capabilities capabilities = 11;

  // Status: current operational state (degrades from capabilities baseline)
  OperationalStatus status = 12;

  // Orders: queue of assigned tasks, processed front-to-back
  repeated Order orders = 13;

  // C2 Hierarchy (graph references — resolved via SurrealDB graph edges)
  string parent_unit_id              = 14;  // Commanding HQ unit ID
  repeated string subordinate_ids    = 15;  // Direct subordinates
  repeated string attached_unit_ids  = 16;  // OPCON/attached, not organic

  // Formation: current tactical posture affecting movement and protection
  FormationPosture posture = 17;

  // Symbol: NATO APP-6 symbol code for COP rendering
  string nato_symbol_sidc = 18;  // e.g. "SFGPUCI----E" for friendly armor
}

// ─── ECHELON ──────────────────────────────────────────────────────────────────
// Echelon defines the organizational size. The adjudicator uses this to scale
// default capability values when units are instantiated from a template.

enum UnitEchelon {
  ECHELON_UNKNOWN    = 0;
  ECHELON_FIRETEAM   = 1;   // 2-4 personnel
  ECHELON_SQUAD      = 2;   // 8-13 personnel
  ECHELON_SECTION    = 3;   // 10-20 personnel
  ECHELON_PLATOON    = 4;   // 30-50 personnel
  ECHELON_COMPANY    = 5;   // 80-200 personnel (Battery/Troop equivalent)
  ECHELON_BATTALION  = 6;   // 300-1000 personnel (Squadron equivalent)
  ECHELON_BRIGADE    = 7;   // 2000-5000 personnel (Regiment equivalent)
  ECHELON_DIVISION   = 8;   // 10,000-20,000 personnel
  ECHELON_CORPS      = 9;   // 30,000-50,000 personnel
  ECHELON_ARMY       = 10;  // 100,000+ personnel
  ECHELON_ARMY_GROUP = 11;  // Strategic
}

// ─── DOMAIN ───────────────────────────────────────────────────────────────────
// Domain determines which movement, terrain, and engagement rules apply.

enum UnitDomain {
  DOMAIN_UNKNOWN    = 0;
  DOMAIN_LAND       = 1;
  DOMAIN_AIR        = 2;
  DOMAIN_SEA        = 3;
  DOMAIN_SUBSURFACE = 4;  // Submarines
  DOMAIN_SPACE      = 5;  // Phase 3: satellite assets
  DOMAIN_CYBER      = 6;  // Phase 3: cyber units
}

// ─── FUNCTION ─────────────────────────────────────────────────────────────────
// High-level functional category. Determines which capability templates apply
// and how the adjudicator weights the unit's role in combat resolution.

enum UnitFunction {
  FUNCTION_UNKNOWN       = 0;
  FUNCTION_MANEUVER      = 1;  // Primary combat: armor, infantry, marines
  FUNCTION_FIRES         = 2;  // Indirect fire: artillery, MLRS, mortars
  FUNCTION_AIR_DEFENSE   = 3;  // Air defense: SHORAD, HIMAD
  FUNCTION_AVIATION      = 4;  // Rotary and fixed wing
  FUNCTION_LOGISTICS     = 5;  // Supply, maintenance, medical, transport
  FUNCTION_INTELLIGENCE  = 6;  // Recon, SIGINT, HUMINT
  FUNCTION_ENGINEER      = 7;  // Combat engineer, bridging, obstacle
  FUNCTION_COMMAND       = 8;  // HQ, C2 nodes
  FUNCTION_ELECTRONIC_WARFARE = 9;  // EW, jamming, deception
  FUNCTION_SPECIAL_OPERATIONS  = 10;
}

// ─── TYPE ─────────────────────────────────────────────────────────────────────
// Specific unit type within a domain/function combination.
// This is what drives capability template lookup in the adjudicator.

enum UnitType {
  // ── Land Maneuver ──
  UNIT_TYPE_UNKNOWN                = 0;
  UNIT_TYPE_ARMOR                  = 1;   // Main Battle Tank
  UNIT_TYPE_MECHANIZED_INFANTRY    = 2;   // IFV/APC mounted
  UNIT_TYPE_LIGHT_INFANTRY         = 3;
  UNIT_TYPE_AIRBORNE               = 4;
  UNIT_TYPE_AIR_ASSAULT            = 5;
  UNIT_TYPE_MARINE                 = 6;
  UNIT_TYPE_SPECIAL_FORCES         = 7;
  UNIT_TYPE_CAVALRY                = 8;   // Reconnaissance/screening

  // ── Land Fires ──
  UNIT_TYPE_SELF_PROPELLED_ARTILLERY = 10;
  UNIT_TYPE_TOWED_ARTILLERY          = 11;
  UNIT_TYPE_ROCKET_ARTILLERY         = 12;  // MLRS, HIMARS, Grad
  UNIT_TYPE_MORTAR                   = 13;
  UNIT_TYPE_COASTAL_DEFENSE_MISSILE  = 14;

  // ── Land Air Defense ──
  UNIT_TYPE_SHORAD          = 20;  // Short Range AD (Stinger, Igla, Gepard)
  UNIT_TYPE_HIMAD           = 21;  // High/Medium Altitude AD (Patriot, S-300)
  UNIT_TYPE_MANPADS_TEAM    = 22;

  // ── Aviation ──
  UNIT_TYPE_ATTACK_HELICOPTER   = 30;  // Apache, Ka-52
  UNIT_TYPE_TRANSPORT_HELICOPTER= 31;  // Blackhawk, Mi-8
  UNIT_TYPE_FIGHTER             = 32;  // F-16, Su-27
  UNIT_TYPE_MULTIROLE           = 33;  // F-35, Su-35
  UNIT_TYPE_ATTACK_AIRCRAFT     = 34;  // A-10, Su-25
  UNIT_TYPE_BOMBER              = 35;
  UNIT_TYPE_TRANSPORT_AIRCRAFT  = 36;
  UNIT_TYPE_AWACS               = 37;
  UNIT_TYPE_ISR_AIRCRAFT        = 38;
  UNIT_TYPE_UAV_RECON           = 39;
  UNIT_TYPE_UCAV                = 40;
  UNIT_TYPE_EW_AIRCRAFT         = 41;

  // ── Naval ──
  UNIT_TYPE_AIRCRAFT_CARRIER    = 50;
  UNIT_TYPE_DESTROYER           = 51;
  UNIT_TYPE_FRIGATE             = 52;
  UNIT_TYPE_CORVETTE            = 53;
  UNIT_TYPE_PATROL_BOAT         = 54;
  UNIT_TYPE_AMPHIBIOUS_SHIP     = 55;
  UNIT_TYPE_REPLENISHMENT_SHIP  = 56;
  UNIT_TYPE_ATTACK_SUBMARINE    = 57;
  UNIT_TYPE_BALLISTIC_MISSILE_SUB = 58;
  UNIT_TYPE_MINE_WARFARE        = 59;

  // ── Logistics ──
  UNIT_TYPE_SUPPLY_COMPANY      = 70;
  UNIT_TYPE_FUEL_COMPANY        = 71;
  UNIT_TYPE_MAINTENANCE_COMPANY = 72;
  UNIT_TYPE_MEDICAL_COMPANY     = 73;
  UNIT_TYPE_TRANSPORT_COMPANY   = 74;

  // ── Engineer ──
  UNIT_TYPE_COMBAT_ENGINEER     = 80;
  UNIT_TYPE_BRIDGE_ENGINEER     = 81;
  UNIT_TYPE_CONSTRUCTION        = 82;
  UNIT_TYPE_MINE_CLEARING       = 83;

  // ── Command ──
  UNIT_TYPE_COMMAND_POST        = 90;
  UNIT_TYPE_SIGNAL              = 91;

  // ── EW / Intel ──
  UNIT_TYPE_SIGINT              = 100;
  UNIT_TYPE_JAMMING             = 101;
  UNIT_TYPE_DECEPTION           = 102;
}

// ─── POSTURE ──────────────────────────────────────────────────────────────────
// Formation posture affects movement speed, protection, and signature.
// The adjudicator applies posture modifiers to base capability values.

enum FormationPosture {
  POSTURE_UNKNOWN       = 0;
  POSTURE_TRAVELING     = 1;  // Column, max speed, low protection
  POSTURE_BOUNDING      = 2;  // Tactical movement, moderate speed, moderate protection
  POSTURE_DELIBERATE    = 3;  // Slow, high protection, highest firepower
  POSTURE_HASTY_DEFENSE = 4;  // Prepared positions, high protection
  POSTURE_DUG_IN        = 5;  // Fortified, maximum protection, limited mobility
  POSTURE_DISPERSED     = 6;  // Wide spacing, low signature, reduced C2
  POSTURE_ASSAULT       = 7;  // Attack formation, high firepower, lower protection
}
```

---

## `capabilities.proto`

What a unit can do at **full operational strength**. These are baseline values from capability templates. The adjudicator multiplies by the unit's `OperationalStatus.combat_effectiveness` to get actual performance.

```protobuf
syntax = "proto3";
package engine;
option go_package = "github.com/yourorg/aressim/internal/gen/engine";

// Full capability profile for a unit type at 100% strength.
// Templates for each UnitType are seeded from real-world references.
message Capabilities {
  MobilityCapabilities   mobility    = 1;
  FirepowerCapabilities  firepower   = 2;
  ProtectionCapabilities protection  = 3;
  SensorCapabilities     sensors     = 4;
  LogisticsProfile       logistics   = 5;
  CommandCapabilities    command     = 6;  // Only relevant for FUNCTION_COMMAND
  EWCapabilities         ew          = 7;  // Only relevant for FUNCTION_ELECTRONIC_WARFARE
}

// ─── MOBILITY ─────────────────────────────────────────────────────────────────
// Movement rates vary by terrain type. All values in meters per second.
// These are sustained rates under tactical conditions, not peak speeds.
// Reference: FM 3-21.10, FM 3-20.97, and comparable doctrine for modeled forces.

message MobilityCapabilities {
  float road_speed         = 1;  // m/s on improved road
  float cross_country_speed= 2;  // m/s on open terrain
  float urban_speed        = 3;  // m/s in built-up areas
  float forest_speed       = 4;  // m/s in forested terrain (0 = impassable)
  float mountain_speed     = 5;  // m/s on steep terrain  (0 = impassable)
  float river_crossing     = 6;  // m/s in water (0 = requires bridge/ferry)
  float max_altitude       = 7;  // Meters MSL ceiling (air units)
  float climb_rate         = 8;  // m/s vertical rate of climb (air units)
  MobilityClass mobility_class = 9;
}

enum MobilityClass {
  MOBILITY_UNKNOWN  = 0;
  MOBILITY_FOOT     = 1;  // Dismounted infantry
  MOBILITY_WHEELED  = 2;  // Truck, MRAP, wheeled IFV
  MOBILITY_TRACKED  = 3;  // Tank, tracked IFV, SP artillery
  MOBILITY_ROTARY   = 4;  // Helicopter
  MOBILITY_FIXED    = 5;  // Fixed-wing aircraft
  MOBILITY_SURFACE  = 6;  // Naval surface
  MOBILITY_SUBMARINE= 7;
}

// ─── FIREPOWER ────────────────────────────────────────────────────────────────
// A unit may have multiple weapon systems. Each WeaponSystem represents
// one class of weapon (e.g., a tank battalion has a main gun AND a coax MG).
// The adjudicator selects the appropriate system based on target type and range.

message FirepowerCapabilities {
  repeated WeaponSystem weapon_systems = 1;
}

message WeaponSystem {
  string name               = 1;  // e.g. "120mm Smoothbore", "AGM-114 Hellfire"
  WeaponClass weapon_class  = 2;
  float effective_range     = 3;  // Meters to effective engagement range
  float max_range           = 4;  // Meters to maximum range (degraded accuracy)
  float rate_of_fire        = 5;  // Rounds (or salvos) per minute
  float anti_armor_value    = 6;  // Penetration index vs armored targets (0-100)
  float anti_personnel_value= 7;  // Lethality index vs dismounted infantry (0-100)
  float anti_air_value      = 8;  // Effectiveness vs air targets (0-100)
  float area_effect_radius  = 9;  // Meters of splash/blast radius
  float accuracy_modifier   = 10; // Multiplier applied to base hit probability (0.0-2.0)
  AmmunitionType ammo_type  = 11;
  float rounds_per_basic_load = 12; // Phase 3: ties to logistics consumption
  bool  guided              = 13; // Guided munitions ignore most weather degradation
  float min_range           = 14; // Meters (mortars, some missiles have minimum arming)
}

enum WeaponClass {
  WEAPON_CLASS_UNKNOWN          = 0;
  WEAPON_CLASS_DIRECT_FIRE_KE   = 1;  // Kinetic energy: tank guns, autocannons
  WEAPON_CLASS_DIRECT_FIRE_HEAT = 2;  // Chemical energy: ATGM, RPG, HEAT rounds
  WEAPON_CLASS_DIRECT_FIRE_SMALL= 3;  // Small arms, HMG
  WEAPON_CLASS_INDIRECT_HE      = 4;  // Artillery HE
  WEAPON_CLASS_INDIRECT_CLUSTER = 5;
  WEAPON_CLASS_INDIRECT_PRECISION=6;  // Excalibur, Krasnopol
  WEAPON_CLASS_SURFACE_TO_AIR   = 7;  // SAM systems
  WEAPON_CLASS_AIR_TO_AIR       = 8;  // AAM
  WEAPON_CLASS_AIR_TO_SURFACE   = 9;  // AGM, bombs
  WEAPON_CLASS_CRUISE_MISSILE   = 10;
  WEAPON_CLASS_BALLISTIC_MISSILE= 11;
  WEAPON_CLASS_ROCKET_UNGUIDED  = 12;  // Grad, FROG-7
  WEAPON_CLASS_TORPEDO          = 13;
  WEAPON_CLASS_MINE             = 14;
}

enum AmmunitionType {
  AMMO_UNKNOWN      = 0;
  AMMO_SABOT        = 1;   // APFSDS kinetic penetrator
  AMMO_HEAT         = 2;
  AMMO_HE           = 3;
  AMMO_SMOKE        = 4;
  AMMO_ILLUMINATION = 5;
  AMMO_GUIDED_ATGM  = 6;
  AMMO_AAM_IR       = 7;
  AMMO_AAM_RADAR    = 8;
  AMMO_GUIDED_BOMB  = 9;
  AMMO_UNGUIDED_BOMB= 10;
  AMMO_ROCKET       = 11;
  AMMO_TORPEDO      = 12;
  AMMO_CRUISE       = 13;
}

// ─── PROTECTION ───────────────────────────────────────────────────────────────

message ProtectionCapabilities {
  float armor_rating         = 1;  // Kinetic protection index (0-100). Reduces KE damage.
  float chemical_protection  = 2;  // CBRN protection level (0-1)
  float air_defense_rating   = 3;  // Organic AD protection (reduces air attack effectiveness)
  bool  active_protection    = 4;  // APS: intercepts HEAT/ATGM (if true, adds deflection roll)
  float signature_radar      = 5;  // Radar cross section index: higher = easier to detect
  float signature_thermal    = 6;  // Thermal signature: higher = easier IR detection
  float signature_acoustic   = 7;  // Acoustic: relevant for submarine detection
  bool  can_fortify          = 8;  // Whether unit can improve protection via engineer work
  float fortify_bonus        = 9;  // Protection multiplier when fully fortified
}

// ─── SENSORS ──────────────────────────────────────────────────────────────────

message SensorCapabilities {
  repeated SensorSystem sensors = 1;
}

message SensorSystem {
  string      name            = 1;  // e.g. "AN/TPY-2 Radar", "FLIR"
  SensorType  sensor_type     = 2;
  float       detection_range = 3;  // Meters: range at which a standard target is detected
  float       id_range        = 4;  // Meters: range at which target type is identified
  float       tracking_range  = 5;  // Meters: range at which target can be continuously tracked
  bool        all_weather     = 6;  // If false, range degrades in precipitation/fog
  bool        night_capable   = 7;
  float       arc_degrees     = 8;  // 360 = omnidirectional; <360 = directional (e.g. nose radar)
}

enum SensorType {
  SENSOR_UNKNOWN        = 0;
  SENSOR_OPTICAL        = 1;  // Visual: weather/night dependent
  SENSOR_THERMAL_IR     = 2;  // FLIR/TIS: night capable, degraded in rain
  SENSOR_RADAR_GROUND   = 3;  // Ground surveillance radar (GSRS)
  SENSOR_RADAR_AIR      = 4;  // Air search radar
  SENSOR_SIGINT         = 5;  // Detects emissions, not physical presence
  SENSOR_ACOUSTIC       = 6;  // Sonar, acoustic detection
  SENSOR_HUMAN          = 7;  // HUMINT: patrol/observation post
  SENSOR_SAR            = 8;  // Synthetic aperture radar
}

// ─── COMMAND CAPABILITY ───────────────────────────────────────────────────────
// Relevant only for FUNCTION_COMMAND units. Defines C2 radius and subordinate capacity.

message CommandCapabilities {
  float command_radius_meters  = 1;  // Range within which subordinates receive C2 bonus
  int32 max_subordinates       = 2;  // Span of control
  float c2_effectiveness       = 3;  // 0.0-1.0: baseline C2 quality of this HQ
  bool  has_fire_support_coord = 4;  // Can coordinate indirect fire missions
}

// ─── ELECTRONIC WARFARE ───────────────────────────────────────────────────────

message EWCapabilities {
  float jamming_radius_meters  = 1;  // Radius within which enemy comms/sensors are degraded
  float jamming_effectiveness  = 2;  // 0.0-1.0: how much enemy sensor range is reduced
  float sigint_range_meters    = 3;  // Range to intercept enemy emissions
  bool  can_spoof              = 4;  // Can generate false radar contacts
}

// ─── LOGISTICS PROFILE ────────────────────────────────────────────────────────
// How much a unit consumes when operating. Connects unit to the supply chain.
// Phase 3: consumption rates multiply against commodity stock levels.

message LogisticsProfile {
  float fuel_capacity_liters   = 1;
  float fuel_consumption_idle  = 2;  // Liters/hour when stationary
  float fuel_consumption_move  = 3;  // Liters/hour when moving at road speed
  float fuel_consumption_combat= 4;  // Liters/hour during active engagement
  float ammo_basic_loads       = 5;  // Number of basic loads carried organically
  float resupply_time_hours    = 6;  // Hours to complete a full resupply from adjacent node
  float maintenance_rate       = 7;  // Equipment breakdowns per 24h of operations (Phase 3)
  string primary_commodity     = 8;  // Phase 3: "fuel_diesel", "fuel_jet", "artillery_155mm"
}
```

---

## `status.proto`

The current operational state of a unit. This degrades from the capability baseline as the simulation runs.

```protobuf
syntax = "proto3";
package engine;
option go_package = "github.com/yourorg/aressim/internal/gen/engine";

message OperationalStatus {
  // ─── Strength ───────────────────────────────────────────────────────────────
  // Personnel and equipment readiness expressed as a fraction of table of
  // organization and equipment (TO&E). 1.0 = fully manned and equipped.
  float personnel_strength  = 1;  // 0.0-1.0
  float equipment_strength  = 2;  // 0.0-1.0

  // Derived composite used by adjudicator as a single combat power scalar.
  // Computed field: min(personnel_strength, equipment_strength) × morale × c2_quality
  float combat_effectiveness = 3;  // 0.0-1.0 (read-only, computed by adjudicator)

  // ─── Combat Effects ──────────────────────────────────────────────────────────
  // These are the OUTPUTS of the combat adjudicator, not HP/health floats.
  // Multiple effects can be simultaneously active.
  CombatEffects effects = 4;

  // ─── Logistics State ─────────────────────────────────────────────────────────
  float fuel_level          = 5;  // Liters remaining
  AmmunitionState ammo      = 6;
  float supply_days         = 7;  // Days of supply on hand (Phase 3: from commodity stocks)

  // ─── Readiness ───────────────────────────────────────────────────────────────
  ReadinessState readiness  = 8;
  float          morale     = 9;   // 0.0-1.0: degraded by attrition, improved by success
  float          fatigue    = 10;  // 0.0-1.0: increases with operations tempo, resets with rest

  // ─── C2 State ────────────────────────────────────────────────────────────────
  C2Status c2 = 11;

  // ─── Logistics Connectivity (Phase 2) ────────────────────────────────────────
  LogisticsConnectivity logistics_connectivity = 12;

  // ─── Detection State (Phase 2) ───────────────────────────────────────────────
  DetectionStatus detection = 13;

  bool is_active = 14;  // False = destroyed/captured/withdrawn from sim
}

// ─── COMBAT EFFECTS ───────────────────────────────────────────────────────────
// Effects-based representation of battle damage. The adjudicator assesses these
// each tick; effects have durations and recovery conditions.
//
// Design principle: a unit under fire is suppressed first, disrupted second,
// and attrited last. This reflects real-world observations that fire suppresses
// more than it kills, and disruption of cohesion often defeats units before
// casualties alone would.

message CombatEffects {
  // SUPPRESSION: Unit cannot initiate offensive action.
  // Caused by: incoming direct/indirect fire in vicinity.
  // Recovery: no fire received for suppression_recovery_ticks.
  bool  suppressed                = 1;
  int32 suppression_ticks_remaining = 2;

  // DISRUPTION: C2 links severed, orders cannot be passed efficiently.
  // Caused by: loss of HQ unit, comms jamming, rapid attrition above threshold.
  // Effect: unit reverts to last order, cannot receive new orders.
  bool  disrupted                 = 3;
  int32 disruption_ticks_remaining = 4;

  // EXPLOITATION VULNERABILITY: Unit is broken and can be pursued.
  // Caused by: combat_effectiveness < 0.2 with active enemy pressure.
  bool  routing                   = 5;

  // EXHAUSTION: Fuel/ammo critical, operations must cease.
  // Caused by: supply consumption without resupply.
  bool  exhausted                 = 6;  // Fuel/ammo below 10%

  // DEGRADED MOBILITY: Vehicle mobility kills, bridge destruction, minefields.
  bool  mobility_kill             = 7;
  float mobility_reduction        = 8;  // 0.0-1.0 fraction of normal speed

  // OBSCURED: Smoke, weather, or EW degrading own sensors.
  bool  obscured                  = 9;
  float sensor_range_reduction    = 10;  // 0.0-1.0 fraction of normal sensor range
}

// ─── AMMUNITION STATE ─────────────────────────────────────────────────────────

message AmmunitionState {
  float primary_rounds_pct    = 1;  // Main weapon rounds as % of basic load
  float secondary_rounds_pct  = 2;  // Secondary weapon system
  float missile_count_pct     = 3;  // Guided missiles
  float mortar_rounds_pct     = 4;
  // Phase 3: Each ammo type maps to a commodity_id for supply chain calculation
}

// ─── READINESS ────────────────────────────────────────────────────────────────

enum ReadinessState {
  READINESS_UNKNOWN       = 0;
  READINESS_DEPLOYED      = 1;  // In position, fully operational
  READINESS_ALERT         = 2;  // Heightened readiness, weapons free
  READINESS_MOVING        = 3;  // In transit
  READINESS_RESTING       = 4;  // Rest/recovery: restores fatigue, cannot engage
  READINESS_REFITTING     = 5;  // Resupply/maintenance: cannot engage
  READINESS_RESERVE       = 6;  // Not committed, awaiting orders
  READINESS_DEGRADED      = 7;  // Operational but below 50% effectiveness
  READINESS_COMBAT_INEFF  = 8;  // Below 25% effectiveness, requires reconstitution
}

// ─── C2 STATUS ────────────────────────────────────────────────────────────────

message C2Status {
  bool  in_command_radius     = 1;  // Within parent HQ command radius
  bool  comms_intact          = 2;  // False if jammed or HQ destroyed
  float c2_quality            = 3;  // 0.0-1.0: effective C2 this tick
  // Degraded C2 reduces: order execution speed, fires coordination, morale recovery
}

// ─── LOGISTICS CONNECTIVITY (Phase 2) ─────────────────────────────────────────

message LogisticsConnectivity {
  bool   is_supplied              = 1;
  string nearest_node_id          = 2;
  float  supply_line_length_meters= 3;
  bool   supply_line_interdicted  = 4;
  float  effectiveness_modifier   = 5;  // 0.0-1.0: combat power reduction when unsupplied
  // Phase 3: days_of_supply computed from commodity stocks at nearest node
}

// ─── DETECTION STATUS (Phase 2) ───────────────────────────────────────────────

message DetectionStatus {
  bool            detected_by_enemy   = 1;
  repeated string detecting_unit_ids  = 2;
  DetectionQuality detection_quality  = 3;
}

enum DetectionQuality {
  DETECTION_NONE       = 0;  // Undetected (FOW)
  DETECTION_SUSPECTED  = 1;  // Something detected, type unknown
  DETECTION_DETECTED   = 2;  // Presence confirmed, type uncertain
  DETECTION_IDENTIFIED = 3;  // Type and approximate strength known
  DETECTION_TRACKED    = 4;  // Continuously tracked, weapons can be cueed
}
```

---

## `orders.proto`

The task queue that drives unit behavior. Orders model doctrine-aligned tactical tasks at the operational level.

```protobuf
syntax = "proto3";
package engine;
option go_package = "github.com/yourorg/aressim/internal/gen/engine";

import "common.proto";

message Order {
  string     order_id  = 1;
  string     issued_by = 2;  // Unit ID of issuing HQ (enforces C2 chain)
  OrderType  type      = 3;
  int32      priority  = 4;  // 0 = highest
  OrderState state     = 5;
  SimTime    issued_at = 6;
  SimTime    execute_at= 7;  // Delayed execution: 0 = immediate

  oneof parameters {
    MoveParams         move         = 10;
    AttackParams       attack       = 11;
    DefendParams       defend       = 12;
    PatrolParams       patrol       = 13;
    HoldParams         hold         = 14;
    WithdrawParams     withdraw     = 15;
    FireMissionParams  fire_mission = 16;  // Indirect fire coordination
    AirStrikeParams    air_strike   = 17;
    ResupplyParams     resupply     = 18;
    ReconParams        recon        = 19;
    AttachParams       attach       = 20;  // C2: attach to another HQ
  }
}

enum OrderType {
  ORDER_UNKNOWN        = 0;
  // ── Maneuver ──
  ORDER_MOVE           = 1;   // Move to position
  ORDER_ATTACK         = 2;   // Close with and destroy enemy
  ORDER_DEFEND         = 3;   // Occupy and hold ground
  ORDER_PATROL         = 4;   // Cyclic movement for recon/security
  ORDER_HOLD           = 5;   // Stay in place for duration
  ORDER_WITHDRAW       = 6;   // Displace rearward
  ORDER_SCREEN         = 7;   // Security mission: observe and report, delay
  ORDER_GUARD          = 8;   // Protect main body flank/rear
  ORDER_COVER          = 9;   // Delay enemy while protecting main force
  ORDER_EXPLOIT        = 10;  // Pursue broken enemy to prevent reconstitution
  // ── Fires ──
  ORDER_FIRE_MISSION   = 11;  // Indirect fire: suppress, neutralize, destroy
  ORDER_AIR_STRIKE     = 12;  // CAS coordination
  ORDER_COUNTER_BATTERY= 13;  // Target enemy indirect fire systems
  ORDER_SMOKE          = 14;  // Obscuration fires
  // ── Logistics ──
  ORDER_RESUPPLY       = 15;  // Move to logistics node for resupply
  ORDER_REARM          = 16;
  ORDER_REFUEL         = 17;
  ORDER_EVACUATE       = 18;  // CASEVAC / equipment recovery
  // ── C2 ──
  ORDER_ATTACH         = 19;  // OPCON transfer
  ORDER_ESTABLISH_CP   = 20;  // Set up command post at location
  // ── Intel ──
  ORDER_RECON          = 21;  // Reconnaissance of a specific area
  ORDER_OBSERVE        = 22;  // Establish observation post
}

enum OrderState {
  ORDER_STATE_PENDING    = 0;  // Queued, not yet active
  ORDER_STATE_ACTIVE     = 1;  // Currently executing
  ORDER_STATE_COMPLETE   = 2;
  ORDER_STATE_FAILED     = 3;  // Could not complete (e.g. unit destroyed en route)
  ORDER_STATE_CANCELLED  = 4;
}

// ─── MOVE ─────────────────────────────────────────────────────────────────────
message MoveParams {
  Position    destination     = 1;
  repeated Position waypoints = 2;  // Optional intermediate points
  MoveType    move_type       = 3;
  float       speed_override  = 4;  // 0 = use unit's standard tactical speed for move_type
}

enum MoveType {
  MOVE_TYPE_UNKNOWN         = 0;
  MOVE_TYPE_TACTICAL        = 1;  // Bounding overwatch, security maintained
  MOVE_TYPE_ROAD_MARCH      = 2;  // Column, maximum speed, reduced security
  MOVE_TYPE_INFILTRATION    = 3;  // Small elements, avoid detection
  MOVE_TYPE_AIR_ASSAULT     = 4;  // Helicopter insertion
  MOVE_TYPE_ADMINISTRATIVE  = 5;  // Logistics movement, not tactically significant
}

// ─── ATTACK ───────────────────────────────────────────────────────────────────
message AttackParams {
  string   target_unit_id         = 1;  // Specific unit target (optional)
  Polygon  attack_objective       = 2;  // Area objective (optional)
  AttackType attack_type          = 3;
  Position  axis_of_advance       = 4;  // Direction of attack
  bool      fire_and_movement     = 5;  // True = use fire+maneuver; False = direct assault
  string    supporting_fires_unit = 6;  // Unit ID assigned to provide supporting fire
}

enum AttackType {
  ATTACK_TYPE_UNKNOWN     = 0;
  ATTACK_TYPE_HASTY       = 1;  // Immediate, minimum planning
  ATTACK_TYPE_DELIBERATE  = 2;  // Planned, coordinated fires
  ATTACK_TYPE_SPOILING    = 3;  // Disrupt enemy assembly
  ATTACK_TYPE_COUNTERATTACK = 4;
  ATTACK_TYPE_RAID        = 5;  // Strike and withdraw
  ATTACK_TYPE_FEINT       = 6;  // Deception: no intent to close
  ATTACK_TYPE_DEMONSTRATION = 7;
}

// ─── DEFEND ───────────────────────────────────────────────────────────────────
message DefendParams {
  Position  position          = 1;
  Polygon   defensive_area    = 2;
  DefenseType defense_type    = 3;
  bool      prepare_positions = 4;  // If true, unit begins fortifying (improves protection over time)
  float     preparation_time_hours = 5;
}

enum DefenseType {
  DEFENSE_TYPE_UNKNOWN   = 0;
  DEFENSE_TYPE_AREA      = 1;  // Hold terrain
  DEFENSE_TYPE_MOBILE    = 2;  // Counterattack from prepared positions
  DEFENSE_TYPE_PERIMETER = 3;  // 360-degree defense
  DEFENSE_TYPE_REVERSE_SLOPE = 4;
}

// ─── PATROL ───────────────────────────────────────────────────────────────────
message PatrolParams {
  repeated Position waypoints = 1;
  bool              loop      = 2;
  PatrolType        type      = 3;
}

enum PatrolType {
  PATROL_TYPE_UNKNOWN   = 0;
  PATROL_TYPE_ROUTE     = 1;  // Security: check MSRs and terrain
  PATROL_TYPE_AREA      = 2;  // Saturation patrol of a zone
  PATROL_TYPE_COMBAT    = 3;  // Seek and engage
  PATROL_TYPE_RECON     = 4;  // Observe and report
}

// ─── HOLD ─────────────────────────────────────────────────────────────────────
message HoldParams {
  float duration_seconds = 1;  // 0 = indefinite (until superseded)
}

// ─── WITHDRAW ─────────────────────────────────────────────────────────────────
message WithdrawParams {
  Position destination   = 1;
  bool     under_pressure = 2;  // True = enemy contact, uses covered withdrawal route
  string   covering_unit_id = 3;  // Unit providing covering fire during withdrawal
}

// ─── FIRE MISSION ─────────────────────────────────────────────────────────────
message FireMissionParams {
  Position    target_position    = 1;
  string      target_unit_id     = 2;   // Optional: if known
  FireMissionType mission_type   = 3;
  AmmunitionType  ammo_requested = 4;
  int32           rounds         = 5;   // Rounds to fire (0 = at commander's discretion)
  float           time_on_target = 6;   // Sim seconds; 0 = fire for effect immediately
  string          observer_unit_id = 7; // Unit providing spot/adjust
}

enum FireMissionType {
  FIRE_MISSION_UNKNOWN    = 0;
  FIRE_MISSION_SUPPRESS   = 1;
  FIRE_MISSION_NEUTRALIZE = 2;
  FIRE_MISSION_DESTROY    = 3;
  FIRE_MISSION_DISRUPT    = 4;
  FIRE_MISSION_INTERDICT  = 5;  // Area denial: ongoing fires to prevent movement
  FIRE_MISSION_COUNTER_BATTERY = 6;
  FIRE_MISSION_SMOKE      = 7;
  FIRE_MISSION_ILLUMINATION= 8;
}

// ─── AIR STRIKE ───────────────────────────────────────────────────────────────
message AirStrikeParams {
  Position        target_position = 1;
  string          target_unit_id  = 2;
  string          aircraft_unit_id= 3;  // Assigned aviation unit
  MunitionType    munition        = 4;
  int32           sorties         = 5;
}

enum MunitionType {
  MUNITION_UNKNOWN      = 0;
  MUNITION_UNGUIDED_BOMB= 1;
  MUNITION_GUIDED_BOMB  = 2;
  MUNITION_CLUSTER_BOMB = 3;
  MUNITION_AGM          = 4;
  MUNITION_ROCKET_POD   = 5;
  MUNITION_CANNON_STRAFE= 6;
  MUNITION_CRUISE_MISSILE = 7;
}

// ─── RESUPPLY ─────────────────────────────────────────────────────────────────
message ResupplyParams {
  string logistics_node_id = 1;
  bool   priority_fuel     = 2;
  bool   priority_ammo     = 3;
  bool   priority_maintenance = 4;
}

// ─── RECON ────────────────────────────────────────────────────────────────────
message ReconParams {
  Polygon  recon_area    = 1;
  ReconType type         = 2;
  bool     report_only   = 3;  // False = engage if contacted; True = avoid contact
}

enum ReconType {
  RECON_TYPE_UNKNOWN     = 0;
  RECON_TYPE_ZONE        = 1;
  RECON_TYPE_ROUTE       = 2;
  RECON_TYPE_AREA        = 3;
  RECON_TYPE_FORCE       = 4;  // Determine enemy strength by probing
}

// ─── ATTACH (C2 TRANSFER) ─────────────────────────────────────────────────────
message AttachParams {
  string new_parent_unit_id = 1;
  bool   opcon             = 2;  // False = organic attachment; True = OPCON only
}
```

---

## `combat.proto`

The adjudicator's inputs and outputs for engagement resolution. This is separate from orders because combat resolution is computed, not commanded.

```protobuf
syntax = "proto3";
package engine;
option go_package = "github.com/yourorg/aressim/internal/gen/engine";

import "common.proto";

// Input to the adjudicator's combat resolution function for a single engagement.
message EngagementInput {
  string attacker_id          = 1;
  string defender_id          = 2;
  float  range_meters         = 3;
  WeaponClass weapon_used     = 4;
  bool   attacker_moving      = 5;   // Degrades accuracy
  bool   defender_moving      = 6;   // Affects protection
  TerrainType defender_terrain= 7;   // Terrain protection modifier
  bool   has_observer         = 8;   // For indirect fire: spotted vs unobserved
  WeatherConditions weather   = 9;
  float  time_of_day_factor   = 10;  // 1.0 = day; <1.0 = night penalty (if not NVG-equipped)
}

// Output of combat resolution.
message EngagementResult {
  string engagement_id        = 1;
  string attacker_id          = 2;
  string defender_id          = 3;
  SimTime resolved_at         = 4;

  // Effects applied to defender
  float  attrition_dealt      = 5;   // Fraction of defender strength destroyed (0.0-1.0)
  bool   suppression_applied  = 6;
  bool   disruption_applied   = 7;
  bool   mobility_kill        = 8;
  float  ammo_consumed_attacker = 9; // Fraction of attacker's basic load expended

  // Narrative for event log (displayed in COP)
  string narrative            = 10;  // e.g. "3-69 AR engages T-90 company, 2 tanks destroyed"
}

// Terrain type at a given position. Used as modifier input in combat resolution.
// The adjudicator looks up terrain from the map layer at unit position.
enum TerrainType {
  TERRAIN_UNKNOWN   = 0;
  TERRAIN_OPEN      = 1;   // Fields, steppe, desert — no protection bonus
  TERRAIN_URBAN     = 2;   // Cities: high protection for defenders, slows attackers
  TERRAIN_FOREST    = 3;   // Concealment, moderate protection, slows vehicles
  TERRAIN_HILLS     = 4;   // Observation advantage, hull-down positions
  TERRAIN_MOUNTAINS = 5;   // Strong defense, nearly impassable to vehicles
  TERRAIN_WETLANDS  = 6;   // Canalized movement, poor for armor
  TERRAIN_COAST     = 7;
  TERRAIN_RIVERLINE = 8;   // Major obstacle
}

// Environmental conditions affecting sensor and weapon performance.
message WeatherConditions {
  WeatherState state         = 1;
  float visibility_km        = 2;  // Optical visibility in kilometers
  float wind_speed_mps       = 3;  // Affects smoke, aircraft stability
  bool  precipitation        = 4;  // Degrades optical sensors, impacts aviation
  float temperature_celsius  = 5;  // Affects engine performance at extremes
}

enum WeatherState {
  WEATHER_CLEAR     = 0;
  WEATHER_OVERCAST  = 1;
  WEATHER_RAIN      = 2;
  WEATHER_SNOW      = 3;
  WEATHER_FOG       = 4;   // Severe optical sensor degradation
  WEATHER_DUST_STORM= 5;   // Severe: radar and optical both degraded
}

// Combat power ratio used by adjudicator for historical/doctrinal reference.
// The adjudicator uses this for high-level validation but does not expose it to frontend.
message CombatPowerRatio {
  string attacker_id          = 1;
  string defender_id          = 2;
  float  attacker_combat_power= 3;
  float  defender_combat_power= 4;
  float  ratio                = 5;  // attacker/defender
  // Historical reference: 3:1 ratio commonly cited for deliberate attack success
}
```

---

## `logistics.proto`

The supply chain model. Phase 1 scaffolding; Phase 2 full implementation.

```protobuf
syntax = "proto3";
package engine;
option go_package = "github.com/yourorg/aressim/internal/gen/engine";

import "common.proto";

// A logistics node is the physical anchor of the supply chain.
// Types range from strategic depots down to forward supply points.
message LogisticsNode {
  string              id                = 1;
  string              name              = 2;
  string              side              = 3;
  string              nation_id         = 4;  // Phase 3
  LogisticsNodeType   type              = 5;
  Position            position          = 6;
  LogisticsCapacity   capacity          = 7;
  LogisticsCapacity   current_stock     = 8;   // Actual on-hand quantities
  float               replenishment_rate= 9;   // Supply units per sim-hour from higher
  string              parent_node_id    = 10;  // Supply chain hierarchy
  bool                is_active         = 11;
}

enum LogisticsNodeType {
  LOGISTICS_UNKNOWN        = 0;
  LOGISTICS_STRATEGIC_DEPOT= 1;  // Corps-level: receives from production
  LOGISTICS_BRIGADE_BSA    = 2;  // Brigade Support Area
  LOGISTICS_BATTALION_LOG  = 3;  // Battalion trains
  LOGISTICS_FORWARD_SUPPLY = 4;  // Forward supply point
  LOGISTICS_AIRFIELD       = 5;  // Air resupply hub (aviation fuel, ordnance)
  LOGISTICS_PORT           = 6;  // Maritime resupply hub
  LOGISTICS_MEDICAL        = 7;  // CASEVAC/treatment facility
}

// Stock quantities for each supply category.
// Phase 3: Each field maps directly to a commodity_id in strategic.proto.
message LogisticsCapacity {
  float fuel_liters          = 1;
  float ammunition_units     = 2;  // Abstracted unit; 1 unit = 1 basic load for one company
  float food_rations         = 3;
  float spare_parts          = 4;
  float medical_supplies     = 5;
}

// A directed supply link between two nodes or a node and a unit.
// Represented as a graph edge in SurrealDB.
message SupplyLink {
  string          id            = 1;
  string          from_id       = 2;  // LogisticsNode or Unit ID
  string          to_id         = 3;  // LogisticsNode or Unit ID
  SupplyLinkType  link_type     = 4;
  float           bandwidth     = 5;  // Supply units per sim-hour
  float           length_meters = 6;
  bool            is_active     = 7;
  bool            is_interdicted= 8;  // Set by combat: enemy fires/maneuver cut the route
  float           vulnerability = 9;  // 0.0-1.0: probability of interdiction per tick under threat
}

enum SupplyLinkType {
  SUPPLY_LINK_UNKNOWN  = 0;
  SUPPLY_LINK_ROAD     = 1;
  SUPPLY_LINK_RAIL     = 2;
  SUPPLY_LINK_AIR      = 3;
  SUPPLY_LINK_SEA      = 4;
  SUPPLY_LINK_PIPELINE = 5;  // Phase 3: fuel only
}
```

---

## `scenario.proto`

The container for a full simulation state, used for save/load and multiplayer sync.

```protobuf
syntax = "proto3";
package engine;
option go_package = "github.com/yourorg/aressim/internal/gen/engine";

import "common.proto";
import "unit.proto";
import "logistics.proto";

message Scenario {
  string            id           = 1;
  string            name         = 2;
  string            description  = 3;
  string            classification = 4;  // e.g. "UNCLASSIFIED // EXERCISE"
  double            start_time_unix = 5;
  SimulationSettings settings    = 6;
  MapSettings        map         = 7;
  repeated Unit           units  = 8;
  repeated LogisticsNode  nodes  = 9;
  // Phase 3:
  // repeated Nation       nations = 10;
  // repeated InfrastructureNode infrastructure = 11;
}

message SimulationSettings {
  float tick_rate_hz          = 1;  // Ticks per second (default 10)
  float time_scale            = 2;  // Sim seconds per wall second (default 1.0)
  bool  fog_of_war_enabled    = 3;
  bool  supply_chain_enabled  = 4;
  bool  weather_enabled       = 5;
  bool  morale_enabled        = 6;
  AdjudicationModel adj_model = 7;
}

enum AdjudicationModel {
  ADJ_MODEL_DEFAULT   = 0;  // Balanced fidelity/performance
  ADJ_MODEL_FAST      = 1;  // Reduced fidelity, maximum unit count
  ADJ_MODEL_HIGH_FIDELITY = 2;  // Full effects system, slower
}

message MapSettings {
  string  terrain_tileset_url = 1;  // CesiumJS terrain asset
  string  imagery_url         = 2;  // Satellite/map imagery asset
  bool    use_3d_terrain       = 3;
  WeatherConditions initial_weather = 4;
}
```

---

## `events.proto`

All event types streamed from the adjudicator to the frontend via the Wails bridge. The frontend is stateless — it rebuilds its world entirely from the initial `FullStateSnapshot` plus the subsequent event stream.

```protobuf
syntax = "proto3";
package engine;
option go_package = "github.com/yourorg/aressim/internal/gen/engine";

import "common.proto";
import "unit.proto";
import "logistics.proto";
import "combat.proto";

// The envelope for all backend→frontend events.
message SimEvent {
  string  event_id   = 1;
  SimTime sim_time   = 2;
  oneof payload {
    FullStateSnapshot   full_state      = 3;  // Sent on connect or scenario load
    BatchUnitUpdate     batch_update    = 4;  // Emitted every tick: all moved units
    UnitSpawnedEvent    unit_spawned    = 5;
    UnitDestroyedEvent  unit_destroyed  = 6;
    EngagementResult    combat          = 7;
    SupplyStatusEvent   supply_status   = 8;
    C2StatusEvent       c2_status       = 9;
    ScenarioStateEvent  scenario_state  = 10;
    NarrativeEvent      narrative       = 11;  // Human-readable event log entry
  }
}

// Sent once on scenario load or client reconnect.
// The frontend discards all prior state and rebuilds from this.
message FullStateSnapshot {
  repeated Unit           units     = 1;
  repeated LogisticsNode  nodes     = 2;
  SimTime                 sim_time  = 3;
  WeatherConditions       weather   = 4;
}

// Batch unit update emitted every tick instead of per-unit events.
// Contains only units whose position or status changed this tick.
// At scale (1000+ units) this is a single ~50KB binary message at 10Hz.
message BatchUnitUpdate {
  repeated UnitDelta deltas = 1;
}

message UnitDelta {
  string          unit_id    = 1;
  Position        position   = 2;  // New position (always present if unit moved)
  OperationalStatus status   = 3;  // New status (present only if status changed)
}

message UnitSpawnedEvent    { Unit unit = 1; }
message UnitDestroyedEvent  { string unit_id = 1; string cause = 2; Position last_position = 3; }

message SupplyStatusEvent {
  string unit_id                        = 1;
  LogisticsConnectivity new_connectivity = 2;
}

message C2StatusEvent {
  string    unit_id     = 1;
  C2Status  new_status  = 2;
}

message ScenarioStateEvent {
  ScenarioPlayState state = 1;
  float             speed = 2;
}

enum ScenarioPlayState {
  SCENARIO_PAUSED  = 0;
  SCENARIO_RUNNING = 1;
  SCENARIO_ENDED   = 2;
}

// Human-readable event for the COP event log panel.
message NarrativeEvent {
  string  text     = 1;  // e.g. "1-64 AR engages enemy armor, 3 T-90s destroyed"
  string  category = 2;  // "combat", "logistics", "c2", "intelligence"
  string  unit_id  = 3;  // Primary unit involved (for highlight on map)
}
```

---

## `strategic.proto`

Phase 3. Defines the strategic layer that all operational-level entities eventually tie into. Scaffolded now so the operational schema never has to be refactored to accommodate it.

```protobuf
syntax = "proto3";
package engine;
option go_package = "github.com/yourorg/aressim/internal/gen/engine";

import "common.proto";

// A nation or faction. The strategic actor.
message Nation {
  string   id                    = 1;
  string   name                  = 2;
  string   side                  = 3;  // Alignment to a simulation side
  float    gdp_index             = 4;  // Relative economic capacity (0.0-1.0)
  float    political_will        = 5;  // 0.0-1.0: tolerance for casualties/cost. At 0, sues for peace.
  float    mobilization_level    = 6;  // 0.0-1.0: fraction of national capacity committed
  repeated string allied_nation_ids = 7;
}

// A node in the physical infrastructure network.
// Bridges the operational and strategic layers: destroying infrastructure nodes
// degrades strategic production and operational logistics simultaneously.
message InfrastructureNode {
  string               id          = 1;
  string               nation_id   = 2;
  InfrastructureType   type        = 3;
  Position             position    = 4;
  float                health      = 5;   // 0.0-1.0: damaged by fires
  float                capacity    = 6;   // Output at 100% health
  string               commodity_output = 7;  // What this node produces
  float                repair_rate = 8;   // Health restored per sim-day if not under attack
}

enum InfrastructureType {
  INFRA_UNKNOWN       = 0;
  INFRA_FACTORY       = 1;   // Produces military equipment
  INFRA_REFINERY      = 2;   // Produces fuel
  INFRA_POWER_PLANT   = 3;   // Powers factories and cities
  INFRA_PORT          = 4;   // Import/export hub
  INFRA_RAIL_HUB      = 5;   // Logistics multiplier
  INFRA_AIRFIELD      = 6;
  INFRA_MINE          = 7;   // Raw commodity extraction
  INFRA_CITY          = 8;   // Population center: political will anchor
}

// A commodity is a strategic resource.
// Commodities flow from extraction → processing → military consumption.
message Commodity {
  string  id              = 1;  // e.g. "fuel_diesel", "steel", "rare_earth"
  string  display_name    = 2;
  float   stock           = 3;  // Current national stockpile in standard units
  float   production_rate = 4;  // Units per sim-day from all producing nodes
  float   consumption_rate= 5;  // Units per sim-day from all consuming forces
  float   import_rate     = 6;  // From allied nations (if supply routes open)
  bool    is_critical     = 7;  // If stock reaches 0, tied units become non-operational
}
```

---

## Adjudicator Integration Contract

The proto schema defines a contract the adjudicator must honor. These are the invariants the Go engine must enforce every tick:

| Rule | Schema Source |
|------|---------------|
| A unit with `status.effects.suppressed = true` cannot have an active `ORDER_ATTACK` or `ORDER_FIRE_MISSION` | `status.proto`, `orders.proto` |
| A unit with `status.effects.exhausted = true` has its `WeaponSystem.rate_of_fire` multiplied by 0.0 (cannot fire) | `status.proto`, `capabilities.proto` |
| A unit with `c2.comms_intact = false` cannot receive new orders until comms are restored | `status.proto`, `orders.proto` |
| `combat_effectiveness` is always recomputed as `min(personnel_strength, equipment_strength) × morale × c2.c2_quality` | `status.proto` |
| A `BatchUnitUpdate` is emitted every tick containing only `UnitDelta` records for units whose position or status changed | `events.proto` |
| Combat resolution inputs must include terrain type looked up at the defender's position from the map layer | `combat.proto` |
| A supply link with `is_interdicted = true` does NOT carry supply; `LogisticsConnectivity.is_supplied` is false for all units downstream | `logistics.proto`, `status.proto` |
| Unit processing order within a tick is deterministic: sorted by `unit.id` ascending | Required for multiplayer determinism |

---

## Proto Compilation

```bash
# scripts/gen_proto.sh
#!/usr/bin/env bash
set -e

# Go: outputs to internal/gen/engine/
protoc \
  --go_out=internal/gen \
  --go_opt=paths=source_relative \
  -I proto \
  proto/common.proto \
  proto/unit.proto \
  proto/capabilities.proto \
  proto/status.proto \
  proto/orders.proto \
  proto/combat.proto \
  proto/logistics.proto \
  proto/intelligence.proto \
  proto/scenario.proto \
  proto/events.proto \
  proto/strategic.proto

# TypeScript: outputs to frontend/src/proto/
# Uses @bufbuild/protoc-gen-es
buf generate
```

`buf.gen.yaml`:
```yaml
version: v1
plugins:
  - plugin: es
    out: frontend/src/proto
    opt: target=ts
```

---

## Field Numbering Conventions

- **1–9**: Core identity fields (id, name, side, type)
- **10–19**: Geospatial and positional fields
- **20–49**: Capability sub-messages
- **50–79**: Status and effects sub-messages
- **80–99**: Order parameters (`oneof` blocks)
- **100–149**: Phase 2 additions (logistics connectivity, detection)
- **150–199**: Phase 3 additions (strategic linkage, commodities)

Field numbers are **never reused**. Deleted fields are marked `reserved`. This ensures binary compatibility across scenario file versions.
