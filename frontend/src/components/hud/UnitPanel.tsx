import { CancelMoveOrder } from "../../../wailsjs/go/main/App";
import { useSimStore } from "../../store/simStore";
import { formatDist, formatETA } from "../../utils/formatters";
import { haversineM } from "../../utils/geo";

const sideColor: Record<string, string> = {
  Blue: "#3b82f6",
  Red: "#ef4444",
  Neutral: "#f59e0b",
};

function canMoveUnit(side: string, view: "debug" | "blue" | "red"): boolean {
  if (view === "debug") return true;
  return view === "blue" ? side === "Blue" : side === "Red";
}

export default function UnitPanel() {
  const selectedUnitId = useSimStore((s) => s.selectedUnitId);
  const units = useSimStore((s) => s.units);
  const weaponDefs = useSimStore((s) => s.weaponDefs);
  const activeView = useSimStore((s) => s.activeView);
  const selectUnit = useSimStore((s) => s.selectUnit);

  if (!selectedUnitId) return null;
  const unit = units.get(selectedUnitId);
  if (!unit) return null;

  const moveable = canMoveUnit(unit.side, activeView);
  const strength = Math.round(unit.status.combatEffectiveness * 100);

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
          onClick={() => selectUnit(null)}
          aria-label="Close unit panel"
        >
          ×
        </button>
      </div>

      <div className="unit-panel-body">
        <div className="unit-full-name">{unit.fullName}</div>

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
                <span className="stat-label">Distance</span>
                <span className="stat-value">{formatDist(totalM)}</span>
              </div>
              <div className="move-order-row">
                <span className="stat-label">ETA</span>
                <span className="stat-value">{formatETA(etaSecs)}</span>
              </div>
              {moveable && (
                <button
                  className="cancel-order-btn"
                  onClick={() => CancelMoveOrder(unit.id).catch(console.error)}
                >
                  Cancel Order
                </button>
              )}
            </div>
          );
        })() : (
          <div className={`move-hint ${moveable ? "" : "move-hint-locked"}`}>
            {moveable ? "↖ Click map to reposition" : "Enemy unit — read only"}
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

        {(unit.status.suppressed || unit.status.disrupted || unit.status.routing) && (
          <div className="unit-status-flags">
            {unit.status.suppressed && <span className="status-flag suppressed">SUPPRESSED</span>}
            {unit.status.disrupted && <span className="status-flag disrupted">DISRUPTED</span>}
            {unit.status.routing && <span className="status-flag routing">ROUTING</span>}
          </div>
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
