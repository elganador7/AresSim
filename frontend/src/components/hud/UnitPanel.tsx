import { useEffect, useMemo, useState } from "react";
import {
  CancelMoveOrder,
  ListUnitDefinitions,
  PreviewCurrentEngagement,
  RemoveMoveWaypoint,
  RequestSync,
  SetUnitEngagement,
  SetUnitLoadoutConfiguration,
} from "../../../wailsjs/go/main/App";
import { useSimStore, type PathViolationPreview, type Unit, type WeaponDef } from "../../store/simStore";
import { formatDist, formatETA } from "../../utils/formatters";
import { haversineM } from "../../utils/geo";
import { selectedPlayerTeam } from "../../utils/playerTeam";
import { ENGAGEMENT_BEHAVIORS } from "../../utils/editorTasking";
import { teamColorHex } from "../../utils/teamColors";

type UnitDefinitionPanelMeta = {
  domain?: number;
  targetClass?: string;
  stationary?: boolean;
  assetClass?: string;
  teamCode?: string;
  embarkedFixedWingCapacity?: number;
  embarkedRotaryWingCapacity?: number;
  embarkedUavCapacity?: number;
  launchCapacityPerInterval?: number;
  recoveryCapacityPerInterval?: number;
  defaultWeaponConfiguration?: string;
  weaponConfigurations?: Array<{
    id: string;
    name: string;
    description: string;
    loadout: Array<{ weaponId: string; maxQty: number; initialQty: number }>;
  }>;
};

const BASE_OPS_STATE_LABEL: Record<number, string> = {
  0: "Unknown",
  1: "Usable",
  2: "Degraded",
  3: "Closed",
};

const DAMAGE_STATE_LABEL: Record<number, string> = {
  0: "Unknown",
  1: "Operational",
  2: "Damaged",
  3: "Mission Killed",
  4: "Destroyed",
};

function canControlUnit(definitionId: string, explicitTeamId: string | undefined, controlTeam: string, definitionMap: Map<string, UnitDefinitionPanelMeta>): boolean {
  if (!controlTeam) return false;
  const teamCode = explicitTeamId?.trim().toUpperCase()
    || definitionMap.get(definitionId)?.teamCode?.trim().toUpperCase()
    || "";
  return teamCode === controlTeam;
}

function ensureSuccess(result: { success: boolean; error?: string }) {
  if (!result.success) {
    throw new Error(result.error || "Command failed");
  }
}

function canHostAircraft(definition: UnitDefinitionPanelMeta | undefined): boolean {
  if (!definition) return false;
  return (definition.embarkedFixedWingCapacity ?? 0) > 0
    || (definition.embarkedRotaryWingCapacity ?? 0) > 0
    || (definition.embarkedUavCapacity ?? 0) > 0
    || (definition.launchCapacityPerInterval ?? 0) > 0
    || (definition.recoveryCapacityPerInterval ?? 0) > 0;
}

function isFixedFacility(definition: UnitDefinitionPanelMeta | undefined): boolean {
  if (!definition) return false;
  return definition.assetClass === "airbase"
    || definition.assetClass === "port"
    || definition.assetClass === "c2_site"
    || definition.assetClass === "radar_site"
    || definition.assetClass === "oil_field"
    || definition.assetClass === "pipeline_node"
    || definition.assetClass === "desalination_plant"
    || definition.assetClass === "power_plant";
}

function normalizeDefinitionId(definitionId: string | undefined): string {
  const raw = String(definitionId ?? "").trim();
  const idx = raw.lastIndexOf(":");
  return idx >= 0 ? raw.slice(idx + 1) : raw;
}

