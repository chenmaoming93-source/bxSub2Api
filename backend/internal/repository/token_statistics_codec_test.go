package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTokenStatisticsKey(t *testing.T) {
	usageDate := time.Date(2026, 7, 13, 20, 30, 0, 0, time.UTC)
	cases := []struct {
		statisticsType TokenStatisticsType
		want           string
	}{
		{TokenStatisticsModel, "sub2api:token_stats:model:2026-07-14"},
		{TokenStatisticsUserModel, "sub2api:token_stats:user_model:2026-07-14"},
		{TokenStatisticsGroupCandidate, "sub2api:token_stats:group_candidate:2026-07-14"},
	}
	for _, tc := range cases {
		got, err := TokenStatisticsKey(tc.statisticsType, usageDate)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}

	_, err := TokenStatisticsKey("unknown", usageDate)
	require.ErrorContains(t, err, `statistics type="unknown"`)
}

func TestTokenStatisticsFieldCodecRoundTrip(t *testing.T) {
	model := "vendor|模型/with spaces?and=chars"
	modelField, err := EncodeModelTokenStatisticsField(model)
	require.NoError(t, err)
	require.NotContains(t, modelField, "|")
	decodedModel, err := DecodeModelTokenStatisticsField(modelField)
	require.NoError(t, err)
	require.Equal(t, model, decodedModel)

	userField, err := EncodeUserModelTokenStatisticsField(42, model)
	require.NoError(t, err)
	userID, decodedModel, err := DecodeUserModelTokenStatisticsField(userField)
	require.NoError(t, err)
	require.Equal(t, int64(42), userID)
	require.Equal(t, model, decodedModel)

	groupField, err := EncodeGroupCandidateTokenStatisticsField(7, "alias|特殊", model)
	require.NoError(t, err)
	groupID, alias, upstreamModel, err := DecodeGroupCandidateTokenStatisticsField(groupField)
	require.NoError(t, err)
	require.Equal(t, int64(7), groupID)
	require.Equal(t, "alias|特殊", alias)
	require.Equal(t, model, upstreamModel)
}

func TestTokenStatisticsTTL(t *testing.T) {
	usageDate := time.Date(2026, 7, 13, 23, 59, 0, 0, tokenStatisticsLocation)
	expiresAt, err := TokenStatisticsExpireAt(usageDate, 2)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 7, 16, 0, 0, 0, 0, tokenStatisticsLocation), expiresAt)
	require.Equal(t, 48*time.Hour+time.Minute, expiresAt.Sub(usageDate))
}

func TestTokenStatisticsCodecInvalidInput(t *testing.T) {
	_, err := EncodeModelTokenStatisticsField("")
	require.ErrorContains(t, err, "type=model")

	_, err = EncodeUserModelTokenStatisticsField(0, "model")
	require.ErrorContains(t, err, "user_id=0")

	_, _, _, err = DecodeGroupCandidateTokenStatisticsField("not base64|")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "type=group_candidate") && strings.Contains(err.Error(), "invalid base64"))

	_, err = TokenStatisticsExpireAt(time.Now(), 0)
	require.ErrorContains(t, err, "retention_days=0")
}
