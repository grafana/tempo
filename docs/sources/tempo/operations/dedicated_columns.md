---
title: Dedicated attribute columns
description: Learn how to use dedicated attribute columns to improve query performance.
weight: 400
---

# Dedicated attribute columns

Dedicated attribute columns improve query performance by storing the most frequently used attributes in their own columns,
rather than in the generic attribute key-value list.

Introduced with `vParquet3`, dedicated attribute columns are available when using `vParquet3` or later storage formats.
Even though `vParquet3` is deprecated, this feature is available when using the current value of `vParquet4`.

With `vParquet5`, dedicated attribute columns gain support for array-valued attributes, event-scoped attributes, and blob attributes using the `options` field.

## Configuration

You can configure dedicated attribute columns in the storage block or via overrides.

```yaml
# Storage configuration for traces
storage:
  trace:
    block:
      # Default dedicated columns for all blocks
      parquet_dedicated_columns:
        - name: <string> # name of the attribute
          type: <string> # type of the attribute. options: string, int
          scope: <string> # scope of the attribute. options: resource, span, event
          options: [<string>] # optional, vParquet5 only. options: array, blob

overrides:
  # Global overrides for dedicated columns configuration
  parquet_dedicated_columns:
    - name: <string> # name of the attribute
      type: <string> # type of the attribute. options: string, int
      scope: <string> # scope of the attribute. options: resource, span, event
      options: [<string>] # optional, vParquet5 only. options: array, blob

  per_tenant_override_config: /conf/overrides.yaml
---
# /conf/overrides.yaml
# Tenant-specific overrides configuration
overrides:
  "<tenant id>":
    parquet_dedicated_columns:
      - name: <string> # name of the attribute
        type: <string> # type of the attribute. options: string, int
        scope: <string> # scope of the attribute. options: resource, span, event
        options: [<string>] # optional, vParquet5 only. options: array, blob

  # A "wildcard" override can be used that will apply to all tenants if a match is not found.
  "*":
    parquet_dedicated_columns:
      - name: <string> # name of the attribute
        type: <string> # type of the attribute. options: string, int
        scope: <string> # scope of the attribute. options: resource, span, event
        options: [<string>] # optional, vParquet5 only. options: array, blob
```

Priority is given to the most specific configuration, so tenant-specific overrides will take precedence over global overrides.
Similarly, default overrides take precedence over storage block configuration.

## Usage

Dedicated attribute columns are limited to 20 string attributes and 10 integer attributes per scope (span, resource, and event).
As a rule of thumb, good candidates for dedicated attribute columns are attributes that contribute the most to the block size,
even if they aren't frequently queried.
Reducing the generic attribute key-value list size significantly improves query performance.

### Integer attribute selection

Integer dedicated columns are most effective when the attribute is present on at least 5% of rows in a scope.
The `tempo-cli analyse block` command shows the percentage of rows for each integer attribute and marks candidates that meet this threshold with a recommendation.
Sparse integer attributes (below 5% prevalence) typically perform better in the generic attribute storage.

### Array-valued attributes

Dedicated attribute columns support array-valued attributes.
This is useful for attributes that can have multiple values, such as HTTP headers or custom tags.

To enable array support, add `options: ["array"]` to the dedicated attribute column configuration.
The `options` field is only available in `vParquet5` and later.
Earlier versions ignore this field and only support single-valued attributes.

When `options: ["array"]` is set, the dedicated column stores multiple values per attribute.
Without this option (default), only single values are supported.

```yaml
storage:
  trace:
    block:
      version: vParquet5
      parquet_dedicated_columns:
        # Single-valued string attribute (default behavior)
        - name: http.method
          type: string
          scope: span

        # Array-valued string attribute
        - name: http.request.header.accept
          type: string
          scope: span
          options: ["array"]

        # Array-valued integer attribute
        - name: custom.retry.counts
          type: int
          scope: span
          options: ["array"]
```

### Blob attributes

Blob attributes are designed for high-cardinality or high-length string values where dictionary encoding becomes inefficient.
Examples include UUIDs, stack traces, request bodies, or any attribute with many unique values.

When an attribute's dictionary size per row group exceeds a threshold (default 4MiB), the column should be marked as a blob.
The `tempo-cli analyse block` command can help identify blob candidates by showing the estimated dictionary size per row group.

To enable blob mode, add `options: ["blob"]` to the dedicated attribute column configuration.
Blob columns use `zstd` compression instead of dictionary encoding, which is more efficient for high-cardinality data.

```yaml
storage:
  trace:
    block:
      version: vParquet5
      parquet_dedicated_columns:
        # Standard dedicated column (uses dictionary encoding)
        - name: http.method
          type: string
          scope: span

        # Blob column for high-cardinality data
        - name: db.statement
          type: string
          scope: span
          options: ["blob"]

        # Blob column for stack traces
        - name: exception.stacktrace
          type: string
          scope: event
          options: ["blob"]
```

Use the `tempo-cli analyse block` command with the `--blob-threshold` option to identify attributes that should be configured as blobs.
Attributes exceeding the threshold are marked as "(blob)" in the output.

### Event-scoped attributes

With `vParquet5`, dedicated columns support event-scoped attributes in addition to span and resource scopes.
Event-scoped columns are useful for frequently queried event attributes such as exception details or custom event data.

```yaml
storage:
  trace:
    block:
      version: vParquet5
      parquet_dedicated_columns:
        # Event-scoped string attribute
        - name: exception.message
          type: string
          scope: event

        # Event-scoped attribute with blob option
        - name: exception.stacktrace
          type: string
          scope: event
          options: ["blob"]
```

### Tempo-cli

You can use the `tempo-cli` tool to find good candidates for dedicated attribute columns.
The `tempo-cli` provides the commands `analyse block <tenant-id> <block-id>` and `analyse blocks <tenant-id>` that will output the
top N attributes by size for a given block or all blocks in a tenant.

Example:

```bash
tempo-cli analyse blocks --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant
```

Refer to the [tempo-cli documentation](../tempo_cli/) for more information.
