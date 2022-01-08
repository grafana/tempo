package trace

import (
	"math/rand"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortTrace(t *testing.T) {
	tests := []struct {
		input    *tempopb.Trace
		expected *tempopb.Trace
	}{
		{
			input:    &tempopb.Trace{},
			expected: &tempopb.Trace{},
		},

		{
			input: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										StartTimeUnixNano: 2,
									},
								},
							},
						},
					},
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										StartTimeUnixNano: 1,
									},
								},
							},
						},
					},
				},
			},
			expected: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										StartTimeUnixNano: 1,
									},
								},
							},
						},
					},
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										StartTimeUnixNano: 2,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		SortTrace(tt.input)

		assert.Equal(t, tt.expected, tt.input)
	}
}

func TestSortTraceBytes(t *testing.T) {
	numTraces := 100

	// create first trace
	traceBytes := &tempopb.TraceBytes{
		Traces: make([][]byte, numTraces),
	}
	for i := range traceBytes.Traces {
		traceBytes.Traces[i] = make([]byte, rand.Intn(10))
		_, err := rand.Read(traceBytes.Traces[i])
		require.NoError(t, err)
	}

	// dupe
	traceBytes2 := &tempopb.TraceBytes{
		Traces: make([][]byte, numTraces),
	}
	for i := range traceBytes.Traces {
		traceBytes2.Traces[i] = make([]byte, len(traceBytes.Traces[i]))
		copy(traceBytes2.Traces[i], traceBytes.Traces[i])
	}

	// randomize dupe
	rand.Shuffle(len(traceBytes2.Traces), func(i, j int) {
		traceBytes2.Traces[i], traceBytes2.Traces[j] = traceBytes2.Traces[j], traceBytes2.Traces[i]
	})

	assert.NotEqual(t, traceBytes, traceBytes2)

	// sort and compare
	SortTraceBytes(traceBytes)
	SortTraceBytes(traceBytes2)

	assert.Equal(t, traceBytes, traceBytes2)
}

func BenchmarkSortTraceBytes(b *testing.B) {
	numTraces := 100

	traceBytes := &tempopb.TraceBytes{
		Traces: make([][]byte, numTraces),
	}
	for i := range traceBytes.Traces {
		traceBytes.Traces[i] = make([]byte, rand.Intn(10))
		_, err := rand.Read(traceBytes.Traces[i])
		require.NoError(b, err)
	}

	for i := 0; i < b.N; i++ {
		rand.Shuffle(len(traceBytes.Traces), func(i, j int) {
			traceBytes.Traces[i], traceBytes.Traces[j] = traceBytes.Traces[j], traceBytes.Traces[i]
		})
		SortTraceBytes(traceBytes)
	}
}
