import { describe, expect, it } from "vitest";
import { resolveUnitVisualProfile } from "./visualModels";

describe("unit visual profiles", () => {
  it("uses 3d proxy profiles for major combat units", () => {
    expect(resolveUnitVisualProfile(10).kind).toBe("box");
    expect(resolveUnitVisualProfile(43).kind).toBe("box");
    expect(resolveUnitVisualProfile(50).kind).toBe("ellipsoid");
    expect(resolveUnitVisualProfile(73, 1, false, "radar_site").kind).toBe("ellipsoid");
  });

  it("prefers specific visual model ids over generic type inference", () => {
    expect(resolveUnitVisualProfile(10, 2, false, "combat_unit", "f35").id).toBe("f35");
    expect(resolveUnitVisualProfile(73, 1, false, "combat_unit", "s300").id).toBe("s300");
    expect(resolveUnitVisualProfile(43, 3, false, "combat_unit", "frigate").id).toBe("frigate");
  });

  it("keeps large fixed infrastructure on billboard fallback for now", () => {
    const profile = resolveUnitVisualProfile(93, 1, true, "airbase");
    expect(profile.kind).toBe("billboard");
  });
});
