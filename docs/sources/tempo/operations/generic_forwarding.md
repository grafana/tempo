---
title: Generic forwarding
menuTitle: Generic forwarding
description: Asynchronous replication of ingested traces
weight: 130
---

# Generic forwarding

Generic forwarding allows asynchronous replication of ingested traces. The distributor writes received spans to both the ingester and defined endpoints, if enabled. This feature works in a "best-effort" manner, meaning that no retries happen if an error occurs during replication.

{{< admonition type="warning" >}}
Generic forwarding does not work retroactively. Once enabled, the distributor only replicates freshly ingested spans.
{{% /admonition %}}


## Configure generic forwarding

Enabling generic forwarding requires the configuration of the `distributor` and `overrides`.

1. First, define a list of forwarders in the `distributor` section. Each forwarder must specify a unique `name`, supported `backend`, and backend-specific configuration.

1. Second, reference these forwarders in the `overrides` section. This allows for fine-grained control over forwarding and makes it possible to enable this feature globally or on a per-tenant basis.

For a detailed view of all the config options for the generic forwarding feature, please refer to [distributor]({{< relref "../configuration#distributor" >}}) and [overrides]({{< relref "../configuration#overrides" >}}) configuration pages.
