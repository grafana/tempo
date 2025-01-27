package livetraces

import (
	"math/rand/v2"
	"testing"
	"time"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestLiveTracesSizesAndLen(t *testing.T) {
	lt := New[*v1.ResourceSpans](func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) })

	expectedSz := uint64(0)
	expectedLen := uint64(0)

	for i := 0; i < 100; i++ {
		id := test.ValidTraceID(nil)
		tr := test.MakeTrace(rand.IntN(5)+1, id)

		cutTime := time.Now()

		// add some traces and confirm size/len
		expectedLen++
		for _, rs := range tr.ResourceSpans {
			expectedSz += uint64(rs.Size())
			lt.Push(id, rs, 0)
		}

		require.Equal(t, expectedSz, lt.Size())
		require.Equal(t, expectedLen, lt.Len())

		// cut some traces and confirm size/len
		cutTraces := lt.CutIdle(cutTime, false)
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
