package main

import (
	"bytes"
	"crypto/tls"
	"errors"
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
	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
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
	tempoPushTLS                  bool

	logger *zap.Logger
)

type traceMetrics struct {
	incorrectResult         int
	missingSpans            int
	notFoundByID            int
	notFoundSearch          int
	notFoundTraceQL         int
	requested               int
	requestFailed           int
	notFoundSearchAttribute int
}

func init() {
	flag.StringVar(&prometheusPath, "prometheus-path", "/metrics", "The path to publish Prometheus metrics to.")
	flag.StringVar(&prometheusListenAddress, "prometheus-listen-address", ":80", "The address to listen on for Prometheus scrapes.")

	flag.StringVar(&tempoQueryURL, "tempo-query-url", "", "The URL (scheme://hostname) at which to query Tempo.")
	flag.StringVar(&tempoPushURL, "tempo-push-url", "", "The URL (scheme://hostname:port) at which to push traces to Tempo.")
	flag.BoolVar(&tempoPushTLS, "tempo-push-tls", false, "Whether to use TLS when pushing spans to Tempo")
	flag.StringVar(&tempoOrgID, "tempo-org-id", "", "The orgID to query in Tempo")
	flag.DurationVar(&tempoWriteBackoffDuration, "tempo-write-backoff-duration", 15*time.Second, "The amount of time to pause between write Tempo calls")
	flag.DurationVar(&tempoLongWriteBackoffDuration, "tempo-long-write-backoff-duration", 1*time.Minute, "The amount of time to pause between long write Tempo calls")
	flag.DurationVar(&tempoReadBackoffDuration, "tempo-read-backoff-duration", 30*time.Second, "The amount of time to pause between read Tempo calls")
	flag.DurationVar(&tempoSearchBackoffDuration, "tempo-search-backoff-duration", 60*time.Second, "The amount of time to pause between search Tempo calls.  Set to 0s to disable search.")
	flag.DurationVar(&tempoRetentionDuration, "tempo-retention-duration", 336*time.Hour, "The block retention that Tempo is using")
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

	r := rand.New(rand.NewSource(actualStartTime.Unix()))

	var tickerRead *time.Ticker
	if tempoReadBackoffDuration > 0 {
		tickerRead = time.NewTicker(tempoReadBackoffDuration)
	}

	var tickerSearch *time.Ticker
	if tempoSearchBackoffDuration > 0 {
		tickerSearch = time.NewTicker(tempoSearchBackoffDuration)
	}

	if tickerRead == nil && tickerSearch == nil {
		log.Fatalf("at least one of tempo-search-backoff-duration or tempo-read-backoff-duration must be set")
	}

	interval := tempoWriteBackoffDuration

	ready := func(info *util.TraceInfo, now time.Time) bool {
		// Don't attempt to read on the first itteration if we can't reasonably
		// expect the write loop to have fired yet.  Double the duration here to
		// avoid a race.
		if info.Timestamp().Before(actualStartTime.Add(2 * tempoWriteBackoffDuration)) {
			return false
		}

		return info.Ready(now, tempoWriteBackoffDuration, tempoLongWriteBackoffDuration)
	}

	// Write
	go func() {
		client, err := newJaegerGRPCClient(tempoPushURL)
		if err != nil {
			panic(err)
		}

		for now := range tickerWrite.C {
			timestamp := now.Round(interval)
			info := util.NewTraceInfo(timestamp, tempoOrgID)

			log := logger.With(
				zap.String("org_id", tempoOrgID),
				zap.Int64("seed", info.Timestamp().Unix()),
			)

			log.Info("sending trace")

			err := info.EmitBatches(client)
			if err != nil {
				metricErrorTotal.Inc()
			}
			queueFutureBatches(client, info)
		}
	}()

	// Read
	if tickerRead != nil {
		go func() {
			for now := range tickerRead.C {
				var seed time.Time
				startTime, seed = selectPastTimestamp(startTime, now, interval, tempoRetentionDuration, r)

				log := logger.With(
					zap.String("org_id", tempoOrgID),
					zap.Int64("seed", seed.Unix()),
				)

				info := util.NewTraceInfo(seed, tempoOrgID)

				// Don't query for a trace we don't expect to be complete
				if !ready(info, now) {
					continue
				}

				client := httpclient.New(tempoQueryURL, tempoOrgID)

				// query the trace
				queryMetrics, err := queryTrace(client, info)
				if err != nil {
					metricErrorTotal.Inc()
					log.Error("query for metrics failed",
						zap.Error(err),
					)
				}
				pushMetrics(queryMetrics)
			}
		}()
	}

	// Search
	if tickerSearch != nil {
		go func() {
			for now := range tickerSearch.C {
				_, seed := selectPastTimestamp(startTime, now, interval, tempoRetentionDuration, r)
				log := logger.With(
					zap.String("org_id", tempoOrgID),
					zap.Int64("seed", seed.Unix()),
				)

				info := util.NewTraceInfo(seed, tempoOrgID)

				if !ready(info, now) {
					continue
				}

				client := httpclient.New(tempoQueryURL, tempoOrgID)

				// query a tag we expect the trace to be found within
				searchMetrics, err := searchTag(client, seed)
				if err != nil {
					metricErrorTotal.Inc()
					log.Error("search tag for metrics failed",
						zap.Error(err),
					)
				}
				pushMetrics(searchMetrics)

				// traceql query
				traceqlSearchMetrics, err := searchTraceql(client, seed)
				if err != nil {
					metricErrorTotal.Inc()
					log.Error("traceql query for metrics failed",
						zap.Error(err),
					)
				}
				pushMetrics(traceqlSearchMetrics)
			}
		}()
	}

	http.Handle(prometheusPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(prometheusListenAddress, nil))
}

