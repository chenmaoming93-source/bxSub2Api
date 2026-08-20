package service

import (
	"context"
	"errors"
	"testing"
	"time"

	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/stretchr/testify/require"
)

type externalTokenDimensionLookupStub struct {
	user                      *User
	group                     *Group
	key                       *APIKey
	userErr, groupErr, keyErr error
}

type externalProjectionStub []tokenstat.ProjectionDefinition

func (s externalProjectionStub) ActiveProjections() []tokenstat.ProjectionDefinition {
	return []tokenstat.ProjectionDefinition(s)
}

type externalQuotaStub []tokenstat.QuotaRule

func (s externalQuotaStub) ActiveQuotaRules() []tokenstat.QuotaRule {
	return []tokenstat.QuotaRule(s)
}

type externalCurrentReaderStub struct {
	values []ExternalTokenUsageCurrentValue
	errAt  int
	calls  int
}

func (s *externalCurrentReaderStub) Read(context.Context, tokenstat.Period, int64, [16]byte, tokenstat.MetricCode) (ExternalTokenUsageCurrentValue, error) {
	s.calls++
	if s.errAt == s.calls {
		return ExternalTokenUsageCurrentValue{}, errors.New("redis down")
	}
	return s.values[s.calls-1], nil
}

func TestExternalTokenUsageQueryCurrentPeriodsAndStates(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	reader := &externalCurrentReaderStub{values: []ExternalTokenUsageCurrentValue{{Exists: true, Value: 9}, {Exists: false}, {Exists: true, Value: 0}}}
	svc := NewExternalTokenUsageService(validExternalTokenLookup())
	svc.ConfigureCurrentUsage(reader, externalProjectionStub{{ID: 7, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionRouteAlias, tokenstat.DimensionGroupID, tokenstat.DimensionUserID, tokenstat.DimensionAPIKeyID}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}}}, location)
	svc.SetClockForTest(func() time.Time { return time.Date(2026, 7, 31, 12, 0, 0, 0, location) })
	got, err := svc.QueryCurrentUsage(context.Background(), ExternalTokenUsageInput{Username: "ldap@example.com", GroupName: "public", APIKey: "sk-ldap-key-0123456789", RouteAlias: "gpt-main"})
	require.NoError(t, err)
	require.Equal(t, int64(9), *got.Day.TotalTokens)
	require.False(t, got.Week.DataPresent)
	require.Equal(t, int64(0), *got.Week.TotalTokens)
	require.True(t, got.Month.DataPresent)
	require.Equal(t, time.Date(2026, 7, 27, 0, 0, 0, 0, location), got.Week.PeriodStart)
	require.Equal(t, 3, reader.calls)
}

func TestExternalTokenUsageQueryReturnsSmallestEnforcedLimitPerPeriod(t *testing.T) {
	location := time.UTC
	reader := &externalCurrentReaderStub{values: []ExternalTokenUsageCurrentValue{{Exists: true, Value: 9}, {Exists: true, Value: 18}, {Exists: true, Value: 27}}}
	svc := NewExternalTokenUsageService(validExternalTokenLookup())
	svc.ConfigureCurrentUsage(reader, externalProjectionStub{{ID: 7, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionUserID, tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID, tokenstat.DimensionRouteAlias}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}}}, location)
	svc.ConfigureQuotaRules(externalQuotaStub{
		{ID: 1, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionUserID}, DimensionValues: map[tokenstat.DimensionCode]tokenstat.DimensionValue{
			tokenstat.DimensionUserID: tokenstat.Int64Value(1),
		}, MetricCode: tokenstat.MetricTotalTokens, PeriodType: tokenstat.PeriodDay, LimitValue: 100, Mode: tokenstat.QuotaModeEnforce},
		{ID: 2, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionUserID, tokenstat.DimensionGroupID}, DimensionValues: map[tokenstat.DimensionCode]tokenstat.DimensionValue{
			tokenstat.DimensionUserID: tokenstat.Int64Value(1), tokenstat.DimensionGroupID: tokenstat.Int64Value(2),
		}, MetricCode: tokenstat.MetricTotalTokens, PeriodType: tokenstat.PeriodDay, LimitValue: 50, Mode: tokenstat.QuotaModeEnforce},
		{ID: 3, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionUserID}, DimensionValues: map[tokenstat.DimensionCode]tokenstat.DimensionValue{
			tokenstat.DimensionUserID: tokenstat.Int64Value(1),
		}, MetricCode: tokenstat.MetricTotalTokens, PeriodType: tokenstat.PeriodDay, LimitValue: 10, Mode: tokenstat.QuotaModeObserve},
		{ID: 4, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionUserID}, DimensionValues: map[tokenstat.DimensionCode]tokenstat.DimensionValue{
			tokenstat.DimensionUserID: tokenstat.Int64Value(1),
		}, MetricCode: tokenstat.MetricTotalTokens, PeriodType: tokenstat.PeriodWeek, LimitValue: 200, Mode: tokenstat.QuotaModeEnforce},
	})
	svc.SetClockForTest(func() time.Time { return time.Date(2026, 8, 18, 12, 0, 0, 0, location) })

	got, err := svc.QueryCurrentUsage(context.Background(), ExternalTokenUsageInput{Username: "ldap@example.com", GroupName: "public", APIKey: "sk-ldap-key-0123456789", RouteAlias: "gpt-main"})
	require.NoError(t, err)
	require.NotNil(t, got.Day.EnforcedLimit)
	require.Equal(t, int64(50), *got.Day.EnforcedLimit)
	require.NotNil(t, got.Week.EnforcedLimit)
	require.Equal(t, int64(200), *got.Week.EnforcedLimit)
	require.Nil(t, got.Month.EnforcedLimit)
}

