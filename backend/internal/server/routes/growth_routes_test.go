package routes

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrowthUFaceRouteContract(t *testing.T) {
	source, err := os.ReadFile("growth.go")
	require.NoError(t, err)
	text := string(source)
	require.Equal(t, 11, strings.Count(text, "growth.POST("))
	for _, route := range []string{
		`"/me"`, `"/rank"`, `"/tasks"`, `"/tasks/:task_id"`, `"/tasks/:task_id/completion-requests"`,
		`"/ledger"`, `"/products"`, `"/products/:product_id"`, `"/exchanges"`,
		`"/exchanges/:order_id"`, `"/checkins"`,
	} {
		require.Contains(t, text, route)
	}
	require.Contains(t, text, "growth.Use(decryptor.Middleware())")
	require.NotContains(t, text, "jwtAuth")
}
