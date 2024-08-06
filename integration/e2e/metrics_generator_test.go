package e2e

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/v2/integration/util"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configMetricsGenerator           = "config-metrics-generator.yaml"
	configMetricsGeneratorTargetInfo = "config-metrics-generator-targetinfo.yaml"
	prometheusImage                  = "prom/prometheus:latest"
)

func TestMetricsGenerator(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configMetricsGenerator, "config.yaml"))
	tempoDistributor := util.NewTempoDistributor()
	tempoIngester := util.NewTempoIngester(1)
	tempoMetricsGenerator := util.NewTempoMetricsGenerator()
	prometheus := newPrometheus()
	require.NoError(t, s.StartAndWaitReady(tempoDistributor, tempoIngester, tempoMetricsGenerator, prometheus))

	// Wait until ingester and metrics-generator are active
	isServiceActiveMatcher := func(service string) []*labels.Matcher {
		return []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "name", service),
			labels.MustNewMatcher(labels.MatchEqual, "state", "ACTIVE"),
		}
	}
	require.NoError(t, tempoDistributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(isServiceActiveMatcher("ingester")...), e2e.WaitMissingMetrics))
	require.NoError(t, tempoDistributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(isServiceActiveMatcher("metrics-generator")...), e2e.WaitMissingMetrics))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerGRPCClient(tempoDistributor.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	// Send two spans that have a client-server relationship
	r := rand.New(rand.NewSource(time.Now().UnixMilli()))
	traceIDLow := r.Int63()
	traceIDHigh := r.Int63()
	parentSpanID := r.Int63()

	err = c.EmitBatch(context.Background(), &thrift.Batch{
		Process: &thrift.Process{ServiceName: "lb"},
		Spans: []*thrift.Span{
			{
				TraceIdLow:    traceIDLow,
				TraceIdHigh:   traceIDHigh,
				SpanId:        parentSpanID,
				ParentSpanId:  0,
				OperationName: "lb-get",
				StartTime:     time.Now().UnixMicro(),
				Duration:      int64(2 * time.Second / time.Microsecond),
				Tags:          []*thrift.Tag{{Key: "span.kind", VStr: stringPtr("client")}},
			},
		},
	})
	require.NoError(t, err)

	err = c.EmitBatch(context.Background(), &thrift.Batch{
		Process: &thrift.Process{ServiceName: "app"},
		Spans: []*thrift.Span{
			{
				TraceIdLow:    traceIDLow,
				TraceIdHigh:   traceIDHigh,
				SpanId:        r.Int63(),
				ParentSpanId:  parentSpanID,
				OperationName: "app-handle",
				StartTime:     time.Now().UnixMicro(),
				Duration:      int64(1 * time.Second / time.Microsecond),
				Tags:          []*thrift.Tag{{Key: "span.kind", VStr: stringPtr("server")}},
			},
		},
	})
	require.NoError(t, err)

	// also send one with 5 minutes old timestamp
	err = c.EmitBatch(context.Background(), &thrift.Batch{
		Process: &thrift.Process{ServiceName: "app"},
		Spans: []*thrift.Span{
			{
				TraceIdLow:    traceIDLow,
				TraceIdHigh:   traceIDHigh,
				SpanId:        r.Int63(),
				ParentSpanId:  parentSpanID,
				OperationName: "app-handle",
				StartTime:     time.Now().Add(-5 * time.Minute).UnixMicro(),
				Duration:      int64(1 * time.Second / time.Microsecond),
				Tags:          []*thrift.Tag{{Key: "span.kind", VStr: stringPtr("server")}},
			},
		},
	})
	require.NoError(t, err)

	// also send one with timestamp 10 days in the future
	err = c.EmitBatch(context.Background(), &thrift.Batch{
		Process: &thrift.Process{ServiceName: "app"},
		Spans: []*thrift.Span{
			{
				TraceIdLow:    traceIDLow,
				TraceIdHigh:   traceIDHigh,
				SpanId:        r.Int63(),
				ParentSpanId:  parentSpanID,
				OperationName: "app-handle",
				StartTime:     time.Now().Add(10 * 24 * time.Hour).UnixMicro(),
				Duration:      int64(1 * time.Second / time.Microsecond),
				Tags:          []*thrift.Tag{{Key: "span.kind", VStr: stringPtr("server")}},
			},
		},
	})
	require.NoError(t, err)

	// Fetch metrics from Prometheus once they are received
	var metricFamilies map[string]*io_prometheus_client.MetricFamily
	for {
		metricFamilies, err = extractMetricsFromPrometheus(prometheus, `{__name__=~"traces_.+"}`)
		require.NoError(t, err)
		if len(metricFamilies) > 0 {
			break
		}
		time.Sleep(time.Second)
	}

	// Print collected metrics for easier debugging
	fmt.Println()
	for key, family := range metricFamilies {
		fmt.Println(key)
		for _, metric := range family.Metric {
			fmt.Println(metric)
		}
	}
	fmt.Println()

	// Service graphs
	lbls := []string{"client", "lb", "server", "app"}
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_total", lbls))

	assert.Equal(t, 0.0, sumValues(metricFamilies, "traces_service_graph_request_client_seconds_bucket", append(lbls, "le", "1")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_client_seconds_bucket", append(lbls, "le", "2")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_client_seconds_bucket", append(lbls, "le", "+Inf")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_client_seconds_count", lbls))
	assert.Equal(t, 2.0, sumValues(metricFamilies, "traces_service_graph_request_client_seconds_sum", lbls))

	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_server_seconds_bucket", append(lbls, "le", "1")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_server_seconds_bucket", append(lbls, "le", "2")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_server_seconds_bucket", append(lbls, "le", "+Inf")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_server_seconds_count", lbls))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_server_seconds_sum", lbls))

	assert.Equal(t, 0.0, sumValues(metricFamilies, "traces_service_graph_request_failed_total", nil))
	assert.Equal(t, 0.0, sumValues(metricFamilies, "traces_service_graph_unpaired_spans_total", nil))
	assert.Equal(t, 0.0, sumValues(metricFamilies, "traces_service_graph_dropped_spans_total", nil))

	// Span metrics
	lbls = []string{"service", "lb", "span_name", "lb-get", "span_kind", "SPAN_KIND_CLIENT", "status_code", "STATUS_CODE_UNSET"}
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_calls_total", lbls))
	assert.NotEqual(t, 0, sumValues(metricFamilies, "traces_spanmetrics_size_total", lbls))
	assert.Equal(t, 0.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "1")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "2")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "+Inf")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 2.0, sumValues(metricFamilies, "traces_spanmetrics_latency_sum", lbls))

	lbls = []string{"service", "app", "span_name", "app-handle", "span_kind", "SPAN_KIND_SERVER", "status_code", "STATUS_CODE_UNSET"}
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_calls_total", lbls))
	assert.NotEqual(t, 0, sumValues(metricFamilies, "traces_spanmetrics_size_total", lbls))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "1")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "2")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "+Inf")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_sum", lbls))

	// Verify metrics
	assert.NoError(t, tempoMetricsGenerator.WaitSumMetrics(e2e.Equals(4), "tempo_metrics_generator_spans_received_total"))
	assert.NoError(t, tempoMetricsGenerator.WaitSumMetrics(e2e.Equals(2), "tempo_metrics_generator_spans_discarded_total"))
	assert.NoError(t, tempoMetricsGenerator.WaitSumMetrics(e2e.Equals(25), "tempo_metrics_generator_registry_active_series"))
	assert.NoError(t, tempoMetricsGenerator.WaitSumMetrics(e2e.Equals(1000), "tempo_metrics_generator_registry_max_active_series"))
	assert.NoError(t, tempoMetricsGenerator.WaitSumMetrics(e2e.Equals(25), "tempo_metrics_generator_registry_series_added_total"))
}

