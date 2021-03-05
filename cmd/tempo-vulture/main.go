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

	tempoQueryURL             string
	tempoPushURL              string
	tempoOrgID                string
	tempoWriteBackoffDuration time.Duration
	tempoReadBackoffDuration  time.Duration
	tempoRetentionDuration    time.Duration
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
	flag.DurationVar(&tempoWriteBackoffDuration, "tempo-write-backoff-duration", 15*time.Second, "The amount of time to pause between write Tempo calls")
	flag.DurationVar(&tempoReadBackoffDuration, "tempo-read-backoff-duration", 30*time.Second, "The amount of time to pause between read Tempo calls")
	flag.DurationVar(&tempoRetentionDuration, "tempo-retention-duration", 336*time.Hour, "The block retention that Tempo is using")
}

func main() {
	flag.Parse()

	glog.Error("Tempo Vulture Starting")

	startTime := time.Now().Unix()
	tickerWrite := time.NewTicker(tempoWriteBackoffDuration)
	tickerRead := time.NewTicker(tempoReadBackoffDuration)
	interval := int64(tempoWriteBackoffDuration / time.Second)

	// Write
	go func() {
		for {
			<-tickerWrite.C

			rand.Seed((time.Now().Unix() / interval) * interval)
			c, err := newJaegerGRPCClient(tempoPushURL)
			if err != nil {
				glog.Error("error creating grpc client", err)
				metricErrorTotal.Inc()
				continue
			}

			traceIDHigh := rand.Int63()
			traceIDLow := rand.Int63()
			for i := int64(0); i < generateRandomInt(1, 100); i++ {
				err = c.EmitBatch(context.Background(), makeThriftBatch(traceIDHigh, traceIDLow))
				if err != nil {
					glog.Error("error pushing batch to Tempo ", err)
					metricErrorTotal.Inc()
					continue
				}
			}
		}
	}()

	// Read
	go func() {
		for {
			<-tickerRead.C

			currentTime := time.Now().Unix()

			// don't query traces before retention
			if (currentTime - startTime) > int64(tempoRetentionDuration/time.Second) {
				startTime = currentTime - int64(tempoRetentionDuration/time.Second)
			}

			// pick past interval and re-generate trace
			rand.Seed((generateRandomInt(startTime, currentTime) / interval) * interval)
			hexID := fmt.Sprintf("%016x%016x", rand.Int63(), rand.Int63())

			// query the trace
			metrics, err := queryTempoAndAnalyze(tempoQueryURL, hexID)
			if err != nil {
				glog.Error("error querying Tempo ", err)
				metricErrorTotal.Inc()
				metricTracesErrors.WithLabelValues("failed").Inc()
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
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, err
	}
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

func generateRandomString() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	s := make([]rune, generateRandomInt(5, 20))
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func generateRandomTags() []*thrift.Tag {
	var tags []*thrift.Tag
	count := generateRandomInt(1, 5)
	for i := int64(0); i < count; i++ {
		value := generateRandomString()
		tags = append(tags, &thrift.Tag{
			Key:  generateRandomString(),
			VStr: &value,
		})
	}
	return tags
}

func generateRandomLogs() []*thrift.Log {
	var logs []*thrift.Log
	count := generateRandomInt(1, 5)
	for i := int64(0); i < count; i++ {
		logs = append(logs, &thrift.Log{
			Timestamp: time.Now().Unix(),
			Fields:    generateRandomTags(),
		})
	}
	return logs
}

func makeThriftBatch(TraceIDHigh int64, TraceIDLow int64) *thrift.Batch {
	var spans []*thrift.Span
	count := generateRandomInt(1, 5)
	for i := int64(0); i < count; i++ {
		spans = append(spans, &thrift.Span{
			TraceIdLow:    TraceIDLow,
			TraceIdHigh:   TraceIDHigh,
			SpanId:        rand.Int63(),
			ParentSpanId:  0,
			OperationName: generateRandomString(),
			References:    nil,
			Flags:         0,
			StartTime:     time.Now().Unix(),
			Duration:      rand.Int63(),
			Tags:          generateRandomTags(),
			Logs:          generateRandomLogs(),
		})
	}
	return &thrift.Batch{Spans: spans}
}

func generateRandomInt(min int64, max int64) int64 {
	number := min + rand.Int63n(max-min)
	if number == min {
		return generateRandomInt(min, max)
	}
	return number
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
