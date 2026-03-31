package sim

import enginev1 "github.com/aressim/internal/gen/engine/v1"

func resolveTrackGroupIDs(units []*enginev1.Unit) map[string]string {
	unitByID := make(map[string]*enginev1.Unit, len(units))
	for _, u := range units {
		if unitCanOperate(u) {
			unitByID[u.Id] = u
		}
	}

	resolved := make(map[string]string, len(unitByID))
	for _, u := range units {
		if !unitCanOperate(u) {
			continue
		}
		root := resolveTrackRoot(u, unitByID, resolved, map[string]bool{})
		resolved[u.Id] = unitTeamID(u) + "|" + root
	}
	return resolved
}

func resolveTrackRoot(unit *enginev1.Unit, unitByID map[string]*enginev1.Unit, resolved map[string]string, visiting map[string]bool) string {
	if groupID := resolved[unit.Id]; groupID != "" {
		if idx := len(unitTeamID(unit)) + 1; len(groupID) > idx {
			return groupID[idx:]
		}
		return unit.Id
	}
	if visiting[unit.Id] {
		return unit.Id
	}
	visiting[unit.Id] = true

	parentID := unit.GetParentUnitId()
	if parentID == "" {
		return unit.Id
	}
	parent, ok := unitByID[parentID]
	if !ok || unitTeamID(parent) != unitTeamID(unit) {
		return parentID
	}
	return resolveTrackRoot(parent, unitByID, resolved, visiting)
}
