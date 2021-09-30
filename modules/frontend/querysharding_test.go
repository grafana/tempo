package frontend

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"
)

func TestCreateBlockBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		queryShards int
		expected    [][]byte
	}{
		{
			name:        "single shard",
			queryShards: 1,
			expected: [][]byte{
				{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			},
		},
		{
			name:        "multiple shards",
			queryShards: 4,
			expected: [][]byte{
				{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},  // 0
				{0x3f, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, // 0x3f = 255/4 * 1
				{0x7e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, // 0x7e = 255/4 * 2
				{0xbd, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, // 0xbd = 255/4 * 3
				{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bb := createBlockBoundaries(tt.queryShards)
			assert.Len(t, bb, len(tt.expected))

			for i := 0; i < len(bb); i++ {
				assert.Equal(t, tt.expected[i], bb[i])
			}
		})
	}
}

type resp struct {
	status int
	body   []byte
}

func TestShardingWareDoRequest(t *testing.T) {
	// create and split a trace
	trace := test.MakeTrace(10, []byte{0x01, 0x02})
	trace1 := &tempopb.Trace{}
	trace2 := &tempopb.Trace{}

	for _, b := range trace.Batches {
		if rand.Int()%2 == 0 {
			trace1.Batches = append(trace1.Batches, b)
		} else {
			trace2.Batches = append(trace2.Batches, b)
		}
	}

	tests := []struct {
		name           string
		status1        int
		status2        int
		trace1         *tempopb.Trace
		trace2         *tempopb.Trace
		err1           error
		err2           error
		expectedStatus int
		expectedTrace  *tempopb.Trace
		expectedError  error
	}{
		{
			name:           "empty returns",
			status1:        200,
			status2:        200,
			expectedStatus: 200,
			expectedTrace:  &tempopb.Trace{},
		},
		{
			name:           "404",
			status1:        404,
			status2:        404,
			expectedStatus: 404,
		},
		{
			name:           "500+404",
			status1:        500,
			status2:        404,
			expectedStatus: 500,
		},
		{
			name:           "404+500",
			status1:        404,
			status2:        500,
			expectedStatus: 500,
		},
		{
			name:           "500+200",
			status1:        500,
			status2:        200,
			trace2:         trace2,
			expectedStatus: 500,
		},
		{
			name:           "200+500",
			status1:        200,
			trace1:         trace1,
			status2:        500,
			expectedStatus: 500,
		},
		{
			name:           "503+200",
			status1:        503,
			status2:        200,
			trace2:         trace2,
			expectedStatus: 503,
		},
		{
			name:           "200+503",
			status1:        200,
			trace1:         trace1,
			status2:        503,
			expectedStatus: 503,
		},
		{
			name:           "200+404",
			status1:        200,
			trace1:         trace1,
			status2:        404,
			expectedStatus: 200,
			expectedTrace:  trace1,
		},
		{
			name:           "404+200",
			status1:        404,
			status2:        200,
			trace2:         trace1,
			expectedStatus: 200,
			expectedTrace:  trace1,
		},
		{
			name:           "200+200",
			status1:        200,
			trace1:         trace1,
			status2:        200,
			trace2:         trace2,
			expectedStatus: 200,
			expectedTrace:  trace,
		},
		{
			name:          "200+err",
			status1:       200,
			trace1:        trace1,
			err2:          errors.New("booo"),
			expectedError: errors.New("booo"),
		},
		{
			name:          "err+200",
			err1:          errors.New("booo"),
			status2:       200,
			trace2:        trace1,
			expectedError: errors.New("booo"),
		},

		{
			name:          "500+err",
			status1:       500,
			trace1:        trace1,
			err2:          errors.New("booo"),
			expectedError: errors.New("booo"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sharder := ShardingWare(2, log.NewNopLogger())

			mtx := sync.Mutex{}
			firstResponse := true
			next := RoundTripperFunc(func(*http.Request) (*http.Response, error) {
				mtx.Lock()
				defer mtx.Unlock()
				var trace *tempopb.Trace
				var statusCode int
				var err error
				if firstResponse {
					trace = tc.trace1
					statusCode = tc.status1
					err = tc.err1
					firstResponse = false
				} else {
					trace = tc.trace2
					err = tc.err2
					statusCode = tc.status2
				}

				if err != nil {
					return nil, err
				}

				var traceBytes []byte
				if trace != nil {
					traceBytes, err = proto.Marshal(trace)
					require.NoError(t, err)
				}

				return &http.Response{
					Body:       ioutil.NopCloser(bytes.NewReader(traceBytes)),
					StatusCode: statusCode,
				}, nil
			})

			testRT := NewRoundTripper(next, sharder)

			req := httptest.NewRequest("GET", "/api/traces/1234", nil)
			ctx := req.Context()
			ctx = user.InjectOrgID(ctx, "blerg")
			req = req.WithContext(ctx)

			resp, err := testRT.RoundTrip(req)
			if tc.expectedError != nil {
				assert.Equal(t, tc.expectedError, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
			if tc.expectedTrace != nil {
				actualTrace := &tempopb.Trace{}
				bytesTrace, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				err = proto.Unmarshal(bytesTrace, actualTrace)
				require.NoError(t, err)

				model.SortTrace(tc.expectedTrace)
				model.SortTrace(actualTrace)
				assert.True(t, proto.Equal(tc.expectedTrace, actualTrace))
			}
		})
	}
}
