package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStreamUpstreamContextControl_PreservesValuesAndBridgesParentCancel(t *testing.T) {
	const key = "request-value"
	parent, parentCancel := context.WithCancel(context.WithValue(context.Background(), key, "kept"))
	control := newStreamUpstreamContextControl(parent, true)
	t.Cleanup(control.Release)

	require.Equal(t, "kept", control.ctx.Value(key))
	require.NoError(t, control.ctx.Err())

	parentCancel()
	select {
	case <-control.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("downstream context was not canceled after parent cancellation")
	}
	require.ErrorIs(t, control.ctx.Err(), context.Canceled)
}

func TestStreamUpstreamContextControl_ExplicitCancelDoesNotCancelParent(t *testing.T) {
	parent := context.Background()
	control := newStreamUpstreamContextControl(parent, true)
	t.Cleanup(control.Release)

	control.Cancel()
	select {
	case <-control.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("downstream context was not canceled explicitly")
	}

	require.NoError(t, parent.Err())
}

func TestStreamUpstreamContextControl_ReleaseStopsBridgeAndCancelsChild(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	control := newStreamUpstreamContextControl(parent, true)

	control.Release()
	select {
	case <-control.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("released downstream context was not canceled")
	}
	require.ErrorIs(t, control.ctx.Err(), context.Canceled)

	// Release must be idempotent and must stop the parent bridge without
	// changing the already-independent parent context.
	control.Release()
	parentCancel()
	require.ErrorIs(t, control.ctx.Err(), context.Canceled)
}

func TestStreamUpstreamContextControl_NonStreamUsesOriginalContext(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	control := newStreamUpstreamContextControl(parent, false)
	require.Same(t, parent, control.ctx)

	control.Cancel()
	require.NoError(t, control.ctx.Err())
	cancel()
	require.ErrorIs(t, control.ctx.Err(), context.Canceled)
}
