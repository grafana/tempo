package querier

import (
	"context"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
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

func (m *mockSharder) Owns(hash string) bool {
	return true
}

func (m *mockSharder) Combine(dataEncoding string, objs ...[]byte) ([]byte, bool) {
	if len(objs) != 2 {
		return nil, false
	}
	combined, wasCombined, _ := model.CombineTraceBytes(objs[0], objs[1], dataEncoding, dataEncoding)
	return combined, wasCombined
}

func TestReturnAllHits(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

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

	r.EnablePolling(&Querier{})

	wal := w.WAL()

	blockCount := 2
	testTraceID := make([]byte, 16)
	_, err = rand.Read(testTraceID)
	require.NoError(t, err)

	// keep track of traces sent
	testTraces := make([]*tempopb.Trace, 0, blockCount)

	// split the same trace across multiple blocks
	for i := 0; i < blockCount; i++ {
		blockID := uuid.New()
		head, err := wal.NewBlock(blockID, util.FakeTenantID, "")
		require.NoError(t, err)

		req := test.MakeRequest(10, testTraceID)
		testTraces = append(testTraces, &tempopb.Trace{Batches: []*v1.ResourceSpans{req.Batch}})
		bReq, err := proto.Marshal(req)
		require.NoError(t, err)

		err = head.Write(testTraceID, bReq)
		require.NoError(t, err, "unexpected error writing req")
		err = head.FlushBuffers()
		require.NoError(t, err)

		_, err = w.CompleteBlock(head, &mockSharder{})
		require.NoError(t, err)
	}

	// sleep for blocklist poll
	time.Sleep(200 * time.Millisecond)

	// find should return both now
	foundBytes, _, err := r.Find(context.Background(), util.FakeTenantID, testTraceID, tempodb.BlockIDMin, tempodb.BlockIDMax)
	assert.NoError(t, err)
	require.Len(t, foundBytes, 2)

	// expected trace
	expectedTrace, _, _, _ := model.CombineTraceProtos(testTraces[0], testTraces[1])
	model.SortTrace(expectedTrace)

	// actual trace
	actualTraceBytes, _, err := model.CombineTraceBytes(foundBytes[1], foundBytes[0], "", "")
	assert.NoError(t, err)
	actualTrace := &tempopb.Trace{}
	err = proto.Unmarshal(actualTraceBytes, actualTrace)
	assert.NoError(t, err)

	model.SortTrace(actualTrace)
	assert.Equal(t, expectedTrace, actualTrace)
}
