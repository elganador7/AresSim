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
  unitSide: string,
  meta?: DefinitionTeamMeta,
): string {
  const sideCode = normalizeTeamCode(unitSide);
  if (/^[A-Z]{3}$/.test(sideCode)) {
    return sideCode;
  }

  const idCode = teamFromUnitID(unitID);
  if (idCode) {
    return idCode;
  }

  const metaCode = normalizeTeamCode(meta?.teamCode);
  if (/^[A-Z]{3}$/.test(metaCode)) {
    return metaCode;
  }

  return sideCode || "UNK";
}
