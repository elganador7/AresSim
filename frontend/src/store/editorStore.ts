/**
 * editorStore.ts
 *
 * Zustand store for the scenario editor. Holds a mutable draft of the scenario
 * being edited. Completely separate from simStore so edits never affect the
 * live simulation state.
 */

import { create } from "zustand";

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
    set({ draft, selectedUnitId: null, editingUnitId: null, isDirty: false }),

  updateMeta: (patch) =>
    set((s) => ({ draft: { ...s.draft, ...patch }, isDirty: true })),

  addUnit: (unit) =>
    set((s) => ({
      draft: { ...s.draft, units: [...s.draft.units, unit] },
      isDirty: true,
      editingUnitId: null,
      pendingPosition: null,
      pendingDrop: null,
    })),

  updateUnit: (id, patch) =>
    set((s) => ({
      draft: {
        ...s.draft,
        units: s.draft.units.map((u) => (u.id === id ? { ...u, ...patch } : u)),
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
