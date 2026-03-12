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
 *   3. DropConfirmDialog asks for Designator + Side + Echelon
 *   4. Confirm → addUnit to editorStore
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
  type UnitDraft,
  type ScenarioDraft,
} from "../../store/editorStore";
import EditorGlobe from "./EditorGlobe";
import UnitPalette, { type DragPayload } from "./UnitPalette";
import DropConfirmDialog from "./DropConfirmDialog";
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
  onSave,
  onCancel,
}: {
  unit: UnitDraft;
  onSave: (patch: Partial<UnitDraft>) => void;
  onCancel: () => void;
}) {
  const [displayName, setDisplayName] = useState(unit.displayName);
  const [side, setSide] = useState<UnitDraft["side"]>(unit.side);
  const [heading, setHeading] = useState(unit.heading);
  const [speed, setSpeed] = useState(unit.speed);
  const [strength, setStrength] = useState(unit.combatEffectiveness);

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
      <div className="field-row" style={{ marginTop: 8 }}>
        <button
          className="btn btn-success btn-sm"
          onClick={() =>
            onSave({
              displayName,
              side,
              heading,
              speed,
              combatEffectiveness: strength,
              personnelStrength: strength,
              equipmentStrength: strength,
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

function PlacedUnits() {
  const units = useEditorStore((s) => s.draft.units);
  const selectedUnitId = useEditorStore((s) => s.selectedUnitId);
  const editingUnitId = useEditorStore((s) => s.editingUnitId);
  const { selectUnit, setEditingUnit, updateUnit, deleteUnit } = useEditorStore();

  const editingUnit =
    editingUnitId && editingUnitId !== "new"
      ? units.find((u) => u.id === editingUnitId)
      : null;

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
            onClick={() => selectUnit(u.id)}
          >
            <span className="unit-dot" style={{ background: SIDE_COLOR[u.side] }} />
            <span className="unit-list-name">{u.displayName}</span>
            <span className="unit-list-side">{u.side}</span>
            <span className="unit-list-actions">
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
          {editingUnit?.id === u.id && (
            <InlineEditForm
              unit={editingUnit}
              onSave={(patch) => {
                updateUnit(u.id, patch);
                setEditingUnit(null);
              }}
              onCancel={() => setEditingUnit(null)}
            />
          )}
        </div>
      ))}
    </div>
  );
}

// ─── METADATA PANEL ───────────────────────────────────────────────────────────

function MetaPanel() {
  const draft = useEditorStore((s) => s.draft);
  const updateMeta = useEditorStore((s) => s.updateMeta);

  const startDate = new Date(draft.startTimeUnix * 1000);
  const dateStr = startDate.toISOString().slice(0, 16);

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

  useState(() => {
    ListScenarios()
      .then((rows) =>
        setScenarios(
          rows.map((r) => ({
            id: String((r.id as { id?: unknown })?.id ?? r.id ?? ""),
            name: String(r.name ?? "Unnamed"),
            author: String(r.author ?? ""),
            description: String(r.description ?? ""),
          })),
        ),
      )
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
  const pendingDrop = useEditorStore((s) => s.pendingDrop);
  const {
    newDraft,
    loadDraft,
    addUnit,
    setEditingUnit,
    setPendingPosition,
    setPendingDrop,
    markClean,
  } = useEditorStore();

  const [showLoadModal, setShowLoadModal] = useState(false);
  const [statusMsg, setStatusMsg] = useState("");
  const [saving, setSaving] = useState(false);

  const flash = (msg: string) => {
    setStatusMsg(msg);
    setTimeout(() => setStatusMsg(""), 3000);
  };

  // ── Toolbar actions ────────────────────────────────────────────────────────

  const handleSave = async () => {
    setSaving(true);
    try {
      const res = await SaveScenario(draftToProtoB64(draft));
      if (res.success) { markClean(); flash("Saved."); }
      else flash(`Error: ${res.error}`);
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
      const save = await SaveScenario(b64);
      if (!save.success) { flash(`Save error: ${save.error}`); return; }
      const load = await LoadScenarioFromProto(b64);
      if (!load.success) { flash(`Load error: ${load.error}`); return; }
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
      const { fromBinary } = await import("@bufbuild/protobuf");
      const { ScenarioSchema: SS } = await import("@proto/engine/v1/scenario_pb");
      const scen = fromBinary(SS, base64ToBytes(b64));
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
        units: scen.units.map((u) => ({
          id: u.id,
          displayName: u.displayName,
          fullName: u.fullName,
          side: (u.side as UnitDraft["side"]) || "Blue",
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
      });
      flash(`Loaded: ${scen.name}`);
    } catch (e) {
      flash(`Load error: ${e}`);
    }
  };

  // ── Globe callbacks ────────────────────────────────────────────────────────

  const handleMapClick = useCallback(
    (lat: number, lon: number) => {
      setPendingPosition({ lat, lon });
      setEditingUnit("new");
    },
    [setPendingPosition, setEditingUnit],
  );

  const handleUnitClick = useCallback((unitId: string) => {
    useEditorStore.getState().selectUnit(unitId);
  }, []);

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
          <div className="placed-units-section">
            <div className="placed-units-header">
              Placed Units
              <span className="placed-units-count">{draft.units.length}</span>
            </div>
            <PlacedUnits />
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
        {statusMsg && <span className="status-value" style={{ color: "#22c55e" }}>{statusMsg}</span>}
      </div>

      {/* ── Drop confirm dialog ── */}
      {pendingDrop && (
        <DropConfirmDialog
          drop={pendingDrop}
          onConfirm={(unit) => {
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
    </div>
  );
}
