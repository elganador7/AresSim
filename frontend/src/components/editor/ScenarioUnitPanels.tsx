import { useEffect, useState } from "react";
import { ListWeaponDefinitions } from "../../../wailsjs/go/main/App";
import {
  useEditorStore,
  type UnitDraft,
  type UnitDefinitionDraft,
} from "../../store/editorStore";
import { areFriendly } from "../../utils/allegiance";
import { assessLoadoutAgainstTarget, type WeaponDefLite } from "../../utils/loadoutValidation";
import { teamColorHex } from "../../utils/teamColors";
import { ATTACK_ORDER_TYPES, DESIRED_EFFECTS, ENGAGEMENT_BEHAVIORS, filterValidEditorTargets } from "../../utils/tasking";
import { formatCountry } from "./scenarioSerialization";

const BASE_OPS_STATE_LABEL: Record<number, string> = {
  0: "Unknown",
  1: "Usable",
  2: "Degraded",
  3: "Closed",
};

function isHostPlatform(definition: UnitDefinitionDraft | undefined): boolean {
  if (!definition) return false;
  return definition.assetClass === "airbase"
    || definition.embarkedFixedWingCapacity > 0
    || definition.embarkedRotaryWingCapacity > 0
    || definition.embarkedUavCapacity > 0
    || definition.launchCapacityPerInterval > 0
    || definition.recoveryCapacityPerInterval > 0;
}

