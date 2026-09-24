package api

// Query operation names used as metric labels across the query path
// (query-frontend and querier), kept here to avoid label drift between components.
const (
	OpTraceByID = "traces"
	OpSearch    = "search"
	OpMetadata  = "metadata"
	OpMetrics   = "metrics"

	// Finer-grained metadata ops recorded by the querier; roll up to OpMetadata.
	OpSearchTags      = "search-tags"
	OpSearchTagValues = "search-tag-values"
)
