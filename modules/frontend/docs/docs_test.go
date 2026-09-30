package docs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDocsContent(t *testing.T) {
	for _, docType := range []string{DocsTypeBasic, DocsTypeAggregates, DocsTypeStructural, DocsTypeMetrics} {
		require.NotEmpty(t, GetDocsContent(docType), "doc type %s should not be empty", docType)
	}

	// unknown types fall back to basic
	require.Equal(t, GetDocsContent(DocsTypeBasic), GetDocsContent("does-not-exist"))
}

func TestGetConfigDocsContent(t *testing.T) {
	overview := GetConfigDocsContent(DocsTypeConfigOverview)
	require.NotEmpty(t, overview)

	reference := GetConfigDocsContent(DocsTypeConfigReference)
	require.NotEmpty(t, reference)
	// the generated reference always contains the top-level target option
	require.True(t, strings.Contains(reference, "target: all"), "config reference should contain the generated manifest")

	// unknown types fall back to the overview
	require.Equal(t, overview, GetConfigDocsContent("does-not-exist"))
}
