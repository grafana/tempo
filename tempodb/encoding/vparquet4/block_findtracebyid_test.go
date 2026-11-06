package vparquet4

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sort"
	"strings"
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

func TestBackendBlockFindTraceByID_BloomAndIndexBytes(t *testing.T) {
	var (
		ctx       = context.Background()
		numTraces = 20
		ids       = make([]common.ID, numTraces)
		meta      = backend.NewBlockMeta("fake", uuid.New(), VersionString)
	)
	meta.DedicatedColumns = test.MakeDedicatedColumns()

	// sorted by trace ID, like in a real block
	for i := range ids {
		ids[i] = test.ValidTraceID(nil)
	}
	sort.Slice(ids, func(i, j int) bool { return bytes.Compare(ids[i], ids[j]) < 0 })

	traces := make([]*Trace, 0, numTraces)
	for _, id := range ids {
		tr, _ := traceToParquet(meta, id, test.MakeTrace(2, id), nil)
		traces = append(traces, tr)
	}
	b := makeBackendBlockWithTraces(t, traces)

	// objectSize returns the size of an object in the block directory, which is what a lookup reads
	objectSize := func(t *testing.T, name string) uint64 {
		obj, err := b.r.Read(ctx, name, uuid.UUID(b.meta.BlockID), b.meta.TenantID, nil)
		require.NoError(t, err)
		require.NotEmpty(t, obj)
		return uint64(len(obj))
	}
	bloomSize := func(t *testing.T, id common.ID) uint64 {
		return objectSize(t, common.BloomName(common.ShardKeyForTraceID(id, int(b.meta.BloomShardCount))))
	}

	// a trace ID that is not in the block and rejected by the bloom filter
	absentID := func(t *testing.T) common.ID {
		for range 1000 {
			id := test.ValidTraceID(nil)
			found, _, err := b.checkBloom(ctx, id)
			require.NoError(t, err)
			if !found {
				return id
			}
		}
		require.Fail(t, "no trace ID rejected by the bloom filter")
		return nil
	}

	// bigger than the last trace ID of the block, so the index rules it out
	beyondLastID := bytes.Repeat([]byte{0xff}, 16)

	// failingBlock returns a copy of the block that fails reads of the given object with failErr, "bloom" matches every bloom shard
	failingBlock := func(failName string, failErr error) *backendBlock {
		rawR := &backend.MockRawReader{
			ReadFn: func(_ context.Context, name string, _ backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
				if name == failName || (failName == "bloom" && strings.HasPrefix(name, "bloom")) {
					return nil, 0, failErr
				}
				obj, err := b.r.Read(ctx, name, uuid.UUID(b.meta.BlockID), b.meta.TenantID, cacheInfo)
				return io.NopCloser(bytes.NewReader(obj)), int64(len(obj)), err
			},
		}
		return newBackendBlock(b.meta, backend.NewReader(rawR))
	}

	// blockWithBloom returns a copy of the block that serves the given bloom filter for every shard
	blockWithBloom := func(t *testing.T, filter *bloom.BloomFilter) (*backendBlock, uint64) {
		buf := &bytes.Buffer{}
		_, err := filter.WriteTo(buf)
		require.NoError(t, err)
		bloomObj := buf.Bytes()

		rawR := &backend.MockRawReader{
			ReadFn: func(_ context.Context, name string, _ backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
				if strings.HasPrefix(name, "bloom") {
					return io.NopCloser(bytes.NewReader(bloomObj)), int64(len(bloomObj)), nil
				}
				obj, err := b.r.Read(ctx, name, uuid.UUID(b.meta.BlockID), b.meta.TenantID, cacheInfo)
				return io.NopCloser(bytes.NewReader(obj)), int64(len(obj)), err
			},
		}
		return newBackendBlock(b.meta, backend.NewReader(rawR)), uint64(len(bloomObj))
	}

	t.Run("trace found", func(t *testing.T) {
		for _, indexEnabled := range []string{"0", EnvVarIndexEnabledValue} {
			t.Run(EnvVarIndexName+"="+indexEnabled, func(t *testing.T) {
				t.Setenv(EnvVarIndexName, indexEnabled)

				expectedBytes := bloomSize(t, ids[3])
				if indexEnabled == EnvVarIndexEnabledValue {
					expectedBytes += objectSize(t, common.NameIndex)
				}

				// the data file is read on top of the bloom filter and index
				opts := common.DefaultSearchOptions()
				if opts.TraceByIDReadBufferSize > 0 {
					opts.ReadBufferSize = opts.TraceByIDReadBufferSize
				}
				_, rowGroup, _, err := b.checkIndex(ctx, ids[3])
				require.NoError(t, err)
				pf, rr, err := b.openForSearch(ctx, opts)
				require.NoError(t, err)
				_, err = findTraceByID(ctx, ids[3], b.meta, pf, rowGroup)
				require.NoError(t, err)
				expectedBytes += rr.BytesRead()

				resp, err := b.FindTraceByID(ctx, ids[3], common.DefaultSearchOptions())
				require.NoError(t, err)
				require.NotNil(t, resp.Trace)
				require.Equal(t, expectedBytes, resp.Metrics.InspectedBytes)
			})
		}
	})

	t.Run("rejected by bloom filter", func(t *testing.T) {
		id := absentID(t)

		resp, err := b.FindTraceByID(ctx, id, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, resp.Trace)
		require.Equal(t, bloomSize(t, id), resp.Metrics.InspectedBytes)
	})

	t.Run("rejected by index", func(t *testing.T) {
		t.Setenv(EnvVarIndexName, EnvVarIndexEnabledValue)

		// the bloom filter lets the ID pass, so the index is what rules the block out
		filter := bloom.NewWithEstimates(100, 0.01)
		filter.Add(beyondLastID)
		block, bloomBytes := blockWithBloom(t, filter)

		resp, err := block.FindTraceByID(ctx, beyondLastID, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, resp.Trace)
		require.Equal(t, bloomBytes+objectSize(t, common.NameIndex), resp.Metrics.InspectedBytes)
	})

	t.Run("check bloom", func(t *testing.T) {
		found, bytesRead, err := b.checkBloom(ctx, ids[0])
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, bloomSize(t, ids[0]), bytesRead)

		found, bytesRead, err = failingBlock("bloom", errors.New("boom")).checkBloom(ctx, ids[0])
		require.Error(t, err)
		require.False(t, found)
		require.Zero(t, bytesRead)
	})

	t.Run("check index", func(t *testing.T) {
		testCases := []struct {
			name          string
			indexEnabled  string
			block         *backendBlock
			id            common.ID
			expectedOK    bool
			expectedBytes uint64
			expectedErr   bool
		}{
			{
				name:         "disabled",
				indexEnabled: "0",
				block:        b,
				id:           beyondLastID,
				expectedOK:   true,
			},
			{
				name:          "trace may be in the block",
				indexEnabled:  EnvVarIndexEnabledValue,
				block:         b,
				id:            ids[5],
				expectedOK:    true,
				expectedBytes: objectSize(t, common.NameIndex),
			},
			{
				name:          "ruled out by the index",
				indexEnabled:  EnvVarIndexEnabledValue,
				block:         b,
				id:            beyondLastID,
				expectedOK:    false,
				expectedBytes: objectSize(t, common.NameIndex),
			},
			{
				name:         "index does not exist",
				indexEnabled: EnvVarIndexEnabledValue,
				block:        failingBlock(common.NameIndex, backend.ErrDoesNotExist),
				id:           beyondLastID,
				expectedOK:   true,
			},
			{
				name:         "read error",
				indexEnabled: EnvVarIndexEnabledValue,
				block:        failingBlock(common.NameIndex, errors.New("boom")),
				id:           ids[5],
				expectedErr:  true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Setenv(EnvVarIndexName, tc.indexEnabled)

				ok, _, bytesRead, err := tc.block.checkIndex(ctx, tc.id)
				if tc.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, tc.expectedOK, ok)
				require.Equal(t, tc.expectedBytes, bytesRead)
			})
		}
	})

	t.Run("read error", func(t *testing.T) {
		resp, err := failingBlock("bloom", errors.New("boom")).FindTraceByID(ctx, ids[0], common.DefaultSearchOptions())
		require.Error(t, err)
		require.Nil(t, resp)
	})
}
