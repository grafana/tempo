# Plan-Based Search Traces Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extend the plan-based execution path to handle search traces (`SearchBlock`), implementing `ProjectNode` late materialization and `SearchEvaluatable`.

**Architecture:** `ProjectNode` becomes a late materialization operator with two children: a driving plan (filtered spans) and a fetch scan tree (metadata columns). The translator builds both into iterators once — the fetch side uses `parquetquery.Iterator` with `SeekTo` for efficient row-number-based column reads. A new `SearchEvaluatable` interface wraps the top-level iterator with a `MetadataCombiner` to produce `SearchResponse`.

**Tech Stack:** Go, `pkg/parquetquery`, `pkg/traceql`, `tempodb/encoding/common`, `tempodb/encoding/vparquet5`

**Design doc:** `docs/plans/2026-03-05-plan-based-search-design.md`

---

## Phase 1 — Export required helpers

### Task 1: Export `AsTraceSearchMetadata` and `AddSpanset`

The `searchEvaluatable` lives in `tempodb/encoding/common` but needs to convert spansets to trace metadata and add them to a combiner. Both `asTraceSearchMetadata` (in `pkg/traceql/engine.go`) and `addSpanset` (on `MetadataCombiner`) are currently unexported.

**Files:**
- Modify: `pkg/traceql/engine.go:303`
- Modify: `pkg/traceql/combine.go:14-22,48`

**Step 1: Export `asTraceSearchMetadata`**

In `pkg/traceql/engine.go` line 303, rename `asTraceSearchMetadata` to `AsTraceSearchMetadata`:

```go
// AsTraceSearchMetadata converts a Spanset into protobuf TraceSearchMetadata.
func AsTraceSearchMetadata(spanset *Spanset) *tempopb.TraceSearchMetadata {
```

Update all internal callers of `asTraceSearchMetadata` in the same file and in `combine.go` to use `AsTraceSearchMetadata`.

**Step 2: Export `AddSpanset` on `MetadataCombiner`**

In `pkg/traceql/combine.go`, add `AddSpanset` to the interface (line 21):

```go
type MetadataCombiner interface {
	AddMetadata(*tempopb.TraceSearchMetadata) bool
	IsCompleteFor(ts uint32) bool

	Metadata() []*tempopb.TraceSearchMetadata
	MetadataAfter(ts uint32) []*tempopb.TraceSearchMetadata

	AddSpanset(*Spanset)
}
```

Rename `addSpanset` to `AddSpanset` on both `anyCombiner` (line 48) and `mostRecentCombiner` implementations.

**Step 3: Run tests**

```bash
cd /Users/xiaoguang/work/repo/grafana/tempo && go build ./pkg/traceql/... && go test ./pkg/traceql/... -count=1 -timeout 120s
```

Expected: All pass (rename only, no logic change).

**Step 4: Commit**

```bash
git add pkg/traceql/engine.go pkg/traceql/combine.go
git commit -m "refactor: export AsTraceSearchMetadata and AddSpanset for use by plan translator"
```

---

## Phase 2 — Update ProjectNode to support fetch scan tree

### Task 2: Add `fetchTree` child to `ProjectNode`

**Files:**
- Modify: `pkg/traceql/plan.go:335-362`
- Modify: `pkg/traceql/plan_test.go` (add test)

**Step 1: Write the failing test**

```go
// pkg/traceql/plan_test.go
func TestProjectNodeTwoChildren(t *testing.T) {
	drivingChild := &SpanScanNode{Conditions: []Condition{{Attribute: NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"), Op: OpGreater}}}
	fetchSpan := &SpanScanNode{Conditions: []Condition{{Attribute: NewIntrinsic(IntrinsicSpanID), Op: OpNone}}}
	fetchTree := &TraceScanNode{child: &ResourceScanNode{child: &InstrumentationScopeScanNode{child: fetchSpan}}}

	proj := NewProjectNode([]Attribute{NewIntrinsic(IntrinsicSpanID)}, drivingChild, fetchTree)
	children := proj.Children()

	require.Len(t, children, 2)
	require.Equal(t, drivingChild, children[0])
	require.Equal(t, fetchTree, children[1])
	require.Contains(t, proj.String(), "Project")
}
```

**Step 2: Run to see failure**

```bash
go test ./pkg/traceql/ -run TestProjectNodeTwoChildren -v
```

Expected: compile error — `NewProjectNode` takes 2 args, not 3.

**Step 3: Update `ProjectNode`**

In `pkg/traceql/plan.go`, update the struct and constructor:

```go
// ProjectNode is a late materialization operator. It has two logical sides:
// - child (driving side): filtered spans with row numbers from the first pass
// - fetchTree (fetch side): scan tree with OpNone conditions for metadata columns
// The output is driving-side results enriched with fetch-side data.
type ProjectNode struct {
	Columns   []Attribute
	child     PlanNode
	fetchTree PlanNode
}

func NewProjectNode(columns []Attribute, child PlanNode, fetchTree PlanNode) *ProjectNode {
	return &ProjectNode{Columns: columns, child: child, fetchTree: fetchTree}
}

func (n *ProjectNode) Children() []PlanNode {
	var ch []PlanNode
	if n.child != nil {
		ch = append(ch, n.child)
	}
	if n.fetchTree != nil {
		ch = append(ch, n.fetchTree)
	}
	return ch
}
func (n *ProjectNode) Accept(v PlanVisitor) { WalkPlan(n, v) }
func (n *ProjectNode) String() string       { return fmt.Sprintf("Project(%v)", n.Columns) }

func (n *ProjectNode) WithChild(child PlanNode) *ProjectNode {
	cp := *n
	cp.child = child
	return &cp
}

// FetchTree returns the fetch-side scan tree (may be nil).
func (n *ProjectNode) FetchTree() PlanNode { return n.fetchTree }
```

