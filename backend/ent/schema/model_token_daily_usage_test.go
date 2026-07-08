package schema

import (
	"testing"

	"entgo.io/ent/schema/field"
)

func TestModelTokenDailyUsageSchemaFieldsAndUniqueKey(t *testing.T) {
	fields := ModelTokenDailyUsage{}.Fields()
	byName := make(map[string]*field.Descriptor, len(fields))
	for _, item := range fields {
		descriptor := item.Descriptor()
		byName[descriptor.Name] = descriptor
	}
	for _, name := range []string{"model", "usage_date", "used_tokens"} {
		if byName[name] == nil {
			t.Errorf("schema missing field %q", name)
		}
	}
	if byName["daily_limit_tokens"] != nil {
		t.Error("usage schema must not contain daily_limit_tokens; limits belong to ModelTokenDailyLimitConfig")
	}

	used := byName["used_tokens"]
	if used == nil {
		t.Fatal("used_tokens field is required")
	}
	if used.Default == nil {
		t.Fatal("used_tokens has no default")
	}

	limitFields := ModelTokenDailyLimitConfig{}.Fields()
	var limit *field.Descriptor
	for _, item := range limitFields {
		if descriptor := item.Descriptor(); descriptor.Name == "daily_limit_tokens" {
			limit = descriptor
			break
		}
	}
	if limit == nil {
		t.Fatal("ModelTokenDailyLimitConfig missing daily_limit_tokens")
	}
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