export default function UnitPanel() {
  const selectedUnitId = useSimStore((s) => s.selectedUnitId);
  const units = useSimStore((s) => s.units);
  const weaponDefs = useSimStore((s) => s.weaponDefs);
  const humanControlledTeam = useSimStore((s) => s.humanControlledTeam);
  const selectUnit = useSimStore((s) => s.selectUnit);
  const routePreview = useSimStore((s) => s.selectedRoutePreview);
  const strikePreview = useSimStore((s) => s.selectedStrikePreview);
  const setRoutePreview = useSimStore((s) => s.setSelectedRoutePreview);
  const setStrikePreview = useSimStore((s) => s.setSelectedStrikePreview);
  const mapCommandMode = useSimStore((s) => s.mapCommandMode);
  const startRouteEdit = useSimStore((s) => s.startRouteEdit);
  const clearMapCommandMode = useSimStore((s) => s.clearMapCommandMode);
  const [definitionMap, setDefinitionMap] = useState<Map<string, UnitDefinitionPanelMeta>>(new Map());

  const unit = selectedUnitId ? units.get(selectedUnitId) : undefined;
  const playerTeam = selectedPlayerTeam(humanControlledTeam);
  const effectiveTeamId = unit ? ((unit.operatorTeamId ?? unit.teamId) ?? "").trim().toUpperCase() : "";
  const ownedByPlayer = unit ? effectiveTeamId === playerTeam : false;
  const controllable = unit
    ? canControlUnit(unit.definitionId, unit.operatorTeamId ?? unit.teamId, playerTeam, definitionMap)
    : false;
  const teamColor = teamColorHex(unit?.operatorTeamId ?? unit?.teamId);
  const strength = unit ? Math.round(unit.status.combatEffectiveness * 100) : 0;
  const routeModeActive = mapCommandMode.type === "route" && mapCommandMode.unitId === unit?.id;
  const [selectedLoadoutId, setSelectedLoadoutId] = useState(unit?.loadoutConfigurationId ?? "");
  const routeWarning = routePreview?.blocked ? (routePreview.reason ?? "Transit blocked.") : null;
  const strikeWarning = strikePreview?.blocked ? (strikePreview.reason ?? "Strike blocked.") : null;
  const definition = unit ? definitionMap.get(normalizeDefinitionId(unit.definitionId)) : undefined;
  const canHost = canHostAircraft(definition);
  const isFacility = isFixedFacility(definition);
  const hostedUnits = useMemo(() => {
    if (!unit) {
      return [];
    }
    return Array.from(units.values()).filter((candidate) => candidate.hostBaseId === unit.id);
  }, [unit, units]);

  const [engagementBehavior, setEngagementBehavior] = useState(unit?.engagementBehavior ?? 1);
  const [engagementPkillThreshold, setEngagementPkillThreshold] = useState(unit?.engagementPkillThreshold ?? 0.5);
  const [busy, setBusy] = useState(false);
  const [engagementPreview, setEngagementPreview] = useState<null | {
    readyToFire: boolean;
    canAssign: boolean;
    weaponId?: string;
    reason?: string;
    reasonCode?: string;
    rangeToTargetM?: number;
    weaponRangeM?: number;
    fireProbability?: number;
    desiredEffectSupport: boolean;
    inStrikeCooldown: boolean;
  }>(null);

  useEffect(() => {
    let cancelled = false;
    ListUnitDefinitions()
      .then((rows) => {
        if (cancelled) {
          return;
        }
        const defs = new Map<string, UnitDefinitionPanelMeta>();
        for (const row of rows) {
          const id = typeof row.id === "string" ? row.id : "";
          if (!id) {
            continue;
          }
          const normalizedId = normalizeDefinitionId(id);
          const meta = {
            domain: Number(row.domain ?? 0),
            targetClass: typeof row.target_class === "string" ? row.target_class : "",
            stationary: Boolean(row.stationary),
            assetClass: typeof row.asset_class === "string" ? row.asset_class : "combat_unit",
            embarkedFixedWingCapacity: Number(row.embarked_fixed_wing_capacity ?? 0),
            embarkedRotaryWingCapacity: Number(row.embarked_rotary_wing_capacity ?? 0),
            embarkedUavCapacity: Number(row.embarked_uav_capacity ?? 0),
            launchCapacityPerInterval: Number(row.launch_capacity_per_interval ?? 0),
            recoveryCapacityPerInterval: Number(row.recovery_capacity_per_interval ?? 0),
            teamCode: Array.isArray(row.employed_by) && row.employed_by.length > 0
              ? String(row.employed_by[0]).trim().toUpperCase()
              : String(row.nation_of_origin ?? "").trim().toUpperCase(),
            defaultWeaponConfiguration: typeof row.default_weapon_configuration === "string" ? row.default_weapon_configuration : "",
            weaponConfigurations: Array.isArray(row.weapon_configurations)
              ? row.weapon_configurations.map((config: any) => ({
                  id: String(config.id ?? ""),
                  name: String(config.name ?? ""),
                  description: String(config.description ?? ""),
                  loadout: Array.isArray(config.loadout)
                    ? config.loadout.map((slot: any) => ({
                        weaponId: String(slot.weapon_id ?? ""),
                        maxQty: Number(slot.max_qty ?? 0),
                        initialQty: Number(slot.initial_qty ?? 0),
                      }))
                    : [],
                })).filter((config) => config.id)
              : [],
          };
          defs.set(id, meta);
          defs.set(normalizedId, meta);
        }
        setDefinitionMap(defs);
      })
      .catch(console.error);
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (unit && playerTeam && !ownedByPlayer) {
      clearMapCommandMode();
      selectUnit(null);
    }
  }, [clearMapCommandMode, ownedByPlayer, playerTeam, selectUnit, unit]);

  useEffect(() => {
    if (!unit) {
      return;
    }
    setEngagementBehavior(unit.engagementBehavior ?? 1);
    setEngagementPkillThreshold(unit.engagementPkillThreshold ?? 0.5);
    setSelectedLoadoutId(unit.loadoutConfigurationId ?? "");
  }, [unit?.id]);

  useEffect(() => {
    let cancelled = false;
    if (!unit?.id || !unit.moveOrder || unit.moveOrder.waypoints.length === 0) {
      setRoutePreview(null);
      return;
    }
    ((window as any).go?.main?.App?.PreviewCurrentTransitPath?.(unit.id) as Promise<PathViolationPreview | null> | undefined)
      ?.then((preview) => {
        if (cancelled) {
          return;
        }
        setRoutePreview(preview ?? null);
      })
      .catch((error) => {
        if (!cancelled) {
          console.error(error);
          setRoutePreview(null);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [setRoutePreview, unit?.id, unit?.moveOrder]);

  useEffect(() => {
    let cancelled = false;
    if (!unit?.id || !unit.attackOrder?.targetUnitId) {
      setEngagementPreview(null);
      return;
    }
    PreviewCurrentEngagement(unit.id)
      .then((preview) => {
        if (cancelled) {
          return;
        }
        setEngagementPreview(preview ?? null);
      })
      .catch((error) => {
        if (!cancelled) {
          console.error(error);
          setEngagementPreview(null);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [unit?.id, unit?.attackOrder]);

  useEffect(() => {
    let cancelled = false;
    if (!unit?.id || !unit.attackOrder?.targetUnitId) {
      setStrikePreview(null);
      return;
    }
    ((window as any).go?.main?.App?.PreviewCurrentStrikePath?.(unit.id) as Promise<PathViolationPreview | null> | undefined)
      ?.then((preview) => {
        if (cancelled) {
          return;
        }
        setStrikePreview(preview ?? null);
      })
      .catch((error) => {
        if (!cancelled) {
          console.error(error);
          setStrikePreview(null);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [setStrikePreview, unit?.id, unit?.attackOrder, unit?.moveOrder]);

  const saveCommands = async () => {
    if (!unit) return;
    setBusy(true);
    try {
      if (selectedLoadoutId !== (unit.loadoutConfigurationId ?? "")) {
        ensureSuccess(await SetUnitLoadoutConfiguration(unit.id, selectedLoadoutId));
      }
      ensureSuccess(await SetUnitEngagement(unit.id, engagementBehavior, engagementPkillThreshold));
      ensureSuccess(await RequestSync());
    } catch (error) {
      console.error(error);
      alert(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy(false);
    }
  };

  const loadoutOptions = definition?.weaponConfigurations ?? [];
  const canSelectLoadout = controllable
    && !isFacility
    && Boolean(unit?.hostBaseId)
    && (unit?.position.altMsl ?? 0) <= 0
    && loadoutOptions.length > 0;
  const removeWaypoint = async (index: number) => {
    if (!unit) return;
    setBusy(true);
    try {
      ensureSuccess(await RemoveMoveWaypoint(unit.id, index));
      ensureSuccess(await RequestSync());
    } catch (error) {
      console.error(error);
      alert(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy(false);
    }
  };

  if (!unit || (playerTeam && !ownedByPlayer)) return null;

  return (
    <div className="unit-panel">
      <div className="unit-panel-header">
        <span
          className="unit-side-indicator"
          style={{ background: teamColor }}
        />
        <span className="unit-display-name">{unit.displayName}</span>
        <button
          className="unit-panel-close"
          onClick={() => {
            clearMapCommandMode();
            selectUnit(null);
          }}
          aria-label="Close unit panel"
        >
          ×
        </button>
      </div>

      <div className="unit-panel-body">
        <div className="unit-full-name">{unit.fullName}</div>
        {controllable && routeWarning && (
          <div className="path-warning-note">
            {routeWarning}
          </div>
        )}
        {controllable && strikeWarning && (
          <div className="path-warning-note strike-warning-note">
            {strikeWarning}
          </div>
        )}
        {!isFacility && unit.moveOrder && unit.moveOrder.waypoints.length > 0 ? (() => {
          const waypoints = unit.moveOrder.waypoints;
          const last = waypoints[waypoints.length - 1];
          const points = [
            { lat: unit.position.lat, lon: unit.position.lon },
            ...waypoints.map((waypoint) => ({ lat: waypoint.lat, lon: waypoint.lon })),
          ];
          let totalM = 0;
          for (let i = 0; i < points.length - 1; i++) {
            totalM += haversineM(points[i].lat, points[i].lon, points[i + 1].lat, points[i + 1].lon);
          }
          const etaSecs = unit.position.speed > 0 ? totalM / unit.position.speed : Infinity;

          return (
            <div className="move-order-info">
              <div className="move-order-row">
                <span className="stat-label">Destination</span>
                <span className="stat-value">
                  {last.lat.toFixed(4)}°, {last.lon.toFixed(4)}°
                </span>
              </div>
              <div className="move-order-row">
                <span className="stat-label">Waypoints</span>
                <span className="stat-value">{waypoints.length}</span>
              </div>
              <div className="move-order-row">
                <span className="stat-label">Distance</span>
                <span className="stat-value">{formatDist(totalM)}</span>
              </div>
              <div className="move-order-row">
                <span className="stat-label">ETA</span>
                <span className="stat-value">{formatETA(etaSecs)}</span>
              </div>
              {controllable && (
                <div className="waypoint-list">
                  {waypoints.map((wp, idx) => (
                    <div key={`${wp.lat}-${wp.lon}-${idx}`} className="waypoint-row">
                      <span className="waypoint-label">WP {idx + 1}</span>
                      <span className="waypoint-value">{wp.lat.toFixed(3)}°, {wp.lon.toFixed(3)}°</span>
                      <button
                        className="waypoint-remove-btn"
                        disabled={busy}
                        onClick={() => removeWaypoint(idx).catch(console.error)}
                      >
                        ×
                      </button>
                    </div>
                  ))}
                </div>
              )}
              {controllable && (
                <div className="unit-command-buttons">
                  <button
                    className={`cancel-order-btn${routeModeActive ? " route-edit-active" : ""}`}
                    onClick={() => (routeModeActive ? clearMapCommandMode() : startRouteEdit(unit.id))}
                  >
                    {routeModeActive ? "Finish Route" : "Append Waypoints"}
                  </button>
                  {routeModeActive && (
                    <button
                      className="cancel-order-btn"
                      onClick={() => clearMapCommandMode()}
                    >
                      Cancel Route Mode
                    </button>
                  )}
                  <button
                    className="cancel-order-btn"
                    onClick={() => CancelMoveOrder(unit.id).catch(console.error)}
                  >
                    Cancel Order
                  </button>
                </div>
              )}
            </div>
          );
        })() : !isFacility ? (
          <div className={`move-hint ${controllable ? "" : "move-hint-locked"}`}>
            {controllable
              ? routeModeActive
                ? "Route mode active — click the map to append waypoints"
                : "Click map to move, or use Append Waypoints to build a route"
              : "Enemy unit — read only"}
          </div>
        ) : (
          <div className="facility-summary-card">
            <div className="facility-summary-row">
              <span className="stat-label">Asset Type</span>
              <span className="stat-value">{definition?.assetClass?.replaceAll("_", " ") ?? "facility"}</span>
            </div>
            <div className="facility-summary-row">
              <span className="stat-label">Damage</span>
              <span className="stat-value">{DAMAGE_STATE_LABEL[unit.damageState] ?? "Unknown"}</span>
            </div>
            <div className="facility-summary-row">
              <span className="stat-label">Operations</span>
              <span className="stat-value">{BASE_OPS_STATE_LABEL[unit.baseOps?.state ?? 0]}</span>
            </div>
            <div className="facility-summary-copy">
              Fixed installation. Base operations and hosted-unit readiness matter here, not movement or onboard fuel.
            </div>
          </div>
        )}

        {controllable && !isFacility && (
          <div className="unit-command-section">
            <div className="unit-command-header">Commands</div>
            <div className="unit-command-row">
              <label className="stat-label">Loadout</label>
              <select
                className="unit-panel-select"
                value={selectedLoadoutId}
                disabled={!canSelectLoadout}
                onChange={(e) => setSelectedLoadoutId(e.target.value)}
              >
                <option value="">Current / Default</option>
                {loadoutOptions.map((option) => (
                  <option key={option.id} value={option.id}>{option.name || option.id}</option>
                ))}
              </select>
            </div>
            {canSelectLoadout && (
              <div className="move-hint">
                Loadout changes are only allowed while grounded at host base.
              </div>
            )}
            <div className="unit-command-row">
              <label className="stat-label">Engagement</label>
              <select
                className="unit-panel-select"
                value={engagementBehavior}
                onChange={(e) => setEngagementBehavior(Number(e.target.value))}
              >
                {ENGAGEMENT_BEHAVIORS.map((option) => (
                  <option key={option.value} value={option.value}>{option.label}</option>
                ))}
              </select>
            </div>
            <div className="unit-command-row">
              <label className="stat-label">Auto Pkill</label>
              <input
                className="unit-panel-input"
                type="number"
                min={0.1}
                max={0.99}
                step={0.05}
                value={engagementPkillThreshold}
                onChange={(e) => setEngagementPkillThreshold(Number(e.target.value))}
              />
            </div>
            {unit.attackOrder?.targetUnitId && (
              <>
                <div className="unit-command-row">
                  <label className="stat-label">Assigned Target</label>
                  <span className="stat-value">
                    {units.get(unit.attackOrder.targetUnitId)?.displayName ?? unit.attackOrder.targetUnitId}
                  </span>
                </div>
                {engagementPreview && (
                  <div className="move-hint">
                    Engagement: {engagementPreview.reason || "No preview."}
                    {engagementPreview.weaponId ? ` Weapon ${engagementPreview.weaponId}.` : ""}
                    {engagementPreview.rangeToTargetM && engagementPreview.weaponRangeM
                      ? ` Range ${Math.round(engagementPreview.rangeToTargetM / 1000)} / ${Math.round(engagementPreview.weaponRangeM / 1000)} km.`
                      : ""}
                    {engagementPreview.fireProbability
                      ? ` P-hit ${Math.round(engagementPreview.fireProbability * 100)}%.`
                      : ""}
                    {!engagementPreview.readyToFire && engagementPreview.canAssign
                      ? " Unit can still be assigned and will move into launch position."
                      : ""}
                  </div>
                )}
              </>
            )}
            <div className="unit-command-buttons">
              <button className="cancel-order-btn" disabled={busy} onClick={() => saveCommands().catch(console.error)}>
                Apply Commands
              </button>
              <button
                className={`cancel-order-btn${routeModeActive ? " route-edit-active" : ""}`}
                disabled={busy}
                onClick={() => (routeModeActive ? clearMapCommandMode() : startRouteEdit(unit.id))}
              >
                {routeModeActive ? "Finish Route" : "Route Mode"}
              </button>
            </div>
          </div>
        )}

        <div className="unit-stat-row">
          <span className="stat-label">Country</span>
          <span className="stat-value" style={{ color: teamColor }}>
            {unit.operatorTeamId || unit.teamId || "UNK"}
          </span>
        </div>
        {unit.coalitionId && (
          <div className="unit-stat-row">
            <span className="stat-label">Coalition</span>
            <span className="stat-value">{unit.coalitionId}</span>
          </div>
        )}
        {!isFacility ? (
          <>
            <div className="unit-stat-row">
              <span className="stat-label">Effectiveness</span>
              <span className="stat-value">{strength}%</span>
            </div>
            <div className="unit-stat-row">
              <span className="stat-label">Personnel</span>
              <span className="stat-value">{unit.status.personnelStrength}</span>
            </div>
            <div className="unit-stat-row">
              <span className="stat-label">Equipment</span>
              <span className="stat-value">{unit.status.equipmentStrength}</span>
            </div>
            <div className="unit-stat-row">
              <span className="stat-label">Fuel (L)</span>
              <span className="stat-value">{Math.round(unit.status.fuelLevelLiters)}</span>
            </div>
            <div className="unit-stat-row">
              <span className="stat-label">Morale</span>
              <span className="stat-value">{Math.round(unit.status.morale * 100)}%</span>
            </div>
            <div className="unit-stat-row">
              <span className="stat-label">Fatigue</span>
              <span className="stat-value">{Math.round(unit.status.fatigue * 100)}%</span>
            </div>
          </>
        ) : (
          <>
            <div className="unit-stat-row">
              <span className="stat-label">Hosted Units</span>
              <span className="stat-value">{hostedUnits.length}</span>
            </div>
            {hostedUnits.length > 0 && (
              <div className="facility-hosted-list">
                {hostedUnits.map((hosted) => (
                  <button
                    key={hosted.id}
                    className="facility-hosted-row"
                    onClick={() => selectUnit(hosted.id)}
                  >
                    <span>{hosted.displayName}</span>
                    <span>{hosted.damageState === 4 ? "Destroyed" : hosted.nextSortieReadySeconds && hosted.nextSortieReadySeconds > 0 ? "Delayed" : "Ready"}</span>
                  </button>
                ))}
              </div>
            )}
          </>
        )}
        {!isFacility && canHost && (
          <>
            <div className="unit-stat-row">
              <span className="stat-label">Embarked Units</span>
              <span className="stat-value">{hostedUnits.length}</span>
            </div>
            {hostedUnits.length > 0 && (
              <div className="facility-hosted-list">
                {hostedUnits.map((hosted) => (
                  <button
                    key={hosted.id}
                    className="facility-hosted-row"
                    onClick={() => selectUnit(hosted.id)}
                  >
                    <span>{hosted.displayName}</span>
                    <span>{hosted.damageState === 4 ? "Destroyed" : hosted.nextSortieReadySeconds && hosted.nextSortieReadySeconds > 0 ? "Delayed" : "Ready"}</span>
                  </button>
                ))}
              </div>
            )}
          </>
        )}

        <div className="unit-position">
          <span className="stat-label">Position</span>
          <span className="stat-value position-value">
            {unit.position.lat.toFixed(4)}°,{" "}
            {unit.position.lon.toFixed(4)}°
            <br />
            {Math.round(unit.position.altMsl)}m MSL
          </span>
        </div>

        {unit.weapons.length > 0 && (
          <div className="weapon-list">
            <div className="weapon-list-header">Loadout</div>
            {unit.weapons.map((weapon) => {
              const def = weaponDefs.get(weapon.weaponId);
              const pct = weapon.maxQty > 0 ? weapon.currentQty / weapon.maxQty : 0;
              return (
                <div key={weapon.weaponId} className="weapon-row">
                  <span className="weapon-name">{def?.name ?? weapon.weaponId}</span>
                  <span className="weapon-qty">
                    {weapon.currentQty}
                    <span className="weapon-qty-max">/{weapon.maxQty}</span>
                  </span>
                  <div className="weapon-bar-track">
                    <div
                      className="weapon-bar-fill"
                      style={{ width: `${Math.round(pct * 100)}%` }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
