import {
  BillboardGraphics,
  BoxGraphics,
  Cartesian3,
  Color,
  ColorMaterialProperty,
  ConstantProperty,
  DistanceDisplayCondition,
  EllipsoidGraphics,
  Entity,
  HeadingPitchRoll,
  HeightReference,
  HorizontalOrigin,
  Math as CesiumMath,
  ModelGraphics,
  NearFarScalar,
  PolylineDashMaterialProperty,
  ShadowMode,
  Transforms,
  VerticalOrigin,
} from "cesium";
import type { ExplosionFx, Munition, Unit, WeaponDef } from "../../store/simStore";
import { useSimStore } from "../../store/simStore";
import { selectedPlayerTeam } from "../../utils/playerTeam";
import { getUnitBillboardUrl } from "../../utils/unitBillboard";
import { inferUnitTeamCode } from "../../utils/unitTeams";
import { areHostile } from "../../utils/allegiance";
import { modelAssetFor } from "./modelAssets";
import { resolveUnitVisualProfile } from "./visualModels";

export type ActiveView = string;

export interface DefInfo {
  generalType: number;
  domain?: number;
  detectionRangeM: number;
  shortName: string;
  teamCode: string;
  stationary?: boolean;
  assetClass?: string;
  coalitionId?: string;
  visualModelId?: string;
}

export type Detections = Map<string, Set<string>>;
export type MunitionDetections = Map<string, Set<string>>;

export const MUNITION_ENTITY_PREFIX = "mun_";
export const EXPLOSION_ENTITY_PREFIX = "explosion_";
export const TRACK_LINK_PREFIX = "tracklink_";
export const SENSOR_COLOR = Color.fromCssColorString("#0f9fb8");
export const BLOCKED_ROUTE_COLOR = Color.fromCssColorString("#ef4444");
export const STRIKE_PATH_COLOR = Color.fromCssColorString("#f59e0b");
export const MUNITION_COLOR = Color.fromCssColorString("#f97316");
export const IMPACT_COLOR = Color.fromCssColorString("#fb923c");
export const KILL_COLOR = Color.fromCssColorString("#facc15");

export function ensureBridgeSuccess(result: { success: boolean; error?: string }) {
  if (!result.success) {
    throw new Error(result.error || "Command failed");
  }
}

export function maxWeaponRangeM(unit: Unit, weaponDefs: Map<string, WeaponDef>): number {
  let best = 0;
  for (const ws of unit.weapons) {
    if (ws.currentQty <= 0) continue;
    const def = weaponDefs.get(ws.weaponId);
    if (def && def.rangeM > best) best = def.rangeM;
  }
  return best;
}

export function isMunitionVisible(
  munition: Munition,
  view: ActiveView,
  munitionDetections: MunitionDetections,
): boolean {
  if (view === "debug") return true;
  const { units } = useSimStore.getState();
  const shooter = units.get(munition.shooterId);
  if (!shooter) return false;
  const shooterTeam = shooter.teamId?.trim().toUpperCase()
    || inferUnitTeamCode(shooter.id, defInfoRefFallback(shooter.definitionId, {})?.teamCode ?? "");
  if (shooterTeam === view) {
    return true;
  }
  return munitionDetections.get(view)?.has(munition.id) ?? false;
}

function defInfoRefFallback(definitionId: string, defInfo: Record<string, DefInfo>) {
  return defInfo[normalizeDefinitionId(definitionId)];
}

export function normalizeDefinitionId(definitionId: string): string {
  const raw = String(definitionId ?? "").trim();
  const idx = raw.lastIndexOf(":");
  return idx >= 0 ? raw.slice(idx + 1) : raw;
}

export function definitionInfoFor(defInfo: Record<string, DefInfo>, definitionId: string): DefInfo | undefined {
  return defInfo[normalizeDefinitionId(definitionId)];
}

export function teamForUnit(unit: Unit, defInfo: Record<string, DefInfo>): string {
  return unit.operatorTeamId?.trim().toUpperCase()
    || unit.teamId?.trim().toUpperCase()
    || inferUnitTeamCode(unit.id, definitionInfoFor(defInfo, unit.definitionId)?.teamCode ?? "");
}

function sharesIntelIntoView(sourceTeam: string, view: ActiveView): boolean {
  const { relationships } = useSimStore.getState();
  return relationships.some((relationship) =>
    relationship.shareIntel
    && relationship.fromCountry.trim().toUpperCase() === sourceTeam
    && relationship.toCountry.trim().toUpperCase() === view,
  );
}

export function relationshipColorHex(unit: Unit, humanControlledTeam: string, units: Map<string, Unit>): string {
  const { activeView } = useSimStore.getState();
  const playerTeam = selectedPlayerTeam(humanControlledTeam) || (activeView !== "debug" ? activeView.toUpperCase() : "");
  if (!playerTeam) {
    return "#94a3b8";
  }
  const unitTeam = (unit.operatorTeamId ?? unit.teamId ?? "").trim().toUpperCase();
  if (unitTeam === playerTeam) {
    return "#3b82f6";
  }
  const playerReference = Array.from(units.values()).find((candidate) => ((candidate.operatorTeamId ?? candidate.teamId) ?? "").trim().toUpperCase() === playerTeam);
  if (!playerReference) {
    return "#94a3b8";
  }
  return areHostile(playerReference, unit) ? "#ef4444" : "#94a3b8";
}

