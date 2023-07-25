package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func newQueue[T any](t *testing.T, size, workerCount int, processFunc ProcessFunc[T]) *Queue[T] {
	cfg := Config{Name: "testName", TenantID: "testTenantID", Size: size, WorkerCount: workerCount}

	logger := log.NewNopLogger()
	q := New(cfg, logger, processFunc)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		require.NoError(t, q.Shutdown(ctx))

		// Metrics are defined on package-level, we need to reset them each time.
		pushesTotalMetrics.Reset()
		pushesFailuresTotalMetric.Reset()
		lengthMetric.Reset()
	})

	return q
}

func newStartedQueue[T any](t *testing.T, size, workerCount int, processFunc ProcessFunc[T]) *Queue[T] {
	q := newQueue(t, size, workerCount, processFunc)
	q.StartWorkers()

	return q
}

func getCounterValue(metric *prometheus.CounterVec) float64 {
	m := &dto.Metric{}
	if err := metric.WithLabelValues("testName", "testTenantID").Write(m); err != nil {
		return 0
	}

	return m.Counter.GetValue()
}

func TestNew_ReturnsNotNilAndSetsCorrectFieldsFromConfig(t *testing.T) {
	// Given
	cfg := Config{Name: "testName", TenantID: "testTenantID", Size: 123, WorkerCount: 321}
	processFunc := func(context.Context, int) {}
	logger := log.NewNopLogger()

	// When
	got := New(cfg, logger, processFunc)

	// Then
	require.NotNil(t, got)
	require.Equal(t, got.name, cfg.Name)
	require.Equal(t, got.tenantID, cfg.TenantID)
	require.Equal(t, got.size, cfg.Size)
	require.Equal(t, got.workerCount, cfg.WorkerCount)
}

func TestQueue_Push_ReturnsNoErrorAndWorkersInvokeProcessFuncCorrectNumberOfTimesWithRunningWorkers(t *testing.T) {
	// Given
	count := atomic.NewUint32(0)
	wg := sync.WaitGroup{}
	size := 10
	workerCount := 3
	processFunc := func(context.Context, any) {
		defer wg.Done()
		count.Inc()
	}
	q := newStartedQueue(t, size, workerCount, processFunc)

	// When
	for i := 0; i < size-3; i++ {
		wg.Add(1)
		require.NoError(t, q.Push(context.Background(), nil))
	}

	// Then
	wg.Wait()
	require.Equal(t, uint32(size-3), count.Load())
	require.Equal(t, float64(size-3), getCounterValue(q.pushesTotalMetrics))
	require.Zero(t, getCounterValue(q.pushesFailuresTotalMetrics))
}

func TestQueue_Push_ReturnsNoErrorWhenPushingLessItemsThanSizeWithStoppedWorkers(t *testing.T) {
	// Given
	size := 10
	workerCount := 3
	processFunc := func(context.Context, any) {}
	q := newQueue(t, size, workerCount, processFunc)

	// When
	for i := 0; i < size-3; i++ {
		require.NoError(t, q.Push(context.Background(), nil))
	}

	// Then
	require.Equal(t, size-3, len(q.reqChan))
	require.Equal(t, float64(size-3), getCounterValue(q.pushesTotalMetrics))
	require.Zero(t, getCounterValue(q.pushesFailuresTotalMetrics))
}

func TestQueue_Push_ReturnsErrorWhenPushingItemsToShutdownQueue(t *testing.T) {
	// Given
	size := 10
	workerCount := 3
	processFunc := func(context.Context, any) {}
	q := newStartedQueue(t, size, workerCount, processFunc)
	require.NoError(t, q.Shutdown(context.Background()))

	// When
	err := q.Push(context.Background(), nil)

	// Then
	require.Error(t, err)
	require.Zero(t, len(q.reqChan))
	require.Zero(t, getCounterValue(q.pushesTotalMetrics))
	require.Zero(t, getCounterValue(q.pushesFailuresTotalMetrics))
}

func TestQueue_Push_QueueGetsProperlyDrainedOnShutdown(t *testing.T) {
	// Given
	count := atomic.NewUint32(0)
	wg := sync.WaitGroup{}
	size := 10
	workerCount := 3
	processFunc := func(context.Context, any) {
		defer wg.Done()
		count.Inc()
	}
	q := newQueue(t, size, workerCount, processFunc)

	// When
	for i := 0; i < size-3; i++ {
		wg.Add(1)
		require.NoError(t, q.Push(context.Background(), nil))
	}

	require.NoError(t, q.Shutdown(context.Background()))
	q.StartWorkers()

	// Then
	wg.Wait()
	require.Zero(t, len(q.reqChan))
	require.Equal(t, float64(size-3), getCounterValue(q.pushesTotalMetrics))
	require.Zero(t, getCounterValue(q.pushesFailuresTotalMetrics))
}

func TestQueue_Push_ReturnsErrorWhenPushingItemsWithCancelledContext(t *testing.T) {
	// Given
	size := 10
	workerCount := 3
	processFunc := func(context.Context, any) {}
	q := newStartedQueue(t, size, workerCount, processFunc)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// When
	err := q.Push(ctx, nil)

	// Then
	require.Error(t, err)
	require.Zero(t, len(q.reqChan))
	require.Equal(t, float64(1), getCounterValue(q.pushesTotalMetrics))
	require.Equal(t, float64(1), getCounterValue(q.pushesFailuresTotalMetrics))
}

func TestQueue_Push_ReturnsErrorWhenPushingItemsToFullQueueWithStoppedWorkers(t *testing.T) {
	// Given
	size := 10
	workerCount := 3
	processFunc := func(context.Context, any) {}
	q := newQueue(t, size, workerCount, processFunc)

	// When
	for i := 0; i < size; i++ {
		require.NoError(t, q.Push(context.Background(), nil))
	}

	require.Error(t, q.Push(context.Background(), nil))

	// Then
	require.Equal(t, size, len(q.reqChan))
	require.Equal(t, float64(size+1), getCounterValue(q.pushesTotalMetrics))
	require.Equal(t, float64(1), getCounterValue(q.pushesFailuresTotalMetrics))
}

func TestQueue_ShouldUpdate_ReturnsTrueWhenWorkerCountDiffersFromOriginalValue(t *testing.T) {
	// Given
	q := newQueue[int](t, 2, 3, nil)

	// When
	got := q.ShouldUpdate(2, 7)

	// Then
	require.True(t, got)
}

func TestQueue_ShouldUpdate_ReturnsTrueWhenSizeDiffersFromOriginalValue(t *testing.T) {
	// Given
	q := newQueue[int](t, 2, 3, nil)

	// When
	got := q.ShouldUpdate(7, 3)

	// Then
	require.True(t, got)
}

func TestQueue_ShouldUpdate_ReturnsTrueWhenBothParametersDifferFromOriginalValue(t *testing.T) {
	// Given
	q := newQueue[int](t, 2, 3, nil)

	// When
	got := q.ShouldUpdate(13, 17)

	// Then
	require.True(t, got)
}

func TestQueue_ShouldUpdate_ReturnsFalseWhenBothParametersEqualOriginalValues(t *testing.T) {
	// Given
	q := newQueue[int](t, 2, 3, nil)

	// When
	got := q.ShouldUpdate(2, 3)

	// Then
	require.False(t, got)
}
