---
title: Maximum trace limit reached
weight: 474
---

# I have reached the maximum trace limit

In high volume tracing environments the default trace limits are sometimes not sufficient. For example, if you reach the [maximum number of live traces allowed](https://github.com/grafana/tempo/blob/3710d944cfe2a51836c3e4ef4a97316ed0526a58/modules/overrides/limits.go#L25) per tenant in the ingester, you will see the following messages:
`max live traces per tenant exceeded: per-user traces limit (local: 10000 global: 0 actual local: 10000) exceeded`.

### Solutions

- Check if you have the `overrides` parameter in your configuration file.
- If it is missing, add overrides using instructions in [Ingestion limits](../configuration/ingestion-limit). You can override the default values of the following parameters:

   - `ingestion_burst_size` : Burst size used in span ingestion. Default is `100,000`.
   - `ingestion_rate_limit` : Per-user ingestion rate limit in spans per second. Default is `100,000`.
   - `max_spans_per_trace` : Maximum number of spans per trace.  `0` to disable. Default is `50,000`.
   - `max_traces_per_user`: Maximum number of active traces per user, per ingester. `0` to disable. Default is `10,000`.
- Increase the maximum limit to a failsafe value. For example, increase the limit for the `max_traces_per_user` parameter from `10,000` like `50000`.
