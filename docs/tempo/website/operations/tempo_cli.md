---
aliases:
- /docs/tempo/v1.2.1/operations/tempo_cli/
description: Guide to using tempo-cli
keywords:
- tempo
- cli
- tempo-cli
- command line interface
title: Tempo CLI
weight: 7
---

# Tempo CLI

Tempo CLI is a separate executable that contains utility functions related to the tempo software.  Although it is not required for a working installation, Tempo CLI can be helpful for deeper analysis or for troubleshooting.

## Tempo CLI command syntax

The general syntax for commands in Tempo CLI is:
```bash
tempo-cli command [subcommand] [options] [arguments...]
```
`--help` or `-h` displays the help for a command or subcommand.

**Example:**
```bash
tempo-cli -h
tempo-cli command [subcommand] -h
```

## Running Tempo CLI
Tempo CLI is currently available as source code. A working Go installation is required to build it. It can be compiled to a native binary and executed normally, or it can be executed using the `go run` command.

**Example:**
```bash
./tempo-cli [arguments...]
go run ./cmd/tempo-cli [arguments...]
```

## Backend options

Tempo CLI connects directly to the storage backend for some commands, meaning that it requires the ability to read from S3, GCS, Azure or file-system storage.
The backend can be configured in a few ways:

* Load an existing tempo configuration file using the `--config-file` (`-c`) option. This is the recommended option
  for frequent usage. Refer to [Configuration](../../configuration/) documentation for more information.
* Specify individual settings:
    * `--backend <value>` The storage backend type, one of `s3`, `gcs`, `azure`, and `local`.
    * `--bucket <value>` The bucket name. The meaning of this value is backend-specific. Refer to [Configuration](../../configuration/) documentation for more information.
    * `--s3-endpoint <value>` The S3 API endpoint (i.e. s3.dualstack.us-east-2.amazonaws.com).
    * `--s3-user <value>`, `--s3-password <value>` The S3 user name and password (or access key and secret key).
      Optional, as Tempo CLI supports the same authentication mechanisms as Tempo. See [S3 permissions documentation](../../configuration/s3/#permissions) for more information.

Each option applies only to the command in which it is used. For example, `--backend <value>` does not permanently change where Tempo stores data. It only changes it for command in which you apply the option.

## Query API Command
Call the tempo API and retrieve a trace by ID.
```bash
tempo-cli query api <api-endpoint> <trace-id>
```

Arguments:
- `api-endpoint` URL for tempo API.
- `trace-id` Trace ID as a hexadecimal string.

Options:
- `--org-id <value>` Organization ID (for use in multi-tenant setup).

**Example:**
```bash
tempo-cli query api http://tempo:3200 f1cfe82a8eef933b
```

## Query Blocks Command
Iterate over all backend blocks and dump all data found for a given trace id.
```bash
tempo-cli query blocks <trace-id> <tenant-id>
```
 **Note:** can be intense as it downloads every bloom filter and some percentage of indexes/trace data.

Arguments:
- `trace-id` Trace ID as a hexadecimal string.
- `tenant-id` Tenant to search.

Options:
See backend options above.

**Example:**
```bash
tempo-cli query blocks f1cfe82a8eef933b single-tenant
```

## List Blocks
Lists information about all blocks for the given tenant, and optionally perform integrity checks on indexes for duplicate records.

```bash
tempo-cli list blocks <tenant-id>
```

Arguments:
- `tenant-id` The tenant ID.  Use `single-tenant` for single tenant setups.

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

**Example:**
```bash
tempo-cli list blocks -c ./tempo.yaml single-tenant
```

## List Block
Lists information about a single block, and optionally, scan its contents.

```bash
tempo-cli list block <tenant-id> <block-id>
```

Arguments:
- `tenant-id` The tenant ID.  Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.

Options:
- `--scan` Also load the block data, perform integrity check for duplicates, and collect statistics. **Note:** can be intense.

**Example:**
```bash
tempo-cli list block -c ./tempo.yaml single-tenant ca314fba-efec-4852-ba3f-8d2b0bbf69f1
```

## List Compaction Summary
Summarizes information about all blocks for the given tenant based on compaction level. This command is useful to analyze or troubleshoot compactor behavior.

```bash
tempo-cli list compaction-summary <tenant-id>
```

Arguments:
- `tenant-id` The tenant ID.  Use `single-tenant` for single tenant setups.

**Example:**
```bash
tempo-cli list compaction-summary -c ./tempo.yaml single-tenant
```

## List Cache Summary
Prints information about the number of bloom filter shards per day per compaction level. This command is useful to
estimate and fine-tune cache storage. Read the [caching topic](../caching) for more information.

```bash
tempo-cli list cache-summary <tenant-id>
```

Arguments:
- `tenant-id` The tenant ID.  Use `single-tenant` for single tenant setups.

**Example:**
```bash
tempo-cli list cache-summary -c ./tempo.yaml single-tenant
```

## List Index
Lists basic index info for the given block.

```bash
tempo-cli list index <tenant-id> <block-id>
```

Arguments:
- `tenant-id` The tenant ID.  Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.

**Example:**
```bash
tempo-cli list index -c ./tempo.yaml single-tenant ca314fba-efec-4852-ba3f-8d2b0bbf69f1
```

## View Index
View the index contents for the given block.

```bash
tempo-cli view index <tenant-id> <block-id>
```

Arguments:
- `tenant-id` The tenant ID.  Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.

**Example:**
```bash
tempo-cli view index -c ./tempo.yaml single-tenant ca314fba-efec-4852-ba3f-8d2b0bbf69f1
```

## Generate Bloom Filter

To generate the bloom filter for a block if the files were deleted/corrupted.

**Note:** ensure that the block is in a local backend in the expected directory hierarchy, i.e. `path / tenant / blocks`.

Arguments:
- `tenant-id` The tenant ID.  Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.
- `bloom-fp` The false positive to be used for the bloom filter.
- `bloom-shard-size` The shard size to be used for the bloom filter.

**Example:**
```bash
tempo-cli gen bloom --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant b18beca6-4d7f-4464-9f72-f343e688a4a0 0.05 100000
```

The bloom filter will be generated at the required location under the block folder.

## Generate Index

To generate the index/bloom for a block if the files were deleted/corrupted.

**Note:** ensure that the block is in a local backend in the expected directory hierarchy, i.e. `path / tenant / blocks`.

Arguments:
- `tenant-id` The tenant ID.  Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.

**Example:**
```bash
tempo-cli gen index --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant b18beca6-4d7f-4464-9f72-f343e688a4a0
```

The index will be generated at the required location under the block folder.

## Search Blocks Command
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

**Example:**
```bash
tempo-cli search blocks http.post GET 2021-09-21T00:00:00 2021-09-21T00:05:00 single-tenant --backend=gcs --bucket=tempo-trace-data
```