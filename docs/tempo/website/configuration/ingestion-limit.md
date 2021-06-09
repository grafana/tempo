---
title: Ingestion limits
weight: 4
---

# Ingestion limits

The default ingestion limits in Tempo may not be sufficient in high volume tracing environments. The following message, for example, indicates that a tenant is exceeding the trace limits imposed on them.
 ```
    max live traces per tenant exceeded: per-user traces limit (local: 10000 global: 0 actual local: 10000) exceeded
```    

The following sections describe the available options, per tenant overrides and global vs. local configurations.

## Configuration options

The following options can be used to limit ingestion:

   - `ingestion_burst_size_bytes` : Burst size (bytes) used in ingestion. Default is `20,000,000` (~20MB).
   - `ingestion_rate_limit_bytes` : Per-user ingestion rate limit (bytes) used in ingestion. Default is `15,000,000` (~15MB).
   - `max_bytes_per_trace` : Maximum size of a single trace in bytes.  `0` to disable. Default is `5,000,000` (~5MB).
   - `max_traces_per_user`: Maximum number of active traces per user, per ingester. `0` to disable. Default is `10,000`.

Both the `ingestion_burst_size_bytes` and `ingestion_rate_limit_bytes` parameters control the rate limit. When these limits exceed the following message is logged:

```
    RATE_LIMITED: ingestion rate limit (15000000 bytes) exceeded while adding 10 bytes
```    

When the limit for the `max_bytes_per_trace` parameter exceeds the following message is logged:

```
    TRACE_TOO_LARGE: max size of trace (5000000) exceeded while adding 387 bytes
```

Finally, when the limit for the `max_traces_per_user` parameter exceeds the following message is logged:

```
LIVE_TRACES_EXCEEDED: max live traces per tenant exceeded: per-user traces limit (local: 10000 global: 0 actual local: 1) exceeded
```

## Standard overrides

To configure new ingestion limits that applies to all tenants of the cluster:

1. Create an `overrides` section at the bottom of your configuration file.
1. Add the parameters under this section. For example:

```
    overrides:
        ingestion_burst_size_bytes: 5_000_000
        ingestion_rate_limit_bytes: 5_000_000
        max_bytes_per_trace: 5_000_000
        max_traces_per_user: 10000 
``` 

A snippet of a `config.yaml` file showing how the `overrides` section is [here](https://github.com/grafana/tempo/blob/a000a0d461221f439f585e7ed55575e7f51a0acd/integration/bench/config.yaml#L39-L40). 

## Tenant-specific overrides

Sometimes you don't want all tenants within the cluster to have the same settings. To add overrides that are specific to an individual tenant:

1. Create an `overrides` section at the bottom of the `config.yaml` file.
1. Add an entry `per_tenant_override_config` to point to a separate file named `overrides.yaml` that contains tenant-specific overrides.

    ```
    overrides:
        per_tenant_override_config: /conf/overrides.yaml
    ```

1. Within the `overrides.yaml` file, add the tenant-specific override parameters. For example:

    ```
    overrides:
        "<tenant id>":
            ingestion_burst_size_bytes: 5_000_000
            ingestion_rate_limit_bytes: 40_000_000
            max_bytes_per_trace: 25_000_000
            max_traces_per_user: 100000
    ```

This overrides file is dynamically loaded.  It can be changed at runtime and will be reloaded by Tempo without restarting the application.

1. Additionally a "wildcard" override can be used that will apply to all tenants if a match is not found otherwise.
```
    overrides:
        "*":
            ingestion_burst_size_bytes: 5_000_000
            ingestion_rate_limit_bytes: 40_000_000
            max_bytes_per_trace: 25_000_000
            max_traces_per_user: 100000
```

## Override strategies

The trace limits specified by the various parameters are, by default, applied as per-distributor limits. For example, a `max_traces_per_user` setting of 10000 means that each distributor within the cluster has a limit of 10000 traces per user. This is known as a `local` strategy in that the specified trace limits are local to each distributor.

A setting that applies at a local level is quite helpful in ensuring that each distributor independently can process traces up to the limit without affecting the tracing limits on other distributors.

However, as a cluster grows quite large, this can lead to quite a large quantity of traces. An alternative strategy may be to set a `global` trace limit that establishes a total budget of all traces across all distributors in the cluster. The global limit is averaged across all distributors by using the distributor ring.

The default setting for ingestion strategy is `local`. The strategy setting can also be changed using the `overrides` section, as shown in the following example:

```
overrides:
    ingestion_rate_strategy: global
```
