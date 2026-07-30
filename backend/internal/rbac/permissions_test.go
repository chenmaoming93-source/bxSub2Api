package rbac

import "testing"

func TestPermissionCatalogIsUniqueAndValid(t *testing.T) {
	seen := make(map[string]struct{}, len(Catalog()))
	for _, item := range Catalog() {
		if item.Code == "" || item.Name == "" || item.Module == "" || item.Description == "" || item.Risk == "" {
			t.Fatalf("incomplete permission definition: %#v", item)
		}
		if _, ok := seen[item.Code]; ok {
			t.Fatalf("duplicate permission code: %s", item.Code)
		}
		seen[item.Code] = struct{}{}
	}
	if _, ok := seen[PermissionAll]; !ok {
		t.Fatal("wildcard permission is missing")
	}
	if !PermissionExists(PermissionUsersRead) {
		t.Fatal("known permission was not found")
	}
	if PermissionExists("unknown.permission") {
		t.Fatal("unknown permission was reported as present")
	}
}

func TestApprovedRouteBaselineCloses(t *testing.T) {
	if ApprovedRouteBaseline.Controlled+ApprovedRouteBaseline.Excluded != ApprovedRouteBaseline.Total {
		t.Fatalf("route baseline does not close: %#v", ApprovedRouteBaseline)
	}
	excluded := 0
	for _, item := range ApprovedExclusions {
		if item.Category == "" || item.Authentication == "" || item.Reason == "" {
			t.Fatalf("incomplete exclusion: %#v", item)
		}
		excluded += item.Count
	}
	if excluded != ApprovedRouteBaseline.Excluded {
		t.Fatalf("excluded route count = %d, want %d", excluded, ApprovedRouteBaseline.Excluded)
	}
}
