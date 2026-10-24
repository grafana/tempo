package search

import (
	"testing"
	"time"

	"github.com/grafana/tempo/v3/integration/util"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestTagEndpointsDedicatedColumns(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		// both attributes are default dedicated columns, so backend blocks store them only there
		batch := util.MakeThriftBatchWithSpanCountAttributeAndName(2, "foo", "ns-1", "GET", "k8s.namespace.name", "http.request.method")
		require.NoError(t, h.WriteJaegerBatch(batch, ""))

		h.WaitTracesWrittenToBackend(t, 1)
		h.ForceBackendQuerying(t)

		now := time.Now()
		start := now.Add(-2 * time.Hour).Unix()
		end := now.Add(2 * time.Hour).Unix()

		t.Run("tags_backend_resource", func(t *testing.T) {
			callSearchTagsV2AndAssert(t, h, "resource", "", &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{{Name: "resource", Tags: []string{"k8s.namespace.name", "service.name"}}},
			}, start, end)
		})

		t.Run("tags_backend_span", func(t *testing.T) {
			callSearchTagsV2AndAssert(t, h, "span", "", &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{{Name: "span", Tags: []string{"http.request.method"}}},
			}, start, end)
		})

		t.Run("values_backend_resource", func(t *testing.T) {
			callSearchTagValuesV2AndAssert(t, h, "resource.k8s.namespace.name", "", &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: "ns-1"}},
			}, start, end)
		})

		t.Run("values_backend_span", func(t *testing.T) {
			callSearchTagValuesV2AndAssert(t, h, "span.http.request.method", "", &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: "GET"}},
			}, start, end)
		})

		t.Run("tags_v1_backend", func(t *testing.T) {
			callSearchTagsAndAssert(t, h, []string{"http.request.method", "k8s.namespace.name", "service.name"}, start, end)
		})

		t.Run("tag_values_v1_backend", func(t *testing.T) {
			callSearchTagValuesAndAssert(t, h, "k8s.namespace.name", []string{"ns-1"}, start, end)
		})
	})
}
