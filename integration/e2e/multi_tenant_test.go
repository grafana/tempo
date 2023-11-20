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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/integration/e2e/backend"
)

const (
	configMultiTenant = "config-multi-tenant-local.yaml"
)

type traceStringsMap struct {
	rKeys     []string
	rValues   []string
	spanNames []string
}

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
			name:       "wildcard tenant",
			tenant:     "*", // tenant id "*" is same as a tenant with name '*', no special handling...
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
			var traceMap traceStringsMap

			tenants := strings.Split(tc.tenant, "|")
			require.Equal(t, tc.tenantSize, len(tenants))

			var expected float64
			// write traces for all tenants
			for _, tenant := range tenants {
				info = tempoUtil.NewTraceInfo(time.Now(), tenant)
				fmt.Printf("==== info: %v, tenant: %v \n", info, tenant)
				require.NoError(t, info.EmitAllBatches(c))

				trace, err := info.ConstructTraceFromEpoch()
				// store it to assert tests
				traceMap = getAttrsAndSpanNames(trace)

				fmt.Printf("==== rKeys: %v, tenant: %v \n", traceMap.rKeys, tenant)
				fmt.Printf("==== rValues: %v, tenant: %v \n", traceMap.rValues, tenant)
				fmt.Printf("==== sNames: %v, tenant: %v \n", traceMap.spanNames, tenant)

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

			// we create one trace for each tenant
			require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(float64(tc.tenantSize)), "tempo_ingester_traces_created_total"))
			// check that all spans are written
			require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(expected), "tempo_distributor_spans_received_total"))

			// Wait for the traces to be written to the WAL
			time.Sleep(time.Second * 3)

			// test echo
			assertEcho(t, "http://"+tempo.Endpoint(3200)+"/api/echo")

			// pass tenantID as id from test case
			apiClient := httpclient.New("http://"+tempo.Endpoint(3200), tc.tenant)

			// query an in-memory trace, this tests trace by id search
			// FIXME: maybe I need to make the API call directly here instead of using queryAndAssertTrace method??
			// queryAndAssertTrace(t, apiClient, info)

			// single traceid can only belong on single tenant so we will only see results from one tenant
			// query a random trace id??
			// FIXME: run this query for all tenants??
			// for _, tname := range tenants {
			// fmt.Printf("==== traceID: %v \n", info[tenants[0]].HexID())
			// response

			resp, err := apiClient.QueryTrace(info.HexID())
			require.NoError(t, err)
			respTm := getAttrsAndSpanNames(resp)
			// fmt.Printf("==== resp rKeys: %v, tenant: %v \n", tm.rKeys)
			// fmt.Printf("==== resp rValues: %v, tenant: %v \n", tm.rValues, tname)
			// fmt.Printf("==== resp sNames: %v, tenant: %v \n", tm.spanNames, tname)

			if tc.tenantSize > 1 {
				// resource keys should contain tenant key in case of a multi-tenant query
				traceMap.rKeys = append(traceMap.rKeys, "tenant")
				// resource values will contain at-least one of tenant ids for multi-tenant query
				// or exactly match in case of single tenant query
				assert.Subset(t, append(traceMap.rValues, tenants...), respTm.rValues)
			} else {
				assert.ElementsMatch(t, traceMap.rValues, respTm.rValues)
			}
			assert.ElementsMatch(t, respTm.rKeys, traceMap.rKeys)
			assert.ElementsMatch(t, traceMap.spanNames, respTm.spanNames)

			// flush trace to backend
			callFlush(t, tempo)

			// SearchAndAssertTrace also calls SearchTagValues
			util.SearchAndAssertTrace(t, apiClient, info)
			util.SearchTraceQLAndAssertTrace(t, apiClient, info)

			// force clear completed block
			callFlush(t, tempo)

			// wait for flush to complete
			time.Sleep(3 * time.Second)

			// Search for tags
			_, err = apiClient.SearchTags()
			require.NoError(t, err)

			// tagsExp := []string{"service.name", "vulture-0", "vulture-1", "vulture-2", "vulture-3", "vulture-process-0", "vulture-process-1", "vulture-process-2", "vulture-process-3"}
			// util.SearchAndAssertTags(t, apiClient, &tempopb.SearchTagsResponse{TagNames: tagsExp})

			// intrinsicScope := &tempopb.SearchTagsV2Scope{Name: "intrinsic", Tags: []string{"duration", "kind", "name", "rootName", "rootServiceName", "status", "statusMessage", "traceDuration"}}
			// resourceScope := &tempopb.SearchTagsV2Scope{Name: "resource", Tags: []string{"service.name", "vulture-process-0", "vulture-process-1", "vulture-process-2", "vulture-process-3"}}
			// spanScope := &tempopb.SearchTagsV2Scope{Name: "span", Tags: []string{"vulture-0", "vulture-1", "vulture-2", "vulture-3"}}
			// util.SearchAndAssertTagsV2(t, apiClient, &tempopb.SearchTagsV2Response{Scopes: []*tempopb.SearchTagsV2Scope{intrinsicScope, resourceScope, spanScope}})
			tagRespV2, err := apiClient.SearchTagsV2()
			require.NoError(t, err)

			// fmt.Printf("==== err: %v\n", err)
			fmt.Printf("==== tagRespV2: %v\n", tagRespV2)

			// FIXME: fix this??
			// v1ValuesExp := &tempopb.SearchTagValuesResponse{TagValues: []string{"bar", "qux"}}
			// util.SearchAndAssertTagValues(t, apiClient, "vulture-0", v1ValuesExp)
			tagValuesResp, err := apiClient.SearchTagValues("vulture-0")
			require.NoError(t, err)
			fmt.Printf("==== tagValuesResp: %v, len: %d \n", tagValuesResp, len(tagValuesResp.TagValues))

			// FIXME: fix this??
			// v2ValuesExp := &tempopb.SearchTagValuesV2Response{TagValues: []*tempopb.TagValue{{Type: "string", Value: "bar"}, {Type: "string", Value: "qux"}}}
			// util.SearchAndAssertTagValuesV2(t, apiClient, "span.vulture-0", "{}", v2ValuesExp)
			tagValuesRespV2, err := apiClient.SearchTagValuesV2("span.vulture-0", "{}")
			require.NoError(t, err)
			fmt.Printf("==== tagValuesRespV2: %v, len: %d\n", tagValuesRespV2, len(tagValuesRespV2.TagValues))

			// dump metrics, TODO: REMOVE THIS
			met := callMetrics(t, tempo)
			// // fmt.Printf("/metrics: %v \n", met)
			err = os.WriteFile("/home/suraj/wd/grafana/tempo/metrics_"+tc.tenant+".txt", met, 0644)
			require.NoError(t, err)

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
				assertRequestCountMetric(t, tempo, rt.route, rt.reqCount)
			}

		})
	}
}

