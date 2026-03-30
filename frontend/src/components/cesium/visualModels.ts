export type UnitVisualKind = "billboard" | "box" | "ellipsoid";

export interface UnitVisualProfile {
  id: string;
  kind: UnitVisualKind;
  dimensions: {
    x: number;
    y: number;
    z: number;
  };
  billboardNearM: number;
}

const BILLBOARD_ONLY: UnitVisualProfile = {
  id: "billboard",
  kind: "billboard",
  dimensions: { x: 0, y: 0, z: 0 },
  billboardNearM: 0,
};

function profile(id: string, kind: UnitVisualKind, x: number, y: number, z: number, billboardNearM: number): UnitVisualProfile {
  return {
    id,
    kind,
    dimensions: { x, y, z },
    billboardNearM,
  };
}

export function resolveUnitVisualProfile(
  generalType: number,
  domain?: number,
  stationary?: boolean,
  assetClass?: string,
  visualModelId?: string,
): UnitVisualProfile {
  if (stationary && assetClass !== "radar_site" && assetClass !== "c2_site") {
    return BILLBOARD_ONLY;
  }

  switch ((visualModelId ?? "").trim().toLowerCase()) {
    case "ddg51":
      return profile("ddg51", "box", 156, 22, 25, 980_000);
    case "f35":
      return profile("f35", "box", 15, 10, 2.6, 760_000);
    case "f22":
      return profile("f22", "box", 17, 11, 3, 760_000);
    case "f15":
      return profile("f15", "box", 20, 13, 3.5, 760_000);
    case "f16":
      return profile("f16", "box", 15, 10, 3, 720_000);
    case "aew":
      return profile("aew", "box", 34, 30, 8, 950_000);
    case "tanker":
      return profile("tanker", "box", 36, 28, 8, 950_000);
    case "transport":
      return profile("transport", "box", 30, 26, 8, 900_000);
    case "patriot":
      return profile("patriot", "box", 14, 6, 5, 360_000);
    case "thaad":
      return profile("thaad", "box", 15, 6, 5.5, 370_000);
    case "iron-dome":
      return profile("iron-dome", "box", 11, 5, 4.5, 340_000);
    case "davids-sling":
      return profile("davids-sling", "box", 12, 5, 5, 350_000);
    case "arrow-2":
    case "arrow-3":
      return profile(visualModelId!, "box", 13, 5, 5.5, 360_000);
    case "spyder":
      return profile("spyder", "box", 10, 4.5, 4.5, 330_000);
    case "barak-mx":
      return profile("barak-mx", "box", 12, 5, 5, 350_000);
    case "s300":
      return profile("s300", "box", 16, 6, 6, 390_000);
    case "s400":
      return profile("s400", "box", 17, 6, 6, 400_000);
    case "bavar373":
      return profile("bavar373", "box", 16, 6, 6, 390_000);
    case "khordad15":
      return profile("khordad15", "box", 13, 5, 5, 350_000);
    case "tor":
      return profile("tor", "box", 9, 4, 4, 320_000);
    case "missile-launcher":
      return profile("missile-launcher", "box", 13, 5, 5, 350_000);
    case "radar-site":
      return profile("radar-site", "ellipsoid", 12, 12, 15, 360_000);
    case "frigate":
      return profile("frigate", "box", 138, 20, 22, 920_000);
    case "destroyer":
      return profile("destroyer", "box", 160, 22, 25, 980_000);
    case "corvette":
      return profile("corvette", "box", 92, 16, 16, 820_000);
    case "patrol-vessel":
      return profile("patrol-vessel", "box", 68, 12, 12, 720_000);
    case "lcs":
      return profile("lcs", "box", 118, 17, 18, 850_000);
    case "carrier":
      return profile("carrier", "box", 290, 50, 30, 1_450_000);
    case "submarine":
      return profile("submarine", "ellipsoid", 56, 8, 8, 860_000);
    case "command-site":
      return profile("command-site", "box", 12, 8, 6, 300_000);
    case "airbase":
    case "port":
      return BILLBOARD_ONLY;
    default:
      break;
  }

  switch (generalType) {
    case 10:
    case 11:
      return profile("fighter", "box", 18, 12, 3, 700_000);
    case 12:
      return profile("attack-aircraft", "box", 20, 14, 4, 700_000);
    case 13:
      return profile("bomber", "box", 30, 22, 6, 850_000);
    case 14:
      return profile("transport", "box", 28, 24, 7, 850_000);
    case 15:
      return profile("maritime-patrol", "box", 26, 22, 6, 850_000);
    case 16:
      return profile("aew", "box", 30, 28, 7, 900_000);
    case 17:
      return profile("tanker", "box", 32, 26, 7, 900_000);
    case 18:
      return profile("recon-aircraft", "box", 24, 18, 5, 850_000);
    case 20:
    case 21:
    case 22:
      return profile("helicopter", "box", 14, 14, 5, 500_000);
    case 30:
      return profile("recon-drone", "box", 8, 6, 2, 450_000);
    case 31:
      return profile("strike-drone", "box", 10, 8, 3, 500_000);
    case 40:
      return profile("carrier", "box", 280, 48, 28, 1_400_000);
    case 41:
      return profile("cruiser", "box", 180, 24, 26, 1_000_000);
    case 42:
      return profile("destroyer", "box", 155, 22, 24, 950_000);
    case 43:
      return profile("frigate", "box", 135, 20, 20, 900_000);
    case 44:
      return profile("corvette", "box", 95, 16, 16, 800_000);
    case 45:
      return profile("patrol-vessel", "box", 65, 12, 12, 700_000);
    case 46:
      return profile("amphib", "box", 210, 34, 26, 1_200_000);
    case 47:
      return profile("mine-warfare", "box", 70, 14, 12, 750_000);
    case 50:
    case 51:
    case 52:
      return profile("submarine", "ellipsoid", 55, 8, 8, 850_000);
    case 60:
      return profile("tank", "box", 10, 4, 3, 350_000);
    case 61:
    case 62:
      return profile("armored-vehicle", "box", 9, 4, 3.5, 350_000);
    case 63:
      return profile("recon-vehicle", "box", 7, 3.5, 3, 320_000);
    case 70:
    case 71:
      return profile("artillery", "box", 11, 4, 4, 320_000);
    case 72:
      return profile("rocket-artillery", "box", 12, 4, 4.5, 350_000);
    case 73:
      if (assetClass === "radar_site") {
        return profile("radar-site", "ellipsoid", 12, 12, 14, 350_000);
      }
      return profile("air-defense", "box", 12, 5, 5, 350_000);
    case 91:
      return profile("logistics", "box", 10, 4, 4, 320_000);
    case 93:
      return profile("command", "box", 10, 5, 5, 320_000);
    case 94:
      return profile("ew", "ellipsoid", 9, 9, 9, 320_000);
    default:
      break;
  }

  switch (domain) {
    case 2:
      return profile("air-default", "box", 18, 12, 4, 700_000);
    case 3:
      return profile("sea-default", "box", 120, 18, 18, 850_000);
    case 4:
      return profile("sub-default", "ellipsoid", 50, 8, 8, 800_000);
    case 1:
      return stationary ? BILLBOARD_ONLY : profile("land-default", "box", 10, 4, 4, 320_000);
    default:
      return BILLBOARD_ONLY;
  }
}
