package e2e

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e_db "github.com/cortexproject/cortex/integration/e2e/db"
	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/integration/e2e/backend"
)

const (
	configMicroservices = "config-microservices.yaml"

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
			s, err := cortex_e2e.NewScenario("tempo_e2e")
			require.NoError(t, err)
			defer s.Close()

			// set up the backend
			cfg := app.Config{}
			buff, err := ioutil.ReadFile(tc.configFile)
			require.NoError(t, err)
			err = yaml.UnmarshalStrict(buff, &cfg)
			require.NoError(t, err)
			_, err = backend.New(s, cfg)
			require.NoError(t, err)

			require.NoError(t, util.CopyFileToSharedDir(s, tc.configFile, "config.yaml"))
			tempo := util.NewTempoAllInOne()
			require.NoError(t, s.StartAndWaitReady(tempo))

			// Get port for the Jaeger gRPC receiver endpoint
			c, err := newJaegerGRPCClient(tempo.Endpoint(14250))
			require.NoError(t, err)
			require.NotNil(t, c)
			batch := makeThriftBatch()
			require.NoError(t, c.EmitBatch(context.Background(), batch))

			// test metrics
			require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_distributor_spans_received_total"))

			hexID := extractHexID(batch)

			// test echo
			assertEcho(t, "http://"+tempo.Endpoint(3200)+"/api/echo")

			// ensure trace is created in ingester (trace_idle_time has passed)
			require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))

			// query an in-memory trace
			queryAndAssertTrace(t, "http://"+tempo.Endpoint(3200)+"/api/traces/"+hexID, "my operation", 1)

			// flush trace to backend
			res, err := cortex_e2e.GetRequest("http://" + tempo.Endpoint(3200) + "/flush")
			require.NoError(t, err)
			require.Equal(t, 204, res.StatusCode)

			// sleep for one maintenance cycle
			time.Sleep(5 * time.Second)

			// force clear completed block
			res, err = cortex_e2e.GetRequest("http://" + tempo.Endpoint(3200) + "/flush")
			require.NoError(t, err)
			require.Equal(t, 204, res.StatusCode)

			// test metrics
			require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
			require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempodb_blocklist_length"))
			require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_query_frontend_queries_total"))

			// query trace - should fetch from backend
			queryAndAssertTrace(t, "http://"+tempo.Endpoint(3200)+"/api/traces/"+hexID, "my operation", 1)
		})
	}
}

func TestMicroservices(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := cortex_e2e_db.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	require.NoError(t, util.CopyFileToSharedDir(s, configMicroservices, "config.yaml"))
	tempoIngester1 := util.NewTempoIngester(1)
	tempoIngester2 := util.NewTempoIngester(2)
	tempoIngester3 := util.NewTempoIngester(3)
	tempoDistributor := util.NewTempoDistributor()
	tempoQueryFrontend := util.NewTempoQueryFrontend()
	tempoQuerier := util.NewTempoQuerier()
	require.NoError(t, s.StartAndWaitReady(tempoIngester1, tempoIngester2, tempoIngester3, tempoDistributor, tempoQueryFrontend, tempoQuerier))

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
	require.NoError(t, tempoDistributor.WaitSumMetricsWithOptions(cortex_e2e.Equals(3), []string{`cortex_ring_members`}, cortex_e2e.WithLabelMatchers(matchers...)))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := newJaegerGRPCClient(tempoDistributor.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)
	batch := makeThriftBatch()

	require.NoError(t, c.EmitBatch(context.Background(), batch))

	// test metrics
	require.NoError(t, tempoDistributor.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_distributor_spans_received_total"))

	hexID := extractHexID(batch)

	// test echo
	assertEcho(t, "http://"+tempoQueryFrontend.Endpoint(3200)+"/api/echo")

	// ensure trace is created in ingester (trace_idle_time has passed)
	require.NoError(t, tempoIngester1.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))
	require.NoError(t, tempoIngester2.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))
	require.NoError(t, tempoIngester3.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))

	// query an in-memory trace
	queryAndAssertTrace(t, "http://"+tempoQueryFrontend.Endpoint(3200)+"/api/traces/"+hexID, "my operation", 1)

	// flush trace to backend
	res, err := cortex_e2e.GetRequest("http://" + tempoIngester1.Endpoint(3200) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	res, err = cortex_e2e.GetRequest("http://" + tempoIngester2.Endpoint(3200) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	res, err = cortex_e2e.GetRequest("http://" + tempoIngester3.Endpoint(3200) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	// sleep for one maintenance cycle
	time.Sleep(5 * time.Second)

	// test metrics
	for _, i := range []*cortex_e2e.HTTPService{tempoIngester1, tempoIngester2, tempoIngester3} {
		require.NoError(t, i.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	}
	require.NoError(t, tempoQuerier.WaitSumMetrics(cortex_e2e.Equals(3), "tempodb_blocklist_length"))
	require.NoError(t, tempoQueryFrontend.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_query_frontend_queries_total"))

	// query trace - should fetch from backend
	queryAndAssertTrace(t, "http://"+tempoQueryFrontend.Endpoint(3200)+"/api/traces/"+hexID, "my operation", 1)

	// stop an ingester and confirm we can still write and query
	err = tempoIngester2.Stop()
	require.NoError(t, err)

	// sleep for heartbeat timeout
	time.Sleep(1 * time.Second)

	batch = makeThriftBatch()
	require.NoError(t, c.EmitBatch(context.Background(), batch))
	hexID = extractHexID(batch)

	// query an in-memory trace
	queryAndAssertTrace(t, "http://"+tempoQueryFrontend.Endpoint(3200)+"/api/traces/"+hexID, "my operation", 1)

	// stop another ingester and confirm things fail
	err = tempoIngester1.Stop()
	require.NoError(t, err)

	batch = makeThriftBatch()
	require.Error(t, c.EmitBatch(context.Background(), batch))
}

func assertEcho(t *testing.T, url string) {
	res, err := cortex_e2e.GetRequest(url)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	defer res.Body.Close()
}

//nolint:unparam
func queryAndAssertTrace(t *testing.T, url string, expectedName string, expectedBatches int) {
	res, err := cortex_e2e.GetRequest(url)
	require.NoError(t, err)
	defer res.Body.Close()

	assertTrace(t, res.Body, expectedBatches, expectedName)
}

func newJaegerGRPCClient(endpoint string) (*jaeger_grpc.Reporter, error) {
	// new jaeger grpc exporter
	conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	return jaeger_grpc.NewReporter(conn, nil, logger), err
}
