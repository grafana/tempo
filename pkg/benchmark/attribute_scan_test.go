package benchmark

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	v1_common "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/tempodb/backend"
)

// The attribute fixture has attributeTraces traces of attributeSpansPerTrace
// spans each, so every expected fraction is a count over attributeSpans.
const (
	attributeTraces        = 200
	attributeSpansPerTrace = 4
	attributeSpans         = attributeTraces * attributeSpansPerTrace
)

// attributeTrace gives trace i attributes whose statistics are known:
//   - resource.service.name alternates svc-a and svc-b by trace; resource.cluster is always prod
//   - span j is named op-j and lasts j+1 seconds; span 3 has status error, the rest ok
//   - span.http.method, a dedicated column, is GET on spans 0 and 1 and POST on span 2
//   - span.http.status_code is 200 on spans 0-2 and 500 on span 3
//   - span.seq is distinct on every span, so it is profiled by quantiles
//   - span.tags, on span 0 only, is the array [a, b, a]
func attributeTrace(i int, id []byte, now time.Time) *tempopb.Trace {
	str := func(k, v string) *v1_common.KeyValue {
		return &v1_common.KeyValue{Key: k, Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: v}}}
	}
	integer := func(k string, v int64) *v1_common.KeyValue {
		return &v1_common.KeyValue{Key: k, Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_IntValue{IntValue: v}}}
	}
	strArray := func(k string, vs ...string) *v1_common.KeyValue {
		arr := &v1_common.ArrayValue{}
		for _, v := range vs {
			arr.Values = append(arr.Values, &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: v}})
		}
		return &v1_common.KeyValue{Key: k, Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_ArrayValue{ArrayValue: arr}}}
	}

	service := "svc-a"
	if i%2 == 1 {
		service = "svc-b"
	}

	spans := make([]*v1_trace.Span, attributeSpansPerTrace)
	for j := range spans {
		seq := i*attributeSpansPerTrace + j
		spanID := make([]byte, 8)
		binary.BigEndian.PutUint64(spanID, uint64(seq+1))

		attrs := []*v1_common.KeyValue{integer("seq", int64(seq))}
		status := &v1_trace.Status{Code: v1_trace.Status_STATUS_CODE_OK}
		switch j {
		case 0:
			attrs = append(attrs, str("http.method", "GET"), integer("http.status_code", 200), strArray("tags", "a", "b", "a"))
		case 1:
			attrs = append(attrs, str("http.method", "GET"), integer("http.status_code", 200))
		case 2:
			attrs = append(attrs, str("http.method", "POST"), integer("http.status_code", 200))
		case 3:
			attrs = append(attrs, integer("http.status_code", 500))
			status = &v1_trace.Status{Code: v1_trace.Status_STATUS_CODE_ERROR}
		}

		spans[j] = &v1_trace.Span{
			TraceId:           id,
			SpanId:            spanID,
			Name:              fmt.Sprintf("op-%d", j),
			Kind:              v1_trace.Span_SPAN_KIND_SERVER,
			Status:            status,
			StartTimeUnixNano: uint64(now.UnixNano()),
			EndTimeUnixNano:   uint64(now.Add(time.Duration(j+1) * time.Second).UnixNano()),
			Attributes:        attrs,
		}
	}

	return &tempopb.Trace{ResourceSpans: []*v1_trace.ResourceSpans{{
		Resource: &v1_resource.Resource{Attributes: []*v1_common.KeyValue{
			str("service.name", service),
			str("cluster", "prod"),
		}},
		ScopeSpans: []*v1_trace.ScopeSpans{{Spans: spans}},
	}}}
}

func attributeTestBlock(t *testing.T) (*backend.BlockMeta, backend.Reader) {
	t.Helper()

	ids := make([][]byte, attributeTraces)
	for i := range ids {
		ids[i] = make([]byte, 16)
		binary.BigEndian.PutUint64(ids[i][8:], uint64(i+1))
	}
	dedicated := backend.DedicatedColumns{{Scope: backend.DedicatedColumnScopeSpan, Name: "http.method", Type: backend.DedicatedColumnTypeString}}

	meta, r, _ := writeTestBlock(t, ids, dedicated, attributeTrace)
	require.Equal(t, dedicated, meta.DedicatedColumns, "the writer dropped the dedicated column")

	rowGroups, err := rowGroupCount(context.Background(), meta, r)
	require.NoError(t, err)
	require.Greater(t, rowGroups, 1, "need more than one row group to read across")

	return meta, r
}

func findAttribute(t *testing.T, ps []AttributeProfile, scope, name, typ string) AttributeProfile {
	t.Helper()
	for _, p := range ps {
		if p.Scope == scope && p.Name == name && p.Type == typ {
			return p
		}
	}
	require.FailNow(t, "attribute not profiled", "%s.%s (%s)", scope, name, typ)
	return AttributeProfile{}
}

