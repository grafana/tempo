package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	integrationutil "github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	util "github.com/grafana/tempo/pkg/util"
)

func TestHasMissingSpans(t *testing.T) {
	cases := []struct {
		trace    *tempopb.Trace
		expected bool
	}{
		{
			&tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{
										ParentSpanId: []byte("01234"),
									},
								},
							},
						},
					},
				},
			},
			true,
		},
		{
			&tempopb.Trace{
				ResourceSpans: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: []byte("01234"),
									},
									{
										ParentSpanId: []byte("01234"),
									},
								},
							},
						},
					},
				},
			},
			false,
		},
	}

	for _, tc := range cases {
		require.Equal(t, tc.expected, hasMissingSpans(tc.trace))
	}
}

func TestResponseFixture(t *testing.T) {
	f, err := os.Open("testdata/trace.json")
	require.NoError(t, err)
	defer f.Close()

	expected := &tempopb.Trace{}
	err = jsonpb.Unmarshal(f, expected)
	require.NoError(t, err)

	seed := time.Unix(1636729665, 0)
	info := util.NewTraceInfo(seed, "")

	generatedTrace, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	// print the generated trace
	var jsonTrace bytes.Buffer
	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(&jsonTrace, generatedTrace)
	require.NoError(t, err)

	assert.True(t, equalTraces(expected, generatedTrace))

	if diff := deep.Equal(expected, generatedTrace); diff != nil {
		for _, d := range diff {
			t.Error(d)
		}
	}
}

func TestEqualTraces(t *testing.T) {
	seed := time.Now()
	info1 := util.NewTraceInfo(seed, "")
	info2 := util.NewTraceInfo(seed, "")

	a, err := info1.ConstructTraceFromEpoch()
	require.NoError(t, err)
	b, err := info2.ConstructTraceFromEpoch()
	require.NoError(t, err)

	require.True(t, equalTraces(a, b))

	// Subsequent calls also reconstruct identical traces
	c, err := info1.ConstructTraceFromEpoch()
	require.NoError(t, err)
	require.True(t, equalTraces(b, c))
}

func TestInitTickers(t *testing.T) {
	tests := []struct {
		name                            string
		writeDuration, readDuration     time.Duration
		searchDuration, metricsDuration time.Duration
		expectedWriteTicker             bool
		expectedReadTicker              bool
		expectedSearchTicker            bool
		expectedMetricsTicker           bool
		expectedError                   string
	}{
		{
			name:                  "Valid write and read durations",
			writeDuration:         1 * time.Second,
			readDuration:          2 * time.Second,
			searchDuration:        0,
			expectedWriteTicker:   true,
			expectedReadTicker:    true,
			expectedSearchTicker:  false,
			expectedMetricsTicker: false,
			expectedError:         "",
		},
		{
			name:                  "Invalid write duration (zero)",
			writeDuration:         0,
			readDuration:          0,
			searchDuration:        0,
			expectedWriteTicker:   false,
			expectedReadTicker:    false,
			expectedSearchTicker:  false,
			expectedMetricsTicker: false,
			expectedError:         "tempo-write-backoff-duration must be greater than 0",
		},
		{
			name:                  "No read durations set",
			writeDuration:         1 * time.Second,
			readDuration:          0,
			searchDuration:        1 * time.Second,
			expectedWriteTicker:   true,
			expectedReadTicker:    false,
			expectedSearchTicker:  true,
			expectedMetricsTicker: false,
			expectedError:         "",
		},
		{
			name:                  "Valid metrics duration",
			writeDuration:         1 * time.Second,
			readDuration:          0,
			searchDuration:        0,
			metricsDuration:       1 * time.Second,
			expectedWriteTicker:   true,
			expectedReadTicker:    false,
			expectedSearchTicker:  false,
			expectedMetricsTicker: true,
			expectedError:         "",
		},
		{
			name:                  "No read or search durations set",
			writeDuration:         1 * time.Second,
			readDuration:          0,
			searchDuration:        0,
			expectedWriteTicker:   false,
			expectedReadTicker:    false,
			expectedSearchTicker:  false,
			expectedMetricsTicker: false,
			expectedError:         "at least one of tempo-search-backoff-duration, tempo-read-backoff-duration or tempo-metrics-backoff-duration must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tickerWrite, tickerRead, tickerSearch, tickerMetrics, err := initTickers(tt.writeDuration, tt.readDuration, tt.searchDuration, tt.metricsDuration)

			assert.Equal(t, tt.expectedWriteTicker, tickerWrite != nil, "TickerWrite")
			assert.Equal(t, tt.expectedReadTicker, tickerRead != nil, "TickerRead")
			assert.Equal(t, tt.expectedSearchTicker, tickerSearch != nil, "TickerSearch")
			assert.Equal(t, tt.expectedMetricsTicker, tickerMetrics != nil, "TickerMetrics")

			if tt.expectedError != "" {
				assert.NotNil(t, err, "Expected error but got nil")
				assert.EqualError(t, err, tt.expectedError, "Error message mismatch")
			} else {
				assert.Nil(t, err, "Expected no error but got one")
			}
		})
	}
}

