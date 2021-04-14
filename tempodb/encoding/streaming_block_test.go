package encoding

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"

	"github.com/stretchr/testify/assert"
)

const (
	testTenantID = "fake"
)

func TestCompactorBlockError(t *testing.T) {
	_, err := NewStreamingBlock(nil, uuid.New(), "", nil, 0)
	assert.Error(t, err)
}

func TestCompactorBlockAddObject(t *testing.T) {
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

	numObjects := (rand.Int() % 20) + 1
	cb, err := NewStreamingBlock(&BlockConfig{
		BloomFP:              .01,
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
		_, err = rand.Read(id)
		assert.NoError(t, err)

		object := make([]byte, rand.Int()%1024)
		_, err = rand.Read(object)
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

	// bloom
	for _, id := range ids {
		has := cb.bloom.Test(id)
		assert.True(t, has)
	}

	records := cb.appender.Records()
	assert.Equal(t, expectedRecords, len(records))
	assert.Equal(t, numObjects, cb.CurrentBufferedObjects())
}

/* jpe - restore
func TestStreamingBlockAll(t *testing.T) {
	for _, enc := range backend.SupportedEncoding {
		t.Run(enc.String(), func(t *testing.T) {
			testCompleteBlockToBackendBlock(t,
				&BlockConfig{
					IndexDownsampleBytes: 1000,
					BloomFP:              .01,
					Encoding:             enc,
					IndexPageSizeBytes:   1000,
				},
			)
		})
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
	require.NoError(t, err, "error writing backend")

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
		id, obj, err := iterator.Next(context.Background())
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
	appender := NewAppender(v0.NewDataWriter(writer))

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

	// calc expected records
	byteCounter := 0
	expectedRecords := 0
	for _, rec := range appender.Records() {
		byteCounter += int(rec.Length)
		if byteCounter > cfg.IndexDownsampleBytes {
			byteCounter = 0
			expectedRecords++
		}
	}
	if byteCounter > 0 {
		expectedRecords++
	}

	iterator := NewRecordIterator(appender.Records(), bytes.NewReader(buffer.Bytes()), v0.NewObjectReaderWriter())
	block, err := NewCompleteBlock(cfg, originatingMeta, iterator, numMsgs, tempDir)
	require.NoError(t, err, "unexpected error completing block")

	// test downsample config
	require.Equal(t, expectedRecords, len(block.records))
	require.True(t, block.FlushedTime().IsZero())
	require.True(t, bytes.Equal(block.meta.MinID, minID))
	require.True(t, bytes.Equal(block.meta.MaxID, maxID))
	require.Equal(t, originatingMeta.StartTime, block.meta.StartTime)
	require.Equal(t, originatingMeta.EndTime, block.meta.EndTime)
	require.Equal(t, originatingMeta.TenantID, block.meta.TenantID)

	// Verify block size was written
	require.Greater(t, block.meta.Size, uint64(0))

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
		IndexDownsampleBytes: indexDownsample,
		BloomFP:              .05,
		Encoding:             encoding,
	}, originatingMeta, iterator, 10000, tempDir)
	require.NoError(b, err, "error creating block")

	lastRecord := cb.records[len(cb.records)-1]
	fmt.Println("size: ", lastRecord.Start+uint64(lastRecord.Length))

	if !benchRead {
		return
	}

	b.ResetTimer()
	file, err := os.Open(cb.fullFilename())
	require.NoError(b, err)
	pr, err := v2.NewDataReader(v0.NewDataReader(backend.NewContextReaderWithAllReader(file)), encoding)
	require.NoError(b, err)
	iterator = newPagedIterator(10*1024*1024, common.Records(cb.records), pr, backendBlock.encoding.newObjectReaderWriter())

	for {
		id, _, err := iterator.Next(context.Background())
		if err != io.EOF {
			require.NoError(b, err)
		}
		if id == nil {
			break
		}
	}
}
*/
