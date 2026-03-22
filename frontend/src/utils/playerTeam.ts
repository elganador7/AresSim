export function normalizeTeamCode(team: string | undefined | null): string {
  return String(team ?? "").trim().toUpperCase();
}

export function selectedPlayerTeam(humanControlledTeam: string | undefined | null): string {
  return normalizeTeamCode(humanControlledTeam);
}
