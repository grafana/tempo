---
title: Metrics from traces
description: Learn about how we can correlate traces and metrics.
weight: 700
aliases:
  - ./getting-started/metrics-from-traces/ # /docs/tempo/next/getting-started/metrics-from-traces/
---

# Metrics from traces

Metrics provide a powerful insight into the systems you are monitoring with your observability strategy.
Instead of running an additional service to generate metrics, you can use Grafana Tempo to generate metrics from traces.

Grafana Tempo can generate metrics from tracing data using the metrics-generator, TraceQL metrics (experimental), and the metrics summary API (deprecated).
Refer to the table for a summary of these metrics and their capabilities.
Metrics summary isn't included in the table because it's deprecated.

|                | Metrics-generator                                                                                                                                                                                                                                                                                                                               | TraceQL metrics                                                                                                                                                                                                                                                                                 |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Functionality  | An optional component within Tempo that processes incoming spans to produce predefined metrics, specifically focusing on RED (Rate, Error, Duration) metrics and service graphs.                                                                                                                                                                | An experimental feature in Tempo that allows for on-the-fly computation of metrics directly from trace data using the TraceQL query language, without the need for a separate metrics storage backend.                                                                                          |
| Capabilities   | **Span metrics:** Calculates the total count and duration of spans based on dimensions like service name, operation, span kind, status code, and other span attributes. <br> **Service graphs**: Analyzes traces to map relationships between services, identifying transactions and recording metrics related to request counts and durations. | Ad-hoc aggregation and analysis of trace data by applying functions to trace query results, similar to how LogQL operates with logs.                                                                                                                                                            |
| Output         | The generated metrics are written to a Prometheus-compatible database, enabling integration with time-series databases for storage and analysis.                                                                                                                                                                                                | Generates metrics dynamically at query time, facilitating flexible and detailed investigations into specific behaviors or patterns within the trace data.                                                                                                                                       |
| Use case       | Ideal for continuous monitoring and alerting, leveraging predefined metrics that will be stored in a time-series database. Less expressive for trace-specific analysis as it focuses on standard telemetry dimensions and RED metrics.                                                                                                          | More expressive and flexible for analyzing trace data directly, enabling complex trace-based queries and fine-grained exploration. Suited for exploratory analysis and debugging, allowing users to derive insights from trace data without prior metric definitions or storage considerations. |
| Setup          | Configure the metrics-generator in the Tempo configuration file, enable processors like span metrics or service graphs, and send metrics to a Prometheus-compatible database.                                                                                                                                                                   | Configure the local-blocks processor in overrides and in the metrics-generator configurations.                                                                                                                                                                                                  |
| Query range    | Supports querying over long time ranges, limited only by retention of the metrics backend.                                                                                                                                                                                                                                                      | Limited to a maximum query range of 3 hours by default (as of now), as metrics are computed from stored traces in real time.                                                                                                                                                                    |
| Query language | Metrics are consumed using PromQL via Prometheus/Grafana.                                                                                                                                                                                                                                                                                       | Uses TraceQL which has a PromQL-inspired syntax, but not all PromQL features are supported; it’s a similar but distinct subset with different semantics.                                                                                                                                        |

## Metrics-generator

Tempo can generate metrics from ingested traces using the metrics-generator, an optional Tempo component. The metrics-generator runs two different processors: [service graphs](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/service_graphs/) and [span metrics](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics).

The metrics-generator looks at incoming spans, and calculates rate, error, and duration (RED) metrics from them, which it then writes to a time series database like Prometheus.
By querying Prometheus, you can see the overall request rate, erroring request rate, and distribution of request latency in your system.
By using the labels on those metrics, you can get a more granular view of request rate, error rate, and latency at a per-service, per-namespace, or per-operation level.

| Useful for investigating                 | Metric   | Meaning                                                        |
| ---------------------------------------- | -------- | -------------------------------------------------------------- |
| Unusual spikes in activity               | Rate     | Number of requests per second                                  |
| Overall issues in your tracing ecosystem | Error    | Number of those requests that are failing                      |
| Response times and latency issues        | Duration | Amount of time those requests take, represented as a histogram |

