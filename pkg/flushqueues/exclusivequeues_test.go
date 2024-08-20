package flushqueues

import (
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/v2/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

type mockOp struct {
	key string
}

func (m mockOp) Key() string {
	return m.key
}

func (m mockOp) Priority() int64 {
	return 0
}

func TestExclusiveQueues(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "test",
		Name:      "testersons",
	})

	q := New(1, gauge)
	op := mockOp{
		key: "not unique",
	}

	// enqueue twice
	err := q.Enqueue(op)
	assert.NoError(t, err)

	length, err := test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(length))

	err = q.Enqueue(op)
	assert.NoError(t, err)

	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(length))

	// dequeue -> requeue
	_ = q.Dequeue(0)
	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 0, int(length))

	err = q.Requeue(op)
	assert.NoError(t, err)

	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(length))

	// dequeue -> clearkey -> enqueue
	_ = q.Dequeue(0)
	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 0, int(length))

	q.Clear(op)
	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 0, int(length))

	err = q.Enqueue(op)
	assert.NoError(t, err)

	length, err = test.GetGaugeValue(gauge)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(length))
}

func TestMultipleQueues(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "test",
		Name:      "testersons",
	})

	totalQueues := 10
	totalItems := 10
	q := New(totalQueues, gauge)

	// add stuff to the queue and confirm the length matches expected
	for i := 0; i < totalItems; i++ {
		op := mockOp{
			key: uuid.New().String(),
		}

		err := q.Enqueue(op)
		assert.NoError(t, err)

		length, err := test.GetGaugeValue(gauge)
		assert.NoError(t, err)
		assert.Equal(t, i+1, int(length))
	}

	// each queue should have 1 thing
	for i := 0; i < totalQueues; i++ {
		op := q.Dequeue(i)
		assert.NotNil(t, op)

		length, err := test.GetGaugeValue(gauge)
		assert.NoError(t, err)
		assert.Equal(t, totalQueues-(i+1), int(length))
	}
}
