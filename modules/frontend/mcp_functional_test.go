package frontend

import (
	"context"
	"flag"
	"strings"
	"testing"

	"github.com/go-kit/log"
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
