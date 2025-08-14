package querier

import (
	"context"
	"sort"
	"testing"

	"github.com/grafana/dskit/user"
	generator_client "github.com/grafana/tempo/modules/generator/client"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	livestore_client "github.com/grafana/tempo/modules/livestore/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestVirtualTagsDoesntHitBackend(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	q, err := New(Config{}, ingester_client.Config{}, nil, generator_client.Config{}, nil, false, livestore_client.Config{}, nil, nil, nil, o)
	require.NoError(t, err)

	ctx := user.InjectOrgID(context.Background(), "blerg")

	// duration should return nothing
	resp, err := q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "duration",
	})
	require.NoError(t, err)
	require.Equal(t, &tempopb.SearchTagValuesV2Response{Metrics: &tempopb.MetadataMetrics{}}, resp)

	// traceDuration should return nothing
	resp, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "traceDuration",
	})
	require.NoError(t, err)
	require.Equal(t, &tempopb.SearchTagValuesV2Response{Metrics: &tempopb.MetadataMetrics{}}, resp)

	// status should return a static list
	resp, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "status",
	})
	require.NoError(t, err)
	sort.Slice(resp.TagValues, func(i, j int) bool { return resp.TagValues[i].Value < resp.TagValues[j].Value })
	require.Equal(t, &tempopb.SearchTagValuesV2Response{
		TagValues: []*tempopb.TagValue{
			{
				Type:  "keyword",
				Value: "error",
			},
			{
				Type:  "keyword",
				Value: "ok",
			},
			{
				Type:  "keyword",
				Value: "unset",
			},
		},
		Metrics: &tempopb.MetadataMetrics{},
	}, resp)

	// kind should return a static list
	resp, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "kind",
	})
	require.NoError(t, err)
	sort.Slice(resp.TagValues, func(i, j int) bool { return resp.TagValues[i].Value < resp.TagValues[j].Value })
	require.Equal(t, &tempopb.SearchTagValuesV2Response{
		TagValues: []*tempopb.TagValue{
			{
				Type:  "keyword",
				Value: "client",
			},
			{
				Type:  "keyword",
				Value: "consumer",
			},
			{
				Type:  "keyword",
				Value: "internal",
			},
			{
				Type:  "keyword",
				Value: "producer",
			},
			{
				Type:  "keyword",
				Value: "server",
			},
			{
				Type:  "keyword",
				Value: "unspecified",
			},
		},
		Metrics: &tempopb.MetadataMetrics{},
	}, resp)

	// this should error b/c it will attempt to hit the un-configured backend
	resp, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: ".foo",
	})
	require.Error(t, err)
	require.Nil(t, resp)
}
