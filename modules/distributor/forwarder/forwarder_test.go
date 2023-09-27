package forwarder

import (
	"context"
	"errors"
	"testing"

	dslog "github.com/grafana/dskit/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type mockCountingForwarder struct {
	next               Forwarder
	forwardTracesCount int
}

func (m *mockCountingForwarder) ForwardTraces(ctx context.Context, traces ptrace.Traces) error {
	m.forwardTracesCount++
	return m.next.ForwardTraces(ctx, traces)
}

func (m *mockCountingForwarder) Shutdown(ctx context.Context) error {
	return m.next.Shutdown(ctx)
}

type mockTraceRecordingForwarder struct {
	next   Forwarder
	traces ptrace.Traces
}

func (m *mockTraceRecordingForwarder) ForwardTraces(ctx context.Context, traces ptrace.Traces) error {
	m.traces = traces
	return m.next.ForwardTraces(ctx, traces)
}

func (m *mockTraceRecordingForwarder) Shutdown(ctx context.Context) error {
	return m.next.Shutdown(ctx)
}

type mockWorkingProcessor struct {
	component.Component
	consumer.Traces
}

func (m *mockWorkingProcessor) Shutdown(_ context.Context) error {
	return nil
}

type mockFailingProcessor struct {
	component.Component
	consumer.Traces
	err error
}

func (m *mockFailingProcessor) Shutdown(_ context.Context) error {
	return m.err
}

func TestList_ForwardTraces_ReturnsNoErrorAndCallsForwardTracesOnAllUnderlyingWorkingForwarders(t *testing.T) {
	// Given
	forwarder1 := &mockCountingForwarder{next: &mockWorkingForwarder{}, forwardTracesCount: 0}
	forwarder2 := &mockCountingForwarder{next: &mockWorkingForwarder{}, forwardTracesCount: 0}
	forwarder3 := &mockCountingForwarder{next: &mockWorkingForwarder{}, forwardTracesCount: 0}
	list := List([]Forwarder{forwarder1, forwarder2, forwarder3})

	// When
	err := list.ForwardTraces(context.Background(), ptrace.Traces{})

	// Then
	require.NoError(t, err)
	require.Equal(t, 1, forwarder1.forwardTracesCount)
	require.Equal(t, 1, forwarder2.forwardTracesCount)
	require.Equal(t, 1, forwarder3.forwardTracesCount)
}

func TestList_ForwardTraces_ReturnsErrorAndCallsForwardTracesOnAllUnderlyingForwardersWithSingleFailingForwarder(t *testing.T) {
	// Given
	forwarder1 := &mockCountingForwarder{next: &mockWorkingForwarder{}, forwardTracesCount: 0}
	forwarder2 := &mockCountingForwarder{next: &mockWorkingForwarder{}, forwardTracesCount: 0}
	forwarder3 := &mockCountingForwarder{next: &mockFailingForwarder{forwardTracesErr: errors.New("forward batches error")}, forwardTracesCount: 0}
	list := List([]Forwarder{forwarder1, forwarder2, forwarder3})

	// When
	err := list.ForwardTraces(context.Background(), ptrace.Traces{})

	// Then
	require.Error(t, err)
	require.Equal(t, 1, forwarder1.forwardTracesCount)
	require.Equal(t, 1, forwarder2.forwardTracesCount)
	require.Equal(t, 1, forwarder3.forwardTracesCount)
}

func TestList_ForwardTraces_ReturnsErrorAndCallsForwardTracesOnAllUnderlyingForwardersWithAllFailingForwarder(t *testing.T) {
	// Given
	forwarder1 := &mockCountingForwarder{next: &mockFailingForwarder{forwardTracesErr: errors.New("1 forward batches error")}, forwardTracesCount: 0}
	forwarder2 := &mockCountingForwarder{next: &mockFailingForwarder{forwardTracesErr: errors.New("2 forward batches error")}, forwardTracesCount: 0}
	forwarder3 := &mockCountingForwarder{next: &mockFailingForwarder{forwardTracesErr: errors.New("3 forward batches error")}, forwardTracesCount: 0}
	list := List([]Forwarder{forwarder1, forwarder2, forwarder3})

	// When
	err := list.ForwardTraces(context.Background(), ptrace.Traces{})

	// Then
	require.Error(t, err)
	require.ErrorContains(t, err, "1")
	require.ErrorContains(t, err, "2")
	require.ErrorContains(t, err, "3")
	require.Equal(t, 1, forwarder1.forwardTracesCount)
	require.Equal(t, 1, forwarder2.forwardTracesCount)
	require.Equal(t, 1, forwarder3.forwardTracesCount)
}