The metrics-generator generates metrics from tracing data using the `services_graphs`, `span_metrics`, and `local_blocks` processors.
The `service_graphs` and `span_metrics` processors generate metrics that are written to a Prometheus-compatible backend.
The `local_blocks` processor adds support for TraceQL metrics and provides the capability of answering TraceQL metric queries to the generators without writing any data to the Prometheus backend.
The metrics-generator processes spans and write metrics using the Prometheus remote write protocol.

For more information, refer to [Metrics generator](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/).

### Use-cases for span metrics

Span metrics are of particular interest if your system isn't monitored with metrics but it has distributed tracing implemented. You get out-of-the-box metrics from your tracing pipeline.

{{< admonition type="note" >}}
In Grafana Cloud, the metrics-generator is disabled by default. Contact Grafana Support to enable metrics generation in your organization.
{{< /admonition >}}

After the metrics-generator is enabled in your organization, refer to [Metrics-generator configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#metrics-generator) for information about metrics-generator options.

![Trace service graph](/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-query-service-graph.png)

These metrics exist in your Hosted Metrics instance and can also be easily used to generate powerful custom dashboards.

<p align="center"><img src="/media/docs/tempo/intro/trace_custom_metrics_dash.png" alt="Trace custom metrics dashboard"></p>

The metrics-generator automatically generates exemplars as well which allows easy metrics to trace linking.
[Exemplars](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/fundamentals/exemplars/) are available in Grafana Cloud.

{{< figure src="/media/docs/grafana/exemplars/screenshot-exemplar-span-details.png" class="docs-image--no-shadow" max-width= "600px" caption="Span details" >}}

## TraceQL metrics (experimental)

Traces are a unique observability signal that contain causal relationships between the components in your system.

- Do you want to know how many database calls across all systems are downstream of your application?
- What services beneath a given endpoint are currently failing?
- What services beneath an endpoint are currently slow?

The experimental TraceQL metrics can answer all these questions by parsing your traces in aggregate.

You can query data generated by TraceQL metrics in a similar way that you would query results stored in Prometheus, Grafana Mimir, or other Prometheus-compatible Time-Series-Database (TSDB).
TraceQL metrics queries allows you to calculate metrics on trace span data on-the-fly with Tempo (your tracing database), without requiring a time-series-database like Prometheus.

TraceQL metrics, powered by the API of the same name, return Prometheus-like time series for a given metrics query.
Metrics queries apply a function to trace query results.
TraceQL metrics uses the `local_blocks` processor in metrics-generator.

TraceQL metrics power the [Grafana Traces Drilldown app](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/explore/simplified-exploration/traces/).
You can explore the power of visualizing your metrics in the Grafana Traces Drilldown app using Grafana Play.

{{< docs/play title="the Grafana Play site" url="https://play.grafana.org/a/grafana-exploretraces-app/explore" >}}

Refer to these resources for additional information:

- [Solves problems with TraceQL metrics queries](https://grafana.com/docs/tempo/<TEMPO_VERSION>/solutions-with-traces/solve-problems-metrics-queries/)
- [Configure TraceQL metrics](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-queries/configure-traceql-metrics/)
- [TraceQL metrics queries](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-queries/)
- [TraceQL metrics functions](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-queries/functions/)

## Metrics summary API (deprecated)

{{< admonition type="warning" >}}
The metrics summary API is deprecated as of Tempo 2.7. Features powered by the metrics summary API, like the [Aggregate by table](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/query-editor/traceql-search/#optional-use-aggregate-by), are also deprecated in Grafana Cloud and Grafana 11.3 and later.
It will be removed in a future release.
{{< /admonition >}}

The metrics summary API was an early capability in Tempo for generating ad hoc RED metrics at query time.
This data was displayed in the Aggregate by table in Grafana Explore, where you could see request rate, error rate, and latency values of your system over the last hour, computed from your trace data.
Those values were broken down by any and all attributes attached to your traces.

When you used the “Aggregate by” option, Grafana made a call to Tempo’s metrics summary API, which returned these RED metrics based on spans of `kind=server` seen in the last hour.

The metrics summary API returns RED metrics for `kind=server` spans sent to Tempo in the last hour.
The metrics summary feature creates metrics from trace data without using metrics-generator.

The metrics summary API and the Aggregate by table have been deprecated in favor of TraceQL metrics.
For more information, refer to [Deprecation in favor of TraceQL metrics](https://grafana.com/docs/tempo/<TEMPO_VERSION>/api_docs/metrics-summary/#deprecation-in-favor-of-traceql-metrics).
