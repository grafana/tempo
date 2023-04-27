---
title: Server-side metrics architecture
menuTitle: Server-side metrics architecture
weight: 15
---

# Server-side metrics architecture

Server-side metrics is a feature that derive metrics from ingested traces.

To generate metrics, it uses an additional component: the [metrics-generator]({{< relref "../metrics-generator" >}}).
If the metrics-generator is present, the distributor will write received spans to both the ingester and the metrics-generator.
The metrics-generator processes spans and writes metrics to a Prometheus datasource using the Prometheus remote write protocol.

## Architecture

Generating and writing metrics introduces a whole new domain to Tempo unlike any other functionality thus far.
For this reason, a component, the metrics-generator, is dedicated to working with metrics.
This results in a clear separation of responsibilities and limits the impact of any issues with the metrics processors or the Prometheus remote write exporter.


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

## Configuration

For a detailed view of all the configuration options for the metrics generator, please refer to [its configuration page]({{< relref "../configuration/#metrics-generator" >}}).
