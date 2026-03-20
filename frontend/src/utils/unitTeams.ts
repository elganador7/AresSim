export interface DefinitionTeamMeta {
  teamCode: string;
}

function normalizeTeamCode(value: string | undefined | null): string {
  return String(value ?? "").trim().toUpperCase();
}

function teamFromUnitID(unitID: string): string {
  const prefix = unitID.split(/[-_:]/)[0] ?? "";
  const normalized = normalizeTeamCode(prefix);
  return /^[A-Z]{3}$/.test(normalized) ? normalized : "";
}

export function inferUnitTeamCode(
  unitID: string,
  fallbackCode: string,
  meta?: DefinitionTeamMeta,
): string {
  const explicitCode = normalizeTeamCode(fallbackCode);
  if (/^[A-Z]{3}$/.test(explicitCode)) {
    return explicitCode;
  }

  const idCode = teamFromUnitID(unitID);
  if (idCode) {
    return idCode;
  }

  const metaCode = normalizeTeamCode(meta?.teamCode);
  if (/^[A-Z]{3}$/.test(metaCode)) {
    return metaCode;
  }

  return explicitCode || "UNK";
}
