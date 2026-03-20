const TEAM_PALETTE = [
  "#60a5fa",
  "#f87171",
  "#34d399",
  "#fbbf24",
  "#a78bfa",
  "#fb7185",
  "#22d3ee",
  "#f97316",
  "#4ade80",
  "#c084fc",
];

function normalizeTeam(value: string | undefined | null): string {
  return String(value ?? "").trim().toUpperCase();
}

function hashTeamCode(teamCode: string): number {
  let hash = 0;
  for (let i = 0; i < teamCode.length; i += 1) {
    hash = ((hash << 5) - hash) + teamCode.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash);
}

export function teamColorHex(teamCode: string | undefined | null): string {
  const normalized = normalizeTeam(teamCode);
  if (!normalized) {
    return "#9ca3af";
  }
  return TEAM_PALETTE[hashTeamCode(normalized) % TEAM_PALETTE.length];
}
