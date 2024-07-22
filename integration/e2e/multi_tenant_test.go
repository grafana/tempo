package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/grafana/e2e"
	"github.com/grafana/tempo/pkg/collector"
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

// TestMultiTenantSearch tests multi tenant query support
func TestMultiTenantSearch(t *testing.T) {
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
			require.NoError(t, s.StartAndWaitReady(tempo, newPrometheus()))

			// Get port for the Jaeger gRPC receiver endpoint
			c, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
			require.NoError(t, err)
			require.NotNil(t, c)

			var info *tempoUtil.TraceInfo
			var traceMap traceStringsMap

			tenants := strings.Split(tc.tenant, "|")
			require.Equal(t, tc.tenantSize, len(tenants))

			var expectedSpans float64
			// write traces for all tenants
			for _, tenant := range tenants {
				info = tempoUtil.NewTraceInfo(time.Now(), tenant)
				require.NoError(t, info.EmitAllBatches(c))

				trace, err := info.ConstructTraceFromEpoch()
				traceMap = getAttrsAndSpanNames(trace) // store it to assert tests

				require.NoError(t, err)
				expectedSpans = expectedSpans + spanCount(trace)
			}

			// assert that we have one trace and each tenant and correct number of spans received
			require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(float64(tc.tenantSize)), "tempo_ingester_traces_created_total"))
			require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(expectedSpans), "tempo_distributor_spans_received_total"))

			// Wait for the traces to be written to the WAL
			time.Sleep(time.Second * 3)

			// test echo
			assertEcho(t, "http://"+tempo.Endpoint(3200)+"/api/echo")

			// client will have testcase tenant id
			apiClient := httpclient.New("http://"+tempo.Endpoint(3200), tc.tenant)

			// check trace by id
			resp, err := apiClient.QueryTrace(info.HexID())
			require.NoError(t, err)
			respTm := getAttrsAndSpanNames(resp)

			assert.ElementsMatch(t, traceMap.rValues, respTm.rValues)
			assert.ElementsMatch(t, respTm.rKeys, traceMap.rKeys)
			assert.ElementsMatch(t, traceMap.spanNames, respTm.spanNames)

			// flush trace to backend
			callFlush(t, tempo)

			// search and traceql search, note: SearchAndAssertTrace also calls SearchTagValues
			util.SearchAndAssertTrace(t, apiClient, info)
			util.SearchTraceQLAndAssertTrace(t, apiClient, info)

			// force clear completed block
			callFlush(t, tempo)

			// wait for flush to complete for all tenants, each tenant will have one block
			require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(float64(tc.tenantSize)), "tempo_ingester_blocks_flushed_total"))

			// call search tags endpoints, ensure no errors and results are not empty
			tagsResp, err := apiClient.SearchTags()
			require.NoError(t, err)
			require.NotEmpty(t, tagsResp.TagNames)

			tagsV2Resp, err := apiClient.SearchTagsV2()
			require.NoError(t, err)
			require.Equal(t, 4, len(tagsV2Resp.GetScopes())) // resource, span, event, intrinsics
			for _, s := range tagsV2Resp.Scopes {
				require.NotEmpty(t, s.Tags)
			}

			tagsValuesResp, err := apiClient.SearchTagValues("vulture-0")
			require.NoError(t, err)
			require.NotEmpty(t, tagsValuesResp.TagValues)

			tagsValuesV2Resp, err := apiClient.SearchTagValuesV2("span.vulture-0", "{}")
			require.NoError(t, err)
			require.NotEmpty(t, tagsValuesV2Resp.TagValues)

			// check metrics for all routes
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

			// test streaming search over grpc
			grpcCtx := user.InjectOrgID(context.Background(), tc.tenant)
			grpcCtx, err = user.InjectIntoGRPCRequest(grpcCtx)
			require.NoError(t, err)

			grpcClient, err := util.NewSearchGRPCClient(grpcCtx, tempo.Endpoint(3200))
			require.NoError(t, err)

			time.Sleep(2 * time.Second) // ensure that blocklist poller has built the blocklist
			now := time.Now()
			util.SearchStreamAndAssertTrace(t, grpcCtx, grpcClient, info, now.Add(-10*time.Minute).Unix(), now.Add(10*time.Minute).Unix())
			assertRequestCountMetric(t, tempo, "/tempopb.StreamingQuerier/Search", 1)

			// test unsupported endpoint
			_, msErr := apiClient.MetricsSummary("{}", "name", 0, 0)
			if tc.tenantSize > 1 {
				// error for multi-tenant request for unsupported endpoints
				require.Error(t, msErr)
			} else {
				require.NoError(t, msErr)
			}
		})
	}
}

func assertRequestCountMetric(t *testing.T, s *e2e.HTTPService, route string, reqCount int) {
	fmt.Printf("==== %s, assertRequestCountMetric route: %v, rt.reqCount: %v \n", t.Name(), route, reqCount)

	err := s.WaitSumMetricsWithOptions(e2e.Equals(float64(reqCount)),
		[]string{"tempo_request_duration_seconds"},
		e2e.WithLabelMatchers(labels.MustNewMatcher(labels.MatchEqual, "route", route)),
		e2e.WithMetricCount, // get count from histogram metric
	)
	require.NoError(t, err)
}

// getAttrsAndSpanNames returns trace attrs and span names
func getAttrsAndSpanNames(trace *tempopb.Trace) traceStringsMap {
	rAttrsKeys := collector.NewDistinctString(0)
	rAttrsValues := collector.NewDistinctString(0)
	spanNames := collector.NewDistinctString(0)

	for _, b := range trace.ResourceSpans {
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
