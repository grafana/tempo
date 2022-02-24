package generator

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	gokitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	prometheus_common_config "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	prometheus_config "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/util/test"
)

const localhost = "localhost"

type metric struct {
	name string
	val  float64
}

func TestGenerator(t *testing.T) {
	// logs will be useful to debug problems
	// TODO pass the logger as a parameter to generator.New instead of overriding a global variable
	log.Logger = gokitlog.NewLogfmtLogger(gokitlog.NewSyncWriter(os.Stdout))

	rwServer, doneCh := remoteWriteServer(t, expectedMetrics)
	defer rwServer.Close()

	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	// Ring
	mockStore, _ := consul.NewInMemoryClient(ring.GetCodec(), gokitlog.NewNopLogger(), nil)

	cfg.Ring.KVStore.Mock = mockStore
	cfg.Ring.ListenPort = 0
	cfg.Ring.InstanceID = localhost
	cfg.Ring.InstanceAddr = localhost

	// Overrides
	limitsTestConfig := defaultLimitsTestConfig()
	limitsTestConfig.MetricsGeneratorProcessors = map[string]struct{}{"service-graphs": {}, "span-metrics": {}}
	limits, err := overrides.NewOverrides(limitsTestConfig)
	require.NoError(t, err, "unexpected error creating overrides")

	// Remote write
	url, err := url.Parse(fmt.Sprintf("http://%s/receive", rwServer.Listener.Addr().String()))
	require.NoError(t, err)
	cfg.RemoteWrite.Enabled = true
	cfg.RemoteWrite.Client = prometheus_config.DefaultRemoteWriteConfig
	cfg.RemoteWrite.Client.URL = &prometheus_common_config.URL{URL: url}

	generator, err := New(cfg, limits, prometheus.NewRegistry())
	require.NoError(t, err, "unexpected error creating generator")

	err = generator.starting(context.Background())
	require.NoError(t, err, "unexpected error starting generator")

	// Send some spans
	req := test.MakeBatch(10, nil)
	ctx := user.InjectOrgID(context.Background(), util.FakeTenantID)
	_, err = generator.PushSpans(ctx, &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{req}})
	require.NoError(t, err, "unexpected error pushing spans")

	generator.collectMetrics()

	select {
	case <-doneCh:
	case <-time.After(time.Second * 5):
		t.Fatal("timeout while waiting for remote write server to receive spans")
	}
}

func TestGenerator_shutdown(t *testing.T) {
	// logs will be useful to debug problems
	// TODO pass the logger as a parameter to generator.New instead of overriding a global variable
	log.Logger = gokitlog.NewLogfmtLogger(gokitlog.NewSyncWriter(os.Stdout))

	rwServer, doneCh := remoteWriteServer(t, expectedMetrics)
	defer rwServer.Close()

	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	// Ring
	mockStore, _ := consul.NewInMemoryClient(ring.GetCodec(), gokitlog.NewNopLogger(), nil)

	cfg.Ring.KVStore.Mock = mockStore
	cfg.Ring.ListenPort = 0
	cfg.Ring.InstanceID = localhost
	cfg.Ring.InstanceAddr = localhost

	// Overrides
	limitsTestConfig := defaultLimitsTestConfig()
	limitsTestConfig.MetricsGeneratorProcessors = map[string]struct{}{"service-graphs": {}, "span-metrics": {}}
	limits, err := overrides.NewOverrides(limitsTestConfig)
	require.NoError(t, err, "unexpected error creating overrides")

	// Remote write
	url, err := url.Parse(fmt.Sprintf("http://%s/receive", rwServer.Listener.Addr().String()))
	require.NoError(t, err)
	cfg.RemoteWrite.Enabled = true
	cfg.RemoteWrite.Client = prometheus_config.DefaultRemoteWriteConfig
	cfg.RemoteWrite.Client.URL = &prometheus_common_config.URL{URL: url}

	// Set incredibly high collection interval
	cfg.CollectionInterval = time.Hour

	generator, err := New(cfg, limits, prometheus.NewRegistry())
	require.NoError(t, err, "unexpected error creating generator")

	err = services.StartAndAwaitRunning(context.Background(), generator)
	require.NoError(t, err, "unexpected error starting generator")

	// Send some spans
	req := test.MakeBatch(10, nil)
	ctx := user.InjectOrgID(context.Background(), util.FakeTenantID)
	_, err = generator.PushSpans(ctx, &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{req}})
	require.NoError(t, err, "unexpected error pushing spans")

	err = services.StopAndAwaitTerminated(context.Background(), generator)
	require.NoError(t, err, "failed to terminate metrics-generator")

	select {
	case <-doneCh:
	case <-time.After(time.Second * 5):
		t.Fatal("timeout while waiting for remote write server to receive spans")
	}
}

func remoteWriteServer(t *testing.T, expected []metric) (*httptest.Server, chan struct{}) {
	doneCh := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
		req, err := remote.DecodeWriteRequest(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		level.Info(log.Logger).Log("msg", "received remote write", "body", req.String())

		for i, ts := range req.Timeseries {
			m := make(model.Metric, len(ts.Labels))
			for _, l := range ts.Labels {
				m[model.LabelName(l.Name)] = model.LabelValue(l.Value)
			}

			// don't bother testing timeseries with exemplars for now, this is already covered by other tests
			if len(ts.Exemplars) != 0 {
				continue
			}

			if i >= len(expected) {
				assert.Fail(t, "received unexpected metric", "%s", m.String())
				continue
			}

			assert.Equal(t, expected[i].name, m.String())

			assert.Len(t, ts.Samples, 1)
			assert.Equal(t, expected[i].val, ts.Samples[0].Value)
		}
		close(doneCh)
	})

	return httptest.NewServer(mux), doneCh
}

var expectedMetrics = []metric{
	{`traces_spanmetrics_calls_total{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 10},
	{`traces_spanmetrics_duration_seconds_count{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 10},
	{`traces_spanmetrics_duration_seconds_sum{service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 10},
	{`traces_spanmetrics_duration_seconds_bucket{le="0.002", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 0},
	{`traces_spanmetrics_duration_seconds_bucket{le="0.004", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 0},
	{`traces_spanmetrics_duration_seconds_bucket{le="0.008", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 0},
	{`traces_spanmetrics_duration_seconds_bucket{le="0.016", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 0},
	{`traces_spanmetrics_duration_seconds_bucket{le="0.032", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 0},
	{`traces_spanmetrics_duration_seconds_bucket{le="0.064", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 0},
	{`traces_spanmetrics_duration_seconds_bucket{le="0.128", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 0},
	{`traces_spanmetrics_duration_seconds_bucket{le="0.256", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 0},
	{`traces_spanmetrics_duration_seconds_bucket{le="0.512", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 0},
	{`traces_spanmetrics_duration_seconds_bucket{le="1.024", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 10},
	{`traces_spanmetrics_duration_seconds_bucket{le="2.048", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 10},
	{`traces_spanmetrics_duration_seconds_bucket{le="4.096", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 10},
	{`traces_spanmetrics_duration_seconds_bucket{le="+Inf", service="test-service", span_kind="SPAN_KIND_CLIENT", span_name="test", span_status="STATUS_CODE_OK", tempo_instance_id="localhost"}`, 10},
}

func defaultLimitsTestConfig() overrides.Limits {
	limits := overrides.Limits{}
	flagext.DefaultValues(&limits)
	return limits
}
