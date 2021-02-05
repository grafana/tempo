package querier

import (
	"context"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
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

func (m *mockSharder) Combine(objA []byte, objB []byte) []byte {
	combined, _ := util.CombineTraces(objA, objB)
	return combined
}

func TestReturnAllHits(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
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
			Encoding:        backend.EncNone,
			IndexDownsample: 10,
			BloomFP:         .05,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 50 * time.Millisecond,
	}, log.NewNopLogger())
	assert.NoError(t, err, "unexpected error creating tempodb")

	wal := w.WAL()
	assert.NoError(t, err)

	blockCount := 2
	testTraceID := make([]byte, 16)
	_, err = rand.Read(testTraceID)
	assert.NoError(t, err)

	// keep track of traces sent
	testTraces := make([]*tempopb.Trace, 0, blockCount)

	// split the same trace across multiple blocks
	for i := 0; i < blockCount; i++ {
		blockID := uuid.New()
		head, err := wal.NewBlock(blockID, util.FakeTenantID)
		assert.NoError(t, err)

		req := test.MakeRequest(10, testTraceID)
		testTraces = append(testTraces, &tempopb.Trace{Batches: []*v1.ResourceSpans{req.Batch}})
		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)

		err = head.Write(testTraceID, bReq)
		assert.NoError(t, err, "unexpected error writing req")

		complete, err := w.CompleteBlock(head, &mockSharder{})
		assert.NoError(t, err)

		err = w.WriteBlock(context.Background(), complete)
		assert.NoError(t, err)
	}

	// sleep for blocklist poll
	time.Sleep(100 * time.Millisecond)

	// find should return both now
	foundBytes, err := r.Find(context.Background(), util.FakeTenantID, testTraceID, tempodb.BlockIDMin, tempodb.BlockIDMax)
	assert.NoError(t, err)
	require.Len(t, foundBytes, 2)

	// expected trace
	expectedTrace, _, _, _ := util.CombineTraceProtos(testTraces[0], testTraces[1])
	test.SortTrace(expectedTrace)

	// actual trace
	actualTraceBytes, err := util.CombineTraces(foundBytes[1], foundBytes[0])
	assert.NoError(t, err)
	actualTrace := &tempopb.Trace{}
	err = proto.Unmarshal(actualTraceBytes, actualTrace)
	assert.NoError(t, err)

	test.SortTrace(actualTrace)
	assert.Equal(t, expectedTrace, actualTrace)
}
