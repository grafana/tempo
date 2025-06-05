package frontend

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/middleware"
	"github.com/grafana/tempo/docs"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	docsCutoff = "<!-- mcp-cutoff -->" // Marker to indicate where to cut off the documentation content

	docsTraceQLQueryURI   = "docs://traceql/query"
	docsTraceQLMetricsURI = "docs://traceql/metrics"

	docsTraceQLQueryDescription   = "Documentation on TraceQL search. Best for retrieval of traces. This covers basic attributes all the way through aggregates, pipelining, structural queries, and more. Includes examples."
	docsTraceQLMetricsDescription = "Documentation on TraceQL metrics. Best for aggregating traces into metrics to understand patterns and trends. This covers how to use TraceQL to generate metrics from tracing data."

	docsTraceQLMimeType = "text/markdown"
)

// fakeHTTPAuthMiddleware is a middleware that does nothing, used when multitenancy is disabled
var fakeHTTPAuthMiddleware = middleware.Func(func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
})

// MCPServer wraps the mcp-go server with Tempo-specific functionality
type MCPServer struct {
	logger   log.Logger
	frontend *QueryFrontend // Assuming Frontend is defined elsewhere in your code

	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer

	pathPrefix     string
	authMiddleware middleware.Interface
	httpHandler    http.Handler
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(frontend *QueryFrontend, pathPrefix string, logger log.Logger, multitenancyEnabled bool, authMiddleware middleware.Interface) *MCPServer {
	// Create the underlying MCP server
	mcpServer := server.NewMCPServer(
		"tempo",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithResourceCapabilities(false, false),

	// TODO: mcp servers also support the concept of prompts, but unsure how to use them or what role they play
	//server.WithPromptCapabilities(true),
	)

	httpServer := server.NewStreamableHTTPServer(mcpServer)

	s := &MCPServer{
		logger:     logger,
		frontend:   frontend,
		mcpServer:  mcpServer,
		httpServer: httpServer,
		pathPrefix: pathPrefix,
	}

	// Set up auth middleware
	s.httpHandler = authMiddleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.httpServer.ServeHTTP(w, r)
	}))

	// Register tools and resources
	s.setupTools()
	s.setupResources()

	return s
}

// ServeHTTP implements http.Handler to handle MCP requests over HTTP
func (s *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.httpHandler.ServeHTTP(w, r)
}

// setupResources registers MCP resources for TraceQL documentation
func (s *MCPServer) setupResources() {
	traceQLQuery := mcp.NewResource(
		docsTraceQLQueryURI,
		"TraceQL Query Docs",
		mcp.WithResourceDescription(docsTraceQLQueryDescription),
		mcp.WithMIMEType(docsTraceQLMimeType),
	)

	s.mcpServer.AddResource(traceQLQuery, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		level.Info(s.logger).Log("msg", "traceql query resource requested")

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      docsTraceQLQueryURI,
				MIMEType: docsTraceQLMimeType,
				Text:     trimDocs(docs.TraceQLMain),
			},
		}, nil
	})

	traceQLMetrics := mcp.NewResource(
		docsTraceQLMetricsURI,
		"TraceQL Metrics Docs",
		mcp.WithResourceDescription(docsTraceQLMetricsDescription),
		mcp.WithMIMEType(docsTraceQLMimeType),
	)

	s.mcpServer.AddResource(traceQLMetrics, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		level.Info(s.logger).Log("msg", "traceql metrics resource requested")

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      docsTraceQLMetricsURI,
				MIMEType: docsTraceQLMimeType,
				Text:     trimDocs(docs.TraceQLMetrics),
			},
		}, nil
	})
}

