package server

import (
	"os"
	"strings"
	"testing"
)

func TestLegacyTokenStatisticsRoutesRemoved(t *testing.T) {
	data, err := os.ReadFile("routes/admin.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, legacy := range []string{`"/token-usage"`, `"/model-token-quotas"`, `"default-model-token-quotas"`} {
		if strings.Contains(source, legacy) {
			t.Fatalf("legacy route still registered: %s", legacy)
		}
	}
	if !strings.Contains(source, `admin.Group("/token-statistics")`) {
		t.Fatal("new configurable token statistics routes are missing")
	}
}