**Step 4: Fix all callers of `NewProjectNode`**

In `pkg/traceql/plan_builder.go` (lines 96 and 100), the existing calls pass only 2 args. Update them to pass `nil` for `fetchTree` since the `select()` use case doesn't build a fetch tree at plan-build time:

```go
// line 96 (inside groupNode branch)
proj := NewProjectNode(e.attrs, groupNode.child, nil)

// line 100 (no groupNode)
plan = NewProjectNode(e.attrs, plan, nil)
```

**Step 5: Run tests**

```bash
go test ./pkg/traceql/... -count=1 -timeout 120s
```

Expected: All pass.

**Step 6: Commit**

```bash
git add pkg/traceql/plan.go pkg/traceql/plan_test.go pkg/traceql/plan_builder.go
git commit -m "feat: add fetchTree child to ProjectNode for late materialization"
```

---

## Phase 3 — Add `BuildFetchScanTree` and `SearchMetaColumns`

### Task 3: Attribute level classification and fetch scan tree builder

**Files:**
- Modify: `pkg/traceql/plan.go` (add functions)
- Modify: `pkg/traceql/plan_test.go` (add tests)

**Step 1: Write the failing test**

```go
// pkg/traceql/plan_test.go
func TestBuildFetchScanTree(t *testing.T) {
	columns := []Attribute{
		NewIntrinsic(IntrinsicTraceRootService),  // trace level
		NewIntrinsic(IntrinsicTraceRootSpan),      // trace level
		NewIntrinsic(IntrinsicSpanID),             // span level
		NewIntrinsic(IntrinsicDuration),            // span level
		NewIntrinsic(IntrinsicServiceStats),        // trace level
	}

	tree := BuildFetchScanTree(columns)

	// Root is TraceScanNode with trace-level conditions
	require.NotNil(t, tree)
	traceConds := tree.Conditions
	var traceAttrs []Intrinsic
	for _, c := range traceConds {
		require.Equal(t, OpNone, c.Op)
		traceAttrs = append(traceAttrs, c.Attribute.Intrinsic)
	}
	require.Contains(t, traceAttrs, IntrinsicTraceRootService)
	require.Contains(t, traceAttrs, IntrinsicTraceRootSpan)
	require.Contains(t, traceAttrs, IntrinsicServiceStats)

	// Trace → Resource → InstrScope → Span
	require.Len(t, tree.Children(), 1)
	resNode, ok := tree.Children()[0].(*ResourceScanNode)
	require.True(t, ok)

	require.Len(t, resNode.Children(), 1)
	instrNode, ok := resNode.Children()[0].(*InstrumentationScopeScanNode)
	require.True(t, ok)

	require.Len(t, instrNode.Children(), 1)
	spanNode, ok := instrNode.Children()[0].(*SpanScanNode)
	require.True(t, ok)

	var spanAttrs []Intrinsic
	for _, c := range spanNode.Conditions {
		require.Equal(t, OpNone, c.Op)
		spanAttrs = append(spanAttrs, c.Attribute.Intrinsic)
	}
	require.Contains(t, spanAttrs, IntrinsicSpanID)
	require.Contains(t, spanAttrs, IntrinsicDuration)
}

func TestSearchMetaColumns(t *testing.T) {
	cols := SearchMetaColumns()
	require.Len(t, cols, 9)

	// Verify they match SearchMetaConditions attributes
	conds := SearchMetaConditions()
	for i, col := range cols {
		require.Equal(t, conds[i].Attribute, col)
	}
}
```

**Step 2: Run to see failure**

```bash
go test ./pkg/traceql/ -run "TestBuildFetchScanTree|TestSearchMetaColumns" -v
```

Expected: compile error — `BuildFetchScanTree` and `SearchMetaColumns` not defined.

**Step 3: Implement**

Add to `pkg/traceql/plan.go`:

