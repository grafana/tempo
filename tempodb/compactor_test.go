package tempodb

import (
	"bytes"
	"context"
	"encoding/binary"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/model"
	v1 "github.com/grafana/tempo/pkg/model/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

type mockSharder struct{}

func (m *mockSharder) Owns(string) bool {
	return true
}

func (m *mockSharder) RecordDiscardedSpans(int, string, string, string, string) {}

type mockJobSharder struct{}

func (m *mockJobSharder) Owns(string) bool { return true }

type mockOverrides struct {
	blockRetention      time.Duration
	disabled            bool
	maxBytesPerTrace    int
	maxCompactionWindow time.Duration
}

func (m *mockOverrides) BlockRetentionForTenant(_ string) time.Duration {
	return m.blockRetention
}

func (m *mockOverrides) CompactionDisabledForTenant(_ string) bool {
	return m.disabled
}

func (m *mockOverrides) MaxBytesPerTraceForTenant(_ string) int {
	return m.maxBytesPerTrace
}

func (m *mockOverrides) MaxCompactionRangeForTenant(_ string) time.Duration {
	return m.maxCompactionWindow
}

func TestCompactionDropsTraces(t *testing.T) {
	for _, enc := range encoding.AllEncodingsForWrites() {
		t.Run(enc.Version(), func(t *testing.T) {
			t.Parallel()
			testCompactionDropsTraces(t, enc.Version())
		})
	}
}

func testCompactionDropsTraces(t *testing.T, targetBlockVersion string) {
	tempDir := t.TempDir()

	r, w, _, err := New(&Config{
		Backend: backend.Local,
		Pool: &pool.Config{
			MaxWorkers: 10,
			QueueDepth: 100,
		},
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			BloomFP:             .01,
			BloomShardSizeBytes: 100_000,
			Version:             targetBlockVersion,
			RowGroupSizeBytes:   30_000_000,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	wal := w.WAL()
	require.NoError(t, err)

	dec := model.MustNewSegmentDecoder(v1.Encoding)

	recordCount := 100
	allIDs := make([]common.ID, 0, recordCount)

	// write a bunch of dummy data
	blockID := backend.NewUUID()
	meta := &backend.BlockMeta{BlockID: blockID, TenantID: testTenantID}
	head, err := wal.NewBlock(meta, v1.Encoding)
	require.NoError(t, err)

	for j := 0; j < recordCount; j++ {
		id := test.ValidTraceID(nil)
		allIDs = append(allIDs, id)

		obj, err := dec.PrepareForWrite(test.MakeTrace(1, id), 0, 0)
		require.NoError(t, err)

		obj2, err := dec.ToObject([][]byte{obj})
		require.NoError(t, err)

		err = head.Append(id, obj2, 0, 0, true)
		require.NoError(t, err, "unexpected error writing req")
	}

	firstBlock, err := w.CompleteBlock(context.Background(), head)
	require.NoError(t, err)

	// choose a random id to drop
	dropID := allIDs[rand.Intn(len(allIDs))]

	rw := r.(*readerWriter)
	// force compact to a new block
	opts := common.CompactionOptions{
		BlockConfig:      *rw.cfg.Block,
		OutputBlocks:     1,
		MaxBytesPerTrace: 0,

		// hook to drop the trace
		DropObject: func(id common.ID) bool {
			return bytes.Equal(id, dropID)
		},

		// setting to prevent panics.
		BytesWritten:      func(_, _ int) {},
		ObjectsCombined:   func(_, _ int) {},
		ObjectsWritten:    func(_, _ int) {},
		SpansDiscarded:    func(_, _, _ string, _ int) {},
		DisconnectedTrace: func() {},
		RootlessTrace:     func() {},
	}

	enc, err := encoding.FromVersion(targetBlockVersion)
	require.NoError(t, err)

	compactor := enc.NewCompactor(opts)
	newMetas, err := compactor.Compact(context.Background(), log.NewNopLogger(), rw.r, rw.w, []*backend.BlockMeta{firstBlock.BlockMeta()})
	require.NoError(t, err)

	// require new meta has len 1
	require.Len(t, newMetas, 1)

	secondBlock, err := enc.OpenBlock(newMetas[0], rw.r)
	require.NoError(t, err)

	// search for all ids. confirm they all return except the dropped one
	for _, id := range allIDs {
		tr, err := secondBlock.FindTraceByID(context.Background(), id, common.DefaultSearchOptions())
		require.NoError(t, err)

		if bytes.Equal(id, dropID) {
			require.Nil(t, tr)
		} else {
			require.NotNil(t, tr)
		}
	}
}

func TestCompactWithConfig(t *testing.T) {
	for _, enc := range encoding.AllEncodingsForWrites() {
		t.Run(enc.Version(), func(t *testing.T) {
			t.Parallel()
			testCompactWithConfig(t, enc.Version())
		})
	}
}

func testCompactWithConfig(t *testing.T, targetBlockVersion string) {
	tempDir := t.TempDir()

	_, w, c, err := New(&Config{
		Backend: backend.Local,
		Pool: &pool.Config{
			MaxWorkers: 10,
			QueueDepth: 100,
		},
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			BloomFP:             .01,
			BloomShardSizeBytes: 100_000,
			Version:             targetBlockVersion,
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()

	blocks := cutTestBlocks(t, w, testTenantID, 10, 10)
	metas := make([]*backend.BlockMeta, 0)
	for _, b := range blocks {
		metas = append(metas, b.BlockMeta())
	}

	_, err = c.CompactWithConfig(
		ctx,
		metas,
		testTenantID,
		&CompactorConfig{
			MaxCompactionRange:      24 * time.Hour,
			BlockRetention:          0,
			CompactedBlockRetention: 0,
			MaxCompactionObjects:    1000,
			MaxBlockBytes:           100_000_000, // Needs to be sized appropriately for the test data
		},
		&mockSharder{},
		&mockOverrides{},
	)
	require.NoError(t, err)
}

func cutTestBlocks(t testing.TB, w Writer, tenantID string, blockCount int, recordCount int) []common.BackendBlock {
	blocks := make([]common.BackendBlock, 0)
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	wal := w.WAL()
	for i := 0; i < blockCount; i++ {
		meta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: tenantID}
		head, err := wal.NewBlock(meta, model.CurrentEncoding)
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			id := makeTraceID(i, j)
			tr := test.MakeTrace(1, id)
			now := uint32(time.Now().Unix())
			writeTraceToWal(t, head, dec, id, tr, now, now)
		}

		b, err := w.CompleteBlock(context.Background(), head)
		require.NoError(t, err)
		blocks = append(blocks, b)
	}

	return blocks
}

func makeTraceID(i int, j int) []byte {
	id := make([]byte, 16)
	binary.LittleEndian.PutUint64(id, uint64(i))
	binary.LittleEndian.PutUint64(id[8:], uint64(j))
	return id
}

func TestCompactWithConfigUnsupportedVersion(t *testing.T) {
	tempDir := t.TempDir()

	// Create backend reader/writer directly to write the block meta
	localCfg := &local.Config{Path: path.Join(tempDir, "traces")}
	_, rawW, _, err := local.New(localCfg)
	require.NoError(t, err)
	backendW := backend.NewWriter(rawW)

	_, _, c, err := New(&Config{
		Backend: backend.Local,
		Pool: &pool.Config{
			MaxWorkers: 10,
			QueueDepth: 100,
		},
		Local: localCfg,
		Block: &common.BlockConfig{
			BloomFP:             .01,
			BloomShardSizeBytes: 100_000,
			Version:             "vParquet4",
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()

	// Create a block meta with an unsupported preview version and write it to storage
	meta := &backend.BlockMeta{
		BlockID:  backend.NewUUID(),
		TenantID: testTenantID,
		Version:  "vParquet5-preview6",
	}
	err = backendW.WriteBlockMeta(ctx, meta)
	require.NoError(t, err)

	// Try to compact
	_, err = c.CompactWithConfig(
		ctx,
		[]*backend.BlockMeta{meta},
		testTenantID,
		&CompactorConfig{
			MaxCompactionRange: 24 * time.Hour,
		},
		&mockSharder{},
		&mockOverrides{},
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "compaction not supported for block version vParquet5-preview6")
}
