import { useEffect, useMemo, useState } from "react";
import { ListUnitDefinitions } from "../../../wailsjs/go/main/App";
import { useSimStore } from "../../store/simStore";
import { inferUnitTeamCode, type DefinitionTeamMeta } from "../../utils/unitTeams";

export default function ViewSwitcher() {
  const activeView = useSimStore((s) => s.activeView);
  const setActiveView = useSimStore((s) => s.setActiveView);
  const units = useSimStore((s) => s.units);
  const [definitionTeams, setDefinitionTeams] = useState<Map<string, DefinitionTeamMeta>>(new Map());

  useEffect(() => {
    let cancelled = false;
    ListUnitDefinitions()
      .then((rows) => {
        if (cancelled) return;
        const next = new Map<string, DefinitionTeamMeta>();
        rows.forEach((row) => {
          const id = String(row.id ?? "").trim();
          if (!id) return;
          const employedBy = Array.isArray(row.employed_by) ? row.employed_by.map((v) => String(v).trim().toUpperCase()).filter(Boolean) : [];
          const teamCode = employedBy[0] || String(row.nation_of_origin ?? "").trim().toUpperCase();
          next.set(id, { teamCode });
        });
        setDefinitionTeams(next);
      })
      .catch(console.error);
    return () => {
      cancelled = true;
    };
  }, []);

  const teams = useMemo(() => {
    const codes = new Set<string>();
    units.forEach((unit) => {
      const code = (unit.teamId?.trim().toUpperCase())
        || inferUnitTeamCode(unit.id, definitionTeams.get(unit.definitionId)?.teamCode ?? "");
      if (/^[A-Z]{3}$/.test(code)) {
        codes.add(code);
      }
    });
    return Array.from(codes).sort();
  }, [definitionTeams, units]);

  return (
    <div className="view-switcher">
      <button
        className={`view-btn ${activeView === "debug" ? "view-btn-debug-active" : ""}`}
        onClick={() => setActiveView("debug")}
        title="Show all units (game master view)"
      >
        DEBUG
      </button>
      {teams.map((team) => (
        <button
          key={team}
          className={`view-btn ${activeView === team ? "view-btn-team-active" : ""}`}
          onClick={() => setActiveView(team)}
          title={`${team} national view`}
        >
          {team}
        </button>
      ))}
    </div>
  );
}
