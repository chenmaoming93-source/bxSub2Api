package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIKeyPlatformGroupIndexArchive(t *testing.T) {
	path := filepath.Join("..", "sqlArchiving", "161_api_key_platform_group_lookup_index.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read index archive: %v", err)
	}
	sql := strings.ToLower(string(content))
	if !strings.Contains(sql, "create index if not exists") {
		t.Fatal("index archive must create a non-unique index with repeat-execution protection")
	}
	if strings.Contains(sql, "create unique index") {
		t.Fatal("platform group lookup index must not be unique")
	}
	if !strings.Contains(sql, "(user_id, platform, group_id, deleted_at)") {
		t.Fatal("index archive has an unexpected field order")
	}
}

func TestAPIKeySchemaDeclaresPlatformGroupIndex(t *testing.T) {
	path := filepath.Join("..", "ent", "schema", "api_key.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read api key schema: %v", err)
	}
	compact := strings.Join(strings.Fields(string(content)), " ")
	if !strings.Contains(compact, `index.Fields("user_id", "platform", "group_id", "deleted_at")`) {
		t.Fatal("api key schema is missing the four-field platform group index")
	}
}
