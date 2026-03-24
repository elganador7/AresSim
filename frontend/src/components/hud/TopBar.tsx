import { useEffect, useMemo, useRef, useState } from "react";
import { PauseSim, RequestSync, SetHumanControlledTeam, SetSimSpeed } from "../../../wailsjs/go/main/App";
import { useSimStore } from "../../store/simStore";
import { formatSimTime } from "../../utils/formatters";
import { selectedPlayerTeam } from "../../utils/playerTeam";
import ObjectivePanel from "./ObjectivePanel";
import RelationshipPanel from "./RelationshipPanel";
import ViewSwitcher from "./ViewSwitcher";

const SPEED_PRESETS = [0.5, 1, 2, 5, 10, 30] as const;

const stateColor: Record<string, string> = {
  idle: "#6b7280",
  paused: "#f59e0b",
  running: "#22c55e",
  ended: "#ef4444",
};

function formatUsdCompact(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value);
}

export default function TopBar({
  onOpenEditor,
  onOpenScenario,
  debugViewMenuVisible,
}: {
  onOpenEditor: () => void;
  onOpenScenario: () => void;
  debugViewMenuVisible: boolean;
}) {
  const scenarioName = useSimStore((s) => s.scenarioName);
  const scenarioState = useSimStore((s) => s.scenarioState);
  const simSeconds = useSimStore((s) => s.simSeconds);
  const timeScale = useSimStore((s) => s.timeScale);
  const tickNumber = useSimStore((s) => s.tickNumber);
  const scores = useSimStore((s) => s.scores);
  const units = useSimStore((s) => s.units);
  const humanControlledTeam = useSimStore((s) => s.humanControlledTeam);
  const setActiveView = useSimStore((s) => s.setActiveView);
  const setHumanControlledTeam = useSimStore((s) => s.setHumanControlledTeam);
  const [sharingOpen, setSharingOpen] = useState(false);
  const [briefingOpen, setBriefingOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const controllableTeams = useMemo(() => {
    const present = new Set<string>();
    units.forEach((unit) => {
      const code = (unit.operatorTeamId ?? unit.teamId)?.trim().toUpperCase() ?? "";
      if (code === "USA" || code === "ISR" || code === "IRN") {
        present.add(code);
      }
    });
    return Array.from(present).sort();
  }, [units]);

  const currentIdx = (() => {
    let best = 0;
    let bestDiff = Infinity;
    SPEED_PRESETS.forEach((preset, index) => {
      const diff = Math.abs(preset - timeScale);
      if (diff < bestDiff) {
        bestDiff = diff;
        best = index;
      }
    });
    return best;
  })();

  const canSlower = scenarioState === "running" && currentIdx > 0;
  const canFaster = scenarioState === "running" && currentIdx < SPEED_PRESETS.length - 1;
  const isActive = scenarioState === "running" || scenarioState === "paused";

  const handlePauseToggle = () => {
    PauseSim(scenarioState === "running").catch(console.error);
  };

  const stepSpeed = (delta: -1 | 1) => {
    const next = SPEED_PRESETS[currentIdx + delta];
    if (next !== undefined) SetSimSpeed(next).catch(console.error);
  };

  const handleHumanTeamChange = (teamId: string) => {
    const normalizedTeam = selectedPlayerTeam(teamId);
    const previousHumanTeam = humanControlledTeam;
    setHumanControlledTeam(normalizedTeam);
    setActiveView(normalizedTeam || "debug");
    SetHumanControlledTeam(teamId)
      .then((result) => {
        if (!result.success) {
          console.error(result.error || "failed to set human-controlled team");
          setHumanControlledTeam(previousHumanTeam);
          setActiveView(selectedPlayerTeam(previousHumanTeam) || "debug");
          return;
        }
        RequestSync().catch(console.error);
      })
      .catch(console.error);
  };

  const toggleMenuAction = (action: () => void) => {
    setMenuOpen(false);
    action();
  };

  useEffect(() => {
    if (!menuOpen) {
      return;
    }
    const handlePointerDown = (event: MouseEvent) => {
      if (!menuRef.current?.contains(event.target as Node)) {
        setMenuOpen(false);
      }
    };
    window.addEventListener("mousedown", handlePointerDown);
    return () => window.removeEventListener("mousedown", handlePointerDown);
  }, [menuOpen]);

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
        {scores.length > 0 && (
          <div className="score-strip">
            {scores.slice(0, 4).map((score) => (
              <div className="score-chip" key={score.teamId} title={`Human ${formatUsdCompact(score.humanLossUsd)} · Replacement ${formatUsdCompact(score.replacementLossUsd)} · Strategic ${formatUsdCompact(score.strategicLossUsd)} · Economic ${formatUsdCompact(score.economicLossUsd)}`}>
                <span className="score-team">{score.teamId}</span>
                <span className="score-value">{formatUsdCompact(score.totalLossUsd)}</span>
              </div>
            ))}
          </div>
        )}
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
        {controllableTeams.length > 0 && (
          <label className="top-bar-select">
            <span>PLAYER</span>
            <select
              value={humanControlledTeam}
              onChange={(e) => handleHumanTeamChange(e.target.value)}
            >
              <option value="">NONE</option>
              {controllableTeams.map((team) => (
                <option key={team} value={team}>
                  {team}
                </option>
              ))}
            </select>
          </label>
        )}
        {debugViewMenuVisible && <ViewSwitcher />}
        <div className="top-bar-menu-wrap" ref={menuRef}>
          <button className="editor-btn top-bar-menu-btn" onClick={() => setMenuOpen((current) => !current)}>
            MENU
          </button>
          {menuOpen && (
            <div className="top-bar-menu">
              <button
                className={`top-bar-menu-item${briefingOpen ? " active" : ""}`}
                onClick={() => toggleMenuAction(() => setBriefingOpen((current) => !current))}
              >
                {briefingOpen ? "Hide Briefing" : "Show Briefing"}
              </button>
              <button
                className={`top-bar-menu-item${sharingOpen ? " active" : ""}`}
                onClick={() => toggleMenuAction(() => setSharingOpen((current) => !current))}
              >
                {sharingOpen ? "Hide Relationships" : "Show Relationships"}
              </button>
              <button
                className="top-bar-menu-item"
                onClick={() => toggleMenuAction(onOpenScenario)}
              >
                Open Scenario
              </button>
              <button
                className="top-bar-menu-item"
                onClick={() => toggleMenuAction(onOpenEditor)}
              >
                Open Editor
              </button>
            </div>
          )}
        </div>
      </div>
      <ObjectivePanel open={briefingOpen} onClose={() => setBriefingOpen(false)} />
      <RelationshipPanel open={sharingOpen} onClose={() => setSharingOpen(false)} />
    </div>
  );
}