// setupTools registers MCP tools for trace operations
func (s *MCPServer) setupTools() {
	// api tools
	searchTool := mcp.NewTool("traceql-search",
		mcp.WithDescription("Search for traces using TraceQL queries"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("TraceQL query string"),
		),
		mcp.WithString("start",
			mcp.Description("Start time for the search (RFC3339 format). If not provided will search the past 1 hour. If provided, must be before end."),
		),
		mcp.WithString("end",
			mcp.Description("End time for the search (RFC3339 format). If not provided will search the past 1 hour. If provided, must be after start."),
		),
	)
	s.mcpServer.AddTool(searchTool, s.handleSearch)

	instantQueryTool := mcp.NewTool("traceql-metrics-instant",
		mcp.WithDescription("Retrieve a single metric value given a TraceQL metrics query. The value is at the current instant or end. Most metrics questions can be answered with instant values."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("TraceQL query string."),
		),
		mcp.WithString("start",
			mcp.Description("Start time for the search (RFC3339 format). If not provided will search the past 1 hour. If provided, must be before end."),
		),
		mcp.WithString("end",
			mcp.Description("End time for the search (RFC3339 format). If not provided will search the past 1 hour. If provided, must be after start."),
		),
	)
	s.mcpServer.AddTool(instantQueryTool, s.handleInstantQuery)

	// TODO: should we even expose this? the LLM would be better at using the instant query tool and giving accurate answers.
	rangeQueryTool := mcp.NewTool("traceql-metrics-range",
		mcp.WithDescription("Retrieve a metric series given a TraceQL metrics query. The series ranges from start to end."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("TraceQL metrics query string."),
		),
		mcp.WithString("start",
			mcp.Description("Start time for the search (RFC3339 format). If not provided will search the past 1 hour. If provided, must be before end."),
		),
		mcp.WithString("end",
			mcp.Description("End time for the search (RFC3339 format). If not provided will search the past 1 hour. If provided, must be after start."),
		),
	)
	s.mcpServer.AddTool(rangeQueryTool, s.handleRangeQuery)

	traceTool := mcp.NewTool("get-trace",
		mcp.WithDescription("Retrieve a specific trace by ID"),
		mcp.WithString("trace_id",
			mcp.Required(),
			mcp.Description("Trace ID to retrieve"),
		),
	)
	s.mcpServer.AddTool(traceTool, s.handleGetTrace)

	attributeNamesTool := mcp.NewTool("get-attribute-names",
		mcp.WithDescription("Get a list of available attribute names that can be used in TraceQL queries. This is useful for finding the names of attributes that can be used in a query."),
		mcp.WithString("scope",
			mcp.Description("Optional scope to filter attributes by (span, resource, event, link, instrumentation). If not provided, returns all attributes."),
		),
	)
	s.mcpServer.AddTool(attributeNamesTool, s.handleGetAttributeNames)

	attributeValuesTool := mcp.NewTool("get-attribute-values",
		mcp.WithDescription("Get a list of values for a fully scoped attribute name. This is useful for finding the values of a specific attribute. i.e. you can find all the services in the data by asking for resource.service.name"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The attribute name to get values for (e.g. 'span.http.method', 'resource.service.name')"),
		),
		mcp.WithString("filter-query",
			mcp.Description("Filter query to apply to the attribute values. It can only have one spanset and only &&'ed conditions like { <cond> && <cond> && ... }.This is useful for filtering the values to a specific set of values. i.e. you can find all endpoints for a given service by asking for span.http.endpoint and filtering resource.service.name."),
		),
	)
	s.mcpServer.AddTool(attributeValuesTool, s.handleGetAttributeValues)

	// docs tools - these are defined as tools as well as resources b/c claude code never asks for resources but it will nicely
	// request the content from these docs tools.
	traceQLAdvanced := mcp.NewTool("docs-traceql-query",
		mcp.WithDescription(docsTraceQLQueryDescription),
	)
	s.mcpServer.AddTool(traceQLAdvanced, s.handleTraceQLQuery)

	traceQLMetrics := mcp.NewTool("docs-traceql-metrics",
		mcp.WithDescription(docsTraceQLMetricsDescription),
	)
	s.mcpServer.AddTool(traceQLMetrics, s.handleTraceQLMetrics)
}

// trimDocs trims the documentation content at the cutoff marker. this allows us
// to load docs that are rendered by hugo but cut things off like their metadata
func trimDocs(content string) string {
	// Trim the content at the cutoff marker
	if idx := strings.Index(content, docsCutoff); idx != -1 {
		return content[:idx]
	}
	return content
}
