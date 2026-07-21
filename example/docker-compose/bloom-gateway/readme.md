## Bloom gateway

This example wires up the bloom gateway's full write path against real binaries: a
Kafka-based ingest pipeline (mirroring [../distributed](../distributed/)), block-builder
and backend-worker with `bloom_gateway_producer` enabled, and a standalone
`-target=bloom-gateway` service consuming the resulting events. It's the first place
the bloom gateway and its producer hooks run together outside unit tests.

Unlike the other examples here, there is no published image to fall back to -- the
bloom gateway only exists on this branch, so building locally is **required**, not
optional.

### Run it

1. Build the local image from the repo root (produces `grafana/tempo:latest`):

```console
make docker-tempo
```

2. Run the smoke test from this directory:

```console
./smoke.sh
```

`smoke.sh` brings the stack up, polls each service's `/metrics` until every assertion
below passes (or a per-assertion/overall timeout is hit), then tears the stack down
(`docker compose down -v`) whether it passed or failed. It builds nothing itself.

To instead run the stack interactively:

```console
docker compose up -d
docker compose logs -f bloom-gateway-0
# ...
docker compose down -v
```

### What's running

| Service | Role |
|---|---|
| `redpanda` | Kafka broker, shared by the ingest path and bloom-gateway events |
| `kafka-init` | pre-creates both topics with explicit partition counts + `max.message.bytes`, so no service races the broker's topic auto-create default |
| `minio` | S3-compatible object store (`tempo` bucket) |
| `distributor` | OTLP receiver, writes to the `tempo-ingest` Kafka topic |
| `alloy`, `k6-tracing` | synthetic trace load, same as [../distributed](../distributed/) |
| `live-store-0` | owns the Kafka partition ring the distributor writes through; the distributor can't produce to Kafka at all without at least one live-store instance registered here, even though this example has no query path |
| `block-builder-0` | consumes `tempo-ingest`, cuts durable blocks, publishes bloom-gateway `AddChunk` events |
| `backend-scheduler` | plans compaction and retention jobs from the object-store blocklist |
| `backend-worker-0` | executes those jobs; publishes `AddChunk` for compaction output and `Delete` when retention clears a block |
| `bloom-gateway-0` | `-target=bloom-gateway`; consumes the events above and serves the read-path gRPC API |

Query-path services (query-frontend/querier) and the visualization stack
(Grafana/Prometheus) are omitted: the smoke only needs the write path, and the
remaining services boot cleanly without them.

### Timing

Compaction, retention, and reconciliation are tuned much faster than any real
deployment (see `tempo.yaml`'s comments) so the smoke can observe a full
publish -> compact -> retain -> delete cycle in a few minutes instead of hours:

- `block_builder.consume_cycle_duration: 10s` -- a new durable block (and a bloom-gateway
  `AddChunk` publish) roughly every cycle.
- `backend_scheduler.provider.compaction.min_cycle_interval: 15s` and
  `backend_scheduler.provider.retention.interval: 20s` -- fast job scheduling.
- `backend_worker.compaction.compacted_block_retention: 1m` -- a compacted block is
  physically cleared (and its bloom-gateway `Delete` published) about a minute after
  compaction, not the default hour. `block_retention` is deliberately left at 10
  minutes -- comfortably longer than the smoke's own run -- so retention never marks a
  fresh, not-yet-compacted block for deletion out from under compaction; only genuinely
  compacted blocks drive the smoke's `Delete` events.
- `bloom_gateway.reconciliation.period: 30s` and `snapshot.interval: 1m`.

### Metrics

Each Tempo-image service exposes Prometheus metrics on its mapped host port:

| Service | URL |
|---|---|
| `distributor` | http://localhost:13200/metrics |
| `live-store-0` | http://localhost:13205/metrics |
| `block-builder-0` | http://localhost:13210/metrics |
| `backend-scheduler` | http://localhost:13220/metrics |
| `backend-worker-0` | http://localhost:13230/metrics |
| `bloom-gateway-0` | http://localhost:13240/metrics |

`smoke.sh` asserts, in order:

1. `block-builder-0`: `tempo_bloom_gateway_publishes_total{result="ok"} >= 1`
2. `bloom-gateway-0`: `tempo_bloom_gateway_blocks_live >= 1` and `entries_total > 0`
   (consumed and committed)
3. `backend-worker-0`: `tempo_bloom_gateway_publishes_total{result="ok"} >= 1`
   (a real compaction Add, requiring >= 2 blocks compacted together)
4. `bloom-gateway-0`: `tempo_bloom_gateway_deletes_total >= 1` (a real retention Delete
   round trip)
5. `bloom-gateway-0`: `tempo_bloom_gateway_reconciliation_repairs_total == 0`, checked on
   every poll of every other assertion, not just once -- this is the producer/consumer
   correctness canary. A nonzero value means a producer publish went missing and
   reconciliation had to patch the gateway's view itself, which is a hard failure here,
   not a working safety net.
6. Restarting the `bloom-gateway-0` container and confirming `blocks_live` recovers
   (via snapshot load or reconstruction) within 2 minutes.