export function routeColorForUnit(unit: Unit, humanControlledTeam: string, units: Map<string, Unit>): Color {
  return Color.fromCssColorString(relationshipColorHex(unit, humanControlledTeam, units));
}

function isAlwaysVisibleFixedSite(unit: Unit, defInfo: Record<string, DefInfo>): boolean {
  const def = definitionInfoFor(defInfo, unit.definitionId);
  if (!def?.stationary) {
    return false;
  }
  switch (def.assetClass) {
    case "airbase":
    case "port":
    case "c2_site":
    case "radar_site":
    case "oil_field":
    case "pipeline_node":
    case "desalination_plant":
    case "power_plant":
      return true;
    default:
      return false;
  }
}

export function isVisible(
  unit: Unit,
  view: ActiveView,
  detections: Detections,
  defInfo: Record<string, DefInfo>,
): boolean {
  if (view === "debug") return true;
  if (isAlwaysVisibleFixedSite(unit, defInfo)) {
    return true;
  }
  const teamCode = teamForUnit(unit, defInfo);
  if (teamCode === view) {
    return true;
  }
  if (teamCode && sharesIntelIntoView(teamCode, view)) {
    return true;
  }
  return detections.get(view)?.has(unit.id) ?? false;
}

export function isTrack(unit: Unit, view: ActiveView, defInfo: Record<string, DefInfo>): boolean {
  if (view === "debug") return false;
  if (isAlwaysVisibleFixedSite(unit, defInfo)) {
    return false;
  }
  const teamCode = teamForUnit(unit, defInfo);
  if (teamCode === view) {
    return false;
  }
  return !(teamCode && sharesIntelIntoView(teamCode, view));
}

export function canMove(unit: Unit, view: ActiveView, defInfo: Record<string, DefInfo>): boolean {
  const { humanControlledTeam } = useSimStore.getState();
  const controlTeam = selectedPlayerTeam(humanControlledTeam);
  if (!controlTeam) return view === "debug";
  return teamForUnit(unit, defInfo) === controlTeam;
}

export function makeUnitEntity(
  unit: Unit,
  generalType: number,
  shortName: string,
  domain?: number,
  stationary?: boolean,
  assetClass?: string,
  visualModelId?: string,
): Entity {
  const entity = new Entity({
    id: unit.id,
    position: Cartesian3.fromDegrees(unit.position.lon, unit.position.lat, unit.position.altMsl),
    show: true,
  });
  applyUnitEntityVisual(entity, unit, generalType, shortName, domain, stationary, assetClass, visualModelId, false, 1.0);
  return entity;
}