func TestMetricsGeneratorTargetInfoEnabled(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configMetricsGeneratorTargetInfo, "config.yaml"))
	tempoDistributor := util.NewTempoDistributor()
	tempoIngester := util.NewTempoIngester(1)
	tempoMetricsGenerator := util.NewTempoMetricsGenerator()
	prometheus := newPrometheus()
	require.NoError(t, s.StartAndWaitReady(tempoDistributor, tempoIngester, tempoMetricsGenerator, prometheus))

	// Wait until ingester and metrics-generator are active
	isServiceActiveMatcher := func(service string) []*labels.Matcher {
		return []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "name", service),
			labels.MustNewMatcher(labels.MatchEqual, "state", "ACTIVE"),
		}
	}
	require.NoError(t, tempoDistributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(isServiceActiveMatcher("ingester")...), e2e.WaitMissingMetrics))
	require.NoError(t, tempoDistributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(isServiceActiveMatcher("metrics-generator")...), e2e.WaitMissingMetrics))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerGRPCClient(tempoDistributor.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	// Send two spans that have a client-server relationship
	r := rand.New(rand.NewSource(time.Now().UnixMilli()))
	traceIDLow := r.Int63()
	traceIDHigh := r.Int63()
	parentSpanID := r.Int63()

	err = c.EmitBatch(context.Background(), &thrift.Batch{
		Process: &thrift.Process{ServiceName: "lb"},
		Spans: []*thrift.Span{
			{
				TraceIdLow:    traceIDLow,
				TraceIdHigh:   traceIDHigh,
				SpanId:        parentSpanID,
				ParentSpanId:  0,
				OperationName: "lb-get",
				StartTime:     time.Now().UnixMicro(),
				Duration:      int64(2 * time.Second / time.Microsecond),
				Tags:          []*thrift.Tag{{Key: "span.kind", VStr: stringPtr("client")}},
			},
		},
	})
	require.NoError(t, err)

	err = c.EmitBatch(context.Background(), &thrift.Batch{
		Process: &thrift.Process{ServiceName: "app"},
		Spans: []*thrift.Span{
			{
				TraceIdLow:    traceIDLow,
				TraceIdHigh:   traceIDHigh,
				SpanId:        r.Int63(),
				ParentSpanId:  parentSpanID,
				OperationName: "app-handle",
				StartTime:     time.Now().UnixMicro(),
				Duration:      int64(1 * time.Second / time.Microsecond),
				Tags:          []*thrift.Tag{{Key: "span.kind", VStr: stringPtr("server")}},
			},
		},
	})
	require.NoError(t, err)

	// also send one with 5 minutes old timestamp
	err = c.EmitBatch(context.Background(), &thrift.Batch{
		Process: &thrift.Process{ServiceName: "app"},
		Spans: []*thrift.Span{
			{
				TraceIdLow:    traceIDLow,
				TraceIdHigh:   traceIDHigh,
				SpanId:        r.Int63(),
				ParentSpanId:  parentSpanID,
				OperationName: "app-handle",
				StartTime:     time.Now().Add(-5 * time.Minute).UnixMicro(),
				Duration:      int64(1 * time.Second / time.Microsecond),
				Tags:          []*thrift.Tag{{Key: "span.kind", VStr: stringPtr("server")}},
			},
		},
	})
	require.NoError(t, err)

	// also send one with timestamp 10 days in the future
	err = c.EmitBatch(context.Background(), &thrift.Batch{
		Process: &thrift.Process{ServiceName: "app"},
		Spans: []*thrift.Span{
			{
				TraceIdLow:    traceIDLow,
				TraceIdHigh:   traceIDHigh,
				SpanId:        r.Int63(),
				ParentSpanId:  parentSpanID,
				OperationName: "app-handle",
				StartTime:     time.Now().Add(10 * 24 * time.Hour).UnixMicro(),
				Duration:      int64(1 * time.Second / time.Microsecond),
				Tags:          []*thrift.Tag{{Key: "span.kind", VStr: stringPtr("server")}},
			},
		},
	})
	require.NoError(t, err)

	// Fetch metrics from Prometheus once they are received
	var metricFamilies map[string]*io_prometheus_client.MetricFamily
	for {
		metricFamilies, err = extractMetricsFromPrometheus(prometheus, `{__name__=~"traces_.+"}`)
		require.NoError(t, err)
		if len(metricFamilies) > 0 {
			break
		}
		time.Sleep(time.Second)
	}

	// Print collected metrics for easier debugging
	fmt.Println()
	for key, family := range metricFamilies {
		fmt.Println(key)
		for _, metric := range family.Metric {
			fmt.Println(metric)
		}
	}
	fmt.Println()

	// Service graphs
	lbls := []string{"client", "lb", "server", "app"}
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_total", lbls))

	assert.Equal(t, 0.0, sumValues(metricFamilies, "traces_service_graph_request_client_seconds_bucket", append(lbls, "le", "1")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_client_seconds_bucket", append(lbls, "le", "2")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_client_seconds_bucket", append(lbls, "le", "+Inf")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_client_seconds_count", lbls))
	assert.Equal(t, 2.0, sumValues(metricFamilies, "traces_service_graph_request_client_seconds_sum", lbls))

	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_server_seconds_bucket", append(lbls, "le", "1")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_server_seconds_bucket", append(lbls, "le", "2")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_server_seconds_bucket", append(lbls, "le", "+Inf")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_server_seconds_count", lbls))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_service_graph_request_server_seconds_sum", lbls))

	assert.Equal(t, 0.0, sumValues(metricFamilies, "traces_service_graph_request_failed_total", nil))
	assert.Equal(t, 0.0, sumValues(metricFamilies, "traces_service_graph_unpaired_spans_total", nil))
	assert.Equal(t, 0.0, sumValues(metricFamilies, "traces_service_graph_dropped_spans_total", nil))

	// Span metrics
	lbls = []string{"service", "lb", "span_name", "lb-get", "span_kind", "SPAN_KIND_CLIENT", "status_code", "STATUS_CODE_UNSET", "job", "lb"}
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_calls_total", lbls))
	assert.NotEqual(t, 0, sumValues(metricFamilies, "traces_spanmetrics_size_total", lbls))
	assert.Equal(t, 0.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "1")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "2")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "+Inf")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 2.0, sumValues(metricFamilies, "traces_spanmetrics_latency_sum", lbls))

	lbls = []string{"service", "app", "span_name", "app-handle", "span_kind", "SPAN_KIND_SERVER", "status_code", "STATUS_CODE_UNSET", "job", "app"}
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_calls_total", lbls))
	assert.NotEqual(t, 0, sumValues(metricFamilies, "traces_spanmetrics_size_total", lbls))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "1")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "2")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_bucket", append(lbls, "le", "+Inf")))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_count", lbls))
	assert.Equal(t, 1.0, sumValues(metricFamilies, "traces_spanmetrics_latency_sum", lbls))

	// Verify metrics
	assert.NoError(t, tempoMetricsGenerator.WaitSumMetrics(e2e.Equals(4), "tempo_metrics_generator_spans_received_total"))
	assert.NoError(t, tempoMetricsGenerator.WaitSumMetrics(e2e.Equals(2), "tempo_metrics_generator_spans_discarded_total"))
	assert.NoError(t, tempoMetricsGenerator.WaitSumMetrics(e2e.Equals(25), "tempo_metrics_generator_registry_active_series"))
	assert.NoError(t, tempoMetricsGenerator.WaitSumMetrics(e2e.Equals(1000), "tempo_metrics_generator_registry_max_active_series"))
	assert.NoError(t, tempoMetricsGenerator.WaitSumMetrics(e2e.Equals(25), "tempo_metrics_generator_registry_series_added_total"))
}

