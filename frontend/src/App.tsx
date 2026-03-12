/**
 * App.tsx
 *
 * Root component. Renders the COP (Common Operating Picture) shell:
 *   - Full-viewport CesiumJS globe container (behind everything)
 *   - HUD overlay (top bar: scenario name, sim time, state controls)
 *   - Event log panel (bottom-left, collapsible)
 *   - Unit detail panel (right side, shown when a unit is selected)
 *
 * Bridge is initialized once on mount. The globe itself is mounted in
 * Phase 1 as a placeholder div; CesiumJS is wired in Phase 2.
 */

import { useEffect, useRef, useState } from "react";
import { initBridge } from "./bridge/bridge";
import { useSimStore } from "./store/simStore";
import CesiumGlobe from "./components/CesiumGlobe";
import ScenarioEditor from "./components/editor/ScenarioEditor";
import { RequestSync } from "../wailsjs/go/main/App";
import "./app.css";

// ─── SUB-COMPONENTS ──────────────────────────────────────────────────────────

function TopBar({ onOpenEditor }: { onOpenEditor: () => void }) {
  const scenarioName = useSimStore((s) => s.scenarioName);
  const scenarioState = useSimStore((s) => s.scenarioState);
  const simSeconds = useSimStore((s) => s.simSeconds);
  const timeScale = useSimStore((s) => s.timeScale);
  const tickNumber = useSimStore((s) => s.tickNumber);

  const formatSimTime = (seconds: number): string => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = Math.floor(seconds % 60);
    return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  };

  const stateColor: Record<string, string> = {
    idle: "#6b7280",
    paused: "#f59e0b",
    running: "#22c55e",
    ended: "#ef4444",
  };

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
        {scenarioState === "running" && (
          <span className="time-scale">×{timeScale.toFixed(1)}</span>
        )}
      </div>

      <div className="top-bar-right">
        <span className="tick-label">T:{tickNumber}</span>
        <button
          onClick={onOpenEditor}
          style={{
            background: "rgba(255,255,255,0.06)",
            border: "1px solid rgba(255,255,255,0.12)",
            borderRadius: 3,
            color: "#9ca3af",
            fontFamily: "Courier New, monospace",
            fontSize: 10,
            fontWeight: 700,
            letterSpacing: "0.08em",
            padding: "3px 9px",
            cursor: "pointer",
            textTransform: "uppercase",
          }}
        >
          Editor
        </button>
      </div>
    </div>
  );
}

function EventLog() {
  const eventLog = useSimStore((s) => s.eventLog);
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [eventLog.length]);

  const categoryColor: Record<string, string> = {
    combat: "#ef4444",
    logistics: "#f59e0b",
    c2: "#3b82f6",
    intelligence: "#a855f7",
    scenario: "#6b7280",
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

function UnitPanel() {
  const selectedUnitId = useSimStore((s) => s.selectedUnitId);
  const units = useSimStore((s) => s.units);
  const selectUnit = useSimStore((s) => s.selectUnit);

  if (!selectedUnitId) return null;
  const unit = units.get(selectedUnitId);
  if (!unit) return null;

  const sideColor: Record<string, string> = {
    Blue: "#3b82f6",
    Red: "#ef4444",
    Neutral: "#f59e0b",
  };

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
          <span className="stat-value">
            {Math.round(unit.status.fuelLevelLiters)}
          </span>
        </div>
        <div className="unit-stat-row">
          <span className="stat-label">Morale</span>
          <span className="stat-value">
            {Math.round(unit.status.morale * 100)}%
          </span>
        </div>
        <div className="unit-stat-row">
          <span className="stat-label">Fatigue</span>
          <span className="stat-value">
            {Math.round(unit.status.fatigue * 100)}%
          </span>
        </div>

        {(unit.status.suppressed ||
          unit.status.disrupted ||
          unit.status.routing) && (
          <div className="unit-status-flags">
            {unit.status.suppressed && (
              <span className="status-flag suppressed">SUPPRESSED</span>
            )}
            {unit.status.disrupted && (
              <span className="status-flag disrupted">DISRUPTED</span>
            )}
            {unit.status.routing && (
              <span className="status-flag routing">ROUTING</span>
            )}
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
  const [view, setView] = useState<"sim" | "editor">("sim");

  useEffect(() => {
    initBridge();
    RequestSync().catch((e) => console.warn("[App] RequestSync:", e));
    console.log("[App] AresSim frontend initialized");
  }, []);

  if (view === "editor") {
    return (
      <ScenarioEditor
        onExit={() => setView("sim")}
        onPlay={() => setView("sim")}
      />
    );
  }

  return (
    <div className="app-shell">
      <CesiumGlobe />

      {/* HUD overlay layer */}
      <div className="hud-overlay">
        <TopBar onOpenEditor={() => setView("editor")} />
        <EventLog />
        <UnitPanel />
      </div>
    </div>
  );
}
