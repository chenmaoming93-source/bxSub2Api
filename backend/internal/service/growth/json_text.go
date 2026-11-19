package growth

import "encoding/json"

func encodeJSONText(value any) string {
	if value == nil {
		return "{}"
	}
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) == 0 || string(encoded) == "null" {
		return "{}"
	}
	return string(encoded)
}

func decodeJSONText(value string) map[string]any {
	result := make(map[string]any)
	if value == "" {
		return result
	}
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return make(map[string]any)
	}
	if result == nil {
		return make(map[string]any)
	}
	return result
}
