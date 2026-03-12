/**
 * UnitPalette.tsx
 *
 * Three-level expandable tree of draggable unit templates.
 * Domain → Type Group → Echelon Variant
 *
 * Dragging an echelon variant fires a HTML5 DragEvent with a JSON
 * payload in dataTransfer. EditorGlobe picks this up on drop.
 */

import { useState } from "react";

// ─── DATA MODEL ──────────────────────────────────────────────────────────────

export interface DragPayload {
  domain: number;
  unitType: number;
  defaultEchelon: number;
  label: string;        // e.g. "Armor · Battalion"
  domainColor: string;
}

interface EchelonVariant {
  echelon: number;
  echelonLabel: string;
}

interface UnitTypeGroup {
  unitType: number;
  label: string;
  echelonVariants: EchelonVariant[];
}

interface DomainCategory {
  domain: number;
  label: string;
  color: string;
  groups: UnitTypeGroup[];
}

const PALETTE: DomainCategory[] = [
  {
    domain: 1,
    label: "Land Forces",
    color: "#4ade80",
    groups: [
      {
        unitType: 1,
        label: "Armor",
        echelonVariants: [
          { echelon: 5, echelonLabel: "Company" },
          { echelon: 6, echelonLabel: "Battalion" },
          { echelon: 7, echelonLabel: "Brigade" },
        ],
      },
      {
        unitType: 2,
        label: "Mech Infantry",
        echelonVariants: [
          { echelon: 4, echelonLabel: "Platoon" },
          { echelon: 5, echelonLabel: "Company" },
          { echelon: 6, echelonLabel: "Battalion" },
        ],
      },
      {
        unitType: 3,
        label: "Light Infantry",
        echelonVariants: [
          { echelon: 2, echelonLabel: "Squad" },
          { echelon: 4, echelonLabel: "Platoon" },
          { echelon: 5, echelonLabel: "Company" },
        ],
      },
      {
        unitType: 4,
        label: "Airborne",
        echelonVariants: [
          { echelon: 4, echelonLabel: "Platoon" },
          { echelon: 5, echelonLabel: "Company" },
        ],
      },
      {
        unitType: 7,
        label: "Special Forces",
        echelonVariants: [
          { echelon: 1, echelonLabel: "Element" },
          { echelon: 2, echelonLabel: "Team" },
          { echelon: 3, echelonLabel: "Section" },
        ],
      },
      {
        unitType: 8,
        label: "Cavalry",
        echelonVariants: [
          { echelon: 4, echelonLabel: "Troop" },
          { echelon: 6, echelonLabel: "Squadron" },
        ],
      },
      {
        unitType: 10,
        label: "SP Artillery",
        echelonVariants: [
          { echelon: 5, echelonLabel: "Battery" },
          { echelon: 6, echelonLabel: "Battalion" },
        ],
      },
      {
        unitType: 11,
        label: "Towed Artillery",
        echelonVariants: [
          { echelon: 5, echelonLabel: "Battery" },
          { echelon: 6, echelonLabel: "Battalion" },
        ],
      },
    ],
  },
  {
    domain: 2,
    label: "Air Forces",
    color: "#94a3b8",
    groups: [
      {
        unitType: 32,
        label: "Fighter",
        echelonVariants: [
          { echelon: 3, echelonLabel: "Flight" },
          { echelon: 5, echelonLabel: "Squadron" },
        ],
      },
      {
        unitType: 33,
        label: "Multirole",
        echelonVariants: [
          { echelon: 3, echelonLabel: "Flight" },
          { echelon: 5, echelonLabel: "Squadron" },
        ],
      },
      {
        unitType: 34,
        label: "Attack Aircraft",
        echelonVariants: [
          { echelon: 3, echelonLabel: "Flight" },
          { echelon: 5, echelonLabel: "Squadron" },
        ],
      },
      {
        unitType: 36,
        label: "Transport",
        echelonVariants: [
          { echelon: 5, echelonLabel: "Squadron" },
        ],
      },
      {
        unitType: 39,
        label: "UAV Recon",
        echelonVariants: [
          { echelon: 3, echelonLabel: "Element" },
        ],
      },
    ],
  },
  {
    domain: 3,
    label: "Naval Forces",
    color: "#3b82f6",
    groups: [
      {
        unitType: 50,
        label: "Aircraft Carrier",
        echelonVariants: [{ echelon: 3, echelonLabel: "Ship" }],
      },
      {
        unitType: 51,
        label: "Destroyer",
        echelonVariants: [{ echelon: 3, echelonLabel: "Ship" }],
      },
      {
        unitType: 52,
        label: "Frigate",
        echelonVariants: [{ echelon: 3, echelonLabel: "Ship" }],
      },
      {
        unitType: 53,
        label: "Corvette",
        echelonVariants: [{ echelon: 3, echelonLabel: "Ship" }],
      },
      {
        unitType: 54,
        label: "Patrol Boat",
        echelonVariants: [{ echelon: 3, echelonLabel: "Vessel" }],
      },
    ],
  },
  {
    domain: 4,
    label: "Subsurface",
    color: "#818cf8",
    groups: [
      {
        unitType: 57,
        label: "Attack Submarine",
        echelonVariants: [{ echelon: 3, echelonLabel: "Boat" }],
      },
    ],
  },
];

