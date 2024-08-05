package combiner

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/status"
	"github.com/grafana/tempo/v2/pkg/api"
	"github.com/grafana/tempo/v2/pkg/tempopb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestErroredResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		respBody   string

		expectedResp *http.Response
		expectedErr  error
	}{
		{
			name:       "no error",
			statusCode: http.StatusOK,
			respBody:   "woooo!",
		},
		{
			name:       "5xx",
			statusCode: http.StatusInternalServerError,
			respBody:   "foo",

			expectedResp: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "Internal Server Error",
				Body:       io.NopCloser(strings.NewReader("foo")),
			},
			expectedErr: status.Error(codes.Internal, "foo"),
		},
		{
			name:       "4xx",
			statusCode: http.StatusBadRequest,
			respBody:   "foo",

			expectedResp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     "Bad Request",
				Body:       io.NopCloser(strings.NewReader("foo")),
			},
			expectedErr: status.Error(codes.InvalidArgument, "foo"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			combiner := &genericCombiner[*tempopb.SearchResponse]{
				httpStatusCode: tc.statusCode,
				httpRespBody:   tc.respBody,
			}

			resp, err := combiner.erroredResponse()
			require.Equal(t, tc.expectedErr, err)

			if tc.expectedResp == nil {
				require.Nil(t, resp)
				return
			}

			// validate the body
			require.Equal(t, tc.expectedResp.Status, resp.Status)
			require.Equal(t, tc.expectedResp.StatusCode, resp.StatusCode)

			expectedBody, err := io.ReadAll(tc.expectedResp.Body)
			require.NoError(t, err)
			actualBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, expectedBody, actualBody)
		})
	}
}

func TestInitHttpCombiner(t *testing.T) {
	combiner := newTestCombiner()

	require.Equal(t, 200, combiner.httpStatusCode)
	require.Equal(t, api.HeaderAcceptJSON, combiner.httpMarshalingFormat)
}

func TestGenericCombiner(t *testing.T) {
	combiner := newTestCombiner()

	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10000; j++ {

				err := combiner.AddResponse(newTestResponse(t))
				require.NoError(t, err)
			}
		}()
	}

	wg.Wait()
	actual, err := combiner.finalize(nil)
	require.NoError(t, err)

	expected := 10 * 10000
	require.Equal(t, expected, int(actual.SpanCount))
	require.Equal(t, expected, int(actual.ErrorCount))
}

func TestGenericCombinerHoldsErrors(t *testing.T) {
	// slam a combiner with successful responses and just one error. confirm that the error is returned
	combiner := newTestCombiner()
	wg := sync.WaitGroup{}

	for j := 0; j < 10; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; i < 10000; i++ {
				err := combiner.AddResponse(newTestResponse(t))
				require.NoError(t, err)
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond)
		err := combiner.AddResponse(newFailedTestResponse())
		require.NoError(t, err)
	}()

	wg.Wait()
	resp, err := combiner.HTTPFinal()
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestGenericCombinerDoesntRace(t *testing.T) {
	combiner := newTestCombiner()
	end := make(chan struct{})

	concurrent := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}
	go concurrent(func() {
		_ = combiner.AddResponse(newTestResponse(t))
	})

	go concurrent(func() {
		// this test is going to add a failed response which cuts off certain code paths. just wait a bit to test the other paths
		time.Sleep(10 * time.Millisecond)
		_ = combiner.AddResponse(newFailedTestResponse())
	})

	go concurrent(func() {
		combiner.ShouldQuit()
	})

	go concurrent(func() {
		combiner.StatusCode()
	})

	go concurrent(func() {
		_, _ = combiner.HTTPFinal()
	})

	go concurrent(func() {
		_, _ = combiner.GRPCFinal()
	})

	go concurrent(func() {
		_, _ = combiner.GRPCDiff()
	})

	time.Sleep(100 * time.Millisecond)
	close(end)
}

type testPipelineResponse struct {
	r *http.Response
}

func newTestResponse(t *testing.T) *testPipelineResponse {
	serviceStats := &tempopb.ServiceStats{
		SpanCount:  1,
		ErrorCount: 1,
	}

	rec := httptest.NewRecorder()
	err := (&jsonpb.Marshaler{}).Marshal(rec, serviceStats)
	require.NoError(t, err)

	return &testPipelineResponse{
		r: rec.Result(),
	}
}

func newFailedTestResponse() *testPipelineResponse {
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusInternalServerError)

	return &testPipelineResponse{
		r: rec.Result(),
	}
}

func (p *testPipelineResponse) HTTPResponse() *http.Response {
	return p.r
}

func (p *testPipelineResponse) RequestData() any {
	return nil
}

func newTestCombiner() *genericCombiner[*tempopb.ServiceStats] {
	count := 0

	gc := &genericCombiner[*tempopb.ServiceStats]{
		new:     func() *tempopb.ServiceStats { return &tempopb.ServiceStats{} },
		current: nil,
		combine: func(_, _ *tempopb.ServiceStats, _ PipelineResponse) error {
			count++
			return nil
		},
		finalize: func(_ *tempopb.ServiceStats) (*tempopb.ServiceStats, error) {
			return &tempopb.ServiceStats{
				SpanCount:  uint32(count),
				ErrorCount: uint32(count),
			}, nil
		},
		quit: func(_ *tempopb.ServiceStats) bool {
			return false
		},
		diff: func(_ *tempopb.ServiceStats) (*tempopb.ServiceStats, error) {
			return &tempopb.ServiceStats{
				SpanCount:  uint32(count),
				ErrorCount: uint32(count),
			}, nil
		},
	}
	initHTTPCombiner(gc, api.HeaderAcceptJSON)
	return gc
}
