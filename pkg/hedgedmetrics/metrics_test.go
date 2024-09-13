package hedgedmetrics

import (
	"sync"
	"testing"
	"time"

	"github.com/cristalhq/hedgedhttp"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCounter satisfies prometheus.Counter
type MockCounter struct {
	mock.Mock
	val float64
}

func (m *MockCounter) Desc() *prometheus.Desc {
	return nil
}

func (m *MockCounter) Write(_ *dto.Metric) error {
	return nil
}

func (m *MockCounter) Describe(_ chan<- *prometheus.Desc) {
	// no-op
}

func (m *MockCounter) Collect(_ chan<- prometheus.Metric) {
	// no-op
}

func (m *MockCounter) Add(v float64) {
	m.val += v
	m.Called(v)
}

func (m *MockCounter) Inc() {
	m.Add(1.0)
}

func TestCounterWithValue(t *testing.T) {
	mockCounter := new(MockCounter)
	mockCounter.On("Add", mock.AnythingOfType("float64")).Return()

	counter := NewCounterWithValue(mockCounter)

	assert.Equal(t, int64(0), counter.Value())

	counter.Add(5)
	assert.Equal(t, int64(5), counter.Value())
	mockCounter.AssertCalled(t, "Add", 5.0)

	counter.Add(3)
	assert.Equal(t, int64(8), counter.Value())
	mockCounter.AssertCalled(t, "Add", 3.0)

	counter.Add(0)
	assert.Equal(t, int64(8), counter.Value())
	mockCounter.AssertCalled(t, "Add", 0.0)
}

// MockStatsProvider is StatsProvider for testing
type MockStatsProvider struct {
	mu                  sync.Mutex
	actualRoundTrips    uint64
	requestedRoundTrips uint64
}

func (m *MockStatsProvider) Snapshot() hedgedhttp.StatsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return hedgedhttp.StatsSnapshot{
		ActualRoundTrips:    m.actualRoundTrips,
		RequestedRoundTrips: m.requestedRoundTrips,
	}
}

func (m *MockStatsProvider) SetStats(actual, requested uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.actualRoundTrips = actual
	m.requestedRoundTrips = requested
}

func TestPublish(t *testing.T) {
	promCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "test",
		Name:      "counter",
		Help:      "test counter",
	})
	counter := NewCounterWithValue(promCounter)
	stats := &MockStatsProvider{}

	// start the Publishing every 10ms
	Publish(stats, counter, 10*time.Millisecond)

	assert.Equal(t, int64(0), counter.Value())

	// Set initial stats values
	stats.SetStats(5, 5)
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, int64(0), counter.Value())

	stats.SetStats(15, 10)
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, int64(5), counter.Value())

	stats.SetStats(28, 20)
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, int64(8), counter.Value())

	stats.SetStats(38, 25)
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, int64(13), counter.Value())

	time.Sleep(30 * time.Millisecond)

	// counter doesn't increase if stats stay same
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, int64(13), counter.Value())
}
