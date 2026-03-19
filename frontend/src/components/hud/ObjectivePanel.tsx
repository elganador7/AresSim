import { useMemo } from "react";
import { useSimStore } from "../../store/simStore";
import { buildWarCostSummary, computeIranWarObjectiveProgress, getIranWarAirbaseConstraints, getIranWarGroundedAircraft, getIranWarKeyTargetStatuses, getIranWarObjectiveSet, getIranWarOpeningWaveStatus, getIranWarScoreboard, getIranWarStrikeForceSummary, getIranWarStrikeUnitStatuses, isIranWarScenario } from "../../utils/iranWarObjectives";

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
  const openingWave = useMemo(
    () => getIranWarOpeningWaveStatus(humanControlledTeam, units),
    [humanControlledTeam, units],
  );
  const keyTargets = useMemo(
    () => getIranWarKeyTargetStatuses(humanControlledTeam, units),
    [humanControlledTeam, units],
  );
  const scoreboard = useMemo(
    () => getIranWarScoreboard(humanControlledTeam, units),
    [humanControlledTeam, units],
  );
  const strikeForce = useMemo(
    () => getIranWarStrikeForceSummary(humanControlledTeam, units),
    [humanControlledTeam, units],
  );
  const groundedAircraft = useMemo(
    () => getIranWarGroundedAircraft(humanControlledTeam, units),
    [humanControlledTeam, units],
  );
  const airbaseConstraints = useMemo(
    () => getIranWarAirbaseConstraints(humanControlledTeam, units),
    [humanControlledTeam, units],
  );
  const strikeUnitStatuses = useMemo(
    () => getIranWarStrikeUnitStatuses(humanControlledTeam, units),
    [humanControlledTeam, units],
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
            <div className="objective-summary">
              <div className="objective-title">First 24h Scoreboard</div>
              <div className="objective-key-grid">
                {scoreboard.map((item) => (
                  <div key={item.label} className="objective-key-card">
                    <span>{item.label}</span>
                    <strong className={`objective-key-${item.severity}`}>{item.value}</strong>
                  </div>
                ))}
              </div>
              <div className="objective-copy">
                Strike-capable force: {strikeForce.ready} ready, {strikeForce.delayed} delayed, {strikeForce.spentOrLost} spent or lost.
              </div>
            </div>
            {openingWave.length > 0 && (
              <div className="objective-summary">
                <div className="objective-title">Opening Wave</div>
                <div className="objective-opening-list">
                  {openingWave.map((item) => (
                    <div key={item.shooterId} className="objective-opening-row">
                      <div>
                        <strong>{item.shooterLabel}</strong>
                        <span>{item.targetLabel}</span>
                      </div>
                      <span className={`objective-status-chip objective-status-${item.status}`}>{item.status}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
            {keyTargets.length > 0 && (
              <div className="objective-summary">
                <div className="objective-title">Key Targets</div>
                <div className="objective-key-grid">
                  {keyTargets.map((item) => (
                    <div key={item.unitId} className="objective-key-card">
                      <span>{item.label}</span>
                      <strong className={`objective-key-${item.severity}`}>{item.status}</strong>
                    </div>
                  ))}
                </div>
              </div>
            )}
            {(groundedAircraft.length > 0 || airbaseConstraints.length > 0 || strikeUnitStatuses.length > 0) && (
              <div className="objective-summary">
                <div className="objective-title">Operational Pressure</div>
                {groundedAircraft.length > 0 && (
                  <div className="objective-subsection">
                    <div className="objective-subtitle">Grounded Aircraft</div>
                    <div className="objective-status-list">
                      {groundedAircraft.map((item) => (
                        <div key={item.unitId} className="objective-status-row">
                          <span>{item.label}</span>
                          <strong className={`objective-key-${item.severity}`}>{item.status}</strong>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                {airbaseConstraints.length > 0 && (
                  <div className="objective-subsection">
                    <div className="objective-subtitle">Airbase Constraints</div>
                    <div className="objective-status-list">
                      {airbaseConstraints.map((item) => (
                        <div key={item.unitId} className="objective-status-row">
                          <span>{item.label}</span>
                          <strong className={`objective-key-${item.severity}`}>{item.status}</strong>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                {strikeUnitStatuses.length > 0 && (
                  <div className="objective-subsection">
                    <div className="objective-subtitle">Strike Units</div>
                    <div className="objective-status-list">
                      {strikeUnitStatuses.map((item) => (
                        <div key={item.unitId} className="objective-status-row">
                          <span>{item.label}</span>
                          <strong className={`objective-key-${item.severity}`}>{item.status}</strong>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
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
