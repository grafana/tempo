---
title: Tempo CLI
description: Guide to using tempo-cli
keywords: ["tempo", "cli", "tempo-cli", "command line interface"]
weight: 800
---

# Tempo CLI

Tempo CLI is a separate executable that contains utility functions related to the Tempo software.
Although it's not required for a working installation, Tempo CLI can be helpful for deeper analysis or for troubleshooting.

## Tempo CLI command syntax

The general syntax for commands in Tempo CLI is:

```bash
tempo-cli command [subcommand] [options] [arguments...]
```

`--help` or `-h` displays the help for a command or subcommand.

Example:

```bash
tempo-cli -h
tempo-cli command [subcommand] -h
```

## Run Tempo CLI

Tempo CLI is currently available as source code.

To build Tempo CLI, you need a working Go installation and a build environment.
You can compile it to a native binary and execute normally, or you can execute it using the `go run` command.
You can also package it as a binary in a Docker container using `make docker-tempo-cli`.

Example:

```bash
./tempo-cli [arguments...]
go run ./cmd/tempo-cli [arguments...]
```

```bash
make docker-tempo-cli
docker run docker.io/grafana/tempo-cli [arguments...]
```

## Backend options

Tempo CLI connects directly to the storage backend for some commands, meaning that it requires the ability to read from S3, GCS, Azure or file-system storage.
You can configure the backend using the following options:

- Load an existing tempo configuration file using the `--config-file` (`-c`) option. This is the recommended option
  for frequent usage. Refer to [Configuration](../../configuration/) documentation for more information.
- Specify individual settings:
  - `--backend <value>` The storage backend type, one of `s3`, `gcs`, `azure`, and `local`.
  - `--bucket <value>` The bucket name. The meaning of this value is backend-specific. Refer to [Configuration](../../configuration/) documentation for more information.
  - `--s3-endpoint <value>` The S3 API endpoint (i.e. s3.dualstack.us-east-2.amazonaws.com).
  - `--s3-user <value>`, `--s3-pass <value>` The S3 user name and password (or access key and secret key).
    Optional, as Tempo CLI supports the same authentication mechanisms as Tempo. Refer to [S3 permissions documentation](../../configuration/hosted-storage/s3/) for more information.
  - `--insecure-skip-verify` skip TLS verification, only applies to S3 and GCS.

Each option applies only to the command in which it's used. For example, `--backend <value>` doesn't permanently change where Tempo stores data. It only changes it for command in which you apply the option.

## Query API command

### Trace ID

Call the Tempo API and retrieve a trace by ID.

```bash
tempo-cli query api trace-id <api-endpoint> <trace-id>
```

Arguments:

- `api-endpoint` URL for the Tempo API.
- `trace-id` Trace ID as a hexadecimal string.

Options:

- `--org-id <value>` Organization ID (for use in multi-tenant setup).
- `--v1` use v1 API (use /api/traces endpoint to fetch traces, default: /api/v2/traces).

Example:

```bash
tempo-cli query api trace-id http://tempo:3200 f1cfe82a8eef933b
```

### Search

Call the Tempo API and search using TraceQL.

```bash
tempo-cli query api search <host-port> <trace-ql> [<start> <end>]
```

Arguments:

- `host-port` A host/port combination for Tempo. The scheme is inferred from the options.
- `trace-ql` TraceQL query.
- `start` Start of the time range to search in RFC3339 format (e.g. `2024-01-01T00:00:00Z`) or relative (e.g. `now-1h`)
- `end` End of the time range to search in RFC3339 format (e.g. `2024-01-01T01:00:00Z`) or relative (e.g. `now`)

Options:

- `--org-id <value>` Organization ID (for use in multi-tenant setup).
- `--use-grpc` Use GRPC streaming
- `--spss <value>` Number of spans to return for each spanset
- `--limit <value>` Number of results to return
- `--path-prefix <value>` String to prefix search paths with
- `--secure` Use HTTPS or gRPC with TLS

