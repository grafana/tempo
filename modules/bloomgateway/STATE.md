# Bloom Gateway — Implementation State

Handoff for the next iteration. Companion to [DESIGN.md](./DESIGN.md), which
remains the authoritative design; this file records only **what exists in the
tree today**, how it was verified, and what is deliberately not done yet.

- **Branch:** `bloom-gateway`
- **Last updated:** 2026-07-10
- **Committed?** No. Everything below is uncommitted in the working tree.
- **Build/test status (verified directly, not inferred):**
  - `go build ./...` — clean
  - `go test -race -count=1 ./modules/bloomgateway/...` — pass (~10s)
  - `go vet`, `gofmt -l`, `golangci-lint run --new-from-rev=$(git merge-base main HEAD)` — all clean (0 issues)
  - `pkg/deltauuid`, `tempodb/encoding/...` (incl. vparquet5 `-run TraceID`), `cmd/tempo/...` — pass

## Scope delivered (this iteration)

The **core gateway service only** — the read/consume/maintain half of the
design. It builds and runs (`-target=bloom-gateway`) but cannot see real data
until the producer side (out of scope, below) publishes events.

Implemented, mapped to DESIGN.md:

- Hashing/addressing (`hash.go`): seeded xxhash64, top-`D`-bits leaf index,
  next-`F`-bits fingerprint; canonical 16-byte trace IDs.
- Leaf / directory / registry / tenant-`A_T` data structures.
- Kafka consumer (all K partitions, manual assignment) + fixed worker pool
  with a byte-bounded queue and an application-level applied-offset watermark.
- Idempotent event apply (`events.go`): AddChunk/Delete, the exactly-once
  completion guard, tombstone terminality, unconditional commit.
- Query gRPC server (`query.go`) + rejection-set wire codec (`pkg/deltauuid`).
- Background sweep, local-disk snapshots, reconstruction queue, reconciliation.
- dskit ring (RF=1, `SpreadMinimizingTokenGenerator`, `name-N` instances).
- Metrics (`metrics.go`), config (`config.go`), `services.Service`
  orchestrator (`bloomgateway.go`), and `cmd/tempo` wiring.
- vparquet5-only trace-ID projection reader (`tempodb/encoding/...`).
- Protobuf schema for both Kafka events and the query RPC.
- `.chloggen/bloom-gateway.yaml` + allowlist entry.

## Where things live

- `modules/bloomgateway/*.go` — the module (42 files incl. tests/benches).
  Orchestrator `bloomgateway.go`; ring `bloomgateway_ring.go`; data structures
  `leaf.go` `directory.go` `registry.go` `tenant.go` `types.go`; write path
  `events.go` `consumer.go` `worker.go`; read path `query.go`; maintenance
  `sweep.go` `snapshot.go` `reconstruction.go` `reconciliation.go`; `hash.go`
  `config.go` `metrics.go`. `regression_phasec_test.go` holds the Phase C
  regression guards.
- `pkg/tempopb/bloomgateway.proto` (+ generated `bloomgateway.pb.go`).
- `pkg/deltauuid/` — sorted-UUID delta codec (standalone; the future QF
  decoder can import it without importing this module).
- `tempodb/encoding/common/traceid_iterator.go`,
  `tempodb/encoding/traceid_projector.go` (optional-capability interface),
  `tempodb/encoding/vparquet5/block_traceid_projection.go` + one added method
  in `vparquet5/encoding.go`.
- `cmd/tempo/app/{config,modules,app}.go` — module wiring (target
  `bloom-gateway`, config prefix `bloom-gateway`, deps `{Common, Store,
  MemberlistKV}`, gRPC registration, `/bloom-gateway/ring` page, `readyHandler`
  branch). Intentionally NOT part of `SingleBinary`/`all` (StatefulSet-only).
