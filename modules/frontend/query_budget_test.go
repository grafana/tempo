package frontend

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/grafana/tempo/v3/modules/frontend/combiner"
	"github.com/grafana/tempo/v3/modules/frontend/jobcost"
	"github.com/grafana/tempo/v3/modules/frontend/pipeline"
	"github.com/grafana/tempo/v3/modules/frontend/querybudget"
	"github.com/grafana/tempo/v3/modules/frontend/queue"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/api"
	"github.com/grafana/tempo/v3/pkg/tempopb"
)

// a search reads more than the budget of its tenant, so the next search is rejected until the bucket refills
func TestFrontendQueryBudget(t *testing.T) {
	const (
		rate  = 100_000
		burst = 1_000_000
	)
	searchReq := &tempopb.SearchRequest{Query: "{resource.service.name = `test`}", Start: 1, End: 100000}

	tests := []struct {
		name             string
		mode             string
		expectedRejected bool
	}{
		{name: "enabled", mode: querybudget.ModeEnabled, expectedRejected: true},
		{name: "dry run", mode: querybudget.ModeDryRun},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			// every job reads 10 MiB, far above its estimate and the whole budget
			next := &mockRoundTripper{responseFn: func() proto.Message {
				return &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{InspectedBytes: 10 * 1024 * 1024}}
			}}
			f := frontendWithSettingsAndRegisterer(t, next, nil, nil, nil, nil, reg, func(_ *Config, o *overrides.Config) {
				o.Defaults.Read.QueryBudget = overrides.QueryBudgetOverrides{Mode: tc.mode, Rate: rate, Burst: burst}
			})

			httpSearch := func() *httptest.ResponseRecorder {
				httpReq, err := api.BuildSearchRequest(httptest.NewRequest("GET", "/api/search", nil), searchReq)
				require.NoError(t, err)
				httpReq = httpReq.WithContext(user.InjectOrgID(httpReq.Context(), "tenant"))

				httpResp := httptest.NewRecorder()
				f.SearchHandler.ServeHTTP(httpResp, httpReq)
				return httpResp
			}

			require.Equal(t, http.StatusOK, httpSearch().Code)

			// 8 jobs were charged with their estimate and settled with what they read
			charged := counterValue(t, reg, "tempo_query_frontend_query_budget_charged_total", map[string]string{"tenant": "tenant", "op": "search"})
			require.InDelta(t, 8*10*1024*1024, charged, 1)

			httpResp := httpSearch()
			grpcErr := f.streamingSearch(searchReq, newMockStreamingServer[*tempopb.SearchResponse]("tenant", nil))

			rejected := counterValue(t, reg, "tempo_query_frontend_query_budget_rejected_queries_total", map[string]string{"tenant": "tenant", "op": "search", "mode": tc.mode})
			require.InDelta(t, 2, rejected, 0.001)

			if !tc.expectedRejected {
				require.Equal(t, http.StatusOK, httpResp.Code)
				require.NoError(t, grpcErr)
				return
			}

			// the debt is capped at burst, which refills in burst/rate seconds
			require.Equal(t, http.StatusTooManyRequests, httpResp.Code)
			require.Equal(t, "10", httpResp.Header().Get(api.HeaderRetryAfter))
			require.Contains(t, httpResp.Body.String(), "query budget used up")

			st, ok := grpcstatus.FromError(grpcErr)
			require.True(t, ok)
			require.Equal(t, codes.ResourceExhausted, st.Code())
			require.Len(t, st.Details(), 1)
			require.InDelta(t, 10, st.Details()[0].(*errdetails.RetryInfo).GetRetryDelay().AsDuration().Seconds(), 0.001)
		})
	}
}

