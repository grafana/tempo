package encoding

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
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
	v1 "github.com/grafana/tempo/tempodb/encoding/v1"
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

	block, ids, reqs := completeBlock(t, &BlockConfig{
		IndexDownsample: 13,
		BloomFP:         .01,
		Encoding:        backend.EncGZIP,
	}, tempDir)

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

func TestCompleteBlockAll(t *testing.T) {
	for _, enc := range backend.SupportedEncoding {
		testCompleteBlockToBackendBlock(t,
			&BlockConfig{
				IndexDownsample: 13,
				BloomFP:         .01,
				Encoding:        enc,
			},
		)
	}
}

func testCompleteBlockToBackendBlock(t *testing.T, cfg *BlockConfig) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	require.NoError(t, err, "unexpected error creating temp dir")

	block, ids, reqs := completeBlock(t, cfg, tempDir)

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

	iterator, err := backendBlock.Iterator(10 * 1024 * 1024)
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

func completeBlock(t *testing.T, cfg *BlockConfig, tempDir string) (*CompleteBlock, [][]byte, [][]byte) {
	rand.Seed(time.Now().Unix())

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
	err := appender.Complete()
	require.NoError(t, err)
	err = writer.Flush()
	require.NoError(t, err, "unexpected error flushing writer")

	originatingMeta := backend.NewBlockMeta(testTenantID, uuid.New(), "should_be_ignored", backend.EncGZIP)
	originatingMeta.StartTime = time.Now().Add(-5 * time.Minute)
	originatingMeta.EndTime = time.Now().Add(5 * time.Minute)

	iterator := v0.NewRecordIterator(appender.Records(), bytes.NewReader(buffer.Bytes()))
	block, err := NewCompleteBlock(cfg, originatingMeta, iterator, numMsgs, tempDir, "")
	require.NoError(t, err, "unexpected error completing block")

	// test downsample config
	require.Equal(t, numMsgs/cfg.IndexDownsample+1, len(block.records))
	require.True(t, block.FlushedTime().IsZero())

	require.True(t, bytes.Equal(block.meta.MinID, minID))
	require.True(t, bytes.Equal(block.meta.MaxID, maxID))
	require.Equal(t, originatingMeta.StartTime, block.meta.StartTime)
	require.Equal(t, originatingMeta.EndTime, block.meta.EndTime)
	require.Equal(t, originatingMeta.TenantID, block.meta.TenantID)

	return block, ids, reqs
}

const benchDownsample = 200

func BenchmarkWriteGzip(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncGZIP, benchDownsample, false)
}
func BenchmarkWriteSnappy(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncSnappy, benchDownsample, false)
}
func BenchmarkWriteLZ4256(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncLZ4_256k, benchDownsample, false)
}
func BenchmarkWriteLZ41M(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncLZ4_1M, benchDownsample, false)
}
func BenchmarkWriteNone(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncNone, benchDownsample, false)
}

func BenchmarkWriteZstd(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncZstd, benchDownsample, false)
}

func BenchmarkReadGzip(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncGZIP, benchDownsample, true)
}
func BenchmarkReadSnappy(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncSnappy, benchDownsample, true)
}
func BenchmarkReadLZ4256(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncLZ4_256k, benchDownsample, true)
}
func BenchmarkReadLZ41M(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncLZ4_1M, benchDownsample, true)
}
func BenchmarkReadNone(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncNone, benchDownsample, true)
}

func BenchmarkReadZstd(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncZstd, benchDownsample, true)
}

// Download a block from your backend and place in ./benchmark_block/fake/<guid>
//nolint:unparam
func benchmarkCompressBlock(b *testing.B, encoding backend.Encoding, indexDownsample int, benchRead bool) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	require.NoError(b, err, "unexpected error creating temp dir")

	r, _, _, err := local.New(&local.Config{
		Path: "./benchmark_block",
	})
	require.NoError(b, err, "error creating backend")

	backendBlock, err := NewBackendBlock(backend.NewBlockMeta("fake", uuid.MustParse("9f15417a-1242-40e4-9de3-a057d3b176c1"), "v0", backend.EncNone), r)
	require.NoError(b, err, "error creating backend block")

	iterator, err := backendBlock.Iterator(10 * 1024 * 1024)
	require.NoError(b, err, "error creating iterator")

	if !benchRead {
		b.ResetTimer()
	}

	originatingMeta := backend.NewBlockMeta(testTenantID, uuid.New(), "should_be_ignored", backend.EncGZIP)
	cb, err := NewCompleteBlock(&BlockConfig{
		IndexDownsample: indexDownsample,
		BloomFP:         .05,
		Encoding:        encoding,
	}, originatingMeta, iterator, 10000, tempDir, "")
	require.NoError(b, err, "error creating block")

	lastRecord := cb.records[len(cb.records)-1]
	fmt.Println("size: ", lastRecord.Start+uint64(lastRecord.Length))

	if !benchRead {
		return
	}

	b.ResetTimer()
	file, err := os.Open(cb.fullFilename())
	require.NoError(b, err)
	pr, err := v1.NewPageReader(file, encoding)
	require.NoError(b, err)
	iterator = v1.NewPagedIterator(10*1024*1024, common.Records(cb.records), pr)

	for {
		id, _, err := iterator.Next()
		if err != io.EOF {
			require.NoError(b, err)
		}
		if id == nil {
			break
		}
	}
}
