package e2e

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e_db "github.com/cortexproject/cortex/integration/e2e/db"
	"github.com/golang/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	svcName = "tempo"
	image   = "tempo:latest"

	configAllInOne = "./all-in-one-config.yaml"
)

func NewTempoAllInOne() (*cortex_e2e.HTTPService, error) {
	args := "-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml")

	return cortex_e2e.NewHTTPService(
		svcName,
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 505),
		3100,
		14250,
	), nil
}

func NewJaegerGRPCClient(endpoint string) (*jaeger_grpc.Reporter, error) {
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

func MakeThriftBatch() *thrift.Batch {
	var spans []*thrift.Span
	spans = append(spans, &thrift.Span{
		TraceIdLow:    rand.Int63(),
		TraceIdHigh:   0,
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
	return &thrift.Batch{Spans: spans}
}

func TestAllInOne(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := cortex_e2e_db.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	require.NoError(t, copyFileToSharedDir(s, configAllInOne, "config.yaml"))
	tempo, err := NewTempoAllInOne()
	require.NoError(t, err)
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Get port for the otlp receiver endpoint
	c, err := NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)
	batch := MakeThriftBatch()
	require.NoError(t, c.EmitBatch(context.Background(), batch))

	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_distributor_spans_received_total"))

	hexID := fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)

	// ensure trace is created in ingester (trace_idle_time has passed)
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_traces_created_total"))

	// query an in-memory trace
	res, err := cortex_e2e.GetRequest("http://" + tempo.Endpoint(3100) + "/api/traces/" + hexID)
	require.NoError(t, err)
	out := &tempopb.Trace{}
	unmarshaller := &jsonpb.Unmarshaler{}
	assert.NoError(t, unmarshaller.Unmarshal(res.Body, out))
	assert.Len(t, out.Batches, 1)
	assert.Equal(t, out.Batches[0].InstrumentationLibrarySpans[0].Spans[0].Name, "my operation")
	defer res.Body.Close()

	// flush trace to backend
	res, err = cortex_e2e.GetRequest("http://" + tempo.Endpoint(3100) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	// sleep for one maintenance cycle
	time.Sleep(2 * time.Second)

	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempo.WaitSumMetrics(cortex_e2e.Equals(1), "tempodb_blocklist_length"))

	// query trace - should fetch from backend
	resp, err := cortex_e2e.GetRequest("http://" + tempo.Endpoint(3100) + "/api/traces/" + hexID)
	require.NoError(t, err)
	assert.NoError(t, unmarshaller.Unmarshal(resp.Body, out))
	assert.Len(t, out.Batches, 1)
	assert.Equal(t, out.Batches[0].InstrumentationLibrarySpans[0].Spans[0].Name, "my operation")
	defer resp.Body.Close()
}
