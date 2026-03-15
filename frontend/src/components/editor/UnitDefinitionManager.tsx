/**
 * UnitDefinitionManager.tsx
 *
 * Full-screen modal for creating, editing, and deleting unit definitions.
 * Shows a list on the left and a form on the right.
 */

import { useState } from "react";
import { useEditorStore, type UnitDefinitionDraft } from "../../store/editorStore";
import { SaveUnitDefinition, DeleteUnitDefinition, ListUnitDefinitions } from "../../../wailsjs/go/main/App";
import UnitTypeIcon from "../UnitTypeIcon";

// ─── LABEL MAPS ───────────────────────────────────────────────────────────────

const DOMAIN_LABELS: Record<number, string> = {
  1: "Land", 2: "Air", 3: "Sea", 4: "Subsurface",
};

const FORM_LABELS: Record<number, string> = {
  10: "Manned Fixed Wing", 11: "Manned Rotary Wing",
  12: "Unmanned Fixed Wing", 13: "Unmanned Rotary Wing",
  20: "Manned Surface Ship", 21: "Manned Submarine",
  22: "Unmanned Surface", 23: "Unmanned Subsurface",
  30: "Dismounted Infantry", 31: "Wheeled Vehicle",
  32: "Tracked Vehicle", 33: "Towed System", 34: "Static Installation",
};

const FORM_BY_DOMAIN: Record<number, number[]> = {
  1: [30, 31, 32, 33, 34],
  2: [10, 11, 12, 13],
  3: [20, 21, 22, 23],
  4: [21, 23],
};

const GENERAL_TYPE_LABELS: Record<number, string> = {
  10: "Fighter", 11: "Multirole", 12: "Attack Aircraft", 13: "Bomber",
  14: "Transport Aircraft", 15: "Maritime Patrol", 16: "AEW", 17: "Tanker", 18: "ISR Fixed Wing",
  20: "Attack Helicopter", 21: "Utility Helicopter", 22: "Naval Helicopter",
  30: "ISR UAV", 31: "UCAV", 32: "Loitering Munition",
  40: "Aircraft Carrier", 41: "Cruiser", 42: "Destroyer", 43: "Frigate",
  44: "Corvette", 45: "Patrol Vessel", 46: "Amphibious Assault", 47: "Mine Warfare",
  50: "Attack Submarine", 51: "Ballistic Missile Sub", 52: "Cruise Missile Sub",
  60: "Main Battle Tank", 61: "Infantry Fighting Vehicle", 62: "Armored Personnel Carrier",
  63: "Reconnaissance Vehicle",
  70: "Self-Propelled Artillery", 71: "Towed Artillery", 72: "Rocket Artillery", 73: "Air Defense",
  80: "Special Forces", 81: "Light Infantry", 82: "Airborne Infantry", 83: "Marine Infantry",
  90: "Engineer", 91: "Logistics", 92: "Medical", 93: "Command", 94: "Electronic Warfare",
};

const GENERAL_TYPE_BY_DOMAIN: Record<number, number[]> = {
  1: [60, 61, 62, 63, 70, 71, 72, 73, 80, 81, 82, 83, 90, 91, 92, 93, 94],
  2: [10, 11, 12, 13, 14, 15, 16, 17, 18, 20, 21, 22, 30, 31, 32],
  3: [40, 41, 42, 43, 44, 45, 46, 47, 50, 51, 52],
  4: [50, 51, 52],
};

const DOMAIN_COLORS: Record<number, string> = {
  1: "#4ade80", 2: "#94a3b8", 3: "#3b82f6", 4: "#818cf8",
};

// ─── BLANK DEF ────────────────────────────────────────────────────────────────

function blankDef(): UnitDefinitionDraft {
  return {
    id: "", name: "", description: "",
    domain: 1, form: 30, generalType: 60, specificType: "", shortName: "",
    nationOfOrigin: "", serviceEntryYear: 2000,
    baseStrength: 0.8, combatRangeM: 1000, accuracy: 0.75,
    maxSpeedMps: 10, cruiseSpeedMps: 7, maxRangeKm: 500,
    survivability: 0.6, detectionRangeM: 5000, radarCrossSectionM2: 5,
    fuelCapacityLiters: 500, fuelBurnRateLph: 100,
  };
}