func TestTraceIsReady(t *testing.T) {
	writeBackoff := 1 * time.Second
	longWriteBackoff := 5 * time.Second
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	ti := util.NewTraceInfo(seed, "test")

	startTime := time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC)
	ready := traceIsReady(ti, time.Now(), startTime, writeBackoff, longWriteBackoff)

	assert.False(t, ready, "trace should not be ready yet")

	startTime = time.Date(2007, 1, 1, 12, 0, 0, 0, time.UTC)
	ready = traceIsReady(ti, seed.Add(2*longWriteBackoff), startTime, writeBackoff, longWriteBackoff)
	assert.True(t, ready, "trace should be ready now")
}

func TestDoWrite(t *testing.T) {
	mockJaegerClient := MockReporter{err: nil}
	// Define the configuration
	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	logger = zap.NewNop()

	doWrite(&mockJaegerClient, ticker, config.tempoWriteBackoffDuration, config, logger)

	time.Sleep(time.Second)
	ticker.Stop()
	assert.Greater(t, len(mockJaegerClient.GetEmittedBatches()), 0)
}

func TestDoWriteWithError(t *testing.T) {
	mockJaegerClient := MockReporter{err: errors.New("an error")}
	// Define the configuration
	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	logger = zap.NewNop()

	doWrite(&mockJaegerClient, ticker, config.tempoWriteBackoffDuration, config, logger)
	ticker.Stop()
	assert.Equal(t, len(mockJaegerClient.GetEmittedBatches()), 0)
}

func TestQueueFutureBatches(t *testing.T) {
	mockJaegerClient := MockReporter{err: nil}
	// Define the configuration
	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second * 0,
	}

	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	traceInfo := util.NewTraceInfoWithMaxLongWrites(seed, 1, "test")
	logger = zap.NewNop()

	queueFutureBatches(&mockJaegerClient, traceInfo, config, logger)
	time.Sleep(time.Second)
	require.Greater(t, len(mockJaegerClient.GetEmittedBatches()), 0)

	// Assert an error
	mockJaegerClient = MockReporter{err: errors.New("an error")}

	queueFutureBatches(&mockJaegerClient, traceInfo, config, logger)
	time.Sleep(time.Second)
	require.Equal(t, len(mockJaegerClient.batchesEmitted), 0)
}

type traceOps func(*tempopb.Trace)

