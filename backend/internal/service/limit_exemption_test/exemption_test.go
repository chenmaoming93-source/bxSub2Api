//go:build unit

package limit_exemption_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/stretchr/testify/require"
)

type rpmCounterStub struct {
	groupCount int
	userCount  int
	groupCalls int
	userCalls  int
}

func (s *rpmCounterStub) IncrementUserGroupRPM(context.Context, int64, int64) (int, error) {
	s.groupCalls++
	return s.groupCount, nil
}

func (s *rpmCounterStub) IncrementUserRPM(context.Context, int64) (int, error) {
	s.userCalls++
	return s.userCount, nil
}

func (s *rpmCounterStub) GetUserGroupRPM(context.Context, int64, int64) (int, error) {
	return 0, nil
}

func (s *rpmCounterStub) GetUserRPM(context.Context, int64) (int, error) {
	return 0, nil
}

type quotaRepositoryStub struct {
	record *service.UserPlatformQuotaRecord
	calls  int
}

func (s *quotaRepositoryStub) GetByUserPlatform(context.Context, int64, string) (*service.UserPlatformQuotaRecord, error) {
	s.calls++
	return s.record, nil
}

func (s *quotaRepositoryStub) BulkInsertInitial(context.Context, []service.UserPlatformQuotaRecord) error {
	return nil
}

func (s *quotaRepositoryStub) IncrementUsageWithReset(context.Context, int64, string, float64, time.Time) error {
	return nil
}

func (s *quotaRepositoryStub) ListByUser(context.Context, int64) ([]service.UserPlatformQuotaRecord, error) {
	return nil, nil
}

func (s *quotaRepositoryStub) UpsertForUser(context.Context, int64, []service.UserPlatformQuotaRecord) error {
	return nil
}

func (s *quotaRepositoryStub) ResetExpiredWindow(context.Context, int64, string, string, time.Time) error {
	return nil
}

func (s *quotaRepositoryStub) BatchSnapshotUsage(context.Context, []service.UserPlatformQuotaSnapshot, time.Time) error {
	return nil
}

func newBillingService(rpm service.UserRPMCache, quotas service.UserPlatformQuotaRepository) *service.BillingCacheService {
	svc := service.NewBillingCacheService(nil, nil, nil, nil, rpm, nil, &config.Config{
		RunMode: config.RunModeStandard,
	}, quotas)
	return svc
}

func TestSuperAdminSkipsUserRPMButKeepsGroupRPM(t *testing.T) {
	t.Run("system admin", func(t *testing.T) {
		rpm := &rpmCounterStub{groupCount: 2, userCount: 2}
		svc := newBillingService(rpm, nil)
		defer svc.Stop()
		user := &service.User{ID: 1, Email: "admin@example.com", Role: service.RoleAdmin, RPMLimit: 1}
		err := svc.CheckBillingEligibility(context.Background(), user, nil, &service.Group{ID: 9, RPMLimit: 1}, nil, "")
		require.ErrorIs(t, err, service.ErrGroupRPMExceeded)
		require.Equal(t, 1, rpm.groupCalls)
		require.Zero(t, rpm.userCalls)
	})

	t.Run("wildcard admin without email", func(t *testing.T) {
		rpm := &rpmCounterStub{groupCount: 1, userCount: 2}
		svc := newBillingService(rpm, nil)
		defer svc.Stop()
		user := &service.User{ID: 2, Role: service.RoleUser, SuperAdmin: true, RPMLimit: 1}
		err := svc.CheckBillingEligibility(context.Background(), user, nil, nil, nil, "")
		require.NoError(t, err)
		require.Zero(t, rpm.userCalls)
	})

	t.Run("normal user remains limited", func(t *testing.T) {
		rpm := &rpmCounterStub{userCount: 2}
		svc := newBillingService(rpm, nil)
		defer svc.Stop()
		user := &service.User{ID: 3, Role: service.RoleUser, RPMLimit: 1}
		err := svc.CheckBillingEligibility(context.Background(), user, nil, nil, nil, "")
		require.ErrorIs(t, err, service.ErrUserRPMExceeded)
		require.Equal(t, 1, rpm.userCalls)
	})
}

