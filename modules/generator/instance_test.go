package generator

import (
	"context"
	"flag"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	prometheus_storage "github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func Test_instance_concurrency(t *testing.T) {
	// Both instances use the same overrides, this map will be accessed by both
	overrides := &mockOverrides{}
	overrides.processors = map[string]struct{}{
		spanmetrics.Name:   {},
		servicegraphs.Name: {},
	}

	instance1, err := newInstance(&Config{}, "test", overrides, &noopStorage{}, prometheus.DefaultRegisterer, log.NewNopLogger(), nil)
	assert.NoError(t, err)

	instance2, err := newInstance(&Config{}, "test", overrides, &noopStorage{}, prometheus.DefaultRegisterer, log.NewNopLogger(), nil)
	assert.NoError(t, err)

	end := make(chan struct{})

	accessor := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	go accessor(func() {
		req := test.MakeBatch(1, nil)
		instance1.pushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{req}})
	})

	go accessor(func() {
		req := test.MakeBatch(1, nil)
		instance2.pushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{req}})
	})

	go accessor(func() {
		err := instance1.updateProcessors()
		assert.NoError(t, err)
	})

	go accessor(func() {
		err := instance2.updateProcessors()
		assert.NoError(t, err)
	})

	time.Sleep(100 * time.Millisecond)

	instance1.shutdown()
	instance2.shutdown()

	time.Sleep(10 * time.Millisecond)
	close(end)
}

