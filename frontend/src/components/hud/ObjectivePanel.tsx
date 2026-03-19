import { useMemo } from "react";
import { useSimStore } from "../../store/simStore";
import { buildWarCostSummary, computeIranWarObjectiveProgress, getIranWarObjectiveSet, isIranWarScenario } from "../../utils/iranWarObjectives";

function formatUsdCompact(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value);
}

export default function ObjectivePanel({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const scenarioName = useSimStore((s) => s.scenarioName);
  const humanControlledTeam = useSimStore((s) => s.humanControlledTeam);
  const scores = useSimStore((s) => s.scores);
  const units = useSimStore((s) => s.units);

  const objectiveSet = useMemo(
    () => (isIranWarScenario(scenarioName) ? getIranWarObjectiveSet(humanControlledTeam) : null),
    [humanControlledTeam, scenarioName],
  );
  const warCost = useMemo(
    () => buildWarCostSummary(humanControlledTeam, units, scores),
    [humanControlledTeam, scores, units],
  );

  if (!open) {
    return null;
  }

  return (
    <div className="sharing-panel objective-panel">
      <div className="sharing-panel-header">
        Briefing
        <button className="sharing-panel-close" onClick={onClose}>×</button>
      </div>
      <div className="sharing-panel-body">
        {!isIranWarScenario(scenarioName) ? (
          <div className="sharing-empty">No scenario-specific briefing is authored for this scenario yet.</div>
        ) : !humanControlledTeam ? (
          <div className="objective-empty">
            <div className="objective-title">Select a Player Side</div>
            <div className="objective-copy">
              Choose `USA`, `ISR`, or `IRN` in the top bar. The AI will keep controlling the other major actors.
            </div>
          </div>
        ) : !objectiveSet ? (
          <div className="sharing-empty">No objectives are defined for {humanControlledTeam} yet.</div>
        ) : (
          <>
            <div className="objective-summary">
              <div className="objective-team">{humanControlledTeam}</div>
              <div className="objective-title">{objectiveSet.title}</div>
              <div className="objective-copy">{objectiveSet.summary}</div>
            </div>
            <div className="objective-costs">
              <div className="objective-cost-chip">
                <span>Your Side Loss</span>
                <strong>{formatUsdCompact(warCost.ownLossUsd)}</strong>
              </div>
              <div className="objective-cost-chip">
                <span>Enemy Loss</span>
                <strong>{formatUsdCompact(warCost.enemyLossUsd)}</strong>
              </div>
            </div>
            {objectiveSet.objectives.map((objective) => {
              const progress = computeIranWarObjectiveProgress(objective, units);
              const percent = progress.total > 0 ? Math.round((progress.completed / progress.total) * 100) : 0;
              return (
                <div key={objective.id} className="objective-card">
                  <div className="objective-card-header">
                    <div className="objective-card-title">{objective.title}</div>
                    <div className="objective-card-progress">{progress.label}</div>
                  </div>
                  <div className="objective-copy">{objective.detail}</div>
                  <div className="objective-progress">
                    <div className="objective-progress-fill" style={{ width: `${percent}%` }} />
                  </div>
                </div>
              );
            })}
          </>
        )}
      </div>
    </div>
  );
}
