---
title: "Tempo CLI"
description: "Guide to using tempo-cli"
keywords: ["tempo", "cli", "tempo-cli", "command line interface"]
weight: 450
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

Tempo CLI connects directly to the storage backend for some commands, meaning that it requires the ability to read from S3, GCS, Azure or file-system storage.  The backend can be configured in a few ways:

* Load an existing tempo configuration file using the `--config-file` (`-c`) option. This is the recommended option for frequent usage. Refer to [Configuration](../configuration/) documentation for more information.
* Specify individual settings:
    * `--backend <value>` The storage backend type, one of `s3`, `gcs`, `azure`, and `local`.
    * `--bucket <value>` The bucket name. The meaning of this value is backend-specific. Refer to [Configuration](../configuration/) documentation for more information.
    * `--s3-endpoint <value>` The S3 API endpoint (i.e. s3.dualstack.us-east-2.amazonaws.com).
    * `--s3-user <value>`, `--s3-password <value>` The S3 user name and password (or access key and secret key). Optional, as Tempo CLI supports the same authentication mechanisms as Tempo. See [S3 permissions documentation](../configuration/s3/#permissions) for more information.

Each option applies only to the command in which it is used. For example, `--backend <value>` does not permanently change where Tempo stores data. It only changes it for command in which you apply the option.

## Query Command
Call the tempo API and retrieve a trace by ID.
```bash
tempo-cli query <api-endpoint> <trace-id>
```

Arguments:
- `api-endpoint` URL for tempo API.
- `trace-id` Trace ID as a hexadecimal string.

Options:
- `--org-id <value>` Organization ID (for use in multi-tenant setup).

**Example:**
```bash
tempo-cli query http://tempo:3100 f1cfe82a8eef933b
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
- `--load-index` Also load the block indexes and perform integrity checks for duplicates. **Note:** This can be a resource intensive process.

**Output:**
Explanation of output:
- `ID` Block ID.
- `Lvl` Compaction level of the block.
- `Count` Number of objects stored in the block.
- `Window` The window of time that was considered for compaction purposes.
- `Start` The earliest timestamp stored in the block.
- `End` The latest timestamp stored in the block.
- `Age` The age of the block.
- `Duration`Duration between the start and end time.
- `Idx` Number of records stored in the index (present when --load-index is specified).
- `Dupe` Number of duplicate entries in the index (present when --load-index is specified). Should be zero. 
- `Cmp` Whether the block has been compacted (present when --include-compacted is specified).

**Example:**
```bash
tempo-cli list blocks -c ./tempo.yaml single-tenant
```

## List Block
Lists information about a single block, and optionally, perform an integrity check for duplicate records.

```bash
tempo-cli list block <tenant-id> <block-id>
```

Arguments:
- `tenant-id` The tenant ID.  Use `single-tenant` for single tenant setups.
- `block-id` The block ID as UUID string.

Options:
- `--check-dupes` Also load the block data and perform integrity check for duplicates. **Note:** can be intense.

**Example:**
```bash
tempo-cli list block -c ./tempo.yaml single-tenant ca314fba-efec-4852-ba3f-8d2b0bbf69f1
```
