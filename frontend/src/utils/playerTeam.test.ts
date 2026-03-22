import { describe, expect, it } from "vitest";
import { normalizeTeamCode, selectedPlayerTeam } from "./playerTeam";

describe("playerTeam", () => {
  it("normalizes team codes", () => {
    expect(normalizeTeamCode(" irn ")).toBe("IRN");
  });

  it("requires an explicitly selected player team", () => {
    expect(selectedPlayerTeam("")).toBe("");
    expect(selectedPlayerTeam(undefined)).toBe("");
  });
});
