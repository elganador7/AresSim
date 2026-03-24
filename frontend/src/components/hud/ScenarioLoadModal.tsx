import { useEffect, useState } from "react";
import { GetScenario, ListScenarios, LoadScenarioFromProto, RunProvingGroundScenario } from "../../../wailsjs/go/main/App";

type ScenarioRow = Record<string, any>;

export default function ScenarioLoadModal({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const [items, setItems] = useState<ScenarioRow[]>([]);
  const [busy, setBusy] = useState(false);
  const [openingId, setOpeningId] = useState("");
  const [runningId, setRunningId] = useState("");
  const [error, setError] = useState("");
  const [benchmarkResults, setBenchmarkResults] = useState<Record<string, Record<string, any>>>({});

  useEffect(() => {
    if (!open) {
      return;
    }
    let cancelled = false;
    setBusy(true);
    setError("");
    ListScenarios()
      .then((rows) => {
        if (!cancelled) {
          setItems(rows);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setBusy(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open]);

  const handleOpen = async (id: string) => {
    setOpeningId(id);
    setError("");
    try {
      const b64 = await GetScenario(id);
      const result = await LoadScenarioFromProto(b64);
      if (!result.success) {
        throw new Error(result.error || "Failed to load scenario");
      }
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setOpeningId("");
    }
  };

  const handleBenchmark = async (id: string, recommendedTrials: number) => {
    setRunningId(id);
    setError("");
    try {
      const result = await RunProvingGroundScenario(id, recommendedTrials || 0);
      setBenchmarkResults((current) => ({ ...current, [id]: result }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setRunningId("");
    }
  };

  if (!open) {
    return null;
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal scenario-quickload-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <span>Open Scenario</span>
          <button className="modal-close" onClick={onClose}>✕</button>
        </div>
        {error && <div className="scenario-quickload-error">{error}</div>}
        {busy ? (
          <div className="modal-empty">Loading…</div>
        ) : items.length === 0 ? (
          <div className="modal-empty">No saved scenarios.</div>
        ) : (
          <div className="modal-body">
            {items.map((row) => {
              const id = String(row.id ?? "");
              const name = String(row.name ?? "Untitled");
              const author = String(row.author ?? "Unknown author");
              const description = String(row.description ?? "").trim();
              const scenarioKind = String(row.scenario_kind ?? "");
              const provingPurpose = String(row.proving_ground_purpose ?? "").trim();
              const provingExpected = String(row.proving_ground_expected ?? "").trim();
              const provingCategory = String(row.proving_ground_category ?? "").trim();
              const recommendedTrials = Number(row.recommended_trials ?? 0);
              const benchmark = benchmarkResults[id];
              return (
                <div key={id} className="modal-scenario-item">
                  <div className="modal-scenario-copy">
                    <div className="modal-scenario-name">{name}</div>
                    <div className="modal-scenario-meta">{author}</div>
                    {scenarioKind === "proving_ground" && (
                      <div className="modal-scenario-meta">Proving Ground{provingCategory ? ` · ${provingCategory}` : ""}</div>
                    )}
                    {description && <div className="modal-scenario-description">{description}</div>}
                    {provingPurpose && <div className="modal-scenario-description">{provingPurpose}</div>}
                    {provingExpected && <div className="modal-scenario-description"><strong>Expected:</strong> {provingExpected}</div>}
                    {benchmark && (
                      <div className="modal-scenario-description">
                        <strong>{benchmark.pass ? "PASS" : "FAIL"}</strong>
                        {` · Trials ${benchmark.trials} · ${benchmark.focusTeam} win ${(Number(benchmark.focusWinRate ?? 0) * 100).toFixed(0)}%`}
                        {Number(benchmark.targetMissionKillRate ?? 0) > 0 ? ` · mission kill ${(Number(benchmark.targetMissionKillRate) * 100).toFixed(0)}%` : ""}
                        {` · mean ${(Number(benchmark.meanElapsedSeconds ?? 0) / 60).toFixed(1)} min`}
                      </div>
                    )}
                    {benchmark && (
                      <div className="modal-scenario-description">
                        {Number(benchmark.meanFirstShotSeconds ?? -1) >= 0 ? `First shot ${(Number(benchmark.meanFirstShotSeconds) / 60).toFixed(1)} min` : "No shots fired"}
                        {` · shots ${Number(benchmark.meanShotsFired ?? 0).toFixed(1)}`}
                        {` · hits ${Number(benchmark.meanHitsScored ?? 0).toFixed(1)}`}
                        {` · losses ${Number(benchmark.meanFocusLosses ?? 0).toFixed(1)} / ${Number(benchmark.meanOpposingLosses ?? 0).toFixed(1)}`}
                      </div>
                    )}
                    {benchmark && benchmark.terminalReasons && (
                      <div className="modal-scenario-description">
                        <strong>Ends:</strong>{" "}
                        {Object.entries(benchmark.terminalReasons as Record<string, number>)
                          .map(([reason, count]) => `${reason} ${count}`)
                          .join(" · ")}
                      </div>
                    )}
                  </div>
                  <div className="modal-list-actions">
                    {scenarioKind === "proving_ground" && (
                      <button
                        className="btn btn-sm"
                        disabled={runningId === id || openingId === id}
                        onClick={() => handleBenchmark(id, recommendedTrials)}
                      >
                        {runningId === id ? "Running…" : `Run ${recommendedTrials || 0}`}
                      </button>
                    )}
                    <button
                      className="btn btn-sm btn-primary"
                      disabled={openingId === id || runningId === id}
                      onClick={() => handleOpen(id)}
                    >
                      {openingId === id ? "Opening…" : "Open"}
                    </button>
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
