package api

import (
	"context"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/api"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestMCP is a smoke test that starts up a tempo instance, writes a trace and queries it back via MCP.
// It also verifies that all expected tools are available.
func TestMCP(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: "config-mcp.yaml",
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		// Write a trace
		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		_, err := info.ConstructTraceFromEpoch()
		require.NoError(t, err)

		// Wait for the trace to be written to the WAL
		h.WaitTracesQueryable(t, 1)

		// now query it back with mcp
		queryFrontend := h.Services[util.ServiceQueryFrontend]
		mcpClient := createMCPClient(t, queryFrontend)

		tools := listTools(t, mcpClient)

		// confirm all tools are listed as read only and no open world
		for _, tool := range tools {
			require.NotNil(t, tool.Annotations.DestructiveHint, "tool %s doesn't specify destructive", tool.Name)
			require.False(t, *tool.Annotations.DestructiveHint, "tool %s is marked destructive", tool.Name)

			require.NotNil(t, tool.Annotations.OpenWorldHint, "tool %s doesn't specify open world", tool.Name)
			require.False(t, *tool.Annotations.OpenWorldHint, "tool %s is marked open world", tool.Name)

			require.NotNil(t, tool.Annotations.ReadOnlyHint, "tool %s doesn't specify read only", tool.Name)
			require.True(t, *tool.Annotations.ReadOnlyHint, "tool %s is marked write", tool.Name)
		}

		// Verify all expected tools are available
		expectedTools := []string{
			"traceql-search",
			"traceql-metrics-instant",
			"traceql-metrics-range",
			"get-trace",
			"get-attribute-names",
			"get-attribute-values",
			"docs-traceql",
		}

		actualTools := make([]string, len(tools))
		for i, tool := range tools {
			actualTools[i] = tool.Name
		}

		// sort both lists
		sort.Strings(actualTools)
		sort.Strings(expectedTools)
		require.Equal(t, expectedTools, actualTools)

		assertTraceOverMCP(t, mcpClient, info.HexID())
	})
}

func createMCPClient(t *testing.T, tempo *e2e.HTTPService) mcpclient.MCPClient {
	mcpClient, err := mcpclient.NewStreamableHttpClient("http://" + path.Join(tempo.Endpoint(3200), api.PathMCP))
	require.NoError(t, err)

	// Initialize the connection with required parameters
	initReq := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "tempo-e2e-test",
				Version: "1.0.0",
			},
		},
	}
	_, err = mcpClient.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("failed to initialize MCP client: %v", err)
	}

	return mcpClient
}

func listTools(t *testing.T, mcpClient mcpclient.MCPClient) []mcp.Tool {
	toolsResponse, err := mcpClient.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}
	return toolsResponse.Tools
}

func assertTraceOverMCP(t *testing.T, mcpClient mcpclient.MCPClient, traceID string) {
	resp, err := mcpClient.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get-trace",
			Arguments: map[string]interface{}{"trace_id": traceID},
		},
	})
	require.NoError(t, err)

	str := toolResult(t, resp)

	// the trace format is subject is to change so let's keep it simple.
	if !strings.Contains(str, traceID) {
		t.Fatalf("expected trace ID %s not found in response", traceID)
	}
}

func toolResult(t *testing.T, resp *mcp.CallToolResult) string {
	if resp.IsError {
		t.Fatalf("tool call failed: %v", resp.Content)
	}

	var result strings.Builder

	for _, content := range resp.Content {
		// Handle different content types
		switch c := content.(type) {
		case mcp.TextContent:
			result.WriteString(c.Text)
		default:
			t.Fatalf("unhandled content type: %T", c)
		}
	}

	return result.String()
}
