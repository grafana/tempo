package tempodb

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

func TestSearchCompleteBlock(t *testing.T) {
	for _, v := range encoding.AllEncodings() {
		vers := v.Version()
		t.Run(vers, func(t *testing.T) {
			testSearchCompleteBlock(t, vers)
		})
	}
}

func testSearchCompleteBlock(t *testing.T, blockVersion string) {
	runCompleteBlockTest(t, blockVersion, func(wantMeta *tempopb.TraceSearchMetadata, searchesThatMatch, searchesThatDontMatch []*tempopb.SearchRequest, meta *backend.BlockMeta, r Reader) {
		ctx := context.Background()

		for _, req := range searchesThatMatch {
			res, err := r.Search(ctx, meta, req, common.DefaultSearchOptions())
			require.NoError(t, err)
			require.Equal(t, 1, len(res.Traces), "search request: %+v", req)
			require.Equal(t, wantMeta, res.Traces[0], "search request:", req)
		}

		for _, req := range searchesThatDontMatch {
			res, err := r.Search(ctx, meta, req, common.DefaultSearchOptions())
			require.NoError(t, err)
			require.Empty(t, res.Traces, "search request:", req)
		}
	})
}

func runCompleteBlockTest(t testing.TB, blockVersion string, runner func(*tempopb.TraceSearchMetadata, []*tempopb.SearchRequest, []*tempopb.SearchRequest, *backend.BlockMeta, Reader)) {
	tempDir := t.TempDir()

	r, w, c, err := New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 17,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              blockVersion,
			IndexPageSizeBytes:   1000,
		},
		WAL: &wal.Config{
			Filepath:       path.Join(tempDir, "wal"),
			IngestionSlack: time.Since(time.Time{}),
		},
		Search: &SearchConfig{
			ChunkSizeBytes:      1_000_000,
			ReadBufferCount:     8,
			ReadBufferSizeBytes: 4 * 1024 * 1024,
		},
		BlocklistPoll: 0,
	}, log.NewNopLogger())
	require.NoError(t, err)

	c.EnableCompaction(&CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
	}, &mockSharder{}, &mockOverrides{})

	r.EnablePolling(&mockJobSharder{})
	rw := r.(*readerWriter)

	id, wantTr, start, end, wantMeta, searchesThatMatch, searchesThatDontMatch, _, _ := trace.SearchTestSuite()

	// Write to wal
	wal := w.WAL()
	head, err := wal.NewBlock(uuid.New(), testTenantID, model.CurrentEncoding)
	require.NoError(t, err)
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)
	b1, err := dec.PrepareForWrite(wantTr, start, end)
	require.NoError(t, err)
	b2, err := dec.ToObject([][]byte{b1})
	require.NoError(t, err)
	err = head.Append(id, b2, start, end)
	require.NoError(t, err, "unexpected error writing req")

	// Complete block
	block, err := w.CompleteBlock(context.Background(), head)
	require.NoError(t, err)
	meta := block.BlockMeta()

	runner(wantMeta, searchesThatMatch, searchesThatDontMatch, meta, rw)
}