func TestQueryTrace(t *testing.T) {
	noOp := func(_ *tempopb.Trace) {}
	setMissingSpan := func(trace *tempopb.Trace) {
		trace.ResourceSpans[0].ScopeSpans[0].Spans[0].ParentSpanId = []byte{'t', 'e', 's', 't'}
	}
	setNoBatchesSpan := func(trace *tempopb.Trace) {
		trace.ResourceSpans = make([]*v1.ResourceSpans, 0)
	}
	setAlteredSpan := func(trace *tempopb.Trace) {
		trace.ResourceSpans[0].ScopeSpans[0].Spans[0].Name = "Different spam"
	}
	tests := []struct {
		name            string
		traceOperation  func(*tempopb.Trace)
		err             error
		expectedMetrics traceMetrics
		expectedError   error
	}{
		{
			name:           "assert querytrace yields an unexpected error",
			traceOperation: noOp,
			err:            errors.New("unexpected error"),
			expectedMetrics: traceMetrics{
				requested:     1,
				requestFailed: 1,
			},
			expectedError: errors.New("unexpected error"),
		},
		{
			name:           "assert querytrace yields traceNotFound error",
			traceOperation: noOp,
			err:            util.ErrTraceNotFound,
			expectedMetrics: traceMetrics{
				requested:    1,
				notFoundByID: 1,
			},
			expectedError: util.ErrTraceNotFound,
		},
		{
			name:           "assert querytrace for ok trace",
			traceOperation: noOp,
			err:            nil,
			expectedMetrics: traceMetrics{
				requested: 1,
			},
			expectedError: nil,
		},
		{
			name:           "assert querytrace for a trace with missing spans",
			traceOperation: setMissingSpan,
			err:            nil,
			expectedMetrics: traceMetrics{
				requested:    1,
				missingSpans: 1,
			},
			expectedError: nil,
		},
		{
			name:           "assert querytrace for a trace without batches",
			traceOperation: setNoBatchesSpan,
			err:            nil,
			expectedMetrics: traceMetrics{
				requested:    1,
				notFoundByID: 1,
			},
			expectedError: nil,
		},
		{
			name:           "assert querytrace for a trace different than the ingested one",
			traceOperation: setAlteredSpan,
			err:            nil,
			expectedMetrics: traceMetrics{
				requested:       1,
				incorrectResult: 1,
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := doQueryTrace(tt.traceOperation, tt.err)
			assert.Equal(t, tt.expectedMetrics, metrics)
			assert.Equal(t, tt.expectedError, err)
		})
	}
}

func doQueryTrace(f traceOps, err error) (traceMetrics, error) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	traceInfo := util.NewTraceInfo(seed, "test")

	trace, _ := traceInfo.ConstructTraceFromEpoch()

	mockHTTPClient := MockHTTPClient{err: err, traceResp: trace}
	logger = zap.NewNop()
	f(trace)
	return queryTrace(&mockHTTPClient, traceInfo, logger)
}

func TestDoReadForAnOkRead(t *testing.T) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	traceInfo := util.NewTraceInfo(seed, "test")

	trace, _ := traceInfo.ConstructTraceFromEpoch()
	mockHTTPClient := MockHTTPClient{err: nil, traceResp: trace}
	// Define the configuration
	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}

	logger = zap.NewNop()

	doRead(&mockHTTPClient, config, traceInfo, logger)
	assert.Equal(t, 1, mockHTTPClient.GetRequestsCount())
}

func TestDoReadForAnErroredRead(t *testing.T) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	traceInfo := util.NewTraceInfo(seed, "test")

	trace, _ := traceInfo.ConstructTraceFromEpoch()

	// Define the configuration
	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}

	logger = zap.NewNop()

	// Assert a read with errors
	mockHTTPClient := MockHTTPClient{err: errors.New("an error"), traceResp: trace}
	doRead(&mockHTTPClient, config, traceInfo, logger)
	assert.Equal(t, 0, mockHTTPClient.GetRequestsCount())
}

func TestSearchTraceql(t *testing.T) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)

	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}

	info := util.NewTraceInfo(seed, config.tempoOrgID)
	hexID := info.HexID()

	searchResponse := []*tempopb.TraceSearchMetadata{
		{
			SpanSets: []*tempopb.SpanSet{
				{
					Spans: []*tempopb.Span{
						{
							SpanID:            hexID,
							StartTimeUnixNano: 1000000000000,
							DurationNanos:     1000000000,
							Name:              "",
							Attributes: []*v1_common.KeyValue{
								{Key: "foo", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "Bar"}}},
							},
						},
					},
				},
			},
		},
	}

	mockHTTPClient := MockHTTPClient{err: nil, searchResponse: searchResponse}
	logger = zap.NewNop()

	metrics, err := searchTraceql(&mockHTTPClient, seed, config, logger)

	assert.Error(t, err)
	assert.Equal(t, traceMetrics{
		requested:       1,
		notFoundTraceQL: 1,
	}, metrics)

	mockHTTPClient = MockHTTPClient{err: errors.New("something wrong happened"), searchResponse: searchResponse}
	logger = zap.NewNop()

	metrics, err = searchTraceql(&mockHTTPClient, seed, config, logger)

	assert.Error(t, err)
	assert.Equal(t, traceMetrics{
		requested:     1,
		requestFailed: 1,
	}, metrics)
}

