package tokenstat

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	require.Len(t, Dimensions(), 6)
	require.Len(t, Metrics(), 1)
	metric, ok := Metric(MetricTotalTokens)
	require.True(t, ok)
	require.True(t, metric.AllowQuota)
}

func TestDimensionSignatureIgnoresSelectionOrder(t *testing.T) {
	left, err := DimensionSignature([]DimensionCode{DimensionUpstreamModel, DimensionUserID, DimensionGroupID})
	require.NoError(t, err)
	right, err := DimensionSignature([]DimensionCode{DimensionGroupID, DimensionUpstreamModel, DimensionUserID})
	require.NoError(t, err)
	require.Equal(t, "user_id,group_id,upstream_model", left)
	require.Equal(t, left, right)
}

func TestDimensionIdentityStableVector(t *testing.T) {
	values := map[DimensionCode]DimensionValue{
		DimensionUserID:        Int64Value(42),
		DimensionRouteAlias:    StringValue("chat"),
		DimensionUpstreamModel: StringValue("gpt-5"),
	}
	identity, err := BuildDimensionIdentity(
		[]DimensionCode{DimensionUpstreamModel, DimensionUserID, DimensionRouteAlias},
		values,
	)
	require.NoError(t, err)
	require.Equal(t, "d314bba8cf9bfa00e5df72e84dd83802", identity.HashHex())
	second, err := BuildDimensionIdentity(
		[]DimensionCode{DimensionRouteAlias, DimensionUpstreamModel, DimensionUserID},
		values,
	)
	require.NoError(t, err)
	require.Equal(t, identity.Canonical, second.Canonical)
}

func TestNaturalPeriodsBoundaries(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	cases := []struct {
		name       string
		at         time.Time
		dayStart   string
		weekStart  string
		monthStart string
		monthEnd   string
	}{
		{"monday", time.Date(2026, 7, 27, 15, 0, 0, 0, location), "2026-07-27", "2026-07-27", "2026-07-01", "2026-08-01"},
		{"month end", time.Date(2026, 7, 31, 23, 59, 0, 0, location), "2026-07-31", "2026-07-27", "2026-07-01", "2026-08-01"},
		{"year end", time.Date(2026, 12, 31, 23, 59, 0, 0, location), "2026-12-31", "2026-12-28", "2026-12-01", "2027-01-01"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			periods := NaturalPeriods(tc.at, location)
			require.Equal(t, tc.dayStart, periods[0].Start.Format(time.DateOnly))
			require.Equal(t, tc.weekStart, periods[1].Start.Format(time.DateOnly))
			require.Equal(t, tc.monthStart, periods[2].Start.Format(time.DateOnly))
			require.Equal(t, tc.monthEnd, periods[2].End.Format(time.DateOnly))
		})
	}
}

func TestRegistryRejectsInvalidDefinitionsAndValues(t *testing.T) {
	_, err := DimensionSignature([]DimensionCode{DimensionUserID, DimensionUserID})
	require.ErrorContains(t, err, "duplicate dimension")
	err = ValidateProjection(ProjectionDefinition{
		Name:           "invalid",
		DimensionCodes: []DimensionCode{DimensionUserID},
		MetricCodes:    []MetricCode{"unknown"},
	})
	require.ErrorContains(t, err, "unknown metric")
	_, err = BuildDimensionIdentity(
		[]DimensionCode{DimensionUserID},
		map[DimensionCode]DimensionValue{DimensionUserID: StringValue("42")},
	)
	require.ErrorContains(t, err, "expects int64")
}