func TestList_ForwardTraces_DoesNotPanicWhenNil(t *testing.T) {
	// Given
	list := List(nil)

	// When
	panicFunc := func() {
		err := list.ForwardTraces(context.Background(), ptrace.Traces{})
		require.NoError(t, err)
	}

	// Then
	require.NotPanics(t, panicFunc)
}

func TestFilterForwarder_ForwardTraces_ReturnsNoErrorAndCallsForwardTracesOnUnderlyingForwarderWithNoFilters(t *testing.T) {
	// Given
	traces := ptrace.NewTraces()
	rss := traces.ResourceSpans().AppendEmpty()
	ss := rss.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetTraceID(pcommon.TraceID{1, 2})

	f := &mockCountingForwarder{next: &mockWorkingForwarder{}, forwardTracesCount: 0}
	cfg := FilterConfig{}
	ff, err := NewFilterForwarder(cfg, f, dslog.Level{})
	require.NoError(t, err)

	// When
	err = ff.ForwardTraces(context.Background(), traces)

	// Then
	require.NoError(t, err)
	require.Equal(t, 1, f.forwardTracesCount)
}

func TestFilterForwarder_ForwardTraces_ReturnsNoErrorAndCallsForwardTracesOnUnderlyingForwarderWithFilters(t *testing.T) {
	// Given
	traces := ptrace.NewTraces()
	rss := traces.ResourceSpans().AppendEmpty()
	ss := rss.ScopeSpans().AppendEmpty()
	ss.Spans().AppendEmpty().SetName("to-filter")
	ss.Spans().AppendEmpty().SetName("to-keep")

	rf := &mockTraceRecordingForwarder{next: &mockWorkingForwarder{}}
	f := &mockCountingForwarder{next: rf, forwardTracesCount: 0}
	cfg := FilterConfig{
		Traces: TraceFiltersConfig{
			SpanConditions:      []string{`name == "to-filter"`},
			SpanEventConditions: nil,
		},
	}
	ff, err := NewFilterForwarder(cfg, f, dslog.Level{})
	require.NoError(t, err)

	// When
	err = ff.ForwardTraces(context.Background(), traces)

	// Then
	require.NoError(t, err)
	require.Equal(t, 1, f.forwardTracesCount)
	require.Equal(t, 1, rf.traces.ResourceSpans().Len())
	require.Equal(t, 1, rf.traces.ResourceSpans().At(0).ScopeSpans().Len())
	require.Equal(t, 1, rf.traces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().Len())
	require.Equal(t, "to-keep", rf.traces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())
}

func TestFilterForwarder_ForwardTraces_ReturnsNoErrorAndDoesNotCallsForwardTracesOnUnderlyingForwarderWithAllSpansFiltered(t *testing.T) {
	// Given
	traces := ptrace.NewTraces()
	rss := traces.ResourceSpans().AppendEmpty()
	ss := rss.ScopeSpans().AppendEmpty()
	ss.Spans().AppendEmpty().SetName("to-filter-1")
	ss.Spans().AppendEmpty().SetName("to-filter-2")

	rf := &mockTraceRecordingForwarder{next: &mockWorkingForwarder{}}
	f := &mockCountingForwarder{next: rf, forwardTracesCount: 0}
	cfg := FilterConfig{
		Traces: TraceFiltersConfig{
			SpanConditions:      []string{`name == "to-filter-1" or name == "to-filter-2"`},
			SpanEventConditions: nil,
		},
	}
	ff, err := NewFilterForwarder(cfg, f, dslog.Level{})
	require.NoError(t, err)

	// When
	err = ff.ForwardTraces(context.Background(), traces)

	// Then
	require.NoError(t, err)
	require.Zero(t, f.forwardTracesCount)
	require.Equal(t, ptrace.Traces{}, rf.traces)
}