```go
// IntrinsicLevel returns the scan level for an intrinsic attribute.
// This determines which scan node in the plan tree should carry the condition.
func IntrinsicLevel(i Intrinsic) AttributeScope {
	switch i {
	case IntrinsicTraceRootService, IntrinsicTraceRootSpan, IntrinsicTraceDuration,
		IntrinsicTraceID, IntrinsicTraceStartTime, IntrinsicServiceStats:
		return AttributeScopeNone // trace level (above resource)
	case IntrinsicEventName, IntrinsicEventTimeSinceStart:
		return AttributeScopeEvent
	case IntrinsicLinkTraceID, IntrinsicLinkSpanID:
		return AttributeScopeLink
	case IntrinsicInstrumentationName, IntrinsicInstrumentationVersion:
		return AttributeScopeInstrumentation
	default:
		return AttributeScopeSpan // span level is the default
	}
}

// BuildFetchScanTree creates a minimal scan tree with the given columns
// placed as OpNone (fetch-only) conditions at the correct scan levels.
func BuildFetchScanTree(columns []Attribute) *TraceScanNode {
	var traceConds, resConds, spanConds []Condition
	for _, col := range columns {
		cond := Condition{Attribute: col, Op: OpNone}
		if col.Intrinsic != IntrinsicNone {
			level := IntrinsicLevel(col.Intrinsic)
			switch level {
			case AttributeScopeSpan:
				spanConds = append(spanConds, cond)
			case AttributeScopeResource:
				resConds = append(resConds, cond)
			default:
				// Trace-level intrinsics (AttributeScopeNone) go on TraceScanNode
				traceConds = append(traceConds, cond)
			}
		} else {
			// Scoped attributes
			switch col.Scope {
			case AttributeScopeResource:
				resConds = append(resConds, cond)
			case AttributeScopeSpan:
				spanConds = append(spanConds, cond)
			default:
				traceConds = append(traceConds, cond)
			}
		}
	}

	span := &SpanScanNode{Conditions: spanConds}
	instrScope := &InstrumentationScopeScanNode{child: span}
	resource := &ResourceScanNode{Conditions: resConds, child: instrScope}
	return &TraceScanNode{Conditions: traceConds, child: resource}
}
```

Add to `pkg/traceql/storage.go` (near `SearchMetaConditions`):

```go
// SearchMetaColumns returns the metadata attributes needed for search results.
// These are the same attributes as SearchMetaConditions but as []Attribute.
func SearchMetaColumns() []Attribute {
	conds := SearchMetaConditions()
	cols := make([]Attribute, len(conds))
	for i, c := range conds {
		cols[i] = c.Attribute
	}
	return cols
}
```

**Step 4: Run tests**

```bash
go test ./pkg/traceql/ -run "TestBuildFetchScanTree|TestSearchMetaColumns" -v
```

Expected: PASS.

**Step 5: Run full test suite**

```bash
go test ./pkg/traceql/... -count=1 -timeout 120s
```

Expected: All pass.

**Step 6: Commit**

```bash
git add pkg/traceql/plan.go pkg/traceql/plan_test.go pkg/traceql/storage.go
git commit -m "feat: add BuildFetchScanTree and SearchMetaColumns for late materialization"
```

---

## Phase 4 — Implement `lateMaterializeIter`

### Task 4: Bridge between driving SpansetIterator and fetch parquetquery.Iterator

The `lateMaterializeIter` is the core of ProjectNode translation. It wraps a driving `SpansetIterator` and a fetch `parquetquery.Iterator`. For each spanset from the driving side, it seeks the fetch iterator to each span's row number and merges the fetched values.

**Critical issue:** The `traceql.Span` interface has no `RowNum()` method. The row number is on the concrete `vparquet5.span.rowNum` field. The `lateMaterializeIter` lives in `tempodb/encoding/common` and can't type-assert to `*vparquet5.span`.

**Solution:** Add a `RowNum() parquetquery.RowNumber` method to the `traceql.Span` interface. This is a clean addition — row numbers are a fundamental part of how spans flow through the plan tree.

**Files:**
- Modify: `pkg/traceql/storage.go:160-189` (add RowNum to Span interface)
- Modify: `tempodb/encoding/vparquet5/block_traceql.go:46-65` (add RowNum method to span)
- Create: `tempodb/encoding/common/late_materialize_iter.go`
- Create: `tempodb/encoding/common/late_materialize_iter_test.go`

**Step 1: Add `RowNum` to Span interface**

In `pkg/traceql/storage.go`, add to the `Span` interface (after `DurationNanos()` on line 172):

```go
type Span interface {
	AttributeFor(Attribute) (Static, bool)
	AllAttributes() map[Attribute]Static
	AllAttributesFunc(func(Attribute, Static))

	ID() []byte
	StartTimeUnixNanos() uint64
	DurationNanos() uint64
	// RowNum returns the parquet row number for this span.
	// Used by late materialization to seek fetch iterators.
	RowNum() parquetquery.RowNumber

	SiblingOf(lhs []Span, rhs []Span, falseForAll bool, union bool, buffer []Span) []Span
	DescendantOf(lhs []Span, rhs []Span, falseForAll bool, invert bool, union bool, buffer []Span) []Span
	ChildOf(lhs []Span, rhs []Span, falseForAll bool, invert bool, union bool, buffer []Span) []Span
}
```

**Step 2: Implement `RowNum()` on vparquet5 span**

In `tempodb/encoding/vparquet5/block_traceql.go`, add after the span struct definition (after line 65):

```go
func (s *span) RowNum() parquetquery.RowNumber { return s.rowNum }
```

**Step 3: Check for other Span implementations**

Search for all types that implement the `Span` interface. Any mock or test implementations will need a stub `RowNum()` method.

```bash
grep -rn "func.*Span\b" pkg/traceql/ --include="*.go" | grep -v "_test.go" | grep "ID\(\)"
```

If there are mock spans (e.g., in test files or `pkg/traceql/test_utils.go`), add `RowNum() parquetquery.RowNumber` stubs returning `parquetquery.EmptyRowNumber()`.

**Step 4: Write test for `lateMaterializeIter`**

