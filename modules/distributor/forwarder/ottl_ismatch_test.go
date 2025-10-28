package forwarder

import (
	"context"
	"testing"

	dslog "github.com/grafana/dskit/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestFilterForwarder_IsMatch(t *testing.T) {
	// This test is mostly to ensure that the IsMatch is loaded correctly, there are other functions that could be affected but we want to load
	// the standard ones. OTTL made a change with allowing custom functions which changed how standard functions are loaded.
	traces := ptrace.NewTraces()
	rss := traces.ResourceSpans().AppendEmpty()
	ss := rss.ScopeSpans().AppendEmpty()

	span1 := ss.Spans().AppendEmpty()
	span1.SetName("keep-me")
	span1.Attributes().PutStr("service", "my-service")

	span2 := ss.Spans().AppendEmpty()
	span2.SetName("filter-me")
	span2.Attributes().PutStr("service", "other-service")

	rf := &mockTraceRecordingForwarder{next: &mockWorkingForwarder{}}
	f := &mockCountingForwarder{next: rf, forwardTracesCount: 0}
	cfg := FilterConfig{
		Traces: TraceFiltersConfig{
			SpanConditions: []string{`IsMatch(attributes["service"], "other-.*")`},
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
	require.Equal(t, "keep-me", rf.traces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())
}
