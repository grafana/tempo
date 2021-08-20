package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	prometheusListenAddress string
	prometheusPath          string

	tempoQueryURL                 string
	tempoPushURL                  string
	tempoOrgID                    string
	tempoWriteBackoffDuration     time.Duration
	tempoLongWriteBackoffDuration time.Duration
	tempoReadBackoffDuration      time.Duration
	tempoRetentionDuration        time.Duration
	tempoIncludeLongTraces        bool

	logger *zap.Logger
)

type traceMetrics struct {
	requested          int
	requestFailed      int
	notFound           int
	missingSpans       int
	incorrectSpanCount int
	incorrectAttribute int
}

func init() {
	flag.StringVar(&prometheusPath, "prometheus-path", "/metrics", "The path to publish Prometheus metrics to.")
	flag.StringVar(&prometheusListenAddress, "prometheus-listen-address", ":80", "The address to listen on for Prometheus scrapes.")

	flag.StringVar(&tempoQueryURL, "tempo-query-url", "", "The URL (scheme://hostname) at which to query Tempo.")
	flag.StringVar(&tempoPushURL, "tempo-push-url", "", "The URL (scheme://hostname:port) at which to push traces to Tempo.")
	flag.StringVar(&tempoOrgID, "tempo-org-id", "", "The orgID to query in Tempo")
	flag.DurationVar(&tempoWriteBackoffDuration, "tempo-write-backoff-duration", 15*time.Second, "The amount of time to pause between write Tempo calls")
	flag.DurationVar(&tempoLongWriteBackoffDuration, "tempo-long-write-backoff-duration", 45*time.Second, "The amount of time to pause between appending spans to long running traces")
	flag.DurationVar(&tempoReadBackoffDuration, "tempo-read-backoff-duration", 30*time.Second, "The amount of time to pause between read Tempo calls")
	flag.DurationVar(&tempoRetentionDuration, "tempo-retention-duration", 336*time.Hour, "The block retention that Tempo is using")
	flag.BoolVar(&tempoIncludeLongTraces, "long-traces", false, "Include sending of long running traces")
}

func main() {
	flag.Parse()

	config := zap.NewDevelopmentEncoderConfig()
	logger = zap.New(zapcore.NewCore(
		zaplogfmt.NewEncoder(config),
		os.Stdout,
		zapcore.DebugLevel,
	))

	logger.Info("Tempo Vulture starting")

	v, err := NewVulture(tempoWriteBackoffDuration, tempoLongWriteBackoffDuration, tempoReadBackoffDuration)
	if err != nil {
		panic(err)
	}

	_, cancel := v.Start()
	defer cancel()

	http.Handle(prometheusPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(prometheusListenAddress, nil))
}

func newOtelGRPCClient(endpoint string) otlptrace.Client {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", endpoint, 55680)),
		otlptracegrpc.WithReconnectionPeriod(50 * time.Millisecond),
	}

	client := otlptracegrpc.NewClient(opts...)

	return client
}

func generateRandomString() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	s := make([]rune, generateRandomInt(5, 20))
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func generateRandomInt(min int64, max int64) int64 {
	number := min + rand.Int63n(max-min)
	if number == min {
		return generateRandomInt(min, max)
	}
	return number
}

func queryTempoAndAnalyze(baseURL string, complete completeTrace) (traceMetrics, error) {
	tm := traceMetrics{
		requested: 1,
	}

	traceID := complete.traceID

	log := logger.With(
		zap.String("query_trace_id", traceID.String()),
		zap.String("tempo_query_url", baseURL+"/api/traces/"+traceID.String()),
	)
	log.Info("querying Tempo")

	trace, err := util.QueryTrace(baseURL, traceID.String(), tempoOrgID)
	if err != nil {
		if err == util.ErrTraceNotFound {
			tm.notFound++
		} else {
			tm.requestFailed++
		}
		log.Error("error querying Tempo", zap.Error(err))
		return tm, err
	}

	if len(trace.Batches) == 0 {
		log.Error("trace contains 0 batches")
		tm.notFound++
	}

	// iterate through
	if hasMissingSpans(trace) {
		log.Error("trace has missing spans")
		tm.missingSpans++
	}

	count := spanCount(trace)
	if count != complete.spanCount {
		log.Error(fmt.Sprintf("trace has incorrect span count; expected %d, have %d", complete.spanCount, count))
		tm.incorrectSpanCount++
	}

	if !equalSpanAttributes(trace, complete) {
		tm.incorrectAttribute++
	}

	return tm, nil
}

func equalSpanAttributes(t *tempopb.Trace, complete completeTrace) bool {
	for _, b := range t.Batches {
		for _, ils := range b.InstrumentationLibrarySpans {
			for _, s := range ils.Spans {
				if len(s.ParentSpanId) > 0 {
					continue
				}

				if len(s.Attributes) != len(complete.attributes) {
					return false
				}

				attrs := pbToAttributes(s.Attributes)
				if reflect.DeepEqual(attrs, complete.attributes) {
					return true
				}

			}
		}
	}

	return false
}

func pbToAttributes(attrs []*v1.KeyValue) []attribute.KeyValue {
	traceAttrs := []attribute.KeyValue{}

	for _, a := range attrs {
		switch a.Value.Value.(type) {
		case *v1.AnyValue_StringValue:
			traceAttrs = append(traceAttrs, attribute.String(a.Key, a.Value.GetStringValue()))
		case *v1.AnyValue_IntValue:
			traceAttrs = append(traceAttrs, attribute.Int64(a.Key, a.Value.GetIntValue()))
		case *v1.AnyValue_BoolValue:
			traceAttrs = append(traceAttrs, attribute.Bool(a.Key, a.Value.GetBoolValue()))
		case *v1.AnyValue_DoubleValue:
			traceAttrs = append(traceAttrs, attribute.Float64(a.Key, a.Value.GetDoubleValue()))
		}
	}

	return traceAttrs
}

func spanCount(t *tempopb.Trace) int {
	count := 0

	for _, b := range t.Batches {
		for _, ils := range b.InstrumentationLibrarySpans {
			for _, s := range ils.Spans {
				if len(s.ParentSpanId) > 0 {
					count++
				}
			}
		}
	}

	return count
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

func logSpan(ctx context.Context, tracer trace.Tracer, span trace.Span) {
	_, s := tracer.Start(ctx, "log")
	defer s.End()

	log := logger.With(
		zap.String("traceID", span.SpanContext().TraceID().String()),
		zap.String("spanID", span.SpanContext().SpanID().String()),
	)

	log.Info("span")
}

type completeTrace struct {
	traceID    trace.TraceID
	spanCount  int
	attributes []attribute.KeyValue
}
