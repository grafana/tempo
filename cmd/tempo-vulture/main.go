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
	"os"
	"reflect"
	"time"

	"github.com/go-test/deep"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	zaplogfmt "github.com/jsternberg/zap-logfmt"

	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/weaveworks/common/user"
	jaegerTrans "go.opentelemetry.io/collector/translator/trace/jaeger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
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
	tempoSearchBackoffDuration    time.Duration
	tempoRetentionDuration        time.Duration
	tempoSearchRetentionDuration  time.Duration

	logger *zap.Logger

	// batchHighWaterMark is used when writing and reading, and needs to match so
	// that we get the expected number of batches on a trace.  A value larger
	// than 25 here results in vulture writing traces that exceed the maximum
	// trace size.
	batchHighWaterMark int64 = 25
)

type traceMetrics struct {
	incorrectResult         int
	missingSpans            int
	notFoundByID            int
	notFoundSearch          int
	requested               int
	requestFailed           int
	notFoundSearchAttribute int
}

type traceInfo struct {
	timestamp   time.Time
	r           *rand.Rand
	traceIDHigh int64
	traceIDLow  int64
}

func init() {
	flag.StringVar(&prometheusPath, "prometheus-path", "/metrics", "The path to publish Prometheus metrics to.")
	flag.StringVar(&prometheusListenAddress, "prometheus-listen-address", ":80", "The address to listen on for Prometheus scrapes.")

	flag.StringVar(&tempoQueryURL, "tempo-query-url", "", "The URL (scheme://hostname) at which to query Tempo.")
	flag.StringVar(&tempoPushURL, "tempo-push-url", "", "The URL (scheme://hostname:port) at which to push traces to Tempo.")
	flag.StringVar(&tempoOrgID, "tempo-org-id", "", "The orgID to query in Tempo")
	flag.DurationVar(&tempoWriteBackoffDuration, "tempo-write-backoff-duration", 15*time.Second, "The amount of time to pause between write Tempo calls")
	flag.DurationVar(&tempoLongWriteBackoffDuration, "tempo-long-write-backoff-duration", 1*time.Minute, "The amount of time to pause between long write Tempo calls")
	flag.DurationVar(&tempoReadBackoffDuration, "tempo-read-backoff-duration", 30*time.Second, "The amount of time to pause between read Tempo calls")
	flag.DurationVar(&tempoSearchBackoffDuration, "tempo-search-backoff-duration", 60*time.Second, "The amount of time to pause between search Tempo calls")
	flag.DurationVar(&tempoRetentionDuration, "tempo-retention-duration", 336*time.Hour, "The block retention that Tempo is using")
	flag.DurationVar(&tempoSearchRetentionDuration, "tempo-search-retention-duration", 10*time.Minute, "The ingester retention we expect to be able to search within")
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

	actualStartTime := time.Now()
	startTime := actualStartTime
	tickerWrite := time.NewTicker(tempoWriteBackoffDuration)
	tickerRead := time.NewTicker(tempoReadBackoffDuration)
	tickerSearch := time.NewTicker(tempoSearchBackoffDuration)
	interval := tempoWriteBackoffDuration

	// Write
	go func() {
		client, err := newJaegerGRPCClient(tempoPushURL)
		if err != nil {
			panic(err)
		}

		for now := range tickerWrite.C {
			timestamp := now.Round(interval)
			r := newRand(timestamp)

			traceIDHigh := r.Int63()
			traceIDLow := r.Int63()

			info := &traceInfo{
				timestamp:   timestamp,
				r:           r,
				traceIDHigh: traceIDHigh,
				traceIDLow:  traceIDLow,
			}

			emitBatches(client, info)
			queueFutureBatches(client, info)
		}
	}()

	// Read
	go func() {
		for now := range tickerRead.C {
			var seed time.Time
			startTime, seed = selectPastTimestamp(startTime, now, interval, tempoRetentionDuration)

			// Don't attempt to read on the first itteration if we can't reasonably
			// expect the write loop to have fired yet.  Double the duration here to
			// avoid a race.
			if seed.Before(actualStartTime.Add(2 * tempoWriteBackoffDuration)) {
				continue
			}

			// Don't use the last time interval to allow the write loop to finish
			// before we try to read it.
			if seed.After(now.Add(-tempoWriteBackoffDuration)) {
				continue
			}

			log := logger.With(
				zap.String("org_id", tempoOrgID),
			)

			client := util.NewClient(tempoQueryURL, tempoOrgID)

			// query the trace
			queryMetrics, err := queryTrace(client, seed)
			if err != nil {
				metricErrorTotal.Inc()
				log.Error("query for metrics failed",
					zap.Error(err),
				)
			}
			pushMetrics(queryMetrics)
		}
	}()

	// Search
	go func() {
		for now := range tickerSearch.C {
			_, seed := selectPastTimestamp(startTime, now, interval, tempoSearchRetentionDuration)

			// Don't attempt to read on the first itteration if we can't reasonably
			// expect the write loop to have fired yet.  Double the duration here to
			// avoid a race.
			if seed.Before(startTime.Add(2 * tempoWriteBackoffDuration)) {
				continue
			}

			// Don't use the last time interval to allow the write loop to finish
			// before we try to read it.
			if seed.After(now.Add(-tempoWriteBackoffDuration)) {
				continue
			}

			log := logger.With(
				zap.String("org_id", tempoOrgID),
			)

			client := util.NewClient(tempoQueryURL, tempoOrgID)

			// query a tag we expect the trace to be found within
			searchMetrics, err := searchTag(client, seed)
			if err != nil {
				metricErrorTotal.Inc()
				log.Error("search for metrics failed",
					zap.Error(err),
				)
			}
			pushMetrics(searchMetrics)
		}
	}()

	http.Handle(prometheusPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(prometheusListenAddress, nil))
}

