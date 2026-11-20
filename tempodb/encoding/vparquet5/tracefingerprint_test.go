package vparquet5

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	v1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/pkg/tracefingerprint"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
)

// TestTraceFingerprintSurvivesRoundTrip checks that a trace has the same
// fingerprint after it is written to a vParquet5 file and read back, including
// resources with several service.name attributes or non-string ones.
func TestTraceFingerprintSurvivesRoundTrip(t *testing.T) {
	meta := backend.BlockMeta{DedicatedColumns: test.MakeDedicatedColumns()}
	traceID := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}

	str := func(s string) *v1.AnyValue { return &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: s}} }
	arr := func(s string) *v1.AnyValue {
		return &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{str(s)}}}}
	}
	num := &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 7}}
	resources := [][]*v1.KeyValue{
		{{Key: LabelServiceName, Value: str("a")}, {Key: LabelServiceName, Value: str("b")}},
		{{Key: LabelServiceName, Value: str("checkout")}, {Key: LabelServiceName, Value: num}},
		{{Key: LabelServiceName, Value: num}, {Key: LabelServiceName, Value: str("payment")}},
		{{Key: LabelServiceName, Value: arr("cart")}},
	}
	withResources := func(resources [][]*v1.KeyValue) *tempopb.Trace {
		tr := ParquetTraceToTempopbTrace(&meta, fullyPopulatedTestTrace(traceID))
		parent := tr.ResourceSpans[0].ScopeSpans[0].Spans[0].SpanId
		for i, attrs := range resources {
			tr.ResourceSpans = append(tr.ResourceSpans, &v1_trace.ResourceSpans{
				Resource: &v1_resource.Resource{Attributes: attrs},
				ScopeSpans: []*v1_trace.ScopeSpans{{Spans: []*v1_trace.Span{{
					TraceId:      traceID,
					SpanId:       []byte{0xf0, byte(i), 0, 0, 0, 0, 0, 1},
					ParentSpanId: parent,
					Name:         fmt.Sprintf("duplicate-service-%d", i),
				}}}},
			})
		}
		return tr
	}
	tr := withResources(resources)

	var f tracefingerprint.Fingerprinter
	want := f.Fingerprint(tr)
	require.Equal(t, tracefingerprint.StatusComputed, want.Status)

	// An array is not a service name, so ["cart"] must count as no service.
	noArray := slices.Clone(resources)
	noArray[3] = nil
	require.Equal(t, want, f.Fingerprint(withResources(noArray)))

	row, _ := traceToParquet(&meta, traceID, tr, nil)
	tsc := TraceSchemaConfig{DedicatedColumns: meta.DedicatedColumns}
	read := readTraceTestFile(t, tsc, writeTraceTestFile(t, tsc, []*Trace{row}), 1)
	require.Equal(t, want, f.Fingerprint(ParquetTraceToTempopbTrace(&meta, read[0])))
}
