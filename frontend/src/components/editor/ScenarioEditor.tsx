/**
 * ScenarioEditor.tsx
 *
 * Three-panel scenario editor:
 *   Left  — Scenario metadata, simulation settings, weather
 *   Center — CesiumJS globe (click to place units)
 *   Right  — Unit list + unit add/edit form
 *
 * Data flow:
 *   editorStore (draft) ──► EditorGlobe (render)
 *   editorStore (draft) ──► serialize to proto binary ──► Go SaveScenario / LoadScenarioFromProto
 */

import { useCallback, useState } from "react";
import { create, toBinary } from "@bufbuild/protobuf";
import { ScenarioSchema } from "@proto/engine/v1/scenario_pb";
import { UnitSchema } from "@proto/engine/v1/unit_pb";
import { PositionSchema } from "@proto/engine/v1/common_pb";
import { OperationalStatusSchema } from "@proto/engine/v1/status_pb";
import {
  SaveScenario,
  LoadScenarioFromProto,
  ListScenarios,
  GetScenario,
  DeleteScenario,
} from "../../../wailsjs/go/main/App";
import {
  useEditorStore,
  blankUnit,
  type UnitDraft,
  type ScenarioDraft,
} from "../../store/editorStore";
import EditorGlobe from "./EditorGlobe";
import "./editor.css";

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
    units: draft.units.map((u) =>
      create(UnitSchema, {
        id: u.id,
        displayName: u.displayName,
        fullName: u.fullName,
        side: u.side,
        domain: u.domain,
        type: u.unitType,
        echelon: u.echelon,
        natoSymbolSidc: u.natoSymbolSidc,
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

// ─── SIDE COLORS ─────────────────────────────────────────────────────────────

const SIDE_COLOR: Record<string, string> = {
  Blue: "#3b82f6",
  Red: "#ef4444",
  Neutral: "#f59e0b",
};

// ─── UNIT FORM ────────────────────────────────────────────────────────────────

const DOMAINS = [
  { value: 1, label: "Land" },
  { value: 2, label: "Air" },
  { value: 3, label: "Sea" },
  { value: 4, label: "Subsurface" },
];

const UNIT_TYPES_BY_DOMAIN: Record<number, { value: number; label: string }[]> = {
  1: [
    { value: 1, label: "Armor" },
    { value: 2, label: "Mech Infantry" },
    { value: 3, label: "Light Infantry" },
    { value: 4, label: "Airborne" },
    { value: 7, label: "Special Forces" },
    { value: 8, label: "Cavalry" },
    { value: 10, label: "SP Artillery" },
    { value: 11, label: "Towed Artillery" },
  ],
  2: [
    { value: 32, label: "Fighter" },
    { value: 33, label: "Multirole" },
    { value: 34, label: "Attack Aircraft" },
    { value: 36, label: "Transport Aircraft" },
    { value: 39, label: "UAV Recon" },
  ],
  3: [
    { value: 50, label: "Aircraft Carrier" },
    { value: 51, label: "Destroyer" },
    { value: 52, label: "Frigate" },
    { value: 53, label: "Corvette" },
    { value: 54, label: "Patrol Boat" },
    { value: 57, label: "Attack Submarine" },
  ],
  4: [
    { value: 57, label: "Attack Submarine" },
  ],
};

const ECHELONS = [
  { value: 1, label: "Fireteam" },
  { value: 2, label: "Squad" },
  { value: 3, label: "Section" },
  { value: 4, label: "Platoon" },
  { value: 5, label: "Company" },
  { value: 6, label: "Battalion" },
  { value: 7, label: "Brigade" },
  { value: 8, label: "Division" },
  { value: 9, label: "Corps" },
  { value: 10, label: "Army" },
];

function UnitForm({
  unit,
  onSave,
  onCancel,
}: {
  unit: UnitDraft;
  onSave: (u: UnitDraft) => void;
  onCancel: () => void;
}) {
  const [form, setForm] = useState<UnitDraft>(unit);
  const set = (patch: Partial<UnitDraft>) => setForm((f) => ({ ...f, ...patch }));

  const typeOptions = UNIT_TYPES_BY_DOMAIN[form.domain] ?? [];

  return (
    <div className="panel-section">
      <div className="panel-section-header">
        {unit.id === (form as UnitDraft).id && form.displayName === "UNIT-1" &&
        form.fullName === ""
          ? "Add Unit"
          : "Edit Unit"}
      </div>

      <div className="field">
        <label className="field-label">Designator</label>
        <input
          className="field-input"
          value={form.displayName}
          onChange={(e) => set({ displayName: e.target.value })}
          placeholder="e.g. DDG-51"
        />
      </div>

      <div className="field">
        <label className="field-label">Full Name</label>
        <input
          className="field-input"
          value={form.fullName}
          onChange={(e) => set({ fullName: e.target.value })}
          placeholder="e.g. USS Arleigh Burke"
        />
      </div>

      <div className="field-row">
        <div className="field">
          <label className="field-label">Side</label>
          <select
            className="field-select"
            value={form.side}
            onChange={(e) => set({ side: e.target.value as UnitDraft["side"] })}
          >
            <option value="Blue">Blue</option>
            <option value="Red">Red</option>
            <option value="Neutral">Neutral</option>
          </select>
        </div>
        <div className="field">
          <label className="field-label">Domain</label>
          <select
            className="field-select"
            value={form.domain}
            onChange={(e) => {
              const d = Number(e.target.value);
              const firstType = (UNIT_TYPES_BY_DOMAIN[d] ?? [])[0]?.value ?? 0;
              set({ domain: d, unitType: firstType });
            }}
          >
            {DOMAINS.map((d) => (
              <option key={d.value} value={d.value}>{d.label}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="field-row">
        <div className="field">
          <label className="field-label">Type</label>
          <select
            className="field-select"
            value={form.unitType}
            onChange={(e) => set({ unitType: Number(e.target.value) })}
          >
            {typeOptions.map((t) => (
              <option key={t.value} value={t.value}>{t.label}</option>
            ))}
          </select>
        </div>
        <div className="field">
          <label className="field-label">Echelon</label>
          <select
            className="field-select"
            value={form.echelon}
            onChange={(e) => set({ echelon: Number(e.target.value) })}
          >
            {ECHELONS.map((e) => (
              <option key={e.value} value={e.value}>{e.label}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="field-row">
        <div className="field">
          <label className="field-label">Lat</label>
          <input
            className="field-input"
            type="number"
            step="0.0001"
            value={form.lat}
            onChange={(e) => set({ lat: Number(e.target.value) })}
          />
        </div>
        <div className="field">
          <label className="field-label">Lon</label>
          <input
            className="field-input"
            type="number"
            step="0.0001"
            value={form.lon}
            onChange={(e) => set({ lon: Number(e.target.value) })}
          />
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
            value={form.heading}
            onChange={(e) => set({ heading: Number(e.target.value) })}
          />
        </div>
        <div className="field">
          <label className="field-label">Speed (m/s)</label>
          <input
            className="field-input"
            type="number"
            min={0}
            step="0.1"
            value={form.speed}
            onChange={(e) => set({ speed: Number(e.target.value) })}
          />
        </div>
      </div>

      <div className="field-row">
        <div className="field">
          <label className="field-label">Strength (0–1)</label>
          <input
            className="field-input"
            type="number"
            min={0}
            max={1}
            step="0.01"
            value={form.combatEffectiveness}
            onChange={(e) => set({ combatEffectiveness: Number(e.target.value), personnelStrength: Number(e.target.value), equipmentStrength: Number(e.target.value) })}
          />
        </div>
        <div className="field">
          <label className="field-label">Morale (0–1)</label>
          <input
            className="field-input"
            type="number"
            min={0}
            max={1}
            step="0.01"
            value={form.morale}
            onChange={(e) => set({ morale: Number(e.target.value) })}
          />
        </div>
      </div>

      <div className="field-row" style={{ marginTop: 10 }}>
        <button className="btn btn-success" onClick={() => onSave(form)}>
          Save Unit
        </button>
        <button className="btn" onClick={onCancel}>
          Cancel
        </button>
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

  // Load list on mount
  useState(() => {
    ListScenarios()
      .then((rows) => {
        setScenarios(
          rows.map((r) => ({
            id: String((r.id as { id?: unknown })?.id ?? r.id ?? ""),
            name: String(r.name ?? "Unnamed"),
            author: String(r.author ?? ""),
            description: String(r.description ?? ""),
          })),
        );
      })
      .catch(console.error)
      .finally(() => setLoading(false));
  });

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
            <div className="modal-empty">No saved scenarios found.</div>
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

// ─── METADATA PANEL ───────────────────────────────────────────────────────────

const WEATHER_STATES = [
  { value: 1, label: "Clear" },
  { value: 2, label: "Overcast" },
  { value: 3, label: "Fog" },
  { value: 4, label: "Rain" },
  { value: 5, label: "Heavy Rain" },
  { value: 6, label: "Snow" },
  { value: 7, label: "Blizzard" },
];

function MetaPanel() {
  const draft = useEditorStore((s) => s.draft);
  const updateMeta = useEditorStore((s) => s.updateMeta);

  const startDate = new Date(draft.startTimeUnix * 1000);
  const dateStr = startDate.toISOString().slice(0, 16); // "YYYY-MM-DDTHH:MM"

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
    </div>
  );
}

// ─── UNIT PANEL ───────────────────────────────────────────────────────────────

function UnitsPanel() {
  const units = useEditorStore((s) => s.draft.units);
  const selectedUnitId = useEditorStore((s) => s.selectedUnitId);
  const editingUnitId = useEditorStore((s) => s.editingUnitId);
  const pendingPosition = useEditorStore((s) => s.pendingPosition);
  const { selectUnit, setEditingUnit, addUnit, updateUnit, deleteUnit } = useEditorStore();

  const editingUnit =
    editingUnitId === "new"
      ? blankUnit(pendingPosition?.lat, pendingPosition?.lon)
      : units.find((u) => u.id === editingUnitId) ?? null;

  const handleSave = (u: UnitDraft) => {
    if (editingUnitId === "new") {
      addUnit(u);
    } else {
      updateUnit(u.id, u);
      setEditingUnit(null);
    }
  };

  return (
    <div className="panel-scroll" style={{ display: "flex", flexDirection: "column" }}>
      <div className="panel-section" style={{ flexShrink: 0 }}>
        <div
          className="panel-section-header"
          style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}
        >
          <span>Units ({units.length})</span>
          <button
            className="btn btn-primary btn-sm"
            onClick={() => setEditingUnit("new")}
            disabled={editingUnitId !== null}
          >
            + Add
          </button>
        </div>

        <div className="unit-list">
          {units.length === 0 && (
            <div className="unit-list-empty">
              Click map or [+ Add] to place units
            </div>
          )}
          {units.map((u) => (
            <div
              key={u.id}
              className={`unit-list-item${selectedUnitId === u.id ? " selected" : ""}`}
              onClick={() => {
                selectUnit(u.id);
                if (editingUnitId !== u.id) setEditingUnit(null);
              }}
            >
              <span className="unit-dot" style={{ background: SIDE_COLOR[u.side] }} />
              <span className="unit-list-name">{u.displayName}</span>
              <span className="unit-list-side">{u.side}</span>
              <span className="unit-list-actions">
                <button
                  className="btn btn-sm"
                  onClick={(e) => { e.stopPropagation(); setEditingUnit(u.id); selectUnit(u.id); }}
                >
                  Edit
                </button>
                <button
                  className="btn btn-danger btn-sm"
                  onClick={(e) => { e.stopPropagation(); deleteUnit(u.id); }}
                >
                  Del
                </button>
              </span>
            </div>
          ))}
        </div>
      </div>

      {editingUnit && (
        <UnitForm
          unit={editingUnit}
          onSave={handleSave}
          onCancel={() => setEditingUnit(null)}
        />
      )}
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
  const { newDraft, loadDraft, setEditingUnit, setPendingPosition, markClean } = useEditorStore();

  const [showLoadModal, setShowLoadModal] = useState(false);
  const [statusMsg, setStatusMsg] = useState("");
  const [saving, setSaving] = useState(false);

  const flash = (msg: string) => {
    setStatusMsg(msg);
    setTimeout(() => setStatusMsg(""), 3000);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const b64 = draftToProtoB64(draft);
      const res = await SaveScenario(b64);
      if (res.success) {
        markClean();
        flash("Saved.");
      } else {
        flash(`Error: ${res.error}`);
      }
    } catch (e) {
      flash(`Error: ${e}`);
    } finally {
      setSaving(false);
    }
  };

  const handlePlay = async () => {
    setSaving(true);
    try {
      const b64 = draftToProtoB64(draft);
      const saveRes = await SaveScenario(b64);
      if (!saveRes.success) { flash(`Save error: ${saveRes.error}`); return; }
      const loadRes = await LoadScenarioFromProto(b64);
      if (!loadRes.success) { flash(`Load error: ${loadRes.error}`); return; }
      markClean();
      onPlay();
    } catch (e) {
      flash(`Error: ${e}`);
    } finally {
      setSaving(false);
    }
  };

  const handleLoadSelect = async (id: string) => {
    setShowLoadModal(false);
    try {
      const b64 = await GetScenario(id);
      // We can't easily deserialize on the frontend, so we'll parse from DB record fields.
      // For now, just flag as "loaded" and rely on the backend sending back a snapshot.
      // Actually we need to deserialize the proto — use fromBinary.
      const { fromBinary } = await import("@bufbuild/protobuf");
      const { ScenarioSchema: SS } = await import("@proto/engine/v1/scenario_pb");
      const scen = fromBinary(SS, base64ToBytes(b64));
      const loaded: ScenarioDraft = {
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
        units: scen.units.map((u) => ({
          id: u.id,
          displayName: u.displayName,
          fullName: u.fullName,
          side: (u.side as "Blue" | "Red" | "Neutral") || "Blue",
          domain: u.domain,
          unitType: u.type,
          echelon: u.echelon,
          natoSymbolSidc: u.natoSymbolSidc,
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
      };
      loadDraft(loaded);
      flash(`Loaded: ${loaded.name}`);
    } catch (e) {
      flash(`Load error: ${e}`);
    }
  };

  const handleMapClick = useCallback(
    (lat: number, lon: number) => {
      setPendingPosition({ lat, lon });
      setEditingUnit("new");
    },
    [setPendingPosition, setEditingUnit],
  );

  const handleUnitClick = useCallback(
    (unitId: string) => {
      useEditorStore.getState().selectUnit(unitId);
      useEditorStore.getState().setEditingUnit(unitId);
    },
    [],
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
        <button className="btn btn-primary" onClick={handleSave} disabled={saving}>
          Save
        </button>
        <button className="btn btn-success" onClick={handlePlay} disabled={saving}>
          ▶ Play
        </button>
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
            placementMode={editingUnitId === "new"}
          />
          {editingUnitId === "new" && (
            <div className="placement-hint">Click globe to place unit</div>
          )}
        </div>

        {/* Right: units */}
        <div className="editor-panel editor-panel-right">
          <UnitsPanel />
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
        {statusMsg && <span className="status-value" style={{ color: "#22c55e" }}>{statusMsg}</span>}
      </div>

      {showLoadModal && (
        <LoadModal
          onClose={() => setShowLoadModal(false)}
          onSelect={handleLoadSelect}
        />
      )}
    </div>
  );
}
