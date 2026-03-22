import { useEffect, useMemo, useState } from "react";
import { ListUnitDefinitions } from "../../../wailsjs/go/main/App";
import { useSimStore } from "../../store/simStore";
import { selectedPlayerTeam } from "../../utils/playerTeam";
import { inferUnitTeamCode, type DefinitionTeamMeta } from "../../utils/unitTeams";

export default function ViewSwitcher() {
  const activeView = useSimStore((s) => s.activeView);
  const setActiveView = useSimStore((s) => s.setActiveView);
  const humanControlledTeam = useSimStore((s) => s.humanControlledTeam);
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

  const options = useMemo(() => {
    const playerTeam = selectedPlayerTeam(humanControlledTeam);
    if (playerTeam) {
      return ["debug", playerTeam];
    }
    return ["debug", ...teams];
  }, [humanControlledTeam, teams]);

  useEffect(() => {
    const playerTeam = selectedPlayerTeam(humanControlledTeam);
    if (playerTeam && activeView !== "debug" && activeView !== playerTeam) {
      setActiveView(playerTeam);
    }
  }, [activeView, humanControlledTeam, setActiveView]);

  return (
    <label className="top-bar-select top-bar-select-debug">
      <span>VIEW</span>
      <select
        value={activeView}
        onChange={(e) => setActiveView(e.target.value)}
        title="Switch map visibility mode"
      >
        {options.map((option) => (
          <option key={option} value={option}>
            {option === "debug" ? "DEBUG" : option}
          </option>
        ))}
      </select>
    </label>
  );
}
