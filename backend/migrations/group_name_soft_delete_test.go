package migrations

import (
	"strings"
	"testing"
)

func TestMigration157DropsBothGroupNameUniqueIndexes(t *testing.T) {
	content, err := FS.ReadFile("157_group_name_soft_delete_unique.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	s := string(content)
	u := strings.ToUpper(s)

	// Must reference both indexes.
	for _, name := range []string{"`name`", "groups_name_unique_active"} {
		if !strings.Contains(s, name) {
			t.Errorf("migration 157 missing index reference %q", name)
		}
	}

	// Exactly two DROP INDEX statements, no CREATE.
	if n := strings.Count(u, "DROP INDEX"); n != 2 {
		t.Errorf("migration 157 must contain exactly two DROP INDEX, got %d", n)
	}
	if strings.Contains(u, "CREATE INDEX") {
		t.Error("migration 157 must not contain CREATE INDEX")
	}
}
