package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/weaveworks/common/user"
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

	logger *zap.Logger
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
	flag.StringVar(&tempoPushURL, "tempo-push-url", "", "The URL (scheme://hostname:port) at which to push traces to Tempo.")
	flag.StringVar(&tempoOrgID, "tempo-org-id", "", "The orgID to query in Tempo")
	flag.DurationVar(&tempoWriteBackoffDuration, "tempo-write-backoff-duration", 15*time.Second, "The amount of time to pause between write Tempo calls")
	flag.DurationVar(&tempoReadBackoffDuration, "tempo-read-backoff-duration", 30*time.Second, "The amount of time to pause between read Tempo calls")
	flag.DurationVar(&tempoRetentionDuration, "tempo-retention-duration", 336*time.Hour, "The block retention that Tempo is using")
}

func main() {
	flag.Parse()

	var err error
	logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	startTime := time.Now().Unix()
	tickerWrite := time.NewTicker(tempoWriteBackoffDuration)
	tickerRead := time.NewTicker(tempoReadBackoffDuration)
	interval := int64(tempoWriteBackoffDuration / time.Second)

	// Write
	go func() {
		c, err := newJaegerGRPCClient(tempoPushURL)
		if err != nil {
			panic(err)
		}

		for {
			<-tickerWrite.C

			rand.Seed((time.Now().Unix() / interval) * interval)
			traceIDHigh := rand.Int63()
			traceIDLow := rand.Int63()

			logger.Info("sending trace",
				zap.String("write_trace_id", fmt.Sprintf("%016x%016x", traceIDLow, traceIDHigh)),
			)

			for i := int64(0); i < generateRandomInt(1, 100); i++ {
				ctx := user.InjectOrgID(context.Background(), tempoOrgID)
				ctx, err := user.InjectIntoGRPCRequest(ctx)
				if err != nil {
					logger.Info("error injecting org id", zap.Error(err))
					metricErrorTotal.Inc()
					continue
				}
				err = c.EmitBatch(ctx, makeThriftBatch(traceIDHigh, traceIDLow))
				if err != nil {
					logger.Info("error pushing batch to Tempo", zap.Error(err))
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
				metricErrorTotal.Inc()
				metricTracesErrors.WithLabelValues("notfound").Inc()
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
	// new jaeger grpc exporter
	conn, err := grpc.Dial(u.Host+":14250", grpc.WithInsecure())
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

	logger := logger.With(
		zap.String("query_trace_id", traceID),
		zap.String("tempo_query_url", baseURL+"/api/traces/"+traceID),
	)
	logger.Info("querying Tempo")

	trace, err := util.QueryTrace(baseURL, traceID, tempoOrgID)
	if err == util.ErrTraceNotFound {
		tm.notfound++
	}
	if err != nil {
		logger.Info("error querying Tempo", zap.Error(err))
		return nil, err
	}

	if len(trace.Batches) == 0 {
		logger.Info("trace contains 0 batches")
		tm.notfound++
	}

	// iterate through
	if hasMissingSpans(trace) {
		logger.Info("trace has missing spans")
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
