/**
 * UnitPalette.tsx
 *
 * Two-level expandable tree of draggable unit templates: Domain → Unit Type.
 * Each leaf is a single draggable item — echelon removed.
 */

import { useState } from "react";

// ─── DATA MODEL ──────────────────────────────────────────────────────────────

export interface DragPayload {
  domain: number;
  unitType: number;
  label: string;
  domainColor: string;
}

interface UnitTypeEntry {
  unitType: number;
  label: string;
}

interface DomainCategory {
  domain: number;
  label: string;
  color: string;
  units: UnitTypeEntry[];
}

const PALETTE: DomainCategory[] = [
  {
    domain: 1,
    label: "Land Forces",
    color: "#4ade80",
    units: [
      { unitType: 1,  label: "Armor" },
      { unitType: 2,  label: "Mech Infantry" },
      { unitType: 3,  label: "Light Infantry" },
      { unitType: 4,  label: "Airborne" },
      { unitType: 7,  label: "Special Forces" },
      { unitType: 8,  label: "Cavalry" },
      { unitType: 10, label: "SP Artillery" },
      { unitType: 11, label: "Towed Artillery" },
    ],
  },
  {
    domain: 2,
    label: "Air Forces",
    color: "#94a3b8",
    units: [
      { unitType: 32, label: "Fighter" },
      { unitType: 33, label: "Multirole" },
      { unitType: 34, label: "Attack Aircraft" },
      { unitType: 36, label: "Transport" },
      { unitType: 39, label: "UAV Recon" },
    ],
  },
  {
    domain: 3,
    label: "Naval Forces",
    color: "#3b82f6",
    units: [
      { unitType: 50, label: "Aircraft Carrier" },
      { unitType: 51, label: "Destroyer" },
      { unitType: 52, label: "Frigate" },
      { unitType: 53, label: "Corvette" },
      { unitType: 54, label: "Patrol Boat" },
    ],
  },
  {
    domain: 4,
    label: "Subsurface",
    color: "#818cf8",
    units: [
      { unitType: 57, label: "Attack Submarine" },
    ],
  },
];

// ─── COMPONENT ────────────────────────────────────────────────────────────────

export default function UnitPalette() {
  const [expanded, setExpanded] = useState<Set<number>>(new Set([1]));

  const toggle = (domain: number) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      next.has(domain) ? next.delete(domain) : next.add(domain);
      return next;
    });

  const handleDragStart = (
    e: React.DragEvent<HTMLDivElement>,
    cat: DomainCategory,
    unit: UnitTypeEntry,
  ) => {
    const payload: DragPayload = {
      domain: cat.domain,
      unitType: unit.unitType,
      label: unit.label,
      domainColor: cat.color,
    };
    e.dataTransfer.setData("text/plain", JSON.stringify(payload));
    e.dataTransfer.effectAllowed = "copy";

    // Custom drag ghost
    const ghost = document.createElement("div");
    ghost.textContent = unit.label.toUpperCase();
    Object.assign(ghost.style, {
      position: "fixed",
      top: "-120px",
      left: "-120px",
      background: "rgba(10,12,16,0.97)",
      border: `1px solid ${cat.color}`,
      color: cat.color,
      fontFamily: "'Courier New', monospace",
      fontSize: "11px",
      fontWeight: "700",
      padding: "5px 12px",
      borderRadius: "3px",
      letterSpacing: "0.08em",
      pointerEvents: "none",
      whiteSpace: "nowrap",
    });
    document.body.appendChild(ghost);
    e.dataTransfer.setDragImage(ghost, ghost.offsetWidth / 2, 14);
    requestAnimationFrame(() => {
      if (document.body.contains(ghost)) document.body.removeChild(ghost);
    });
  };

  return (
    <div className="palette-root">
      <div className="palette-header">Unit Palette</div>

      {PALETTE.map((cat) => {
        const open = expanded.has(cat.domain);
        return (
          <div key={cat.domain} className="palette-domain-block">
            <div className="palette-domain-row" onClick={() => toggle(cat.domain)}>
              <span className="palette-chevron" style={{ color: cat.color }}>
                {open ? "▾" : "▸"}
              </span>
              <span className="palette-domain-swatch" style={{ background: cat.color }} />
              <span className="palette-domain-label" style={{ color: cat.color }}>
                {cat.label}
              </span>
            </div>

            {open && (
              <div className="palette-unit-list">
                {cat.units.map((unit) => (
                  <div
                    key={unit.unitType}
                    className="palette-echelon-item"
                    draggable
                    onDragStart={(e) => handleDragStart(e, cat, unit)}
                    title={`Drag to place ${unit.label}`}
                  >
                    <span className="palette-echelon-dot" style={{ background: cat.color }} />
                    <span className="palette-grip">⠿</span>
                    <span className="palette-echelon-label">{unit.label}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
