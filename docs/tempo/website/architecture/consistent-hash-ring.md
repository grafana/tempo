---
title: Consistent Hash Ring
---

## Consistent Hash Ring

Tempo uses the [Consistent Hash Ring](https://cortexmetrics.io/docs/architecture/#the-hash-ring) implementation from Cortex.  By default the ring is gossiped between all Tempo components.  However, it can be configured to use [Consul](https://www.consul.io/) or [Etcd](https://etcd.io/) if desired.

### Accessing the rings

#### Distributor ring
Accessible at `/distributor/ring` on connecting to the distributor's http endpoint

#### Ingester ring
Accessible at `/ingester/ring` on connecting to the distributor's http endpoint

#### Compactor ring
Accessible at `/compactor/ring` on connecting to the compactor's http endpoint