package servicegraphs

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs/store"
	"github.com/grafana/tempo/modules/generator/registry"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	semconvnew "go.opentelemetry.io/otel/semconv/v1.34.0"
)

// NOTE: This is a way to know if the contents of the semconv package have changed.
// Since we rely on the key contents in the span attributes, we want to know if
// there is ever a change to the ones we rely on.  This is not a complete test,
// but just a quick way to know about changes upstream.
func TestSemconvKeys(t *testing.T) {
	require.Equal(t, string(semconv.DBNameKey), "db.name")
	require.Equal(t, string(semconv.DBSystemKey), "db.system")
	require.Equal(t, string(semconv.PeerServiceKey), "peer.service")
	require.Equal(t, string(semconv.NetworkPeerAddressKey), "network.peer.address")
	require.Equal(t, string(semconv.NetworkPeerPortKey), "network.peer.port")
	require.Equal(t, string(semconv.ServerAddressKey), "server.address")
	require.Equal(t, string(semconvnew.DBNamespaceKey), "db.namespace")
}

func TestServiceGraphs(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.HistogramBuckets = []float64{0.04}
	cfg.Dimensions = []string{"beast", "god"}
	cfg.EnableMessagingSystemLatencyHistogram = true

	p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-queue-database.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	requesterToServerLabels := labels.FromMap(map[string]string{
		"client": "mythical-requester",
		"server": "mythical-server",
		"beast":  "manticore",
		"god":    "zeus",
	})
	serverToDatabaseLabels := labels.FromMap(map[string]string{
		"client":          "mythical-server",
		"server":          "postgres",
		"connection_type": "database",
	})
	requesterToRecorderLabels := labels.FromMap(map[string]string{
		"client":          "mythical-requester",
		"server":          "mythical-recorder",
		"connection_type": "messaging_system",
	})

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

	p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-queue-database.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	requesterToServerLabels := labels.FromMap(map[string]string{
		"client":       "mythical-requester",
		"server":       "mythical-server",
		"client_beast": "manticore",
		"server_beast": "manticore",
		"client_god":   "ares",
		"server_god":   "zeus",
	})

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

	p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-queue-database.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	requesterToRecorderLabels := labels.FromMap(map[string]string{
		"client":          "mythical-requester",
		"server":          "mythical-recorder",
		"connection_type": "messaging_system",
	})

	// counters
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_messaging_system_seconds_count`, requesterToRecorderLabels))
}

func TestServiceGraphs_failedRequests(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-failed-requests.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	requesterToServerLabels := labels.FromMap(map[string]string{
		"client": "mythical-requester",
		"server": "mythical-server",
	})
	serverToDatabaseLabels := labels.FromMap(map[string]string{
		"client":          "mythical-server",
		"server":          "postgres",
		"connection_type": "database",
	})

	// counters
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, requesterToServerLabels))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_failed_total`, requesterToServerLabels))

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, serverToDatabaseLabels))
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_failed_total`, serverToDatabaseLabels))
}

func TestServiceGraphs_applyFilterPolicy(t *testing.T) {
	cases := []struct {
		name                     string
		filterPolicies           []filterconfig.FilterPolicy
		expectedRequesterServer  float64
		expectedServerDatabase   float64
		expectedRequesterMessage float64
	}{
		{
			name:                     "no_filters",
			filterPolicies:           nil,
			expectedRequesterServer:  1.0,
			expectedServerDatabase:   1.0,
			expectedRequesterMessage: 1.0,
		},
		{
			name: "include_requester_only",
			filterPolicies: []filterconfig.FilterPolicy{
				{
					Include: &filterconfig.PolicyMatch{
						MatchType: filterconfig.Strict,
						Attributes: []filterconfig.MatchPolicyAttribute{
							{Key: "resource.service.name", Value: "mythical-requester"},
						},
					},
				},
			},
			expectedRequesterServer:  0.0,
			expectedServerDatabase:   0.0,
			expectedRequesterMessage: 0.0,
		},
		{
			name: "include_server_only",
			filterPolicies: []filterconfig.FilterPolicy{
				{
					Include: &filterconfig.PolicyMatch{
						MatchType: filterconfig.Strict,
						Attributes: []filterconfig.MatchPolicyAttribute{
							{Key: "resource.service.name", Value: "mythical-server"},
						},
					},
				},
			},
			expectedRequesterServer:  0.0,
			expectedServerDatabase:   1.0,
			expectedRequesterMessage: 0.0,
		},
		{
			name: "exclude_requester",
			filterPolicies: []filterconfig.FilterPolicy{
				{
					Exclude: &filterconfig.PolicyMatch{
						MatchType: filterconfig.Strict,
						Attributes: []filterconfig.MatchPolicyAttribute{
							{Key: "resource.service.name", Value: "mythical-requester"},
						},
					},
				},
			},
			expectedRequesterServer:  0.0,
			expectedServerDatabase:   1.0,
			expectedRequesterMessage: 0.0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testRegistry := registry.NewTestRegistry()

			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			cfg.HistogramBuckets = []float64{0.04}
			cfg.EnableMessagingSystemLatencyHistogram = true
			cfg.FilterPolicies = tc.filterPolicies

			p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
			require.NoError(t, err)
			defer p.Shutdown(context.Background())

			request, err := loadTestData("testdata/trace-with-queue-database.json")
			require.NoError(t, err)

			p.PushSpans(context.Background(), request)

			requesterToServerLabels := labels.FromMap(map[string]string{
				"client": "mythical-requester",
				"server": "mythical-server",
			})
			serverToDatabaseLabels := labels.FromMap(map[string]string{
				"client":          "mythical-server",
				"server":          "postgres",
				"connection_type": "database",
			})
			requesterToRecorderLabels := labels.FromMap(map[string]string{
				"client":          "mythical-requester",
				"server":          "mythical-recorder",
				"connection_type": "messaging_system",
			})

			assert.Equal(t, tc.expectedRequesterServer, testRegistry.Query(`traces_service_graph_request_total`, requesterToServerLabels))
			assert.Equal(t, tc.expectedServerDatabase, testRegistry.Query(`traces_service_graph_request_total`, serverToDatabaseLabels))
			assert.Equal(t, tc.expectedRequesterMessage, testRegistry.Query(`traces_service_graph_request_total`, requesterToRecorderLabels))
		})
	}
}

func TestServiceGraphs_tooManySpansErr(t *testing.T) {
	testRegistry := registry.TestRegistry{}

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.MaxItems = 1
	p, err := New(cfg, "test", &testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
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

	p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
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

	virtualProducerToConsumer := labels.FromMap(map[string]string{
		"client":          "external-producer",
		"server":          "internal-consumer",
		"connection_type": "virtual_node",
	})

	// counters
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, userToServerLabels))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, userToServerLabels))

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, clientToVirtualPeerLabels))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, clientToVirtualPeerLabels))

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, virtualProducerToConsumer))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, virtualProducerToConsumer))
}

func TestServiceGraphs_virtualNodesExtraLabelsForUninstrumentedServices(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.EnableVirtualNodeLabel = true
	cfg.Wait = time.Nanosecond

	p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-virtual-nodes.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	p.(*Processor).store.Expire()

	userToServerLabels := labels.FromMap(map[string]string{
		"client":          "user",
		"server":          "mythical-server",
		"connection_type": "virtual_node",
		virtualNodeLabel:  "client",
	})

	clientToVirtualPeerLabels := labels.FromMap(map[string]string{
		"client":          "mythical-requester",
		"server":          "external-payments-platform",
		"connection_type": "virtual_node",
		virtualNodeLabel:  "server",
	})

	// counters
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, userToServerLabels))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, userToServerLabels))

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, clientToVirtualPeerLabels))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, clientToVirtualPeerLabels))
}

func TestServiceGraphs_expiredEdges(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.EnableVirtualNodeLabel = true
	cfg.Wait = time.Nanosecond

	const tenant = "expired-edge-test"

	p, err := New(cfg, tenant, testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	/*
		1 unmatched edge - root span with type server
		1 matched edge - client/server spans
		1 unmatched edge - client span with a db name
		1 unmatched edge - server span. this should count as expired!
	*/
	request, err := loadTestData("testdata/trace-with-expired-edges.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	p.(*Processor).store.Expire()

	expiredEdges, err := test.GetCounterVecValue(metricExpiredEdges, tenant)
	require.NoError(t, err)
	assert.Equal(t, 1.0, expiredEdges)

	totalEdges, err := test.GetCounterVecValue(metricTotalEdges, tenant)
	require.NoError(t, err)
	assert.Equal(t, 4.0, totalEdges)

	droppedSpans, err := test.GetCounterVecValue(metricDroppedSpans, tenant)
	require.NoError(t, err)
	assert.Equal(t, 0.0, droppedSpans)
}

func TestServiceGraphs_droppedEdgesMetric(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	const tenant = "dropped-edge-test"

	p, err := New(cfg, tenant, testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	traceID := []byte{0x01}
	spanID := []byte{0x02}
	key := buildKey(hex.EncodeToString(traceID), hex.EncodeToString(spanID))
	p.(*Processor).store.AddDroppedSpanSide(key, store.Server)

	request := &tempopb.PushSpansRequest{
		Batches: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*v1.KeyValue{
						{
							Key: "service.name",
							Value: &v1.AnyValue{
								Value: &v1.AnyValue_StringValue{StringValue: "svc-a"},
							},
						},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Spans: []*tracev1.Span{
							{
								TraceId:           traceID,
								SpanId:            spanID,
								Kind:              tracev1.Span_SPAN_KIND_CLIENT,
								StartTimeUnixNano: 1,
								EndTimeUnixNano:   2,
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), request)

	droppedEdges, err := test.GetCounterVecValue(metricDroppedEdges, tenant)
	require.NoError(t, err)
	assert.Equal(t, 1.0, droppedEdges)
}

func TestServiceGraphs_droppedEdgesMetric_fromFilteredCounterpart(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.FilterPolicies = []filterconfig.FilterPolicy{
		{
			Exclude: &filterconfig.PolicyMatch{
				MatchType: filterconfig.Strict,
				Attributes: []filterconfig.MatchPolicyAttribute{
					{Key: "resource.service.name", Value: "svc-a"},
				},
			},
		},
	}

	const tenant = "dropped-edge-filtered-counterpart-test"

	p, err := New(cfg, tenant, testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	traceID := []byte{0x01}
	clientSpanID := []byte{0x02}

	request := &tempopb.PushSpansRequest{
		Batches: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*v1.KeyValue{
						{
							Key: "service.name",
							Value: &v1.AnyValue{
								Value: &v1.AnyValue_StringValue{StringValue: "svc-a"},
							},
						},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Spans: []*tracev1.Span{
							{
								TraceId:           traceID,
								SpanId:            clientSpanID,
								Kind:              tracev1.Span_SPAN_KIND_CLIENT,
								StartTimeUnixNano: 1,
								EndTimeUnixNano:   2,
							},
						},
					},
				},
			},
			{
				Resource: &resourcev1.Resource{
					Attributes: []*v1.KeyValue{
						{
							Key: "service.name",
							Value: &v1.AnyValue{
								Value: &v1.AnyValue_StringValue{StringValue: "svc-b"},
							},
						},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Spans: []*tracev1.Span{
							{
								TraceId:           traceID,
								ParentSpanId:      clientSpanID,
								SpanId:            []byte{0x03},
								Kind:              tracev1.Span_SPAN_KIND_SERVER,
								StartTimeUnixNano: 3,
								EndTimeUnixNano:   4,
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), request)

	droppedEdges, err := test.GetCounterVecValue(metricDroppedEdges, tenant)
	require.NoError(t, err)
	assert.Equal(t, 1.0, droppedEdges)
}

func TestServiceGraphs_droppedEdgesMetric_whenFilteredSpanDropsBufferedCounterpart(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.FilterPolicies = []filterconfig.FilterPolicy{
		{
			Exclude: &filterconfig.PolicyMatch{
				MatchType: filterconfig.Strict,
				Attributes: []filterconfig.MatchPolicyAttribute{
					{Key: "resource.service.name", Value: "svc-a"},
				},
			},
		},
	}

	const tenant = "dropped-edge-filtered-buffered-counterpart-test"

	p, err := New(cfg, tenant, testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	traceID := []byte{0x11}
	clientSpanID := []byte{0x22}

	request := &tempopb.PushSpansRequest{
		Batches: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*v1.KeyValue{
						{
							Key: "service.name",
							Value: &v1.AnyValue{
								Value: &v1.AnyValue_StringValue{StringValue: "svc-b"},
							},
						},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Spans: []*tracev1.Span{
							{
								TraceId:           traceID,
								ParentSpanId:      clientSpanID,
								SpanId:            []byte{0x33},
								Kind:              tracev1.Span_SPAN_KIND_SERVER,
								StartTimeUnixNano: 3,
								EndTimeUnixNano:   4,
							},
						},
					},
				},
			},
			{
				Resource: &resourcev1.Resource{
					Attributes: []*v1.KeyValue{
						{
							Key: "service.name",
							Value: &v1.AnyValue{
								Value: &v1.AnyValue_StringValue{StringValue: "svc-a"},
							},
						},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Spans: []*tracev1.Span{
							{
								TraceId:           traceID,
								SpanId:            clientSpanID,
								Kind:              tracev1.Span_SPAN_KIND_CLIENT,
								StartTimeUnixNano: 1,
								EndTimeUnixNano:   2,
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), request)

	droppedEdges, err := test.GetCounterVecValue(metricDroppedEdges, tenant)
	require.NoError(t, err)
	assert.Equal(t, 1.0, droppedEdges)
}

func TestServiceGraphs_filteredRootServerSpanDoesNotAddDroppedCounterpart(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.FilterPolicies = []filterconfig.FilterPolicy{
		{
			Exclude: &filterconfig.PolicyMatch{
				MatchType: filterconfig.Strict,
				Attributes: []filterconfig.MatchPolicyAttribute{
					{Key: "resource.service.name", Value: "svc-a"},
				},
			},
		},
	}

	p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	traceID := []byte{0x01}
	request := &tempopb.PushSpansRequest{
		Batches: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*v1.KeyValue{
						{
							Key: "service.name",
							Value: &v1.AnyValue{
								Value: &v1.AnyValue_StringValue{StringValue: "svc-a"},
							},
						},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Spans: []*tracev1.Span{
							{
								TraceId:           traceID,
								Kind:              tracev1.Span_SPAN_KIND_SERVER,
								StartTimeUnixNano: 1,
								EndTimeUnixNano:   2,
							},
						},
					},
				},
			},
		},
	}

	p.PushSpans(context.Background(), request)

	emptyParentKey := buildKey(hex.EncodeToString(traceID), "")
	assert.False(t, p.(*Processor).store.HasDroppedSpanSide(emptyParentKey, store.Server))
}

func TestServiceGraphs_databaseVirtualNodes(t *testing.T) {
	cases := []struct {
		name           string
		fixturePath    string
		databaseLabels labels.Labels
		total          float64
		errors         float64
	}{
		{
			name:        "virtualNodesWithoutDatabase",
			fixturePath: "testdata/trace-with-virtual-nodes.json",
			databaseLabels: labels.FromMap(map[string]string{
				"client":          "mythical-server",
				"server":          "mythical-database",
				"connection_type": "database",
			}),
			total:  0.0,
			errors: 0.0,
		},
		{
			name:        "withoutDatabaseName",
			fixturePath: "testdata/trace-without-database-name.json",
			databaseLabels: labels.FromMap(map[string]string{
				"client":          "mythical-server",
				"server":          "mythical-database",
				"connection_type": "database",
			}),
			total:  1.0,
			errors: 0.0,
		},
		{
			name:        "semconv118",
			fixturePath: "testdata/trace-with-queue-database.json",
			databaseLabels: labels.FromMap(map[string]string{
				"client":          "mythical-server",
				"server":          "postgres",
				"connection_type": "database",
			}),
			total:  1.0,
			errors: 0.0,
		},
		{
			name:        "semconv125",
			fixturePath: "testdata/trace-with-queue-database2.json",
			databaseLabels: labels.FromMap(map[string]string{
				"client":          "mythical-server",
				"server":          "mythical-database",
				"connection_type": "database",
			}),
			total:  1.0,
			errors: 0.0,
		},
		{
			name:        "semconv125PeerService",
			fixturePath: "testdata/trace-with-queue-database3.json",
			databaseLabels: labels.FromMap(map[string]string{
				"client":          "mythical-server",
				"server":          "mythical-database",
				"connection_type": "database",
			}),
			total:  1.0,
			errors: 0.0,
		},
		{
			name:        "semconv125NetworkPeerWithPort",
			fixturePath: "testdata/trace-with-queue-database4.json",
			databaseLabels: labels.FromMap(map[string]string{
				"client":          "mythical-server",
				"server":          "mythical-database:5432",
				"connection_type": "database",
			}),
			total:  1.0,
			errors: 0.0,
		},
		{
			name:        "semconv125NetworkPeerWithoutPort",
			fixturePath: "testdata/trace-with-queue-database5.json",
			databaseLabels: labels.FromMap(map[string]string{
				"client":          "mythical-server",
				"server":          "mythical-database",
				"connection_type": "database",
			}),
			total:  1.0,
			errors: 0.0,
		},
		{
			name:        "dbNamespaceAttribute",
			fixturePath: "testdata/trace-with-db-namespace.json",
			databaseLabels: labels.FromMap(map[string]string{
				"client":          "mythical-server",
				"server":          "mydb",
				"connection_type": "database",
			}),
			total:  1.0,
			errors: 0.0,
		},
		{
			name:        "bothDbNameAndNamespace",
			fixturePath: "testdata/trace-with-both-db-attributes.json",
			databaseLabels: labels.FromMap(map[string]string{
				"client":          "mythical-server",
				"server":          "priority-db",
				"connection_type": "database",
			}),
			total:  1.0,
			errors: 0.0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testRegistry := registry.NewTestRegistry()

			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)

			cfg.HistogramBuckets = []float64{0.04}
			cfg.EnableMessagingSystemLatencyHistogram = true

			p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
			require.NoError(t, err)
			defer p.Shutdown(context.Background())

			request, err := loadTestData(tc.fixturePath)
			require.NoError(t, err)

			p.PushSpans(context.Background(), request)

			// counters
			assert.Equal(t, tc.total, testRegistry.Query(`traces_service_graph_request_total`, tc.databaseLabels))
			assert.Equal(t, tc.errors, testRegistry.Query(`traces_service_graph_request_failed_total`, tc.databaseLabels))

			// histograms
			assert.Equal(t, tc.total, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(tc.databaseLabels, 0.04)))
			assert.Equal(t, tc.total, testRegistry.Query(`traces_service_graph_request_client_seconds_bucket`, withLe(tc.databaseLabels, math.Inf(1))))
			assert.Equal(t, tc.total, testRegistry.Query(`traces_service_graph_request_client_seconds_count`, tc.databaseLabels))
			// assert.InDelta(t, 0.023, testRegistry.Query(`traces_service_graph_request_client_seconds_sum`, tc.databaseLabels), 0.001)

			assert.Equal(t, tc.total, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(tc.databaseLabels, 0.04)))
			assert.Equal(t, tc.total, testRegistry.Query(`traces_service_graph_request_server_seconds_bucket`, withLe(tc.databaseLabels, math.Inf(1))))
			assert.Equal(t, tc.total, testRegistry.Query(`traces_service_graph_request_server_seconds_count`, tc.databaseLabels))
			// assert.InDelta(t, 0.023, testRegistry.Query(`traces_service_graph_request_server_seconds_sum`, tc.databaseLabels), 0.001)
		})
	}
}

func TestServiceGraphs_prefixDimensionsAndEnableExtraLabels(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.HistogramBuckets = []float64{0.04}
	cfg.Dimensions = []string{"db.system", "messaging.system"}
	cfg.EnableClientServerPrefix = true
	cfg.EnableVirtualNodeLabel = true

	p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-queue-database.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	messagingSystemLabels := labels.FromMap(map[string]string{
		"client":                  "mythical-requester",
		"client_messaging_system": "rabbitmq",
		"connection_type":         "messaging_system",
		"server_messaging_system": "rabbitmq",
		"server":                  "mythical-recorder",
	})

	dbSystemSystemLabels := labels.FromMap(map[string]string{
		"client":           "mythical-server",
		"client_db_system": "postgresql",
		"connection_type":  "database",
		"server":           "postgres",
	})

	// counters
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, messagingSystemLabels))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, messagingSystemLabels))

	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, dbSystemSystemLabels))
	assert.Equal(t, 0.0, testRegistry.Query(`traces_service_graph_request_failed_total`, dbSystemSystemLabels))
}

func TestServiceGraphs_DatabaseNameAttributes(t *testing.T) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.HistogramBuckets = []float64{0.04}
	cfg.Dimensions = []string{"beast", "god"}
	cfg.DatabaseNameAttributes = []string{"db.system"}

	p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-queue-database.json")
	require.NoError(t, err)

	p.PushSpans(context.Background(), request)

	// The server label should be set to the value of db.system
	labels := labels.FromMap(map[string]string{
		"client":          "mythical-server",
		"server":          "postgresql",
		"connection_type": "database",
		"beast":           "",
		"god":             "",
	})
	assert.Equal(t, 1.0, testRegistry.Query(`traces_service_graph_request_total`, labels))
}

func BenchmarkServiceGraphs(b *testing.B) {
	testRegistry := registry.NewTestRegistry()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	cfg.HistogramBuckets = []float64{0.04}
	cfg.Dimensions = []string{"beast", "god"}

	p, err := New(cfg, "test", testRegistry, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(b, err)
	defer p.Shutdown(context.Background())

	request, err := loadTestData("testdata/trace-with-queue-database.json")
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		p.PushSpans(context.Background(), request)
	}
}

func loadTestData(path string) (*tempopb.PushSpansRequest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	trace := &tempopb.Trace{}
	err = jsonpb.Unmarshal(f, trace)
	return &tempopb.PushSpansRequest{Batches: trace.ResourceSpans}, err
}

func withLe(lbls labels.Labels, le float64) labels.Labels {
	lb := labels.NewBuilder(lbls)
	lb = lb.Set(labels.BucketLabel, strconv.FormatFloat(le, 'f', -1, 64))
	return lb.Labels()
}
