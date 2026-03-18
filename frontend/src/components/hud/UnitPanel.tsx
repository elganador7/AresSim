import { useEffect, useMemo, useState } from "react";
import {
  CancelMoveOrder,
  ListUnitDefinitions,
  RemoveMoveWaypoint,
  RequestSync,
  SetUnitAttackOrder,
  SetUnitEngagement,
} from "../../../wailsjs/go/main/App";
import { useSimStore, type PathViolationPreview, type WeaponDef } from "../../store/simStore";
import { formatDist, formatETA } from "../../utils/formatters";
import { haversineM } from "../../utils/geo";
import { type UnitDefinitionTargetLite } from "../../utils/loadoutValidation";
import { inferUnitTeamCode } from "../../utils/unitTeams";
import { ATTACK_ORDER_TYPES, DESIRED_EFFECTS, ENGAGEMENT_BEHAVIORS, filterValidLiveTargets } from "../../utils/tasking";

type UnitDefinitionPanelMeta = UnitDefinitionTargetLite & { teamCode?: string };

const sideColor: Record<string, string> = {
  Blue: "#3b82f6",
  Red: "#ef4444",
  Neutral: "#f59e0b",
};

function canControlUnit(unitId: string, side: string, definitionId: string, explicitTeamId: string | undefined, view: string, definitionMap: Map<string, UnitDefinitionPanelMeta>): boolean {
  if (view === "debug") return true;
  const teamCode = explicitTeamId?.trim().toUpperCase()
    || inferUnitTeamCode(unitId, side, { teamCode: definitionMap.get(definitionId)?.teamCode ?? "" });
  return teamCode === view;
}

function ensureSuccess(result: { success: boolean; error?: string }) {
  if (!result.success) {
    throw new Error(result.error || "Command failed");
  }
}