func assertRequestCountMetric(t *testing.T, s *e2e.HTTPService, route string, reqCount int) {
	fmt.Printf("=== assertRequestCountMetric route: %v, rt.reqCount: %v \n", route, reqCount)

	err := s.WaitSumMetricsWithOptions(e2e.Equals(float64(reqCount)),
		[]string{"tempo_request_duration_seconds"},
		e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "route", route)),
		e2e.WithMetricCount, // get count from histogram metric
	)
	require.NoError(t, err)
}

// getAttrsAndSpanNames returns trace attrs and span names
func getAttrsAndSpanNames(trace *tempopb.Trace) traceStringsMap {
	rAttrsKeys := tempoUtil.NewDistinctStringCollector(0)
	rAttrsValues := tempoUtil.NewDistinctStringCollector(0)
	spanNames := tempoUtil.NewDistinctStringCollector(0)

	for _, b := range trace.Batches {
		for _, ss := range b.ScopeSpans {
			for _, s := range ss.Spans {
				if s.Name != "" {
					spanNames.Collect(s.Name)
				}
			}
		}
		for _, a := range b.Resource.Attributes {
			if a.Key != "" {
				rAttrsKeys.Collect(a.Key)
			}
			if a.Value.GetStringValue() != "" {
				rAttrsValues.Collect(a.Value.GetStringValue())
			}
		}
	}

	return traceStringsMap{
		rKeys:     rAttrsKeys.Strings(),
		rValues:   rAttrsValues.Strings(),
		spanNames: spanNames.Strings(),
	}
}
