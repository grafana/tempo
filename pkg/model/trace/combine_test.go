package trace

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"sort"
	"strconv"
	"testing"

	"github.com/grafana/tempo/v2/pkg/tempopb"
	"github.com/grafana/tempo/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombineProtoTotals(t *testing.T) {
	methods := []func(a, b *tempopb.Trace) (*tempopb.Trace, int){
		func(a, b *tempopb.Trace) (*tempopb.Trace, int) {
			c := NewCombiner(0)
			_, err := c.Consume(a)
			require.NoError(t, err)
			_, err = c.Consume(b)
			require.NoError(t, err)
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

func TestCombinerChecksMaxBytes(t *testing.T) {
	// Ensure that the combiner checks max bytes when consuming a trace.
	for _, maxBytes := range []int{0, 100, 1000, 10000} {
		c := NewCombiner(maxBytes)
		curSize := 0

		// attempt up to 20 traces to exceed max bytes
		for i := 0; i < 20; i++ {
			tr := test.MakeTraceWithSpanCount(1, 1, []byte{0x01})
			curSize += tr.Size()

			_, err := c.Consume(tr)
			if curSize > maxBytes && maxBytes != 0 {
				require.Error(t, err)
				continue
			}
			require.NoError(t, err)
		}
	}
}

func TestTokenForIDCollision(t *testing.T) {
	// Estimate the hash collision rate of tokenForID.

	n := 1_000_000
	h := newHash()
	buf := make([]byte, 4)

	tokens := map[token]struct{}{}
	IDs := [][]byte{}

	spanID := make([]byte, 8)
	for i := 0; i < n; i++ {
		_, err := crand.Read(spanID)
		require.NoError(t, err)

		cpy := append([]byte(nil), spanID...)
		IDs = append(IDs, cpy)

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

	// There shouldn't be any collisions.
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
	requests := 100 // 100K spans per part
	spansEach := 1000
	id := test.ValidTraceID(nil)

	methods := []struct {
		name   string
		method func(traces []*tempopb.Trace) int
	}{
		{
			"Combiner",
			func(traces []*tempopb.Trace) int {
				c := NewCombiner(0)
				for i := range traces {
					_, err := c.ConsumeWithFinal(traces[i], i == len(traces)-1)
					require.NoError(b, err)
				}
				_, spanCount := c.Result()
				return spanCount
			},
		},
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