// the jobs of a multi-tenant query are charged to their own tenant and labeled with the op of the pipeline
func TestFrontendQueryBudgetMultiTenant(t *testing.T) {
	reg := prometheus.NewRegistry()
	f := frontendWithSettingsAndRegisterer(t, nil, nil, nil, nil, nil, reg, func(_ *Config, o *overrides.Config) {
		o.Defaults.Read.QueryBudget = overrides.QueryBudgetOverrides{Mode: querybudget.ModeDryRun}
	})

	httpReq, err := api.BuildSearchRequest(httptest.NewRequest("GET", "/api/search", nil), &tempopb.SearchRequest{Query: "{}", Start: 1, End: 100000})
	require.NoError(t, err)
	httpReq = httpReq.WithContext(user.InjectOrgID(httpReq.Context(), "t1|t2"))
	httpResp := httptest.NewRecorder()
	f.SearchHandler.ServeHTTP(httpResp, httpReq)
	require.Equal(t, http.StatusOK, httpResp.Code)

	// 8 jobs per tenant, settled with the 1 byte each of them read
	for _, tenant := range []string{"t1", "t2"} {
		labels := map[string]string{"tenant": tenant, "op": "search"}
		charged := counterValue(t, reg, "tempo_query_frontend_query_budget_charged_total", labels)
		refunded := counterValue(t, reg, "tempo_query_frontend_query_budget_refunded_total", labels)
		require.Positive(t, charged)
		require.InDelta(t, 8, charged-refunded, 0.001)
	}
	require.Zero(t, counterValue(t, reg, "tempo_query_frontend_query_budget_charged_total", map[string]string{"op": "unknown"}))
}

func TestQueryBudgetAdmitWare(t *testing.T) {
	tests := []struct {
		name          string
		orgID         string
		chargedBefore map[string]uint64

		multiTenantDisabled bool

		expectedStatus int
		expectedQuery  bool
	}{
		{name: "admitted", orgID: "a", expectedStatus: http.StatusOK, expectedQuery: true},
		{name: "rejected", orgID: "a", chargedBefore: map[string]uint64{"a": 200}, expectedStatus: http.StatusTooManyRequests},
		{name: "dry run", orgID: "dry", chargedBefore: map[string]uint64{"dry": 200}, expectedStatus: http.StatusOK, expectedQuery: true},
		{name: "multi-tenant query is rejected if one tenant is", orgID: "b|a", chargedBefore: map[string]uint64{"a": 200}, expectedStatus: http.StatusTooManyRequests},
		{name: "multi-tenant queries are left to the multi-tenant middleware if disabled", orgID: "b|a", chargedBefore: map[string]uint64{"a": 200}, multiTenantDisabled: true, expectedStatus: http.StatusOK},
		{name: "disabled tenant", orgID: "disabled", expectedStatus: http.StatusOK},
		{name: "unknown tenant", expectedStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			budget := querybudget.New(testBudgetLimits{
				"a":   {querybudget.ModeEnabled, 100, 100},
				"b":   {querybudget.ModeEnabled, 100, 100},
				"dry": {querybudget.ModeDryRun, 100, 100},
			}, prometheus.NewRegistry())
			for tenant, units := range tc.chargedBefore {
				budget.Admit([]string{tenant}, jobcost.OpSearch).Query.Charge(tenant, jobcost.Cost{Kind: jobcost.KindBlock, Estimate: units})
			}

			var called bool
			next := pipeline.AsyncRoundTripperFunc[combiner.PipelineResponse](func(pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
				called = true
				return pipeline.NewSuccessfulResponse(""), nil
			})
			rt := newQueryBudgetAdmitWare(budget, jobcost.OpSearch, !tc.multiTenantDisabled).Wrap(next)

			ctx := pipeline.WithQueryShapeCell(context.Background())
			if tc.orgID != "" {
				ctx = user.InjectOrgID(ctx, tc.orgID)
			}
			req := pipeline.NewHTTPRequest(httptest.NewRequest("GET", "/api/search", nil).WithContext(ctx))

			resps, err := rt.RoundTrip(req)
			require.NoError(t, err)
			resp, _, err := resps.Next(ctx)
			require.NoError(t, err)

			require.Equal(t, tc.expectedStatus, resp.HTTPResponse().StatusCode)
			require.Equal(t, tc.expectedStatus == http.StatusOK, called)
			require.Equal(t, tc.expectedQuery, pipeline.BudgetQueryFromContext(ctx) != nil)
			if tc.expectedStatus == http.StatusTooManyRequests {
				// the debt is capped at burst, so the bucket refills in a second
				require.Equal(t, "1", resp.HTTPResponse().Header.Get(api.HeaderRetryAfter))
			}
		})
	}
}

