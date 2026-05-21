package storage

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/grafana/e2e"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpcstatus "google.golang.org/grpc/status"
)

func TestBackendScheduler(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components:    util.ComponentsBackendWork,
		Backends:      util.BackendObjectStorageAll,
		ConfigOverlay: "config-backend-scheduler.yaml",
	}, func(h *util.TempoHarness) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s := h.TestScenario

		// Get the config from the harness
		cfg, err := h.GetConfig()
		require.NoError(t, err)

		objStorage := h.Services[util.ServiceObjectStorage]
		backendEndpoint := objStorage.HTTPEndpoint()
		t.Logf("Endpoint: %s", backendEndpoint)

		// Get the scheduler and worker services from the harness
		scheduler := h.Services[util.ServiceBackendScheduler]
		worker := h.Services[util.ServiceBackendWorker]

		// Stop the worker initially - we'll start it later after populating data
		require.NoError(t, s.Stop(worker))

		// Setup tempodb with local backend
		tempodbWriter := setupBackendWithEndpoint(t, &cfg.StorageConfig.Trace, backendEndpoint)

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

				// Check the metrics for each tenant
				for _, tenantID := range tenants {
					tenantMatcher := e2e.WithLabelMatchers(&labels.Matcher{Type: labels.MatchEqual, Name: "tenant", Value: tenantID})

					if _, ok := accumulant[tenantID]; !ok {
						accumulant[tenantID] = &expectations{}
					}

					accumulant[tenantID].blocks += tc.expectedBlocks
					accumulant[tenantID].outstanding += tc.expectedOutstandingBlocks

					expectedBlocks := accumulant[tenantID].blocks
					expectedOutstanding := accumulant[tenantID].outstanding
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
	})
}

// TestBackendSchedulerRedaction verifies the end-to-end redaction flow:
//   - SubmitRedaction fans out one pending job per block
//   - Duplicate submissions are rejected with AlreadyExists
//   - After the worker processes all jobs, the trace is no longer findable
func TestBackendSchedulerRedaction(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components:    util.ComponentsBackendWork,
		Backends:      util.BackendObjectStorageAll,
		ConfigOverlay: "config-backend-scheduler-redaction.yaml",
	}, func(h *util.TempoHarness) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg, err := h.GetConfig()
		require.NoError(t, err)

		objStorage := h.Services[util.ServiceObjectStorage]
		scheduler := h.Services[util.ServiceBackendScheduler]
		worker := h.Services[util.ServiceBackendWorker]

		// Stop the worker so pending redaction jobs are not drained before we assert.
		require.NoError(t, h.TestScenario.Stop(worker))

		// Write a small number of blocks each containing the same trace ID.
		tempodbReader, tempodbWriter := setupBackendReaderWriterWithEndpoint(t, &cfg.StorageConfig.Trace, objStorage.HTTPEndpoint())
		const blockCount = 3
		testTenant := tenant + "0"
		traceID := test.ValidTraceID(nil)
		populateBackendWithTrace(ctx, t, tempodbWriter, testTenant, blockCount, traceID)

		// Wait until the scheduler's blocklist reflects all written blocks.
		tenantMatcher := e2e.WithLabelMatchers(&labels.Matcher{
			Type: labels.MatchEqual, Name: "tenant", Value: testTenant,
		})
		require.NoError(t, scheduler.WaitSumMetricsWithOptions(
			e2e.Equals(float64(blockCount)),
			[]string{"tempodb_blocklist_length"},
			e2e.WaitMissingMetrics,
			tenantMatcher,
			printMetricValue(t, fmt.Sprintf("%d", blockCount), "tempodb_blocklist_length"),
		))

		// Verify the trace is findable before redaction by querying object storage directly.
		tempodbReader.EnablePolling(ctx, nil, false)
		trs, failedBlocks, err := tempodbReader.Find(ctx, testTenant, traceID, tempodb.BlockIDMin, tempodb.BlockIDMax, time.Time{}, time.Time{}, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.Empty(t, failedBlocks, "no blocks should fail lookup")
		require.NotEmpty(t, trs, "trace must be findable in all blocks before redaction")

		// Dial the scheduler's gRPC endpoint and build a client.
		conn, err := grpc.NewClient(
			scheduler.Endpoint(9095),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()
		schedulerClient := tempopb.NewBackendSchedulerClient(conn)

		// Build an authenticated context: inject the tenant into the outgoing gRPC metadata.
		// With multitenancy_enabled, the scheduler's auth interceptor reads X-Scope-OrgID and
		// sets the tenant in the handler context; the body tenant_id field is ignored.
		tenantCtx := user.InjectOrgID(ctx, testTenant)
		tenantCtx, err = user.InjectIntoGRPCRequest(tenantCtx)
		require.NoError(t, err)

		// Submit a redaction — expect one pending job per block.
		resp, err := schedulerClient.SubmitRedaction(tenantCtx, &tempopb.SubmitRedactionRequest{
			TraceIds: [][]byte{traceID},
		})
		require.NoError(t, err)
		require.Equal(t, int32(blockCount), resp.JobsCreated)

		// A second submission for the same tenant must be rejected.
		_, err = schedulerClient.SubmitRedaction(tenantCtx, &tempopb.SubmitRedactionRequest{
			TraceIds: [][]byte{traceID},
		})
		require.Error(t, err)
		st, ok := grpcstatus.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.AlreadyExists, st.Code())

		// Start the worker to process redaction jobs.
		require.NoError(t, h.TestScenario.StartAndWaitReady(worker))

		// Wait until all redaction jobs have been completed.
		jobTypeMatcher := e2e.WithLabelMatchers(&labels.Matcher{
			Type: labels.MatchEqual, Name: "job_type", Value: "JOB_TYPE_REDACTION",
		})
		require.NoError(t, scheduler.WaitSumMetricsWithOptions(
			e2e.Equals(float64(blockCount)),
			[]string{"tempo_backend_scheduler_jobs_completed_total"},
			e2e.WaitMissingMetrics,
			jobTypeMatcher,
			printMetricValue(t, fmt.Sprintf("%d", blockCount), "tempo_backend_scheduler_jobs_completed_total"),
		))

		// RedactBlock marks the original blocks as compacted and writes new blocks
		// without the trace. Find() includes recently-compacted blocks for a lookback
		// window of 2 × blocklist_poll. Poll repeatedly until the old blocks age out
		// of the lookback window and the trace is no longer findable.
		require.Eventually(t, func() bool {
			tempodbReader.PollNow(ctx)
			trs, _, err = tempodbReader.Find(ctx, testTenant, traceID, tempodb.BlockIDMin, tempodb.BlockIDMax, time.Time{}, time.Time{}, common.DefaultSearchOptions())
			return err == nil && len(trs) == 0
		}, 60*time.Second, 2*time.Second, "trace must not be findable in any block after redaction")
	})
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
	_, w := setupBackendReaderWriterWithEndpoint(t, cfg, endpoint)
	return w
}

