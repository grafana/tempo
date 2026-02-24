package livestore

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend/local"
)

const (
	testTenant = "test-tenant"
)

// testSetup holds common test resources
type testSetup struct {
	tmpDir       string
	localBackend *local.Backend
	overrides    overrides.Interface
	instance     *instance
	cleanup      func()
}

// setupTest creates a test instance with all required dependencies
func setupTest(t *testing.T) *testSetup {
	t.Helper()
	tmpDir := t.TempDir()

	// Create overrides with a separate registry to avoid conflicts
	registry := prometheus.NewRegistry()
	o, err := overrides.NewOverrides(overrides.Config{}, nil, registry)
	require.NoError(t, err)

	// Create instance
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	blockEnc, _, err := coalesceBlockVersions(cfg)
	require.NoError(t, err)

	// Setup local backend for block storage
	localBackend, err := local.NewBackend(&local.Config{
		Path: tmpDir,
	})
	require.NoError(t, err)

	instance, err := newInstance(testTenant, *cfg, localBackend, blockEnc, o, log.NewNopLogger())
	require.NoError(t, err)

	return &testSetup{
		tmpDir:       tmpDir,
		localBackend: localBackend,
		overrides:    o,
		instance:     instance,
		cleanup:      func() {},
	}
}

func TestMetrics_InitialValues(t *testing.T) {
	// Record baseline values before creating the instance.
	// Global metrics persist across test runs, so we can't assume they start at zero.
	baselineTracesCreated, _ := getCounterVecValue(metricTracesCreatedTotal, testTenant)
	baselineLiveTraces, _ := test.GetGaugeVecValue(metricLiveTraces, testTenant)
	baselineLiveTraceBytes, _ := test.GetGaugeVecValue(metricLiveTraceBytes, testTenant)
	baselineBytesReceived, _ := getCounterVecValue(metricBytesReceivedTotal, testTenant, "trace")

	setup := setupTest(t)
	defer setup.cleanup()

	// Verify creating a new instance does not change any metric values
	tracesCreatedValue, err := test.GetCounterValue(setup.instance.tracesCreatedTotal)
	require.NoError(t, err)
	assert.Equal(t, baselineTracesCreated, tracesCreatedValue, "traces created should not change after instance creation")

	liveTracesValue, err := test.GetGaugeVecValue(metricLiveTraces, testTenant)
	require.NoError(t, err)
	assert.Equal(t, baselineLiveTraces, liveTracesValue, "live traces should not change after instance creation")

	liveTraceBytesValue, err := test.GetGaugeVecValue(metricLiveTraceBytes, testTenant)
	require.NoError(t, err)
	assert.Equal(t, baselineLiveTraceBytes, liveTraceBytesValue, "live trace bytes should not change after instance creation")

	bytesReceivedValue, err := getCounterVecValue(metricBytesReceivedTotal, testTenant, "trace")
	require.NoError(t, err)
	assert.Equal(t, baselineBytesReceived, bytesReceivedValue, "bytes received should not change after instance creation")
}

func TestMetrics_PushBytesTracking(t *testing.T) {
	setup := setupTest(t)
	defer setup.cleanup()

	// Create a trace and push it
	traceID := test.ValidTraceID(nil)
	trace := test.MakeTrace(10, traceID)
	traceData, err := trace.Marshal()
	require.NoError(t, err)

	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: traceData}},
		Ids:    [][]byte{traceID},
	}

	expectedBytes := float64(len(traceData))

	// Record initial values to ensure we're measuring the delta correctly
	initialBytesReceived, err := getCounterVecValue(metricBytesReceivedTotal, testTenant, "trace")
	require.NoError(t, err)
	initialTracesCreated, err := test.GetCounterValue(setup.instance.tracesCreatedTotal)
	require.NoError(t, err)

	// Push bytes
	setup.instance.pushBytes(t.Context(), time.Now(), req)

	// Verify bytes received metric increased by expected amount
	finalBytesReceived, err := getCounterVecValue(metricBytesReceivedTotal, testTenant, "trace")
	require.NoError(t, err)
	assert.Equal(t, initialBytesReceived+expectedBytes, finalBytesReceived,
		"bytes received should increase by trace data size")

	// Check live traces metric after cutting
	setup.instance.cutIdleTraces(true) // immediate cut

	// Verify traces were created
	finalTracesCreated, err := test.GetCounterValue(setup.instance.tracesCreatedTotal)
	require.NoError(t, err)
	assert.Equal(t, initialTracesCreated+1.0, finalTracesCreated,
		"one trace should be created after cutting")
}

func TestMetrics_CompletionFlow(t *testing.T) {
	setup := setupTest(t)
	defer setup.cleanup()

	// Create a trace and push it
	traceID := test.ValidTraceID(nil)
	trace := test.MakeTrace(10, traceID)
	traceData, err := trace.Marshal()
	require.NoError(t, err)

	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: traceData}},
		Ids:    [][]byte{traceID},
	}
	setup.instance.pushBytes(t.Context(), time.Now(), req)

	// Cut traces to pending
	setup.instance.cutIdleTraces(true)

	// Record initial completion size histogram count
	initialCompletionSize := getHistogramCount(t, metricCompletionSize)

	// Create complete block from pending
	err = setup.instance.createBlockFromPending(context.Background())
	require.NoError(t, err)

	// Verify completion size metric was updated
	finalCompletionSize := getHistogramCount(t, metricCompletionSize)
	assert.Equal(t, initialCompletionSize+1, finalCompletionSize,
		"completion size histogram should record one additional sample")
}

func TestMetrics_EmptyPushBytesRequest(t *testing.T) {
	setup := setupTest(t)
	defer setup.cleanup()

	// Record initial values
	initialBytesReceived, err := getCounterVecValue(metricBytesReceivedTotal, testTenant, "trace")
	require.NoError(t, err)

	// Push empty request
	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{},
		Ids:    [][]byte{},
	}
	setup.instance.pushBytes(t.Context(), time.Now(), req)

	// Verify no bytes were recorded
	finalBytesReceived, err := getCounterVecValue(metricBytesReceivedTotal, testTenant, "trace")
	require.NoError(t, err)
	assert.Equal(t, initialBytesReceived, finalBytesReceived,
		"empty request should not increment bytes received")
}

func getCounterVecValue(metric *prometheus.CounterVec, labels ...string) (float64, error) {
	m := &dto.Metric{}
	err := metric.WithLabelValues(labels...).Write(m)
	if err != nil {
		return 0, err
	}
	return m.Counter.GetValue(), nil
}

func getHistogramCount(t *testing.T, histogram prometheus.Histogram) float64 {
	m := &dto.Metric{}
	err := histogram.Write(m)
	require.NoError(t, err)
	return float64(m.Histogram.GetSampleCount())
}
