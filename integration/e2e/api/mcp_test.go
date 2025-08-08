package api

import (
	"context"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestMCP is a smoke test that starts up a tempo instance, writes a trace and queries it back via MCP.
// It also verifies that all expected tools are available.
func TestMCP(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e_mcp")
	require.NoError(t, err)
	defer s.Close()

	minio := e2edb.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(jaegerClient))

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	// now query it back with mcp
	mcpClient := createMCPClient(t, tempo)

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

	trace := traceOverMCP(t, mcpClient, info.HexID())
	util.AssertEqualTrace(t, expected, trace)
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

func traceOverMCP(t *testing.T, mcpClient mcpclient.MCPClient, traceID string) *tempopb.Trace {
	resp, err := mcpClient.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get-trace",
			Arguments: map[string]interface{}{"trace_id": traceID},
		},
	})
	require.NoError(t, err)

	str := toolResult(t, resp)

	trace := &tempopb.TraceByIDResponse{}
	err = jsonpb.UnmarshalString(str, trace)
	require.NoError(t, err)

	return trace.Trace
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
