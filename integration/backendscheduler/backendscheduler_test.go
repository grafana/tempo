package backendscheduler

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/e2e"
	"github.com/grafana/tempo/cmd/tempo/app"
	e2eBackend "github.com/grafana/tempo/integration/e2e/backend"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const (
	tenant     = "test"
	configFile = "config.yaml"
)

func TestBackendScheduler(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		tenantCount = 1
	)
	defer cancel()

	// e2e
	s, err := e2e.NewScenario("tempo-integration")
	require.NoError(t, err)
	defer s.Close()

	cfg := app.Config{}
	buff, err := os.ReadFile(configFile)
	require.NoError(t, err)
	err = yaml.UnmarshalStrict(buff, &cfg)
	require.NoError(t, err)

	b, err := e2eBackend.New(s, cfg)
	require.NoError(t, err)

	err = b.WaitReady()
	require.NoError(t, err)

	err = b.Ready()
	require.NoError(t, err)

	// Give some time for startup
	time.Sleep(1 * time.Second)

	require.NoError(t, util.CopyFileToSharedDir(s, configFile, "config.yaml"))

	e := b.Endpoint(b.HTTPPort())
	t.Logf("Endpoint: %s", e)
	// cfg.StorageConfig.Trace.Backend = backend.S3
	// cfg.StorageConfig.Trace.S3.Endpoint = e
	// var rr backend.RawReader
	// var ww backend.RawWriter
	// var cc backend.Compactor
	// rr, ww, cc, err = s3.New(cfg.StorageConfig.Trace.S3)
	// require.NoError(t, err)

	// r := backend.NewReader(rr)
	// w := backend.NewWriter(ww)

	// tmpDir := t.TempDir()

	scheduler := util.NewTempoTarget("backend-scheduler", configFile)
	worker := util.NewTempoTarget("backend-worker", configFile)
	require.NoError(t, s.StartAndWaitReady(scheduler, worker))

	// // Setup tempodb with local backend
	// tempodbWriter := setupBackend(t, tmpDir)
	tempodbWriter := setupBackendWithEndpoint(t, &cfg.StorageConfig.Trace, e)

	// // Push some data to a few tenants
	for i := 0; i < tenantCount; i++ {
		testTenant := tenant + strconv.Itoa(i)
		populateBackend(ctx, t, tempodbWriter, testTenant)
	}

	time.Sleep(1 * time.Second)

	require.NoError(t, scheduler.WaitSumMetrics(e2e.Greater(0), "tempo_backend_scheduler_scheduling_cycles_total"))

	time.Sleep(2 * time.Second)

	require.NoError(t, scheduler.WaitSumMetrics(e2e.Greater(0), "tempo_backend_scheduler_jobs_completed_total"))

	// limits, err := overrides.NewOverrides(overrides.Config{
	// 	Defaults: overrides.Overrides{
	// 		// Global: overrides.GlobalOverrides{
	// 		// 	MaxBytesPerTrace: maxBytes,
	// 		// },
	// 		// Ingestion: overrides.IngestionOverrides{
	// 		// 	MaxLocalTracesPerUser: 4,
	// 		// },
	// 	},
	// }, nil, prometheus.DefaultRegisterer)
	// require.NoError(t, err)
	//
	// store := newStore(ctx, t, tmpDir)
	//
	// scheduler, err := backendscheduler.New(backendscheduler.Config{
	// 	ScheduleInterval: 100 * time.Millisecond,
	// }, store, limits)
	// require.NoError(t, err)
	//
	// nextResp, err := scheduler.Next(ctx, &tempopb.NextJobRequest{
	// 	WorkerId: "test-worker",
	// 	Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
	// })
	// require.NoError(t, err)
	// require.Nil(t, nextResp)
	//
	// err = scheduler.ScheduleOnce(ctx)
	// require.NoError(t, err)
	//
	// time.Sleep(100 * time.Millisecond)
	//
	// nextResp, err = scheduler.Next(ctx, &tempopb.NextJobRequest{
	// 	WorkerId: "test-worker",
	// 	Type:     tempopb.JobType_JOB_TYPE_COMPACTION,
	// })
	// require.NoError(t, err)
	// require.Equal(t, tempopb.JobType_JOB_TYPE_COMPACTION, nextResp.Type)
}

func setupBackendWithEndpoint(t testing.TB, cfg *tempodb.Config, endpoint string) tempodb.Writer {
	cfg.Block = &common.BlockConfig{
		IndexDownsampleBytes: 11,
		BloomFP:              .01,
		BloomShardSizeBytes:  100_000,
		Version:              encoding.LatestEncoding().Version(),
		Encoding:             backend.EncNone,
		IndexPageSizeBytes:   1000,
		RowGroupSizeBytes:    30_000_000,
		DedicatedColumns:     backend.DedicatedColumns{{Scope: "span", Name: "key", Type: "string"}},
	}
	cfg.WAL = &wal.Config{
		Filepath: "/var/tempo/wal",
	}
	cfg.S3.Endpoint = endpoint

	_, w, _, err := tempodb.New(cfg, nil, log.NewNopLogger())
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
