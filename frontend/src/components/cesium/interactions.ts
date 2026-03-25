import {
  Cartesian2,
  Cartesian3,
  Cartographic,
  Entity,
  Math as CesiumMath,
  ScreenSpaceEventType,
  Viewer,
} from "cesium";
import type { MutableRefObject } from "react";
import { AppendMoveWaypoint, MoveUnit, UpdateMoveWaypoint } from "../../../wailsjs/go/main/App";
import { useSimStore } from "../../store/simStore";
import { areHostile } from "../../utils/allegiance";
import { selectedPlayerTeam } from "../../utils/playerTeam";
import type { DefInfo } from "./helpers";
import { canMove, ensureBridgeSuccess } from "./helpers";

function pickLatLon(viewer: Viewer, position: Cartesian2): { lat: number; lon: number } | null {
  const ray = viewer.camera.getPickRay(position);
  if (!ray) return null;
  const pos = viewer.scene.globe.pick(ray, viewer.scene);
  if (!pos) return null;
  const carto = Cartographic.fromCartesian(pos);
  return {
    lat: CesiumMath.toDegrees(carto.latitude),
    lon: CesiumMath.toDegrees(carto.longitude),
  };
}

function previewDraggedWaypoint(
  viewer: Viewer,
  unitID: string,
  waypointIndex: number,
  lat: number,
  lon: number,
) {
  const waypointEntity = viewer.entities.getById(`${unitID}_wp_${waypointIndex}`);
  if (waypointEntity) {
    (waypointEntity.position as unknown as { setValue: (p: Cartesian3) => void })
      .setValue(Cartesian3.fromDegrees(lon, lat));
  }

  const unit = useSimStore.getState().units.get(unitID);
  if (!unit?.moveOrder) return;
  const positions: Cartesian3[] = [
    Cartesian3.fromDegrees(unit.position.lon, unit.position.lat),
    ...unit.moveOrder.waypoints.map((wp, idx) =>
      idx === waypointIndex
        ? Cartesian3.fromDegrees(lon, lat)
        : Cartesian3.fromDegrees(wp.lon, wp.lat),
    ),
  ];
  const routeEntity = viewer.entities.getById(`${unitID}_route`);
  if (routeEntity?.polyline) {
    (routeEntity.polyline.positions as unknown as { setValue: (p: Cartesian3[]) => void }).setValue(positions);
  }
  const destEntity = viewer.entities.getById(`${unitID}_dest`);
  if (destEntity && waypointIndex === unit.moveOrder.waypoints.length - 1) {
    (destEntity.position as unknown as { setValue: (p: Cartesian3) => void })
      .setValue(Cartesian3.fromDegrees(lon, lat));
  }
}

