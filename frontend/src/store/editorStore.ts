/**
 * editorStore.ts
 *
 * Zustand store for the scenario editor. Holds a mutable draft of the scenario
 * being edited. Completely separate from simStore so edits never affect the
 * live simulation state.
 */

import { create } from "zustand";
import { pointH3Fields } from "../utils/h3";

// ─── TYPES ────────────────────────────────────────────────────────────────────

export interface UnitDraft {
  id: string;
  displayName: string;
  fullName: string;
  teamId: string;
  coalitionId: string;
  definitionId: string;
  hostBaseId?: string;
  parentUnitId?: string;
  loadoutConfigurationId: string;
  natoSymbolSidc: string;
  lat: number;
  lon: number;
  h3Cell?: string;
  h3ParentCell?: string;
  altMsl: number;
  heading: number;       // degrees 0–359
  speed: number;         // m/s
  // Status (normalised 0–1 except fuel)
  personnelStrength: number;
  equipmentStrength: number;
  combatEffectiveness: number;
  fuelLevelLiters: number;
  morale: number;
  fatigue: number;
  damageState: number;
  engagementBehavior: number;
  engagementPkillThreshold: number;
  attackOrder?: {
    orderType: number;
    targetUnitId: string;
    desiredEffect: number;
    pkillThreshold: number;
  };
  nextSortieReadySeconds?: number;
  baseOps?: {
    state: number;
    nextLaunchAvailableSeconds: number;
    nextRecoveryAvailableSeconds: number;
  };
  moveOrder?: {
    waypoints: {
      lat: number;
      lon: number;
      h3Cell?: string;
      h3ParentCell?: string;
      altMsl: number;
    }[];
  };
}

export interface WeaponConfigSlotDraft {
  weaponId: string;
  maxQty: number;
  initialQty: number;
}

export interface WeaponConfigurationDraft {
  id: string;
  name: string;
  description: string;
  loadout: WeaponConfigSlotDraft[];
}

export interface ScenarioDraft {
  id: string;
  name: string;
  description: string;
  classification: string;
  author: string;
  startTimeUnix: number;
  version: string;
  tickRateHz: number;
  timeScale: number;
  // Weather
  weatherState: number;   // WeatherState enum value
  visibilityKm: number;
  windSpeedMps: number;
  temperatureC: number;
  units: UnitDraft[];
  relationships: CountryRelationshipDraft[];
}

export interface CountryRelationshipDraft {
  fromCountry: string;
  toCountry: string;
  shareIntel: boolean;
  airspaceTransitAllowed: boolean;
  airspaceStrikeAllowed: boolean;
  defensivePositioningAllowed: boolean;
  maritimeTransitAllowed: boolean;
  maritimeStrikeAllowed: boolean;
}

export interface PendingDrop {
  lat: number;
  lon: number;
  domain: number;      // kept for display color
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

export interface UnitDefinitionDraft {
  id: string;
  name: string;
  description: string;
  domain: number;
  form: number;
  generalType: number;
  specificType: string;
  shortName: string;
  assetClass: string;
  targetClass: string;
  employmentRole: string;
  authorizedPersonnel: number;
  stationary: boolean;
  affiliation: string;
  nationOfOrigin: string;
  operators: string[];
  employedBy: string[];
  serviceEntryYear: number;
  baseStrength: number;
  combatRangeM: number;
  accuracy: number;
  maxSpeedMps: number;
  cruiseSpeedMps: number;
  maxRangeKm: number;
  survivability: number;
  detectionRangeM: number;
  radarCrossSectionM2: number;
  fuelCapacityLiters: number;
  fuelBurnRateLph: number;
  embarkedFixedWingCapacity: number;
  embarkedRotaryWingCapacity: number;
  embarkedUavCapacity: number;
  embarkedSurfaceConnectorCapacity: number;
  launchCapacityPerInterval: number;
  recoveryCapacityPerInterval: number;
  sortieIntervalMinutes: number;
  replacementCostUsd: number;
  strategicValueUsd: number;
  economicValueUsd: number;
  dataConfidence: string;
  sourceBasis: string;
  sourceNotes: string;
  sourceLinks: string[];
  sensorSuite: {
    sensorType: string;
    maxRangeM: number;
    targetStates: string[];
    fireControl: boolean;
  }[];
  defaultWeaponConfiguration: string;
  weaponConfigurations: WeaponConfigurationDraft[];
}

interface EditorState {
  draft: ScenarioDraft;
  selectedUnitId: string | null;
  /** ID of unit being edited in the form, or "new" when adding, or null */
  editingUnitId: string | null;
  isDirty: boolean;
  /** Drag-drop pending — shown in DropConfirmDialog */
  pendingDrop: PendingDrop | null;
  /** Position set by clicking the globe — auto-fills lat/lon in unit form */
  pendingPosition: { lat: number; lon: number } | null;
  unitDefinitions: UnitDefinitionDraft[];
  selectedCountryCode: string;

