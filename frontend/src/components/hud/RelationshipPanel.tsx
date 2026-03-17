import { useMemo, useState } from "react";
import { RequestSync, SetCountryRelationship } from "../../../wailsjs/go/main/App";
import { useSimStore, type CountryRelationship } from "../../store/simStore";
import { inferUnitTeamCode } from "../../utils/unitTeams";

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
  const units = useSimStore((s) => s.units);
  const [busyKey, setBusyKey] = useState("");

  const countries = useMemo(() => {
    const codes = new Set<string>();
    units.forEach((unit) => {
      const code = unit.teamId?.trim().toUpperCase()
        || inferUnitTeamCode(unit.id, unit.side, undefined);
      if (/^[A-Z]{3}$/.test(code)) {
        codes.add(code);
      }
    });
    return Array.from(codes).sort();
  }, [units]);

  const getRelationship = (fromCountry: string, toCountry: string): CountryRelationship => (
    relationships.find((rel) => rel.fromCountry === fromCountry && rel.toCountry === toCountry) ?? {
      fromCountry,
      toCountry,
      shareIntel: false,
      airspaceTransitAllowed: false,
      airspaceStrikeAllowed: false,
      defensivePositioningAllowed: false,
    }
  );

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
      ));
      ensureSuccess(await RequestSync());
    } catch (error) {
      console.error(error);
      alert(error instanceof Error ? error.message : String(error));
    } finally {
      setBusyKey("");
    }
  };

  if (!open) {
    return null;
  }

  return (
    <div className="sharing-panel">
      <div className="sharing-panel-header">
        Country Relationships
        <button className="sharing-panel-close" onClick={onClose}>×</button>
      </div>
      <div className="sharing-panel-body">
        {countries.length < 2 ? (
          <div className="sharing-empty">Need at least two countries in the scenario.</div>
        ) : (
          countries.flatMap((fromCountry) =>
            countries
              .filter((toCountry) => toCountry !== fromCountry)
              .map((toCountry) => {
                const key = `${fromCountry}-${toCountry}`;
                const relationship = getRelationship(fromCountry, toCountry);
                return (
                  <div key={key} className="sharing-row">
                    <span className="sharing-label">{fromCountry} → {toCountry}</span>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={relationship.shareIntel}
                        disabled={busyKey === key}
                        onChange={(e) => updateRelationship({ ...relationship, shareIntel: e.target.checked })}
                      />
                      Intel
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={relationship.airspaceTransitAllowed}
                        disabled={busyKey === key}
                        onChange={(e) => updateRelationship({ ...relationship, airspaceTransitAllowed: e.target.checked })}
                      />
                      Transit
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={relationship.airspaceStrikeAllowed}
                        disabled={busyKey === key}
                        onChange={(e) => updateRelationship({ ...relationship, airspaceStrikeAllowed: e.target.checked })}
                      />
                      Strike
                    </label>
                    <label className="sharing-flag">
                      <input
                        type="checkbox"
                        checked={relationship.defensivePositioningAllowed}
                        disabled={busyKey === key}
                        onChange={(e) => updateRelationship({ ...relationship, defensivePositioningAllowed: e.target.checked })}
                      />
                      Defensive
                    </label>
                  </div>
                );
              }),
          )
        )}
      </div>
    </div>
  );
}
