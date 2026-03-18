import { useEffect, useMemo, useState } from "react";
import { PreviewCurrentRelationships, RequestSync, SetCountryRelationship } from "../../../wailsjs/go/main/App";
import { useSimStore, type CountryRelationship } from "../../store/simStore";

function ensureSuccess(result: { success: boolean; error?: string }) {
  if (!result.success) {
    throw new Error(result.error || "Command failed");
  }
}

export default function RelationshipPanel({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const relationships = useSimStore((s) => s.relationships);
  const activeView = useSimStore((s) => s.activeView);
  const [busyKey, setBusyKey] = useState("");
  const [drafts, setDrafts] = useState<Record<string, CountryRelationship>>({});
  const [effectiveRelationships, setEffectiveRelationships] = useState<CountryRelationship[]>([]);

  useEffect(() => {
    if (!open) {
      return;
    }
    let cancelled = false;
    PreviewCurrentRelationships()
      .then((rows) => {
        if (!cancelled) {
          setEffectiveRelationships(rows.map((row) => ({
            fromCountry: row.fromCountry,
            toCountry: row.toCountry,
            shareIntel: row.shareIntel,
            airspaceTransitAllowed: row.airspaceTransitAllowed,
            airspaceStrikeAllowed: row.airspaceStrikeAllowed,
            defensivePositioningAllowed: row.defensivePositioningAllowed,
            maritimeTransitAllowed: row.maritimeTransitAllowed,
            maritimeStrikeAllowed: row.maritimeStrikeAllowed,
          })));
        }
      })
      .catch(console.error);
    return () => {
      cancelled = true;
    };
  }, [open, relationships]);

  const countries = useMemo(() => Array.from(new Set(
    effectiveRelationships.flatMap((relationship) => [relationship.fromCountry, relationship.toCountry]),
  )).sort(), [effectiveRelationships]);

  const getRelationship = (fromCountry: string, toCountry: string): CountryRelationship => {
    return effectiveRelationships.find((rel) => rel.fromCountry === fromCountry && rel.toCountry === toCountry) ?? {
      fromCountry,
      toCountry,
      shareIntel: false,
      airspaceTransitAllowed: false,
      airspaceStrikeAllowed: false,
      defensivePositioningAllowed: false,
      maritimeTransitAllowed: false,
      maritimeStrikeAllowed: false,
    };
  };

  const updateRelationship = async (next: CountryRelationship) => {
    const key = `${next.fromCountry}-${next.toCountry}`;
    setBusyKey(key);
    try {
      ensureSuccess(await SetCountryRelationship(
        next.fromCountry,
        next.toCountry,
        next.shareIntel,
        next.airspaceTransitAllowed,
        next.airspaceStrikeAllowed,
        next.defensivePositioningAllowed,
        next.maritimeTransitAllowed,
        next.maritimeStrikeAllowed,
      ));
      ensureSuccess(await RequestSync());
    } catch (error) {
      console.error(error);
      alert(error instanceof Error ? error.message : String(error));
    } finally {
      setBusyKey("");
    }
  };

  const stageRelationship = (next: CountryRelationship) => {
    const key = `${next.fromCountry}-${next.toCountry}`;
    setDrafts((current) => ({ ...current, [key]: next }));
  };

  const applyRelationship = async (relationship: CountryRelationship) => {
    const key = `${relationship.fromCountry}-${relationship.toCountry}`;
    await updateRelationship(relationship);
    setDrafts((current) => {
      const next = { ...current };
      delete next[key];
      return next;
    });
  };

  const resetRelationship = (relationship: CountryRelationship) => {
    const key = `${relationship.fromCountry}-${relationship.toCountry}`;
    setDrafts((current) => {
      const next = { ...current };
      delete next[key];
      return next;
    });
  };

  if (!open) {
    return null;
  }

  const focusedCountry = activeView !== "debug" && countries.includes(activeView) ? activeView : null;

  return (
    <div className="sharing-panel">
      <div className="sharing-panel-header">
        {focusedCountry ? `${focusedCountry} Relationships` : "Country Relationships"}
        <button className="sharing-panel-close" onClick={onClose}>×</button>
      </div>
      <div className="sharing-panel-body">
        {countries.length < 2 ? (
          <div className="sharing-empty">Need at least two countries in the scenario.</div>
        ) : focusedCountry ? (
          countries
            .filter((country) => country !== focusedCountry)
            .map((otherCountry) => {
              const outgoingKey = `${focusedCountry}-${otherCountry}`;
              const baselineOutgoing = getRelationship(focusedCountry, otherCountry);
              const outgoing = drafts[outgoingKey] ?? baselineOutgoing;
              const incoming = getRelationship(otherCountry, focusedCountry);
              return (
                <div key={outgoingKey} className="sharing-pair">
                  <div className="sharing-pair-header">{focusedCountry} ↔ {otherCountry}</div>
                  <div className="sharing-row sharing-row-grouped">
                    <span className="sharing-label">Outgoing</span>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={outgoing.shareIntel}
                        disabled={busyKey === outgoingKey}
                        onChange={(e) => stageRelationship({ ...outgoing, shareIntel: e.target.checked })}
                      />
                      Intel
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={outgoing.airspaceTransitAllowed}
                        disabled={busyKey === outgoingKey}
                        onChange={(e) => stageRelationship({ ...outgoing, airspaceTransitAllowed: e.target.checked })}
                      />
                      Transit
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={outgoing.airspaceStrikeAllowed}
                        disabled={busyKey === outgoingKey}
                        onChange={(e) => stageRelationship({ ...outgoing, airspaceStrikeAllowed: e.target.checked })}
                      />
                      Strike
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={outgoing.defensivePositioningAllowed}
                        disabled={busyKey === outgoingKey}
                        onChange={(e) => stageRelationship({ ...outgoing, defensivePositioningAllowed: e.target.checked })}
                      />
                      Defensive
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={outgoing.maritimeTransitAllowed}
                        disabled={busyKey === outgoingKey}
                        onChange={(e) => stageRelationship({ ...outgoing, maritimeTransitAllowed: e.target.checked })}
                      />
                      Sea Transit
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={outgoing.maritimeStrikeAllowed}
                        disabled={busyKey === outgoingKey}
                        onChange={(e) => stageRelationship({ ...outgoing, maritimeStrikeAllowed: e.target.checked })}
                      />
                      Sea Strike
                    </label>
                  </div>
                  <div className="sharing-row sharing-row-grouped sharing-row-actions">
                    <span className="sharing-label">Actions</span>
                    <button
                      className="btn btn-sm btn-success"
                      disabled={busyKey === outgoingKey}
                      onClick={() => applyRelationship(outgoing)}
                    >
                      Apply
                    </button>
                    <button
                      className="btn btn-sm"
                      disabled={busyKey === outgoingKey}
                      onClick={() => resetRelationship(baselineOutgoing)}
                    >
                      Reset
                    </button>
                  </div>
                  <div className="sharing-row sharing-row-grouped sharing-row-readonly">
                    <span className="sharing-label">Incoming</span>
                    <label className="sharing-flag">
                      <input type="checkbox" checked={incoming.shareIntel} disabled />
                      Intel
                    </label>
                    <label className="sharing-flag">
                      <input type="checkbox" checked={incoming.airspaceTransitAllowed} disabled />
                      Transit
                    </label>
                    <label className="sharing-flag">
                      <input type="checkbox" checked={incoming.airspaceStrikeAllowed} disabled />
                      Strike
                    </label>
                    <label className="sharing-flag">
                      <input type="checkbox" checked={incoming.defensivePositioningAllowed} disabled />
                      Defensive
                    </label>
                    <label className="sharing-flag">
                      <input type="checkbox" checked={incoming.maritimeTransitAllowed} disabled />
                      Sea Transit
                    </label>
                    <label className="sharing-flag">
                      <input type="checkbox" checked={incoming.maritimeStrikeAllowed} disabled />
                      Sea Strike
                    </label>
                  </div>
                </div>
              );
            })
        ) : (
          countries.flatMap((fromCountry) =>
            countries
              .filter((toCountry) => toCountry !== fromCountry)
              .map((toCountry) => {
                const key = `${fromCountry}-${toCountry}`;
                const baselineRelationship = getRelationship(fromCountry, toCountry);
                const relationship = drafts[key] ?? baselineRelationship;
                return (
                  <div key={key} className="sharing-row">
                    <span className="sharing-label">{fromCountry} → {toCountry}</span>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={relationship.shareIntel}
                        disabled={busyKey === key}
                        onChange={(e) => stageRelationship({ ...relationship, shareIntel: e.target.checked })}
                      />
                      Intel
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={relationship.airspaceTransitAllowed}
                        disabled={busyKey === key}
                        onChange={(e) => stageRelationship({ ...relationship, airspaceTransitAllowed: e.target.checked })}
                      />
                      Transit
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={relationship.airspaceStrikeAllowed}
                        disabled={busyKey === key}
                        onChange={(e) => stageRelationship({ ...relationship, airspaceStrikeAllowed: e.target.checked })}
                      />
                      Strike
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={relationship.defensivePositioningAllowed}
                        disabled={busyKey === key}
                        onChange={(e) => stageRelationship({ ...relationship, defensivePositioningAllowed: e.target.checked })}
                      />
                      Defensive
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={relationship.maritimeTransitAllowed}
                        disabled={busyKey === key}
                        onChange={(e) => stageRelationship({ ...relationship, maritimeTransitAllowed: e.target.checked })}
                      />
                      Sea Transit
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={relationship.maritimeStrikeAllowed}
                        disabled={busyKey === key}
                        onChange={(e) => stageRelationship({ ...relationship, maritimeStrikeAllowed: e.target.checked })}
                      />
                      Sea Strike
                    </label>
                    <button className="btn btn-sm btn-success" disabled={busyKey === key} onClick={() => applyRelationship(relationship)}>
                      Apply
                    </button>
                    <button className="btn btn-sm" disabled={busyKey === key} onClick={() => resetRelationship(baselineRelationship)}>
                      Reset
                    </button>
                  </div>
                );
              }),
          )
        )}
      </div>
    </div>
  );
}
