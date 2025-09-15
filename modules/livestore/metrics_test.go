package livestore

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/wal"
)

const (
	testTenant = "test-tenant"
)

// testSetup holds common test resources
type testSetup struct {
	tmpDir    string
	wal       *wal.WAL
	overrides overrides.Interface
	instance  *instance
	cleanup   func()
}

// setupTest creates a test instance with all required dependencies
func setupTest(t *testing.T) *testSetup {
	t.Helper()
	tmpDir := t.TempDir()

	// Setup WAL config
	w, err := wal.New(&wal.Config{
		Filepath: tmpDir,
		Version:  "vParquet4",
	})
	require.NoError(t, err)

	// Create overrides with a separate registry to avoid conflicts
	registry := prometheus.NewRegistry()
	o, err := overrides.NewOverrides(overrides.Config{}, nil, registry)
	require.NoError(t, err)

	// Create instance
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	instance, err := newInstance(testTenant, cfg, w, o, log.NewNopLogger())
	require.NoError(t, err)

	return &testSetup{
		tmpDir:    tmpDir,
		wal:       w,
		overrides: o,
		instance:  instance,
		cleanup:   func() {},
	}
}

func TestMetrics_InitialValues(t *testing.T) {
	setup := setupTest(t)
	defer setup.cleanup()

	// Test initial metric values - should all be zero for a new instance
	tracesCreatedValue, err := test.GetCounterValue(setup.instance.tracesCreatedTotal)
	require.NoError(t, err)
	assert.Equal(t, 0.0, tracesCreatedValue, "traces created should start at 0")

	liveTracesValue, err := test.GetGaugeVecValue(metricLiveTraces, testTenant)
	require.NoError(t, err)
	assert.Equal(t, 0.0, liveTracesValue, "live traces should start at 0")

	liveTraceBytesValue, err := test.GetGaugeVecValue(metricLiveTraceBytes, testTenant)
	require.NoError(t, err)
	assert.Equal(t, 0.0, liveTraceBytesValue, "live trace bytes should start at 0")

	bytesReceivedValue, err := getCounterVecValue(metricBytesReceivedTotal, testTenant, "trace")
	require.NoError(t, err)
	assert.Equal(t, 0.0, bytesReceivedValue, "bytes received should start at 0")
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
	err = setup.instance.cutIdleTraces(true) // immediate cut
	require.NoError(t, err)

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

	// Cut traces to head block
	err = setup.instance.cutIdleTraces(true)
	require.NoError(t, err)

	// Cut block to prepare for completion
	blockID, err := setup.instance.cutBlocks(true)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, blockID, "should generate a valid block ID")

	// Record initial completion size histogram count
	initialCompletionSize := getHistogramCount(t, metricCompletionSize)

	// Complete the block
	err = setup.instance.completeBlock(context.Background(), blockID)
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
