# Tempo: Bloom gateway

| Author(s) | [Oleg Kozliuk](mailto:oleg.kozliuk@grafana.com) |  |
| :---- | :---- | ----- |
| **Created** | May 6, 2026 |  |
| **Revised** | July 6, 2026 |  |
| **Status** | **In Discussion** |  |
| **Reviewer(s)** |  |  |
| **Informed** | Tempo team |  |
| **Product DNA(s)** |  |  |
| **Delivery Plan(s)** |  |  |
| **Readiness Review** |  |  |

# Bloom Gateway

> **Naming note.** The component keeps its working name for continuity. The leaf structure in this revision is a *fingerprint index*, not a bloom filter — the bloom variant is retained in § Alternatives considered. The membership-error model is the same one-sided one: no false negatives, tunable false positives.

## Problem statement

Trace-by-id queries in Tempo are dominated by per-block fan-out. To locate a trace today, the query-frontend shards the query into `⌈blocks / blocks_per_shard⌉` block-range jobs (~3,300 jobs at 100k blocks with the default `blocks_per_shard = 30`), and the queriers executing those jobs read **one bloom-filter shard per block** (`ShardKeyForTraceID`) — 100,000 bloom reads per query for a 100,000-block tenant. The per-block blooms are sized for ~1% FPR, so after the bloom phase the queriers still open ~1,000 false-candidate blocks whether or not the trace exists. Trace-by-id has **no default lookback** — a request without `start`/`end` (the common case for API-driven lookups) scans the full blocklist.

A fictional deployment approximating the current state:

- 100,000 blocks per tenant
- 200,000 trace IDs per block on average
- 20 × 10⁹ (trace, block) pairs total per tenant
- Per-block sharded bloom filters sized for ~1% FPR
- Default block retention 14 days

Goal: prune the set of blocks the query path must consult per trace-by-id query — jobs, bloom reads, and block opens — to ≤1% of the tenant's block count, without re-reading trace IDs from blocks in steady state.

## Goals

1. Prune the per-query candidate set to ≤1% of the tenant's block count. (The design overshoots: single-digit candidate blocks; see § Sizing.)
2. Tolerate gateway unavailability: the query-frontend falls back to the existing full-scan sharding. The gateway is a latency optimization, never a correctness dependency.
3. Bound per-instance memory at ≤ 20 GiB at the largest planned tenant, with headroom before resharding.
4. Update incrementally as block-builders and compaction jobs produce or retire blocks. **Steady state never re-reads trace IDs from blocks**; object-store re-reads happen only on reconstruction events (cold start without snapshot, ownership acquisition, `D`/`F`/seed change, reconciliation repair).
5. Cells of varying size scale by instance count alone, autoscaled on memory (§ Scaling and resharding); no per-cell parameter tuning in the common case.

## Design constraints

- **Rejection is authoritative.** A block reported as rejected for a given trace must be guaranteed not to contain the trace. Anything not rejected — candidate or unknown — is checked by the query path as today. The gateway never rejects on the basis of state it doesn't have.
- **Trace IDs are iterated only at block creation.** The component that writes a block (block-builder or a compaction job on a backend-worker) is the only place raw trace IDs are enumerated. Removal from the gateway must not require re-reading trace IDs from blocks — satisfied here by attributable entries and an in-memory sweep (§ Garbage collection).
- **A leaf is never served from partial state.** The completeness invariant (§ Leaf lifecycle) is the load-bearing safety property: a leaf answers queries only when it reflects every committed block and is receiving every live write. Partial state is not a degraded mode — it is a correctness violation, because the reject-all inference ("no match ⇒ trace is in no known block") is only sound over complete state.
- **Raw trace ID distribution is skewed and client-controlled.** The gateway partitions on `h = xxhash64(trace_id, seed)` with a **per-cell secret seed**. The hash recovers uniformity for ring routing and leaf placement; the seed prevents tenants from crafting trace IDs that concentrate load on a chosen leaf or instance (xxhash64 is trivially invertible without it). The seed is shared by gateway instances and query-frontends via configuration; changing it invalidates all state and is operationally a reshard (§ Changing D, F, or the seed). `h` is computed over the canonical **16-byte zero-padded** trace ID — the form vparquet stores and the query API parser produces; producer, gateway, and query-frontend must agree byte-for-byte (conformance-tested), else an entire class of IDs (e.g. 64-bit Jaeger-style) would be systematically mis-hashed into wrong rejections.

## Architecture overview

The gateway is a stateful service partitioned on `h = xxhash64(trace_id, seed)`. Query routing is decentralized: **query-frontends** (the only read clients) resolve leaf ownership against a dskit token ring shared over memberlist — the same ring/lifecycler primitives the backend-worker and distributor rings use. Writes from block-builders and compaction/retention jobs flow through a Kafka-compatible topic (WarpStream in Grafana Cloud cells) with 24h retention; producers have no relationship with the ring. The design presupposes the Kafka-ingest architecture (block-builder, backend-scheduler/worker); classic ingester-flush deployments are out of scope for v1.

```
        Query-frontend: trace-by-id for T, tenant X
                        |
                        |  h = xxhash64(T, seed)
                        |  leaf_idx = top D bits of h
                        |  ring lookup (client-side)
                        v
        +--------------------------------------+
        |  Gateway instance (cell-wide,        |
        |  multi-tenant)                       |
        |                                      |
        |  leaf directory[leaf_idx] ---------> |  nil / constructing -> empty
        |        |                             |  response (safe fallback)
        |        v                             |
        |  Leaf: sorted (fingerprint -> block) |  exact candidates on match;
        |  entries, one per (trace, block)     |  reject-all on no match
        |                                      |
        |  Tenant sets A_T (time-bucketed)     |  scope to tenant X + window
        +--------------------------------------+
                        |
                        v
        rejection set (block UUIDs); QF creates block jobs
        only for candidates and gateway-unknown blocks
```

**Hash ring.** Gateway instances register deterministic tokens into a memberlist-backed dskit ring (`BasicLifecycler`, RF = 1). A leaf maps to the single ring position `leaf_idx << (32 − D)`; the instance whose token range covers that position owns the leaf — exactly one owner by construction, with no token-quantization requirement. Query-frontends walk the ring client-side (ACTIVE instances only) and read directly from the owner. Tokens are derived deterministically from the instance ordinal via dskit's `SpreadMinimizingTokenGenerator` (present in the vendored dskit; single-zone capable; expects the `name-N` naming a StatefulSet provides). dskit's file-based `TokensPersistencyDelegate` exists but is deliberately unused: determinism is stronger — a replacement inherits its predecessor's ranges even when the volume is lost, with no file to protect. Producers do not participate in or observe the ring. There is no distributor tier.

**Leaves** form a single global address space of `2^D` slots addressed by the top `D` bits of `h`. The leaf directory is a flat array of `2^D` references — nil for leaves this instance does not own or has not finished constructing.

**Each leaf is a fingerprint map**: a sorted array of `(fingerprint, block reference)` entries, one entry per (trace, live block) pair whose hash lands in the leaf, across all tenants. The fingerprint is the next `F` bits of `h` below the leaf index (`F = 16` default; bits are disjoint from the leaf address and uniform by construction). The map fuses membership *and* attribution — there is no separate contributor list.

- **No false negatives, by construction.** If trace T is in live block B, entry `(fp(T) → B)` is present in T's leaf: it was inserted when B's Add was applied and is only removed after B is deleted. A lookup for T always matches its own entries, so a block containing T can never be rejected. This is the same one-sided guarantee a bloom provides.
- **False positives are surgical.** A different trace's entry can collide on `fp(T)` with probability `entries_per_leaf / 2^F` (~0.9% at the reference sizing), contributing ~1 spurious candidate block — not a whole contributor set.

