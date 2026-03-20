/**
 * ScenarioEditor.tsx
 *
 * Three-panel scenario editor:
 *   Left   — scenario metadata and relationship matrix
 *   Center — CesiumJS globe (placement / route editing)
 *   Right  — unit palette, selected-unit commands, placed-units list
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { fromBinary } from "@bufbuild/protobuf";
import { ScenarioSchema } from "@proto/engine/v1/scenario_pb";
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
  type UnitDefinitionDraft,
} from "../../store/editorStore";
import { buildCountryCoalitionMap } from "../../utils/countryRelationships";
import EditorGlobe from "./EditorGlobe";
import ScenarioMetaPanel from "./ScenarioMetaPanel";
import { PlacedUnits, SelectedUnitPanel } from "./ScenarioUnitPanels";
import UnitPalette, { type DragPayload } from "./UnitPalette";
import DropConfirmDialog from "./DropConfirmDialog";
import UnitDefinitionManager from "./UnitDefinitionManager";
import {
  base64ToBytes,
  draftPointsJSON,
  draftRelationshipsJSON,
  draftToProtoB64,
  getUnitCountry,
  isMaritimeDomain,
} from "./scenarioSerialization";
import "./editor.css";

function LoadModal({
  onClose,
  onSelect,
}: {
  onClose: () => void;
  onSelect: (id: string) => void;
}) {
  const [items, setItems] = useState<Array<Record<string, any>>>([]);
  const [busy, setBusy] = useState(true);

  useEffect(() => {
    ListScenarios()
      .then((rows) => setItems(rows))
      .catch(console.error)
      .finally(() => setBusy(false));
  }, []);

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <span>Open Scenario</span>
          <button className="modal-close" onClick={onClose}>✕</button>
        </div>
        {busy ? (
          <div className="modal-empty">Loading…</div>
        ) : items.length === 0 ? (
          <div className="modal-empty">No saved scenarios</div>
        ) : (
          <div className="modal-body">
            {items.map((row) => {
              const id = String(row.id ?? "");
              const name = String(row.name ?? "Untitled");
              const author = String(row.author ?? "");
              return (
                <div key={id} className="modal-scenario-item">
                  <div>
                    <div className="modal-scenario-name">{name}</div>
                    <div className="modal-scenario-meta">{author || "Unknown author"}</div>
                  </div>
                  <div className="modal-list-actions">
                    <button className="btn btn-sm btn-primary" onClick={() => onSelect(id)}>Open</button>
                    <button
                      className="btn btn-sm btn-danger"
                      onClick={async () => {
                        const res = await DeleteScenario(id);
                        if (!res.success) {
                          alert(`Delete failed:\n${res.error}`);
                          return;
                        }
                        setItems((current) => current.filter((item) => String(item.id ?? "") !== id));
                      }}
                    >
                      Delete
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

export default function ScenarioEditor({
  onExit,
  onPlay,
}: {
  onExit: () => void;
  onPlay: () => void;
}) {
  const {
    draft,
    isDirty,
    pendingDrop,
    selectedUnitId,
    editingUnitId,
    newDraft,
    loadDraft,
    addUnit,
    updateUnit,
    selectUnit,
    setEditingUnit,
    setPendingDrop,
    setPendingPosition,
    markClean,
  } = useEditorStore();

  const [showLoadModal, setShowLoadModal] = useState(false);
  const [showDefManager, setShowDefManager] = useState(false);
  const [statusMsg, setStatusMsg] = useState("");
  const [saving, setSaving] = useState(false);
  const [routeEditUnitId, setRouteEditUnitId] = useState<string | null>(null);
  const unitDefinitions = useEditorStore((s) => s.unitDefinitions);
  const countryCoalitions = useMemo(() => buildCountryCoalitionMap(draft.units), [draft.units]);
  const countryCoalitionsJSON = useMemo(() => JSON.stringify(countryCoalitions), [countryCoalitions]);
  const relationshipsJSON = useMemo(() => draftRelationshipsJSON(draft.relationships), [draft.relationships]);

  const flash = (msg: string) => {
    setStatusMsg(msg);
    setTimeout(() => setStatusMsg(""), 3000);
  };

  const previewPlacementViolation = useCallback(async (
    ownerCountry: string,
    definition: UnitDefinitionDraft | undefined,
    lat: number,
    lon: number,
  ): Promise<string> => {
    if (!definition) {
      return "";
    }
    const preview = await ((window as any).go?.main?.App?.PreviewDraftPlacement?.(
      ownerCountry,
      isMaritimeDomain(definition.domain),
      definition.employmentRole || "dual_use",
      relationshipsJSON,
      countryCoalitionsJSON,
      lat,
      lon,
    ) as Promise<{ blocked: boolean; reason?: string } | null> | undefined);
    return preview?.blocked ? (preview.reason ?? "Placement blocked.") : "";
  }, [countryCoalitionsJSON, relationshipsJSON]);

  const previewTransitViolation = useCallback(async (
    ownerCountry: string,
    maritime: boolean,
    points: { lat: number; lon: number }[],
  ): Promise<string> => {
    const preview = await ((window as any).go?.main?.App?.PreviewDraftTransitPath?.(
      ownerCountry,
      maritime,
      relationshipsJSON,
      countryCoalitionsJSON,
      draftPointsJSON(points),
    ) as Promise<{ blocked: boolean; reason?: string } | null> | undefined);
    return preview?.blocked ? (preview.reason ?? "Transit blocked.") : "";
  }, [countryCoalitionsJSON, relationshipsJSON]);

  const previewStrikeViolation = useCallback(async (
    unit: UnitDraft,
    target: UnitDraft | undefined,
  ): Promise<string> => {
    if (!target) {
      return "";
    }
    const points = [
      { lat: unit.lat, lon: unit.lon },
      ...(unit.moveOrder?.waypoints ?? []).map((wp) => ({ lat: wp.lat, lon: wp.lon })),
      { lat: target.lat, lon: target.lon },
    ];
    const definition = unitDefinitions.find((def) => def.id === unit.definitionId);
    const preview = await ((window as any).go?.main?.App?.PreviewDraftStrikePath?.(
      getUnitCountry(unit),
      isMaritimeDomain(definition?.domain),
      relationshipsJSON,
      countryCoalitionsJSON,
      draftPointsJSON(points),
    ) as Promise<{ blocked: boolean; reason?: string } | null> | undefined);
    return preview?.blocked ? (preview.reason ?? "Strike blocked.") : "";
  }, [countryCoalitionsJSON, relationshipsJSON, unitDefinitions]);

  const validateUnitPatch = useCallback(
    async (unit: UnitDraft, patch: Partial<UnitDraft>) => {
      const nextUnit = { ...unit, ...patch };
      const definition = unitDefinitions.find((def) => def.id === nextUnit.definitionId);
      const placementViolation = await previewPlacementViolation(nextUnit.teamId, definition, nextUnit.lat, nextUnit.lon);
      if (placementViolation) {
        flash(placementViolation);
        return false;
      }
      if (nextUnit.attackOrder?.targetUnitId) {
        const target = draft.units.find((candidate) => candidate.id === nextUnit.attackOrder?.targetUnitId);
        const strikeViolation = await previewStrikeViolation(nextUnit, target);
        if (strikeViolation) {
          flash(strikeViolation);
          return false;
        }
      }
      return true;
    },
    [draft.units, previewPlacementViolation, previewStrikeViolation, unitDefinitions],
  );

  const handleSave = async () => {
    setSaving(true);
    try {
      const res = await SaveScenario(draftToProtoB64(draft));
      if (res.success) {
        markClean();
        flash("Saved.");
      } else {
        alert(`Save failed:\n${res.error}`);
      }
    } catch (error) {
      alert(`Save error:\n${error}`);
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
        alert(`Save failed:\n${save.error}`);
        return;
      }
      const load = await LoadScenarioFromProto(b64);
      if (!load.success) {
        alert(`Load failed:\n${load.error}`);
        return;
      }
      markClean();
      onPlay();
    } catch (error) {
      alert(`Play error:\n${error}`);
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
          maritimeTransitAllowed: rel.maritimeTransitAllowed,
          maritimeStrikeAllowed: rel.maritimeStrikeAllowed,
        })),
        units: scen.units.map((u) => ({
          id: u.id,
          displayName: u.displayName,
          fullName: u.fullName,
          teamId: u.teamId || "",
          coalitionId: u.coalitionId || u.teamId || "",
          definitionId: u.definitionId,
          hostBaseId: u.hostBaseId || undefined,
          parentUnitId: u.parentUnitId || undefined,
          loadoutConfigurationId: u.loadoutConfigurationId,
          natoSymbolSidc: u.natoSymbolSidc,
          damageState: u.damageState ?? 1,
          engagementBehavior: u.engagementBehavior ?? 1,
          engagementPkillThreshold: u.engagementPkillThreshold ?? 0.5,
          attackOrder: u.attackOrder ? {
            orderType: u.attackOrder.orderType,
            targetUnitId: u.attackOrder.targetUnitId,
            desiredEffect: u.attackOrder.desiredEffect,
            pkillThreshold: u.attackOrder.pkillThreshold,
          } : undefined,
          nextSortieReadySeconds: u.nextSortieReadySeconds ?? 0,
          baseOps: u.baseOps ? {
            state: u.baseOps.state,
            nextLaunchAvailableSeconds: u.baseOps.nextLaunchAvailableSeconds,
            nextRecoveryAvailableSeconds: u.baseOps.nextRecoveryAvailableSeconds,
          } : undefined,
          moveOrder: u.moveOrder?.waypoints?.length ? {
            waypoints: u.moveOrder.waypoints.map((wp) => ({
              lat: wp.lat,
              lon: wp.lon,
              altMsl: wp.altMsl,
            })),
          } : undefined,
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
    } catch (error) {
      flash(`Load error: ${error}`);
    }
  };

  const handleMapClick = useCallback(
    async (lat: number, lon: number) => {
      if (routeEditUnitId) {
        const unit = draft.units.find((candidate) => candidate.id === routeEditUnitId);
        if (!unit) {
          setRouteEditUnitId(null);
          return;
        }
        const start = unit.moveOrder?.waypoints?.length
          ? unit.moveOrder.waypoints[unit.moveOrder.waypoints.length - 1]
          : { lat: unit.lat, lon: unit.lon };
        const definition = unitDefinitions.find((def) => def.id === unit.definitionId);
        const transitViolation = await previewTransitViolation(
          unit.teamId,
          isMaritimeDomain(definition?.domain),
          [start, { lat, lon }],
        );
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
    [draft.units, previewTransitViolation, routeEditUnitId, selectUnit, setPendingPosition, setEditingUnit, unitDefinitions, updateUnit],
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

  const handleUnitDrop = useCallback((lat: number, lon: number, payload: DragPayload) => {
    setPendingDrop({ lat, lon, ...payload });
  }, [setPendingDrop]);

  return (
    <div className="editor-shell">
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

      <div className="editor-body">
        <div className="editor-panel editor-panel-left">
          <ScenarioMetaPanel />
        </div>

        <div className="editor-panel-center">
          <EditorGlobe
            onMapClick={handleMapClick}
            onUnitClick={handleUnitClick}
            onUnitDrop={handleUnitDrop}
            placementMode={editingUnitId === "new"}
          />
        </div>

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

      {pendingDrop && (
        <DropConfirmDialog
          drop={pendingDrop}
          onConfirm={async (unit) => {
            const definition = unitDefinitions.find((def) => def.id === unit.definitionId);
            const violation = await previewPlacementViolation(unit.teamId, definition, unit.lat, unit.lon);
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

      {showLoadModal && <LoadModal onClose={() => setShowLoadModal(false)} onSelect={handleLoadSelect} />}
      {showDefManager && <UnitDefinitionManager onClose={() => setShowDefManager(false)} />}
    </div>
  );
}
