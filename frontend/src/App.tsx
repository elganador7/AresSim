/**
 * App.tsx
 *
 * Root component. Renders the COP (Common Operating Picture) shell:
 *   - Full-viewport CesiumJS globe
 *   - HUD overlay: TopBar, ViewSwitcher, EventLog, UnitPanel
 *
 * `appView` ("sim" | "editor") controls routing between the sim globe and
 * the scenario editor. It is separate from `activeView` ("debug" | "blue" |
 * "red") which controls fog-of-war inside the sim globe.
 */

import { useEffect, useRef, useState } from "react";
import { initBridge } from "./bridge/bridge";
import { useSimStore } from "./store/simStore";
import CesiumGlobe from "./components/CesiumGlobe";
import ScenarioEditor from "./components/editor/ScenarioEditor";
import { RequestSync, PauseSim, CancelMoveOrder, SetSimSpeed } from "../wailsjs/go/main/App";
import "./app.css";

// ─── VIEW SWITCHER ────────────────────────────────────────────────────────────

function ViewSwitcher() {
  const activeView    = useSimStore((s) => s.activeView);
  const setActiveView = useSimStore((s) => s.setActiveView);

  return (
    <div className="view-switcher">
      <button
        className={`view-btn view-btn-debug ${activeView === "debug" ? "view-btn-debug-active" : ""}`}
        onClick={() => setActiveView("debug")}
        title="Show all units (game master view)"
      >
        DEBUG
      </button>
      <button
        className={`view-btn view-btn-blue ${activeView === "blue" ? "view-btn-blue-active" : ""}`}
        onClick={() => setActiveView("blue")}
        title="Blue force view — your units only"
      >
        BLUE
      </button>
      <button
        className={`view-btn view-btn-red ${activeView === "red" ? "view-btn-red-active" : ""}`}
        onClick={() => setActiveView("red")}
        title="Red force view — your units only"
      >
        RED
      </button>
    </div>
  );
}

// ─── TOP BAR ──────────────────────────────────────────────────────────────────

const SPEED_PRESETS = [0.5, 1, 2, 5, 10, 30] as const;

function TopBar({ onOpenEditor }: { onOpenEditor: () => void }) {
  const scenarioName  = useSimStore((s) => s.scenarioName);
  const scenarioState = useSimStore((s) => s.scenarioState);
  const simSeconds    = useSimStore((s) => s.simSeconds);
  const timeScale     = useSimStore((s) => s.timeScale);
  const tickNumber    = useSimStore((s) => s.tickNumber);

  const formatSimTime = (seconds: number): string => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = Math.floor(seconds % 60);
    return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  };

  const stateColor: Record<string, string> = {
    idle:    "#6b7280",
    paused:  "#f59e0b",
    running: "#22c55e",
    ended:   "#ef4444",
  };

  const handlePauseToggle = () => {
    PauseSim(scenarioState === "running").catch(console.error);
  };

  // Find nearest preset index for the current timeScale.
  const currentIdx = (() => {
    let best = 0;
    let bestDiff = Infinity;
    SPEED_PRESETS.forEach((p, i) => {
      const d = Math.abs(p - timeScale);
      if (d < bestDiff) { bestDiff = d; best = i; }
    });
    return best;
  })();

  // Speed controls are intentionally disabled while paused: changing speed
  // while the MockLoop is stopped has no effect until the loop restarts, and
  // the atomic update would be silently lost if the user forgets to resume.
  const canSlower = scenarioState === "running" && currentIdx > 0;
  const canFaster = scenarioState === "running" && currentIdx < SPEED_PRESETS.length - 1;

  const stepSpeed = (delta: -1 | 1) => {
    const next = SPEED_PRESETS[currentIdx + delta];
    if (next !== undefined) SetSimSpeed(next).catch(console.error);
  };

  const isActive = scenarioState === "running" || scenarioState === "paused";

  return (
    <div className="top-bar">
      <div className="top-bar-left">
        <span className="scenario-name">{scenarioName || "No Scenario"}</span>
        <span
          className="state-badge"
          style={{ color: stateColor[scenarioState] ?? "#6b7280" }}
        >
          {scenarioState.toUpperCase()}
        </span>
      </div>

      <div className="top-bar-center">
        <span className="sim-time">{formatSimTime(simSeconds)}</span>

        {isActive && (
          <div className="speed-control">
            <button
              className="speed-step-btn"
              onClick={() => stepSpeed(-1)}
              disabled={!canSlower}
              title="Slower"
            >◄</button>
            <span className="speed-label">
              ×{Number.isInteger(timeScale) ? timeScale : timeScale.toFixed(1)}
            </span>
            <button
              className="speed-step-btn"
              onClick={() => stepSpeed(1)}
              disabled={!canFaster}
              title="Faster"
            >►</button>
          </div>
        )}

        {isActive && (
          <button
            className="pause-btn"
            onClick={handlePauseToggle}
            title={scenarioState === "running" ? "Pause" : "Resume"}
          >
            {scenarioState === "running" ? "⏸" : "▶"}
          </button>
        )}
      </div>

      <div className="top-bar-right">
        <span className="tick-label">T:{tickNumber}</span>
        <button className="editor-btn" onClick={onOpenEditor}>
          EDITOR
        </button>
      </div>
    </div>
  );
}

