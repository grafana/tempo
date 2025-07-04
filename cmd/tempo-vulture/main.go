package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"

	"github.com/go-test/deep"
	"github.com/grafana/tempo/pkg/api"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	utilpkg "github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

var (
	prometheusListenAddress string
	prometheusPath          string

	tempoQueryURL                   string
	tempoPushURL                    string
	tempoOrgID                      string
	tempoWriteBackoffDuration       time.Duration
	tempoLongWriteBackoffDuration   time.Duration
	tempoReadBackoffDuration        time.Duration
	tempoSearchBackoffDuration      time.Duration
	tempoMetricsBackoffDuration     time.Duration
	tempoRetentionDuration          time.Duration
	tempoRecentTracesCutoffDuration time.Duration
	tempoPushTLS                    bool

	rf1After time.Time

	logger *zap.Logger
)

type traceMetrics struct {
	incorrectResult         int
	incorrectMetricsResult  int
	missingSpans            int
	notFoundByID            int
	notFoundSearch          int
	notFoundTraceQL         int
	notFoundByMetrics       int
	inaccurateMetrics       int
	requested               int
	requestFailed           int
	notFoundSearchAttribute int
}

const (
	defaultOTLPGRPCEndpoint = 4317
)

type vultureConfiguration struct {
	tempoQueryURL                   string
	tempoPushURL                    string
	tempoOrgID                      string
	tempoWriteBackoffDuration       time.Duration
	tempoLongWriteBackoffDuration   time.Duration
	tempoReadBackoffDuration        time.Duration
	tempoSearchBackoffDuration      time.Duration
	tempoMetricsBackoffDuration     time.Duration
	tempoRetentionDuration          time.Duration
	tempoRecentTracesCutoffDuration time.Duration
	tempoPushTLS                    bool
}

var _ flag.Value = (*timeVar)(nil)

type timeVar struct {
	t *time.Time
}

func newTimeVar(t *time.Time) *timeVar { return &timeVar{t: t} }

func (v timeVar) String() string {
	return (*v.t).Format(time.RFC3339)
}

