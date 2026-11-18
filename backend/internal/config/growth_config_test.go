//go:build unit

package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestGrowthCryptoWindowDuration(t *testing.T) {
	require.Equal(t, 5*time.Minute, (GrowthCryptoConfig{TimeWindow: "5m"}).WindowDuration())
	require.Equal(t, 5*time.Minute, (GrowthCryptoConfig{}).WindowDuration())
	require.Equal(t, 5*time.Minute, (GrowthCryptoConfig{TimeWindow: "invalid"}).WindowDuration())
}

func TestGrowthCryptoDefaultRegistered(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setDefaults()
	require.Equal(t, "5m", viper.GetString("growth.crypto.time_window"))
}

func TestGrowthCryptoValidation(t *testing.T) {
	require.NoError(t, ValidateGrowthConfig(GrowthConfig{Crypto: GrowthCryptoConfig{TimeWindow: "5m"}}))
	require.Error(t, ValidateGrowthConfig(GrowthConfig{Crypto: GrowthCryptoConfig{TimeWindow: "not-a-duration"}}))
}
