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
- `start` Start of the time range to search: (YYYY-MM-DDThh:mm:ss)
- `end` End of the time range to search: (YYYY-MM-DDThh:mm:ss)

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

### Search tags

Call the Tempo API and search attribute names.

```bash
tempo-cli query api search-tags <host-port> [<start> <end>]
```

Arguments:

- `host-port` A host/port combination for Tempo. The scheme will be inferred based on the options provided.
- `start` Start of the time range to search: (YYYY-MM-DDThh:mm:ss)
- `end` End of the time range to search: (YYYY-MM-DDThh:mm:ss)

Options:

- `--org-id <value>` Organization ID (for use in multi-tenant setup).
- `--use-grpc` Use GRPC streaming
- `--path-prefix <value>` String to prefix search paths with
- `--secure` Use HTTPS or gRPC with TLS

{{< admonition type="note" >}}
Set the `stream_over_http_enabled` flag to true in the Tempo configuration to enable streaming over HTTP. For more information, refer to [Tempo GRPC API documentation](../../api_docs/).
{{< /admonition >}}

### Search tag values

Call the Tempo API and search attribute values.

```bash
tempo-cli query api search-tag-values <host-port> <tag> [<start> <end>]
```

Arguments:

- `host-port` A host/port combination for Tempo. The scheme is inferred from the options.
- `tag` The fully qualified TraceQL tag to search for. For example, `resource.service.name`.
- `start` Start of the time range to search: (YYYY-MM-DDThh:mm:ss)
- `end` End of the time range to search: (YYYY-MM-DDThh:mm:ss)

Options:

- `--org-id <value>` Organization ID (for use in multi-tenant setup).
- `--use-grpc` Use GRPC streaming
- `--path-prefix <value>` String to prefix search paths with
- `--secure` Use HTTP or gRPC with TLS

{{< admonition type="note" >}}
Set the `stream_over_http_enabled` flag to true in the Tempo configuration to enable streaming over HTTP. For more information, refer to [Tempo GRPC API documentation](../../api_docs/).
{{< /admonition >}}

### Metrics

Call the Tempo API and generate metrics from traces using TraceQL.

```bash
tempo-cli query api metrics <host-port> <trace-ql metrics query> [<start> <end>]
```

Arguments:

- `host-port` A host/port combination for Tempo. The scheme will be inferred based on the options provided.
- `trace-ql metrics query` TraceQL metrics query.
- `start` Start of the time range to search: (YYYY-MM-DDThh:mm:ss)
- `end` End of the time range to search: (YYYY-MM-DDThh:mm:ss)

Options:

- `--org-id <value>` Organization ID (for use in multi-tenant setup).
- `--use-grpc` Use GRPC streaming
- `--path-prefix <value>` String to prefix search paths with
- `--secure` Use HTTPS or gRPC with TLS

{{< admonition type="note" >}}
Set the `stream_over_http_enabled` flag to true in the Tempo configuration to enable streaming over HTTP. For more information, refer to [Tempo GRPC API documentation](../../api_docs/).
{{< /admonition >}}

## Query blocks command

Iterate over all backend blocks and dump all data found for a given trace id.

```bash
tempo-cli query blocks <trace-id> <tenant-id>
```

{{< admonition type="note" >}}
This can be intense as it downloads every bloom filter and some percentage of indexes/trace data.
{{< /admonition >}}

Arguments:

- `trace-id` Trace ID as a hexadecimal string.
- `tenant-id` Tenant to search.

Options:
See backend options above.

Example:

```bash
tempo-cli query blocks f1cfe82a8eef933b single-tenant
```

## Query trace summary command

Iterate over all backend blocks and dump a summary for a given trace id.

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

**Note:** can be intense as it downloads every bloom filter and some percentage of indexes/trace data.

Arguments:

- `trace-id` Trace ID as a hexadecimal string.
- `tenant-id` Tenant to search.

Options:
See backend options above.

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
- `Encoding` Block encoding (compression algorithm).
- `Vers` Block version.
- `Window` The window of time that was considered for compaction purposes.
- `Start` The earliest timestamp stored in the block.
- `End` The latest timestamp stored in the block.
- `Duration`Duration between the start and end time.
- `Age` The age of the block.
- `Cmp` Whether the block has been compacted (present when --include-compacted is specified).

Example:

```bash
tempo-cli list blocks -c ./tempo.yaml single-tenant
```

## List block

Lists information about a single block, and optionally, scan its contents.

```bash
tempo-cli list block <tenant-id> <block-id>
```

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.

Options:

- `--scan` Also load the block data, perform integrity check for duplicates, and collect statistics. **Note:** can be intense.

Example:

