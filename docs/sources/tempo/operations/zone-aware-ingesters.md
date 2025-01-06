---
title: Zone-aware replication for ingesters
menuTitle: Zone-aware ingesters
description: Configure zone-aware ingesters for Tempo
weight: 900
---

# Zone-aware replication for ingesters

Zone awareness is a feature that ensures data is replicated across failure domains (which we refer to as “zones”) to provide greater reliability.
A failure domain is whatever you define it to be, but commonly may be an availability zone, data center, or server rack.

When zone awareness is enabled for ingesters, incoming trace data is guaranteed to be replicated to ingesters in different zones.
This allows the system to withstand the loss of one or more zones (depending on the replication factor).

Example:

```yaml
# use the following fields in _config field of JSonnet config, to enable zone-aware ingesters.
    multi_zone_ingester_enabled: false,
    multi_zone_ingester_migration_enabled: false,
    multi_zone_ingester_replicas: 0,
    multi_zone_ingester_max_unavailable: 25,
```

For an configuration, refer to the [JSonnet microservices operations example](https://github.com/grafana/tempo/blob/main/operations/jsonnet/microservices/README.md)