func TestQueryBudgetChargeWare(t *testing.T) {
	block := jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 100}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name     string
		cost     jobcost.Cost
		ctx      context.Context
		resp     *http.Response
		err      error
		dequeued bool // a querier took the job

		expectedCharged, expectedRefunded float64
	}{
		{name: "completed job", cost: block, resp: &http.Response{StatusCode: http.StatusOK}, dequeued: true, expectedCharged: 100},
		{name: "failed job", cost: block, resp: &http.Response{StatusCode: http.StatusInternalServerError}, dequeued: true, expectedCharged: 100},
		{name: "queue full", cost: block, err: queue.ErrTooManyRequests, expectedCharged: 100, expectedRefunded: 100},
		{name: "queue stopped", cost: block, err: queue.ErrStopped, expectedCharged: 100, expectedRefunded: 100},
		{name: "query canceled while the job was queued", cost: block, ctx: canceled, err: context.Canceled, expectedCharged: 100, expectedRefunded: 100},
		{name: "query canceled while the job ran", cost: block, ctx: canceled, err: context.Canceled, dequeued: true, expectedCharged: 100},
		{name: "querier stream failed", cost: block, err: errors.New("stream closed"), dequeued: true, expectedCharged: 100},
		{name: "no estimate", resp: &http.Response{StatusCode: http.StatusOK}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			budget := querybudget.New(testBudgetLimits{"a": {querybudget.ModeEnabled, 1000, 1000}}, reg)
			next := pipeline.RoundTripperFunc(func(r pipeline.Request) (*http.Response, error) {
				if tc.dequeued {
					pipeline.MarkDequeued(r.Context())
				}
				return tc.resp, tc.err
			})
			rt := newQueryBudgetChargeWare().Wrap(next)

			ctx := tc.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			req := pipeline.NewHTTPRequest(httptest.NewRequest("GET", "/api/search", nil).WithContext(admittedContext(ctx, budget, "a")))
			req.SetCost(tc.cost)

			resp, err := rt.RoundTrip(req)
			require.Equal(t, tc.resp, resp)
			require.Equal(t, tc.err, err)

			labels := map[string]string{"tenant": "a", "op": "search"}
			require.InDelta(t, tc.expectedCharged, counterValue(t, reg, "tempo_query_frontend_query_budget_charged_total", labels), 0.001)
			require.InDelta(t, tc.expectedRefunded, counterValue(t, reg, "tempo_query_frontend_query_budget_refunded_total", labels), 0.001)
		})
	}
}

// every attempt of a retried job is charged
func TestQueryBudgetChargesRetries(t *testing.T) {
	reg := prometheus.NewRegistry()
	budget := querybudget.New(testBudgetLimits{"a": {querybudget.ModeEnabled, 1000, 1000}}, reg)

	attempts := 0
	next := pipeline.RoundTripperFunc(func(pipeline.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	rt := pipeline.Build(nil, []pipeline.Middleware{
		pipeline.NewRetryWare(2, false, prometheus.NewRegistry(), nil),
		newQueryBudgetChargeWare(),
	}, next)

	req := pipeline.NewHTTPRequest(httptest.NewRequest("GET", "/api/search", nil).WithContext(admittedContext(context.Background(), budget, "a")))
	req.SetCost(jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 100})

	resps, err := rt.RoundTrip(req)
	require.NoError(t, err)
	resp, _, err := resps.Next(context.Background())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.HTTPResponse().StatusCode)

	require.Equal(t, 2, attempts)
	require.InDelta(t, 200, counterValue(t, reg, "tempo_query_frontend_query_budget_charged_total", map[string]string{"tenant": "a", "op": "search"}), 0.001)
}