func queueFutureBatches(client *jaeger_grpc.Reporter, info *util.TraceInfo) {
	if info.LongWritesRemaining() == 0 {
		return
	}

	log := logger.With(
		zap.String("org_id", tempoOrgID),
		zap.String("write_trace_id", info.HexID()),
		zap.Int64("seed", info.Timestamp().Unix()),
		zap.Int64("longWritesRemaining", info.LongWritesRemaining()),
	)
	log.Info("queueing future batches")

	info.Done()

	go func() {
		time.Sleep(tempoLongWriteBackoffDuration)

		log := logger.With(
			zap.String("org_id", tempoOrgID),
			zap.String("write_trace_id", info.HexID()),
			zap.Int64("seed", info.Timestamp().Unix()),
			zap.Int64("longWritesRemaining", info.LongWritesRemaining()),
		)
		log.Info("sending trace")

		err := info.EmitBatches(client)
		if err != nil {
			log.Error("failed to queue batches",
				zap.Error(err),
			)
		}

		queueFutureBatches(client, info)
	}()
}

func pushMetrics(metrics traceMetrics) {
	metricTracesInspected.Add(float64(metrics.requested))
	metricTracesErrors.WithLabelValues("incorrectresult").Add(float64(metrics.incorrectResult))
	metricTracesErrors.WithLabelValues("missingspans").Add(float64(metrics.missingSpans))
	metricTracesErrors.WithLabelValues("notfound_search").Add(float64(metrics.notFoundSearch))
	metricTracesErrors.WithLabelValues("notfound_traceql").Add(float64(metrics.notFoundTraceQL))
	metricTracesErrors.WithLabelValues("notfound_byid").Add(float64(metrics.notFoundByID))
	metricTracesErrors.WithLabelValues("requestfailed").Add(float64(metrics.requestFailed))
	metricTracesErrors.WithLabelValues("notfound_search_attribute").Add(float64(metrics.notFoundSearchAttribute))
}

func selectPastTimestamp(start, stop time.Time, interval, retention time.Duration, r *rand.Rand) (newStart, ts time.Time) {
	oldest := stop.Add(-retention)

	if oldest.After(start) {
		newStart = oldest
	} else {
		newStart = start
	}

	ts = time.Unix(generateRandomInt(newStart.Unix(), stop.Unix(), r), 0)

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

	var dialOpts []grpc.DialOption

	if tempoPushTLS {
		dialOpts = []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
			})),
		}
	} else {
		dialOpts = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
	}

	// new jaeger grpc exporter
	conn, err := grpc.Dial(u.Host+":14250", dialOpts...)
	if err != nil {
		return nil, err
	}
	return jaeger_grpc.NewReporter(conn, nil, logger), err
}

func generateRandomInt(min, max int64, r *rand.Rand) int64 {
	min++
	number := min + r.Int63n(max-min)
	return number
}

