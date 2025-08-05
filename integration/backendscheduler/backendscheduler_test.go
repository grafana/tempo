package backendscheduler

import (
	"context"
	"fmt"
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
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const (
	tenant = "test"
)

func TestBackendSchedulerConfigurations(t *testing.T) {
	cases := []struct {
		name       string
		configFile string
	}{
		{
			name:       "default",
			configFile: "config.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testWithConfig(t, tc.configFile)
		})
	}
}

func testWithConfig(t *testing.T, configFile string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	scheduler := util.NewTempoTarget("backend-scheduler", "config.yaml")
	require.NoError(t, s.StartAndWaitReady(scheduler))

	// Setup tempodb with local backend
	tempodbWriter := setupBackendWithEndpoint(t, &cfg.StorageConfig.Trace, e)

	cases := []struct {
		name                      string
		tenantCount               int
		blockCount                int
		expectedBlocks            int // accumulated expected blocks.  Each test adds this to the total
		expectedOutstandingBlocks int // accumulated expected outstanding blocks.  Each test adds this to the total, but 1 outstanding block is between 2-4 blocks.
	}{
		{
			name:                      "a bunch of tenants with 1 block each is 0 outstanding",
			tenantCount:               11,
			blockCount:                1,
			expectedBlocks:            1,
			expectedOutstandingBlocks: 0,
		},
		{
			name:                      "a bunch of tenants with 12 blocks each 12 outstanding",
			tenantCount:               11,
			blockCount:                12,
			expectedBlocks:            12,
			expectedOutstandingBlocks: 12,
		},
	}
	var tenants []string

	type expectations struct {
		blocks      int
		outstanding int
	}

	// We continue to write blocks through these tests, so our expectations need
	// to grow as well.
	accumulant := make(map[string]*expectations)
	totalOutstanding := 0
	totalBlocksWritten := 0

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Populate the backend with the specified number of tenants, blocks, and records
			tenants = populateBackend(ctx, t, tempodbWriter, tc.tenantCount, tc.blockCount, 1)

			totalBlocksWritten += tc.tenantCount * tc.blockCount

			if _, ok := accumulant[tenant]; !ok {
				accumulant[tenant] = &expectations{}
			}

			accumulant[tenant].blocks += tc.expectedBlocks
			accumulant[tenant].outstanding += tc.expectedOutstandingBlocks

			// Check the metrics for each tenant
			for _, tenantID := range tenants {
				tenantMatcher := e2e.WithLabelMatchers(&labels.Matcher{Type: labels.MatchEqual, Name: "tenant", Value: tenantID})

				expectedBlocks := accumulant[tenant].blocks
				expectedOutstanding := accumulant[tenant].outstanding
				totalOutstanding += expectedOutstanding

				t.Logf("Waiting for %d blocks in the blocklist for tenant: %s", expectedBlocks, tenantID)

				require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Equals(float64(expectedBlocks)), []string{"tempodb_blocklist_length"},
					e2e.WaitMissingMetrics,
					tenantMatcher,
					printMetricValue(t, fmt.Sprintf("%d", expectedBlocks), "tempodb_blocklist_length"),
				))
			}
		})
	}

	time.Sleep(8 * time.Second) // Wait for polling and measurement to catch up

	// Capture the total blocks which have been written
	totalBlocksPreCompactions, err := scheduler.SumMetrics([]string{"tempodb_blocklist_length"})
	require.NoError(t, err)
	require.Len(t, totalBlocksPreCompactions, 1, "expected only one blocklist length metric")
	require.Equal(t, float64(totalBlocksWritten), totalBlocksPreCompactions[0], "expected total blocks to match the sum of all tenants")

	// NOTE: the compaction provider has a channel capacity of 1, and the
	// backendscheduler has a channel capacity of 1.  This means that the
	// compaction provider can create 4 jobs before they are processed in the
	// Next call of the scheduler.
	// - one recorded before pushing into the compaction provider channel
	// - one in the compaction provider channel
	// - one between the provider pull and the scheduler push
	// - one in the merged channel

	// NOTE: Since the measurement of outstanding blocks also skips the blocks
	// from jobs which have not been processed, we expect our outstanding blocks
	// to be N-M, where N is the actual outstanding blocks and M is the number of
	// blocks attached to jobs which have not been recorded by the scheduler. The
	// order of tenants is non-defined when all tenants have the same block
	// count, so we don't know which tenant will be first. Measure total
	// outstanding blocks instead of per tenant for this reason.

	// NOTE: Due to timing and window selection, we may not fully populate a job,
	// and we may end up with jobs which between 2 and 4 blocks outstanding.
	// - 4 jobs not yet recorded by the scheduler (no worker running yet)
	// - between 2 and 4 blocks per job (default settings)
	expectedTotalOutstandingMin := totalOutstanding - 16
	expectedTotalOutstandingMax := totalOutstanding - 8

	outstanding, err := scheduler.SumMetrics([]string{"tempodb_compaction_outstanding_blocks"})
	require.NoError(t, err)
	require.Len(t, outstanding, 1, "expected only one outstanding block metric")
	t.Logf("Total outstanding blocks: %f", outstanding[0])
	t.Logf("Expected outstanding blocks range: [%d, %d]", expectedTotalOutstandingMin, expectedTotalOutstandingMax)
	t.Logf("Total outstanding blocks: %d", totalOutstanding)
	require.LessOrEqual(t, outstanding[0], float64(expectedTotalOutstandingMax), "unexpected maximum outstanding blocks count")
	require.GreaterOrEqual(t, outstanding[0], float64(expectedTotalOutstandingMin), "unexpected minimum outstanding blocks count")

	// Delay starting the work to ensure we have a clean state of the data before
	// the worker starts processing jobs.
	worker := util.NewTempoTarget("backend-worker", "config.yaml")
	require.NoError(t, s.StartAndWaitReady(worker))

	// Allow the worker some time to process the blocks.
	// Wait until the last tenant has processed its blocks

	for _, tenantID := range tenants {
		t.Run(fmt.Sprintf("work-finished-check-wait-%s", tenantID), func(t *testing.T) {
			tenantMatcher := e2e.WithLabelMatchers(&labels.Matcher{Type: labels.MatchEqual, Name: "tenant", Value: tenantID})

			// We should have at least 1 block for each tenant
			require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1.0), []string{"tempodb_blocklist_length"},
				e2e.WaitMissingMetrics,
				tenantMatcher,
				printMetricValue(t, fmt.Sprintf("%f+", 1.0), "tempodb_blocklist_length"),
			))
		})
	}

	time.Sleep(5 * time.Second)

	// Stop the worker to ensure it is not running while we check the metrics.
	require.NoError(t, s.Stop(worker))

	// Require that all jobs which have been processed are not failed.
	_, err = scheduler.SumMetrics([]string{"tempo_backend_scheduler_jobs_failed_total"})
	require.Error(t, err, "metric not found")

	totalBlocksPostCompactions, err := scheduler.SumMetrics([]string{"tempodb_blocklist_length"})
	require.NoError(t, err)
	require.Len(t, totalBlocksPostCompactions, 1, "expected only one blocklist length metric")
	require.Less(t, totalBlocksPostCompactions[0], totalBlocksPreCompactions[0], "expected total blocks to be less after compaction")

	// Some variance in the number of expected due to the notes above.  We should
	// expect that a fully compacted tenant has between 1 and 4 blocks, and
	// between 0 and 1 outstanding blocks.  The sleep above should be enough to
	// allow the worker to finish processing all the outstanding blocks.
	expectedMin := &expectations{
		blocks:      1,
		outstanding: 0,
	}

	expectedMax := &expectations{
		blocks:      4,
		outstanding: 1,
	}

	for _, tenantID := range tenants {
		t.Run(fmt.Sprintf("prestop-check-%s", tenantID), func(t *testing.T) {
			tenantMatcher := e2e.WithLabelMatchers(&labels.Matcher{Type: labels.MatchEqual, Name: "tenant", Value: tenantID})

			t.Logf("Waiting for blocklist metric: %s: %d-%d", tenantID, expectedMin.blocks, expectedMax.blocks)

			require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Between(float64(expectedMin.blocks), float64(expectedMax.blocks)), []string{"tempodb_blocklist_length"},
				e2e.WaitMissingMetrics,
				tenantMatcher,
				printMetricValue(t, fmt.Sprintf("%d-%d", expectedMin.blocks, expectedMax.blocks), "tempodb_blocklist_length"),
			))

			t.Logf("Waiting for outstanding blocks metric: %s: %d-%d", tenantID, expectedMin.outstanding, expectedMax.outstanding)

			require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Between(float64(expectedMin.outstanding), float64(expectedMax.outstanding)), []string{"tempodb_compaction_outstanding_blocks"},
				e2e.WaitMissingMetrics,
				tenantMatcher,
				printMetricValue(t, fmt.Sprintf("%d-%d", expectedMin.outstanding, expectedMax.outstanding), "tempodb_compaction_outstanding_blocks"),
			))
		})
	}

	// Stop and restart the scheduler to ensure we replay the state correctly.
	// It should match the outstanding blocks above since we have not modified
	// the backend.
	require.NoError(t, s.Stop(scheduler))
	require.NoError(t, s.StartAndWaitReady(scheduler, worker))

	time.Sleep(2 * time.Second)

	for _, tenantID := range tenants {
		t.Run(fmt.Sprintf("final-check-%s", tenantID), func(t *testing.T) {
			tenantMatcher := e2e.WithLabelMatchers(&labels.Matcher{Type: labels.MatchEqual, Name: "tenant", Value: tenantID})

			t.Logf("Waiting for blocklist metric: %s: %d-%d", tenantID, expectedMin.blocks, expectedMax.blocks)

			require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Between(float64(expectedMin.blocks), float64(expectedMax.blocks)), []string{"tempodb_blocklist_length"},
				e2e.WaitMissingMetrics,
				tenantMatcher,
				printMetricValue(t, fmt.Sprintf("%d-%d", expectedMin.blocks, expectedMax.blocks), "tempodb_blocklist_length"),
			))

			t.Logf("Waiting for outstanding blocks metric: %s: %d-%d", tenantID, expectedMin.outstanding, expectedMax.outstanding)

			require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Between(float64(expectedMin.outstanding), float64(expectedMax.outstanding)), []string{"tempodb_compaction_outstanding_blocks"},
				e2e.WaitMissingMetrics,
				tenantMatcher,
				printMetricValue(t, fmt.Sprintf("%d-%d", expectedMin.outstanding, expectedMax.outstanding), "tempodb_compaction_outstanding_blocks"),
			))
		})
	}
}

