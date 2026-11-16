package observability

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHistogramQuantilesWithinInterpolationError(t *testing.T) {
	hist := Histogram{
		SampleCount: 100,
		MinMS:       80,
		MaxMS:       6000,
		Buckets:     [8]int64{10, 20, 20, 20, 10, 10, 9, 1},
	}
	got := hist.Quantiles(0.5, 0.95)

	require.InDelta(t, 500, got[0.5], 125)
	require.InDelta(t, 10000, got[0.95], 2500)
}

func TestHistogramQuantilesEmpty(t *testing.T) {
	require.Empty(t, (Histogram{}).Quantiles(0.5, 0.95))
}
