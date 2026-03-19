import { useState } from "react";
import { PauseSim, SetSimSpeed } from "../../../wailsjs/go/main/App";
import { useSimStore } from "../../store/simStore";
import { formatSimTime } from "../../utils/formatters";
import RelationshipPanel from "./RelationshipPanel";

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

export default function TopBar({ onOpenEditor }: { onOpenEditor: () => void }) {
  const scenarioName = useSimStore((s) => s.scenarioName);
  const scenarioState = useSimStore((s) => s.scenarioState);
  const simSeconds = useSimStore((s) => s.simSeconds);
  const timeScale = useSimStore((s) => s.timeScale);
  const tickNumber = useSimStore((s) => s.tickNumber);
  const scores = useSimStore((s) => s.scores);
  const [sharingOpen, setSharingOpen] = useState(false);

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
        <button className="editor-btn" onClick={() => setSharingOpen((current) => !current)}>
          SHARING
        </button>
        <button className="editor-btn" onClick={onOpenEditor}>
          EDITOR
        </button>
      </div>
      <RelationshipPanel open={sharingOpen} onClose={() => setSharingOpen(false)} />
    </div>
  );
}