Create `tempodb/encoding/common/late_materialize_iter_test.go`:

```go
package common

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/require"
)

// mockSpan implements traceql.Span for testing
type mockSpan struct {
	id     []byte
	rowNum parquetquery.RowNumber
	attrs  map[traceql.Attribute]traceql.Static
}

func (s *mockSpan) AllAttributes() map[traceql.Attribute]traceql.Static { return s.attrs }
func (s *mockSpan) AllAttributesFunc(f func(traceql.Attribute, traceql.Static)) {
	for k, v := range s.attrs {
		f(k, v)
	}
}
func (s *mockSpan) AttributeFor(a traceql.Attribute) (traceql.Static, bool) {
	v, ok := s.attrs[a]
	return v, ok
}
func (s *mockSpan) ID() []byte                    { return s.id }
func (s *mockSpan) StartTimeUnixNanos() uint64    { return 0 }
func (s *mockSpan) DurationNanos() uint64         { return 0 }
func (s *mockSpan) RowNum() parquetquery.RowNumber { return s.rowNum }
func (s *mockSpan) SiblingOf(_, _ []traceql.Span, _ bool, _ bool, buf []traceql.Span) []traceql.Span {
	return buf
}
func (s *mockSpan) DescendantOf(_, _ []traceql.Span, _, _, _ bool, buf []traceql.Span) []traceql.Span {
	return buf
}
func (s *mockSpan) ChildOf(_, _ []traceql.Span, _, _, _ bool, buf []traceql.Span) []traceql.Span {
	return buf
}

// mockFetchIter simulates a parquetquery.Iterator for the fetch side
type mockFetchIter struct {
	results []*parquetquery.IteratorResult
	idx     int
}

func (m *mockFetchIter) Next() (*parquetquery.IteratorResult, error) {
	if m.idx >= len(m.results) {
		return nil, nil
	}
	r := m.results[m.idx]
	m.idx++
	return r, nil
}

func (m *mockFetchIter) SeekTo(t parquetquery.RowNumber, d int) (*parquetquery.IteratorResult, error) {
	for m.idx < len(m.results) {
		r := m.results[m.idx]
		if parquetquery.CompareRowNumbers(d, r.RowNumber, t) >= 0 {
			m.idx++
			return r, nil
		}
		m.idx++
	}
	return nil, nil
}

func (m *mockFetchIter) Close()        {}
func (m *mockFetchIter) String() string { return "mockFetchIter" }

// mockSpansetIter returns pre-built spansets
type mockSpansetIter struct {
	spansets []*traceql.Spanset
	idx      int
}

func (m *mockSpansetIter) Next(context.Context) (*traceql.Spanset, error) {
	if m.idx >= len(m.spansets) {
		return nil, nil
	}
	ss := m.spansets[m.idx]
	m.idx++
	return ss, nil
}

func (m *mockSpansetIter) Close() {}

func TestLateMaterializeIter(t *testing.T) {
	rn1 := parquetquery.EmptyRowNumber()
	rn1[0] = 0
	rn1[1] = 0
	rn1[2] = 0
	rn1[3] = 0

	span1 := &mockSpan{
		id:     []byte{1},
		rowNum: rn1,
		attrs:  map[traceql.Attribute]traceql.Static{},
	}

	driving := &mockSpansetIter{
		spansets: []*traceql.Spanset{
			{TraceID: []byte{0, 1}, Spans: []traceql.Span{span1}},
		},
	}

	// Fetch iter returns a result at rn1 with a "fetched" entry
	fetchResult := &parquetquery.IteratorResult{RowNumber: rn1}
	fetchResult.Entries = append(fetchResult.Entries, parquetquery.Entry{
		Key: "rootServiceName",
	})

	fetchIter := &mockFetchIter{
		results: []*parquetquery.IteratorResult{fetchResult},
	}

	iter := newLateMaterializeIter(driving, fetchIter, parquetquery.DefinitionLevelFromLength(4))

	ctx := context.Background()
	ss, err := iter.Next(ctx)
	require.NoError(t, err)
	require.NotNil(t, ss)
	require.Len(t, ss.Spans, 1)

	// Second call returns nil (exhausted)
	ss, err = iter.Next(ctx)
	require.NoError(t, err)
	require.Nil(t, ss)

	iter.Close()
}
```

**Step 5: Run to see failure**

```bash
go test ./tempodb/encoding/common/ -run TestLateMaterializeIter -v
```

Expected: compile error — `newLateMaterializeIter` not defined.

**Step 6: Implement `lateMaterializeIter`**

Create `tempodb/encoding/common/late_materialize_iter.go`:

