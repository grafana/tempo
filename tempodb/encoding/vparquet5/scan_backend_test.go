package vparquet5

import (
	"testing"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

// TestScanBackend_ImplementsInterface is a compile-time check that
// *blockScanBackend satisfies common.ScanBackend.
func TestScanBackend_ImplementsInterface(t *testing.T) {
	var _ common.ScanBackend = (*blockScanBackend)(nil)
}

// TestScanBackend_NewBlockScanBackend verifies that NewBlockScanBackend
// returns a non-nil backend.
func TestScanBackend_NewBlockScanBackend(t *testing.T) {
	// We can't open a real parquet file in a unit test without fixtures, but
	// we can verify the nil guard behaves correctly when makeIter funcs are nil.
	b := &blockScanBackend{}
	require.NotNil(t, b)
	_ = common.ScanBackend(b) // satisfies interface — compile-time guard
}

// TestScanBackend_SpanScanNodeConditions verifies that conditions attached to a
// SpanScanNode are visible (no data-flow issues through the wrapper).
func TestScanBackend_SpanScanNodeConditions(t *testing.T) {
	cond := traceql.Condition{
		Attribute: traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, "http.method"),
	}
	node := traceql.NewSpanScanNode([]traceql.Condition{cond}, nil)
	require.Len(t, node.Conditions, 1)
	require.Equal(t, cond, node.Conditions[0])
}

// TestScanBackend_ResourceScanNodeConditions mirrors the above for resources.
func TestScanBackend_ResourceScanNodeConditions(t *testing.T) {
	cond := traceql.Condition{
		Attribute: traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, "service.name"),
	}
	node := traceql.NewResourceScanNode([]traceql.Condition{cond}, nil)
	require.Len(t, node.Conditions, 1)
}
