//go:build unit

package growth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeJSONMap(t *testing.T) {
	require.Empty(t, normalizeJSONMap(nil))

	value := map[string]any{
		"enabled": true,
		"limit":   5,
		"remark":  "json",
	}
	require.Equal(t, value, normalizeJSONMap(value))
}
