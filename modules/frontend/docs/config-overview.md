# Tempo Configuration Overview

This is an orientation map of Tempo's configuration. It explains what each top-level
configuration block controls so you can find the right area quickly. For exact key names,
nesting, and default values, request the "config-reference" doc, which lists every option.

## How Tempo configuration works

Tempo is a single binary that runs one or more components. The `target` option selects
which components run in a process (`all` runs everything; otherwise name a component such
as `distributor` or `querier`). All components read from the same configuration file, so
a block such as `storage` applies wherever it's relevant.

Configuration is set in a YAML file and may be overridden by command-line flags. Flag
names mirror the YAML path, for example `query_frontend.mcp_server.enabled` becomes
`--query-frontend.mcp-server.enabled`.

## Top-level configuration blocks

- `target`: Which component(s) this process runs. Use `all` for single-binary mode.
- `http_api_prefix`: Optional prefix applied to Tempo's HTTP API paths.
- `memory`: Process memory management, including automatic memory limit settings.
- `server`: HTTP and gRPC listen addresses, ports, TLS, and logging for the public API.
- `internal_server`: An optional separate server for internal endpoints.
- `distributor`: Receives incoming spans. Configures the OTLP, Jaeger, and Zipkin
  receivers and any forwarders.
- `querier`: Executes queries against the storage and live data, including any external
  query endpoints.
- `query_frontend`: Splits, queues, and shards incoming queries. Holds `search`,
  `trace_by_id`, and `metrics` query settings, and the `mcp_server` block.
- `metrics_generator`: Generates metrics from incoming spans, such as service graphs and
  span metrics, and remote-writes them. Configures processors and generator storage.
- `ingest`: Configures ingestion from Kafka-compatible queues.
- `block_builder`: Builds storage blocks from ingested data.
- `live_store` and `live_store_client`: Hold and serve recent trace data before it's
  flushed to long-term storage, and the client used to query it.
- `storage`: The trace backend (object storage such as S3, GCS, or Azure, or local),
  plus the write-ahead log, connection pool, and caching.
- `overrides`: Per-tenant limits and settings, such as ingestion rate limits, read limits,
  and metrics-generator options. Supports runtime-loaded user-configurable overrides.
- `memberlist`: Gossip-based ring membership shared by components that use rings.
- `usage_report`: Anonymous usage reporting settings.
- `cache`: Named cache backends (for example, Memcached or Redis) referenced elsewhere.
- `backend_scheduler` and `backend_scheduler_client`: Schedule background maintenance work
  such as compaction.
- `backend_worker`: Runs background maintenance jobs assigned by the scheduler.

## Finding exact options

Request the "config-reference" doc for the complete list of options with defaults. Most
installations set only 10 to 20 options; start from the examples and override what you
need.
