import { useEffect, useMemo, useState } from "react";
import {
  ListUnitDefinitions,
  PreviewTargetEngagementOptions,
  RequestSync,
  SetUnitAttackOrder,
} from "../../../wailsjs/go/main/App";
import { useSimStore, type Unit } from "../../store/simStore";
import { areFriendly } from "../../utils/allegiance";
import { selectedPlayerTeam } from "../../utils/playerTeam";

type UnitDefinitionTargetMeta = {
  assetClass?: string;
  targetClass?: string;
};

function normalizeDefinitionId(definitionId: string | undefined): string {
  const raw = String(definitionId ?? "").trim();
  const idx = raw.lastIndexOf(":");
  return idx >= 0 ? raw.slice(idx + 1) : raw;
}

function isPreplannedTargetDefinition(definition: UnitDefinitionTargetMeta | undefined): boolean {
  if (!definition) return false;
  return definition.assetClass === "airbase"
    || definition.assetClass === "port"
    || definition.targetClass === "runway"
    || definition.targetClass === "hardened_infrastructure"
    || definition.targetClass === "soft_infrastructure"
    || definition.targetClass === "civilian_energy"
    || definition.targetClass === "civilian_water"
    || definition.targetClass === "sam_battery";
}

function formatUsd(value: number | undefined): string {
  const amount = Number(value ?? 0);
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    notation: amount >= 1_000_000 ? "compact" : "standard",
    maximumFractionDigits: amount >= 1_000_000 ? 1 : 0,
  }).format(amount);
}

function trackStatusForTarget(
  target: Unit | undefined,
  playerTeam: string,
  activeView: string,
  detections: Map<string, Set<string>>,
  detectionContacts: Map<string, Map<string, { shared: boolean; sourceTeam: string }>>,
): string {
  if (!target) return "Unknown";
  const contact = activeView !== "debug" ? detectionContacts.get(playerTeam)?.get(target.id) : undefined;
  if (contact?.shared) {
    return `Shared track · ${contact.sourceTeam}`;
  }
  if (activeView === "debug" || detections.get(playerTeam)?.has(target.id)) {
    return "Tracked";
  }
  return "Last Known";
}

