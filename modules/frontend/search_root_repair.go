package frontend

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
	"github.com/gorilla/mux"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
)

// rootSpanServiceNameKey mirrors the "service.name" resource attribute used everywhere else in
// Tempo to identify a resource's service (see e.g. vparquet5's LabelServiceName).
const rootSpanServiceNameKey = "service.name"

// rootSpanRepairTimeout bounds a single repair lookup so that a slow or unlucky trace-by-ID
// fetch can't hold up an otherwise-complete search response.
const rootSpanRepairTimeout = 2 * time.Second

// newRootSpanRepairFunc builds a combiner.RootSpanRepairFunc that resolves a trace's root
// service/span name by fetching it by ID through the same trace pipeline used to serve
// /api/v2/traces/{traceID}. That path combines every block containing the trace ID regardless
// of whether those blocks were ever compacted together, which is why opening a trace directly
// always shows the root span even when TraceQL search can't.
func newRootSpanRepairFunc(ctx context.Context, tenant string, headers http.Header, apiPrefix string, tracePipeline pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, dataAccessController DataAccessController, logger log.Logger) combiner.RootSpanRepairFunc {
	return func(traceID string) (serviceName, spanName string, ok bool) {
		repairCtx, cancel := context.WithTimeout(ctx, rootSpanRepairTimeout)
		defer cancel()

		req := buildRootSpanRepairRequest(repairCtx, apiPrefix, traceID, headers)

		var traceRedactor combiner.TraceRedactor
		if dataAccessController != nil {
			redactor, err := dataAccessController.HandleHTTPTraceByIDReq(req)
			if err != nil {
				return "", "", false
			}
			traceRedactor = redactor
		}

		resps, err := tracePipeline.RoundTrip(pipeline.NewHTTPRequest(req))
		if err != nil {
			level.Warn(logger).Log("msg", "search: root span repair failed to fetch trace", "traceID", traceID, "err", err)
			return "", "", false
		}

		comb := combiner.NewTypedTraceByIDV2(o.MaxBytesPerTrace(tenant), api.MarshallingFormatProtobuf, traceRedactor, combiner.TraceByIDV2Options{})
		for {
			resp, done, err := resps.Next(repairCtx)
			if err != nil {
				level.Warn(logger).Log("msg", "search: root span repair failed to read trace response", "traceID", traceID, "err", err)
				return "", "", false
			}
			if resp != nil {
				if err := comb.AddResponse(resp); err != nil {
					level.Warn(logger).Log("msg", "search: root span repair failed to combine trace response", "traceID", traceID, "err", err)
					return "", "", false
				}
			}
			if comb.ShouldQuit() || done {
				break
			}
		}

		traceResp, err := comb.GRPCFinal()
		if err != nil || traceResp == nil {
			return "", "", false
		}

		return rootSpanFromTrace(traceResp.Trace)
	}
}

func buildRootSpanRepairRequest(ctx context.Context, apiPrefix string, traceID string, headers http.Header) *http.Request {
	u := &url.URL{
		Path: path.Join(apiPrefix, "/api/v2/traces", traceID),
	}

	reqHeaders := headers.Clone()
	if reqHeaders == nil {
		reqHeaders = http.Header{}
	}

	req := (&http.Request{
		Method: http.MethodGet,
		URL:    u,
		Header: reqHeaders,
		Body:   http.NoBody,
	}).WithContext(ctx)
	req.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)

	return mux.SetURLVars(req, map[string]string{"traceID": traceID})
}

// rootSpanFromTrace finds the span with no parent and the earliest start time, mirroring how
// blocks determine a trace's root span at write time (see traceToParquetWithMapping).
func rootSpanFromTrace(trace *tempopb.Trace) (serviceName, spanName string, ok bool) {
	if trace == nil {
		return "", "", false
	}

	var earliestStart uint64
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				if len(s.ParentSpanId) != 0 {
					continue
				}
				if ok && s.StartTimeUnixNano >= earliestStart {
					continue
				}

				spanName = s.Name
				serviceName = resourceServiceName(rs.Resource)
				earliestStart = s.StartTimeUnixNano
				ok = true
			}
		}
	}

	return serviceName, spanName, ok
}

func resourceServiceName(res *v1_resource.Resource) string {
	if res == nil {
		return ""
	}
	for _, a := range res.Attributes {
		if a.Key == rootSpanServiceNameKey {
			return a.Value.GetStringValue()
		}
	}
	return ""
}
