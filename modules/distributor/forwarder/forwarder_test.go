package forwarder

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

type mockCountingForwarder struct {
	next                Forwarder
	forwardBatchesCount int
}

func (m *mockCountingForwarder) ForwardBatches(ctx context.Context, trace tempopb.Trace) error {
	m.forwardBatchesCount++
	return m.next.ForwardBatches(ctx, trace)
}

func (m *mockCountingForwarder) Shutdown(ctx context.Context) error {
	return m.next.Shutdown(ctx)
}

func TestList_ForwardBatches_ReturnsNoErrorAndCallsForwardBatchesOnAllUnderlyingWorkingForwarders(t *testing.T) {
	// Given
	forwarder1 := &mockCountingForwarder{next: &mockWorkingForwarder{}, forwardBatchesCount: 0}
	forwarder2 := &mockCountingForwarder{next: &mockWorkingForwarder{}, forwardBatchesCount: 0}
	forwarder3 := &mockCountingForwarder{next: &mockWorkingForwarder{}, forwardBatchesCount: 0}
	list := List([]Forwarder{forwarder1, forwarder2, forwarder3})

	// When
	err := list.ForwardBatches(context.Background(), tempopb.Trace{})

	// Then
	require.NoError(t, err)
	require.Equal(t, 1, forwarder1.forwardBatchesCount)
	require.Equal(t, 1, forwarder2.forwardBatchesCount)
	require.Equal(t, 1, forwarder3.forwardBatchesCount)
}

func TestList_ForwardBatches_ReturnsErrorAndCallsForwardBatchesOnAllUnderlyingForwardersWithSingleFailingForwarder(t *testing.T) {
	// Given
	forwarder1 := &mockCountingForwarder{next: &mockWorkingForwarder{}, forwardBatchesCount: 0}
	forwarder2 := &mockCountingForwarder{next: &mockWorkingForwarder{}, forwardBatchesCount: 0}
	forwarder3 := &mockCountingForwarder{next: &mockFailingForwarder{forwardBatchesErr: errors.New("forward batches error")}, forwardBatchesCount: 0}
	list := List([]Forwarder{forwarder1, forwarder2, forwarder3})

	// When
	err := list.ForwardBatches(context.Background(), tempopb.Trace{})

	// Then
	require.Error(t, err)
	require.Equal(t, 1, forwarder1.forwardBatchesCount)
	require.Equal(t, 1, forwarder2.forwardBatchesCount)
	require.Equal(t, 1, forwarder3.forwardBatchesCount)
}

func TestList_ForwardBatches_ReturnsErrorAndCallsForwardBatchesOnAllUnderlyingForwardersWithAllFailingForwarder(t *testing.T) {
	// Given
	forwarder1 := &mockCountingForwarder{next: &mockFailingForwarder{forwardBatchesErr: errors.New("1 forward batches error")}, forwardBatchesCount: 0}
	forwarder2 := &mockCountingForwarder{next: &mockFailingForwarder{forwardBatchesErr: errors.New("2 forward batches error")}, forwardBatchesCount: 0}
	forwarder3 := &mockCountingForwarder{next: &mockFailingForwarder{forwardBatchesErr: errors.New("3 forward batches error")}, forwardBatchesCount: 0}
	list := List([]Forwarder{forwarder1, forwarder2, forwarder3})

	// When
	err := list.ForwardBatches(context.Background(), tempopb.Trace{})

	// Then
	require.Error(t, err)
	require.ErrorContains(t, err, "1")
	require.ErrorContains(t, err, "2")
	require.ErrorContains(t, err, "3")
	require.Equal(t, 1, forwarder1.forwardBatchesCount)
	require.Equal(t, 1, forwarder2.forwardBatchesCount)
	require.Equal(t, 1, forwarder3.forwardBatchesCount)
}
