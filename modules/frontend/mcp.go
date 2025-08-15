package frontend

import (
	"context"
	"net/http"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/middleware"
	frontendDocs "github.com/grafana/tempo/modules/frontend/docs"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	docsTraceQLQueryURI      = "docs://traceql/query"
	docsTraceQLMetricsURI    = "docs://traceql/metrics"
	docsTraceQLBasicURI      = "docs://traceql/basic"
	docsTraceQLAggregatesURI = "docs://traceql/aggregates"
	docsTraceQLStructuralURI = "docs://traceql/structural"

	docsTraceQLQueryDescription      = "Documentation on TraceQL search. Best for retrieval of traces. This covers basic attributes all the way through aggregates, pipelining, structural queries, and more. Includes examples."
	docsTraceQLMetricsDescription    = "Documentation on TraceQL metrics. Best for aggregating traces into metrics to understand patterns and trends. This covers how to use TraceQL to generate metrics from tracing data."
	docsTraceQLBasicDescription      = "Basic TraceQL documentation covering intrinsics, operators, and attribute syntaxes. Includes overview of other doc types."
	docsTraceQLAggregatesDescription = "TraceQL aggregates documentation covering count, sum, and other aggregation functions."
	docsTraceQLStructuralDescription = "TraceQL structural queries documentation covering advanced query patterns and structural operations."

	docsTraceQLMimeType = "text/markdown"

	// Tool names
	toolTraceQLSearch         = "traceql-search"
	toolTraceQLMetricsInstant = "traceql-metrics-instant"
	toolTraceQLMetricsRange   = "traceql-metrics-range"
	toolGetTrace              = "get-trace"
	toolGetAttributeNames     = "get-attribute-names"
	toolGetAttributeValues    = "get-attribute-values"
	toolDocsTraceQL           = "docs-traceql"
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

	pathPrefix  string
	httpHandler http.Handler
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(frontend *QueryFrontend, pathPrefix string, logger log.Logger, authMiddleware middleware.Interface) *MCPServer {
	// Create the underlying MCP server
	mcpServer := server.NewMCPServer(
		"tempo",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithResourceCapabilities(false, false),

	// TODO: mcp servers also support the concept of prompts, but unsure how to use them or what role they play
	// server.WithPromptCapabilities(true),
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
	// Basic TraceQL docs
	traceQLBasic := mcp.NewResource(
		docsTraceQLBasicURI,
		"TraceQL Basic Docs",
		mcp.WithResourceDescription(docsTraceQLBasicDescription),
		mcp.WithMIMEType(docsTraceQLMimeType),
	)

	s.mcpServer.AddResource(traceQLBasic, func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		level.Info(s.logger).Log("msg", "traceql basic resource requested")

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      docsTraceQLBasicURI,
				MIMEType: docsTraceQLMimeType,
				Text:     frontendDocs.GetDocsContent(frontendDocs.DocsTypeBasic),
			},
		}, nil
	})

	// Aggregates TraceQL docs
	traceQLAggregates := mcp.NewResource(
		docsTraceQLAggregatesURI,
		"TraceQL Aggregates Docs",
		mcp.WithResourceDescription(docsTraceQLAggregatesDescription),
		mcp.WithMIMEType(docsTraceQLMimeType),
	)

	s.mcpServer.AddResource(traceQLAggregates, func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		level.Info(s.logger).Log("msg", "traceql aggregates resource requested")

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      docsTraceQLAggregatesURI,
				MIMEType: docsTraceQLMimeType,
				Text:     frontendDocs.GetDocsContent(frontendDocs.DocsTypeAggregates),
			},
		}, nil
	})

	// Structural TraceQL docs
	traceQLStructural := mcp.NewResource(
		docsTraceQLStructuralURI,
		"TraceQL Structural Docs",
		mcp.WithResourceDescription(docsTraceQLStructuralDescription),
		mcp.WithMIMEType(docsTraceQLMimeType),
	)

	s.mcpServer.AddResource(traceQLStructural, func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		level.Info(s.logger).Log("msg", "traceql structural resource requested")

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      docsTraceQLStructuralURI,
				MIMEType: docsTraceQLMimeType,
				Text:     frontendDocs.GetDocsContent(frontendDocs.DocsTypeStructural),
			},
		}, nil
	})

	// Metrics TraceQL docs
	traceQLMetrics := mcp.NewResource(
		docsTraceQLMetricsURI,
		"TraceQL Metrics Docs",
		mcp.WithResourceDescription(docsTraceQLMetricsDescription),
		mcp.WithMIMEType(docsTraceQLMimeType),
	)

	s.mcpServer.AddResource(traceQLMetrics, func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		level.Info(s.logger).Log("msg", "traceql metrics resource requested")

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      docsTraceQLMetricsURI,
				MIMEType: docsTraceQLMimeType,
				Text:     frontendDocs.GetDocsContent(frontendDocs.DocsTypeMetrics),
			},
		}, nil
	})

	// Keep the legacy query resource for backward compatibility
	traceQLQuery := mcp.NewResource(
		docsTraceQLQueryURI,
		"TraceQL Query Docs",
		mcp.WithResourceDescription(docsTraceQLQueryDescription),
		mcp.WithMIMEType(docsTraceQLMimeType),
	)

	s.mcpServer.AddResource(traceQLQuery, func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		level.Info(s.logger).Log("msg", "traceql query resource requested")

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      docsTraceQLQueryURI,
				MIMEType: docsTraceQLMimeType,
				Text:     frontendDocs.GetDocsContent(frontendDocs.DocsTypeBasic),
			},
		}, nil
	})
}

