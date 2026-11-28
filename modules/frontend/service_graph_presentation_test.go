package frontend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/frontend/combiner"
	"github.com/grafana/tempo/v3/modules/frontend/jobcost"
	"github.com/grafana/tempo/v3/modules/frontend/pipeline"
	"github.com/grafana/tempo/v3/modules/overrides"
	overridesapi "github.com/grafana/tempo/v3/modules/overrides/userconfigurable/api"
	"github.com/grafana/tempo/v3/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/v3/pkg/api"
	"github.com/grafana/tempo/v3/pkg/sharedconfig"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
)

type presentationAPI struct {
	server       *httptest.Server
	manager      overrides.Service
	trace        *tempopb.Trace
	backendCalls atomic.Int32
	missing      atomic.Bool
}

// newPresentationAPI serves both trace APIs and the overrides API with a real
// tenant override manager and disk-backed configuration. Its trace backend returns
// A -> gateway -> B and rejects presentation parameters forwarded to storage.
func newPresentationAPI(t *testing.T) *presentationAPI {
	t.Helper()
	fixture := &presentationAPI{trace: &tempopb.Trace{}}
	id := func(v byte) []byte { out := make([]byte, 8); out[7] = v; return out }
	traceID := make([]byte, 16)
	traceID[15] = 1
	for i, service := range []string{"A", "gateway", "B"} {
		var parent []byte
		if i > 0 {
			parent = id(byte(i))
		}
		fixture.trace.ResourceSpans = append(fixture.trace.ResourceSpans, &tracev1.ResourceSpans{
			Resource:   &resourcev1.Resource{Attributes: []*commonv1.KeyValue{tempopb.MakeKeyValueStringPtr("service.name", service)}},
			ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{{TraceId: traceID, SpanId: id(byte(i + 1)), ParentSpanId: parent, Name: service, Kind: tracev1.Span_SPAN_KIND_INTERNAL}}}},
		})
		if i > 0 {
			fixture.trace.ServiceGraphEdges = append(fixture.trace.ServiceGraphEdges, &tempopb.ServiceGraphEdge{ParentSpanId: parent, ChildSpanId: id(byte(i + 1)), ParentService: []string{"A", "gateway"}[i-1], ChildService: service, ParentKind: 1, ChildKind: 1})
		}
	}
	storage := client.Config{Backend: backend.Local, Local: &local.Config{Path: t.TempDir()}}
	ocfg := overrides.Config{
		Defaults:                        overrides.Overrides{MetricsGenerator: overrides.MetricsGeneratorOverrides{Processor: overrides.ProcessorOverrides{ServiceGraphs: overrides.ServiceGraphsOverrides{BypassServices: []string{"gateway"}, BypassServiceRegex: []string{`^proxy-.*$`}}}}},
		UserConfigurableOverridesConfig: overrides.UserConfigurableOverridesConfig{Enabled: true, PollInterval: 10 * time.Millisecond, Client: storage, API: overrides.UserConfigurableOverridesAPIConfig{EnableServiceGraphBypass: true}},
	}
	manager, err := overrides.NewOverrides(ocfg, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), manager))
	t.Cleanup(func() { require.NoError(t, services.StopAndAwaitTerminated(context.Background(), manager)) })
	fixture.manager = manager
	configAPI, err := overridesapi.New(&ocfg.UserConfigurableOverridesConfig.API, &storage, manager, presentationSettingsValidator{})
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), configAPI))
	t.Cleanup(func() { require.NoError(t, services.StopAndAwaitTerminated(context.Background(), configAPI)) })
	backendTrace := proto.Clone(fixture.trace).(*tempopb.Trace)
	backendTrace.ServiceGraphPresentation = &tempopb.ServiceGraphPresentation{BypassServices: []string{"untrusted-backend-hint"}}
	payload, err := proto.Marshal(&tempopb.TraceByIDResponse{Trace: backendTrace, Metrics: &tempopb.TraceByIDMetrics{}})
	require.NoError(t, err)
	next := pipeline.AsyncRoundTripperFunc[combiner.PipelineResponse](func(req pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
		fixture.backendCalls.Add(1)
		if fixture.missing.Load() {
			return pipeline.NewHTTPToAsyncResponse(&http.Response{StatusCode: http.StatusNotFound, Header: http.Header{}, Body: http.NoBody}), nil
		}
		for _, key := range []string{"include_service_graph_presentation", "bypass_service", "bypass_service_regex"} {
			if req.HTTPRequest().URL.Query().Has(key) {
				return nil, fmt.Errorf("presentation parameter leaked into backend request: %s", key)
			}
		}
		return pipeline.NewHTTPToAsyncResponse(&http.Response{StatusCode: http.StatusOK, Header: http.Header{api.HeaderContentType: []string{api.HeaderAcceptProtobuf}}, Body: io.NopCloser(bytes.NewReader(payload))}), nil
	})
	cfg := Config{ResponseConsumers: 1, MaxQueryExpressionSizeBytes: 128 * 1024}
	estimator := newTestEstimator()
	reporter := newJobReporter(jobcost.OpTraceByID, estimator, newTestJobCostMetrics(estimator, prometheus.NewRegistry()))
	v1 := newTraceIDHandler(cfg, next, manager, combiner.NewTypedTraceByID, reporter, log.NewNopLogger(), nil)
	v2 := newTraceIDV2Handler(cfg, next, manager, combiner.NewTypedTraceByIDV2, reporter, log.NewNopLogger(), nil)
	handler := func(rt http.RoundTripper) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			response, err := rt.RoundTrip(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			body, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if readErr != nil || closeErr != nil {
				http.Error(w, "read response failed", http.StatusInternalServerError)
				return
			}
			for key, values := range response.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(response.StatusCode)
			if _, err := w.Write(body); err != nil {
				t.Errorf("write test response: %v", err)
			}
		}
	}
	router := mux.NewRouter()
	router.HandleFunc("/api/traces/{traceID}", handler(v1)).Methods(http.MethodGet)
	router.HandleFunc("/api/v2/traces/{traceID}", handler(v2)).Methods(http.MethodGet)
	router.HandleFunc("/api/overrides", configAPI.PatchHandler).Methods(http.MethodPatch)
	fixture.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r.WithContext(user.InjectOrgID(r.Context(), r.Header.Get("X-Scope-OrgID"))))
	}))
	t.Cleanup(fixture.server.Close)
	return fixture
}

