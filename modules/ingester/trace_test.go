package ingester

import (
	"context"
	"testing"

	prom_dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestTraceMaxSearchBytes(t *testing.T) {
	tenantID := "fake"
	maxSearchBytes := 100
	tr := newTrace(nil, 0, maxSearchBytes)

	getMetric := func() float64 {
		m := &prom_dto.Metric{}
		err := metricTraceSearchBytesDiscardedTotal.WithLabelValues(tenantID).Write(m)
		require.NoError(t, err)
		return m.Counter.GetValue()
	}

	err := tr.Push(context.TODO(), tenantID, nil, make([]byte, maxSearchBytes))
	require.NoError(t, err)
	require.Equal(t, float64(0), getMetric())

	tooMany := 123

	err = tr.Push(context.TODO(), tenantID, nil, make([]byte, tooMany))
	require.NoError(t, err)
	require.Equal(t, float64(tooMany), getMetric())

	err = tr.Push(context.TODO(), tenantID, nil, make([]byte, tooMany))
	require.NoError(t, err)
	require.Equal(t, float64(tooMany*2), getMetric())
}
