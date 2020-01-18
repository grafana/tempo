package ingester

import (
	"context"
	"testing"
	"time"

	"github.com/joe-elliott/frigg/pkg/friggpb"
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

	request := makeRequest(10)

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

func makeRequest(spans int) *friggpb.PushRequest {

	sampleSpan := friggpb.Span{
		Name:    "test",
		TraceID: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SpanID:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
	}

	req := &friggpb.PushRequest{
		Spans: []*friggpb.Span{},
		Process: &friggpb.Process{
			Name: "test",
		},
	}

	for i := 0; i < spans; i++ {
		req.Spans = append(req.Spans, &sampleSpan)
	}

	return req
}
