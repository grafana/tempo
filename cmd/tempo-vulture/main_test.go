package main

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/gogo/protobuf/jsonpb"
	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
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

			// Check ticker existence
			assert.Equal(t, tt.expectedWriteTicker, tickerWrite != nil, "TickerWrite")
			assert.Equal(t, tt.expectedReadTicker, tickerRead != nil, "TickerRead")
			assert.Equal(t, tt.expectedSearchTicker, tickerSearch != nil, "TickerSearch")

			// Check error
			if tt.expectedError != "" {
				assert.NotNil(t, err, "Expected error but got nil")
				assert.EqualError(t, err, tt.expectedError, "Error message mismatch")
			} else {
				assert.Nil(t, err, "Expected no error but got one")
			}
		})
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

	require.Greater(t, len(mockJaegerClient.batches_emitted), 0)
}
