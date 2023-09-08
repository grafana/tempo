---
title: Dedicated attribute columns
description: Learn how to use dedicated attribute columns to improve query performance.
weight: 42
---

# Dedicated attribute columns

Dedicated attribute columns improve query performance by storing the most frequently used attributes in their own columns,
rather than in the generic attribute key-value list.

Introduced with `vParquet3`, dedicated attribute columns are only available when using this storage format.
To read more about the design of `vParquet3`, see [the design proposal](https://github.com/grafana/tempo/blob/main/docs/design-proposals/2023-05%20vParquet3.md).

## Configuration

Dedicated attribute columns can be configured in the storage block or via overrides.

```yaml
# Storage configuration for traces
storage:
  trace:
    block:
      version: vParquet3
      # Default dedicated columns for all blocks
      parquet_dedicated_columns:
        - name: <string>, # name of the attribute
          type: <string>, # type of the attribute. options: string
          scope: <string> # scope of the attribute. options: resource, span

overrides:
  # Global overrides for dedicated columns configuration
  parquet_dedicated_columns:
    - name: <string>, # name of the attribute
      type: <string>, # type of the attribute. options: string
      scope: <string> # scope of the attribute. options: resource, span

  per_tenant_override_config: /conf/overrides.yaml
---
# /conf/overrides.yaml
# Tenant-specific overrides configuration
overrides:
  "<tenant id>":
    parquet_dedicated_columns:
      - name: <string>, # name of the attribute
        type: <string>, # type of the attribute. options: string
        scope: <string> # scope of the attribute. options: resource, span

  # A "wildcard" override can be used that will apply to all tenants if a match is not found.
  "*":
    parquet_dedicated_columns:
      - name: <string>, # name of the attribute
        type: <string>, # type of the attribute. options: string
        scope: <string> # scope of the attribute. options: resource, span
```

Priority is given to the most specific configuration, so tenant-specific overrides will take precedence over global overrides.
Similarly, default overrides take precedence over storage block configuration.

## Usage

Dedicated attribute columns are limited to 10 span attributes and 10 resource attributes with string values.
As a rule of thumb, good candidates for dedicated attribute columns are attributes that contribute the most to the block size, 
even if they are not frequently queried.
Reducing the generic attribute key-value list size significantly improves query performance.

### Tempo-cli

You can use  the `tempo-cli` tool to find good candidates for dedicated attribute columns.
The `tempo-cli` provides the commands `analyse block <tenant-id> <block-id>` and `analyse blocks <tenant-id>` that will output the
top N attributes by size for a given block or all blocks in a tenant.

**Example:**
```bash
tempo-cli analyse blocks --backend=local --bucket=./cmd/tempo-cli/test-data/ single-tenant
```

Refer to the [tempo-cli documentation]({{< relref "./tempo_cli" >}}) for more information.