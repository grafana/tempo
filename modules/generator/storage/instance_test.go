package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	prometheus_common_config "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

// Verify basic functionality like sending metrics and exemplars, buffering and retrying failed
// requests.
func TestInstance(t *testing.T) {
	var err error
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))

	mockServer := newMockPrometheusRemoteWriterServer(logger)
	defer mockServer.close()

	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.Path = t.TempDir()
	cfg.RemoteWrite = mockServer.remoteWriteConfig()

	instance, err := New(&cfg, &mockOverrides{}, "test-tenant", &noopRegisterer{}, logger)
	require.NoError(t, err)

	// Refuse requests - the WAL should buffer data until requests succeed
	mockServer.refuseRequests.Store(true)

	sendCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Append some data every second
	go poll(sendCtx, time.Second, func() {
		appender := instance.Appender(context.Background())

		lbls := labels.FromMap(map[string]string{"__name__": "my-metrics"})
		ref, err := appender.Append(0, lbls, time.Now().UnixMilli(), 1.0)
		assert.NoError(t, err)

		_, err = appender.AppendExemplar(ref, lbls, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"traceID": "123"}),
			Value:  1.2,
		})
		assert.NoError(t, err)

		if sendCtx.Err() != nil {
			return
		}

		err = appender.Commit()
		assert.NoError(t, err)
	})

	// Wait until remote.Storage has tried at least once to send data
	err = waitUntil(20*time.Second, func() bool {
		mockServer.mtx.Lock()
		defer mockServer.mtx.Unlock()

		return mockServer.refusedRequests > 0
	})
	require.NoError(t, err, "timed out while waiting for refused requests")

	// Allow requests
	mockServer.refuseRequests.Store(false)

	// Shutdown the instance - even though previous requests failed, remote.Storage should flush pending data
	err = instance.Close()
	assert.NoError(t, err)

	// WAL should be empty again
	entries, err := os.ReadDir(cfg.Path)
	assert.NoError(t, err)
	assert.Len(t, entries, 0)

	// Verify we received metrics
	assert.Len(t, mockServer.timeSeries, 1)
	assert.Contains(t, mockServer.timeSeries, "test-tenant")
	// We should have received at least 2 time series: one for the sample and one for the examplar
	assert.GreaterOrEqual(t, len(mockServer.timeSeries["test-tenant"]), 2)
}

// Verify multiple instances function next to each other, don't trample over each other and are isolated.
func TestInstance_multiTenancy(t *testing.T) {
	var err error
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))

	mockServer := newMockPrometheusRemoteWriterServer(logger)
	defer mockServer.close()

	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.Path = t.TempDir()
	cfg.RemoteWrite = mockServer.remoteWriteConfig()

	var instances []Storage

	for i := 0; i < 3; i++ {
		instance, err := New(&cfg, &mockOverrides{}, strconv.Itoa(i), &noopRegisterer{}, logger)
		assert.NoError(t, err)
		instances = append(instances, instance)
	}

	sendCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Append some data every second
	go poll(sendCtx, time.Second, func() {
		for i, instance := range instances {
			appender := instance.Appender(context.Background())

			lbls := labels.FromMap(map[string]string{"__name__": "my-metric"})
			_, err := appender.Append(0, lbls, time.Now().UnixMilli(), float64(i))
			assert.NoError(t, err)

			if sendCtx.Err() != nil {
				return
			}

			err = appender.Commit()
			assert.NoError(t, err)
		}
	})

	// Wait until every tenant received at least one request
	err = waitUntil(20*time.Second, func() bool {
		mockServer.mtx.Lock()
		defer mockServer.mtx.Unlock()

		for i := range instances {
			if mockServer.acceptedRequests[strconv.Itoa(i)] == 0 {
				return false
			}
		}
		return true
	})
	require.NoError(t, err, "timed out while waiting for accepted requests")

	for _, instance := range instances {
		// Shutdown the instance - remote write should flush pending data
		err = instance.Close()
		assert.NoError(t, err)
	}

	// WAL should be empty again
	entries, err := os.ReadDir(cfg.Path)
	assert.NoError(t, err)
	require.Len(t, entries, 0)

	for i := range instances {
		lenOk := assert.GreaterOrEqual(t, len(mockServer.timeSeries[strconv.Itoa(i)]), 1, "instance %d did not receive the expected amount of time series", i)
		if lenOk {
			sample := mockServer.timeSeries[strconv.Itoa(i)][0]
			assert.Equal(t, float64(i), sample.GetSamples()[0].GetValue())
		}
	}
}

func TestInstance_cantWriteToWAL(t *testing.T) {
	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	// We are obviously not allowed to write here
	cfg.Path = "/root"

	// We should be able to attempt to create the instance multiple times
	_, err := New(&cfg, &mockOverrides{}, "test-tenant", &noopRegisterer{}, log.NewNopLogger())
	require.Error(t, err)
	_, err = New(&cfg, &mockOverrides{}, "test-tenant", &noopRegisterer{}, log.NewNopLogger())
	require.Error(t, err)
}

