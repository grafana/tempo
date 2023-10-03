package v2

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	testTenantID = "fake"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func TestStreamingBlockError(t *testing.T) {
	// no block metas
	_, err := NewStreamingBlock(nil, uuid.New(), "", nil, 0)
	assert.Error(t, err)

	// mixed data encodings
	_, err = NewStreamingBlock(nil, uuid.New(), "", []*backend.BlockMeta{
		backend.NewBlockMeta("", uuid.New(), "", backend.EncNone, "foo"),
		backend.NewBlockMeta("", uuid.New(), "", backend.EncNone, "bar"),
	}, 0)
	assert.Error(t, err)
}

func TestStreamingBlockAddObject(t *testing.T) {
	indexDownsample := 500

	metas := []*backend.BlockMeta{
		{
			StartTime: time.Unix(10000, 0),
			EndTime:   time.Unix(20000, 0),
		},
		{
			StartTime: time.Unix(15000, 0),
			EndTime:   time.Unix(25000, 0),
		},
	}

	numObjects := (rng.Int() % 20) + 1
	cb, err := NewStreamingBlock(&common.BlockConfig{
		BloomFP:              0.01,
		BloomShardSizeBytes:  100,
		IndexDownsampleBytes: indexDownsample,
		Encoding:             backend.EncGZIP,
	}, uuid.New(), testTenantID, metas, numObjects)
	assert.NoError(t, err)

	var minID common.ID
	var maxID common.ID

	expectedRecords := 0
	byteCounter := 0

	ids := make([][]byte, 0)
	for i := 0; i < numObjects; i++ {
		id := make([]byte, 16)
		_, err = rng.Read(id)
		assert.NoError(t, err)

		object := make([]byte, rng.Int()%1024)
		_, err = rng.Read(object)
		assert.NoError(t, err)

		ids = append(ids, id)

		err = cb.AddObject(id, object)
		assert.NoError(t, err)

		byteCounter += len(id) + len(object) + 4 + 4
		if byteCounter > indexDownsample {
			byteCounter = 0
			expectedRecords++
		}

		if len(minID) == 0 || bytes.Compare(id, minID) == -1 {
			minID = id
		}
		if len(maxID) == 0 || bytes.Compare(id, maxID) == 1 {
			maxID = id
		}
	}
	if byteCounter > 0 {
		expectedRecords++
	}

	err = cb.appender.Complete()
	assert.NoError(t, err)
	assert.Equal(t, numObjects, cb.Length())

	// test meta
	meta := cb.BlockMeta()

	assert.Equal(t, time.Unix(10000, 0), meta.StartTime)
	assert.Equal(t, time.Unix(25000, 0), meta.EndTime)
	assert.Equal(t, minID, common.ID(meta.MinID))
	assert.Equal(t, maxID, common.ID(meta.MaxID))
	assert.Equal(t, testTenantID, meta.TenantID)
	assert.Equal(t, numObjects, meta.TotalObjects)
	assert.Greater(t, meta.Size, uint64(0))
	assert.Greater(t, cb.bloom.GetShardCount(), 0)

	// bloom
	for _, id := range ids {
		has := cb.bloom.Test(id)
		assert.True(t, has)
	}

	records := cb.appender.Records()
	assert.Equal(t, expectedRecords, len(records))
	assert.Equal(t, numObjects, cb.CurrentBufferedObjects())
}

func TestStreamingBlockAll(t *testing.T) {
	for i := 0; i < 10; i++ {
		indexDownsampleBytes := rng.Intn(5000) + 1000
		bloomFP := float64(rng.Intn(99)+1) / 100.0
		bloomShardSize := rng.Intn(10_000) + 10_000
		indexPageSize := rng.Intn(5000) + 1000

		for _, enc := range backend.SupportedEncoding {
			t.Run(enc.String(), func(t *testing.T) {
				testStreamingBlockToBackendBlock(t,
					&common.BlockConfig{
						IndexDownsampleBytes: indexDownsampleBytes,
						BloomFP:              bloomFP,
						BloomShardSizeBytes:  bloomShardSize,
						Encoding:             enc,
						IndexPageSizeBytes:   indexPageSize,
					},
				)
			})
		}
	}
}

