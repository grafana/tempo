package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/overrides"
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
`, user1, spanmetrics.Name)), os.ModePerm))

	o, err := overrides.NewOverrides(overridesConfig, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), o))

	generatorConfig := &Config{}
	generatorConfig.Storage.Path = t.TempDir()
	generatorConfig.Ring.KVStore.Store = "inmemory"
	generatorConfig.Processor.SpanMetrics.RegisterFlagsAndApplyDefaults("", nil)
	g, err := New(generatorConfig, o, prometheus.NewRegistry(), newTestLogger(t))
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
`, user1, spanmetrics.Count.String())), os.ModePerm))
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