func (p *presentationAPI) request(t *testing.T, method, path, tenant, format, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, p.server.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("X-Scope-OrgID", tenant)
	req.Header.Set(api.HeaderAccept, format)
	req.Header.Set(api.HeaderContentType, api.HeaderAcceptJSON)
	response, err := p.server.Client().Do(req)
	require.NoError(t, err)
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	return response.StatusCode, data
}

func (p *presentationAPI) checkTrace(t *testing.T, path, tenant, format string, wantPresentation bool, names, patterns []string) {
	t.Helper()
	status, body := p.request(t, http.MethodGet, path, tenant, format, "")
	require.Equal(t, http.StatusOK, status, string(body))
	v2 := strings.HasPrefix(path, "/api/v2/")
	var tr tempopb.Trace
	switch {
	case format == api.HeaderAcceptProtobuf:
		if v2 {
			var response tempopb.TraceByIDResponse
			require.NoError(t, proto.Unmarshal(body, &response))
			tr = *response.Trace
		} else {
			require.NoError(t, proto.Unmarshal(body, &tr))
		}
	case v2:
		var response tempopb.TraceByIDResponse
		require.NoError(t, new(jsonpb.Unmarshaler).Unmarshal(bytes.NewReader(body), &response))
		tr = *response.Trace
	default:
		require.NoError(t, tempopb.UnmarshalFromJSONV1(body, &tr))
	}
	encoded, err := new(jsonpb.Marshaler).MarshalToString(&tr)
	require.NoError(t, err)
	var object map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(encoded), &object))
	value, exists := object["serviceGraphPresentation"]
	require.Equal(t, wantPresentation, exists, encoded)
	if wantPresentation {
		var rules struct {
			Names    []string `json:"bypassServices"`
			Patterns []string `json:"bypassServiceRegex"`
		}
		require.NoError(t, json.Unmarshal(value, &rules))
		require.ElementsMatch(t, names, rules.Names)
		require.ElementsMatch(t, patterns, rules.Patterns)
	}
	// Ignore only the new metadata: the remaining trace must match every original
	// span and edge, including the gateway that a UI may choose to hide.
	delete(object, "serviceGraphPresentation")
	raw, err := json.Marshal(object)
	require.NoError(t, err)
	var original tempopb.Trace
	require.NoError(t, new(jsonpb.Unmarshaler).Unmarshal(bytes.NewReader(raw), &original))
	require.True(t, proto.Equal(p.trace, &original), "presentation must not change spans or original edges")
}

