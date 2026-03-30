import type { SimStore } from "./simTypes";

export const selectTopBarState = (state: SimStore) => ({
  scenarioName: state.scenarioName,
  scenarioState: state.scenarioState,
  simSeconds: state.simSeconds,
  timeScale: state.timeScale,
  tickNumber: state.tickNumber,
  scores: state.scores,
  units: state.units,
  humanControlledTeam: state.humanControlledTeam,
  oilLayerVisible: state.oilLayerVisible,
  oilGraph: state.oilGraph,
  oilLoadError: state.oilLoadError,
  requestOilFocus: state.requestOilFocus,
  setActiveView: state.setActiveView,
  setHumanControlledTeam: state.setHumanControlledTeam,
  setOilLayerVisible: state.setOilLayerVisible,
});

export const selectOilPanelState = (state: SimStore) => ({
  oilGraph: state.oilGraph,
  selectedOilNodeId: state.selectedOilNodeId,
  selectedOilEdgeId: state.selectedOilEdgeId,
  selectOilNode: state.selectOilNode,
  selectOilEdge: state.selectOilEdge,
});

export const selectCesiumGlobeState = (state: SimStore) => ({
  mapCommandMode: state.mapCommandMode,
  oilGraph: state.oilGraph,
  oilFocusToken: state.oilFocusToken,
});

export const selectUnitPanelState = (state: SimStore) => ({
  selectedUnitId: state.selectedUnitId,
  units: state.units,
  weaponDefs: state.weaponDefs,
  humanControlledTeam: state.humanControlledTeam,
  simSeconds: state.simSeconds,
  selectUnit: state.selectUnit,
  routePreview: state.selectedRoutePreview,
  strikePreview: state.selectedStrikePreview,
  setRoutePreview: state.setSelectedRoutePreview,
  setStrikePreview: state.setSelectedStrikePreview,
  mapCommandMode: state.mapCommandMode,
  startRouteEdit: state.startRouteEdit,
  clearMapCommandMode: state.clearMapCommandMode,
});

export const selectTargetPanelState = (state: SimStore) => ({
  selectedTargetId: state.selectedTargetId,
  units: state.units,
  humanControlledTeam: state.humanControlledTeam,
  detections: state.detections,
  detectionContacts: state.detectionContacts,
  selectTarget: state.selectTarget,
  selectUnit: state.selectUnit,
});

export const selectViewSwitcherState = (state: SimStore) => ({
  activeView: state.activeView,
  setActiveView: state.setActiveView,
  humanControlledTeam: state.humanControlledTeam,
  units: state.units,
});
