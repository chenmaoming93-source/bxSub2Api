package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestMarkClientDisconnectedErrorLogSetsStableClassification(t *testing.T) {
	status := 502
	message := "provider detail must not leak"
	entry := &OpsInsertErrorLogInput{
		RequestType:          func() *int16 { v := int16(2); return &v }(),
		Stream:               true,
		ErrorMessage:         message,
		ErrorBody:            "sensitive body",
		UpstreamStatusCode:   &status,
		UpstreamErrorMessage: &message,
		IsCountTokens:        true,
	}

	MarkClientDisconnectedErrorLog(entry)

	require.Equal(t, "network", entry.ErrorPhase)
	require.Equal(t, OpsErrorTypeClientDisconnected, entry.ErrorType)
	require.Equal(t, "client", entry.ErrorOwner)
	require.Equal(t, "client_request", entry.ErrorSource)
	require.Equal(t, 499, entry.StatusCode)
	require.Equal(t, "P3", entry.Severity)
	require.False(t, entry.IsBusinessLimited)
	require.False(t, entry.IsCountTokens)
	require.Equal(t, message, entry.ErrorMessage)
	require.Empty(t, entry.ErrorBody)
	require.Nil(t, entry.UpstreamStatusCode)
	require.Nil(t, entry.UpstreamErrorMessage)
	require.NotNil(t, entry.RequestType)
	require.Equal(t, int16(2), *entry.RequestType)
}

func TestMarkClientDisconnectedErrorLogProvidesStableMessage(t *testing.T) {
	entry := &OpsInsertErrorLogInput{}
	MarkClientDisconnectedErrorLog(entry)
	require.Equal(t, "Upstream client disconnected before response completed", entry.ErrorMessage)
}

func TestRecordClientDisconnectedErrorUsesIndependentContextAndRespectsMonitoringGate(t *testing.T) {
	var gotCtx context.Context
	var gotEntry *OpsInsertErrorLogInput
	repo := &opsRepoMock{
		InsertErrorLogFn: func(ctx context.Context, entry *OpsInsertErrorLogInput) (int64, error) {
			gotCtx = ctx
			gotEntry = entry
			return 1, nil
		},
	}
	svc := &OpsService{opsRepo: repo}
	entry := &OpsInsertErrorLogInput{}
	MarkClientDisconnectedErrorLog(entry)

	requestCtx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, svc.RecordError(context.Background(), entry))
	require.NotNil(t, gotCtx)
	require.NoError(t, gotCtx.Err())
	require.Same(t, entry, gotEntry)

	disabled := &OpsService{opsRepo: repo, cfg: &config.Config{Ops: config.OpsConfig{Enabled: false}}}
	gotEntry = nil
	require.NoError(t, disabled.RecordError(requestCtx, entry))
	require.Nil(t, gotEntry)
}