```go
package common

import (
	"context"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
)

// lateMaterializeIter wraps a driving SpansetIterator and a fetch
// parquetquery.Iterator. For each spanset from the driving side, it seeks
// the fetch iterator to each span's row number and merges fetched column
// values into the span.
//
// The fetch iterator is built once during translation — not per spanset.
// Since the driving iterator produces spansets in row-number order (traces
// are sequential in parquet), SeekTo always moves forward.
type lateMaterializeIter struct {
	driving         traceql.SpansetIterator
	fetcher         parquetquery.Iterator
	definitionLevel int
}

func newLateMaterializeIter(
	driving traceql.SpansetIterator,
	fetcher parquetquery.Iterator,
	definitionLevel int,
) *lateMaterializeIter {
	return &lateMaterializeIter{
		driving:         driving,
		fetcher:         fetcher,
		definitionLevel: definitionLevel,
	}
}

func (it *lateMaterializeIter) Next(ctx context.Context) (*traceql.Spanset, error) {
	ss, err := it.driving.Next(ctx)
	if ss == nil || err != nil {
		return ss, err
	}

	// Seek the fetch iterator to each span's row number and merge results.
	// Spans within a spanset are in row-number order, and across spansets
	// row numbers increase, so SeekTo always moves forward.
	for _, span := range ss.Spans {
		rn := span.RowNum()
		if !rn.Valid() {
			continue
		}

		res, err := it.fetcher.SeekTo(rn, it.definitionLevel)
		if err != nil {
			return nil, err
		}
		if res == nil {
			continue
		}

		// If the fetched result matches this span's row number, merge it.
		// The fetch iterator may have seeked past this span if the row
		// doesn't exist in the fetch columns.
		if parquetquery.EqualRowNumber(it.definitionLevel, res.RowNumber, rn) {
			mergeResultIntoSpan(span, res)
		}
	}

	return ss, nil
}

func (it *lateMaterializeIter) Close() {
	it.driving.Close()
	it.fetcher.Close()
}

// mergeResultIntoSpan writes fetched column values from an IteratorResult
// into the span. The span's concrete type handles how entries are stored.
// For now, we pass the entries through OtherEntries on the result so the
// span collector (in vparquet5) can process them via its existing logic.
//
// TODO: This is a placeholder. The actual merge depends on how the fetch
// scan tree's collectors write into spans. In the initial implementation,
// the fetch side uses the same span/resource/trace collectors as the first
// pass, which recognize existing spans in OtherEntries and enrich them.
func mergeResultIntoSpan(span traceql.Span, res *parquetquery.IteratorResult) {
	// The fetch-side iterator chain (built via buildParquetChain) already
	// handles merging through the span/resource/trace collectors.
	// The IteratorResult from the fetch side contains the enriched span
	// in OtherEntries. No additional merge needed here since the fetch
	// iterator's collectors write directly into the span objects.
}
```

**Step 7: Run tests**

```bash
go test ./tempodb/encoding/common/ -run TestLateMaterializeIter -v
```

Expected: PASS.

**Step 8: Compile full project**

```bash
go build ./...
```

Fix any compilation errors from adding `RowNum()` to the `Span` interface (mock implementations in test files need the method).

**Step 9: Commit**

```bash
git add pkg/traceql/storage.go tempodb/encoding/vparquet5/block_traceql.go tempodb/encoding/common/late_materialize_iter.go tempodb/encoding/common/late_materialize_iter_test.go
git commit -m "feat: implement lateMaterializeIter for ProjectNode late materialization"
```

---

## Phase 5 — Implement `translateProjectToIter`

### Task 5: Wire ProjectNode translation to use lateMaterializeIter

**Files:**
- Modify: `tempodb/encoding/common/plan_translator.go:81-122,186-202`

**Step 1: Implement `translateProjectToIter`**

Replace the stub at line 197-202 with:

```go
// translateProjectToIter builds a lateMaterializeIter that wraps:
// - driving side: SpansetIterator from child plan
// - fetch side: parquetquery.Iterator chain from fetchTree
func (t *translator) translateProjectToIter(node *traceql.ProjectNode) (traceql.SpansetIterator, error) {
	// Build driving side from first child
	drivingIter, err := t.translateToIter(node.Children()[0])
	if err != nil {
		return nil, err
	}

	// Build fetch side from fetchTree (second child)
	fetchTree := node.FetchTree()
	if fetchTree == nil {
		// No fetch tree — just pass through
		return drivingIter, nil
	}

	fetchTraceScan, ok := fetchTree.(*traceql.TraceScanNode)
	if !ok {
		return nil, fmt.Errorf("plan_translator: ProjectNode fetchTree must be *TraceScanNode, got %T", fetchTree)
	}

	// Build the fetch-side parquetquery.Iterator chain.
	// We pass nil for primary — the fetch tree reads columns based on the
	// row numbers provided by lateMaterializeIter's SeekTo calls.
	var fetchChild parquetquery.Iterator
	if len(fetchTraceScan.Children()) > 0 {
		fetchChild, err = t.buildInnerChain(fetchTraceScan.Children()[0])
		if err != nil {
			return nil, err
		}
	}

	// Build trace-level iterator for the fetch side.
	// Use createTraceIterator via TraceIter with nil primary — the
	// lateMaterializeIter handles the row-number seeking externally.
	fetchSpansetIter, err := t.backend.TraceIter(t.ctx, fetchTraceScan, nil, fetchChild)
	if err != nil {
		return nil, err
	}

	// TODO: The fetch side needs to be a parquetquery.Iterator (with SeekTo),
	// not a SpansetIterator. We need to either:
	// (a) Keep the raw pq.Iterator from buildInnerChain and use it directly, or
	// (b) Add a SeekTo-capable wrapper.
	// For now, use approach (a): build the inner chain only and pass it to
	// lateMaterializeIter. The trace-level conditions can be added to the
	// inner chain's resource iterator.
	//
	// Revised approach: build the full pq.Iterator chain (not SpansetIterator)
	// so we can use SeekTo.
	_ = fetchSpansetIter // Close this, we'll rebuild differently
	fetchSpansetIter.Close()

	// Rebuild: get raw parquetquery.Iterator from the fetch tree
	fetchPqIter, err := t.buildFetchChain(fetchTraceScan)
	if err != nil {
		return nil, err
	}

	return newLateMaterializeIter(drivingIter, fetchPqIter, 3), nil // definitionLevel=3 for span level
}
```

