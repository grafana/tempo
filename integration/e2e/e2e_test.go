package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	"github.com/grafana/tempo/cmd/tempo/app"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"gopkg.in/yaml.v2"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e_db "github.com/cortexproject/cortex/integration/e2e/db"
	"github.com/gogo/protobuf/jsonpb"
	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	configAllInOne      = "config-all-in-one.yaml"
	configMicroservices = "config-microservices.yaml"

	configAllInOneAzurite = "config-all-in-one-azurite.yaml"
)

func TestAllInOne(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := cortex_e2e_db.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOne, "config.yaml"))
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

	hexID := fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)

	// test echo
	assertEcho(t, "http://"+tempo.Endpoint(3100)+"/api/echo")

	// ensure trace is created in ingester (trace_idle_time has passed)
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))

	// query an in-memory trace
	queryAndAssertTrace(t, "http://"+tempo.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)

	// flush trace to backend
	res, err := cortex_e2e.GetRequest("http://" + tempo.Endpoint(3100) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	// sleep for one maintenance cycle
	time.Sleep(5 * time.Second)

	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempodb_blocklist_length"))
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_query_frontend_queries_total"))

	// query trace - should fetch from backend
	queryAndAssertTrace(t, "http://"+tempo.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)
}

func TestAzuriteAllInOne(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	azurite := util.NewAzurite()
	require.NoError(t, s.StartAndWaitReady(azurite))
	ctx := context.TODO()

	//Create the azurite container for tempo
	cfg := app.Config{}
	buff, err := ioutil.ReadFile(configAllInOneAzurite)
	require.NoError(t, err)
	err = yaml.UnmarshalStrict(buff, &cfg)
	require.NoError(t, err)
	cfg.StorageConfig.Trace.Azure.Endpoint = azurite.Endpoint(10000)
	_, err = azure.CreateContainer(ctx, cfg.StorageConfig.Trace.Azure)
	require.NoError(t, err)

	//Start tempo
	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneAzurite, "config.yaml"))
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

	hexID := fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)

	// test echo
	assertEcho(t, "http://"+tempo.Endpoint(3100)+"/api/echo")

	// ensure trace is created in ingester (trace_idle_time has passed)
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))

	// query an in-memory trace
	queryAndAssertTrace(t, "http://"+tempo.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)

	// flush trace to backend
	res, err := cortex_e2e.GetRequest("http://" + tempo.Endpoint(3100) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	// sleep for one maintenance cycle
	time.Sleep(5 * time.Second)

	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempodb_blocklist_length"))

	// query trace - should fetch from backend
	queryAndAssertTrace(t, "http://"+tempo.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)

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

	hexID := fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)

	// test echo
	assertEcho(t, "http://"+tempoQueryFrontend.Endpoint(3100)+"/api/echo")

	// ensure trace is created in ingester (trace_idle_time has passed)
	require.NoError(t, tempoIngester1.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))
	require.NoError(t, tempoIngester2.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))
	require.NoError(t, tempoIngester3.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))

	// query an in-memory trace
	queryAndAssertTrace(t, "http://"+tempoQueryFrontend.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)

	// flush trace to backend
	res, err := cortex_e2e.GetRequest("http://" + tempoIngester1.Endpoint(3100) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	res, err = cortex_e2e.GetRequest("http://" + tempoIngester2.Endpoint(3100) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	// sleep for one maintenance cycle
	time.Sleep(5 * time.Second)

	// test metrics
	for _, i := range []*cortex_e2e.HTTPService{tempoIngester1, tempoIngester2, tempoIngester3} {
		require.NoError(t, i.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
		require.NoError(t, i.WaitSumMetrics(cortex_e2e.Equals(3), "tempodb_blocklist_length"))
	}
	require.NoError(t, tempoQueryFrontend.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_query_frontend_queries_total"))

	// query trace - should fetch from backend
	queryAndAssertTrace(t, "http://"+tempoQueryFrontend.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)

	// stop an ingester and confirm we can still write and query
	err = tempoIngester2.Stop()
	require.NoError(t, err)

	// sleep for heartbeat timeout
	time.Sleep(1 * time.Second)

	batch = makeThriftBatch()
	require.NoError(t, c.EmitBatch(context.Background(), batch))
	hexID = fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)

	// query an in-memory trace
	queryAndAssertTrace(t, "http://"+tempoQueryFrontend.Endpoint(3100)+"/api/traces/"+hexID, "my operation", 1)

	// stop another ingester and confirm things fail
	err = tempoIngester1.Stop()
	require.NoError(t, err)

	batch = makeThriftBatch()
	require.Error(t, c.EmitBatch(context.Background(), batch))
}

func makeThriftBatch() *thrift.Batch {
	return makeThriftBatchWithSpanCount(1)
}

func makeThriftBatchWithSpanCount(n int) *thrift.Batch {
	var spans []*thrift.Span

	traceIDLow := rand.Int63()
	traceIDHigh := rand.Int63()
	for i := 0; i < n; i++ {
		spans = append(spans, &thrift.Span{
			TraceIdLow:    traceIDLow,
			TraceIdHigh:   traceIDHigh,
			SpanId:        rand.Int63(),
			ParentSpanId:  0,
			OperationName: "my operation",
			References:    nil,
			Flags:         0,
			StartTime:     time.Now().Unix(),
			Duration:      1,
			Tags:          nil,
			Logs:          nil,
		})
	}
	return &thrift.Batch{Spans: spans}
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
	out := &tempopb.Trace{}
	unmarshaller := &jsonpb.Unmarshaler{}
	require.NoError(t, unmarshaller.Unmarshal(res.Body, out))
	require.Len(t, out.Batches, expectedBatches)
	assert.Equal(t, expectedName, out.Batches[0].InstrumentationLibrarySpans[0].Spans[0].Name)
	defer res.Body.Close()
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
