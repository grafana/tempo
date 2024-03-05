package servicegraphs

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
)

func TestServiceGraphs(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.HistogramBuckets = []float64{0.04}
	cfg.Dimensions = []string{"beast", "god"}
	cfg.EnableMessagingSystemLatencyHistogram = true

	p := New(cfg, "test", testRegistry, log.NewNopLogger())
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-queue-database.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	requesterToServerLabels := labels.FromMap(map[string]string{
		"client":          "mythical-requester",
		"server":          "mythical-server",
		"connection_type": "",
		"beast":           "manticore",
		"god":             "zeus",
	})
	serverToDatabaseLabels := labels.FromMap(map[string]string{
		"client":          "mythical-server",
		"server":          "postgres",
		"connection_type": "database",
		"beast":           "",
		"god":             "",
	})
	requesterToRecorderLabels := labels.FromMap(map[string]string{
		"client":          "mythical-requester",
		"server":          "mythical-recorder",
		"connection_type": "messaging_system",
		"beast":           "",
		"god":             "",
	})

	fmt.Println(testRegistry)

	// counters
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, requesterToServerLabels))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, requesterToServerLabels))

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, serverToDatabaseLabels))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, serverToDatabaseLabels))

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, requesterToRecorderLabels))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, requesterToRecorderLabels))

	// histograms
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(requesterToServerLabels, 0.04)))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(requesterToServerLabels, math.Inf(1))))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_client_seconds_count`, requesterToServerLabels))
	assert.InDelta(t, 0.045, testRegistry.Query(`traces_service_graph_request_client_seconds_sum`, requesterToServerLabels), 0.001)

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(requesterToServerLabels, 0.04)))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(requesterToServerLabels, math.Inf(1))))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_server_seconds_count`, requesterToServerLabels))
	assert.InDelta(t, 0.029, testRegistry.Query(`traces_service_graph_request_server_seconds_sum`, requesterToServerLabels), 0.001)

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(serverToDatabaseLabels, 0.04)))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(serverToDatabaseLabels, math.Inf(1))))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_client_seconds_count`, serverToDatabaseLabels))
	assert.InDelta(t, 0.023, testRegistry.Query(`traces_service_graph_request_client_seconds_sum`, serverToDatabaseLabels), 0.001)

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(serverToDatabaseLabels, 0.04)))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(serverToDatabaseLabels, math.Inf(1))))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_server_seconds_count`, serverToDatabaseLabels))
	assert.InDelta(t, 0.023, testRegistry.Query(`traces_service_graph_request_server_seconds_sum`, serverToDatabaseLabels), 0.001)

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(requesterToRecorderLabels, 0.04)))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(requesterToRecorderLabels, math.Inf(1))))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_client_seconds_count`, requesterToRecorderLabels))
	assert.InDelta(t, 0.000068, testRegistry.Query(`traces_service_graph_request_client_seconds_sum`, requesterToRecorderLabels), 0.001)

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(requesterToRecorderLabels, 0.04)))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(requesterToRecorderLabels, math.Inf(1))))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_server_seconds_count`, requesterToRecorderLabels))
	assert.InDelta(t, 0.000693, testRegistry.Query(`traces_service_graph_request_server_seconds_sum`, requesterToRecorderLabels), 0.001)

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_messaging_system_seconds_bucket`, withLe(requesterToRecorderLabels, 0.04)))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_messaging_system_seconds_bucket`, withLe(requesterToRecorderLabels, math.Inf(1))))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_messaging_system_seconds_count`, requesterToRecorderLabels))
	assert.Equal(t, 0.0098816, testRegistry.Query(`traces_service_graph_request_messaging_system_seconds_sum`, requesterToRecorderLabels))
}

func TestServiceGraphs_prefixDimensions(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.HistogramBuckets = []float64{0.04}
	cfg.Dimensions = []string{"beast", "god"}
	cfg.EnableClientServerPrefix = true

	p := New(cfg, "test", testRegistry, log.NewNopLogger())
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-queue-database.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	requesterToServerLabels := labels.FromMap(map[string]string{
		"client":          "mythical-requester",
		"server":          "mythical-server",
		"connection_type": "",
		"client_beast":    "manticore",
		"server_beast":    "manticore",
		"client_god":      "ares",
		"server_god":      "zeus",
	})

	fmt.Println(testRegistry)

	// counters
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, requesterToServerLabels))
}

func TestServiceGraphs_MessagingSystemLatencyHistogram(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.HistogramBuckets = []float64{0.04}
	cfg.Dimensions = []string{"beast", "god"}
	cfg.EnableMessagingSystemLatencyHistogram = true

	p := New(cfg, "test", testRegistry, log.NewNopLogger())
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-queue-database.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	requesterToRecorderLabels := labels.FromMap(map[string]string{
		"client":          "mythical-requester",
		"server":          "mythical-recorder",
		"connection_type": "messaging_system",
		"beast":           "",
		"god":             "",
	})

	fmt.Println(testRegistry)

	// counters
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_messaging_system_seconds_count`, requesterToRecorderLabels))
}

func TestServiceGraphs_failedRequests(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	p := New(cfg, "test", testRegistry, log.NewNopLogger())
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-failed-requests.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	requesterToServerLabels := labels.FromMap(map[string]string{
		"client":          "mythical-requester",
		"server":          "mythical-server",
		"connection_type": "",
	})
	serverToDatabaseLabels := labels.FromMap(map[string]string{
		"client":          "mythical-server",
		"server":          "postgres",
		"connection_type": "database",
	})

	fmt.Println(testRegistry)

	// counters
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, requesterToServerLabels))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_failed_total`, requesterToServerLabels))

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, serverToDatabaseLabels))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_failed_total`, serverToDatabaseLabels))
}

func TestServiceGraphs_tooManySpansErr(t *testing.T) {
	testRegistry := registry.TestRegistry{}

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.MaxItems = 1
	p := New(cfg, "test", &testRegistry, log.NewNopLogger())
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-queue-database.json")
	require.NoError(t, err)

	err = p.(*Processor).consume(request.Batches)
	var tmsErr *tooManySpansError
	assert.True(t, errors.As(err, &tmsErr))
}

func TestServiceGraphs_virtualNodes(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.HistogramBuckets = []float64{0.04}
	cfg.Wait = time.Nanosecond

	p := New(cfg, "test", testRegistry, log.NewNopLogger())
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-virtual-nodes.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	p.(*Processor).store.Expire()

	userToServerLabels := labels.FromMap(map[string]string{
		"client":          "user",
		"server":          "mythical-server",
		"connection_type": "virtual_node",
	})

	clientToVirtualPeerLabels := labels.FromMap(map[string]string{
		"client":          "mythical-requester",
		"server":          "external-payments-platform",
		"connection_type": "virtual_node",
	})

	// counters
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, userToServerLabels))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, userToServerLabels))

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, clientToVirtualPeerLabels))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, clientToVirtualPeerLabels))
}

func loadTestData(path string) (*tempopb.PushSpansRequest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	trace := &tempopb.Trace{}
	err = jsonpb.Unmarshal(f, trace)
	return &tempopb.PushSpansRequest{Batches: trace.Batches}, err
}

func withLe(lbls labels.Labels, le float64) labels.Labels {
	lb := labels.NewBuilder(lbls)
	lb = lb.Set(labels.BucketLabel, strconv.FormatFloat(le, 'f', -1, 64))
	return lb.Labels()
}
