package querier

import (
	"context"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	v1 "github.com/grafana/tempo/pkg/model/v1"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

type mockSharder struct {
}

func (m *mockSharder) Owns(string) bool {
	return true
}

func (m *mockSharder) Combine(dataEncoding string, objs ...[]byte) ([]byte, bool, error) {
	if len(objs) != 2 {
		return nil, false, nil
	}
	return model.ObjectCombiner.Combine(dataEncoding, objs...)
}

func TestReturnAllHits(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "")
	defer os.RemoveAll(tempDir)
	require.NoError(t, err, "unexpected error creating temp dir")

	r, w, _, err := tempodb.New(&tempodb.Config{
		Backend: "local",
		Pool: &pool.Config{
			MaxWorkers: 10,
			QueueDepth: 100,
		},
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &encoding.BlockConfig{
			Encoding:             backend.EncNone,
			IndexDownsampleBytes: 10,
			BloomFP:              0.01,
			BloomShardSizeBytes:  100_000,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll:         50 * time.Millisecond,
		BlocklistPollFallback: true,
	}, log.NewNopLogger())
	require.NoError(t, err, "unexpected error creating tempodb")

	r.EnablePolling(nil)

	wal := w.WAL()

	blockCount := 2
	testTraceID := make([]byte, 16)
	_, err = rand.Read(testTraceID)
	require.NoError(t, err)

	// keep track of traces sent
	testTraces := make([]*tempopb.Trace, 0, blockCount)

	d := v1.NewDecoder()

	// split the same trace across multiple blocks
	for i := 0; i < blockCount; i++ {
		blockID := uuid.New()
		head, err := wal.NewBlock(blockID, util.FakeTenantID, "")
		require.NoError(t, err)

		req := test.MakeTrace(10, testTraceID)
		testTraces = append(testTraces, req)
		bReq, err := d.Marshal(req)
		require.NoError(t, err)

		err = head.Append(testTraceID, bReq)
		require.NoError(t, err, "unexpected error writing req")

		_, err = w.CompleteBlock(head, &mockSharder{})
		require.NoError(t, err)
	}

	// sleep for blocklist poll
	time.Sleep(200 * time.Millisecond)

	// find should return both now
	foundBytes, _, failedBLocks, err := r.Find(context.Background(), util.FakeTenantID, testTraceID, tempodb.BlockIDMin, tempodb.BlockIDMax)
	require.NoError(t, err)
	require.Nil(t, failedBLocks)
	require.Len(t, foundBytes, 2)

	// expected trace
	expectedTrace, _ := trace.CombineTraceProtos(testTraces[0], testTraces[1])
	trace.SortTrace(expectedTrace)

	// actual trace
	actualTraceBytes, _, err := model.ObjectCombiner.Combine(v1.Encoding, foundBytes...)
	require.NoError(t, err)
	actualTrace, err := d.PrepareForRead(actualTraceBytes)
	require.NoError(t, err)

	trace.SortTrace(actualTrace)
	require.Equal(t, expectedTrace, actualTrace)
}