func TestExternalTokenUsageQueryUnconfiguredAndRedisFailure(t *testing.T) {
	location := time.UTC
	svc := NewExternalTokenUsageService(validExternalTokenLookup())
	svc.ConfigureCurrentUsage(&externalCurrentReaderStub{}, externalProjectionStub{}, location)
	got, err := svc.QueryCurrentUsage(context.Background(), ExternalTokenUsageInput{Username: "ldap@example.com", GroupName: "public", APIKey: "sk-ldap-key-0123456789", RouteAlias: "gpt-main"})
	require.NoError(t, err)
	require.False(t, got.Day.DimensionConfigured)
	require.Nil(t, got.Day.TotalTokens)

	reader := &externalCurrentReaderStub{values: make([]ExternalTokenUsageCurrentValue, 3), errAt: 2}
	svc.ConfigureCurrentUsage(reader, externalProjectionStub{{ID: 7, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionUserID, tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID, tokenstat.DimensionRouteAlias}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}}}, location)
	_, err = svc.QueryCurrentUsage(context.Background(), ExternalTokenUsageInput{Username: "ldap@example.com", GroupName: "public", APIKey: "sk-ldap-key-0123456789", RouteAlias: "gpt-main"})
	require.ErrorIs(t, err, ErrTokenUsageUnavailable)
	require.Equal(t, 2, reader.calls)
}

func TestExternalTokenUsageQueryRejectsExtraDimensionProjection(t *testing.T) {
	svc := NewExternalTokenUsageService(validExternalTokenLookup())
	svc.ConfigureCurrentUsage(&externalCurrentReaderStub{}, externalProjectionStub{{ID: 7, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionUserID, tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID, tokenstat.DimensionRouteAlias, tokenstat.DimensionAccountID}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}}}, time.UTC)
	got, err := svc.QueryCurrentUsage(context.Background(), ExternalTokenUsageInput{Username: "ldap@example.com", GroupName: "public", APIKey: "sk-ldap-key-0123456789", RouteAlias: "gpt-main"})
	require.NoError(t, err)
	require.False(t, got.Day.DimensionConfigured)
}

func (s *externalTokenDimensionLookupStub) FindUserByEmail(context.Context, string) (*User, error) {
	return s.user, s.userErr
}
func (s *externalTokenDimensionLookupStub) FindGroupByName(context.Context, string) (*Group, error) {
	return s.group, s.groupErr
}
func (s *externalTokenDimensionLookupStub) FindAPIKeyByKey(context.Context, string) (*APIKey, error) {
	return s.key, s.keyErr
}

func validExternalTokenLookup() *externalTokenDimensionLookupStub {
	groupID := int64(2)
	return &externalTokenDimensionLookupStub{
		user:  &User{ID: 1, Email: "ldap@example.com"},
		group: &Group{ID: groupID, ModelRouting: map[string][]int64{"gpt-main": {3}}},
		key:   &APIKey{ID: 4, UserID: 1, GroupID: &groupID, Name: "ldap-key", Key: "sk-ldap-key-0123456789"},
	}
}

func TestExternalTokenUsageResolveDimensions(t *testing.T) {
	svc := NewExternalTokenUsageService(validExternalTokenLookup())
	got, err := svc.ResolveDimensions(context.Background(), ExternalTokenUsageInput{Username: " ldap@example.com ", GroupName: " public ", APIKey: " sk-ldap-key-0123456789 ", RouteAlias: " gpt-main "})
	require.NoError(t, err)
	require.Equal(t, ExternalTokenUsageDimensions{UserID: 1, GroupID: 2, APIKeyID: 4, RouteAlias: "gpt-main"}, got)
}

func TestExternalTokenUsageResolveDimensionsNotFoundOrder(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*externalTokenDimensionLookupStub)
		want   error
	}{
		{"user", func(s *externalTokenDimensionLookupStub) { s.userErr = ErrUserNotFound; s.groupErr = ErrGroupNotFound }, ErrUserNotFound},
		{"group", func(s *externalTokenDimensionLookupStub) { s.groupErr = ErrGroupNotFound; s.keyErr = ErrAPIKeyNotFound }, ErrGroupNotFound},
		{"api key", func(s *externalTokenDimensionLookupStub) { s.keyErr = ErrAPIKeyNotFound }, ErrAPIKeyNotFound},
		{"api key belongs to another user", func(s *externalTokenDimensionLookupStub) { s.key.UserID = 99 }, ErrAPIKeyMismatch},
		{"api key belongs to another group", func(s *externalTokenDimensionLookupStub) { other := int64(99); s.key.GroupID = &other }, ErrAPIKeyMismatch},
		{"api key has no group", func(s *externalTokenDimensionLookupStub) { s.key.GroupID = nil }, ErrAPIKeyMismatch},
		{"route alias", func(s *externalTokenDimensionLookupStub) { s.group.ModelRouting = map[string][]int64{"other": {3}} }, ErrRouteAliasNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := validExternalTokenLookup()
			tt.mutate(lookup)
			_, err := NewExternalTokenUsageService(lookup).ResolveDimensions(context.Background(), ExternalTokenUsageInput{Username: "ldap@example.com", GroupName: "public", APIKey: "sk-ldap-key-0123456789", RouteAlias: "gpt-main"})
			require.True(t, errors.Is(err, tt.want), "error=%v", err)
		})
	}
}
