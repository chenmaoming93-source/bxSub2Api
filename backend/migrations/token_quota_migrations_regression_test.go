package migrations

import "testing"

func TestTokenQuotaMigrationsDefaultsAndUniqueIdentities(t *testing.T) {
	t.Run("global model", TestModelTokenDailyUsagesMigrationIsIdempotentAndComplete)
	t.Run("user model", TestUserModelTokenDailyUsagesMigrationConstraints)
	t.Run("group candidate", TestGroupCandidateTokenDailyUsagesMigrationIdentity)
}