func Test_instance_updateProcessors(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	overrides := mockOverrides{}

	instance, err := newInstance(&cfg, "test", &overrides, &noopStorage{}, prometheus.DefaultRegisterer, logger, nil)
	assert.NoError(t, err)

	// stop the update goroutine
	close(instance.shutdownCh)

	// no processors should be present initially
	assert.Len(t, instance.processors, 0)

	t.Run("add servicegraphs processors", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			servicegraphs.Name: {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		assert.Len(t, instance.processors, 1)
		assert.Equal(t, instance.processors[servicegraphs.Name].Name(), servicegraphs.Name)
	})

	t.Run("add unknown processor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			"span-metricsss": {}, // typo in the overrides
		}
		err := instance.updateProcessors()
		assert.Error(t, err)

		// existing processors should not be removed when adding a new processor fails
		assert.Len(t, instance.processors, 1)
		assert.Equal(t, instance.processors[servicegraphs.Name].Name(), servicegraphs.Name)
	})

	t.Run("add spanmetrics processor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			servicegraphs.Name: {},
			spanmetrics.Name:   {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		assert.Len(t, instance.processors, 2)
		assert.Equal(t, instance.processors[servicegraphs.Name].Name(), servicegraphs.Name)
		assert.Equal(t, instance.processors[spanmetrics.Name].Name(), spanmetrics.Name)
	})

	t.Run("replace spanmetrics processor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			servicegraphs.Name: {},
			spanmetrics.Name:   {},
		}
		overrides.spanMetricsDimensions = []string{"namespace"}
		overrides.spanMetricsIntrinsicDimensions = map[string]bool{"status_message": true}

		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig spanmetrics.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.Dimensions = []string{"namespace"}
		expectedConfig.IntrinsicDimensions.StatusMessage = true

		assert.Equal(t, expectedConfig, instance.processors[spanmetrics.Name].(*spanmetrics.Processor).Cfg)
	})

	t.Run("remove processor", func(t *testing.T) {
		overrides.processors = nil
		err := instance.updateProcessors()
		assert.NoError(t, err)

		assert.Len(t, instance.processors, 0)
	})

	t.Run("add span-latency subprocessor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			servicegraphs.Name:           {},
			spanmetrics.Latency.String(): {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig spanmetrics.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.Dimensions = []string{"namespace"}
		expectedConfig.IntrinsicDimensions.StatusMessage = true
		expectedConfig.HistogramBuckets = prometheus.ExponentialBuckets(0.002, 2, 14)
		expectedConfig.Subprocessors[spanmetrics.Latency] = true
		expectedConfig.Subprocessors[spanmetrics.Count] = false
		expectedConfig.Subprocessors[spanmetrics.Size] = false

		assert.Equal(t, expectedConfig, instance.processors[spanmetrics.Name].(*spanmetrics.Processor).Cfg)

		expectedProcessors := []string{servicegraphs.Name, spanmetrics.Name}
		actualProcessors := make([]string, 0, len(instance.processors))

		for name := range instance.processors {
			actualProcessors = append(actualProcessors, name)
		}

		sort.Strings(actualProcessors)

		assert.Equal(t, expectedProcessors, actualProcessors)
	})

	t.Run("replace span-latency subprocessor with span-count", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			servicegraphs.Name:         {},
			spanmetrics.Count.String(): {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig spanmetrics.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.Dimensions = []string{"namespace"}
		expectedConfig.IntrinsicDimensions.StatusMessage = true
		expectedConfig.HistogramBuckets = nil
		expectedConfig.Subprocessors[spanmetrics.Latency] = false
		expectedConfig.Subprocessors[spanmetrics.Count] = true
		expectedConfig.Subprocessors[spanmetrics.Size] = false

		assert.Equal(t, expectedConfig, instance.processors[spanmetrics.Name].(*spanmetrics.Processor).Cfg)

		expectedProcessors := []string{servicegraphs.Name, spanmetrics.Name}
		actualProcessors := make([]string, 0, len(instance.processors))

		for name := range instance.processors {
			actualProcessors = append(actualProcessors, name)
		}

		sort.Strings(actualProcessors)

		assert.Equal(t, expectedProcessors, actualProcessors)
	})

	t.Run("use all three subprocessors at once", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			servicegraphs.Name:           {},
			spanmetrics.Count.String():   {},
			spanmetrics.Latency.String(): {},
			spanmetrics.Size.String():    {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig spanmetrics.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.Dimensions = []string{"namespace"}
		expectedConfig.IntrinsicDimensions.StatusMessage = true
		expectedConfig.HistogramBuckets = prometheus.ExponentialBuckets(0.002, 2, 14)
		expectedConfig.Subprocessors[spanmetrics.Latency] = true
		expectedConfig.Subprocessors[spanmetrics.Count] = true
		expectedConfig.Subprocessors[spanmetrics.Size] = true

		assert.Equal(t, expectedConfig, instance.processors[spanmetrics.Name].(*spanmetrics.Processor).Cfg)

		expectedProcessors := []string{servicegraphs.Name, spanmetrics.Name}
		actualProcessors := make([]string, 0, len(instance.processors))

		for name := range instance.processors {
			actualProcessors = append(actualProcessors, name)
		}

		sort.Strings(actualProcessors)

		assert.Equal(t, expectedProcessors, actualProcessors)
	})

	t.Run("replace subprocessors with span-metrics processor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			servicegraphs.Name: {},
			spanmetrics.Name:   {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig spanmetrics.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.Dimensions = []string{"namespace"}
		expectedConfig.IntrinsicDimensions.StatusMessage = true
		expectedConfig.HistogramBuckets = prometheus.ExponentialBuckets(0.002, 2, 14)
		expectedConfig.Subprocessors[spanmetrics.Latency] = true
		expectedConfig.Subprocessors[spanmetrics.Count] = true

		assert.Equal(t, expectedConfig, instance.processors[spanmetrics.Name].(*spanmetrics.Processor).Cfg)

		expectedProcessors := []string{servicegraphs.Name, spanmetrics.Name}
		actualProcessors := make([]string, 0, len(instance.processors))

		for name := range instance.processors {
			actualProcessors = append(actualProcessors, name)
		}

		sort.Strings(actualProcessors)

		assert.Equal(t, expectedProcessors, actualProcessors)
	})
}

type noopStorage struct{}

var _ storage.Storage = (*noopStorage)(nil)

func (m noopStorage) Appender(context.Context) prometheus_storage.Appender {
	return &noopAppender{}
}

func (m noopStorage) Close() error { return nil }

type noopAppender struct{}

var _ prometheus_storage.Appender = (*noopAppender)(nil)

func (n noopAppender) Append(prometheus_storage.SeriesRef, labels.Labels, int64, float64) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendExemplar(prometheus_storage.SeriesRef, labels.Labels, exemplar.Exemplar) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendHistogram(prometheus_storage.SeriesRef, labels.Labels, int64, *histogram.Histogram, *histogram.FloatHistogram) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) Commit() error { return nil }

func (n noopAppender) Rollback() error { return nil }

func (n noopAppender) UpdateMetadata(prometheus_storage.SeriesRef, labels.Labels, metadata.Metadata) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}
