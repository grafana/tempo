package generator

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	user1 = "user1"
	user2 = "user2"
)

func TestGeneratorSpanMetrics_subprocessorConcurrency(t *testing.T) {
	overridesFile := filepath.Join(t.TempDir(), "Overrides.yaml")
	overridesConfig := overrides.Config{
		Defaults: overrides.Overrides{
			MetricsGenerator: overrides.MetricsGeneratorOverrides{
				Processors: map[string]struct{}{
					spanmetrics.Name: {},
				},
				CollectionInterval: 2 * time.Second,
			},
		},
		PerTenantOverrideConfig: overridesFile,
		PerTenantOverridePeriod: model.Duration(time.Second),
	}

	require.NoError(t, os.WriteFile(overridesFile, []byte(fmt.Sprintf(`
overrides:
  %s:
    metrics_generator:
      collection_interval: 1s
      processors:
        - %s
`, user1, spanmetrics.Name)), 0o700))

	o, err := overrides.NewOverrides(overridesConfig, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), o))

	generatorConfig := &Config{}
	generatorConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	generatorConfig.Storage.Path = t.TempDir()
	generatorConfig.Ring.KVStore.Store = "inmemory"
	ifaces, err := net.Interfaces()
	require.NoError(t, err)
	netWorkInteraces := make([]string, len(ifaces))
	for i, iface := range ifaces {
		netWorkInteraces[i] = iface.Name
	}
	generatorConfig.Ring.InstanceInterfaceNames = netWorkInteraces
	g, err := New(generatorConfig, o, prometheus.NewRegistry(), nil, nil, newTestLogger(t))
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), g))

	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), o))
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), g))
	})

	allSubprocessors := map[spanmetrics.Subprocessor]bool{spanmetrics.Count: true, spanmetrics.Latency: true, spanmetrics.Size: true}

	// All subprocessors should be enabled for user1
	instance1, err := g.getOrCreateInstance(user1)
	require.NoError(t, err)
	verifySubprocessors(t, instance1, allSubprocessors)

	// All subprocessors should be enabled for user2
	instance2, err := g.getOrCreateInstance(user2)
	require.NoError(t, err)
	verifySubprocessors(t, instance2, allSubprocessors)

	// Change overrides for user1
	require.NoError(t, os.WriteFile(overridesFile, []byte(fmt.Sprintf(`
overrides:
  %s:
    metrics_generator:
      collection_interval: 1s
      processors:
        - %s
`, user1, spanmetrics.Count.String())), 0o700))
	time.Sleep(15 * time.Second) // Wait for overrides to be applied. Reload is hardcoded to 10s :(

	// Only Count should be enabled for user1
	instance1, err = g.getOrCreateInstance(user1)
	require.NoError(t, err)
	verifySubprocessors(t, instance1, map[spanmetrics.Subprocessor]bool{spanmetrics.Count: true, spanmetrics.Latency: false, spanmetrics.Size: false})

	// All subprocessors should be enabled for user2
	instance2, err = g.getOrCreateInstance(user2)
	require.NoError(t, err)
	verifySubprocessors(t, instance2, allSubprocessors)
}

func verifySubprocessors(t *testing.T, instance *instance, expected map[spanmetrics.Subprocessor]bool) {
	instance.processorsMtx.RLock()
	defer instance.processorsMtx.RUnlock()

	require.Len(t, instance.processors, 1)

	processor, ok := instance.processors[spanmetrics.Name]
	require.True(t, ok)

	require.Equal(t, len(processor.(*spanmetrics.Processor).Cfg.Subprocessors), len(expected))

	cfg := processor.(*spanmetrics.Processor).Cfg
	for sub, enabled := range expected {
		assert.Equal(t, enabled, cfg.Subprocessors[sub])
	}
}

var _ log.Logger = (*testLogger)(nil)

type testLogger struct {
	t *testing.T
}

func newTestLogger(t *testing.T) log.Logger {
	return testLogger{t: t}
}

