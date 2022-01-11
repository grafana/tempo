package trace

import (
	"hash/fnv"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestCombineProtoTotals(t *testing.T) {
	sameTrace := test.MakeTraceWithSpanCount(10, 10, []byte{0x01, 0x03})

	tests := []struct {
		traceA        *tempopb.Trace
		traceB        *tempopb.Trace
		expectedTotal int
	}{
		{
			traceA:        nil,
			traceB:        test.MakeTraceWithSpanCount(10, 10, []byte{0x01, 0x03}),
			expectedTotal: -1,
		},
		{
			traceA:        test.MakeTraceWithSpanCount(10, 10, []byte{0x01, 0x03}),
			traceB:        nil,
			expectedTotal: -1,
		},
		{
			traceA:        test.MakeTraceWithSpanCount(10, 10, []byte{0x01, 0x03}),
			traceB:        test.MakeTraceWithSpanCount(10, 10, []byte{0x01, 0x01}),
			expectedTotal: 200,
		},
		{
			traceA:        sameTrace,
			traceB:        sameTrace,
			expectedTotal: 100,
		},
	}

	for _, tt := range tests {
		_, actualTotal := CombineTraceProtos(tt.traceA, tt.traceB)
		assert.Equal(t, tt.expectedTotal, actualTotal)
	}
}

func BenchmarkTokenForID(b *testing.B) {
	h := fnv.New32()
	id := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	buffer := make([]byte, 4)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tokenForID(h, buffer, 0, id)
	}
}