func printValues(t *testing.T, expected string, metric string) e2e.GetMetricValueFunc {
	return func(m *io_prometheus_client.Metric) float64 {
		v := e2e.DefaultMetricsOptions.GetValue(m)
		t.Logf("metric %q: label %q: current %f, expected %s", metric, m.GetLabel()[0].GetValue(), v, expected)
		return v
	}
}

func printMetricValue(t *testing.T, expectedValue string, metric string) e2e.MetricsOption {
	return func(opts *e2e.MetricsOptions) {
		opts.GetValue = printValues(t, expectedValue, metric)
	}
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
		Filepath: t.TempDir(),
	}
	cfg.S3.Endpoint = endpoint

	_, w, _, err := tempodb.New(cfg, nil, log.NewNopLogger())
	require.NoError(t, err)

	return w
}

func populateBackend(ctx context.Context, t testing.TB, w tempodb.Writer, tenantCount, blockCount, recordCount int) []string {
	wal := w.WAL()

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	tenants := make([]string, tenantCount)

	for i := range tenantCount {
		tenantID := tenant + strconv.Itoa(i)

		tenants[i] = tenantID

		for range blockCount {
			blockID := backend.NewUUID()
			meta := &backend.BlockMeta{BlockID: blockID, TenantID: tenantID, DataEncoding: model.CurrentEncoding}
			head, err := wal.NewBlock(meta, model.CurrentEncoding)
			require.NoError(t, err)

			for range recordCount {
				id := test.ValidTraceID(nil)
				req := test.MakeTrace(10, id)

				writeTraceToWal(t, head, dec, id, req, 0, 0)
			}

			_, err = w.CompleteBlock(ctx, head)
			require.NoError(t, err)
		}
	}

	return tenants
}

func writeTraceToWal(t require.TestingT, b common.WALBlock, dec model.SegmentDecoder, id common.ID, tr *tempopb.Trace, start, end uint32) {
	b1, err := dec.PrepareForWrite(tr, 0, 0)
	require.NoError(t, err)

	b2, err := dec.ToObject([][]byte{b1})
	require.NoError(t, err)

	err = b.Append(id, b2, start, end, true)
	require.NoError(t, err, "unexpected error writing req")
}