func searchTag(client *httpclient.Client, seed time.Time) (traceMetrics, error) {
	tm := traceMetrics{
		requested: 1,
	}

	info := util.NewTraceInfo(seed, tempoOrgID)
	hexID := info.HexID()

	// Get the expected
	expected, err := info.ConstructTraceFromEpoch()
	if err != nil {
		logger.Error("unable to construct trace from epoch", zap.Error(err))
		return traceMetrics{}, err
	}

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

	attr := util.RandomAttrFromTrace(expected)
	if attr == nil {
		tm.notFoundSearchAttribute++
		return tm, fmt.Errorf("no search attr selected from trace")
	}

	logger := logger.With(
		zap.Int64("seed", seed.Unix()),
		zap.String("hexID", hexID),
		zap.Duration("ago", time.Since(seed)),
		zap.String("key", attr.Key),
		zap.String("value", util.StringifyAnyValue(attr.Value)),
	)
	logger.Info("searching Tempo via search tag")

	// Use the search API to find details about the expected trace. give an hour range
	//  around the seed.
	start := seed.Add(-30 * time.Minute).Unix()
	end := seed.Add(30 * time.Minute).Unix()
	resp, err := client.SearchWithRange(fmt.Sprintf("%s=%s", attr.Key, util.StringifyAnyValue(attr.Value)), start, end)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to search traces with tag %s: %s", attr.Key, err.Error()))
		tm.requestFailed++
		return tm, err
	}

	if !traceInTraces(hexID, resp.Traces) {
		tm.notFoundSearch++
		return tm, fmt.Errorf("trace %s not found in search response: %+v", hexID, resp.Traces)
	}

	return tm, nil
}

func searchTraceql(client *httpclient.Client, seed time.Time) (traceMetrics, error) {
	tm := traceMetrics{
		requested: 1,
	}

	info := util.NewTraceInfo(seed, tempoOrgID)
	hexID := info.HexID()

	// Get the expected
	expected, err := info.ConstructTraceFromEpoch()
	if err != nil {
		logger.Error("unable to construct trace from epoch", zap.Error(err))
		return traceMetrics{}, err
	}

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

	attr := util.RandomAttrFromTrace(expected)
	if attr == nil {
		tm.notFoundSearchAttribute++
		return tm, fmt.Errorf("no search attr selected from trace")
	}

	logger := logger.With(
		zap.Int64("seed", seed.Unix()),
		zap.String("hexID", hexID),
		zap.Duration("ago", time.Since(seed)),
		zap.String("key", attr.Key),
		zap.String("value", util.StringifyAnyValue(attr.Value)),
	)
	logger.Info("searching Tempo via traceql")

	start := seed.Add(-30 * time.Minute).Unix()
	end := seed.Add(30 * time.Minute).Unix()
	resp, err := client.SearchTraceQLWithRange(fmt.Sprintf(`{.%s = "%s"}`, attr.Key, util.StringifyAnyValue(attr.Value)), start, end)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to search traces with traceql %s: %s", attr.Key, err.Error()))
		tm.requestFailed++
		return tm, err
	}

	if !traceInTraces(hexID, resp.Traces) {
		tm.notFoundTraceQL++
		return tm, fmt.Errorf("trace %s not found in search traceql response: %+v", hexID, resp.Traces)
	}

	return tm, nil
}

func queryTrace(client *httpclient.Client, info *util.TraceInfo) (traceMetrics, error) {
	tm := traceMetrics{
		requested: 1,
	}

	hexID := info.HexID()

	logger := logger.With(
		zap.Int64("seed", info.Timestamp().Unix()),
		zap.String("hexID", hexID),
		zap.Duration("ago", time.Since(info.Timestamp())),
	)
	logger.Info("querying Tempo")

	trace, err := client.QueryTrace(hexID)
	if err != nil {
		if errors.Is(err, util.ErrTraceNotFound) {
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
	expected, err := info.ConstructTraceFromEpoch()
	if err != nil {
		logger.Error("unable to construct trace from epoch", zap.Error(err))
		return tm, err
	}

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

func equalTraces(a, b *tempopb.Trace) bool {
	trace.SortTraceAndAttributes(a)
	trace.SortTraceAndAttributes(b)

	return reflect.DeepEqual(a, b)
}

func hasMissingSpans(t *tempopb.Trace) bool {
	// collect all parent span IDs
	linkedSpanIDs := make([][]byte, 0)

	for _, b := range t.Batches {
		for _, ss := range b.ScopeSpans {
			for _, s := range ss.Spans {
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
			for _, ss := range b.ScopeSpans {
				for _, s := range ss.Spans {
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
