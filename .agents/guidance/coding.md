# Go Coding Standards

Guidelines for writing clean, idiomatic Go in this codebase. Apply these when implementing new features or fixing bugs.

---

## Core Philosophy

- Write clean, boring, idiomatic Go — simple solutions over clever ones
- Standard library first; reach for external packages only when necessary
- Explicit over implicit; clarity over brevity

---

## Testing

Tests are expected for all new and changed behaviour. Prefer writing the test first — it forces you to think about the interface before the implementation and proves the test actually catches failures. This is the default approach for new features and bug fixes, but it is a strong preference, not an absolute rule.

**Preferred workflow:**
1. Write the test first
2. Verify the test fails (proves it actually tests something)
3. Write the minimal implementation to make it pass
4. Verify the test passes
5. Refactor if needed

**Test structure — table-driven with subtests:**
```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name string
        in   *Input
        want Output
        err  bool
    }{
        {"nil input", nil, Output{}, true},
        {"empty", &Input{}, Output{}, false},
        {"valid", &Input{Value: 10}, Output{Result: 10}, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Something(tt.in)
            if (err != nil) != tt.err {
                t.Fatalf("error = %v, wantErr %v", err, tt.err)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Coverage targets** (guidelines, not hard rules — the goal is always to improve coverage, not just meet a number):
- Critical paths: >80%
- Business logic: >75%
- Overall: >70%

Always test: happy path, error cases, edge cases (nil, empty, zero values, boundary conditions).

---

## Error Handling

**Wrap errors with context:**
```go
// Good — caller can trace the failure
if err != nil {
    return fmt.Errorf("failed to open config file: %w", err)
}

// Bad — opaque to callers
if err != nil {
    return err
}
```

**Always check errors — never silence them:**
```go
// Bad
_ = someFunc()
result, _ := os.Open(path)

// Good
if err := someFunc(); err != nil {
    return fmt.Errorf("someFunc: %w", err)
}
```

**Compare errors correctly:**
```go
// Good
if errors.Is(err, ErrNotFound) { ... }
if errors.As(err, &notFoundErr) { ... }

// Bad — breaks with wrapped errors
if err == ErrNotFound { ... }
```

**Do not panic in production server or library code.** Use error returns so callers can handle failures. If initialization truly cannot continue, return an error up to `main` and exit there. Limited `panic(...)` may be acceptable in CLI tooling or for genuine programmer errors (e.g., impossible state at startup), but never for normal error handling in long-running services.

---

## Concurrency

**Goroutines must have a defined lifetime:**
```go
// Good — goroutine respects context cancellation
go func() {
    defer wg.Done()
    select {
    case <-ctx.Done():
        return
    case work := <-ch:
        process(work)
    }
}()
```

**Protect shared state explicitly:**
```go
type SafeCounter struct {
    mu    sync.Mutex
    count int
}

func (c *SafeCounter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}
```

**Rules:**
- Use channels for orchestration/signaling; mutexes for protecting shared state
- Always run `go test -race` — fix all races, do not suppress
- Context must be the first parameter and propagated through all goroutines
- Worker pools must bound concurrency explicitly — no unbounded goroutine spawning
- No goroutine leaks — every goroutine must terminate or be terminatable via context/channel

---

## Resource Management

```go
// Defer cleanup immediately after acquiring the resource
f, err := os.Open(path)
if err != nil {
    return fmt.Errorf("open %s: %w", path, err)
}
defer f.Close()
```

Applies to: files, HTTP response bodies, database rows, locks, connections.

---

## Idiomatic Patterns

**Accept interfaces, return concrete types:**
```go
// Good — flexible input, predictable output
func Process(r io.Reader) (*Result, error)

// Avoid — locks callers into your type
func Process(r *MyReader) (*Result, error)
```

**Early returns over nested conditions:**
```go
// Good
func validate(cfg *Config) error {
    if cfg == nil {
        return errors.New("config is nil")
    }
    if cfg.Timeout <= 0 {
        return errors.New("timeout must be positive")
    }
    return nil
}
```

**Zero values should be useful** — design types so the zero value is valid and safe.

**Avoid:**
- Global mutable state (breaks testability and concurrency safety)
- `init()` unless strictly necessary
- Reflection unless there is no alternative
- `any`/`interface{}` when practical — prefer concrete types or typed/constrained generics where they make the API clearer

---

## Code Organization

```
package/
  doc.go          # Package-level doc comment (for non-trivial packages)
  feature.go      # Implementation
  feature_test.go # Tests alongside implementation