{{< admonition type="note" >}}
Set the `stream_over_http_enabled` flag to true in the Tempo configuration to enable streaming over HTTP. For more information, refer to [Tempo GRPC API documentation](../../api_docs/).
{{< /admonition >}}

Example searching for error spans using relative time:

```bash
tempo-cli query api search localhost:3200 '{status = error}' now-1h now
```

Example searching for error spans using absolute time:

```bash
tempo-cli query api search localhost:3200 '{status = error}' 2024-01-01T00:00:00Z 2024-01-01T01:00:00Z
```

Example using GRPC streaming with organization ID:

```bash
tempo-cli query api search --use-grpc --org-id my-org localhost:3200 '{span.http.status_code >= 400}' now-1h now
```

### Search tags

Call the Tempo API and search attribute names.

```bash
tempo-cli query api search-tags <host-port> [<start> <end>]
```

Arguments:

- `host-port` A host/port combination for Tempo. The scheme will be inferred based on the options provided.
- `start` Start of the time range to search in RFC3339 format (e.g. `2024-01-01T00:00:00Z`) or relative (e.g. `now-1h`)
- `end` End of the time range to search in RFC3339 format (e.g. `2024-01-01T01:00:00Z`) or relative (e.g. `now`)

Options:

- `--org-id <value>` Organization ID (for use in multi-tenant setup).
- `--use-grpc` Use GRPC streaming
- `--path-prefix <value>` String to prefix search paths with
- `--secure` Use HTTPS or gRPC with TLS

{{< admonition type="note" >}}
Set the `stream_over_http_enabled` flag to true in the Tempo configuration to enable streaming over HTTP. For more information, refer to [Tempo GRPC API documentation](../../api_docs/).
{{< /admonition >}}

Example:

```bash
tempo-cli query api search-tags localhost:3200
```

Example with relative time range:

```bash
tempo-cli query api search-tags localhost:3200 now-1h now
```

Example with absolute time range:

```bash
tempo-cli query api search-tags localhost:3200 2024-01-01T00:00:00Z 2024-01-02T00:00:00Z
```

### Search tag values

Call the Tempo API and search attribute values.

```bash
tempo-cli query api search-tag-values <host-port> <tag> [<start> <end>]
```

Arguments:

- `host-port` A host/port combination for Tempo. The scheme is inferred from the options.
- `tag` The fully qualified TraceQL tag to search for. For example, `resource.service.name`.
- `start` Start of the time range to search in RFC3339 format (e.g. `2024-01-01T00:00:00Z`) or relative (e.g. `now-1h`)
- `end` End of the time range to search in RFC3339 format (e.g. `2024-01-01T01:00:00Z`) or relative (e.g. `now`)

Options:

- `--org-id <value>` Organization ID (for use in multi-tenant setup).
- `--query <value>` TraceQL query to filter attribute results by.
- `--use-grpc` Use GRPC streaming
- `--path-prefix <value>` String to prefix search paths with
- `--secure` Use HTTP or gRPC with TLS

{{< admonition type="note" >}}
Set the `stream_over_http_enabled` flag to true in the Tempo configuration to enable streaming over HTTP. For more information, refer to [Tempo GRPC API documentation](../../api_docs/).
{{< /admonition >}}

Example to find all service names:

```bash
tempo-cli query api search-tag-values localhost:3200 resource.service.name
```

Example with query filter to find service names that have errors:

```bash
tempo-cli query api search-tag-values --query '{status = error}' localhost:3200 resource.service.name
```

### Metrics

Call the Tempo API and generate metrics from traces using TraceQL.

```bash
tempo-cli query api metrics <host-port> <trace-ql metrics query> [<start> <end>]
```

Arguments:

- `host-port` A host/port combination for Tempo. The scheme will be inferred based on the options provided.
- `trace-ql metrics query` TraceQL metrics query.
- `start` Start of the time range to search in RFC3339 format (e.g. `2024-01-01T00:00:00Z`) or relative (e.g. `now-1h`)
- `end` End of the time range to search in RFC3339 format (e.g. `2024-01-01T01:00:00Z`) or relative (e.g. `now`)