// ─── COMPONENT ────────────────────────────────────────────────────────────────

export default function UnitPalette() {
  const [expandedDomains, setExpandedDomains] = useState<Set<number>>(
    new Set([1]), // Land open by default
  );
  const [expandedTypes, setExpandedTypes] = useState<Set<string>>(new Set());

  const toggleDomain = (domain: number) =>
    setExpandedDomains((prev) => {
      const next = new Set(prev);
      next.has(domain) ? next.delete(domain) : next.add(domain);
      return next;
    });

  const toggleType = (key: string) =>
    setExpandedTypes((prev) => {
      const next = new Set(prev);
      next.has(key) ? next.delete(key) : next.add(key);
      return next;
    });

  const handleDragStart = (
    e: React.DragEvent<HTMLDivElement>,
    cat: DomainCategory,
    group: UnitTypeGroup,
    variant: EchelonVariant,
  ) => {
    const label = `${group.label} · ${variant.echelonLabel}`;
    const payload: DragPayload = {
      domain: cat.domain,
      unitType: group.unitType,
      defaultEchelon: variant.echelon,
      label,
      domainColor: cat.color,
    };
    e.dataTransfer.setData("text/plain", JSON.stringify(payload));
    e.dataTransfer.effectAllowed = "copy";

    // Custom ghost drag image
    const ghost = document.createElement("div");
    ghost.textContent = label.toUpperCase();
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
        const domainOpen = expandedDomains.has(cat.domain);
        return (
          <div key={cat.domain} className="palette-domain-block">
            {/* Domain header */}
            <div
              className="palette-domain-row"
              onClick={() => toggleDomain(cat.domain)}
            >
              <span className="palette-chevron" style={{ color: cat.color }}>
                {domainOpen ? "▾" : "▸"}
              </span>
              <span className="palette-domain-swatch" style={{ background: cat.color }} />
              <span className="palette-domain-label" style={{ color: cat.color }}>
                {cat.label}
              </span>
            </div>

            {/* Type groups */}
            {domainOpen &&
              cat.groups.map((group) => {
                const typeKey = `${cat.domain}-${group.unitType}`;
                const typeOpen = expandedTypes.has(typeKey);
                return (
                  <div key={group.unitType} className="palette-type-group">
                    {/* Type header */}
                    <div
                      className="palette-type-row"
                      onClick={() => toggleType(typeKey)}
                    >
                      <span className="palette-chevron">
                        {typeOpen ? "▾" : "▸"}
                      </span>
                      <span className="palette-type-label">{group.label}</span>
                    </div>

                    {/* Echelon variants */}
                    {typeOpen &&
                      group.echelonVariants.map((variant) => (
                        <div
                          key={variant.echelon}
                          className="palette-echelon-item"
                          draggable
                          onDragStart={(e) =>
                            handleDragStart(e, cat, group, variant)
                          }
                          title={`Drag to place ${group.label} (${variant.echelonLabel})`}
                        >
                          <span
                            className="palette-echelon-dot"
                            style={{ background: cat.color }}
                          />
                          <span className="palette-grip">⠿</span>
                          <span className="palette-echelon-label">
                            {variant.echelonLabel}
                          </span>
                        </div>
                      ))}
                  </div>
                );
              })}
          </div>
        );
      })}
    </div>
  );
}