- `go.mod`/`go.sum`/`vendor/` — one new dependency:
  `github.com/RoaringBitmap/roaring/v2` (+ transitive `mschoch/smat`) for
  `A_T`.

## Design decisions made during implementation

Open items DESIGN.md left for implementation, resolved here:

- **Schema format:** protobuf (gogoproto), one file for events + query RPC.
- **Seed:** `flagext.Secret`; hashing seed and the exposed `seed_fingerprint`
  are **domain-separated** (`xxhash64(seed‖0x00)` vs `‖0x01`) so disclosing
  the fingerprint never discloses the hashing seed.
- **Ring `CanJoin`:** `canJoinEnabled=false` (deterministic tokens; transient
  ring disagreement is already tolerated by the design). Uses
  `ring.NewInstanceRegisterDelegate` (the only delegate that honors the
  configured token generator — a subtle in-repo trap).
- **Kafka:** own `ingest.KafkaConfig` instance; local snapshot offsets are
  authoritative on restart, broker commits are lag-metrics only; byte-bounded
  queue blocks the fetch loop (never silent-drops).
- **Unsupported encodings:** a non-vparquet5 block encountered during
  reconstruction/reconciliation becomes `BlockLiveUnsupportedEncoding`, is
  **never added to `A_T`**, and is therefore never rejectable (always
  searched). Live Kafka Adds are encoding-agnostic, so this is purely a
  backfill concern.
- **v1 leaf storage** is `fp16`, so `Config.Validate` rejects `F > 16`.

Measured facts that differ from DESIGN.md's estimates (re-derived by
benchmarks; the doc's operational numbers should eventually be updated):

- deltauuid wire size converges to ~15–17 B/UUID at realistic densities, NOT
  the ~14 B the doc implies — delta-coding barely beats raw 16 B.
- vparquet5 TraceID column is stored uncompressed/undictionaried; the
  reconstruction bytes-read figure is likely higher than the doc's
  ~2.8 MiB/200k. The env-gated `BenchmarkTraceIDProjection` needs a real block
  to produce the number.

## Validation performed

Two independent Opus review passes on top of the coding:

1. Mid-implementation review of the invariant core (data structures +
   `events.go`) — 4 must-fix issues found and fixed with regression tests.
2. Phase C: adversarial design-conformance review of all 6 subsystems, then
   per-finding adversarial verification. The read path had **zero**
   rejection-authority or cross-tenant violations — the core safety math is
   sound. Six confirmed concurrency/lifecycle defects were fixed:

   1. **Sweep lost-update** (a false negative): a concurrent `InsertLive`
      between the sweep's clone and swap was dropped. Fixed with atomic
      `Directory.CompactLeaf` (read-modify-write under one stripe write lock).
      `TestSweep_ConcurrentInsertSurvivesCompaction` was confirmed to fail
      against the old code.
   2. **Snapshot race**: `buildSnapshotState` serialized live leaf pointers
      while reconstruction/reconciliation wrote them (`WorkerPool.Pause` only
      stops Kafka workers). Fixed by cloning under lock (`CloneLeaf`).
   3. **Reconstruction batch failure** stranded leaves in `LeafConstructing`
      (readiness gate never opens). Fixed with `Directory.Abandon` +
      defer-revert on error.
   4. **ctx-cancelled sweep** could reclaim a tombstone without a complete
      walk. Fixed by skipping reclamation unless the walk finished.
   5. **`runOwnershipWatch`** did an O(2^D) full directory walk every second.
      Fixed with a bounded range-set difference.
   6. **Metrics hygiene**: `CommitUnsupportedEncoding` chunk-progress leak +
      `adds_total{dropped}` double-count. Both fixed.

## NOT done — next-iteration work

### Out of scope this iteration (needs its own run)

The gateway is inert until these land; the proto/API is shaped so they need
**no schema change**.