func (v timeVar) Set(s string) error {
	if s == "" {
		return nil
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	*v.t = t

	return nil
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
	flag.DurationVar(&tempoMetricsBackoffDuration, "tempo-metrics-backoff-duration", 0, "The amount of time to pause between TraceQL Metrics Tempo calls.  Set to 0s to disable.")
	flag.DurationVar(&tempoRetentionDuration, "tempo-retention-duration", 336*time.Hour, "The block retention that Tempo is using")
	flag.DurationVar(&tempoRecentTracesCutoffDuration, "tempo-recent-traces-backoff-duration", 14*time.Minute, "Cutoff between recent and old traces query checks")

	flag.Var(newTimeVar(&rf1After), "rhythm-rf1-after", "Timestamp (RFC3339) after which only blocks with RF==1 are included in search and ID lookups")
}

func main() {
	flag.Parse()

	config := zap.NewDevelopmentEncoderConfig()
	logger = zap.New(zapcore.NewCore(
		zaplogfmt.NewEncoder(config),
		os.Stdout,
		zapcore.DebugLevel,
	))

	vultureConfig := vultureConfiguration{
		tempoQueryURL:                   tempoQueryURL,
		tempoPushURL:                    tempoPushURL,
		tempoOrgID:                      tempoOrgID,
		tempoWriteBackoffDuration:       tempoWriteBackoffDuration,
		tempoLongWriteBackoffDuration:   tempoLongWriteBackoffDuration,
		tempoReadBackoffDuration:        tempoReadBackoffDuration,
		tempoSearchBackoffDuration:      tempoSearchBackoffDuration,
		tempoMetricsBackoffDuration:     tempoMetricsBackoffDuration,
		tempoRetentionDuration:          tempoRetentionDuration,
		tempoRecentTracesCutoffDuration: tempoRecentTracesCutoffDuration,
		tempoPushTLS:                    tempoPushTLS,
	}
	pushEndpoint, err := getGRPCEndpoint(vultureConfig.tempoPushURL)
	if err != nil {
		panic(err)
	}

	logger.Info("Tempo Vulture starting", zap.String("tempoQueryURL", vultureConfig.tempoQueryURL), zap.String("tempoPushURL", pushEndpoint))

	jaegerClient, err := utilpkg.NewJaegerToOTLPExporter(pushEndpoint)
	if err != nil {
		panic(err)
	}

	httpClient := httpclient.New(vultureConfig.tempoQueryURL, vultureConfig.tempoOrgID)

	if !rf1After.IsZero() {
		httpClient.SetQueryParam(api.URLParamRF1After, rf1After.Format(time.RFC3339))
	}

	tickerWrite, tickerRead, tickerSearch, tickerMetrics, err := initTickers(
		vultureConfig.tempoWriteBackoffDuration,
		vultureConfig.tempoReadBackoffDuration,
		vultureConfig.tempoSearchBackoffDuration,
		vultureConfig.tempoMetricsBackoffDuration,
	)
	if err != nil {
		panic(err)
	}
	startTime := time.Now()
	r := rand.New(rand.NewSource(startTime.Unix()))
	interval := vultureConfig.tempoWriteBackoffDuration

	selectRecentTimestamp := func(now time.Time) (newStart, ts time.Time, skip bool) {
		oldest := now.Add(-vultureConfig.tempoRecentTracesCutoffDuration)
		if oldest.Before(startTime) { // if vulture's just started
			oldest = startTime
		}
		newStart, ts = selectPastTimestamp(oldest, now, interval, vultureConfig.tempoRetentionDuration, r)
		return
	}

	selectOldTimestamp := func(now time.Time) (newStart, ts time.Time, skip bool) {
		newest := now.Add(-vultureConfig.tempoRecentTracesCutoffDuration)
		if newest.Before(startTime) { // if vulture's just started and no traces to query
			skip = true
			return
		}
		newStart, ts = selectPastTimestamp(startTime, newest, interval, vultureConfig.tempoRetentionDuration, r)
		return
	}

	doWrite(jaegerClient, tickerWrite, interval, vultureConfig, logger)

	// Recent traces
	runChecker(tickerRead, vultureConfig, selectRecentTimestamp, func(info *util.TraceInfo, l *zap.Logger) {
		doRead(httpClient, vultureConfig, info, l)
	}, logger)
	runChecker(tickerSearch, vultureConfig, selectRecentTimestamp, func(info *util.TraceInfo, l *zap.Logger) {
		doSearch(httpClient, vultureConfig, info, l)
	}, logger)
	runChecker(tickerMetrics, vultureConfig, selectRecentTimestamp, func(info *util.TraceInfo, l *zap.Logger) {
		doMetrics(httpClient, vultureConfig, info, l)
	}, logger)

	// Old traces
	runChecker(tickerRead, vultureConfig, selectOldTimestamp, func(info *util.TraceInfo, l *zap.Logger) {
		doRead(httpClient, vultureConfig, info, l)
	}, logger)
	runChecker(tickerSearch, vultureConfig, selectOldTimestamp, func(info *util.TraceInfo, l *zap.Logger) {
		doSearch(httpClient, vultureConfig, info, l)
	}, logger)
	runChecker(tickerMetrics, vultureConfig, selectOldTimestamp, func(info *util.TraceInfo, l *zap.Logger) {
		doMetrics(httpClient, vultureConfig, info, l)
	}, logger)

	http.Handle(prometheusPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(prometheusListenAddress, nil))
}

func getGRPCEndpoint(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	url := u.String()

	if u.Port() == "" {
		url = fmt.Sprintf("%s:%d", url, defaultOTLPGRPCEndpoint)
	}
	return url, nil
}

func initTickers(
	tempoWriteBackoffDuration time.Duration,
	tempoReadBackoffDuration time.Duration,
	tempoSearchBackoffDuration time.Duration,
	tempoMetricsBackoffDuration time.Duration,
) (
	tickerWrite *time.Ticker,
	tickerRead *time.Ticker,
	tickerSearch *time.Ticker,
	tickerMetrics *time.Ticker,
	err error,
) {
	if tempoWriteBackoffDuration <= 0 {
		return nil, nil, nil, nil, errors.New("tempo-write-backoff-duration must be greater than 0")
	}
	tickerWrite = time.NewTicker(tempoWriteBackoffDuration)
	if tempoReadBackoffDuration > 0 {
		tickerRead = time.NewTicker(tempoReadBackoffDuration)
	}
	if tempoSearchBackoffDuration > 0 {
		tickerSearch = time.NewTicker(tempoSearchBackoffDuration)
	}
	if tempoMetricsBackoffDuration > 0 {
		tickerMetrics = time.NewTicker(tempoMetricsBackoffDuration)
	}
	if tickerRead == nil && tickerSearch == nil && tickerMetrics == nil {
		return nil, nil, nil, nil, errors.New("at least one of tempo-search-backoff-duration, tempo-read-backoff-duration or tempo-metrics-backoff-duration must be set")
	}
	return tickerWrite, tickerRead, tickerSearch, tickerMetrics, nil
}

// Don't attempt to read on the first iteration if we can't reasonably
// expect the write loop to have fired yet.  Double the duration here to
// avoid a race.
func traceIsReady(info *util.TraceInfo, now time.Time, startTime time.Time, writeBackoff time.Duration, longBackoff time.Duration) bool {
	if info.Timestamp().Before(startTime.Add(2 * writeBackoff)) {
		return false
	}

	return info.Ready(now, writeBackoff, longBackoff)
}

func doWrite(jaegerClient util.JaegerClient, tickerWrite *time.Ticker, interval time.Duration, config vultureConfiguration, l *zap.Logger) {
	go func() {
		for now := range tickerWrite.C {
			timestamp := now.Round(interval)
			info := util.NewTraceInfo(timestamp, config.tempoOrgID)

			logger := l.With(
				zap.String("org_id", config.tempoOrgID),
				zap.Int64("seed", info.Timestamp().Unix()),
			)

			logger.Info("sending trace")

			err := info.EmitBatches(jaegerClient)
			if err != nil {
				metricErrorTotal.Inc()
			}
			queueFutureBatches(jaegerClient, info, config, l)
		}
	}()
}

func queueFutureBatches(client util.JaegerClient, info *util.TraceInfo, config vultureConfiguration, l *zap.Logger) {
	if info.LongWritesRemaining() == 0 {
		return
	}

	logger := l.With(
		zap.String("org_id", config.tempoOrgID),
		zap.String("write_trace_id", info.HexID()),
		zap.Int64("seed", info.Timestamp().Unix()),
		zap.Int64("longWritesRemaining", info.LongWritesRemaining()),
	)
	logger.Info("queueing future batches")

	info.Done()

	go func() {
		time.Sleep(config.tempoLongWriteBackoffDuration)

		logger := l.With(
			zap.String("org_id", config.tempoOrgID),
			zap.String("write_trace_id", info.HexID()),
			zap.Int64("seed", info.Timestamp().Unix()),
			zap.Int64("longWritesRemaining", info.LongWritesRemaining()),
		)
		logger.Info("sending trace")

		err := info.EmitBatches(client)
		if err != nil {
			logger.Error("failed to queue batches",
				zap.Error(err),
			)
		}

		queueFutureBatches(client, info, config, l)
	}()
}

func runChecker(
	ticker *time.Ticker,
	config vultureConfiguration,
	selectPastTimestamp func(now time.Time) (newStart, ts time.Time, skip bool),
	checker func(*util.TraceInfo, *zap.Logger),
	l *zap.Logger,
) {
	if ticker == nil {
		return
	}
	go func() {
		for now := range ticker.C {
			startTime, seed, skip := selectPastTimestamp(now)
			if skip {
				continue
			}

			logger := l.With(
				zap.String("org_id", config.tempoOrgID),
				zap.Int64("seed", seed.Unix()),
			)

			info := util.NewTraceInfo(seed, config.tempoOrgID)

			// Don't query for a trace we don't expect to be complete
			if !traceIsReady(info, now, startTime,
				config.tempoWriteBackoffDuration, config.tempoLongWriteBackoffDuration) {
				continue
			}

			checker(info, logger)
		}
	}()
}

func doRead(httpClient httpclient.TempoHTTPClient, _ vultureConfiguration, info *util.TraceInfo, l *zap.Logger) {
	// query the trace
	queryMetrics, err := queryTrace(httpClient, info, l)
	if err != nil {
		metricErrorTotal.Inc()
		logger.Error("query for metrics failed",
			zap.Error(err),
		)
	}
	pushVultureMetrics(queryMetrics)
}

func doSearch(httpClient httpclient.TempoHTTPClient, config vultureConfiguration, info *util.TraceInfo, l *zap.Logger) {
	// query a tag we expect the trace to be found within
	searchMetrics, err := searchTag(httpClient, info.Timestamp(), config, l)
	if err != nil {
		metricErrorTotal.Inc()
		logger.Error("search tag for metrics failed",
			zap.Error(err),
		)
	}
	pushVultureMetrics(searchMetrics)

	// traceql query
	traceqlSearchMetrics, err := searchTraceql(httpClient, info.Timestamp(), config, l)
	if err != nil {
		metricErrorTotal.Inc()
		logger.Error("traceql query for metrics failed",
			zap.Error(err),
		)
	}
	pushVultureMetrics(traceqlSearchMetrics)
}

func doMetrics(httpClient httpclient.TempoHTTPClient, config vultureConfiguration, info *util.TraceInfo, l *zap.Logger) {
	m, err := queryMetrics(httpClient, info.Timestamp(), config, l)
	if err != nil {
		logger.Error("query metrics failed", zap.Error(err))
	}
	pushVultureMetrics(m)
}

func pushVultureMetrics(metrics traceMetrics) {
	metricTracesInspected.Add(float64(metrics.requested))
	metricTracesErrors.WithLabelValues("incorrectresult").Add(float64(metrics.incorrectResult))
	metricTracesErrors.WithLabelValues("incorrect_metrics_result").Add(float64(metrics.incorrectMetricsResult))
	metricTracesErrors.WithLabelValues("missingspans").Add(float64(metrics.missingSpans))
	metricTracesErrors.WithLabelValues("notfound_search").Add(float64(metrics.notFoundSearch))
	metricTracesErrors.WithLabelValues("notfound_traceql").Add(float64(metrics.notFoundTraceQL))
	metricTracesErrors.WithLabelValues("notfound_byid").Add(float64(metrics.notFoundByID))
	metricTracesErrors.WithLabelValues("notfound_metrics").Add(float64(metrics.notFoundByMetrics))
	metricTracesErrors.WithLabelValues("inaccurate_metrics").Add(float64(metrics.inaccurateMetrics))
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

func generateRandomInt(min, max int64, r *rand.Rand) int64 {
	min++
	var duration int64
	duration = 1
	// This is to prevent a panic when min == max since subtracting them will end in a negative number
	if min < max {
		duration = max - min
	}
	number := min + r.Int63n(duration)
	return number
}

func traceInTraces(traceID string, traces []*tempopb.TraceSearchMetadata) bool {
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

func searchTag(client httpclient.TempoHTTPClient, seed time.Time, config vultureConfiguration, l *zap.Logger) (traceMetrics, error) {
	tm := traceMetrics{
		requested: 1,
	}

	info := util.NewTraceInfo(seed, config.tempoOrgID)
	hexID := info.HexID()

	// Get the expected
	expected, err := info.ConstructTraceFromEpoch()
	if err != nil {
		l.Error("unable to construct trace from epoch", zap.Error(err))
		return traceMetrics{}, err
	}

	attr := util.RandomAttrFromTrace(expected)
	if attr == nil {
		tm.notFoundSearchAttribute++
		return tm, fmt.Errorf("no search attr selected from trace")
	}

	logger := l.With(
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

func searchTraceql(client httpclient.TempoHTTPClient, seed time.Time, config vultureConfiguration, l *zap.Logger) (traceMetrics, error) {
	tm := traceMetrics{
		requested: 1,
	}

	info := util.NewTraceInfo(seed, config.tempoOrgID)
	hexID := info.HexID()

	// Get the expected
	expected, err := info.ConstructTraceFromEpoch()
	if err != nil {
		l.Error("unable to construct trace from epoch", zap.Error(err))
		return traceMetrics{}, err
	}

	attr := util.RandomAttrFromTrace(expected)
	if attr == nil {
		tm.notFoundSearchAttribute++
		return tm, fmt.Errorf("no search attr selected from trace")
	}

	logger := l.With(
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

func queryTrace(client httpclient.TempoHTTPClient, info *util.TraceInfo, l *zap.Logger) (traceMetrics, error) {
	tm := traceMetrics{
		requested: 1,
	}

	hexID := info.HexID()
	start := info.Timestamp().Add(-30 * time.Minute).Unix()
	end := info.Timestamp().Add(30 * time.Minute).Unix()

	logger := l.With(
		zap.Int64("seed", info.Timestamp().Unix()),
		zap.String("hexID", hexID),
		zap.Duration("ago", time.Since(info.Timestamp())),
	)
	logger.Info("querying Tempo trace")

	// We want to define a time range to reduce the number of lookups
	trace, err := client.QueryTraceWithRange(hexID, start, end)
	if err != nil {
		if errors.Is(err, util.ErrTraceNotFound) {
			tm.notFoundByID++
		} else {
			tm.requestFailed++
		}
		logger.Error("error querying Tempo", zap.Error(err))
		return tm, err
	}

	if len(trace.ResourceSpans) == 0 {
		logger.Error("trace contains 0 batches")
		tm.notFoundByID++
		return tm, nil
	}

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

// queryMetrics performs a TraceQL metrics query and verifies the results.
// It is a basic smoke test to ensure that the traceql query is working.
// It randomly selects an attribute from an expected trace and queries for its presence
// in the metrics API within a time window around the seed time.
func queryMetrics(client httpclient.TempoHTTPClient, seed time.Time, config vultureConfiguration, l *zap.Logger) (traceMetrics, error) {
	tm := traceMetrics{
		requested: 1,
	}

	info := util.NewTraceInfo(seed, config.tempoOrgID)
	hexID := info.HexID()

	expected, err := info.ConstructTraceFromEpoch()
	if err != nil {
		err = fmt.Errorf("unable to construct trace from epoch: %w", err)
		return traceMetrics{}, err
	}

	attr := util.RandomAttrFromTrace(expected)
	if attr == nil {
		tm.notFoundSearchAttribute++
		return tm, fmt.Errorf("no search attr selected from trace")
	}

	logger := l.With(
		zap.Int64("seed", seed.Unix()),
		zap.String("hexID", hexID),
		zap.Duration("ago", time.Since(seed)),
		zap.String("key", attr.Key),
		zap.String("value", util.StringifyAnyValue(attr.Value)),
	)
	logger.Info("searching Tempo via metrics")

	// Use the API to find details about the expected trace. give an hour range around the seed.
	start := seed.Add(-30 * time.Minute).Unix()
	end := seed.Add(30 * time.Minute).Unix()

	resp, err := client.MetricsQueryRange(
		fmt.Sprintf(`{.%s = "%s"} | count_over_time()`, attr.Key, util.StringifyAnyValue(attr.Value)),
		int(start), int(end), "1m", 0,
	)
	if err != nil {
		logger.Error("failed to query metrics", zap.Error(err))
		tm.requestFailed++
		return tm, err
	}

	if len(resp.Series) == 0 {
		tm.notFoundByMetrics++
		logger.Error("failed to find trace by metrics", zap.Error(err))
		return tm, fmt.Errorf("expected trace %s not found in metrics", hexID)
	}

	if len(resp.Series) > 1 {
		tm.incorrectMetricsResult++
		return tm, fmt.Errorf("expected exactly 1 series, got %d", len(resp.Series))
	}
	timeSeries := resp.Series[0]
	if timeSeries == nil {
		tm.incorrectMetricsResult++
		return tm, errors.New("expected time series, got nil")
	}

	var sum float64
	for _, sample := range timeSeries.Samples {
		sum += sample.Value
	}

	if sum < 1 {
		tm.notFoundByMetrics++
		logger.Error("failed to find trace by metrics", zap.Error(err))
		return tm, fmt.Errorf("expected trace %s not found in metrics", hexID)
	}

	// Advanced check: ensure metric results are accurate
	// by checking actual number of spans
	// skip if search check is disabled
	if config.tempoSearchBackoffDuration == 0 {
		return tm, nil
	}
	searchResp, err := client.SearchTraceQLWithRange(fmt.Sprintf(`{.%s = "%s"}`, attr.Key, util.StringifyAnyValue(attr.Value)), start, end)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to search traces with traceql %s: %s", attr.Key, err.Error()))
		tm.requestFailed++
		return tm, err
	}

	var spansCount int
	var traces []*tempopb.TraceSearchMetadata
	if searchResp != nil {
		traces = searchResp.Traces
	}
	for _, trace := range traces {
		if trace == nil {
			continue
		}
		for _, spanSet := range trace.SpanSets {
			if spanSet == nil {
				continue
			}
			spansCount += int(spanSet.Matched)
		}
	}

	const delta = 1e-6
	// if number of traces is not equal to the sum of metric values
	if math.Abs(float64(spansCount)-sum) > delta {
		tm.inaccurateMetrics++
		err = fmt.Errorf(
			"TraceQL Metrics results are inaccurate: metric count sum=%f, actual span count=%d",
			sum, spansCount,
		)
		return tm, err
	}

	return tm, nil
}

func equalTraces(a, b *tempopb.Trace) bool {
	trace.SortTraceAndAttributes(a)
	trace.SortTraceAndAttributes(b)

	return reflect.DeepEqual(a, b)
}

func hasMissingSpans(t *tempopb.Trace) bool {
	// check that all parent spans exist in the trace
	parentSpanIDs := make([][]byte, 0)

	for _, b := range t.ResourceSpans {
		for _, ss := range b.ScopeSpans {
			for _, s := range ss.Spans {
				if len(s.ParentSpanId) > 0 {
					parentSpanIDs = append(parentSpanIDs, s.ParentSpanId)
				}
			}
		}
	}

	for _, id := range parentSpanIDs {
		found := false

	B:
		for _, b := range t.ResourceSpans {
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