  // Actions
  newDraft: () => void;
  loadDraft: (draft: ScenarioDraft) => void;
  updateMeta: (patch: Partial<Omit<ScenarioDraft, "units">>) => void;
  addUnit: (unit: UnitDraft) => void;
  updateUnit: (id: string, patch: Partial<UnitDraft>) => void;
  deleteUnit: (id: string) => void;
  selectUnit: (id: string | null) => void;
  setEditingUnit: (id: string | null) => void;
  setPendingPosition: (pos: { lat: number; lon: number } | null) => void;
  setPendingDrop: (drop: PendingDrop | null) => void;
  markClean: () => void;
  loadUnitDefinitions: (defs: UnitDefinitionDraft[]) => void;
  upsertUnitDefinition: (def: UnitDefinitionDraft) => void;
  removeUnitDefinition: (id: string) => void;
  setSelectedCountryCode: (code: string) => void;
}

function coordinatesChanged(
  next: { lat: number; lon: number },
  previous: { lat: number; lon: number },
): boolean {
  return next.lat !== previous.lat || next.lon !== previous.lon;
}

function mergeSpatialPoint<T extends { lat: number; lon: number; h3Cell?: string; h3ParentCell?: string }>(
  existing: T,
  patch: Partial<T>,
): T {
  const merged = { ...existing, ...patch } as T;
  const hasExplicitH3 = Object.prototype.hasOwnProperty.call(patch, "h3Cell")
    || Object.prototype.hasOwnProperty.call(patch, "h3ParentCell");
  if (hasExplicitH3) {
    return merged;
  }
  if (coordinatesChanged(merged, existing)) {
    Object.assign(merged, pointH3Fields(merged.lat, merged.lon));
  } else {
    merged.h3Cell = existing.h3Cell;
    merged.h3ParentCell = existing.h3ParentCell;
  }
  return merged;
}

function withSpatialH3<T extends { lat: number; lon: number; h3Cell?: string; h3ParentCell?: string }>(point: T): T {
  if (point.h3Cell || point.h3ParentCell) {
    return point;
  }
  return { ...point, ...pointH3Fields(point.lat, point.lon) };
}

function mergeMoveOrder(
  existing: UnitDraft["moveOrder"],
  patch: Partial<NonNullable<UnitDraft["moveOrder"]>>,
): UnitDraft["moveOrder"] {
  const nextWaypoints = patch.waypoints ?? existing?.waypoints ?? [];
  return {
    waypoints: nextWaypoints.map((waypoint, index) => {
      const existingWaypoint = existing?.waypoints?.[index];
      if (!existingWaypoint) {
        return withSpatialH3(waypoint);
      }
      return mergeSpatialPoint(existingWaypoint, waypoint);
    }),
  };
}

function mergeUnitDraft(existing: UnitDraft, patch: Partial<UnitDraft>): UnitDraft {
  const merged = { ...existing, ...patch } as UnitDraft;
  const hasPositionPatch = Object.prototype.hasOwnProperty.call(patch, "lat")
    || Object.prototype.hasOwnProperty.call(patch, "lon")
    || Object.prototype.hasOwnProperty.call(patch, "h3Cell")
    || Object.prototype.hasOwnProperty.call(patch, "h3ParentCell");
  if (hasPositionPatch) {
    const spatial = mergeSpatialPoint(existing, patch);
    merged.lat = spatial.lat;
    merged.lon = spatial.lon;
    merged.h3Cell = spatial.h3Cell;
    merged.h3ParentCell = spatial.h3ParentCell;
  }
  if (Object.prototype.hasOwnProperty.call(patch, "moveOrder")) {
    if (patch.moveOrder?.waypoints) {
      merged.moveOrder = mergeMoveOrder(existing.moveOrder, patch.moveOrder);
    } else {
      merged.moveOrder = patch.moveOrder;
    }
  }
  return merged;
}

// ─── DEFAULT VALUES ────────────────────────────────────────────────────────────

function blankDraft(): ScenarioDraft {
  return {
    id: crypto.randomUUID(),
    name: "New Scenario",
    description: "",
    classification: "UNCLASSIFIED",
    author: "",
    startTimeUnix: Date.UTC(2025, 5, 1, 6, 0, 0) / 1000,
    version: "1.0.0",
    tickRateHz: 10,
    timeScale: 1.0,
    weatherState: 1, // WEATHER_CLEAR
    visibilityKm: 40,
    windSpeedMps: 5,
    temperatureC: 20,
    units: [],
    relationships: [],
  };
}

export function blankUnit(lat = 35.0, lon = 25.0): UnitDraft {
  return {
    id: crypto.randomUUID(),
    displayName: "UNIT-1",
    fullName: "",
    teamId: "",
    coalitionId: "",
    definitionId: "",
    hostBaseId: undefined,
    parentUnitId: undefined,
    loadoutConfigurationId: "",
    natoSymbolSidc: "",
    lat,
    lon,
    ...pointH3Fields(lat, lon),
    altMsl: 0,
    heading: 0,
    speed: 0,
    personnelStrength: 1.0,
    equipmentStrength: 1.0,
    combatEffectiveness: 1.0,
    fuelLevelLiters: 10000,
    morale: 1.0,
    fatigue: 0.0,
    damageState: 1,
    engagementBehavior: 1,
    engagementPkillThreshold: 0.5,
    attackOrder: undefined,
    nextSortieReadySeconds: 0,
    baseOps: undefined,
    moveOrder: undefined,
  };
}

// ─── STORE ────────────────────────────────────────────────────────────────────

export const useEditorStore = create<EditorState>((set) => ({
  draft: blankDraft(),
  selectedUnitId: null,
  editingUnitId: null,
  isDirty: false,
  pendingDrop: null,
  pendingPosition: null,
  unitDefinitions: [],
  selectedCountryCode: "",

  newDraft: () =>
    set({
      draft: blankDraft(),
      selectedUnitId: null,
      editingUnitId: null,
      isDirty: false,
      pendingPosition: null,
    }),

  loadDraft: (draft) =>
    set({
      draft: {
        ...draft,
        units: draft.units.map((unit) => ({
          ...withSpatialH3(unit),
          moveOrder: unit.moveOrder
            ? {
                ...unit.moveOrder,
                waypoints: unit.moveOrder.waypoints.map((waypoint) => withSpatialH3(waypoint)),
              }
            : undefined,
        })),
      },
      selectedUnitId: null,
      editingUnitId: null,
      isDirty: false,
    }),

  updateMeta: (patch) =>
    set((s) => ({ draft: { ...s.draft, ...patch }, isDirty: true })),

  addUnit: (unit) =>
    set((s) => ({
      draft: { ...s.draft, units: [...s.draft.units, withSpatialH3(unit)] },
      isDirty: true,
      editingUnitId: null,
      pendingPosition: null,
      pendingDrop: null,
    })),

  updateUnit: (id, patch) =>
    set((s) => ({
      draft: {
        ...s.draft,
        units: s.draft.units.map((u) => (u.id === id ? mergeUnitDraft(u, patch) : u)),
      },
      isDirty: true,
    })),

  deleteUnit: (id) =>
    set((s) => ({
      draft: {
        ...s.draft,
        units: s.draft.units
          .filter((u) => u.id !== id)
          .map((u) => ({
            ...u,
            attackOrder: u.attackOrder?.targetUnitId === id ? undefined : u.attackOrder,
          })),
      },
      isDirty: true,
      selectedUnitId: s.selectedUnitId === id ? null : s.selectedUnitId,
      editingUnitId: s.editingUnitId === id ? null : s.editingUnitId,
    })),

  selectUnit: (id) => set({ selectedUnitId: id }),

  setEditingUnit: (id) => set({ editingUnitId: id }),

  setPendingPosition: (pos) => set({ pendingPosition: pos }),

  setPendingDrop: (drop) => set({ pendingDrop: drop }),

  markClean: () => set({ isDirty: false }),

  loadUnitDefinitions: (defs) => set({ unitDefinitions: defs }),
  upsertUnitDefinition: (def) =>
    set((s) => {
      const existing = s.unitDefinitions.find((d) => d.id === def.id);
      return {
        unitDefinitions: existing
          ? s.unitDefinitions.map((d) => (d.id === def.id ? def : d))
          : [...s.unitDefinitions, def],
      };
    }),
  removeUnitDefinition: (id) =>
    set((s) => ({ unitDefinitions: s.unitDefinitions.filter((d) => d.id !== id) })),

  setSelectedCountryCode: (code) => set({ selectedCountryCode: code }),
}));
