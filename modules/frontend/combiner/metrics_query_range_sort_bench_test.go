package combiner

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/v3/pkg/api"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	v1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/require"
)

func TestSortResponseAnyValueOrder(t *testing.T) {
	values := []*v1.AnyValue{
		nil,
		{},
		{Value: &v1.AnyValue_StringValue{StringValue: ""}},
		{Value: &v1.AnyValue_StringValue{StringValue: "a"}},
		{Value: &v1.AnyValue_StringValue{StringValue: "a "}},
		{Value: &v1.AnyValue_StringValue{StringValue: "a!"}},
		{Value: &v1.AnyValue_StringValue{StringValue: "a#"}},
		{Value: &v1.AnyValue_StringValue{StringValue: "alpha"}},
		{Value: &v1.AnyValue_StringValue{StringValue: "alpha"}},
		{Value: &v1.AnyValue_StringValue{StringValue: "zeta"}},
		{Value: &v1.AnyValue_StringValue{StringValue: "quote\""}},
		{Value: &v1.AnyValue_StringValue{StringValue: "line\nfeed"}},
		{Value: &v1.AnyValue_StringValue{StringValue: "binary\x00"}},
		{Value: &v1.AnyValue_StringValue{StringValue: "unicode-é"}},
		{Value: &v1.AnyValue_IntValue{IntValue: -10}},
		{Value: &v1.AnyValue_IntValue{IntValue: 2}},
		{Value: &v1.AnyValue_BoolValue{BoolValue: false}},
		{Value: &v1.AnyValue_BoolValue{BoolValue: true}},
		{Value: &v1.AnyValue_BytesValue{BytesValue: []byte{0, '\n', 'x'}}},
		{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{{Value: &v1.AnyValue_StringValue{StringValue: "nested"}}}}}},
	}

	for i, left := range values {
		for j, right := range values {
			series := []*tempopb.TimeSeries{
				{Labels: []v1.KeyValue{{Key: "label", Value: left}}},
				{Labels: []v1.KeyValue{{Key: "label", Value: right}}},
			}
			want := append([]*tempopb.TimeSeries(nil), series...)
			sort.SliceStable(want, func(i, j int) bool {
				left := want[i].Labels[0].Value
				right := want[j].Labels[0].Value
				if _, ok := left.GetValue().(*v1.AnyValue_StringValue); ok {
					if _, ok := right.GetValue().(*v1.AnyValue_StringValue); ok {
						return left.GetStringValue() < right.GetStringValue()
					}
				}
				return left.String() < right.String()
			})

			sortResponse(&tempopb.QueryRangeResponse{Series: series})
			for k := range series {
				require.Same(t, want[k], series[k], "values %d and %d, position %d", i, j, k)
			}
		}
	}
}

func TestCompareAnyValuesMatchesStringOrderForEveryByte(t *testing.T) {
	values := make([]*v1.AnyValue, 256)
	for i := range values {
		values[i] = &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: string([]byte{byte(i)})}}
	}
	for i, a := range values {
		for j, b := range values {
			want := cmp.Compare(a.GetStringValue(), b.GetStringValue())
			got := compareAnyValues(a, b)
			require.Equal(t, want, got, "bytes %d and %d", i, j)
		}
	}
}

func TestSortResponseEqualStringLabels(t *testing.T) {
	newSeries := func(instance string) *tempopb.TimeSeries {
		return &tempopb.TimeSeries{Labels: []v1.KeyValue{
			{Key: "service", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "checkout"}}},
			{Key: "instance", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: instance}}},
		}}
	}
	first := newSeries("a")
	second := newSeries("a ")
	duplicate := newSeries("a")
	response := &tempopb.QueryRangeResponse{Series: []*tempopb.TimeSeries{second, first, duplicate}}

	sortResponse(response)

	require.Same(t, first, response.Series[0])
	require.Same(t, duplicate, response.Series[1])
	require.Same(t, second, response.Series[2])
}

func BenchmarkQueryRangeHTTPFinalWithStringLabels(b *testing.B) {
	for _, scenario := range []struct {
		name        string
		seriesCount int
		escaped     bool
		sharedLabel bool
	}{
		{name: "plain/16", seriesCount: 16, sharedLabel: true},
		{name: "plain/512", seriesCount: 512, sharedLabel: true},
		{name: "escaped/16", seriesCount: 16, escaped: true, sharedLabel: true},
		{name: "escaped/512", seriesCount: 512, escaped: true, sharedLabel: true},
		{name: "plain-unique/16", seriesCount: 16},
		{name: "plain-unique/512", seriesCount: 512},
		{name: "escaped-unique/16", seriesCount: 16, escaped: true},
		{name: "escaped-unique/512", seriesCount: 512, escaped: true},
	} {
		b.Run(scenario.name, func(b *testing.B) {
			benchmarkQueryRangeHTTPFinalWithStringLabels(b, scenario.seriesCount, scenario.escaped, scenario.sharedLabel)
		})
	}
}

func benchmarkQueryRangeHTTPFinalWithStringLabels(b *testing.B, seriesCount int, escaped, sharedLabel bool) {
	partial := &tempopb.QueryRangeResponse{Series: make([]*tempopb.TimeSeries, seriesCount)}
	for i := range partial.Series {
		instance := fmt.Sprintf("instance-%04d", seriesCount-i)
		if escaped {
			instance += "\n"
		}
		labels := []v1.KeyValue{{Key: "instance", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: instance}}}}
		if sharedLabel {
			labels = append([]v1.KeyValue{{Key: "service", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "checkout"}}}}, labels...)
		}
		partial.Series[i] = &tempopb.TimeSeries{
			Labels:  labels,
			Samples: []tempopb.Sample{{TimestampMs: 1000, Value: 1}},
		}
	}
	payload, err := proto.Marshal(partial)
	require.NoError(b, err)
	req := &tempopb.QueryRangeRequest{
		Query: "{} | rate()",
		Start: uint64(time.Second),
		End:   uint64(2 * time.Second),
		Step:  uint64(time.Second),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		c, err := NewQueryRange(req, 0)
		if err != nil {
			b.Fatal(err)
		}
		err = c.AddResponse(&testPipelineResponse{r: &http.Response{
			Body:          io.NopCloser(bytes.NewReader(payload)),
			StatusCode:    http.StatusOK,
			ContentLength: int64(len(payload)),
			Header:        http.Header{api.HeaderContentType: {string(api.HeaderAcceptProtobuf)}},
		}})
		if err != nil {
			b.Fatal(err)
		}
		resp, err := c.HTTPFinal()
		if err != nil {
			b.Fatal(err)
		}
		_ = resp.Body.Close()
	}
}
