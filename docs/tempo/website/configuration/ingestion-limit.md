---
title: Ingestion limits
weight: 30
---

# Ingestion limits

The default per user or per tenant trace limits in Tempo may not be sufficient in high volume tracing environments. The following message, for example, indicates that your tenants are exceeding the trace limits imposed on them.
 ```
    max live traces per tenant exceeded: per-user traces limit (local: 10000 global: 0 actual local: 10000) exceeded
```    

In such environments, increase the limits either individually by tenants or globally across the cluster
so that your traces are not limited by the default settings.

This following sections describe the parameters whose limits can be overridden, as well as the methods of
overriding the default limits.

## Override parameters

You can override the default values of the following parameters:

   - `ingestion_burst_size` : Burst size used in span ingestion. Default is `100,000`.
   - `ingestion_rate_limit` : Per-user ingestion rate limit in spans per second. Default is `100,000`.
   - `max_spans_per_trace` : Maximum number of spans per trace.  `0` to disable. Default is `50,000`.
   - `max_traces_per_user`: Maximum number of active traces per user, per ingester. `0` to disable. Default is `10,000`.

Both the `ingestion_burst_size` and `ingestion_rate_limit` parameters control the rate limit. When these limits exceed the following message is logged:

```
    RATE_LIMITED: ingestion rate limit (100000 spans) exceeded while adding 10 spans
```    

When the limit for the `max_spans_per_trace` parameter exceeds the following message is logged:

```
    TRACE_TOO_LARGE: totalSpans (50000) exceeded while adding 2 spans
```

Finally, when the limit for the `max_traces_per_user` parameter exceeds the following message is logged:

```
LIVE_TRACES_EXCEEDED: max live traces per tenant exceeded: per-user traces limit (local: 10000 global: 0 actual local: 1) exceeded
```

## Standard overrides

To set a new trace limit that applies to all tenants of the cluster:

1. Create an `overrides` section at the bottom of the `/tempo/integration/bench/config.yaml` file.
1. Add the override parameters under this section. For example:

```
    overrides:
    ingestion_burst_size: 50000
    ingestion_rate_limit: 50000
    max_spans_per_trace: 50000
    max_traces_per_user: 10000 
``` 

A snippet of a `config.yaml` file showing how the `overrides` section is [here](https://github.com/grafana/tempo/blob/a000a0d461221f439f585e7ed55575e7f51a0acd/integration/bench/config.yaml#L39-L40). A list of common override parameters and their description can  be found in [Override parameters](#override-parameters).

Setting a maximum value for the `overrides` setting is a failsafe way to prevent one tenant from negatively impacting others within the cluster. See also, [Distributor is not accepting traces](../../troubleshooting/#problem-4-distributor-is-not-accepting-traces) for more information.

## Tenant-specific overrides

Sometimes you don't want all tenants within the cluster to have the same settings. To add overrides that are specific to an individual tenant:

1. Create an `overrides` section at the bottom of the `config.yaml` file.
1.  Add an entry `per_tenant_override_config` to point to a separate file named `overrides.yaml` that contains tenant-specific overrides.

    ```
    overrides:
        per_tenant_override_config: /conf/overrides.yaml
    ```

1. Within the `overrides.yaml` file, add the tenant-specific override parameters. For example:

    ```
    overrides:
        "<tenant id>":
            ingestion_max_batch_size: 5000
            ingestion_rate_limit: 400000
            max_spans_per_trace: 250000
            max_traces_per_user: 100000
    ```
## Override strategies

The trace limits specified by the various parameters are, by default, applied as per-distributor limits. For example, a `max_traces_per_user` setting of 10000 means that each distributor within the cluster has a limit of 10000 traces per user. This is known as a `local` strategy in that the specified trace limits are local to each distributor.

A setting that applies at a local level is quite helpful in ensuring that each distributor independently can process traces up to the limit without affecting the tracing limits on other distributors.

However, as a cluster grows quite large, this can lead to quite a large quantity of traces. An alternative strategy may be to set a `global` trace limit that establishes a total budget of all traces across all distributors in the cluster. The global limit is averaged across all distributors by using a distributor ring.

The default setting for ingestion strategy is `local`. The strategy setting can also be changed using the `overrides` section, as shown in the following example:

```
overrides:
    ingestion_rate_strategy: global
```
