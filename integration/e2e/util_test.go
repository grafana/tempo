package e2e

import (
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

func makeThriftBatch() *thrift.Batch {
	return makeThriftBatchWithSpanCount(1)
}

func makeThriftBatchWithSpanCount(n int) *thrift.Batch {
	var spans []*thrift.Span

	traceIDLow := rand.Int63()
	traceIDHigh := rand.Int63()
	for i := 0; i < n; i++ {
		spans = append(spans, &thrift.Span{
			TraceIdLow:    traceIDLow,
			TraceIdHigh:   traceIDHigh,
			SpanId:        rand.Int63(),
			ParentSpanId:  0,
			OperationName: "my operation",
			References:    nil,
			Flags:         0,
			StartTime:     time.Now().Unix(),
			Duration:      1,
			Tags:          nil,
			Logs:          nil,
		})
	}
	return &thrift.Batch{Spans: spans}
}

func extractHexID(batch *thrift.Batch) string {
	return fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)
}

func assertTrace(t *testing.T, reader io.Reader, expectedBatches int, expectedName string) {
	out := &tempopb.Trace{}
	unmarshaller := &jsonpb.Unmarshaler{}
	require.NoError(t, unmarshaller.Unmarshal(reader, out))
	require.Len(t, out.Batches, expectedBatches)
	assert.Equal(t, expectedName, out.Batches[0].InstrumentationLibrarySpans[0].Spans[0].Name)
}
