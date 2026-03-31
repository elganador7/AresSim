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
import { reportError } from "../../utils/errors";
import { selectedPlayerTeam } from "../../utils/playerTeam";
import type { DefInfo } from "./helpers";
import { canMove, ensureBridgeSuccess } from "./helpers";

export function resolvePickedEntity(picked: unknown): Entity | null {
  const value = picked as {
    id?: unknown;
    primitive?: { id?: unknown };
  } | null;
  if (!value) {
    return null;
  }
  if (value.id instanceof Entity) {
    return value.id;
  }
  if (value.primitive?.id instanceof Entity) {
    return value.primitive.id;
  }
  return null;
}

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

type InteractionSelectionState = {
  selectedUnitId: string | null;
  selectedTargetId: string | null;
  selectedOilNodeId: string | null;
  selectedOilEdgeId: string | null;
  oilGraphPresent: boolean;
  activeView: string;
  humanControlledTeam: string;
  units: Map<string, ReturnType<typeof useSimStore.getState>["units"] extends Map<string, infer U> ? U : never>;
};

type InteractionSelectionActions = {
  selectUnit: (id: string | null) => void;
  selectTarget: (id: string | null) => void;
  selectOilNode: (id: string | null) => void;
  selectOilEdge: (id: string | null) => void;
  clearMapCommandMode: () => void;
  setSelectedRoutePreview: (preview: null) => void;
};

function clearTransientSelections(actions: InteractionSelectionActions) {
  actions.clearMapCommandMode();
  actions.setSelectedRoutePreview(null);
}

export function handlePickedSelection(
  clickedId: string | null,
  state: InteractionSelectionState,
  actions: InteractionSelectionActions,
): boolean {
  if (clickedId) {
    if (clickedId.startsWith("oil_node_")) {
      const nodeId = clickedId.slice("oil_node_".length);
      const nextId = state.selectedOilNodeId === nodeId ? null : nodeId;
      actions.selectOilNode(nextId);
      clearTransientSelections(actions);
      return true;
    }
    if (clickedId.startsWith("oil_edge_")) {
      const rawEdgeId = clickedId.slice("oil_edge_".length);
      const edgeId = rawEdgeId.split("__part_")[0];
      const nextId = state.selectedOilEdgeId === edgeId ? null : edgeId;
      actions.selectOilEdge(nextId);
      clearTransientSelections(actions);
      return true;
    }
    if (state.units.has(clickedId)) {
      const clickedUnit = state.units.get(clickedId);
      const playerTeam = selectedPlayerTeam(state.humanControlledTeam);
      const clickedTeam = (clickedUnit?.operatorTeamId ?? clickedUnit?.teamId ?? "").trim().toUpperCase();
      const fallbackViewTeam = state.activeView !== "debug" ? state.activeView.trim().toUpperCase() : "";
      if (!playerTeam) {
        if (clickedUnit && (state.activeView === "debug" || clickedTeam === fallbackViewTeam)) {
          const nextSelectedId = state.selectedUnitId === clickedId ? null : clickedId;
          actions.selectUnit(nextSelectedId);
          actions.selectTarget(null);
          clearTransientSelections(actions);
        }
        return true;
      }
      const ownsClickedUnit = clickedUnit && clickedTeam === playerTeam;
      if (!ownsClickedUnit) {
        const playerReference = Array.from(state.units.values()).find((candidate) => (candidate.teamId ?? "").trim().toUpperCase() === playerTeam);
        if (clickedUnit && playerReference && areHostile(playerReference, clickedUnit)) {
          const nextSelectedTargetId = state.selectedTargetId === clickedId ? null : clickedId;
          actions.selectTarget(nextSelectedTargetId);
          clearTransientSelections(actions);
        }
        return true;
      }
      const nextSelectedId = state.selectedUnitId === clickedId ? null : clickedId;
      actions.selectUnit(nextSelectedId);
      actions.selectTarget(null);
      clearTransientSelections(actions);
      return true;
    }
  }

  if (state.selectedOilNodeId || state.selectedOilEdgeId) {
    actions.selectOilNode(null);
    actions.selectOilEdge(null);
    if (state.oilGraphPresent) {
      return true;
    }
  }

  return false;
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
      const pickedEntity = resolvePickedEntity(viewer.scene.pick(evt.position));
      if (!pickedEntity) return;
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
          reportError("CesiumInteractions:UpdateMoveWaypoint", error);
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
        oilGraph,
        selectedUnitId,
        selectedTargetId,
        selectedOilNodeId,
        selectedOilEdgeId,
        mapCommandMode,
        activeView,
        humanControlledTeam,
        selectUnit,
        selectTarget,
        selectOilNode,
        selectOilEdge,
        clearMapCommandMode,
        setSelectedRoutePreview,
      } = useSimStore.getState();

      const pickedEntity = resolvePickedEntity(viewer.scene.pick(evt.position));
      const handledSelection = handlePickedSelection(
        pickedEntity ? String(pickedEntity.id ?? "") : null,
        {
          selectedUnitId,
          selectedTargetId,
          selectedOilNodeId,
          selectedOilEdgeId,
          oilGraphPresent: !!oilGraph,
          activeView,
          humanControlledTeam,
          units,
        },
        {
          selectUnit,
          selectTarget,
          selectOilNode,
          selectOilEdge,
          clearMapCommandMode,
          setSelectedRoutePreview,
        },
      );
      if (handledSelection) {
        return;
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
            reportError("CesiumInteractions:AppendMoveWaypoint", error);
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
          reportError("CesiumInteractions:MoveUnit", error);
          alert(error instanceof Error ? error.message : String(error));
        });
    },
    ScreenSpaceEventType.LEFT_CLICK,
  );
}
