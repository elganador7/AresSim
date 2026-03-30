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