// setupTools registers MCP tools for trace operations
func (s *MCPServer) setupTools() {
	// api tools
	searchTool := newReadOnlyTool(toolTraceQLSearch,
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

	instantQueryTool := newReadOnlyTool(toolTraceQLMetricsInstant,
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
		mcp.WithDestructiveHintAnnotation(false),
	)
	s.mcpServer.AddTool(instantQueryTool, s.handleInstantQuery)

	// TODO: should we even expose this? the LLM would be better at using the instant query tool and giving accurate answers.
	rangeQueryTool := newReadOnlyTool(toolTraceQLMetricsRange,
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
		mcp.WithDestructiveHintAnnotation(false),
	)
	s.mcpServer.AddTool(rangeQueryTool, s.handleRangeQuery)

	traceTool := newReadOnlyTool(toolGetTrace,
		mcp.WithDescription("Retrieve a specific trace by ID"),
		mcp.WithString("trace_id",
			mcp.Required(),
			mcp.Description("Trace ID to retrieve"),
		),
		mcp.WithDestructiveHintAnnotation(false),
	)
	s.mcpServer.AddTool(traceTool, s.handleGetTrace)

	attributeNamesTool := newReadOnlyTool(toolGetAttributeNames,
		mcp.WithDescription("Get a list of available attribute names that can be used in TraceQL queries. This is useful for finding the names of attributes that can be used in a query."),
		mcp.WithString("scope",
			mcp.Description("Optional scope to filter attributes by (span, resource, event, link, instrumentation). If not provided, returns all attributes."),
		),
		mcp.WithDestructiveHintAnnotation(false),
	)
	s.mcpServer.AddTool(attributeNamesTool, s.handleGetAttributeNames)

	attributeValuesTool := newReadOnlyTool(toolGetAttributeValues,
		mcp.WithDescription("Get a list of values for a fully scoped attribute name. This is useful for finding the values of a specific attribute. i.e. you can find all the services in the data by asking for resource.service.name"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The attribute name to get values for (e.g. 'span.http.method', 'resource.service.name')"),
		),
		mcp.WithString("filter-query",
			mcp.Description("Filter query to apply to the attribute values. It can only have one spanset and only &&'ed conditions like { <cond> && <cond> && ... }.This is useful for filtering the values to a specific set of values. i.e. you can find all endpoints for a given service by asking for span.http.endpoint and filtering resource.service.name."),
		),
		mcp.WithDestructiveHintAnnotation(false),
	)

	s.mcpServer.AddTool(attributeValuesTool, s.handleGetAttributeValues)

	// docs tools - these are defined as tools as well as resources b/c claude code never asks for resources but it will nicely
	// request the content from these docs tools.
	traceQLDocs := newReadOnlyTool(toolDocsTraceQL,
		mcp.WithDescription(docsTraceQLQueryDescription),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The type of TraceQL documentation to retrieve"),
			mcp.Enum(frontendDocs.DocsTypeBasic, frontendDocs.DocsTypeAggregates, frontendDocs.DocsTypeStructural, frontendDocs.DocsTypeMetrics),
		),
		mcp.WithDestructiveHintAnnotation(false),
	)
	s.mcpServer.AddTool(traceQLDocs, s.handleTraceQLDocs)
}

func newReadOnlyTool(name string, opts ...mcp.ToolOption) mcp.Tool {
	standardReadOnlyOpts := []mcp.ToolOption{
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	}

	opts = append(opts, standardReadOnlyOpts...)

	return mcp.NewTool(name, opts...)
}
