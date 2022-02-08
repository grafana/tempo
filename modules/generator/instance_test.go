package generator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
	close(end)

	err = instance.shutdown(context.Background())
	assert.NoError(t, err)
}

type mockOverrides struct {
	processors map[string]struct{}
}

var _ metricsGeneratorOverrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorProcessors(userID string) map[string]struct{} {
	return m.processors
}
