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

export function syncMunition(
  viewer: Viewer,
  munition: Munition,
  view: string,
  munitionDetections: MunitionDetections,
) {
  const entityId = `${MUNITION_ENTITY_PREFIX}${munition.id}`;
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
) {
  munitions.forEach((munition) => syncMunition(viewer, munition, view, munitionDetections));
  const liveIds = new Set(
    Array.from(munitions.keys()).map((id) => `${MUNITION_ENTITY_PREFIX}${id}`),
  );
  Array.from(viewer.entities.values)
    .map((entity) => entity.id as string)
    .filter((id) => id.startsWith(MUNITION_ENTITY_PREFIX) && !liveIds.has(id))
    .forEach((id) => viewer.entities.removeById(id));
}

export function syncExplosion(viewer: Viewer, explosion: ExplosionFx) {
  const entityId = `${EXPLOSION_ENTITY_PREFIX}${explosion.id}`;
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

export function syncExplosions(viewer: Viewer, explosions: Map<string, ExplosionFx>) {
  explosions.forEach((explosion) => syncExplosion(viewer, explosion));
  const liveIds = new Set(
    Array.from(explosions.keys()).map((id) => `${EXPLOSION_ENTITY_PREFIX}${id}`),
  );
  Array.from(viewer.entities.values)
    .map((entity) => entity.id as string)
    .filter((id) => id.startsWith(EXPLOSION_ENTITY_PREFIX) && !liveIds.has(id))
    .forEach((id) => viewer.entities.removeById(id));
}
