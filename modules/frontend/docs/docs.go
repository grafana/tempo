package docs

import (
	_ "embed"
)

const (
	DocsTypeBasic      = "basic"      // all intrinsics, operators and attributes syntaxes. overview of other doc types
	DocsTypeAggregates = "aggregates" // count, sum, etc.
	DocsTypeStructural = "structural"
	DocsTypeMetrics    = "metrics"
)

//go:embed basic.md
var docsBasic string

//go:embed aggregates.md
var docsAggregates string

//go:embed structural.md
var docsStructural string

//go:embed metrics.md
var docsMetrics string

// GetDocsContent returns the appropriate documentation content based on the doc type
func GetDocsContent(docType string) string {
	switch docType {
	case DocsTypeBasic:
		return docsBasic
	case DocsTypeAggregates:
		return docsAggregates
	case DocsTypeStructural:
		return docsStructural
	case DocsTypeMetrics:
		return docsMetrics
	default:
		return docsBasic // fallback to basic docs
	}
}