func TestSearchTag(t *testing.T) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)

	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}

	info := util.NewTraceInfo(seed, config.tempoOrgID)
	hexID := info.HexID()

	searchResponse := []*tempopb.TraceSearchMetadata{
		{
			SpanSets: []*tempopb.SpanSet{
				{
					Spans: []*tempopb.Span{
						{
							SpanID:            hexID,
							StartTimeUnixNano: 1000000000000,
							DurationNanos:     1000000000,
							Name:              "",
							Attributes: []*v1_common.KeyValue{
								{Key: "foo", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "Bar"}}},
							},
						},
					},
				},
			},
		},
	}

	mockHTTPClient := MockHTTPClient{err: nil, searchResponse: searchResponse}
	logger = zap.NewNop()

	metrics, err := searchTag(&mockHTTPClient, seed, config, logger)

	assert.Error(t, err)
	assert.Equal(t, traceMetrics{
		requested:      1,
		notFoundSearch: 1,
	}, metrics)

	mockHTTPClient = MockHTTPClient{err: errors.New("something wrong happened"), searchResponse: searchResponse}
	logger = zap.NewNop()

	metrics, err = searchTag(&mockHTTPClient, seed, config, logger)

	assert.Error(t, err)
	assert.Equal(t, traceMetrics{
		requested:     1,
		requestFailed: 1,
	}, metrics)
}

func TestDoSearch(t *testing.T) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	traceInfo := util.NewTraceInfo(seed, "test")

	// Define the configuration
	config := vultureConfiguration{
		tempoOrgID: "orgID",
		// This is a hack to ensure the trace is "ready"
		tempoWriteBackoffDuration: -time.Hour * 10000,
		tempoRetentionDuration:    time.Second * 10,
	}

	logger = zap.NewNop()

	searchResponse := []*tempopb.TraceSearchMetadata{
		{
			SpanSets: []*tempopb.SpanSet{
				{
					Spans: []*tempopb.Span{
						{
							SpanID:            traceInfo.HexID(),
							StartTimeUnixNano: 1000000000000,
							DurationNanos:     1000000000,
							Name:              "",
							Attributes: []*v1_common.KeyValue{
								{Key: "foo", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "Bar"}}},
							},
						},
					},
				},
			},
		},
	}

	mockHTTPClient := MockHTTPClient{err: nil, searchResponse: searchResponse}

	doSearch(&mockHTTPClient, config, traceInfo, logger)
	assert.Greater(t, mockHTTPClient.GetSearchesCount(), 0)
}

func TestDoSearchError(t *testing.T) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	traceInfo := util.NewTraceInfo(seed, "test")

	// Define the configuration
	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}

	logger = zap.NewNop()

	// Assert an errored search
	mockHTTPClient := MockHTTPClient{err: errors.New("an error")}

	doSearch(&mockHTTPClient, config, traceInfo, logger)
	assert.Equal(t, 0, mockHTTPClient.GetSearchesCount())
}

func TestGetGrpcEndpoint(t *testing.T) {
	_, err := getGRPCEndpoint("http://%gh&%ij")
	require.Error(t, err)

	got, err := getGRPCEndpoint("http://localhost:4000")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:4000", got, "Address endpoint should keep the given port")

	got, err = getGRPCEndpoint("http://localhost")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:4317", got, "Address without a port should be defaulted to 4317")
}

func TestNewJaegerToOTLPExportert(t *testing.T) {
	configValid := vultureConfiguration{
		tempoPushURL: "localhost:4317",
	}
	clientValid, errValid := integrationutil.NewJaegerToOTLPExporter(configValid.tempoPushURL)
	assert.NoError(t, errValid)
	assert.NotNil(t, clientValid)
}

