package repository

// sanitizeRecord removes nil values recursively so SurrealDB option<T> fields
// are omitted instead of being written as NULL.
func sanitizeRecord(record map[string]any) map[string]any {
	if record == nil {
		return nil
	}
	sanitized := make(map[string]any, len(record))
	for key, value := range record {
		if cleaned, ok := sanitizeValue(value); ok {
			sanitized[key] = cleaned
		}
	}
	return sanitized
}

func sanitizeValue(value any) (any, bool) {
	if value == nil {
		return nil, false
	}
	switch typed := value.(type) {
	case map[string]any:
		cleaned := sanitizeRecord(typed)
		if len(cleaned) == 0 {
			return cleaned, true
		}
		return cleaned, true
	case []map[string]any:
		cleaned := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if sanitized := sanitizeRecord(item); sanitized != nil {
				cleaned = append(cleaned, sanitized)
			}
		}
		return cleaned, true
	case []any:
		cleaned := make([]any, 0, len(typed))
		for _, item := range typed {
			if sanitized, ok := sanitizeValue(item); ok {
				cleaned = append(cleaned, sanitized)
			}
		}
		return cleaned, true
	default:
		return value, true
	}
}
