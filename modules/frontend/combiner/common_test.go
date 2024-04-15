package combiner

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gogo/status"
	"github.com/grafana/tempo/pkg/tempopb"
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
