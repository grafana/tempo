package vparquet4

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/v3/pkg/io"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

func TestBackendBlockFindTraceByID(t *testing.T) {
	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)
	cr := &countingRawReader{RawReader: rawR}

	r := backend.NewReader(cr)
	w := backend.NewWriter(rawW)
	ctx := context.Background()

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
	}

	// Test data - sorted by trace ID
	// Find trace by ID uses the column and page bounds,
	// which by default only stores 16 bytes, which is the first
	// half of the trace ID (which is stored as 32 hex text)
	// Therefore it is important that the test data here has
	// full-length trace IDs.
	var traces []*Trace
	for i := 0; i < 16; i++ {
		bar := "bar"
		traces = append(traces, &Trace{
			TraceID: test.ValidTraceID(nil),
			ResourceSpans: []ResourceSpans{
				{
					Resource: Resource{
						ServiceName: "s",
					},
					ScopeSpans: []ScopeSpans{
						{
							Spans: []Span{
								{
									Name: "hello",
									Attrs: []Attribute{
										attr("foo", bar),
									},
									SpanID:       []byte{},
									ParentSpanID: []byte{},
								},
							},
						},
					},
				},
			},
		})
	}

	// Sort
	sort.Slice(traces, func(i, j int) bool {
		return bytes.Compare(traces[i].TraceID, traces[j].TraceID) == -1
	})

	meta := backend.NewBlockMeta("fake", uuid.New(), VersionString)
	meta.TotalObjects = int64(len(traces))
	s, newMeta := newStreamingBlock(ctx, cfg, meta, r, w, tempo_io.NewBufferedWriter)

	// Write test data, occasionally flushing (cutting new row group)
	rowGroupSize := 5
	for _, tr := range traces {
		err := s.Add(tr, 0, 0)
		require.NoError(t, err)
		if s.CurrentBufferedObjects() >= rowGroupSize {
			_, err = s.Flush()
			require.NoError(t, err)
		}
	}
	_, err = s.Complete()
	require.NoError(t, err)

	b := newBackendBlock(newMeta, r)

	// Now find and verify all test traces
	for _, tr := range traces {
		wantProto := ParquetTraceToTempopbTrace(meta, tr)

		gotProto, err := b.FindTraceByID(ctx, tr.TraceID, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.Equal(t, wantProto, gotProto.Trace)
		require.Greater(t, gotProto.Metrics.InspectedBytes, uint64(60000)) // approximate value
	}

	// A smaller TraceByIDReadBufferSize replaces ReadBufferSize, so the lookup takes more reads.
	readsWith := func(bufSize int) int64 {
		opts := common.DefaultSearchOptions()
		opts.TraceByIDReadBufferSize = bufSize
		before := cr.reads.Load()
		_, err := b.FindTraceByID(ctx, traces[0].TraceID, opts)
		require.NoError(t, err)
		return cr.reads.Load() - before
	}
	require.Greater(t, readsWith(64), readsWith(0))
}

type countingRawReader struct {
	backend.RawReader
	reads atomic.Int64
}

func (c *countingRawReader) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, cacheInfo *backend.CacheInfo) error {
	c.reads.Add(1)
	return c.RawReader.ReadRange(ctx, name, keypath, offset, buffer, cacheInfo)
}

func TestBackendBlockFindTraceByID_TestData(t *testing.T) {
	rawR, _, _, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	ctx := context.Background()

	blocks, _, _, err := r.Blocks(ctx, "single-tenant")
	require.NoError(t, err)
	assert.Len(t, blocks, 1)

	meta, err := r.BlockMeta(ctx, blocks[0], "single-tenant")
	require.NoError(t, err)

	b := newBackendBlock(meta, r)

	iter, err := b.rawIter(context.Background(), newRowPool(10))
	require.NoError(t, err)

	sch := parquet.SchemaOf(new(Trace))
	for {
		_, row, err := iter.Next(context.Background())
		require.NoError(t, err)

		if row == nil {
			break
		}

		tr := &Trace{}
		err = sch.Reconstruct(tr, row)
		require.NoError(t, err)

		protoTr, err := b.FindTraceByID(ctx, tr.TraceID, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, protoTr)
		require.NotNil(t, protoTr.Trace)
	}
}

func BenchmarkFindTraceByID(b *testing.B) {
	ctx := context.TODO()
	traceID := []byte{}
	block := blockForBenchmarks(b)

	// index := genIndex(b, block)
	// writeBlockMeta(ctx, ww, meta, &common.ShardedBloomFilter{}, index)

	for _, tc := range []string{"0", EnvVarIndexEnabledValue} {
		b.Run(EnvVarIndexName+"="+tc, func(b *testing.B) {
			os.Setenv(EnvVarIndexName, tc)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				tr, err := block.FindTraceByID(ctx, traceID, common.DefaultSearchOptions())
				require.NoError(b, err)
				require.NotNil(b, tr)
				require.NotNil(b, tr.Trace)
			}
		})
	}
}

// blockObjectReader serves the objects of a block and counts the bytes read from them. Objects in overrides are
// replaced by the given content or fail with the given error.
type blockObjectReader struct {
	backend.RawReader
	r         backend.Reader
	meta      *backend.BlockMeta
	overrides map[string]objectOverride
	bytesRead atomic.Uint64
}

type objectOverride struct {
	obj []byte
	err error
}

