package ingester

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	prom_dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceMaxSearchBytes(t *testing.T) {
	tenantID := "fake"
	maxSearchBytes := 100
	tr := newTrace(nil, 0, maxSearchBytes)
	fakeTrace := make([]byte, 64)

	getMetric := func() float64 {
		m := &prom_dto.Metric{}
		err := metricTraceSearchBytesDiscardedTotal.WithLabelValues(tenantID).Write(m)
		require.NoError(t, err)
		return m.Counter.GetValue()
	}

	err := tr.Push(context.TODO(), tenantID, fakeTrace, make([]byte, maxSearchBytes))
	require.NoError(t, err)
	require.Equal(t, float64(0), getMetric())

	tooMany := 123

	err = tr.Push(context.TODO(), tenantID, fakeTrace, make([]byte, tooMany))
	require.NoError(t, err)
	require.Equal(t, float64(tooMany), getMetric())

	err = tr.Push(context.TODO(), tenantID, fakeTrace, make([]byte, tooMany))
	require.NoError(t, err)
	require.Equal(t, float64(tooMany*2), getMetric())
}

func TestTraceStartEndTime(t *testing.T) {
	s := model.MustNewSegmentDecoder(model.CurrentEncoding)

	tr := newTrace(nil, 0, 0)

	// initial push
	buff, err := s.PrepareForWrite(&tempopb.Trace{}, 10, 20)
	require.NoError(t, err)
	err = tr.Push(context.Background(), "test", buff, nil)
	require.NoError(t, err)

	assert.Equal(t, uint32(10), tr.start)
	assert.Equal(t, uint32(20), tr.end)

	// overwrite start
	buff, err = s.PrepareForWrite(&tempopb.Trace{}, 5, 15)
	require.NoError(t, err)
	err = tr.Push(context.Background(), "test", buff, nil)
	require.NoError(t, err)

	assert.Equal(t, uint32(5), tr.start)
	assert.Equal(t, uint32(20), tr.end)

	// overwrite end
	buff, err = s.PrepareForWrite(&tempopb.Trace{}, 15, 25)
	require.NoError(t, err)
	err = tr.Push(context.Background(), "test", buff, nil)
	require.NoError(t, err)

	assert.Equal(t, uint32(5), tr.start)
	assert.Equal(t, uint32(25), tr.end)
}
