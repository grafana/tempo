package tempopb

import (
	"io"
	"os"
	"strings"
	"testing"

	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalToJSONV1(t *testing.T) {
	trace := &Trace{
		ResourceSpans: make([]v1.ResourceSpans, 0),
	}
	jsonBytes, err := MarshalToJSONV1(trace)

	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(jsonBytes), "batches"))

	trace2 := &Trace{}
	err = UnmarshalFromJSONV1(jsonBytes, trace2)

	assert.NoError(t, err)
	assert.Equal(t, trace, trace2)
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

// TestMarshalUnmarshalEntityRefs guards the Resource.EntityRefs field (and its
// EntityRef message type), added when opentelemetry-proto picked up entity
// references: a trace carrying entity_refs must survive a binary
// Marshal/Unmarshal round trip unchanged.
func TestMarshalUnmarshalEntityRefs(t *testing.T) {
	trace := &Trace{
		ResourceSpans: []v1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []commonv1.KeyValue{
						{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "my-service"}}},
					},
					EntityRefs: []*commonv1.EntityRef{
						{
							SchemaUrl:       "https://opentelemetry.io/schemas/1.30.0",
							Type:            "service",
							IdKeys:          []string{"service.name"},
							DescriptionKeys: []string{"service.version"},
						},
					},
				},
			},
		},
	}

	data, err := trace.Marshal()
	require.NoError(t, err)

	trace2 := &Trace{}
	err = trace2.Unmarshal(data)
	require.NoError(t, err)

	assert.Equal(t, trace, trace2)
}
