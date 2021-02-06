package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/glog"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	prometheusListenAddress string
	prometheusPath          string

	tempoQueryURL        string
	tempoPushURL         string
	tempoOrgID           string
	tempoBackoffDuration time.Duration
)

type traceMetrics struct {
	requested    int
	notfound     int
	missingSpans int
}

func init() {
	flag.StringVar(&prometheusPath, "prometheus-path", "/metrics", "The path to publish Prometheus metrics to.")
	flag.StringVar(&prometheusListenAddress, "prometheus-listen-address", ":80", "The address to listen on for Prometheus scrapes.")

	flag.StringVar(&tempoQueryURL, "tempo-query-url", "", "The URL (scheme://hostname) at which to query Tempo.")
	flag.StringVar(&tempoPushURL, "tempo-push-url", "", "The URL (scheme://hostname) at which to push traces to Tempo.")
	flag.StringVar(&tempoOrgID, "tempo-org-id", "", "The orgID to query in Tempo")
	flag.DurationVar(&tempoBackoffDuration, "tempo-backoff-duration", time.Second, "The amount of time to pause between Tempo calls")
}

func main() {
	flag.Parse()

	glog.Error("Tempo Vulture Starting")

	startTime := time.Now().UTC().UnixNano()
	ticker := time.NewTicker(tempoBackoffDuration)
	iteration := int64(0)

	// Write
	go func() {
		for {
			<-ticker.C

			iteration++
			rand.Seed(startTime / iteration)

			c, err := newJaegerGRPCClient(tempoPushURL)
			if err != nil {
				glog.Error("error creating grpc client", err)
				metricErrorTotal.Inc()
				continue
			}

			batch := makeThriftBatch()
			err = c.EmitBatch(context.Background(), batch)
			if err != nil {
				glog.Error("error pushing trace to Tempo ", err)
				metricErrorTotal.Inc()
				continue
			}
		}
	}()

	// Read
	go func() {
		for {
			<-ticker.C

			if iteration == 1 {
				continue
			}

			// pick past iteration and re-generate trace
			rand.Seed(startTime / randInt(1, iteration-1))
			batch := makeThriftBatch()
			hexID := fmt.Sprintf("%016x%016x", batch.Spans[0].TraceIdHigh, batch.Spans[0].TraceIdLow)

			// query the trace
			metrics, err := queryTempoAndAnalyze(tempoQueryURL, hexID)
			if err != nil {
				glog.Error("error querying Tempo ", err)
				metricErrorTotal.Inc()
				continue
			}

			metricTracesInspected.Add(float64(metrics.requested))
			metricTracesErrors.WithLabelValues("notfound").Add(float64(metrics.notfound))
			metricTracesErrors.WithLabelValues("missingspans").Add(float64(metrics.missingSpans))
		}
	}()

	http.Handle(prometheusPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(prometheusListenAddress, nil))
}

func newJaegerGRPCClient(endpoint string) (*jaeger_grpc.Reporter, error) {
	// remove scheme and port
	u, _ := url.Parse(endpoint)
	host, _, _ := net.SplitHostPort(u.Host)
	// new jaeger grpc exporter
	conn, err := grpc.Dial(host+":14250", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	return jaeger_grpc.NewReporter(conn, nil, logger), err
}

func makeThriftBatch() *thrift.Batch {
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

func randInt(min int64, max int64) int64 {
	if min == max {
		return 1
	}
	return min + rand.Int63n(max-min)
}

func queryTempoAndAnalyze(baseURL string, traceID string) (*traceMetrics, error) {
	tm := &traceMetrics{
		requested: 1,
	}
	glog.Error("tempo url ", baseURL+"/api/traces/"+traceID)
	trace, err := util.QueryTrace(baseURL, traceID, tempoOrgID)
	if err == util.ErrTraceNotFound {
		glog.Error("trace not found ", traceID)
		tm.notfound++
	}
	if err != nil {
		return nil, err
	}

	if len(trace.Batches) == 0 {
		glog.Error("trace not found", traceID)
		tm.notfound++
	}

	// iterate through
	if hasMissingSpans(trace) {
		glog.Error("has missing spans", traceID)
		tm.missingSpans++
	}

	return tm, nil
}

func hasMissingSpans(t *tempopb.Trace) bool {
	// collect all parent span IDs
	linkedSpanIDs := make([][]byte, 0)

	for _, b := range t.Batches {
		for _, ils := range b.InstrumentationLibrarySpans {
			for _, s := range ils.Spans {
				if len(s.ParentSpanId) > 0 {
					linkedSpanIDs = append(linkedSpanIDs, s.ParentSpanId)
				}
			}
		}
	}

	for _, id := range linkedSpanIDs {
		found := false

	B:
		for _, b := range t.Batches {
			for _, ils := range b.InstrumentationLibrarySpans {
				for _, s := range ils.Spans {
					if bytes.Equal(s.SpanId, id) {
						found = true
						break B
					}
				}
			}
		}

		if !found {
			return true
		}
	}

	return false
}
