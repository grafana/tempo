package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"

	"github.com/grafana/tempo/cmd/tempo/app"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/integration/e2e/backend"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
)

const (
	configMicroservices = "config-microservices.tmpl.yaml"
	configHA            = "config-scalable-single-binary.yaml"

	configAllInOneS3      = "config-all-in-one-s3.yaml"
	configAllInOneAzurite = "config-all-in-one-azurite.yaml"
	configAllInOneGCS     = "config-all-in-one-gcs.yaml"
)

func TestAllInOne(t *testing.T) {
	testBackends := []struct {
		name       string
		configFile string
	}{
		{
			name:       "s3",
			configFile: configAllInOneS3,
		},
		{
			name:       "azure",
			configFile: configAllInOneAzurite,
		},
		{
			name:       "gcs",
			configFile: configAllInOneGCS,
		},
	}

	for _, tc := range testBackends {
		t.Run(tc.name, func(t *testing.T) {
			s, err := e2e.NewScenario("tempo_e2e")
			require.NoError(t, err)
			defer s.Close()

			// set up the backend
			cfg := app.Config{}
			buff, err := os.ReadFile(tc.configFile)
			require.NoError(t, err)
			err = yaml.UnmarshalStrict(buff, &cfg)
			require.NoError(t, err)
			_, err = backend.New(s, cfg)
			require.NoError(t, err)

			require.NoError(t, util.CopyFileToSharedDir(s, tc.configFile, "config.yaml"))
			tempo := util.NewTempoAllInOne()
			require.NoError(t, s.StartAndWaitReady(tempo))

			// Get port for the Jaeger gRPC receiver endpoint
			c, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
			require.NoError(t, err)
			require.NotNil(t, c)

			info := tempoUtil.NewTraceInfo(time.Now(), "")
			require.NoError(t, info.EmitAllBatches(c))

			expected, err := info.ConstructTraceFromEpoch()
			require.NoError(t, err)

			// test metrics
			require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(spanCount(expected)), "tempo_distributor_spans_received_total"))

			// test echo
			assertEcho(t, "http://"+tempo.Endpoint(3200)+"/api/echo")

			apiClient := httpclient.New("http://"+tempo.Endpoint(3200), "")

			// query an in-memory trace
			queryAndAssertTrace(t, apiClient, info)

			// wait trace_idle_time and ensure trace is created in ingester
			require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

			// flush trace to backend
			callFlush(t, tempo)

			// search for trace in backend
			util.SearchAndAssertTrace(t, apiClient, info)
			util.SearchTraceQLAndAssertTrace(t, apiClient, info)

			// sleep
			time.Sleep(10 * time.Second)

			// force clear completed block
			callFlush(t, tempo)

			// test metrics
			require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
			require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempodb_blocklist_length"}, e2e.WaitMissingMetrics))
			require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(3), "tempo_query_frontend_queries_total"))

			// query trace - should fetch from backend
			queryAndAssertTrace(t, apiClient, info)

			// search the backend. this works b/c we're passing a start/end AND setting query ingesters within min/max to 0
			now := time.Now()
			util.SearchAndAssertTraceBackend(t, apiClient, info, now.Add(-20*time.Minute).Unix(), now.Unix())

			// find the trace with streaming. using the http server b/c that's what Grafana will do
			grpcClient, err := util.NewSearchGRPCClient(context.Background(), tempo.Endpoint(3200))
			require.NoError(t, err)

			util.SearchStreamAndAssertTrace(t, grpcClient, info, now.Add(-20*time.Minute).Unix(), now.Unix())

			// test websockets
			wsClient := httpclient.New("ws://"+tempo.Endpoint(3200), "")
			util.SearchWSStreamAndAssertTrace(t, wsClient, info, now.Add(-20*time.Minute).Unix(), now.Unix())
		})
	}
}

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
			require.NoError(t, tempoDistributor.WaitSumMetrics(e2e.Equals(spanCount(expected)), "tempo_distributor_spans_received_total"))

			// test echo
			assertEcho(t, "http://"+tempoQueryFrontend.Endpoint(3200)+"/api/echo")

			apiClient := httpclient.New("http://"+tempoQueryFrontend.Endpoint(3200), "")

			// query an in-memory trace
			queryAndAssertTrace(t, apiClient, info)

			// wait trace_idle_time and ensure trace is created in ingester
			require.NoError(t, tempoIngester1.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))
			require.NoError(t, tempoIngester2.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))
			require.NoError(t, tempoIngester3.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

			// flush trace to backend
			callFlush(t, tempoIngester1)
			callFlush(t, tempoIngester2)
			callFlush(t, tempoIngester3)

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
			queryAndAssertTrace(t, apiClient, info)

			// stop an ingester and confirm we can still write and query
			err = tempoIngester2.Kill()
			require.NoError(t, err)

			// sleep for heartbeat timeout
			time.Sleep(1 * time.Second)

			info = tempoUtil.NewTraceInfo(time.Now(), "")
			require.NoError(t, info.EmitAllBatches(c))

			// query by id
			queryAndAssertTrace(t, apiClient, info)

			// wait trace_idle_time and ensure trace is created in ingester
			require.NoError(t, tempoIngester1.WaitSumMetricsWithOptions(e2e.Less(4), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))
			require.NoError(t, tempoIngester3.WaitSumMetricsWithOptions(e2e.Less(4), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

			// flush trace to backend
			callFlush(t, tempoIngester1)
			callFlush(t, tempoIngester3)

			// search for trace
			util.SearchAndAssertTrace(t, apiClient, info)

			// stop another ingester and confirm things fail
			err = tempoIngester1.Kill()
			require.NoError(t, err)

			require.Error(t, info.EmitBatches(c))
		})
	}
}

