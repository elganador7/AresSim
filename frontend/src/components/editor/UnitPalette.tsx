/**
 * UnitPalette.tsx
 *
 * Displays unit definitions from the store as a draggable two-level tree:
 * Domain → Definition. Fetches definitions from the backend on mount.
 */

import { useEffect, useState } from "react";
import {
  useEditorStore,
  type UnitDefinitionDraft,
  type WeaponConfigurationDraft,
} from "../../store/editorStore";
import { ListUnitDefinitions } from "../../../wailsjs/go/main/App";
import UnitTypeIcon from "../UnitTypeIcon";
import { EDITOR_COUNTRIES, EDITOR_COUNTRY_NAME_BY_CODE } from "../../data/editorCountries";

// ─── DRAG PAYLOAD ─────────────────────────────────────────────────────────────

export interface DragPayload {
  domain: number;
  definitionId: string;
  label: string;
  shortName: string;
  domainColor: string;
  defaultWeaponConfiguration: string;
  weaponConfigurations: WeaponConfigurationDraft[];
  nationOfOrigin: string;
  employedBy: string[];
  employmentRole: string;
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
  const firstNonEmpty = (...values: string[]) => values.find((v) => v.trim().length > 0) ?? "";
  const weaponConfigurations = Array.isArray(r["weapon_configurations"])
    ? (r["weapon_configurations"] as Record<string, unknown>[]).map((cfg) => ({
        id: String(cfg["id"] ?? ""),
        name: String(cfg["name"] ?? ""),
        description: String(cfg["description"] ?? ""),
        loadout: Array.isArray(cfg["loadout"])
          ? (cfg["loadout"] as Record<string, unknown>[]).map((slot) => ({
              weaponId: String(slot["weapon_id"] ?? ""),
              maxQty: Number(slot["max_qty"] ?? 0),
              initialQty: Number(slot["initial_qty"] ?? 0),
            }))
          : [],
      }))
    : [];
  return {
    id:                 str("id"),
    name:               str("name"),
    description:        str("description"),
    domain:             num("domain"),
    form:               num("form"),
    generalType:        num("general_type"),
    specificType:       str("specific_type"),
    shortName:          firstNonEmpty(str("short_name"), str("specific_type"), str("name")),
    assetClass:         str("asset_class") || "combat_unit",
    targetClass:        str("target_class") || "soft_infrastructure",
    employmentRole:     str("employment_role") || "dual_use",
    authorizedPersonnel: num("authorized_personnel"),
    stationary:         Boolean(r["stationary"]),
    affiliation:        str("affiliation") || "military",
    nationOfOrigin:     str("nation_of_origin"),
    operators:          Array.isArray(r["operators"]) ? (r["operators"] as unknown[]).map(String) : [],
    employedBy:         Array.isArray(r["employed_by"])
      ? (r["employed_by"] as unknown[]).map(String)
      : (Array.isArray(r["operators"]) ? (r["operators"] as unknown[]).map(String) : []),
    serviceEntryYear:   num("service_entry_year"),
    baseStrength:       num("base_strength"),
    combatRangeM:       num("combat_range_m"),
    accuracy:           num("accuracy"),
    maxSpeedMps:        num("max_speed_mps"),
    cruiseSpeedMps:     num("cruise_speed_mps"),
    maxRangeKm:         num("max_range_km"),
    survivability:      num("survivability"),
    detectionRangeM:    num("detection_range_m"),
    radarCrossSectionM2: num("radar_cross_section_m2"),
    fuelCapacityLiters: num("fuel_capacity_liters"),
    fuelBurnRateLph:    num("fuel_burn_rate_lph"),
    embarkedFixedWingCapacity: num("embarked_fixed_wing_capacity"),
    embarkedRotaryWingCapacity: num("embarked_rotary_wing_capacity"),
    embarkedUavCapacity: num("embarked_uav_capacity"),
    embarkedSurfaceConnectorCapacity: num("embarked_surface_connector_capacity"),
    launchCapacityPerInterval: num("launch_capacity_per_interval"),
    recoveryCapacityPerInterval: num("recovery_capacity_per_interval"),
    sortieIntervalMinutes: num("sortie_interval_minutes"),
    replacementCostUsd: num("replacement_cost_usd"),
    strategicValueUsd: num("strategic_value_usd"),
    economicValueUsd: num("economic_value_usd"),
    dataConfidence: str("data_confidence") || "heuristic",
    sourceBasis: str("source_basis") || "heuristic",
    sourceNotes: str("source_notes"),
    sourceLinks: Array.isArray(r["source_links"]) ? (r["source_links"] as unknown[]).map(String) : [],
    defaultWeaponConfiguration: str("default_weapon_configuration"),
    weaponConfigurations,
  };
}