func rankedKeys(ps []AttributeProfile) []string {
	keys := make([]string, 0, len(ps))
	for _, p := range ps {
		keys = append(keys, p.Scope+"/"+p.Name)
	}
	return keys
}

func TestProfileAttributes(t *testing.T) {
	meta, r := attributeTestBlock(t)

	got, err := profileBlockAttributes(context.Background(), meta, r, 100)
	require.NoError(t, err)
	require.NoError(t, got.validate())
	require.Equal(t, uint64(attributeSpans), got.Spans)

	// Resource attributes are stored once per trace here, so they rank below
	// span attributes carried by every span. Ties on bytes break on scope then
	// name.
	require.Equal(t, []string{
		"intrinsic/duration", "span/http.status_code", "span/seq", // 8 bytes on every span
		"intrinsic/name",
		"span/http.method",
		"resource/service.name",
		"intrinsic/kind", "intrinsic/status", "resource/cluster",
		"span/tags",
	}, rankedKeys(got.Ranked))

	for _, want := range []AttributeProfile{
		{
			Scope: AttributeScopeResource, Name: "service.name", Type: AttributeTypeString,
			Dedicated: true, TotalBytes: 5 * attributeTraces, Cardinality: 2, Density: 1,
			Values: []AttributeValue{{Value: "svc-a", Selectivity: 0.5}, {Value: "svc-b", Selectivity: 0.5}},
		},
		{
			Scope: AttributeScopeResource, Name: "cluster", Type: AttributeTypeString,
			TotalBytes: 4 * attributeTraces, Cardinality: 1, Density: 1,
			Values: []AttributeValue{{Value: "prod", Selectivity: 1}},
		},
		{
			Scope: AttributeScopeSpan, Name: "http.method", Type: AttributeTypeString,
			Dedicated: true, TotalBytes: 3*400 + 4*200, Cardinality: 2, Density: 0.75,
			Values: []AttributeValue{{Value: "GET", Selectivity: 0.5}, {Value: "POST", Selectivity: 0.25}},
		},
		{
			Scope: AttributeScopeSpan, Name: "http.status_code", Type: AttributeTypeInt,
			TotalBytes: 8 * attributeSpans, Cardinality: 2, Density: 1,
			Values: []AttributeValue{{Value: "200", Selectivity: 0.75}, {Value: "500", Selectivity: 0.25}},
		},
		{
			Scope: AttributeScopeSpan, Name: "tags", Type: AttributeTypeString,
			TotalBytes: 3 * attributeTraces, Cardinality: 2, Density: 0.25,
			Values: []AttributeValue{{Value: "a", Selectivity: 0.25}, {Value: "b", Selectivity: 0.25}},
		},
		{
			Scope: AttributeScopeIntrinsic, Name: "name", Type: AttributeTypeString,
			Dedicated: true, TotalBytes: 4 * attributeSpans, Cardinality: 4, Density: 1,
			Values: []AttributeValue{
				{Value: "op-0", Selectivity: 0.25},
				{Value: "op-1", Selectivity: 0.25},
				{Value: "op-2", Selectivity: 0.25},
				{Value: "op-3", Selectivity: 0.25},
			},
		},
		{
			Scope: AttributeScopeIntrinsic, Name: "status", Type: AttributeTypeStatus,
			Dedicated: true, TotalBytes: attributeSpans, Cardinality: 2, Density: 1,
			Values: []AttributeValue{{Value: "ok", Selectivity: 0.75}, {Value: "error", Selectivity: 0.25}},
		},
		{
			Scope: AttributeScopeIntrinsic, Name: "kind", Type: AttributeTypeKind,
			Dedicated: true, TotalBytes: attributeSpans, Cardinality: 1, Density: 1,
			Values: []AttributeValue{{Value: "server", Selectivity: 1}},
		},
		{
			// Few distinct durations, so they are listed rather than bucketed.
			Scope: AttributeScopeIntrinsic, Name: "duration", Type: AttributeTypeDuration,
			Dedicated: true, TotalBytes: 8 * attributeSpans, Cardinality: 4, Density: 1,
			Values: []AttributeValue{
				{Value: "1s", Selectivity: 0.25},
				{Value: "2s", Selectivity: 0.25},
				{Value: "3s", Selectivity: 0.25},
				{Value: "4s", Selectivity: 0.25},
			},
		},
	} {
		require.Equal(t, want, findAttribute(t, got.Ranked, want.Scope, want.Name, want.Type))
	}

	seq := findAttribute(t, got.Ranked, AttributeScopeSpan, "seq", AttributeTypeInt)
	require.False(t, seq.Dedicated)
	require.Empty(t, seq.Values)
	require.Greater(t, seq.Cardinality, uint64(topValues))
	require.Len(t, seq.Quantiles, len(profileQuantiles))
	for _, q := range seq.Quantiles {
		v, err := strconv.Atoi(q.Value)
		require.NoError(t, err)

		above := 0
		for s := range attributeSpans {
			if s > v {
				above++
			}
		}
		require.InDelta(t, float64(above)/attributeSpans, q.Selectivity, 1e-12, "q=%v value=%v", q.Q, q.Value)
	}
}