**Per-tenant block sets `A_T`** live alongside the leaves: for each tenant, time-bucketed sets of block references (1h buckets; a block is a member of **every bucket its time range overlaps**, so window queries need no assumptions about block time spans). Tenant and time scoping live here, not in the leaves.

A query for trace T from tenant X visits exactly one leaf and binary-searches one fingerprint. On **no match**, every block of tenant X in the query window is rejected — T is in no block known to this instance. On a **match**, candidates = matched block references ∩ `A_X_window`; the rejection set is `A_X_window` minus candidates. Matches against other tenants' entries fall out in the intersection, so cross-tenant traffic does not inflate tenant X's candidates.

## Data model

The design is specified in terms of objects and references. Compact encodings are an implementation concern, isolated in § Representation notes so the sizing stays honest.

**Block object** — one per live block known to the instance: `{uuid, tenant, time_range, state: pending | live | deleted}`. Created when the first Add chunk for the block arrives, committed to `live` once all chunks have been applied (§ Write path), marked `deleted` on Delete — a terminal state: Adds for a `deleted` block are no-ops (no resurrection), keeping replayed and late chunks harmless. The **block registry** maps UUID → block object and is the single source of truth for "this instance may reject this block": only `live` blocks appear in `A_T`, and only `A_T` membership makes a block rejectable.

**Leaf object** — sorted array of `(fingerprint, block reference)` entries, ordered by `(fingerprint, reference)`. Insert-if-absent on the pair makes redelivery idempotent. Entries referencing `deleted` blocks are garbage until swept; they are harmless in the meantime (a deleted block is absent from `A_T`, so it can neither become a candidate nor be rejected).

**Tenant object** — the bucketed `A_T` sets plus per-tenant accounting.

**Leaf directory** — flat `2^D` array of leaf references. `nil` means "not serving": not owned, or owned but not yet constructed. A reference to an **empty leaf object** is distinct from nil: an owned, complete, empty leaf legitimately rejects everything in-window.

### Leaf lifecycle and the completeness invariant

Every leaf is in exactly one state, and all safety arguments reduce to this lifecycle:

1. **nil** — not served; incoming writes for this leaf are dropped. Queries receive an empty rejection set and fall back. Safe because absence never rejects.
2. **constructing** — not served, but **accumulating every live write**. A reconstruction task is concurrently backfilling history (§ Reconstruction). Queries still receive an empty rejection set.
3. **complete** — served. The leaf reflects every block committed to the registry that contributes to it, and receives every live write.

**Invariant: a leaf is served if and only if it is complete, and completeness requires that the leaf has received all writes continuously since its backfill covered history.** The transitions are one-way per episode: `nil → constructing` when a reconstruction task starts; `constructing → complete` when the backfill pass is done and topic replay has caught up past the backfill's capture point; `complete → nil` when ownership is shed (a single reference swap, which atomically stops both serving and write application — there is no window where writes stop but serving continues). A leaf acquired at runtime **must not** be populated from topic tail alone: the topic holds 24h, the leaf's history is older, and a leaf missing an old block's entries would wrongly reject that block via the reject-all path. Runtime acquisition always goes through `nil → constructing → complete`.

Three consequences worth stating because they make ring skew safe: a query mis-routed to a non-owner hits nil and falls back (safe); a query routed by a stale ring to a *previous* owner that has not yet shed the leaf hits a still-complete, still-written leaf (correct answers, slightly stale ownership — safe); and duplicate ownership under ring disagreement is harmless — consumption is global, so every complete leaf is correct no matter who else serves it, and the ring only picks *which* correct instance answers.

### Mutation modes

- **Lock-and-apply** — the only write-path mode. Per-leaf deltas from an Add are always small: a block contributes ~`items_per_block / 2^D` entries to any leaf (≈ 0.006 at reference sizing — in practice zero or one, even for the largest compacted blocks). Insert under the leaf lock; readers never observe a torn structure.
- **Copy, rewire, place back** — reconstruction and sweep-compaction. Build a replacement leaf object aside (from object-store backfill and/or by filtering the current object), while the live side of the lifecycle rules above keeps writes flowing to it; swap the directory reference when complete.

### Representation notes

Non-normative; the logic above never depends on these choices, but the § Sizing arithmetic does.

- Reference encoding: block references intern to per-instance monotonic 32-bit handles (never reclaimed; 2³² handles outlast any realistic cell lifetime; sweep passes may renumber wholesale if ever needed). Entry encoding v1: fixed-width `fp16 + handle32` = **6 B per entry**, directly binary-searchable. A packed variant (delta-coded fingerprints in anchored pages + bit-packed handles, ~3.5–4 B/entry) is the escape hatch if the memory budget presses; it trades search simplicity.
- `A_T` buckets: roaring bitmaps over handles.
- Leaf directory: `2^D` machine-word references = 256 MiB at `D = 25`, paid uniformly per instance for O(1) branchless lookup.
- Leaf arrays are slab-allocated with amortized growth; the sweep doubles as compaction.

## Query path

### Placement

The caller is the **query-frontend**, in the trace-by-id sharder, after tenant resolution and before job creation. Rationale: the QF already maintains the blocklist for sharding, already joins memberlist, and makes exactly one gateway request per query — versus per-job calls from queriers (~3,300 at 100k blocks) that would each carry a rejection set. Multi-tenant queries (`"a|b"`) are decomposed into per-tenant sub-requests upstream of the sharder, so the request schema stays single-tenant. The ingester/live-store leg of trace-by-id is untouched — the gateway covers backend blocks only.

The QF filters its block-metadata list against the rejection set and creates jobs **only for candidates and gateway-unknown blocks**. Mechanical note: today's trace-by-id jobs are equal-width hash ranges over the block-ID space (dynamic count targeting `blocks_per_shard`), with membership resolved querier-side against the blocklist — the filtered path therefore emits narrow ranges around each surviving block (or explicit block-ID jobs, as search sharding already does). Queriers are unchanged either way: they receive ordinary (narrower) jobs. A trace-by-id query whose trace doesn't exist typically produces zero block jobs; one that does exist produces jobs covering ~1–2 blocks instead of ~3,300 jobs covering all of them.

### Protocol

Request:

```
{
  tenant_id:  string,
  trace_id:   [16]byte,
  start_time: int64 (optional),  // nanoseconds, lower bound
  end_time:   int64 (optional),  // nanoseconds, upper bound
}
```

The request carries the **raw trace ID**; the gateway derives `h`, the leaf index, and the fingerprint from its own configured seed. The client's hash is used only for routing. This makes seed skew between QF and gateway degrade to fallback (the mis-routed instance finds nil and returns empty) instead of corrupting answers.

Response: `{version, flags, seed_fingerprint, rejected: delta-encoded sorted block UUIDs}`, over in-cell gRPC. The response is self-contained — UUIDs, not internal references — and the QF interprets it against the blocklist it already holds. The seed fingerprint (hash of the configured seed) lets the QF detect seed drift actively and alert, rather than inferring it from fallback rates.

**Why an enumerated rejection set.** Any compact "reject everything except…" or candidate-set encoding is unsafe under skew: the client cannot distinguish "rejected" from "unknown to the gateway" (consumer lag, warming instance), and a block unknown to the gateway must be searched. Enumerating rejections makes the unknown-block case safe by default. Block UUIDs (deterministic v5 from the block-builder, random v4 from compaction) are uniform in 128-bit space; sorted delta encoding yields ~14 B/UUID.

### Gateway-side execution

1. `leaf := directory[top D bits of h]`. Nil (unowned, constructing, or ring skew) → return empty rejection set; the QF proceeds unfiltered for this query.
2. Binary-search the leaf's entries for `fp = next F bits of h`.
3. `A_X_window` := union of tenant X's `A_T` buckets overlapping the query range (all buckets if unscoped).
4. **No match** → rejection set = `A_X_window`. **Match** → candidates = matched references ∩ `A_X_window`; rejection set = `A_X_window` minus candidates.
5. Resolve references to UUIDs via the registry, sort, delta-encode, respond.