export default function UnitPanel() {
  const selectedUnitId = useSimStore((s) => s.selectedUnitId);
  const units = useSimStore((s) => s.units);
  const weaponDefs = useSimStore((s) => s.weaponDefs);
  const activeView = useSimStore((s) => s.activeView);
  const selectUnit = useSimStore((s) => s.selectUnit);
  const routePreview = useSimStore((s) => s.selectedRoutePreview);
  const strikePreview = useSimStore((s) => s.selectedStrikePreview);
  const setRoutePreview = useSimStore((s) => s.setSelectedRoutePreview);
  const setStrikePreview = useSimStore((s) => s.setSelectedStrikePreview);
  const mapCommandMode = useSimStore((s) => s.mapCommandMode);
  const startRouteEdit = useSimStore((s) => s.startRouteEdit);
  const startTargetPick = useSimStore((s) => s.startTargetPick);
  const clearMapCommandMode = useSimStore((s) => s.clearMapCommandMode);
  const detectionContacts = useSimStore((s) => s.detectionContacts);
  const [definitionMap, setDefinitionMap] = useState<Map<string, UnitDefinitionPanelMeta>>(new Map());

  const unit = selectedUnitId ? units.get(selectedUnitId) : undefined;
  const controllable = unit
    ? canControlUnit(unit.id, unit.side, unit.definitionId, unit.teamId, activeView, definitionMap)
    : false;
  const contactMeta = unit && activeView !== "debug"
    ? detectionContacts.get(activeView)?.get(unit.id)
    : undefined;
  const strength = unit ? Math.round(unit.status.combatEffectiveness * 100) : 0;
  const routeModeActive = mapCommandMode.type === "route" && mapCommandMode.unitId === unit?.id;
  const targetPickActive = mapCommandMode.type === "target_pick" && mapCommandMode.unitId === unit?.id;
  const validTargets = useMemo(() => {
    if (!unit) {
      return [];
    }
    return filterValidLiveTargets(unit, units, weaponDefs as Map<string, WeaponDef>, definitionMap);
  }, [definitionMap, unit, units, weaponDefs]);
  const routeWarning = routePreview?.blocked ? (routePreview.reason ?? "Transit blocked.") : null;
  const strikeWarning = strikePreview?.blocked ? (strikePreview.reason ?? "Strike blocked.") : null;

  const [engagementBehavior, setEngagementBehavior] = useState(unit?.engagementBehavior ?? 1);
  const [engagementPkillThreshold, setEngagementPkillThreshold] = useState(unit?.engagementPkillThreshold ?? 0.5);
  const [attackOrderType, setAttackOrderType] = useState(unit?.attackOrder?.orderType ?? 0);
  const [targetUnitId, setTargetUnitId] = useState(unit?.attackOrder?.targetUnitId ?? "");
  const [desiredEffect, setDesiredEffect] = useState(unit?.attackOrder?.desiredEffect ?? 3);
  const [pkillThreshold, setPkillThreshold] = useState(unit?.attackOrder?.pkillThreshold ?? 0.7);
  const [busy, setBusy] = useState(false);

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
          defs.set(id, {
            domain: Number(row.domain ?? 0),
            targetClass: typeof row.target_class === "string" ? row.target_class : "soft_infrastructure",
            stationary: Boolean(row.stationary),
            assetClass: typeof row.asset_class === "string" ? row.asset_class : "combat_unit",
            teamCode: Array.isArray(row.employed_by) && row.employed_by.length > 0
              ? String(row.employed_by[0]).trim().toUpperCase()
              : String(row.nation_of_origin ?? "").trim().toUpperCase(),
          });
        }
        setDefinitionMap(defs);
      })
      .catch(console.error);
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!unit) {
      return;
    }
    setEngagementBehavior(unit.engagementBehavior ?? 1);
    setEngagementPkillThreshold(unit.engagementPkillThreshold ?? 0.5);
    setAttackOrderType(unit.attackOrder?.orderType ?? 0);
    setTargetUnitId(unit.attackOrder?.targetUnitId ?? "");
    setDesiredEffect(unit.attackOrder?.desiredEffect ?? 3);
    setPkillThreshold(unit.attackOrder?.pkillThreshold ?? 0.7);
  }, [unit?.id]);

  useEffect(() => {
    const onTargetPicked = (event: Event) => {
      const custom = event as CustomEvent<{ shooterId: string; targetUnitId: string }>;
      if (!unit || custom.detail?.shooterId !== unit.id) {
        return;
      }
      setTargetUnitId(custom.detail.targetUnitId);
      setAttackOrderType((current) => (current === 0 ? 1 : current));
      clearMapCommandMode();
    };
    window.addEventListener("sim:target-picked", onTargetPicked);
    return () => window.removeEventListener("sim:target-picked", onTargetPicked);
  }, [clearMapCommandMode, unit]);

  useEffect(() => {
    if (!targetUnitId) {
      return;
    }
    if (!validTargets.some((candidate) => candidate.id === targetUnitId)) {
      setTargetUnitId("");
    }
  }, [targetUnitId, validTargets]);

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
      ensureSuccess(await SetUnitEngagement(unit.id, engagementBehavior, engagementPkillThreshold));
      ensureSuccess(await SetUnitAttackOrder(unit.id, attackOrderType, targetUnitId, desiredEffect, pkillThreshold));
      ensureSuccess(await RequestSync());
    } catch (error) {
      console.error(error);
      alert(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy(false);
    }
  };

  const clearAttackTask = async () => {
    if (!unit) return;
    setBusy(true);
    try {
      ensureSuccess(await SetUnitAttackOrder(unit.id, 0, "", desiredEffect, pkillThreshold));
      ensureSuccess(await RequestSync());
      setAttackOrderType(0);
      setTargetUnitId("");
    } catch (error) {
      console.error(error);
      alert(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy(false);
    }
  };

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

  if (!unit) return null;

  return (
    <div className="unit-panel">
      <div className="unit-panel-header">
        <span
          className="unit-side-indicator"
          style={{ background: sideColor[unit.side] ?? "#6b7280" }}
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
        {!controllable && contactMeta && (
          <div className="track-source-note">
            {contactMeta.shared ? `Shared track from ${contactMeta.sourceTeam}` : "Locally detected track"}
          </div>
        )}
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

        {unit.moveOrder && unit.moveOrder.waypoints.length > 0 ? (() => {
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
        })() : (
          <div className={`move-hint ${controllable ? "" : "move-hint-locked"}`}>
            {controllable
              ? routeModeActive
                ? "Route mode active — click the map to append waypoints"
                : "Click map to move, or use Append Waypoints to build a route"
              : "Enemy unit — read only"}
          </div>
        )}

        {controllable && (
          <div className="unit-command-section">
            <div className="unit-command-header">Commands</div>
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
            <div className="unit-command-row">
              <label className="stat-label">Attack Task</label>
              <select
                className="unit-panel-select"
                value={attackOrderType}
                onChange={(e) => setAttackOrderType(Number(e.target.value))}
              >
                {ATTACK_ORDER_TYPES.map((option) => (
                  <option key={option.value} value={option.value}>{option.label}</option>
                ))}
              </select>
            </div>
            {attackOrderType !== 0 && (
              <>
                <div className="unit-command-row">
                  <label className="stat-label">Target</label>
                  <select
                    className="unit-panel-select"
                    value={targetUnitId}
                    onChange={(e) => setTargetUnitId(e.target.value)}
                  >
                    <option value="">Select target…</option>
                    {validTargets.map((candidate) => (
                      <option key={candidate.id} value={candidate.id}>
                        {candidate.displayName} · {candidate.side}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="unit-command-buttons">
                  <button
                    className={`cancel-order-btn${targetPickActive ? " route-edit-active" : ""}`}
                    onClick={() => (targetPickActive ? clearMapCommandMode() : startTargetPick(unit.id))}
                  >
                    {targetPickActive ? "Pick Enemy On Map" : "Pick Target On Map"}
                  </button>
                  {targetPickActive && (
                    <button
                      className="cancel-order-btn"
                      onClick={() => clearMapCommandMode()}
                    >
                      Cancel Target Pick
                    </button>
                  )}
                </div>
                <div className="unit-command-row">
                  <label className="stat-label">Effect</label>
                  <select
                    className="unit-panel-select"
                    value={desiredEffect}
                    onChange={(e) => setDesiredEffect(Number(e.target.value))}
                  >
                    {DESIRED_EFFECTS.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </div>
                <div className="unit-command-row">
                  <label className="stat-label">Order Pkill</label>
                  <input
                    className="unit-panel-input"
                    type="number"
                    min={0.1}
                    max={0.99}
                    step={0.05}
                    value={pkillThreshold}
                    onChange={(e) => setPkillThreshold(Number(e.target.value))}
                  />
                </div>
              </>
            )}
            <div className="unit-command-buttons">
              <button className="cancel-order-btn" disabled={busy} onClick={() => saveCommands().catch(console.error)}>
                Apply Commands
              </button>
              <button className="cancel-order-btn" disabled={busy} onClick={() => clearAttackTask().catch(console.error)}>
                Clear Task
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
          <span className="stat-label">Side</span>
          <span className="stat-value" style={{ color: sideColor[unit.side] }}>
            {unit.side}
          </span>
        </div>
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
