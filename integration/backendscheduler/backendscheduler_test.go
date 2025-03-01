package backendscheduler

import (
	"context"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/backendscheduler"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

const (
	tenant = "test"
)

func TestBackendScheduler(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		tenantCount = 1
	)

	defer cancel()

	tmpDir := t.TempDir()

	t.Logf("Temp dir: %s", tmpDir)

	// Setup tempodb with local backend
	tempodbWriter := setupBackend(t, tmpDir)

	// Push some data to a few tenants
	for i := 0; i < tenantCount; i++ {
		testTenant := tenant + strconv.Itoa(i)
		populateBackend(ctx, t, tempodbWriter, testTenant)
	}

	store := newStore(ctx, t, tmpDir)

	scheduler, err := backendscheduler.New(backendscheduler.Config{
		Enabled:          true,
		ScheduleInterval: 100 * time.Millisecond,
	}, store)
	require.NoError(t, err)

	nextResp, err := scheduler.Next(ctx, &tempopb.NextJobRequest{
		WorkerId: "test-worker",
		Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
	})
	require.NoError(t, err)
	require.Nil(t, nextResp)

	err = scheduler.ScheduleOnce(ctx)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	nextResp, err = scheduler.Next(ctx, &tempopb.NextJobRequest{
		WorkerId: "test-worker",
		Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
	})
	require.NoError(t, err)
	require.Equal(t, tempopb.JobType_JOB_TYPE_COMPACTION, nextResp.Type)
}

func newStore(ctx context.Context, t testing.TB, tmpDir string) storage.Store {
	return newStoreWithLogger(ctx, t, test.NewTestingLogger(t), tmpDir)
}

func newStoreWithLogger(ctx context.Context, t testing.TB, log log.Logger, tmpDir string) storage.Store {
	s, err := storage.NewStore(storage.Config{
		Trace: tempodb.Config{
			Backend: backend.Local,
			Local: &local.Config{
				Path: path.Join(tmpDir, "traces"),
			},
			Block: &common.BlockConfig{
				IndexDownsampleBytes: 2,
				BloomFP:              0.01,
				BloomShardSizeBytes:  100_000,
				Version:              encoding.LatestEncoding().Version(),
				Encoding:             backend.EncLZ4_1M,
				IndexPageSizeBytes:   1000,
			},
			WAL: &wal.Config{
				Filepath: tmpDir,
			},
			BlocklistPoll: 100 * time.Millisecond,
		},
	}, nil, log)
	require.NoError(t, err)

	s.EnablePolling(ctx, &ownsEverythingSharder{})

	// NOTE: Call EnableCompaction to set the overrides, but pass a canceled
	// context so we don't run the compaction and retention loops.
	canceldCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = s.EnableCompaction(canceldCtx, &tempodb.CompactorConfig{
		ChunkSizeBytes:          10,
		MaxCompactionRange:      24 * time.Hour,
		BlockRetention:          0,
		CompactedBlockRetention: 0,
		MaxCompactionObjects:    1000,
		MaxBlockBytes:           100_000_000, // Needs to be sized appropriately for the test data or no jobs will get scheduled.
	}, &ownsEverythingSharder{}, &mockOverrides{})
	require.NoError(t, err)

	return s
}

func setupBackend(t testing.TB, tempDir string) tempodb.Writer {
	_, w, _, err := tempodb.New(&tempodb.Config{
		Backend: backend.Local,
		Pool: &pool.Config{
			MaxWorkers: 10,
			QueueDepth: 100,
		},
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		Block: &common.BlockConfig{
			IndexDownsampleBytes: 11,
			BloomFP:              .01,
			BloomShardSizeBytes:  100_000,
			Version:              encoding.LatestEncoding().Version(),
			Encoding:             backend.EncNone,
			IndexPageSizeBytes:   1000,
			RowGroupSizeBytes:    30_000_000,
			DedicatedColumns:     backend.DedicatedColumns{{Scope: "span", Name: "key", Type: "string"}},
		},
		WAL: &wal.Config{
			Filepath: path.Join(tempDir, "wal"),
		},
		BlocklistPoll: 0,
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	return w
}

func populateBackend(ctx context.Context, t testing.TB, w tempodb.Writer, tenantID string) {
	wal := w.WAL()

	blockCount := 4
	recordCount := 100

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	for i := 0; i < blockCount; i++ {
		blockID := backend.NewUUID()
		meta := &backend.BlockMeta{BlockID: blockID, TenantID: tenantID, DataEncoding: model.CurrentEncoding}
		head, err := wal.NewBlock(meta, model.CurrentEncoding)
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			id := test.ValidTraceID(nil)
			req := test.MakeTrace(10, id)

			writeTraceToWal(t, head, dec, id, req, 0, 0)
		}

		_, err = w.CompleteBlock(ctx, head)
		require.NoError(t, err)
	}
}

func writeTraceToWal(t require.TestingT, b common.WALBlock, dec model.SegmentDecoder, id common.ID, tr *tempopb.Trace, start, end uint32) {
	b1, err := dec.PrepareForWrite(tr, 0, 0)
	require.NoError(t, err)

	b2, err := dec.ToObject([][]byte{b1})
	require.NoError(t, err)

	err = b.Append(id, b2, start, end, true)
	require.NoError(t, err, "unexpected error writing req")
}

// OwnsEverythingSharder owns everything.
var OwnsEverythingSharder = ownsEverythingSharder{}

type ownsEverythingSharder struct{}

func (ownsEverythingSharder) Owns(_ string) bool {
	return true
}

func (m *ownsEverythingSharder) Combine(dataEncoding string, _ string, objs ...[]byte) ([]byte, bool, error) {
	return model.StaticCombiner.Combine(dataEncoding, objs...)
}

func (m *ownsEverythingSharder) RecordDiscardedSpans(int, string, string, string, string) {}

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