// jobs without an admitted query are not charged
func TestQueryBudgetChargeWareWithoutAdmission(t *testing.T) {
	reg := prometheus.NewRegistry()
	querybudget.New(testBudgetLimits{"a": {querybudget.ModeEnabled, 1000, 1000}}, reg)
	next := pipeline.RoundTripperFunc(func(pipeline.Request) (*http.Response, error) { return &http.Response{StatusCode: http.StatusOK}, nil })

	ctx := user.InjectOrgID(pipeline.WithQueryShapeCell(context.Background()), "a")
	req := pipeline.NewHTTPRequest(httptest.NewRequest("GET", "/api/search", nil).WithContext(ctx))
	req.SetCost(jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 100})

	_, err := newQueryBudgetChargeWare().Wrap(next).RoundTrip(req)
	require.NoError(t, err)
	require.Zero(t, counterValue(t, reg, "tempo_query_frontend_query_budget_charged_total", map[string]string{"tenant": "a"}))
}

// the settle option settles the jobs of the query whose cell is on the context
func TestQueryBudgetSettleOption(t *testing.T) {
	reg := prometheus.NewRegistry()
	budget := querybudget.New(testBudgetLimits{"a": {querybudget.ModeEnabled, 1000, 1000}}, reg)

	ctx := admittedContext(context.Background(), budget, "a")
	mc := combiner.NewTraceByIDMetricsCombiner(queryBudgetSettleOption(ctx))
	mc.Combine(&tempopb.TraceByIDMetrics{InspectedBytes: 300}, budgetTestResponse{cost: jobcost.Cost{Kind: jobcost.KindBlock, Base: 1, Estimate: 100}, tenant: "a"})

	require.InDelta(t, 200, counterValue(t, reg, "tempo_query_frontend_query_budget_charged_total", map[string]string{"tenant": "a", "op": "search"}), 0.001)
}

type budgetTestResponse struct {
	cost   jobcost.Cost
	tenant string
}

func (r budgetTestResponse) HTTPResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK}
}
func (r budgetTestResponse) RequestData() any   { return nil }
func (r budgetTestResponse) IsMetadata() bool   { return false }
func (r budgetTestResponse) Cost() jobcost.Cost { return r.cost }
func (r budgetTestResponse) Tenant() string     { return r.tenant }

// admittedContext returns a context of tenant with a query cell, whose query budget admitted the query.
func admittedContext(ctx context.Context, budget *querybudget.Budget, tenant string) context.Context {
	ctx = user.InjectOrgID(pipeline.WithQueryShapeCell(ctx), tenant)
	pipeline.AdmitQueryBudgetOnce(ctx, func() querybudget.Decision { return budget.Admit([]string{tenant}, jobcost.OpSearch) })
	return ctx
}

type testBudgetLimit struct {
	mode        string
	rate, burst uint64
}

type testBudgetLimits map[string]testBudgetLimit

func (l testBudgetLimits) QueryBudgetMode(tenant string) string  { return l[tenant].mode }
func (l testBudgetLimits) QueryBudgetRate(tenant string) uint64  { return l[tenant].rate }
func (l testBudgetLimits) QueryBudgetBurst(tenant string) uint64 { return l[tenant].burst }

// counterValue returns the value of the counter name with labels, or 0 if it does not exist.
func counterValue(t *testing.T, reg *prometheus.Registry, name string, labels map[string]string) float64 {
	t.Helper()

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, f := range families {
		if f.GetName() != name {
			continue
		}
	metrics:
		for _, m := range f.GetMetric() {
			for _, l := range m.GetLabel() {
				if v, ok := labels[l.GetName()]; ok && v != l.GetValue() {
					continue metrics
				}
			}
			return m.GetCounter().GetValue()
		}
	}
	return 0
}
