/**
 * ScenarioEditor.tsx
 *
 * Three-panel scenario editor:
 *   Left   — Scenario metadata, simulation settings, initial weather
 *   Center — CesiumJS globe (drag-and-drop unit placement)
 *   Right  — Unit palette (draggable tree) + placed units list
 *
 * Drop flow:
 *   1. User drags unit type from palette → drops on globe
 *   2. EditorGlobe fires onUnitDrop(lat, lon, payload)
 *   3. DropConfirmDialog asks for Designator + Side
 *   4. Confirm → addUnit to editorStore
 */

import { useCallback, useEffect, useState } from "react";
import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import { ScenarioSchema } from "@proto/engine/v1/scenario_pb";
import { UnitSchema } from "@proto/engine/v1/unit_pb";
import { MoveOrderSchema, PositionSchema } from "@proto/engine/v1/common_pb";
import { OperationalStatusSchema } from "@proto/engine/v1/status_pb";
import {
  SaveScenario,
  LoadScenarioFromProto,
  ListScenarios,
  GetScenario,
  DeleteScenario,
  ListWeaponDefinitions,
} from "../../../wailsjs/go/main/App";
import {
  useEditorStore,
  type CountryRelationshipDraft,
  type UnitDraft,
  type ScenarioDraft,
  type UnitDefinitionDraft,
} from "../../store/editorStore";
import EditorGlobe from "./EditorGlobe";
import UnitPalette, { type DragPayload } from "./UnitPalette";
import DropConfirmDialog from "./DropConfirmDialog";
import UnitDefinitionManager from "./UnitDefinitionManager";
import "./editor.css";
import { assessLoadoutAgainstTarget, type WeaponDefLite } from "../../utils/loadoutValidation";
import { getCountriesAlongRoute, getCountriesAlongSegment, getCountryCodeForPoint } from "../../utils/theaterCountries";
import { getRelationshipRule, normalizeCountryCode } from "../../utils/countryRelationships";
import { EDITOR_COUNTRY_NAME_BY_CODE } from "../../data/editorCountries";
import { ATTACK_ORDER_TYPES, DESIRED_EFFECTS, ENGAGEMENT_BEHAVIORS, filterValidEditorTargets } from "../../utils/tasking";

// ─── HELPERS ─────────────────────────────────────────────────────────────────

function bytesToBase64(bytes: Uint8Array): string {
  let binary = "";
  for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
  return btoa(binary);
}

