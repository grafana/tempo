package main

import (
	"bytes"
	"errors"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

func TestHasMissingSpans(t *testing.T) {
	cases := []struct {
		trace    *tempopb.Trace
		expected bool
	}{
		{
			&tempopb.Trace{
				Batches: []*v1.ResourceSpans{
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
				Batches: []*v1.ResourceSpans{
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
}

func TestInitTickers(t *testing.T) {
	tests := []struct {
		name                                        string
		writeDuration, readDuration, searchDuration time.Duration
		expectedWriteTicker                         bool
		expectedReadTicker                          bool
		expectedSearchTicker                        bool
		expectedError                               string
	}{
		{
			name:                 "Valid write and read durations",
			writeDuration:        1 * time.Second,
			readDuration:         2 * time.Second,
			searchDuration:       0,
			expectedWriteTicker:  true,
			expectedReadTicker:   true,
			expectedSearchTicker: false,
			expectedError:        "",
		},
		{
			name:                 "Invalid write duration (zero)",
			writeDuration:        0,
			readDuration:         0,
			searchDuration:       0,
			expectedWriteTicker:  false,
			expectedReadTicker:   false,
			expectedSearchTicker: false,
			expectedError:        "tempo-write-backoff-duration must be greater than 0",
		},
		{
			name:                 "No read durations set",
			writeDuration:        1 * time.Second,
			readDuration:         0,
			searchDuration:       1 * time.Second,
			expectedWriteTicker:  true,
			expectedReadTicker:   false,
			expectedSearchTicker: true,
			expectedError:        "",
		},
		{
			name:                 "No read or search durations set",
			writeDuration:        1 * time.Second,
			readDuration:         0,
			searchDuration:       0,
			expectedWriteTicker:  false,
			expectedReadTicker:   false,
			expectedSearchTicker: false,
			expectedError:        "at least one of tempo-search-backoff-duration or tempo-read-backoff-duration must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tickerWrite, tickerRead, tickerSearch, err := initTickers(tt.writeDuration, tt.readDuration, tt.searchDuration)

			assert.Equal(t, tt.expectedWriteTicker, tickerWrite != nil, "TickerWrite")
			assert.Equal(t, tt.expectedReadTicker, tickerRead != nil, "TickerRead")
			assert.Equal(t, tt.expectedSearchTicker, tickerSearch != nil, "TickerSearch")

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

	assert.Greater(t, len(mockJaegerClient.batchesEmitted), 0)

	// assert an error
	mockJaegerClient = MockReporter{err: errors.New("an error")}
	doWrite(&mockJaegerClient, ticker, config.tempoWriteBackoffDuration, config, logger)
	time.Sleep(time.Second)
	ticker.Stop()

	assert.Greater(t, len(mockJaegerClient.batchesEmitted), 0)
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
	require.Greater(t, len(mockJaegerClient.batchesEmitted), 0)

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
		trace.Batches[0].ScopeSpans[0].Spans[0].ParentSpanId = []byte{'t', 'e', 's', 't'}
	}
	setNoBatchesSpan := func(trace *tempopb.Trace) {
		trace.Batches = make([]*v1.ResourceSpans, 0)
	}
	setAlteredSpan := func(trace *tempopb.Trace) {
		trace.Batches[0].ScopeSpans[0].Spans[0].Name = "Different spam"
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

func TestDoRead(t *testing.T) {
	seed := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	traceInfo := util.NewTraceInfo(seed, "test")

	trace, _ := traceInfo.ConstructTraceFromEpoch()
	startTime := time.Date(2007, 1, 1, 12, 0, 0, 0, time.UTC)
	mockHTTPClient := MockHTTPClient{err: nil, traceResp: trace}
	// Define the configuration
	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	logger = zap.NewNop()
	r := rand.New(rand.NewSource(startTime.Unix()))

	// Assert ticker is nil
	doRead(&mockHTTPClient, nil, startTime, config.tempoWriteBackoffDuration, r, config, logger)
	assert.Equal(t, 0, mockHTTPClient.requestsCount)

	// Assert an ok read
	doRead(&mockHTTPClient, ticker, startTime, config.tempoWriteBackoffDuration, r, config, logger)
	time.Sleep(time.Second)
	ticker.Stop()
	assert.Greater(t, mockHTTPClient.requestsCount, 0)

	// Assert a read with errors
	mockHTTPClient = MockHTTPClient{err: errors.New("an error"), traceResp: trace}
	doRead(&mockHTTPClient, ticker, startTime, config.tempoWriteBackoffDuration, r, config, logger)
	time.Sleep(time.Second)
	ticker.Stop()
	assert.Equal(t, 1, mockHTTPClient.requestsCount)
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
	startTime := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)

	// Define the configuration
	config := vultureConfiguration{
		tempoOrgID: "orgID",
		// This is a hack to ensure the trace is "ready"
		tempoWriteBackoffDuration: -time.Hour * 10000,
		tempoRetentionDuration:    time.Second * 10,
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	logger = zap.NewNop()
	r := rand.New(rand.NewSource(startTime.Unix()))

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
	logger = zap.NewNop()

	mockHTTPClient := MockHTTPClient{err: nil, searchResponse: searchResponse}
	// Assert when ticker is nil
	doSearch(&mockHTTPClient, ticker, startTime, config.tempoWriteBackoffDuration, r, config, logger)
	assert.Equal(t, mockHTTPClient.searchesCount, 0)

	// Assert an ok search
	doSearch(&mockHTTPClient, ticker, startTime, config.tempoWriteBackoffDuration, r, config, logger)
	time.Sleep(time.Second)
	ticker.Stop()

	assert.Greater(t, mockHTTPClient.searchesCount, 0)

	// Assert an errored search
	mockHTTPClient = MockHTTPClient{err: errors.New("an error"), searchResponse: searchResponse}

	doSearch(&mockHTTPClient, ticker, startTime, config.tempoWriteBackoffDuration, r, config, logger)
	time.Sleep(time.Second)
	ticker.Stop()

	assert.Equal(t, mockHTTPClient.searchesCount, 2)
}

func TestNewJaegerGRPCClient(t *testing.T) {
	config := vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
	}
	client, err := newJaegerGRPCClient("http://localhost", config, zap.NewNop())

	assert.NoError(t, err)
	assert.NotNil(t, client)

	config = vultureConfiguration{
		tempoOrgID:                "orgID",
		tempoWriteBackoffDuration: time.Second,
		tempoPushTLS:              true,
	}
	client, err = newJaegerGRPCClient("http://localhost", config, zap.NewNop())

	assert.NoError(t, err)
	assert.NotNil(t, client)
}
