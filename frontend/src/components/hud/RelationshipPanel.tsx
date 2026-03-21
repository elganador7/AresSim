import { useEffect, useMemo, useState } from "react";
import { PreviewCurrentRelationships, RequestSync, SetCountryRelationship } from "../../../wailsjs/go/main/App";
import { useSimStore, type CountryRelationship } from "../../store/simStore";
import { buildCountryCoalitionMap } from "../../utils/countryRelationships";

type RelationshipFlagKey =
  | "shareIntel"
  | "airspaceTransitAllowed"
  | "airspaceStrikeAllowed"
  | "defensivePositioningAllowed"
  | "maritimeTransitAllowed"
  | "maritimeStrikeAllowed";

type PairCard = {
  anchor: string;
  other: string;
  outgoing: CountryRelationship;
  incoming: CountryRelationship;
  atWar: boolean;
};

const FLAG_GROUPS: Array<{
  title: string;
  flags: Array<{ key: RelationshipFlagKey; label: string; shortLabel: string }>;
}> = [
  {
    title: "Intel And Air",
    flags: [
      { key: "shareIntel", label: "Share intel", shortLabel: "Intel" },
      { key: "airspaceTransitAllowed", label: "Allow air transit", shortLabel: "Transit" },
      { key: "airspaceStrikeAllowed", label: "Allow air strikes", shortLabel: "Strike" },
      { key: "defensivePositioningAllowed", label: "Allow defensive basing", shortLabel: "Defensive" },
    ],
  },
  {
    title: "Maritime",
    flags: [
      { key: "maritimeTransitAllowed", label: "Allow sea transit", shortLabel: "Sea Transit" },
      { key: "maritimeStrikeAllowed", label: "Allow maritime strikes", shortLabel: "Sea Strike" },
    ],
  },
];

function ensureSuccess(result: { success: boolean; error?: string }) {
  if (!result.success) {
    throw new Error(result.error || "Command failed");
  }
}

function defaultRelationship(fromCountry: string, toCountry: string): CountryRelationship {
  return {
    fromCountry,
    toCountry,
    shareIntel: false,
    airspaceTransitAllowed: false,
    airspaceStrikeAllowed: false,
    defensivePositioningAllowed: false,
    maritimeTransitAllowed: false,
    maritimeStrikeAllowed: false,
  };
}

function relationshipSummary(relationship: CountryRelationship): string {
  const active: string[] = [];
  if (relationship.shareIntel) active.push("intel");
  if (relationship.airspaceTransitAllowed) active.push("air transit");
  if (relationship.airspaceStrikeAllowed) active.push("air strike");
  if (relationship.defensivePositioningAllowed) active.push("defensive basing");
  if (relationship.maritimeTransitAllowed) active.push("sea transit");
  if (relationship.maritimeStrikeAllowed) active.push("sea strike");
  return active.length > 0 ? active.join(", ") : "No permissions granted";
}

function warStatusSummary(atWar: boolean, outgoing: CountryRelationship, incoming: CountryRelationship): string {
  if (!atWar) {
    return "Not at war";
  }
  const strike = outgoing.airspaceStrikeAllowed && incoming.airspaceStrikeAllowed;
  const transit = outgoing.airspaceTransitAllowed && incoming.airspaceTransitAllowed;
  if (strike && transit) {
    return "At war · mutual transit and strike allowed";
  }
  if (strike) {
    return "At war · strike allowed";
  }
  if (transit) {
    return "At war · transit allowed";
  }
  return "At war · permissions constrained by override";
}

