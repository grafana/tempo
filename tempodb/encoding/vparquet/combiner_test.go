package vparquet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCombiner(t *testing.T) {

	methods := []func(a, b *Trace) (*Trace, int){
		func(a, b *Trace) (*Trace, int) {
			c := NewCombiner()
			c.Consume(a)
			c.Consume(b)
			return c.Result()
		},
	}

	tests := []struct {
		traceA        *Trace
		traceB        *Trace
		expectedTotal int
	}{
		{
			traceA:        nil,
			traceB:        &Trace{},
			expectedTotal: -1,
		},
		{
			traceA:        &Trace{},
			traceB:        nil,
			expectedTotal: -1,
		},
		{
			traceA:        &Trace{},
			traceB:        &Trace{},
			expectedTotal: 0,
		},
		/*{
			traceA:        sameTrace,
			traceB:        sameTrace,
			expectedTotal: 100,
		},*/
	}

	for _, tt := range tests {
		for _, m := range methods {
			_, actualTotal := m(tt.traceA, tt.traceB)
			assert.Equal(t, tt.expectedTotal, actualTotal)
		}
	}
}
