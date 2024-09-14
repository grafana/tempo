package tempopb

import (
	"io"
	"os"
	"strings"
	"testing"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
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

func TestUnMarshalToJSONV1(t *testing.T) {
	file, _ := os.Open("trace.json")
	defer file.Close()
	trace := &Trace{}
	content, _ := io.ReadAll(file)
	err := UnmarshalFromJSONV1(content, trace)

	assert.NoError(t, err)
	assert.Equal(t, "my.library", trace.ResourceSpans[0].ScopeSpans[0].Scope.Name)
}