function RelationshipStateGrid({
  relationship,
  editable,
  disabled,
  onChange,
}: {
  relationship: CountryRelationship;
  editable: boolean;
  disabled: boolean;
  onChange?: (next: CountryRelationship) => void;
}) {
  return (
    <div className={`relationship-state-grid${editable ? " editable" : ""}`}>
      {FLAG_GROUPS.map((group) => (
        <div key={group.title} className="relationship-flag-group">
          <div className="relationship-flag-group-title">{group.title}</div>
          <div className="relationship-flag-list">
            {group.flags.map((flag) => (
              <label
                key={flag.key}
                className={`relationship-toggle-pill${relationship[flag.key] ? " on" : " off"}${editable ? " editable" : ""}`}
              >
                <input
                  type="checkbox"
                  checked={relationship[flag.key]}
                  disabled={!editable || disabled}
                  onChange={(e) => onChange?.({ ...relationship, [flag.key]: e.target.checked })}
                />
                <span className="relationship-toggle-label">{flag.shortLabel}</span>
              </label>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
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
  const units = useSimStore((s) => s.units);
  const [busyKey, setBusyKey] = useState("");
  const [drafts, setDrafts] = useState<Record<string, CountryRelationship>>({});
  const [effectiveRelationships, setEffectiveRelationships] = useState<CountryRelationship[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) {
      return;
    }
    let cancelled = false;
    setError("");
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
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open, relationships]);

  const countries = useMemo(() => Array.from(new Set(
    effectiveRelationships.flatMap((relationship) => [relationship.fromCountry, relationship.toCountry]),
  )).sort(), [effectiveRelationships]);

  const focusedCountry = activeView !== "debug" && countries.includes(activeView) ? activeView : null;
  const countryCoalitions = useMemo(
    () => buildCountryCoalitionMap(Array.from(units.values())),
    [units],
  );

  const getRelationship = (fromCountry: string, toCountry: string): CountryRelationship =>
    effectiveRelationships.find((rel) => rel.fromCountry === fromCountry && rel.toCountry === toCountry)
    ?? defaultRelationship(fromCountry, toCountry);

  const cards = useMemo<PairCard[]>(() => {
    if (focusedCountry) {
      return countries
        .filter((country) => country !== focusedCountry)
        .map((other) => ({
          anchor: focusedCountry,
          other,
          outgoing: getRelationship(focusedCountry, other),
          incoming: getRelationship(other, focusedCountry),
          atWar: Boolean(
            countryCoalitions[focusedCountry]
            && countryCoalitions[other]
            && countryCoalitions[focusedCountry] !== countryCoalitions[other],
          ),
        }));
    }

    const pairs: PairCard[] = [];
    for (let i = 0; i < countries.length; i += 1) {
      for (let j = i + 1; j < countries.length; j += 1) {
        const anchor = countries[i];
        const other = countries[j];
        pairs.push({
          anchor,
          other,
          outgoing: getRelationship(anchor, other),
          incoming: getRelationship(other, anchor),
          atWar: Boolean(
            countryCoalitions[anchor]
            && countryCoalitions[other]
            && countryCoalitions[anchor] !== countryCoalitions[other],
          ),
        });
      }
    }
    return pairs;
  }, [countries, countryCoalitions, focusedCountry, effectiveRelationships]);

  const stageRelationship = (next: CountryRelationship) => {
    const key = `${next.fromCountry}-${next.toCountry}`;
    setDrafts((current) => ({ ...current, [key]: next }));
  };

  const resetRelationship = (relationship: CountryRelationship) => {
    const key = `${relationship.fromCountry}-${relationship.toCountry}`;
    setDrafts((current) => {
      const next = { ...current };
      delete next[key];
      return next;
    });
  };

  const applyRelationship = async (relationship: CountryRelationship) => {
    const key = `${relationship.fromCountry}-${relationship.toCountry}`;
    setBusyKey(key);
    setError("");
    try {
      ensureSuccess(await SetCountryRelationship(
        relationship.fromCountry,
        relationship.toCountry,
        relationship.shareIntel,
        relationship.airspaceTransitAllowed,
        relationship.airspaceStrikeAllowed,
        relationship.defensivePositioningAllowed,
        relationship.maritimeTransitAllowed,
        relationship.maritimeStrikeAllowed,
      ));
      ensureSuccess(await RequestSync());
      setDrafts((current) => {
        const next = { ...current };
        delete next[key];
        return next;
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyKey("");
    }
  };

  if (!open) {
    return null;
  }

  return (
    <div className="sharing-panel relationship-panel">
      <div className="sharing-panel-header">
        <div className="relationship-panel-title-wrap">
          <div className="relationship-panel-title">
            {focusedCountry ? `${focusedCountry} Relationship Map` : "Country Relationships"}
          </div>
          <div className="relationship-panel-subtitle">
            {focusedCountry
              ? "Edit outgoing permissions from your country. Incoming permissions are shown for comparison."
              : "Review and edit directed permissions between every country pair in the scenario."}
          </div>
        </div>
        <button className="sharing-panel-close" onClick={onClose}>×</button>
      </div>
      <div className="sharing-panel-body relationship-panel-body">
        {error && <div className="relationship-panel-error">{error}</div>}
        {countries.length < 2 ? (
          <div className="sharing-empty">Need at least two countries in the scenario.</div>
        ) : (
          <>
            <div className="relationship-legend">
              <span className="relationship-legend-chip editable">Editable</span>
              <span className="relationship-legend-chip readonly">Read-only incoming view</span>
            </div>
            <div className="relationship-card-list">
              {cards.map((card) => {
                const outgoingKey = `${card.outgoing.fromCountry}-${card.outgoing.toCountry}`;
                const outgoing = drafts[outgoingKey] ?? card.outgoing;
                const dirty = drafts[outgoingKey] !== undefined;
                const editableOutgoing = focusedCountry ? outgoing.fromCountry === focusedCountry : true;
                return (
                  <div key={`${card.anchor}-${card.other}`} className="relationship-card">
                    <div className="relationship-card-header">
                      <div>
                        <div className="relationship-card-title">{card.anchor} ↔ {card.other}</div>
                        <div className="relationship-card-summary">
                          {card.anchor} → {card.other}: {relationshipSummary(outgoing)}
                        </div>
                        <div className={`relationship-war-badge${card.atWar ? " at-war" : ""}`}>
                          {warStatusSummary(card.atWar, outgoing, card.incoming)}
                        </div>
                      </div>
                      {dirty && <span className="relationship-card-dirty">Unsaved</span>}
                    </div>

                    <div className="relationship-direction-grid">
                      <div className="relationship-direction-block editable">
                        <div className="relationship-direction-header">
                          <span className="relationship-direction-label">{outgoing.fromCountry} → {outgoing.toCountry}</span>
                          <span className="relationship-direction-note">Editable</span>
                        </div>
                        <RelationshipStateGrid
                          relationship={outgoing}
                          editable={editableOutgoing}
                          disabled={busyKey === outgoingKey}
                          onChange={stageRelationship}
                        />
                        <div className="relationship-direction-actions">
                          <button
                            className="btn btn-sm btn-success"
                            disabled={!dirty || busyKey === outgoingKey}
                            onClick={() => applyRelationship(outgoing)}
                          >
                            Apply
                          </button>
                          <button
                            className="btn btn-sm"
                            disabled={!dirty || busyKey === outgoingKey}
                            onClick={() => resetRelationship(card.outgoing)}
                          >
                            Reset
                          </button>
                        </div>
                      </div>

                      <div className="relationship-direction-block readonly">
                        <div className="relationship-direction-header">
                          <span className="relationship-direction-label">{card.incoming.fromCountry} → {card.incoming.toCountry}</span>
                          <span className="relationship-direction-note">Read-only</span>
                        </div>
                        <RelationshipStateGrid
                          relationship={card.incoming}
                          editable={false}
                          disabled
                        />
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
