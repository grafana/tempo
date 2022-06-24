---
title: Server-side metrics
weight: 12
---

# Server-side metrics

<span style="background-color:#f3f973;">Server-side metrics is an experimental feature.</span>

Server-side metrics is a feature that derive metrics from ingested traces.

To generate metrics, it uses an additional component: the metrics-generator.
If present, the distributor will write received spans to both the ingester and the metrics-generator.
The metrics-generator processes spans and writes metrics to a Prometheus datasource using the Prometheus remote write protocol.

## Architecture

Generating and writing metrics introduces a whole new domain to Tempo unlike any other functionality thus far.
For this reason, a component, the metrics-generator, is dedicated to working with metrics.
This results in a clean division of responsibility and limits the blast radius from a metrics processors or the Prometheus remote write exporter blowing up.


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

For a detailed view of all the config options for the metrics generator, visit [its config page]({{< relref "../configuration/#metrics-generator" >}})
