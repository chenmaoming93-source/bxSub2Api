//go:build unit

package growth

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestBusinessCycleKey(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2026, 11, 17, 16, 30, 0, 0, time.UTC)

	daily, err := BusinessCycleKey(now, location, "DAILY")
	require.NoError(t, err)
	require.Equal(t, "2026-11-18", daily)

	weekly, err := BusinessCycleKey(now, location, "WEEKLY")
	require.NoError(t, err)
	require.Equal(t, "2026-W47", weekly)

	once, err := BusinessCycleKey(now, location, "ONCE")
	require.NoError(t, err)
	require.Equal(t, "once", once)
}

func TestLedgerSignedSum(t *testing.T) {
	entries := []*ent.GrowthAccountLedger{
		{AccountType: "WALLET", Direction: "IN", Points: 30},
		{AccountType: "WALLET", Direction: "OUT", Points: 20},
		{AccountType: "REDEEMED", Direction: "IN", Points: 20},
	}
	require.Equal(t, int64(10), LedgerSignedSum(entries, "WALLET"))
	require.Equal(t, int64(20), LedgerSignedSum(entries, "REDEEMED"))
}