func testStreamingBlockToBackendBlock(t *testing.T, cfg *common.BlockConfig) {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)
	require.NoError(t, err, "error creating backend")
	_, ids, reqs := streamingBlock(t, cfg, w)

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
		foundBytes, err := backendBlock.find(context.Background(), id)
		require.NoError(t, err)

		require.Equal(t, reqs[i], foundBytes)
	}

	// test Iterator
	idsToObjs := map[uint32][]byte{}
	for i, id := range ids {
		idsToObjs[util.TokenForTraceID(id)] = reqs[i]
	}
	sort.Slice(ids, func(i, j int) bool { return bytes.Compare(ids[i], ids[j]) == -1 })

	iterator, err := backendBlock.Iterator(50 * 1024)
	require.NoError(t, err, "error getting iterator")
	i := 0
	for {
		id, obj, err := iterator.NextBytes(context.Background())
		if id == nil {
			break
		}

		require.NoError(t, err)
		require.Equal(t, ids[i], []byte(id))
		require.Equal(t, idsToObjs[util.TokenForTraceID(id)], obj)
		i++
	}
	require.Equal(t, len(ids), i)
}

func streamingBlock(t *testing.T, cfg *common.BlockConfig, w backend.Writer) (*StreamingBlock, [][]byte, [][]byte) {
	buffer := &bytes.Buffer{}
	writer := bufio.NewWriter(buffer)
	dataWriter, err := NewDataWriter(writer, backend.EncNone)
	require.NoError(t, err)
	appender := NewAppender(dataWriter)

	numMsgs := 1000
	reqs := make([][]byte, 0, numMsgs)
	ids := make([][]byte, 0, numMsgs)
	var maxID, minID []byte
	for i := 0; i < numMsgs; i++ {
		id := make([]byte, 16)
		rng.Read(id)
		ids = append(ids, id)
		req := make([]byte, rng.Intn(100)+1)
		rng.Read(req)
		reqs = append(reqs, req)

		err = appender.Append(id, req)
		require.NoError(t, err, "unexpected error writing req")

		if len(maxID) == 0 || bytes.Compare(id, maxID) == 1 {
			maxID = id
		}
		if len(minID) == 0 || bytes.Compare(id, minID) == -1 {
			minID = id
		}
	}
	err = appender.Complete()
	require.NoError(t, err)
	err = writer.Flush()
	require.NoError(t, err, "unexpected error flushing writer")

	originatingMeta := backend.NewBlockMeta(testTenantID, uuid.New(), "should_be_ignored", backend.EncGZIP, "")
	originatingMeta.StartTime = time.Now().Add(-5 * time.Minute)
	originatingMeta.EndTime = time.Now().Add(5 * time.Minute)
	originatingMeta.DataEncoding = "foo"
	originatingMeta.TotalObjects = numMsgs

	// calc expected records
	dataReader, err := NewDataReader(backend.NewContextReaderWithAllReader(bytes.NewReader(buffer.Bytes())), backend.EncNone)
	require.NoError(t, err)
	iter := newRecordIterator(appender.Records(),
		dataReader,
		NewObjectReaderWriter())

	block, err := NewStreamingBlock(cfg, originatingMeta.BlockID, originatingMeta.TenantID, []*backend.BlockMeta{originatingMeta}, originatingMeta.TotalObjects)
	require.NoError(t, err, "unexpected error completing block")

	expectedBloomShards := block.bloom.GetShardCount()

	ctx := context.Background()
	for {
		id, data, err := iter.NextBytes(ctx)
		if !errors.Is(err, io.EOF) {
			require.NoError(t, err)
		}

		if id == nil {
			break
		}

		err = block.AddObject(id, data)
		require.NoError(t, err)
	}
	var tracker backend.AppendTracker
	tracker, _, err = block.FlushBuffer(ctx, tracker, w)
	require.NoError(t, err)
	_, err = block.Complete(ctx, tracker, w)
	require.NoError(t, err)

	// test downsample config
	require.True(t, bytes.Equal(block.BlockMeta().MinID, minID))
	require.True(t, bytes.Equal(block.BlockMeta().MaxID, maxID))
	require.Equal(t, originatingMeta.StartTime, block.BlockMeta().StartTime)
	require.Equal(t, originatingMeta.EndTime, block.BlockMeta().EndTime)
	require.Equal(t, originatingMeta.TenantID, block.BlockMeta().TenantID)
	require.Equal(t, originatingMeta.DataEncoding, block.BlockMeta().DataEncoding)
	require.Equal(t, expectedBloomShards, int(block.BlockMeta().BloomShardCount))

	// Verify block size was written
	require.Greater(t, block.BlockMeta().Size, uint64(0))

	return block, ids, reqs
}

