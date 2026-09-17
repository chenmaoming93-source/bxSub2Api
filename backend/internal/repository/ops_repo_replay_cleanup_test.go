package repository

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestOpsErrorLogInsertDoesNotPersistRequestReplayFields(t *testing.T) {
	disallowedColumns := []string{
		"request_body",
		"request_headers",
		"request_body_truncated",
		"request_body_bytes",
		"retry_count",
		"resolved_retry_id",
		"is_retryable",
	}

	insertSQL := strings.ToLower(insertOpsErrorLogSQL)
	for _, column := range disallowedColumns {
		if strings.Contains(insertSQL, column) {
			t.Fatalf("ops error log insert still references dropped replay column %q", column)
		}
	}

	inputType := reflect.TypeOf(service.OpsInsertErrorLogInput{})
	disallowedFields := []string{
		"RequestBodyJSON",
		"RequestBodyTruncated",
		"RequestBodyBytes",
		"RequestHeadersJSON",
		"RetryCount",
		"ResolvedRetryID",
		"IsRetryable",
	}
	for _, field := range disallowedFields {
		if _, ok := inputType.FieldByName(field); ok {
			t.Fatalf("OpsInsertErrorLogInput still carries replay field %q", field)
		}
	}
}

// Guards against insert-statement drift: a column list that no longer matches the
// placeholder or argument count fails in production with a MySQL syntax error, so
// keep the three counts locked together here.
func TestOpsErrorLogInsertColumnPlaceholderAndArgCountsMatch(t *testing.T) {
	open := strings.Index(insertOpsErrorLogSQL, "(")
	closeIdx := strings.Index(insertOpsErrorLogSQL, ")")
	if open < 0 || closeIdx < open {
		t.Fatalf("unexpected ops error log insert SQL shape")
	}

	columns := 0
	for _, line := range strings.Split(insertOpsErrorLogSQL[open+1:closeIdx], "\n") {
		if strings.TrimSpace(line) != "" {
			columns++
		}
	}
	if columns == 0 {
		t.Fatal("failed to parse ops error log insert column list")
	}

	placeholders := strings.Count(insertOpsErrorLogSQL, "?")
	if placeholders != columns {
		t.Fatalf("ops error log insert has %d placeholders for %d columns", placeholders, columns)
	}

	args := len(opsInsertErrorLogArgs(&service.OpsInsertErrorLogInput{}))
	if args != columns {
		t.Fatalf("ops error log insert passes %d args for %d columns", args, columns)
	}
}
