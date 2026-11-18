package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStopReasonHotfixCoversUsageAndOpsTables(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "sqlArchiving", "175_add_stop_reason_columns.sql"))
	require.NoError(t, err)

	sql := strings.ToLower(string(content))
	require.Contains(t, sql, "alter table usage_logs")
	require.Contains(t, sql, "alter table ops_error_logs")
	require.Equal(t, 2, strings.Count(sql, "column stop_reason varchar(64) null"))
	require.Contains(t, sql, "show columns from usage_logs")
	require.Contains(t, sql, "show columns from ops_error_logs")
}
