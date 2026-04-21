---
title: Tune the consistent hash rings
menuTitle: Tune the consistent hash rings
description: Optimize the consistent hash rings for Tempo.
weight: 500
aliases:
  - ../consistent_hash_ring/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/consistent_hash_ring/
---

# Tune the consistent hash rings

Tempo uses the [Consistent Hash Ring](https://cortexmetrics.io/docs/architecture/#the-hash-ring) implementation from Cortex.
By default, the ring is gossiped between all Tempo components.
However, it can be configured to use [Consul](https://www.consul.io/) or [Etcd](https://etcd.io/), if desired.

Tempo uses several consistent hash rings.
Each hash ring exists for a distinct reason.

## Distributor

**Participants:** Distributors

**Used by:** Distributors

Unless you are running with limits, this ring does not impact Tempo operation.

This ring is only used when `global` rate limits are used. The distributors use it to count the other active distributors. Incoming traffic is assumed to be evenly spread across all distributors and `(global_rate_limit / # of distributors)` is used to rate limit locally.

## Live-store (partition ring)

**Participants:** Live-stores

**Used by:** Distributors, Queriers, Block-builders

The partition ring tracks which Tempo partitions are active and which live-stores own them. Distributors use this ring to determine which partitions to write to when sending data to Kafka. Queriers use it to find the live-stores for querying recent traces. Block-builders use it to determine which partitions to consume from.

## Backend-worker

**Participants:** Backend-workers

**Used by:** Backend-workers

The backend-worker ring shards tenant index writing across backend-workers. This ring is only active when the backend-worker is configured with an external KV store for ring membership.

## Metrics-generator (partition ring)

**Participants:** Metrics-generators

**Used by:** Metrics-generators

In microservices mode, the metrics-generator partition ring tracks which generator instances own which partitions. This ring is only active when `metrics_generator.ring_mode` is set to `generator`.

## Interacting with the rings

Web pages are available at the following endpoints. They show every ring member, their tokens, and include the ability to "Forget" a ring member. "Forgetting" is useful when a ring member leaves the ring without properly shutting down, and therefore leaves its tokens in the ring.

To access a ring page, send a GET request to the Tempo HTTP API. By default, Tempo listens on port `3200` (configured with `server.http_listen_port`). For example:

```
http://<tempo-host>:3200/live-store/ring
```

In single-binary mode, any enabled ring endpoints are available on the same host. In microservices mode, each endpoint is available on the component listed in the **Available on** field.

### Distributor

**Available on:** Distributors

**Path:** `/distributor/ring`

{{< admonition type="note" >}}
This endpoint is only available when Tempo is configured with [the global ingestion rate strategy](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#ingestion-rate-strategy).
{{< /admonition >}}

Unhealthy distributors have little impact but should be forgotten to reduce the cost of maintaining the ring.

### Live-store (partition ring)

**Available on:** Distributors, Queriers, Live-stores

**Path:** `/partition-ring`

The partition ring shows partition ownership across live-stores. Unhealthy live-stores may cause recent data queries to degrade.

### Live-store

**Available on:** Distributors, Queriers, Live-stores

**Path:** `/live-store/ring`

### Backend-worker

**Available on:** Backend-workers

**Path:** `/backend-worker/ring`

{{< admonition type="note" >}}
This endpoint is only available when `backend_worker.ring.kvstore.store` is set to a non-empty value other than `inmemory` (for example, `memberlist`, `consul`, or `etcd`).
{{< /admonition >}}

The backend-worker ring page shows how tenant index writing is distributed across workers. Forget unhealthy workers so that sharding redistributes correctly.

### Metrics-generator (partition ring)

**Available on:** Metrics-generators

**Path:** `/partition/ring`

{{< admonition type="note" >}}
This endpoint is only available in microservices mode when `metrics_generator.ring_mode` is set to `generator`.
{{< /admonition >}}

The metrics-generator partition ring shows partition ownership across generator instances.

## Configuring the rings

Ring/Lifecycler configuration control how a component interacts with the ring. Refer to the [configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/) topic for more details.
