---
aliases:
- /docs/tempo/v1.2.1/operations/consistent_hash_ring/
title: Consistent Hash Ring
weight: 2
---

## Consistent Hash Ring

Tempo uses the [Consistent Hash Ring](https://cortexmetrics.io/docs/architecture/#the-hash-ring) implementation from Cortex.  By default the ring is gossiped between all Tempo components.  However, it can be configured to use [Consul](https://www.consul.io/) or [Etcd](https://etcd.io/) if desired.

### Lord of the Rings?

In Tempo there are three consistent hash rings that are used for three distinct reasons: distributor, ingester and compactor. 

#### Distributor
**Participants:** Distributors  
**Used by:** Distributors

Unless you are running with limits this ring does not impact Tempo operation.

This ring is only used when "global" rate limits are used. The distributors use it to count the other active distributors. Incoming traffic is assumed to be evenly spread across all distributors and (global_rate_limit / # of distributors) is used to rate limit locally.

#### Ingester
**Participants:** Ingesters  
**Used by:** Distributors,Queriers

This ring is used by the distributors to load balance traffic into the ingesters. When spans are received the trace id is hashed and they are sent to the appropriate ingesters based on token ownership in the ring. Queriers also use this ring to find the ingesters for querying recent traces.

#### Compactor
**Participants:** Compactors  
**Used by:** Compactors

This ring is used by the compactors to shard compaction jobs. Jobs are hashed into the ring and the owning compactor is the only one allowed to compact a specific set of blocks to prevent race conditions on compaction.

### Interacting with the rings

Web pages are available at the following endpoints. They show every ring member, their tokens and includes the ability to "Forget" a ring member. "Forgetting" is useful when a
ring member leaves the ring without properly shutting down (and therefore leaves its tokens in the ring).

#### Distributor
**Available on:** Distributors  
**Path:** `/distributor/ring`

Unhealthy distributors have little impact but should be forgotten to reduce cost of maintaining the ring .

#### Ingester
**Available on:** Distributors  
**Path:** `/ingester/ring`

Unhealthy ingesters will cause writes to fail. If the ingester is really gone forget immediately!

#### Compactor
**Available on:** Compactors  
**Path:** `/compactor/ring`

Unhealthy compactors will allow the blocklist to grow significantly. If the compactor is really gone forget immediately!

### Configuring the rings

Ring/Lifecycler configuration control how a component interacts with the ring. See [configuration]({{< relref "../configuration" >}}) for more details.
