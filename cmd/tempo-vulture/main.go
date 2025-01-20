package main

import (
	"bytes"
	"context"
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
	"strconv"
	"time"

	"github.com/go-test/deep"
	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grafana/dskit/user"

	testUtil "github.com/grafana/tempo/integration/util"
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

const (
	defaultJaegerGRPCEndpoint = 14250
	specialSpanName           = "specialVultureSpanName"
	specialServiceName        = "specialVultureServiceName"
	specialAttributeKey       = "specialVultureAttributeKey"
	specialAttributeValue     = "specialVultureAttributeValue"
)

type vultureConfiguration struct {
	tempoQueryURL                 string
	tempoPushURL                  string
	tempoOrgID                    string
	tempoWriteBackoffDuration     time.Duration
	tempoLongWriteBackoffDuration time.Duration
	tempoReadBackoffDuration      time.Duration
	tempoSearchBackoffDuration    time.Duration
	tempoRetentionDuration        time.Duration
	tempoPushTLS                  bool
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

	vultureConfig := vultureConfiguration{
		tempoQueryURL:                 tempoQueryURL,
		tempoPushURL:                  tempoPushURL,
		tempoOrgID:                    tempoOrgID,
		tempoWriteBackoffDuration:     tempoWriteBackoffDuration,
		tempoLongWriteBackoffDuration: tempoLongWriteBackoffDuration,
		tempoReadBackoffDuration:      tempoReadBackoffDuration,
		tempoSearchBackoffDuration:    tempoSearchBackoffDuration,
		tempoRetentionDuration:        tempoRetentionDuration,
		tempoPushTLS:                  tempoPushTLS,
	}

	jaegerClient, err := newJaegerGRPCClient(vultureConfig, logger)
	if err != nil {
		panic(err)
	}
	httpClient := httpclient.New(vultureConfig.tempoQueryURL, vultureConfig.tempoOrgID)

	//tickerWrite, tickerRead, tickerSearch, err := initTickers(vultureConfig.tempoWriteBackoffDuration, vultureConfig.tempoReadBackoffDuration, vultureConfig.tempoSearchBackoffDuration)
	// if err != nil {
	// 	panic(err)
	// }
	startTime := time.Now()
	r := rand.New(rand.NewSource(startTime.Unix()))
	//interval := vultureConfig.tempoWriteBackoffDuration

	//doWrite(jaegerClient, tickerWrite, interval, vultureConfig, logger)
	//doRead(httpClient, tickerRead, startTime, interval, r, vultureConfig, logger)
	//doSearch(httpClient, tickerSearch, startTime, interval, r, vultureConfig, logger)
	doLongTests(jaegerClient, httpClient, vultureConfig, *r, logger)

	http.Handle(prometheusPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(prometheusListenAddress, nil))
}

func getGRPCEndpoint(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	dialAddress := u.Host

	if u.Port() == "" {
		dialAddress = fmt.Sprintf("%s:%d", dialAddress, defaultJaegerGRPCEndpoint)
	}
	return dialAddress, nil
}

func initTickers(tempoWriteBackoffDuration time.Duration, tempoReadBackoffDuration time.Duration, tempoSearchBackoffDuration time.Duration) (tickerWrite *time.Ticker, tickerRead *time.Ticker, tickerSearch *time.Ticker, err error) {
	if tempoWriteBackoffDuration <= 0 {
		return nil, nil, nil, errors.New("tempo-write-backoff-duration must be greater than 0")
	}
	tickerWrite = time.NewTicker(tempoWriteBackoffDuration)
	if tempoReadBackoffDuration > 0 {
		tickerRead = time.NewTicker(tempoReadBackoffDuration)
	}
	if tempoSearchBackoffDuration > 0 {
		tickerSearch = time.NewTicker(tempoSearchBackoffDuration)
	}
	if tickerRead == nil && tickerSearch == nil {
		return nil, nil, nil, errors.New("at least one of tempo-search-backoff-duration or tempo-read-backoff-duration must be set")
	}
	return tickerWrite, tickerRead, tickerSearch, nil
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

func doRead(httpClient httpclient.TempoHTTPClient, tickerRead *time.Ticker, startTime time.Time, interval time.Duration, r *rand.Rand, config vultureConfiguration, l *zap.Logger) {
	if tickerRead != nil {
		go func() {
			for now := range tickerRead.C {
				var seed time.Time
				startTime, seed = selectPastTimestamp(startTime, now, interval, tempoRetentionDuration, r)

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

				// query the trace
				queryMetrics, err := queryTrace(httpClient, info, l)
				if err != nil {
					metricErrorTotal.Inc()
					logger.Error("query for metrics failed",
						zap.Error(err),
					)
				}
				pushMetrics(queryMetrics)
			}
		}()
	}
}

func doSearch(httpClient httpclient.TempoHTTPClient, tickerSearch *time.Ticker, startTime time.Time, interval time.Duration, r *rand.Rand, config vultureConfiguration, l *zap.Logger) {
	if tickerSearch != nil {
		go func() {
			for now := range tickerSearch.C {
				_, seed := selectPastTimestamp(startTime, now, interval, config.tempoRetentionDuration, r)
				logger := l.With(
					zap.String("org_id", config.tempoOrgID),
					zap.Int64("seed", seed.Unix()),
				)

				info := util.NewTraceInfo(seed, config.tempoOrgID)

				if !traceIsReady(info, now, startTime,
					config.tempoWriteBackoffDuration, config.tempoLongWriteBackoffDuration) {
					continue
				}

				// query a tag we expect the trace to be found within
				searchMetrics, err := searchTag(httpClient, seed, config, l)
				if err != nil {
					metricErrorTotal.Inc()
					logger.Error("search tag for metrics failed",
						zap.Error(err),
					)
				}
				pushMetrics(searchMetrics)

				// traceql query
				traceqlSearchMetrics, err := searchTraceql(httpClient, seed, config, l)
				if err != nil {
					metricErrorTotal.Inc()
					logger.Error("traceql query for metrics failed",
						zap.Error(err),
					)
				}
				pushMetrics(traceqlSearchMetrics)
			}
		}()
	}
}

type SpanTracker struct {
	count                  int
	timeStamps             []int64
	spanCountSameName      []int
	spanCountSameService   []int
	spanCountSameAttribute []int
	spanName               string
	serviceName            string
	attributeKey           string
	attributeValue         string
}

func newSpanTracker() *SpanTracker {
	timeNow := strconv.Itoa(int(time.Now().UnixMilli()))
	return &SpanTracker{
		serviceName:    specialServiceName + timeNow,
		spanName:       specialSpanName + timeNow,
		attributeKey:   specialAttributeKey,
		attributeValue: specialAttributeValue + timeNow,
	}
}

func (st *SpanTracker) MakeBatch(r rand.Rand, l *zap.Logger) *thrift.Batch {
	// make starting batch with service name
	startingSpanCount := r.Intn(4) + 1
	batch := testUtil.MakeThriftBatchWithSpanCountResourceAndSpanAttr(startingSpanCount, st.serviceName, "span-name", "vulture", "vulture", "key", st.attributeKey)

	// reuse trace id & start time
	traceIDHigh := batch.Spans[0].TraceIdHigh
	traceIDLow := batch.Spans[0].TraceIdLow
	startTime := batch.Spans[0].StartTime

	// inject more spans with special name
	spanNameSpanCount := r.Intn(4) + 1
	for i := 0; i < spanNameSpanCount; i++ {
		batch.Spans = append(batch.Spans, &thrift.Span{
			TraceIdLow:    traceIDLow,
			TraceIdHigh:   traceIDHigh,
			SpanId:        rand.Int63(),
			ParentSpanId:  0,
			OperationName: st.spanName,
			Flags:         0,
			StartTime:     startTime,
			Duration:      1,
		})
	}

	// inject more spans with special attribute
	spanAttributeSpanCount := r.Intn(4) + 1
	for i := 0; i < spanAttributeSpanCount; i++ {
		batch.Spans = append(batch.Spans, &thrift.Span{
			TraceIdLow:    traceIDLow,
			TraceIdHigh:   traceIDHigh,
			SpanId:        rand.Int63(),
			ParentSpanId:  0,
			OperationName: "my operation",
			Flags:         0,
			StartTime:     startTime,
			Duration:      1,
			Tags: []*thrift.Tag{
				{
					Key:  st.attributeKey,
					VStr: &st.attributeValue,
				},
			},
		})
	}
	st.count++
	st.timeStamps = append(st.timeStamps, startTime)
	st.spanCountSameName = append(st.spanCountSameName, spanNameSpanCount)
	st.spanCountSameAttribute = append(st.spanCountSameAttribute, spanAttributeSpanCount)
	st.spanCountSameService = append(st.spanCountSameService, startingSpanCount+spanNameSpanCount+spanAttributeSpanCount)
	return batch
}

func (st *SpanTracker) GetSpanCount(scenario string, start, end int) int {
	tracker := st.spanCountSameName
	if scenario == "service" {
		tracker = st.spanCountSameService
	} else if scenario == "attribute" {
		tracker = st.spanCountSameAttribute
	}

	spanCount := 0
	for i := start; i <= end; i++ {
		spanCount += tracker[i]
	}
	return spanCount
}

func (st *SpanTracker) GetSpanCounts(scenario string, start, end int) []int {
	tracker := st.spanCountSameName
	if scenario == "service" {
		tracker = st.spanCountSameService
	} else if scenario == "attribute" {
		tracker = st.spanCountSameAttribute
	}

	spanCounts := make([]int, end-start + 1)
	for i := start; i <= end; i++ {
		spanCounts[i-start] = tracker[i]
	}
	return spanCounts
}

func (st *SpanTracker) GetTraceCount(start, end int) int {
	count := 0

	for i := start; i <= end; i++ {
		if st.spanCountSameService[i] != 0 {
			count++
		}
	}
	return count
}

func (st *SpanTracker) GetRandomStartEndPosition(r rand.Rand) (int, int) {
	// with rhythm we are ingesting spans at a higher latency than before
	// so we are allowing 3 minutes slack time (by excluding the last 3 values)
	// choosing a random start and end time with at least 20 data points
	count := len(st.timeStamps) - 3
	start := r.Intn(count - 20) //  ensuring 20 counts
	end := 20 + start
	return start, end // the end is inclusive
}

func (st *SpanTracker) ValidateTraceQLSearches(tempoClient httpclient.TempoHTTPClient, startPosition, endPosition int, logger *zap.Logger) error {
	serviceQuery := fmt.Sprintf(`{resource.service.name = "%s"}`, st.serviceName)
	nameQuery := fmt.Sprintf(`{span:name = "%s"}`, st.spanName)
	attributeQuery := fmt.Sprintf(`{span.%s = "%s"}`, st.attributeKey, st.attributeValue)

	scenarios := []string{serviceQuery, nameQuery, attributeQuery}

	for _, scenario := range scenarios {
		queryType := "service"
		if scenario == nameQuery {
			queryType = "name"
		} else if scenario == attributeQuery {
			queryType = "attribute"
		}

		// add some slack to start and end time in the query
		start := time.UnixMicro(st.timeStamps[startPosition]).Add(-1 * time.Second).Unix()
		end := time.UnixMicro(st.timeStamps[endPosition]).Add(1 * time.Second).Unix()
		resp, err := tempoClient.SearchTraceQLWithRangeAndLimit(scenario, start, end, 5000, 100)
		if err != nil {
			logger.Error("error searching Tempo traceql query", zap.Error(err))
			return err
		}
		pass := true
		expectedCount := st.GetTraceCount(startPosition, endPosition)
		if len(resp.Traces) != expectedCount {
			pass = false
			logger.Error("incorrect number of traces returned", zap.String("scenarios", scenario), zap.Int("expected", expectedCount), zap.Int("actual", len(resp.Traces)))
		}
		actualSpanCount := 0
		for _, trace := range resp.Traces {
			for _, spanset := range trace.SpanSets{
				actualSpanCount += len(spanset.Spans)
			}
		}
		expectedSpanCount := st.GetSpanCount(queryType, startPosition, endPosition)
		if actualSpanCount != expectedSpanCount {
			pass = false
			logger.Error("incorrect number of spans returned", zap.String("scenarios", scenario), zap.Int("expected", expectedSpanCount), zap.Int("actual", actualSpanCount))
			// metricTracesErrors.WithLabelValues("notfound_search_attribute").Add(float64(metrics.notFoundSearchAttribute))
		}

		if !pass {
			metricTracesErrors.WithLabelValues("traceql_incorrect_result").Add(float64(1))
		}
	}
	return nil
}

func (st *SpanTracker) ValidateTraceQLMetricsSearches(tempoClient httpclient.TempoHTTPClient, startPosition, endPosition int, ticketDuration time.Duration, logger *zap.Logger) error{
	serviceQuery := fmt.Sprintf(`{resource.service.name = "%s"} | rate()`, st.serviceName)
	nameQuery := fmt.Sprintf(`{span:name = "%s"} | rate()`, st.spanName)
	attributeQuery := fmt.Sprintf(`{span.%s = "%s"} | rate()`, st.attributeKey, st.attributeValue)

	scenarios := []string{serviceQuery, nameQuery, attributeQuery}

	for _, scenario := range scenarios {
		queryType := "service"
		if scenario == nameQuery {
			queryType = "name"
		} else if scenario == attributeQuery {
			queryType = "attribute"
		}
		spanCounts := st.GetSpanCounts(queryType, startPosition, endPosition)
		start := time.UnixMicro(st.timeStamps[startPosition]).Add(-1 * time.Second).Unix()
		end := time.UnixMicro(st.timeStamps[endPosition]).Add(1 * time.Second).Unix()
		if end > time.Now().Unix() {
			end = time.Now().Unix()
		}
		stepSecond := ticketDuration.Seconds()
		step := int64(ticketDuration)
		
		resp, err := tempoClient.SearchQueryRange(scenario, start, end, step)
		if err != nil {
			logger.Error("error searching Tempo query range query", zap.Error(err), zap.String("query", scenario))	
			return err
		}
		// since we send and record count every 30 seconds and we set the step to 30 seconds
		// we expect the count to be the same between span tracker count and time series
		pass := true
		for i, sample := range resp.Series[0].Samples {
			if i >= len(spanCounts) { continue } // for when start/end time creates additional samples
			expectedSpanCountRate := float64(spanCounts[i])/stepSecond
			if (sample.Value != 0 && sample.Value != expectedSpanCountRate) || (sample.Value == 0 && spanCounts[i] != 0) {
				logger.Error("incorrect number of spans returned for query range test", zap.String("scenarios", scenario), zap.Float64("expected", expectedSpanCountRate), zap.Float64("actual", sample.Value))
			}
		}
		if !pass {
			metricTracesErrors.WithLabelValues("metrics_query_incorrect_result").Add(float64(1))
		}


	}
	return nil
}

func doLongTests(jaegerClient util.JaegerClient, tempoClient httpclient.TempoHTTPClient, config vultureConfiguration, r rand.Rand, l *zap.Logger) {

	// run every 30 seconds
	ticketDuration := time.Duration(30) * time.Second
	ticker := time.NewTicker(ticketDuration)
	spanTracker := newSpanTracker()

	go func() {
		for range ticker.C {
			// create a new span tracker every 500 times to clear out old data
			if len(spanTracker.timeStamps) >= 500 {
				spanTracker = newSpanTracker()
			}

			// emit traces and keep track of span counts
			ctx := user.InjectOrgID(context.Background(), config.tempoOrgID)
			ctx, err := user.InjectIntoGRPCRequest(ctx)
			if err != nil {
				logger.Error("error injecting org id", zap.Error(err))
				continue
			}

			batch := spanTracker.MakeBatch(r, l)
			err = jaegerClient.EmitBatch(ctx, batch)
			if err != nil {
				logger.Error("error pushing batch to Tempo", zap.Error(err))
				// don't record the last span count if it failed but still record the timestamps for metrics queries
				spanTracker.spanCountSameName[len(spanTracker.spanCountSameName)-1] = 0
				spanTracker.spanCountSameService[len(spanTracker.spanCountSameService)-1] = 0
				spanTracker.spanCountSameAttribute[len(spanTracker.spanCountSameAttribute)-1] = 0
				spanTracker.count--
				continue
			}
			logger.Info("pushed batch to Tempo", zap.Int("count", spanTracker.count))

			// only search after at least 30 pushes
			if spanTracker.count < 30 {
				logger.Info("pushed only", zap.Int("count", spanTracker.count))
				continue
			}

			// choose random start/end for searches (the end position is inclusive)
			startPosition, endPosition := spanTracker.GetRandomStartEndPosition(r)
			logger.Info("random positions", zap.Int("start", startPosition), zap.Int("end", endPosition))
			
			// traceql
			spanTracker.ValidateTraceQLSearches(tempoClient, startPosition, endPosition, l)

			// metrics searches
			spanTracker.ValidateTraceQLMetricsSearches(tempoClient, startPosition, endPosition, ticketDuration, l)

		}
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

func newJaegerGRPCClient(config vultureConfiguration, logger *zap.Logger) (*jaeger_grpc.Reporter, error) {
	endpoint, err := getGRPCEndpoint(config.tempoPushURL)
	if err != nil {
		return nil, err
	}

	logger.Info("dialing grpc",
		zap.String("endpoint", endpoint),
	)

	var dialOpts []grpc.DialOption

	if config.tempoPushTLS {
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
	conn, err := grpc.NewClient(endpoint, dialOpts...)
	if err != nil {
		return nil, err
	}
	return jaeger_grpc.NewReporter(conn, nil, logger), nil
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
		logger.Error("unable to construct trace from epoch", zap.Error(err))
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

	for _, b := range t.ResourceSpans {
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
