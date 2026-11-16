package observability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRequiredAlertEvaluationsDebouncesSingleWindow(t *testing.T) {
	require.Equal(t, 3, RequiredAlertEvaluations(0, time.Minute, 3))
	require.Equal(t, 3, RequiredAlertEvaluations(1, time.Minute, 3))
	require.Equal(t, 5, RequiredAlertEvaluations(5, time.Minute, 3))
	require.Equal(t, 1, RequiredAlertEvaluations(0, time.Minute, 1))
}
