---
title: Consistent Hash Ring
draft: true
---

## Consistent Hash Ring

Tempo uses the [Consistent Hash Ring](https://cortexmetrics.io/docs/architecture/#the-hash-ring) implementation from Cortex.  By default the ring is gossiped between all Tempo components.  However, it can be configured to use [Consul](https://www.consul.io/) or [Etcd](https://etcd.io/) if desired.
