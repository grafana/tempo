package e2e

import (
	"context"
	"testing"

	util "github.com/grafana/tempo/integration"
	"github.com/prometheus/prometheus/model/labels"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	"github.com/stretchr/testify/require"
)

const (
	configLimits = "config-limits.yaml"
)

func TestLimits(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configLimits, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Get port for the otlp receiver endpoint
	c, err := newJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	// should fail b/c the trace is too large. each batch should be ~70 bytes
	batch := makeThriftBatchWithSpanCount(2)
	require.Error(t, c.EmitBatch(context.Background(), batch))

	// should fail b/c this will be too many traces
	batch = makeThriftBatch()
	require.Error(t, c.EmitBatch(context.Background(), batch))

	// should fail b/c due to ingestion rate limit
	batch = makeThriftBatchWithSpanCount(10)
	require.Error(t, c.EmitBatch(context.Background(), batch))

	// test limit metrics
	err = tempo.WaitSumMetricsWithOptions(cortex_e2e.Equals(2),
		[]string{"tempo_discarded_spans_total"},
		cortex_e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "trace_too_large")),
	)
	require.NoError(t, err)
	err = tempo.WaitSumMetricsWithOptions(cortex_e2e.Equals(1),
		[]string{"tempo_discarded_spans_total"},
		cortex_e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "live_traces_exceeded")),
	)
	require.NoError(t, err)
	err = tempo.WaitSumMetricsWithOptions(cortex_e2e.Equals(10),
		[]string{"tempo_discarded_spans_total"},
		cortex_e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "reason", "rate_limited")),
	)
	require.NoError(t, err)
}
