---
title: Server-side metrics
weight: 12
---

# Server-side metrics

<span style="background-color:#f3f973;">Server-side metrics is an experimental feature.</span>

Server-side metrics is a feature that allows to derive metrics from ingested traces.

To generate metrics, it uses an additional component: the metrics-generator.
If present, the distributor will write received spans to both the ingester and the metrics-generator.
The metrics-generator processes spans and writes metrics to a Prometheus datasource using the Prometheus remote write protocol.

<!-- TODO: Expand section -->

## Architecture

```
                                                                      │
                                                                      │
                                                                   Ingress
                                                                      │
                                                                      ▼
                                                          ┌──────────────────────┐
                                                          │                      │
                                                          │     Distributor      │
                                                          │                      │
                                                          └──────────────────────┘
                                                                    2│ │1
                                                                     │ │
                                                  ┌──────────────────┘ └────────┐
                                                  │                             │
                                                  ▼                             ▼
┌ ─ ─ ─ ─ ─ ─ ─ ─                     ┏━━━━━━━━━━━━━━━━━━━━━━┓      ┌──────────────────────┐
                 │                    ┃                      ┃      │                      │
│   Prometheus    ◀────Prometheus ────┃  Metrics-generator   ┃      │       Ingester       │◀───Queries────
                 │    Remote Write    ┃                      ┃      │                      │
└ ─ ─ ─ ─ ─ ─ ─ ─                     ┗━━━━━━━━━━━━━━━━━━━━━━┛      └──────────────────────┘
                                                                                │
                                                                                │
                                                                                │
                                                                                ▼
                                                                       ┌─────────────────┐
                                                                       │                 │
                                                                       │     Backend     │
                                                                       │                 │
                                                                       └─────────────────┘
```
