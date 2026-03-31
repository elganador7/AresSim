import { describe, expect, it, vi } from "vitest";
import type { Unit } from "../../store/simTypes";
import { handlePickedSelection } from "./interactions";

function makeUnit(id: string, teamId = "USA", operatorTeamId?: string): Unit {
  return {
    id,
    displayName: id,
    fullName: id,
    teamId,
    coalitionId: teamId === "IRN" ? "COALITION_IRAN" : "COALITION_WEST",
    operatorTeamId,
    natoPendingSymbol: "",
    definitionId: "def-1",
    damageState: 1,
    position: { lat: 0, lon: 0, altMsl: 0, heading: 0, speed: 0 },
    status: {
      personnelStrength: 100,
      equipmentStrength: 100,
      combatEffectiveness: 1,
      fuelLevelLiters: 100,
      morale: 1,
      fatigue: 0,
      isActive: true,
      suppressed: false,
      disrupted: false,
      routing: false,
    },
    weapons: [],
  };
}

function makeActions() {
  return {
    selectUnit: vi.fn(),
    selectTarget: vi.fn(),
    selectOilNode: vi.fn(),
    selectOilEdge: vi.fn(),
    clearMapCommandMode: vi.fn(),
    setSelectedRoutePreview: vi.fn(),
  };
}

describe("handlePickedSelection", () => {
  it("toggles oil node selection and clears transient route UI", () => {
    const actions = makeActions();
    const handled = handlePickedSelection("oil_node_project-1", {
      selectedUnitId: null,
      selectedTargetId: null,
      selectedOilNodeId: null,
      selectedOilEdgeId: null,
      oilGraphPresent: true,
      activeView: "debug",
      humanControlledTeam: "",
      units: new Map(),
    }, actions);
    expect(handled).toBe(true);
    expect(actions.selectOilNode).toHaveBeenCalledWith("project-1");
    expect(actions.clearMapCommandMode).toHaveBeenCalled();
    expect(actions.setSelectedRoutePreview).toHaveBeenCalledWith(null);
  });

  it("normalizes multipart oil edge ids before selecting", () => {
    const actions = makeActions();
    const handled = handlePickedSelection("oil_edge_edge-1__part_3", {
      selectedUnitId: null,
      selectedTargetId: null,
      selectedOilNodeId: null,
      selectedOilEdgeId: null,
      oilGraphPresent: true,
      activeView: "debug",
      humanControlledTeam: "",
      units: new Map(),
    }, actions);
    expect(handled).toBe(true);
    expect(actions.selectOilEdge).toHaveBeenCalledWith("edge-1");
  });

  it("selects friendly units for the human-controlled team", () => {
    const actions = makeActions();
    const units = new Map<string, Unit>([["u1", makeUnit("u1", "USA")]]);
    const handled = handlePickedSelection("u1", {
      selectedUnitId: null,
      selectedTargetId: null,
      selectedOilNodeId: null,
      selectedOilEdgeId: null,
      oilGraphPresent: false,
      activeView: "debug",
      humanControlledTeam: "USA",
      units,
    }, actions);
    expect(handled).toBe(true);
    expect(actions.selectUnit).toHaveBeenCalledWith("u1");
    expect(actions.selectTarget).toHaveBeenCalledWith(null);
  });

  it("selects hostile units as targets for the player team", () => {
    const actions = makeActions();
    const units = new Map<string, Unit>([
      ["u1", makeUnit("u1", "USA")],
      ["u2", makeUnit("u2", "IRN")],
    ]);
    const handled = handlePickedSelection("u2", {
      selectedUnitId: "u1",
      selectedTargetId: null,
      selectedOilNodeId: null,
      selectedOilEdgeId: null,
      oilGraphPresent: false,
      activeView: "debug",
      humanControlledTeam: "USA",
      units,
    }, actions);
    expect(handled).toBe(true);
    expect(actions.selectTarget).toHaveBeenCalledWith("u2");
  });

  it("uses active view when no human-controlled team is set", () => {
    const actions = makeActions();
    const units = new Map<string, Unit>([["u1", makeUnit("u1", "ISR")]]);
    const handled = handlePickedSelection("u1", {
      selectedUnitId: null,
      selectedTargetId: null,
      selectedOilNodeId: null,
      selectedOilEdgeId: null,
      oilGraphPresent: false,
      activeView: "ISR",
      humanControlledTeam: "",
      units,
    }, actions);
    expect(handled).toBe(true);
    expect(actions.selectUnit).toHaveBeenCalledWith("u1");
  });

  it("clears oil selections on empty map click and consumes the click when an oil graph is visible", () => {
    const actions = makeActions();
    const handled = handlePickedSelection(null, {
      selectedUnitId: null,
      selectedTargetId: null,
      selectedOilNodeId: "project-1",
      selectedOilEdgeId: "edge-1",
      oilGraphPresent: true,
      activeView: "debug",
      humanControlledTeam: "",
      units: new Map(),
    }, actions);
    expect(handled).toBe(true);
    expect(actions.selectOilNode).toHaveBeenCalledWith(null);
    expect(actions.selectOilEdge).toHaveBeenCalledWith(null);
  });

  it("allows fallthrough after clearing stale oil selections when no oil graph is present", () => {
    const actions = makeActions();
    const handled = handlePickedSelection(null, {
      selectedUnitId: null,
      selectedTargetId: null,
      selectedOilNodeId: "project-1",
      selectedOilEdgeId: null,
      oilGraphPresent: false,
      activeView: "debug",
      humanControlledTeam: "",
      units: new Map(),
    }, actions);
    expect(handled).toBe(false);
    expect(actions.selectOilNode).toHaveBeenCalledWith(null);
  });
});