export default function TargetPanel() {
  const selectedTargetId = useSimStore((s) => s.selectedTargetId);
  const units = useSimStore((s) => s.units);
  const humanControlledTeam = useSimStore((s) => s.humanControlledTeam);
  const detections = useSimStore((s) => s.detections);
  const detectionContacts = useSimStore((s) => s.detectionContacts);
  const selectTarget = useSimStore((s) => s.selectTarget);
  const selectUnit = useSimStore((s) => s.selectUnit);
  const target = selectedTargetId ? units.get(selectedTargetId) : undefined;
  const playerTeam = selectedPlayerTeam(humanControlledTeam);

  const [definitionMap, setDefinitionMap] = useState<Map<string, UnitDefinitionTargetMeta>>(new Map());
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [options, setOptions] = useState<Array<{
    shooterUnitId: string;
    shooterDisplayName: string;
    shooterTeamId: string;
    loadoutConfigurationId?: string;
    readyToFire: boolean;
    canPursue: boolean;
    hasTrack: boolean;
    weaponId?: string;
    reason?: string;
    reasonCode?: string;
    rangeToTargetM?: number;
    weaponRangeM?: number;
    fireProbability?: number;
    desiredEffectSupport: boolean;
    inStrikeCooldown: boolean;
    pathBlocked: boolean;
    pathReason?: string;
    engagementCostUsd?: number;
    expectedTargetValueUsd?: number;
    expectedValueExchangeUsd?: number;
  }>>([]);

  useEffect(() => {
    let cancelled = false;
    ListUnitDefinitions()
      .then((rows) => {
        if (cancelled) return;
        const defs = new Map<string, UnitDefinitionTargetMeta>();
        for (const row of rows) {
          const id = typeof row.id === "string" ? row.id : "";
          if (!id) continue;
          const meta = {
            assetClass: typeof row.asset_class === "string" ? row.asset_class : "",
            targetClass: typeof row.target_class === "string" ? row.target_class : "",
          };
          defs.set(id, meta);
          defs.set(normalizeDefinitionId(id), meta);
        }
        setDefinitionMap(defs);
      })
      .catch(console.error);
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    if (!selectedTargetId) {
      setOptions([]);
      setError("");
      return;
    }
    if (!playerTeam) {
      setOptions([]);
      setError("Select a PLAYER before evaluating shooters for this target.");
      return;
    }
    PreviewTargetEngagementOptions(selectedTargetId, playerTeam)
      .then((rows) => {
        if (cancelled) return;
        setOptions(rows ?? []);
        setError("");
      })
      .catch((err) => {
        if (cancelled) return;
        setOptions([]);
        setError(err instanceof Error ? err.message : String(err));
      });
    return () => {
      cancelled = true;
    };
  }, [selectedTargetId, units, playerTeam]);

  useEffect(() => {
    setSearchQuery("");
  }, [selectedTargetId]);

  const targetDefinition = target ? definitionMap.get(normalizeDefinitionId(target.definitionId)) : undefined;
  const targetTrackStatus = useMemo(
    () => trackStatusForTarget(target, playerTeam, playerTeam ? playerTeam : "debug", detections, detectionContacts),
    [detectionContacts, detections, playerTeam, target],
  );
  const playerReference = useMemo(
    () => Array.from(units.values()).find((unit) => (unit.teamId ?? "").trim().toUpperCase() === playerTeam),
    [playerTeam, units],
  );
  const currentAttackers = useMemo(
    () => Array.from(units.values())
      .filter((unit) => {
        if (!target || !playerReference) return false;
        return unit.attackOrder?.targetUnitId === target.id
          && areFriendly(
            { teamId: unit.teamId, coalitionId: unit.coalitionId },
            { teamId: playerReference.teamId, coalitionId: playerReference.coalitionId },
          );
      })
      .sort((a, b) => a.displayName.localeCompare(b.displayName)),
    [playerReference, target, units],
  );
  const filteredOptions = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    if (query === "") {
      return options;
    }
    return options.filter((option) => {
      const haystack = [
        option.shooterDisplayName,
        option.shooterTeamId,
        option.weaponId,
        option.loadoutConfigurationId,
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return haystack.includes(query);
    });
  }, [options, searchQuery]);

  const engage = async (shooterId: string) => {
    if (!target) return;
    const orderType = isPreplannedTargetDefinition(targetDefinition) ? 2 : 1;
    const desiredEffect = isPreplannedTargetDefinition(targetDefinition) ? 2 : 3;
    setBusy(true);
    try {
      const result = await SetUnitAttackOrder(shooterId, orderType, target.id, desiredEffect, 0.7);
      if (!result.success) {
        throw new Error(result.error || "Failed to assign attack order");
      }
      const sync = await RequestSync();
      if (!sync.success) {
        throw new Error(sync.error || "Failed to sync state");
      }
      selectTarget(null);
      selectUnit(shooterId);
    } catch (err) {
      console.error(err);
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  if (!target) return null;

  return (
    <div className="target-panel">
      <div className="unit-panel-header">
        <span className="unit-side-indicator target-indicator" />
        <span className="unit-display-name">{target.displayName}</span>
        <button className="unit-panel-close" onClick={() => selectTarget(null)} aria-label="Close target panel">×</button>
      </div>
      <div className="unit-panel-body">
        <div className="unit-full-name">{target.fullName}</div>
        <div className="track-source-note">{targetTrackStatus}</div>
        <div className="target-summary-grid">
          <div className="unit-stat-row">
            <span className="stat-label">Country</span>
            <span className="stat-value">{target.teamId || "UNK"}</span>
          </div>
          <div className="unit-stat-row">
            <span className="stat-label">Damage</span>
            <span className="stat-value">{target.damageState === 4 ? "Destroyed" : target.damageState === 3 ? "Mission Killed" : target.damageState === 2 ? "Damaged" : "Operational"}</span>
          </div>
          <div className="unit-stat-row">
            <span className="stat-label">Type</span>
            <span className="stat-value">{targetDefinition?.assetClass?.replaceAll("_", " ") || targetDefinition?.targetClass?.replaceAll("_", " ") || "combat unit"}</span>
          </div>
          <div className="unit-stat-row">
            <span className="stat-label">Target Value</span>
            <span className="stat-value">{formatUsd(options[0]?.expectedTargetValueUsd)}</span>
          </div>
          <div className="unit-stat-row">
            <span className="stat-label">Position</span>
            <span className="stat-value">{target.position.lat.toFixed(2)}°, {target.position.lon.toFixed(2)}°</span>
          </div>
        </div>
        {currentAttackers.length > 0 && (
          <div className="weapon-list">
            <div className="weapon-list-header">Current Attackers</div>
            <div className="move-hint">
              {currentAttackers.map((unit) => unit.displayName).join(", ")}
            </div>
          </div>
        )}
        {error && <div className="path-warning-note strike-warning-note">{error}</div>}
        <div className="weapon-list">
          <div className="weapon-list-header-row">
            <div className="weapon-list-header">Friendly Shooters</div>
            <input
              className="target-search-input"
              type="search"
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              placeholder="Search shooters"
              aria-label="Search friendly shooters"
            />
          </div>
          {options.length === 0 ? (
            <div className="move-hint">
              {playerTeam
                ? `No friendly shooters were found for PLAYER ${playerTeam} against this target.`
                : "Select a PLAYER before evaluating shooters for this target."}
            </div>
          ) : filteredOptions.length === 0 ? (
            <div className="move-hint">No friendly shooters match that search.</div>
          ) : filteredOptions.map((option) => (
            <div key={option.shooterUnitId} className="target-option-card">
              <div className="target-option-row">
                <div>
                  <div className="target-option-title">{option.shooterDisplayName}</div>
                  <div className="target-option-subtitle">
                    {option.shooterTeamId}
                    {option.weaponId ? ` · ${option.weaponId}` : ""}
                    {option.loadoutConfigurationId ? ` · ${option.loadoutConfigurationId}` : ""}
                  </div>
                </div>
                <button
                  className="cancel-order-btn"
                  disabled={busy || (!option.readyToFire && !option.canPursue) || option.pathBlocked}
                  onClick={() => engage(option.shooterUnitId).catch(console.error)}
                >
                  {option.readyToFire ? "Engage" : option.canPursue ? "Pursue" : "Blocked"}
                </button>
              </div>
              <div className="target-option-metrics">
                <span>{option.readyToFire ? "Ready" : option.canPursue ? "Can Pursue" : option.reason || "Blocked"}</span>
                <span>Pk {Math.round((option.fireProbability ?? 0) * 100)}%</span>
                <span>Cost {formatUsd(option.engagementCostUsd)}</span>
                <span>Value Exchange {formatUsd(option.expectedValueExchangeUsd)}</span>
              </div>
              {(option.reason || option.pathReason) && (
                <div className="move-hint">
                  {option.pathReason || option.reason}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