func TestQueryMetrics(t *testing.T) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)

	config := vultureConfiguration{
		tempoOrgID:                 "orgID",
		tempoWriteBackoffDuration:  time.Second,
		tempoSearchBackoffDuration: time.Second,
	}

	info := util.NewTraceInfo(seed, config.tempoOrgID)
	hexID := info.HexID()

	successMetricsResponse := &tempopb.QueryRangeResponse{
		Series: []*tempopb.TimeSeries{
			{
				Samples: []tempopb.Sample{
					{
						TimestampMs: seed.UnixMilli(),
						Value:       2.0,
					},
				},
			},
		},
	}

	tests := []struct {
		name            string
		response        *tempopb.QueryRangeResponse
		searchResponse  []*tempopb.TraceSearchMetadata
		err             error
		expectedMetrics traceMetrics
		expectedError   string
	}{
		{
			name:     "successful metrics query: 1 trace, 2 spans",
			response: successMetricsResponse,
			searchResponse: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Matched: 2,
						},
					},
				},
			},
			expectedMetrics: traceMetrics{
				requested: 1,
			},
		},
		{
			name:     "successful metrics query: 2 traces, 1 span each",
			response: successMetricsResponse,
			searchResponse: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Matched: 1,
						},
					},
				},
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Matched: 1,
						},
					},
				},
			},
			expectedMetrics: traceMetrics{
				requested: 1,
			},
		},
		{
			name:     "Less than expected",
			response: successMetricsResponse,
			searchResponse: []*tempopb.TraceSearchMetadata{
				{
					SpanSets: []*tempopb.SpanSet{
						{
							Matched: 4,
						},
					},
				},
			},
			expectedMetrics: traceMetrics{
				inaccurateMetrics: 1,
				requested:         1,
			},
			expectedError: "TraceQL Metrics results are inaccurate: metric count sum=2.000000, actual span count=4",
		},
		{
			name:           "No traces",
			response:       successMetricsResponse,
			searchResponse: make([]*tempopb.TraceSearchMetadata, 0),
			expectedMetrics: traceMetrics{
				inaccurateMetrics: 1,
				requested:         1,
			},
			expectedError: "TraceQL Metrics results are inaccurate: metric count sum=2.000000, actual span count=0",
		},
		{
			name: "no series in response",
			response: &tempopb.QueryRangeResponse{
				Series: []*tempopb.TimeSeries{},
			},
			expectedMetrics: traceMetrics{
				requested:         1,
				notFoundByMetrics: 1,
			},
			expectedError: fmt.Sprintf("expected trace %s not found in metrics", hexID),
		},
		{
			name: "no series in response (nil)",
			response: &tempopb.QueryRangeResponse{
				Series: nil,
			},
			expectedMetrics: traceMetrics{
				requested:         1,
				notFoundByMetrics: 1,
			},
			expectedError: fmt.Sprintf("expected trace %s not found in metrics", hexID),
		},
		{
			name: "invalid series data",
			response: &tempopb.QueryRangeResponse{
				Series: make([]*tempopb.TimeSeries, 1),
			},
			expectedMetrics: traceMetrics{
				requested:              1,
				incorrectMetricsResult: 1,
			},
			expectedError: "expected time series, got nil",
		},
		{
			name: "too many series",
			response: &tempopb.QueryRangeResponse{
				Series: make([]*tempopb.TimeSeries, 2),
			},
			expectedMetrics: traceMetrics{
				requested:              1,
				incorrectMetricsResult: 1,
			},
			expectedError: "expected exactly 1 series, got 2",
		},
		{
			name:     "query error",
			response: nil,
			err:      errors.New("metrics query failed"),
			expectedMetrics: traceMetrics{
				requested:     1,
				requestFailed: 1,
			},
			expectedError: "metrics query failed",
		},
	}

	logger = zap.NewNop()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTPClient := &MockHTTPClient{
				err:            tt.err,
				metricsResp:    tt.response,
				searchResponse: tt.searchResponse,
			}

			metrics, err := queryMetrics(mockHTTPClient, seed, config, logger)
			assert.Equal(t, tt.expectedMetrics, metrics)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDoMetrics(t *testing.T) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	traceInfo := util.NewTraceInfo(seed, "test")

	// Define the configuration
	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}

	logger = zap.NewNop()

	mockHTTPClient := &MockHTTPClient{
		err: nil,
		metricsResp: &tempopb.QueryRangeResponse{
			Series: []*tempopb.TimeSeries{
				{
					Samples: []tempopb.Sample{
						{
							TimestampMs: seed.UnixMilli(),
							Value:       1.0,
						},
					},
				},
			},
		},
	}

	doMetrics(mockHTTPClient, config, traceInfo, logger)
	assert.Equal(t, 1, mockHTTPClient.GetMetricsCount())
}