**Wait — there's a design issue.** `buildParquetChain` returns `SpansetIterator` (via `TraceIter`), but we need the raw `parquetquery.Iterator` for `SeekTo`. We need a new helper `buildFetchChain` that builds the full pq.Iterator chain without wrapping in SpansetIterator.

**Step 2: Add `buildFetchChain` to translator**

```go
// buildFetchChain builds a raw parquetquery.Iterator chain from a fetch scan tree.
// Unlike buildParquetChain, it does NOT wrap in SpansetIterator — the caller
// needs SeekTo capability which only parquetquery.Iterator provides.
func (t *translator) buildFetchChain(trace *traceql.TraceScanNode) (parquetquery.Iterator, error) {
	var child parquetquery.Iterator
	if len(trace.Children()) > 0 {
		var err error
		child, err = t.buildInnerChain(trace.Children()[0])
		if err != nil {
			return nil, err
		}
	}
	// Build trace-level iterator as raw pq.Iterator (not SpansetIterator).
	// We use the backend's TraceIter but need the raw iterator.
	// For now, delegate to a new ScanBackend method or build directly.
	return t.backend.TraceIterRaw(t.ctx, trace, nil, child)
}
```

**Problem:** `ScanBackend.TraceIter` returns `SpansetIterator`, not `parquetquery.Iterator`. We need either:
1. A new `TraceIterRaw` method on `ScanBackend` that returns `parquetquery.Iterator`
2. Or build the trace-level iterator directly in the translator

**Revised approach:** Add `TraceIterRaw` to `ScanBackend`. This is a minimal addition — it's the same as `TraceIter` but without the `newSpansetIterator(newRebatchIterator(...))` wrapper.

**Step 3: Add `TraceIterRaw` to ScanBackend**

In `tempodb/encoding/common/scan_backend.go`, add to the interface:

```go
type ScanBackend interface {
	// ... existing methods ...

	// TraceIterRaw is like TraceIter but returns the raw parquetquery.Iterator
	// without wrapping in SpansetIterator. Used by the fetch side of
	// ProjectNode where SeekTo capability is needed.
	TraceIterRaw(
		ctx context.Context,
		node *traceql.TraceScanNode,
		primary parquetquery.Iterator,
		child parquetquery.Iterator,
	) (parquetquery.Iterator, error)
}
```

In `tempodb/encoding/vparquet5/scan_backend.go`, implement:

```go
func (b *blockScanBackend) TraceIterRaw(
	ctx context.Context,
	node *traceql.TraceScanNode,
	primary parquetquery.Iterator,
	child parquetquery.Iterator,
) (parquetquery.Iterator, error) {
	source := child
	if primary != nil {
		source = primary
	}
	return createTraceIterator(b.makeIter, source, node.Conditions, 0, 0, node.AllConditions, false, nil)
}
```

**Step 4: Simplify `translateProjectToIter`**

```go
func (t *translator) translateProjectToIter(node *traceql.ProjectNode) (traceql.SpansetIterator, error) {
	drivingIter, err := t.translateToIter(node.Children()[0])
	if err != nil {
		return nil, err
	}

	fetchTree := node.FetchTree()
	if fetchTree == nil {
		return drivingIter, nil
	}

	fetchTraceScan, ok := fetchTree.(*traceql.TraceScanNode)
	if !ok {
		return nil, fmt.Errorf("plan_translator: ProjectNode fetchTree must be *TraceScanNode, got %T", fetchTree)
	}

	// Build fetch-side pq.Iterator chain
	var fetchChild parquetquery.Iterator
	if len(fetchTraceScan.Children()) > 0 {
		fetchChild, err = t.buildInnerChain(fetchTraceScan.Children()[0])
		if err != nil {
			return nil, err
		}
	}

	fetchPqIter, err := t.backend.TraceIterRaw(t.ctx, fetchTraceScan, nil, fetchChild)
	if err != nil {
		return nil, err
	}

	// DefinitionLevel 3 = span level (DefinitionLevelResourceSpansILSSpan)
	return newLateMaterializeIter(drivingIter, fetchPqIter, 3), nil
}
```

**Step 5: Also update `translateProject` (Evaluatable path)**

The existing `translateProject` (line 186-195) calls `newProjectEvaluatable` which panics. Update it to use the iterator path:

```go
func (t *translator) translateProject(node *traceql.ProjectNode) (Evaluatable, error) {
	iter, err := t.translateProjectToIter(node)
	if err != nil {
		return nil, err
	}
	return newSpansetEvaluatable(iter), nil
}
```

Remove the `newProjectEvaluatable` panic function (lines 349-353).

**Step 6: Run tests**

