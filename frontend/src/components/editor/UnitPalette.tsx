/**
 * UnitPalette.tsx
 *
 * Displays unit definitions from the store as a draggable two-level tree:
 * Domain → Definition. Fetches definitions from the backend on mount.
 */

import { useEffect } from "react";
import { useEditorStore, type UnitDefinitionDraft } from "../../store/editorStore";
import { ListUnitDefinitions } from "../../../wailsjs/go/main/App";
import UnitTypeIcon from "../UnitTypeIcon";

// ─── DRAG PAYLOAD ─────────────────────────────────────────────────────────────

export interface DragPayload {
  domain: number;
  definitionId: string;
  label: string;
  domainColor: string;
}

// ─── DOMAIN META ──────────────────────────────────────────────────────────────

const DOMAIN_META: Record<number, { label: string; color: string }> = {
  1: { label: "Land Forces",   color: "#4ade80" },
  2: { label: "Air Forces",    color: "#94a3b8" },
  3: { label: "Naval Forces",  color: "#3b82f6" },
  4: { label: "Subsurface",    color: "#818cf8" },
};

// ─── HELPERS ──────────────────────────────────────────────────────────────────

function rowToDef(r: Record<string, unknown>): UnitDefinitionDraft {
  const num = (k: string) => Number(r[k] ?? 0);
  const str = (k: string) => String(r[k] ?? "");
  return {
    id:                 str("id"),
    name:               str("name"),
    description:        str("description"),
    domain:             num("domain"),
    form:               num("form"),
    generalType:        num("general_type"),
    specificType:       str("specific_type"),
    nationOfOrigin:     str("nation_of_origin"),
    serviceEntryYear:   num("service_entry_year"),
    baseStrength:       num("base_strength"),
    combatRangeM:       num("combat_range_m"),
    accuracy:           num("accuracy"),
    maxSpeedMps:        num("max_speed_mps"),
    cruiseSpeedMps:     num("cruise_speed_mps"),
    maxRangeKm:         num("max_range_km"),
    survivability:      num("survivability"),
    detectionRangeM:    num("detection_range_m"),
    fuelCapacityLiters: num("fuel_capacity_liters"),
    fuelBurnRateLph:    num("fuel_burn_rate_lph"),
  };
}

// ─── COMPONENT ────────────────────────────────────────────────────────────────

export default function UnitPalette() {
  const definitions = useEditorStore((s) => s.unitDefinitions);
  const loadUnitDefinitions = useEditorStore((s) => s.loadUnitDefinitions);

  useEffect(() => {
    ListUnitDefinitions()
      .then((rows) => loadUnitDefinitions(rows.map(rowToDef)))
      .catch(console.error);
  }, [loadUnitDefinitions]);

  const handleDragStart = (e: React.DragEvent<HTMLDivElement>, def: UnitDefinitionDraft) => {
    const meta = DOMAIN_META[def.domain] ?? { color: "#888" };
    const payload: DragPayload = {
      domain: def.domain,
      definitionId: def.id,
      label: def.name,
      domainColor: meta.color,
    };
    e.dataTransfer.setData("text/plain", JSON.stringify(payload));
    e.dataTransfer.effectAllowed = "copy";

    const ghost = document.createElement("div");
    ghost.textContent = def.name.toUpperCase();
    Object.assign(ghost.style, {
      position: "fixed", top: "-120px", left: "-120px",
      background: "rgba(10,12,16,0.97)",
      border: `1px solid ${meta.color}`,
      color: meta.color,
      fontFamily: "'Courier New', monospace",
      fontSize: "11px", fontWeight: "700",
      padding: "5px 12px", borderRadius: "3px",
      letterSpacing: "0.08em", pointerEvents: "none", whiteSpace: "nowrap",
    });
    document.body.appendChild(ghost);
    e.dataTransfer.setDragImage(ghost, ghost.offsetWidth / 2, 14);
    requestAnimationFrame(() => { if (document.body.contains(ghost)) document.body.removeChild(ghost); });
  };

  // Group by domain
  const byDomain = new Map<number, UnitDefinitionDraft[]>();
  for (const def of definitions) {
    const list = byDomain.get(def.domain) ?? [];
    list.push(def);
    byDomain.set(def.domain, list);
  }
  const domainOrder = [1, 2, 3, 4];

  return (
    <div className="palette-root">
      <div className="palette-header">Unit Palette</div>

      {definitions.length === 0 && (
        <div className="palette-empty">No unit definitions. Create some in the Definitions editor.</div>
      )}

      {domainOrder.map((domainId) => {
        const defs = byDomain.get(domainId);
        if (!defs || defs.length === 0) return null;
        const meta = DOMAIN_META[domainId];
        return (
          <div key={domainId} className="palette-domain-block">
            <div className="palette-domain-row">
              <span className="palette-domain-swatch" style={{ background: meta.color }} />
              <span className="palette-domain-label" style={{ color: meta.color }}>{meta.label}</span>
            </div>
            <div className="palette-unit-list">
              {defs.map((def) => (
                <div
                  key={def.id}
                  className="palette-echelon-item"
                  draggable
                  onDragStart={(e) => handleDragStart(e, def)}
                  title={`${def.specificType || def.name} — drag to place`}
                >
                  <span className="palette-unit-icon">
                    <UnitTypeIcon generalType={def.generalType} size={28} />
                  </span>
                  <span className="palette-grip">⠿</span>
                  <span className="palette-echelon-label">{def.name}</span>
                </div>
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}
