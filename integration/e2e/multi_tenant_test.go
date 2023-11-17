package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/integration/e2e/backend"
)

const (
	configMultiTenant = "config-multi-tenant-local.yaml"
)

func TestMultiTenantSearch(t *testing.T) {
	// test multi tenant query support

	// allows multi tenant query for following endpoints
	// search, search streaming, tracebyid, search tags
	// handles following cases: 1. single tenant, 2. multiple tenants, 3. * is treated as regular tenant

	testTenants := []struct {
		name       string
		tenant     string
		tenantSize int
	}{
		{
			name:       "single tenant",
			tenant:     "test",
			tenantSize: 1,
		},
		{
			name:       "two tenants",
			tenant:     "test|test2",
			tenantSize: 2,
		},
		{
			name:       "multiple tenants",
			tenant:     "test|test2|test3",
			tenantSize: 3,
		},
		// FIXME: see what mimir and loki are doing for * and follow the same behaviour here
		{
			name:       "wildcard tenant",
			tenant:     "*",
			tenantSize: 1,
		},
	}

	for _, tc := range testTenants {
		t.Run(tc.name, func(t *testing.T) {
			s, err := e2e.NewScenario("tempo_e2e")
			require.NoError(t, err)
			defer s.Close()

			// set up the backend
			cfg := app.Config{}
			buff, err := os.ReadFile(configMultiTenant)
			require.NoError(t, err)
			err = yaml.UnmarshalStrict(buff, &cfg)
			require.NoError(t, err)
			_, err = backend.New(s, cfg)
			require.NoError(t, err)

			require.NoError(t, util.CopyFileToSharedDir(s, configMultiTenant, "config.yaml"))
			tempo := util.NewTempoAllInOne()
			require.NoError(t, s.StartAndWaitReady(tempo))

			// Get port for the Jaeger gRPC receiver endpoint
			c, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
			require.NoError(t, err)
			require.NotNil(t, c)

			var info *tempoUtil.TraceInfo
			tenants := strings.Split(tc.tenant, "|")
			require.Equal(t, tc.tenantSize, len(tenants))

			var expected float64
			// write traces for all tenants
			for _, tenant := range tenants {
				info = tempoUtil.NewTraceInfo(time.Now(), tenant)
				fmt.Printf("==== info: %v, tenant: %v \n", info, tenant)
				require.NoError(t, info.EmitAllBatches(c))

				trace, err := info.ConstructTraceFromEpoch()
				// rKeys, rValues, sNames := getAttrsAndSpanNames(trace)
				// fmt.Printf("==== rKeys: %v, tenant: %v \n", rKeys, tenant)
				// fmt.Printf("==== rValues: %v, tenant: %v \n", rValues, tenant)
				// fmt.Printf("==== sNames: %v, tenant: %v \n", sNames, tenant)

				// fmt.Printf("==== trace: %v, tenant: %v \n", trace, tenant)
				require.NoError(t, err)
				expected = expected + spanCount(trace)

				// emit some spans with tags and values to assert later
				// batch := makeThriftBatchWithSpanCountAttributeAndName(2, "foo", "bar")
				// require.NoError(t, c.EmitBatch(context.Background(), batch))
				//
				// batch2 := makeThriftBatchWithSpanCountAttributeAndName(2, "baz", "qux")
				// require.NoError(t, c.EmitBatch(context.Background(), batch2))
			}

			// test metrics to check that traces for all tenants are written
			require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(expected), "tempo_distributor_spans_received_total"))

			// Wait for the traces to be written to the WAL
			time.Sleep(time.Second * 3)

			// test echo
			assertEcho(t, "http://"+tempo.Endpoint(3200)+"/api/echo")

			// pass tenantID as id from test case
			apiClient := httpclient.New("http://"+tempo.Endpoint(3200), tc.tenant)

			// query an in-memory trace, this tests trace by id search
			// FIXME: maybe I need to make the API call directly here instead of using queryAndAssertTrace method??
			queryAndAssertTrace(t, apiClient, info)

			// wait trace_idle_time and ensure trace is created in ingester
			// FIXME: skip this test for a while? need to figure this out for now??
			// FIXME: match for labels for each tenant in this case, we want metrics for each tenant??
			// require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

			// flush trace to backend
			callFlush(t, tempo)

			// TODO: SearchAndAssertTrace also calls SearchTagValues??
			util.SearchAndAssertTrace(t, apiClient, info)
			util.SearchTraceQLAndAssertTrace(t, apiClient, info)

			// force clear completed block
			callFlush(t, tempo)

			// wait for flush to complete
			time.Sleep(3 * time.Second)

			// Search for tags
			tagsExp := []string{"service.name", "vulture-0", "vulture-1", "vulture-2", "vulture-3", "vulture-process-0", "vulture-process-1", "vulture-process-2", "vulture-process-3"}
			util.SearchAndAssertTags(t, apiClient, &tempopb.SearchTagsResponse{TagNames: tagsExp})

			intrinsicScope := &tempopb.SearchTagsV2Scope{Name: "intrinsic", Tags: []string{"duration", "kind", "name", "rootName", "rootServiceName", "status", "statusMessage", "traceDuration"}}
			resourceScope := &tempopb.SearchTagsV2Scope{Name: "resource", Tags: []string{"service.name", "vulture-process-0", "vulture-process-1", "vulture-process-2", "vulture-process-3"}}
			spanScope := &tempopb.SearchTagsV2Scope{Name: "span", Tags: []string{"vulture-0", "vulture-1", "vulture-2", "vulture-3"}}
			util.SearchAndAssertTagsV2(t, apiClient, &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{intrinsicScope, resourceScope, spanScope}})

			v1ValuesExp := &tempopb.SearchTagValuesResponse{TagValues: []string{"bar", "qux"}}
			util.SearchAndAssertTagValues(t, apiClient, "vulture-0", v1ValuesExp)

			v2ValuesExp := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Type: "string", Value: "bar"}, {Type: "string", Value: "qux"}}}
			util.SearchAndAssertTagValuesV2(t, apiClient, "span.vulture-0", "{}", v2ValuesExp)

			// dump metrics, REMOVE THIS
			// met := callMetrics(t, tempo)
			// // fmt.Printf("/metrics: %v \n", met)
			// err = os.WriteFile("/home/suraj/wd/grafana/tempo/metrics_"+tc.tenant+".txt", met, 0644)
			// require.NoError(t, err)

			if tc.tenantSize > 1 {
				for _, ta := range tenants {
					matcher, err := labels.NewMatcher(labels.MatchEqual, "tenant", ta)
					require.NoError(t, err)
					// check multi-tenant search metrics, 8 calls for each tenant, and 0 failures
					err = tempo.WaitSumMetricsWithOptions(e2e.Equals(8),
						[]string{"tempo_tenant_federation_success_total"},
						e2e.WithLabelMatchers(matcher),
					)
					require.NoError(t, err)

					err = tempo.WaitSumMetricsWithOptions(e2e.Equals(8),
						[]string{"tempo_tenant_federation_failures_total"},
						e2e.WithLabelMatchers(matcher),
					)
					require.NoError(t, err)
				}
			}

			routeTable := []struct {
				route    string
				reqCount int
			}{
				// query frontend routes
				{route: "api_search", reqCount: 2}, // called twice
				{route: "api_traces_traceid", reqCount: 1},
				{route: "api_search_tags", reqCount: 1},
				{route: "api_search_tag_tagname_values", reqCount: 2}, // called twice
				{route: "api_v2_search_tags", reqCount: 1},
				{route: "api_v2_search_tag_tagname_values", reqCount: 1},

				// Querier routes, we make one request for each tenant
				{route: "/tempopb.Querier/SearchRecent", reqCount: 2 * tc.tenantSize}, // called twice
				{route: "/tempopb.Querier/FindTraceByID", reqCount: tc.tenantSize},
				{route: "/tempopb.Querier/SearchTags", reqCount: tc.tenantSize},
				{route: "/tempopb.Querier/SearchTagsV2", reqCount: tc.tenantSize},
				{route: "/tempopb.Querier/SearchTagValues", reqCount: 2 * tc.tenantSize}, // called twice
				{route: "/tempopb.Querier/SearchTagValuesV2", reqCount: tc.tenantSize},
			}

			for _, rt := range routeTable {
				fmt.Printf("=== route: %v, rt.reqCount: %v \n", rt.route, rt.reqCount)
				assertRequestCountMetric(t, tempo, rt.route, rt.reqCount)
			}

		})
	}
}