Options:

- `--org-id <value>` Organization ID (for use in multi-tenant setup).
- `--use-grpc` Use GRPC streaming
- `--instant` Perform an instant query instead of a range query.
- `--path-prefix <value>` String to prefix search paths with
- `--secure` Use HTTPS or gRPC with TLS

{{< admonition type="note" >}}
Set the `stream_over_http_enabled` flag to true in the Tempo configuration to enable streaming over HTTP. For more information, refer to [Tempo GRPC API documentation](../../api_docs/).
{{< /admonition >}}

Example range query for request rates by service:

```bash
tempo-cli query api metrics localhost:3200 '{} | rate() by (resource.service.name)' now-1h now
```

Example instant query for current error rates:

```bash
tempo-cli query api metrics --instant localhost:3200 '{status = error} | rate()' now-1h now
```

## Query trace-id command

Iterate over all backend blocks and dump all data found for a given trace ID.

```bash
tempo-cli query trace-id <trace-id> <tenant-id>
```

{{< admonition type="note" >}}
This can be intense as it downloads every bloom filter and some percentage of indexes/trace data.
{{< /admonition >}}

Arguments:

- `trace-id` Trace ID as a hexadecimal string.
- `tenant-id` Tenant to search.

Options:

- [Backend options](#backend-options)
- `--percentage <value>` Percentage of blocks to scan (for example, 0.1 for 10%). Useful for sampling large datasets.

Example:

```bash
tempo-cli query trace-id f1cfe82a8eef933b single-tenant
```

Example scanning only 10% of blocks:

```bash
tempo-cli query trace-id --percentage 0.1 f1cfe82a8eef933b single-tenant
```

## Query trace summary command

Iterate over all backend blocks and dump a summary for a given trace ID.

The summary includes:

- number of blocks the trace is found in
- span count
- trace size
- trace duration
- root service name
- root span info
- top frequent service names

```bash
tempo-cli query trace-summary <trace-id> <tenant-id>
```

{{< admonition type="note" >}}
This can be intense as it downloads every bloom filter and some percentage of indexes/trace data.
{{< /admonition >}}

Arguments:

- `trace-id` Trace ID as a hexadecimal string.
- `tenant-id` Tenant to search.

Options:

- [Backend options](#backend-options)
- `--percentage <value>` Percentage of blocks to scan (for example, 0.1 for 10%). Useful for sampling large datasets.

Example:

```bash
tempo-cli query trace-summary f1cfe82a8eef933b single-tenant
```

## List blocks

Lists information about all blocks for the given tenant, and optionally perform integrity checks on indexes for duplicate records.

```bash
tempo-cli list blocks <tenant-id>
```

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.

Options:

- `--include-compacted` Include blocks that have been compacted. Default behavior is to display only active blocks.

**Output:**
Explanation of output:

- `ID` Block ID.
- `Lvl` Compaction level of the block.
- `Objects` Number of objects stored in the block.
- `Size` Data size of the block after any compression.
- `Vers` Block version.
- `Window` The window of time that was considered for compaction purposes.
- `Start` The earliest timestamp stored in the block.
- `End` The latest timestamp stored in the block.
- `Duration` Duration between the start and end time.
- `Age` The age of the block.
- `Cmp` Whether the block has been compacted (present when --include-compacted is specified).

Example:

```bash
tempo-cli list blocks -c ./tempo.yaml single-tenant
```

## List compaction summary

Summarizes information about all blocks for the given tenant based on compaction level. This command is useful to analyze or troubleshoot compactor behavior.

```bash
tempo-cli list compaction-summary <tenant-id>
```

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.

Example:

```bash
tempo-cli list compaction-summary -c ./tempo.yaml single-tenant
```

## List cache summary

Prints information about the number of bloom filter shards per day per compaction level. This command is useful to
estimate and fine-tune cache storage. Read the [caching topic](../caching/) for more information.

```bash
tempo-cli list cache-summary <tenant-id>
```

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.

Example:

```bash
tempo-cli list cache-summary -c ./tempo.yaml single-tenant
```

## List column

Lists values in a given column of a block. Useful for inspecting parquet block data directly.

```bash
tempo-cli list column <tenant-id> <block-id> [column-name]
```

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.
- `column-name` Column name to list values of (default: `TraceID`).

Options:

- [Backend options](#backend-options)

Example:

```bash
tempo-cli list column -c ./tempo.yaml single-tenant ca314fba-efec-4852-ba3f-8d2b0bbf69f1 TraceID
```

## View schema

View block metadata, parquet schema structure, and column sizes for a given block.

```bash
tempo-cli view schema <tenant-id> <block-id>
```

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.

Options:

- [Backend options](#backend-options)

The output includes:

- Block metadata
- Parquet schema structure
- Column sizes in KB

Example:

```bash
tempo-cli view schema -c ./tempo.yaml single-tenant ca314fba-efec-4852-ba3f-8d2b0bbf69f1
```

## Query search command

Search blocks in a given time range for a specific key/value pair.

```bash
tempo-cli query search <name> <value> <start> <end> <tenant-id>
```

{{< admonition type="note" >}}
This can be intense as it downloads all relevant blocks and iterates through them.
{{< /admonition >}}

Arguments:

- `name` Name of the attribute to search for, for example, `http.method`.
- `value` Value of the attribute to search for, for example, `GET`.
- `start` Start of the time range to search in RFC3339 format (e.g. `2024-01-01T00:00:00Z`) or relative (e.g. `now-1h`)
- `end` End of the time range to search in RFC3339 format (e.g. `2024-01-01T01:00:00Z`) or relative (e.g. `now`)
- `tenant-id` Tenant to search.

Options:

- [Backend options](#backend-options)

Example using relative time:

```bash
tempo-cli query search http.method GET now-1h now single-tenant --backend=gcs --bucket=tempo-trace-data
```

Example using absolute time:

```bash
tempo-cli query search http.method GET 2024-01-01T00:00:00Z 2024-01-01T00:05:00Z single-tenant --backend=gcs --bucket=tempo-trace-data
```

## Parquet convert A to B command

Converts a vParquet file (actual data.parquet) of format A to a block of newer format B with an optional list of dedicated attribute columns.
Actual supported versions for A and B vary by Tempo release. This utility command is useful when testing the impact of different combinations
of dedicated columns.

### Convert vParquet3 to vParquet4

```bash
tempo-cli parquet convert-3to4 <in file> [<out path>] [<list of dedicated columns>]
```

Arguments:

- `in file` Path to an existing vParquet3 block directory.
- `out path` Path to write the vParquet4 block to. The default is `./out`.
- `list of dedicated columns` Optional list of columns to make dedicated. Columns use TraceQL syntax with scope. For example, `span.db.statement`, `resource.namespace`.

Example:

```bash
tempo-cli parquet convert-3to4 ./block-in ./out span.db.statement span.db.name
```

### Convert vParquet4 to vParquet5

Converts a vParquet4 block to vParquet5 format with an optional list of dedicated attribute columns.

```bash
tempo-cli parquet convert-4to5 <in file> [<out path>] [<list of dedicated columns>]
```

Arguments:

- `in file` Path to an existing vParquet4 block directory.
- `out path` Path to write the vParquet5 block to. The default is `./out`.
- `list of dedicated columns` Optional list of columns to make dedicated. Columns use TraceQL syntax with scope. For example, `span.http.method`, `resource.namespace`, `event.exception.message`.

Column prefixes:

- `int/` marks the column as an integer type. For example, `int/span.http.status_code`.
- `blob/` marks the column for blob encoding. For example, `blob/span.db.statement`.

Example:

```bash
tempo-cli parquet convert-4to5 ./block-in ./block-out "span.http.method" "int/span.http.status_code" "blob/span.db.statement"
```

## Migrate tenant command

Copy blocks from one backend and tenant to another. Blocks can be copied within the same backend or between two
different backends. The data format isn't converted but the tenant ID in `meta.json` is rewritten.

```bash
tempo-cli migrate tenant <source tenant> <dest tenant>
```

Arguments:

- `source tenant` Tenant to copy blocks from
- `dest tenant` Tenant to copy blocks into

Options:

- `--source-config-file <value>` (required) Configuration file for the source backend.
- `--config-file <value>` Configuration file for the destination backend.

Example:

```bash
tempo-cli migrate tenant --source-config-file source.yaml --config-file dest.yaml my-tenant my-other-tenant
```

## Migrate overrides config command

Migrate overrides configuration from inline format (legacy) to idented YAML format (new).

```bash
tempo-cli migrate overrides-config <source config file>
```

Arguments:

- `source config file` Configuration file to migrate

Options:

- `--config-dest <value>` Destination file for the migrated configuration. If not specified, configuration is printed to `stdout`.
- `--overrides-dest <value>` Destination file for the migrated overrides. If not specified, overrides are printed to `stdout`.

Example:

```bash
tempo-cli migrate overrides-config config.yaml --config-dest config-tmp.yaml --overrides-dest overrides-tmp.yaml
```

## Analyse block

<!-- Note that the command uses analyse and not analyze -->

Analyses a block and outputs a summary of the block's generic attributes.

It's of particular use when trying to determine candidates for dedicated attribute columns in vParquet3+.
The output includes span, resource, and event attributes with cardinality and size information.

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.

Options:

- [Backend options](#backend-options)
- `--num-attr <value>` Number of attributes to output (default: 15)
- `--num-int-attr <value>` Number of integer attributes to display. If set to 0, uses the `--num-attr` value (default: 5)
- `--blob-threshold <value>` Mark attributes as blob candidates when their dictionary size per row group exceeds this value. Set to 0 to disable. (default: 4MiB)
- `--include-well-known` Include well-known attributes in the analysis. Enable when generating dedicated columns for vParquet5 or higher. (default: false)
- `--generate-jsonnet` Generate Jsonnet overrides for dedicated columns
- `--generate-cli-args` Generate command-line arguments for the parquet conversion command
- `--int-percent-threshold <value>` Threshold for integer attributes in dedicated columns (default: 0.05)
- `--simple-summary` Print only a single line of top attributes (default: false)
- `--print-full-summary` Print full summary of the analysed block (default: true)

Example:

```bash
tempo-cli analyse block --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant b18beca6-4d7f-4464-9f72-f343e688a4a0
```

Example with blob detection:

```bash
tempo-cli analyse block --blob-threshold=4MiB --generate-jsonnet --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant b18beca6-4d7f-4464-9f72-f343e688a4a0
```

## Analyse blocks

Analyses all blocks in a given time range and outputs a summary of the blocks' generic attributes.

It's of particular use when trying to determine candidates for dedicated attribute columns in vParquet3+.
The output includes span, resource, and event attributes with cardinality and size information.

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single-tenant setups.

Options:

- [Backend options](#backend-options)
- `--num-attr <value>` Number of attributes to output (default: 15)
- `--num-int-attr <value>` Number of integer attributes to display. If set to 0, uses the `--num-attr` value (default: 5)
- `--min-compaction-level <value>` Minimum compaction level to include in the analysis (default: 3)
- `--max-blocks <value>` Maximum number of blocks to analyze (default: 10)
- `--max-start-time <value>` Oldest start time for a block to be processed. RFC3339 format (default: disabled)
- `--min-start-time <value>` Newest start time for a block to be processed. RFC3339 format (default: disabled)
- `--blob-threshold <value>` Mark attributes as blob candidates when their dictionary size per row group exceeds this value. Set to 0 to disable. (default: 4MiB)
- `--include-well-known` Include well-known attributes in the analysis. (default: false)
- `--jsonnet` Generate Jsonnet overrides for dedicated columns
- `--cli` Generate command-line arguments for the parquet conversion command
- `--int-percent-threshold <value>` Threshold for integer attributes in dedicated columns (default: 0.05)
- `--simple-summary` Print only a single line of top attributes (default: false)
- `--print-full-summary` Print full summary of the analysed block (default: true)

Example:

```bash
tempo-cli analyse blocks --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant
```

Example with blob detection and Jsonnet output:

```bash
tempo-cli analyse blocks --blob-threshold=4MiB --jsonnet --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant
```

## Suggest columns

Suggests dedicated columns for a tenant based on analysis of blocks. This command analyzes block data and outputs configuration recommendations in YAML or Jsonnet format that can be used to configure dedicated attribute columns.

```bash
tempo-cli suggest columns <tenant-id>
```

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.

Options:

- [Backend options](#backend-options)
- `--block-id <value>` Specific block ID to analyse. If not provided, analyzes multiple blocks.
- `--min-compaction-level <value>` Minimum compaction level to analyse (default: 3)
- `--max-blocks <value>` Maximum number of blocks to analyse (default: 10)
- `--num-attr <value>` Number of attributes to display (default: 10)
- `--num-int-attr <value>` Number of integer attributes to display. If set to 0, uses the `--num-attr` value (default: 0)
- `--int-percent-threshold <value>` Threshold for integer attributes put in dedicated columns (default: 0.05)
- `--include-well-known` Include well-known attributes in the analysis (default: true)
- `--blob-threshold <value>` Convert column to blob when dictionary size reaches this value (default: 4MiB)
- `--max-start-time <value>` Oldest start time for a block to be processed. RFC3339 format.
- `--min-start-time <value>` Newest start time for a block to be processed. RFC3339 format.
- `-o, --out <value>` File to write output to. If not specified, output is printed to stdout.
- `-f, --format <jsonnet|yaml>` Output format (default: yaml)

Example outputting YAML to a file:

```bash
tempo-cli suggest columns -c ./tempo.yaml --format yaml --out suggestions.yaml single-tenant
```

Example outputting Jsonnet for use in overrides:

```bash
tempo-cli suggest columns -c ./tempo.yaml --format jsonnet single-tenant
```

## Generate attribute index

{{< admonition type="warning" >}}
This command is EXPERIMENTAL and meant to facilitate experimentation with different kinds of indexes.
{{< /admonition >}}

Generate an attribute index for a parquet block. This creates an index file that can be used for faster attribute lookups.

```bash
tempo-cli gen attr-index <input-path>
```

Arguments:

- `input-path` Path to the input parquet block directory.

Options:

- `--add-intrinsics` Add intrinsic attributes to the index such as name, kind, status, and others.
- `--index-types <rows|codes|rows,codes>` Type of index to generate (default: rows,codes)

Example:

```bash
tempo-cli gen attr-index ./path/to/block
```

Example with intrinsic attributes:

```bash
tempo-cli gen attr-index --add-intrinsics ./path/to/block
```

## Drop traces by ID

Rewrites all blocks for a tenant that contain specific trace IDs. The traces are dropped from
the new blocks and the rewritten blocks are marked compacted so they will be cleaned up.

```bash
tempo-cli rewrite-blocks drop-traces <tenant-id> <trace-ids>
```

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.
- `trace-ids` The comma-separated trace IDs to drop (also supports single trace ID).

Options:

- [Backend options](#backend-options)
- `--drop-trace` By default, this command runs in dry run mode. Supplying this argument causes it to actually rewrite blocks with the traces dropped.
- `--background` Run in background mode (default: false). Suppresses progress dots for use in automated scripts.

### Examples

Dry run (default) to see which blocks would be affected:

```bash
tempo-cli rewrite-blocks drop-traces --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant 04d5f549746c96e4f3daed6202571db2
```

Drop one trace (actually perform the operation):

```bash
tempo-cli rewrite-blocks drop-traces --drop-trace --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant 04d5f549746c96e4f3daed6202571db2
```

Drop multiple traces:

```bash
tempo-cli rewrite-blocks drop-traces --drop-trace --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant 04d5f549746c96e4f3daed6202571db2,111fa1850042aea83c17cd7e674210b8
```