export function applyUnitEntityVisual(
  entity: Entity,
  unit: Unit,
  generalType: number,
  shortName: string,
  domain: number | undefined,
  stationary: boolean | undefined,
  assetClass: string | undefined,
  visualModelId: string | undefined,
  isSelected: boolean,
  trackAlpha: number,
) {
  const { humanControlledTeam, units } = useSimStore.getState();
  const frameColorHex = relationshipColorHex(unit, humanControlledTeam, units);
  const frameColor = Color.fromCssColorString(frameColorHex);
  const profile = resolveUnitVisualProfile(generalType, domain, stationary, assetClass, visualModelId);
  const modelAsset = modelAssetFor(visualModelId);
  entity.orientation = new ConstantProperty(
    Transforms.headingPitchRollQuaternion(
      Cartesian3.fromDegrees(unit.position.lon, unit.position.lat, unit.position.altMsl),
      new HeadingPitchRoll(CesiumMath.toRadians(unit.position.heading || 0), 0, 0),
    ),
  );
  entity.billboard = new BillboardGraphics({
      image: new ConstantProperty(getUnitBillboardUrl(generalType, frameColorHex, shortName)),
      width: new ConstantProperty(62),
      height: new ConstantProperty(62),
      verticalOrigin: new ConstantProperty(VerticalOrigin.CENTER),
      horizontalOrigin: new ConstantProperty(HorizontalOrigin.CENTER),
      scaleByDistance: new ConstantProperty(new NearFarScalar(1.5e5, 1.2, 8e6, 0.4)),
      disableDepthTestDistance: new ConstantProperty(Number.POSITIVE_INFINITY),
      heightReference: new ConstantProperty(HeightReference.CLAMP_TO_GROUND),
      scale: new ConstantProperty(isSelected ? 1.2 : 1.0),
      color: new ConstantProperty(Color.WHITE.withAlpha(trackAlpha)),
      distanceDisplayCondition: new ConstantProperty(new DistanceDisplayCondition(
        profile.kind === "billboard" && !modelAsset ? 0 : profile.billboardNearM * 0.6,
        8e6,
      )),
    });

  if (modelAsset) {
    entity.model = new ModelGraphics({
      uri: new ConstantProperty(modelAsset.uri),
      scale: new ConstantProperty(modelAsset.scale),
      minimumPixelSize: new ConstantProperty(modelAsset.minimumPixelSize),
      maximumScale: modelAsset.maximumScale != null ? new ConstantProperty(modelAsset.maximumScale) : undefined,
      color: new ConstantProperty(frameColor.withAlpha(0.22)),
      silhouetteColor: new ConstantProperty(frameColor.withAlpha(0.95)),
      silhouetteSize: new ConstantProperty(isSelected ? 2.5 : 0),
      shadows: new ConstantProperty(ShadowMode.DISABLED),
      runAnimations: new ConstantProperty(false),
      distanceDisplayCondition: new ConstantProperty(
        new DistanceDisplayCondition(0, modelAsset.closeDisplayM ?? profile.billboardNearM),
      ),
    });
    entity.box = undefined;
    entity.ellipsoid = undefined;
    return;
  }
  entity.model = undefined;

  if (profile.kind === "box") {
    entity.box = new BoxGraphics({
      dimensions: new ConstantProperty(
        new Cartesian3(
          profile.dimensions.x * (isSelected ? 1.08 : 1),
          profile.dimensions.y * (isSelected ? 1.08 : 1),
          profile.dimensions.z * (isSelected ? 1.08 : 1),
        ),
      ),
      material: new ColorMaterialProperty(frameColor.withAlpha(Math.max(0.55, trackAlpha))),
      outline: new ConstantProperty(true),
      outlineColor: new ConstantProperty(Color.WHITE.withAlpha(0.85)),
      distanceDisplayCondition: new ConstantProperty(new DistanceDisplayCondition(0, profile.billboardNearM)),
    });
    entity.ellipsoid = undefined;
  } else if (profile.kind === "ellipsoid") {
    entity.ellipsoid = new EllipsoidGraphics({
      radii: new ConstantProperty(
        new Cartesian3(
          profile.dimensions.x * (isSelected ? 1.08 : 1),
          profile.dimensions.y * (isSelected ? 1.08 : 1),
          profile.dimensions.z * (isSelected ? 1.08 : 1),
        ),
      ),
      material: new ColorMaterialProperty(frameColor.withAlpha(Math.max(0.55, trackAlpha))),
      outline: new ConstantProperty(true),
      outlineColor: new ConstantProperty(Color.WHITE.withAlpha(0.85)),
      distanceDisplayCondition: new ConstantProperty(new DistanceDisplayCondition(0, profile.billboardNearM)),
    });
    entity.box = undefined;
  } else {
    entity.box = undefined;
    entity.ellipsoid = undefined;
  }
}

export function getExplosionBillboard(kind: ExplosionFx["kind"]): string {
  const center = kind === "kill" ? "#fff7b0" : "#ffe2bf";
  const middle = kind === "kill" ? "#f59e0b" : "#fb923c";
  const outer = kind === "kill" ? "#dc2626" : "#ea580c";
  const size = kind === "kill" ? 72 : 54;
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 72 72">
    <defs>
      <radialGradient id="g" cx="50%" cy="50%" r="50%">
        <stop offset="0%" stop-color="${center}" stop-opacity="1"/>
        <stop offset="38%" stop-color="${middle}" stop-opacity="0.95"/>
        <stop offset="70%" stop-color="${outer}" stop-opacity="0.82"/>
        <stop offset="100%" stop-color="${outer}" stop-opacity="0"/>
      </radialGradient>
    </defs>
    <circle cx="36" cy="36" r="30" fill="url(#g)"/>
    <circle cx="36" cy="36" r="12" fill="${center}" fill-opacity="0.95"/>
  </svg>`;
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
}

export function updateMapCursor(
  container: HTMLDivElement | null,
  mapCommandMode: { type: "none" | "route"; unitId: string | null },
  units: Map<string, Unit>,
  selectedId: string | null,
  view: ActiveView,
  defInfo: Record<string, DefInfo>,
) {
  if (!container) return;
  if (mapCommandMode.type === "route" && mapCommandMode.unitId) {
    container.style.cursor = "copy";
    return;
  }
  const unit = selectedId ? units.get(selectedId) : null;
  const moveable = unit ? canMove(unit, view, defInfo) : false;
  container.style.cursor = moveable ? "crosshair" : "default";
}

export function makeDashedPolyline(
  start: { lat: number; lon: number },
  end: { lat: number; lon: number },
  color: Color,
  dashLength: number,
) {
  return {
    positions: new ConstantProperty([
      Cartesian3.fromDegrees(start.lon, start.lat),
      Cartesian3.fromDegrees(end.lon, end.lat),
    ]),
    material: new PolylineDashMaterialProperty({ color, dashLength }),
    clampToGround: false,
  };
}
