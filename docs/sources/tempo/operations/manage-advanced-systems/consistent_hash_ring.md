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

There are three consistent hash rings: distributor, live-store (partition ring), and metrics-generator.
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

## Metrics-generator

**Participants:** Metrics-generators

**Used by:** Queriers

This ring is used by queriers to find the metrics-generators for generating TraceQL metrics from recent traces.

## Interacting with the rings

Web pages are available at the following endpoints. They show every ring member, their tokens and includes the ability to "Forget" a ring member. "Forgetting" is useful when a
ring member leaves the ring without properly shutting down, and therefore leaves its tokens in the ring.

### Distributor

**Available on:** Distributors

**Path:** `/distributor/ring`

Unhealthy distributors have little impact but should be forgotten to reduce cost of maintaining the ring .

### Live-store (partition ring)

**Available on:** Distributors, Queriers, Live-stores

**Path:** `/partition-ring`

The partition ring shows partition ownership across live-stores. Unhealthy live-stores may cause recent data queries to degrade.

### Metrics-generators

**Available on:** Distributors

**Path:** `/metrics-generator/ring`

Unhealthy metrics-generators will cause writes to fail. If the metrics-generator is really gone, forget it immediately.

## Configuring the rings

Ring/Lifecycler configuration control how a component interacts with the ring. Refer to the [configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/) topic for more details.
