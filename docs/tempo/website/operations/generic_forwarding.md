---
title: Generic forwarding
weight: 13
---

# Generic forwarding

Generic forwarding allows asynchronous replication of ingested traces. The distributor writes received spans to both the ingester and defined endpoints, if enabled. This feature works in a "best-effort" manner, meaning that no retries happen if an error occurs during replication. 

>**Warning:** Generic forwarding does not work retroactively. Once enabled, the distributor only replicates freshly ingested spans.

## Configuration

Enabling generic forwarding requires the configuration of two sections. First, define a list of forwarders in the `distributor` section. Each forwarder must specify a unique `name`, supported `backend`, and backend-specific configuration. Second, reference these forwarders in the `overrides` section. This allows for fine-grained control over forwarding and makes it possible to enable this feature globally or on a per-tenant basis.

For a detailed view of all the config options for the generic forwarding feature, please refer to [distributor]({{< relref "../configuration/#distributor" >}}) and [overrides]({{< relref "../configuration/#overrides" >}}) config pages.