### Time-range scoping

The optional range scopes the rejection set to blocks the QF would consider, shrinking the wire size; blocks outside the range are omitted (the QF's own time filter makes omission correct). Because blocks are members of every bucket they overlap, no bound on compacted-block time spans is assumed. Bucketing errors can only omit (block opened unnecessarily — safe) or over-include (rejecting a block the QF wasn't going to open — harmless).

### Client policy

- Timeout: single-digit-to-tens of ms budget, in-cell. On timeout, error, circuit-open, or non-ACTIVE owner: **no filtering for this query** — identical to today's behavior.
- Per-instance circuit breaker in the QF. No hedging is possible at RF = 1; the breaker plus fallback is the whole story.
- Response-size guard: a truncated rejection set is *safe* (unrejected blocks get searched) but forfeits pruning; the gateway prefers alerting over truncation (§ Guardrails).
- **Ring health timeout is the QF's own, not the gateway's (2026-07-16 amendment).** The gateway widens ITS OWN ring-reader heartbeat timeout well past a routine restart (§ Availability model) so ownership doesn't reshuffle on an ordinary reschedule; a query-frontend's ring reader is a separate, independently configured consumer of the same KV/gossip data — dskit's heartbeat timeout is a per-reader, in-process value, never replicated via the ring itself — and must keep its OWN short timeout for routing health. Inheriting the gateway's widened value here would make the QF keep routing to an instance that's actually down for most of a restart window, relying entirely on the per-query RPC timeout above to notice — still safe (bounded, falls back), just slower than a short QF-side timeout that notices sooner and skips straight to fallback.

### Cost

Per query at the gateway: one directory read + one binary search (~10 probes over a ~3.6 KiB array) + one bucketed-set subtraction + UUID serialization. Sub-ms except for large rejection sets, where serialization dominates.

End-to-end at the reference tenant: a nonexistent trace resolves to reject-all ~99.1% of the time (zero block jobs; ~0.9% of misses cost 1–2 spurious candidate jobs, and the per-block bloom then kills ~99% of those). An existing trace yields ~δ true candidate blocks — ≈1 for compacted data, small-single-digit for recent traces whose spans straddle block-builder cycles — plus ~0.01 spurious, versus ~3,300 jobs and 100k bloom reads today. Bloom-shard reads survive only for candidate blocks.

### Response size

| Tenant size | Time range | Rejection set wire size |
| :---- | :---- | :---- |
| 100k blocks | unscoped | ~1.4 MiB |
| 100k blocks | last 24h (~1k blocks in range) | ~14 KiB |
| 100k blocks | last 1h (~100 blocks in range) | ~1.4 KiB |

Unscoped queries are the **norm** for API-driven lookups (no default lookback), so the 1.4 MiB row is the common case at the largest tenant: ~140 MB/s of QF↔gateway traffic at 100 qps cell-wide, in-cell. Two optional mitigations, deliberately deferred: response compression, and memoizing the serialized reject-all body per (tenant, bucket-set, `A_T` generation) — the no-match path returns an identical body for every miss with the same window until the tenant's block set changes.

## Write path

Producers publish to a Kafka-compatible topic; they know nothing about instances, tokens, `D`, or the seed.

| Property | Value |
| :---- | :---- |
| Name | `tempo.bloom-gateway.events.<cell>` |
| Partitions | `K` (`K = 16` covers reference sizing; resize is correctness-neutral — application is order-independent, tombstones absorb reorder — but operator-driven and rare) |
| Partition key | `block_id` |
| Retention | 24h |
| Cleanup policy | delete |

