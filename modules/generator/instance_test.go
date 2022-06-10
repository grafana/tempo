package generator

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	prometheus_storage "github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func Test_instance_concurrency(t *testing.T) {
	overrides := &mockOverrides{}
	instance, err := newInstance(&Config{}, "test", overrides, &noopStorage{}, prometheus.DefaultRegisterer, log.NewNopLogger())
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
		instance.pushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{req}})
	})

	go accessor(func() {
		overrides.processors = map[string]struct{}{
			"span-metrics": {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		overrides.processors = map[string]struct{}{
			"service-graphs": {},
		}
		err = instance.updateProcessors()
		assert.NoError(t, err)
	})

	time.Sleep(100 * time.Millisecond)

	instance.shutdown()

	time.Sleep(10 * time.Millisecond)
	close(end)

}

func Test_instance_updateProcessors(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	overrides := mockOverrides{}

	instance, err := newInstance(&cfg, "test", &overrides, &noopStorage{}, prometheus.DefaultRegisterer, logger)
	assert.NoError(t, err)

	// stop the update goroutine
	close(instance.shutdownCh)

	// no processors should be present initially
	assert.Len(t, instance.processors, 0)

	t.Run("add new processor", func(t *testing.T) {
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

	t.Run("replace processor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			servicegraphs.Name: {},
		}
		overrides.serviceGraphsDimensions = []string{"namespace"}

		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig servicegraphs.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.Dimensions = []string{"namespace"}

		assert.Equal(t, expectedConfig, instance.processors[servicegraphs.Name].(*servicegraphs.Processor).Cfg)
	})

	t.Run("remove processor", func(t *testing.T) {
		overrides.processors = nil
		err := instance.updateProcessors()
		assert.NoError(t, err)

		assert.Len(t, instance.processors, 0)
	})
}

type noopStorage struct{}

var _ storage.Storage = (*noopStorage)(nil)

func (m noopStorage) Appender(ctx context.Context) prometheus_storage.Appender {
	return &noopAppender{}
}

func (m noopStorage) Close() error {
	return nil
}

type noopAppender struct{}

var _ prometheus_storage.Appender = (*noopAppender)(nil)

func (n noopAppender) Append(ref prometheus_storage.SeriesRef, l labels.Labels, t int64, v float64) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendExemplar(ref prometheus_storage.SeriesRef, l labels.Labels, e exemplar.Exemplar) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) Commit() error {
	return nil
}

func (n noopAppender) Rollback() error {
	return nil
}
