package vparquet4

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	v1_common "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

func TestOpNoneColumns(t *testing.T) {
	tcs := []struct {
		name                       string
		operands                   traceql.Operands
		wantStr, wantNum, wantBool bool
	}{
		{"no operands", nil, true, true, true},
		{"string", traceql.Operands{traceql.NewStaticString("a")}, true, false, false},
		{"int", traceql.Operands{traceql.NewStaticInt(1)}, false, true, false},
		{"float", traceql.Operands{traceql.NewStaticFloat(1.5)}, false, true, false},
		{"bool", traceql.Operands{traceql.NewStaticBool(true)}, false, false, true},
		{"several types", traceql.Operands{traceql.NewStaticString("a"), traceql.NewStaticBool(true)}, true, false, true},
		{"nil", traceql.Operands{traceql.NewStaticNil()}, true, true, true},
		{"duration", traceql.Operands{traceql.NewStaticDuration(1)}, true, true, true},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			gotStr, gotNum, gotBool := opNoneColumns(tc.operands)
			require.Equal(t, tc.wantStr, gotStr)
			require.Equal(t, tc.wantNum, gotNum)
			require.Equal(t, tc.wantBool, gotBool)
		})
	}
}

// TestFetchOpNoneOperands: the operands of an OpNone condition don't filter, they limit the types that
// are fetched. Only spans with a value of one of those types are returned.
func TestFetchOpNoneOperands(t *testing.T) {
	attr := traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, "n")

	var spans []*v1.Span
	for i, value := range []*v1_common.AnyValue{
		{Value: &v1_common.AnyValue_StringValue{StringValue: "a"}},
		{Value: &v1_common.AnyValue_IntValue{IntValue: 5}},
		{Value: &v1_common.AnyValue_DoubleValue{DoubleValue: 1.5}},
		{Value: &v1_common.AnyValue_BoolValue{BoolValue: true}},
		nil, // missing
	} {
		span := &v1.Span{
			SpanId:            test.ValidTraceID(nil)[:8],
			StartTimeUnixNano: uint64(1000 + i),
			EndTimeUnixNano:   2000,
		}
		if value != nil {
			span.Attributes = []*v1_common.KeyValue{{Key: "n", Value: value}}
		}
		spans = append(spans, span)
	}
	id := test.ValidTraceID(nil)
	pb := &tempopb.Trace{ResourceSpans: []*v1.ResourceSpans{{
		Resource:   &v1_resource.Resource{Attributes: []*v1_common.KeyValue{{Key: "service.name", Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "svc"}}}}},
		ScopeSpans: []*v1.ScopeSpans{{Spans: spans}},
	}}}
	tr, _ := traceToParquet(&backend.BlockMeta{}, id, pb, nil)
	block := makeBackendBlockWithTraces(t, []*Trace{tr})

	all := []traceql.StaticType{traceql.TypeString, traceql.TypeInt, traceql.TypeFloat, traceql.TypeBoolean}
	tcs := []struct {
		name     string
		operands traceql.Operands
		want     []traceql.StaticType
	}{
		{"no operands", nil, all},
		{"string", traceql.Operands{traceql.NewStaticString("x")}, []traceql.StaticType{traceql.TypeString}},
		{"int", traceql.Operands{traceql.NewStaticInt(1)}, []traceql.StaticType{traceql.TypeInt, traceql.TypeFloat}},
		{"float", traceql.Operands{traceql.NewStaticFloat(1)}, []traceql.StaticType{traceql.TypeInt, traceql.TypeFloat}},
		{"bool", traceql.Operands{traceql.NewStaticBool(false)}, []traceql.StaticType{traceql.TypeBoolean}},
		{"other type", traceql.Operands{traceql.NewStaticNil()}, all},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := block.Fetch(ctx, traceql.FetchSpansRequest{
				Conditions: []traceql.Condition{{Attribute: attr, Op: traceql.OpNone, Operands: tc.operands}},
			}, common.DefaultSearchOptions())
			require.NoError(t, err)

			var got []traceql.StaticType
			for {
				ss, err := resp.Results.Next(ctx)
				require.NoError(t, err)
				if ss == nil {
					break
				}
				for _, sp := range ss.Spans {
					v, ok := sp.AttributeFor(attr)
					require.True(t, ok)
					got = append(got, v.Type)
				}
			}
			require.ElementsMatch(t, tc.want, got)
		})
	}
}
