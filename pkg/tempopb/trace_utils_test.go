package tempopb

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	commonv1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalToJSONV1(t *testing.T) {
	trace := &Trace{
		ResourceSpans: make([]*v1.ResourceSpans, 0),
	}
	jsonBytes, err := MarshalToJSONV1(trace)

	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(jsonBytes), "batches"))

	trace2 := &Trace{}
	err = UnmarshalFromJSONV1(jsonBytes, trace2)

	assert.NoError(t, err)
	assert.Equal(t, trace, trace2)
}

// Keep the previous encoder as a compatibility oracle, including its nil and
// empty-slice behavior. Compare bytes, not just decoded JSON, to catch changes
// to field names, enum formatting, escaping, and integer representation.
func TestMarshalToJSONV1Compatibility(t *testing.T) {
	data, err := os.ReadFile("trace.json")
	require.NoError(t, err)
	fixture := &Trace{}
	require.NoError(t, UnmarshalFromJSONV1(data, fixture))
	fixture.ResourceSpans[0].ScopeSpans[0].Spans[0].Attributes = append(
		fixture.ResourceSpans[0].ScopeSpans[0].Spans[0].Attributes,
		&commonv1.KeyValue{Key: "resourceSpans", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "resourceSpans: \"batches\" \\ \n <>&"}}},
		&commonv1.KeyValue{Key: "large.integer", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: 1<<63 - 1}}},
		&commonv1.KeyValue{Key: "bytes", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_BytesValue{BytesValue: []byte{0, 1, 254, 255}}}},
	)

	for _, tc := range []struct {
		name  string
		trace *Trace
	}{
		{"nil", nil},
		{"nil batches", &Trace{}},
		{"empty batches", &Trace{ResourceSpans: []*v1.ResourceSpans{}}},
		{"empty batch", &Trace{ResourceSpans: []*v1.ResourceSpans{{}}}},
		{"nested fields", fixture},
		{"multiple batches", &Trace{ResourceSpans: []*v1.ResourceSpans{fixture.ResourceSpans[0], fixture.ResourceSpans[0]}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want, wantErr := new(jsonpb.Marshaler).MarshalToString(tc.trace)
			got, err := MarshalToJSONV1(tc.trace)
			if wantErr != nil {
				require.EqualError(t, err, wantErr.Error())
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, strings.Replace(want, `"resourceSpans":`, `"batches":`, 1), string(got))
		})
	}
}

func TestMarshalToJSONV1DoesNotMutateTrace(t *testing.T) {
	data, err := os.ReadFile("trace.json")
	require.NoError(t, err)
	trace := &Trace{}
	require.NoError(t, UnmarshalFromJSONV1(data, trace))
	original := proto.Clone(trace)
	encoded, err := MarshalToJSONV1(trace)
	require.NoError(t, err)
	require.True(t, proto.Equal(original, trace))

	// A subsequent encode must not reuse or overwrite previously returned bytes.
	first := append([]byte(nil), encoded...)
	trace.ResourceSpans[0].ScopeSpans[0].Spans[0].Name = "changed"
	_, err = MarshalToJSONV1(trace)
	require.NoError(t, err)
	require.Equal(t, first, encoded)
}

func TestUnMarshalToJSONV1(t *testing.T) {
	file, _ := os.Open("trace.json")
	defer file.Close()
	trace := &Trace{}
	content, _ := io.ReadAll(file)
	err := UnmarshalFromJSONV1(content, trace)

	assert.NoError(t, err)
	assert.Equal(t, "my.library", trace.ResourceSpans[0].ScopeSpans[0].Scope.Name)
}