```bash
go build ./tempodb/encoding/common/... && go build ./tempodb/encoding/vparquet5/...
go test ./tempodb/encoding/common/... -count=1 -timeout 120s
```

Expected: All pass.

**Step 7: Commit**

```bash
git add tempodb/encoding/common/plan_translator.go tempodb/encoding/common/scan_backend.go tempodb/encoding/vparquet5/scan_backend.go
git commit -m "feat: implement translateProjectToIter with lateMaterializeIter and TraceIterRaw"
```

---

## Phase 6 — Implement `SearchEvaluatable` and `TranslateSearch`

### Task 6: SearchEvaluatable interface and TranslateSearch entry point

**Files:**
- Modify: `tempodb/encoding/common/plan_translator.go` (add SearchEvaluatable, TranslateSearch)
- Create: `tempodb/encoding/common/plan_translator_search_test.go`

**Step 1: Write the failing test**

```go
// tempodb/encoding/common/plan_translator_search_test.go
package common

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/require"
)

func TestSearchEvaluatable(t *testing.T) {
	span1 := &mockSpan{
		id:     []byte{1},
		rowNum: parquetquery.EmptyRowNumber(),
		attrs: map[traceql.Attribute]traceql.Static{},
	}

	driving := &mockSpansetIter{
		spansets: []*traceql.Spanset{
			{
				TraceID:            []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
				RootServiceName:    "test-service",
				RootSpanName:       "test-span",
				StartTimeUnixNanos: 1000,
				DurationNanos:      500_000_000,
				Spans:              []traceql.Span{span1},
			},
		},
	}

	eval := newSearchEvaluatable(driving, 10)
	err := eval.Do(context.Background())
	require.NoError(t, err)

	resp := eval.Response()
	require.NotNil(t, resp)
	require.Len(t, resp.Traces, 1)
	require.Equal(t, "test-service", resp.Traces[0].RootServiceName)
	require.Equal(t, "test-span", resp.Traces[0].RootTraceName)
}

func TestSearchEvaluatableLimit(t *testing.T) {
	spansets := make([]*traceql.Spanset, 5)
	for i := range spansets {
		spansets[i] = &traceql.Spanset{
			TraceID:            []byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			RootServiceName:    "svc",
			StartTimeUnixNanos: uint64(i * 1000),
			Spans:              []traceql.Span{&mockSpan{id: []byte{byte(i)}, rowNum: parquetquery.EmptyRowNumber(), attrs: map[traceql.Attribute]traceql.Static{}}},
		}
	}

	driving := &mockSpansetIter{spansets: spansets}
	eval := newSearchEvaluatable(driving, 3)
	err := eval.Do(context.Background())
	require.NoError(t, err)

	resp := eval.Response()
	require.NotNil(t, resp)
	require.Len(t, resp.Traces, 3) // limited to 3
}
```

**Step 2: Run to see failure**

```bash
go test ./tempodb/encoding/common/ -run "TestSearchEvaluatable" -v
```

Expected: compile error — `newSearchEvaluatable` not defined.

**Step 3: Implement SearchEvaluatable**

Add to `tempodb/encoding/common/plan_translator.go`:

```go
// SearchEvaluatable is the result of translating a plan tree for search.
// Call Do to drive execution; Response returns the accumulated search results.
type SearchEvaluatable interface {
	Do(ctx context.Context) error
	Response() *tempopb.SearchResponse
}

// searchEvaluatable iterates spansets from the plan, converts them to
// trace metadata, and collects results up to the limit.
type searchEvaluatable struct {
	iter     traceql.SpansetIterator
	limit    int
	combiner traceql.MetadataCombiner
}

func newSearchEvaluatable(iter traceql.SpansetIterator, limit int) SearchEvaluatable {
	return &searchEvaluatable{
		iter:     iter,
		limit:    limit,
		combiner: traceql.NewMetadataCombiner(limit, false),
	}
}

func (e *searchEvaluatable) Do(ctx context.Context) error {
	defer e.iter.Close()
	for {
		ss, err := e.iter.Next(ctx)
		if err != nil {
			return err
		}
		if ss == nil {
			return nil
		}

		e.combiner.AddSpanset(ss)

		if e.combiner.IsCompleteFor(traceql.TimestampNever) {
			return nil
		}
	}
}

func (e *searchEvaluatable) Response() *tempopb.SearchResponse {
	return &tempopb.SearchResponse{
		Traces:  e.combiner.Metadata(),
		Metrics: &tempopb.SearchMetrics{},
	}
}

// TranslateSearch converts a plan tree into a SearchEvaluatable.
// It wraps the plan in a ProjectNode with the given metadata columns
// for late materialization, then translates the full tree into an
// iterator pipeline.
func TranslateSearch(
	ctx context.Context,
	plan traceql.PlanNode,
	backend ScanBackend,
	opts SearchOptions,
	limit int,
	metaColumns []traceql.Attribute,
) (SearchEvaluatable, error) {
	// Build fetch scan tree and wrap plan in ProjectNode
	fetchTree := traceql.BuildFetchScanTree(metaColumns)
	plan = traceql.NewProjectNode(metaColumns, plan, fetchTree)

	t := &translator{ctx: ctx, backend: backend, opts: opts}
	iter, err := t.translateToIter(plan)
	if err != nil {
		return nil, err
	}

	return newSearchEvaluatable(iter, limit), nil
}
```