func setupBackendReaderWriterWithEndpoint(t testing.TB, cfg *tempodb.Config, endpoint string) (tempodb.Reader, tempodb.Writer) {
	cfg.Block = &common.BlockConfig{
		BloomFP:             .01,
		BloomShardSizeBytes: 100_000,
		Version:             encoding.LatestEncoding().Version(),
		RowGroupSizeBytes:   30_000_000,
		DedicatedColumns:    backend.DedicatedColumns{{Scope: "span", Name: "key", Type: "string"}},
	}
	cfg.WAL = &wal.Config{
		Filepath: t.TempDir(),
	}
	if cfg.S3 != nil {
		cfg.S3.Endpoint = endpoint
	}
	if cfg.Azure != nil {
		cfg.Azure.Endpoint = endpoint
	}
	if cfg.GCS != nil {
		cfg.GCS.Endpoint = endpoint
	}

	r, w, _, err := tempodb.New(cfg, nil, log.NewNopLogger())
	require.NoError(t, err)

	return r, w
}

func populateBackendWithTrace(ctx context.Context, t testing.TB, w tempodb.Writer, tenantID string, blockCount int, traceID common.ID) {
	walInstance := w.WAL()
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	for range blockCount {
		blockID := backend.NewUUID()
		meta := &backend.BlockMeta{BlockID: blockID, TenantID: tenantID}
		head, err := walInstance.NewBlock(meta, model.CurrentEncoding)
		require.NoError(t, err)

		req := test.MakeTrace(10, traceID)
		writeTraceToWal(t, head, dec, traceID, req, 0, 0)

		_, err = w.CompleteBlock(ctx, head)
		require.NoError(t, err)
	}
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
			meta := &backend.BlockMeta{BlockID: blockID, TenantID: tenantID}
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