func emitBatches(c *jaeger_grpc.Reporter, t *traceInfo) {
	log := logger.With(
		zap.String("org_id", tempoOrgID),
		zap.String("write_trace_id", fmt.Sprintf("%016x%016x", t.traceIDHigh, t.traceIDLow)),
		zap.Int64("seed", t.timestamp.Unix()),
	)

	log.Info("sending trace")

	for i := int64(0); i < generateRandomInt(1, batchHighWaterMark, t.r); i++ {
		ctx := user.InjectOrgID(context.Background(), tempoOrgID)
		ctx, err := user.InjectIntoGRPCRequest(ctx)
		if err != nil {
			log.Error("error injecting org id", zap.Error(err))
			metricErrorTotal.Inc()
			continue
		}
		err = c.EmitBatch(ctx, makeThriftBatch(t.traceIDHigh, t.traceIDLow, t.r, t.timestamp))
		if err != nil {
			log.Error("error pushing batch to Tempo", zap.Error(err))
			metricErrorTotal.Inc()
			continue
		}
	}

}

func queueFutureBatches(client *jaeger_grpc.Reporter, info *traceInfo) {
	log := logger.With(
		zap.String("org_id", tempoOrgID),
		zap.String("write_trace_id", fmt.Sprintf("%016x%016x", info.traceIDHigh, info.traceIDLow)),
		zap.Int64("seed", info.timestamp.Unix()),
	)

	if maybe(info.r) {
		log.Info("queueing future batches")
		go func() {
			time.Sleep(tempoLongWriteBackoffDuration)
			emitBatches(client, info)
			queueFutureBatches(client, info)
		}()
	}
}

func maybe(r *rand.Rand) bool {
	return r.Intn(10) >= 8
}

func pushMetrics(metrics traceMetrics) {
	metricTracesInspected.Add(float64(metrics.requested))
	metricTracesErrors.WithLabelValues("incorrectresult").Add(float64(metrics.incorrectResult))
	metricTracesErrors.WithLabelValues("missingspans").Add(float64(metrics.missingSpans))
	metricTracesErrors.WithLabelValues("notfound_search").Add(float64(metrics.notFoundSearch))
	metricTracesErrors.WithLabelValues("notfound_byid").Add(float64(metrics.notFoundByID))
	metricTracesErrors.WithLabelValues("requestfailed").Add(float64(metrics.requestFailed))
	metricTracesErrors.WithLabelValues("notfound_search_attribute").Add(float64(metrics.notFoundSearchAttribute))
}

func selectPastTimestamp(start, stop time.Time, interval time.Duration, retention time.Duration) (newStart, ts time.Time) {
	oldest := stop.Add(-retention)

	if oldest.After(start) {
		newStart = oldest
	} else {
		newStart = start
	}

	ts = time.Unix(generateRandomInt(newStart.Unix(), stop.Unix(), newRand(start)), 0)

	return newStart.Round(interval), ts.Round(interval)
}

