package benchmark

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// testBlock writes a block of numTraces traces, with enough row groups to
// exercise per-row-group work.
func testBlock(t *testing.T, numTraces int) (*backend.BlockMeta, backend.Reader, string) {
	t.Helper()

	// Real trace IDs are random across all 16 bytes, and code under test hashes
	// them, so a fixed seed keeps the fixture both realistic and reproducible.
	rng := rand.New(rand.NewSource(0xbeef))

	ids := make([][]byte, 0, numTraces)
	for range numTraces {
		id := make([]byte, 16)
		_, err := rng.Read(id)
		require.NoError(t, err)
		ids = append(ids, test.ValidTraceID(id))
	}

	meta, r, bucket := testBlockWithTraceIDs(t, ids)

	rowGroups, err := rowGroupCount(context.Background(), meta, r)
	require.NoError(t, err)
	require.Greater(t, rowGroups, 1, "need more than one row group to exercise per-row-group work")

	return meta, r, bucket
}

// testBlockWithTraceIDs writes a block holding exactly the given trace IDs,
// using a small row group size. It returns the bucket root alongside the block,
// for tests that address it by path.
func testBlockWithTraceIDs(t *testing.T, ids [][]byte) (*backend.BlockMeta, backend.Reader, string) {
	t.Helper()

	bucket := t.TempDir()
	rawR, rawW, _, err := local.New(&local.Config{Path: bucket})
	require.NoError(t, err)
	r, w := backend.NewReader(rawR), backend.NewWriter(rawW)

	ids = slices.Clone(ids)
	// vParquet blocks are sorted by trace ID.
	sort.Slice(ids, func(i, j int) bool { return bytes.Compare(ids[i], ids[j]) < 0 })

	iter := &sliceIterator{}
	for _, id := range ids {
		iter.add(id, test.MakeTraceWithSpanCount(1, 4, id))
	}

	// test.MakeSpan timestamps spans at time.Now(), so the block's window has
	// to bracket that or every query filters them all out.
	now := time.Now()

	enc := encoding.LatestEncoding()
	meta := backend.NewBlockMeta("test-tenant", uuid.New(), enc.Version())
	meta.TotalObjects = int64(len(ids))
	meta.StartTime = now.Add(-time.Hour)
	meta.EndTime = now.Add(time.Hour)

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
		RowGroupSizeBytes:   8 * 1024,
		Version:             enc.Version(),
	}

	out, err := enc.CreateBlock(context.Background(), cfg, meta, iter, r, w)
	require.NoError(t, err)

	return out, r, bucket
}

type sliceIterator struct {
	ids    []common.ID
	traces []*tempopb.Trace
}

var _ common.Iterator = (*sliceIterator)(nil)

func (i *sliceIterator) add(id common.ID, tr *tempopb.Trace) {
	i.ids = append(i.ids, id)
	i.traces = append(i.traces, tr)
}

func (i *sliceIterator) Next(context.Context) (common.ID, *tempopb.Trace, error) {
	if len(i.ids) == 0 {
		return nil, nil, io.EOF
	}
	id, tr := i.ids[0], i.traces[0]
	i.ids, i.traces = i.ids[1:], i.traces[1:]
	return id, tr, nil
}

func (i *sliceIterator) Close() {}