func TestSuperAdminSkipsUserPlatformQuota(t *testing.T) {
	limit, usage := 1.0, 1.0
	now := time.Now()
	repo := &quotaRepositoryStub{record: &service.UserPlatformQuotaRecord{
		UserID: 1, Platform: "anthropic",
		DailyLimitUSD: &limit, DailyUsageUSD: usage, DailyWindowStart: &now,
		WeeklyWindowStart: &now, MonthlyWindowStart: &now,
	}}
	svc := newBillingService(nil, repo)
	defer svc.Stop()

	require.NoError(t, svc.CheckUserPlatformQuotaEligibilityForUser(context.Background(), &service.User{
		ID: 1, Role: service.RoleAdmin,
	}, "anthropic"))
	require.Zero(t, repo.calls)

	err := svc.CheckUserPlatformQuotaEligibilityForUser(context.Background(), &service.User{
		ID: 1, Role: service.RoleUser,
	}, "anthropic")
	require.ErrorIs(t, err, service.ErrUserPlatformDailyQuotaExhausted)
	require.Equal(t, 1, repo.calls)
}

type alwaysExceededQuotaReader struct{}

func (alwaysExceededQuotaReader) Read(context.Context, string, string) (int64, error) {
	return 1000, nil
}

func TestSuperAdminSkipsUserTokenRulesButKeepsGroupRules(t *testing.T) {
	checker := tokenstat.NewQuotaChecker(alwaysExceededQuotaReader{}, 16)
	tokenstat.SetDefaultQuotaChecker(checker)
	t.Cleanup(func() { tokenstat.SetDefaultQuotaChecker(nil) })
	at := time.Now()
	groupID := int64(9)
	userRule := tokenstat.QuotaRule{
		ID: 1, ProjectionID: 1,
		DimensionCodes:  []tokenstat.DimensionCode{tokenstat.DimensionUserID},
		DimensionValues: map[tokenstat.DimensionCode]tokenstat.DimensionValue{tokenstat.DimensionUserID: tokenstat.Int64Value(7)},
		MetricCode:      tokenstat.MetricTotalTokens, PeriodType: tokenstat.PeriodDay,
		LimitValue: 1, Mode: tokenstat.QuotaModeEnforce, EffectiveFrom: &at,
	}
	groupRule := tokenstat.QuotaRule{
		ID: 2, ProjectionID: 2,
		DimensionCodes:  []tokenstat.DimensionCode{tokenstat.DimensionGroupID},
		DimensionValues: map[tokenstat.DimensionCode]tokenstat.DimensionValue{tokenstat.DimensionGroupID: tokenstat.Int64Value(groupID)},
		MetricCode:      tokenstat.MetricTotalTokens, PeriodType: tokenstat.PeriodDay,
		LimitValue: 1, Mode: tokenstat.QuotaModeEnforce, EffectiveFrom: &at,
	}
	identity := service.DynamicTokenRequestIdentity{UserID: 7, SuperAdmin: true}

	checker.ReplaceRules([]tokenstat.QuotaRule{userRule})
	require.NoError(t, service.CheckDynamicTokenRouteCandidate(
		context.Background(), nil, identity, "", nil, "",
	))

	withoutExemption := identity
	withoutExemption.SuperAdmin = false
	err := service.CheckDynamicTokenRouteCandidate(context.Background(), nil, withoutExemption, "", nil, "")
	require.True(t, errors.Is(err, service.ErrDynamicTokenQuotaExceeded))

	checker.ReplaceRules([]tokenstat.QuotaRule{groupRule})
	err = service.CheckDynamicTokenRouteCandidate(context.Background(), &groupID, identity, "", nil, "")
	require.True(t, errors.Is(err, service.ErrDynamicTokenQuotaExceeded))
}