func newJaegerGRPCClient(endpoint string) (*jaeger_grpc.Reporter, error) {
	// remove scheme and port
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	logger.Info("dialing grpc",
		zap.String("endpoint", fmt.Sprintf("%s:14250", u.Host)),
	)

	// new jaeger grpc exporter
	conn, err := grpc.Dial(u.Host+":14250", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return jaeger_grpc.NewReporter(conn, nil, logger), err
}

func newRand(t time.Time) *rand.Rand {
	return rand.New(rand.NewSource(t.Unix()))
}

func generateRandomString(r *rand.Rand) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	s := make([]rune, generateRandomInt(5, 20, r))
	for i := range s {
		s[i] = letters[r.Intn(len(letters))]
	}
	return string(s)
}

func generateRandomTags(r *rand.Rand) []*thrift.Tag {
	var tags []*thrift.Tag
	count := generateRandomInt(1, 5, r)
	for i := int64(0); i < count; i++ {
		value := generateRandomString(r)
		tags = append(tags, &thrift.Tag{
			Key:  fmt.Sprintf("vulture-%d", i),
			VStr: &value,
		})
	}
	return tags
}

func generateRandomLogs(r *rand.Rand, now time.Time) []*thrift.Log {
	var logs []*thrift.Log
	count := generateRandomInt(1, 5, r)
	for i := int64(0); i < count; i++ {
		logs = append(logs, &thrift.Log{
			Timestamp: now.Unix(),
			Fields:    generateRandomTags(r),
		})
	}
	return logs
}

func makeThriftBatch(TraceIDHigh int64, TraceIDLow int64, r *rand.Rand, now time.Time) *thrift.Batch {
	var spans []*thrift.Span
	count := generateRandomInt(1, 5, r)
	for i := int64(0); i < count; i++ {
		spans = append(spans, &thrift.Span{
			TraceIdLow:    TraceIDLow,
			TraceIdHigh:   TraceIDHigh,
			SpanId:        r.Int63(),
			ParentSpanId:  0,
			OperationName: generateRandomString(r),
			References:    nil,
			Flags:         0,
			StartTime:     now.Unix(),
			Duration:      generateRandomInt(0, 100, r),
			Tags:          generateRandomTags(r),
			Logs:          generateRandomLogs(r, now),
		})
	}

	return &thrift.Batch{Spans: spans}
}

func generateRandomInt(min int64, max int64, r *rand.Rand) int64 {
	min++
	number := min + r.Int63n(max-min)
	return number
}

func searchTag(client *util.Client, seed time.Time) (traceMetrics, error) {
	tm := traceMetrics{
		requested: 1,
	}

	r := newRand(seed)
	seedHex := fmt.Sprintf("%016x%016x", r.Int63(), r.Int63())
	traceID, err := util.HexStringToTraceID(seedHex)
	if err != nil {
		return tm, err
	}

	hexID := util.TraceIDToHexString(traceID)

	// Get the expected
	expected := constructTraceFromEpoch(seed)

	traceInTraces := func(traceID string, traces []*tempopb.TraceSearchMetadata) bool {
		for _, t := range traces {
			equal, err := util.EqualHexStringTraceIDs(t.TraceID, traceID)
			if err != nil {
				logger.Error("error comparing trace IDs", zap.Error(err))
				continue
			}

			if equal {
				return true
			}
		}

		return false
	}

	attr := randomAttrFromTrace(expected, r)
	if attr == nil {
		tm.notFoundSearchAttribute++
		return tm, fmt.Errorf("no search attr selected from trace")
	}

	logger := logger.With(
		zap.Int64("seed", seed.Unix()),
		zap.String("hexID", hexID),
		zap.Duration("ago", time.Since(seed)),
		zap.String("key", attr.Key),
		zap.String("value", attr.Value.GetStringValue()),
	)
	logger.Info("searching Tempo")

	// Use the search API to find details about the expected trace
	resp, err := client.SearchTag(attr.Key, attr.Value.GetStringValue())
	if err != nil {
		logger.Error(fmt.Sprintf("failed to query tag values for %s: %s", attr.Key, err.Error()))
		tm.requestFailed++
		return tm, err
	}

	if !traceInTraces(hexID, resp.Traces) {
		tm.notFoundSearch++
		return tm, fmt.Errorf("trace %s not found in search response: %+v", hexID, resp.Traces)
	}

	return tm, nil
}

