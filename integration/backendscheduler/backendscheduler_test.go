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
	tenant     = "test"
	configFile = "config.yaml"
)

func TestBackendScheduler(t *testing.T) {
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

	scheduler := util.NewTempoTarget("backend-scheduler", configFile)
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Populate the backend with the specified number of tenants, blocks, and records
			tenants = populateBackend(ctx, t, tempodbWriter, tc.tenantCount, tc.blockCount, 1)

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

				t.Logf("Waiting for %d blocks in the blocklist for tenant: %s", expectedBlocks, tenantID)

				require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Equals(float64(expectedBlocks)), []string{"tempodb_blocklist_length"},
					e2e.WaitMissingMetrics,
					tenantMatcher,
					printMetricValue(t),
				))

				t.Logf("Waiting for %d outstanding blocks for tenant: %s", expectedOutstanding, tenantID)

				require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Equals(float64(expectedOutstanding)), []string{"tempodb_compaction_outstanding_blocks"},
					e2e.WaitMissingMetrics,
					tenantMatcher,
					printMetricValue(t),
				))
			}
		})
	}

	// Delay starting the work to ensure we have a clean state of the data before
	// the worker starts processing jobs.
	worker := util.NewTempoTarget("backend-worker", configFile)
	require.NoError(t, s.StartAndWaitReady(worker))

	// Allow the worker some time to process the blocks.
	// Wait until the last tenant has processed its blocks

	for _, tenantID := range tenants {
		t.Run(fmt.Sprintf("work-finished-check-wait-%s", tenantID), func(t *testing.T) {
			tenantMatcher := e2e.WithLabelMatchers(&labels.Matcher{Type: labels.MatchEqual, Name: "tenant", Value: tenantID})

			require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Equals(float64(2)), []string{"tempodb_blocklist_length"},
				e2e.WaitMissingMetrics,
				tenantMatcher,
				printMetricValue(t),
			))
		})
	}

	time.Sleep(2 * time.Second)

	// Stop the worker to ensure it is not running while we check the metrics.
	require.NoError(t, s.Stop(worker))

	// each tenant has the same expectations
	expected := &expectations{
		blocks:      2,
		outstanding: 0, // two blocks remain, but their group is not the same, and so they will not be compacted together.
	}

	for _, tenantID := range tenants {
		t.Run(fmt.Sprintf("prestop-check-%s", tenantID), func(t *testing.T) {
			tenantMatcher := e2e.WithLabelMatchers(&labels.Matcher{Type: labels.MatchEqual, Name: "tenant", Value: tenantID})

			t.Logf("Waiting for blocklist metric: %s: %d", tenantID, expected.blocks)

			require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Equals(float64(expected.blocks)), []string{"tempodb_blocklist_length"},
				e2e.WaitMissingMetrics,
				tenantMatcher,
				printMetricValue(t),
			))

			t.Logf("Waiting for outstanding blocks metric: %s: %d", tenantID, expected.outstanding)

			require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Equals(float64(expected.outstanding)), []string{"tempodb_compaction_outstanding_blocks"},
				e2e.WaitMissingMetrics,
				tenantMatcher,
				printMetricValue(t),
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

			t.Logf("Waiting for blocklist metric: %s: %d", tenantID, expected.blocks)

			require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Equals(float64(expected.blocks)), []string{"tempodb_blocklist_length"},
				e2e.WaitMissingMetrics,
				tenantMatcher,
				printMetricValue(t),
			))

			t.Logf("Waiting for outstanding blocks metric: %s: %d", tenantID, expected.outstanding)

			require.NoError(t, scheduler.WaitSumMetricsWithOptions(e2e.Equals(float64(expected.outstanding)), []string{"tempodb_compaction_outstanding_blocks"},
				e2e.WaitMissingMetrics,
				tenantMatcher,
				printMetricValue(t),
			))
		})
	}

	// Check some additional metrics to ensure the scheduler is working properly.
}

func printValues(t *testing.T) e2e.GetMetricValueFunc {
	return func(m *io_prometheus_client.Metric) float64 {
		v := e2e.DefaultMetricsOptions.GetValue(m)
		t.Logf("metric %q: %f", m.GetLabel()[0].GetValue(), v)
		return v
	}
}

func printMetricValue(t *testing.T) e2e.MetricsOption {
	return func(opts *e2e.MetricsOptions) {
		opts.GetValue = printValues(t)
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