function rowToDef(r: Record<string, unknown>): UnitDefinitionDraft {
  const num = (k: string) => Number(r[k] ?? 0);
  const str = (k: string) => String(r[k] ?? "");
  const firstNonEmpty = (...values: string[]) => values.find((v) => v.trim().length > 0) ?? "";
  return {
    id: str("id"), name: str("name"), description: str("description"),
    domain: num("domain"), form: num("form"), generalType: num("general_type"),
    specificType: str("specific_type"), shortName: firstNonEmpty(str("short_name"), str("specific_type"), str("name")), nationOfOrigin: str("nation_of_origin"),
    serviceEntryYear: num("service_entry_year"),
    baseStrength: num("base_strength"), combatRangeM: num("combat_range_m"), accuracy: num("accuracy"),
    maxSpeedMps: num("max_speed_mps"), cruiseSpeedMps: num("cruise_speed_mps"), maxRangeKm: num("max_range_km"),
    survivability: num("survivability"), detectionRangeM: num("detection_range_m"), radarCrossSectionM2: num("radar_cross_section_m2"),
    fuelCapacityLiters: num("fuel_capacity_liters"), fuelBurnRateLph: num("fuel_burn_rate_lph"),
  };
}

// ─── DEFINITION FORM ──────────────────────────────────────────────────────────

