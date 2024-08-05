package deployments

import (
	"fmt"
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/v2/integration/util"
	"github.com/grafana/tempo/v2/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/v2/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

const configMicroservices = "config-microservices.tmpl.yaml"

func TestMicroservicesWithKVStores(t *testing.T) {
	testKVStores := []struct {
		name     string
		kvconfig func(hostname string, port int) string
	}{
		{
			name: "memberlist",
			kvconfig: func(string, int) string {
				return `
        store: memberlist`
			},
		},
		{
			name: "etcd",
			kvconfig: func(hostname string, port int) string {
				return fmt.Sprintf(`
        store: etcd
        etcd:
          endpoints:
            - http://%s:%d`, hostname, port)
			},
		},
		{
			name: "consul",
			kvconfig: func(hostname string, port int) string {
				return fmt.Sprintf(`
        store: consul
        consul:
          host: http://%s:%d`, hostname, port)
			},
		},
	}

	for _, tc := range testKVStores {
		t.Run(tc.name, func(t *testing.T) {
			s, err := e2e.NewScenario("tempo_e2e")
			require.NoError(t, err)
			defer s.Close()

			// Set up KVStore
			var kvstore *e2e.HTTPService
			switch tc.name {
			case "etcd":
				kvstore = e2edb.NewETCD()
				require.NoError(t, s.StartAndWaitReady(kvstore))
			case "consul":
				kvstore = e2edb.NewConsul()
				require.NoError(t, s.StartAndWaitReady(kvstore))
			case "memberlist":
			default:
				t.Errorf("unknown KVStore %s", tc.name)
			}

			KVStoreConfig := tc.kvconfig("", 0)
			if kvstore != nil {
				KVStoreConfig = tc.kvconfig(kvstore.Name(), kvstore.HTTPPort())
			}

			// copy config template to shared directory and expand template variables
			tmplConfig := map[string]any{"KVStore": KVStoreConfig}
			_, err = util.CopyTemplateToSharedDir(s, configMicroservices, "config.yaml", tmplConfig)
			require.NoError(t, err)

			minio := e2edb.NewMinio(9000, "tempo")
			require.NotNil(t, minio)
			require.NoError(t, s.StartAndWaitReady(minio))

			tempoIngester1 := util.NewTempoIngester(1)
			tempoIngester2 := util.NewTempoIngester(2)
			tempoIngester3 := util.NewTempoIngester(3)

			tempoDistributor := util.NewTempoDistributor()
			tempoQueryFrontend := util.NewTempoQueryFrontend()
			tempoQuerier := util.NewTempoQuerier()
			require.NoError(t, s.StartAndWaitReady(tempoIngester1, tempoIngester2, tempoIngester3, tempoDistributor, tempoQueryFrontend, tempoQuerier))

			// wait for active ingesters
			time.Sleep(1 * time.Second)
			matchers := []*labels.Matcher{
				{
					Type:  labels.MatchEqual,
					Name:  "name",
					Value: "ingester",
				},
				{
					Type:  labels.MatchEqual,
					Name:  "state",
					Value: "ACTIVE",
				},
			}
			require.NoError(t, tempoDistributor.WaitSumMetricsWithOptions(e2e.Equals(3), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(matchers...), e2e.WaitMissingMetrics))

			// Get port for the Jaeger gRPC receiver endpoint
			c, err := util.NewJaegerGRPCClient(tempoDistributor.Endpoint(14250))
			require.NoError(t, err)
			require.NotNil(t, c)

			info := tempoUtil.NewTraceInfo(time.Now(), "")
			require.NoError(t, info.EmitAllBatches(c))

			expected, err := info.ConstructTraceFromEpoch()
			require.NoError(t, err)

			// test metrics
			require.NoError(t, tempoDistributor.WaitSumMetrics(e2e.Equals(util.SpanCount(expected)), "tempo_distributor_spans_received_total"))

			// test echo
			util.AssertEcho(t, "http://"+tempoQueryFrontend.Endpoint(3200)+"/api/echo")

			apiClient := httpclient.New("http://"+tempoQueryFrontend.Endpoint(3200), "")

			// query an in-memory trace
			util.QueryAndAssertTrace(t, apiClient, info)

			// wait trace_idle_time and ensure trace is created in ingester
			require.NoError(t, tempoIngester1.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))
			require.NoError(t, tempoIngester2.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))
			require.NoError(t, tempoIngester3.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

			// flush trace to backend
			util.CallFlush(t, tempoIngester1)
			util.CallFlush(t, tempoIngester2)
			util.CallFlush(t, tempoIngester3)

			// search for trace
			util.SearchAndAssertTrace(t, apiClient, info)
			util.SearchTraceQLAndAssertTrace(t, apiClient, info)

			// sleep for one maintenance cycle
			time.Sleep(5 * time.Second)

			// test metrics
			for _, i := range []*e2e.HTTPService{tempoIngester1, tempoIngester2, tempoIngester3} {
				require.NoError(t, i.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
			}
			require.NoError(t, tempoQuerier.WaitSumMetrics(e2e.Equals(3), "tempodb_blocklist_length"))
			require.NoError(t, tempoQueryFrontend.WaitSumMetrics(e2e.Equals(3), "tempo_query_frontend_queries_total"))

			// query trace - should fetch from backend
			util.QueryAndAssertTrace(t, apiClient, info)

			// stop an ingester and confirm we can still write and query
			err = tempoIngester2.Kill()
			require.NoError(t, err)

			// sleep for heartbeat timeout
			time.Sleep(1 * time.Second)

			info = tempoUtil.NewTraceInfo(time.Now(), "")
			require.NoError(t, info.EmitAllBatches(c))

			// query by id
			util.QueryAndAssertTrace(t, apiClient, info)

			// wait trace_idle_time and ensure trace is created in ingester
			require.NoError(t, tempoIngester1.WaitSumMetricsWithOptions(e2e.Less(4), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))
			require.NoError(t, tempoIngester3.WaitSumMetricsWithOptions(e2e.Less(4), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

			// flush trace to backend
			util.CallFlush(t, tempoIngester1)
			util.CallFlush(t, tempoIngester3)

			// search for trace
			util.SearchAndAssertTrace(t, apiClient, info)

			// stop another ingester and confirm things fail
			err = tempoIngester1.Kill()
			require.NoError(t, err)

			require.Error(t, info.EmitBatches(c))
		})
	}
}
