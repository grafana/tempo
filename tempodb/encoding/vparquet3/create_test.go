package vparquet3

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
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

	meta := backend.NewBlockMeta("fake", uuid.New(), VersionString, backend.EncNone, model.CurrentEncoding)
	meta.TotalObjects = 1
	meta.StartTime = time.Unix(300, 0)
	meta.EndTime = time.Unix(305, 0)

	outMeta, err := CreateBlock(ctx, cfg, meta, iter, r, w)
	require.NoError(t, err)
	require.Equal(t, 300, int(outMeta.StartTime.Unix()))
	require.Equal(t, 305, int(outMeta.EndTime.Unix()))
}

// func TestEstimateTraceSize(t *testing.T) {
// 	f := "<put data.parquet file here>"
// 	file, err := os.OpenFile(f, os.O_RDONLY, 0644)
// 	require.NoError(t, err)

// 	count := 10000

// 	totalProtoSz := 0
// 	totalParqSz := 0

// 	r := parquet.NewGenericReader[*Trace](file)
// 	tr := make([]*Trace, 1)
// 	for {
// 		count--
// 		if count == 0 {
// 			break
// 		}

// 		_, err := r.Read(tr)
// 		require.NoError(t, err)

// 		if tr[0] == nil {
// 			break
// 		}
// 		protoTr, err := parquetTraceToTempopbTrace(tr[0])
// 		require.NoError(t, err)

// 		protoSz := protoTr.Size()
// 		parqSz := estimateTraceSize(tr[0])

// 		totalProtoSz += protoSz
// 		totalParqSz += parqSz

// 		if float64(parqSz)/float64(protoSz) < .7 ||
// 			float64(parqSz)/float64(protoSz) > 1.3 {
// 			fmt.Println(protoTr)
// 			break
// 		}
// 	}
// 	fmt.Println(totalParqSz, totalProtoSz)
// }

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
