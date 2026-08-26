package frontend

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model/tracediff"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"

	frontendDocs "github.com/grafana/tempo/modules/frontend/docs"
)

// newTestMCPClient builds a query frontend with the MCP server enabled and returns an
// in-process MCP client connected to it.
func newTestMCPClient(t *testing.T) *mcpclient.Client {
	t.Helper()

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("query-frontend", fs)
	require.NoError(t, fs.Parse([]string{"-query-frontend.mcp-server.enabled=true"}))

	qf, err := New(*cfg, &mockRoundTripper{}, nil, nil, nil, "", fakeHTTPAuthMiddleware, nil, log.NewNopLogger(), nil)
	require.NoError(t, err)

	mcpServer, ok := qf.MCPHandler.(*MCPServer)
	require.True(t, ok, "expected MCP handler to be enabled")

	return newInProcessMCPClient(t, mcpServer)
}

func newInProcessMCPClient(t *testing.T, mcpServer *MCPServer) *mcpclient.Client {
	t.Helper()

	client, err := mcpclient.NewInProcessClient(mcpServer.mcpServer)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	_, err = client.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "tempo-unit-test", Version: "1.0.0"},
		},
	})
	require.NoError(t, err)

	return client
}

func callDocsConfig(t *testing.T, client *mcpclient.Client, name string) string {
	t.Helper()

	resp, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolDocsConfig,
			Arguments: map[string]any{"name": name},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, "docs-config(%q) returned an error result", name)

	var b strings.Builder
	for _, c := range resp.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// TestMCPConfigDocsTool exercises the docs-config tool over the real MCP protocol.
func TestMCPConfigDocsTool(t *testing.T) {
	client := newTestMCPClient(t)

	// the tool is registered and advertised
	toolsResp, err := client.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)

	var configTool *mcp.Tool
	for i := range toolsResp.Tools {
		if toolsResp.Tools[i].Name == toolDocsConfig {
			configTool = &toolsResp.Tools[i]
		}
	}
	require.NotNil(t, configTool, "docs-config tool should be listed")
	require.NotNil(t, configTool.Annotations.ReadOnlyHint)
	require.True(t, *configTool.Annotations.ReadOnlyHint, "docs-config should be read-only")
	require.NotNil(t, configTool.Annotations.DestructiveHint)
	require.False(t, *configTool.Annotations.DestructiveHint)

	// reference returns the generated manifest
	reference := callDocsConfig(t, client, frontendDocs.DocsTypeConfigReference)
	require.Contains(t, reference, "target: all")
	require.Contains(t, reference, "query_frontend:")

	// overview returns the hand-curated map
	overview := callDocsConfig(t, client, frontendDocs.DocsTypeConfigOverview)
	require.Contains(t, overview, "Tempo Configuration Overview")

	// unknown doc types fall back to the overview instead of erroring
	require.Equal(t, overview, callDocsConfig(t, client, "does-not-exist"))
}

