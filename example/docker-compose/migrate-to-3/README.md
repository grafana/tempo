# Migrate Tempo 2.x → 3.0 (microservices)

A self-contained Docker Compose environment to walk through and validate the
migration documented at
[Migrate from Tempo 2.x to 3.0](../../../docs/sources/tempo/set-up-for-tracing/setup-tempo/migrate-to-3.md).

It runs a Tempo 2.x microservices deployment and a Tempo 3.0 microservices
deployment side by side, both pointed at the same MinIO bucket, with Alloy as
the trace router so you can flip traffic between the two with a single env-var
change.

## What's in the box

| Profile  | Components                                                                                                                                              |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `v2`     | distributor, ingester, querier, query-frontend, compactor, metrics-generator (`grafana/tempo:2.9.2`)                                                    |
| `v3`     | distributor, live-store ×2, block-builder ×2, querier, query-frontend, backend-scheduler, backend-worker, metrics-generator + Redpanda (`grafana/tempo:3.0.0-rc.1`) |
| _shared_ | MinIO (object storage), Prometheus, Grafana, Alloy (trace router), `xk6-client-tracing` (load generator)                            |

| URL                                            | What                          |
| ---------------------------------------------- | ----------------------------- |
| http://localhost:3000                          | Grafana (anonymous admin)     |
| http://localhost:3200                          | Tempo 2.x query frontend      |
| http://localhost:3201                          | Tempo 3.0 query frontend      |
| http://localhost:9090                          | Prometheus                    |
| http://localhost:9001 (`tempo` / `supersecret`) | MinIO console                |
| http://localhost:8080                          | Redpanda console (v3 only)    |
| http://localhost:12345                         | Alloy UI                      |

## Prerequisites

- Docker Compose v2 (`docker compose ...`).
- ~3 GB of free RAM.

> **Note.** Both image tags are pinned in `docker-compose.yaml`
> (`x-tempo-v2-image`, `x-tempo-v3-image`). Bump them to test other versions.

---

## The playbook

This mirrors the structure of the migration guide. Each step lists the commands
to run plus what to look for before moving on.

> **Where to run the queries.** All `promql` snippets below are intended for
> Grafana Explore: open <http://localhost:3000>, click **Explore**, select the
> **Prometheus** datasource, and paste the query. TraceQL queries use the
> **Tempo 2.x** or **Tempo 3.0** datasources from the same Explore view.

### 1. Start Tempo 2.x and generate traffic

```bash
docker compose --profile v2 up -d
```

This brings up the shared infrastructure and the full v2 stack. `k6-tracing`
starts pushing spans into Alloy, which forwards them to `tempo-v2-distributor`
(the default `ALLOY_OTLP_UPSTREAM`).

Wait ~30 s, then confirm in Grafana Explore:

- **Tempo 2.x** datasource: a TraceQL search like `{}` returns traces.
- **Prometheus** datasource: `tempo_distributor_spans_received_total{job="tempo-v2"}`
  is greater than 0.

### 2. Bring up Tempo 3.0 alongside 2.x

```bash
docker compose --profile v3 up -d
```

This starts Redpanda, creates the `tempo-ingest` topic with 2 partitions, and
brings up every v3 component. Traffic is **still flowing to v2** — Alloy hasn't
been touched.

**First, verify that `compaction_disabled: true` is set in `tempo-v3.yaml`
overrides** (it already is in this example). If both compactors run against
the shared MinIO bucket at the same time, they can corrupt each other's work.

Then confirm v3 is healthy before moving on. Open Grafana at
<http://localhost:3000> → **Explore** → select the **Prometheus** datasource
and run:

```promql
# Both live-store replicas should report 1.
tempo_live_store_ready{job="tempo-v3"}

# Each block-builder should own one partition.
tempo_block_builder_owned_partitions
```

You can also tail the logs:

```bash
docker compose logs -f tempo-v3-live-store-0 tempo-v3-live-store-1 | grep -E 'ready to serve|fail'
```