func queryTrace(client *util.Client, seed time.Time) (traceMetrics, error) {
	tm := traceMetrics{
		requested: 1,
	}

	r := newRand(seed)
	hexID := fmt.Sprintf("%016x%016x", r.Int63(), r.Int63())

	logger := logger.With(
		zap.Int64("seed", seed.Unix()),
		zap.String("hexID", hexID),
		zap.Duration("ago", time.Since(seed)),
	)
	logger.Info("querying Tempo")

	trace, err := client.QueryTrace(hexID)
	if err != nil {
		if err == util.ErrTraceNotFound {
			tm.notFoundByID++
		} else {
			tm.requestFailed++
		}
		logger.Error("error querying Tempo", zap.Error(err))
		return tm, err
	}

	if len(trace.Batches) == 0 {
		logger.Error("trace contains 0 batches")
		tm.notFoundByID++
		return tm, nil
	}

	// iterate through
	if hasMissingSpans(trace) {
		logger.Error("trace has missing spans")
		tm.missingSpans++
		return tm, nil
	}

	// Get the expected
	expected := constructTraceFromEpoch(seed)

	match := equalTraces(expected, trace)
	if !match {
		tm.incorrectResult++
		if diff := deep.Equal(expected, trace); diff != nil {
			for _, d := range diff {
				logger.Error("incorrect result",
					zap.String("expected -> response", d),
				)
			}
		}
		return tm, nil
	}

	return tm, nil
}

func randomAttrFromTrace(t *tempopb.Trace, r *rand.Rand) *v1common.KeyValue {
	if len(t.Batches) == 0 {
		return nil
	}
	iBatch := r.Intn(len(t.Batches))

	if len(t.Batches[iBatch].InstrumentationLibrarySpans) == 0 {
		return nil
	}
	iSpans := r.Intn(len(t.Batches[iBatch].InstrumentationLibrarySpans))

	if len(t.Batches[iBatch].InstrumentationLibrarySpans[iSpans].Spans) == 0 {
		return nil
	}
	iSpan := r.Intn(len(t.Batches[iBatch].InstrumentationLibrarySpans[iSpans].Spans))

	if len(t.Batches[iBatch].InstrumentationLibrarySpans[iSpans].Spans[iSpan].Attributes) == 0 {
		return nil
	}
	iAttr := r.Intn(len(t.Batches[iBatch].InstrumentationLibrarySpans[iSpans].Spans[iSpan].Attributes))

	return t.Batches[iBatch].InstrumentationLibrarySpans[iSpans].Spans[iSpan].Attributes[iAttr]
}

func equalTraces(a, b *tempopb.Trace) bool {
	model.SortTrace(a)
	model.SortTrace(b)

	return reflect.DeepEqual(a, b)
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

func constructTraceFromEpoch(epoch time.Time) *tempopb.Trace {
	r := newRand(epoch)

	info := &traceInfo{
		timestamp:   epoch,
		r:           r,
		traceIDHigh: r.Int63(),
		traceIDLow:  r.Int63(),
	}

	trace := &tempopb.Trace{}

	addBatches := func(t *traceInfo, trace *tempopb.Trace) {
		for i := int64(0); i < generateRandomInt(1, batchHighWaterMark, r); i++ {
			batch := makeThriftBatch(t.traceIDHigh, t.traceIDLow, r, epoch)
			internalTrace := jaegerTrans.ThriftBatchToInternalTraces(batch)
			conv, err := internalTrace.ToOtlpProtoBytes()
			if err != nil {
				logger.Error(err.Error())
			}

			t := tempopb.Trace{}
			err = t.Unmarshal(conv)
			if err != nil {
				logger.Error(err.Error())
			}

			// Due to the several transforms above, some manual mangling is required to
			// get the parentSpanID to match.  In the case of an empty []byte in place
			// for the ParentSpanId, we set to nil here to ensure that the final result
			// matches the json.Unmarshal value when tempo is queried.
			for _, b := range t.Batches {
				for _, l := range b.InstrumentationLibrarySpans {
					for _, s := range l.Spans {
						if len(s.GetParentSpanId()) == 0 {
							s.ParentSpanId = nil
						}
					}
				}
			}

			trace.Batches = append(trace.Batches, t.Batches...)
		}
	}

	addBatches(info, trace)

	lastWrite := info.timestamp
	for maybe(info.r) {
		lastWrite = lastWrite.Add(tempoLongWriteBackoffDuration)

		// log := logger.With(
		// 	zap.Int64("seed", info.timestamp.Unix()),
		// 	zap.Duration("since", time.Since(lastWrite)),
		// )

		// If the last write has happened very recently, we'll wait a bit to make
		// sure the write has taken place, and then add the batches that would have
		// just been written.
		if time.Since(lastWrite) < (1 * time.Second) {
			time.Sleep(1500 * time.Millisecond)
		}

		if time.Since(lastWrite) > (1 * time.Second) {
			// log.Info("adding batches")
			addBatches(info, trace)
		}
	}

	return trace
}