// TestTraceAPIServiceGraphPresentation exercises both API versions and encodings.
// Tenant defaults, request overrides, and empty lists affect only presentation
// metadata; original spans and edges remain intact and invalid rules fail early.
func TestTraceAPIServiceGraphPresentation(t *testing.T) {
	p := newPresentationAPI(t)
	for _, prefix := range []string{"/api/traces/", "/api/v2/traces/"} {
		for _, format := range []string{api.HeaderAcceptJSON, api.HeaderAcceptProtobuf} {
			t.Run(prefix+format, func(t *testing.T) {
				base := prefix + "00000000000000000000000000000001"
				// Each list can inherit defaults, be replaced, or be explicitly cleared.
				// A later request must still see the original tenant defaults.
				for _, tc := range []struct {
					name, query     string
					present         bool
					names, patterns []string
				}{
					{name: "legacy response unchanged"},
					{name: "explicit opt out", query: "?include_service_graph_presentation=false"},
					{name: "tenant defaults", query: "?include_service_graph_presentation=true", present: true, names: []string{"gateway"}, patterns: []string{`^proxy-.*$`}},
					{name: "change exact list for one request", query: "?bypass_service=ingress&bypass_service=sidecar", present: true, names: []string{"ingress", "sidecar"}, patterns: []string{`^proxy-.*$`}},
					{name: "change regex for one request", query: "?bypass_service_regex=%5Esidecar%24", present: true, names: []string{"gateway"}, patterns: []string{`^sidecar$`}},
					{name: "clear both lists", query: "?bypass_service=&bypass_service_regex=", present: true},
					{name: "defaults unchanged after requests", query: "?include_service_graph_presentation=true", present: true, names: []string{"gateway"}, patterns: []string{`^proxy-.*$`}},
				} {
					t.Run(tc.name, func(t *testing.T) {
						p.checkTrace(t, base+tc.query, "presentation-a", format, tc.present, tc.names, tc.patterns)
					})
				}
				_, legacy := p.request(t, http.MethodGet, base, "presentation-a", format, "")
				_, optedOut := p.request(t, http.MethodGet, base+"?include_service_graph_presentation=false", "presentation-a", format, "")
				require.Equal(t, legacy, optedOut, "opting out must preserve legacy response bytes")
				// Reject malformed rules before doing any backend trace lookup.
				for _, query := range []string{"?bypass_service_regex=%5B", "?include_service_graph_presentation=invalid", "?include_service_graph_presentation=false&bypass_service=gateway", "?bypass_service=&bypass_service=gateway"} {
					before := p.backendCalls.Load()
					status, _ := p.request(t, http.MethodGet, base+query, "presentation-a", format, "")
					require.Equal(t, http.StatusBadRequest, status)
					require.Equal(t, before, p.backendCalls.Load(), "invalid parameters must fail before querying storage")
				}
			})
		}
	}
}