```

**Naming:**
- Packages: lowercase, single word, no underscores (`traceql`, not `trace_ql`)
- No stuttering: `store.Store` is fine, `store.StoreStore` is not
- Exported: `PascalCase`; unexported: `camelCase`
- Constants: `PascalCase` for exported; avoid `ALL_CAPS`

**Complexity:**
- Keep cyclomatic complexity < 15 (hard limit: 40)
- Extract complex logic into named helper functions
- A function should do one thing

---

## Performance

### Measure first

Profile before optimizing. A 3x slower implementation is acceptable if it is not on the hot path — context matters more than benchmark numbers alone.

```bash
# CPU and memory profile
go test -cpuprofile=cpu.out -memprofile=mem.out -bench=. ./...
go tool pprof cpu.out

# Allocation count
go test -bench=. -benchmem ./...
```

Microbenchmarks (`go test -bench`) are valuable, especially for isolated paths like TraceQL engine methods — less CPU and less memory is a clear improvement. When a change has an unclear trade-off (e.g., less CPU but more memory), profile the affected area against a realistic workload to decide if the trade-off is acceptable.

### Allocation patterns

- Pre-allocate slices and maps when size is known: `make([]T, 0, n)`
- Avoid allocations in tight loops — reuse buffers where possible
- Use `sync.Pool` for frequently allocated short-lived objects
- Use `strings.Builder` for string construction in loops, never `+=`
- Pass large structs by pointer to avoid copies on every call

### Algorithm complexity

- Prefer O(n log n) over O(n²) for any operation on unbounded input
- Avoid repeated linear scans over the same data — build an index
- For Parquet iterators: use `SeekTo()` to skip row groups rather than scanning exhausted ranges

### Sorting

Sorting in Tempo is best-effort for compression. It does not need to be perfect. Do not over-engineer sort stability if it is not on the critical path.

### Known hot paths

These run on every span or every row — changes here require benchmarks:

| Path | File | Evidence |
|------|------|----------|
| Span ingestion — `instance.push()` | `modules/ingester/instance.go` | `BenchmarkInstancePush`, `BenchmarkInstanceContention` |
| Live trace append — `liveTrace.Push()` | `modules/ingester/trace.go` | Called per-span under mutex |
| TraceQL binary op evaluation | `pkg/traceql/ast_execute.go` | `BenchmarkBinOp` |
| TraceQL group operation | `pkg/traceql/ast_execute.go` | Per-span map lookup and slice append |
| Parquet row number comparison | `pkg/parquetquery/iters.go` | `BenchmarkEqualRowNumber`, `BenchmarkColumnIterator` |
| Parquet TraceQL block execution | `tempodb/encoding/vparquet5/block_traceql.go` | 4+ relationship operator benchmarks |
| String interning | `pkg/parquetquery/intern/intern.go` | Uses `unsafe` for repeated values |

### Hot path checklist

Before landing code on a known hot path:
- Run `go test -bench=. -benchmem` on the affected package and check allocs/op
- Check escape analysis: `go build -gcflags="-m"` to see what escapes to heap
- Confirm with a realistic trace dataset, not just unit-scale inputs
- For ingester paths: run `BenchmarkInstanceContention` to check mutex pressure

---

## Before Submitting

See `.agents/guidance/precommit.md` for the full checklist. Minimum bar: `make fmt`, `make lint base=origin/main`, `make test`.

---

## Tempo-Specific Conventions

Patterns observed consistently across the codebase. These go beyond generic Go idioms and reflect how Tempo is actually built.

### Warnings vs Errors in Config Validation

Return both `warnings []error` and `err error` from validation functions. Warnings are non-fatal issues (Tempo starts but with degraded config); errors are fatal.

```go
func (r *validator) Validate(cfg *Config) (warnings []error, err error) {
    if cfg.SomeField <= 0 {
        return nil, errors.New("some_field must be positive")
    }
    if cfg.OptionalField == "" {
        warnings = append(warnings, ConfigWarning{
            Message: "optional_field not set",
            Explain: "feature X will be disabled",
        })
    }
    return warnings, nil
}
```

This pattern lets operators run Tempo with partial config rather than failing completely.

### Functional Options for Complex Constructors

Use typed option interfaces (not `func(*T)`) so options are self-documenting and can carry state:

```go
type LeftJoinIteratorOption interface {
    applyToLeftJoinIterator(*LeftJoinIterator)
}

