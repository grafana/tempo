package benchmark

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"math/rand"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	v1_common "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
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

	// Spans are timestamped at now, so the block's window has to bracket that
	// or every query filters them all out.
	now := time.Now()

	iter := &sliceIterator{}
	for _, id := range ids {
		iter.add(id, fixedShapeTrace(id, now))
	}

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

// fixedShapeTrace builds a trace whose shape is the same for every ID. The
// writer cuts row groups on an estimate of that shape, so the fixture's row
// group layout is identical on every run, unlike test.MakeTrace* which
// randomizes attribute, event and link counts from the unseeded global RNG.
func fixedShapeTrace(id []byte, now time.Time) *tempopb.Trace {
	str := func(k, v string) *v1_common.KeyValue {
		return &v1_common.KeyValue{Key: k, Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: v}}}
	}
	hexID := hex.EncodeToString(id)

	spans := make([]*v1_trace.Span, 4)
	for i := range spans {
		spanID := slices.Clone(id[8:])
		spanID[7] ^= byte(i)
		spans[i] = &v1_trace.Span{
			TraceId:           id,
			SpanId:            spanID,
			Name:              "test",
			Kind:              v1_trace.Span_SPAN_KIND_CLIENT,
			Status:            &v1_trace.Status{Code: 1, Message: "OK"},
			StartTimeUnixNano: uint64(now.UnixNano()),
			EndTimeUnixNano:   uint64(now.Add(time.Second).UnixNano()),
			Attributes: []*v1_common.KeyValue{
				str("attr.a", hexID[:10]),
				str("attr.b", hexID[10:20]),
				str("attr.c", hexID[20:30]),
				str("attr.d", hexID[:10]),
				str("attr.e", hexID[10:20]),
				str("key", "value"),
			},
		}
	}

	return &tempopb.Trace{ResourceSpans: []*v1_trace.ResourceSpans{{
		Resource: &v1_resource.Resource{Attributes: []*v1_common.KeyValue{
			str("random.res.attr", hexID[20:30]),
			str("service.name", "test-service"),
		}},
		ScopeSpans: []*v1_trace.ScopeSpans{{
			Scope: &v1_common.InstrumentationScope{Name: "super library", Version: "1.0.1"},
			Spans: spans,
		}},
	}}}
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
