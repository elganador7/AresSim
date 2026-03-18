import {
  Cartesian3,
  Color,
  ConstantProperty,
  Entity,
  HeightReference,
  HorizontalOrigin,
  NearFarScalar,
  PolylineDashMaterialProperty,
  VerticalOrigin,
} from "cesium";
import type { ExplosionFx, Munition, Unit, WeaponDef } from "../../store/simStore";
import { useSimStore } from "../../store/simStore";
import { getUnitBillboardUrl } from "../../utils/unitBillboard";
import { inferUnitTeamCode } from "../../utils/unitTeams";

export type ActiveView = string;

export interface DefInfo {
  generalType: number;
  detectionRangeM: number;
  shortName: string;
  teamCode: string;
}

export type Detections = Map<string, Set<string>>;
export type MunitionDetections = Map<string, Set<string>>;

export const ROUTE_COLOR: Record<string, Color> = {
  Blue: Color.fromCssColorString("#60a5fa"),
  Red: Color.fromCssColorString("#f87171"),
  Neutral: Color.fromCssColorString("#fcd34d"),
};

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
    || inferUnitTeamCode(shooter.id, shooter.side, defInfoRefFallback(shooter.definitionId, {}));
  if (shooterTeam === view) {
    return true;
  }
  return munitionDetections.get(view)?.has(munition.id) ?? false;
}

function defInfoRefFallback(definitionId: string, defInfo: Record<string, DefInfo>) {
  return defInfo[definitionId];
}

export function teamForUnit(unit: Unit, defInfo: Record<string, DefInfo>): string {
  return unit.teamId?.trim().toUpperCase()
    || inferUnitTeamCode(unit.id, unit.side, defInfo[unit.definitionId]);
}

export function isVisible(
  unit: Unit,
  view: ActiveView,
  detections: Detections,
  defInfo: Record<string, DefInfo>,
): boolean {
  if (view === "debug") return true;
  const teamCode = teamForUnit(unit, defInfo);
  if (teamCode === view) {
    return true;
  }
  return detections.get(view)?.has(unit.id) ?? false;
}

export function isTrack(unit: Unit, view: ActiveView, defInfo: Record<string, DefInfo>): boolean {
  if (view === "debug") return false;
  return teamForUnit(unit, defInfo) !== view;
}

export function canMove(unit: Unit, view: ActiveView, defInfo: Record<string, DefInfo>): boolean {
  if (view === "debug") return true;
  return teamForUnit(unit, defInfo) === view;
}

export function makeUnitEntity(unit: Unit, generalType: number, shortName: string): Entity {
  return new Entity({
    id: unit.id,
    position: Cartesian3.fromDegrees(unit.position.lon, unit.position.lat, unit.position.altMsl),
    show: true,
    billboard: {
      image: getUnitBillboardUrl(generalType, unit.side, shortName),
      width: 62,
      height: 62,
      verticalOrigin: VerticalOrigin.CENTER,
      horizontalOrigin: HorizontalOrigin.CENTER,
      scaleByDistance: new NearFarScalar(1.5e5, 1.2, 8e6, 0.4),
      disableDepthTestDistance: Number.POSITIVE_INFINITY,
      heightReference: HeightReference.CLAMP_TO_GROUND,
    },
  });
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
  mapCommandMode: { type: "none" | "route" | "target_pick"; unitId: string | null },
  units: Map<string, Unit>,
  selectedId: string | null,
  view: ActiveView,
  defInfo: Record<string, DefInfo>,
) {
  if (!container) return;
  if (mapCommandMode.type === "target_pick" && mapCommandMode.unitId) {
    container.style.cursor = "crosshair";
    return;
  }
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
