package vparquet

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	v2 "github.com/grafana/tempo/pkg/model/v2"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func TestCreateBlockHonorsTraceStartEndTimes(t *testing.T) {
	ctx := context.Background()

	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)

	iter := newTestIterator()

	iter.Add(test.MakeTrace(10, nil), 100, 101)
	iter.Add(test.MakeTrace(10, nil), 101, 102)
	iter.Add(test.MakeTrace(10, nil), 102, 103)

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
	}

	meta := backend.NewBlockMeta("fake", uuid.New(), VersionString, backend.EncNone, "")
	meta.TotalObjects = 1

	outMeta, err := CreateBlock(ctx, cfg, meta, iter, iter.decoder, r, w)
	require.NoError(t, err)
	require.Equal(t, 100, int(outMeta.StartTime.Unix()))
	require.Equal(t, 103, int(outMeta.EndTime.Unix()))
}

type testIterator struct {
	traces  [][]byte
	decoder model.ObjectDecoder
	segment model.SegmentDecoder
}

var _ common.Iterator = (*testIterator)(nil)

func newTestIterator() *testIterator {
	return &testIterator{
		decoder: v2.NewObjectDecoder(),
		segment: v2.NewSegmentDecoder(),
	}
}

func (i *testIterator) Add(tr *tempopb.Trace, start, end uint32) {
	b, _ := i.segment.PrepareForWrite(tr, start, end)
	b2, _ := i.segment.ToObject([][]byte{b})
	i.traces = append(i.traces, b2)
}

func (i *testIterator) Next(ctx context.Context) (common.ID, []byte, error) {
	if len(i.traces) == 0 {
		return nil, nil, io.EOF
	}
	tr := i.traces[0]
	i.traces = i.traces[1:]
	return nil, tr, nil
}

func (i *testIterator) Close() {
}