```bash
tempo-cli list block -c ./tempo.yaml single-tenant ca314fba-efec-4852-ba3f-8d2b0bbf69f1
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

## List index

Lists basic index info for the given block.

```bash
tempo-cli list index <tenant-id> <block-id>
```

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.

Example:

```bash
tempo-cli list index -c ./tempo.yaml single-tenant ca314fba-efec-4852-ba3f-8d2b0bbf69f1
```

## View index

View the index contents for the given block.

```bash
tempo-cli view index <tenant-id> <block-id>
```

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.

Example:

```bash
tempo-cli view index -c ./tempo.yaml single-tenant ca314fba-efec-4852-ba3f-8d2b0bbf69f1
```

## Generate bloom filter

To generate the bloom filter for a block if the files were deleted/corrupted.

**Note:** ensure that the block is in a local backend in the expected directory hierarchy, i.e. `path / tenant / blocks`.

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.
- `bloom-fp` The false positive to be used for the bloom filter.
- `bloom-shard-size` The shard size to be used for the bloom filter.

Example:

```bash
tempo-cli gen bloom --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant b18beca6-4d7f-4464-9f72-f343e688a4a0 0.05 100000
```

The bloom filter will be generated at the required location under the block folder.

## Generate index

To generate the index/bloom for a block if the files were deleted/corrupted.

**Note:** ensure that the block is in a local backend in the expected directory hierarchy, i.e. `path / tenant / blocks`.

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.

Example:

```bash
tempo-cli gen index --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant b18beca6-4d7f-4464-9f72-f343e688a4a0
```

The index will be generated at the required location under the block folder.

## Search blocks command

Search blocks in a given time range for a specific key/value pair.

```bash
tempo-cli search blocks <name> <value> <start> <end> <tenant-id>
```

**Note:** can be intense as it downloads all relevant blocks and iterates through them.

Arguments:

- `name` Name of the attribute to search for e.g. `http.post`
- `value` Value of the attribute to search for e.g. `GET`
- `start` Start of the time range to search: (YYYY-MM-DDThh:mm:ss)
- `end` End of the time range to search: (YYYY-MM-DDThh:mm:ss)
- `tenant-id` Tenant to search.

Options:
See backend options above.

Example:

```bash
tempo-cli search blocks http.post GET 2021-09-21T00:00:00 2021-09-21T00:05:00 single-tenant --backend=gcs --bucket=tempo-trace-data
```

## Parquet convert command

Converts a parquet file from its existing schema to the one currently in the repository. This utility command is useful when
attempting to determine the impact of changing compression or encoding of columns.

```bash
tempo-cli parquet convert <in file> <out file>
```

Arguments:

- `in file` Filename of an existing parquet file containing Tempo trace data
- `out file` File to write to. (Existing file is overwritten.)

Example:

```bash
tempo-cli parquet convert data.parquet out.parquet
```

## Parquet convert A to B command

Converts a vParquet file (actual data.parquet) of format A to a block of newer format B with an optional list of dedicated attribute columns.
Actual supported versions for A and B vary by Tempo release. This utility command is useful when testing the impact of different combinations
of dedicated columns.

### Convert vParquet3 to vParquet4

```bash
tempo-cli parquet convert-3-to-4 <in file> [<out path>] [<list of dedicated columns>]
```

Arguments:

- `in file` Path to an existing vParquet3 block directory.
- `out path` Path to write the vParquet4 block to. The default is `./out`.
- `list of dedicated columns` Optional list of columns to make dedicated. Columns use TraceQL syntax with scope. For example, `span.db.statement`, `resource.namespace`.

Example:

```bash
tempo-cli parquet convert-3-to-4 data.parquet ./out span.db.statement span.db.name
```

### Convert vParquet4 to vParquet5

Converts a vParquet4 block to vParquet5 format with an optional list of dedicated attribute columns.

```bash
tempo-cli parquet convert-4-to-5 <in file> [<out path>] [<list of dedicated columns>]
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
tempo-cli parquet convert-4-to-5 ./block-in ./block-out "span.http.method" "int/span.http.status_code" "blob/span.db.statement"
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

- `--source-config-file <value>` Configuration file for the source backend
- `--config-file <value>` Configuration file for the destination backend

Example:

```bash
tempo-cli migrate tenant --source-config source.yaml --config-file dest.yaml my-tenant my-other-tenant
```

## Migrate overrides config command

Migrate overrides config from inline format (legacy) to idented YAML format (new).

```bash
tempo-cli migrate overrides-config <source config file>
```

Arguments:

- `source config file` Configuration file to migrate

Options:

- `--config-dest <value>` Destination file for the migrated config. If not specified, config is printed to stdout.
- `--overrides-dest <value>` Destination file for the migrated overrides. If not specified, overrides are printed to stdout.

Example:

```bash
tempo-cli migrate overrides-config config.yaml --config-dest config-tmp.yaml --overrides-dest overrides-tmp.yaml
```

## Analyse block

<!-- Note that the command uses analyse and not analyze -->

Analyses a block and outputs a summary of the block's generic attributes.
It's of particular use when trying to determine candidates for dedicated attribute columns.
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
It's of particular use when trying to determine candidates for dedicated attribute columns.
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

## Drop traces by ID

Rewrites all blocks for a tenant that contain a specific trace IDs. The traces are dropped from
the new blocks and the rewritten blocks are marked compacted so they will be cleaned up.

Arguments:

- `tenant-id` The tenant ID. Use `single-tenant` for single tenant setups.
- `trace-ids` The comma-separated trace IDs to drop (also supports single trace ID)

Options:

- [Backend options](#backend-options)
- `--drop-traces` By default, this command runs in dry run mode. Supplying this argument causes it to actually rewrite blocks with the traces dropped.

### Examples

Drop one trace:

```bash
tempo-cli rewrite-blocks drop-trace --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant 04d5f549746c96e4f3daed6202571db2
```

Drop multiple traces:

```bash
tempo-cli rewrite-blocks drop-trace --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant 04d5f549746c96e4f3daed6202571db2,111fa1850042aea83c17cd7e674210b8
```