func (l testLogger) Log(keyvals ...interface{}) error {
	l.t.Log(keyvals...)
	return nil
}

func BenchmarkPushSpans(b *testing.B) {
	var (
		tenant = "test-tenant"
		reg    = prometheus.NewRegistry()
		ctx    = context.Background()
		log    = log.NewNopLogger()
		cfg    = &Config{}

		walcfg = &storage.Config{
			Path: b.TempDir(),
		}

		o = &mockOverrides{
			processors: map[string]struct{}{
				"span-metrics":   {},
				"service-graphs": {},
			},
			spanMetricsEnableTargetInfo:             true,
			spanMetricsTargetInfoExcludedDimensions: []string{"excluded}"},
		}
	)

	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	wal, err := storage.New(walcfg, o, tenant, reg, log)
	require.NoError(b, err)

	inst, err := newInstance(cfg, tenant, o, wal, log, nil, nil, nil)
	require.NoError(b, err)
	defer inst.shutdown()

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
		},
	}

	// Add more resource attributes to get closer to real data
	// Add integer to increase cardinality.
	// Currently this is about 80 active series
	// TODO - Get more series
	for i, b := range req.Batches {
		b.Resource.Attributes = append(b.Resource.Attributes, []*common_v1.KeyValue{
			{Key: "k8s.cluster.name", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
			{Key: "k8s.namespace.name", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
			{Key: "k8s.node.name", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
			{Key: "k8s.pod.ip", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
			{Key: "k8s.pod.name", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
			{Key: "service.instance.id", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
			{Key: "excluded", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
		}...)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		inst.pushSpans(ctx, req)
	}

	b.StopTimer()
	runtime.GC()
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	b.ReportMetric(float64(mem.HeapInuse), "heap_in_use")
	b.ReportMetric(float64(mem.HeapAlloc), "heap_alloc")
}

func BenchmarkCollect(b *testing.B) {
	var (
		tenant = "test-tenant"
		reg    = prometheus.NewRegistry()
		ctx    = context.Background()
		log    = log.NewNopLogger()
		cfg    = &Config{}

		walcfg = &storage.Config{
			Path: b.TempDir(),
		}

		o = &mockOverrides{
			processors: map[string]struct{}{
				"span-metrics":   {},
				"service-graphs": {},
			},
			spanMetricsDimensions:                   []string{"k8s.cluster.name", "k8s.namespace.name"},
			spanMetricsEnableTargetInfo:             true,
			spanMetricsTargetInfoExcludedDimensions: []string{"excluded}"},
			nativeHistograms:                        overrides.HistogramMethodBoth,
		}
	)

	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	wal, err := storage.New(walcfg, o, tenant, reg, log)
	require.NoError(b, err)

	inst, err := newInstance(cfg, tenant, o, wal, log, nil, nil, nil)
	require.NoError(b, err)
	defer inst.shutdown()

	req := &tempopb.PushSpansRequest{
		Batches: []*trace_v1.ResourceSpans{
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
			test.MakeBatch(100, nil),
		},
	}

	// Add more resource attributes to get closer to real data
	// Add integer to increase cardinality.
	// Currently this is about 80 active series
	// TODO - Get more series
	for i, b := range req.Batches {
		b.Resource.Attributes = append(b.Resource.Attributes, []*common_v1.KeyValue{
			{Key: "k8s.cluster.name", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
			{Key: "k8s.namespace.name", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
			{Key: "k8s.node.name", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
			{Key: "k8s.pod.ip", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
			{Key: "k8s.pod.name", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
			{Key: "excluded", Value: &common_v1.AnyValue{Value: &common_v1.AnyValue_StringValue{StringValue: "test" + strconv.Itoa(i)}}},
		}...)
	}
	inst.pushSpans(ctx, req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inst.registry.CollectMetrics(ctx)
	}

	b.StopTimer()
	runtime.GC()
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	b.ReportMetric(float64(mem.HeapInuse), "heap_in_use")
}