function DefinitionForm({
  initial,
  onSave,
  onCancel,
  onDelete,
  isNew,
}: {
  initial: UnitDefinitionDraft;
  onSave: (def: UnitDefinitionDraft) => Promise<void>;
  onCancel: () => void;
  onDelete?: () => Promise<void>;
  isNew: boolean;
}) {
  const [def, setDef] = useState<UnitDefinitionDraft>(initial);
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState("");

  const patch = (update: Partial<UnitDefinitionDraft>) =>
    setDef((d) => ({ ...d, ...update }));

  const handleSave = async () => {
    if (!def.id.trim()) { setErr("ID (slug) is required"); return; }
    if (!def.name.trim()) { setErr("Name is required"); return; }
    setSaving(true);
    setErr("");
    try { await onSave(def); } catch (e) { setErr(String(e)); }
    finally { setSaving(false); }
  };

  const availableForms = FORM_BY_DOMAIN[def.domain] ?? [];
  const availableTypes = GENERAL_TYPE_BY_DOMAIN[def.domain] ?? [];

  const numField = (
    label: string,
    key: keyof UnitDefinitionDraft,
    step = 1,
    min = 0,
  ) => (
    <div className="field">
      <label className="field-label">{label}</label>
      <input
        className="field-input"
        type="number"
        step={step}
        min={min}
        value={def[key] as number}
        onChange={(e) => patch({ [key]: Number(e.target.value) } as Partial<UnitDefinitionDraft>)}
      />
    </div>
  );

  return (
    <div className="defform-root">
      <div className="defform-scroll">

        <div className="panel-section">
          <div className="panel-section-header">Identity</div>
          <div className="field">
            <label className="field-label">ID (slug){isNew && " *"}</label>
            <input
              className="field-input"
              value={def.id}
              readOnly={!isNew}
              style={!isNew ? { opacity: 0.5 } : undefined}
              onChange={(e) => patch({ id: e.target.value.toLowerCase().replace(/\s+/g, "-") })}
              placeholder="e.g. mbt-m1a2-abrams"
            />
          </div>
          <div className="field">
            <label className="field-label">Name *</label>
            <input className="field-input" value={def.name}
              onChange={(e) => patch({ name: e.target.value })} />
          </div>
          <div className="field">
            <label className="field-label">Description</label>
            <textarea className="field-textarea" value={def.description}
              onChange={(e) => patch({ description: e.target.value })} />
          </div>
        </div>

        <div className="panel-section">
          <div className="panel-section-header">Hierarchy</div>
          <div className="field">
            <label className="field-label">Domain</label>
            <select className="field-select" value={def.domain}
              onChange={(e) => {
                const d = Number(e.target.value);
                const forms = FORM_BY_DOMAIN[d] ?? [];
                const types = GENERAL_TYPE_BY_DOMAIN[d] ?? [];
                patch({ domain: d, form: forms[0] ?? 0, generalType: types[0] ?? 0 });
              }}>
              {Object.entries(DOMAIN_LABELS).map(([v, l]) => (
                <option key={v} value={v}>{l}</option>
              ))}
            </select>
          </div>
          <div className="field">
            <label className="field-label">Form</label>
            <select className="field-select" value={def.form}
              onChange={(e) => patch({ form: Number(e.target.value) })}>
              {availableForms.map((v) => (
                <option key={v} value={v}>{FORM_LABELS[v] ?? v}</option>
              ))}
            </select>
          </div>
          <div className="field">
            <label className="field-label">General Type</label>
            <select className="field-select" value={def.generalType}
              onChange={(e) => patch({ generalType: Number(e.target.value) })}>
              {availableTypes.map((v) => (
                <option key={v} value={v}>{GENERAL_TYPE_LABELS[v] ?? v}</option>
              ))}
            </select>
          </div>
          <div className="field">
            <label className="field-label">Specific Type</label>
            <input className="field-input" value={def.specificType}
              onChange={(e) => patch({ specificType: e.target.value })}
              placeholder="e.g. F-35A Lightning II" />
          </div>
          <div className="field">
            <label className="field-label">Short Name</label>
            <input className="field-input" value={def.shortName}
              onChange={(e) => patch({ shortName: e.target.value })}
              placeholder="e.g. F-22A or MiG-35" />
          </div>
        </div>

        <div className="panel-section">
          <div className="panel-section-header">Origin</div>
          <div className="field-row">
            <div className="field">
              <label className="field-label">Nation (ISO-3)</label>
              <input className="field-input" value={def.nationOfOrigin}
                onChange={(e) => patch({ nationOfOrigin: e.target.value.toUpperCase().slice(0,3) })}
                placeholder="USA" maxLength={3} />
            </div>
            <div className="field">
              <label className="field-label">Service Entry</label>
              <input className="field-input" type="number" min={1900} max={2100}
                value={def.serviceEntryYear}
                onChange={(e) => patch({ serviceEntryYear: Number(e.target.value) })} />
            </div>
          </div>
        </div>

        <div className="panel-section">
          <div className="panel-section-header">Combat</div>
          <div className="field-row">
            {numField("Base Strength (0–1)", "baseStrength", 0.01, 0)}
            {numField("Accuracy (0–1)", "accuracy", 0.01, 0)}
          </div>
          {numField("Combat Range (m)", "combatRangeM", 100, 0)}
        </div>

        <div className="panel-section">
          <div className="panel-section-header">Mobility</div>
          <div className="field-row">
            {numField("Max Speed (m/s)", "maxSpeedMps", 0.1, 0)}
            {numField("Cruise Speed (m/s)", "cruiseSpeedMps", 0.1, 0)}
          </div>
          {numField("Max Range (km)", "maxRangeKm", 10, 0)}
        </div>

        <div className="panel-section">
          <div className="panel-section-header">Survivability &amp; Sensors</div>
          <div className="field-row">
            {numField("Survivability (0–1)", "survivability", 0.01, 0)}
            {numField("Detection Range (m)", "detectionRangeM", 1000, 0)}
          </div>
          {numField("Radar Cross Section (m²)", "radarCrossSectionM2", 0.01, 0)}
        </div>

        <div className="panel-section">
          <div className="panel-section-header">Logistics</div>
          <div className="field-row">
            {numField("Fuel Capacity (L)", "fuelCapacityLiters", 100, 0)}
            {numField("Burn Rate (L/h)", "fuelBurnRateLph", 10, 0)}
          </div>
        </div>

      </div>

      {err && <div className="defform-error">{err}</div>}

      <div className="defform-footer">
        <button className="btn btn-success" onClick={handleSave} disabled={saving}>
          {saving ? "Saving…" : "Save"}
        </button>
        <button className="btn" onClick={onCancel}>Cancel</button>
        {!isNew && onDelete && (
          <button className="btn btn-danger" style={{ marginLeft: "auto" }} onClick={onDelete}>
            Delete
          </button>
        )}
      </div>
    </div>
  );
}

// ─── MAIN COMPONENT ───────────────────────────────────────────────────────────

interface Props {
  onClose: () => void;
}