func TestInstance_remoteWriteHeaders(t *testing.T) {
	var err error
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))

	mockServer := newMockPrometheusRemoteWriterServer(logger)
	defer mockServer.close()

	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.Path = t.TempDir()
	cfg.RemoteWrite = mockServer.remoteWriteConfig()

	headers := map[string]string{user.OrgIDHeaderName: "my-other-tenant"}

	instance, err := New(&cfg, &mockOverrides{headers}, "test-tenant", &noopRegisterer{}, logger)
	require.NoError(t, err)

	// Refuse requests - the WAL should buffer data until requests succeed
	mockServer.refuseRequests.Store(true)

	sendCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Append some data every second
	go poll(sendCtx, time.Second, func() {
		appender := instance.Appender(context.Background())

		lbls := labels.FromMap(map[string]string{"__name__": "my-metrics"})
		ref, err := appender.Append(0, lbls, time.Now().UnixMilli(), 1.0)
		assert.NoError(t, err)

		_, err = appender.AppendExemplar(ref, lbls, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"traceID": "123"}),
			Value:  1.2,
		})
		assert.NoError(t, err)

		if sendCtx.Err() != nil {
			return
		}

		assert.NoError(t, appender.Commit())
	})

	// Wait until remote.Storage has tried at least once to send data
	err = waitUntil(20*time.Second, func() bool {
		mockServer.mtx.Lock()
		defer mockServer.mtx.Unlock()

		return mockServer.refusedRequests > 0
	})
	require.NoError(t, err, "timed out while waiting for refused requests")

	// Allow requests
	mockServer.refuseRequests.Store(false)

	// Shutdown the instance - even though previous requests failed, remote.Storage should flush pending data
	err = instance.Close()
	assert.NoError(t, err)

	// WAL should be empty again
	entries, err := os.ReadDir(cfg.Path)
	assert.NoError(t, err)
	assert.Len(t, entries, 0)

	// Verify we received metrics
	assert.Len(t, mockServer.timeSeries, 1)
	assert.Contains(t, mockServer.timeSeries, "my-other-tenant")
	// We should have received at least 2 time series: one for the sample and one for the examplar
	assert.GreaterOrEqual(t, len(mockServer.timeSeries["my-other-tenant"]), 2)
}

type mockPrometheusRemoteWriteServer struct {
	mtx sync.Mutex

	server         *httptest.Server
	refuseRequests *atomic.Bool
	timeSeries     map[string][]prompb.TimeSeries

	// metrics
	refusedRequests  int
	acceptedRequests map[string]int

	logger log.Logger
}

func newMockPrometheusRemoteWriterServer(logger log.Logger) *mockPrometheusRemoteWriteServer {
	logger = log.With(logger, "component", "mockserver")

	m := &mockPrometheusRemoteWriteServer{
		refuseRequests:   atomic.NewBool(false),
		timeSeries:       make(map[string][]prompb.TimeSeries),
		acceptedRequests: map[string]int{},
		logger:           logger,
	}
	m.server = httptest.NewServer(m)
	return m
}

func (m *mockPrometheusRemoteWriteServer) remoteWriteConfig() []config.RemoteWriteConfig {
	rwCfg := config.DefaultRemoteWriteConfig
	rwCfg.URL = &prometheus_common_config.URL{URL: urlMustParse(fmt.Sprintf("%s/receive", m.server.URL))}
	rwCfg.SendExemplars = true
	// Aggressive queue settings to speed up tests
	rwCfg.QueueConfig.BatchSendDeadline = model.Duration(10 * time.Millisecond)
	return []config.RemoteWriteConfig{rwCfg}
}

func (m *mockPrometheusRemoteWriteServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if m.refuseRequests.Load() {
		m.logger.Log("msg", "refusing request")
		m.refusedRequests++
		http.Error(res, "request refused", http.StatusServiceUnavailable)
		return
	}

	tenant := req.Header.Get(user.OrgIDHeaderName)
	m.logger.Log("msg", "received request", "tenant", tenant)
	m.acceptedRequests[tenant]++

	writeRequest, err := remote.DecodeWriteRequest(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	m.timeSeries[tenant] = append(m.timeSeries[tenant], writeRequest.GetTimeseries()...)
}

func (m *mockPrometheusRemoteWriteServer) close() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.server.Close()
}

// poll executes f every interval until ctx is done or cancelled.
func poll(ctx context.Context, interval time.Duration, f func()) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f()
		}
	}
}

// waitUntil executes f until it returns true or timeout is reached.
func waitUntil(timeout time.Duration, f func() bool) error {
	start := time.Now()

	for {
		if f() {
			return nil
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("timed out while waiting for condition")
		}

		time.Sleep(50 * time.Millisecond)
	}
}

var _ Overrides = (*mockOverrides)(nil)

type mockOverrides struct {
	headers map[string]string
}

func (m *mockOverrides) MetricsGeneratorRemoteWriteHeaders(string) map[string]string {
	return m.headers
}

var _ prometheus.Registerer = (*noopRegisterer)(nil)

type noopRegisterer struct{}

func (n *noopRegisterer) Register(prometheus.Collector) error { return nil }

func (n *noopRegisterer) MustRegister(...prometheus.Collector) {}

func (n *noopRegisterer) Unregister(prometheus.Collector) bool { return true }
