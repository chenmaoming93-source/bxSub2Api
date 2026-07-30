package rbac

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestCompatibilitySeedMatchesPermissionCatalog(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "sqlArchiving", "163_seed_rbac_compatibility.sql"))
	if err != nil {
		t.Fatalf("read compatibility seed: %v", err)
	}
	permissionBlockPattern := regexp.MustCompile(`(?s)INSERT INTO rbac_permissions.+?\nON DUPLICATE KEY UPDATE`)
	permissionBlock := permissionBlockPattern.Find(data)
	if len(permissionBlock) == 0 {
		t.Fatal("compatibility seed has no permission insert block")
	}
	codePattern := regexp.MustCompile(`(?m)^\s*\('([^']+)',\s*'[^']*',\s*'[^']+',`)
	matches := codePattern.FindAllStringSubmatch(string(permissionBlock), -1)
	seedCodes := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		seedCodes[match[1]] = struct{}{}
	}
	for _, permission := range Catalog() {
		if _, ok := seedCodes[permission.Code]; !ok {
			t.Errorf("permission %q is missing from compatibility seed", permission.Code)
		}
	}
	if len(seedCodes) != len(Catalog()) {
		t.Errorf("seed has %d permission codes, catalog has %d", len(seedCodes), len(Catalog()))
	}
}
