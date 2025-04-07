---
title: Usage tracker
description: Learn how to configure the usage tracker for cost attribution.
weight: 700
aliases:
- /docs/tempo/configuration/usage-tracker   
---

# Usage tracker

The usage tracker accurately tracks the amount of ingested traffic using a set of custom labels on a per-tenant basis, providing fine-grained control over your data.

Use the `cost_attributes` option to configure usage trackers in the distributor, which expose metrics of ingested traffic grouped by configurable attributes exposed on `/usage_metrics`.


## Enable the usage tracker

To use this feature, you need to enable it in the [distributor](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#distributor) and configure the overrides to handle the data. 

1. Open your configuration file, for example, `tempo.yaml`. 
1. In the `#distributor` section, locate the `cost_attribution` line. If it is not present, then copy and paste the example below into your distributor section and update the options. 

```
 usage:
        cost_attribution:
            # Enables the "cost-attribution" usage tracker. Per-tenant attributes are configured in overrides.
            [enabled: <boolean> | default = false]
            # Maximum number of series per tenant.
            [max_cardinality: <int> | default = 10000]
            # Interval after which a series is considered stale and will be deleted from the registry.
            # Once a metrics series is deleted, it won't be emitted anymore, keeping active series low.
            [stale_duration: <duration> | default = 15m0s]
```


### Configure overrides 

You also need to configure the dimensions to break down your the usage data in the standard overrides. 
In the overrides section, you can define attributes to group ingested data by and you can rename and combine attributes. 

For more information, refer to the [the standard overrides](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#standard-overrides) section of the Configuration documentation. 

In this example, usage data is grouped by `service.name`. 

```
# Overrides configuration block
overrides:
  # Global ingestion limits configurations
  defaults:
    # Cost attribution usage tracker configuration
    cost_attribution:
      dimensions: 
        - service.name
```

You can also configure per tenant in the [runtime-overrides](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#runtime-overrides) or in the [user-configurable-overrides](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#user-configurable-overrides).



## Review usage metrics

Once everything is configured, the usage metrics are exposed in the distributor:


```
GET /usage_metrics
```

Example:
```
curl http://localhost:3200/usage_metrics
# HELP tempo_usage_tracker_bytes_received_total bytes total received with these attributes
# TYPE tempo_usage_tracker_bytes_received_total counter
tempo_usage_tracker_bytes_received_total{service_name="article-service",tenant="single-tenant",tracker="cost-attribution"} 7327
tempo_usage_tracker_bytes_received_total{service_name="auth-service",tenant="single-tenant",tracker="cost-attribution"} 8938
tempo_usage_tracker_bytes_received_total{service_name="billing-service",tenant="single-tenant",tracker="cost-attribution"} 2401
tempo_usage_tracker_bytes_received_total{service_name="cart-service",tenant="single-tenant",tracker="cost-attribution"} 4116
tempo_usage_tracker_bytes_received_total{service_name="postgres",tenant="single-tenant",tracker="cost-attribution"} 3571
tempo_usage_tracker_bytes_received_total{service_name="shop-backend",tenant="single-tenant",tracker="cost-attribution"} 17619
```