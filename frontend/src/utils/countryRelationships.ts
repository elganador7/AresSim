export interface CountryRelationshipLike {
  fromCountry: string;
  toCountry: string;
  shareIntel: boolean;
  airspaceTransitAllowed: boolean;
  airspaceStrikeAllowed: boolean;
  defensivePositioningAllowed: boolean;
  maritimeTransitAllowed: boolean;
  maritimeStrikeAllowed: boolean;
}

export interface RelationshipRule {
  shareIntel: boolean;
  airspaceTransitAllowed: boolean;
  airspaceStrikeAllowed: boolean;
  defensivePositioningAllowed: boolean;
  maritimeTransitAllowed: boolean;
  maritimeStrikeAllowed: boolean;
}

export type CountryCoalitionMap = Record<string, string>;

export function normalizeCountryCode(code: string): string {
  return code.trim().toUpperCase();
}

export function getRelationshipRule(
  relationships: CountryRelationshipLike[],
  fromCountry: string,
  toCountry: string,
  countryCoalitions?: CountryCoalitionMap,
): RelationshipRule {
  const from = normalizeCountryCode(fromCountry);
  const to = normalizeCountryCode(toCountry);
  if (!from || !to || from === to) {
    return {
      shareIntel: true,
      airspaceTransitAllowed: true,
      airspaceStrikeAllowed: true,
      defensivePositioningAllowed: true,
      maritimeTransitAllowed: true,
      maritimeStrikeAllowed: true,
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
      maritimeTransitAllowed: !!direct.maritimeTransitAllowed,
      maritimeStrikeAllowed: !!direct.maritimeStrikeAllowed,
    };
  }
  const fromCoalition = normalizeCountryCode(countryCoalitions?.[from] ?? "");
  const toCoalition = normalizeCountryCode(countryCoalitions?.[to] ?? "");
  if (fromCoalition && toCoalition && fromCoalition !== toCoalition) {
    return {
      shareIntel: false,
      airspaceTransitAllowed: true,
      airspaceStrikeAllowed: true,
      defensivePositioningAllowed: false,
      maritimeTransitAllowed: true,
      maritimeStrikeAllowed: true,
    };
  }
  return {
    shareIntel: false,
    airspaceTransitAllowed: false,
    airspaceStrikeAllowed: false,
    defensivePositioningAllowed: false,
    maritimeTransitAllowed: false,
    maritimeStrikeAllowed: false,
  };
}

export function buildCountryCoalitionMap<T extends { teamId?: string; coalitionId?: string }>(
  units: Iterable<T>,
): CountryCoalitionMap {
  const result: CountryCoalitionMap = {};
  for (const unit of units) {
    const country = normalizeCountryCode(unit.teamId ?? "");
    const coalition = normalizeCountryCode(unit.coalitionId ?? "");
    if (!country || !coalition || result[country]) {
      continue;
    }
    result[country] = coalition;
  }
  return result;
}

export function collectRelationshipCountries<T extends { teamId?: string }>(
  units: Iterable<T>,
  relationships: CountryRelationshipLike[],
): string[] {
  const result = new Set<string>();
  for (const unit of units) {
    const code = normalizeCountryCode(unit.teamId ?? "");
    if (code) {
      result.add(code);
    }
  }
  for (const relationship of relationships) {
    const from = normalizeCountryCode(relationship.fromCountry);
    const to = normalizeCountryCode(relationship.toCountry);
    if (from) {
      result.add(from);
    }
    if (to) {
      result.add(to);
    }
  }
  return Array.from(result).sort();
}
