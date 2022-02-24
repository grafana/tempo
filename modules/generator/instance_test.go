package generator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/remotewrite"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func Test_instance_concurrency(t *testing.T) {
	instance, err := newInstance(&Config{}, "test", &mockOverrides{}, &remotewrite.NoopAppender{})
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
		err := instance.pushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{req}})
		assert.NoError(t, err)
	})

	go accessor(func() {
		err := instance.collectAndPushMetrics(context.Background())
		assert.NoError(t, err)
	})

	go accessor(func() {
		processors := map[string]struct{}{
			"span-metrics": {},
		}
		err := instance.updateProcessors(processors)
		assert.NoError(t, err)
	})

	go accessor(func() {
		processors := map[string]struct{}{
			"service-graphs": {},
		}
		err := instance.updateProcessors(processors)
		assert.NoError(t, err)
	})

	time.Sleep(100 * time.Millisecond)

	err = instance.shutdown(context.Background())
	assert.NoError(t, err)

	time.Sleep(10 * time.Millisecond)
	close(end)

}

func Test_instance_updateProcessors(t *testing.T) {
	instance, err := newInstance(&Config{}, "test", &mockOverrides{}, &remotewrite.NoopAppender{})
	assert.NoError(t, err)

	// shutdown the instance to stop the update goroutine
	err = instance.shutdown(context.Background())
	assert.NoError(t, err)

	// no processors should be present initially
	assert.Len(t, instance.processors, 0)

	t.Run("add new processor", func(t *testing.T) {
		processors := map[string]struct{}{
			servicegraphs.Name: {},
		}
		err := instance.updateProcessors(processors)
		assert.NoError(t, err)

		assert.Len(t, instance.processors, 1)
		assert.Equal(t, instance.processors[servicegraphs.Name].Name(), servicegraphs.Name)
	})

	t.Run("add unknown processor", func(t *testing.T) {
		processors := map[string]struct{}{
			"span-metricsss": {}, // typo in the overrides
		}
		err := instance.updateProcessors(processors)
		assert.Error(t, err)

		// existing processors should not be removed when adding a new processor fails
		assert.Len(t, instance.processors, 1)
		assert.Equal(t, instance.processors[servicegraphs.Name].Name(), servicegraphs.Name)
	})

	t.Run("remove processor", func(t *testing.T) {
		err := instance.updateProcessors(nil)
		assert.NoError(t, err)

		assert.Len(t, instance.processors, 0)
	})
}

type mockOverrides struct {
	processors map[string]struct{}
}

var _ metricsGeneratorOverrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorProcessors(userID string) map[string]struct{} {
	return m.processors
}
