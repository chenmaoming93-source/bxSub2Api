package schema

import "testing"

func TestUserModelTokenDailyUsageSchemaKeysAndUserEdge(t *testing.T) {
	indexes := UserModelTokenDailyUsage{}.Indexes()
	if len(indexes) != 2 {
		t.Fatalf("index count = %d, want 2", len(indexes))
	}
	unique := indexes[0].Descriptor()
	want := []string{"user_id", "model", "usage_date"}
	if !unique.Unique || len(unique.Fields) != len(want) {
		t.Fatalf("unique index = %+v", unique)
	}
	for i := range want {
		if unique.Fields[i] != want[i] {
			t.Fatalf("unique fields = %v, want %v", unique.Fields, want)
		}
	}
	edges := UserModelTokenDailyUsage{}.Edges()
	if len(edges) != 1 || edges[0].Descriptor().Field != "user_id" || !edges[0].Descriptor().Unique || !edges[0].Descriptor().Required {
		t.Fatalf("user edge = %+v", edges[0].Descriptor())
	}
}