func assertRequestCountMetric(t *testing.T, s *e2e.HTTPService, route string, reqCount int) {
	err := s.WaitSumMetricsWithOptions(e2e.Equals(float64(reqCount)),
		[]string{"tempo_request_duration_seconds"},
		e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "route", route)),
		e2e.WithMetricCount, // get count from histogram metric
	)
	require.NoError(t, err)
}

// FIXME: remove??
func getAttrsAndSpanNames(trace *tempopb.Trace) ([]string, []string, []string) {
	// trace.Batches loop over
	// Resource.Attributes loop over and get key and values -> this is resource stuff
	// ScopeSpans.Spans loop over and collect name
	// this will give us enough info to assert stuff??

	rAttrsKeys := make([]string, 10)
	rAttrsValues := make([]string, 10)
	spanNames := make([]string, 10)

	for _, b := range trace.Batches {
		for _, l := range b.ScopeSpans {
			for _, s := range l.Spans {
				spanNames = append(spanNames, s.Name)
			}
		}
		for _, a := range b.Resource.Attributes {
			rAttrsKeys = append(rAttrsKeys, a.Key)
			rAttrsValues = append(rAttrsValues, a.Value.GetStringValue())
		}
	}
	return rAttrsKeys, rAttrsValues, spanNames
}
