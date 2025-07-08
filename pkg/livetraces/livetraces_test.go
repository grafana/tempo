package livetraces

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestLiveTracesSizesAndLen(t *testing.T) {
	lt := New[*v1.ResourceSpans](func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) }, time.Millisecond, time.Second)

	expectedSz := uint64(0)
	expectedLen := uint64(0)

	for i := 0; i < 100; i++ {
		id := test.ValidTraceID(nil)
		tr := test.MakeTrace(rand.IntN(5)+1, id)

		nowTime := time.Now()

		// add some traces and confirm size/len
		expectedLen++
		for _, rs := range tr.ResourceSpans {
			expectedSz += uint64(rs.Size())
			lt.Push(id, rs, 0)
		}

		require.Equal(t, expectedSz, lt.Size())
		require.Equal(t, expectedLen, lt.Len())

		// cut some traces and confirm size/len
		cutTraces := lt.CutIdle(nowTime, false)
		for _, tr := range cutTraces {
			for _, rs := range tr.Batches {
				expectedSz -= uint64(rs.Size())
			}
			expectedLen--
		}

		require.Equal(t, expectedSz, lt.Size())
		require.Equal(t, expectedLen, lt.Len())
	}
}

func TestCutIdleDueToIdleTime(t *testing.T) {
	lt := New(func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) }, time.Second, time.Hour)

	id := test.ValidTraceID(nil)
	tr := test.MakeTrace(1, id)

	rootTime := time.Unix(0, 0)

	lt.PushWithTimestampAndLimits(rootTime, id, tr.ResourceSpans[0], 0, 0)

	// cut at 500 ms, should cut nothing
	cutTraces := lt.CutIdle(rootTime.Add(500*time.Millisecond), false)
	require.Equal(t, 0, len(cutTraces))

	// push at 1 second
	lt.PushWithTimestampAndLimits(rootTime.Add(1000*time.Millisecond), id, tr.ResourceSpans[0], 0, 0)

	// cut at 1.5 seconds, should cut nothing
	cutTraces = lt.CutIdle(rootTime.Add(1500*time.Millisecond), false)
	require.Equal(t, 0, len(cutTraces))

	// cut at 2.5 seconds, should cut the trace b/c it's been idle for 1.5 seconds
	cutTraces = lt.CutIdle(rootTime.Add(2500*time.Millisecond), false)
	require.Equal(t, 1, len(cutTraces))
	require.Equal(t, id, cutTraces[0].ID)

	require.Equal(t, 0, len(lt.Traces))
}

func TestCutIdleDueToLiveTime(t *testing.T) {
	lt := New(func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) }, time.Hour, time.Second)

	id := test.ValidTraceID(nil)
	tr := test.MakeTrace(1, id)

	rootTime := time.Unix(0, 0)

	lt.PushWithTimestampAndLimits(rootTime, id, tr.ResourceSpans[0], 0, 0)

	// cut at 500 ms, should cut nothing
	cutTraces := lt.CutIdle(rootTime.Add(500*time.Millisecond), false)
	require.Equal(t, 0, len(cutTraces))

	// push at 1 second
	lt.PushWithTimestampAndLimits(rootTime.Add(1000*time.Millisecond), id, tr.ResourceSpans[0], 0, 0)

	// cut at 1.5 seconds, should cut the trace b/c it's been live for 1.5 seconds!
	cutTraces = lt.CutIdle(rootTime.Add(1500*time.Millisecond), false)
	require.Equal(t, 1, len(cutTraces))
	require.Equal(t, id, cutTraces[0].ID)

	// cut at 2.5 seconds, should cut nothing
	cutTraces = lt.CutIdle(rootTime.Add(2500*time.Millisecond), false)
	require.Equal(t, 0, len(cutTraces))

	require.Equal(t, 0, len(lt.Traces))
}

func BenchmarkLiveTracesWrite(b *testing.B) {
	lt := New[*v1.ResourceSpans](func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) }, 0, 0)

	var traces []*tempopb.Trace
	for i := 0; i < 100_000; i++ {
		traces = append(traces, test.MakeTrace(1, nil))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, tr := range traces {
			lt.Push(tr.ResourceSpans[0].ScopeSpans[0].Spans[0].TraceId, tr.ResourceSpans[0], 0)
		}
	}
}

func BenchmarkLiveTracesRead(b *testing.B) {
	lt := New[*v1.ResourceSpans](func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) }, 0, 0)

	for i := 0; i < 100_000; i++ {
		tr := test.MakeTrace(1, nil)
		lt.Push(tr.ResourceSpans[0].ScopeSpans[0].Spans[0].TraceId, tr.ResourceSpans[0], 0)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// This won't anything, instead of will benchmark the map iteration performance.
		lt.CutIdle(time.Now().Add(-time.Hour), false)
	}
}
