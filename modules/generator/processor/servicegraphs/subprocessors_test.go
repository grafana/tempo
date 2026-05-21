package servicegraphs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/generator/processor"
)

func TestSubprocessor_String(t *testing.T) {
	require.Equal(t, processor.ServiceGraphsRequestName, Request.String())
	require.Equal(t, processor.ServiceGraphsLatencyName, Latency.String())
	require.Equal(t, processor.ServiceGraphsConnectionInfoName, ConnectionInfo.String())
	require.Equal(t, "unsupported", Subprocessor(99).String())
}

func TestParseSubprocessor(t *testing.T) {
	for _, name := range []string{
		processor.ServiceGraphsRequestName,
		processor.ServiceGraphsLatencyName,
		processor.ServiceGraphsConnectionInfoName,
	} {
		require.True(t, ParseSubprocessor(name), "expected %s to be a recognized subprocessor", name)
	}

	require.True(t, ParseSubprocessor("Service-Graphs-Connection-Info"), "match should be case-insensitive")
	require.False(t, ParseSubprocessor("service-graphs"), "bare name is not a sub-name")
	require.False(t, ParseSubprocessor("span-metrics-count"), "span-metrics sub-name should not match")
	require.False(t, ParseSubprocessor(""), "empty string should not match")
}
