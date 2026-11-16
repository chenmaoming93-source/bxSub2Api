package repository

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsErrorInsertPersistsRouteAlias(t *testing.T) {
	input := &service.OpsInsertErrorLogInput{
		ErrorPhase: "upstream",
		ErrorType:  "upstream_error",
		RouteAlias: "coding-fast",
	}
	args := opsInsertErrorLogArgs(input)

	require.Contains(t, insertOpsErrorLogSQL, "route_alias")
	require.Equal(t, sql.NullString{String: "coding-fast", Valid: true}, args[15])
	require.Equal(t, len(args), strings.Count(insertOpsErrorLogSQL, "?"))
}
