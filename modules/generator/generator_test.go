package generator

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"
	"gopkg.in/yaml.v3"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
)

type metric struct {
	name string
	val  float64
}

func TestGenerator(t *testing.T) {
	rwServer, doneCh := remoteWriteServer(t, expectedMetrics)
	defer rwServer.Close()

	var cfg Config

	url, err := url.Parse(fmt.Sprintf("http://%s/receive", rwServer.Listener.Addr().String()))
	require.NoError(t, err)

	rwStrCfg := fmt.Sprintf(`
enabled: true
client:
  url: %s
`, url.String())

	var rwCfg RemoteWriteConfig
	err = yaml.NewDecoder(strings.NewReader(rwStrCfg)).Decode(&rwCfg)
	require.NoError(t, err, "failed to decode remote write config")
	cfg.RemoteWrite = rwCfg

	flagext.DefaultValues(&cfg.LifecyclerConfig)
	mockStore, _ := consul.NewInMemoryClient(
		ring.GetCodec(),
		log.NewNopLogger(),
		nil,
	)

	cfg.LifecyclerConfig.RingConfig.KVStore.Mock = mockStore
	cfg.LifecyclerConfig.NumTokens = 1
	cfg.LifecyclerConfig.ListenPort = 0
	cfg.LifecyclerConfig.Addr = "localhost"
	cfg.LifecyclerConfig.ID = "localhost"
	cfg.LifecyclerConfig.FinalSleep = 0

	limits, err := overrides.NewOverrides(defaultLimitsTestConfig())
	require.NoError(t, err, "unexpected error creating overrides")

	generator, err := New(cfg, limits, prometheus.NewRegistry())
	require.NoError(t, err, "unexpected error creating generator")

	err = generator.starting(context.Background())
	require.NoError(t, err, "unexpected error starting ingester")

	req := test.MakeBatch(10, nil)
	ctx := user.InjectOrgID(context.Background(), util.FakeTenantID)
	_, err = generator.PushSpans(ctx, &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{req}})
	require.NoError(t, err, "unexpected error pushing spans")

	generator.collectMetrics()

	select {
	case <-doneCh:
	case <-time.After(time.Second * 5):
		t.Fatal("timeout waiting for remote write server to receive spans")
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

		for i, ts := range req.Timeseries {
			m := make(model.Metric, len(ts.Labels))
			for _, l := range ts.Labels {
				m[model.LabelName(l.Name)] = model.LabelValue(l.Value)
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
	{"tempo_calls_total{service=\"test-service\", span_kind=\"SPAN_KIND_CLIENT\", span_name=\"test\", span_status=\"STATUS_CODE_OK\", tenant=\"single-tenant\"}", 10},
	{"tempo_latency_count{service=\"test-service\", span_kind=\"SPAN_KIND_CLIENT\", span_name=\"test\", span_status=\"STATUS_CODE_OK\", tenant=\"single-tenant\"}", 10},
	{"tempo_latency_sum{service=\"test-service\", span_kind=\"SPAN_KIND_CLIENT\", span_name=\"test\", span_status=\"STATUS_CODE_OK\", tenant=\"single-tenant\"}", 0},
	{"tempo_latency_bucket{le=\"1\", service=\"test-service\", span_kind=\"SPAN_KIND_CLIENT\", span_name=\"test\", span_status=\"STATUS_CODE_OK\", tenant=\"single-tenant\"}", 0},
	{"tempo_latency_bucket{le=\"10\", service=\"test-service\", span_kind=\"SPAN_KIND_CLIENT\", span_name=\"test\", span_status=\"STATUS_CODE_OK\", tenant=\"single-tenant\"}", 0},
	{"tempo_latency_bucket{le=\"50\", service=\"test-service\", span_kind=\"SPAN_KIND_CLIENT\", span_name=\"test\", span_status=\"STATUS_CODE_OK\", tenant=\"single-tenant\"}", 0},
	{"tempo_latency_bucket{le=\"100\", service=\"test-service\", span_kind=\"SPAN_KIND_CLIENT\", span_name=\"test\", span_status=\"STATUS_CODE_OK\", tenant=\"single-tenant\"}", 0},
	{"tempo_latency_bucket{le=\"500\", service=\"test-service\", span_kind=\"SPAN_KIND_CLIENT\", span_name=\"test\", span_status=\"STATUS_CODE_OK\", tenant=\"single-tenant\"}", 0},
	{"tempo_latency_bucket{le=\"+Inf\", service=\"test-service\", span_kind=\"SPAN_KIND_CLIENT\", span_name=\"test\", span_status=\"STATUS_CODE_OK\", tenant=\"single-tenant\"}", 0},
}

func defaultLimitsTestConfig() overrides.Limits {
	limits := overrides.Limits{}
	flagext.DefaultValues(&limits)
	return limits
}