const benchDownsample = 1024 * 1024

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

func BenchmarkWriteS2(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncS2, benchDownsample, false)
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

func BenchmarkReadS2(b *testing.B) {
	benchmarkCompressBlock(b, backend.EncS2, benchDownsample, true)
}

// Download a block from your backend and place in ./benchmark_block/<tenant id>/<guid>

//nolint:unparam
func benchmarkCompressBlock(b *testing.B, encoding backend.Encoding, indexDownsample int, benchRead bool) {
	rawR, _, _, err := local.New(&local.Config{
		Path: "./benchmark_block",
	})
	require.NoError(b, err, "error creating backend")

	r := backend.NewReader(rawR)
	meta, err := r.BlockMeta(context.Background(), uuid.MustParse("20a614f8-8cda-4b9d-9789-cb626f9fab28"), "1")
	require.NoError(b, err)

	backendBlock, err := NewBackendBlock(meta, r)
	require.NoError(b, err, "error creating backend block")

	iter, err := backendBlock.Iterator(10 * 1024 * 1024)
	require.NoError(b, err, "error creating iterator")

	backendTmpDir := b.TempDir()

	_, rawW, _, err := local.New(&local.Config{
		Path: backendTmpDir,
	})
	require.NoError(b, err, "error creating backend")

	w := backend.NewWriter(rawW)
	if !benchRead {
		b.ResetTimer()
	}

	block, err := NewStreamingBlock(&common.BlockConfig{
		IndexDownsampleBytes: indexDownsample,
		BloomFP:              .05,
		Encoding:             encoding,
		IndexPageSizeBytes:   10 * 1024 * 1024,
		BloomShardSizeBytes:  100000,
	}, uuid.New(), meta.TenantID, []*backend.BlockMeta{meta}, meta.TotalObjects)
	require.NoError(b, err, "unexpected error completing block")

	ctx := context.Background()
	for {
		id, data, err := iter.NextBytes(ctx)
		if !errors.Is(err, io.EOF) {
			require.NoError(b, err)
		}
		if errors.Is(err, io.EOF) {
			break
		}

		err = block.AddObject(id, data)
		require.NoError(b, err)
	}
	var tracker backend.AppendTracker
	tracker, _, err = block.FlushBuffer(ctx, tracker, w)
	require.NoError(b, err)
	_, err = block.Complete(ctx, tracker, w)
	require.NoError(b, err)

	lastRecord := block.appender.Records()[len(block.appender.Records())-1]
	fmt.Println("size: ", lastRecord.Start+uint64(lastRecord.Length))

	if !benchRead {
		return
	}

	b.ResetTimer()

	fullFilename := path.Join(backendTmpDir, block.meta.TenantID, block.meta.BlockID.String(), "data")
	file, err := os.Open(fullFilename)
	require.NoError(b, err)
	pr, err := NewDataReader(backend.NewContextReaderWithAllReader(file), encoding)
	require.NoError(b, err)

	var tempBuffer []byte
	o := NewObjectReaderWriter()
	for {
		tempBuffer, _, err = pr.NextPage(tempBuffer)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(b, err)

		bufferReader := bytes.NewReader(tempBuffer)

		for {
			_, _, err = o.UnmarshalObjectFromReader(bufferReader)
			if errors.Is(err, io.EOF) {
				break
			}
		}
	}
}