function base64ToBytes(b64: string): Uint8Array {
  const binary = atob(b64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

function draftToProtoB64(draft: ScenarioDraft): string {
  const scenario = create(ScenarioSchema, {
    id: draft.id,
    name: draft.name,
    description: draft.description,
    classification: draft.classification,
    author: draft.author,
    startTimeUnix: draft.startTimeUnix,
    version: draft.version,
    settings: { tickRateHz: draft.tickRateHz, timeScale: draft.timeScale },
    map: {
      initialWeather: {
        state: draft.weatherState,
        visibilityKm: draft.visibilityKm,
        windSpeedMps: draft.windSpeedMps,
        temperatureC: draft.temperatureC,
      },
    },
    relationships: draft.relationships.map((rel) => ({
      fromCountry: rel.fromCountry,
      toCountry: rel.toCountry,
      shareIntel: rel.shareIntel,
      airspaceTransitAllowed: rel.airspaceTransitAllowed,
      airspaceStrikeAllowed: rel.airspaceStrikeAllowed,
      defensivePositioningAllowed: rel.defensivePositioningAllowed,
    })),
    units: draft.units.map((u) =>
      create(UnitSchema, {
        id: u.id,
        displayName: u.displayName,
        fullName: u.fullName,
        side: u.side,
        teamId: u.teamId,
        coalitionId: u.coalitionId,
        definitionId: u.definitionId,
        parentUnitId: u.parentUnitId,
        loadoutConfigurationId: u.loadoutConfigurationId,
        natoSymbolSidc: u.natoSymbolSidc,
        damageState: u.damageState,
        engagementBehavior: u.engagementBehavior,
        engagementPkillThreshold: u.engagementPkillThreshold,
        attackOrder: u.attackOrder
          ? {
              orderType: u.attackOrder.orderType,
              targetUnitId: u.attackOrder.targetUnitId,
              desiredEffect: u.attackOrder.desiredEffect,
              pkillThreshold: u.attackOrder.pkillThreshold,
            }
          : undefined,
        moveOrder: u.moveOrder
          ? create(MoveOrderSchema, {
              waypoints: u.moveOrder.waypoints.map((wp) => ({
                lat: wp.lat,
                lon: wp.lon,
                altMsl: wp.altMsl,
              })),
            })
          : undefined,
        position: create(PositionSchema, {
          lat: u.lat,
          lon: u.lon,
          altMsl: u.altMsl,
          heading: u.heading,
          speed: u.speed,
        }),
        status: create(OperationalStatusSchema, {
          personnelStrength: u.personnelStrength,
          equipmentStrength: u.equipmentStrength,
          combatEffectiveness: u.combatEffectiveness,
          fuelLevelLiters: u.fuelLevelLiters,
          morale: u.morale,
          fatigue: u.fatigue,
          isActive: true,
        }),
      }),
    ),
  });
  return bytesToBase64(toBinary(ScenarioSchema, scenario));
}

function formatCountry(code: string): string {
  return EDITOR_COUNTRY_NAME_BY_CODE[code] ?? code;
}

function getUnitCountry(unit: UnitDraft): string {
  return normalizeCountryCode(unit.teamId);
}

function validatePlacementAccess(
  country: string,
  def: UnitDefinitionDraft | undefined,
  lat: number,
  lon: number,
  relationships: CountryRelationshipDraft[],
): string {
  const owner = normalizeCountryCode(country);
  const host = getCountryCodeForPoint(lat, lon);
  if (!owner || !host || owner === host || !def) {
    return "";
  }
  const relationship = getRelationshipRule(relationships, owner, host);
  const role = (def.employmentRole || "dual_use").trim().toLowerCase();
  if (role === "defensive") {
    if (!relationship.defensivePositioningAllowed) {
      return `${formatCountry(owner)} cannot position defensive assets inside ${formatCountry(host)}.`;
    }
    return "";
  }
  if (!relationship.airspaceStrikeAllowed) {
    return `${formatCountry(owner)} cannot base offensive-capable assets inside ${formatCountry(host)}.`;
  }
  return "";
}

function findTransitViolation(
  ownerCountry: string,
  start: { lat: number; lon: number },
  end: { lat: number; lon: number },
  relationships: CountryRelationshipDraft[],
): string {
  const owner = normalizeCountryCode(ownerCountry);
  if (!owner) {
    return "";
  }
  for (const country of getCountriesAlongSegment(start, end)) {
    if (!country || country === owner) {
      continue;
    }
    const relationship = getRelationshipRule(relationships, owner, country);
    if (!relationship.airspaceTransitAllowed) {
      return `${formatCountry(owner)} cannot transit ${formatCountry(country)} airspace.`;
    }
  }
  return "";
}

function findStrikeViolation(
  unit: UnitDraft,
  target: UnitDraft | undefined,
  relationships: CountryRelationshipDraft[],
): string {
  const owner = getUnitCountry(unit);
  if (!owner || !target) {
    return "";
  }
  const pathCountries = getCountriesAlongRoute(
    { lat: unit.lat, lon: unit.lon },
    [...(unit.moveOrder?.waypoints ?? []), { lat: target.lat, lon: target.lon }],
  );
  for (const country of pathCountries) {
    if (!country || country === owner) {
      continue;
    }
    const relationship = getRelationshipRule(relationships, owner, country);
    if (!relationship.airspaceTransitAllowed) {
      return `${formatCountry(owner)} cannot route the strike through ${formatCountry(country)} airspace.`;
    }
    if (!relationship.airspaceStrikeAllowed) {
      return `${formatCountry(owner)} cannot conduct strike operations in ${formatCountry(country)} airspace.`;
    }
  }
  return "";
}

// ─── CONSTANTS ────────────────────────────────────────────────────────────────

const SIDE_COLOR: Record<string, string> = {
  Blue: "#3b82f6",
  Red: "#ef4444",
  Neutral: "#f59e0b",
};

const WEATHER_STATES = [
  { value: 1, label: "Clear" },
  { value: 2, label: "Overcast" },
  { value: 3, label: "Fog" },
  { value: 4, label: "Rain" },
  { value: 5, label: "Heavy Rain" },
  { value: 6, label: "Snow" },
  { value: 7, label: "Blizzard" },
];

// ─── INLINE EDIT FORM (for placed units) ─────────────────────────────────────
// Only edits Designator, Side, and Speed/Heading — position set by placement.

function InlineEditForm({
  unit,
  units,
  unitDefinitions,
  weaponDefs,
  availableLoadouts,
  isRouteMode,
  onToggleRouteMode,
  onSave,
  onCancel,
}: {
  unit: UnitDraft;
  units: UnitDraft[];
  unitDefinitions: UnitDefinitionDraft[];
  weaponDefs: Map<string, WeaponDefLite>;
  availableLoadouts: { id: string; name: string }[];
  isRouteMode: boolean;
  onToggleRouteMode: () => void;
  onSave: (patch: Partial<UnitDraft>) => void;
  onCancel: () => void;
}) {
  const [displayName, setDisplayName] = useState(unit.displayName);
  const [side, setSide] = useState<UnitDraft["side"]>(unit.side);
  const [teamId, setTeamId] = useState(unit.teamId);
  const [heading, setHeading] = useState(unit.heading);
  const [speed, setSpeed] = useState(unit.speed);
  const [strength, setStrength] = useState(unit.combatEffectiveness);
  const [loadoutConfigurationId, setLoadoutConfigurationId] = useState(unit.loadoutConfigurationId);
  const [engagementBehavior, setEngagementBehavior] = useState(unit.engagementBehavior ?? 1);
  const [engagementPkillThreshold, setEngagementPkillThreshold] = useState(unit.engagementPkillThreshold ?? 0.5);
  const [attackOrderType, setAttackOrderType] = useState(unit.attackOrder?.orderType ?? 0);
  const [targetUnitId, setTargetUnitId] = useState(unit.attackOrder?.targetUnitId ?? "");
  const [desiredEffect, setDesiredEffect] = useState(unit.attackOrder?.desiredEffect ?? 3);
  const [pkillThreshold, setPkillThreshold] = useState(unit.attackOrder?.pkillThreshold ?? 0.7);

  const selectedConfiguration = availableLoadouts.length > 0
    ? unitDefinitions
        .find((candidate) => candidate.id === unit.definitionId)
        ?.weaponConfigurations.find((cfg) => cfg.id === loadoutConfigurationId)
    : undefined;
  const loadedWeapons = (selectedConfiguration?.loadout ?? [])
    .filter((slot) => slot.initialQty > 0 || slot.maxQty > 0)
    .map((slot) => weaponDefs.get(slot.weaponId))
    .filter((weapon): weapon is WeaponDefLite => Boolean(weapon));
  const validTargets = filterValidEditorTargets(unit, units, loadedWeapons, unitDefinitions, side);
  const targetUnit = validTargets.find((candidate) => candidate.id === targetUnitId);
  const targetDef = unitDefinitions.find((candidate) => candidate.id === targetUnit?.definitionId);
  const selectedDefinition = unitDefinitions.find((candidate) => candidate.id === unit.definitionId);
  const countryOptions = Array.from(
    new Set([
      teamId,
      selectedDefinition?.nationOfOrigin ?? "",
      ...(selectedDefinition?.employedBy ?? []),
      ...units.map((candidate) => candidate.teamId),
    ].filter(Boolean)),
  ).sort();
  const loadoutAssessment =
    attackOrderType !== 0 && targetUnitId
      ? assessLoadoutAgainstTarget(selectedConfiguration, targetDef, weaponDefs, desiredEffect)
      : { severity: "none" as const, message: "" };

  useEffect(() => {
    if (!targetUnitId) {
      return;
    }
    if (!validTargets.some((candidate) => candidate.id === targetUnitId)) {
      setTargetUnitId("");
    }
  }, [targetUnitId, validTargets]);

  return (
    <div className="inline-edit-form">
      <div className="field">
        <label className="field-label">Designator</label>
        <input
          className="field-input"
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
        />
      </div>
      <div className="field">
        <label className="field-label">Country</label>
        <select
          className="field-select"
          value={teamId}
          onChange={(e) => setTeamId(e.target.value)}
        >
          <option value="">Select country…</option>
          {countryOptions.map((code) => (
            <option key={code} value={code}>
              {formatCountry(code)}
            </option>
          ))}
        </select>
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
      <div className="field-row">
        <div className="field">
          <label className="field-label">Heading (°)</label>
          <input
            className="field-input"
            type="number"
            min={0}
            max={359}
            value={heading}
            onChange={(e) => setHeading(Number(e.target.value))}
          />
        </div>
        <div className="field">
          <label className="field-label">Speed (m/s)</label>
          <input
            className="field-input"
            type="number"
            min={0}
            step="0.1"
            value={speed}
            onChange={(e) => setSpeed(Number(e.target.value))}
          />
        </div>
      </div>
      <div className="field">
        <label className="field-label">Strength (0–1)</label>
        <input
          className="field-input"
          type="number"
          min={0}
          max={1}
          step="0.01"
          value={strength}
          onChange={(e) => setStrength(Number(e.target.value))}
        />
      </div>
      {availableLoadouts.length > 0 && (
        <div className="field">
          <label className="field-label">Mission Loadout</label>
          <select
            className="field-select"
            value={loadoutConfigurationId}
            onChange={(e) => setLoadoutConfigurationId(e.target.value)}
          >
            {availableLoadouts.map((cfg) => (
              <option key={cfg.id} value={cfg.id}>{cfg.name}</option>
            ))}
          </select>
        </div>
      )}
      <div className="field">
        <label className="field-label">Engagement Behavior</label>
        <select
          className="field-select"
          value={engagementBehavior}
          onChange={(e) => setEngagementBehavior(Number(e.target.value))}
        >
          {ENGAGEMENT_BEHAVIORS.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </select>
      </div>
      <div className="field">
        <label className="field-label">Autonomous Pkill Threshold</label>
        <input
          className="field-input"
          type="number"
          min={0.1}
          max={0.99}
          step={0.05}
          value={engagementPkillThreshold}
          onChange={(e) => setEngagementPkillThreshold(Number(e.target.value))}
        />
      </div>
      <div className="field">
        <label className="field-label">Attack Task</label>
        <select
          className="field-select"
          value={attackOrderType}
          onChange={(e) => setAttackOrderType(Number(e.target.value))}
        >
          {ATTACK_ORDER_TYPES.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </select>
      </div>
      {attackOrderType !== 0 && (
        <>
          <div className="field">
            <label className="field-label">Assigned Target</label>
            <select
              className="field-select"
              value={targetUnitId}
              onChange={(e) => setTargetUnitId(e.target.value)}
            >
              <option value="">Select enemy unit…</option>
              {validTargets.map((candidate) => (
                <option key={candidate.id} value={candidate.id}>
                  {candidate.displayName} · {candidate.side}
                </option>
              ))}
            </select>
          </div>
          <div className="field-row">
            <div className="field">
              <label className="field-label">Desired Effect</label>
              <select
                className="field-select"
                value={desiredEffect}
                onChange={(e) => setDesiredEffect(Number(e.target.value))}
              >
                {DESIRED_EFFECTS.map((option) => (
                  <option key={option.value} value={option.value}>{option.label}</option>
                ))}
              </select>
            </div>
            <div className="field">
              <label className="field-label">Order Pkill Threshold</label>
              <input
                className="field-input"
                type="number"
                min={0.1}
                max={0.99}
                step={0.05}
                value={pkillThreshold}
                onChange={(e) => setPkillThreshold(Number(e.target.value))}
              />
            </div>
          </div>
          {loadoutAssessment.severity !== "none" && loadoutAssessment.message && (
            <div className={`order-validation-note ${loadoutAssessment.severity}`}>
              {loadoutAssessment.message}
            </div>
          )}
        </>
      )}
      <div className="field-row" style={{ marginTop: 8 }}>
        <button className={`btn btn-sm${isRouteMode ? " btn-primary" : ""}`} onClick={onToggleRouteMode}>
          {isRouteMode ? "Finish Route" : "Edit Route"}
        </button>
        <button className="btn btn-sm" onClick={() => onSave({ moveOrder: undefined })}>
          Clear Route
        </button>
        <button
          className="btn btn-success btn-sm"
          disabled={loadoutAssessment.severity === "invalid"}
          onClick={() =>
            onSave({
              displayName,
              side,
              teamId,
              coalitionId: side,
              loadoutConfigurationId,
              engagementBehavior,
              engagementPkillThreshold,
              heading,
              speed,
              combatEffectiveness: strength,
              personnelStrength: strength,
              equipmentStrength: strength,
              attackOrder:
                attackOrderType !== 0 && targetUnitId
                  ? {
                      orderType: attackOrderType,
                      targetUnitId,
                      desiredEffect,
                      pkillThreshold,
                    }
                  : undefined,
            })
          }
        >
          Apply
        </button>
        <button className="btn btn-sm" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </div>
  );
}

// ─── PLACED UNITS LIST ────────────────────────────────────────────────────────

function PlacedUnits({
  routeEditUnitId,
  onToggleRouteMode,
}: {
  routeEditUnitId: string | null;
  onToggleRouteMode: (unitId: string) => void;
}) {
  const units = useEditorStore((s) => s.draft.units);
  const selectedUnitId = useEditorStore((s) => s.selectedUnitId);
  const editingUnitId = useEditorStore((s) => s.editingUnitId);
  const { selectUnit, setEditingUnit, deleteUnit } = useEditorStore();

  if (units.length === 0) {
    return (
      <div className="placed-units-empty">
        Drag units from the palette above onto the map
      </div>
    );
  }

  return (
    <div className="placed-units-list">
      {units.map((u) => (
        <div key={u.id}>
          <div
            className={`unit-list-item${selectedUnitId === u.id ? " selected" : ""}`}
            onClick={() => {
              selectUnit(u.id);
              setEditingUnit(u.id);
            }}
          >
            <span className="unit-dot" style={{ background: SIDE_COLOR[u.side] }} />
            <span className="unit-list-name">{u.displayName}</span>
            {u.attackOrder?.targetUnitId && (
              <span className="unit-list-order">
                ↦ {units.find((candidate) => candidate.id === u.attackOrder?.targetUnitId)?.displayName ?? "Target"}
              </span>
            )}
            {u.moveOrder?.waypoints?.length ? (
              <span className="unit-list-order">Route {u.moveOrder.waypoints.length}</span>
            ) : null}
            <span className="unit-list-side">{u.side}</span>
            <span className="unit-list-actions">
              <button
                className={`btn btn-sm${routeEditUnitId === u.id ? " btn-primary" : ""}`}
                onClick={(e) => {
                  e.stopPropagation();
                  selectUnit(u.id);
                  setEditingUnit(u.id);
                  onToggleRouteMode(u.id);
                }}
              >
                {routeEditUnitId === u.id ? "Routing" : "Route"}
              </button>
              <button
                className="btn btn-sm"
                onClick={(e) => {
                  e.stopPropagation();
                  selectUnit(u.id);
                  setEditingUnit(editingUnitId === u.id ? null : u.id);
                }}
              >
                {editingUnitId === u.id ? "▴" : "Edit"}
              </button>
              <button
                className="btn btn-danger btn-sm"
                onClick={(e) => {
                  e.stopPropagation();
                  deleteUnit(u.id);
                }}
              >
                Del
              </button>
            </span>
          </div>
        </div>
      ))}
    </div>
  );
}

function SelectedUnitPanel({
  routeEditUnitId,
  onToggleRouteMode,
  onSavePatch,
}: {
  routeEditUnitId: string | null;
  onToggleRouteMode: (unitId: string) => void;
  onSavePatch: (unit: UnitDraft, patch: Partial<UnitDraft>) => boolean;
}) {
  const units = useEditorStore((s) => s.draft.units);
  const unitDefinitions = useEditorStore((s) => s.unitDefinitions);
  const editingUnitId = useEditorStore((s) => s.editingUnitId);
  const { updateUnit, setEditingUnit } = useEditorStore();
  const editingUnit =
    editingUnitId && editingUnitId !== "new"
      ? units.find((u) => u.id === editingUnitId)
      : null;
  const [weaponDefs, setWeaponDefs] = useState<Map<string, WeaponDefLite>>(new Map());

  useEffect(() => {
    ListWeaponDefinitions()
      .then((rows) => {
        const next = new Map<string, WeaponDefLite>();
        rows.forEach((row) => {
          next.set(String(row.id ?? ""), {
            id: String(row.id ?? ""),
            domainTargets: Array.isArray(row.domain_targets) ? row.domain_targets.map(Number) : [],
            effectType: Number(row.effect_type ?? 0),
          });
        });
        setWeaponDefs(next);
      })
      .catch(console.error);
  }, []);

  if (!editingUnit) {
    return (
      <div className="selected-unit-empty">
        Select a unit on the map or in the list to issue orders and edit routes.
      </div>
    );
  }

  return (
    <div className="selected-unit-panel">
      <div className="selected-unit-header">
        Commands
        <span className="selected-unit-name">{editingUnit.displayName}</span>
      </div>
      <InlineEditForm
        unit={editingUnit}
        units={units}
        unitDefinitions={unitDefinitions}
        weaponDefs={weaponDefs}
        availableLoadouts={
          unitDefinitions
            .find((def) => def.id === editingUnit.definitionId)
            ?.weaponConfigurations.map((cfg) => ({
              id: cfg.id,
              name: cfg.name || cfg.id,
            })) ?? []
        }
        isRouteMode={routeEditUnitId === editingUnit.id}
        onToggleRouteMode={() => onToggleRouteMode(editingUnit.id)}
        onSave={(patch) => {
          if (!onSavePatch(editingUnit, patch)) {
            return;
          }
          updateUnit(editingUnit.id, patch);
          setEditingUnit(editingUnit.id);
        }}
        onCancel={() => setEditingUnit(null)}
      />
    </div>
  );
}

// ─── METADATA PANEL ───────────────────────────────────────────────────────────

function MetaPanel() {
  const draft = useEditorStore((s) => s.draft);
  const updateMeta = useEditorStore((s) => s.updateMeta);
  const countries = Array.from(
    new Set(draft.units.map((unit) => normalizeCountryCode(unit.teamId)).filter(Boolean)),
  ).sort();

  const startDate = new Date(draft.startTimeUnix * 1000);
  const dateStr = startDate.toISOString().slice(0, 16);

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
        <div className="panel-section-header">Initial Weather</div>
        <div className="field">
          <label className="field-label">Condition</label>
          <select
            className="field-select"
            value={draft.weatherState}
            onChange={(e) => updateMeta({ weatherState: Number(e.target.value) })}
          >
            {WEATHER_STATES.map((w) => (
              <option key={w.value} value={w.value}>{w.label}</option>
            ))}
          </select>
        </div>
        <div className="field-row">
          <div className="field">
            <label className="field-label">Visibility (km)</label>
            <input
              className="field-input"
              type="number"
              min={0}
              max={100}
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
              max={100}
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
            min={-60}
            max={60}
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
                  const relationship = getRelationshipRule(draft.relationships, fromCountry, toCountry);
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

// ─── LOAD MODAL ───────────────────────────────────────────────────────────────

function LoadModal({
  onClose,
  onSelect,
}: {
  onClose: () => void;
  onSelect: (id: string) => void;
}) {
  const [scenarios, setScenarios] = useState<
    { id: string; name: string; author: string; description: string }[]
  >([]);
  const [loading, setLoading] = useState(true);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  useEffect(() => {
    ListScenarios()
      .then((rows) =>
        setScenarios(
          rows.map((r) => ({
            id: String(r.id ?? ""),
            name: String(r.name ?? "Unnamed"),
            author: String(r.author ?? ""),
            description: String(r.description ?? ""),
          })),
        ),
      )
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const handleDelete = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!confirm(`Delete scenario "${scenarios.find((s) => s.id === id)?.name}"?`)) return;
    setDeletingId(id);
    await DeleteScenario(id);
    setScenarios((prev) => prev.filter((s) => s.id !== id));
    setDeletingId(null);
  };

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          Load Scenario
          <button className="modal-close" onClick={onClose}>×</button>
        </div>
        <div className="modal-body">
          {loading && <div className="modal-empty">Loading…</div>}
          {!loading && scenarios.length === 0 && (
            <div className="modal-empty">No saved scenarios.</div>
          )}
          {scenarios.map((s) => (
            <div
              key={s.id}
              className="modal-scenario-item"
              onClick={() => onSelect(s.id)}
            >
              <div>
                <div className="modal-scenario-name">{s.name}</div>
                <div className="modal-scenario-meta">
                  {s.author && `${s.author} · `}{s.description || "No description"}
                </div>
              </div>
              <button
                className="btn btn-danger btn-sm"
                disabled={deletingId === s.id}
                onClick={(e) => handleDelete(s.id, e)}
              >
                Del
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ─── ROOT EDITOR ──────────────────────────────────────────────────────────────

interface ScenarioEditorProps {
  onExit: () => void;
  onPlay: () => void;
}

export default function ScenarioEditor({ onExit, onPlay }: ScenarioEditorProps) {
  const draft = useEditorStore((s) => s.draft);
  const isDirty = useEditorStore((s) => s.isDirty);
  const editingUnitId = useEditorStore((s) => s.editingUnitId);
  const selectedUnitId = useEditorStore((s) => s.selectedUnitId);
  const pendingDrop = useEditorStore((s) => s.pendingDrop);
  const {
    newDraft,
    loadDraft,
    addUnit,
    updateUnit,
    selectUnit,
    setEditingUnit,
    setPendingPosition,
    setPendingDrop,
    markClean,
  } = useEditorStore();

  const [showLoadModal, setShowLoadModal] = useState(false);
  const [showDefManager, setShowDefManager] = useState(false);
  const [statusMsg, setStatusMsg] = useState("");
  const [saving, setSaving] = useState(false);
  const [routeEditUnitId, setRouteEditUnitId] = useState<string | null>(null);
  const unitDefinitions = useEditorStore((s) => s.unitDefinitions);

  const flash = (msg: string) => {
    setStatusMsg(msg);
    setTimeout(() => setStatusMsg(""), 3000);
  };

  const validateUnitPatch = useCallback(
    (unit: UnitDraft, patch: Partial<UnitDraft>) => {
      const nextUnit = { ...unit, ...patch };
      const definition = unitDefinitions.find((def) => def.id === nextUnit.definitionId);
      const placementViolation = validatePlacementAccess(
        nextUnit.teamId,
        definition,
        nextUnit.lat,
        nextUnit.lon,
        draft.relationships,
      );
      if (placementViolation) {
        flash(placementViolation);
        return false;
      }
      if (nextUnit.attackOrder?.targetUnitId) {
        const target = draft.units.find((candidate) => candidate.id === nextUnit.attackOrder?.targetUnitId);
        const strikeViolation = findStrikeViolation(nextUnit, target, draft.relationships);
        if (strikeViolation) {
          flash(strikeViolation);
          return false;
        }
      }
      return true;
    },
    [draft.relationships, draft.units, unitDefinitions],
  );

  // ── Toolbar actions ────────────────────────────────────────────────────────

  const handleSave = async () => {
    setSaving(true);
    try {
      const res = await SaveScenario(draftToProtoB64(draft));
      if (res.success) {
        markClean();
        flash("Saved.");
      } else {
        console.error("[editor] SaveScenario failed:", res.error);
        alert(`Save failed:\n${res.error}`);
      }
    } catch (e) {
      console.error("[editor] SaveScenario threw:", e);
      alert(`Save error:\n${e}`);
    } finally {
      setSaving(false);
    }
  };

  const handlePlay = async () => {
    setSaving(true);
    try {
      const b64 = draftToProtoB64(draft);
      const save = await SaveScenario(b64);
      if (!save.success) {
        console.error("[editor] SaveScenario failed:", save.error);
        alert(`Save failed:\n${save.error}`);
        return;
      }
      const load = await LoadScenarioFromProto(b64);
      if (!load.success) {
        console.error("[editor] LoadScenarioFromProto failed:", load.error);
        alert(`Load failed:\n${load.error}`);
        return;
      }
      markClean();
      onPlay();
    } catch (e) {
      console.error("[editor] handlePlay threw:", e);
      alert(`Play error:\n${e}`);
    } finally {
      setSaving(false);
    }
  };

  const handleLoadSelect = async (id: string) => {
    setShowLoadModal(false);
    try {
      const b64 = await GetScenario(id);
      const scen = fromBinary(ScenarioSchema, base64ToBytes(b64));
      loadDraft({
        id: scen.id,
        name: scen.name,
        description: scen.description,
        classification: scen.classification,
        author: scen.author,
        startTimeUnix: scen.startTimeUnix,
        version: scen.version,
        tickRateHz: scen.settings?.tickRateHz ?? 10,
        timeScale: scen.settings?.timeScale ?? 1.0,
        weatherState: scen.map?.initialWeather?.state ?? 1,
        visibilityKm: scen.map?.initialWeather?.visibilityKm ?? 40,
        windSpeedMps: scen.map?.initialWeather?.windSpeedMps ?? 5,
        temperatureC: scen.map?.initialWeather?.temperatureC ?? 20,
        relationships: scen.relationships.map((rel) => ({
          fromCountry: rel.fromCountry,
          toCountry: rel.toCountry,
          shareIntel: rel.shareIntel,
          airspaceTransitAllowed: rel.airspaceTransitAllowed,
          airspaceStrikeAllowed: rel.airspaceStrikeAllowed,
          defensivePositioningAllowed: rel.defensivePositioningAllowed,
        })),
        units: scen.units.map((u) => ({
          id: u.id,
          displayName: u.displayName,
          fullName: u.fullName,
          side: (u.side as UnitDraft["side"]) || "Blue",
          teamId: u.teamId || "",
          coalitionId: u.coalitionId || ((u.side as UnitDraft["side"]) || "Blue"),
          definitionId: u.definitionId,
          parentUnitId: u.parentUnitId || undefined,
          loadoutConfigurationId: u.loadoutConfigurationId,
          natoSymbolSidc: u.natoSymbolSidc,
          damageState: u.damageState ?? 1,
          engagementBehavior: u.engagementBehavior ?? 1,
          engagementPkillThreshold: u.engagementPkillThreshold ?? 0.5,
          attackOrder: u.attackOrder
            ? {
                orderType: u.attackOrder.orderType,
                targetUnitId: u.attackOrder.targetUnitId,
                desiredEffect: u.attackOrder.desiredEffect,
                pkillThreshold: u.attackOrder.pkillThreshold,
              }
            : undefined,
          moveOrder: u.moveOrder?.waypoints?.length
            ? {
                waypoints: u.moveOrder.waypoints.map((wp) => ({
                  lat: wp.lat,
                  lon: wp.lon,
                  altMsl: wp.altMsl,
                })),
              }
            : undefined,
          lat: u.position?.lat ?? 0,
          lon: u.position?.lon ?? 0,
          altMsl: u.position?.altMsl ?? 0,
          heading: u.position?.heading ?? 0,
          speed: u.position?.speed ?? 0,
          personnelStrength: u.status?.personnelStrength ?? 1,
          equipmentStrength: u.status?.equipmentStrength ?? 1,
          combatEffectiveness: u.status?.combatEffectiveness ?? 1,
          fuelLevelLiters: u.status?.fuelLevelLiters ?? 10000,
          morale: u.status?.morale ?? 1,
          fatigue: u.status?.fatigue ?? 0,
        })),
      });
      flash(`Loaded: ${scen.name}`);
    } catch (e) {
      flash(`Load error: ${e}`);
    }
  };

  // ── Globe callbacks ────────────────────────────────────────────────────────

  const handleMapClick = useCallback(
    (lat: number, lon: number) => {
      if (routeEditUnitId) {
        const unit = draft.units.find((candidate) => candidate.id === routeEditUnitId);
        if (!unit) {
          setRouteEditUnitId(null);
          return;
        }
        const start = unit.moveOrder?.waypoints?.length
          ? unit.moveOrder.waypoints[unit.moveOrder.waypoints.length - 1]
          : { lat: unit.lat, lon: unit.lon };
        const transitViolation = findTransitViolation(unit.teamId, start, { lat, lon }, draft.relationships);
        if (transitViolation) {
          flash(transitViolation);
          return;
        }
        const nextWaypoints = [...(unit.moveOrder?.waypoints ?? []), { lat, lon, altMsl: unit.altMsl }];
        updateUnit(routeEditUnitId, { moveOrder: { waypoints: nextWaypoints } });
        selectUnit(routeEditUnitId);
        setEditingUnit(routeEditUnitId);
        flash(`Waypoint ${nextWaypoints.length} added to ${unit.displayName}`);
        return;
      }
      setPendingPosition({ lat, lon });
      setEditingUnit("new");
    },
    [draft.relationships, draft.units, routeEditUnitId, selectUnit, setPendingPosition, setEditingUnit, updateUnit],
  );

  const handleUnitClick = useCallback((unitId: string) => {
    useEditorStore.getState().selectUnit(unitId);
    useEditorStore.getState().setEditingUnit(unitId);
  }, []);

  const toggleRouteMode = useCallback((unitId: string) => {
    setRouteEditUnitId((current) => (current === unitId ? null : unitId));
  }, []);

  useEffect(() => {
    if (routeEditUnitId && routeEditUnitId !== selectedUnitId) {
      setRouteEditUnitId(null);
    }
  }, [routeEditUnitId, selectedUnitId]);

  const handleUnitDrop = useCallback(
    (lat: number, lon: number, payload: DragPayload) => {
      setPendingDrop({ lat, lon, ...payload });
    },
    [setPendingDrop],
  );

  return (
    <div className="editor-shell">
      {/* ── Toolbar ── */}
      <div className="editor-toolbar">
        <button className="btn" onClick={onExit}>← Sim</button>
        <span className="editor-toolbar-title">Scenario Editor</span>
        {isDirty && <span className="editor-dirty-badge">● UNSAVED</span>}
        <span className="editor-toolbar-spacer" />
        <button className="btn" onClick={() => newDraft()}>New</button>
        <button className="btn" onClick={() => setShowLoadModal(true)}>Open</button>
        <button className="btn" onClick={() => setShowDefManager(true)}>Definitions</button>
        <button className="btn btn-primary" onClick={handleSave} disabled={saving}>Save</button>
        <button className="btn btn-success" onClick={handlePlay} disabled={saving}>▶ Play</button>
      </div>

      {/* ── Three-panel body ── */}
      <div className="editor-body">
        {/* Left: metadata */}
        <div className="editor-panel editor-panel-left">
          <MetaPanel />
        </div>

        {/* Center: globe */}
        <div className="editor-panel-center">
          <EditorGlobe
            onMapClick={handleMapClick}
            onUnitClick={handleUnitClick}
            onUnitDrop={handleUnitDrop}
            placementMode={editingUnitId === "new"}
          />
        </div>

        {/* Right: palette + placed units */}
        <div className="editor-panel editor-panel-right">
          <UnitPalette />
          <div className="palette-divider" />
          <SelectedUnitPanel
            routeEditUnitId={routeEditUnitId}
            onToggleRouteMode={toggleRouteMode}
            onSavePatch={validateUnitPatch}
          />
          <div className="palette-divider" />
          <div className="placed-units-section">
            <div className="placed-units-header">
              Placed Units
              <span className="placed-units-count">{draft.units.length}</span>
            </div>
            <PlacedUnits routeEditUnitId={routeEditUnitId} onToggleRouteMode={toggleRouteMode} />
          </div>
        </div>
      </div>

      {/* ── Status bar ── */}
      <div className="editor-status">
        <span className="status-item">
          Units: <span className="status-value">{draft.units.length}</span>
        </span>
        <span className="status-item">
          ID: <span className="status-value">{draft.id.slice(0, 8)}…</span>
        </span>
        {routeEditUnitId && (
          <span className="status-item">
            Route Mode: <span className="status-value">Click the map to add waypoints</span>
          </span>
        )}
        {statusMsg && <span className="status-value" style={{ color: "#22c55e" }}>{statusMsg}</span>}
      </div>

      {/* ── Drop confirm dialog ── */}
      {pendingDrop && (
        <DropConfirmDialog
          drop={pendingDrop}
          onConfirm={(unit) => {
            const definition = unitDefinitions.find((def) => def.id === unit.definitionId);
            const violation = validatePlacementAccess(
              unit.teamId,
              definition,
              unit.lat,
              unit.lon,
              draft.relationships,
            );
            if (violation) {
              flash(violation);
              return;
            }
            addUnit(unit);
            setPendingDrop(null);
          }}
          onCancel={() => setPendingDrop(null)}
        />
      )}

      {showLoadModal && (
        <LoadModal
          onClose={() => setShowLoadModal(false)}
          onSelect={handleLoadSelect}
        />
      )}

      {showDefManager && (
        <UnitDefinitionManager onClose={() => setShowDefManager(false)} />
      )}
    </div>
  );
}
