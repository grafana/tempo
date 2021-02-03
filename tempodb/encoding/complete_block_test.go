package encoding

import (
	"bufio"
	"bytes"
	"context"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCombiner struct {
}

func (m *mockCombiner) Combine(objA []byte, objB []byte) []byte {
	if len(objA) > len(objB) {
		return objA
	}

	return objB
}

func TestCompleteBlock(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	require.NoError(t, err, "unexpected error creating temp dir")

	block, ids, reqs := completeBlock(t, tempDir)

	// test Find
	for i, id := range ids {
		foundBytes, err := block.Find(id, &mockCombiner{})
		assert.NoError(t, err)

		assert.Equal(t, reqs[i], foundBytes)
		assert.True(t, block.bloom.Test(id))
	}

	// confirm order
	var prev *common.Record
	for _, r := range block.records {
		if prev != nil {
			assert.Greater(t, r.Start, prev.Start)
		}

		prev = r
	}
}

func TestCompleteBlockToBackendBlock(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	require.NoError(t, err, "unexpected error creating temp dir")

	block, ids, reqs := completeBlock(t, tempDir)

	backendTmpDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(backendTmpDir)
	require.NoError(t, err, "unexpected error creating temp dir")

	r, w, _, err := local.New(&local.Config{
		Path: backendTmpDir,
	})
	require.NoError(t, err, "error creating backend")

	err = block.Write(context.Background(), w)
	require.EqualError(t, err, "remove : no such file or directory") // we expect an error here b/c there is no wal file to clear

	// meta?
	uuids, err := r.Blocks(context.Background(), testTenantID)
	require.NoError(t, err, "error listing blocks")
	require.Len(t, uuids, 1)

	meta, err := r.BlockMeta(context.Background(), uuids[0], testTenantID)
	require.NoError(t, err, "error getting meta")

	backendBlock, err := NewBackendBlock(meta, r)
	require.NoError(t, err, "error creating block")

	// test Find
	for i, id := range ids {
		foundBytes, err := backendBlock.Find(context.Background(), id)
		assert.NoError(t, err)

		assert.Equal(t, reqs[i], foundBytes)
	}

	// test Iterator
	idsToObjs := map[uint32][]byte{}
	for i, id := range ids {
		idsToObjs[util.TokenForTraceID(id)] = reqs[i]
	}
	sort.Slice(ids, func(i int, j int) bool { return bytes.Compare(ids[i], ids[j]) == -1 })

	iterator, err := backendBlock.Iterator(10)
	require.NoError(t, err, "error getting iterator")
	i := 0
	for {
		id, obj, err := iterator.Next()
		if id == nil {
			break
		}

		assert.NoError(t, err)
		assert.Equal(t, ids[i], []byte(id))
		assert.Equal(t, idsToObjs[util.TokenForTraceID(id)], obj)
		i++
	}
	assert.Equal(t, len(ids), i)
}

func completeBlock(t *testing.T, tempDir string) (*CompleteBlock, [][]byte, [][]byte) {
	rand.Seed(time.Now().Unix())

	indexDownsample := 13
	buffer := &bytes.Buffer{}
	writer := bufio.NewWriter(buffer)
	appender := v0.NewAppender(writer)

	numMsgs := 1000
	reqs := make([][]byte, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	var maxID, minID []byte
	for i := 0; i < numMsgs; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		req := test.MakeRequest(rand.Int()%10, id)
		ids = append(ids, id)
		bReq, err := proto.Marshal(req)
		require.NoError(t, err)
		reqs = append(reqs, bReq)

		err = appender.Append(id, bReq)
		require.NoError(t, err, "unexpected error writing req")

		if len(maxID) == 0 || bytes.Compare(id, maxID) == 1 {
			maxID = id
		}
		if len(minID) == 0 || bytes.Compare(id, minID) == -1 {
			minID = id
		}
	}
	appender.Complete()
	err := writer.Flush()
	require.NoError(t, err, "unexpected error flushing writer")

	originatingMeta := backend.NewBlockMeta(testTenantID, uuid.New(), "should_be_ignored", backend.EncGZIP)
	originatingMeta.StartTime = time.Now().Add(-5 * time.Minute)
	originatingMeta.EndTime = time.Now().Add(5 * time.Minute)

	iterator := v0.NewRecordIterator(appender.Records(), bytes.NewReader(buffer.Bytes()))
	block, err := NewCompleteBlock(&BlockConfig{
		BloomFP:         .01,
		IndexDownsample: indexDownsample,
		Encoding:        backend.EncLZ4_1M,
	}, originatingMeta, iterator, numMsgs, tempDir, "")
	require.NoError(t, err, "unexpected error completing block")

	// test downsample config
	require.Equal(t, numMsgs/indexDownsample+1, len(block.records))
	require.True(t, block.FlushedTime().IsZero())

	require.True(t, bytes.Equal(block.meta.MinID, minID))
	require.True(t, bytes.Equal(block.meta.MaxID, maxID))
	require.Equal(t, originatingMeta.StartTime, block.meta.StartTime)
	require.Equal(t, originatingMeta.EndTime, block.meta.EndTime)
	require.Equal(t, originatingMeta.TenantID, block.meta.TenantID)

	return block, ids, reqs
}
