---
title: Configure Tempo
menuTitle: Configure
description: Learn about available options in Tempo and how to configure them.
weight: 400
---

# Configure Tempo

This document explains the configuration options for Tempo as well as the details of what they impact.

{{< admonition type="tip" >}}
Instructions for configuring Tempo data sources are available in the [Grafana Cloud](/docs/grafana-cloud/send-data/traces/) and [Grafana](/docs/grafana/latest/datasources/tempo/) documentation.
{{< /admonition >}}

The Tempo configuration options include:

- [Configure Tempo](#configure-tempo)
  - [Use environment variables in the configuration](#use-environment-variables-in-the-configuration)
  - [Deployment modes](#deployment-modes)
    - [Configuration by deployment mode](#configuration-by-deployment-mode)
  - [Server](#server)
  - [Memory](#memory)
  - [Distributor](#distributor)
    - [Set max attribute size to help control out of memory errors](#set-max-attribute-size-to-help-control-out-of-memory-errors)
    - [gRPC compression](#grpc-compression)
  - [Ingest](#ingest)
  - [Block-builder](#block-builder)
  - [Live-store](#live-store)
  - [Metrics-generator](#metrics-generator)
  - [Query-frontend](#query-frontend)
    - [Limit query size to improve performance and stability](#limit-query-size-to-improve-performance-and-stability)
      - [Limit the spans per spanset](#limit-the-spans-per-spanset)
      - [Cap the maximum query length](#cap-the-maximum-query-length)
  - [Querier](#querier)
  - [Backend scheduler](#backend-scheduler)
  - [Backend worker](#backend-worker)
  - [Storage](#storage)
    - [Local storage recommendations](#local-storage-recommendations)
    - [Hedged requests](#hedged-requests)
    - [Storage block configuration example](#storage-block-configuration-example)
  - [Memberlist](#memberlist)
  - [Configuration blocks](#configuration-blocks)
    - [Block](#block)
    - [Compaction](#compaction)
    - [Filter policies](#filter-policies)
      - [Filter policy](#filter-policy)
      - [Policy match](#policy-match)
      - [Examples](#examples)
    - [GRPC client](#grpc-client)
    - [KVStore](#kvstore)
    - [Search](#search)
    - [WAL](#wal)
  - [Overrides](#overrides)
    - [Ingestion limits](#ingestion-limits)
      - [Standard overrides](#standard-overrides)
      - [Tenant-specific overrides](#tenant-specific-overrides)
        - [Runtime overrides](#runtime-overrides)
        - [User-configurable overrides](#user-configurable-overrides)
      - [Ingestion rate strategy](#ingestion-rate-strategy)
        - [Examples](#examples-1)
  - [Usage-report](#usage-report)
    - [Configure usage-reporting](#configure-usage-reporting)
  - [Cache](#cache)
  - [Configure authentication](#configure-authentication)

Additionally, you can review [TLS](network/tls/) to configure the cluster components to communicate over TLS, or receive traces over TLS.

{{< admonition type="tip" >}}
Throughout the configuration, the `duration` values support the following units: `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`. See the [Go time package](https://pkg.go.dev/time#ParseDuration) for more information.
{{< /admonition >}}

## Use environment variables in the configuration

You can use environment variable references in the configuration file to set values that need to be configurable during deployment. To do this, pass `-config.expand-env=true` and use:

```
${VAR}
```

Where `VAR` is the name of the environment variable.

Each variable reference is replaced at startup by the value of the environment variable.
The replacement is case-sensitive and occurs before the YAML file is parsed.
References to undefined variables are replaced by empty strings unless you specify a default value or custom error text.

To specify a default value, use:

```
${VAR:-default_value}
```

where `default_value` is the value to use if the environment variable is undefined.

You can find more about other supported syntax [here](https://github.com/drone/envsubst/blob/master/readme.md).

## Deployment modes

Tempo supports two deployment modes: monolithic and microservices.

* Monolithic mode: The required components run in a single process using `-target=all`, which is the default. No Kafka is required.
* Microservices mode: Each component runs as a separate process with its own `-target` flag. For example, `-target=distributor` or `-target=querier`. This mode requires a Kafka-compatible system, such as Apache Kafka, Redpanda, or WarpStream, as the durable queue between the distributor and downstream components.

Refer to the [Deployment modes](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/plan/deployment-modes/) documentation for more information on when to use each mode.

### Configuration by deployment mode

Not all configuration blocks apply to both deployment modes.

In monolithic mode, the most important configuration blocks are:

| Config block | Description |
|---|---|
| `distributor` | Configure receivers (OTLP, Jaeger, Zipkin) and ingestion limits. |
| `storage` | Configure the backend used to flush and store trace blocks. |
| `metrics_generator` | Optional. Configure span-metrics and service-graph generation. |
| `query_frontend` | Configure query splitting, caching, and result streaming. |
| `overrides` | Set per-tenant rate limits and trace size limits. |

The `ingest` and `block_builder` blocks are only used in microservices mode.
Other blocks, including `live_store_client`, `backend_scheduler`, `backend_worker`, `memberlist`, and `cache`, apply in both modes but run in-process in monolithic mode.

For the complete mapping of all configuration blocks to deployment modes, refer to the [Components by deployment mode](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/#components-by-deployment-mode) table.


## Server

Tempo uses the server from `dskit/server`. For the full list of available server options, refer to the [dskit server configuration](https://github.com/grafana/dskit/blob/main/server/server.go#L66) and the [manifest](/docs/tempo/<TEMPO_VERSION>/configuration/manifest/).
For details on how server settings apply across deployment modes, refer to the [Deployment modes](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/) documentation.

Additional root-level options such as `target`, `shutdown_delay`, `auth_enabled`, `enable_go_runtime_metrics`, and `span_profiling` are available as [command-line flags](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/command-line-flags/).

```yaml
# Optional. Setting to true enables multitenancy and requires X-Scope-OrgID header on all requests.
[multitenancy_enabled: <bool> | default = false]

# Optional. String prefix for all http api endpoints. Must include beginning slash.
[http_api_prefix: <string>]

# Optional. Enables streaming query results over HTTP.
[stream_over_http_enabled: <bool> | default = false]

# Optional. Enables span profiling via otelpyroscope. When enabled, Tempo attaches pprof goroutine
# labels (span_id, span_name) to OTel spans and adds a pyroscope.profile.id attribute to root spans,
# enabling profile-to-trace correlation in Pyroscope.
# Requires OTEL_TRACES_EXPORTER, OTEL_EXPORTER_OTLP_ENDPOINT, or
# OTEL_EXPORTER_OTLP_TRACES_ENDPOINT to be set.
[span_profiling: <bool> | default = false]

server:
    # HTTP server listen host
    [http_listen_address: <string>]

    # HTTP server listen port
    [http_listen_port: <int> | default = 3200]

    # gRPC server listen host
    [grpc_listen_address: <string>]

    # gRPC server listen port
    [grpc_listen_port: <int> | default = 9095]

    # Register instrumentation handlers (/metrics, etc.)
    [register_instrumentation: <boolean> | default = true]

    # Timeout for graceful shutdowns
    [graceful_shutdown_timeout: <duration> | default = 30s]

    # Read timeout for HTTP server
    [http_server_read_timeout: <duration> | default = 30s]

    # Write timeout for HTTP server
    [http_server_write_timeout: <duration> | default = 30s]

    # Idle timeout for HTTP server
    [http_server_idle_timeout: <duration> | default = 120s]

    # Max gRPC message size that can be received
    # This value may need to be increased if you have large traces
    [grpc_server_max_recv_msg_size: <int> | default = 16777216]

    # Max gRPC message size that can be sent
    # This value may need to be increased if you have large traces
    [grpc_server_max_send_msg_size: <int> | default = 16777216]

    # Minimum time between pings from clients before the server sends a GOAWAY.
    # Tempo sets this lower than the dskit default to prevent GOAWAY errors
    # when there is little real traffic between components.
    [grpc_server_min_time_between_pings: <duration> | default = 10s]

    # Allow clients to send pings even when there are no active streams.
    # Tempo enables this by default to prevent GOAWAY errors during idle periods.
    [grpc_server_ping_without_stream_allowed: <bool> | default = true]
```

## Memory

Tempo supports automatic GOMEMLIMIT configuration using the [automemlimit](https://github.com/KimMachineGun/automemlimit) library.
When enabled, it automatically sets Go's memory limit based on available container (via CGroups) or system memory every 15 seconds.
For information on memory considerations across deployment modes, refer to the [Deployment modes](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/) documentation.

NOTE: enabling this will override value set in GOMEMLIMIT environment variable

```yaml
memory:
    # Enable automatic GOMEMLIMIT configuration based on cgroup/system memory.
    # When enabled, Tempo will automatically detect available memory from cgroups (v2 or v1)
    # or system memory and set GOMEMLIMIT accordingly.
    [automemlimit_enabled: <bool> | default = false]

    # Ratio of available memory to use for GOMEMLIMIT.
    # For example, 0.8 means 80% of available memory will be used for GOMEMLIMIT.
    [automemlimit_ratio: <float> | default = 0.8]
```

## Distributor

For more information on configuration options, refer to [this file](https://github.com/grafana/tempo/blob/main/modules/distributor/config.go).
For architectural details, refer to the [Distributor architecture](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/distributor/) documentation.

The distributor is the entry point for all trace data into Tempo.
It receives spans from instrumented applications, validates them against configured [ingestion limits](#ingestion-limits), and forwards them for processing.

How the distributor forwards data depends on the [deployment mode](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/):

- **Microservices mode**: The distributor shards traces by trace ID and writes them to [Kafka](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/kafka/). Kafka settings are configured in the [Ingest](#ingest) section. Downstream, [block-builders](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/block-builder/) and [live-stores](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/live-store/) consume from Kafka independently.
- **Monolithic mode** (`target: all`): The distributor pushes data in-process directly to the live-store and metrics-generator. No Kafka is required in this mode. It's suitable for local development, testing, and small installations.

The following configuration enables all available receivers with their default configuration. For a production deployment, enable only the receivers you need.
Additional documentation and more advanced configuration options are available in [the receiver README](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/README.md).

```yaml
# Distributor config block
distributor:

    # Ring configuration for the distributor.
    # Used only with the global ingestion rate strategy to discover healthy
    # peers and divide rate limits across the cluster.
    # The distributor watches the partition ring for Kafka routing but does
    # not run it. This ring is separate from the partition ring.
    ring:
      kvstore: <KVStore config>
        [store: <string> | default = memberlist]
        [prefix: <string> | default = "collectors/"]

      # Period at which to heartbeat to the ring.
      [heartbeat_period: <duration> | default = 5s]

      # The heartbeat timeout after which distributors are considered unhealthy in the ring.
      [heartbeat_timeout: <duration> | default = 5m]

      # Instance ID to register in the ring.
      [instance_id: <string> | default = os.Hostname()]

      # Name of the network interface to read the address from.
      [instance_interface_names: <list of string> | default = ["eth0", "en0"]]

      # IP address to advertise in the ring. Defaults to the address from instance_interface_names.
      [instance_addr: <string>]

      # Port to advertise in the ring.
      [instance_port: <int>]

      # Enable registering IPv6 addresses in the ring.
      [enable_inet6: <bool> | default = false]

    # Receiver configuration for different protocols.
    # The config is passed down to OpenTelemetry receivers.
    # By default, receivers listen to localhost and need a configured IP to
    # listen on an external interface.
    # For a production deployment, you should only enable the receivers you need.
    receivers:
        otlp:
            protocols:
                grpc:    # default localhost:4317
                http:    # default localhost:4318
        jaeger:
            protocols:
                thrift_http:
                grpc:
                thrift_binary:
                thrift_compact:
        zipkin:
        kafka:

    # Optional.
    # Configures forwarders that asynchronously replicate ingested traces
    # to specified endpoints. Forwarders work on per-tenant basis, so to
    # fully enable this feature, overrides configuration must also be updated.
    #
    # Note: Forwarders work asynchronously and can fail or decide not to forward
    # some traces. This feature works in a "best-effort" manner.
    forwarders:

        # Forwarder name. Must be unique within the list of forwarders.
        # This name can be referenced in the overrides configuration to
        # enable forwarder for a tenant.
      - name: <string>

        # The forwarder backend to use
        # Should be "otlpgrpc".
        backend: <string>

        # otlpgrpc configuration. Will be used only if value of backend is "otlpgrpc".
        otlpgrpc:

          # List of otlpgrpc compatible endpoints.
          endpoints: <list of string>
          tls:

            # Optional.
            # Disables TLS if set to true.
            [insecure: <boolean> | default = false]

            # Optional.
            # Path to the TLS certificate. This field must be set if insecure = false.
            [cert_file: <string | default = "">]

        # Optional.
        # Configures filtering in forwarder that lets you drop spans and span events using
        # the OpenTelemetry Transformation Language (OTTL) syntax. For detailed overview of
        # the OTTL syntax, please refer to the official Open Telemetry documentation.
        filter:
            traces:
                span: <list of string>
                spanevent: <list of string>
      - (repetition of above...)


    # Optional.
    # Enable to log every received span to help debug ingestion or calculate span error distributions using the logs.
    # This is not recommended for production environments
    log_received_spans:
        [enabled: <boolean> | default = false]
        [include_all_attributes: <boolean> | default = false]
        [filter_by_status_error: <boolean> | default = false]

    # Optional.
    # Enable to log every discarded span to help debug ingestion or calculate span error distributions using the logs.
    log_discarded_spans:
        [enabled: <boolean> | default = false]
        [include_all_attributes: <boolean> | default = false]
        [filter_by_status_error: <boolean> | default = false]

    # Optional.
    # Enable to metric every received span to help debug ingestion
    # This is not recommended for production environments
    metric_received_spans:
        [enabled: <boolean> | default = false]
        [root_only: <boolean> | default = false]

    # Optional.
    # Configures the time to retry after returned to the client when Tempo returns a GRPC ResourceExhausted.
    # Set this to `0` to disable retries on ResourceExhausted at cluster level.
    # per-tenant override `ingestion.retry_info_enabled` can also be used to disable it at tenent level.
    [retry_after_on_resource_exhausted: <duration> | default = 5s]

    # Optional
    # Configures the max size an attribute can be. Any key or value that exceeds this limit will be truncated before storing
    # Setting this parameter to '0' would disable this check against attribute size
    [max_attribute_bytes: <int> | default = 2048]

    # Optional.
    # Configures usage trackers in the distributor which expose metrics of ingested traffic grouped by configurable
    # attributes exposed on /usage_metrics.
    usage:
        cost_attribution:
            # Enables the "cost-attribution" usage tracker. Per-tenant attributes are configured in overrides.
            [enabled: <boolean> | default = false]
            # Maximum number of series per tenant.
            [max_cardinality: <int> | default = 10000]
            # Interval after which a series is considered stale and will be deleted from the registry.
            # Once a metrics series is deleted, it won't be emitted anymore, keeping active series low.
            [stale_duration: <duration> | default = 15m0s]
```

### Set max attribute size to help control out of memory errors

Tempo queriers can run out of memory when fetching traces that have spans with very large attributes.
This issue has been observed when trying to fetch a single trace using the [`tracebyID` endpoint](https://grafana.com/docs/tempo/<TEMPO_VERSION>/api_docs/#query).
While a trace might not have a lot of spans (roughly 500), it can have a larger size (approximately 250KB).
Some of the spans in that trace had attributes whose values were very large in size.

To avoid these out-of-memory crashes, use `max_attribute_bytes` to limit the maximum allowable size of any individual attribute.
Any key or values that exceed the configured limit are truncated before storing.
The default value is `2048`.

Use the `tempo_distributor_attributes_truncated_total` metric to track how many attributes are truncated.

For additional information, refer to [Troubleshoot out-of-memory errors](https://grafana.com/docs/tempo/<TEMPO_VERSION>/troubleshooting/out-of-memory-errors/).

### gRPC compression

By default, gRPC compression between all components uses `snappy`.
This provides a balanced approach to compression that works for most installations.

If you prefer a different balance of CPU/memory and bandwidth, consider disabling compression or using `zstd`.
Disabling compression may reduce CPU and memory usage on queriers and distributors, but can increase network traffic and [live-store](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/live-store/) data volume, especially in larger clusters.
Increased data can impact billing for Grafana Cloud.

You can configure the gRPC compression in the `live_store_client` and `querier.frontend_worker` gRPC clients.

To disable compression, remove `snappy` from the `grpc_compression` lines.

To set the compression, use `snappy` with the following settings:

```yaml
live_store_client:
  grpc_client_config:
    grpc_compression: "snappy"
querier:
  frontend_worker:
    grpc_client_config:
      grpc_compression: "snappy"
```

## Ingest

In distributed mode, the ingest configuration controls the Kafka-compatible system that Tempo uses as a durable write-ahead log for trace data.
Distributors write to Kafka, and downstream consumers (block-builders, live-stores, and metrics-generators) each consume from it independently.
In monolithic mode (`target: all`), distributors bypass Kafka entirely, so these settings don't apply.

For more information on configuration options, refer to [this file](https://github.com/grafana/tempo/blob/main/pkg/ingest/config.go).
For architectural details, refer to the [Kafka architecture](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/kafka/) documentation.

```yaml
# Ingest configuration block
ingest:

    kafka:
        # The Kafka backend address.
        [address: <string> | default = "localhost:9092"]

        # The Kafka topic name.
        [topic: <string>]

        # The Kafka client ID.
        [client_id: <string>]

        # The maximum time allowed to open a connection to a Kafka broker.
        [dial_timeout: <duration> | default = 2s]

        # How long to wait for an incoming write request to be successfully committed to the Kafka backend.
        [write_timeout: <duration> | default = 10s]

        # The SASL username for authentication.
        [sasl_username: <string>]

        # The SASL password for authentication.
        [sasl_password: <string>]

        # Enable auto-creation of Kafka topic if it doesn't exist.
        [auto_create_topic_enabled: <bool> | default = true]

        # Default number of partitions for auto-created topics.
        [auto_create_topic_default_partitions: <int> | default = 1000]

        # The maximum size of a Kafka record data that should be generated by the producer.
        # An incoming write request larger than this size is split into multiple Kafka records.
        [producer_max_record_size_bytes: <int> | default = 15983616]

        # The maximum size of (uncompressed) buffered and unacknowledged produced records sent to Kafka.
        # The produce request fails once this limit is reached. This limit is per Kafka client. 0 to disable.
        [producer_max_buffered_bytes: <int> | default = 1073741824]

        # The consumer group used by the consumer to track the last consumed offset.
        # If the value contains the '<partition>' placeholder, it is replaced with the partition ID.
        # When empty (recommended), Tempo uses the instance ID to guarantee uniqueness.
        [consumer_group: <string>]

        # How frequently a consumer should commit the consumed offset to Kafka.
        # The last committed offset is used at startup to continue consumption from where it was left.
        [consumer_group_offset_commit_interval: <duration> | default = 1s]

        # How long to retry a failed request to get the last produced offset.
        [last_produced_offset_retry_timeout: <duration> | default = 10s]

        # The best-effort maximum lag a consumer tries to achieve at startup.
        # Set both target and max consumer lag to 0 to disable waiting at startup.
        [target_consumer_lag_at_startup: <duration> | default = 2s]

        # The guaranteed maximum lag before a consumer is considered to have caught up
        # at startup, becomes ACTIVE in the hash ring, and passes the readiness check.
        # Set both target and max consumer lag to 0 to disable waiting at startup.
        [max_consumer_lag_at_startup: <duration> | default = 15s]

        # Disable Kafka client metrics reporting to the broker.
        # When false (default), the Kafka client reports its internal metrics to the broker.
        [disable_kafka_telemetry: <bool> | default = false]

        # How often the consumer group lag metric is updated. Set to 0 to disable.
        [consumer_group_lag_metric_update_interval: <duration> | default = 1m]
```

## Block-builder

The block-builder consumes trace data from Kafka, organizes it into Parquet blocks, and flushes them to object storage for long-term retention and querying.

For more information on configuration options, refer to [this file](https://github.com/grafana/tempo/blob/main/modules/blockbuilder/config.go).
For architectural details, refer to the [Block-builder architecture](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/block-builder/) documentation.

```yaml
# Block-builder configuration block
block_builder:

    # Instance id.
    [instance_id: <string> | default = <hostname>]

    # Map of instance names to partition IDs assigned to each block builder.
    [assigned_partitions: <map of string to list of int>]

    # Number of partitions assigned to this block builder.
    [partitions_per_instance: <int> | default = 0]

    # Interval between consumption cycles.
    [consume_cycle_duration: <duration> | default = 5m]

    # Maximum number of bytes that can be consumed in a single cycle. 0 to disable.
    [max_consuming_bytes: <uint64> | default = 5000000000]

    # Block configuration for the block builder.
    block: <Block config>
      [max_block_bytes: <uint64> | default = 20971520]

    # Write ahead log configuration for the block builder.
    wal: <WAL config>
      [path: <string> | default = "/var/tempo/block-builder/traces"]
```

## Live-store

The live-store holds recent trace data in memory and serves queries during the window between ingestion and block availability in object storage.
It periodically flushes traces to a local WAL in Parquet format for TraceQL search and metrics queries.

For more information on configuration options, refer to [this file](https://github.com/grafana/tempo/blob/main/modules/livestore/config.go).
For architectural details, refer to the [Live-store architecture](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/live-store/) documentation.

```yaml
# Live-store configuration block
live_store:

    # Ring configuration for the live-store.
    ring: <ring config>

    # How often the partition reader commits to Kafka. 0s means synchronous commits.
    [commit_interval: <duration> | default = 5s]

    # How often to sweep all tenants and move traces from live -> wal -> completed blocks.
    [flush_check_period: <duration> | default = 5s]

    # Interval for periodic cleanup work, and the maximum time to wait for background processes during shutdown.
    [flush_op_timeout: <duration> | default = 5m]

    # Amount of time a trace must be idle before flushing it to the WAL.
    [max_trace_idle: <duration> | default = 5s]

    # Amount of time after which a trace is flushed to the WAL regardless of idle period.
    [max_trace_live: <duration> | default = 30s]

    # Maximum size of live traces in bytes. 0 to disable.
    [max_live_traces_bytes: <uint64> | default = 250000000]

    # Maximum size of a block before cutting it.
    [max_block_bytes: <uint64> | default = 52428800]

    # Maximum length of time before cutting a block.
    [max_block_duration: <duration> | default = 30s]

    # Number of concurrent blocks to query for metrics.
    [query_block_concurrency: <uint> | default = 10]

    # Number of concurrent block completion operations.
    [complete_block_concurrency: <int> | default = 2]

    # Duration to keep blocks in the live-store after they have been completed.
    [complete_block_timeout: <duration> | default = 20m]

    # Target consumer lag threshold before the live-store is considered ready to serve queries.
    # Set to 0 to disable readiness waiting.
    [readiness_target_lag: <duration> | default = 0]

    # Maximum time to wait for catching up at startup. Only used if readiness_target_lag > 0.
    [readiness_max_wait: <duration> | default = 30m]

    # Fail on search and metrics requests if lag is high and live-store cannot guarantee completeness.
    [fail_on_high_lag: <bool> | default = false]

    # Remove partition owner from the ring on shutdown.
    [remove_owner_on_shutdown: <bool> | default = true]

    # Path to the shutdown marker directory.
    [shutdown_marker_dir: <string> | default = "/var/tempo/live-store/shutdown-marker"]

    # Partition ring configuration for the live-store.
    partition_ring:
      # Backend storage used for the partition ring.
      kvstore: <KVStore config>

      # Minimum number of owners to wait before a PENDING partition is switched to ACTIVE.
      [min_partition_owners_count: <int> | default = 1]

      # How long the minimum number of owners are enforced before a PENDING partition
      # is switched to ACTIVE.
      [min_partition_owners_duration: <duration> | default = 10s]

      # How long to wait before an INACTIVE partition is eligible for deletion.
      # The partition is deleted only if it has no owners registered. 0 to disable.
      [delete_inactive_partition_after: <duration> | default = 13h]

    # Metrics query tuning configuration.
    metrics:
      # Time overlap cutoff ratio for metrics queries (0.0-1.0). Controls whether trace-level
      # timestamp columns are loaded. Lower values skip columns more often, reducing I/O but
      # increasing spans evaluated. 1.0 always loads columns, 0.0 never does.
      [time_overlap_cutoff: <float> | default = 0.2]

    # Write ahead log configuration for the live-store.
    wal:
      [path: <string> | default = "/var/tempo/live-store/traces"]
```

## Metrics-generator

The metrics-generator processes spans and writes metrics using the Prometheus remote write protocol.
For more information on the metrics-generator, refer to the [Metrics-generator documentation](../metrics-from-traces/metrics-generator/).

Metrics-generator processors are disabled by default. To enable it for a specific tenant, set `metrics_generator.processors` in the [overrides](#overrides) section.

For more information on configuration options, refer to [this file](https://github.com/grafana/tempo/blob/main/modules/generator/config.go).
For architectural details, refer to the [Metrics-generator architecture](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/metrics-generator/) documentation.

{{< admonition type="note" >}}
If you want to enable metrics-generator for your Grafana Cloud account, refer to the [Metrics-generator in Grafana Cloud](https://grafana.com/docs/grafana-cloud/send-data/traces/metrics-generator/) documentation.
{{< /admonition >}}

Spans with end times older than `metrics_ingestion_time_range_slack` are excluded from metrics generation.
The default value is 30 seconds, which means spans that ended more than 30 seconds ago are discarded.

```yaml
# Metrics-generator configuration block
metrics_generator:

    # Ring configuration
    ring:
      kvstore: <KVStore config>
        [store: <string> | default = memberlist]
        [prefix: <string> | default = "collectors/"]

      # Period at which to heartbeat the instance
      # 0 disables heartbeat altogether
      [heartbeat_period: <duration> | default = 5s]

      # The heartbeat timeout, after which, the instance is skipped.
      # 0 disables timeout.
      [heartbeat_timeout: <duration> | default = 1m]

      # Our Instance ID to register as in the ring.
      [instance_id: <string> | default = os.Hostname()]

      # Name of the network interface to read address from.
      [instance_interface_names: <list of string> | default = ["eth0", "en0"] ]

      # Our advertised IP address in the ring, (useful if the local ip =/= the external ip)
      # Will default to the configured `instance_id` ip address,
      # if unset, will fallback to ip reported by `instance_interface_names`
      # (Affected by `enable_inet6`)
      [instance_addr: <string> | default = auto(instance_id, instance_interface_names)]

      # Our advertised port in the ring
      # Defaults to the configured GRPC listing port
      [instance_port: <int> | default = auto(listen_port)]

      # Enables the registering of ipv6 addresses in the ring.
      [enable_inet6: <bool> | default = false]

    # Processor-specific configuration
    processor:

        service_graphs:

            # Wait is the value to wait for an edge to be completed.
            [wait: <duration> | default = 10s]

            # MaxItems is the amount of edges that will be stored in the store.
            [max_items: <int> | default = 10000]

            # Workers is the amount of workers that will be used to process the edges
            [workers: <int> | default = 10]

            # Buckets for the latency histogram in seconds.
            [histogram_buckets: <list of float> | default = 0.1, 0.2, 0.4, 0.8, 1.6, 3.2, 6.4, 12.8]

            # Additional dimensions to add to the metrics. Dimensions are searched for in the
            # resource and span attributes and are added to the metrics if present.
            [dimensions: <list of string>]

            # Prefix additional dimensions with "client_" and "_server". Adds two labels
            # per additional dimension instead of one.
            [enable_client_server_prefix: <bool> | default = false]

            # If enabled another histogram will be produced for interactions over messaging systems middlewares
            # If this feature is relevant over long time ranges (high latencies) - consider increasing
            # `wait` value for this processor.
            [enable_messaging_system_latency_histogram: <bool> | default = false]

            # Attributes that will be used to create a peer edge
            # Attributes are searched in the order they are provided
            # See: https://pkg.go.dev/go.opentelemetry.io/otel/semconv/v1.25.0
            # Example: ["peer.service", "db.name", "db.system", "host.name"]
            [peer_attributes: <list of string> | default = ["peer.service", "db.name", "db.system"] ]

            # Attribute Key to multiply span metrics
            # Note that the attribute name is searched for in both
            # resource and span level attributes
            [span_multiplier_key: <string> | default = ""]

            # Enable extracting the span multiplier from the W3C tracestate header
            # using the OpenTelemetry probability sampling threshold (ot=th:<hex>).
            # When enabled, the tracestate threshold takes priority over span_multiplier_key.
            # Refer to https://opentelemetry.io/docs/specs/otel/trace/tracestate-probability-sampling/
            [enable_tracestate_span_multiplier: <bool> | default = false]

            # Enables additional labels for services and virtual nodes.
            [enable_virtual_node_label: <bool> | default = false]

            # List of attribute names used to identify the database name from span attributes. If it isn't set, the order is peer.service -> server.address -> network.peer.address -> db.name
            [database_name_attributes: <list of string> | default = ["db.namespace","db.name","db.system"]]

            # List of policies that will be applied to spans for inclusion or exclusion.
            [filter_policies: <list of filter policies config> | default = []]

        span_metrics:

            # Buckets for the latency histogram in seconds.
            [histogram_buckets: <list of float> | default = 0.002, 0.004, 0.008, 0.016, 0.032, 0.064, 0.128, 0.256, 0.512, 1.024, 2.048, 4.096, 8.192, 16.384]

            # Configure intrinsic dimensions to add to the metrics. Intrinsic dimensions are taken
            # directly from the respective resource and span properties.
            intrinsic_dimensions:
                # Whether to add the name of the service the span is associated with.
                [service: <bool> | default = true]
                # Whether to add the name of the span.
                [span_name: <bool> | default = true]
                # Whether to add the span kind describing the relationship between spans.
                [span_kind: <bool> | default = true]
                # Whether to add the span status code.
                [status_code: <bool> | default = true]
                # Whether to add a status message. Important note: The span status message may
                # contain arbitrary strings and thus have a very high cardinality.
                [status_message: <bool> | default = false]

            # Additional dimensions to add to the metrics along with the intrinsic dimensions.
            # Dimensions are searched for in the resource and span attributes and are added to
            # the metrics if present.
            [dimensions: <list of string>]

            # Custom labeling mapping to rename attributes or combine multiple attributes into a single label.
            # Use dimension_mappings to rename a single attribute to a custom label name or combine multiple attributes into a composite label.
            dimension_mappings: <list of label mappings>
                # The new label name (will be sanitized for Prometheus compatibility)
              - [name: <string>]
                # List of attribute names to map. Can be a single attribute (for renaming) or multiple attributes (for combining)
                [source_labels: <list of strings>]
                # Separator used to join attribute values together when multiple source_labels are provided.
                # For example, with source_labels: ["service.name", "service.namespace"] and join: "/",
                # if service.name="abc" and service.namespace="def", the result is "abc/def".
                # Ignored if only one source_label is provided.
                [join: <string> | default = ""]

            # Enable traces_target_info metrics
            [enable_target_info: <bool> | default = false]

            # Attribute Key to multiply span metrics
            # Note that the attribute name is searched for in both
            # resource and span level attributes
            [span_multiplier_key: <string> | default = ""]

            # Enable extracting the span multiplier from the W3C tracestate header
            # using the OpenTelemetry probability sampling threshold (ot=th:<hex>).
            # When enabled, the tracestate threshold takes priority over span_multiplier_key.
            # Refer to https://opentelemetry.io/docs/specs/otel/trace/tracestate-probability-sampling/
            [enable_tracestate_span_multiplier: <bool> | default = false]

            # List of policies that will be applied to spans for inclusion or exclusion.
            [filter_policies: <list of filter policies config> | default = []]

            # Drop specific labels from `traces_target_info` metrics
            [target_info_excluded_dimensions: <list of string>]

            # Add instance label to all span metrics series when enable_target_info is true
            [enable_instance_label: <bool> | default = true]

        host_info:

            # Resource attributes used to derive a unique host identifier.
            [host_identifiers: <list of string> | default = ["k8s.node.name", "host.id"]]

            # Name of the metric that will be generated.
            [metric_name: <string> | default = "traces_host_info"]

    # Registry configuration
    registry:

        # Interval to collect metrics and remote write them.
        [collection_interval: <duration> | default = 15s]

        # Interval after which a series is considered stale and will be deleted from the registry.
        # Once a metrics series is deleted, it won't be emitted anymore, keeping active series low.
        [stale_duration: <duration> | default = 15m]

        # A list of labels that will be added to all generated metrics.
        [external_labels: <map>]

        # If set, the tenant ID will added as label with the given label name to all generated metrics.
        [inject_tenant_id_as: <string>]

        # The maximum length of label names. Label names exceeding this limit will be truncated.
        [max_label_name_length: <int> | default = 1024]

        # The maximum length of label values. Label values exceeding this limit will be truncated.
        [max_label_value_length: <int> | default = 2048]

    # Type of limiter to use for controlling metrics-generator memory usage.
    # Options: "series" (default) or "entity".
    # - "series": Limits the total number of active metric series. Use with max_active_series override.
    # - "entity": Limits the number of unique label combinations (entities). Use with max_active_entities override.
    [limiter_type: <string> | default = "series"]

    # Controls which ring mode the metrics-generator uses.
    # "partition": Uses the partition ring (default). "generator": Uses the legacy generator ring.
    [ring_mode: <string> | default = "partition"]

    # Which decoder to use for data consumed from Kafka.
    # Valid values: "push-bytes", "otlp".
    [codec: <string> | default = "push-bytes"]

    # Number of concurrent Kafka consumers.
    [ingest_concurrency: <uint> | default = 16]

    # Instance ID used by the metrics-generator Kafka client.
    [instance_id: <string> | default = <hostname>]

    # Storage and remote write configuration
    storage:

        # Path to store the WAL. Each tenant will be stored in its own subdirectory.
        path: <string>

        # Configuration for the Prometheus Agent WAL
        # https://github.com/prometheus/prometheus/blob/v2.51.2/tsdb/agent/db.go#L62-L84
        wal: <prometheus agent WAL config>

        # How long to wait when flushing samples on shutdown
        [remote_write_flush_deadline: <duration> | default = 1m]

        # Whether to add X-Scope-OrgID header in remote write requests
        [remote_write_add_org_id_header: <bool> | default = true]

        # A list of remote write endpoints.
        # https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write
        remote_write:
            [- <Prometheus remote write config>]

    # This option only allows spans with end times that occur within the configured duration to be
    # considered in metrics generation.
    # This is to filter out spans that are outdated.
    [metrics_ingestion_time_range_slack: <duration> | default = 30s]

    # Overrides the key used to register the metrics-generator in the ring.
    [override_ring_key: <string> | default = "metrics-generator"]
```

## Query-frontend

The query frontend is responsible for sharding incoming requests for faster processing in parallel by the queriers.

For more information on configuration options, refer to [this file](https://github.com/grafana/tempo/blob/main/modules/frontend/config.go).
For architectural details, refer to the [Query frontend architecture](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/query-frontend/) documentation.

```yaml
# Query Frontend configuration block
query_frontend:

    # number of times to retry a request sent to a querier
    # (default: 2)
    [max_retries: <int>]

    # The number of goroutines dedicated to consuming, unmarshalling and recombining responses per request. This
    # same parameter is used for all endpoints.
    # (default: 10)
    [response_consumers: <int>]

    # Maximum number of outstanding requests per tenant per frontend; requests beyond this error with HTTP 429.
    # (default: 2000)
    [max_outstanding_per_tenant: <int>]

    # The number of jobs to batch together in one http request to the querier. Set to 1 to
    # disable.
    # (default: 7)
    [max_batch_size: <int>]

    # Enable multi-tenant queries.
    # If enabled, queries can be federated across multiple tenants.
    # The tenant IDs involved need to be specified separated by a '|'
    # character in the 'X-Scope-OrgID' header.
    # note: this is no-op if cluster doesn't have `multitenancy_enabled: true`
    # (default: true)
    [multi_tenant_queries_enabled: <bool>]

    # Comma-separated list of request header names to include in query logs. Applies
    # to both query stats and slow queries logs.
    [log_query_request_headers: <string> | default = ""]

    # Set a maximum timeout for all api queries at which point the frontend will cancel queued jobs
    # and return cleanly. HTTP will return a 503 and GRPC will return a context canceled error.
    # This timeout impacts all http and grpc streaming queries as part of the Tempo api surface such as
    # search, metrics queries, tags and tag values lookups, etc.
    # Generally it is preferred to let the client cancel context. This is a failsafe to prevent a client
    # from imposing more work on Tempo than desired.
    # (default: 0)
    [api_timeout: <duration>]

    # Prevents querying incomplete recent data by excluding the most recent portion of the time range.
    # Useful when live-store data may not yet be fully available for querying.
    # 0 disables this cutoff.
    # (default: 0)
    [query_end_cutoff: <duration>]

    # A list of regular expressions for refusing matching requests, these will apply for every request regardless of the endpoint.
    [url_deny_list: <list of strings> | default = <empty list>]

    # Max allowed TraceQL expression size, in bytes. queries bigger then this size will be rejected.
    # (default: 128 KiB)
    [max_query_expression_size_bytes: <int> | default = 131072]

    search:

        # The number of concurrent jobs to execute when searching the backend.
        # (default: 1000)
        [concurrent_jobs: <int>]

        # The target number of bytes for each job to handle when performing a backend search.
        # (default: 104857600)
        [target_bytes_per_job: <int>]

        # Limit used for search requests if none is set by the caller
        # (default: 20)
        [default_result_limit: <int>]

        # The maximum allowed value of the limit parameter on search requests. If the search request limit parameter
        # exceeds the value configured here the frontend will return a 400.
        # The default value is 262144 (256*1024). Set to 0 to disable this limit.
        # (default: 262144)
        [max_result_limit: <int>]

        # The maximum allowed time range for a search.
        # 0 disables this limit.
        # (default: 168h)
        [max_duration: <duration>]

        # query_backend_after controls where the query-frontend searches for traces.
        # Time ranges newer than query_backend_after will be searched in the live-stores only.
        # Time ranges older than query_backend_after will be searched in the backend/object storage only.
        # (default: 15m)
        [query_backend_after: <duration>]

        # If set to a non-zero value, it's value will be used to decide if query is within SLO or not.
        # Query is within SLO if it returned 200 within duration_slo seconds OR processed throughput_slo bytes/s data.
        # NOTE: Requires `duration_slo` AND `throughput_bytes_slo` to be configured.
        [duration_slo: <duration> | default = 0s ]

        # If set to a non-zero value, it's value will be used to decide if query is within SLO or not.
        # Query is within SLO if it returned 200 within duration_slo seconds OR processed throughput_slo bytes/s data.
        [throughput_bytes_slo: <float> | default = 0 ]

        # The number of time windows to break a search up into when doing a most recent TraceQL search. This only impacts TraceQL
        # searches with (most_recent=true).
        [most_recent_shards: <int> | default = 200]

        # The number of shards to break live-store queries into.
        [ingester_shards: <int> | default = 3]

        # The default number of spans to return per span set when not specified in the request.
        # Set to 0 to return unlimited spans by default.
        [default_spans_per_span_set: <int> | default = 3]

        # The maximum allowed value of spans per span set. 0 disables this limit.
        [max_spans_per_span_set: <int> | default = 100]

        # SLO configuration for Metadata (tags and tag values) endpoints.
        metadata_slo:
            # If set to a non-zero value, it's value will be used to decide if metadata query is within SLO or not.
            # Query is within SLO if it returned 200 within duration_slo seconds OR processed throughput_slo bytes/s data.
            # NOTE: Requires `duration_slo` AND `throughput_bytes_slo` to be configured.
            [duration_slo: <duration> | default = 0s ]


            # If set to a non-zero value, it's value will be used to decide if metadata query is within SLO or not.
            # Query is within SLO if it returned 200 within duration_slo seconds OR processed throughput_slo bytes/s data.
            [throughput_bytes_slo: <float> | default = 0 ]

    # Trace by ID lookup configuration
    trace_by_id:
        # The number of shards to split a trace by id query into.
        # (default: 50)
        [query_shards: <int>]

        # The maximum number of shards to execute at once. If set to 0 query_shards is used.
        # (default: 0)
        [concurrent_shards: <int>]

        # Enable external trace source for trace-by-ID queries. When enabled,
        # the frontend will create an additional shard to query the external endpoint
        # configured in the querier.
        [external_enabled: <bool> | default = false]

        # If set to a non-zero value, it's value will be used to decide if metadata query is within SLO or not.
        # Query is within SLO if it returned 200 within duration_slo seconds OR processed throughput_slo bytes/s data.
        # NOTE: Requires `duration_slo` AND `throughput_bytes_slo` to be configured.
        [duration_slo: <duration> | default = 0s ]

        # If set to a non-zero value, it's value will be used to decide if metadata query is within SLO or not.
        # Query is within SLO if it returned 200 within duration_slo seconds OR processed throughput_slo bytes/s data.
        [throughput_bytes_slo: <float> | default = 0 ]

    # Metrics query configuration
    metrics:
        # The number of concurrent jobs to execute when querying the backend.
        [concurrent_jobs: <int> | default = 1000 ]

        # The target number of bytes for each job to handle when querying the backend.
        [target_bytes_per_job: <int> | default = 100MiB ]

        # The maximum allowed time range for a metrics query.
        # 0 disables this limit.
        [max_duration: <duration> | default = 24h ]

        # Maximum number of exemplars per range query.
        # Set to 0 to disable exemplars.
        [max_exemplars: <int> | default = 100 ]

        # Maximum number of time series returned for a metrics query.
        # Default is 0, which means there is no limit
        [max_response_series: <int> | default = 0]

        # query_backend_after controls where the query-frontend searches for traces.
        # Time ranges older than query_backend_after will be searched in the backend/object storage only.
        # Time ranges between query_backend_after and now will be queried from the metrics-generators.
        [query_backend_after: <duration> | default = 15m ]

        # The target length of time for each job to handle when querying the backend.
        [interval: <duration> | default = 5m ]

        # If set to a non-zero value, it's value will be used to decide if query is within SLO or not.
        # Query is within SLO if it returned 200 within duration_slo seconds OR processed throughput_slo bytes/s data.
        # NOTE: `duration_slo` and `throughput_bytes_slo` both must be configured for it to work
        [duration_slo: <duration> | default = 0s ]

        # If set to a non-zero value, it's value will be used to decide if query is within SLO or not.
        # Query is within SLO if it returned 200 within duration_slo seconds OR processed throughput_slo bytes/s data.
        [throughput_bytes_slo: <float> | default = 0 ]

        # The number of shards to use when streaming metrics queries back to the user. A shard must be fully completed before
        # the results are returned to the user. More shards results in a more granular effect at the cost of additional bookkeeping.
        [streaming_shards: <int> | default = 200]

    # MCP server configuration. Enabling the MCP server allows tracing data
    # to be exposed to LLM-based tools. Requires explicit opt-in.
    mcp_server:
        [enabled: <bool> | default = false]

```

### Limit query size to improve performance and stability

Querying large tracing data presents several challenges.
Span sets with large number of spans impact query performance and stability.
In a similar manner, excessive queries result size can also negatively impact query performance.

#### Limit the spans per spanset

You can control spans per spanset behavior using two configuration options:

- `default_spans_per_span_set`: Sets the default number of spans returned when not specified in the query (default: 3). Set to `0` to return unlimited spans by default.
- `max_spans_per_span_set`: Sets the maximum allowed value (default: 100). Set to `0` to disable the limit entirely.

In Grafana or Grafana Cloud, you can use the **Span Limit** field in the [TraceQL query editor](https://grafana.com/docs/grafana-cloud/connect-externally-hosted/data-sources/tempo/query-editor/) in Grafana Explore.
This field sets the number of spans to return for each span set (the `spss` query parameter).
If not specified, the value from `default_spans_per_span_set` is used.
The maximum value that you can set for **Span Limit** (or the `spss` query parameter) is controlled by `max_spans_per_span_set`.
To disable the maximum spans per span set limit, set `max_spans_per_span_set` to `0`.
When set to `0`, there is no maximum and users can request any number of spans, including unlimited spans by setting `spss=0`.

#### Cap the maximum query length

You can set the maximum length of a query using `query_frontend.max_query_expression_size_bytes` configuration parameter for the query-frontend. The default value is 128 KB.

This limit is used to protect the system’s stability from potential abuse or mistakes, when running a large potentially expensive query.

You can set the value lower of higher by setting it in the `query_frontend` configuration section, for example:

```
query_frontend:
  max_query_expression_size_bytes: 10000
```

## Querier

The querier executes query jobs dispatched by the query frontend. It fetches trace data from both live-stores (for recent data) and object storage (for historical data), then returns results to the query frontend for merging.

For more information on configuration options, refer to [this file](https://github.com/grafana/tempo/blob/main/modules/querier/config.go).
For architectural details, refer to the [Querier architecture](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/querier/) documentation.

```yaml
# querier config block
querier:

    # The query frontend turns both trace by id (/api/traces/<id>) and search (/api/search?<params>) requests
    # into subqueries that are then pulled and serviced by the queriers.
    # This value controls the overall number of simultaneous subqueries that the querier will service at once. It does
    # not distinguish between the types of queries.
    [max_concurrent_queries: <int> | default = 20]

    trace_by_id:
        # Timeout for trace lookup requests
        [query_timeout: <duration> | default = 10s]

        # External trace source configuration. When configured, trace-by-ID queries
        # also fetch trace data from an external HTTP endpoint that returns
        # an OpenTelemetry protobuf formatted trace. Enable this feature using
        # query_frontend.trace_by_id.external_enabled.
        external:
            # The URL of the external service.
            # Example: "http://external-service:3200"
            [endpoint: <string>]

            # Timeout for requests to the external endpoint.
            [timeout: <duration> | default = 10s]

    search:
        # Timeout for search requests
        [query_timeout: <duration> | default = 30s]

    # Metrics query configuration
    metrics:
        # Number of blocks to process concurrently during a metrics query.
        [concurrent_blocks: <int> | default = 2]

        # Controls whether trace-level timestamp columns are used in a metrics query.
        # Loading these columns has a cost, so in some cases it is faster to skip them,
        # reducing I/O but increasing the number of spans evaluated and discarded.
        # The value is a ratio between 0.0 and 1.0. If a block overlaps the time window
        # by less than this value, the columns are skipped.
        # 1.0 always loads columns; 0.0 never loads them.
        [time_overlap_cutoff: <float> | default = 0.2]

    # config of the worker that connects to the query frontend
    frontend_worker:

        # the address of the query frontend to connect to, and process queries
        # Example: "frontend_address: query-frontend-discovery.default.svc.cluster.local:9095"
        [frontend_address: <string>]

    # Partition ring configuration for distributing queries across live-stores.
    partition_ring:
        # When enabled, the querier minimizes the number of live-store requests
        # by preferring a single zone per partition.
        [minimize_requests: <bool> | default = true]

        # Delay before sending a hedged request to another live-store
        # when minimize_requests is enabled.
        [hedging_delay: <duration> | default = 3s]

        # Preferred availability zone for routing queries. If set, the querier
        # routes to live-stores in this zone first.
        [preferred_zone: <string> | default = ""]
```

The querier also queries compacted blocks that fall within a range of `2 * storage.trace.blocklist_poll`, where
`storage.trace.blocklist_poll` is the blocklist poll duration configured in the storage section below.

## Backend scheduler

For architectural details, refer to the [Compaction architecture](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/compaction/) documentation.

The backend scheduler is responsible for scheduling and tracking jobs which are assigned to backend workers for processing.
Only one scheduler should be running at a time.

```yaml
backend_scheduler:

  # Work cache configuration
  work:

    # How long to keep completed or failed jobs in the work cache before pruning them.
    [prune_age: <duration> | default = 1h]

    # After this duration, jobs that have not been updated are considered dead and will be reassigned.
    [dead_job_timeout: <duration> | default = 24h]

  # How often to perform maintenance tasks (pruning old jobs, checking for dead jobs, etc.)
  [maintenance_interval: <duration> | default = 1m]

  # How often to flush the work cache to backend storage
  [backend_flush_interval: <duration> | default = 1m]

  # Provider configuration for job generation
  provider:

    # Retention job configuration
    retention:

      # How often to check for blocks that need to be deleted due to retention
      [interval: <duration> | default = 1h]

    # Compaction job configuration
    compaction:

      # How often to measure tenant block lists and create new compaction jobs
      [measure_interval: <duration> | default = 1m]

      # Compaction settings
      # Refer to the Compaction block section for details
      compaction: <Compaction config>

      # Maximum number of compaction jobs to create per tenant
      [max_jobs_per_tenant: <int> | default = 1000]

      # Minimum number of blocks required for compaction
      [min_input_blocks: <int> | default = 2]

      # Maximum number of blocks to compact together
      [max_input_blocks: <int> | default = 4]

      # Maximum compaction level (0 means no limit)
      [max_compaction_level: <int> | default = 0]

      # Minimum time between compaction cycles for a tenant
      [min_cycle_interval: <duration> | default = 30s]

    # Redaction job configuration
    redaction:

      # How often to check for pending redaction jobs when the queue is empty
      [poll_interval: <duration> | default = 2s]

      # How long to wait before rescanning for output blocks from compaction
      # jobs that were active when the redaction was submitted.
      # Must be less than work.prune_age or Tempo will fail to start.
      [rescan_delay: <duration> | default = 5m]

      # Maximum number of rescan attempts before requiring operator resubmission
      [max_rescan_generations: <int> | default = 5]

  # How long to wait for a worker to complete a job before timing out internally
  [job_timeout: <duration> | default = 15s]

  # Path to store local work cache files
  [local_work_path: <string> | default = "/var/tempo"]
```

## Backend worker

For architectural details, refer to the [Compaction architecture](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/compaction/) documentation.

The backend worker connects to the backend scheduler to receive and process jobs.
Workers are responsible for executing compaction and retention and other jobs, and updating the scheduler on job status.

```yaml
# gRPC client configuration for connecting to the backend scheduler
backend_scheduler_client:
  grpc_client_config: <GRPC client config>
backend_worker:

  # Address of the backend scheduler to connect to
  [backend_scheduler_addr: <string>]

  # Backoff configuration for retrying failed jobs
  backoff:
    [min_period: <duration> | default = 100ms]
    [max_period: <duration> | default = 1m]
    [max_retries: <int> | default = 0]

  # Compaction settings
  # Refer to the Compaction block section for details
  compaction: <Compaction config>

  # Override the default ring key used by the backend worker
  [override_ring_key: <string> | default = "backend-worker"]

  # Ring configuration for coordinating tenant polling across workers
  ring:
    kvstore: <KVStore config>

    # Period at which to heartbeat the instance
    # 0 disables heartbeat altogether
    [heartbeat_period: <duration> | default = 5s]

    # The heartbeat timeout, after which, the instance is skipped.
    # 0 disables timeout.
    [heartbeat_timeout: <duration> | default = 1m]

    # Our Instance ID to register as in the ring.
    [instance_id: <string> | default = os.Hostname()]

    # Name of the network interface to read address from.
    [instance_interface_names: <list of string> | default = ["eth0", "en0"] ]

    # Our advertised IP address in the ring, (useful if the local ip =/= the external ip)
    # Will default to the configured `instance_id` ip address,
    # if unset, will fallback to ip reported by `instance_interface_names`
    # (Effected by `enable_inet6`)
    [instance_addr: <string> | default = auto(instance_id, instance_interface_names)]

    # Our advertised port in the ring
    # Defaults to the configured GRPC listing port
    [instance_port: <int> | default = auto(listen_port)]

    # Enables the registering of ipv6 addresses in the ring.
    [enable_inet6: <bool> | default = false]

  # Timeout for finishing the current job before shutting down the worker
  [finish_on_shutdown_timeout: <duration> | default = 30s]
```

## Storage

Tempo supports Amazon S3, GCS, Azure, and local file system for storage. In addition, you can use Memcached or Redis for increased query performance.

For more information on configuration options, refer to [this file](https://github.com/grafana/tempo/blob/main/tempodb/config.go).

### Local storage recommendations

While you can use local storage, object storage is recommended for production workloads.
A local backend won't correctly retrieve traces with a distributed deployment unless all components have access to the same disk.
Tempo is designed for object storage more than local storage.

At Grafana Labs, we've run Tempo with SSDs when using local storage.
Hard drives haven't been tested.

You can estimate how much storage space you need by considering the ingested bytes and retention.
For example, ingested bytes per day _times_ retention days = stored bytes.

You can not use both local and object storage in the same Tempo deployment.

### Hedged requests

Each storage backend (GCS, S3, and Azure) supports hedged requests.
Hedging reduces long-tail read latency by issuing a duplicate backend request after a configured delay.
Tempo uses whichever response arrives first and discards the other.

Configure hedging with two parameters in each backend block:

- `hedge_requests_at` -- the delay before sending a duplicate request. Set this to approximately the p99 latency of your backend requests. Default is `0` (disabled).
- `hedge_requests_up_to` -- the maximum number of requests to issue, including the original. Default is `2`. Requires `hedge_requests_at` to be set.

Hedging is most effective on querier nodes, where read latency directly affects query performance.
It has minimal impact on other components.

### Storage block configuration example

The storage block configures TempoDB.
The following example shows common options.
For further platform-specific information, refer to the following:

- [GCS](hosted-storage/gcs/)
- [S3](hosted-storage/s3/)
- [Azure](hosted-storage/azure/)
- [Parquet](parquet/)

```yaml
# Storage configuration for traces
storage:

    trace:

        # The storage backend to use
        # Should be one of "gcs", "s3", "azure" or "local" (only supported in the monolithic mode)
        # CLI flag -storage.trace.backend
        [backend: <string>]

        # Local backend configuration. Will be used only if value of backend is "local".
        # For development and testing only. Use object storage for production workloads.
        local:

            # Path to store trace data on local disk
            [path: <string>]

        # GCS configuration. Will be used only if value of backend is "gcs"
        # Check the GCS doc within this folder for information on GCS specific permissions.
        gcs:

            # Bucket name in gcs
            # Tempo requires a bucket to maintain a top-level object structure. You can use prefix option with this to nest all objects within a shared bucket.
            # Example: "bucket_name: tempo"
            [bucket_name: <string>]

            # optional.
            # Prefix name in gcs
            # Tempo has this additional option to support a custom prefix to nest all the objects within a shared bucket.
            [prefix: <string>]

            # Buffer size for reads. Default is 10MB
            # Example: "chunk_buffer_size: 5_000_000"
            [chunk_buffer_size: <int>]

            # Optional
            # Api endpoint override
            # Example: "endpoint: https://storage.googleapis.com/storage/v1/"
            [endpoint: <string>]

            # Optional. Default is false.
            # Example: "insecure: true"
            # Set to true to disable authentication and certificate checks on gcs requests
            [insecure: <bool>]

            # The number of list calls to make in parallel to the backend per instance.
            # Adjustments here will impact the polling time, as well as the number of Go routines.
            # Default is 3
            [list_blocks_concurrency: <int>]

            # Optional. Default is 0 (disabled)
            # Example: "hedge_requests_at: 500ms"
            # If set to a non-zero value a second request will be issued at the provided duration. Recommended to
            # be set to p99 of GCS requests to reduce long tail latency. This setting is most impactful when
            # used with queriers and has minimal to no impact on other pieces.
            [hedge_requests_at: <duration>]

            # Optional. Default is 2
            # Example: "hedge_requests_up_to: 2"
            # The maximum number of requests to execute when hedging. Requires hedge_requests_at to be set.
            [hedge_requests_up_to: <int>]

            # Optional
            # Example: "object_cache_control: "no-cache""
            # A string to specify the behavior with respect to caching of the objects stored in GCS.
            # See the GCS documentation for more detail: https://cloud.google.com/storage/docs/metadata
            [object_cache_control: <string>]

            # Optional
            # Example: "object_metadata: {'key': 'value'}"
            # A map key value strings for user metadata to store on the GCS objects.
            # See the GCS documentation for more detail: https://cloud.google.com/storage/docs/metadata
            [object_metadata: <map[string]string>]

            # Optional. Default is 3.
            # Number of times to retry GCS copy and delete operations used by compaction and retention.
            [max_retries: <int> | default = 3]

        # S3 configuration. Will be used only if value of backend is "s3"
        # Check the S3 doc within this folder for information on s3 specific permissions.
        s3:

            # Bucket name in s3
            # Tempo requires a bucket to maintain a top-level object structure. You can use prefix option with this to nest all objects within a shared bucket.
            [bucket: <string>]

            # optional.
            # Prefix name in s3
            # Tempo has this additional option to support a custom prefix to nest all the objects within a shared bucket.
            [prefix: <string>]

            # api endpoint to connect to. use AWS S3 or any S3 compatible object storage endpoint.
            # Example: "endpoint: s3.dualstack.us-east-2.amazonaws.com"
            [endpoint: <string>]

            # The number of list calls to make in parallel to the backend per instance.
            # Adjustments here will impact the polling time, as well as the number of Go routines.
            # Default is 3
            [list_blocks_concurrency: <int>]

            # optional.
            # By default the region is inferred from the endpoint,
            # but is required for some S3-compatible storage engines.
            # Example: "region: us-east-2"
            [region: <string>]

            # optional.
            # access key when using static credentials.
            [access_key: <string>]

            # optional.
            # secret key when using static credentials.
            [secret_key: <string>]

            # optional.
            # session token when using static credentials.
            [session_token: <string>]

            # optional.
            # enable if endpoint is http
            [insecure: <bool>]

            # Optional. Default is 0 (disabled).
            # Part size in bytes for multipart uploads. Set to 0 to disable multipart uploads.
            # Non-zero values must be at least 5 MiB (5242880 bytes).
            [part_size: <int>]

            # optional.
            # Path to the client certificate file.
            [tls_cert_path: <string>]

            # optional.
            # Path to the private client key file.
            [tls_key_path: <string>]

            # optional.
            # Path to the CA certificate file.
            [tls_ca_path: <string>]

            # optional.
            # Path to the CA certificate file.
            [tls_server_name: <string>]

            # optional.
            # Set to true to disable verification of a TLS endpoint. The default value is false.
            [tls_insecure_skip_verify: <bool>]

            # optional.
            # Override the default cipher suite list, separated by commas.
            [tls_cipher_suites: <string>]

            # optional.
            # Override the default minimum TLS version. The default value is VersionTLS12. Allowed values: VersionTLS10, VersionTLS11, VersionTLS12, VersionTLS13
            [tls_min_version: <string>]

            # Optional. Default is false.
            # Use V2 signing instead of V4 for S3 requests.
            [signature_v2: <bool>]

            # optional.
            # enable to use path-style requests.
            [forcepathstyle: <bool>]

            # Optional.
            # Enable to use dualstack endpoint for DNS resolution.
            # Check out the (S3 documentation on dualstack endpoints)[https://docs.aws.amazon.com/AmazonS3/latest/userguide/dual-stack-endpoints.html]
            [enable_dual_stack: <bool>]

            # Optional. Default is 0
            # Example: "bucket_lookup_type: 0"
            # options: 0: BucketLookupAuto, 1: BucketLookupDNS, 2: BucketLookupPath
            # See the [S3 documentation on virtual-hosted–style and path-style](https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html#path-style-access) for more detail.
            # See the [Minio-API documentation on opts.BucketLookup](https://github.com/minio/minio-go/blob/master/docs/API.md#newendpoint-string-opts-options-client-error)] for more detail.
            # Notice: ignore this option if `forcepathstyle` is set true, this option allow expose minio's sdk configure.
            [bucket_lookup_type: <int> | default = 0]

            # Optional. Default is 0 (disabled)
            # Example: "hedge_requests_at: 500ms"
            # If set to a non-zero value a second request will be issued at the provided duration. Recommended to
            # be set to p99 of S3 requests to reduce long tail latency. This setting is most impactful when
            # used with queriers and has minimal to no impact on other pieces.
            [hedge_requests_at: <duration>]

            # Optional. Default is 2
            # Example: "hedge_requests_up_to: 2"
            # The maximum number of requests to execute when hedging. Requires hedge_requests_at to be set.
            [hedge_requests_up_to: <int>]

            # Optional. Default is 10 (minio default)
            # Example: "retry_max_attempts: 10"
            # The maximum number of retry attempts on failed S3 requests. Set to 1 to disable retries.
            [retry_max_attempts: <int>]

            # Optional. Default is 200ms (minio default)
            # Example: "retry_backoff_initial: 200ms"
            # The baseline time after which a retry is attempted on failed S3 requests. This time is
            # doubled each retry until it hits retry_backoff_max, after which it remains at retry_backoff_max.
            [retry_backoff_initial: <duration>]

            # Optional. Default is 1s (minio default)
            # Example: "retry_backoff_max: 1s"
            # The maximum duration to wait between retry attempts on failed S3 requests.
            [retry_backoff_max: <duration>]

            # Optional
            # Example: "tags: {'key': 'value'}"
            # A map of key value strings for user tags to store on the S3 objects. This helps set up filters in S3 lifecycles.
            # See the [S3 documentation on object tagging](https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-tagging.html) for more detail.
            [tags: <map[string]string>]

            # Optional.
            # S3 storage class for uploaded objects. If unset, the default storage class of the bucket is used.
            # See the [S3 documentation on storage classes](https://docs.aws.amazon.com/AmazonS3/latest/userguide/storage-class-intro.html) for valid values.
            [storage_class: <string>]

            # Optional.
            # A map of key-value strings for user-defined metadata to store on S3 objects.
            [metadata: <map[string]string>]

            # Deprecated and ignored. This setting has no effect other than emitting a startup
            # warning, and will be removed in a future release. Use native AWS authentication
            # mechanisms (IAM roles, environment variables) instead.
            [native_aws_auth_enabled: <bool> | default = false]

            [sse: <map[string]string>]:
              # Optional
              # Example: type: SSE-S3
              # Type of encryption to use with s3 bucket, either SSE-KMS, SSE-S3, or SSE-C
              [type: string]:

              # Optional
              # Example: kms_key_id: "1234abcd-12ab-34cd-56ef-1234567890ab"
              # the kms key id is the identification of the key in an account or region
              kms_key_id:
              # Optional
              # Example: kms_encryption_context: "encryptionContext": {"department": "10103.0"}
              # KMS Encryption Context used for object encryption. It expects JSON formatted string
              kms_encryption_context:

              # Optional
              # Example: encryption_key: <32-byte-long-key>
              # SSE-C Encryption Key used for object encryption with customer provided keys.
              # It expects a 32 byte long string.
              encryption_key:

        # Azure configuration. Will be used only if value of backend is "azure".
        azure:

            # store traces in this container.
            # Tempo requires bucket to  maintain a top-level object structure. You can use prefix option to nest all objects within a shared bucket
            [container_name: <string>]

            # optional.
            # Prefix for azure.
            # Tempo has this additional option to support a custom prefix to nest all the objects within a shared bucket.
            [prefix: <string>]

            # optional.
            # Azure endpoint to use, defaults to Azure global(core.windows.net) for other
            # regions this needs to be changed e.g Azure China(blob.core.chinacloudapi.cn),
            # Azure German(blob.core.cloudapi.de), Azure US Government(blob.core.usgovcloudapi.net).
            [endpoint_suffix: <string>]

            # Name of the azure storage account
            [storage_account_name: <string>]

            # optional.
            # access key when using access key credentials.
            [storage_account_key: <string>]

            # optional.
            # use Azure Managed Identity to access Azure storage.
            [use_managed_identity: <bool>]

            # optional.
            # Use a Federated Token to authenticate to the Azure storage account.
            # Enable if you want to use Azure Workload Identity. Expects AZURE_CLIENT_ID,
            # AZURE_TENANT_ID, AZURE_AUTHORITY_HOST and AZURE_FEDERATED_TOKEN_FILE envs to be present
            # (these are set automatically when using Azure Workload Identity).
            [use_federated_token: <bool>]

            # optional.
            # The Client ID for the user-assigned Azure Managed Identity used to access Azure storage.
            [user_assigned_id: <string>]

            # Optional. Default is 4
            # Number of simultaneous uploads to Azure.
            [max_buffers: <int> | default = 4]

            # Optional. Default is 3145728 (3 MiB)
            # Buffer size for uploads to Azure.
            [buffer_size: <int> | default = 3145728]

            # Optional. Default is 0 (disabled)
            # Example: "hedge_requests_at: 500ms"
            # If set to a non-zero value a second request will be issued at the provided duration. Recommended to
            # be set to p99 of Azure Block Storage requests to reduce long tail latency. This setting is most impactful when
            # used with queriers and has minimal to no impact on other pieces.
            [hedge_requests_at: <duration>]

            # Optional. Default is 2
            # Example: "hedge_requests_up_to: 2"
            # The maximum number of requests to execute when hedging. Requires hedge_requests_at to be set.
            [hedge_requests_up_to: <int>]

        # How often to repoll the backend for new blocks. Default is 5m
        [blocklist_poll: <duration>]

        # Number of blocks to process in parallel during polling. Default is 50.
        [blocklist_poll_concurrency: <int>]

        # By default components will pull the blocklist from the tenant index. If that fails the component can
        # fallback to scanning the entire bucket. Set to false to disable this behavior. Default is true.
        [blocklist_poll_fallback: <bool>]

        # Maximum number of workers that should build the tenant index. All other components will download
        # the index. Default 2.
        [blocklist_poll_tenant_index_builders: <int>]

        # Number of tenants to poll concurrently. Default is 1.
        [blocklist_poll_tenant_concurrency: <int>]

        # The oldest allowable tenant index. If an index is pulled that is older than this duration,
        # the polling will consider this an error. Note that `blocklist_poll_fallback` applies here.
        # If fallback is true and a tenant index exceeds this duration, it will fall back to listing
        # the bucket contents.
        # Default 0 (disabled).
        [blocklist_poll_stale_tenant_index: <duration>]

        # Offsets the concurrent blocklist polling by a random amount. The maximum amount of offset
        # is the provided value in milliseconds. This configuration value can be used if the polling
        # cycle is overwhelming your backend with concurrent requests.
        # Default 0 (disabled)
        [blocklist_poll_jitter_ms: <int>]

        # Polling will tolerate this many consecutive errors during the poll of
        # a single tenant before marking the tenant as failed.
        # This can be set to 0 which means a single error is sufficient to mark the tenant failed
        # and exit early.  Any previous results for the failing tenant will be kept.
        # See also `blocklist_poll_tolerate_tenant_failures` below.
        # Default 1
        [blocklist_poll_tolerate_consecutive_errors: <int>]

        # Polling will tolerate this number of tenants which have failed to poll.
        # This can be set to 0 which means a single tenant failure  sufficient to fail and exit
        # early.
        # Default 1
        [blocklist_poll_tolerate_tenant_failures: <int>]

        # Used to tune how quickly the poller will delete any remaining backend
        # objects found in the tenant path.  This functionality requires enabling
        # below.
        # Default: 12h
        [empty_tenant_deletion_age: <duration>]

        # Polling will delete the index for a tenant if no blocks are found to
        # exist.  If this setting is enabled, the poller will also delete any
        # remaining backend objects found in the tenant path.  This is used to
        # clean up partial blocks which may have not been cleaned up by the
        # retention.
        [empty_tenant_deletion_enabled: <bool> | default = false]

        # Cache type to use. Should be one of "redis", "memcached"
        # Example: "cache: memcached"
        # Deprecated. See [cache](#cache) section.
        [cache: <string>]

        # Minimum compaction level of block to qualify for bloom filter caching. Default is 0 (disabled), meaning
        # that compaction level is not used to determine if the bloom filter should be cached.
        # Example: "cache_min_compaction_level: 2"
        [cache_min_compaction_level: <int>]

        # Max block age to qualify for bloom filter caching. Default is 0 (disabled), meaning that block age is not
        # used to determine if the bloom filter should be cached.
        # Example: "cache_max_block_age: 48h"
        [cache_max_block_age: <duration>]

        # Configuration parameters that impact trace search
        search: <Search config>

        # Background cache configuration. Requires having a cache configured.
        # Deprecated. See [cache](#cache) section.
        background_cache:

        # Memcached caching configuration block
        # Deprecated. See [cache](#cache) section.
        memcached:

        # Redis configuration block
        # EXPERIMENTAL
        # Deprecated. See [cache](#cache) section.
        redis:

        # the worker pool is used primarily when finding traces by id, but is also used by other
        pool:

            # total number of workers pulling jobs from the queue
            [max_workers: <int> | default = 400]

            # length of job queue. important for querier as it queues a job for every block it has to search
            [queue_depth: <int> | default = 20000]

        # configuration block for the Write Ahead Log (WAL)
        wal: <WAL config>
          [path: <string> | default = "/var/tempo/wal"]
          [ingestion_time_range_slack: <duration> | default = 2m]

        # block configuration
        block: <Block config>
```

## Memberlist

[Memberlist](https://github.com/hashicorp/memberlist) is the default mechanism for all of the Tempo pieces to coordinate with each other.

```yaml
memberlist:
    # Name of the node in memberlist cluster. Defaults to hostname.
    [node_name: <string> | default = ""]

    # Add random suffix to the node name.
    [randomize_node_name: <boolean> | default = true]

    # The timeout for establishing a connection with a remote node, and for
    # read/write operations.
    [stream_timeout: <duration> | default = 2s]

    # Multiplication factor used when sending out messages (factor * log(N+1)).
    [retransmit_factor: <int> | default = 2]

    # How often to use pull/push sync.
    [pull_push_interval: <duration> | default = 30s]

    # How often to gossip.
    [gossip_interval: <duration> | default = 1s]

    # How many nodes to gossip to.
    [gossip_nodes: <int> | default = 2]

    # How long to keep gossiping to dead nodes, to give them chance to refute their
    # death.
    [gossip_to_dead_nodes_time: <duration> | default = 30s]

    # How soon can dead node's name be reclaimed with new address. Defaults to 0,
    # which is disabled.
    [dead_node_reclaim_time: <duration> | default = 0s]

    # Enable message compression. Reduces bandwidth usage at the cost of slightly
    # more CPU utilization. Tempo disables this by default for performance.
    [compression_enabled: <boolean> | default = false]

    # Other cluster members to join. Can be specified multiple times. It can be an
    # IP, hostname or an entry specified in the DNS Service Discovery format (see
    # https://grafana.com/docs/mimir/latest/configure/about-dns-service-discovery/
    # for more details).
    # A "Headless" Cluster IP service in Kubernetes.
    # Example:
    #   - gossip-ring.tracing.svc.cluster.local:7946
    [join_members: <list of string> | default = ]

    # Min backoff duration to join other cluster members.
    [min_join_backoff: <duration> | default = 1s]

    # Max backoff duration to join other cluster members.
    [max_join_backoff: <duration> | default = 1m]

    # Max number of retries to join other cluster members.
    [max_join_retries: <int> | default = 10]

    # If this node fails to join memberlist cluster, abort.
    [abort_if_cluster_join_fails: <boolean> | default = false]

    # If not 0, how often to rejoin the cluster. Occasional rejoin can help to fix
    # the cluster split issue, and is harmless otherwise. For example when using
    # only few components as a seed nodes (via -memberlist.join), then it's
    # recommended to use rejoin. If -memberlist.join points to dynamic service that
    # resolves to all gossiping nodes (eg. Kubernetes headless service), then rejoin
    # is not needed.
    [rejoin_interval: <duration> | default = 0s]

    # Timeout for leaving memberlist cluster.
    [leave_timeout: <duration> | default = 20s]

    # IP address to listen on for gossip messages.
    # Multiple addresses may be specified.
    [bind_addr: <list of string> | default = ["0.0.0.0"] ]

    # Port to listen on for gossip messages.
    [bind_port: <int> | default = 7946]

    # Gossip address to advertise to other members in the cluster.
    # Used for NAT traversal.
    [advertise_addr: <string> | default = ""]

    # Gossip port to advertise to other members in the cluster.
    # Used for NAT traversal.
    [advertise_port: <int> | default = 0]

    # Optional string to include in outbound packets and gossip streams.
    # Other members discard any message whose label doesn't match, unless
    # cluster_label_verification_disabled is true.
    [cluster_label: <string> | default = ""]

    # When true, memberlist doesn't verify that inbound packets and gossip
    # streams have the cluster label matching the configured one.
    # Disable verification while rolling out a cluster label change.
    [cluster_label_verification_disabled: <boolean> | default = false]

    # Timeout used when connecting to other nodes to send packet.
    [packet_dial_timeout: <duration> | default = 5s]

    # Timeout for writing 'packet' data.
    [packet_write_timeout: <duration> | default = 5s]

```

## Configuration blocks

Defines re-used configuration blocks.

### Block

```yaml
# block format version. options: vParquet4, vParquet5
[version: <string> | default = vParquet4]

# bloom filter false positive rate. lower values create larger filters but fewer false positives
[bloom_filter_false_positive: <float> | default = 0.01]

# maximum size of each bloom filter shard
[bloom_filter_shard_size_bytes: <int> | default = 100KiB]

# an estimate of the number of bytes per row group when cutting Parquet blocks. lower values will
#  create larger footers but will be harder to shard when searching. It is difficult to calculate
#  this field directly and it may vary based on workload. This is roughly a lower bound.
[parquet_row_group_size_bytes: <int> | default = 100MB]

# Configures attributes to be stored in dedicated columns within the parquet file, rather than in the
# generic attribute key-value list. This allows for more efficient searching of these attributes.
# Up to 10 span attributes and 10 resource attributes can be configured as dedicated columns.
# Requires vParquet4 or later.
parquet_dedicated_columns: <list of columns>

      # name of the attribute
    - [name: <string>]

      # type of the attribute. options: string
      [type: <string>]

      # scope of the attribute.
      # options: resource, span
      [scope: <string>]
```

### Compaction

The `compaction` configuration block is used by the scheduler and worker.

```yaml
# Optional. Duration to keep blocks.
[block_retention: <duration> | default=336h]

# Optional
# Duration to keep blocks that have been compacted elsewhere.
[compacted_block_retention: <duration> | default=1h]

# Optional
# Blocks in this time window will be compacted together.
[compaction_window: <duration> | default=1h]

# Optional
# Maximum number of traces in a compacted block.
# WARNING: Deprecated. Use max_block_bytes instead.
[max_compaction_objects: <int> | default=6000000]

# Optional
# Maximum size of a compacted block in bytes.
[max_block_bytes: <int> | default=107374182400]

# Optional
# Number of tenants to process in parallel during retention.
[retention_concurrency: <int> | default=10]

# Optional
# The maximum amount of time to spend compacting a single tenant before moving to the next.
[max_time_per_tenant: <duration> | default=5m]

# Optional
# The time between compaction cycles.
# Note: The default will be used if the value is set to 0.
[compaction_cycle: <duration> | default=30s]
```

### Filter policies

Span filter configuration policies block

#### Filter policy

```yaml
# Exclude filters (positive matching)
[include: <policy match>]

# Exclude filters (negative matching)
[exclude: <policy match>]
```

#### Policy match

```yaml
# How to match the value of attributes
# Options: "strict", "regex"
[match_type: <string>]

# List of attributes to match
attributes: <list of policy attributes>

    # Attribute key
  - [key: <string>]

    # Attribute value
    [value: <any>]
```

#### Examples

```yaml
exclude:
  match_type: "regex"
  attributes:
    - key: "resource.service.name"
      value: "unknown_service:myservice"
```

```yaml
include:
  match_type: "strict"
  attributes:
    - key: "foo.bar"
      value: "baz"
```

### GRPC client

These settings are used to configure various gRPC clients used throughout Tempo.

```
[max_recv_msg_size: <int> | default = 104857600]
[max_send_msg_size: <int> | default = 104857600]
[grpc_compression: <string> | default = "snappy"]
[rate_limit: <float> | default = 0]
[rate_limit_burst: <int> | default = 0]
[backoff_on_ratelimits: <bool> | default = false]
backoff_config:
  [min_period: <duration> | default = 100ms]
  [max_period: <duration> | default = 10s]
  [max_retries: <int> | default = 10]
[initial_stream_window_size: <int>]
[initial_connection_window_size: <int>]
[tls_enabled: <bool> | default = false]
[tls_cert_path: <string>]
[tls_key_path: <string>]
[tls_ca_path: <string>]
[tls_server_name: <string>]
[tls_insecure_skip_verify: <bool> | default = false]
[tls_cipher_suites: <string>]
[tls_min_version: <string>]
[connect_timeout: <duration> | default = 5s]
[connect_backoff_base_delay: <duration> | default = 1s]
[connect_backoff_max_delay: <duration> | default = 5s]
```

### KVStore

The `kvstore` configuration block

```yaml
# Set backing store to use
[store: <string> | default = "consul"]

# What prefix to use for keys
[prefix: <string> | default = "ring."]

# Store specific configs
consul:
  [host: <string> | default = "localhost:8500"]
  [acl_token: <secret string> | default = "" ]
  [http_client_timeout: <duration> | default = 20s]
  [consistent_reads: <bool> | default = false]
  [watch_rate_limit: <float64> | default = 1.0]
  [watch_burst_size: <int> | default = 1]
  [cas_retry_delay: <duration> | default 1s]

etcd:
  [endpoints: <list of string> | default = [] ]
  [dial_timeout: <duration> | default = 10s]
  [max_retries: <int> | default = 10 ]
  [tls_enabled: <bool> | default = false]

  # TLS config
  [tls_cert_path: <string> | default = ""]
  [tls_key_path: <string> | default = ""]
  [tls_ca_path: <string> | default = ""]
  [tls_server_name: <string> | default = ""]
  [tls_insecure_skip_verify: <bool> | default = false]
  [tls_cipher_suites: <string> | default = ""]
  [tls_min_version: <string> | default = ""]

  [username: <string> | default = ""]
  [password: <secret string> | default = ""]

multi:
  [primary: <string> | default = ""]
  [secondary: <string> | default = ""]
  [mirror_enabled: <bool> | default = false]
  [mirror_timeout: <bool> | default = 2s]
```

### Search

```yaml
# Target number of bytes per GET request while scanning blocks. Default is 1MB. Reducing
# this value could positively impact trace search performance at the cost of more requests
# to object storage.
[chunk_size_bytes: <uint32> | default = 1000000]

# Number of traces to prefetch while scanning blocks. Default is 1000. Increasing this value
# can improve trace search performance at the cost of memory.
[prefetch_trace_count: <int> | default = 1000]

# Number of read buffers used when performing search on a vparquet block. This value times the  read_buffer_size_bytes
# is the total amount of bytes used for buffering when performing search on a parquet block.
[read_buffer_count: <int> | default = 32]

# Size of read buffers used when performing search on a vparquet block. This value times the read_buffer_count
# is the total amount of bytes used for buffering when performing search on a parquet block.
[read_buffer_size_bytes: <int> | default = 1048576]

# Granular cache control settings for parquet metadata objects
# Deprecated. See [Cache](#cache) section.
cache_control:

    # Specifies if footer should be cached
    [footer: <bool> | default = false]

    # Specifies if column index should be cached
    [column_index: <bool> | default = false]

    # Specifies if offset index should be cached
    [offset_index: <bool> | default = false]
```

### WAL

The storage WAL configuration block.

```yaml
# Where to store the wal files while they are being appended to.
# Must be set.
# Example: "/var/tempo/wal
[path: <string> | default = ""]

# When a span is written to the WAL it adjusts the start and end times of the block it is written to.
# This block start and end time range is then used when choosing blocks for search.
# This is also used for querying traces by ID when the start and end parameters are specified. To prevent spans too far
# in the past or future from impacting the block start and end times we use this configuration option.
# This option only allows spans that occur within the configured duration to adjust the block start and
# end times.
# This can result in trace not being found if the trace falls outside the slack configuration value as the
# start and end times of the block will not be updated in this case.
[ingestion_time_range_slack: <duration> | default = 2m]
```

## Overrides

Tempo provides an overrides module for users to set global or per-tenant override settings.

### Ingestion limits

Tempo enforces ingestion limits at different points in the write path.
For an overview of where each component sits in the pipeline, refer to [About the Tempo architecture](https://grafana.com/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/about-tempo-architecture/).

The default limits in Tempo may not be sufficient in high-volume tracing environments.
Errors including `RATE_LIMITED`/`TRACE_TOO_LARGE`/`LIVE_TRACES_EXCEEDED` occur when these limits are exceeded.
See below for how to override these limits globally or per tenant.

#### Standard overrides

You can create an `overrides` section to configure ingestion limits that apply to all tenants of the cluster.

```yaml
# Overrides configuration block
overrides:

  # Global ingestion limits configurations
  defaults:

    # Ingestion related overrides
    ingestion:

      # Specifies whether the ingestion rate limits should be applied by each instance
      # of the distributor individually, or the limits are to be shared
      # across all instances. Refer to the "ingestion rate strategy" section for an example.
      # Only applies to rate_limit_bytes.
      [rate_strategy: <global|local> | default = local]

      # Per-user ingestion rate limit (bytes) used in ingestion.
      # Honored by both global and local strategies. With global, this value
      # is divided across healthy distributors.
      # Results in errors like
      #   RATE_LIMITED: ingestion rate limit (30000000 bytes) exceeded while
      #   adding 10 bytes
      [rate_limit_bytes: <int> | default = 30000000 (30MB) ]

      # Burst size (bytes) used in ingestion.
      # Results in errors like
      #   RATE_LIMITED: ingestion rate limit (30000000 bytes) exceeded while
      #   adding 10 bytes
      # Ignores rate strategy and is always local.
      [burst_size_bytes: <int> | default = 30000000 (30MB) ]

      # Maximum number of active traces per user, per live-store instance.
      # Not affected by rate_strategy.
      # A value of 0 disables the check.
      # Results in errors like
      #    LIVE_TRACES_EXCEEDED: max live traces per tenant exceeded:
      #    per-user traces limit (local: 10000 global: 0 actual local: 1) exceeded
      [max_traces_per_user: <int> | default = 10000]

      # Maximum number of active traces per user, across the cluster.
      # Not enforced at runtime in Tempo 3.0. Exposed as a metric only.
      # A value of 0 disables the check.
      [max_global_traces_per_user: <int> | default = 0]

      # Shuffle sharding shards used for this user. A value of 0 uses all partitions.
      [tenant_shard_size: <int> | default = 0]

      # Maximum bytes any attribute can be for both keys and values.
      [max_attribute_bytes: <int> | default = 0]

      # Pad push requests with an artificial delay, if set push requests will be delayed to ensure
      # an average latency of at least artificial_delay.
      [artificial_delay: <duration> | default = 0ms]

      # When enabled, the distributor includes retry information in rate-limit responses.
      [retry_info_enabled: <bool> | default = true]

    # Read related overrides
    read:
      # Maximum size in bytes of a tag-values query. Tag-values query is used mainly
      # to populate the autocomplete dropdown. This limit protects the system from
      # tags with high cardinality or large values such as HTTP URLs or SQL queries.
      # A value of 0 disables the limit.
      [max_bytes_per_tag_values_query: <int> | default = 1000000 (1MB) ]

      # Maximum number of blocks to be inspected for a tag values query. Tag-values
      # query is used mainly to populate the autocomplete dropdown. This limit
      # protects the system from long block lists.
      # A value of 0 disables the limit.
      [max_blocks_per_tag_values_query: <int> | default = 0 (disabled) ]

      # Per-user max search duration. If this value is set to 0 (default), then max_duration
      #  in the front-end configuration is used.
      [max_search_duration: <duration> | default = 0s]

      # Per-user max duration for metrics queries. If this value is set to 0 (default), then metrics max_duration
      #  in the front-end configuration is used.
      [max_metrics_duration: <duration> | default = 0s]

      # Per-user option to left-pad trace IDs with zeros to 32 hex characters in search API responses.
      # When enabled, trace IDs like "8efff798038103d269b633813fc703" will be returned as
      # "008efff798038103d269b633813fc703" to comply with the OpenTelemetry and W3C Trace Context specifications.
      [left_pad_trace_ids: <bool> | default = false]

      # Maximum number of OR-expanded condition groups allowed in a tag search query.
      # Queries that expand beyond this limit are rejected.
      [max_condition_groups_per_tag_query: <int> | default = 100]

      # Per-user toggle for unsafe query hints. When enabled, allows query hints that
      # bypass safety checks.
      [unsafe_query_hints: <bool> | default = false]

      # Per-user toggle for the span-only fetch layer for TraceQL metrics queries.
      # When not set, the default behavior is used. May be overridden by query hints.
      [metrics_spanonly_fetch: <bool>]

    # Compaction related overrides
    compaction:
      # Per-user block retention. If this value is set to 0 (default),
      # then block_retention in the compaction configuration is used.
      [block_retention: <duration> | default = 0s]
      # Per-user compaction window. If this value is set to 0 (default),
      # then compaction_window in the compaction configuration is used.
      [compaction_window: <duration> | default = 0s]
      # Allow compaction and retention to be deactivated on a per-tenant basis. Default value
      # is false (compaction active). Useful to perform operations on the backend
      # that require compaction to be disabled for a period of time.
      [compaction_disabled: <bool> | default = false]

    # Metrics-generator related overrides
    metrics_generator:

      # Per-user configuration of the metrics-generator ring size. If set, the tenant will use a
      # ring with at most the given amount of instances. Shuffle sharding is used to spread out
      # smaller rings across all instances. If the value 0 or a value larger than the total amount
      # of instances is used, all instances will be included in the ring.
      #
      # Together with metrics_generator.max_active_series this can be used to control the total
      # amount of active series. The total max active series for a specific tenant will be:
      #   metrics_generator.ring_size * metrics_generator.max_active_series
      [ring_size: <int>]

      # Per-user configuration of the metrics-generator processors. The following processors are
      # supported:
      #  - service-graphs
      #  - span-metrics
      #  - host-info
      [processors: <list of strings>]

      # Maximum number of active series in the registry, per instance of the metrics-generator. A
      # value of 0 disables this check.
      # If the limit is reached, no new series will be added but existing series will still be
      # updated. The amount of limited series can be observed with the metric
      #   tempo_metrics_generator_registry_series_limited_total
      # This setting only applies when limiter_type is set to "series" (the default).
      [max_active_series: <int>]

      # Maximum number of active entities (unique label combinations) in the registry, per instance
      # of the metrics-generator. A value of 0 disables this check.
      # If the limit is reached, no new entities will be added but existing entities will still be
      # updated. The amount of limited entities can be observed with the metric
      #   tempo_metrics_generator_registry_entities_limited_total
      # This setting only applies when limiter_type is set to "entity".
      [max_active_entities: <int>]

      # Maximum number of distinct values any single label can have. When a label exceeds the
      # configured threshold, all new label value is replaced with `__cardinality_overflow__`.
      # All other labels that is under the limit are preserved
      # If the limit is reached, no new label values will be added to the limit label.
      # The amount of limited entities can be observed with the metric:
      #   tempo_metrics_generator_registry_label_values_limited_total
      # To view the estimated cardinality demand per label:
      #   tempo_metrics_generator_registry_label_cardinality_demand_estimate
      # This setting only applies when limiter_type is set to "entity".
      # A value of 0 disables this limiter.
      [max_cardinality_per_label:  <uint64> | default = 0]

      # Per-user configuration of the collection interval. A value of 0 means the global default is
      # used set in the metrics_generator config block.
      [collection_interval: <duration>]

      # Per-user flag of the registry collection operation. If set, the registry will not be
      # collected and no samples will be exported from the metrics-generator. The metrics-generator
      # will still ingest spans and update its internal counters, including the amount of active
      # series. To disable metrics generation entirely, clear metrics_generator.processors for this
      # tenant.
      #
      # This setting is useful if you wish to test how many active series a tenant will generate, without
      # actually writing these metrics.
      [disable_collection: <bool> | default = false]

      # Per-user configuration of the trace-id label name. This value will be used as name for the label to store the
      # trace ID of exemplars in generated metrics. If not set, the default value "traceID" will be used.
      # Note it is different to the OTEL convention: https://opentelemetry.io/docs/specs/otel/compatibility/prometheus_and_openmetrics/#exemplars
      [trace_id_label_name: <string> | default = "traceID"]

      # This option only allows spans with end time that occur within the configured duration to be
      # considered in metrics generation.
      # This is to filter out spans that are outdated.
      [ingestion_time_range_slack: <duration>]

      # Configures the histogram implementation to use for span metrics and
      # service graphs processors.  If native histograms are desired, the
      # receiver must be configured to ingest native histograms.
      [generate_native_histograms: <classic|native|both> | default = classic]

      # Enables span name sanitization using DRAIN clustering to reduce cardinality.
      # Similar span names are clustered together (e.g., "GET /users/123" becomes "GET /users/<*>").
      # Options:
      #   - "" (empty string): Disabled (default)
      #   - "dry_run": Produces a demand metric for the sanitized cardinality without applying changes
      #   - "enabled": Applies DRAIN clustering to span names
      [span_name_sanitization: <string> | default = ""]

      # Per-tenant headers to include in remote write requests to the metrics backend.
      [remote_write_headers: <map of string to string>]

      # Bucket growth factor for native histograms. Only applies when
      # generate_native_histograms is set to "native" or "both".
      [native_histogram_bucket_factor: <float> | default = 1.1]

      # Maximum number of buckets for native histograms.
      [native_histogram_max_bucket_number: <int> | default = 100]

      # Minimum duration between native histogram counter resets.
      [native_histogram_min_reset_duration: <duration> | default = 15m]

      # Distributor -> metrics-generator forwarder related overrides
      forwarder:
        # Spans are stored in a queue in the distributor before being sent to the metrics-generators.
        # The length of the queue and the amount of workers pulling from the queue can be configured.
        [queue_size: <int> | default = 100]
        [workers: <int> | default = 2]

      # Per processor configuration
      processor:

        # Configuration for the service-graphs processor
        service_graphs:
          [histogram_buckets: <list of float>]
          [dimensions: <list of string>]
          [peer_attributes: <list of string>]
          [enable_client_server_prefix: <bool>]
          [enable_messaging_system_latency_histogram: <bool>]
          [enable_virtual_node_label: <bool>]
          [span_multiplier_key: <string>]
          [enable_tracestate_span_multiplier: <bool>]
          [filter_policies: [
            [
              include/include_any/exclude:
                match_type: <string> # options: strict, regex
                attributes:
                  - key: <string>
                    value: <any>
            ]
          ]]
        # Configuration for the span-metrics processor
        span_metrics:
          [histogram_buckets: <list of float>]
          # Allowed keys for intrinsic dimensions are: service, span_name, span_kind, status_code, and status_message.
          [dimensions: <list of string>]
          [intrinsic_dimensions: <map string to bool>]
          [filter_policies: [
            [
              include/include_any/exclude:
                match_type: <string> # options: strict, regex
                attributes:
                  - key: <string>
                    value: <any>
            ]
          ]]
          [dimension_mappings: <list of map>]
          # Enable target_info metrics
          [enable_target_info: <bool>]
          # Drop specific resource labels from traces_target_info
          [target_info_excluded_dimensions: <list of string>]
          # add instance label to all span metrics series when enable_target_info is true
          [enable_instance_label: <bool> | default = true]

        # Configuration for the host-info processor
        host_info:
          # Attributes used to identify the host. Checked in order until a match is found.
          [host_identifiers: <list of string> | default = ["k8s.node.name", "host.id"]]
          # Name of the generated host info metric.
          [metric_name: <string> | default = "traces_host_info"]

    # Generic forwarding configuration

    # Per-user configuration of generic forwarder feature. Each forwarder in the list
    # must refer by name to a forwarder defined in the distributor.forwarders configuration.
    forwarders: <list of string>

    # Global enforced overrides
    global:
      # Maximum size of a single trace in bytes. A value of 0 disables the size
      # check.
      # This limit is used in 3 places:
      #  - During search, traces will be skipped when they exceed this threshold.
      #  - During ingestion, traces that exceed this threshold will be refused.
      #  - During compaction (run by backend workers), traces that exceed this threshold will be partially dropped.
      # During ingestion, exceeding the threshold results in errors like
      #    TRACE_TOO_LARGE: max size of trace (5000000) exceeded while adding 387 bytes
      [max_bytes_per_trace: <int> | default = 5000000 (5MB) ]

    # Storage enforced overrides
    storage:
      # Configures attributes to be stored in dedicated columns within the parquet file, rather than in the
      # generic attribute key-value list. This allows for more efficient searching of these attributes.
      # Up to 10 span attributes and 10 resource attributes can be configured as dedicated columns.
      # Requires vParquet4 or later.
      parquet_dedicated_columns:
        [
          name: <string>, # name of the attribute
          type: <string>, # type of the attribute. options: string
          scope: <string> # scope of the attribute. options: resource, span
        ]

    # Cost attribution usage tracker configuration
    cost_attribution:
      # List of attributes to group ingested data by.  Map value is optional. Can be used to rename and
      # combine attributes.
      dimensions: <map string to string>


  # Tenant-specific overrides settings configuration file. The empty string (default
  # value) disables using an overrides file.
  [per_tenant_override_config: <string> | default = ""]

  # How frequent tenant-specific overrides are read from the configuration file.
  [per_tenant_override_period: <duration> | default = 10s]

  # Enable the deprecated legacy overrides format.
  # NOTE: This is disabled by default and will be removed in a future release.
  [enable_legacy_overrides: <bool> | default = false]

  # User-configurable overrides configuration
  user_configurable_overrides:

    # Enable the user-configurable overrides module
    [enabled: <bool> | default = false]

    # How often to poll the backend for new user-configurable overrides
    [poll_interval: <duration> | default = 60s]

    client:
      # The storage backend to use
      # Should be one of "gcs", "s3", "azure" or "local"
      [backend: <string>]

      # Backend-specific configuration, support the same configuration options as the
      # trace backend configuration
      local:
      gcs:
      s3:
      azure:

      # Check whether the backend supports versioning at startup. If enabled Tempo will not start if
      # the backend doesn't support versioning.
      [confirm_versioning: <bool> | default = true]

    api:
      # When enabled, Tempo will refuse request that modify overrides that are already set in the
      # runtime overrides. For more details, see user-configurable overrides docs.
      [check_for_conflicting_runtime_overrides: <bool> | default = false]
```

#### Tenant-specific overrides

There are two types of tenant-specific overrides:

- runtime overrides
- user-configurable overrides

##### Runtime overrides

You can set tenant-specific overrides settings in a separate file and point `per_tenant_override_config` to it.
This overrides file is dynamically loaded.
It can be changed at runtime and reloaded by Tempo without restarting the application.
These override settings can be set per tenant.

```yaml
# /conf/tempo.yaml
# Overrides configuration block
overrides:
   per_tenant_override_config: /conf/overrides.yaml

---
# /conf/overrides.yaml
# Tenant-specific overrides configuration
overrides:

  "<tenant-id>":
      ingestion:
        [rate_limit_bytes: <int>] # Honored by both global and local strategies.
        [burst_size_bytes: <int>] # Always local, regardless of rate strategy.
        [max_traces_per_user: <int>]
      global:
        [max_bytes_per_trace: <int>]

  # A "wildcard" override can be used that will apply to all tenants if a match is not found otherwise.
  "*":
    ingestion:
      [burst_size_bytes: <int>]
      [rate_limit_bytes: <int>]
      [max_traces_per_user: <int>]
    global:
      [max_bytes_per_trace: <int>]
```

##### User-configurable overrides

These tenant-specific overrides are stored in an object store and can be modified using API requests.
User-configurable overrides have priority over runtime overrides.
Refer to [user-configurable overrides](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/user-configurable-overrides/) for more details.

#### Ingestion rate strategy

Ingestion limits are enforced at different points in the write path. The distributor enforces rate and burst limits before writing to Kafka. Live-stores and block-builders enforce per-trace size and active trace count limits asynchronously after consuming from Kafka.

The `rate_strategy` setting controls how the distributor's rate limit scales across instances. It only affects `rate_limit_bytes`.

| Strategy | When to use | How it works |
|---|---|---|
| **`local`** (default) | You want each distributor to independently handle a fixed rate, and you accept that the effective cluster rate grows as you add distributors. | Each distributor enforces the full configured `rate_limit_bytes` value. With 5 distributors at `30 MB/s`, the cluster allows up to `150 MB/s`. |
| **`global`** | You need a predictable cluster-wide ingestion budget that stays constant regardless of how many distributors you run. | The configured `rate_limit_bytes` is divided across healthy distributors. With 5 distributors at `30 MB/s`, each allows `6 MB/s`. |

```yaml
overrides:
  defaults:
    ingestion:
      [rate_strategy: <global|local> | default = local]
```

The following table shows where each ingestion limit is enforced and whether it is affected by `rate_strategy`:

| Setting | Enforced by | Affected by `rate_strategy`? |
|---|---|---|
| `rate_limit_bytes` | Distributor | Yes |
| `burst_size_bytes` | Distributor | No (always per instance) |
| `max_traces_per_user` | Live-store | No (always per instance) |
| `max_global_traces_per_user` | Not enforced at runtime | No (metric only, disabled by default) |
| `max_bytes_per_trace` | Live-store, block-builder | No |

##### Examples

Each distributor instance independently allows `30 MB/s`:

```yaml
overrides:
  defaults:
    ingestion:
      rate_strategy: local
      rate_limit_bytes: 30000000
```

All distributors share a total cluster rate of `30 MB/s`.
With 5 distributors, each instance allows `6 MB/s`:

```yaml
overrides:
  defaults:
    ingestion:
      rate_strategy: global
      rate_limit_bytes: 30000000
```

For guidance on sizing these limits for your workload, refer to [Manage trace ingestion](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-trace-ingestion/).

## Usage-report

By default, Tempo reports anonymous usage data about the shape of a deployment to Grafana Labs.
This data is used to determine how common the deployment of certain features are, if a feature flag has been enabled,
and which replication factor or compression levels are used.

By providing information on how people use Tempo, usage reporting helps the Tempo team decide where to focus their development and documentation efforts. No private information is collected, and all reports are completely anonymous.

The following configuration values are used:

- Receivers enabled
- Frontend concurrency and version
- Storage cache, backend, WAL and block encodings
- Ring replication factor, and `kvstore`
- Features toggles enabled

No performance data is collected.

You can view the report by visiting this address on your Tempo instance:
`http://localhost:3200/status/usage-stats`

Refer to [Anonymous usage reporting](../configuration/anonymous-usage-reporting/) for detailed information on the information included in the report.

### Configure usage-reporting

Reporting is controlled by a configuration option.
You can disable the automatic reporting of this generic information using the following
configuration:

```yaml
usage_report:
  reporting_enabled: false
```

If you are using a Helm chart, you can enable or disable usage reporting by changing the `reportingEnabled` value.
This value is available in the [tempo-distributed](https://github.com/grafana-community/helm-charts/tree/main/charts/tempo-distributed) and the [tempo](https://github.com/grafana/helm-charts/tree/main/charts/tempo) Helm charts.

```yaml
# -- If true, Tempo will report anonymous usage data about the shape of a deployment to Grafana Labs
reportingEnabled: true
```

## Cache

Use this block to configure caches available throughout the application. Multiple caches can be created and assigned roles
which determine how they are used by Tempo.

```yaml
cache:
    # Background cache configuration. Requires having a cache configured. These settings apply
    # to all configured caches.
    background:

        # At what concurrency to write back to cache. Default is 10.
        [writeback_goroutines: <int>]

        # How many key batches to buffer for background write-back. Default is 10000.
        [writeback_buffer: <int>]

    caches:

        # Roles determine how this cache is used in Tempo. Roles must be unique across all caches and
        # every cache must have at least one role.
        # Allowed values:
        #   bloom              - Bloom filters for trace ID lookup.
        #   trace-id-index     - Trace ID index used to locate traces within blocks.
        #   parquet-footer     - Parquet footer values. Useful for search and trace by ID lookup.
        #   parquet-column-idx - Parquet column index sections.
        #   parquet-offset-idx - Parquet offset index sections.
        #   parquet-page       - Parquet data pages. WARNING: This caches most reads from Parquet and is very high volume.
        #   frontend-search    - Frontend search job results.

    -   roles:
        - <role1>
        - <role2>

        # Memcached caching configuration block
        memcached:

            # Hostname for memcached service to use. If empty and if addresses is unset, no memcached will be used.
            # Example: "host: memcached"
            [host: <string>]

            # Optional
            # SRV service used to discover memcache servers. (default: memcached)
            # Example: "service: memcached-client"
            [service: <string>]

            # Optional
            # Comma separated addresses list in DNS Service Discovery format. Refer - https://cortexmetrics.io/docs/configuration/arguments/#dns-service-discovery.
            # (default: "")
            # Example: "addresses: dns+memcached:11211"
            [addresses: <comma separated strings>]

            # Optional
            # Maximum time to wait before giving up on memcached requests.
            # (default: 100ms)
            [timeout: <duration>]

            # Optional
            # Maximum number of idle connections in pool.
            # (default: 16)
            [max_idle_conns: <int>]

            # Optional
            # Period with which to poll DNS for memcache servers.
            # (default: 1m)
            [update_interval: <duration>]

            # Optional
            # Use consistent hashing to distribute keys to memcache servers.
            # (default: true)
            [consistent_hash: <bool>]

            # Optional
            # The maximum size of an item stored in memcached, in bytes.
            # Bigger items are not stored. A value of 0 disables the limit.
            # (default: 0)
            [max_item_size: <int>]

            # Optional
            # Trip circuit-breaker after this number of consecutive dial failures.
            # (default: 10)
            [circuit_breaker_consecutive_failures: 10]

            # Optional
            # Duration circuit-breaker remains open after tripping.
            # (default: 10s)
            [circuit_breaker_timeout: 10s]

            # Optional
            # Reset circuit-breaker counts after this long.
            # (default: 10s)
            [circuit_breaker_interval: 10s]

            # Enable connecting to Memcached with TLS.
            # CLI flag: -<prefix>.memcached.tls-enabled
            [tls_enabled: <boolean> | default = false]

            # Path to the client certificate, which will be used for authenticating with
            # the server. Also requires the key path to be configured.
            # CLI flag: -<prefix>.memcached.tls-cert-path
            [tls_cert_path: <string> | default = ""]

            # Path to the key for the client certificate. Also requires the client
            # certificate to be configured.
            # CLI flag: -<prefix>.memcached.tls-key-path
            [tls_key_path: <string> | default = ""]

            # Path to the CA certificates to validate server certificate against. If not
            # set, the host's root CA certificates are used.
            # CLI flag: -<prefix>.memcached.tls-ca-path
            [tls_ca_path: <string> | default = ""]

            # Override the expected name on the server certificate.
            # CLI flag: -<prefix>.memcached.tls-server-name
            [tls_server_name: <string> | default = ""]

            # Skip validating server certificate.
            # CLI flag: -<prefix>.memcached.tls-insecure-skip-verify
            [tls_insecure_skip_verify: <boolean> | default = false]

            # Override the default cipher suite list (separated by commas). Allowed
            # values:
            #
            # Secure Ciphers:
            # - TLS_RSA_WITH_AES_128_CBC_SHA
            # - TLS_RSA_WITH_AES_256_CBC_SHA
            # - TLS_RSA_WITH_AES_128_GCM_SHA256
            # - TLS_RSA_WITH_AES_256_GCM_SHA384
            # - TLS_AES_128_GCM_SHA256
            # - TLS_AES_256_GCM_SHA384
            # - TLS_CHACHA20_POLY1305_SHA256
            # - TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA
            # - TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA
            # - TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA
            # - TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA
            # - TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
            # - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
            # - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
            # - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
            # - TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
            # - TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
            #
            # Insecure Ciphers:
            # - TLS_RSA_WITH_RC4_128_SHA
            # - TLS_RSA_WITH_3DES_EDE_CBC_SHA
            # - TLS_RSA_WITH_AES_128_CBC_SHA256
            # - TLS_ECDHE_ECDSA_WITH_RC4_128_SHA
            # - TLS_ECDHE_RSA_WITH_RC4_128_SHA
            # - TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA
            # - TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256
            # - TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256
            # CLI flag: -<prefix>.memcached.tls-cipher-suites
            [tls_cipher_suites: <string> | default = ""]

            # Override the default minimum TLS version. Allowed values: VersionTLS10,
            # VersionTLS11, VersionTLS12, VersionTLS13
            # CLI flag: -<prefix>.memcached.tls-min-version
            [tls_min_version: <string> | default = ""]

            # sets the TTL of keys in memcached
            [ttl: <string> | default = ""]

        # Redis configuration block
        # EXPERIMENTAL
        redis:

            # Redis Server endpoint to use for caching. A comma-separated list of
            # endpoints for Redis Cluster or Redis Sentinel.
            [endpoint: <string>]

            # Optional
            # Redis Sentinel master name. (default "")
            [master_name: <string>]

            # Optional
            # Maximum time to wait before giving up on redis requests. (default 500ms)
            [timeout: <duration> | default = 500ms]

            # Optional
            # How long keys stay in the redis. (default 0)
            [expiration: <duration> | default = 0s]

            # Optional
            # Database index. (default 0)
            [db: <int> | default = 0]

            # Optional
            # Maximum number of connections in the pool. (default 0)
            [pool_size: <int> | default = 0]

            # Optional
            # Username to use when connecting to redis (Redis 6+ ACL-based AUTH). (default "")
            [username: <string>]

            # Optional
            # Password to use when connecting to redis. (default "")
            [password: <string>]

            # Optional
            # Username to use when connecting to redis sentinel. (default "")
            [sentinel_username: <string>]

            # Optional
            # Password to use when connecting to redis sentinel. (default "")
            [sentinel_password: <string>]

            # Optional
            # Enable connecting to redis with TLS. (default false)
            [tls_enabled: <bool> | default = false]

            # Optional
            # Skip validating server certificate. (default false)
            [tls_insecure_skip_verify: <bool> | default = false]

            # Optional
            # Close connections after remaining idle for this duration. (default 0s)
            [idle_timeout: <duration> | default = 0s]

            # Optional
            # Close connections older than this duration. (default 0s)
            [max_connection_age: <duration> | default = 0s]

            # Optional
            # TTL for cached keys.
            [ttl: <duration>]
```

Example configuration:

```yaml
cache:
  background:
    writeback_goroutines: 5
  caches:
    - roles:
        - parquet-footer
      memcached:
        host: memcached-instance
    - roles:
        - bloom
      redis:
        endpoint: redis-instance
```

## Configure authentication

Grafana Tempo does not come with any included authentication layer. You must run an authenticating reverse proxy in front of your services to prevent unauthorized access to Tempo (for example, nginx). [Manage authentication](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/authentication/) for more details