function InlineEditForm({
  unit,
  units,
  unitDefinitions,
  weaponDefs,
  availableLoadouts,
  isRouteMode,
  onToggleRouteMode,
  onSave,
  onCancel,
}: {
  unit: UnitDraft;
  units: UnitDraft[];
  unitDefinitions: UnitDefinitionDraft[];
  weaponDefs: Map<string, WeaponDefLite>;
  availableLoadouts: { id: string; name: string }[];
  isRouteMode: boolean;
  onToggleRouteMode: () => void;
  onSave: (patch: Partial<UnitDraft>) => void;
  onCancel: () => void;
}) {
  const [displayName, setDisplayName] = useState(unit.displayName);
  const [teamId, setTeamId] = useState(unit.teamId);
  const [coalitionId, setCoalitionId] = useState(unit.coalitionId || unit.teamId);
  const [hostBaseId, setHostBaseId] = useState(unit.hostBaseId ?? "");
  const [heading, setHeading] = useState(unit.heading);
  const [speed, setSpeed] = useState(unit.speed);
  const [strength, setStrength] = useState(unit.combatEffectiveness);
  const [loadoutConfigurationId, setLoadoutConfigurationId] = useState(unit.loadoutConfigurationId);
  const [engagementBehavior, setEngagementBehavior] = useState(unit.engagementBehavior ?? 1);
  const [engagementPkillThreshold, setEngagementPkillThreshold] = useState(unit.engagementPkillThreshold ?? 0.5);
  const [attackOrderType, setAttackOrderType] = useState(unit.attackOrder?.orderType ?? 0);
  const [targetUnitId, setTargetUnitId] = useState(unit.attackOrder?.targetUnitId ?? "");
  const [desiredEffect, setDesiredEffect] = useState(unit.attackOrder?.desiredEffect ?? 3);
  const [pkillThreshold, setPkillThreshold] = useState(unit.attackOrder?.pkillThreshold ?? 0.7);

  const selectedConfiguration = availableLoadouts.length > 0
    ? unitDefinitions
        .find((candidate) => candidate.id === unit.definitionId)
        ?.weaponConfigurations.find((cfg) => cfg.id === loadoutConfigurationId)
    : undefined;
  const loadedWeapons = (selectedConfiguration?.loadout ?? [])
    .filter((slot) => slot.initialQty > 0 || slot.maxQty > 0)
    .map((slot) => weaponDefs.get(slot.weaponId))
    .filter((weapon): weapon is WeaponDefLite => Boolean(weapon));
  const validTargets = filterValidEditorTargets(unit, units, loadedWeapons, unitDefinitions, {
    teamId,
    coalitionId,
  });
  const targetUnit = validTargets.find((candidate) => candidate.id === targetUnitId);
  const targetDef = unitDefinitions.find((candidate) => candidate.id === targetUnit?.definitionId);
  const selectedDefinition = unitDefinitions.find((candidate) => candidate.id === unit.definitionId);
  const hostBaseOptions = units.filter((candidate) => {
    if (candidate.id === unit.id) {
      return false;
    }
    if (!areFriendly({ teamId, coalitionId }, candidate)) {
      return false;
    }
    const candidateDefinition = unitDefinitions.find((definition) => definition.id === candidate.definitionId);
    return isHostPlatform(candidateDefinition);
  });
  const canAssignHostBase = selectedDefinition?.domain === 2;
  const isFacility = isHostPlatform(selectedDefinition);
  const hostedUnits = units.filter((candidate) => candidate.hostBaseId === unit.id);
  const countryOptions = Array.from(
    new Set([
      teamId,
      selectedDefinition?.nationOfOrigin ?? "",
      ...(selectedDefinition?.employedBy ?? []),
      ...units.map((candidate) => candidate.teamId),
    ].filter(Boolean)),
  ).sort();
  const loadoutAssessment =
    attackOrderType !== 0 && targetUnitId
      ? assessLoadoutAgainstTarget(selectedConfiguration, targetDef, weaponDefs, desiredEffect)
      : { severity: "none" as const, message: "" };

  useEffect(() => {
    if (!targetUnitId) {
      return;
    }
    if (!validTargets.some((candidate) => candidate.id === targetUnitId)) {
      setTargetUnitId("");
    }
  }, [targetUnitId, validTargets]);

  useEffect(() => {
    if (!hostBaseId) {
      return;
    }
    if (!hostBaseOptions.some((candidate) => candidate.id === hostBaseId)) {
      setHostBaseId("");
    }
  }, [hostBaseId, hostBaseOptions]);

  return (
    <div className="inline-edit-form">
      <div className="field">
        <label className="field-label">Designator</label>
        <input className="field-input" value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
      </div>
      <div className="field">
        <label className="field-label">Country</label>
        <select className="field-select" value={teamId} onChange={(e) => setTeamId(e.target.value)}>
          <option value="">Select country…</option>
          {countryOptions.map((code) => (
            <option key={code} value={code}>{formatCountry(code)}</option>
          ))}
        </select>
      </div>
      <div className="field">
        <label className="field-label">Coalition</label>
        <input
          className="field-input"
          value={coalitionId}
          onChange={(e) => setCoalitionId(e.target.value.trim().toUpperCase())}
          placeholder="Optional coalition tag"
        />
      </div>
      {canAssignHostBase && (
        <div className="field">
          <label className="field-label">Host Base</label>
          <select className="field-select" value={hostBaseId} onChange={(e) => setHostBaseId(e.target.value)}>
            <option value="">None assigned</option>
            {hostBaseOptions.map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {candidate.displayName}
              </option>
            ))}
          </select>
        </div>
      )}
      {!isFacility && (
        <>
          <div className="field-row">
            <div className="field">
              <label className="field-label">Heading (°)</label>
              <input className="field-input" type="number" min={0} max={359} value={heading} onChange={(e) => setHeading(Number(e.target.value))} />
            </div>
            <div className="field">
              <label className="field-label">Speed (m/s)</label>
              <input className="field-input" type="number" min={0} step="0.1" value={speed} onChange={(e) => setSpeed(Number(e.target.value))} />
            </div>
          </div>
          <div className="field">
            <label className="field-label">Strength (0–1)</label>
            <input className="field-input" type="number" min={0} max={1} step="0.01" value={strength} onChange={(e) => setStrength(Number(e.target.value))} />
          </div>
        </>
      )}
      {isFacility && (
        <div className="editor-facility-block">
          <div className="editor-facility-meta">
            <span className="editor-facility-tag">{selectedDefinition?.assetClass?.replaceAll("_", " ") ?? "facility"}</span>
            <span className="editor-facility-state">{BASE_OPS_STATE_LABEL[unit.baseOps?.state ?? 0]}</span>
          </div>
          <div className="editor-facility-copy">
            Fixed facility. Launch and recovery state is driven by base operations, not mobile unit movement or fuel.
          </div>
          <div className="editor-hosted-header">Stationed Units</div>
          {hostedUnits.length > 0 ? (
            <div className="editor-hosted-list">
              {hostedUnits.map((hosted) => (
                <div key={hosted.id} className="editor-hosted-row">
                  <span>{hosted.displayName}</span>
                  <span>{formatCountry(hosted.teamId || "UNK")}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="editor-hosted-empty">No units assigned to this base.</div>
          )}
        </div>
      )}
      {availableLoadouts.length > 0 && (
        <div className="field">
          <label className="field-label">Mission Loadout</label>
          <select className="field-select" value={loadoutConfigurationId} onChange={(e) => setLoadoutConfigurationId(e.target.value)}>
            {availableLoadouts.map((cfg) => (
              <option key={cfg.id} value={cfg.id}>{cfg.name}</option>
            ))}
          </select>
        </div>
      )}
      {!isFacility && (
        <>
          <div className="field">
            <label className="field-label">Engagement Behavior</label>
            <select className="field-select" value={engagementBehavior} onChange={(e) => setEngagementBehavior(Number(e.target.value))}>
              {ENGAGEMENT_BEHAVIORS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </div>
          <div className="field">
            <label className="field-label">Autonomous Pkill Threshold</label>
            <input className="field-input" type="number" min={0.1} max={0.99} step={0.05} value={engagementPkillThreshold} onChange={(e) => setEngagementPkillThreshold(Number(e.target.value))} />
          </div>
          <div className="field">
            <label className="field-label">Attack Task</label>
            <select className="field-select" value={attackOrderType} onChange={(e) => setAttackOrderType(Number(e.target.value))}>
              {ATTACK_ORDER_TYPES.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </div>
          {attackOrderType !== 0 && (
            <>
              <div className="field">
                <label className="field-label">Assigned Target</label>
                <select className="field-select" value={targetUnitId} onChange={(e) => setTargetUnitId(e.target.value)}>
                  <option value="">Select enemy unit…</option>
                  {validTargets.map((candidate) => (
                    <option key={candidate.id} value={candidate.id}>{candidate.displayName} · {(candidate.teamId || "UNK")}</option>
                  ))}
                </select>
              </div>
              <div className="field-row">
                <div className="field">
                  <label className="field-label">Desired Effect</label>
                  <select className="field-select" value={desiredEffect} onChange={(e) => setDesiredEffect(Number(e.target.value))}>
                    {DESIRED_EFFECTS.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </div>
                <div className="field">
                  <label className="field-label">Order Pkill Threshold</label>
                  <input className="field-input" type="number" min={0.1} max={0.99} step={0.05} value={pkillThreshold} onChange={(e) => setPkillThreshold(Number(e.target.value))} />
                </div>
              </div>
              {loadoutAssessment.severity !== "none" && loadoutAssessment.message && (
                <div className={`order-validation-note ${loadoutAssessment.severity}`}>{loadoutAssessment.message}</div>
              )}
            </>
          )}
        </>
      )}
      <div className="field-row" style={{ marginTop: 8 }}>
        {!isFacility && (
          <>
            <button className={`btn btn-sm${isRouteMode ? " btn-primary" : ""}`} onClick={onToggleRouteMode}>
              {isRouteMode ? "Finish Route" : "Edit Route"}
            </button>
            <button className="btn btn-sm" onClick={() => onSave({ moveOrder: undefined })}>
              Clear Route
            </button>
          </>
        )}
        <button
          className="btn btn-success btn-sm"
          disabled={loadoutAssessment.severity === "invalid"}
          onClick={() => onSave({
            displayName,
            teamId,
            coalitionId: coalitionId || teamId,
            hostBaseId: hostBaseId || undefined,
            loadoutConfigurationId,
            engagementBehavior,
            engagementPkillThreshold,
            heading,
            speed,
            combatEffectiveness: strength,
            personnelStrength: strength,
            equipmentStrength: strength,
            attackOrder: attackOrderType !== 0 && targetUnitId ? {
              orderType: attackOrderType,
              targetUnitId,
              desiredEffect,
              pkillThreshold,
            } : undefined,
          })}
        >
          Apply
        </button>
        <button className="btn btn-sm" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  );
}

export function PlacedUnits({
  routeEditUnitId,
  onToggleRouteMode,
}: {
  routeEditUnitId: string | null;
  onToggleRouteMode: (unitId: string) => void;
}) {
  const units = useEditorStore((s) => s.draft.units);
  const selectedUnitId = useEditorStore((s) => s.selectedUnitId);
  const editingUnitId = useEditorStore((s) => s.editingUnitId);
  const { selectUnit, setEditingUnit, deleteUnit } = useEditorStore();

  if (units.length === 0) {
    return <div className="placed-units-empty">Drag units from the palette above onto the map</div>;
  }

  return (
    <div className="placed-units-list">
      {units.map((u) => (
        <div key={u.id}>
          <div
            className={`unit-list-item${selectedUnitId === u.id ? " selected" : ""}`}
            onClick={() => {
              selectUnit(u.id);
              setEditingUnit(u.id);
            }}
          >
            <span className="unit-dot" style={{ background: teamColorHex(u.teamId) }} />
            <span className="unit-list-name">{u.displayName}</span>
            {u.attackOrder?.targetUnitId && (
              <span className="unit-list-order">
                ↦ {units.find((candidate) => candidate.id === u.attackOrder?.targetUnitId)?.displayName ?? "Target"}
              </span>
            )}
            {u.moveOrder?.waypoints?.length ? <span className="unit-list-order">Route {u.moveOrder.waypoints.length}</span> : null}
            <span className="unit-list-side">{u.teamId || "UNK"}</span>
            <span className="unit-list-actions">
              <button
                className={`btn btn-sm${routeEditUnitId === u.id ? " btn-primary" : ""}`}
                onClick={(e) => {
                  e.stopPropagation();
                  selectUnit(u.id);
                  setEditingUnit(u.id);
                  onToggleRouteMode(u.id);
                }}
              >
                {routeEditUnitId === u.id ? "Routing" : "Route"}
              </button>
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
        </div>
      ))}
    </div>
  );
}

export function SelectedUnitPanel({
  routeEditUnitId,
  onToggleRouteMode,
  onSavePatch,
}: {
  routeEditUnitId: string | null;
  onToggleRouteMode: (unitId: string) => void;
  onSavePatch: (unit: UnitDraft, patch: Partial<UnitDraft>) => Promise<boolean>;
}) {
  const units = useEditorStore((s) => s.draft.units);
  const unitDefinitions = useEditorStore((s) => s.unitDefinitions);
  const editingUnitId = useEditorStore((s) => s.editingUnitId);
  const { updateUnit, setEditingUnit } = useEditorStore();
  const editingUnit =
    editingUnitId && editingUnitId !== "new"
      ? units.find((u) => u.id === editingUnitId)
      : null;
  const [weaponDefs, setWeaponDefs] = useState<Map<string, WeaponDefLite>>(new Map());

  useEffect(() => {
    ListWeaponDefinitions()
      .then((rows) => {
        const next = new Map<string, WeaponDefLite>();
        rows.forEach((row) => {
          next.set(String(row.id ?? ""), {
            id: String(row.id ?? ""),
            domainTargets: Array.isArray(row.domain_targets) ? row.domain_targets.map(Number) : [],
            effectType: Number(row.effect_type ?? 0),
          });
        });
        setWeaponDefs(next);
      })
      .catch(console.error);
  }, []);

  if (!editingUnit) {
    return <div className="selected-unit-empty">Select a unit on the map or in the list to issue orders and edit routes.</div>;
  }

  return (
    <div className="selected-unit-panel">
      <div className="selected-unit-header">
        Commands
        <span className="selected-unit-name">{editingUnit.displayName}</span>
      </div>
      <InlineEditForm
        key={editingUnit.id}
        unit={editingUnit}
        units={units}
        unitDefinitions={unitDefinitions}
        weaponDefs={weaponDefs}
        availableLoadouts={
          unitDefinitions
            .find((def) => def.id === editingUnit.definitionId)
            ?.weaponConfigurations.map((cfg) => ({
              id: cfg.id,
              name: cfg.name || cfg.id,
            })) ?? []
        }
        isRouteMode={routeEditUnitId === editingUnit.id}
        onToggleRouteMode={() => onToggleRouteMode(editingUnit.id)}
        onSave={async (patch) => {
          if (!(await onSavePatch(editingUnit, patch))) {
            return;
          }
          updateUnit(editingUnit.id, patch);
          setEditingUnit(editingUnit.id);
        }}
        onCancel={() => setEditingUnit(null)}
      />
    </div>
  );
}
