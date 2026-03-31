import {
  Cartesian3,
  Color,
  ConstantProperty,
  Entity,
  HorizontalOrigin,
  NearFarScalar,
  VerticalOrigin,
  Viewer,
} from "cesium";
import type { ExplosionFx, Munition } from "../../store/simStore";
import {
  type MunitionDetections,
  EXPLOSION_ENTITY_PREFIX,
  IMPACT_COLOR,
  KILL_COLOR,
  MUNITION_COLOR,
  MUNITION_ENTITY_PREFIX,
  getExplosionBillboard,
  isMunitionVisible,
} from "./helpers";

export interface EffectsLayerSyncState {
  munitionEntityIds: Set<string>;
  explosionEntityIds: Set<string>;
}

export function createEffectsLayerSyncState(): EffectsLayerSyncState {
  return {
    munitionEntityIds: new Set<string>(),
    explosionEntityIds: new Set<string>(),
  };
}

export function syncMunition(
  viewer: Viewer,
  munition: Munition,
  view: string,
  munitionDetections: MunitionDetections,
  syncState?: EffectsLayerSyncState,
) {
  const entityId = `${MUNITION_ENTITY_PREFIX}${munition.id}`;
  syncState?.munitionEntityIds.add(entityId);
  const visible = isMunitionVisible(munition, view, munitionDetections);
  const pos = Cartesian3.fromDegrees(munition.lon, munition.lat, munition.altMsl);

  const existing = viewer.entities.getById(entityId);
  if (existing) {
    (existing.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(pos);
    existing.show = visible;
  } else {
    viewer.entities.add(new Entity({
      id: entityId,
      show: visible,
      position: pos,
      point: {
        pixelSize: 6,
        color: MUNITION_COLOR,
        outlineColor: Color.WHITE,
        outlineWidth: 1,
        disableDepthTestDistance: Number.POSITIVE_INFINITY,
      },
    }));
  }
}

export function syncMunitions(
  viewer: Viewer,
  munitions: Map<string, Munition>,
  view: string,
  munitionDetections: MunitionDetections,
  syncState?: EffectsLayerSyncState,
) {
  munitions.forEach((munition) => syncMunition(viewer, munition, view, munitionDetections, syncState));
  const liveIds = new Set(
    Array.from(munitions.keys()).map((id) => `${MUNITION_ENTITY_PREFIX}${id}`),
  );
  const trackedIds = syncState?.munitionEntityIds ?? liveIds;
  Array.from(trackedIds).forEach((id) => {
    if (!liveIds.has(id)) {
      viewer.entities.removeById(id);
      syncState?.munitionEntityIds.delete(id);
    }
  });
}

export function syncExplosion(viewer: Viewer, explosion: ExplosionFx, syncState?: EffectsLayerSyncState) {
  const entityId = `${EXPLOSION_ENTITY_PREFIX}${explosion.id}`;
  syncState?.explosionEntityIds.add(entityId);
  const pos = Cartesian3.fromDegrees(explosion.lon, explosion.lat, explosion.altMsl);
  const pixelSize = explosion.kind === "kill" ? 16 : 10;
  const color = explosion.kind === "kill" ? KILL_COLOR : IMPACT_COLOR;
  const existing = viewer.entities.getById(entityId);
  if (existing) {
    (existing.position as unknown as { setValue: (p: Cartesian3) => void }).setValue(pos);
    existing.show = true;
    if (existing.billboard) {
      existing.billboard.image = new ConstantProperty(getExplosionBillboard(explosion.kind));
    }
    if (existing.point) {
      existing.point.pixelSize = new ConstantProperty(pixelSize);
      existing.point.color = new ConstantProperty(color.withAlpha(0.95));
    }
  } else {
    viewer.entities.add(new Entity({
      id: entityId,
      show: true,
      position: pos,
      billboard: {
        image: getExplosionBillboard(explosion.kind),
        width: explosion.kind === "kill" ? 58 : 42,
        height: explosion.kind === "kill" ? 58 : 42,
        verticalOrigin: VerticalOrigin.CENTER,
        horizontalOrigin: HorizontalOrigin.CENTER,
        disableDepthTestDistance: Number.POSITIVE_INFINITY,
        scaleByDistance: new NearFarScalar(2e5, 1.0, 8e6, 0.5),
      },
      point: {
        pixelSize,
        color: color.withAlpha(0.95),
        outlineColor: Color.WHITE.withAlpha(0.85),
        outlineWidth: 1,
        disableDepthTestDistance: Number.POSITIVE_INFINITY,
      },
    }));
  }
}

export function syncExplosions(
  viewer: Viewer,
  explosions: Map<string, ExplosionFx>,
  syncState?: EffectsLayerSyncState,
) {
  explosions.forEach((explosion) => syncExplosion(viewer, explosion, syncState));
  const liveIds = new Set(
    Array.from(explosions.keys()).map((id) => `${EXPLOSION_ENTITY_PREFIX}${id}`),
  );
  const trackedIds = syncState?.explosionEntityIds ?? liveIds;
  Array.from(trackedIds).forEach((id) => {
    if (!liveIds.has(id)) {
      viewer.entities.removeById(id);
      syncState?.explosionEntityIds.delete(id);
    }
  });
}