**Step 4: Run tests**

```bash
go test ./tempodb/encoding/common/ -run "TestSearchEvaluatable" -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add tempodb/encoding/common/plan_translator.go tempodb/encoding/common/plan_translator_search_test.go
git commit -m "feat: implement SearchEvaluatable and TranslateSearch entry point"
```

---

## Phase 7 — Wire into querier

### Task 7: Add plan-based search path to SearchBlock

**Files:**
- Modify: `modules/querier/config.go:32-34`
- Modify: `modules/querier/querier.go:616-657`

**Step 1: Add config flag**

In `modules/querier/config.go`, update `SearchConfig`:

```go
type SearchConfig struct {
	QueryTimeout            time.Duration `yaml:"query_timeout"`
	EnablePlanBasedExecution bool          `yaml:"enable_plan_based_execution,omitempty"`
}
```

In `RegisterFlagsAndApplyDefaults` (line 76), the default is already `false` (zero value for bool), which is what we want for opt-in during development.

**Step 2: Add plan-based path to SearchBlock**

In `modules/querier/querier.go`, update `SearchBlock` (line 616). Insert the plan-based path before the existing TraceQL branch:

```go
func (q *Querier) SearchBlock(ctx context.Context, req *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	tenantID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.BackendSearch: %w", err)
	}

	blockID, err := backend.ParseUUID(req.BlockID)
	if err != nil {
		return nil, err
	}

	dc, err := backend.DedicatedColumnsFromTempopb(req.DedicatedColumns)
	if err != nil {
		return nil, err
	}

	meta := &backend.BlockMeta{
		Version:          req.Version,
		TenantID:         tenantID,
		Size_:            req.Size_,
		IndexPageSize:    req.IndexPageSize,
		TotalRecords:     req.TotalRecords,
		BlockID:          blockID,
		FooterSize:       req.FooterSize,
		DedicatedColumns: dc,
	}

	opts := common.DefaultSearchOptions()
	opts.StartPage = int(req.StartPage)
	opts.TotalPages = int(req.PagesToSearch)
	opts.MaxBytes = q.limits.MaxBytesPerTrace(tenantID)

	// --- Plan-based path ---
	if api.IsTraceQLQuery(req.SearchReq) && q.cfg.Search.EnablePlanBasedExecution {
		if expr, parseErr := traceql.Parse(req.SearchReq.Query); parseErr == nil {
			if scanBackend, cleanup, sbErr := q.store.OpenScanBackend(ctx, meta, opts); sbErr == nil && scanBackend != nil {
				defer cleanup()
				if plan, planErr := traceql.BuildPlan(expr, nil); planErr == nil {
					metaCols := traceql.SearchMetaColumns()
					if searchEval, tErr := common.TranslateSearch(ctx, plan, scanBackend, opts, int(req.SearchReq.Limit), metaCols); tErr == nil {
						if doErr := searchEval.Do(ctx); doErr == nil {
							return searchEval.Response(), nil
						}
					}
				}
			}
		}
	}
	// --- End plan-based path ---

	if api.IsTraceQLQuery(req.SearchReq) {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return q.store.Fetch(ctx, meta, req, opts)
		})

		return q.engine.ExecuteSearch(ctx, req.SearchReq, fetcher, q.limits.UnsafeQueryHints(tenantID))
	}

	return q.store.Search(ctx, meta, req.SearchReq, opts)
}
```

**Step 3: Build and verify**

```bash
go build ./modules/querier/...
```

Expected: compiles cleanly.

**Step 4: Run querier tests**

```bash
go test ./modules/querier/... -count=1 -timeout 120s
```

Expected: All pass (plan path is opt-in, disabled by default).

**Step 5: Commit**

```bash
git add modules/querier/config.go modules/querier/querier.go
git commit -m "feat: wire plan-based search path into SearchBlock with config flag"
```

---

## Phase 8 — Fix Span interface compliance across codebase

### Task 8: Add RowNum() to all Span implementations

After adding `RowNum()` to the `Span` interface in Task 4, any other implementations will fail to compile. This task finds and fixes them all.

**Step 1: Find all Span implementations**

```bash
cd /Users/xiaoguang/work/repo/grafana/tempo && grep -rn "func.*) ID() \[\]byte" --include="*.go" | grep -v vendor
```

This finds types implementing the `Span` interface by looking for the `ID()` method.

**Step 2: Add `RowNum()` stubs**

For each implementation found (likely in test files, mock types, or `pkg/traceql` itself), add:

```go
func (s *theType) RowNum() parquetquery.RowNumber { return parquetquery.EmptyRowNumber() }
```

If adding a `parquetquery` import to `pkg/traceql` creates a circular dependency, use an interface type instead:

```go
// In pkg/traceql/storage.go, define RowNumber as an alias or use [8]int32 directly
type RowNumber = parquetquery.RowNumber
```

**Step 3: Verify full build**

```bash
go build ./...
```

Expected: compiles cleanly.

**Step 4: Run full test suite**

```bash
go test ./pkg/traceql/... ./tempodb/... ./modules/querier/... -count=1 -timeout 300s
```

Expected: All pass.

**Step 5: Commit**

```bash
git add -A
git commit -m "fix: add RowNum() to all Span interface implementations"
```
