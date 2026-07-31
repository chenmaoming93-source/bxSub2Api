package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

type fixedDynamicQuotaReader struct{ value int64 }

func (r fixedDynamicQuotaReader) Read(context.Context, string, string) (int64, error) {
	return r.value, nil
}

type gatewayProjectionSource struct{}

func (gatewayProjectionSource) ActiveProjections() []tokenstat.ProjectionDefinition {
	return []tokenstat.ProjectionDefinition{{
		ID: 1, Name: "full",
		DimensionCodes: []tokenstat.DimensionCode{
			tokenstat.DimensionUserID, tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID,
			tokenstat.DimensionRouteAlias, tokenstat.DimensionAccountID, tokenstat.DimensionUpstreamModel,
		},
		MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens},
	}}
}

type userProjectionSource struct{}

func (userProjectionSource) ActiveProjections() []tokenstat.ProjectionDefinition {
	return []tokenstat.ProjectionDefinition{{
		ID: 2, Name: "user",
		DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionUserID},
		MetricCodes:    []tokenstat.MetricCode{tokenstat.MetricTotalTokens},
	}}
}

type gatewayAccountingWriter struct {
	mu         sync.Mutex
	operations []tokenstat.AccountingOperation
}

func TestSubmitDynamicTokenUsageUserProjectionDoesNotRequireRoutingDimensions(t *testing.T) {
	writer := &gatewayAccountingWriter{}
	pipeline, err := tokenstat.NewAsyncPipeline(4, 1, time.Second, 0, "Asia/Shanghai", userProjectionSource{}, writer)
	require.NoError(t, err)
	tokenstat.SetDefaultPipeline(pipeline)
	submitDynamicTokenUsage(&UsageLog{
		UserID: 42, Model: "requested-model", InputTokens: 3, OutputTokens: 4,
		RequestType: RequestTypeSync, CreatedAt: time.Now(),
	})
	pipeline.Close()
	tokenstat.SetDefaultPipeline(nil)
	require.Len(t, writer.operations, 3)
	require.Equal(t, int64(42), writer.operations[0].DimensionValues["user_id"])
	require.Equal(t, int64(7), writer.operations[0].Delta)
}

func (w *gatewayAccountingWriter) Add(_ context.Context, operations []tokenstat.AccountingOperation) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.operations = append(w.operations, operations...)
	return nil
}

func TestSubmitDynamicTokenUsageRepresentativeProtocols(t *testing.T) {
	for _, requestType := range []RequestType{RequestTypeSync, RequestTypeStream, RequestTypeWSV2} {
		t.Run(requestType.String(), func(t *testing.T) {
			writer := &gatewayAccountingWriter{}
			pipeline, err := tokenstat.NewAsyncPipeline(4, 1, time.Second, 0, "Asia/Shanghai", gatewayProjectionSource{}, writer)
			require.NoError(t, err)
			tokenstat.SetDefaultPipeline(pipeline)
			groupID := int64(3)
			model := "upstream-model"
			submitDynamicTokenUsage(&UsageLog{
				UserID: 1, APIKeyID: 2, GroupID: &groupID, RouteAlias: "route",
				AccountID: 4, UpstreamModel: &model, InputTokens: 5, OutputTokens: 7,
				RequestType: requestType, CreatedAt: time.Now(),
			})
			pipeline.Close()
			tokenstat.SetDefaultPipeline(nil)
			require.Len(t, writer.operations, 3)
			require.Equal(t, int64(12), writer.operations[0].Delta)
			require.Len(t, writer.operations[0].DimensionValues, 6)
		})
	}
}

func TestDynamicRouteQuotaReceivesAllRegisteredDimensions(t *testing.T) {
	groupID := int64(3)
	account := &Account{ID: 4}
	values := map[tokenstat.DimensionCode]tokenstat.DimensionValue{
		tokenstat.DimensionUserID:        tokenstat.Int64Value(1),
		tokenstat.DimensionAPIKeyID:      tokenstat.Int64Value(2),
		tokenstat.DimensionGroupID:       tokenstat.Int64Value(groupID),
		tokenstat.DimensionRouteAlias:    tokenstat.StringValue("route"),
		tokenstat.DimensionAccountID:     tokenstat.Int64Value(account.ID),
		tokenstat.DimensionUpstreamModel: tokenstat.StringValue("upstream-model"),
	}
	checker := tokenstat.NewQuotaChecker(fixedDynamicQuotaReader{value: 100}, 16)
	checker.ReplaceRules([]tokenstat.QuotaRule{{
		ID: 1, ProjectionID: 1,
		DimensionCodes: []tokenstat.DimensionCode{
			tokenstat.DimensionUserID, tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID,
			tokenstat.DimensionRouteAlias, tokenstat.DimensionAccountID, tokenstat.DimensionUpstreamModel,
		},
		DimensionValues: values, MetricCode: tokenstat.MetricTotalTokens,
		PeriodType: tokenstat.PeriodDay, LimitValue: 100, Mode: tokenstat.QuotaModeEnforce,
	}})
	tokenstat.SetDefaultQuotaChecker(checker)
	t.Cleanup(func() { tokenstat.SetDefaultQuotaChecker(nil) })

	err := CheckDynamicTokenRouteCandidate(context.Background(), &groupID,
		DynamicTokenRequestIdentity{UserID: 1, APIKeyID: 2}, "route", account, "upstream-model")
	require.ErrorIs(t, err, ErrDynamicTokenQuotaExceeded)
}
