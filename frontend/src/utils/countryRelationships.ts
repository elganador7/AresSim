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

export type CountryCoalitionMap = Record<string, string>;

export function normalizeCountryCode(code: string): string {
  return code.trim().toUpperCase();
}

export function isDisplayCountryCode(code: string): boolean {
  const normalized = normalizeCountryCode(code);
  if (!normalized) {
    return false;
  }
  if (normalized === "BLUE" || normalized === "RED" || normalized === "NEUTRAL" || normalized === "DEBUG" || normalized === "NON_ALIGNED") {
    return false;
  }
  return !normalized.startsWith("COALITION_");
}

export function buildCountryCoalitionMap<T extends { teamId?: string; coalitionId?: string }>(
  units: Iterable<T>,
): CountryCoalitionMap {
  const result: CountryCoalitionMap = {};
  for (const unit of units) {
    const country = normalizeCountryCode(unit.teamId ?? "");
    const coalition = normalizeCountryCode(unit.coalitionId ?? "");
    if (!isDisplayCountryCode(country) || !coalition || result[country]) {
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
    if (isDisplayCountryCode(code)) {
      result.add(code);
    }
  }
  for (const relationship of relationships) {
    const from = normalizeCountryCode(relationship.fromCountry);
    const to = normalizeCountryCode(relationship.toCountry);
    if (isDisplayCountryCode(from)) {
      result.add(from);
    }
    if (isDisplayCountryCode(to)) {
      result.add(to);
    }
  }
  return Array.from(result).sort();
}
