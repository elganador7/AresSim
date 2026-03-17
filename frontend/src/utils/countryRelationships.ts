import { getCountriesAlongSegment } from "./theaterCountries";

export interface CountryRelationshipLike {
  fromCountry: string;
  toCountry: string;
  shareIntel: boolean;
  airspaceTransitAllowed: boolean;
  airspaceStrikeAllowed: boolean;
  defensivePositioningAllowed: boolean;
}

export interface RelationshipRule {
  shareIntel: boolean;
  airspaceTransitAllowed: boolean;
  airspaceStrikeAllowed: boolean;
  defensivePositioningAllowed: boolean;
}

export interface GeoPointLike {
  lat: number;
  lon: number;
}

export function normalizeCountryCode(code: string): string {
  return code.trim().toUpperCase();
}

export function getRelationshipRule(
  relationships: CountryRelationshipLike[],
  fromCountry: string,
  toCountry: string,
): RelationshipRule {
  const from = normalizeCountryCode(fromCountry);
  const to = normalizeCountryCode(toCountry);
  if (!from || !to || from === to) {
    return {
      shareIntel: true,
      airspaceTransitAllowed: true,
      airspaceStrikeAllowed: true,
      defensivePositioningAllowed: true,
    };
  }
  const direct = relationships.find(
    (rel) => normalizeCountryCode(rel.fromCountry) === from && normalizeCountryCode(rel.toCountry) === to,
  );
  if (direct) {
    return {
      shareIntel: !!direct.shareIntel,
      airspaceTransitAllowed: !!direct.airspaceTransitAllowed,
      airspaceStrikeAllowed: !!direct.airspaceStrikeAllowed,
      defensivePositioningAllowed: !!direct.defensivePositioningAllowed,
    };
  }
  return {
    shareIntel: false,
    airspaceTransitAllowed: false,
    airspaceStrikeAllowed: false,
    defensivePositioningAllowed: false,
  };
}

function findBlockedCountryAlongPath(
  relationships: CountryRelationshipLike[],
  fromCountry: string,
  points: GeoPointLike[],
  mode: "transit" | "strike",
): { country: string; legIndex: number } | null {
  const owner = normalizeCountryCode(fromCountry);
  if (!owner || points.length < 2) {
    return null;
  }
  for (let idx = 0; idx < points.length - 1; idx += 1) {
    for (const country of getCountriesAlongSegment(points[idx], points[idx + 1])) {
      if (!country || country === owner) {
        continue;
      }
      const relationship = getRelationshipRule(relationships, owner, country);
      const blocked = mode === "strike"
        ? !relationship.airspaceStrikeAllowed
        : !relationship.airspaceTransitAllowed;
      if (blocked) {
        return { country, legIndex: idx + 1 };
      }
    }
  }
  return null;
}

export function explainBlockedTransitPath(
  relationships: CountryRelationshipLike[],
  fromCountry: string,
  points: GeoPointLike[],
): string | null {
  const blocked = findBlockedCountryAlongPath(relationships, fromCountry, points, "transit");
  if (!blocked) {
    return null;
  }
  return `Transit blocked by ${blocked.country} airspace on leg ${blocked.legIndex}.`;
}

export function explainBlockedStrikePath(
  relationships: CountryRelationshipLike[],
  fromCountry: string,
  points: GeoPointLike[],
): string | null {
  const blocked = findBlockedCountryAlongPath(relationships, fromCountry, points, "strike");
  if (!blocked) {
    return null;
  }
  return `Strike blocked by ${blocked.country} airspace on leg ${blocked.legIndex}.`;
}
