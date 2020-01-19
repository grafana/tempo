package ingester

import (
	"context"
	"testing"
	"time"

	"github.com/joe-elliott/frigg/pkg/util/test"
	"github.com/joe-elliott/frigg/pkg/util/validation"

	"github.com/stretchr/testify/assert"
)

type ringCountMock struct {
	count int
}

func (m *ringCountMock) HealthyInstancesCount() int {
	return m.count
}

func TestInstance(t *testing.T) {
	limits, err := validation.NewOverrides(validation.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	request := test.MakeRequest(10)

	i := newInstance("fake", limiter)
	i.Push(context.Background(), request)

	i.CutCompleteTraces(0, true)

	ready := i.IsBlockReady(5, 0)
	assert.True(t, ready, "block should be ready due to time")

	ready = i.IsBlockReady(0, 30*time.Hour)
	assert.True(t, ready, "block should be ready due to max traces")

	block := i.GetBlock()
	assert.Equal(t, 1, len(block))
	assert.Equal(t, request, block[0].batches[0])
}
