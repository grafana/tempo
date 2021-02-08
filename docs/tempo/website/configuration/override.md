---
title: Override trace limit
weight: 30
---

# Override trace limit

The default per user or per tenant trace limits in Tempo may not be sufficient in high volume tracing environments. The `overrides` parameter allows you to set the maximum number of live traces allowed in the ingester. The default limit for live traces is `10000`, and for default max trace idle is `30` seconds.

Setting a maximum value for the `overrides` setting is a failsafe way to prevent one tenant from negatively impacting others within the cluster. See also, [Distributor is not accepting traces](../../troubleshooting/#problem-4-distributor-is-not-accepting-traces) for more information.

Change the default Tempo settings:

- By adding `overrides` setting directly in the configuration file.
- By creating a new yaml file containing the `overrides` settings and then referencing it from the configuration file.

You can set the `overrides` either globally, to all tenants, or to individual tenants.

## Global overrides

To add global overrides:

1. Create an `overrides` section at the bottom of the `/tempo/integration/bench/config.yaml` file.
1. Add the override parameters under this section. For example:

    ```
    overrides: 
    max_traces_per_user: 50000
    ```

A snippet of a `config.yaml` file showing how the `overrides` section is [here](https://github.com/grafana/tempo/blob/a000a0d461221f439f585e7ed55575e7f51a0acd/integration/bench/config.yaml#L39-L40). A list of common override parameters and their description can  be found in [Override parameters](#override-parameters).


## Per tenant overrides

To add per tenant overrides:
1. Create an `overrides` section at the bottom of the `config.yaml` file.
1.  Add an entry `per_tenant_override_config` to point to a separate file named `overrides.yaml` that contains per tenant overrides.

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

A list of common override parameters and their description can  be found in [Override parameters](#override-parameters).

## Override parameters

Some parameters whose default values can be overridden using the `overrides` section are:

   - `max_traces_per_user`: Maximum number of active traces per user, per ingester. 0 to disable. Default is `4323`.
   - `max_global_traces_per_user`: Maximum number of active traces per user, across the cluster. 0 to disable. Default is `0`.