func (m *blockObjectReader) Read(ctx context.Context, name string, _ backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
	var (
		obj []byte
		err error
	)
	if ov, ok := m.overrides[name]; ok {
		obj, err = ov.obj, ov.err
	} else {
		obj, err = m.r.Read(ctx, name, uuid.UUID(m.meta.BlockID), m.meta.TenantID, cacheInfo)
	}
	if err != nil {
		return nil, 0, err
	}

	m.bytesRead.Add(uint64(len(obj)))
	return io.NopCloser(bytes.NewReader(obj)), int64(len(obj)), nil
}

func (m *blockObjectReader) ReadRange(ctx context.Context, name string, _ backend.KeyPath, offset uint64, buffer []byte, cacheInfo *backend.CacheInfo) error {
	err := m.r.ReadRange(ctx, name, uuid.UUID(m.meta.BlockID), m.meta.TenantID, offset, buffer, cacheInfo)
	if err == nil {
		m.bytesRead.Add(uint64(len(buffer)))
	}
	return err
}

func TestBackendBlockFindTraceByID_BloomAndIndexBytes(t *testing.T) {
	var (
		ctx       = context.Background()
		numTraces = 20
		ids       = make([]common.ID, numTraces)
		meta      = backend.NewBlockMeta("fake", uuid.New(), VersionString)
	)
	meta.DedicatedColumns = test.MakeDedicatedColumns()

	// fixed IDs make the outcome of the bloom filter deterministic, they are sorted like in a real block
	newID := func(first byte) common.ID {
		id := bytes.Repeat([]byte{0xa5}, 16)
		id[0] = first
		return id
	}
	traces := make([]*Trace, 0, numTraces)
	for i := range ids {
		ids[i] = newID(byte(i + 1))
		tr, _ := traceToParquet(meta, ids[i], test.MakeTrace(2, ids[i]), nil)
		traces = append(traces, tr)
	}
	b := makeBackendBlockWithTraces(t, traces)

	// a trace ID that is not in the block and rejected by its bloom filter
	absentID := newID(0x80)
	// bigger than the last trace ID of the block, so the index rules it out
	beyondLastID := bytes.Repeat([]byte{0xff}, 16)

	bloomName := func(id common.ID) string {
		return common.BloomName(common.ShardKeyForTraceID(id, int(b.meta.BloomShardCount)))
	}
	objectSize := func(t *testing.T, name string) uint64 {
		obj, err := b.r.Read(ctx, name, uuid.UUID(b.meta.BlockID), b.meta.TenantID, nil)
		require.NoError(t, err)
		require.NotEmpty(t, obj)
		return uint64(len(obj))
	}

	// blockWith returns a copy of the block that reads the objects like the given reader
	blockWith := func(overrides map[string]objectOverride) (*backendBlock, *blockObjectReader) {
		rawR := &blockObjectReader{r: b.r, meta: b.meta, overrides: overrides}
		return newBackendBlock(b.meta, backend.NewReader(rawR)), rawR
	}

	t.Run("trace found", func(t *testing.T) {
		for _, indexEnabled := range []string{"0", EnvVarIndexEnabledValue} {
			t.Run(EnvVarIndexName+"="+indexEnabled, func(t *testing.T) {
				t.Setenv(EnvVarIndexName, indexEnabled)

				block, reader := blockWith(nil)
				resp, err := block.FindTraceByID(ctx, ids[3], common.DefaultSearchOptions())
				require.NoError(t, err)
				require.NotNil(t, resp.Trace)

				// every byte read from the block is reported, which includes the bloom filter and the index
				require.Equal(t, reader.bytesRead.Load(), resp.Metrics.InspectedBytes)
			})
		}
	})

	t.Run("trace found without index", func(t *testing.T) {
		t.Setenv(EnvVarIndexName, EnvVarIndexEnabledValue)

		block, reader := blockWith(map[string]objectOverride{common.NameIndex: {err: backend.ErrDoesNotExist}})
		resp, err := block.FindTraceByID(ctx, ids[3], common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, resp.Trace)
		require.Equal(t, reader.bytesRead.Load(), resp.Metrics.InspectedBytes)
	})

	t.Run("rejected by bloom filter", func(t *testing.T) {
		resp, err := b.FindTraceByID(ctx, absentID, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, resp.Trace)
		require.Equal(t, objectSize(t, bloomName(absentID)), resp.Metrics.InspectedBytes)
	})

	t.Run("rejected by index", func(t *testing.T) {
		t.Setenv(EnvVarIndexName, EnvVarIndexEnabledValue)

		// the bloom filter lets the ID pass, so the index is what rules the block out
		filter := bloom.NewWithEstimates(100, 0.01)
		filter.Add(beyondLastID)
		buf := &bytes.Buffer{}
		_, err := filter.WriteTo(buf)
		require.NoError(t, err)
		bloomBytes := uint64(buf.Len())

		block, _ := blockWith(map[string]objectOverride{bloomName(beyondLastID): {obj: buf.Bytes()}})
		resp, err := block.FindTraceByID(ctx, beyondLastID, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, resp.Trace)
		require.Equal(t, bloomBytes+objectSize(t, common.NameIndex), resp.Metrics.InspectedBytes)
	})

	t.Run("read error", func(t *testing.T) {
		t.Setenv(EnvVarIndexName, EnvVarIndexEnabledValue)

		for name, failName := range map[string]string{"bloom filter": bloomName(ids[0]), "index": common.NameIndex} {
			t.Run(name, func(t *testing.T) {
				block, _ := blockWith(map[string]objectOverride{failName: {err: errors.New("boom")}})
				resp, err := block.FindTraceByID(ctx, ids[0], common.DefaultSearchOptions())
				require.Error(t, err)
				require.Nil(t, resp)
			})
		}
	})
}