func TestProfileAttributesKeepsTopN(t *testing.T) {
	meta, r := attributeTestBlock(t)

	got, err := profileBlockAttributes(context.Background(), meta, r, 3)
	require.NoError(t, err)
	require.Equal(t, []string{"intrinsic/duration", "span/http.status_code", "span/seq"}, rankedKeys(got.Ranked))
}

// Two resources with the same attributes are still stored twice, and a
// resource without spans matches nothing, so it is not counted at all.
func TestProfileAttributesCountsEachResource(t *testing.T) {
	str := func(k, v string) *v1_common.KeyValue {
		return &v1_common.KeyValue{Key: k, Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: v}}}
	}
	resource := func(spans int) *v1_trace.ResourceSpans {
		ss := &v1_trace.ScopeSpans{}
		for range spans {
			ss.Spans = append(ss.Spans, &v1_trace.Span{Name: "op", EndTimeUnixNano: 1})
		}
		return &v1_trace.ResourceSpans{
			Resource:   &v1_resource.Resource{Attributes: []*v1_common.KeyValue{str("service.name", "svc"), {Key: "empty"}}},
			ScopeSpans: []*v1_trace.ScopeSpans{ss},
		}
	}

	iter := &sliceIterator{}
	iter.add([]byte{1}, &tempopb.Trace{ResourceSpans: []*v1_trace.ResourceSpans{resource(2), resource(1), resource(0)}})

	got, err := profileAttributes(context.Background(), iter, nil, 100)
	require.NoError(t, err)
	require.Equal(t, uint64(3), got.Spans)

	require.Equal(t, AttributeProfile{
		Scope: AttributeScopeResource, Name: "service.name", Type: AttributeTypeString,
		Dedicated: true, TotalBytes: 2 * 3, Cardinality: 1, Density: 1,
		Values: []AttributeValue{{Value: "svc", Selectivity: 1}},
	}, findAttribute(t, got.Ranked, AttributeScopeResource, "service.name", AttributeTypeString))
	require.NotContains(t, rankedKeys(got.Ranked), "resource/empty", "an attribute without a value is skipped")
}

func TestProfileAttributesNoSpans(t *testing.T) {
	_, err := profileAttributes(context.Background(), &sliceIterator{}, nil, 100)
	require.ErrorContains(t, err, "no spans")
}

func TestOpenTraceIterator(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 50)

	iter, err := openTraceIterator(ctx, meta, r)
	require.NoError(t, err)
	defer iter.Close()

	var traces, spans int
	for {
		id, tr, err := iter.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		require.Len(t, id, 16)
		traces++
		for _, rs := range tr.ResourceSpans {
			for _, ss := range rs.ScopeSpans {
				spans += len(ss.Spans)
			}
		}
	}
	require.Equal(t, 50, traces)
	require.Equal(t, 50*4, spans)
}

func TestOpenTraceIteratorRejectsOtherVersions(t *testing.T) {
	meta := backend.NewBlockMeta("test-tenant", uuid.New(), "vParquet4")
	_, err := openTraceIterator(context.Background(), meta, nil)
	require.ErrorContains(t, err, "vParquet4 blocks is not supported")
}

func TestHasOwnColumn(t *testing.T) {
	dedicated := backend.DedicatedColumns{{Scope: backend.DedicatedColumnScopeSpan, Name: "http.method", Type: backend.DedicatedColumnTypeString}}

	for _, tc := range []struct {
		key  attrKey
		want bool
	}{
		{attrKey{AttributeScopeIntrinsic, "name", AttributeTypeString}, true},
		{attrKey{AttributeScopeResource, "service.name", AttributeTypeString}, true},
		{attrKey{AttributeScopeSpan, "service.name", AttributeTypeString}, false},
		{attrKey{AttributeScopeSpan, "http.method", AttributeTypeString}, true},
		// A dedicated string column does not hold the attribute's int values.
		{attrKey{AttributeScopeSpan, "http.method", AttributeTypeInt}, false},
		{attrKey{AttributeScopeResource, "http.method", AttributeTypeString}, false},
	} {
		require.Equal(t, tc.want, hasOwnColumn(tc.key, dedicated), "%+v", tc.key)
	}
}
