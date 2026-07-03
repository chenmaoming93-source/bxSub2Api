package schema

import "testing"

func TestGroupCandidateTokenDailyUsageSchemaIdentity(t *testing.T) {
	indexes := GroupCandidateTokenDailyUsage{}.Indexes()
	if len(indexes) != 2 {
		t.Fatalf("index count = %d, want 2", len(indexes))
	}
	unique := indexes[0].Descriptor()
	want := []string{"group_id", "route_alias", "upstream_model", "usage_date"}
	if !unique.Unique || len(unique.Fields) != len(want) {
		t.Fatalf("unique index = %+v", unique)
	}
	for i := range want {
		if unique.Fields[i] != want[i] {
			t.Fatalf("unique fields = %v, want %v", unique.Fields, want)
		}
	}
	edges := GroupCandidateTokenDailyUsage{}.Edges()
	if len(edges) != 1 || edges[0].Descriptor().Field != "group_id" || !edges[0].Descriptor().Required {
		t.Fatalf("group edge = %+v", edges[0].Descriptor())
	}
}