func TestFilterForwarder_ForwardTraces_ReturnsErrorAndDoesNotCallForwardTracesOnUnderlyingForwarderWithFatalError(t *testing.T) {
	// Given
	fatalErr := errors.New("fatal error")
	f := &mockCountingForwarder{next: &mockWorkingForwarder{}}
	cfg := FilterConfig{
		Traces: TraceFiltersConfig{
			SpanConditions:      []string{`name == "to-filter-1" or name == "to-filter-2"`},
			SpanEventConditions: nil,
		},
	}
	ff, err := NewFilterForwarder(cfg, f, dslog.Level{})
	require.NoError(t, err)

	ff.fatalError = fatalErr

	// When
	err = ff.ForwardTraces(context.Background(), ptrace.NewTraces())

	// Then
	require.ErrorIs(t, err, fatalErr)
	require.Zero(t, f.forwardTracesCount)
}

func TestFilterForwarder_ForwardTraces_ReturnsErrorWithFailingUnderlyingForwarder(t *testing.T) {
	// Given
	traces := ptrace.NewTraces()
	rss := traces.ResourceSpans().AppendEmpty()
	ss := rss.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetTraceID(pcommon.TraceID{1, 2})

	forwardTracesErr := errors.New("forward traces error")
	f := &mockCountingForwarder{next: &mockFailingForwarder{forwardTracesErr: forwardTracesErr}}
	cfg := FilterConfig{
		Traces: TraceFiltersConfig{
			SpanConditions:      []string{`name == "to-filter-1" or name == "to-filter-2"`},
			SpanEventConditions: nil,
		},
	}
	ff, err := NewFilterForwarder(cfg, f, dslog.Level{})
	require.NoError(t, err)

	// When
	err = ff.ForwardTraces(context.Background(), traces)

	// Then
	require.ErrorIs(t, err, forwardTracesErr)
	require.Equal(t, 1, f.forwardTracesCount)
}

func TestFilterForwarder_Shutdown_ReturnsNoErrorWithWorkingProcessorAndForwarder(t *testing.T) {
	// Given
	ff := &FilterForwarder{
		filterProcessor: &mockWorkingProcessor{},
		next:            &mockWorkingForwarder{},
	}

	// When
	err := ff.Shutdown(context.Background())

	// Then
	require.NoError(t, err)
}

func TestFilterForwarder_Shutdown_ReturnsErrorWithFailingProcessor(t *testing.T) {
	// Given
	shutdownErr := errors.New("shutdown error")
	ff := &FilterForwarder{
		filterProcessor: &mockFailingProcessor{err: shutdownErr},
		next:            &mockWorkingForwarder{},
	}

	// When
	err := ff.Shutdown(context.Background())

	// Then
	require.ErrorIs(t, err, shutdownErr)
}

func TestFilterForwarder_Shutdown_ReturnsErrorWithFailingForwarder(t *testing.T) {
	// Given
	shutdownErr := errors.New("shutdown error")
	ff := &FilterForwarder{
		filterProcessor: &mockWorkingProcessor{},
		next:            &mockFailingForwarder{shutdownErr: shutdownErr},
	}

	// When
	err := ff.Shutdown(context.Background())

	// Then
	require.ErrorIs(t, err, shutdownErr)
}

func TestFilterForwarder_Shutdown_ReturnsErrorWithFailingProcessorAndForwarder(t *testing.T) {
	// Given
	processorShutdownErr := errors.New("processor shutdown error")
	forwarderShutdownErr := errors.New("forwarder shutdown error")
	ff := &FilterForwarder{
		filterProcessor: &mockFailingProcessor{err: processorShutdownErr},
		next:            &mockFailingForwarder{shutdownErr: forwarderShutdownErr},
	}

	// When
	err := ff.Shutdown(context.Background())

	// Then
	require.ErrorIs(t, err, processorShutdownErr)
	require.ErrorIs(t, err, forwarderShutdownErr)
}
