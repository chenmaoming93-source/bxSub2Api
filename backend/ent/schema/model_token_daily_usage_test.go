package schema

import "testing"

func TestModelTokenDailyUsageSchemaFieldsAndUniqueKey(t *testing.T) {
	fields := ModelTokenDailyUsage{}.Fields()
	byName := make(map[string]bool, len(fields))
	for _, item := range fields {
		byName[item.Descriptor().Name] = true
	}
	for _, name := range []string{"model", "usage_date", "used_tokens", "daily_limit_tokens"} {
		if !byName[name] {
			t.Errorf("schema missing field %q", name)
		}
	}

	used := fields[2].Descriptor()
	if used.Default == nil {
		t.Fatal("used_tokens has no default")
	}
	limit := fields[3].Descriptor()
	if !limit.Optional || !limit.Nillable {
		t.Fatalf("daily_limit_tokens optional=%v nillable=%v", limit.Optional, limit.Nillable)
	}

	indexes := ModelTokenDailyUsage{}.Indexes()
	if len(indexes) != 1 {
		t.Fatalf("index count = %d, want 1", len(indexes))
	}
	descriptor := indexes[0].Descriptor()
	if !descriptor.Unique || len(descriptor.Fields) != 2 || descriptor.Fields[0] != "model" || descriptor.Fields[1] != "usage_date" {
		t.Fatalf("unique index = %+v", descriptor)
	}
}