// ─── EVENT LOG ────────────────────────────────────────────────────────────────

function EventLog() {
  const eventLog = useSimStore((s) => s.eventLog);
  const endRef   = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [eventLog.length]);

  const categoryColor: Record<string, string> = {
    combat:       "#ef4444",
    logistics:    "#f59e0b",
    c2:           "#3b82f6",
    intelligence: "#a855f7",
    scenario:     "#6b7280",
  };

  if (eventLog.length === 0) {
    return (
      <div className="event-log">
        <div className="event-log-header">EVENT LOG</div>
        <div className="event-log-empty">Awaiting simulation events…</div>
      </div>
    );
  }

  return (
    <div className="event-log">
      <div className="event-log-header">
        EVENT LOG <span className="event-count">({eventLog.length})</span>
      </div>
      <div className="event-log-entries">
        {eventLog.map((entry) => (
          <div key={entry.id} className="event-entry">
            <span
              className="entry-category"
              style={{ color: categoryColor[entry.category] ?? "#6b7280" }}
            >
              [{entry.category.toUpperCase()}]
            </span>{" "}
            <span className="entry-text">{entry.text}</span>
          </div>
        ))}
        <div ref={endRef} />
      </div>
    </div>
  );
}

// ─── UNIT PANEL ───────────────────────────────────────────────────────────────

function canMoveUnit(side: string, view: "debug" | "blue" | "red"): boolean {
  if (view === "debug") return true;
  return view === "blue" ? side === "Blue" : side === "Red";
}

function haversineM(lat1: number, lon1: number, lat2: number, lon2: number): number {
  const R = 6_371_000;
  const φ1 = lat1 * Math.PI / 180, φ2 = lat2 * Math.PI / 180;
  const Δφ = (lat2 - lat1) * Math.PI / 180, Δλ = (lon2 - lon1) * Math.PI / 180;
  const a = Math.sin(Δφ/2)**2 + Math.cos(φ1)*Math.cos(φ2)*Math.sin(Δλ/2)**2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

function formatDist(m: number): string {
  return m >= 1000 ? `${(m / 1000).toFixed(1)} km` : `${Math.round(m)} m`;
}

function formatETA(secs: number): string {
  if (!isFinite(secs)) return "—";
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  const s = Math.floor(secs % 60);
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

function UnitPanel() {
  const selectedUnitId = useSimStore((s) => s.selectedUnitId);
  const units          = useSimStore((s) => s.units);
  const activeView     = useSimStore((s) => s.activeView);
  const selectUnit     = useSimStore((s) => s.selectUnit);

  if (!selectedUnitId) return null;
  const unit = units.get(selectedUnitId);
  if (!unit) return null;

  const sideColor: Record<string, string> = {
    Blue:    "#3b82f6",
    Red:     "#ef4444",
    Neutral: "#f59e0b",
  };

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
          const wps = unit.moveOrder.waypoints;
          const last = wps[wps.length - 1];
          // Total remaining route distance.
          let pts = [
            { lat: unit.position.lat, lon: unit.position.lon },
            ...wps.map((w) => ({ lat: w.lat, lon: w.lon })),
          ];
          let totalM = 0;
          for (let i = 0; i < pts.length - 1; i++) {
            totalM += haversineM(pts[i].lat, pts[i].lon, pts[i+1].lat, pts[i+1].lon);
          }
          const speed = unit.position.speed; // m/s
          const etaSecs = speed > 0 ? totalM / speed : Infinity;
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
            {unit.status.disrupted  && <span className="status-flag disrupted">DISRUPTED</span>}
            {unit.status.routing    && <span className="status-flag routing">ROUTING</span>}
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
      </div>
    </div>
  );
}

// ─── ROOT COMPONENT ───────────────────────────────────────────────────────────

export default function App() {
  const [appView, setAppView] = useState<"sim" | "editor">("sim");

  useEffect(() => {
    initBridge();
    RequestSync().catch((e) => console.warn("[App] RequestSync:", e));
    console.log("[App] AresSim frontend initialized");
  }, []);

  if (appView === "editor") {
    return (
      <ScenarioEditor
        onExit={() => setAppView("sim")}
        onPlay={() => setAppView("sim")}
      />
    );
  }

  return (
    <div className="app-shell">
      <CesiumGlobe />

      <div className="hud-overlay">
        <TopBar onOpenEditor={() => setAppView("editor")} />
        <ViewSwitcher />
        <EventLog />
        <UnitPanel />
      </div>
    </div>
  );
}