export default function UnitDefinitionManager({ onClose }: Props) {
  const definitions = useEditorStore((s) => s.unitDefinitions);
  const { loadUnitDefinitions, upsertUnitDefinition, removeUnitDefinition } = useEditorStore();

  const [selected, setSelected] = useState<UnitDefinitionDraft | null>(null);
  const [isNew, setIsNew] = useState(false);
  const [status, setStatus] = useState("");

  const refresh = async () => {
    try {
      const rows = await ListUnitDefinitions();
      loadUnitDefinitions(rows.map(rowToDef));
    } catch (e) { console.error(e); }
  };

  const handleNew = () => {
    setSelected(blankDef());
    setIsNew(true);
    setStatus("");
  };

  const handleSelect = (def: UnitDefinitionDraft) => {
    setSelected({ ...def });
    setIsNew(false);
    setStatus("");
  };

  const handleSave = async (def: UnitDefinitionDraft) => {
    const payload = {
      id: def.id, name: def.name, description: def.description,
      domain: def.domain, form: def.form, general_type: def.generalType,
      specific_type: def.specificType, short_name: def.shortName, nation_of_origin: def.nationOfOrigin,
      service_entry_year: def.serviceEntryYear,
      base_strength: def.baseStrength, combat_range_m: def.combatRangeM,
      accuracy: def.accuracy, max_speed_mps: def.maxSpeedMps,
      cruise_speed_mps: def.cruiseSpeedMps, max_range_km: def.maxRangeKm,
      survivability: def.survivability, detection_range_m: def.detectionRangeM, radar_cross_section_m2: def.radarCrossSectionM2,
      fuel_capacity_liters: def.fuelCapacityLiters, fuel_burn_rate_lph: def.fuelBurnRateLph,
    };
    const res = await SaveUnitDefinition(JSON.stringify(payload));
    if (!res.success) throw new Error(res.error);
    upsertUnitDefinition(def);
    setSelected(def);
    setIsNew(false);
    setStatus("Saved.");
    setTimeout(() => setStatus(""), 2000);
  };

  const handleDelete = async () => {
    if (!selected) return;
    if (!confirm(`Delete "${selected.name}"? Units using this definition will lose their reference.`)) return;
    const res = await DeleteUnitDefinition(selected.id);
    if (!res.success) { setStatus(`Error: ${res.error}`); return; }
    removeUnitDefinition(selected.id);
    setSelected(null);
    setIsNew(false);
    await refresh();
  };

  // Group definitions by domain for sidebar
  const DOMAIN_ORDER = [1, 2, 3, 4];

  return (
    <div className="defmgr-overlay" onClick={onClose}>
      <div className="defmgr-shell" onClick={(e) => e.stopPropagation()}>

        {/* Header */}
        <div className="defmgr-header">
          <span>Unit Definitions</span>
          <div className="defmgr-header-actions">
            {status && <span className="defmgr-status">{status}</span>}
            <button className="btn btn-primary btn-sm" onClick={handleNew}>+ New</button>
            <button className="modal-close" onClick={onClose}>×</button>
          </div>
        </div>

        <div className="defmgr-body">
          {/* Sidebar */}
          <div className="defmgr-sidebar">
            {DOMAIN_ORDER.map((domainId) => {
              const defs = definitions.filter((d) => d.domain === domainId);
              if (defs.length === 0) return null;
              const color = DOMAIN_COLORS[domainId] ?? "#888";
              return (
                <div key={domainId} className="defmgr-domain-group">
                  <div className="defmgr-domain-label" style={{ color }}>
                    {DOMAIN_LABELS[domainId] ?? `Domain ${domainId}`}
                  </div>
                  {defs.map((def) => (
                    <div
                      key={def.id}
                      className={`defmgr-item${selected?.id === def.id && !isNew ? " selected" : ""}`}
                      onClick={() => handleSelect(def)}
                    >
                      <span className="defmgr-item-icon">
                        <UnitTypeIcon generalType={def.generalType} size={24} />
                      </span>
                      {def.name}
                    </div>
                  ))}
                </div>
              );
            })}
            {definitions.length === 0 && (
              <div className="defmgr-empty">No definitions yet. Click "+ New" to create one.</div>
            )}
          </div>

          {/* Form panel */}
          <div className="defmgr-form-panel">
            {(selected || isNew) ? (
              <DefinitionForm
                key={isNew ? "__new__" : selected!.id}
                initial={selected ?? blankDef()}
                onSave={handleSave}
                onCancel={() => { setSelected(null); setIsNew(false); }}
                onDelete={!isNew ? handleDelete : undefined}
                isNew={isNew}
              />
            ) : (
              <div className="defmgr-empty-form">
                Select a definition to edit, or click "+ New" to create one.
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
