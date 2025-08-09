package frontend

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

// TraceLookupRequest represents the request body for trace lookup
type TraceLookupRequest struct {
	IDs []string `json:"ids"`
}

// newTraceLookupHandler creates a http.handler for trace lookup requests
func newTraceLookupHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger) http.RoundTripper {

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, err := user.ExtractOrgID(req.Context())
		if err != nil {
			level.Error(logger).Log("msg", "trace lookup: failed to extract tenant id", "err", err)
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(err.Error())),
			}, nil
		}

		// Parse request body
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("failed to read request body")),
				Header:     http.Header{},
			}, nil
		}

		var lookupReq TraceLookupRequest
		if err := json.Unmarshal(body, &lookupReq); err != nil {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("invalid JSON in request body")),
				Header:     http.Header{},
			}, nil
		}

		if len(lookupReq.IDs) == 0 {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("no trace IDs provided")),
				Header:     http.Header{},
			}, nil
		}

		// Convert trace IDs to bytes
		traceIDs := make([][]byte, len(lookupReq.IDs))
		for i, traceID := range lookupReq.IDs {
			byteID, err := util.HexStringToTraceID(traceID)
			if err != nil {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("invalid trace ID: " + traceID)),
					Header:     http.Header{},
				}, nil
			}
			traceIDs[i] = byteID
		}

		// Create protobuf request
		pbReq := &tempopb.TraceLookupRequest{
			TraceIDs: traceIDs,
		}

		// Convert to HTTP request body
		reqBytes, err := pbReq.Marshal()
		if err != nil {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("failed to marshal request")),
				Header:     http.Header{},
			}, nil
		}

		// Update request with protobuf body
		req.Body = io.NopCloser(strings.NewReader(string(reqBytes)))
		req.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)
		req.Header.Set(api.HeaderContentType, api.HeaderAcceptProtobuf)

		level.Info(logger).Log(
			"msg", "trace lookup request",
			"tenant", tenant,
			"trace_count", len(lookupReq.IDs))

		comb := combiner.NewTraceLookup()
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)

		start := time.Now()
		resp, err := rt.RoundTrip(req)
		elapsed := time.Since(start)

		level.Info(logger).Log(
			"msg", "trace lookup response",
			"tenant", tenant,
			"trace_count", len(lookupReq.IDs),
			"duration_seconds", elapsed.Seconds(),
			"err", err)

		return resp, err
	})
}