import { describe, expect, it } from "vitest";
import { modelAssetFor } from "./modelAssets";

describe("model assets", () => {
  it("returns undefined for unknown assets", () => {
    expect(modelAssetFor("unknown-platform")).toBeUndefined();
  });

  it("can disable specific real model assets while keeping the pipeline", () => {
    expect(modelAssetFor("ddg51")).toBeUndefined();
  });
});