function formatAssetClass(assetClass: string): string {
  return assetClass
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

// ─── COMPONENT ────────────────────────────────────────────────────────────────

export default function UnitPalette() {
  const definitions = useEditorStore((s) => s.unitDefinitions);
  const loadUnitDefinitions = useEditorStore((s) => s.loadUnitDefinitions);
  const selectedCountryCode = useEditorStore((s) => s.selectedCountryCode);
  const setSelectedCountryCode = useEditorStore((s) => s.setSelectedCountryCode);
  const [query, setQuery] = useState("");
  const [collapsedDomains, setCollapsedDomains] = useState<Record<number, boolean>>({});

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
      shortName: def.shortName || def.specificType || def.name,
      domainColor: meta.color,
      defaultWeaponConfiguration: def.defaultWeaponConfiguration,
      weaponConfigurations: def.weaponConfigurations,
      nationOfOrigin: def.nationOfOrigin,
      employedBy: def.employedBy,
      employmentRole: def.employmentRole,
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
      fontFamily: "'Avenir Next', 'Segoe UI', sans-serif",
      fontSize: "11px", fontWeight: "700",
      padding: "5px 12px", borderRadius: "3px",
      letterSpacing: "0.08em", pointerEvents: "none", whiteSpace: "nowrap",
    });
    document.body.appendChild(ghost);
    e.dataTransfer.setDragImage(ghost, ghost.offsetWidth / 2, 14);
    requestAnimationFrame(() => { if (document.body.contains(ghost)) document.body.removeChild(ghost); });
  };

  // Group by domain
  const normalizedQuery = query.trim().toLowerCase();
  const byDomain = new Map<number, UnitDefinitionDraft[]>();
  for (const def of definitions) {
    const availableToCountry =
      !selectedCountryCode ||
      def.employedBy.includes(selectedCountryCode) ||
      def.nationOfOrigin === selectedCountryCode;
    if (!availableToCountry) {
      continue;
    }
    const haystack = [
      def.name,
      def.shortName,
      def.specificType,
      def.description,
      def.nationOfOrigin,
      def.operators.join(" "),
      def.employedBy.join(" "),
      EDITOR_COUNTRY_NAME_BY_CODE[def.nationOfOrigin] ?? "",
      def.employedBy.map((code) => EDITOR_COUNTRY_NAME_BY_CODE[code] ?? code).join(" "),
    ].join(" ").toLowerCase();
    if (normalizedQuery && !haystack.includes(normalizedQuery)) {
      continue;
    }
    const list = byDomain.get(def.domain) ?? [];
    list.push(def);
    byDomain.set(def.domain, list);
  }
  for (const defs of byDomain.values()) {
    defs.sort((a, b) => {
      const stationaryDelta = Number(b.stationary) - Number(a.stationary);
      if (stationaryDelta !== 0) {
        return stationaryDelta;
      }
      return (a.shortName || a.name).localeCompare(b.shortName || b.name);
    });
  }
  const domainOrder = [1, 2, 3, 4];
  const visibleCount = Array.from(byDomain.values()).reduce((sum, defs) => sum + defs.length, 0);

  const toggleDomain = (domainId: number) => {
    setCollapsedDomains((current) => ({
      ...current,
      [domainId]: !current[domainId],
    }));
  };

  return (
    <div className="palette-root">
      <div className="palette-header">Unit Palette</div>
      <div className="palette-search-wrap">
        <select
          className="palette-country-select"
          value={selectedCountryCode}
          onChange={(e) => setSelectedCountryCode(e.target.value)}
        >
          <option value="">All Supported Countries</option>
          {EDITOR_COUNTRIES.map((country) => (
            <option key={country.code} value={country.code}>
              {country.name}
            </option>
          ))}
        </select>
      </div>
      <div className="palette-search-wrap">
        <input
          className="palette-search-input"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search units, types, countries..."
        />
      </div>

      {definitions.length === 0 && (
        <div className="palette-empty">No unit definitions. Create some in the Definitions editor.</div>
      )}

      {definitions.length > 0 && visibleCount === 0 && (
        <div className="palette-empty">No units match the current country and search filter.</div>
      )}

      {domainOrder.map((domainId) => {
        const defs = byDomain.get(domainId);
        if (!defs || defs.length === 0) return null;
        const meta = DOMAIN_META[domainId];
        const collapsed = collapsedDomains[domainId] ?? false;
        return (
          <div key={domainId} className="palette-domain-block">
            <button
              type="button"
              className="palette-domain-row"
              onClick={() => toggleDomain(domainId)}
              aria-expanded={!collapsed}
            >
              <span className="palette-domain-swatch" style={{ background: meta.color }} />
              <span className="palette-domain-label" style={{ color: meta.color }}>{meta.label}</span>
              <span className="palette-domain-count">{defs.length}</span>
              <span className={`palette-domain-chevron${collapsed ? " collapsed" : ""}`}>⌄</span>
            </button>
            {!collapsed && (
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
                  <span className="palette-echelon-label">
                    {(def.shortName || def.specificType || def.name)} {def.name !== (def.shortName || def.specificType || def.name) ? `- ${def.name}` : ""}
                  </span>
                  {def.stationary && (
                    <span className="palette-echelon-tag">
                      Fixed Site
                    </span>
                  )}
                  {selectedCountryCode && (
                    <span className="palette-echelon-country">
                      {EDITOR_COUNTRY_NAME_BY_CODE[selectedCountryCode] ?? selectedCountryCode}
                    </span>
                  )}
                  <span className="palette-echelon-meta">
                    {formatAssetClass(def.assetClass)} • {def.affiliation === "dual_use" ? "Dual-use" : def.affiliation.charAt(0).toUpperCase() + def.affiliation.slice(1)}
                  </span>
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
