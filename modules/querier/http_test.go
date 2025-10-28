package querier

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
)

func TestWriteFormattedContentForRequest_NilResponse(t *testing.T) {
	tests := []struct {
		name     string
		response proto.Message
		wantCode int
		wantBody string
	}{
		{
			name:     "explicit nil",
			response: nil,
			wantCode: http.StatusInternalServerError,
			wantBody: "internal error: nil response\n",
		},
		{
			name:     "typed nil pointer",
			response: (*tempopb.SearchResponse)(nil),
			wantCode: http.StatusInternalServerError,
			wantBody: "internal error: nil response\n",
		},
		{
			name:     "empty struct is valid",
			response: &tempopb.SearchResponse{},
			wantCode: http.StatusOK,
			wantBody: "", // empty protobuf
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/search", nil)
			req.Header.Set("Accept", "application/protobuf")
			w := httptest.NewRecorder()

			// Call the function with the test response
			writeFormattedContentForRequest(w, req, tt.response, nil)

			assert.Equal(t, tt.wantCode, w.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}
