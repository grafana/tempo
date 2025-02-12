package generator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func BenchmarkOTLPDecoder(b *testing.B) {
	traceBytes := marshalBatches(b, []*v1.ResourceSpans{
		test.MakeBatch(15, []byte("test batch 1")),
		test.MakeBatch(50, []byte("test batch 2")),
		test.MakeBatch(42, []byte("test batch 3")),
	})

	b.ReportAllocs()
	decoder := newOTLPDecoder()
	for i := 0; i < b.N; i++ {
		reqs, err := decoder.decode(traceBytes)
		require.NoError(b, err)
		for range reqs { // nolint:revive // we want to run the side effects of ranging itself
		}
	}
}

func marshalBatches(t testing.TB, batches []*v1.ResourceSpans) []byte {
	t.Helper()

	trace := tempopb.Trace{ResourceSpans: batches}

	m, err := trace.Marshal()
	require.NoError(t, err)

	return m
}
