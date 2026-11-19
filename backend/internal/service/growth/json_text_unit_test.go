//go:build unit

package growth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONTextRoundTrip(t *testing.T) {
	require.Equal(t, "{}", encodeJSONText(nil))
	require.Empty(t, decodeJSONText(""))

	value := map[string]any{
		"enabled": true,
		"limit":   float64(5),
		"remark":  "goldenDB",
	}
	require.Equal(t, value, decodeJSONText(encodeJSONText(value)))
	require.Empty(t, decodeJSONText("not-json"))
}