- **Producer hooks.** Publish `AddChunk` after a block is durable
  (block-builder `tenant_store.go` `WriteBlock`; compaction output in
  backend-worker) and `Delete` after `ClearBlock` (`tempodb/retention.go`).
  Reuse `pkg/ingest` writer plumbing; enforce K=16 partitions and a
  broker-side `message.max.bytes` ≥ 4 MiB (the client 16 MB batch limit is not
  enough). Until this lands, every test synthesizes events by hand.
- **Query-frontend integration.** Ring-pooled client
  (`modules/livestore/client` is the template) + circuit breaker
  (`sony/gobreaker`, already vendored), rejection-set filtering in the
  trace-by-id sharder, per-tenant enable flag, shadow mode. The read ring is
  already exposed at `App` level for this.
- **Vulture** completeness canary (`tempo_vulture_bloom_gateway_mismatch_total`,
  dual-read).
- **vparquet3/vparquet4** trace-ID projection readers (v5-only today; other
  encodings ride the always-searched `LiveUnsupportedEncoding` path).
- **Per-tenant guardrails/overrides** (rate limits, block-count ceilings) —
  producer/QF-side, unbuildable without those halves.
- **docker/e2e** under `integration/`.

### Deferred should-fixes (safe to ship without; cheap follow-ups)

- `encoding.TraceIDProjector` has **no `context` parameter**, so
  reconstruction/reconciliation column reads use `context.Background()` and
  can't be cancelled by the caller's ctx. Adding `ctx` to the interface +
  vparquet5 impl is the fix.
- On partial reconstruction fetch failure (`stats.Failed > 0`) a batch still
  flips its leaves to complete. This is **safe** (a failed-fetch block never
  enters `A_T`, so it's not rejectable, and reconciliation repair-Adds it
  later) but does not match DESIGN.md's "constructing leaves stay unserved;
  the queue retries" wording. Consider re-enqueueing failed ranges.
- Test-coverage gaps flagged by Phase C: no direct test of
  `FetchAndApplyBlockColumn`'s backend-not-found (404 → skip) branch; no
  concurrent/duplicate-call test for `CommitUnsupportedEncoding`.

### Deferred design open items (DESIGN.md §Open concerns)

Snapshot v2 copy-on-write (eliminate the apply-pause), reject-all response
memoization, shadow-phase δ/churn validation against a real cell, response
compression. None are needed for correctness.

## How to verify / run

```
go build ./...
go test -race -count=1 ./modules/bloomgateway/... ./pkg/deltauuid/...
go test -race -count=1 -run TraceID ./tempodb/encoding/...
make gen-proto && make vendor-check        # proto/vendor drift (docker-based)
go run ./cmd/tempo -target=bloom-gateway -config.file=<cfg>   # smoke
```

Benchmarks (not CI-gated): `go test -bench=. -benchmem -run=NONE
./modules/bloomgateway/... ./pkg/deltauuid/...`. The vparquet5 projection
benchmark is env-gated on `VP5_BENCH_BLOCKID` and needs a real block.

## Gotchas for the next iterator

- **Post-workflow diagnostics can be stale.** During this work, tooling
  reported phantom errors (`PassStats redeclared`, missing imports, stray
  `zz_*` files) that did not match the real tree. Always confirm with
  `go build` / `grep` / `ls` before acting on them.
- **Large single-agent codegen tasks stall.** The invariant core (`events.go`)
  had to be written directly rather than by a subagent after repeated
  output-cap / API stalls. Keep per-agent scope small and mandate incremental
  writes.
- **`Leaf` is not concurrency-safe on its own** — every access goes through
  `Directory` under its striped locks. Anything running alongside live writes
  (sweep, snapshot) must use `CompactLeaf`/`CloneLeaf`, never the raw `Leaf()`
  accessor.
- **Rejection authority is the one inviolable rule:** a block may be rejected
  only if it is live in the registry AND in `A_T`, and a leaf answers only when
  complete. Every fix above was in service of never violating it.
