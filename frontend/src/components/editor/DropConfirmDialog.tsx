/**
 * DropConfirmDialog.tsx
 *
 * Compact modal shown after a unit is dropped on the globe.
 * Asks for Designator and Side only.
 */

import { useRef, useEffect, useState } from "react";
import { blankUnit, type PendingDrop, type UnitDraft } from "../../store/editorStore";

const SIDE_COLOR: Record<string, string> = {
  Blue: "#3b82f6",
  Red: "#ef4444",
  Neutral: "#f59e0b",
};

interface Props {
  drop: PendingDrop;
  onConfirm: (unit: UnitDraft) => void;
  onCancel: () => void;
}

export default function DropConfirmDialog({ drop, onConfirm, onCancel }: Props) {
  const [designator, setDesignator] = useState(() => {
    const base = drop.label.toUpperCase().replace(/\s+/g, "-");
    return `${base}-1`;
  });
  const [side, setSide] = useState<"Blue" | "Red" | "Neutral">("Blue");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const t = setTimeout(() => inputRef.current?.focus(), 50);
    return () => clearTimeout(t);
  }, []);

  const handleConfirm = () => {
    const unit: UnitDraft = {
      ...blankUnit(drop.lat, drop.lon),
      displayName: designator.trim() || drop.label,
      side,
      definitionId: drop.definitionId,
      lat: drop.lat,
      lon: drop.lon,
    };
    onConfirm(unit);
  };

  const handleKey = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") handleConfirm();
    if (e.key === "Escape") onCancel();
  };

  return (
    <div className="modal-backdrop" onClick={onCancel}>
      <div className="drop-dialog" onClick={(e) => e.stopPropagation()} onKeyDown={handleKey}>
        <div className="drop-dialog-header">
          <span className="drop-dialog-domain-swatch" style={{ background: drop.domainColor }} />
          <span className="drop-dialog-label">{drop.label}</span>
          <span className="drop-dialog-coords">
            {drop.lat.toFixed(3)}° {drop.lon.toFixed(3)}°
          </span>
        </div>

        <div className="drop-dialog-body">
          <div className="field">
            <label className="field-label">Designator</label>
            <input
              ref={inputRef}
              className="field-input"
              value={designator}
              onChange={(e) => setDesignator(e.target.value)}
              placeholder="e.g. 1-68 AR"
            />
          </div>

          <div className="field">
            <label className="field-label">Side</label>
            <div className="drop-side-tabs">
              {(["Blue", "Red", "Neutral"] as const).map((s) => (
                <button
                  key={s}
                  className={`drop-side-tab${side === s ? " active" : ""}`}
                  data-side={s}
                  onClick={() => setSide(s)}
                  style={
                    side === s
                      ? {
                          background: `${SIDE_COLOR[s]}22`,
                          borderColor: `${SIDE_COLOR[s]}88`,
                          color: SIDE_COLOR[s],
                        }
                      : undefined
                  }
                >
                  {s}
                </button>
              ))}
            </div>
          </div>
        </div>

        <div className="drop-dialog-footer">
          <button className="btn btn-success" onClick={handleConfirm}>Confirm</button>
          <button className="btn" onClick={onCancel}>Cancel</button>
        </div>
      </div>
    </div>
  );
}
