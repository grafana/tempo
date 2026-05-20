package servicegraphs

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/generator/processor"
)

func TestSubprocessor_String(t *testing.T) {
	assert.Equal(t, processor.ServiceGraphsRequestName, Request.String())
	assert.Equal(t, processor.ServiceGraphsLatencyName, Latency.String())
	assert.Equal(t, processor.ServiceGraphsConnectionInfoName, ConnectionInfo.String())
	assert.Equal(t, "unsupported", Subprocessor(99).String())
}

func TestParseSubprocessor(t *testing.T) {
	for _, name := range []string{
		processor.ServiceGraphsRequestName,
		processor.ServiceGraphsLatencyName,
		processor.ServiceGraphsConnectionInfoName,
	} {
		assert.True(t, ParseSubprocessor(name), "expected %s to be a recognized subprocessor", name)
	}

	assert.True(t, ParseSubprocessor("Service-Graphs-Connection-Info"), "match should be case-insensitive")
	assert.False(t, ParseSubprocessor("service-graphs"), "bare name is not a sub-name")
	assert.False(t, ParseSubprocessor("span-metrics-count"), "span-metrics sub-name should not match")
	assert.False(t, ParseSubprocessor(""), "empty string should not match")
}
