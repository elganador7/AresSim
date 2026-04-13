import { beforeEach, describe, expect, it } from "vitest";
import { blankUnit, useEditorStore, type ScenarioDraft } from "./editorStore";

function resetStore() {
  const draft: ScenarioDraft = {
    id: "draft-1",
    name: "Draft",
    description: "",
    classification: "UNCLASSIFIED",
    author: "",
    startTimeUnix: 0,
    version: "1.0.0",
    tickRateHz: 10,
    timeScale: 1,
    weatherState: 1,
    visibilityKm: 40,
    windSpeedMps: 5,
    temperatureC: 20,
    relationships: [],
    units: [],
  };
  useEditorStore.setState({
    draft,
    selectedUnitId: null,
    editingUnitId: null,
    isDirty: false,
    pendingDrop: null,
    pendingPosition: null,
    unitDefinitions: [],
    selectedCountryCode: "",
  });
}

describe("editorStore H3 consistency", () => {
  beforeEach(resetStore);

  it("recomputes unit H3 fields when coordinates change without replacement cells", () => {
    const unit = {
      ...blankUnit(25.2854, 51.5310),
      id: "unit-1",
      h3Cell: "8c2a100d291b5ff",
      h3ParentCell: "8b2a100d291ffff",
    };
    useEditorStore.getState().addUnit(unit);
    useEditorStore.getState().updateUnit("unit-1", { lat: 26.5660, lon: 56.2500 });
    const updated = useEditorStore.getState().draft.units[0];
    expect(updated.h3Cell).toBeDefined();
    expect(updated.h3ParentCell).toBeDefined();
    expect(updated.h3Cell).not.toBe("8c2a100d291b5ff");
    expect(updated.h3ParentCell).not.toBe("8b2a100d291ffff");
  });

  it("preserves unit H3 fields when non-spatial properties change", () => {
    const unit = {
      ...blankUnit(25.2854, 51.5310),
      id: "unit-1",
      h3Cell: "8c2a100d291b5ff",
      h3ParentCell: "8b2a100d291ffff",
    };
    useEditorStore.getState().addUnit(unit);
    useEditorStore.getState().updateUnit("unit-1", { displayName: "Updated" });
    const updated = useEditorStore.getState().draft.units[0];
    expect(updated.h3Cell).toBe("8c2a100d291b5ff");
    expect(updated.h3ParentCell).toBe("8b2a100d291ffff");
  });

  it("recomputes waypoint H3 fields when waypoint coordinates change without replacement cells", () => {
    const unit = {
      ...blankUnit(25.2854, 51.5310),
      id: "unit-1",
      moveOrder: {
        waypoints: [
          {
            lat: 26.5660,
            lon: 56.2500,
            altMsl: 1000,
            h3Cell: "8c3b0e8e33a89ff",
            h3ParentCell: "8b3b0e8e33a9fff",
          },
        ],
      },
    };
    useEditorStore.getState().addUnit(unit);
    useEditorStore.getState().updateUnit("unit-1", {
      moveOrder: {
        waypoints: [{ lat: 27.0, lon: 56.8, altMsl: 1000 }],
      },
    });
    const waypoint = useEditorStore.getState().draft.units[0].moveOrder?.waypoints[0];
    expect(waypoint?.h3Cell).toBeDefined();
    expect(waypoint?.h3ParentCell).toBeDefined();
    expect(waypoint?.h3Cell).not.toBe("8c3b0e8e33a89ff");
    expect(waypoint?.h3ParentCell).not.toBe("8b3b0e8e33a9fff");
  });

  it("assigns H3 fields to new units and newly added waypoints", () => {
    const unit = {
      ...blankUnit(25.2854, 51.5310),
      id: "unit-1",
      h3Cell: undefined,
      h3ParentCell: undefined,
    };
    useEditorStore.getState().addUnit(unit);
    let stored = useEditorStore.getState().draft.units[0];
    expect(stored.h3Cell).toBeDefined();
    expect(stored.h3ParentCell).toBeDefined();

    useEditorStore.getState().updateUnit("unit-1", {
      moveOrder: {
        waypoints: [{ lat: 26.5660, lon: 56.2500, altMsl: 1000 }],
      },
    });
    stored = useEditorStore.getState().draft.units[0];
    expect(stored.moveOrder?.waypoints[0].h3Cell).toBeDefined();
    expect(stored.moveOrder?.waypoints[0].h3ParentCell).toBeDefined();
  });

  it("backfills unit and waypoint H3 fields when loading a draft without them", () => {
    const draft: ScenarioDraft = {
      id: "draft-2",
      name: "Draft",
      description: "",
      classification: "UNCLASSIFIED",
      author: "",
      startTimeUnix: 0,
      version: "1.0.0",
      tickRateHz: 10,
      timeScale: 1,
      weatherState: 1,
      visibilityKm: 40,
      windSpeedMps: 5,
      temperatureC: 20,
      relationships: [],
      units: [
        {
          ...blankUnit(25.2854, 51.5310),
          id: "unit-1",
          h3Cell: undefined,
          h3ParentCell: undefined,
          moveOrder: {
            waypoints: [{ lat: 26.5660, lon: 56.2500, altMsl: 1000 }],
          },
        },
      ],
    };
    useEditorStore.getState().loadDraft(draft);
    const loaded = useEditorStore.getState().draft.units[0];
    expect(loaded.h3Cell).toBeDefined();
    expect(loaded.h3ParentCell).toBeDefined();
    expect(loaded.moveOrder?.waypoints[0].h3Cell).toBeDefined();
    expect(loaded.moveOrder?.waypoints[0].h3ParentCell).toBeDefined();
  });
});