type CollectorOption struct{ collector Collector }

func (c CollectorOption) applyToLeftJoinIterator(j *LeftJoinIterator) {
    j.collector = c.collector
}

func WithCollector(c Collector) CollectorOption { return CollectorOption{c} }

func NewLeftJoinIterator(opts ...LeftJoinIteratorOption) (*LeftJoinIterator, error) {
    j := &LeftJoinIterator{}
    for _, opt := range opts {
        opt.applyToLeftJoinIterator(j)
    }
    return j, nil
}
```

### Per-Tenant Overrides

Pass `overrides.Interface` through handler chains. Always extract the tenant ID from context and pass it into override calls — never hardcode tenant-level behavior.

```go
func newHandler(o overrides.Interface, logger log.Logger) handlerFunc {
    return func(ctx context.Context, req *tempopb.SearchRequest) error {
        tenant, _ := user.ExtractOrgID(ctx)
        comb, err := newCombiner(req, o.LeftPadTraceIDs(tenant))
        // ...
    }
}
```

### Generics for Type-Safe Queues and Containers

Use generics to eliminate runtime type assertions in queue/container types:

```go
// Prefer this
type PriorityQueue[T Op] struct {
    queue queue[T]
}
func (pq *PriorityQueue[T]) Dequeue() T { return heap.Pop(&pq.queue).(T) }

// Over this — forces callers to cast
func (pq *PriorityQueue) Dequeue() Op { return heap.Pop(&pq.queue).(Op) }
```

### Instrumentation in Queues and Components

Embed Prometheus gauges directly in structs. Guard against nil for test-only instantiation:

```go
type PriorityQueue[T Op] struct {
    mu          sync.Mutex
    lengthGauge prometheus.Gauge
    queue       queue[T]
}

func (pq *PriorityQueue[T]) Enqueue(op T) {
    pq.mu.Lock()
    defer pq.mu.Unlock()
    heap.Push(&pq.queue, op)
    if pq.lengthGauge != nil {
        pq.lengthGauge.Inc()
    }
}
```

### Mock Structs for Interface Testing

Use `_` for unused parameters in mocks, `//nolint:all` to suppress linting on stubs, and thread-safe counters for assertions:

```go
type mockClient struct {
    mu    sync.Mutex
    count int
    resp  *tempopb.QueryRangeResponse
    err   error
}

//nolint:all
func (m *mockClient) QueryRange(_ string, _, _ int64, _ string) (*tempopb.QueryRangeResponse, error) {
    if m.err != nil {
        return nil, m.err
    }
    m.mu.Lock()
    defer m.mu.Unlock()
    m.count++
    return m.resp, nil
}
```

### Closure Helpers for Repeated Patterns

When the same struct initialization or operation repeats 3+ times in a function, extract it as a local closure rather than a package-level helper:

```go
doStringSummary := func(summary attributeSummary, scope backend.DedicatedColumnScope) {
    if summary.rowCount == 0 {
        return
    }
    for _, attr := range topN(settings.NumStringAttr, summary.attributes) {
        // ... process attr
    }
}

doStringSummary(s.spanSummary, backend.DedicatedColumnScopeSpan)
doStringSummary(s.resourceSummary, backend.DedicatedColumnScopeResource)
doStringSummary(s.eventSummary, backend.DedicatedColumnScopeEvent)
```

### Backward Compatibility When Adding Query Paths

When introducing a new query implementation, keep the old path available and gate the new one behind a hint or flag. The engine chooses which path to take — callers don't change.

```go
// Engine picks the path; both implementations live in parallel
if hints.Has("new") {
    return newFasterPath(ctx, fetcher)
}
return legacyPath(ctx, fetcher)
```

Document the old path's removal timeline in the PR and CHANGELOG.

### Config Defaults and Breaking Changes

Document default changes and breaking changes in `CHANGELOG.md` and in code comments at the point of change. When removing a config field, leave a comment explaining what replaced it:

```go
// QueryIngestersUntil removed in v2.7; use QueryBackendAfter instead.
// See CHANGELOG.md for migration guidance.
type SearchSharderConfig struct {
    QueryBackendAfter time.Duration
}
```
