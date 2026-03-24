export interface AllegianceLike {
  teamId?: string;
  coalitionId?: string;
  operatorTeamId?: string;
}

function normalize(value: string | undefined | null): string {
  return String(value ?? "").trim().toUpperCase();
}

export function sameTeam(a: AllegianceLike, b: AllegianceLike): boolean {
  const aTeam = normalize(a.operatorTeamId || a.teamId);
  const bTeam = normalize(b.operatorTeamId || b.teamId);
  return aTeam !== "" && aTeam === bTeam;
}

export function sameCoalition(a: AllegianceLike, b: AllegianceLike): boolean {
  const aCoalition = normalize(a.coalitionId);
  const bCoalition = normalize(b.coalitionId);
  return aCoalition !== "" && aCoalition === bCoalition;
}

export function areFriendly(a: AllegianceLike, b: AllegianceLike): boolean {
  return sameTeam(a, b) || sameCoalition(a, b);
}

export function areHostile(a: AllegianceLike, b: AllegianceLike): boolean {
  const aTeam = normalize(a.operatorTeamId || a.teamId);
  const bTeam = normalize(b.operatorTeamId || b.teamId);
  if (aTeam === "" || bTeam === "" || aTeam === bTeam) {
    return false;
  }
  const aCoalition = normalize(a.coalitionId);
  const bCoalition = normalize(b.coalitionId);
  if (aCoalition === "" || bCoalition === "") {
    return true;
  }
  return aCoalition !== bCoalition;
}
