import { useEffect, useState } from "react";
import { GetScenario, ListScenarios, LoadScenarioFromProto } from "../../../wailsjs/go/main/App";

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
  const [error, setError] = useState("");

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
              return (
                <div key={id} className="modal-scenario-item">
                  <div className="modal-scenario-copy">
                    <div className="modal-scenario-name">{name}</div>
                    <div className="modal-scenario-meta">{author}</div>
                    {description && <div className="modal-scenario-description">{description}</div>}
                  </div>
                  <div className="modal-list-actions">
                    <button
                      className="btn btn-sm btn-primary"
                      disabled={openingId === id}
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
