package vparquet5

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func TestCreateBlockHonorsTraceStartEndTimesFromWalMeta(t *testing.T) {
	ctx := context.Background()

	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)

	iter := newTestIterator()

	iter.Add(test.MakeTrace(10, nil), 100, 401)
	iter.Add(test.MakeTrace(10, nil), 101, 402)
	iter.Add(test.MakeTrace(10, nil), 102, 403)

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
	}

	meta := backend.NewBlockMeta("fake", uuid.New(), VersionString)
	meta.TotalObjects = 1
	meta.StartTime = time.Unix(300, 0)
	meta.EndTime = time.Unix(305, 0)

	outMeta, err := CreateBlock(ctx, cfg, meta, iter, r, w)
	require.NoError(t, err)
	require.Equal(t, 300, int(outMeta.StartTime.Unix()))
	require.Equal(t, 305, int(outMeta.EndTime.Unix()))
}

type testIterator struct {
	traces []*tempopb.Trace
}

var _ common.Iterator = (*testIterator)(nil)

func newTestIterator() *testIterator {
	return &testIterator{}
}

func (i *testIterator) Add(tr *tempopb.Trace, _, _ uint32) {
	i.traces = append(i.traces, tr)
}

func (i *testIterator) Next(context.Context) (common.ID, *tempopb.Trace, error) {
	if len(i.traces) == 0 {
		return nil, nil, io.EOF
	}
	tr := i.traces[0]
	i.traces = i.traces[1:]
	return nil, tr, nil
}

func (i *testIterator) Close() {
}

func TestEstimateAttrSizeCountsValueBytes(t *testing.T) {
	for _, size := range []int{1 << 10, 1 << 20} {
		attrs := []Attribute{{Key: "k", Value: []string{strings.Repeat("x", size)}}}
		require.GreaterOrEqual(t, estimateAttrSize(attrs), size/20)
	}

	unsupported := strings.Repeat("x", 1<<10)
	require.GreaterOrEqual(t, estimateAttrSize([]Attribute{{Key: "k", ValueUnsupported: &unsupported}}), (1<<10)/20)

	// Strings under 20 bytes still count.
	require.Greater(t, estimateAttrSize([]Attribute{{Key: "k", Value: []string{"short"}}}), estimateAttrSize([]Attribute{{Key: "k"}}))
}

func TestCreateBlockCutsRowGroupAtMaxSize(t *testing.T) {
	defer func(v int64) { maxRowGroupSizeBytes = v }(maxRowGroupSizeBytes)
	maxRowGroupSizeBytes = 1 << 20

	rawR, rawW, _, err := local.New(&local.Config{Path: t.TempDir()})
	require.NoError(t, err)

	// Unique multi-KB string attributes grow the dictionary far faster than the
	// size estimate.
	iter := newTestIterator()
	n := 0
	for range 64 {
		tr := test.MakeTrace(1, nil)
		for _, rs := range tr.ResourceSpans {
			for _, ss := range rs.ScopeSpans {
				for _, s := range ss.Spans {
					n++
					s.Attributes = append(s.Attributes, &v1_common.KeyValue{
						Key:   "body",
						Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: fmt.Sprintf("%08d", n) + strings.Repeat("x", 16<<10)}},
					})
				}
			}
		}
		iter.Add(tr, 0, 0)
	}

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
		RowGroupSizeBytes:   1 << 40, // never reached by the estimate
	}
	meta := backend.NewBlockMeta("fake", uuid.New(), VersionString)
	meta.TotalObjects = 64

	outMeta, err := CreateBlock(context.Background(), cfg, meta, iter, backend.NewReader(rawR), backend.NewWriter(rawW))
	require.NoError(t, err)
	require.Greater(t, outMeta.TotalRecords, uint32(1))
}
