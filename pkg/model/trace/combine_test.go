package trace

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sort"
	"strconv"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombineProtoTotals(t *testing.T) {

	methods := []func(a, b *tempopb.Trace) (*tempopb.Trace, int){
		CombineTraceProtos,
		func(a, b *tempopb.Trace) (*tempopb.Trace, int) {
			c := NewCombiner()
			c.Consume(a)
			c.Consume(b)
			return c.Result()
		},
	}

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
		for _, m := range methods {
			_, actualTotal := m(tt.traceA, tt.traceB)
			assert.Equal(t, tt.expectedTotal, actualTotal)
		}
	}
}

func TestTokenForIDCollision(t *testing.T) {

	n := 100_000_0
	h := newHash()
	buf := make([]byte, 4)

	tokens := map[token]struct{}{}
	IDs := [][]byte{}

	spanID := make([]byte, 8)
	for i := 0; i < n; i++ {
		rand.Read(spanID)

		copy := append([]byte(nil), spanID...)
		IDs = append(IDs, copy)

		tokens[tokenForID(h, buf, 0, spanID)] = struct{}{}
	}

	// Ensure no duplicate span IDs accidentally generated
	sort.Slice(IDs, func(i, j int) bool {
		return bytes.Compare(IDs[i], IDs[j]) == -1
	})
	for i := 1; i < len(IDs); i++ {
		if bytes.Equal(IDs[i-1], IDs[i]) {
			panic("same span ID was generated, oops")
		}
	}

	missing := n - len(tokens)
	if missing > 0 {
		fmt.Printf("missing 1 out of every %.2f spans", float32(n)/float32(missing))
	}

	require.Equal(t, n, len(tokens))
}

func BenchmarkTokenForID(b *testing.B) {
	h := newHash()
	id := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	buffer := make([]byte, 4)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tokenForID(h, buffer, 0, id)
	}
}

func BenchmarkCombine(b *testing.B) {
	parts := []int{2, 3, 4, 8}
	requests := 100
	spansEach := 1000
	id := test.ValidTraceID(nil)

	methods := []struct {
		name   string
		method func(traces []*tempopb.Trace) int
	}{
		{
			"CombineTraceProtos",
			func(traces []*tempopb.Trace) int {
				var tr *tempopb.Trace
				var spanCount int
				for _, t := range traces {
					tr, spanCount = CombineTraceProtos(tr, t)
				}
				return spanCount
			}},
		{
			"Combiner",
			func(traces []*tempopb.Trace) int {
				c := NewCombiner()
				c.ConsumeAll(traces...)
				_, spanCount := c.Result()
				return spanCount
			}},
	}
	for _, p := range parts {
		b.Run(strconv.Itoa(p), func(b *testing.B) {
			for _, m := range methods {
				b.Run(m.name, func(b *testing.B) {
					for n := 0; n < b.N; n++ {

						// Generate input data. Since combination is destructive
						// this must be done each time.
						b.StopTimer()
						var traces []*tempopb.Trace
						for i := 0; i < p; i++ {
							traces = append(traces, test.MakeTraceWithSpanCount(requests, spansEach, id))
						}
						b.StartTimer()

						m.method(traces)
					}
				})
			}
		})
	}
}