func TestScalableSingleBinary(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := e2edb.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	// copy configuration file over to shared dir
	require.NoError(t, util.CopyFileToSharedDir(s, configHA, "config.yaml"))

	// start three scalable single binary tempos in parallel
	var wg sync.WaitGroup
	var tempo1, tempo2, tempo3 *e2e.HTTPService
	wg.Add(3)
	go func() {
		tempo1 = util.NewTempoScalableSingleBinary(1)
		wg.Done()
	}()
	go func() {
		tempo2 = util.NewTempoScalableSingleBinary(2)
		wg.Done()
	}()
	go func() {
		tempo3 = util.NewTempoScalableSingleBinary(3)
		wg.Done()
	}()
	wg.Wait()
	require.NoError(t, s.StartAndWaitReady(tempo1, tempo2, tempo3))

	// wait for 2 active ingesters
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

	t.Logf("tempo1.Endpoint(): %+v", tempo1.Endpoint(3200))

	require.NoError(t, tempo1.WaitSumMetricsWithOptions(e2e.Equals(3), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(matchers...), e2e.WaitMissingMetrics))
	require.NoError(t, tempo2.WaitSumMetricsWithOptions(e2e.Equals(3), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(matchers...), e2e.WaitMissingMetrics))
	require.NoError(t, tempo3.WaitSumMetricsWithOptions(e2e.Equals(3), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(matchers...), e2e.WaitMissingMetrics))

	c1, err := util.NewJaegerGRPCClient(tempo1.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c1)

	c2, err := util.NewJaegerGRPCClient(tempo2.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c2)

	c3, err := util.NewJaegerGRPCClient(tempo3.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c3)

	info := tempoUtil.NewTraceInfo(time.Unix(1632169410, 0), "")
	require.NoError(t, info.EmitBatches(c1))

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	// test metrics
	require.NoError(t, tempo1.WaitSumMetrics(e2e.Equals(spanCount(expected)), "tempo_distributor_spans_received_total"))

	// wait trace_idle_time and ensure trace is created in ingester
	time.Sleep(1 * time.Second)
	require.NoError(t, tempo1.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

	for _, i := range []*e2e.HTTPService{tempo1, tempo2, tempo3} {
		callFlush(t, i)
		require.NoError(t, i.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
		callIngesterRing(t, i)
		callCompactorRing(t, i)
		callStatus(t, i)
		callBuildinfo(t, i)
	}

	apiClient1 := httpclient.New("http://"+tempo1.Endpoint(3200), "")

	queryAndAssertTrace(t, apiClient1, info)

	err = tempo1.Kill()
	require.NoError(t, err)

	// Push to one of the instances that are still running.
	require.NoError(t, info.EmitBatches(c2))

	err = tempo2.Kill()
	require.NoError(t, err)

	err = tempo3.Kill()
	require.NoError(t, err)
}

func makeThriftBatch() *thrift.Batch {
	return makeThriftBatchWithSpanCount(1)
}

func makeThriftBatchWithSpanCount(n int) *thrift.Batch {
	return makeThriftBatchWithSpanCountAttributeAndName(n, "my operation", "", "y")
}

func makeThriftBatchWithSpanCountAttributeAndName(n int, name, resourceTag, spanTag string) *thrift.Batch {
	var spans []*thrift.Span

	traceIDLow := rand.Int63()
	traceIDHigh := rand.Int63()
	for i := 0; i < n; i++ {
		spans = append(spans, &thrift.Span{
			TraceIdLow:    traceIDLow,
			TraceIdHigh:   traceIDHigh,
			SpanId:        rand.Int63(),
			ParentSpanId:  0,
			OperationName: name,
			References:    nil,
			Flags:         0,
			StartTime:     time.Now().UnixNano() / 1000, // microsecconds
			Duration:      1,
			Tags: []*thrift.Tag{
				{
					Key:  "x",
					VStr: &spanTag,
				},
			},
			Logs: nil,
		})
	}

	return &thrift.Batch{
		Process: &thrift.Process{
			ServiceName: "my-service",
			Tags: []*thrift.Tag{
				{
					Key:   "xx",
					VType: thrift.TagType_STRING,
					VStr:  &resourceTag,
				},
			},
		},
		Spans: spans,
	}
}

func callFlush(t *testing.T, ingester *e2e.HTTPService) {
	fmt.Printf("Calling /flush on %s\n", ingester.Name())
	res, err := e2e.DoGet("http://" + ingester.Endpoint(3200) + "/flush")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, res.StatusCode)
}

func callIngesterRing(t *testing.T, svc *e2e.HTTPService) {
	endpoint := "/ingester/ring"
	fmt.Printf("Calling %s on %s\n", endpoint, svc.Name())
	res, err := e2e.DoGet("http://" + svc.Endpoint(3200) + endpoint)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func callCompactorRing(t *testing.T, svc *e2e.HTTPService) {
	endpoint := "/compactor/ring"
	fmt.Printf("Calling %s on %s\n", endpoint, svc.Name())
	res, err := e2e.DoGet("http://" + svc.Endpoint(3200) + endpoint)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func callStatus(t *testing.T, svc *e2e.HTTPService) {
	endpoint := "/status/endpoints"
	fmt.Printf("Calling %s on %s\n", endpoint, svc.Name())
	res, err := e2e.DoGet("http://" + svc.Endpoint(3200) + endpoint)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func callBuildinfo(t *testing.T, svc *e2e.HTTPService) {
	endpoint := "/api/status/buildinfo"
	fmt.Printf("Calling %s on %s\n", endpoint, svc.Name())
	res, err := e2e.DoGet("http://" + svc.Endpoint(3200) + endpoint)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// Check that the actual JSON response contains all the expected keys (we disregard the values)
	var jsonResponse map[string]any
	keys := []string{"version", "revision", "branch", "buildDate", "buildUser", "goVersion"}
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &jsonResponse)
	require.NoError(t, err)
	for _, key := range keys {
		_, ok := jsonResponse[key]
		require.True(t, ok)
	}
	defer res.Body.Close()
}

func assertEcho(t *testing.T, url string) {
	res, err := e2e.DoGet(url)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	defer res.Body.Close()
}

func queryAndAssertTrace(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo) {
	resp, err := client.QueryTrace(info.HexID())
	require.NoError(t, err)

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	assertEqualTrace(t, resp, expected)
}

func assertEqualTrace(t *testing.T, a, b *tempopb.Trace) {
	t.Helper()
	trace.SortTraceAndAttributes(a)
	trace.SortTraceAndAttributes(b)

	assert.Equal(t, a, b)
}

func spanCount(a *tempopb.Trace) float64 {
	count := 0
	for _, batch := range a.Batches {
		for _, spans := range batch.ScopeSpans {
			count += len(spans.Spans)
		}
	}

	return float64(count)
}
