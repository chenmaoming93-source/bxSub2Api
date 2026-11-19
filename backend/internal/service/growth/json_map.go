package growth

func normalizeJSONMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