func newPrometheus() *e2e.HTTPService {
	return e2e.NewHTTPService(
		"prometheus",
		prometheusImage,
		e2e.NewCommandWithoutEntrypoint("/bin/prometheus", "--config.file=/etc/prometheus/prometheus.yml", "--web.enable-remote-write-receiver"),
		e2e.NewHTTPReadinessProbe(9090, "/-/ready", 200, 299),
		9090,
	)
}

// extractMetricsFromPrometheus extracts metrics stored in Prometheus using the /federate endpoint.
func extractMetricsFromPrometheus(prometheus *e2e.HTTPService, matcher string) (map[string]*io_prometheus_client.MetricFamily, error) {
	url := fmt.Sprintf("http://%s/federate?match[]=%s", prometheus.HTTPEndpoint(), url.QueryEscape(matcher))

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code %d while fetching federate metrics", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var tp expfmt.TextParser
	return tp.TextToMetricFamilies(strings.NewReader(string(body)))
}

// sumValues calculates the sum of all metrics in the metricFamily that contain the given labels.
// filterLabels must be key-value pairs.
func sumValues(metricFamily map[string]*io_prometheus_client.MetricFamily, metric string, filterLabels []string) float64 {
	if len(filterLabels)%2 != 0 {
		panic(fmt.Sprintf("filterLabels must be pairs: %v", filterLabels))
	}
	filterLabelsMap := map[string]string{}
	for i := 0; i < len(filterLabels); i += 2 {
		filterLabelsMap[filterLabels[i]] = filterLabels[i+1]
	}

	sum := 0.0

outer:
	for _, metric := range metricFamily[metric].GetMetric() {
		labelMap := map[string]string{}
		for _, label := range metric.GetLabel() {
			labelMap[label.GetName()] = label.GetValue()
		}

		for key, expectedValue := range filterLabelsMap {
			value, ok := labelMap[key]
			if !ok || value != expectedValue {
				continue outer
			}
		}

		// since we fetch metrics using /federate they are all untyped
		sum += metric.GetUntyped().GetValue()
	}

	return sum
}

func stringPtr(s string) *string {
	return &s
}
