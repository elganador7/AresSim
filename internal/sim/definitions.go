package sim

import "strings"

func normalizeDefinitionID(definitionID string) string {
	definitionID = strings.TrimSpace(definitionID)
	if idx := strings.LastIndex(definitionID, ":"); idx >= 0 {
		return definitionID[idx+1:]
	}
	return definitionID
}

func definitionStatsFor(defs map[string]DefStats, definitionID string) DefStats {
	if def, ok := defs[definitionID]; ok {
		return def
	}
	return defs[normalizeDefinitionID(definitionID)]
}
