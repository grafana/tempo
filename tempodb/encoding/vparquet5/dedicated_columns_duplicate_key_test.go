package vparquet5

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func stringAttr(key, val string) *v1.KeyValue {
	return &v1.KeyValue{Key: key, Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: val}}}
}

func intAttr(key string, val int64) *v1.KeyValue {
	return &v1.KeyValue{Key: key, Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: val}}}
}

// A single event/span/resource can legitimately carry the same attribute key more than
// once (e.g. one attribute per loop iteration in the emitting instrumentation, such as
// Mimir's lookup_plan_predicate event logging one "matcher"/"selectivity" pair per
// predicate). When that key is mapped to a dedicated column, every occurrence must be
// preserved -- not just the last one.

func TestDuplicateEventAttributeKeyAccumulatesInDedicatedColumn(t *testing.T) {
	meta := backend.BlockMeta{DedicatedColumns: test.MakeDedicatedColumns()}
	traceID := common.ID{0x01}

	trace := &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{{
			Resource: &v1_resource.Resource{
				Attributes: []*v1.KeyValue{stringAttr("service.name", "service-a")},
			},
			ScopeSpans: []*v1_trace.ScopeSpans{{
				Spans: []*v1_trace.Span{{
					Name:   "span-a",
					SpanId: common.ID{0x01},
					Events: []*v1_trace.Span_Event{{
						Name: "lookup_plan_predicate",
						Attributes: []*v1.KeyValue{
							// "dedicated.event.1" is mapped to a dedicated column (see
							// pkg/util/test/req.go dedicatedColumnsEvent). Same key, three
							// occurrences within the SAME event, like repeated predicates.
							stringAttr("dedicated.event.1", "matcher-a"),
							stringAttr("dedicated.event.1", "matcher-b"),
							stringAttr("dedicated.event.1", "matcher-c"),
						},
					}},
				}},
			}},
		}},
	}

	out, connected := traceToParquet(&meta, traceID, trace, nil)
	require.True(t, connected)
	event := out.ResourceSpans[0].ScopeSpans[0].Spans[0].Events[0]

	require.Equal(t, []string{"matcher-a", "matcher-b", "matcher-c"}, event.DedicatedAttributes.String01)
	require.Empty(t, event.Attrs, "no attributes should have spilled into the generic list")

	// Round trip back to proto: all three values must come back as one attribute
	// with an array value, not just the last one.
	roundTripped := ParquetTraceToTempopbTrace(&meta, out)
	rtEvent := roundTripped.ResourceSpans[0].ScopeSpans[0].Spans[0].Events[0]
	require.Len(t, rtEvent.Attributes, 1)
	arr := rtEvent.Attributes[0].Value.GetArrayValue()
	require.NotNil(t, arr)
	require.Len(t, arr.Values, 3)
	require.Equal(t, "matcher-a", arr.Values[0].GetStringValue())
	require.Equal(t, "matcher-b", arr.Values[1].GetStringValue())
	require.Equal(t, "matcher-c", arr.Values[2].GetStringValue())
}

func TestDuplicateSpanAttributeKeyAccumulatesInDedicatedColumn(t *testing.T) {
	meta := backend.BlockMeta{DedicatedColumns: test.MakeDedicatedColumns()}
	traceID := common.ID{0x01}

	trace := &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{{
			Resource: &v1_resource.Resource{
				Attributes: []*v1.KeyValue{stringAttr("service.name", "service-a")},
			},
			ScopeSpans: []*v1_trace.ScopeSpans{{
				Spans: []*v1_trace.Span{{
					Name:   "span-with-repeated-key",
					SpanId: common.ID{0x01},
					Attributes: []*v1.KeyValue{
						// "dedicated.span.1" is mapped to a dedicated column.
						stringAttr("dedicated.span.1", "first-value"),
						stringAttr("dedicated.span.1", "second-value"),
					},
				}},
			}},
		}},
	}

	out, connected := traceToParquet(&meta, traceID, trace, nil)
	require.True(t, connected)
	span := out.ResourceSpans[0].ScopeSpans[0].Spans[0]

	require.Equal(t, []string{"first-value", "second-value"}, span.DedicatedAttributes.String01)
	require.Empty(t, span.Attrs)
}

