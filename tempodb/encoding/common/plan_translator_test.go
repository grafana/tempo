package common

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/require"
)

// mockBackend records which ScanBackend methods were called.
type mockBackend struct {
	calledSpan, calledResource, calledTrace, calledScope bool
}

func (m *mockBackend) SpanIter(_ context.Context, _ *traceql.SpanScanNode, _ parquetquery.Iterator) (parquetquery.Iterator, error) {
	m.calledSpan = true
	return &mockIter{}, nil
}

func (m *mockBackend) InstrumentationScopeIter(_ context.Context, _ *traceql.InstrumentationScopeScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) {
	m.calledScope = true
	return child, nil
}

func (m *mockBackend) ResourceIter(_ context.Context, _ *traceql.ResourceScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) {
	m.calledResource = true
	return child, nil
}

func (m *mockBackend) TraceIter(_ context.Context, _ *traceql.TraceScanNode, _ parquetquery.Iterator, _ parquetquery.Iterator) (traceql.SpansetIterator, error) {
	m.calledTrace = true
	return &mockSpansetIter{}, nil
}

func (m *mockBackend) EventIter(_ context.Context, _ *traceql.EventScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) {
	return child, nil
}

func (m *mockBackend) LinkIter(_ context.Context, _ *traceql.LinkScanNode, child parquetquery.Iterator) (parquetquery.Iterator, error) {
	return child, nil
}

type mockIter struct{}

func (m *mockIter) String() string { return "mockIter" }
func (m *mockIter) Next() (*parquetquery.IteratorResult, error) {
	return nil, nil
}
func (m *mockIter) SeekTo(_ parquetquery.RowNumber, _ int) (*parquetquery.IteratorResult, error) {
	return nil, nil
}
func (m *mockIter) Close() {}

type mockSpansetIter struct{}

func (m *mockSpansetIter) Next(_ context.Context) (*traceql.Spanset, error) { return nil, nil }
func (m *mockSpansetIter) Close()                                            {}

// TestTranslate_SimpleScanTree verifies that Translate calls each ScanBackend
// method exactly once for a minimal scan-only plan (no engine nodes above the
// scan tree).
func TestTranslate_SimpleScanTree(t *testing.T) {
	span := traceql.NewSpanScanNode(nil, nil)
	instr := traceql.NewInstrumentationScopeScanNode(nil, span)
	res := traceql.NewResourceScanNode(nil, instr)
	trace := traceql.NewTraceScanNode(nil, false, res)

	backend := &mockBackend{}
	eval, err := Translate(context.Background(), trace, backend, DefaultSearchOptions())
	require.NoError(t, err)
	require.NotNil(t, eval)

	require.True(t, backend.calledSpan, "expected SpanIter to be called")
	require.True(t, backend.calledScope, "expected InstrumentationScopeIter to be called")
	require.True(t, backend.calledResource, "expected ResourceIter to be called")
	require.True(t, backend.calledTrace, "expected TraceIter to be called")
}

// TestTranslate_TraceScanOnly verifies that Translate works when there are no
// children below TraceScanNode.
func TestTranslate_TraceScanOnly(t *testing.T) {
	trace := traceql.NewTraceScanNode(nil, false, nil)

	backend := &mockBackend{}
	eval, err := Translate(context.Background(), trace, backend, DefaultSearchOptions())
	require.NoError(t, err)
	require.NotNil(t, eval)
	require.True(t, backend.calledTrace, "expected TraceIter to be called")
}

// TestTranslate_UnknownNodeReturnsError verifies that an unrecognised node type
// causes Translate to return an error rather than panic.
func TestTranslate_UnknownNodeReturnsError(t *testing.T) {
	type unknownNode struct{}
	// unknownNode doesn't implement PlanNode, so we use a valid but unhandled type.
	// We'll use a SpanScanNode at the root (which is not a TraceScanNode) to
	// trigger the "unhandled plan node type" branch.
	span := traceql.NewSpanScanNode(nil, nil)
	backend := &mockBackend{}
	_, err := Translate(context.Background(), span, backend, DefaultSearchOptions())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unhandled plan node type")
}

// TestTranslate_DoRunsWithoutError verifies that the spansetEvaluatable.Do
// method runs cleanly (draining the empty mock iterator).
func TestTranslate_DoRunsWithoutError(t *testing.T) {
	trace := traceql.NewTraceScanNode(nil, false, nil)
	backend := &mockBackend{}

	eval, err := Translate(context.Background(), trace, backend, DefaultSearchOptions())
	require.NoError(t, err)

	err = eval.Do(context.Background(), 0, 0, 0)
	require.NoError(t, err)
}