// TestTraceAPIDefaultBlacklistUpdates changes saved tenant defaults through PATCH,
// waits for the override manager to reload them, and verifies subsequent trace
// queries. Request-only overrides must neither persist nor affect another tenant.
func TestTraceAPIDefaultBlacklistUpdates(t *testing.T) {
	p := newPresentationAPI(t)
	for _, prefix := range []string{"/api/traces/", "/api/v2/traces/"} {
		base := prefix + "00000000000000000000000000000001"
		p.checkTrace(t, base+"?include_service_graph_presentation=true", "presentation-a", api.HeaderAcceptJSON, true, []string{"gateway"}, []string{`^proxy-.*$`})
	}
	// Persist a new blacklist through the same API an operator would use.
	status, body := p.request(t, http.MethodPatch, "/api/overrides", "presentation-a", api.HeaderAcceptJSON, `{"metrics_generator":{"processor":{"service_graphs":{"bypass_services":["new-gateway"],"bypass_service_regex":["^new-proxy-.*$"]}}}}`)
	require.Equal(t, http.StatusOK, status, string(body))
	// Saved overrides become visible after the manager's asynchronous reload.
	require.Eventually(t, func() bool {
		return slices.Equal([]string{"new-gateway"}, p.manager.MetricsGeneratorProcessorServiceGraphsBypassServices("presentation-a"))
	}, 5*time.Second, 10*time.Millisecond)
	for _, prefix := range []string{"/api/traces/", "/api/v2/traces/"} {
		base := prefix + "00000000000000000000000000000001"
		p.checkTrace(t, base+"?include_service_graph_presentation=true", "presentation-a", api.HeaderAcceptJSON, true, []string{"new-gateway"}, []string{`^new-proxy-.*$`})
		// A query override is temporary; subsequent queries and other tenants
		// must continue to use their own saved or runtime defaults.
		p.checkTrace(t, base+"?bypass_service=request-only&bypass_service_regex=", "presentation-a", api.HeaderAcceptJSON, true, []string{"request-only"}, nil)
		p.checkTrace(t, base+"?include_service_graph_presentation=true", "presentation-a", api.HeaderAcceptJSON, true, []string{"new-gateway"}, []string{`^new-proxy-.*$`})
		p.checkTrace(t, base+"?include_service_graph_presentation=true", "presentation-b", api.HeaderAcceptJSON, true, []string{"gateway"}, []string{`^proxy-.*$`})
	}
}

type presentationSettingsValidator struct{}

func (presentationSettingsValidator) Validate(limits *client.Limits) error {
	sg := limits.GetMetricsGenerator().GetProcessor().GetServiceGraphs()
	patterns, _ := sg.GetBypassServiceRegex()
	names, _ := sg.GetBypassServices()
	return sharedconfig.ValidateServiceGraphPresentation(names, patterns)
}

// TestTraceAPIPresentationPreservesNotFound ensures requesting presentation hints
// cannot turn a missing trace into a successful response in either API version.
func TestTraceAPIPresentationPreservesNotFound(t *testing.T) {
	fixture := newPresentationAPI(t)
	fixture.missing.Store(true)
	for _, version := range []string{"/api/traces", "/api/v2/traces"} {
		for _, format := range []string{api.HeaderAcceptJSON, api.HeaderAcceptProtobuf} {
			t.Run(version+format, func(t *testing.T) {
				status, _ := fixture.request(t, http.MethodGet, version+"/00000000000000000000000000000001?include_service_graph_presentation=true", "presentation-a", format, "")
				require.Equal(t, http.StatusNotFound, status)
			})
		}
	}
}

// Oversized or malformed saved blacklists must fail at write time; rejecting
// them must preserve the last usable defaults returned by trace queries.
func TestTraceAPIRejectsInvalidSavedBlacklist(t *testing.T) {
	p := newPresentationAPI(t)
	for _, sg := range []map[string][]string{
		{"bypass_services": strings.Fields(strings.Repeat("gateway ", 129))},
		{"bypass_services": []string{strings.Repeat("x", 65537)}},
		{"bypass_service_regex": []string{"["}},
	} {
		body, err := json.Marshal(map[string]interface{}{"metrics_generator": map[string]interface{}{"processor": map[string]interface{}{"service_graphs": sg}}})
		require.NoError(t, err)
		status, _ := p.request(t, http.MethodPatch, "/api/overrides", "presentation-a", api.HeaderAcceptJSON, string(body))
		require.Equal(t, http.StatusBadRequest, status)
		p.checkTrace(t, "/api/v2/traces/00000000000000000000000000000001?include_service_graph_presentation=true", "presentation-a", api.HeaderAcceptJSON, true, []string{"gateway"}, []string{`^proxy-.*$`})
	}
}
