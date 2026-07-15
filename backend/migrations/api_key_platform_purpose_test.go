package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIKeyPlatformPurposeMigration(t *testing.T) {
	path := filepath.Join("..", "sqlArchiving", "160_api_key_platform_purpose.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := strings.ToLower(string(content))
	checks := []string{
		"add column if not exists platform varchar(50) null",
		"add column if not exists purpose varchar(20) not null default 'user_created'",
	}
	for _, want := range checks {
		if !strings.Contains(sql, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