export function setupCesiumInteractions(
  viewer: Viewer,
  defInfoRef: MutableRefObject<Record<string, DefInfo>>,
  draggingWaypointRef: MutableRefObject<{ unitId: string; waypointIndex: number } | null>,
  suppressClickRef: MutableRefObject<boolean>,
) {
  viewer.screenSpaceEventHandler.setInputAction(
    (evt: { position: Cartesian2 }) => {
      const { mapCommandMode } = useSimStore.getState();
      if (mapCommandMode.type !== "route" || !mapCommandMode.unitId) return;
      const picked = viewer.scene.pick(evt.position);
      if (!(picked?.id instanceof Entity)) return;
      const pickedEntity = picked.id as Entity;
      const waypointUnitId = pickedEntity.properties?.waypointUnitId?.getValue?.();
      const waypointIndex = pickedEntity.properties?.waypointIndex?.getValue?.();
      if (typeof waypointUnitId === "string" && typeof waypointIndex === "number" && waypointUnitId === mapCommandMode.unitId) {
        draggingWaypointRef.current = { unitId: waypointUnitId, waypointIndex };
        suppressClickRef.current = true;
        viewer.scene.screenSpaceCameraController.enableRotate = false;
      }
    },
    ScreenSpaceEventType.LEFT_DOWN,
  );

  viewer.screenSpaceEventHandler.setInputAction(
    (evt: { endPosition: Cartesian2 }) => {
      const drag = draggingWaypointRef.current;
      const next = pickLatLon(viewer, evt.endPosition);
      if (!next) return;
      if (drag) {
        previewDraggedWaypoint(viewer, drag.unitId, drag.waypointIndex, next.lat, next.lon);
      }
    },
    ScreenSpaceEventType.MOUSE_MOVE,
  );

  viewer.screenSpaceEventHandler.setInputAction(
    (evt: { position: Cartesian2 }) => {
      const drag = draggingWaypointRef.current;
      if (!drag) return;
      const next = pickLatLon(viewer, evt.position);
      draggingWaypointRef.current = null;
      viewer.scene.screenSpaceCameraController.enableRotate = true;
      if (!next) return;
      previewDraggedWaypoint(viewer, drag.unitId, drag.waypointIndex, next.lat, next.lon);
      UpdateMoveWaypoint(drag.unitId, drag.waypointIndex, next.lat, next.lon)
        .then(ensureBridgeSuccess)
        .catch((error) => {
          console.error(error);
          alert(error instanceof Error ? error.message : String(error));
        });
    },
    ScreenSpaceEventType.LEFT_UP,
  );

  viewer.screenSpaceEventHandler.setInputAction(
    (evt: { position: Cartesian2 }) => {
      if (suppressClickRef.current) {
        suppressClickRef.current = false;
        return;
      }
      if (draggingWaypointRef.current) {
        return;
      }
      const {
        units,
        selectedUnitId,
        selectedTargetId,
        mapCommandMode,
        activeView,
        humanControlledTeam,
        selectUnit,
        selectTarget,
        startRouteEdit,
        clearMapCommandMode,
        setSelectedRoutePreview,
      } = useSimStore.getState();

      const picked = viewer.scene.pick(evt.position);
      if (picked?.id instanceof Entity) {
        const clickedId = (picked.id as Entity).id;
        if (units.has(clickedId)) {
          const clickedUnit = units.get(clickedId);
          const playerTeam = selectedPlayerTeam(humanControlledTeam);
          if (!playerTeam) {
            return;
          }
          const ownsClickedUnit = clickedUnit && (clickedUnit.teamId ?? "").trim().toUpperCase() === playerTeam;
          if (!ownsClickedUnit) {
            const playerReference = Array.from(units.values()).find((candidate) => (candidate.teamId ?? "").trim().toUpperCase() === playerTeam);
            if (clickedUnit && playerReference && areHostile(playerReference, clickedUnit)) {
              const nextSelectedTargetId = selectedTargetId === clickedId ? null : clickedId;
              selectTarget(nextSelectedTargetId);
              clearMapCommandMode();
              setSelectedRoutePreview(null);
            }
            return;
          }
          const nextSelectedId = selectedUnitId === clickedId ? null : clickedId;
          selectUnit(nextSelectedId);
          selectTarget(null);
          clearMapCommandMode();
          setSelectedRoutePreview(null);
          if (nextSelectedId && clickedUnit && canMove(clickedUnit, activeView, defInfoRef.current)) {
            startRouteEdit(nextSelectedId);
          }
          return;
        }
      }

      if (!selectedUnitId) return;
      const unit = units.get(selectedUnitId);
      if (!unit || !canMove(unit, activeView, defInfoRef.current)) return;

      const next = pickLatLon(viewer, evt.position);
      if (!next) return;
      const { lat, lon } = next;

      if (mapCommandMode.type === "route" && mapCommandMode.unitId === selectedUnitId) {
        AppendMoveWaypoint(selectedUnitId, lat, lon)
          .then(ensureBridgeSuccess)
          .then(() => setSelectedRoutePreview(null))
          .catch((error) => {
            console.error(error);
            alert(error instanceof Error ? error.message : String(error));
          });
        return;
      }

      MoveUnit(selectedUnitId, lat, lon)
        .then(ensureBridgeSuccess)
        .then(() => {
          setSelectedRoutePreview(null);
          selectUnit(null);
          selectTarget(null);
        })
        .catch((error) => {
          console.error(error);
          alert(error instanceof Error ? error.message : String(error));
        });
    },
    ScreenSpaceEventType.LEFT_CLICK,
  );
}