func TestMCPTraceDiffTool(t *testing.T) {
	var receivedRequest *http.Request
	var receivedBody api.TraceDiffRequest
	handlerCalls := 0
	frontend := &QueryFrontend{
		TraceDiffHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalls++
			receivedRequest = r
			require.NoError(t, json.NewDecoder(r.Body).Decode(&receivedBody))
			_, err := w.Write([]byte(`{"version":"trace-summary-v0-composed"}`))
			require.NoError(t, err)
		}),
	}
	mcpServer := NewMCPServer(frontend, "", log.NewNopLogger(), fakeHTTPAuthMiddleware, 0)
	client := newInProcessMCPClient(t, mcpServer)

	toolsResp, err := client.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)

	var traceDiffTool *mcp.Tool
	for i := range toolsResp.Tools {
		if toolsResp.Tools[i].Name == toolTraceDiff {
			traceDiffTool = &toolsResp.Tools[i]
		}
	}
	require.NotNil(t, traceDiffTool, "trace-diff tool should be listed")
	require.NotNil(t, traceDiffTool.Annotations.ReadOnlyHint)
	require.True(t, *traceDiffTool.Annotations.ReadOnlyHint)
	require.NotNil(t, traceDiffTool.Annotations.DestructiveHint)
	require.False(t, *traceDiffTool.Annotations.DestructiveHint)
	require.NotNil(t, traceDiffTool.Annotations.OpenWorldHint)
	require.False(t, *traceDiffTool.Annotations.OpenWorldHint)
	require.ElementsMatch(t, []string{"base_trace_id", "compare_trace_id"}, traceDiffTool.InputSchema.Required)

	formatSchema, ok := traceDiffTool.InputSchema.Properties["format"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, tracediff.VersionTraceSummaryV0Composed, formatSchema["default"])
	require.Equal(t, float64(1), formatSchema["minLength"])
	enumJSON, err := json.Marshal(formatSchema["enum"])
	require.NoError(t, err)
	require.JSONEq(t, `[
		"trace-summary-v0-composed",
		"trace-summary-v0-native",
		"trace-patch-v0"
	]`, string(enumJSON))

	resp, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: toolTraceDiff,
			Arguments: map[string]any{
				"base_trace_id":    "12345678abcdef90",
				"compare_trace_id": "abcdef9012345678",
			},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.NotNil(t, receivedRequest)
	require.Equal(t, http.MethodPost, receivedRequest.Method)
	require.Equal(t, api.PathTraceDiffV2, receivedRequest.URL.Path)
	require.Equal(t, api.HeaderAcceptJSON, receivedRequest.Header.Get(api.HeaderContentType))
	require.Equal(t, api.HeaderAcceptLLM, receivedRequest.Header.Get(api.HeaderAccept))
	require.Equal(t, "12345678abcdef90", receivedBody.Base.TraceID)
	require.Equal(t, "abcdef9012345678", receivedBody.Compare.TraceID)
	require.Equal(t, tracediff.VersionTraceSummaryV0Composed, receivedBody.Format)
	require.Equal(t, map[string]any{
		"type":     MetaTypeTraceDiff,
		"encoding": "json",
		"version":  "2",
	}, resp.Meta.AdditionalFields)
	require.Len(t, resp.Content, 1)
	textContent, ok := resp.Content[0].(mcp.TextContent)
	require.True(t, ok)
	require.JSONEq(t, `{"version":"trace-summary-v0-composed"}`, textContent.Text)

	emptyFormatResp, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: toolTraceDiff,
			Arguments: map[string]any{
				"base_trace_id":    "12345678abcdef90",
				"compare_trace_id": "abcdef9012345678",
				"format":           "",
			},
		},
	})
	require.NoError(t, err)
	require.True(t, emptyFormatResp.IsError)
	require.Equal(t, 1, handlerCalls)
}

// TestMCPConfigDocsResources exercises the docs://config resources over the real MCP protocol.
func TestMCPConfigDocsResources(t *testing.T) {
	client := newTestMCPClient(t)

	resourcesResp, err := client.ListResources(context.Background(), mcp.ListResourcesRequest{})
	require.NoError(t, err)

	got := map[string]string{}
	for _, r := range resourcesResp.Resources {
		got[r.URI] = r.MIMEType
	}
	require.Contains(t, got, docsConfigOverviewURI)
	require.Contains(t, got, docsConfigReferenceURI)
	require.Equal(t, docsTraceQLMimeType, got[docsConfigReferenceURI])

	// read the reference resource
	readResp, err := client.ReadResource(context.Background(), mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: docsConfigReferenceURI},
	})
	require.NoError(t, err)
	require.Len(t, readResp.Contents, 1)

	textContents, ok := readResp.Contents[0].(mcp.TextResourceContents)
	require.True(t, ok)
	require.Equal(t, docsTraceQLMimeType, textContents.MIMEType)
	require.Contains(t, textContents.Text, "target: all")
}
