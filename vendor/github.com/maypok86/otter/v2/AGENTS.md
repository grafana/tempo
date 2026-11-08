# otter

A concurrent in-memory cache for Go: W-TinyLFU eviction with an adaptive window,
variable expiration on a hierarchical timer wheel, loading and refresh with
singleflight, and lock-free reads.

## Layout

| Path | What lives there |
|---|---|
| `cache.go`, `options.go`, `entry.go`, `deletion.go`, `loader.go`, `*_calculator.go` | Public API and its documentation |
| `cache_impl.go` | The cache: read path, the API's writes (`set`, `doCompute`), loading and refresh, maintenance |
| `mutation.go` | Key mutations: `mutate` decides and applies a change of a key under its bucket lock, `publish` hands it to the policies and listeners |
| `policy.go`, `sketch.go` | W-TinyLFU eviction policy, hill climber, count-min sketch |
| `singleflight.go` | In-flight loads and refreshes |
| `task.go` | Write-buffer tasks replayed by maintenance |
| `persistence.go` | `SaveCacheTo` / `LoadCacheFrom` |
| `internal/hashmap` | Hash table with per-bucket locks and lock-free `Get` |
| `internal/lossy`, `internal/deque/queue` | Lossy striped read buffer, bounded MPSC write buffer |
| `internal/expiration` | Timer wheel (`Wheel`) |
| `internal/generated/node` | **Generated** node types. Never edit; change `cmd/generator` and run `make generate` |
| `cmd/generator` | Node generator (`main.go`, `node.go`, `manager.go`) |
| `benchmarks/` | Separate module: throughput, memory, eviction benchmarks and the hit-rate simulator |
| `lincheck/` | Separate module: checks histories of random concurrent operations for linearizability with porcupine |
| `docs/` | mkdocs site |

## Commands

```bash
make generate              # regenerate nodes and format (gennode + fmt); must leave no diff
make lint                  # golangci-lint (CI pins v2.13.2 on Go 1.25)
make test                  # race tests with coverage
go test -race ./...        # the whole suite runs in ~10 s
GOOS=linux GOARCH=386 go vet ./...  # CI also runs the suite on 386 (darwin has no 386 port)
cd lincheck && LINCHECK_DURATION=5m go test .  # linearizability check (default 2s); CI runs it weekly
```

golangci-lint must be built with a Go version at least as new as the one it analyzes; an
older v2.1.6 binary cannot load the standard library of Go 1.25+.

The `benchmarks` module may need `go mod tidy` before it builds; do that in a scratch copy
when the change is not about the benchmarks.

## Conventions

- Commits: conventional commits (`fix:`, `feat:`, `perf:`, `test:`, `chore:`, `docs:`), an
  imperative subject, and a body that explains the problem and the fix. No co-author trailers.
- One bug fix per commit, with a test that fails before the fix.
- Comments explain why, in full sentences; match the density of the surrounding code.
- Every change to the eviction policy is checked with the invariant validator
  (`validateCache` in `invariants_test.go`) and, if it can change hit rates, with the
  simulator (`/sim-compare`).
- Performance claims need interleaved `benchstat` runs (`-count >= 8`); a single run means
  nothing on a loaded machine.

## Agent guidance

Audit and review skills live in `.claude/skills` (linked from `.agents/skills` and
`.codex/skills`). Shared references are in `.claude/docs`:

- `concurrency.md` — locks, node lifecycle, buffers, in-place updates; read before touching
  `cache_impl.go`, `policy.go` or the generator.
- `design-decisions.md` — behavior that looks wrong but is intentional.
- `finding-taxonomy.md`, `audit-output.md`, `audit-rounds.md` — how audits classify,
  store and triage findings. Reports go to `.local/` (gitignored), never to the source tree.