// If a duplicate key's later occurrence doesn't match the dedicated column's configured
// type, the earlier, correctly-typed value(s) already written to the dedicated column
// must not be wiped out -- the mismatched occurrence falls back to the generic list on
// its own.
func TestDuplicateAttributeKeyMixedTypeFallsBackToGenericWithoutLosingDedicatedValue(t *testing.T) {
	meta := backend.BlockMeta{DedicatedColumns: test.MakeDedicatedColumns()}
	traceID := common.ID{0x01}

	trace := &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{{
			Resource: &v1_resource.Resource{
				Attributes: []*v1.KeyValue{stringAttr("service.name", "service-a")},
			},
			ScopeSpans: []*v1_trace.ScopeSpans{{
				Spans: []*v1_trace.Span{{
					Name:   "span-a",
					SpanId: common.ID{0x01},
					Events: []*v1_trace.Span_Event{{
						Name: "mixed-type-event",
						Attributes: []*v1.KeyValue{
							// "dedicated.event.1" is configured as a string column.
							stringAttr("dedicated.event.1", "matcher-a"),
							intAttr("dedicated.event.1", 42),
						},
					}},
				}},
			}},
		}},
	}

	out, connected := traceToParquet(&meta, traceID, trace, nil)
	require.True(t, connected)
	event := out.ResourceSpans[0].ScopeSpans[0].Spans[0].Events[0]

	require.Equal(t, []string{"matcher-a"}, event.DedicatedAttributes.String01,
		"first, correctly-typed occurrence must survive in the dedicated column")
	require.Len(t, event.Attrs, 1, "mismatched second occurrence must fall back to generic attrs")
	require.Equal(t, "dedicated.event.1", event.Attrs[0].Key)
	require.Equal(t, []int64{42}, event.Attrs[0].ValueInt)
}

// The OTel spec does not promise attributes are ordered or that duplicate keys (where
// they occur at all) are adjacent -- attribute collections are unordered by spec.
// Accumulation must not depend on duplicate occurrences of a key being contiguous in
// the input list.
func TestDuplicateAttributeKeyAccumulatesRegardlessOfContiguity(t *testing.T) {
	meta := backend.BlockMeta{DedicatedColumns: test.MakeDedicatedColumns()}
	traceID := common.ID{0x01}

	trace := &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{{
			Resource: &v1_resource.Resource{
				Attributes: []*v1.KeyValue{stringAttr("service.name", "service-a")},
			},
			ScopeSpans: []*v1_trace.ScopeSpans{{
				Spans: []*v1_trace.Span{{
					Name:   "span-a",
					SpanId: common.ID{0x01},
					Events: []*v1_trace.Span_Event{{
						Name: "lookup_plan_predicate",
						Attributes: []*v1.KeyValue{
							// "dedicated.event.1" occurrences are scattered, interleaved
							// with a different dedicated column ("dedicated.event.2")
							// and a plain, non-dedicated attribute.
							stringAttr("dedicated.event.1", "matcher-a"),
							stringAttr("dedicated.event.2", "other-value"),
							stringAttr("dedicated.event.1", "matcher-b"),
							stringAttr("plain.attr", "unrelated"),
							stringAttr("dedicated.event.1", "matcher-c"),
						},
					}},
				}},
			}},
		}},
	}

	out, connected := traceToParquet(&meta, traceID, trace, nil)
	require.True(t, connected)
	event := out.ResourceSpans[0].ScopeSpans[0].Spans[0].Events[0]

	require.Equal(t, []string{"matcher-a", "matcher-b", "matcher-c"}, event.DedicatedAttributes.String01,
		"scattered occurrences of the same key must still accumulate in encounter order")
	require.Equal(t, []string{"other-value"}, event.DedicatedAttributes.String02,
		"an unrelated dedicated column interleaved between occurrences must be unaffected")
	require.Len(t, event.Attrs, 1, "the non-dedicated attribute must land in generic attrs, untouched")
	require.Equal(t, "plain.attr", event.Attrs[0].Key)
}

// End-to-end: a TraceQL equality query for a value that only exists because it was
// accumulated from a duplicate key (i.e. not the first occurrence) must still find it.
func TestDuplicateEventAttributeKeyIsSearchable(t *testing.T) {
	meta := backend.BlockMeta{DedicatedColumns: test.MakeDedicatedColumns()}
	traceID := common.ID{0x01}

	trace := &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{{
			Resource: &v1_resource.Resource{
				Attributes: []*v1.KeyValue{stringAttr("service.name", "service-a")},
			},
			ScopeSpans: []*v1_trace.ScopeSpans{{
				Spans: []*v1_trace.Span{{
					Name:   "span-a",
					SpanId: common.ID{0x01},
					Events: []*v1_trace.Span_Event{{
						Name: "lookup_plan_predicate",
						Attributes: []*v1.KeyValue{
							stringAttr("dedicated.event.1", "matcher-a"),
							stringAttr("dedicated.event.1", "matcher-b"),
						},
					}},
				}},
			}},
		}},
	}

	out, connected := traceToParquet(&meta, traceID, trace, nil)
	require.True(t, connected)

	block := makeBackendBlockWithTracesWithDedicatedColumns(t, []*Trace{out}, meta.DedicatedColumns)

	ctx := context.Background()
	opts := common.DefaultSearchOptions()

	req, err := traceql.ExtractFetchSpansRequest(`{event.dedicated.event.1 = "matcher-b"}`)
	require.NoError(t, err)

	resp, err := block.Fetch(ctx, req, opts)
	require.NoError(t, err)
	defer resp.Results.Close()

	found := false
	for {
		ss, err := resp.Results.Next(ctx)
		require.NoError(t, err)
		if ss == nil {
			break
		}
		found = true
	}
	require.True(t, found, `"matcher-b" is the second occurrence of a duplicated key and must still be searchable`)
}
