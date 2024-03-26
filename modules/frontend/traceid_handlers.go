package frontend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

// newTraceIDHandler creates a http.handler for trace by id requests
//
// todo: the trace by id path consumes a lot of resources for large traces b/c it
// repeatedly unmarshals/marshals the trace data. Note that it occurs once here
// to marshal into the proto format and again in the deduper middleware. we should
// collapse this into the combiner where the data is already unmarshalled
//
//	jpe - can i get one of these?
func newTraceIDHandler(cfg Config, o overrides.Interface, next pipeline.AsyncRoundTripper[*http.Response], logger log.Logger) http.RoundTripper {
	traceIDRT := pipeline.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, err := user.ExtractOrgID(req.Context())
		if err != nil {
			level.Error(logger).Log("msg", "trace id: failed to extract tenant id", "err", err)
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(err.Error())),
			}, nil
		}

		// validate traceID
		_, err = api.ParseTraceID(req)
		if err != nil {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(err.Error())),
				Header:     http.Header{},
			}, nil
		}

		// validate start and end parameter
		_, _, _, _, _, reqErr := api.ValidateAndSanitizeRequest(req)
		if reqErr != nil {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(reqErr.Error())),
				Header:     http.Header{},
			}, nil
		}

		// check marshalling format
		marshallingFormat := api.HeaderAcceptJSON
		if req.Header.Get(api.HeaderAccept) == api.HeaderAcceptProtobuf {
			marshallingFormat = api.HeaderAcceptProtobuf
		}

		// enforce all communication internal to Tempo to be in protobuf bytes
		req.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)
		prepareRequestForDownstream(req, tenant, req.RequestURI, nil)

		level.Info(logger).Log(
			"msg", "trace id request",
			"tenant", tenant,
			"path", req.URL.Path)

		combiner := combiner.NewTraceByID(o.MaxBytesPerTrace(tenant))
		rt := pipeline.NewHTTPCollector(next, combiner)

		resp, err := rt.RoundTrip(req)

		// marshal/unmarshal into requested format
		if resp != nil && resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("error reading response body at query frontend: %w", err)
			}
			responseObject := &tempopb.TraceByIDResponse{}
			err = proto.Unmarshal(body, responseObject)
			if err != nil {
				return nil, err
			}

			// jpe - response should be in proto b/c that's what the combiner does
			if marshallingFormat == api.HeaderAcceptJSON {
				var jsonTrace bytes.Buffer
				marshaller := &jsonpb.Marshaler{}
				err = marshaller.Marshal(&jsonTrace, responseObject.Trace) // jpe - how does this work? deduper expects a TraceByIDResponse?
				if err != nil {
					return nil, err
				}
				resp.Body = io.NopCloser(bytes.NewReader(jsonTrace.Bytes()))
			} else {
				traceBuffer, err := proto.Marshal(responseObject.Trace) // jpe - how does this work? deduper expects a TraceByIDResponse?
				if err != nil {
					return nil, err
				}
				resp.Body = io.NopCloser(bytes.NewReader(traceBuffer))
			}

			if resp.Header != nil {
				resp.Header.Set(api.HeaderContentType, marshallingFormat)
			}
		}

		level.Info(logger).Log(
			"msg", "trace id response",
			"tenant", tenant,
			"path", req.URL.Path,
			"err", err)

		return resp, err
	})

	// wrap the round tripper with the deduper middleware
	deduperMW := newDeduper(logger)
	return deduperMW.Wrap(traceIDRT)
}
