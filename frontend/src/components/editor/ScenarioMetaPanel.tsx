import { useEffect, useMemo, useState } from "react";
import { PreviewDraftRelationships } from "../../../wailsjs/go/main/App";
import { useEditorStore, type CountryRelationshipDraft } from "../../store/editorStore";
import { buildCountryCoalitionMap, collectRelationshipCountries, normalizeCountryCode } from "../../utils/countryRelationships";
import { formatCountry } from "./scenarioSerialization";

const WEATHER_STATES = [
  { value: 1, label: "Clear" },
  { value: 2, label: "Overcast" },
  { value: 3, label: "Fog" },
  { value: 4, label: "Rain" },
  { value: 5, label: "Heavy Rain" },
  { value: 6, label: "Snow" },
  { value: 7, label: "Blizzard" },
];

export default function ScenarioMetaPanel() {
  const draft = useEditorStore((s) => s.draft);
  const updateMeta = useEditorStore((s) => s.updateMeta);
  const countries = useMemo(() => collectRelationshipCountries(
    draft.units.map((unit) => ({ teamId: normalizeCountryCode(unit.teamId) })),
    draft.relationships,
  ), [draft.relationships, draft.units]);
  const countryCoalitionsJSON = useMemo(
    () => JSON.stringify(buildCountryCoalitionMap(draft.units)),
    [draft.units],
  );
  const countriesJSON = useMemo(() => JSON.stringify(countries), [countries]);
  const relationshipsJSON = useMemo(() => JSON.stringify(draft.relationships), [draft.relationships]);
  const [effectiveRelationships, setEffectiveRelationships] = useState<CountryRelationshipDraft[]>([]);

  const startDate = new Date(draft.startTimeUnix * 1000);
  const dateStr = startDate.toISOString().slice(0, 16);

  useEffect(() => {
    let cancelled = false;
    PreviewDraftRelationships(
      relationshipsJSON,
      countryCoalitionsJSON,
      countriesJSON,
    )
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
  }, [countriesJSON, countryCoalitionsJSON, relationshipsJSON]);

  const patchRelationship = (
    fromCountry: string,
    toCountry: string,
    key: keyof CountryRelationshipDraft,
    value: boolean,
  ) => {
    const next = [...draft.relationships];
    const index = next.findIndex(
      (rel) => normalizeCountryCode(rel.fromCountry) === fromCountry && normalizeCountryCode(rel.toCountry) === toCountry,
    );
    if (index >= 0) {
      next[index] = { ...next[index], [key]: value };
    } else {
      next.push({
        fromCountry,
        toCountry,
        shareIntel: false,
        airspaceTransitAllowed: false,
        airspaceStrikeAllowed: false,
        defensivePositioningAllowed: false,
        maritimeTransitAllowed: false,
        maritimeStrikeAllowed: false,
        [key]: value,
      });
    }
    updateMeta({ relationships: next });
  };

  return (
    <div className="panel-scroll">
      <div className="panel-section">
        <div className="panel-section-header">Scenario</div>
        <div className="field">
          <label className="field-label">Name</label>
          <input
            className="field-input"
            value={draft.name}
            onChange={(e) => updateMeta({ name: e.target.value })}
          />
        </div>
        <div className="field">
          <label className="field-label">Description</label>
          <textarea
            className="field-textarea"
            value={draft.description}
            onChange={(e) => updateMeta({ description: e.target.value })}
          />
        </div>
        <div className="field">
          <label className="field-label">Classification</label>
          <input
            className="field-input"
            value={draft.classification}
            onChange={(e) => updateMeta({ classification: e.target.value })}
          />
        </div>
        <div className="field">
          <label className="field-label">Author</label>
          <input
            className="field-input"
            value={draft.author}
            onChange={(e) => updateMeta({ author: e.target.value })}
          />
        </div>
        <div className="field">
          <label className="field-label">Start Date / Time (UTC)</label>
          <input
            className="field-input"
            type="datetime-local"
            value={dateStr}
            onChange={(e) =>
              updateMeta({ startTimeUnix: new Date(e.target.value + "Z").getTime() / 1000 })
            }
          />
        </div>
      </div>

      <div className="panel-section">
        <div className="panel-section-header">Simulation</div>
        <div className="field-row">
          <div className="field">
            <label className="field-label">Tick Rate (Hz)</label>
            <input
              className="field-input"
              type="number"
              min={1}
              max={60}
              value={draft.tickRateHz}
              onChange={(e) => updateMeta({ tickRateHz: Number(e.target.value) })}
            />
          </div>
          <div className="field">
            <label className="field-label">Time Scale</label>
            <input
              className="field-input"
              type="number"
              min={0.1}
              max={3600}
              step={0.1}
              value={draft.timeScale}
              onChange={(e) => updateMeta({ timeScale: Number(e.target.value) })}
            />
          </div>
        </div>
      </div>

      <div className="panel-section">
        <div className="panel-section-header">Weather</div>
        <div className="field">
          <label className="field-label">State</label>
          <select
            className="field-select"
            value={draft.weatherState}
            onChange={(e) => updateMeta({ weatherState: Number(e.target.value) })}
          >
            {WEATHER_STATES.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
        </div>
        <div className="field-row">
          <div className="field">
            <label className="field-label">Visibility (km)</label>
            <input
              className="field-input"
              type="number"
              min={0.1}
              step={0.1}
              value={draft.visibilityKm}
              onChange={(e) => updateMeta({ visibilityKm: Number(e.target.value) })}
            />
          </div>
          <div className="field">
            <label className="field-label">Wind (m/s)</label>
            <input
              className="field-input"
              type="number"
              min={0}
              step={0.1}
              value={draft.windSpeedMps}
              onChange={(e) => updateMeta({ windSpeedMps: Number(e.target.value) })}
            />
          </div>
        </div>
        <div className="field">
          <label className="field-label">Temperature (°C)</label>
          <input
            className="field-input"
            type="number"
            step={0.1}
            value={draft.temperatureC}
            onChange={(e) => updateMeta({ temperatureC: Number(e.target.value) })}
          />
        </div>
      </div>

      <div className="panel-section">
        <div className="panel-section-header">Country Relationships</div>
        {countries.length < 2 ? (
          <div className="selected-unit-empty">
            Add units from at least two countries to configure access and intel sharing.
          </div>
        ) : (
          <div className="relationship-grid">
            {countries.flatMap((fromCountry) =>
              countries
                .filter((toCountry) => toCountry !== fromCountry)
                .map((toCountry) => {
                  const relationship = effectiveRelationships.find(
                    (candidate) => candidate.fromCountry === fromCountry && candidate.toCountry === toCountry,
                  ) ?? {
                    fromCountry,
                    toCountry,
                    shareIntel: false,
                    airspaceTransitAllowed: false,
                    airspaceStrikeAllowed: false,
                    defensivePositioningAllowed: false,
                    maritimeTransitAllowed: false,
                    maritimeStrikeAllowed: false,
                  };
                  return (
                    <div key={`${fromCountry}-${toCountry}`} className="relationship-row">
                      <div className="relationship-label">
                        {formatCountry(fromCountry)} → {formatCountry(toCountry)}
                      </div>
                      <label className="relationship-toggle">
                        <input
                          type="checkbox"
                          checked={relationship.shareIntel}
                          onChange={(e) => patchRelationship(fromCountry, toCountry, "shareIntel", e.target.checked)}
                        />
                        Intel
                      </label>
                      <label className="relationship-toggle">
                        <input
                          type="checkbox"
                          checked={relationship.airspaceTransitAllowed}
                          onChange={(e) => patchRelationship(fromCountry, toCountry, "airspaceTransitAllowed", e.target.checked)}
                        />
                        Transit
                      </label>
                      <label className="relationship-toggle">
                        <input
                          type="checkbox"
                          checked={relationship.airspaceStrikeAllowed}
                          onChange={(e) => patchRelationship(fromCountry, toCountry, "airspaceStrikeAllowed", e.target.checked)}
                        />
                        Strike
                      </label>
                      <label className="relationship-toggle">
                        <input
                          type="checkbox"
                          checked={relationship.defensivePositioningAllowed}
                          onChange={(e) =>
                            patchRelationship(fromCountry, toCountry, "defensivePositioningAllowed", e.target.checked)
                          }
                        />
                        Defensive
                      </label>
                      <label className="relationship-toggle">
                        <input
                          type="checkbox"
                          checked={relationship.maritimeTransitAllowed}
                          onChange={(e) => patchRelationship(fromCountry, toCountry, "maritimeTransitAllowed", e.target.checked)}
                        />
                        Sea Transit
                      </label>
                      <label className="relationship-toggle">
                        <input
                          type="checkbox"
                          checked={relationship.maritimeStrikeAllowed}
                          onChange={(e) => patchRelationship(fromCountry, toCountry, "maritimeStrikeAllowed", e.target.checked)}
                        />
                        Sea Strike
                      </label>
                    </div>
                  );
                }),
            )}
          </div>
        )}
      </div>
    </div>
  );
}