func TestRunCheckerWithNilTicker(t *testing.T) {
	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}

	logger = zap.NewNop()

	checkerCalled := false
	selectPastTimestamp := func(_ time.Time) (newStart, ts time.Time, skip bool) {
		return time.Now(), time.Now(), false
	}
	checker := func(_ *util.TraceInfo, _ *zap.Logger) {
		checkerCalled = true
	}

	runChecker(nil, config, selectPastTimestamp, checker, logger)
	assert.False(t, checkerCalled)
}

func TestRunCheckerWithSkip(t *testing.T) {
	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}

	logger = zap.NewNop()
	ticker := time.NewTicker(time.Millisecond) // fires immediately
	defer ticker.Stop()

	// Checker function that signals completion
	var checkerCalled bool
	checker := func(_ *util.TraceInfo, _ *zap.Logger) {
		checkerCalled = true
	}

	alwaysSkip := func(_ time.Time) (newStart, ts time.Time, skip bool) {
		return time.Now(), time.Now(), true
	}

	runChecker(ticker, config, alwaysSkip, checker, logger)
	time.Sleep(5 * time.Millisecond)
	// Ensure the checker was not called due to skip=true
	assert.False(t, checkerCalled)
}

func TestRunCheckerTraceNotReady(t *testing.T) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)

	config := vultureConfiguration{
		tempoOrgID:                    "orgID",
		tempoWriteBackoffDuration:     10 * time.Hour, // Very long to ensure trace not ready
		tempoLongWriteBackoffDuration: 20 * time.Hour,
	}

	logger = zap.NewNop()

	ticker := time.NewTicker(time.Millisecond) // fires immediately

	// Checker function that signals completion
	var checkerCalled bool
	checker := func(_ *util.TraceInfo, _ *zap.Logger) {
		checkerCalled = true
	}

	selectPastTimestamp := func(_ time.Time) (newStart, ts time.Time, skip bool) {
		return seed, seed, false
	}
	runChecker(ticker, config, selectPastTimestamp, checker, logger)
	time.Sleep(5 * time.Millisecond)
	// Ensure the checker was not called because trace is not ready
	assert.False(t, checkerCalled)
}

func TestRunCheckerSuccess(t *testing.T) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	startTime := time.Date(2007, 1, 1, 12, 0, 0, 0, time.UTC)
	now := time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC) // Far in the future to ensure trace is ready

	config := vultureConfiguration{
		tempoOrgID:                    "orgID",
		tempoWriteBackoffDuration:     time.Second, // Small value to ensure trace is ready
		tempoLongWriteBackoffDuration: time.Second,
	}

	logger = zap.NewNop()

	tickChan := make(chan time.Time, 1)
	mockTicker := &time.Ticker{C: tickChan}

	// Send the time on the channel to trigger the ticker
	go func() {
		tickChan <- now
	}()

	// Checker function that signals completion
	mx := sync.Mutex{}
	checkerCalled := false
	var checkedInfo *util.TraceInfo
	checker := func(info *util.TraceInfo, _ *zap.Logger) {
		mx.Lock()
		defer mx.Unlock()
		checkerCalled = true
		checkedInfo = info
	}

	selectPastTimestamp := func(_ time.Time) (newStart, ts time.Time, skip bool) {
		return startTime, seed, false
	}
	runChecker(mockTicker, config, selectPastTimestamp, checker, logger)
	time.Sleep(10 * time.Millisecond)

	mx.Lock()
	defer mx.Unlock()
	assert.True(t, checkerCalled)
	require.NotNil(t, checkedInfo)
	assert.Equal(t, seed.Unix(), checkedInfo.Timestamp().Unix())
}