Producer and topic setup reuse the `pkg/ingest` client plumbing (the block-builder's kgo stack); its topic auto-create default of 1000 partitions must be overridden to `K`.

Event types (versioned envelope; schema format — protobuf vs hand-rolled — is an open item):

- `AddChunk(block_id, tenant_id, time_range, chunk_index, chunk_count, trace_ids[])` — published by the block writer after the block is durable in object store. Payloads are chunked at ~200k trace IDs (~3.2 MiB) per message; the topic must permit ≥4 MiB messages (in-repo precedent: the ingest path runs 16 MB producer batches). A default-config compacted block (`max_compaction_objects = 6M`) is ~30 chunks. Chunks share the block's partition key; delivery order is **not** relied upon — application is order-independent (§ Event processing).
- `Delete(block_id)` — published by the **retention job** strictly **after** a successful `ClearBlock`, matching the Add's partition key. The ordering is load-bearing: Delete implies the block is physically gone, so any post-Delete rejection of it is vacuously safe — this is what makes late and replayed Adds harmless (§ Event processing). The price is a crash window: a worker dying between `ClearBlock` and publish loses the Delete *permanently*, because the next retention run no longer sees the deleted block (scheduler jobs are re-emitted per tick, not retried with state) — healed by § Reconciliation. This also covers the compaction sequence (compaction publishes Adds for outputs; the later cleanup of compacted inputs publishes their Deletes).

**Publish policy.** The writer retries within a bounded budget and then **drops and counts**. Publishing is not allowed to block block production indefinitely: the gateway is optional, and coupling ingest durability to its topic would violate Goal 2 in spirit. A dropped Add costs pruning for that block until § Reconciliation repairs it; it never costs correctness. Block-builder retries may republish the same deterministic block UUID with an identical payload — idempotent application makes this safe (this is why "UUIDs are never recycled" is *not* an assumption this design needs).

### Consumers

Every gateway instance is an independent consumer of **all `K` partitions** (manual partition assignment, as the block-builder already does; per-instance group id for offset observability). The fan-out is consumer-side: producers publish once; each instance filters by the leaves it owns. Offsets recorded in the local snapshot are authoritative for restart; broker-side commits exist only to power lag metrics. Each instance reads the full stream (~3.2 MiB/s at the reference churn of ~1 block/s cell-wide) and applies only its share.

### Event processing

A worker pool (16 at reference sizing) drains a bounded in-memory queue fed by the consumer. Per `AddChunk`:

1. Look up or create the block object (state `pending`) and its handle.
2. For each trace ID: compute `h`; if `directory[leaf_idx]` is nil, drop it; if constructing or complete, insert `(fp, handle)` into the leaf under its lock (insert-if-absent).
3. When every chunk index `0..chunk_count−1` has been applied (per-block bitmap — arrival order, worker-pool parallelism, and redelivery don't matter): insert the handle into every `A_T` bucket the block's time range overlaps (tenant lock), then commit the block object to `live` (registry lock). **Commit is unconditional on local entry count**: a block whose trace IDs all hashed to unowned leaves still enters the registry and `A_T` — that is what lets reject-all cover it, and it is safe by the completeness invariant: a query for a trace in that block is only ever answered by an instance serving that trace's leaf, which holds the block's entry (via the live write if the leaf was owned at apply time, via the reconstruction column pass if acquired later) — rejecting the block from anywhere else is vacuously correct. Without unconditional commit, an instance would omit `(1−1/N)^k` of a tenant's k-trace blocks from its rejection sets — a permanent pruning floor for low-volume tenants. A block whose payload fails validation is dropped and counted (producer bug), never committed.

A block is invisible to the read path until step 3: entries inserted earlier match queries but are subtracted out by the `A_T` intersection, exactly as absent blocks are. Every step is idempotent, so at-least-once redelivery and replay converge. Per-Add cost at reference sizing: ~200k hashes + ~25k sorted-array inserts per instance, ~30–60 ms wall — the pool idles in steady state and matters only during replay.

Per `Delete`: look up the block; remove its handle from its `A_T` buckets; mark the object `deleted` (it leaves the registry's live view). Leaf entries are left for the sweep. Delete for an unknown block is a no-op; `deleted` is terminal — subsequent Add chunks for the UUID are no-ops, so replay and late redelivery cannot resurrect a block whose entries the sweep already removed.

### Backpressure and memory pressure

The event queue is bounded (bytes); when full, the consumer pauses fetching and lag grows — lag is the alerting signal, not process memory. If instance memory reaches a hard cap, the instance stops consuming and alerts; it keeps serving its complete leaves (which only ever err toward staleness = missing rejections = fallback). It never partially applies an event and never evicts state.

## Garbage collection (sweep)

Deletes are exact but lazy: entries referencing deleted blocks remain in leaves until a **background in-memory sweep** removes them. The sweep walks the leaf directory incrementally (a slice of leaves per tick, copy-rewire-swap or in-place under the leaf lock), dropping entries whose block object is deleted and compacting the array.

- Cost: a full pass visits ~2.5 × 10⁹ entries per instance — seconds of one core; run continuously with a full-pass period of ~1–2h.
- Garbage bound: `delete_rate × entries_per_block_per_instance × entry_size × pass_period` ≈ 1 block/s × 25k × 6 B × 2h ≈ **~1 GiB** average — budgeted in § Sizing.
- The sweep is also where tenant deletion converges (drop `A_T`, entries swept on the next pass) and where allocation compaction happens.
- Tombstone reclamation rides the sweep: a `deleted` block object leaves the registry once a full pass confirms zero remaining leaf entries **and** its Delete is older than the replay horizon (topic retention + slack). Earlier reclamation would reopen resurrection-by-replay; never reclaiming grows the registry by ~86k tombstones/day at reference churn. Empty `A_T` buckets are dropped on the same pass.

**There is no steady-state object-store rebuild.** The bloom design's fill-rate decay, popcount monitoring, rebuild scheduler, and rebuild double-buffering have no equivalent here; the sweep replaces all of it with CPU.

## Reconstruction

All expensive state-building funnels through one mechanism: the **reconstruction queue**. Work items are leaf ranges (or single-block repairs from § Reconciliation). Producers of work: cold start without a usable snapshot; snapshot-load reconciliation (newly owned ranges are **enqueued, never topic-filled**); ownership acquisition during scale-in or replacement; `D`/`F`/seed changes; operator request.

Procedure, per batch of ranges:

1. Mark the ranges' leaves `constructing`. From this moment they accumulate all live writes (and remain unserved).
2. Enumerate live blocks from every tenant's **tenant index** (`index.json.gz`). Because the index lags reality by up to `blocklist_poll` (5 min default) plus build time, note the consumer's current position (the flip gate in step 4) and **rewind it** to (oldest index build time − margin) — a block created inside the staleness window would otherwise be missed by both the index read and the replay. Over-replay is free: application is idempotent. There is deliberately a single consumer per instance (matching the live-store's one-reader design): the rewind re-delivers ~10–15 min of stream to the instance's *complete* leaves too — a few GB at reference churn, under a minute of catch-up — during which their freshness lags; alerting must treat rewind-induced lag as expected (`reconstruction_queue_ranges > 0` annotates it).
3. For each live block, fetch its trace-ID column from object store (bounded concurrency, cell-wide rate limit; 404 = block deleted meanwhile, skip) and apply it as a synthetic Add restricted to the constructing ranges. **Prerequisite work item:** no projected column read exists in tempodb today — the only ID enumeration is compaction's full-row iterator (a 100k-block pass through it would be petabytes, not 280 GiB). The parquet layout supports the projection (TraceID is a dedicated leading column; ~2.8 MiB compressed per 200k IDs vs their 3.2 MiB raw on the topic); a trace-ID-projection reader covering every supported encoding (vparquet3/4/5) is new work this design depends on.
4. When the column pass is done and the consumer has caught back up past its noted pre-rewind position, flip the ranges' leaves to `complete` (per-leaf reference swaps).

Cost is dominated by the column pass, and — a direct consequence of hashing away locality — **is the same regardless of range width**: reconstructing a scale-in sliver reads the same ~100k columns as a full instance. Reference numbers: 100k × ~2.8 MiB ≈ 280 GiB per pass, ~6 min at fetch concurrency 16 against warm object store, ~10 min cold; concurrent passes share the cell's rate limit. This is the price of deferring backend-side hash-ordered artifacts (§ Alternatives considered #3); it is paid only on the events listed above, and the balance is deliberate — a single unpruned 100k-block tenant costs comparable object-store reads *per query*.

An instance is JOINING in the ring only during initial cold start; thereafter it stays ACTIVE while individual ranges reconstruct — constructing leaves answer empty, which is the universal safe fallback.

## Reconciliation

A periodic loop (per tenant, every few `blocklist_poll` cycles) diffs the tenant index against the block registry:

- In the index, not in the registry (and older than a grace of ~2× poll interval, to avoid racing in-flight Adds) → **repair-Add**: fetch the trace-ID column and apply it as a synthetic Add.
- In the registry (live), not in the index (past the same grace) → **synthesize Delete**.

Repair-Adds are **lag-gated**: while consumer lag exceeds the grace window, in-flight blocks merely *look* missing, and repairing them would mass-fetch columns that replay is about to deliver — a redundant object-store storm during exactly the recovery it should be helping. The loop skips repair-Adds until lag is back under threshold (Delete synthesis is unaffected — it is correct early and costs no reads), and repair fetches share the cell-wide reconstruction rate limit.

This heals every missed-event class — dropped publishes, lost chunks, missed Deletes, producer bugs — within one reconciliation period, and bounds `A_T` garbage that would otherwise inflate responses forever. Repair counters are first-class metrics: nonzero steady-state repair rates indicate a broken producer, not a working safety net.

## Snapshots

Optional local-disk snapshots make restarts cheap:

```
/var/lib/tempo/bloom-gateway/<instance>/snapshot.bin
```

Contents: format version; `D`, `F`, a seed fingerprint (hash of the seed, to detect config mismatch); instance tokens; per-partition resume offsets; the leaf directory with leaf payloads for **complete** leaves; constructing/pending ranges (re-enqueued on load); block registry; `A_T` sets. On load: any mismatch in format version, `D`, `F`, or seed fingerprint discards the snapshot (→ reconstruction). Owned-range reconciliation against the current ring: no-longer-owned leaves are dropped; newly owned ranges go to the reconstruction queue. Deploy corollary: a release that bumps the snapshot format (or changes `D`/`F`/seed) turns every restart in that rollout into a full reconstruction — budget ~6–10 min per instance, not the 2–4 min restore figure.

- **Consistency:** v1 pauses the worker pool between events for the duration of the serialization (~15 GiB to local NVMe ≈ 10–20 s; reads continue; the consumer buffers and lag blips; the consumption-progress liveness probe treats the pause as expected). Per-leaf copy-on-write snapshotting is the later refinement if the pause matters.
- **Save memory (2026-07-16 amendment).** Save streams complete leaves one at a time — collect owned indexes cheaply (a `Directory.Range` pass over indexes only, no cloning), then clone-and-serialize each leaf individually, discarding it immediately — rather than cloning every owned leaf into memory before writing anything. Peak *additional* memory during a save is therefore O(one leaf), never O(owned leaves). The earlier bulk-clone-then-serialize shape doubled live heap at production scale and OOM-killed the pod mid-assembly on every snapshot tick, with no snapshot file ever produced (incident: ~2.1M owned leaves / ~7.3 GiB of cloned leaf data landing on an already ~11.9 GiB heap against a 13.74 GiB GOMEMLIMIT / 16 GiB cgroup limit). A leaf that stops being complete between collection and cloning (an ownership change shedding it, most plausibly) is skipped outright — safe by the completeness invariant (§ Leaf lifecycle): a leaf missing from the snapshot is simply re-enqueued for reconstruction on the next load. The on-disk format is unchanged; the leaf count is written as a placeholder and patched once streaming finishes (skipped leaves shrink the true count below what was initially collected), and the trailing checksum is computed from the finalized file rather than incrementally — one extra bounded-memory sequential read over the body, comparable in size to the write itself, so it roughly DOUBLES the pause budgeted above (worse still on network-attached volumes like gp3 EBS, versus local NVMe).
- **Medium:** a persistent volume is strongly recommended. On ephemeral disk, every pod reschedule is a full reconstruction.
- **Cadence:** every 4–6h, well inside the 24h topic retention. Restart cost = load + replay: a 6h-old snapshot implies ~21k messages ≈ ~67 GB of replay — **~2–4 minutes** to live tail at reference churn. (Not "tens of seconds"; the number is honest and the availability model uses it.)

## Availability model

The gateway runs at **replication factor 1**: each leaf lives on exactly one instance. The cell tolerates instance loss through graceful degradation:

- **Rejection-only semantics** make missing state operationally identical to no gateway: the QF's fallback is today's behavior.
- **Snapshots** bound routine restarts to minutes; only snapshotless reconstruction is expensive.
- **Tempo's existing read path is the backstop** — the gateway's absence reproduces the status quo, not an outage. This stays true only if querier capacity is not quietly right-sized down once pruning lands: full-scan fallback needs pre-gateway fleet headroom. Retaining that headroom (or explicitly accepting degraded latency during gateway outages) is a standing capacity policy, not an assumption.

Degradation windows at 8 instances, uniform tokens (affected share ≈ the instance's token weight, across all tenants):

| Event | Affected queries | Duration |
| :---- | :---- | :---- |
| Restart with snapshot ≤ 6h old | ~1/8 | ~2–4 min |
| Snapshotless reconstruction | ~1/8 | ~6–10 min |
| Rolling deploy (one at a time) | ~1/8, rotating | ~2–4 min × N |
| Instance replacement (deterministic tokens) | ~1/8 | ~6–10 min |
| Scale-in by one instance | leaver's share, as slivers across survivors | ~15–30 min (rate-limited rehydration) |

Operational levers: rolling-restart pacing enforced by the **readiness gate** — a pod is ready only when ACTIVE in the ring, snapshot loaded, and zero `constructing` leaves, so a StatefulSet rollout cannot stack reconstructions (never more than one instance down); snapshot freshness (alert on `snapshot_age`); and the reconstruction rate limit. Tenants needing stricter availability get dedicated cells, not a higher replication factor.

**2026-07-16 amendment — graceful reschedule no longer costs survivors.** Before this amendment, EVERY graceful stop (not just an intentional scale-down) unconditionally left the ring (dskit's `LeaveOnStoppingDelegate`), so an ordinary Karpenter-consolidation reschedule of one instance made ALL survivors observe a "newly owned" range and each run a full coalesced reconstruction pass (§ Reconstruction: cost is independent of range width) — on top of the restarting instance's own cost already reflected in the table above. Under sustained node-consolidation pressure this compounded into a cascade (observed directly, 2026-07-16: repeated Karpenter evictions on a fleet already carrying an unrelated snapshot-save OOM bug, § Snapshots amendment, kept re-triggering this exact reconstruction storm on every eviction, since `karpenter.sh/do-not-disrupt` is not an option — Grafana-net policy requires workloads to tolerate consolidation, not opt out of it). § Graceful reschedule / restart (below) is the fix: a bare graceful stop now keeps the instance ACTIVE in the ring by default, so the table above's restart rows describe the ONLY cost of a routine reschedule — no survivor-side reconstruction, no cascade. Only an intentional removal, prepared via `POST /bloom-gateway/prepare-downscale` (§ Scale-in), still reassigns promptly; an unprepared scale-down now waits out the ring's heartbeat timeout instead — a deliberate trade, see § Graceful reschedule / restart's own runbook note below.

## Scaling and resharding

Two independent levers:

- **Instance count `N`** — per-instance memory; online; HPA-driven; the common case.
- **`D`, `F`, or the seed** — leaf addressing and fingerprint width; offline-ish; rare.

There is deliberately **no state transfer between instances and no backend-resident index** in v1 (§ Alternatives considered #3, #5): every ownership change is satisfied by rehydration — the reconstruction queue rebuilds acquired ranges from trace-ID columns and topic replay. The queue **coalesces** all pending ranges on an instance into a single column pass, so the cost of a topology change is one pass per affected instance regardless of how many slivers it acquired.

### Graceful degradation under growth

This is the one property the fingerprint map gives up relative to the bloom variant, so it is engineered back in at the system level rather than the data-structure level. A fixed-size bloom absorbs unplanned item growth by degrading precision at constant memory (1e-5 → ~1e-2 FPR at 2× load), and its marginal memory cost of growth is only the contributor list (~1.5 B/pair). The fingerprint map has no precision-for-memory trade at runtime: memory grows linearly at ~6 B/pair, full stop. The compensating chain, in order:

1. **Headroom + autoscaling.** The HPA holds per-instance steady-state memory at ~80% of the 20 GiB budget — the reference sizing's operating point. At realistic net-growth rates, that headroom is days of runway against a ~10-minute scale-out; growth never races the autoscaler unless a tenant is onboarded pathologically fast.
2. **Packed encoding** (§ Representation notes) is the stored lever: ~4 B/entry, −33% memory, deployable as a rolling per-instance reconstruction.
3. **The hard-cap safety valve.** An instance at its memory cap stops consuming and alerts (§ Backpressure and memory pressure). It keeps serving its complete leaves; only *freshness* degrades — new blocks go unpruned (fallback), existing blocks keep pruning, correctness is untouched. This is the map's analogue of bloom FPR creep: gradual, observable, reversible by scaling out.

### Autoscaling

Gateway instances run as a StatefulSet with tokens derived deterministically from the pod ordinal, which makes HPA scaling well-defined: scale-out appends ordinals, scale-in removes the highest, replacement reuses them.

- **Signal:** `tempo_bloom_gateway_steady_state_memory_bytes` — entries + directory + registry + `A_T` + garbage estimate, deliberately *excluding* reconstruction transients and queue buffers so rebuilds and replay don't flap the autoscaler. Target ~80% of budget.
- **Scale-out:** automatic, short stabilization (minutes). Effective capacity lands only after the new instance's reconstruction (~6–10 min), so the target threshold — not HPA reaction time — is the real lead-time control.
- **Scale-in:** expensive under RF = 1 (survivors rehydrate; see below) and therefore gated: long stabilization (≥ 24h), at most one instance per window, and only when cell-wide reconstruction queues are empty. In practice operator-approved rather than fully automatic.

### Scale-out (online, routine)

1. HPA (or operator) increments replicas. The new pod derives its tokens, registers as JOINING, and its owned ranges enter its reconstruction queue.
2. One coalesced column pass builds all owned ranges (~6–10 min, rate-limited); live writes accumulate into the constructing leaves throughout (§ Leaf lifecycle).
3. Ranges flip to complete; the instance goes ACTIVE; query-frontends start routing its share to it.
4. Previous owners shed the moved leaves as their ring view updates — a per-leaf `complete → nil` swap that stops serving and write-application atomically, reclaiming memory. Queries routed by stale rings in the interim land on still-complete leaves — safe.

Scaling is continuous (8 → 9 is routine); nothing about the topic, producers, or queriers changes.

### Scale-in (gated)

Withdrawing one instance scatters its token ranges as slivers across the survivors, and — because hashing removes read locality — each survivor's coalesced reconstruction pass reads the full column set. Cost model at reference sizing: one scale-in event ≈ `(N−1)` × 280 GiB of rate-limited object-store reads; at a 2–3 GB/s cell budget, ~15–30 min until full pruning coverage returns.

1. Confirm reconstruction queues are empty cell-wide and no other topology change is in flight.
2. **`POST /bloom-gateway/prepare-downscale` to the highest-ordinal instance first (2026-07-16 amendment, § Graceful reschedule / restart, below).** Since a graceful stop now keeps an instance in the ring by default, skipping this step means the instance's keyspace sits unreassigned until the ring's heartbeat timeout elapses, not immediately. Then stop it gracefully: it unregisters directly, with no LEAVING stopover (see § Graceful reschedule / restart's own note on why one isn't needed) — the same net removal from the ring as before this amendment. Its state is discarded — no handoff, per the rehydration-only rule above.
3. Survivors observe their newly owned ranges and enqueue them; each rehydrates in one coalesced pass. Until a range completes, its queries fall back (empty responses) — bounded, and visible in `owned_leaves{state}` and client fallback metrics.
4. Wait for queues to drain before any further topology change.

### Graceful reschedule / restart (routine, 2026-07-16 amendment)

Karpenter-style consolidation and ordinary pod rescheduling send a graceful SIGTERM, not a scale-down: the instance is coming back, usually on a different node, moments to minutes later. Treating this the same as an intentional removal (LEAVING, then unregister — Scale-in's own step 2, historically) made every such reschedule cost the WHOLE cell a coalesced reconstruction pass on the survivors (§ Reconstruction: "is the same regardless of range width"), even though nothing about the keyspace was actually supposed to change. `karpenter.sh/do-not-disrupt` is not an option (Grafana-net policy: workloads must tolerate consolidation, not opt out of it), so this is handled in the ring lifecycle instead:

1. A graceful stop, by default, does **not** unregister: the instance's ring entry stays exactly as it was — ACTIVE, same tokens — for the whole down window; only its heartbeat goes stale.
2. Ownership (`OwnedLeafRanges`, § Hash ring) is computed against a ring heartbeat timeout wide enough to comfortably outlast a normal restart, including a slow node move/EBS reattach (~5–10 min observed) — see § Availability model's own amendment note. Survivors' `runOwnershipWatch` therefore never observes a "newly owned" range for an ordinary reschedule and does no reconstruction work at all.
3. The returning pod re-registers with the SAME tokens (deterministic, like § Replacement below) and, if a snapshot exists, resumes from it almost immediately; only its OWN share degrades to fallback for the restart's actual duration — unchanged from § Availability model's existing restart rows. No other instance is ever affected.

An INTENTIONAL removal (scale-in, decommission) still needs the ring entry to actually leave promptly, so survivors pick up the vacated keyspace without waiting out the heartbeat-timeout window — that is what the prepare-downscale endpoint is for (§ Scale-in step 2, above): `POST /bloom-gateway/prepare-downscale` flips this one instance's `KeepInstanceInTheRingOnShutdown` flag off, so its very next graceful stop unregisters directly instead of staying put; `DELETE` reverses it. This does **not** go through a LEAVING stopover the way the pre-amendment delegate's transition did — `BasicLifecycler.stopping()` reads the flag natively and calls `unregisterInstance` itself, with no delegate involved at all (there is no way to reach LEAVING from a delegate outside the ring package once `stopping()` has begun, and none is needed: this package's `ringOp={ACTIVE}` treats LEAVING and absent identically, so the missing stopover changes nothing observable).

**Operational risk, worth stating plainly:** an operator who scales down without calling prepare-downscale first does not lose correctness — but reclaiming that instance's keyspace now takes as long as the (widened) ring heartbeat timeout plus the auto-forget window, not the near-immediate reassignment every stop gave before this amendment (when an unconditional LEAVING transition preceded every unregister). This is a deliberate trade (routine reschedules, the overwhelmingly common case, are now nearly free) — any scale-down runbook MUST call the endpoint first.

The POST/DELETE handlers themselves write the durable marker before flipping the in-process flag (downscale.go): a narrow race between a POST and an immediately-following graceful stop could, in principle, let that one stop cycle observe the pre-POST flag value and keep the instance in the ring for one extra cycle. This is deliberate and bounded, not a gap — crash-consistent, since the marker re-derives the flag at every startup (`checkShutdownMarker`) regardless of what any one in-memory flag happened to hold; and bounded either way by the ring heartbeat timeout / auto-forget window, the same backstop an entirely-skipped prepare-downscale call already relies on. Mirrors live-store's own marker-then-flag ordering for its partition ring.

### Replacement (crash or node loss)

The replacement pod reuses the lost ordinal and therefore the same tokens: no ranges move to survivors. It restores from snapshot if the volume survived, otherwise runs a full reconstruction, and returns to ACTIVE; the affected share falls back meanwhile (§ Availability model). Replacement costs what scale-out costs, never what scale-in costs.

Distinct from § Graceful reschedule / restart above: a crash or node loss never runs the lifecycler's own graceful-stopping delegate at all — there is no SIGTERM to catch — so the dead instance's ring entry simply goes stale where it was left (ACTIVE, same tokens) until either it restarts (self-heals via the same deterministic-token register path) or the ring's heartbeat timeout elapses and ownership reassigns, same as any other unhealthy-instance case. No prepare-downscale step applies here; this path was never gated by the graceful-stop delegate to begin with.

### Changing D, F, or the seed (offline-ish, rare)

Any of the three changes leaf identity or entry content wholesale: `D` moves entries between leaves, widening `F` needs hash bits that were never stored, and the seed changes every hash.

1. Deploy a parallel pool with the new parameters (or reconstruct rolling, instance by instance, accepting a longer mixed window).
2. Each new-parameter instance runs the standard reconstruction procedure while the old pool keeps serving.
3. Flip ring ownership to the new pool; retire the old.

Expected triggers: widen `F` if the measured miss-FP rate (`pairs / 2^(D+F)`, exported) exceeds budget — preferred over `D` because it leaves the directory alone; move `D` only if per-leaf arrays outgrow the intended object granularity; rotate the seed only on suspected compromise.

## Sizing

Reference cell: dominant tenant with 100k blocks, 200k trace IDs per block, 20 × 10⁹ (trace, block) pairs; `D = 25`, `F = 16`, 8 instances, v1 encoding (6 B/entry). Poisson spread holds because partitioning is on a seeded uniform hash.

**Stated assumptions** (validate during shadow rollout, § Rollout):

- Duplication factor δ (live blocks simultaneously containing the same trace) ≈ 1 in aggregate. Accepted by design decision: traces are partition-pinned, compaction dedupes, and superseded blocks live only ~1h of `compacted_block_retention`. Caveat: recent traces whose spans straddle block-builder cycles sit at small-single-digit δ until compaction converges — and they dominate real lookups, hence the age-stratified shadow measurement (§ Rollout). Entry count scales with δ.
- Cell-wide block churn ~1 block/s including compaction outputs (creation ≈ deletion in steady state).
- 14-day default block retention.

| Quantity | Value |
| :---- | :---- |
| (trace, block) pairs in the cell | 20 × 10⁹ |
| Pairs per instance (1/8 of keyspace) | 2.5 × 10⁹ |
| Total leaves / leaves per instance | 2²⁵ ≈ 33.6 × 10⁶ / ~4.2 × 10⁶ |
| Entries per leaf | 20 × 10⁹ / 2²⁵ ≈ **596** |
| Leaf object size (6 B/entry) | ~3.6 KiB |
| Miss-path FP rate `pairs / 2^(D+F)` | 20 × 10⁹ / 2⁴¹ ≈ **0.9%**, ~1 spurious block each |
| Candidates for an existing trace | δ + 0.009 ≈ **~1** |
| Entry storage per instance | 2.5 × 10⁹ × 6 B ≈ **14 GiB** |
| Leaf directory (2²⁵ references) | **256 MiB** |
| Block registry (100k blocks) | ~10 MiB |
| `A_T` (per tenant) | ~100 KiB |
| Sweep garbage allowance (2h pass) | ~1 GiB |
| Event-queue bound + transients | ~0.5 GiB |
| **Total per instance, steady state** | **~16 GiB** |

Derivations worth keeping visible:

- **`D + F` is the false-positive knob**: miss FP rate = `pairs / 2^(D+F)`. `F` widening is cheap precision (+1 B/entry per +8 bits: `F = 24` → 3.6 × 10⁻⁵ at ~17 GiB/instance); `D` sets per-leaf object granularity (array size, lock scope, insert memmove) and the 256 MiB directory floor. Note `D` is **not** pruning-constrained here — attribution is exact, so candidates don't scale with `2^D` the way contributor lists did.
- **`N` is the memory lever**: entry bytes per instance ≈ `pairs × entry_size / N` ≈ 112 GiB / N at reference size. The 20 GiB budget leaves ~4 GiB headroom at N = 8; the packed encoding (~4 B/entry → ~9.5 GiB) is the next lever before resharding.
- Adding tenants is marginal: an `A_T` set plus entries in proportion to their pairs. Small cells run few instances; the 256 MiB directory floor is the deliberate fixed cost of keeping `D` constant everywhere.
- CPU is not a sizing axis: steady state is ~30–60 ms of apply per block (~1/s), sub-ms queries, seconds-per-pass sweep. Provision cores for replay and reconstruction bursts, memory for steady state.

## Multi-tenant cells

One global pool, one ring, one topic; `tenant_id` is payload, never a routing key. Under the seeded uniform hash every tenant's data lands on every instance — which is the § Potential downsides isolation trade, and also what makes per-instance load uniform.

- Tenant separation lives exclusively in `A_T`: matches against other tenants' entries are filtered by the intersection, so cross-tenant traffic does not create candidates for tenant X (a structural improvement over the shared bloom, which mixed all tenants' bits in one filter). Other tenants' entries cost memory and contribute to the *cell-wide* miss-FP denominator only.
- Tenant lifecycle: first Add creates state lazily; deletion drops `A_T` and the sweep collects entries — no reconstruction required.
- **Guardrails (required, not optional):** per-tenant Add/Delete rate limits at the producers, per-tenant block-count ceilings, per-tenant query rate limits at the QF, and a response-size alert threshold. Truncation is safe but self-defeating; the operator answer to a tenant outgrowing the cell is a dedicated cell.

## Concurrency

Striped leaf locks (`leaf_idx mod 1024`) cover entry inserts and sweep compaction; a per-tenant lock covers `A_T`; a registry lock covers block commit. The partition consumer is single-threaded per partition and feeds the bounded queue; the worker pool applies in parallel — under a uniform hash, two workers rarely contend on a stripe. Reads take the leaf lock briefly (or read an immutable current version under copy-on-write; representation choice). Steady state is single-digit events/s per instance; the pool's throughput exists for replay and reconstruction.

## Failure handling

One defense covers everything: **a block can be rejected only if it is `live` in the responding instance's registry and present in `A_T`, and a leaf answers only when complete.** Every failure mode below degrades to "some blocks absent from rejection sets" — never to a wrong rejection.

- **Gateway unreachable / slow / circuit-open.** QF proceeds unfiltered for the affected queries. Other instances unaffected.
- **In-flight, lost, or retried Add.** Absent until applied; idempotent under redelivery; dropped publishes healed by reconciliation; chunks arriving after the block's Delete are no-ops (tombstone).
- **Topic unavailable.** Producers retry within budget, then drop and count; reconciliation backfills after recovery. The gateway's read path and existing state are unaffected.
- **Consumer lag.** Recent blocks missing from rejection sets; alerting SLO on lag; at RF = 1 there is no peer to steer to, so the lag SLO is load-bearing. A hung-but-alive process is caught by a liveness probe on consumption progress.
- **Offset loss / corruption.** Fall back to snapshot offsets; failing that, reconstruction.
- **Restart / cold start.** JOINING until snapshot load (or first reconstruction) completes; minutes per § Availability.
- **Permanent instance loss.** Replacement inherits deterministic tokens and reconstructs; affected share falls back meanwhile.
- **Snapshot unreadable / parameter mismatch.** Discard, reconstruct.
- **Reconstruction failure mid-flight.** Constructing leaves stay unserved; the queue retries; no partial state is ever served.
- **Blocklist/gateway skew.** A deleted block still in the QF's blocklist is opened (wasted IO, bounded by poll interval); a block in the gateway but not yet in the QF's blocklist is simply not iterated. Neither affects correctness.

## Metrics

Prefix `tempo_bloom_gateway_*` unless noted. Labels include `tenant` where relevant.

Gateway gauges: `owned_leaves{state="constructing|complete"}`, `blocks_live`, `entries_total`, `garbage_entries_estimate`, `memory_bytes{structure}`, `steady_state_memory_bytes` (the HPA signal, § Autoscaling), `tenant_blocks{tenant}`, `topic_lag_messages`, `topic_lag_bytes`, `reconstruction_queue_ranges`, `snapshot_age_seconds`, `snapshot_bytes`, `miss_fp_rate_estimate` (pairs / 2^(D+F)).

Gateway histograms/counters: `query_duration_seconds`, `query_candidates`, `response_bytes`, `queries_total{result="reject_all|candidates|empty"}`, `add_apply_duration_seconds`, `adds_total{status}`, `add_chunks_total`, `deletes_total`, `sweep_pass_duration_seconds`, `sweep_entries_removed_total`, `reconstruction_duration_seconds`, `reconstruction_blocks_total`, `reconciliation_repairs_total{kind="add|delete"}`, `snapshot_duration_seconds`.

Query-frontend (client-side): `bloom_gateway_client_requests_total{outcome="ok|timeout|error|circuit_open|not_active"}`, `bloom_gateway_client_duration_seconds`, `blocks_filtered_total` vs `blocks_searched_total` (measured pruning), and shadow-mode pruning/response-size stats. The completeness canary lives in vulture: `tempo_vulture_bloom_gateway_mismatch_total` (§ Rollout) — nonzero pages.

Producer-side: `bloom_gateway_publishes_total{result="ok|dropped|rate_limited"}` (retries occur inside the client's bounded delivery budget and are not separately observable), `bloom_gateway_publish_duration_seconds`.

## Rollout and validation

1. **Build-only.** Deploy the cell; consume, reconstruct, snapshot. Watch memory, lag, sweep, reconciliation-repair rates (should be ~0).
2. **Shadow.** QF queries the gateway on every trace-by-id but does not filter; it collects would-be pruning ratios and response sizes, and validates the δ and churn assumptions from § Sizing against reality — δ measured **stratified by trace age** (recent multi-cycle traces are the honest worst case). Correctness is validated by **Vulture**: extend tempo-vulture to read each of its traces twice — once normally, once with a per-request override that forces gateway filtering ahead of tenant enablement — and count any divergence as a mismatch. This is a zero-baseline paging signal; a bloom cross-check is not (per-block blooms fire spuriously at ~1% FPR, which over ~100k-block rejection sets means ~1,000 false mismatches per query with zero gateway bugs).
3. **Enable per tenant** via override, watching vulture mismatch (must stay 0), trace-by-id not-found rates (must not move), and fallback rates. Rollback is the same override: disabling the flag reverts the read path instantly; the component is removable without data migration.

Configuration surface: per-tenant enable flag (QF), per-request force-filter override (vulture/debug), cell seed, `D`/`F` (immutable without reshard), timeouts/breaker, snapshot path+interval, sweep period, reconstruction concurrency + cell rate limit, reconciliation period + lag gate, queue bound, publisher retry budget, ring heartbeat timeout + auto-forget timeout + unregister-on-shutdown default + shutdown-marker directory (§ Graceful reschedule / restart).

## Operational summary

| Operation | Cost | Frequency |
| :---- | :---- | :---- |
| Query | sub-ms + response serialization; 1 binary search + 1 set subtraction | per trace-by-id |
| Add (publish) | 1–30 chunked publishes, bounded retry | per new block |
| Add (consume) | ~30–60 ms/instance: ~200k hashes + ~25k entry inserts | per new block (~1/s) |
| Delete | ~ms: `A_T` removal + registry mark | per retired block |
| Sweep | continuous background; full pass ~seconds of CPU per 1–2h | always on |
| Reconciliation | tenant-index diff; repairs only on producer failure | every ~15 min |
| Snapshot | ~15 GiB to local disk, ~10–20 s apply-pause | every 4–6h |
| Restart (snapshot ≤ 6h) | ~2–4 min to live tail | routine |
| Reconstruction (cold / acquired ranges / reshard) | ~280 GiB column pass, ~6–10 min | rare, rate-limited |
| Scale-out (HPA) | one reconstruction pass on the new instance | growth-driven, automatic |
| Scale-in | one coalesced pass per survivor, ~15–30 min cell-wide | rare, gated |

## Alternatives considered

1. **Per-leaf bloom + contributor list** (the previous revision of this design). Same zero-false-negative guarantee; memory is equivalent, not better: the pruning regime forces `items_per_(block, leaf)` ≪ 1, so the contributor list costs ~one entry per (trace, block) pair on top of 24 bloom bits/pair — ~4.5 B/pair versus this design's 5–6 B (fixed) or ~4 B (packed). What the bloom buys is superior pure-membership precision per bit (1e-5 at 24 bits; the map needs `F ≈ 26` for that) and structure-level graceful degradation: an overfilled bloom loses precision instead of growing, so its marginal memory cost of unplanned growth is only the contributor list (~1.5 B/pair) versus the map's ~6 B/pair — this design compensates at the system level instead (§ Graceful degradation under growth). What it costs: candidates on the found path are the *entire* ~595-block contributor set (no attribution); deletes are impossible in-place, so fill-rate decay forces periodic source-driven rebuilds — and under a uniform hash all leaves cross the fill threshold together (~0.29 × retention ≈ day 4 at 14-day retention), per-leaf rebuild re-fetches each block's column ~200k times, and instance-scale rebuild needs either double-buffered arenas (breaks the 20 GiB budget) or recurring restart windows. Rejected: the rebuild machinery and the found-path candidate blast radius outweigh the membership-bit advantage.
2. **Bloom-only (no attribution).** ~3 B/pair, but a positive can't be attributed or subtracted, so every query for an *existing* trace — the common user-facing case — gets zero pruning. Fails Goal 1 where it matters most.
3. **Hash-ordered per-block artifact (+ optionally metadata-only topic).** Block writers also write a sorted `xxhash64` list per block (~1.6 MB per 200k IDs); reconstruction becomes ranged reads with bytes ∝ range share (a scale-in sliver: ~seconds and single-digit GiB instead of a 280 GiB pass), and a metadata-only topic eliminates payload chunking and consumer bandwidth. **Deferred, not rejected**: it introduces new backend state (block-format addition, producer coupling, backfill, seed baked into artifacts making rotation heavier). The design is forward-compatible — same hash, same events; the artifact slots into § Reconstruction and § Consumers without protocol changes. Revisit when reconstruction frequency or cell size makes column passes the binding constraint.
4. **State in object store, instances as caches** (Loki bloom-gateway shape). Solves RF-1 recovery and reconstruction wholesale, at the cost of a background build/compaction pipeline for the index files and higher read latency. A different, larger design; not pursued for v1.
5. **State handoff between instances** on planned scale events (leaver streams leaf objects to successors). Removes object-store reads for planned changes; useless for crashes; adds an inter-instance transfer protocol with offset-alignment subtleties. Possible later optimization on top of the reconstruction queue.
6. **Per-tenant partitioning / dedicated gateways per large tenant.** Restores isolation at the cost of per-tenant capacity management and hotspot exposure; rejected as the default (dedicated *cells* remain the isolation escape hatch).

## Potential downsides

- **No per-tenant isolation inside a cell.** Every instance serves every tenant; abnormal churn from one tenant occupies every instance's worker pool and bandwidth. Guardrails (§ Multi-tenant cells) are required, not optional.
- **Miss-path false positives are a rate, not a rarity.** ~0.9% of nonexistent-trace queries yield 1–2 spurious candidate jobs at `F = 16`. Expected extra work matches the bloom design (rare × huge vs. common × tiny); the tail is far better; the *rate* is visible in metrics and tunable via `F`.
- **Memory scales linearly with pairs, with no runtime precision-for-memory trade.** The bloom variant absorbed unplanned growth by degrading FPR at constant memory; the map must grow or scale out. Compensated by headroom + HPA + the stop-consuming safety valve (§ Graceful degradation under growth).
- **RF = 1 fallback storms.** Multiple simultaneous instance losses multiply the fallback share and spike object-store load across all tenants — against a querier fleet that must have retained full-scan headroom (§ Availability model). Mitigations: readiness-gated restart pacing, snapshot freshness alerts, fallback-rate monitoring.
- **Full-stream consumption per instance.** Every instance reads every Add chunk (~3.2 MiB/s at reference churn) and applies ~1/N of it. Fine at reference scale; the § Alternatives #3 artifact removes it if churn grows 10×.
- **Reconstruction reads are all-or-nothing.** Hashing removes locality, so any acquired range costs a full column pass until the artifact lands. Scale-in and crash-replacement inherit this cost.
- **Memory rides on the sweep.** Garbage between passes is bounded by churn × cadence; a stalled sweep shows up as memory growth, not correctness loss. Monitored.
- **256 MiB directory floor** per instance regardless of cell size — the price of constant `D` and O(1) lookup everywhere.
- **Unscoped rejection sets are large** (~1.4 MiB at 100k blocks) and unscoped is the norm; egress is in-cell and QF-side; memoization/compression are the deferred levers.
- **The topic is a write-path dependency** — softened to "bounded retry, drop, reconcile", so an outage degrades pruning freshness, never ingest or correctness.
- **QF adds one in-cell RPC** (~ms) to every trace-by-id query, gated by timeout + breaker.

## Open concerns

Working list; items here are open, everything previously listed has been folded into the design above.

1. **Event and response schema format** — protobuf vs hand-rolled framing; version-byte discipline; compression flag. Needs a decision before implementation.
2. **Deterministic token generation** — resolved to dskit's `SpreadMinimizingTokenGenerator` (present in the vendored version; single-zone capable; requires `name-N` instance naming; plugs into `BasicLifecyclerConfig.RingTokenGenerator`). Remaining: first in-repo adoption — validate `CanJoin` sequencing and the 512-token spread against the leaf-position math in a dev cell.
3. **Seed management** — distribution (runtime config vs secret) and the rotation runbook (parallel-pool reshard). Drift detection is handled: snapshot fingerprint on load, response fingerprint checked by the QF per request (§ Protocol).
4. **Snapshot consistency v2** — per-leaf copy-on-write to eliminate the apply-pause if the 10–20 s blip proves disruptive.
5. **Reject-all response memoization** — worth it if unscoped-query egress dominates in practice; needs an `A_T` generation counter.
6. **Shadow-phase validation targets** — confirm δ (stratified by trace age), cell churn, and entries-per-leaf spread against a production cell before freezing `F = 16` and the N = 8 sizing.
7. **Caller placement** — this revision commits to the query-frontend (single call per query, blocklist already present, queriers unchanged). If a deployment needs querier-side calls, the request would grow an optional block-ID-range field; not designed here.
