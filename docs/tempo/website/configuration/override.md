---
title: Override trace limits
weight: 30
---

# Override trace limits

The default per user or per tenant trace limits in Tempo may not be sufficient in high volume tracing environments. For example. the following message indicates that your tenants are running up against the trace limits imposed on them.
 ```
    max live traces per tenant exceeded: per-user traces limit (local: 10000 global: 0 actual local: 10000) exceeded
```    

In such environments, bump up the limits either globally across the cluster or individually by tenants
so that your traces are not limited by the default settings. 

This topic lists the parameters whose limits can be overridden, as well as describes various methods of
overriding these default limits.

## Override parameters

The parameters whose default values can be overridden are:

   - `ingestion_burst_size` : `NEED INFO`
   - `ingestion_max_batch_size` : Per-user allowed ingestion max batch size (in number of spans). Default is `1000`.
   - `ingestion_rate_limit` : Per-user ingestion rate limit in spans per second. Default is `100,000`.
   - `max_spans_per_trace` : Maximum number of spans per trace.  0 to disable. Default is `50,000`.
   - `max_traces_per_user`: Maximum number of active traces per user, per ingester. 0 to disable. Default is `10,000`.

`NEED INFO:The trigger points for each of the trace limits, and the error message to help the user identify which trace limit has been hit.`

## Specify overrides to the default settings

To change the default Tempo settings for various trace limits, you have to use an `overrides` section. The two approaches for specifying the overrides are:

- By adding `overrides` setting directly in the configuration file.
- By creating a new yaml file containing the `overrides` settings and then referencing it from the configuration file.

You can set the `overrides` to apply to all tenants within the cluster, to specific tenants, or globally to the cluster. For more information, refer to the [Override strategies](#override-strategies) section.

### Standard overrides

To set a new trace limit that applies to all tenants of the cluster:

1. Create an `overrides` section at the bottom of the `/tempo/integration/bench/config.yaml` file.
1. Add the override parameters under this section. For example:

    ```
    overrides: 
        max_traces_per_user: 50000
    ```

A snippet of a `config.yaml` file showing how the `overrides` section is [here](https://github.com/grafana/tempo/blob/a000a0d461221f439f585e7ed55575e7f51a0acd/integration/bench/config.yaml#L39-L40). A list of common override parameters and their description can  be found in [Override parameters](#override-parameters).

Setting a maximum value for the `overrides` setting is a failsafe way to prevent one tenant from negatively impacting others within the cluster. See also, [Distributor is not accepting traces](../../troubleshooting/#problem-4-distributor-is-not-accepting-traces) for more information.

### Tenant-specific overrides

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

The trace limits specified by the various parameters are, by default, applied as tenant-level limits. For example, a max_traces_per_user setting of 10000 means that each tenant within the cluster has a limit of 10000 traces per user. This is known as a `local` strategy in that the specified trace limits are local to each tenant.

A setting that applies at a tenant level is quite helpful in ensuring that each tenant independently can generate traces up to the limit without affecting the tracing limits on other tenants. This strategy is well suited for situations where all tenants are generating similar amounts of traces.

However, as a cluster grows quite large, this can lead to quite a large quantity of traces. An alternative strategy may be to set a cluster-level trace limit that establishes a total budget of all traces across all tenants in the cluster. Such a strategy, called `global` strategy, can be useful when different tenants produce different quantities of tracing. Here, a busy tenant can use some capacity from a less busy tenant to generate larger quantities of traces than local limits would have otherwise permitted.

The default setting for ingestion strategy is `local`. The strategy setting can also be changed using the `overrides` section, as shown in the following example:

```
overrides:
    ingestion_rate_strategy: global
```
