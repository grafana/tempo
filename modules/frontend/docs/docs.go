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

const (
	DocsTypeConfigOverview  = "config-overview"  // hand-curated map of the configuration tree
	DocsTypeConfigReference = "config-reference" // generated complete reference of all options and defaults
)

//go:embed config-overview.md
var docsConfigOverview string

//go:embed config-reference.md
var docsConfigReference string

// GetConfigDocsContent returns Tempo configuration documentation based on the doc type.
// Unknown types fall back to the orientation overview.
func GetConfigDocsContent(docType string) string {
	switch docType {
	case DocsTypeConfigOverview:
		return docsConfigOverview
	case DocsTypeConfigReference:
		return docsConfigReference
	default:
		return docsConfigOverview // fallback to the overview
	}
}