> **Why does the live-store take 30 s to be ready?** With an empty Kafka topic
> the live-store has no record to use as a high-water mark, so it waits
> `live_store.readiness_max_wait` (30 s in this example, 30 m default) before
> giving up and going ready. Once you switch traffic in step 4, restarts are
> instant because there's data in the topic.

The `v3` profile also brings up [Tempo Vulture](https://grafana.com/docs/tempo/latest/operations/tempo-vulture/),
which pushes synthetic traces directly to the v3 distributor and reads them
back through the v3 query-frontend (trace-by-id, search tag, TraceQL, metrics).
The migration guide recommends letting Vulture run for 10–15 minutes. In
Grafana Explore (Prometheus):

```promql
# Trace-level issues (missing spans, incorrect results, etc.) — should stay 0.
# Note: this is a *Vec counter, so on a clean run no samples are emitted at
# all; using sum() correctly returns 0.
sum(tempo_vulture_trace_error_total)

# Generic transport/HTTP errors — should also stay 0.
tempo_vulture_error_total

# Should be increasing once Vulture has been running ~2 min, proving the
# end-to-end write + read path works.
tempo_vulture_trace_total
```

> Vulture writes a trace and waits a couple of minutes before reading it back,
> so `tempo_vulture_trace_total` stays at 0 for the first ~2 minutes. That's
> normal.

### 3. Validate the v3 deployment can read v2 blocks

Pick a trace ID **that's already been flushed to MinIO** and query it through
the **v3** frontend. "Already flushed" means older than v2's `max_block_duration`
(5 min in `tempo-v2.yaml`) — fresher traces still live in the v2 ingester's
memory, which v3 has no way to read.

Let the demo run for at least `max_block_duration` (5 min in `tempo-v2.yaml`)
plus ~1 min of flush overhead — so **6 min** total — after step 1, so v2 has
guaranteed to have flushed at least one block. Then in Grafana Explore:

1. Select the **Tempo 2.x** datasource. Set the **end** of the time range to
   `now - 6m` (older than `max_block_duration + flush overhead`, so the result
   excludes traces still only in the v2 ingester's memory). A start of
   `now - 30m` is a reasonable upper bound. Run a search like `{}` and copy
   any returned trace ID.
2. Switch to the **Tempo 3.0** datasource. Paste the trace ID into the **TraceID**
   query type and run it.

The trace should resolve with the same spans as on the v2 datasource, proving
the v3 querier successfully read a v2 block from shared object storage. (This
example uses RF=1 for simplicity; real 2.x deployments typically use RF=3, and
v3 transparently reads both.)

> **404 from v3 even though the block is in MinIO?** The v3 querier polls
> object storage on a fixed cycle (`storage.trace.blocklist_poll`, set to 1 m
> in `tempo-v3.yaml` for this demo; 5 m by default in real deployments).
> Newly flushed v2 blocks aren't visible to v3 until the next poll. Wait up to
> a poll interval and retry, or check the v3 querier logs for
> `"blocklist poll complete"` messages.
>
> In a real migration this is a non-issue — v2 has been running for hours or
> days, and every block has been polled many times before you bring up v3.

### 4. Switch traffic to Tempo 3.0

Repoint Alloy at the v3 distributor:

```bash
ALLOY_OTLP_UPSTREAM=tempo-v3-distributor:4317 docker compose up -d alloy
```

Within ~15 s, check that v3 is now ingesting. In Grafana Explore (Prometheus):

```promql
# v3 distributor writes to Kafka.
sum(rate(tempo_distributor_kafka_write_bytes_total[1m]))

# v3 block-builders are consuming.
sum by (instance) (rate(tempo_block_builder_fetch_records_total[1m]))

# v2 distributor receives nothing new (should be 0).
sum(rate(tempo_distributor_bytes_received_total{job="tempo-v2"}[1m]))
```

Recent traces from the new pipeline should also be queryable via Grafana's
**Tempo 3.0** datasource (Explore → select **Tempo 3.0** → search `{}`).

### 5. Drain the 2.x ingesters

With no new traffic arriving, the 2.x ingesters need to flush their in-memory
traces to MinIO. In Grafana Explore (Prometheus), watch these drift toward 0:

```promql
tempo_ingester_live_traces
tempo_ingester_flush_queue_length
```

With this example's `max_block_duration: 5m` and default `complete_block_timeout`,
the drain finishes in well under 20 minutes. (Real deployments take ~45 minutes
with default settings.)

### 6. Decommission v2 and re-enable v3 compaction

Once the ingester is drained, scale v2 down:

```bash
docker compose stop \
  tempo-v2-distributor \
  tempo-v2-ingester \
  tempo-v2-querier \
  tempo-v2-query-frontend \
  tempo-v2-compactor \
  tempo-v2-metrics-generator
```

Now remove the `compaction_disabled` override from `tempo-v3.yaml`. Delete this
block:

```yaml
    compaction:
      compaction_disabled: true
```

…and restart the v3 components that read overrides:

```bash
docker compose restart \
  tempo-v3-block-builder-0 tempo-v3-block-builder-1 \
  tempo-v3-live-store-0 tempo-v3-live-store-1 \
  tempo-v3-backend-scheduler tempo-v3-backend-worker \
  tempo-v3-querier tempo-v3-query-frontend
```

The backend-worker should now start compacting any blocks left behind by the
v2 ingesters plus new RF1 blocks from the block-builders. Watch:

```bash
docker compose logs -f tempo-v3-backend-worker | grep -i 'compact'
```

### 7. Verify the migration

Mirrors the **Verify the migration** section of the doc. Run these in Grafana
Explore against the **Prometheus** datasource (last column shows the expected
healthy state):

| Check                               | Query                                                                                                    | Expected     |
| ----------------------------------- | -------------------------------------------------------------------------------------------------------- | ------------ |
| Distributors writing to Kafka       | `sum(rate(tempo_distributor_kafka_write_bytes_total[1m]))`                                               | > 0          |
| Block-builders consuming, no errors | `sum by (instance) (rate(tempo_block_builder_fetch_records_total[1m]))` / `tempo_block_builder_fetch_errors_total` | > 0  /  0    |
| Live-stores ready                   | `tempo_live_store_ready`                                                                                 | 1            |
| No dropped records                  | `tempo_live_store_records_dropped_total`                                                                 | 0            |
| Kafka lag healthy                   | `tempo_ingest_group_partition_lag_seconds`                                                               | sub-second   |
| Historical queries work             | TraceQL search via the **Tempo 3.0** datasource (Explore → Tempo 3.0 → `{}`)                              | results return |
| Vulture clean (10–15 min run)       | `sum(tempo_vulture_trace_error_total)` and `tempo_vulture_trace_total`                                   | 0  /  rising |

---

## Roll back

To revert during the parallel-deployment phase (steps 2–5):

```bash
ALLOY_OTLP_UPSTREAM=tempo-v2-distributor:4317 docker compose up -d alloy
```

Traffic flips back to v2 instantly. Stop the v3 stack with
`docker compose --profile v3 down` if you want to abandon the migration.

After step 6 there's no in-place rollback — you'd have to restart the v2 stack
and replay everything from MinIO.

## Tear down

```bash
docker compose --profile v2 --profile v3 down -v
```

The `-v` flag removes MinIO and Alloy data volumes.

## Troubleshooting

- **v3 live-store stuck "starting"** → Check Redpanda is up and the
  `tempo-ingest` topic exists with 2 partitions: `docker compose exec redpanda
  rpk topic describe tempo-ingest`.
- **`grafana/tempo:latest` doesn't have 3.0 components** → The `latest` tag may
  lag behind `main` for unreleased majors. Pin to a 3.0 RC tag explicitly in
  `docker-compose.yaml`'s `x-tempo-v3-image` anchor.
- **v3 queries return nothing for old traces** → Confirm v2 used vParquet4+
  blocks (this example sets `storage.trace.block.version: vParquet4`).
