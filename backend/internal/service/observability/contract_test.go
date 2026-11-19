package observability

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContractGolden(t *testing.T) {
	payload, err := json.MarshalIndent(Contract(), "", "  ")
	require.NoError(t, err)
	payload = append(payload, '\n')

	golden, err := os.ReadFile(filepath.Join("testdata", "observability_contract.golden.json"))
	require.NoError(t, err)
	require.Equal(t, string(golden), string(payload))
}

func TestValidateWindowBucket(t *testing.T) {
	for _, test := range []struct {
		window Window
		bucket Bucket
		valid  bool
	}{
		{Window1h, Bucket5m, true},
		{Window6h, Bucket15m, true},
		{Window24h, Bucket1h, true},
		{Window7d, Bucket6h, true},
		{Window1h, Bucket1h, false},
		{Window7d, Bucket5m, false},
		{Window("30d"), Bucket1h, false},
	} {
		err := ValidateWindowBucket(test.window, test.bucket)
		require.Equal(t, test.valid, err == nil, "%s/%s", test.window, test.bucket)
	}
}

func TestParseFilterDeduplicatesAndCaps(t *testing.T) {
	filter, err := ParseFilter(url.Values{
		"user_id":     {"1", "1", "2"},
		"group_id":    {"8"},
		"route_alias": {" coding-fast ", "coding-fast", ""},
		"account_id":  {"31"},
	})
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2}, filter.UserIDs)
	require.Equal(t, []string{"coding-fast"}, filter.RouteAliases)

	values := make([]string, MaxFilterValues+1)
	for i := range values {
		values[i] = strconv.Itoa(i + 1)
	}
	_, err = ParseFilter(url.Values{"user_id": values})
	require.Error(t, err)
}

func TestValidatePageAndRatio(t *testing.T) {
	require.NoError(t, ValidatePage(1, MaxPageSize))
	require.Error(t, ValidatePage(0, DefaultPageSize))
	require.Error(t, ValidatePage(1, MaxPageSize+1))
	require.NoError(t, ValidateRatio(0.0082, "failure_rate"))
	require.NoError(t, ValidateRatio(0.82, "failure_rate"))
	require.Error(t, ValidateRatio(82, "failure_rate"))
}

func TestResolveRangePresetsAndCustomBounds(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2026, 11, 12, 10, 30, 0, 0, loc)

	today, err := ResolveRange(WindowToday, "", "", "", now, loc)
	require.NoError(t, err)
	require.Equal(t, WindowToday, today.Window)
	require.Equal(t, Bucket15m, today.Bucket)
	require.Equal(t, time.Date(2026, 11, 12, 0, 0, 0, 0, loc), today.Start.In(loc))
	require.Equal(t, now.UTC(), today.End)

	yesterday, err := ResolveRange(WindowYesterday, "", "", "", now, loc)
	require.NoError(t, err)
	require.Equal(t, Bucket1h, yesterday.Bucket)
	require.Equal(t, time.Date(2026, 11, 11, 0, 0, 0, 0, loc), yesterday.Start.In(loc))
	require.Equal(t, time.Date(2026, 11, 12, 0, 0, 0, 0, loc), yesterday.End.In(loc))

	custom, err := ResolveRange(
		WindowCustom,
		Bucket15m,
		"2026-11-12T09:00:00+08:00",
		"2026-11-12T10:00:00+08:00",
		now,
		loc,
	)
	require.NoError(t, err)
	require.Equal(t, Bucket15m, custom.Bucket)
	require.Equal(t, time.Hour, custom.End.Sub(custom.Start))
}

func TestResolveRangeRejectsInvalidBounds(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, 11, 12, 18, 0, 0, 0, loc)

	_, err := ResolveRange(WindowCustom, "", "2026-11-12T12:00:00+08:00", "2026-11-12T12:00:00+08:00", now, loc)
	require.Error(t, err)
	_, err = ResolveRange(WindowCustom, "", "2026-11-12T12:00:00+08:00", "2026-11-12T19:00:00+08:00", now, loc)
	require.Error(t, err)
	_, err = ResolveRange(WindowCustom, Bucket5m, "2026-11-01T00:00:00+08:00", "2026-11-10T00:00:00+08:00", now, loc)
	require.Error(t, err)
	_, err = ResolveRange(WindowCustom, Bucket5m, "2026-11-10T00:00:00+08:00", "2026-11-12T00:00:00+08:00", now, loc)
	require.Error(t, err)
}
