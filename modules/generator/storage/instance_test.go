package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
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
	"github.com/weaveworks/common/user"
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

	instance, err := New(&cfg, "test-tenant", prometheus.DefaultRegisterer, logger)
	assert.NoError(t, err)

	// Remote storage must start tailing the WAL before we append data, the WAL watcher has a timeout of 5 seconds
	time.Sleep(6 * time.Second)

	// Refuse requests - the WAL should buffer data until requests succeed
	mockServer.refuseRequests.Store(true)

	// Append some data
	appender := instance.Appender(context.Background())

	lbls := labels.FromMap(map[string]string{"__name__": "my-metrics"})
	ref, err := appender.Append(0, lbls, time.Now().UnixMilli(), 1.0)
	assert.NoError(t, err)

	_, err = appender.AppendExemplar(ref, lbls, exemplar.Exemplar{
		Labels: labels.FromMap(map[string]string{"traceID": "123"}),
		Value:  1.2,
	})
	assert.NoError(t, err)

	err = appender.Commit()
	assert.NoError(t, err)

	// Give remote write some time to try sending data
	time.Sleep(100 * time.Millisecond)

	// Allow requests
	mockServer.refuseRequests.Store(false)

	// Shutdown the instance - remote write should flush pending data
	err = instance.Close()
	assert.NoError(t, err)

	// WAL should be empty again
	entries, err := os.ReadDir(cfg.Path)
	assert.NoError(t, err)
	assert.Len(t, entries, 0)

	// Verify we received metrics
	assert.Len(t, mockServer.timeSeries, 1)
	// We should have received 2 time series: one for the sample and one for the examplar
	assert.Len(t, mockServer.timeSeries["test-tenant"], 2)
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

	for i := 0; i < 2; i++ {
		instance, err := New(&cfg, strconv.Itoa(i), prometheus.DefaultRegisterer, logger)
		assert.NoError(t, err)
		instances = append(instances, instance)
	}

	// Remote storage must start tailing the WAL before we append data, the WAL watcher has a timeout of 5 seconds
	time.Sleep(6 * time.Second)

	for i, instance := range instances {
		appender := instance.Appender(context.Background())

		lbls := labels.FromMap(map[string]string{"__name__": "my-metrics"})
		_, err = appender.Append(0, lbls, time.Now().UnixMilli(), float64(i))
		assert.NoError(t, err)

		err = appender.Commit()
		assert.NoError(t, err)
	}

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
		lenOk := assert.Len(t, mockServer.timeSeries[strconv.Itoa(i)], 1, "instance %d did not receive the expected amount of time series", i)
		if lenOk {
			sample := mockServer.timeSeries[strconv.Itoa(i)][0]
			assert.Equal(t, float64(i), sample.GetSamples()[0].GetValue())
		}
	}
}

type mockPrometheusRemoteWriteServer struct {
	server         *httptest.Server
	refuseRequests *atomic.Bool
	timeSeries     map[string][]prompb.TimeSeries

	logger log.Logger
}

func newMockPrometheusRemoteWriterServer(logger log.Logger) *mockPrometheusRemoteWriteServer {
	logger = log.With(logger, "component", "mockserver")

	m := &mockPrometheusRemoteWriteServer{
		refuseRequests: atomic.NewBool(false),
		timeSeries:     make(map[string][]prompb.TimeSeries),
		logger:         logger,
	}
	m.server = httptest.NewServer(m)
	return m
}

func (m *mockPrometheusRemoteWriteServer) remoteWriteConfig() []*config.RemoteWriteConfig {
	rwCfg := &config.DefaultRemoteWriteConfig
	rwCfg.URL = &prometheus_common_config.URL{URL: urlMustParse(fmt.Sprintf("%s/receive", m.server.URL))}
	rwCfg.SendExemplars = true
	// Aggressive queue settings to speed up tests
	rwCfg.QueueConfig.BatchSendDeadline = model.Duration(10 * time.Millisecond)
	return []*config.RemoteWriteConfig{rwCfg}
}

func (m *mockPrometheusRemoteWriteServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if m.refuseRequests.Load() {
		m.logger.Log("msg", "refusing request")
		http.Error(res, "request refused", http.StatusServiceUnavailable)
		return
	}

	tenant := req.Header.Get(user.OrgIDHeaderName)
	m.logger.Log("msg", "received request", "tenant", tenant)

	writeRequest, err := remote.DecodeWriteRequest(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	m.timeSeries[tenant] = append(m.timeSeries[tenant], writeRequest.GetTimeseries()...)
}

func (m *mockPrometheusRemoteWriteServer) close() {
	m.server.Close()
}
